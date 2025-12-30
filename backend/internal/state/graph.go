package state

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-git/go-billy/v5/util"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// GetGraphState returns the current state of the repository for frontend visualization
func (sm *SessionManager) GetGraphState(sessionID string, showAll bool) (*GraphState, error) {
	session, ok := sm.GetSession(sessionID)
	if !ok {
		return nil, fmt.Errorf("session not found")
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	repo := session.GetRepo()

	// Delegate to BuildGraphState for the repo-specific data
	// But we need to merge it with Session-specific data (Projects, proper Path)

	// Create base structure from Session data
	state := BuildGraphState(repo, showAll)

	// Override/Augment with Session Data
	state.PotentialCommits = session.PotentialCommits
	state.CurrentPath = session.CurrentDir

	sm.mu.RLock()
	for name := range sm.SharedRemotes {
		state.SharedRemotes = append(state.SharedRemotes, name)
	}
	sm.mu.RUnlock()
	sort.Strings(state.SharedRemotes)

	// 6. File System (Explorer) - Session specific
	populateFiles(session, state)

	// 7. Projects - Session specific
	populateProjects(session, state)

	return state, nil
}

// BuildGraphState constructs a GraphState from a git.Repository.
// It can be used for both local session repos and shared remotes.
func BuildGraphState(repo *gogit.Repository, showAll bool) *GraphState {
	state := &GraphState{
		Commits:        []Commit{},
		Branches:       make(map[string]string),
		RemoteBranches: make(map[string]string),
		Tags:           make(map[string]string),
		References:     make(map[string]string),
		FileStatuses:   make(map[string]string),
		Remotes:        []Remote{},
		SharedRemotes:  []string{},
		Initialized:    (repo != nil),
	}

	// 1. Get HEAD
	populateHEAD(repo, state)

	if repo != nil {
		// 2. Get Branches & Tags
		if err := populateBranchesAndTags(repo, state); err != nil {
			log.Printf("BuildGraphState warning: %v", err)
		}

		// 3. Walk Commits
		// Use BFS from Refs (if showAll=false) or iterate all objects (if showAll=true)
		populateCommits(repo, state, showAll)
		// Let's assume for Shared Remote we want to show everything we have.
		// Actually, populateCommits logic for ancestors might be better.
		// But for "Server View", showing the reachable history from branches is correct.

		// 4. Git Status (Might be empty for bare repos, but harmless)
		if err := populateGitStatus(repo, state); err != nil {
			// Bare repos often fail Worktree(), ignore
			log.Printf("populateGitStatus ignored error: %v", err)
		}

		// 5. Remotes
		populateRemotes(repo, state)
	}

	return state
}

func populateHEAD(repo *gogit.Repository, state *GraphState) {
	if repo == nil {
		state.HEAD = Head{Type: "none"}
		return
	}
	ref, err := repo.Head()
	if err != nil {
		if err.Error() == "reference not found" {
			// Unborn branch (orphan): HEAD is a symbolic ref to a non-existent branch
			// Read the symbolic reference directly to get the branch name
			headRef, symErr := repo.Reference(plumbing.HEAD, false)
			if symErr == nil && headRef.Type() == plumbing.SymbolicReference {
				// Extract branch name from refs/heads/<name>
				branchName := headRef.Target().Short()
				state.HEAD = Head{Type: "branch", Ref: branchName}
				return
			}
			// Fallback to main if we can't read HEAD
			state.HEAD = Head{Type: "branch", Ref: "main"}
		} else {
			// Log error or set to none
			state.HEAD = Head{Type: "none"}
		}
	} else {
		if ref.Name().IsBranch() {
			state.HEAD = Head{Type: "branch", Ref: ref.Name().Short()}
		} else {
			state.HEAD = Head{Type: "commit", ID: ref.Hash().String()}
		}
	}
}

func populateBranchesAndTags(repo *gogit.Repository, state *GraphState) error {
	iter, err := repo.Branches()
	if err != nil {
		return err
	}
	err = iter.ForEach(func(r *plumbing.Reference) error {
		state.Branches[r.Name().Short()] = r.Hash().String()
		return nil
	})
	if err != nil {
		return err
	}

	// We do a single pass for robustness.
	refs, err := repo.References()
	if err == nil {
		_ = refs.ForEach(func(r *plumbing.Reference) error {
			if r.Name().IsRemote() {
				state.RemoteBranches[r.Name().Short()] = r.Hash().String()
				// log.Printf("Graph: Found Remote Branch %s -> %s", r.Name().Short(), r.Hash().String())
			} else if r.Name().IsTag() {
				hash := r.Hash().String()
				// Check if it's an annotated tag
				tagObj, tagErr := repo.TagObject(r.Hash())
				if tagErr == nil {
					hash = tagObj.Target.String()
				}
				state.Tags[r.Name().Short()] = hash
			}
			return nil
		})
	}

	// Get Special Refs (ORIG_HEAD)
	origHeadRef, err := repo.Reference("ORIG_HEAD", true)
	if err == nil {
		state.References["ORIG_HEAD"] = origHeadRef.Hash().String()
	}

	return nil
}

func populateFiles(session *Session, state *GraphState) {
	// Show detailed file list based on the WORKTREE (filesystem), including untracked.

	repo := session.GetRepo()
	if repo == nil {
		return
	}

	w, err := repo.Worktree()
	if err != nil {
		// Bare repo or other issue
		return
	}

	// Walk the filesystem to get ALL files including untracked
	// PERFORMANCE GUARD: Limit file count to preventing UI freezing
	const MaxFileCount = 1000
	count := 0

	_ = util.Walk(w.Filesystem, "/", func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if count >= MaxFileCount {
			return filepath.SkipDir // Stop walking
		}

		if fi.IsDir() {
			if path == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Clean path if needed (billy usually returns clean paths)
		if path != "" && path[0] == '/' {
			path = path[1:]
		}

		state.Files = append(state.Files, path)
		count++
		return nil
	})

	if count >= MaxFileCount {
		state.Files = append(state.Files, "... (limit reached)")
	}
}

func populateGitStatus(repo *gogit.Repository, state *GraphState) error {
	w, err := repo.Worktree()
	if err != nil {
		return err
	}

	status, err := w.Status()
	if err != nil {
		return err
	}

	for file, s := range status {
		if s.Staging == gogit.Untracked {
			state.Untracked = append(state.Untracked, file)
		}
		if s.Worktree != gogit.Unmodified && s.Staging != gogit.Untracked {
			state.Modified = append(state.Modified, file)
		}
		if s.Staging != gogit.Unmodified && s.Staging != gogit.Untracked {
			state.Staging = append(state.Staging, file)
		}
		x := statusCodeToChar(s.Staging)
		y := statusCodeToChar(s.Worktree)
		state.FileStatuses[file] = string(x) + string(y)
	}
	return nil
}

func populateProjects(session *Session, state *GraphState) {
	rootInfos, err := session.Filesystem.ReadDir("/")
	if err == nil {
		for _, info := range rootInfos {
			if info.IsDir() && info.Name() != ".git" {
				state.Projects = append(state.Projects, info.Name())
			}
		}
		log.Printf("Scan Projects: found %d projects: %v", len(state.Projects), state.Projects)
	} else {
		log.Printf("Scan Projects Error: %v", err)
	}
}

func statusCodeToChar(c gogit.StatusCode) rune {
	switch c {
	case gogit.Unmodified:
		return ' '
	case gogit.Modified:
		return 'M'
	case gogit.Added:
		return 'A'
	case gogit.Deleted:
		return 'D'
	case gogit.Renamed:
		return 'R'
	case gogit.Copied:
		return 'C'
	case gogit.UpdatedButUnmerged:
		return 'U'
	case gogit.Untracked:
		return '?'
	default:
		return '-'
	}
}

func populateRemotes(repo *gogit.Repository, state *GraphState) {
	remotes, err := repo.Remotes()
	if err != nil {
		return
	}
	for _, r := range remotes {
		cfg := r.Config()
		state.Remotes = append(state.Remotes, Remote{
			Name: cfg.Name,
			URLs: cfg.URLs,
		})
	}
}
