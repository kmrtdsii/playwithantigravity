package state

import (
	"sort"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func populateCommits(repo *gogit.Repository, state *GraphState, showAll bool) {
	var collectedCommits []*object.Commit

	if showAll {
		// Scan ALL objects to find every commit
		cIter, err := repo.CommitObjects()
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

		// 1. Seed with ALL Refs (HEAD, Branches, Tags, Remotes)
		// This ensures we show "Active" branches even if they are not merged into HEAD.

		// HEAD
		h, err := repo.Head()
		if err == nil {
			queue = append(queue, h.Hash())
		}

		// Local Branches
		bIter, err := repo.Branches()
		if err == nil {
			bIter.ForEach(func(r *plumbing.Reference) error {
				queue = append(queue, r.Hash())
				return nil
			})
		}

		// Remote Branches
		// Note: repo.References() includes everything, but we can filter or just add them.
		// Adding all refs is safer for visibility.
		refs, err := repo.References()
		if err == nil {
			refs.ForEach(func(r *plumbing.Reference) error {
				// We want remotes and tags specifically if not covered above
				if r.Name().IsRemote() {
					queue = append(queue, r.Hash())
				} else if r.Name().IsTag() {
					// Resolve annotated tag for seeding
					hash := r.Hash()
					tagObj, err := repo.TagObject(hash)
					if err == nil {
						hash = tagObj.Target
					}
					queue = append(queue, hash)
				}
				return nil
			})
		}

		// BFS
		for len(queue) > 0 {
			if len(collectedCommits) >= 20000 {
				break
			}
			current := queue[0]
			queue = queue[1:]

			if seen[current.String()] {
				continue
			}
			seen[current.String()] = true

			c, err := repo.CommitObject(current)
			if err != nil {
				continue
			}

			collectedCommits = append(collectedCommits, c)
			queue = append(queue, c.ParentHashes...)
		}
	}

	// Helper for Ancestry
	commitMap := make(map[string]*object.Commit)
	for _, c := range collectedCommits {
		commitMap[c.Hash.String()] = c
	}

	isAncestor := func(i, j *object.Commit) bool {
		if i.Hash == j.Hash {
			return true
		}
		q := []string{j.Hash.String()}
		visited := make(map[string]bool)
		visited[j.Hash.String()] = true

		steps := 0
		maxSteps := 500

		for len(q) > 0 {
			if steps > maxSteps {
				return false
			}
			steps++

			currID := q[0]
			q = q[1:]

			if currID == i.Hash.String() {
				return true
			}

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
			if isAncestor(cI, cJ) {
				return false
			}
			if isAncestor(cJ, cI) {
				return true
			}
			return cI.Hash.String() > cJ.Hash.String()
		}
		return tI.After(tJ)
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
			TreeID:         c.TreeHash.String(),
		})
	}
}
