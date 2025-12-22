package git

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Session holds the state of a user's simulated git repo
type Session struct {
	ID         string
	Filesystem billy.Filesystem
	Repos      map[string]*git.Repository // Map path (e.g., "repo1") to Repository
	CurrentDir string                     // e.g., "/", "/repo1"
	CreatedAt  time.Time
	Reflog     []ReflogEntry
	mu         sync.RWMutex
}

// SessionManager handles concurrent access to sessions
type SessionManager struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
	}
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(id string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	session, ok := sm.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return session, nil
}

// CreateSession creates a new session if it doesn't exist
func (sm *SessionManager) CreateSession(id string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.sessions[id]; ok {
		return sm.sessions[id], nil // Return existing if already there (idempotent-ish)
	}

	fs := memfs.New()
	session := &Session{
		ID:         id,
		Filesystem: fs,
		Repos:      make(map[string]*git.Repository),
		CurrentDir: "/",
		CreatedAt:  time.Now(),
		Reflog:     []ReflogEntry{},
	}
	sm.sessions[id] = session
	return session, nil
}

// ForkSession creates a copy of an existing session
func (sm *SessionManager) ForkSession(srcID, dstID string) (*Session, error) {
	sm.mu.RLock()
	srcSession, ok := sm.sessions[srcID]
	sm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("source session %s not found", srcID)
	}

	// Create new session
	dstSession, err := sm.CreateSession(dstID)
	if err != nil {
		return nil, err
	}

	// Lock source for reading
	srcSession.mu.RLock()
	defer srcSession.mu.RUnlock()

	// Lock dest for writing
	dstSession.mu.Lock()
	defer dstSession.mu.Unlock()

	// Deep Copy Status
	dstSession.CurrentDir = srcSession.CurrentDir
	dstSession.Reflog = append([]ReflogEntry{}, srcSession.Reflog...) // Deep copy slice

	// Recursive Copy Filesystem
	if err := copyFilesystem(srcSession.Filesystem, dstSession.Filesystem, "/"); err != nil {
		return nil, fmt.Errorf("failed to copy filesystem: %w", err)
	}

	// Re-initialize Repositories
	// We iterate keys in src.Repos and try to open them in dst.Filesystem
	for path := range srcSession.Repos {
		// Just PlainOpen based on fs
		// We need to chroot if path is not root?
		// Logic in init: repoFS, _ := fs.Chroot(path) -> git.Init(st, repoFS) or PlainOpen

		// Assuming we copied the .git directory validly, PlainOpen or Open(st, fs) should work.
		// Since we use memory storage in init.go (st := memory.NewStorage()),
		// JUST copying files is NOT ENOUGH for the git database if it was purely in memory storage!

		// Wait, init.go uses:
		// st := memory.NewStorage()
		// repo, err := gogit.Init(st, repoFS)

		// `memory.NewStorage()` creates an object database in Golang heap, NOT in billy filesystem (.git/objects).
		// Creating a bare init with `memory.NewStorage` means data is NOT on disk (FS).

		// CRITICAL: We need to copy the MEMORY STORAGE content too.
		// Or... we should have used filesystem storage if we wanted persistence/copying via FS.

		// If we use memory storage, we have to iterate all objects in src repo and copy them to dst repo's storage.
		// Similar to our `push` simulation logic.

		// Re-init new memory storage for dst
		var repoFS billy.Filesystem
		if path != "" {
			var err error
			repoFS, err = dstSession.Filesystem.Chroot(path)
			if err != nil {
				return nil, err
			}
		} else {
			repoFS = dstSession.Filesystem
		}

		// Determine if bare?
		// We can check if `srcRepo.Worktree()` errors?
		srcRepo := srcSession.Repos[path]
		isBare := false
		if _, err := srcRepo.Worktree(); err == git.ErrIsBareRepository {
			isBare = true
		}

		// Initialize new Repo holder
		dstSt := memory.NewStorage()
		var dstRepo *git.Repository
		var err error

		if isBare {
			dstRepo, err = git.Init(dstSt, nil) // Bare
		} else {
			dstRepo, err = git.Init(dstSt, repoFS)
		}
		if err != nil {
			// Maybe it already exists because git.Init might fail if .git exists in FS?
			// Actually git.Init handles existing? Or use Open?
			// If we copied .git files (config, HEAD, etc) but not objects (memory),
			// PlainOpen might fail due to missing objects.
			// Best to Init fresh -> copy objects.
			return nil, fmt.Errorf("failed to init dst repo: %w", err)
		}

		// COPY ALL OBJECTS (Commits, Trees, Blobs, Tags) from srcSt to dstSt
		// Since both are in-memory, we can iterate all objects.
		srcSt := srcRepo.Storer

		// 1. Copy Objects
		iter, err := srcSt.IterEncodedObjects(plumbing.AnyObject)
		if err == nil {
			iter.ForEach(func(obj plumbing.EncodedObject) error {
				dstSt.SetEncodedObject(obj)
				return nil
			})
		}

		// 2. Copy References
		refs, err := srcRepo.References()
		if err == nil {
			refs.ForEach(func(ref *plumbing.Reference) error {
				dstRepo.Storer.SetReference(ref)
				return nil
			})
		}

		// 3. Copy Config (Remotes, etc)
		cfg, err := srcRepo.Config()
		if err == nil {
			dstRepo.SetConfig(cfg)
		}

		dstSession.Repos[path] = dstRepo
	}

	return dstSession, nil
}

