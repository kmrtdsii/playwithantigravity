package git

import (
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage"
)

// HybridStorer implements storage.Storer by embedding the Local storer.
// It overrides specific ObjectRead methods to delegate to Shared if not found locally.
type HybridStorer struct {
	storage.Storer // Embed Local storer to inherit all standard methods (Refs, Config, Index, etc.)
	Shared         storage.Storer
}

func NewHybridStorer(local, shared storage.Storer) *HybridStorer {
	return &HybridStorer{
		Storer: local,
		Shared: shared,
	}
}

// -- ObjectStorer Overrides (Blobs, Trees, Commits) --

// EncodedObject tries Local first, then Shared
func (s *HybridStorer) EncodedObject(t plumbing.ObjectType, h plumbing.Hash) (plumbing.EncodedObject, error) {
	// Try Local first
	obj, err := s.Storer.EncodedObject(t, h)
	if err == nil {
		return obj, nil
	}

	// If not found locally, try Shared
	return s.Shared.EncodedObject(t, h)
}

// EncodedObjectSize tries Local first, then Shared
func (s *HybridStorer) EncodedObjectSize(h plumbing.Hash) (int64, error) {
	sz, err := s.Storer.EncodedObjectSize(h)
	if err == nil {
		return sz, nil
	}
	return s.Shared.EncodedObjectSize(h)
}

// HasEncodedObject checks Local first, then Shared
func (s *HybridStorer) HasEncodedObject(h plumbing.Hash) (err error) {
	err = s.Storer.HasEncodedObject(h)
	if err == nil {
		return nil
	}
	return s.Shared.HasEncodedObject(h)
}

// IterEncodedObjects - We likely only want to iterate LOCAL objects for most operations
// (e.g. GC, Pack). Iterating SHARED objects would be huge.
// Since we embed storage.Storer, this method is automatically delegated to s.Storer.IterEncodedObjects
// unless we override it.
// We DO NOT override it, so it defaults to Local iteration.
