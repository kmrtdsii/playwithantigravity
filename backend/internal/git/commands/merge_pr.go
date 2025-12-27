package commands

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("merge-pr", func() git.Command { return &MergePRCommand{} })
}

type MergePRCommand struct {
	prID       int
	remoteName string

	pr     *git.PullRequest
	repo   *gogit.Repository
	engine *git.Session
}

func (c *MergePRCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	c.engine = s

	if err := c.parseArgs(args); err != nil {
		return "", err
	}

	if err := c.resolveContext(ctx); err != nil {
		return "", err
	}

	return c.performAction(ctx)
}

func (c *MergePRCommand) parseArgs(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: merge-pr <pr-id> <remote-name>")
	}

	prID, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Errorf("invalid PR ID %q: %w", args[1], err)
	}

	c.prID = prID
	c.remoteName = args[2]
	return nil
}

func (c *MergePRCommand) resolveContext(_ context.Context) error {
	sm := c.engine.Manager
	sm.RLock()
	defer sm.RUnlock()

	// 1. Find Pull Request
	var foundPR *git.PullRequest
	for _, p := range sm.PullRequests {
		if p.ID == c.prID {
			foundPR = p
			break
		}
	}
	if foundPR == nil {
		return fmt.Errorf("pull request #%d not found", c.prID)
	}

	if foundPR.State != "OPEN" {
		return fmt.Errorf("pull request #%d is not OPEN (current state: %s)", c.prID, foundPR.State)
	}
	c.pr = foundPR

	// 2. Resolve Remote Repository
	repo, ok := sm.SharedRemotes[c.remoteName]
	if !ok {
		return fmt.Errorf("remote repository %q not found", c.remoteName)
	}
	c.repo = repo

	return nil
}

func (c *MergePRCommand) performAction(_ context.Context) (string, error) {
	log.Printf("MergePRCommand: Merging PR #%d (%s -> %s) on remote %q", c.prID, c.pr.HeadRef, c.pr.BaseRef, c.remoteName)

	// Resolve references
	baseRefName := plumbing.ReferenceName("refs/heads/" + c.pr.BaseRef)
	headRefName := plumbing.ReferenceName("refs/heads/" + c.pr.HeadRef)

	log.Printf("MergePRCommand: Resolving base ref: %s", baseRefName)
	baseRef, err := c.repo.Reference(baseRefName, true)
	if err != nil {
		log.Printf("MergePRCommand: Base branch %q NOT found: %v", c.pr.BaseRef, err)
		return "", fmt.Errorf("base branch %q not found in remote: %w", c.pr.BaseRef, err)
	}

	log.Printf("MergePRCommand: Resolving source ref: %s", headRefName)
	headRef, err := c.repo.Reference(headRefName, true)
	if err != nil {
		log.Printf("MergePRCommand: Source branch %q NOT found: %v", c.pr.HeadRef, err)
		return "", fmt.Errorf("source branch %q not found in remote: %w", c.pr.HeadRef, err)
	}

	log.Printf("MergePRCommand: Found base %s and head %s", baseRef.Hash(), headRef.Hash())

	// 1. Get Base Commit
	baseCommit, err := c.repo.CommitObject(baseRef.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to retrieve base commit %s: %w", baseRef.Hash(), err)
	}
	// 2. Get Head Commit
	headCommit, err := c.repo.CommitObject(headRef.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to retrieve source commit %s: %w", headRef.Hash(), err)
	}

	// 3. Create Merge Commit using "Theirs" tree (Head's tree snapshot)
	mergeCommit := &object.Commit{
		Author: object.Signature{
			Name:  "GitGym Merge Bot",
			Email: "bot@gitgym.com",
			When:  time.Now(),
		},
		Committer: object.Signature{
			Name:  "GitGym Merge Bot",
			Email: "bot@gitgym.com",
			When:  time.Now(),
		},
		Message:  fmt.Sprintf("Merge pull request #%d from %s\n\n%s", c.prID, c.pr.HeadRef, c.pr.Title),
		TreeHash: headCommit.TreeHash,
		ParentHashes: []plumbing.Hash{
			baseCommit.Hash,
			headCommit.Hash,
		},
	}

	obj := c.repo.Storer.NewEncodedObject()
	if err := mergeCommit.Encode(obj); err != nil {
		return "", fmt.Errorf("failed to encode merge commit: %w", err)
	}

	newHash, err := c.repo.Storer.SetEncodedObject(obj)
	if err != nil {
		return "", fmt.Errorf("failed to store merge commit: %w", err)
	}

	// 4. Update Remote Reference
	log.Printf("MergePRCommand: Updating %s to %s", baseRefName, newHash)
	newRef := plumbing.NewHashReference(baseRefName, newHash)
	if err := c.repo.Storer.SetReference(newRef); err != nil {
		return "", fmt.Errorf("failed to update remote branch %q: %w", c.pr.BaseRef, err)
	}

	// 5. Update PR State
	c.engine.Manager.Lock()
	c.pr.State = "MERGED"
	c.engine.Manager.Unlock()

	log.Printf("MergePRCommand: PR #%d merged successfully", c.prID)
	return fmt.Sprintf("Successfully merged PR #%d into %s", c.prID, c.pr.BaseRef), nil
}

func (c *MergePRCommand) Help() string {
	return "usage: merge-pr <pr-id> <remote-name>"
}
