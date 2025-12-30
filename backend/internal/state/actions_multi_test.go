package state

import (
	"context"
	"testing"
)

func TestMultiRemotePersistence(t *testing.T) {
	ctx := context.TODO()
	sm := NewSessionManager()
	sm.DataDir = "/tmp/gitgym-test-multi-remote"

	// 1. Create Remote A
	if err := sm.CreateBareRepository(ctx, "session1", "repo-a"); err != nil {
		t.Fatalf("Failed to create repo-a: %v", err)
	}

	// 2. Create Remote B
	if err := sm.CreateBareRepository(ctx, "session1", "repo-b"); err != nil {
		t.Fatalf("Failed to create repo-b: %v", err)
	}

	// 3. Verify BOTH exist
	sm.mu.RLock()
	_, okA := sm.SharedRemotes["repo-a"]
	_, okB := sm.SharedRemotes["repo-b"]
	sm.mu.RUnlock()

	if !okA {
		t.Error("repo-a should exist but was deleted (Single Residency check failed)")
	}
	if !okB {
		t.Error("repo-b should exist")
	}
}

func TestPRRemoteAssociation(t *testing.T) {
	ctx := context.TODO()
	sm := NewSessionManager()
	sm.DataDir = "/tmp/gitgym-test-pr-assoc"

	// 1. Setup Remotes
	_ = sm.CreateBareRepository(ctx, "s1", "origin")
	_ = sm.CreateBareRepository(ctx, "s1", "upstream")

	// 2. Create PRs
	// PR 1 on origin
	_, _ = sm.CreatePullRequest("PR Origin", "Desc", "feat", "main", "dev", "origin")
	// PR 2 on upstream
	_, _ = sm.CreatePullRequest("PR Upstream", "Desc", "fix", "main", "dev", "upstream")

	// 3. Remove upstream
	if err := sm.RemoveRemote("upstream"); err != nil {
		t.Fatalf("RemoveRemote failed: %v", err)
	}

	// 4. Verify PRs
	prs := sm.GetPullRequests()
	foundOrigin := false
	foundUpstream := false

	for _, pr := range prs {
		if pr.RemoteName == "origin" {
			foundOrigin = true
		}
		if pr.RemoteName == "upstream" {
			foundUpstream = true
		}
	}

	if !foundOrigin {
		t.Error("PR on 'origin' was incorrectly deleted")
	}
	if foundUpstream {
		t.Error("PR on 'upstream' should have been deleted")
	}
}
