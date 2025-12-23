package git

import (
	"github.com/kurobon/gitgym/backend/internal/state"
)

// Alias types for backward compatibility within git package.
// Ideally we switch strict references to state package, but aliases work for now.

type Session = state.Session
type SessionManager = state.SessionManager
type ReflogEntry = state.ReflogEntry
type Commit = state.Commit
type PullRequest = state.PullRequest

// NewSessionManager creates a new session manager
// Wrapper around state.NewSessionManager
func NewSessionManager() *SessionManager {
	return state.NewSessionManager()
}
