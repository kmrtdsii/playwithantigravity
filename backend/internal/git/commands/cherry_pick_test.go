package commands

import (
	"context"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kurobon/gitgym/backend/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestCherryPickRange(t *testing.T) {
	// Setup repo
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	// 1. Initial Commit (Base) - Master
	fs.Create("base.txt")
	w.Add("base.txt")
	baseHash, _ := w.Commit("Base", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "user@test.com", When: time.Now()},
	})

	// 2. Commit A
	fs.Create("a.txt")
	w.Add("a.txt")
	aHash, _ := w.Commit("Commit A", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "user@test.com", When: time.Now()},
	})

	// 3. Commit B
	fs.Create("b.txt")
	w.Add("b.txt")
	bHash, _ := w.Commit("Commit B", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "user@test.com", When: time.Now()},
	})
	_ = bHash

	// 4. Commit C
	fs.Create("c.txt")
	w.Add("c.txt")
	cHash, _ := w.Commit("Commit C", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "user@test.com", When: time.Now()},
	})

	// Now we have Base -> A -> B -> C
	// Switch to a new branch 'target' at Base
	w.Checkout(&gogit.CheckoutOptions{Hash: baseHash, Force: true})
	w.Checkout(&gogit.CheckoutOptions{Branch: plumbing.ReferenceName("refs/heads/target"), Create: true, Force: true})

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}
	cmd := &CherryPickCommand{}

	// git cherry-pick A..C
	// Range: (A, C] -> Should include B and C.
	// Input uses commit hashes converted to string.
	// For range, A is start, C is end.
	rangeArg := aHash.String() + ".." + cHash.String()

	output, err := cmd.Execute(context.Background(), session, []string{"cherry-pick", rangeArg})
	assert.NoError(t, err)
	assert.Contains(t, output, "Picked 2 commits")

	// Verify HEAD
	head, _ := r.Head()
	headCommit, _ := r.CommitObject(head.Hash())
	assert.Equal(t, "Commit C", headCommit.Message) // Copied message

	// Parent of C' should be B'
	parent, _ := headCommit.Parent(0)
	assert.Equal(t, "Commit B", parent.Message)

	// Parent of B' should be Base
	grandParent, _ := parent.Parent(0)
	assert.Equal(t, baseHash, grandParent.Hash)

	// Verify Content
	// Should have base.txt, b.txt, c.txt
	// NOT a.txt (since A was excluded from range)
	_, err = fs.Stat("base.txt")
	assert.NoError(t, err)
	_, err = fs.Stat("b.txt")
	assert.NoError(t, err)
	_, err = fs.Stat("c.txt")
	assert.NoError(t, err)

	_, err = fs.Stat("a.txt")
	assert.Error(t, err, "a.txt should not be present")
}

func TestCherryPickMultiArg(t *testing.T) {
	// Setup repo
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	// 1. Initial Commit
	fs.Create("base.txt")
	w.Add("base.txt")
	baseHash, _ := w.Commit("Base", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "u@t.com", When: time.Now()},
	})

	// 2. Commit A
	fs.Create("a.txt")
	w.Add("a.txt")
	aHash, _ := w.Commit("Commit A", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "u@t.com", When: time.Now()},
	})

	// 3. Commit B
	w.Checkout(&gogit.CheckoutOptions{Hash: baseHash, Force: true})
	fs.Create("b.txt")
	w.Add("b.txt")
	bHash, _ := w.Commit("Commit B", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "u@t.com", When: time.Now()},
	})
	_ = bHash

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}
	cmd := &CherryPickCommand{}

	// git cherry-pick A B
	// Should pick both onto current (B) -> Result: Base + B + A + B copy? No wait.
	// Current is B.
	// Pick A: Base+B+A
	// Pick B: Base+B+A+B (empty?)
	// Actually B is already in history? No, we are on B.
	// Wait, picking B onto B?
	// B is "Theirs". Ours is B. Base is Base.
	// Ours==Theirs -> No-op?
	// Let's pick A and then create C on master and pick that.

	// Real Scenario:
	// Master: Base -> A
	// Feature: Base -> B
	// Cherry pick A onto Feature.

	// Reset to B
	w.Checkout(&gogit.CheckoutOptions{Hash: bHash, Force: true})

	// Pick A
	_, err := cmd.Execute(context.Background(), session, []string{"cherry-pick", aHash.String()})
	assert.NoError(t, err)

	// Check for a.txt (from A) and b.txt (from B)
	_, err = fs.Stat("a.txt")
	assert.NoError(t, err)
	_, err = fs.Stat("b.txt")
	assert.NoError(t, err)
}

func TestCherryPickConflict(t *testing.T) {
	// Setup repo
	fs := memfs.New()
	storer := memory.NewStorage()
	r, _ := gogit.Init(storer, fs)
	w, _ := r.Worktree()

	// 1. Initial Commit
	utilFile := func(name, content string) {
		f, _ := fs.Create(name)
		f.Write([]byte(content))
		f.Close()
		w.Add(name)
	}

	utilFile("file.txt", "base\n")
	baseHash, _ := w.Commit("Base", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "u@t.com", When: time.Now()},
	})

	// 2. Commit A (Change file.txt)
	utilFile("file.txt", "base\nchangeA\n")
	aHash, _ := w.Commit("Commit A", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "u@t.com", When: time.Now()},
	})

	// 3. Commit B (Change file.txt differently)
	w.Checkout(&gogit.CheckoutOptions{Hash: baseHash, Force: true})
	utilFile("file.txt", "base\nchangeB\n")
	_, _ = w.Commit("Commit B", &gogit.CommitOptions{
		Author: &object.Signature{Name: "User", Email: "u@t.com", When: time.Now()},
	})

	// Current is B. Cherry-pick A.
	// Base: "base\n"
	// Ours: "base\nchangeB\n"
	// Theirs: "base\nchangeA\n"
	// Conflict!

	session := &git.Session{
		ID:         "test-session",
		Filesystem: fs,
		Repos:      map[string]*gogit.Repository{"repo": r},
		CurrentDir: "/repo",
	}
	cmd := &CherryPickCommand{}

	output, err := cmd.Execute(context.Background(), session, []string{"cherry-pick", aHash.String()})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error: could not apply")
	assert.Contains(t, output, "") // Output is empty on error usually, or partial? wrapper returns err.

	// Check Content
	f, _ := fs.Open("file.txt")
	content := make([]byte, 100)
	n, _ := f.Read(content)
	sContent := string(content[:n])
	assert.Contains(t, sContent, "<<<<<<< HEAD")
	assert.Contains(t, sContent, "changeB")
	assert.Contains(t, sContent, "=======")
	assert.Contains(t, sContent, "changeA")
	assert.Contains(t, sContent, ">>>>>>>")
}
