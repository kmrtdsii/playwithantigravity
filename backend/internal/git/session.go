package git

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Session holds the state of a user's simulated git repo
type Session struct {
	ID               string
	Filesystem       billy.Filesystem
	Repos            map[string]*git.Repository // Map path (e.g., "repo1") to Repository
	CurrentDir       string                     // e.g., "/", "/repo1"
	CreatedAt        time.Time
	Reflog           []ReflogEntry
	PotentialCommits []Commit
	Manager          *SessionManager // Reference to manager for shared state
	mu               sync.RWMutex
}

// SessionManager handles concurrent access to sessions
type SessionManager struct {
	sessions      map[string]*Session
	SharedRemotes map[string]*git.Repository // Share repositories across all sessions
	PullRequests  []*PullRequest
	mu            sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions:      make(map[string]*Session),
		SharedRemotes: make(map[string]*git.Repository),
		PullRequests:  []*PullRequest{},
	}
}

// GetSharedRemote safely retrieves a shared remote repository
func (sm *SessionManager) GetSharedRemote(name string) (*git.Repository, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	repo, ok := sm.SharedRemotes[name]
	return repo, ok
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
		Manager:    sm,
	}
	sm.sessions[id] = session
	return session, nil
}

// ForkSession is REMOVED
// (Cleaned up as per user request to abolish Sandbox Mode)

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

// IngestRemote transforms a target URL into a shared remote template
func (sm *SessionManager) IngestRemote(name string, url string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Clone REAL content from the URL
	st := memory.NewStorage()

	// Use Depth 50 to avoid downloading full history of huge repos like freeCodeCamp,
	// but provide enough context for a graph.
	repo, err := git.Clone(st, nil, &git.CloneOptions{
		URL:      url,
		Progress: os.Stdout,
		Depth:    50,
		Tags:     git.NoTags, // Skip tags to save time/space
	})

	if err != nil {
		// Fallback: If network clone fails, try empty init?
		// No, user wants real content. Return error.
		return fmt.Errorf("failed to ingest remote %s: %w", url, err)
	}

	// Store under Name AND URL so CloneCommand can find it by URL
	sm.SharedRemotes[name] = repo
	sm.SharedRemotes[url] = repo
	return nil
}

// SimulateCommit creates a dummy commit on the specified remote's HEAD
func (sm *SessionManager) SimulateCommit(remoteName string, msg string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	repo, ok := sm.SharedRemotes[remoteName]
	if !ok {
		return fmt.Errorf("remote %s not found", remoteName)
	}

	// 1. Get HEAD
	headRef, err := repo.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// 2. Create Commit Object manually (bypass Worktree to allow bare/bare-ish repos)
	// We'll just use the current TreeHash so no files change, but history moves forward.
	// Or we could try to add a dummy file?
	// For "Visualizing Tree", just moving the graph is enough.
	// Let's create an empty commit (allow-empty logic equivalent)

	// Get Parent Commit to retrieve TreeHash
	parentCommit, err := repo.CommitObject(headRef.Hash())
	if err != nil {
		return fmt.Errorf("failed to get parent commit: %w", err)
	}

	commit := &object.Commit{
		Author: object.Signature{
			Name:  "Another Developer",
			Email: "dev@example.com",
			When:  time.Now(),
		},
		Committer: object.Signature{
			Name:  "Another Developer",
			Email: "dev@example.com",
			When:  time.Now(),
		},
		Message:      msg,
		TreeHash:     parentCommit.TreeHash, // Reuse tree = empty commit
		ParentHashes: []plumbing.Hash{headRef.Hash()},
	}

	// We need to encode the commit into the repo's storage

	objEncoded := repo.Storer.NewEncodedObject()
	if err := commit.Encode(objEncoded); err != nil {
		return err
	}

	newHash, err := repo.Storer.SetEncodedObject(objEncoded)
	if err != nil {
		return err
	}

	// 3. Update HEAD reference
	name := headRef.Name() // e.g. refs/heads/main
	newRef := plumbing.NewHashReference(name, newHash)
	return repo.Storer.SetReference(newRef)
}

// CreatePullRequest adds a new PR to the shared state
func (sm *SessionManager) CreatePullRequest(title, desc, source, target, creator string) (*PullRequest, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	pr := &PullRequest{
		ID:           len(sm.PullRequests) + 1,
		Title:        title,
		Description:  desc,
		SourceBranch: source,
		TargetBranch: target,
		Status:       PROpen,
		Creator:      creator,
		CreatedAt:    time.Now(),
	}
	sm.PullRequests = append(sm.PullRequests, pr)
	return pr, nil
}

// GetPullRequests returns all PRs
func (sm *SessionManager) GetPullRequests() []*PullRequest {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.PullRequests
}

// MergePullRequest simulates merging a PR in the shared remote
func (sm *SessionManager) MergePullRequest(id int, remoteName string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var targetPR *PullRequest
	for _, pr := range sm.PullRequests {
		if pr.ID == id {
			targetPR = pr
			break
		}
	}

	if targetPR == nil {
		return fmt.Errorf("PR #%d not found", id)
	}

	if targetPR.Status != PROpen {
		return fmt.Errorf("PR #%d is already %s", id, targetPR.Status)
	}

	repo, ok := sm.SharedRemotes[remoteName]
	if !ok {
		return fmt.Errorf("remote %s not found", remoteName)
	}

	// Simulation: Simply move the target branch to the source branch's hash
	srcRef, err := repo.Reference(plumbing.NewBranchReferenceName(targetPR.SourceBranch), true)
	if err != nil {
		return fmt.Errorf("source branch %s not found in remote", targetPR.SourceBranch)
	}

	targetRefName := plumbing.NewBranchReferenceName(targetPR.TargetBranch)
	newRef := plumbing.NewHashReference(targetRefName, srcRef.Hash())

	if err := repo.Storer.SetReference(newRef); err != nil {
		return err
	}

	targetPR.Status = PRMerged
	return nil
}

// RemoveRemote removes a shared remote repository from the session manager
func (sm *SessionManager) RemoveRemote(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, ok := sm.SharedRemotes[name]; !ok {
		return fmt.Errorf("remote %s not found", name)
	}

	delete(sm.SharedRemotes, name)
	return nil
}
