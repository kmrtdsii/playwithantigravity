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

func TestBlameBasic(t *testing.T) {
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	author := &object.Signature{Name: "Tester", Email: "test@example.com", When: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)}

	// Commit 1: Add file
	f, _ := fs.Create("file.txt")
	f.Write([]byte("line1\nline2\n"))
	f.Close()
	w.Add("file.txt")
	c1, _ := w.Commit("Initial", &gogit.CommitOptions{Author: author})

	// Commit 2: Modify line 2
	f, _ = fs.Create("file.txt")
	f.Write([]byte("line1\nline2 modified\n"))
	f.Close()
	w.Add("file.txt")
	author2 := &object.Signature{Name: "Modifier", Email: "mod@example.com", When: time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)}
	c2, _ := w.Commit("Modify line 2", &gogit.CommitOptions{Author: author2})

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}
	cmd := &BlameCommand{}

	output, err := cmd.Execute(context.Background(), session, []string{"blame", "file.txt"})
	assert.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 2, len(lines))

	// Line 1: From c1, Author Tester
	assert.Contains(t, lines[0], c1.String()[:8])
	assert.Contains(t, lines[0], "test@example.com")
	assert.Contains(t, lines[0], "line1")

	// Line 2: From c2, Author Modifier
	assert.Contains(t, lines[1], c2.String()[:8])
	assert.Contains(t, lines[1], "mod@example.com")
	assert.Contains(t, lines[1], "line2 modified")
}
