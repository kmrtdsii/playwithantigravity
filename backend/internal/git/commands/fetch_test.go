package commands

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
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
