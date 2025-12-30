package state

import (
	"sync"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Session holds the state of a user's simulated git repo
type Session struct {
	ID               string
	Filesystem       billy.Filesystem
	Repos            map[string]*gogit.Repository // Map path (e.g., "repo1") to Repository
	CurrentDir       string                       // e.g., "/", "/repo1"
	CreatedAt        time.Time
	Reflog           []ReflogEntry
	PotentialCommits []Commit
	Manager          *SessionManager // Reference to manager for shared state
	mu               sync.RWMutex
}

// SessionManager handles concurrent access to sessions
type SessionManager struct {
	sessions          map[string]*Session
	SharedRemotes     map[string]*gogit.Repository // Share repositories across all sessions
	SharedRemotePaths map[string]string            // Maps remote name to local filesystem path
	PullRequests      []*PullRequest
	NextPRID          int
	DataDir           string
	mu                sync.RWMutex
	ingestMu          sync.Mutex // Serializes ingestion operations
}

// ReflogEntry records a command executed in the session
type ReflogEntry struct {
	Command   string
	Timestamp time.Time
	Context   string // CurrentDir, Branch etc
	Hash      string
	Message   string
}

// Commit represents a commit structure for visualization/API
type Commit struct {
	ID             string `json:"id"`
	Message        string `json:"message"`
	ParentID       string `json:"parentId"`
	SecondParentID string `json:"secondParentId,omitempty"` // For merge commits
	Timestamp      string `json:"timestamp"`
	Author         string `json:"author,omitempty"`
	TreeID         string `json:"treeId,omitempty"`
}

// PullRequest structure
type PullRequest struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	State       string    `json:"status"`       // "OPEN", "CLOSED", "MERGED"
	RemoteName  string    `json:"remoteName"`   // The shared remote this PR belongs to
	HeadRepo    string    `json:"headRepo"`     // simulating fork
	HeadRef     string    `json:"sourceBranch"` // branch
	BaseRepo    string    `json:"baseRepo"`
	BaseRef     string    `json:"targetBranch"`
	Creator     string    `json:"creator"`
	CreatedAt   time.Time `json:"createdAt"`
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:          make(map[string]*Session),
		SharedRemotes:     make(map[string]*gogit.Repository),
		SharedRemotePaths: make(map[string]string),
		PullRequests:      []*PullRequest{},
		NextPRID:          1,
		DataDir:           ".gitgym-data/remotes",
	}
}

// CreateSession initializes a new session
func (sm *SessionManager) CreateSession(id string) (*Session, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if s, exists := sm.sessions[id]; exists {
		return s, nil
	}

	fs := memfs.New()
	s := &Session{
		ID:         id,
		Filesystem: fs,
		Repos:      make(map[string]*gogit.Repository),
		CurrentDir: "/",
		CreatedAt:  time.Now(),
		Manager:    sm,
	}
	sm.sessions[id] = s
	return s, nil
}

// GetSession retrieves a session by ID
func (sm *SessionManager) GetSession(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[id]
	return s, ok
}

// GetSharedRemote safely retrieves a shared remote repository
func (sm *SessionManager) GetSharedRemote(name string) (*gogit.Repository, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	repo, ok := sm.SharedRemotes[name]
	return repo, ok
}

// Global Lock/RLock for Manager if needed (though mostly internal methods handle it)
func (sm *SessionManager) Lock() {
	sm.mu.Lock()
}

func (sm *SessionManager) Unlock() {
	sm.mu.Unlock()
}

func (sm *SessionManager) RLock() {
	sm.mu.RLock()
}

func (sm *SessionManager) RUnlock() {
	sm.mu.RUnlock()
}

// Lock locks the session for writing
func (s *Session) Lock() {
	s.mu.Lock()
}

// Unlock unlocks the session
func (s *Session) Unlock() {
	s.mu.Unlock()
}

// RLock locks the session for reading
func (s *Session) RLock() {
	s.mu.RLock()
}

// RUnlock unlocks the session for reading
func (s *Session) RUnlock() {
	s.mu.RUnlock()
}

// GetRepo returns the repository associated with the current directory
// Returns nil if no repository is active in the current directory
func (s *Session) GetRepo() *gogit.Repository {
	path := s.CurrentDir
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}

	if repo, ok := s.Repos[path]; ok {
		return repo
	}

	return nil
}

// RecordReflog adds an entry to the session reflog
func (s *Session) RecordReflog(cmd string) {
	s.Reflog = append(s.Reflog, ReflogEntry{
		Command:   cmd,
		Timestamp: time.Now(),
		Context:   s.CurrentDir,
		Hash:      "0000000",
		Message:   cmd,
	})
}

// UpdateOrigHead updates the ORIG_HEAD reference (simplified for now)
func (s *Session) UpdateOrigHead() {
	// Implementation placeholder - logic moved from session.go
}

