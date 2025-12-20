package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
)

// Session holds the state of a user's simulated git repo
type Session struct {
	ID         string
	Filesystem billy.Filesystem
	Repo       *git.Repository
	CreatedAt  time.Time
}

var sessions = make(map[string]*Session)

// InitSession creates a new session with empty filesystem
func InitSession(id string) error {
	fs := memfs.New()
	// st := memory.NewStorage() // Storage created on init

	// No git init here
	// No files created here

	sessions[id] = &Session{
		ID:         id,
		Filesystem: fs,
		Repo:       nil, // Repo is nil until git init
		CreatedAt:  time.Now(),
	}
	return nil
}

// ExecuteGitCommand parses a simple command string and executes it on the repo
// This is a naive implementation. In a real app, we'd parse args properly.
func ExecuteGitCommand(sessionID string, args []string) (string, error) {
	session, ok := sessions[sessionID]
	if !ok {
		return "", fmt.Errorf("session not found")
	}

	if len(args) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	cmd := args[0]
	// Check for repo existence for non-init commands
	if session.Repo == nil && cmd != "init" {
		return "", fmt.Errorf("fatal: not a git repository (or any of the parent directories): .git")
	}

	switch cmd {
	case "status":
		w, _ := session.Repo.Worktree()
		status, _ := w.Status()
		return status.String(), nil

	case "init":
		if session.Repo != nil {
			return "Git repository already initialized", nil
		}

		st := memory.NewStorage()
		repo, err := git.Init(st, session.Filesystem)
		if err != nil {
			return "", err
		}
		session.Repo = repo

		// Set default branch to main
		headRef := plumbing.NewSymbolicReference(plumbing.HEAD, plumbing.ReferenceName("refs/heads/main"))
		session.Repo.Storer.SetReference(headRef)

		// Create initial files (simulating project start)
		f, _ := session.Filesystem.Create("README.md")
		f.Write([]byte("# My Project\n"))
		f.Close()

		return "Initialized empty Git repository in /", nil

	case "add":
		w, _ := session.Repo.Worktree()
		if len(args) < 2 {
			return "", fmt.Errorf("usage: git add <file>")
		}
		file := args[1]
		var err error
		if file == "." {
			_, err = w.Add(".")
		} else {
			_, err = w.Add(file)
		}
		if err != nil {
			return "", err
		}
		return "Added " + file, nil

	case "commit":
		w, _ := session.Repo.Worktree()
		// simple parsing
		msg := "Default commit message"
		if len(args) >= 3 && args[1] == "-m" {
			msg = args[2] // This is very naive split
		}

		commit, err := w.Commit(msg, &git.CommitOptions{
			Author: &object.Signature{
				Name:  "User",
				Email: "user@example.com",
				When:  time.Now(),
			},
		})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Commit created: %s", commit.String()), nil

	case "log":
		// ... implementation
		return "Log not implemented yet", nil

	case "branch":
		if len(args) == 1 {
			// List branches
			iter, err := session.Repo.Branches()
			if err != nil {
				return "", err
			}
			var branches []string
			iter.ForEach(func(r *plumbing.Reference) error {
				branches = append(branches, r.Name().Short())
				return nil
			})
			return strings.Join(branches, "\n"), nil
		}

		// Create branch
		branchName := args[1]
		headRef, err := session.Repo.Head()
		if err != nil {
			return "", fmt.Errorf("cannot create branch: %v (maybe no commits yet?)", err)
		}

		// Create new reference
		refName := plumbing.ReferenceName("refs/heads/" + branchName)
		newRef := plumbing.NewHashReference(refName, headRef.Hash())

		if err := session.Repo.Storer.SetReference(newRef); err != nil {
			return "", err
		}

		return "Created branch " + branchName, nil

	case "checkout":
		w, _ := session.Repo.Worktree()
		if len(args) < 2 {
			return "", fmt.Errorf("usage: git checkout <branch> | git checkout -b <branch>")
		}

		// Handle -b
		if args[1] == "-b" {
			if len(args) < 3 {
				return "", fmt.Errorf("usage: git checkout -b <branch>")
			}
			branchName := args[2]

			// Create branch reference manually first (like git branch <name>)
			// We do this because Checkout with Create: true might fail if we don't handle it right,
			// or we can use the CheckoutOptions.Create but let's stick to standard go-git flow.
			// Actually go-git CheckoutOptions has Create: true.

			err := w.Checkout(&git.CheckoutOptions{
				Create: true,
				Force:  false,
				Branch: plumbing.ReferenceName("refs/heads/" + branchName),
			})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("Switched to a new branch '%s'", branchName), nil
		}

		// Handle normal checkout (branch or commit)
		target := args[1]

		// 1. Try as branch
		branchRef := plumbing.ReferenceName("refs/heads/" + target)
		err := w.Checkout(&git.CheckoutOptions{
			Branch: branchRef,
		})
		if err == nil {
			return fmt.Sprintf("Switched to branch '%s'", target), nil
		}

		// 2. Try as hash (Detached HEAD)
		hash := plumbing.NewHash(target)
		err = w.Checkout(&git.CheckoutOptions{
			Hash: hash,
		})
		if err == nil {
			return fmt.Sprintf("Note: switching to '%s'.\n\nYou are in 'detached HEAD' state.", target), nil
		}

		return "", fmt.Errorf("pathspec '%s' did not match any file(s) known to git", target)

	case "merge":
		w, _ := session.Repo.Worktree()
		if len(args) < 2 {
			return "", fmt.Errorf("usage: git merge <branch>")
		}
		targetName := args[1]

		// 1. Resolve HEAD
		headRef, err := session.Repo.Head()
		if err != nil {
			return "", err
		}
		headCommit, err := session.Repo.CommitObject(headRef.Hash())
		if err != nil {
			return "", err
		}

		// 2. Resolve Target
		// Try resolving as branch first
		targetRef, err := session.Repo.Reference(plumbing.ReferenceName("refs/heads/"+targetName), true)
		var targetHash plumbing.Hash
		if err == nil {
			targetHash = targetRef.Hash()
		} else {
			// Try as hash
			targetHash = plumbing.NewHash(targetName)
		}

		targetCommit, err := session.Repo.CommitObject(targetHash)
		if err != nil {
			return "", fmt.Errorf("merge: %s - not something we can merge", targetName)
		}

		// 3. Analyze Ancestry
		base, err := targetCommit.MergeBase(headCommit)
		if err == nil && len(base) > 0 {
			// Check for "Already up to date"
			// If target is ancestor of HEAD (base == target), then we have nothing to do
			if base[0].Hash == targetCommit.Hash {
				return "Already up to date.", nil
			}

			// Check for Fast-Forward
			// If HEAD is ancestor of target (base == head), then we can FF
			if base[0].Hash == headCommit.Hash {
				// Perform Checkout (Fast-Forward)
				err = w.Checkout(&git.CheckoutOptions{
					Hash: targetCommit.Hash,
				})
				if err != nil {
					return "", err
				}

				// If we were on a branch, update the branch ref too?
				// w.Checkout(Hash) puts us in Detached HEAD if we don't specify Branch.
				// But we want to move the current branch pointer.
				// go-git's w.Checkout behavior:
				// If we are on a branch, and we merge, we want to update THAT branch to point to new commit.

				// If we use w.Checkout with Hash, it creates detached HEAD.
				// We need to manually update the reference of the current HEAD branch.

				if headRef.Name().IsBranch() {
					newRef := plumbing.NewHashReference(headRef.Name(), targetCommit.Hash)
					session.Repo.Storer.SetReference(newRef)
					// And we need to update working tree files?
					// w.Checkout with Keep: true?
					// Or just w.Reset?
					w.Reset(&git.ResetOptions{
						Commit: targetCommit.Hash,
						Mode:   git.HardReset,
					})
					return fmt.Sprintf("Updating %s..%s\nFast-forward", headCommit.Hash.String()[:7], targetCommit.Hash.String()[:7]), nil
				}

				// If we were detached, just checkout target
				w.Checkout(&git.CheckoutOptions{Hash: targetCommit.Hash})
				return fmt.Sprintf("Fast-forward to %s", targetName), nil
			}
		}

		// 4. Merge Commit
		// Simplified "Strategy Ours" for file content (ignoring conflicts for visualization demo)
		// We just create a commit with 2 parents.

		msg := fmt.Sprintf("Merge branch '%s'", targetName)
		parents := []plumbing.Hash{headCommit.Hash, targetCommit.Hash}

		newCommitHash, err := w.Commit(msg, &git.CommitOptions{
			Parents: parents,
			Author: &object.Signature{
				Name:  "User",
				Email: "user@example.com",
				When:  time.Now(),
			},
		})
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("Merge made by the 'ort' strategy.\n %s", newCommitHash.String()), nil

	default:
		return "", fmt.Errorf("command not supported: %s", cmd)
	}
}

