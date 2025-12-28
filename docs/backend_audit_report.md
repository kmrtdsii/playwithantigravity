# Backend Audit Report (AI Knowledge Base Alignment)

## 1. Overview
This audit evaluates the current `backend/` implementation against the newly established "Progressive Intelligence" standards in `.ai/` and project rules in `docs/`.

**Auditor**: Gemini Pro 3.0
**Date**: 2025-12-28
**Scope**: `backend/internal/git/commands`, `backend/internal/state`

## 2. Findings Summary

| Category | Status | Key Findings |
| :--- | :--- | :--- |
| **Command Architecture** | 🟢 Compliant | `clone`, `push`, `checkout` strictly follow the "Command Phasing" pattern (`parse` -> `resolve` -> `perform`). |
| **Performance** | 🔴 Critical | `GetGraphState` performs a synchronous full-filesystem walk on every request. |
| **Code Structure** | 🟡 Warning | `checkout.go` is approaching "God Object" status (handling files, commits, branches, orphans). |
| **Security/Config** | 🟡 Warning | Hardcoded paths for `.gitgym-data` prevent flexible deployment or testing isolation. |

## 3. Detailed Observations

### A. Performance Violation: Synchronous Filesystem Walk
*   **Location**: `backend/internal/state/graph.go` -> `populateFiles`
*   **Issue**: The function uses `util.Walk` to traverse the *entire* worktree every time the frontend requests a graph update.
*   **Conflict with `.ai/guidelines/performance_base.md`**: "Avoid reading entire files... utilize concurrency".
*   **Risk**: On large repositories (GitGym's "Infinite Context" target), this will cause UI freezing and API timeouts.
*   **Recommendation**: Implement an **Adaptive Indexer**. Cache the file list and only invalidate on file watcher events or specific git commands.

### B. Context Complexity: `checkout.go` Monolith
*   **Location**: `backend/internal/git/commands/checkout.go`
*   **Issue**: The file handles 4 distinct modes (`modeFiles`, `modeOrphan`, `modeNewBranch`, `modeRefOrPath`) in one file ~400 lines long.
*   **Conflict with `.ai/meta/standard_context/efficient_loading.md`**: "Split monolithic files into smaller modules... helps LLMs reason about code".
*   **Recommendation**: Refactor into strategies: `internal/git/strategies/checkout/branch.go`, `files.go`, etc.

### C. Configuration Hardcoding
*   **Location**: `backend/internal/state/actions.go` -> `IngestRemote`
*   **Issue**: `baseDir = ".gitgym-data/remotes"` is hardcoded.
*   **Conflict with `.ai/guidelines/security_base.md`**: "Environment Variables... for config/secrets".
*   **Recommendation**: Use `os.Getenv("GITGYM_DATA_ROOT")` with a default fallback.

## 4. Proposed Action Plan (Draft)

1.  **Refactor `graph.go`**: Introduce `FileCache` struct to store file tree state. Update it asynchronously.
2.  **Split `checkout.go`**: Extract logic into `internal/git/domain/checkout/`.
3.  **Configvar**: Add `config.DataDir` to global config loader.

---
**Request for Feedback**:
Claude Opus 4.5, please review these findings. Do you agree with the prioritization of the Performance issue over the Refactoring issue?
