package commands

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5/memfs"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func TestFetchCommand_Reproduction(t *testing.T) {
	// 1. Setup
	tempDir := t.TempDir()
	dataDir := filepath.Join(tempDir, "data")
	sm := git.NewSessionManager()
	sm.DataDir = dataDir

	// 2. Create Remote
	remotePath := filepath.Join(tempDir, "remote")
	r, _ := gogit.PlainInit(remotePath, false)
	w, _ := r.Worktree()
	w.Filesystem.Create("README.md")
	w.Add("README.md")
	w.Commit("Init", &gogit.CommitOptions{
		Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
	})

	// 3. Ingest
	err := sm.IngestRemote(context.Background(), "origin", remotePath)
	if err != nil {
		t.Fatal(err)
	}

	session, _ := sm.CreateSession("test-session")

	// 4. Clone
	cloneCmd := &CloneCommand{}
	_, err = cloneCmd.Execute(context.Background(), session, []string{"clone", remotePath})
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// 5. Fetch (Should Success now)
	fetchCmd := &FetchCommand{}
	// "remote" is the repo name derived from path
	// clone auto-cds into "remote" directory
	// origin point to internal data dir path
	output, err := fetchCmd.Execute(context.Background(), session, []string{"fetch", "origin"})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	t.Logf("Fetch success: %s", output)
}

func TestFetchCommand_All(t *testing.T) {
	// 1. Setup
	sm := git.NewSessionManager()
	sm.DataDir = t.TempDir()

	createInMemoryRepo := func() *gogit.Repository {
		fs := memfs.New()
		st := memory.NewStorage()
		r, _ := gogit.Init(st, fs)

		// Base commit
		w, _ := r.Worktree()
		f, _ := w.Filesystem.Create("base.txt")
		f.Close()
		w.Add("base.txt")
		w.Commit("Base", &gogit.CommitOptions{
			Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
		})
		return r
	}

	originRepo := createInMemoryRepo()
	upstreamRepo := createInMemoryRepo()

	originURL := "https://example.com/origin.git"
	upstreamURL := "https://example.com/upstream.git"

	// 2. Inject into SharedRemotes
	sm.Lock()
	sm.SharedRemotes[originURL] = originRepo
	sm.SharedRemotes[upstreamURL] = upstreamRepo
	sm.Unlock()

	session, _ := sm.CreateSession("test-fetch-all")

	// 3. Clone origin
	cloneCmd := &CloneCommand{}
	_, err := cloneCmd.Execute(context.Background(), session, []string{"clone", originURL})
	if err != nil {
		t.Fatalf("Clone failed: %v", err)
	}

	// 4. Add upstream remote manually
	repo := session.GetRepo()
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "upstream",
		URLs: []string{upstreamURL},
	})
	if err != nil {
		t.Fatalf("Failed to add upstream remote: %v", err)
	}

	// 5. Update both remotes to ensure they produce output
	updateRepo := func(r *gogit.Repository, filename string) {
		w, _ := r.Worktree()
		f, _ := w.Filesystem.Create(filename)
		f.Close()
		w.Add(filename)
		w.Commit("Update "+filename, &gogit.CommitOptions{
			Author: &object.Signature{Name: "Dev", Email: "dev@example.com", When: time.Now()},
		})
	}
	updateRepo(originRepo, "origin_update.txt")
	updateRepo(upstreamRepo, "upstream_update.txt")

	// 6. Fetch --all
	fetchCmd := &FetchCommand{}
	output, err := fetchCmd.Execute(context.Background(), session, []string{"fetch", "--all"})
	if err != nil {
		t.Fatalf("Fetch --all failed: %v", err)
	}
	t.Logf("Fetch --all output: %s", output)

	// 7. Verify Output
	if !strings.Contains(output, "From "+originURL) {
		t.Errorf("Output missing origin fetch")
	}
	if !strings.Contains(output, "From "+upstreamURL) {
		t.Errorf("Output missing upstream fetch")
	}

	// 8. Verify Refs
	_, err = repo.Reference("refs/remotes/origin/master", true)
	if err != nil {
		t.Errorf("Missing origin/master ref")
	}
	_, err = repo.Reference("refs/remotes/upstream/master", true)
	if err != nil {
		t.Errorf("Missing upstream/master ref")
	}
}
