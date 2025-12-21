package main

import (
	"strings"
	"testing"
)

func TestGitBranchDelete(t *testing.T) {
	sessionID := "test-session"
	if err := InitSession(sessionID); err != nil {
		t.Fatalf("Failed to init session: %v", err)
	}

	// 1. Init repo
	if _, err := ExecuteGitCommand(sessionID, []string{"init"}); err != nil {
		t.Fatalf("Failed to init git: %v", err)
	}

	// 2. Need a commit to create a branch
	if err := TouchFile(sessionID, "README.md"); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	if _, err := ExecuteGitCommand(sessionID, []string{"add", "README.md"}); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}
	if _, err := ExecuteGitCommand(sessionID, []string{"commit", "-m", "Initial commit"}); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// 3. Create a branch
	branchName := "feature-branch"
	if _, err := ExecuteGitCommand(sessionID, []string{"branch", branchName}); err != nil {
		t.Fatalf("Failed to create branch: %v", err)
	}

	// 4. Verify branch exists
	output, err := ExecuteGitCommand(sessionID, []string{"branch"})
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}
	if !strings.Contains(output, branchName) {
		t.Errorf("Expected branch list to contain %s, got:\n%s", branchName, output)
	}

	// 5. Delete branch
	if _, err := ExecuteGitCommand(sessionID, []string{"branch", "-d", branchName}); err != nil {
		t.Fatalf("Failed to delete branch: %v", err)
	}

	// 6. Verify branch is gone
	output, err = ExecuteGitCommand(sessionID, []string{"branch"})
	if err != nil {
		t.Fatalf("Failed to list branches after delete: %v", err)
	}
	if strings.Contains(output, branchName) {
		t.Errorf("Expected branch list to NOT contain %s, got:\n%s", branchName, output)
	}

	// 7. Test deleting non-existent branch
	if _, err := ExecuteGitCommand(sessionID, []string{"branch", "-d", "non-existent"}); err == nil {
		t.Error("Expected error when deleting non-existent branch, got nil")
	}

	// 8. Test invalid flag (prevent creation of branch named "-r")
	if _, err := ExecuteGitCommand(sessionID, []string{"branch", "-r"}); err == nil {
		t.Error("Expected error when using invalid flag -r, got nil")
	}
	output, err = ExecuteGitCommand(sessionID, []string{"branch"})
	if err != nil {
		t.Fatalf("Failed to list branches: %v", err)
	}
	if strings.Contains(output, "-r") {
		t.Errorf("Expected branch list to NOT contain '-r', got:\n%s", output)
	}
}
