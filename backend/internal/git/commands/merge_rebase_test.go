package commands

import (
	"context"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestMergeCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-merge")
	initCmd := &InitCommand{}
	initCmd.Execute(context.Background(), s, []string{"init"})
	repo := s.GetRepo()
	w, _ := repo.Worktree()

	// Base commit
	f, _ := w.Filesystem.Create("base.txt")
	f.Write([]byte("base"))
	f.Close()
	w.Add("base.txt")
	w.Commit("base", &gogit.CommitOptions{Author: &object.Signature{Name: "User", When: time.Now()}})

	// Create branch feature
	w.Checkout(&gogit.CheckoutOptions{Branch: "refs/heads/feature", Create: true})
	f, _ = w.Filesystem.Create("feature.txt")
	f.Write([]byte("feature"))
	f.Close()
	w.Add("feature.txt")
	w.Commit("feature", &gogit.CommitOptions{Author: &object.Signature{Name: "User", When: time.Now()}})

	// Switch back to master
	w.Checkout(&gogit.CheckoutOptions{Branch: "refs/heads/master"})

	// Merge feature
	cmd := &MergeCommand{}
	res, err := cmd.Execute(context.Background(), s, []string{"merge", "feature"})
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if !strings.Contains(res, "Fast-forward") && !strings.Contains(res, "Merge made") {
		t.Errorf("Unexpected merge result: %s", res)
	}

	// Check content
	_, err = w.Filesystem.Stat("feature.txt")
	if err != nil {
		t.Error("feature.txt should exist after merge")
	}
}

func TestRebaseCommand(t *testing.T) {
	sm := git.NewSessionManager()
	s, _ := sm.CreateSession("test-rebase")
	initCmd := &InitCommand{}
	initCmd.Execute(context.Background(), s, []string{"init"})
	repo := s.GetRepo()
	w, _ := repo.Worktree()

	// Base commit
	f, _ := w.Filesystem.Create("base.txt")
	f.Write([]byte("base"))
	f.Close()
	w.Add("base.txt")
	w.Commit("base", &gogit.CommitOptions{Author: &object.Signature{Name: "User", When: time.Now()}})

	// Create branch feature
	w.Checkout(&gogit.CheckoutOptions{Branch: "refs/heads/feature", Create: true})
	f, _ = w.Filesystem.Create("feature.txt")
	f.Write([]byte("feature"))
	f.Close()
	w.Add("feature.txt")
	w.Commit("feature", &gogit.CommitOptions{Author: &object.Signature{Name: "User", When: time.Now()}})

	// Switch back to master and advance it
	w.Checkout(&gogit.CheckoutOptions{Branch: "refs/heads/master"})
	f, _ = w.Filesystem.Create("master.txt")
	f.Write([]byte("master"))
	f.Close()
	w.Add("master.txt")
	w.Commit("master", &gogit.CommitOptions{Author: &object.Signature{Name: "User", When: time.Now()}})

	// Rebase feature onto master
	w.Checkout(&gogit.CheckoutOptions{Branch: "refs/heads/feature"})

	cmd := &RebaseCommand{}
	res, err := cmd.Execute(context.Background(), s, []string{"rebase", "master"})
	if err != nil {
		t.Fatalf("Rebase failed: %v", err)
	}

	if !strings.Contains(res, "Successfully rebased") {
		t.Errorf("Unexpected rebase result: %s", res)
	}

	// Validate log: master -> feature
	head, _ := repo.Head()
	c, _ := repo.CommitObject(head.Hash())
	if c.Message != "feature" {
		t.Errorf("Expected HEAD message 'feature', got '%s'", c.Message)
	}

	parent, _ := c.Parent(0)
	if parent.Message != "master" {
		t.Errorf("Expected parent message 'master', got '%s'", parent.Message)
	}
}