// copyFilesystem recursively copies files from src to dst.
func copyFilesystem(src, dst billy.Filesystem, path string) error {
	// Read Dir
	fileInfos, err := src.ReadDir(path)
	if err != nil {
		return err
	}

	for _, fi := range fileInfos {
		fullPath := path + "/" + fi.Name()
		if path == "/" {
			fullPath = fi.Name()
		}

		if fi.IsDir() {
			if err := dst.MkdirAll(fullPath, fi.Mode()); err != nil {
				return err
			}
			if err := copyFilesystem(src, dst, fullPath); err != nil {
				return err
			}
		} else {
			// Copy File
			srcFile, err := src.Open(fullPath)
			if err != nil {
				return err
			}

			dstFile, err := dst.OpenFile(fullPath, 0644|1|2, fi.Mode()) // Create, WRONLY, TRUNC?
			// Billy flags are standard os flags usually?
			// os.O_RDWR | os.O_CREATE | os.O_TRUNC = 2 | 64 | 512 = 578?
			// No, clean usage: Create file
			if err != nil {
				srcFile.Close()
				// Try Create
				dstFile, err = dst.Create(fullPath)
				if err != nil {
					srcFile.Close()
					return err
				}
			}

			_, err = io.Copy(dstFile, srcFile)
			srcFile.Close()
			dstFile.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetRepo returns the repository associated with the current directory
// Returns nil if no repository is active in the current directory
func (s *Session) GetRepo() *git.Repository {
	// Simple logic: if CurrentDir is a repo root, return it.
	// If CurrentDir is "/", return nil (or handle nested if we supported it, but flat is easier)
	// Normalize path
	path := s.CurrentDir
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	// Direct match (assuming we are at root of repo)
	if repo, ok := s.Repos[path]; ok {
		return repo
	}

	return nil
}

// RecordReflog adds an entry to the session's reflog.
// Note: Callers must hold the session lock.
func (s *Session) RecordReflog(msg string) {
	repo := s.GetRepo()
	if repo == nil {
		return
	}
	headRef, err := repo.Head()
	hash := ""
	if err == nil {
		hash = headRef.Hash().String()
	} else {
		return // HEAD not resolving usually means no commits yet
	}

	// Prepend for newest top
	s.Reflog = append([]ReflogEntry{{Hash: hash, Message: msg}}, s.Reflog...)
}

func (s *Session) Lock() {
	s.mu.Lock()
}

func (s *Session) Unlock() {
	s.mu.Unlock()
}

// UpdateOrigHead saves the current HEAD to ORIG_HEAD ref.
// Note: Callers must hold the session lock.
func (s *Session) UpdateOrigHead() error {
	repo := s.GetRepo()
	if repo == nil {
		return nil
	}
	headRef, err := repo.Head()
	if err != nil {
		return err // No HEAD to save
	}

	origHeadRef := plumbing.NewHashReference(plumbing.ReferenceName("ORIG_HEAD"), headRef.Hash())
	return repo.Storer.SetReference(origHeadRef)
}

// RemoveAll removes path and any children it contains.
func (s *Session) RemoveAll(path string) error {
	// memfs Remove might not be recursive.
	// We implement a simple recursive removal.

	fi, err := s.Filesystem.Stat(path)
	if err != nil {
		return nil // Already gone?
	}

	if !fi.IsDir() {
		return s.Filesystem.Remove(path)
	}

	// Directory: ReadDir and remove children first
	entries, err := s.Filesystem.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		childPath := path + "/" + entry.Name()
		if path == "/" {
			childPath = entry.Name() // Handle root special case if needed, though usually strict paths
		}

		if err := s.RemoveAll(childPath); err != nil {
			return err
		}
	}

	return s.Filesystem.Remove(path)
}
