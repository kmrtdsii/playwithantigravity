package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/kurobon/gitgym/backend/internal/git"
)

// helper to create a session with an initialized repo and one commit
func setupBranchTestSession(t *testing.T, sm *git.SessionManager, id string) *git.Session {
	s, _ := sm.CreateSession(id)
	ctx := context.Background()

	// Init repo
	initCmd := &InitCommand{}
	_, err := initCmd.Execute(ctx, s, []string{"init", "testrepo"})
	if err != nil {
		t.Fatalf("setup: init failed: %v", err)
	}
	s.CurrentDir = "/testrepo"

	// Create a file and commit
	touchCmd := &TouchCommand{}
	_, err = touchCmd.Execute(ctx, s, []string{"touch", "file.txt"})
	if err != nil {
		t.Fatalf("setup: touch failed: %v", err)
	}

	addCmd := &AddCommand{}
	_, err = addCmd.Execute(ctx, s, []string{"add", "."})
	if err != nil {
		t.Fatalf("setup: add failed: %v", err)
	}

	commitCmd := &CommitCommand{}
	_, err = commitCmd.Execute(ctx, s, []string{"commit", "-m", "Initial commit"})
	if err != nil {
		t.Fatalf("setup: commit failed: %v", err)
	}

	return s
}

func TestBranchCommand_Help(t *testing.T) {
	sm := git.NewSessionManager()
	s := setupBranchTestSession(t, sm, "test-branch-help")
	ctx := context.Background()

	cmd := &BranchCommand{}
	res, err := cmd.Execute(ctx, s, []string{"branch", "--help"})
	if err != nil {
		t.Fatalf("--help failed: %v", err)
	}
	if !strings.Contains(res, "usage:") {
		t.Errorf("Expected help text, got: %s", res)
	}
	if !strings.Contains(res, "-d") {
		t.Errorf("Expected help to include -d option, got: %s", res)
	}
}

func TestBranchCommand_List(t *testing.T) {
	sm := git.NewSessionManager()
	s := setupBranchTestSession(t, sm, "test-branch-list")
	ctx := context.Background()

	cmd := &BranchCommand{}
	res, err := cmd.Execute(ctx, s, []string{"branch"})
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	// Should contain 'master' or 'main' (default branch)
	if res == "" {
		t.Errorf("Expected at least one branch, got empty")
	}
}

