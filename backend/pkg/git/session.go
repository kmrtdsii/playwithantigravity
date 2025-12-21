package git

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// Session holds the state of a user's simulated git repo
type Session struct {
	ID         string
	Filesystem billy.Filesystem
	Repo       *git.Repository
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
		Repo:       nil,
		CreatedAt:  time.Now(),
		Reflog:     []ReflogEntry{},
	}
	sm.sessions[id] = session
	return session, nil
}

// RecordReflog adds an entry to the session's reflog.
// Note: Callers must hold the session lock.
func (s *Session) RecordReflog(msg string) {
	if s.Repo == nil {
		return
	}
	headRef, err := s.Repo.Head()
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
	if s.Repo == nil {
		return nil
	}
	headRef, err := s.Repo.Head()
	if err != nil {
		return err // No HEAD to save
	}
	
	origHeadRef := plumbing.NewHashReference(plumbing.ReferenceName("ORIG_HEAD"), headRef.Hash())
	return s.Repo.Storer.SetReference(origHeadRef)
}
