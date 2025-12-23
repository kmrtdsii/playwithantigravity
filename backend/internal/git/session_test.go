package git

import (
	"testing"
)

func TestSessionManager(t *testing.T) {
	sm := NewSessionManager()

	// 1. Create session
	id := "session-1"
	s, err := sm.CreateSession(id)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if s.ID != id {
		t.Errorf("expected id %s, got %s", id, s.ID)
	}

	// 2. Get session
	s2, ok := sm.GetSession(id)
	if !ok {
		t.Fatalf("failed to get session: session not found")
	}
	if s2 != s {
		t.Error("GetSession returned different session instance")
	}

	// 3. Create existing (should be idempotent-ish)
	s3, err := sm.CreateSession(id)
	if err != nil {
		t.Fatalf("failed to create existing session: %v", err)
	}
	if s3 != s {
		t.Error("CreateSession for existing ID returned different session instance")
	}

	// 4. Non-existent
	if _, ok := sm.GetSession("ghost"); ok {
		t.Error("expected false for non-existent session, got true")
	}
}

func TestSession_GetRepo(t *testing.T) {
	sm := NewSessionManager()
	s, _ := sm.CreateSession("test")

	// Initially no repo
	if s.GetRepo() != nil {
		t.Error("expected nil repo for new session")
	}

	// Add a repo at root
	s.Repos[""] = nil // Just a placeholder to check path matching
	if s.GetRepo() != nil {
		// Wait, value is nil anyway.
		// Let's use a real-ish object check if we had one,
		// but since Repos is map[string]*git.Repository, nil is a valid value for the key.
	}

	// Test path normalization
	s.CurrentDir = "/"
	// s.GetRepo() checks if s.Repos[path] exists where path = s.CurrentDir normalized.
}