func TestBranchCommand_Create(t *testing.T) {
	sm := git.NewSessionManager()
	s := setupBranchTestSession(t, sm, "test-branch-create")
	ctx := context.Background()

	cmd := &BranchCommand{}
	res, err := cmd.Execute(ctx, s, []string{"branch", "feature-x"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if !strings.Contains(res, "Created branch feature-x") {
		t.Errorf("Expected creation message, got: %s", res)
	}

	// Verify branch exists
	listRes, err := cmd.Execute(ctx, s, []string{"branch"})
	if err != nil {
		t.Fatalf("list after create failed: %v", err)
	}
	if !strings.Contains(listRes, "feature-x") {
		t.Errorf("Expected feature-x in list, got: %s", listRes)
	}
}

func TestBranchCommand_Delete(t *testing.T) {
	sm := git.NewSessionManager()
	s := setupBranchTestSession(t, sm, "test-branch-delete")
	ctx := context.Background()

	cmd := &BranchCommand{}

	// Create a branch to delete
	_, err := cmd.Execute(ctx, s, []string{"branch", "to-delete"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Delete it
	res, err := cmd.Execute(ctx, s, []string{"branch", "-d", "to-delete"})
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if !strings.Contains(res, "Deleted branch to-delete") {
		t.Errorf("Expected deletion message, got: %s", res)
	}

	// Verify branch is gone
	listRes, err := cmd.Execute(ctx, s, []string{"branch"})
	if err != nil {
		t.Fatalf("list after delete failed: %v", err)
	}
	if strings.Contains(listRes, "to-delete") {
		t.Errorf("Branch to-delete should not exist, got: %s", listRes)
	}
}

func TestBranchCommand_Move(t *testing.T) {
	sm := git.NewSessionManager()
	s := setupBranchTestSession(t, sm, "test-branch-move")
	ctx := context.Background()

	cmd := &BranchCommand{}

	// Create a branch to rename
	_, err := cmd.Execute(ctx, s, []string{"branch", "old-name"})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}

	// Rename it
	res, err := cmd.Execute(ctx, s, []string{"branch", "-m", "old-name", "new-name"})
	if err != nil {
		t.Fatalf("move failed: %v", err)
	}
	if !strings.Contains(res, "Renamed branch old-name to new-name") {
		t.Errorf("Expected rename message, got: %s", res)
	}

	// Verify old name is gone and new name exists
	listRes, err := cmd.Execute(ctx, s, []string{"branch"})
	if err != nil {
		t.Fatalf("list after move failed: %v", err)
	}
	if strings.Contains(listRes, "old-name") {
		t.Errorf("old-name should not exist, got: %s", listRes)
	}
	if !strings.Contains(listRes, "new-name") {
		t.Errorf("new-name should exist, got: %s", listRes)
	}
}

func TestBranchCommand_CreateWithStartPoint(t *testing.T) {
	sm := git.NewSessionManager()
	s := setupBranchTestSession(t, sm, "test-branch-startpoint")
	ctx := context.Background()
	cmd := &BranchCommand{}

	// 1. Get HEAD object
	repo := s.GetRepo()
	head, _ := repo.Head()
	headHash := head.Hash().String()

	// 2. Create branch from HEAD explicit
	_, err := cmd.Execute(ctx, s, []string{"branch", "from-head", headHash})
	if err != nil {
		t.Fatalf("create with hash failed: %v", err)
	}

	// 3. Create branch from previous branch
	_, err = cmd.Execute(ctx, s, []string{"branch", "from-branch", "from-head"})
	if err != nil {
		t.Fatalf("create with branch name failed: %v", err)
	}

	// 4. Create branch from unknown (should fail)
	_, err = cmd.Execute(ctx, s, []string{"branch", "bad", "unknown-ref"})
	if err == nil {
		t.Fatal("expected error creating from unknown ref, got nil")
	}
}

func TestBranchCommand_DeleteSafety(t *testing.T) {
	sm := git.NewSessionManager()
	s := setupBranchTestSession(t, sm, "test-branch-safety")
	ctx := context.Background()
	cmd := &BranchCommand{}

	// 1. Create a divergent branch
	// Get current branch name (default)
	repo := s.GetRepo()
	head, _ := repo.Head()
	defaultBranch := head.Name().Short()

	// switch to new branch, commit, switch back
	switchCmd := &SwitchCommand{}
	_, _ = switchCmd.Execute(ctx, s, []string{"switch", "-c", "divergent"})

	touchCmd := &TouchCommand{}
	_, _ = touchCmd.Execute(ctx, s, []string{"touch", "divergent.txt"})

	addCmd := &AddCommand{}
	_, _ = addCmd.Execute(ctx, s, []string{"add", "."})

	commitCmd := &CommitCommand{}
	_, _ = commitCmd.Execute(ctx, s, []string{"commit", "-m", "Divergent commit"})

	// Switch back to master
	_, err := switchCmd.Execute(ctx, s, []string{"switch", defaultBranch})
	if err != nil {
		t.Fatalf("failed to switch back to master: %v", err)
	}
	// master does not have "Divergent commit"

	// 2. Try delete divergent with -d (should fail)
	_, err = cmd.Execute(ctx, s, []string{"branch", "-d", "divergent"})
	if err == nil {
		t.Fatal("expected error deleting unmerged branch, got nil")
	}
	if !strings.Contains(err.Error(), "not fully merged") {
		t.Errorf("expected 'not fully merged' error, got: %v", err)
	}

	// 3. Try delete divergent with -D (should succeed)
	res, err := cmd.Execute(ctx, s, []string{"branch", "-D", "divergent"})
	if err != nil {
		t.Fatalf("force delete failed: %v", err)
	}
	if !strings.Contains(res, "Deleted branch divergent") {
		t.Errorf("Expected deletion message, got: %s", res)
	}
}
