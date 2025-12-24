# GitGym Architecture & Core Concepts

## 1. Project Philosophy
GitGym is an interactive Git learning sandbox designed to visualize the internal state of Git operations in real-time.
- **Visual First**: Every Git operation must have an immediate visual feedback (Graph, File Tree, or Terminal).
- **Simulation**: We use `go-git` to perform actual Git operations in a sandboxed environment, but we simulate "Multi-User" interactions via session isolation.

## 2. Architecture Overview

### Frontend (React 19 + TypeScript + Vite)
- **State Management**: `GitAPIContext.tsx` acts as the central store. It uses a Flux-like pattern where:
    1.  User acts (Terminal Command / UI Click)
    2.  Action calls `gitService` API
    3.  Backend executes command
    4.  Frontend fetches *fresh state* (`fetchState`) and replaces the local state.
    -   *Rule*: We intentionally avoid complex client-side state prediction to ensure truth comes from the backend.
- **Terminal**: xterm.js wrapper using a "Recorder Pattern" to ensure exact session reproduction.
- **Visualizations**: D3/SVG based graphs for commit history (`GitGraphViz`).

### Backend (Go 1.25 + go-git)
- **Structure**: Follows idiomatic Go layout (`cmd/server`, `internal/`).
- **Command Pattern**: All Git features are implemented as atomic `Command` structs in `internal/git/commands/`.
- **Session Isolation**: Each user session has a distinct directory in `/tmp` (or configured loc), ensuring thread safety via `SessionManager`.

## 3. Pseudo-Remote Architecture
To simulate a remote server (like GitHub) without external dependencies:

- **Ingest**: Valid GitHub URLs are cloned into **persistent bare repositories** on disk (`.gitgym-data/remotes/`).
- **Isolation**:
    - The `origin` remote in the user's session is configured to point to these local file paths, **not** the real HTTPS URL.
    - `git push` updates the local bare repo. It **never** pushes to the real GitHub.
- **Switching**: Creating a new remote with a different URL wipes the old bare repository to enforce a 1:1 mapping and clean state.
