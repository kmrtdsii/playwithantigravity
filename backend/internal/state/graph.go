package state

import (
	"fmt"
	"log"
	"sort"

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
	state := BuildGraphState(repo)

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
func BuildGraphState(repo *gogit.Repository) *GraphState {
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
		// Use BFS from Refs (showAll=false) to ensure HybridStorer works
		// (Iterating objects fails because Local is empty)
		populateCommits(repo, state, false)
		// Let's assume for Shared Remote we want to show everything we have.
		// Actually, populateCommits logic for ancestors might be better.
		// But for "Server View", showing the reachable history from branches is correct.

		// 4. Git Status (Might be empty for bare repos, but harmless)
		if err := populateGitStatus(repo, state); err != nil {
			// Bare repos often fail Worktree(), ignore
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
			state.HEAD = Head{Type: "branch", Ref: "main"} // Default
		} else {
			// Log error or set to none?
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
	iter.ForEach(func(r *plumbing.Reference) error {
		state.Branches[r.Name().Short()] = r.Hash().String()
		return nil
	})

	// Get Remote Branches and Tags from References
	// Note: repo.Tags() sometimes fails while repo.References() works.
	// We do a single pass for robustness.
	refs, err := repo.References()
	if err == nil {
		refs.ForEach(func(r *plumbing.Reference) error {
			if r.Name().IsRemote() {
				state.RemoteBranches[r.Name().Short()] = r.Hash().String()
			} else if r.Name().IsTag() {
				hash := r.Hash().String()
				// Check if it's an annotated tag
				tagObj, err := repo.TagObject(r.Hash())
				if err == nil {
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
	walkPath := session.CurrentDir
	if len(walkPath) > 0 && walkPath[0] == '/' {
		walkPath = walkPath[1:]
	}
	if walkPath == "" {
		walkPath = "."
	}

	infos, err := session.Filesystem.ReadDir(walkPath)
	if err == nil {
		for _, info := range infos {
			name := info.Name()
			if info.IsDir() {
				if name == ".git" {
					continue
				}
				name = name + "/"
			}
			state.Files = append(state.Files, name)
		}
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
