package commands

import (
	"context"
	"strings"
	"testing"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestBranchCommand_ForceCreate(t *testing.T) {
	sm := git.NewSessionManager()
	s := setupBranchTestSession(t, sm, "test-branch-force")
	ctx := context.Background()
	cmd := &BranchCommand{}

	repo := s.GetRepo()
	head, _ := repo.Head()
	headHash := head.Hash().String()

	// Helper to create a commit and return hash
	createCommit := func(msg string) string {
		tCmd := &TouchCommand{}
		tCmd.Execute(ctx, s, []string{"touch", msg})
		aCmd := &AddCommand{}
		aCmd.Execute(ctx, s, []string{"add", "."})
		cCmd := &CommitCommand{}
		cCmd.Execute(ctx, s, []string{"commit", "-m", msg})
		h, _ := repo.Head()
		return h.Hash().String()
	}

	secondCommitHash := createCommit("Second commit")

	// 1. git branch branchA (create from HEAD/default)
	_, err := cmd.Execute(ctx, s, []string{"branch", "branchA"})
	if err != nil {
		t.Fatalf("1. Create branchA failed: %v", err)
	}
	// Verify branchA -> secondCommitHash
	refA, _ := repo.Reference(plumbing.ReferenceName("refs/heads/branchA"), true)
	if refA.Hash().String() != secondCommitHash {
		t.Errorf("branchA should point to %s, got %s", secondCommitHash, refA.Hash().String())
	}

	// 2. git branch -f branchA (reset from HEAD)
	// First, let's move HEAD to make it interesting, or just verify it passes.
	// Actually, let's reset branchA to headHash (initial commit) using -f
	// But command is: git branch -f branchA  (implies HEAD)
	// So if we are at secondCommitHash, it just resets to same.
	// Let's create branchB at initial commit first
	cmd.Execute(ctx, s, []string{"branch", "branchB", headHash})
	// Now force branchB to HEAD (secondCommitHash)
	_, err = cmd.Execute(ctx, s, []string{"branch", "-f", "branchB"})
	if err != nil {
		t.Fatalf("2. Force branchB to HEAD failed: %v", err)
	}
	refB, _ := repo.Reference(plumbing.ReferenceName("refs/heads/branchB"), true)
	if refB.Hash().String() != secondCommitHash {
		t.Errorf("branchB should moved to %s, got %s", secondCommitHash, refB.Hash().String())
	}

	// 2b. Fail without -f
	_, err = cmd.Execute(ctx, s, []string{"branch", "branchA"}) // Exists
	if err == nil {
		t.Errorf("Expected checking existing branch to fail without -f")
	} else if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Expected 'already exists' error, got: %v", err)
	}

	// 3. git branch branchC branchA (create from branchA)
	_, err = cmd.Execute(ctx, s, []string{"branch", "branchC", "branchA"})
	if err != nil {
		t.Fatalf("3. Create branchC from branchA failed: %v", err)
	}
	refC, _ := repo.Reference(plumbing.ReferenceName("refs/heads/branchC"), true)
	if refC.Hash().String() != secondCommitHash {
		t.Errorf("branchC should point to %s, got %s", secondCommitHash, refC.Hash().String())
	}

	// 4. git branch branchD [commit hash]
	_, err = cmd.Execute(ctx, s, []string{"branch", "branchD", headHash})
	if err != nil {
		t.Fatalf("4. Create branchD from hash failed: %v", err)
	}
	refD, _ := repo.Reference(plumbing.ReferenceName("refs/heads/branchD"), true)
	if refD.Hash().String() != headHash {
		t.Errorf("branchD should point to %s, got %s", headHash, refD.Hash().String())
	}

	// 5. git branch -f branchA branchD (force reset branchA to branchD's commit)
	_, err = cmd.Execute(ctx, s, []string{"branch", "-f", "branchA", "branchD"})
	if err != nil {
		t.Fatalf("5. Force branchA to branchD failed: %v", err)
	}
	refA, _ = repo.Reference(plumbing.ReferenceName("refs/heads/branchA"), true)
	if refA.Hash().String() != headHash {
		t.Errorf("branchA should now point to %s, got %s", headHash, refA.Hash().String())
	}

	// 6. git branch -f branchA [commit hash] (force reset branchA to secondCommitHash)
	_, err = cmd.Execute(ctx, s, []string{"branch", "-f", "branchA", secondCommitHash})
	if err != nil {
		t.Fatalf("6. Force branchA to hash failed: %v", err)
	}
	refA, _ = repo.Reference(plumbing.ReferenceName("refs/heads/branchA"), true)
	if refA.Hash().String() != secondCommitHash {
		t.Errorf("branchA should now point to %s, got %s", secondCommitHash, refA.Hash().String())
	}

	// Extra: Ensure current branch protection
	// We are on default branch (likely master/main). Check name.
	currHead, _ := repo.Head()
	currName := currHead.Name().Short()
	_, err = cmd.Execute(ctx, s, []string{"branch", "-f", currName, headHash})
	if err == nil {
		t.Errorf("Expected current branch protection error, got nil")
	} else if !strings.Contains(err.Error(), "current branch") {
		t.Errorf("Expected 'current branch' error, got: %v", err)
	}
}
