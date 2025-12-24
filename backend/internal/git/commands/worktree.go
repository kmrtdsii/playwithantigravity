package commands

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/kurobon/gitgym/backend/internal/git"
)

func init() {
	git.RegisterCommand("worktree", func() git.Command { return &WorktreeCommand{} })
}

type WorktreeCommand struct{}

func (c *WorktreeCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
	s.Lock()
	defer s.Unlock()

	repo := s.GetRepo()
	if repo == nil {
		return "", fmt.Errorf("fatal: not a git repository")
	}

	if len(args) < 2 {
		return "", fmt.Errorf("usage: git worktree <add|list|prune> ...")
	}

	subcmd := args[1]

	switch subcmd {
	case "add":
		// Usage: git worktree add <path> [branch]
		if len(args) < 3 {
			return "", fmt.Errorf("usage: git worktree add <path> [branch]")
		}
		path := args[2]
		branch := ""
		if len(args) > 3 {
			branch = args[3]
		}

		targetPath := resolvePath(s.CurrentDir, path)
		// Remove leading slash for key
		targetPathName := strings.TrimPrefix(targetPath, "/")
		if targetPathName == "" {
			targetPathName = "root"
		}

		if _, exists := s.Repos[targetPathName]; exists {
			return "", fmt.Errorf("fatal: '%s' already exists", path)
		}

		// 2. Create Directory (Virtual)
		if err := s.Filesystem.MkdirAll(targetPath, 0755); err != nil {
			return "", err
		}

		// 3. Setup Worktree Structs
		wtStorer := git.NewWorktreeStorer(repo.Storer)

		// 4. Resolve Branch/HEAD
		var headHash plumbing.Hash
		var refName plumbing.ReferenceName

		if branch != "" {
			bRef, err := repo.Reference(plumbing.NewBranchReferenceName(branch), true)
			if err == nil {
				_ = bRef.Hash() // Checked for existence
				refName = bRef.Name()
			} else {
				// Create new branch
				currHead, err := repo.Head()
				if err != nil {
					return "", err
				}

				headHash = currHead.Hash()
				refName = plumbing.NewBranchReferenceName(branch)

				newBranchRef := plumbing.NewHashReference(refName, headHash)
				if err := repo.Storer.SetReference(newBranchRef); err != nil {
					return "", err
				}
			}
		} else {
			// Infer branch name
			baseName := path
			if idx := strings.LastIndex(path, "/"); idx != -1 {
				baseName = path[idx+1:]
			}
			refName = plumbing.NewBranchReferenceName(baseName)

			currHead, err := repo.Head()
			if err != nil {
				return "", err
			}
			headHash = currHead.Hash()

			newBranchRef := plumbing.NewHashReference(refName, headHash)
			if err := repo.Storer.SetReference(newBranchRef); err != nil {
				return "", err
			}
		}

		// Set wrapper HEAD (Symbolic)
		headRef := plumbing.NewSymbolicReference(plumbing.HEAD, refName)
		if err := wtStorer.SetReference(headRef); err != nil {
			return "", err
		}

		// 5. Open new Repository
		wtFS, err := s.Filesystem.Chroot(targetPath)
		if err != nil {
			return "", err
		}

		newRepo, err := gogit.Init(wtStorer, wtFS)
		if err != nil && err != gogit.ErrRepositoryAlreadyExists {
			return "", err
		}

		// Reset to populate files
		w, err := newRepo.Worktree()
		if err != nil {
			return "", err
		}

		if err := w.Reset(&gogit.ResetOptions{Mode: gogit.HardReset}); err != nil {
			// ignore
			_ = err
		}

		// 6. Register
		s.Repos[targetPathName] = newRepo

		return fmt.Sprintf("Preparing worktree (new branch '%s')", refName.Short()), nil

	case "list":
		var sb strings.Builder
		for path, repo := range s.Repos {
			head, err := repo.Head()
			headStr := "DETACHED"
			if err == nil {
				headStr = head.Hash().String()[:7]
				if head.Name().IsBranch() {
					headStr = fmt.Sprintf("[%s]", head.Name().Short())
				}
			}
			sb.WriteString(fmt.Sprintf("%s  %s\n", path, headStr))
		}
		return sb.String(), nil

	case "prune":
		return "", nil

	default:
		return "", fmt.Errorf("unknown subcommand: %s", subcmd)
	}
}

func (c *WorktreeCommand) Help() string {
	return "git worktree <add|list|prune>"
}

func resolvePath(current, target string) string {
	if strings.HasPrefix(target, "/") {
		return target
	}

	if !strings.HasPrefix(current, "/") {
		current = "/" + current
	}

	if strings.HasPrefix(target, "../") {
		lastSlash := strings.LastIndex(current, "/")
		if lastSlash <= 0 {
			return "/" + strings.TrimPrefix(target, "../")
		}
		parent := current[:lastSlash]
		if parent == "" {
			parent = "/"
		}
		return parent + "/" + strings.TrimPrefix(target, "../")
	}

	if current == "/" {
		return "/" + target
	}
	return current + "/" + target
}
