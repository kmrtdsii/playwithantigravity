package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// ReflogEntry represents an entry in the reflog
type ReflogEntry struct {
	Hash    string
	Message string
}

// Session holds the state of a user's simulated git repo
type Session struct {
	ID         string
	Filesystem billy.Filesystem
	Repo       *git.Repository
	CreatedAt  time.Time
	Reflog     []ReflogEntry
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
		Reflog:     []ReflogEntry{},
	}
	return nil
}

// Helper to record reflog
func (s *Session) recordReflog(msg string) {
	if s.Repo == nil {
		return
	}
	headRef, err := s.Repo.Head()
	hash := ""
	if err == nil {
		hash = headRef.Hash().String()
	} else {
		return // HEAD not resolving usually means no commits yet
	}
	
	// Prepend for newest top
	s.Reflog = append([]ReflogEntry{{Hash: hash, Message: msg}}, s.Reflog...)
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
	
	case "help":
		if len(args) > 1 {
			subcmd := args[1]
			switch subcmd {
			case "init":
				return "usage: git init\n\nInitialize a new git repository.", nil
			case "status":
				return "usage: git status\n\nShow the working tree status.", nil
			case "add":
				return "usage: git add <file>...\n\nAdd file contents to the index.", nil
			case "commit":
				return "usage: git commit [-m <msg>] [--amend]\n\nRecord changes to the repository.\n\nOptions:\n  -m <msg>    Use the given <msg> as the commit message\n  --amend     Redo the previous commit", nil
			case "log":
				return "usage: git log [--oneline]\n\nShow commit logs.\n\nOptions:\n  --oneline   Show a concise one-line log", nil
			case "branch":
				return "usage: git branch [-d] [<branchname>]\n\nList, create, or delete branches.\n\nOptions:\n  -d          Delete a branch", nil
			case "checkout":
				return "usage: git checkout [-b] <branch>|-- <file>\n\nSwitch branches or restore working tree files.\n\nOptions:\n  -b          Create and checkout a new branch\n  -- <file>   Discard changes in working directory for file", nil
			case "merge":
				return "usage: git merge <commit>\n\nJoin two or more development histories together.", nil
			case "diff":
				return "usage: git diff <commit> <commit>\n\nShow changes between two commits.", nil
			case "tag":
				return "usage: git tag [-a] [-m <msg>] [-d] <tagname>\n\nCreate, list, delete tags.\n\nOptions:\n  -a          Make an annotated tag\n  -m <msg>    Tag message\n  -d          Delete a tag", nil
			case "reset":
				return "usage: git reset [--soft | --mixed | --hard] <commit>\n\nReset current HEAD to the specified state.\n\nOptions:\n  --soft      Reset HEAD only\n  --mixed     Reset HEAD and index (default)\n  --hard      Reset HEAD, index, and working tree", nil
			default:
				return fmt.Sprintf("git help: unknown command '%s'", subcmd), nil
			}
		}
		return `Supported commands:
   init       Initialize a repository
   status     Show the working tree status
   add        Add file contents to the index
   commit     Record changes to the repository
   log        Show commit logs
   branch     List, create, or delete branches
   checkout   Switch branches or restore working tree files
   merge      Join two or more development histories together
   diff       Show changes between commits, commit and working tree, etc
   tag        Create, list, delete or verify a tag object signed with GPG
   reset      Reset current HEAD to the specified state

Type 'git help <command>' for more information about a specific command.`, nil

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
		
		// Parse options
		msg := "Default commit message"
		amend := false

		// Naive arg parsing
		for i := 1; i < len(args); i++ {
			if args[i] == "-m" && i+1 < len(args) {
				msg = args[i+1]
				i++
			} else if args[i] == "--amend" {
				amend = true
			}
		}

		if amend {
			// Amend logic
			// 1. Get HEAD
			headRef, err := session.Repo.Head()
			if err != nil {
				return "", fmt.Errorf("cannot amend without HEAD: %v", err)
			}
			headCommit, err := session.Repo.CommitObject(headRef.Hash())
			if err != nil {
				return "", err
			}

			// 2. Reuse parent
			parents := headCommit.ParentHashes

			// 3. Reuse message if not provided
			if !strings.Contains(strings.Join(args, " "), "-m") {
				msg = headCommit.Message
			}

			// Save ORIG_HEAD before amending
			updateOrigHead(session)


			// 4. Commit with HEAD's parents
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
			session.recordReflog("commit (amend): " + strings.Split(msg, "\n")[0])

			return fmt.Sprintf("Commit amended: %s", newCommitHash.String()), nil
		}

		// Normal commit
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
		session.recordReflog(fmt.Sprintf("commit: %s", strings.Split(msg, "\n")[0]))
		return fmt.Sprintf("Commit created: %s", commit.String()), nil

	case "log":
		// Options
		oneline := false
		if len(args) > 1 && args[1] == "--oneline" {
			oneline = true
		}

		cIter, err := session.Repo.Log(&git.LogOptions{All: false}) // HEAD only usually
		if err != nil {
			return "", err
		}

		var sb strings.Builder
		err = cIter.ForEach(func(c *object.Commit) error {
			if oneline {
				// 7-char hash + message
				sb.WriteString(fmt.Sprintf("%s %s\n", c.Hash.String()[:7], strings.Split(c.Message, "\n")[0]))
			} else {
				sb.WriteString(fmt.Sprintf("commit %s\nAuthor: %s <%s>\nDate:   %s\n\n    %s\n\n",
					c.Hash.String(),
					c.Author.Name,
					c.Author.Email,
					c.Author.When.Format(time.RFC3339),
					strings.TrimSpace(c.Message),
				))
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		return sb.String(), nil

	case "rebase":
		if len(args) < 2 {
			return "", fmt.Errorf("usage: git rebase <upstream>")
		}
		
		// Update ORIG_HEAD before rebase starts
		updateOrigHead(session)
		
		upstreamName := args[1]
		
		// 1. Resolve Upstream
		upstreamHash, err := session.Repo.ResolveRevision(plumbing.Revision(upstreamName))
		if err != nil {
			return "", fmt.Errorf("invalid upstream '%s': %v", upstreamName, err)
		}
		upstreamCommit, err := session.Repo.CommitObject(*upstreamHash)
		if err != nil {
			return "", err
		}

		// 2. Resolve HEAD
		headRef, err := session.Repo.Head()
		if err != nil {
			return "", err
		}
		headCommit, err := session.Repo.CommitObject(headRef.Hash())
		if err != nil {
			return "", err
		}

		// 3. Find Merge Base
		mergeBases, err := upstreamCommit.MergeBase(headCommit)
		if err != nil {
			return "", fmt.Errorf("failed to find merge base: %v", err)
		}
		if len(mergeBases) == 0 {
			return "", fmt.Errorf("no common ancestor found")
		}
		base := mergeBases[0]

		if base.Hash == headCommit.Hash {
			return "Current branch is up to date.", nil
		}
		if base.Hash == upstreamCommit.Hash {
			return "Current branch is up to date (or ahead of upstream).", nil
		}

		// 4. Collect commits to replay (base..HEAD]
		var commitsToReplay []*object.Commit
		iter := headCommit
		for iter.Hash != base.Hash {
			commitsToReplay = append(commitsToReplay, iter)
			if iter.NumParents() == 0 {
				break
			}
			p, err := iter.Parent(0)
			if err != nil {
				return "", fmt.Errorf("failed to traverse parents: %v", err)
			}
			iter = p
		}
		// Reverse order
		for i, j := 0, len(commitsToReplay)-1; i < j; i, j = i+1, j-1 {
			commitsToReplay[i], commitsToReplay[j] = commitsToReplay[j], commitsToReplay[i]
		}

		// 5. Hard Reset to Upstream
		w, _ := session.Repo.Worktree()
		if err := w.Reset(&git.ResetOptions{Commit: *upstreamHash, Mode: git.HardReset}); err != nil {
			return "", fmt.Errorf("failed to reset to upstream: %v", err)
		}

		// 6. Replay Commits (Cherry-pick)
		replayedCount := 0
		for _, c := range commitsToReplay {
			parent, _ := c.Parent(0)
			pTree, _ := parent.Tree()
			cTree, _ := c.Tree()
			patch, err := pTree.Patch(cTree)
			if err != nil {
				return "", fmt.Errorf("failed to compute patch: %v", err)
			}

			for _, fp := range patch.FilePatches() {
				from, to := fp.Files()
				if to == nil {
					if from != nil {
						session.Filesystem.Remove(from.Path())
					}
					continue
				}
				path := to.Path()
				file, err := c.File(path)
				if err != nil { continue }
				content, err := file.Contents()
				if err != nil { continue }
				
				f, _ := session.Filesystem.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
				f.Write([]byte(content))
				f.Close()
				w.Add(path)
			}
			
			// Ensure timestamp distinctness
			time.Sleep(10 * time.Millisecond)

			_, err = w.Commit(c.Message, &git.CommitOptions{
				Author: &object.Signature{
					Name:  "User",
					Email: "user@example.com",
					When:  time.Now(),
				},
			})
			if err != nil {
				return "", fmt.Errorf("failed to commit replayed change: %v", err)
			}
			replayedCount++
		}

		session.recordReflog(fmt.Sprintf("rebase: finished rebase onto %s", upstreamName))
		return fmt.Sprintf("Successfully rebased and updated %s.\nReplayed %d commits.", headRef.Name().Short(), replayedCount), nil

	case "reflog":
		var sb strings.Builder
		for i, entry := range session.Reflog {
			sb.WriteString(fmt.Sprintf("%s HEAD@{%d}: %s\n", entry.Hash[:7], i, entry.Message))
		}
		return sb.String(), nil

	case "tag":
		// List tags
		if len(args) == 1 {
			tags, err := session.Repo.Tags()
			if err != nil {
				return "", err
			}
			var sb strings.Builder
			tags.ForEach(func(r *plumbing.Reference) error {
				sb.WriteString(r.Name().Short() + "\n")
				return nil
			})
			return sb.String(), nil
		}

		// Delete tag
		if args[1] == "-d" {
			if len(args) < 3 {
				return "", fmt.Errorf("tag name required")
			}
			tagName := args[2]
			if err := session.Repo.DeleteTag(tagName); err != nil {
				return "", err
			}
			return "Deleted tag " + tagName, nil
		}

		// Create Tag
		// Check for options
		if args[1] == "-a" {
			if len(args) < 4 {
				return "", fmt.Errorf("tag name and message required for annotated tag") // usage: git tag -a v1 -m "msg"
			}
			tagName := args[2]
			msg := "Tag message"
			if len(args) >= 5 && args[3] == "-m" {
				msg = args[4]
			}
			headRef, err := session.Repo.Head()
			if err != nil {
				return "", err
			}
			_, err = session.Repo.CreateTag(tagName, headRef.Hash(), &git.CreateTagOptions{
				Message: msg,
				Tagger: &object.Signature{
					Name:  "User",
					Email: "user@example.com",
					When:  time.Now(),
				},
			})
			if err != nil {
				return "", err
			}
			return "Created annotated tag " + tagName, nil
		}

		// Lightweight tag
		tagName := args[1]
		headRef, err := session.Repo.Head()
		if err != nil {
			return "", err
		}
		_, err = session.Repo.CreateTag(tagName, headRef.Hash(), nil) // nil opts = lightweight?? No, CreateTag creates annotated usually. 
		// Actually go-git `CreateTag` creates an object. For lightweight we just set ref.
		// Let's use CreateTag for now as lightweight might need manual Storer manipulation which is verbose.
		// Wait, CreateTag doc says: "If opts is nil, an annotated tag is created with default values". 
		// Real lightweight tag is just a ref to a commit.
		refName := plumbing.ReferenceName("refs/tags/" + tagName)
		ref := plumbing.NewHashReference(refName, headRef.Hash())
		if err := session.Repo.Storer.SetReference(ref); err != nil {
			return "", err
		}
		return "Created tag " + tagName, nil

		return "Created tag " + tagName, nil

	case "diff":
		if len(args) < 3 {
			// Naive: just show status diff is hard to implement nicely without full patch engine for worktree
			// For now, let's just support diffing two commits/refs
			return "usage: git diff <ref1> <ref2>\n(Worktree diff not yet supported)", nil
		}
		ref1 := args[1]
		ref2 := args[2]

		// Resolve refs
		h1, err := session.Repo.ResolveRevision(plumbing.Revision(ref1))
		if err != nil {
			return "", err
		}
		h2, err := session.Repo.ResolveRevision(plumbing.Revision(ref2))
		if err != nil {
			return "", err
		}

		c1, err := session.Repo.CommitObject(*h1)
		if err != nil {
			return "", err
		}
		c2, err := session.Repo.CommitObject(*h2)
		if err != nil {
			return "", err
		}

		tree1, err := c1.Tree()
		if err != nil {
			return "", err
		}
		tree2, err := c2.Tree()
		if err != nil {
			return "", err
		}

		patch, err := tree1.Patch(tree2)
		if err != nil {
			return "", err
		}

		return patch.String(), nil

	case "reset":
		// git reset [<mode>] [<commit>]
		// modes: --soft, --mixed, --hard
		// default mixed
		mode := git.MixedReset
		target := "HEAD"

		argsIdx := 1
		if len(args) > argsIdx && strings.HasPrefix(args[argsIdx], "--") {
			switch args[argsIdx] {
			case "--soft":
				mode = git.SoftReset
			case "--mixed":
				mode = git.MixedReset
			case "--hard":
				mode = git.HardReset
			default:
				return "", fmt.Errorf("unknown reset mode: %s", args[argsIdx])
			}
			argsIdx++
		}

		if len(args) > argsIdx {
			target = args[argsIdx]
		}

		// Resolve target
		h, err := session.Repo.ResolveRevision(plumbing.Revision(target))
		if err != nil {
			return "", err
		}

		w, _ := session.Repo.Worktree()
		
		// Update ORIG_HEAD before reset
		updateOrigHead(session)

		if err := w.Reset(&git.ResetOptions{
			Commit: *h,
			Mode:   mode,
		}); err != nil {
			return "", err
		}
		session.recordReflog(fmt.Sprintf("reset: moving to %s", target))

		return fmt.Sprintf("HEAD is now at %s", h.String()[:7]), nil
	
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

		// Handle branch deletion
		if args[1] == "-d" {
			if len(args) < 3 {
				return "", fmt.Errorf("branch name required")
			}
			branchName := args[2]

			// Validate branch exists
			refName := plumbing.ReferenceName("refs/heads/" + branchName)
			_, err := session.Repo.Reference(refName, true)
			if err != nil {
				return "", fmt.Errorf("branch '%s' not found.", branchName)
			}

			// Prevent deleting current branch
			headRef, err := session.Repo.Head()
			if err == nil && headRef.Name() == refName {
				return "", fmt.Errorf("cannot delete branch '%s' checked out at '%s'", branchName, "." /* worktree path info unavailable here */)
			}

			// Delete reference
			if err := session.Repo.Storer.RemoveReference(refName); err != nil {
				return "", err
			}
			return "Deleted branch " + branchName, nil
		}

		// Create branch
		branchName := args[1]
		if strings.HasPrefix(branchName, "-") {
			return "", fmt.Errorf("unknown switch `c' configuration: %s", branchName)
		}

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

	case "switch":
		w, _ := session.Repo.Worktree()
		if len(args) < 2 {
			return "", fmt.Errorf("usage: git switch [-c] <branch>")
		}

		// Handle -c (create and switch)
		if args[1] == "-c" {
			if len(args) < 3 {
				return "", fmt.Errorf("usage: git switch -c <branch>")
			}
			branchName := args[2]

			// Create new branch logic (similar to checkout -b)
			opts := &git.CheckoutOptions{
				Create: true,
				Force:  false,
				Branch: plumbing.ReferenceName("refs/heads/" + branchName),
			}
			if err := w.Checkout(opts); err != nil {
				return "", err
			}
			session.recordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", branchName))
			return fmt.Sprintf("Switched to a new branch '%s'", branchName), nil
		}

		// Handle normal switch (existing branch)
		target := args[1]
		
		// Validate that target is actually a branch (local)
		branchRefName := "refs/heads/" + target
		_, err := session.Repo.Reference(plumbing.ReferenceName(branchRefName), true)
		if err != nil {
			return "", fmt.Errorf("invalid reference: %s", target)
		}

		branchRef := plumbing.ReferenceName(branchRefName)
		err = w.Checkout(&git.CheckoutOptions{
			Branch: branchRef,
		})
		if err == nil {
			session.recordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", target))
			return fmt.Sprintf("Switched to branch '%s'", target), nil
		}
		return "", err

	case "checkout":
		w, _ := session.Repo.Worktree()
		if len(args) < 2 {
			return "", fmt.Errorf("usage: git checkout <branch> | git checkout -b <branch> | git checkout -- <file>")
		}

		// Handle file checkout (git checkout -- <file>)
		if args[1] == "--" {
			if len(args) < 3 {
				return "", fmt.Errorf("filename required after --")
			}
			filename := args[2]

			// Restore file from HEAD
			headRef, err := session.Repo.Head()
			if err == nil {
				headCommit, _ := session.Repo.CommitObject(headRef.Hash())
				file, err := headCommit.File(filename)
				if err != nil {
					return "", fmt.Errorf("file %s not found in HEAD", filename)
				}
				content, _ := file.Contents()
				
				f, _ := session.Filesystem.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
				f.Write([]byte(content))
				f.Close()
				return "Updated " + filename, nil
			}
			return "", fmt.Errorf("cannot checkout file without HEAD")
		}

		// Handle -b
		if args[1] == "-b" {
			if len(args) < 3 {
				return "", fmt.Errorf("usage: git checkout -b <branch>")
			}
			branchName := args[2]

			opts := &git.CheckoutOptions{
				Create: true,
				Force:  false,
				Branch: plumbing.ReferenceName("refs/heads/" + branchName),
			}
			if err := w.Checkout(opts); err != nil {
				return "", err
			}
			session.recordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", branchName))
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
			session.recordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", target)) // simplified from
			return fmt.Sprintf("Switched to branch '%s'", target), nil
		}

		// 2. Try as hash (Detached HEAD) / Tag / Short Hash
		// Use ResolveRevision to handle short hashes, tags, etc. properly AND verify existence.
		hash, err := session.Repo.ResolveRevision(plumbing.Revision(target))
		if err == nil {
			// Verify it's a commit
			if _, err := session.Repo.CommitObject(*hash); err != nil {
				return "", fmt.Errorf("reference is not a commit: %v", err)
			}
			
			err = w.Checkout(&git.CheckoutOptions{
				Hash: *hash,
			})
			if err == nil {
				session.recordReflog(fmt.Sprintf("checkout: moving from %s to %s", "HEAD", target))
				return fmt.Sprintf("Note: switching to '%s'.\n\nYou are in 'detached HEAD' state.", target), nil
			}
			return "", err
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
					
					// Update ORIG_HEAD before reset
					updateOrigHead(session)
					
					w.Reset(&git.ResetOptions{
						Commit: targetCommit.Hash,
						Mode:   git.HardReset,
					})
					return fmt.Sprintf("Updating %s..%s\nFast-forward", headCommit.Hash.String()[:7], targetCommit.Hash.String()[:7]), nil
				}

				// If we were detached, just checkout target
				updateOrigHead(session)
				w.Checkout(&git.CheckoutOptions{Hash: targetCommit.Hash})
				return fmt.Sprintf("Fast-forward to %s", targetName), nil
			}
		}

		// 4. Merge Commit
		// Simplified "Strategy Ours" for file content (ignoring conflicts for visualization demo)
		// We just create a commit with 2 parents.

		msg := fmt.Sprintf("Merge branch '%s'", targetName)
		parents := []plumbing.Hash{headCommit.Hash, targetCommit.Hash}
		
		// Update ORIG_HEAD before merge commit
		updateOrigHead(session)
		
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
	References map[string]string `json:"references"` // New field for other refs like ORIG_HEAD
	HEAD     Head              `json:"HEAD"`
	Files    []string          `json:"files"`
	Staging  []string          `json:"staging"`
	Modified []string          `json:"modified"`
	Untracked []string         `json:"untracked"`
	FileStatuses map[string]string `json:"fileStatuses"`
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

func GetGraphState(sessionID string, showAll bool) (*GraphState, error) {
	session, ok := sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	state := &GraphState{
		Commits:      []Commit{},
		Branches:     make(map[string]string),
		References:   make(map[string]string),
		FileStatuses: make(map[string]string),
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

		// Get Special Refs (ORIG_HEAD)
		origHeadRef, err := session.Repo.Reference("ORIG_HEAD", true)
		if err == nil {
			state.References["ORIG_HEAD"] = origHeadRef.Hash().String()
		}
	}

	// 3. Walk Commits
	if session.Repo != nil {
		if showAll {
			// Scan ALL objects to find every commit (including dangling ones)
			// Strategy: Collect all object.Commit pointers first, then Sort, then Convert.
			var collectedCommits []*object.Commit
			
			cIter, err := session.Repo.CommitObjects()
			if err == nil {
				cIter.ForEach(func(c *object.Commit) error {
					collectedCommits = append(collectedCommits, c)
					return nil
				})

				// Sort collected commits by FULL timestamp (high precision)
				// Use SliceStable to preserve BFS order (Child < Parent) where applicable
				// Note: CommitObjects() iteration order is undefined (not BFS), so the "Stable" part 
				// acts on the arbitrary iterator order. BUT, our primary sorting key is Time.
				// The specific logic for Tie-Breaker (Parent/Child check) is robust regardless of input order.
				// Pre-compute map for fast lookup
				commitMap := make(map[string]*object.Commit)
				for _, c := range collectedCommits {
					commitMap[c.Hash.String()] = c
				}

				// Helper: Is i ancestor of j? (Is j reachable from i?)
				// i is older, j is newer.
				// SearchBFS: start from j, look for i.
				isAncestor := func(i, j *object.Commit) bool {
					// bfs queue
					q := []string{j.Hash.String()}
					visited := make(map[string]bool)
					visited[j.Hash.String()] = true
					
					// Limit depth to avoid performance hit on large repos
					depth := 0
					maxDepth := 100 

					for len(q) > 0 {
						if depth > maxDepth {
							return false
						}
						currID := q[0]
						q = q[1:]

						if currID == i.Hash.String() {
							return true
						}

						// Expand parents
						if c, ok := commitMap[currID]; ok {
							for _, p := range c.ParentHashes {
								pID := p.String()
								if !visited[pID] {
									visited[pID] = true
									q = append(q, pID)
								}
							}
						}
						// Note: If parent is not in collectedCommits (e.g. standard view limited traversal), 
						// we can't traverse it efficiently without repo access.
						// BUT in 'showAll', we have all commits.
						// In 'Standard View', we only have reachable commits.
					}
					return false
				}

				sort.SliceStable(collectedCommits, func(i, j int) bool {
					tI := collectedCommits[i].Committer.When
					tJ := collectedCommits[j].Committer.When
					
					if tI.Equal(tJ) {
						// Time Equal. Use Topology.
						cI := collectedCommits[i]
						cJ := collectedCommits[j]

						// 1. Is i ancestor of j? (i reaches j) -> i is Older -> return false
						if isAncestor(cI, cJ) {
							return false
						}
						// 2. Is j ancestor of i? (j reaches i) -> j is Older -> return true
						if isAncestor(cJ, cI) {
							return true
						}

						// Fallback: Deterministic ID comparison
						return cI.Hash.String() > cJ.Hash.String()
					}
					return tI.After(tJ) // i > j (Newest first)
				})

				// Convert to View Model
				for _, c := range collectedCommits {
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
				}
			}
		} else {
			// Standard Graph Traversal (Reachable from Branches/Tags/HEAD only)
			
			seen := make(map[string]bool)
			var queue []plumbing.Hash
			var collectedCommits []*object.Commit

			if session.Repo != nil {
				// 1. HEAD
				h, err := session.Repo.Head()
				if err == nil {
					queue = append(queue, h.Hash())
				}
				// 2. Branches
				bIter, _ := session.Repo.Branches()
				bIter.ForEach(func(r *plumbing.Reference) error {
					queue = append(queue, r.Hash())
					return nil
				})
				// 3. Tags
				tIter, _ := session.Repo.Tags()
				tIter.ForEach(func(r *plumbing.Reference) error {
					queue = append(queue, r.Hash())
					return nil
				})
			}

			// BFS to find all reachable commits
			for len(queue) > 0 {
				current := queue[0]
				queue = queue[1:]

				if seen[current.String()] {
					continue
				}
				seen[current.String()] = true

				c, err := session.Repo.CommitObject(current)
				if err != nil {
					continue
				}

				collectedCommits = append(collectedCommits, c)
				
				// Enqueue parents
				queue = append(queue, c.ParentHashes...)
			}
			
			// Sort collected commits by FULL timestamp (high precision)
			// Use SliceStable to preserve BFS order (Child < Parent) when times are equal
			sort.SliceStable(collectedCommits, func(i, j int) bool {
				tI := collectedCommits[i].Committer.When
				tJ := collectedCommits[j].Committer.When
				
				if tI.Equal(tJ) {
					// Time is Equal.
					// BFS traversal guarantees children are visited before parents.
					// collectedCommits has children at lower indices than parents.
					// We want "Newest First".
					// Children (Index Small) are Newer. Parents (Index Large) are Older.
					// So if Equal Time:
					// We want Small Index (Child) to come BEFORE Large Index (Parent).
					// sort.SliceStable preserves strict original order if Less returns false.
					// Less(i, j) -> Is i newer than j?
					// If Equal, neither is newer. Return False.
					// Stable sort keeps i before j if i < j.
					return false
				}
				return tI.After(tJ) // i > j (Newest first)
			})

			// Convert to View Model
			for _, c := range collectedCommits {
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
			}
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
			// 1. Untracked
			if s.Staging == git.Untracked {
				state.Untracked = append(state.Untracked, file)
			}

			// 2. Modified (Worktree)
			// Must NOT be Unmodified AND NOT Untracked
			if s.Worktree != git.Unmodified && s.Staging != git.Untracked {
				state.Modified = append(state.Modified, file)
			}

			// 3. Staged
			if s.Staging != git.Unmodified && s.Staging != git.Untracked {
				state.Staging = append(state.Staging, file)
			}

			// 4. Status Codes (XY)
			x := statusCodeToChar(s.Staging)
			y := statusCodeToChar(s.Worktree)
			state.FileStatuses[file] = string(x) + string(y)
		}
	}

	return state, nil
}

func statusCodeToChar(c git.StatusCode) rune {
	switch c {
	case git.Unmodified:
		return ' '
	case git.Modified:
		return 'M'
	case git.Added:
		return 'A'
	case git.Deleted:
		return 'D'
	case git.Renamed:
		return 'R'
	case git.Copied:
		return 'C'
	case git.UpdatedButUnmerged:
		return 'U'
	case git.Untracked:
		return '?'
	default:
		return '-'
	}
}

// TouchFile updates the modification time and appends content to a file to ensure it's treated as modified
func TouchFile(sessionID, filename string) error {
	session, ok := sessions[sessionID]
	if !ok {
		return fmt.Errorf("session not found")
	}

	// Check if file exists
	_, err := session.Filesystem.Stat(filename)
	if err != nil {
		// File likely doesn't exist, create it (empty)
		f, err := session.Filesystem.Create(filename)
		if err != nil {
			return err
		}
		f.Close()
		return nil
	}

	// File exists, append to it to update hash/modification
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

// ListFiles returns a list of files in the worktree
func ListFiles(sessionID string) (string, error) {
	session, ok := sessions[sessionID]
	if !ok {
		return "", fmt.Errorf("session not found")
	}

	var files []string
	util.Walk(session.Filesystem, "/", func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if fi.IsDir() {
			if path == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		// Clean path
		if path != "" && path[0] == '/' {
			path = path[1:]
		}
		files = append(files, path)
		return nil
	})

	if len(files) == 0 {
		return "", nil
	}
	return strings.Join(files, "\n"), nil
}

// updateOrigHead saves the current HEAD to ORIG_HEAD ref
func updateOrigHead(session *Session) error {
	if session.Repo == nil {
		return nil
	}
	headRef, err := session.Repo.Head()
	if err != nil {
		return err // No HEAD to save
	}
	
	// Store ORIG_HEAD
	// We can use plumbing.ReferenceName("ORIG_HEAD")
	// But go-git might expect full ref paths or specialized handling.
	// Let's try to set it as a simplified reference.
	
	origHeadRef := plumbing.NewHashReference(plumbing.ReferenceName("ORIG_HEAD"), headRef.Hash())
	return session.Repo.Storer.SetReference(origHeadRef)
}