// GraphState represents the serialized state for the frontend
type GraphState struct {
	Commits  []Commit          `json:"commits"`
	Branches map[string]string `json:"branches"`
	HEAD     Head              `json:"HEAD"`
	Files    []string          `json:"files"`
	Staging  []string          `json:"staging"`
	Modified []string          `json:"modified"`
}

type Commit struct {
	ID             string `json:"id"`
	Message        string `json:"message"`
	ParentID       string `json:"parentId"`
	SecondParentID string `json:"secondParentId"`
	Branch         string `json:"branch"` // Naive branch inference
	Timestamp      string `json:"timestamp"`
}

type Head struct {
	Type string `json:"type"` // "branch" or "commit"
	Ref  string `json:"ref,omitempty"`
	ID   string `json:"id,omitempty"`
}

func GetGraphState(sessionID string) (*GraphState, error) {
	session, ok := sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	state := &GraphState{
		Commits:  []Commit{},
		Branches: make(map[string]string),
	}

	// 1. Get HEAD
	if session.Repo == nil {
		// No repo, no HEAD
		state.HEAD = Head{Type: "none"}
	} else {
		ref, err := session.Repo.Head()
		if err != nil {
			// If empty repo (no commits yet)
			if err.Error() == "reference not found" {
				state.HEAD = Head{Type: "branch", Ref: "main"} // Default
				// Continue to get files even if no commits
			} else {
				return nil, err
			}
		} else {
			if ref.Name().IsBranch() {
				state.HEAD = Head{Type: "branch", Ref: ref.Name().Short()}
			} else {
				state.HEAD = Head{Type: "commit", ID: ref.Hash().String()}
			}
		}
	}

	// 2. Get Branches
	if session.Repo != nil {
		iter, err := session.Repo.Branches()
		if err != nil {
			return nil, err
		}
		iter.ForEach(func(r *plumbing.Reference) error {
			state.Branches[r.Name().Short()] = r.Hash().String()
			return nil
		})
	}

	// 3. Walk Commits
	if session.Repo != nil {
		cIter, err := session.Repo.Log(&git.LogOptions{All: true})
		if err == nil {
			cIter.ForEach(func(c *object.Commit) error {
				parentID := ""
				if len(c.ParentHashes) > 0 {
					parentID = c.ParentHashes[0].String()
				}
				secondParentID := ""
				if len(c.ParentHashes) > 1 {
					secondParentID = c.ParentHashes[1].String()
				}

				state.Commits = append(state.Commits, Commit{
					ID:             c.Hash.String(),
					Message:        c.Message,
					ParentID:       parentID,
					SecondParentID: secondParentID,
					Timestamp:      c.Committer.When.Format(time.RFC3339),
				})
				return nil
			})
		}
	}

	// 4. Get Status (Files, Staging, Modified)

	// Walk filesystem to find all files (tracked and untracked)
	// Even if no repo, we can list files (which should be empty initially)
	fmt.Println("Searching for files in root...")
	util.Walk(session.Filesystem, "/", func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Walk error: %v\n", err)
			return nil
		}
		if fi.IsDir() {
			if path == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		fmt.Printf("Found file: %s\n", path)

		// Clean path
		if path != "" && path[0] == '/' {
			path = path[1:]
		}

		state.Files = append(state.Files, path)
		return nil
	})
	fmt.Printf("Total files found: %d\n", len(state.Files))

	if session.Repo != nil {
		w, _ := session.Repo.Worktree()
		status, _ := w.Status()
		for file, s := range status {
			// Only add to Staging if it is NOT Unmodified AND NOT Untracked
			if s.Staging != git.Unmodified && s.Staging != git.Untracked {
				state.Staging = append(state.Staging, file)
			}
			if s.Worktree != git.Unmodified {
				state.Modified = append(state.Modified, file)
			}
		}
	}

	return state, nil
}

// TouchFile updates the modification time and appends content to a file to ensure it's treated as modified
func TouchFile(sessionID, filename string) error {
	session, ok := sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found")
	}

	// Append a newline to ensure checksum changes (go-git relies on hash)
	f, err := session.Filesystem.OpenFile(filename, os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := f.Write([]byte("\n// Update")); err != nil {
		return err
	}

	return nil
}
