package commands

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestStash(t *testing.T) {
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	// 1. Initial Commit
	f, _ := fs.Create("file.txt")
	f.Write([]byte("base\n"))
	f.Close()
	w.Add("file.txt")

	author := &object.Signature{Name: "Tester", Email: "test@example.com", When: time.Now()}
	w.Commit("Base", &gogit.CommitOptions{Author: author})

	// 2. Make it dirty
	f, _ = fs.Open("file.txt")
	// wait, open truncates? No.
	f, _ = fs.Create("file.txt")
	f.Write([]byte("base\ndirty\n"))
	f.Close()

	status, _ := w.Status()
	assert.False(t, status.IsClean())

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}
	cmd := &StashCommand{}

	// 3. Stash Push
	output, err := cmd.Execute(context.Background(), session, []string{"stash", "push"})
	assert.NoError(t, err)
	assert.Contains(t, output, "Saved working directory")

	// Verify Clean
	status, _ = w.Status()
	assert.True(t, status.IsClean())

	f, _ = fs.Open("file.txt")
	content := make([]byte, 100)
	n, _ := f.Read(content)
	assert.Equal(t, "base\n", string(content[:n]))

	// 4. Stash List
	output, err = cmd.Execute(context.Background(), session, []string{"stash", "list"})
	assert.NoError(t, err)
	assert.Contains(t, output, "stash@{0}")

	// 5. Stash Pop
	output, err = cmd.Execute(context.Background(), session, []string{"stash", "pop"})
	assert.NoError(t, err)
	assert.Contains(t, output, "Dropped refs/stash")

	// Verify Dirty Again
	status, _ = w.Status()
	assert.False(t, status.IsClean())

	f, _ = fs.Open("file.txt")
	n, _ = f.Read(content)
	assert.Equal(t, "base\ndirty\n", string(content[:n]))

	// Verify Stash is Gone
	output, err = cmd.Execute(context.Background(), session, []string{"stash", "list"})
	assert.NoError(t, err)
	assert.Equal(t, "", strings.TrimSpace(output))
}

func TestStashStack(t *testing.T) {
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	_, _ = fs.Create("a")
	w.Add("a")

	author := &object.Signature{Name: "Tester", Email: "test@example.com", When: time.Now()}
	w.Commit("Base", &gogit.CommitOptions{Author: author})

	session := &git.Session{
		ID:         "t",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}
	cmd := &StashCommand{}

	// Push 1
	f, _ := fs.Create("a")
	f.Write([]byte("1"))
	f.Close()
	cmd.Execute(context.Background(), session, []string{"stash"})

	// Push 2
	f, _ = fs.Create("a")
	f.Write([]byte("2"))
	f.Close()
	cmd.Execute(context.Background(), session, []string{"stash"})

	// List
	output, _ := cmd.Execute(context.Background(), session, []string{"stash", "list"})
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 2, len(lines))
	assert.Contains(t, lines[0], "stash@{0}")
	assert.Contains(t, lines[1], "stash@{1}")

	// Pop (Should get "2")
	cmd.Execute(context.Background(), session, []string{"stash", "pop"})
	f, _ = fs.Open("a")
	b := make([]byte, 10)
	n, _ := f.Read(b)
	assert.Equal(t, "2", string(b[:n]))

	// List (Should have 1 left)
	output, _ = cmd.Execute(context.Background(), session, []string{"stash", "list"})
	assert.Contains(t, output, "stash@{0}")
	assert.NotContains(t, output, "stash@{1}")
}
