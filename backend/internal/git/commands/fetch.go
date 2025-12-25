package commands

// fetch.go - Simulated Git Fetch Command
//
// IMPORTANT: This implementation does NOT perform actual network operations.
// It copies objects from in-memory virtual remotes (SharedRemotes or session-local).

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("fetch", func() git.Command { return &FetchCommand{} })
}

type FetchCommand struct{}

func (c *FetchCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// Parse Flags
	isDryRun := false
	fetchAll := false
	var positionalArgs []string

	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch arg {
		case "-n", "--dry-run":
			isDryRun = true
		case "--all":
			fetchAll = true
		case "-h", "--help":
			return c.Help(), nil
		default:
			if strings.HasPrefix(arg, "-") {
				// Unknown flag
				return "", fmt.Errorf("unknown flag: %s", arg)
			}
			positionalArgs = append(positionalArgs, arg)
		}
	}

	var remotes []*gogit.Remote
	var err error

	if fetchAll {
		remotes, err = repo.Remotes()
		if err != nil {
			return "", fmt.Errorf("failed to list remotes: %w", err)
		}
	} else {
		// Single remote
		remoteName := "origin"
		if len(positionalArgs) > 0 {
			remoteName = positionalArgs[0]
		}
		rem, err := repo.Remote(remoteName)
		if err != nil {
			return "", fmt.Errorf("fatal: '%s' does not appear to be a git repository", remoteName)
		}
		remotes = []*gogit.Remote{rem}
	}

	var allResults []string

	for _, rem := range remotes {
		res, err := c.fetchRemote(s, repo, rem, isDryRun)
		if err != nil {
			// In git fetch --all, one failure usually doesn't stop others, but returns non-zero.
			// We will just log error in results and continue?
			// Or fail immediately. Git usually continues but reports.
			allResults = append(allResults, fmt.Sprintf("error: fetching %s: %v", rem.Config().Name, err))
		} else {
			if res != "" {
				allResults = append(allResults, res)
			}
		}
	}

	if len(allResults) == 0 {
		return "Already up to date.", nil
	}

	return strings.Join(allResults, "\n"), nil
}

func (c *FetchCommand) fetchRemote(s *git.Session, repo *gogit.Repository, rem *gogit.Remote, isDryRun bool) (string, error) {
	cfg := rem.Config()
	remoteName := cfg.Name
	if len(cfg.URLs) == 0 {
		return "", fmt.Errorf("remote %s has no URL defined", remoteName)
	}
	url := cfg.URLs[0]

	// Look up simulated remote
	lookupKey := strings.TrimPrefix(url, "/")

	var srcRepo *gogit.Repository
	var ok bool

	// Check Session-local
	srcRepo, ok = s.Repos[lookupKey]
	if !ok && s.Manager != nil {
		// Check Shared
		srcRepo, ok = s.Manager.SharedRemotes[lookupKey]
		// Fallback: Check using full URL
		if !ok {
			srcRepo, ok = s.Manager.SharedRemotes[url]
		}
	}

	if !ok {
		return "", fmt.Errorf("remote repository '%s' not found (simulated path or URL required)", url)
	}

	// Scan remote refs (branches) and fetch them
	refs, err := srcRepo.References()
	if err != nil {
		return "", err
	}

	updated := 0
	results := []string{fmt.Sprintf("From %s", url)}

	err = refs.ForEach(func(r *plumbing.Reference) error {
		if r.Name().IsBranch() {
			branchName := r.Name().Short()
			localRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remoteName, branchName))

			// Check if update needed
			currentLocal, errRef := repo.Reference(localRefName, true)
			if errRef == nil && currentLocal.Hash() == r.Hash() {
				return nil // up to date
			}

			if isDryRun {
				results = append(results, fmt.Sprintf(" * [dry-run] %s -> %s/%s", branchName, remoteName, branchName))
				return nil
			}

			// 1. Copy Objects using Shared Helper
			err = git.CopyCommitRecursive(srcRepo, repo, r.Hash())
			if err != nil {
				return err
			}

			// 2. Update Local Reference: refs/remotes/<remote>/<branch>
			newRef := plumbing.NewHashReference(localRefName, r.Hash())
			err = repo.Storer.SetReference(newRef)
			if err != nil {
				return err
			}

			status := "updated"
			if errRef != nil {
				status = "new branch"
			}

			results = append(results, fmt.Sprintf(" * [%s] %s -> %s/%s",
				status,
				branchName, remoteName, branchName))
			updated++
		}
		return nil
	})

	if err != nil {
		return "", err
	}

	if updated == 0 {
		return "", nil // Nothing to report for this remote if up to date
	}

	return strings.Join(results, "\n"), nil
}

func (c *FetchCommand) Help() string {
	return `usage: git fetch [options] [<remote>]

Options:
    -n, --dry-run     dry run (show what would be fetched without doing it)
    --help            display this help message

Download objects and refs from another repository.
Note: This is a simulated fetch from virtual remotes.
`
}
