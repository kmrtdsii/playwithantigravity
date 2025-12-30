package commands

import (
	"context"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestRevertClean(t *testing.T) {
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	// 1. Base Commit
	f, _ := fs.Create("file.txt")
	f.Write([]byte("base\n"))
	f.Close()
	w.Add("file.txt")
	w.Commit("Base", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "u@t.com", When: time.Now()},
	})

	// 2. Commit to Revert
	f, _ = fs.Create("file.txt")
	f.Write([]byte("base\nchange\n"))
	f.Close()
	w.Add("file.txt")
	cHash, _ := w.Commit("Bad Commit", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "u@t.com", When: time.Now()},
	})

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}
	cmd := &RevertCommand{}

	// Revert HEAD
	output, err := cmd.Execute(context.Background(), session, []string{"revert", "HEAD"})
	assert.NoError(t, err)
	assert.Contains(t, output, "Revert successful")

	// Verify Content (Should be back to "base\n")
	f, _ = fs.Open("file.txt")
	content := make([]byte, 100)
	n, _ := f.Read(content)
	sContent := string(content[:n])
	assert.Equal(t, "base\n", sContent)

	// Verify Log
	head, _ := r.Head()
	commit, _ := r.CommitObject(head.Hash())
	assert.Contains(t, commit.Message, "Revert \"Bad Commit\"")
	assert.Contains(t, commit.Message, cHash.String())
}