// Helper: RemoveAll (Recursive delete for memfs/billy)
func (s *Session) RemoveAll(path string) error {
	fi, err := s.Filesystem.Stat(path)
	if err != nil {
		return nil // Already gone
	}

	if !fi.IsDir() {
		return s.Filesystem.Remove(path)
	}

	entries, err := s.Filesystem.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		childPath := path + "/" + entry.Name()
		if len(path) > 0 && path[len(path)-1] == '/' {
			childPath = path + entry.Name()
		}

		if err := s.RemoveAll(childPath); err != nil {
			return err
		}
	}

	return s.Filesystem.Remove(path)
}

// Helper: InitRepo (moved logic)
func (s *Session) InitRepo(name string) (*gogit.Repository, error) {
	// logic for init
	path := name
	if s.CurrentDir != "/" {
		// handle relative path if needed, but keeping simple for now
		_ = s.CurrentDir
	}

	err := s.Filesystem.MkdirAll(path, 0755)
	if err != nil {
		return nil, err
	}

	// Chroot
	fs, err := s.Filesystem.Chroot(path)
	if err != nil {
		return nil, err
	}

	storer := memory.NewStorage()
	repo, err := gogit.Init(storer, fs)
	if err != nil {
		return nil, err
	}

	// Set default branch to 'main' instead of 'master'
	headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.NewBranchReferenceName("main"))
	if err := storer.SetReference(headRef); err != nil {
		// Ignore error, fallback to master
		_ = err
	}

	// Create .git placeholder directory so users can see it with ls -a
	_ = fs.MkdirAll(".git", 0755)

	s.Repos[path] = repo
	return repo, nil
}

// GetIndexTree returns a tree object reflecting the current state of the index.
func (s *Session) GetIndexTree(repo *gogit.Repository) (*object.Tree, error) {
	// For simulation purposes, we can't easily convert Index to Tree without writing objects.
	// However, we can use a temporary commit to generate the tree, or manually build it.
	// Since we are in-memory, we can afford a temporary commit if we roll it back or use a side storage.

	// A simpler way: use the current Worktree to build a tree if we assume Index is clean.
	// But Index might have staged changes that are NOT in HEAD.

	// Better: Use go-git's internal tree builder logic.
	// Since we don't want to pollute history, we'll use a temporary storer or just a temporary commit.

	w, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	// For now, let's use a "dirty" approach:
	// 1. Get current HEAD commit tree
	// 2. Overlay index changes
	// This is hard.

	// Let's try the "Temporary Commit" approach:
	// 1. Save current HEAD
	// 2. Commit (staging all)
	// 3. Get that commit's tree
	// 4. Restore HEAD

	// But we only want to commit what is STAGED.
	head, headErr := repo.Head()

	// We use git commit --allow-empty --no-verify -m "temp"
	// and then hash the tree.

	// Since we are in state package, we don't have access to github.com/kurobon/gitgym/backend/internal/git
	// But we can call go-git directly.

	commitHash, err := w.Commit("temp index tree", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "System",
			Email: "system@gitgym.io",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, err
	}

	commit, err := repo.CommitObject(commitHash)
	if err != nil {
		return nil, err
	}

	tree, err := commit.Tree()
	if err != nil {
		return nil, err
	}

	// Restore HEAD
	if headErr == nil {
		// Restore to previous HEAD
		err = repo.Storer.SetReference(head)
	} else {
		// No previous HEAD (empty repo), remove it?
		// go-git doesn't have an easy way to "un-init" HEAD.
		// We'll leave it for now or use a detached head if possible.
	}

	return tree, nil
}

// GetWorktreeTree returns a tree object reflecting the current state of the filesystem.
func (s *Session) GetWorktreeTree(repo *gogit.Repository) (*object.Tree, error) {
	// To get a tree from the filesystem, we must stage everything first.
	w, err := repo.Worktree()
	if err != nil {
		return nil, err
	}

	// Save current Index?
	// Billy Filesystem doesn't easily support index backup.

	// Let's use a side-repository in memory with the same FS to avoid messing with the main repo's index.
	storage := memory.NewStorage()
	fs := w.Filesystem // Reuse same filesystem!

	tempRepo, err := gogit.Init(storage, fs)
	if err != nil {
		return nil, err
	}

	tempW, _ := tempRepo.Worktree()
	_, err = tempW.Add(".")
	if err != nil {
		return nil, err
	}

	commitHash, err := tempW.Commit("temp worktree tree", &gogit.CommitOptions{
		Author: &object.Signature{
			Name:  "System",
			Email: "system@gitgym.io",
			When:  time.Now(),
		},
	})
	if err != nil {
		return nil, err
	}

	commit, _ := tempRepo.CommitObject(commitHash)
	return commit.Tree()
}
