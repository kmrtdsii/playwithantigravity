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

func (c *FetchCommand) Execute(_ context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	// 1. Parse Arguments & Flags
	// git fetch [<options>] [<repository> [<refspec>...]]
	var (
		isDryRun   bool
		fetchAll   bool
		fetchTags  bool
		prune      bool
		remoteName string
		refspecs   []string
	)

	// Simple flag parsing
	cmdArgs := args[1:]
	for i := 0; i < len(cmdArgs); i++ {
		arg := cmdArgs[i]
		switch {
		case arg == "-n" || arg == "--dry-run":
			isDryRun = true
		case arg == "--all":
			fetchAll = true
		case arg == "-t" || arg == "--tags":
			fetchTags = true
		case arg == "-p" || arg == "--prune":
			prune = true
		case arg == "-h" || arg == "--help":
			return c.Help(), nil
		case strings.HasPrefix(arg, "-"):
			return "", fmt.Errorf("error: unknown option `%s`", arg)
		default:
			if remoteName == "" {
				remoteName = arg
			} else {
				refspecs = append(refspecs, arg)
			}
		}
	}

	// 2. Determine Remotes to Fetch
	var remotes []*gogit.Remote
	var err error

	if fetchAll {
		remotes, err = repo.Remotes()
		if err != nil {
			return "", fmt.Errorf("failed to list remotes: %w", err)
		}
	} else {
		if remoteName == "" {
			remoteName = "origin" // Default to origin
		}
		rem, err := repo.Remote(remoteName)
		if err != nil {
			return "", fmt.Errorf("fatal: '%s' does not appear to be a git repository", remoteName)
		}
		remotes = []*gogit.Remote{rem}
	}

	// 3. Execute Fetch for each remote
	var allResults []string
	failed := false

	for _, rem := range remotes {
		res, err := c.fetchRemote(s, repo, rem, refspecs, fetchTags, prune, isDryRun)
		if err != nil {
			allResults = append(allResults, fmt.Sprintf("error: fetching %s: %v", rem.Config().Name, err))
			failed = true
		} else {
			if res != "" {
				allResults = append(allResults, res)
			}
		}
	}

	if failed && len(remotes) == 1 {
		return "", fmt.Errorf("fetch failed") // Return error for single remote failure
	}

	if len(allResults) == 0 {
		return "", nil // Standard git is silent if nothing happened? Or "Already up to date."? Git is usually silent if nothing printed.
	}

	return strings.Join(allResults, "\n"), nil
}

