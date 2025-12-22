package git

import (
	"errors"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-git/go-git/v5/storage"
)

// WorktreeStorer wraps an underlying Storer to provide a separate HEAD and Index,
// conforming to standard git worktree behavior where refs are shared but HEAD/Index are local.
type WorktreeStorer struct {
	storage.Storer // Embed storage.Storer to satisfy the full interface

	// Local overrides
	headRef *plumbing.Reference
	idx     *index.Index
}

// NewWorktreeStorer creates a new wrapper.
func NewWorktreeStorer(s storage.Storer) *WorktreeStorer {
	return &WorktreeStorer{
		Storer:  s,
		headRef: nil,
		idx:     &index.Index{Version: 2},
	}
}

// --- ReferenceStorer Overrides ---

func (ws *WorktreeStorer) SetReference(ref *plumbing.Reference) error {
	if ref.Name() == plumbing.HEAD {
		ws.headRef = ref
		return nil
	}
	return ws.Storer.SetReference(ref)
}

func (ws *WorktreeStorer) Reference(n plumbing.ReferenceName) (*plumbing.Reference, error) {
	if n == plumbing.HEAD {
		if ws.headRef == nil {
			return nil, plumbing.ErrReferenceNotFound
		}
		return ws.headRef, nil
	}
	return ws.Storer.Reference(n)
}

func (ws *WorktreeStorer) CheckAndSetReference(new, old *plumbing.Reference) error {
	if new.Name() == plumbing.HEAD {
		// Simple atomic check for local HEAD
		if ws.headRef == nil {
			if old != nil {
				return plumbing.ErrReferenceNotFound
			}
		} else {
			if old != nil && ws.headRef.Hash() != old.Hash() {
				return errors.New("reference has changed")
			}
		}
		ws.headRef = new
		return nil
	}
	return ws.Storer.CheckAndSetReference(new, old)
}

// IterReferences: We should ideally inject our HEAD into the iterator,
// and potentially filter out the underlying storer's HEAD?
// For simplicity, we just return the underlying iterator.
// Standard `git log` lookup usually uses `HEAD` explicitly or branches.
// If `IterReferences` returns the "Main" HEAD, visualization might show it, but our `Reference("HEAD")` will return ours.
func (ws *WorktreeStorer) IterReferences() (storer.ReferenceIter, error) {
	// Ideally we wrap this to replace HEAD, but go-git's Iter interfaces are complex to wrap manually.
	// Let's rely on fallback: most operations resolve HEAD directly via Reference().
	return ws.Storer.IterReferences()
}

// --- IndexStorer Overrides ---

func (ws *WorktreeStorer) Index() (*index.Index, error) {
	if ws.idx == nil {
		ws.idx = &index.Index{Version: 2}
	}
	return ws.idx, nil
}

func (ws *WorktreeStorer) SetIndex(idx *index.Index) error {
	ws.idx = idx
	return nil
}

// --- ConfigStorer Overrides ---
// We try to delegate to the underlying storer if it supports ConfigStorer.
// Otherwise we return a default empty config or error.

func (ws *WorktreeStorer) Config() (*config.Config, error) {
	if cs, ok := ws.Storer.(config.ConfigStorer); ok {
		return cs.Config()
	}
	// Fallback: return empty config to satisfy interface if needed, or error
	return config.NewConfig(), nil
}

func (ws *WorktreeStorer) SetConfig(c *config.Config) error {
	if cs, ok := ws.Storer.(config.ConfigStorer); ok {
		return cs.SetConfig(c)
	}
	return nil // No-op if underlying doesn't support config
}
