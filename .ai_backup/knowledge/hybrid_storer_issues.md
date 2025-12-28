# HybridStorer Architecture & Known Issues

> [!NOTE]
> This document captures lessons learned from debugging issues related to the `HybridStorer` pattern used in GitGym's simulated remote architecture.

## 1. Overview

GitGym uses a **HybridStorer** (`backend/internal/git/storage.go`) to allow cloned repositories to share object storage with the "shared remote" (bare repository). This enables efficient cloning without copying all objects.

### Architecture
```
┌─────────────────────────────────────────────────────────┐
│ User Session Repository (e.g., /Spoon-Knife)            │
│   └─ HybridStorer                                       │
│       ├─ Local Storage (filesystem)  ← Writes go here  │
│       └─ Shared Storage (bare repo)  ← Fallback reads  │
└─────────────────────────────────────────────────────────┘
```

## 2. Known Issues & Fixes

### Issue A: Local Graph Shows Remote-Only Commits (PR Merge Visibility)
**Symptom**: When a PR is merged on the "remote server", the merge commit immediately appears in the local graph view without running `git fetch` or `git pull`.

**Root Cause**: 
- `HybridStorer.EncodedObject()` and `IterEncodedObjects()` fall back to shared storage
- `populateCommits()` in `graph_traversal.go` uses `repo.CommitObject()` which goes through HybridStorer
- Commits created on the shared remote (e.g., PR merge commits) become visible locally

**Fix** (`backend/internal/state/graph_traversal.go`):
```go
// Define interface to avoid import cycle with git package
type localStorerProvider interface {
    LocalStorer() storage.Storer
}

// In BFS loop, check if commit exists locally before adding
hybridSt, isHybrid := repo.Storer.(localStorerProvider)
if isHybrid {
    localSt := hybridSt.LocalStorer()
    if err := localSt.HasEncodedObject(current); err != nil {
        continue // Skip remote-only commits
    }
}
```

**Key Learning**: Use interface-based type assertion to avoid import cycles between `state` and `git` packages.

---

### Issue B: Duplicate Branch Labels (main + origin/main)
**Symptom**: Same commit shows both `(main)` and `(origin/main)` labels in the graph after PR merge.

**Root Cause**:
- Previous `--mirror` clones left `refs/remotes/origin/*` entries in bare repos
- `IngestRemote` was updated to use `+refs/heads/*:refs/heads/*` refspec
- Both old `refs/remotes/*` and new `refs/heads/*` coexisted

**Fix** (`backend/internal/state/actions.go`):
```go
// After successful fetch, clean up stale refs/remotes/*
refs, refErr := r.References()
if refErr == nil {
    var staleRefs []plumbing.ReferenceName
    _ = refs.ForEach(func(ref *plumbing.Reference) error {
        if ref.Name().IsRemote() {
            staleRefs = append(staleRefs, ref.Name())
        }
        return nil
    })
    for _, refName := range staleRefs {
        _ = r.Storer.RemoveReference(refName)
    }
}
```

---

### Issue C: Malformed `git push` Output
**Symptom**: Push output showed `aef0f97 -> main` instead of proper format.

**Root Cause**: Output format string didn't include old hash or proper ref update notation.

**Fix** (`backend/internal/git/commands/push.go`):
```go
// Get old hash for display
oldHashStr := "0000000"
if refName.IsBranch() {
    existingRef, refErr := targetRepo.Reference(refName, true)
    if refErr == nil {
        oldHashStr = existingRef.Hash().String()[:7]
    }
}
return fmt.Sprintf("To %s\n   %s..%s  %s -> %s/%s", 
    pCtx.RemoteURL, oldHashStr, hashToSync.String()[:7], 
    refName.Short(), pCtx.RemoteName, refName.Short()), nil
```

## 3. Testing Considerations

When testing features involving HybridStorer:
1. **Clear remote data** before tests: `rm -rf .gitgym-data/remotes/*`
2. **Restart backend** after clearing to trigger fresh `IngestRemote`
3. **Check logs** for `IngestRemote: Cleaned up X stale remote refs`

## 4. Related Files
- `backend/internal/git/storage.go` - HybridStorer implementation
- `backend/internal/state/graph_traversal.go` - Graph BFS with local filtering
- `backend/internal/state/actions.go` - IngestRemote with cleanup
- `backend/internal/git/commands/push.go` - Push output formatting

---

### Issue D: ShowAll Mode Shows Remote-Only Commits in Local Graph
**Symptom**: When "全表示" (Show All) is enabled, the local graph displays commits that only exist on the shared remote (e.g., PR merge commits that haven't been fetched).

**Root Cause**:
- `populateCommits()` with `showAll=true` calls `repo.CommitObjects()`
- This uses `HybridStorer.IterEncodedObjects()` which returns commits from **both** local and shared storage
- Result: Remote-only commits appear in the local graph

**Fix** (`backend/internal/state/graph_traversal.go`):
```go
// Check if this repo uses HybridStorer
_, isHybrid := repo.Storer.(localStorerProvider)

if showAll && !isHybrid {
    // Only scan ALL objects for non-hybrid repos (e.g., shared bare repo)
    cIter, err := repo.CommitObjects()
    // ...
} else {
    // Use reference-based BFS for HybridStorer repos
    // ...
}
```

**Key Insight**: For `HybridStorer` repos, we must NEVER use object iteration (`CommitObjects()`) because it includes shared storage. Always use reference-based BFS traversal instead.

---

## 5. Design Principle Summary

| Repo Type | Object Storage | Reference Storage | ShowAll Behavior |
|-----------|---------------|-------------------|-----------------|
| Shared Bare Repo | Local only | Local only | Object iteration OK |
| User Local Repo (HybridStorer) | Shared + Local writes | Local only | BFS from refs only |