func (c *FetchCommand) fetchRemote(s *git.Session, repo *gogit.Repository, rem *gogit.Remote, refspecs []string, fetchTags bool, prune bool, isDryRun bool) (string, error) {
	cfg := rem.Config()
	remoteName := cfg.Name
	if len(cfg.URLs) == 0 {
		return "", fmt.Errorf("remote %s has no URL defined", remoteName)
	}
	url := cfg.URLs[0]

	// Look up simulated remote source
	srcRepo, err := c.resolveSimulatedRemote(s, url)
	if err != nil {
		return "", err
	}

	// Scan remote refs
	refs, err := srcRepo.References()
	if err != nil {
		return "", err
	}

	results := []string{fmt.Sprintf("From %s", url)}
	nothingFetched := true

	// Filter targets based on refspecs/tags
	// Map: RemoteBranchName -> TargetLocalRef
	fetchTargets := make(map[string]plumbing.ReferenceName)

	// Default fetchspec if no refspecs provided: +refs/heads/*:refs/remotes/origin/*
	// If refspecs provided (e.g. "main"), it usually means fetch that branch and map to FETCH_HEAD,
	// BUT for simplicity in this simulated env, we will map "main" -> "refs/remotes/origin/main"
	// unless a full refspec is given (which we honestly support minimally).

	// Build a list of candidate remote branches we care about
	candidates := make(map[string]*plumbing.Reference)

	err = refs.ForEach(func(r *plumbing.Reference) error {
		candidates[r.Name().String()] = r
		return nil
	})
	if err != nil {
		return "", err
	}

	// Logic to decide what to fetch
	if len(refspecs) > 0 {
		for _, spec := range refspecs {
			// Simplification: Assume spec is a branch name like "main" or "feature/1"
			// Try to find it in candidates as refs/heads/<spec>
			fullRemoteName := "refs/heads/" + spec
			if _, ok := candidates[fullRemoteName]; ok {
				localRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remoteName, spec))
				fetchTargets[fullRemoteName] = localRefName
				// Note: In real git, explicit refspecs also update FETCH_HEAD. We skip that complexity for now.
			} else {
				// Also check if it is a tag?
				fullTagName := "refs/tags/" + spec
				if _, ok := candidates[fullTagName]; ok {
					fetchTargets[fullTagName] = plumbing.ReferenceName(fullTagName)
				} else {
					return "", fmt.Errorf("fatal: couldn't find remote ref %s", spec)
				}
			}
		}
	} else {
		// Default behavior: Fetch all matching heads
		// Currently hardcoded to refs/heads/* -> refs/remotes/<remote>/*
		// This should respect remote config fetch refspecs in reality.
		for name := range candidates {
			if strings.HasPrefix(name, "refs/heads/") {
				branchName := strings.TrimPrefix(name, "refs/heads/")
				localRefName := plumbing.ReferenceName(fmt.Sprintf("refs/remotes/%s/%s", remoteName, branchName))
				fetchTargets[name] = localRefName
			}
		}
	}

	// Handle --tags
	// "This default behavior can be changed by using the --tags or --no-tags options"
	// --tags: Fetch all tags from the remote
	if fetchTags {
		for name := range candidates {
			if strings.HasPrefix(name, "refs/tags/") {
				fetchTargets[name] = plumbing.ReferenceName(name) // Tags map 1:1
			}
		}
	}

	// --- 1. PRUNE PHASE ---
	if prune {
		// Iterate local refs/remotes/<remoteName>/* and see if they exist in candidates
		// Note: This logic needs to be careful not to prune things we just chose NOT to fetch if we have specific refspecs.
		// "git fetch --prune" usually prunes remote-tracking branches that no longer exist on remote.
		// It only applies to the configured fetch refspec mappings.
		// For simplicity/safety, we only prune if NO specific refspecs were given (default fetch all mode).
		if len(refspecs) == 0 {
			iter, _ := repo.References()
			prefix := fmt.Sprintf("refs/remotes/%s/", remoteName)
			var toDelete []plumbing.ReferenceName

			iter.ForEach(func(r *plumbing.Reference) error {
				if strings.HasPrefix(r.Name().String(), prefix) {
					// Extract branch name
					branchPart := strings.TrimPrefix(r.Name().String(), prefix)
					remoteSide := "refs/heads/" + branchPart

					if _, exists := candidates[remoteSide]; !exists {
						toDelete = append(toDelete, r.Name())
					}
				}
				return nil
			})

			for _, refName := range toDelete {
				if !isDryRun {
					repo.Storer.RemoveReference(refName)
				}
				results = append(results, fmt.Sprintf(" - [deleted]         (none) -> %s", strings.TrimPrefix(refName.String(), "refs/remotes/")))
				nothingFetched = false
			}
		}
	}

	// --- 2. FETCH UPDATE PHASE ---
	for remoteRefName, localRefName := range fetchTargets {
		remoteRef := candidates[remoteRefName]

		// 1. Copy Objects
		if !isDryRun {
			err = git.CopyCommitRecursive(srcRepo, repo, remoteRef.Hash())
			if err != nil {
				return "", err
			}
		}

		// 2. Update Ref
		currentLocal, errRef := repo.Reference(localRefName, true)

		var action string
		var summary string

		if errRef != nil {
			// New
			action = "[new]"
			// Try to detect if it is branch or tag for display
			if strings.HasPrefix(localRefName.String(), "refs/tags/") {
				action = "[new tag]"
				summary = fmt.Sprintf("%s -> %s", remoteRef.Name().Short(), localRefName.Short())
			} else {
				action = "[new branch]"
				summary = fmt.Sprintf("%s -> %s/%s", remoteRef.Name().Short(), remoteName, remoteRef.Name().Short())
			}

			if !isDryRun {
				newRef := plumbing.NewHashReference(localRefName, remoteRef.Hash())
				repo.Storer.SetReference(newRef)
			}
			nothingFetched = false
			results = append(results, fmt.Sprintf(" * %-18s %s", action, summary))

		} else if currentLocal.Hash() != remoteRef.Hash() {
			// Updated
			// Check for forced update? Assuming fast-forward for now or force.
			// Git fetch usually forces remote refs updates.
			action = "   " + remoteRef.Hash().String()[:7] + ".." + currentLocal.Hash().String()[:7] // rough approx
			summary = fmt.Sprintf("%s -> %s/%s", remoteRef.Name().Short(), remoteName, remoteRef.Name().Short())

			if !isDryRun {
				newRef := plumbing.NewHashReference(localRefName, remoteRef.Hash())
				repo.Storer.SetReference(newRef)
			}
			nothingFetched = false
			results = append(results, fmt.Sprintf("   %-18s %s", action, summary))
		}
		// If equal, say nothing
	}

	if nothingFetched {
		return "", nil
	}

	return strings.Join(results, "\n"), nil
}

func (c *FetchCommand) resolveSimulatedRemote(s *git.Session, url string) (*gogit.Repository, error) {
	lookupKey := strings.TrimPrefix(url, "/")

	// Check Session-local
	if repo, ok := s.Repos[lookupKey]; ok {
		return repo, nil
	}

	if s.Manager != nil {
		// Check Shared
		if repo, ok := s.Manager.SharedRemotes[lookupKey]; ok {
			return repo, nil
		}
		// Fallback: Check using full URL
		if repo, ok := s.Manager.SharedRemotes[url]; ok {
			return repo, nil
		}
	}

	return nil, fmt.Errorf("remote repository '%s' not found (simulated path or URL required)", url)
}

func (c *FetchCommand) Help() string {
	return `usage: git fetch [options] [<remote> [<refspec>...]]

Options:
    -n, --dry-run          dry run (show what would be fetched)
    -t, --tags             fetch all tags
    -p, --prune            prune remote-tracking branches no longer on remote
    --all                  fetch from all remotes
    --help                 display this help message
`
}
