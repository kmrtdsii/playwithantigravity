package git

import (
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/go-git/go-billy/v5/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// GetGraphState returns the current state of the repository for frontend visualization
func (sm *SessionManager) GetGraphState(sessionID string, showAll bool) (*GraphState, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	state := &GraphState{
		Commits:      []Commit{},
		Branches:     make(map[string]string),
		References:   make(map[string]string),
		FileStatuses: make(map[string]string),
	}

	// 1. Get HEAD
	if session.Repo == nil {
		state.HEAD = Head{Type: "none"}
	} else {
		ref, err := session.Repo.Head()
		if err != nil {
			if err.Error() == "reference not found" {
				state.HEAD = Head{Type: "branch", Ref: "main"} // Default
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
		var collectedCommits []*object.Commit

		if showAll {
			// Scan ALL objects to find every commit
			cIter, err := session.Repo.CommitObjects()
			if err == nil {
				cIter.ForEach(func(c *object.Commit) error {
					collectedCommits = append(collectedCommits, c)
					return nil
				})
			}
		} else {
			// Standard Graph Traversal (Reachable from Branches/Tags/HEAD only)
			seen := make(map[string]bool)
			var queue []plumbing.Hash

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

			// BFS
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
				queue = append(queue, c.ParentHashes...)
			}
		}

		// Pre-compute map for fast lookup in sort
		commitMap := make(map[string]*object.Commit)
		for _, c := range collectedCommits {
			commitMap[c.Hash.String()] = c
		}

		// Helper: Is i ancestor of j? (Is j reachable from i?)
		// i is older (ancestor), j is newer (descendant).
		// SearchBFS: start from j, look for i.
		isAncestor := func(i, j *object.Commit) bool {
			if i.Hash == j.Hash {
				return true
			}
			// BFS queue
			q := []string{j.Hash.String()}
			visited := make(map[string]bool)
			visited[j.Hash.String()] = true
			
			// Limit depth/steps to avoid apparent hang on huge repos with equal timestamps
			steps := 0
			maxSteps := 500

			for len(q) > 0 {
				if steps > maxSteps {
					return false // Assume not ancestor if too far (fallback to hash sort)
				}
				steps++
				
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
			}
			return false
		}

		// Sort commits
		sort.SliceStable(collectedCommits, func(i, j int) bool {
			tI := collectedCommits[i].Committer.When
			tJ := collectedCommits[j].Committer.When
			
			if tI.Equal(tJ) {
				cI := collectedCommits[i]
				cJ := collectedCommits[j]

				// 1. Is i ancestor of j? (i reaches j) -> i is Older -> return false (we want Newest First)
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
			return tI.After(tJ) // Newest first
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

	// 4. Get Status (Files, Staging, Modified)
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
		state.Files = append(state.Files, path)
		return nil
	})

	if session.Repo != nil {
		w, _ := session.Repo.Worktree()
		status, _ := w.Status()
		for file, s := range status {
			// Untracked
			if s.Staging == git.Untracked {
				state.Untracked = append(state.Untracked, file)
			}
			// Modified
			if s.Worktree != git.Unmodified && s.Staging != git.Untracked {
				state.Modified = append(state.Modified, file)
			}
			// Staged
			if s.Staging != git.Unmodified && s.Staging != git.Untracked {
				state.Staging = append(state.Staging, file)
			}
			// Status Codes
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
