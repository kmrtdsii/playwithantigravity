# GitGym System Context

> [!IMPORTANT]
> This file is the primary source of truth for AI Agents (Gemini, Claude, etc.) working on GitGym. Read this before starting any complex task.

## 1. Project Philosophy
GitGym is an interactive Git learning sandbox designed to visualize the internal state of Git operations in real-time. It bridges the gap between CLI commands and mental models.
- **Visual First**: Every Git operation must have an immediate visual feedback (Graph, File Tree, or Terminal).
- **Simulation**: We use `go-git` to perform actual Git operations in a sandboxed environment, but we simulate "Multi-User" interactions via session isolation.
- **AI-Native**: The codebase is structured to be easily understood and manipulated by AI agents.

## 2. Architecture Overview

### Frontend (React 19 + TypeScript + Vite)
- **State Management**: `GitAPIContext.tsx` acts as the central store. It uses a Flux-like pattern where:
    1.  User acts (Terminal Command / UI Click)
    2.  Action calls `gitService` API
    3.  Backend executes command
    4.  Frontend fetches *fresh state* (`fetchState`) and replaces the local state.
    -   *Note*: We intentionally avoid complex client-side state prediction to ensure truth comes from the backend.
- **Components**:
    -   `GitTerminal`: xterm.js wrapper. Uses "Recorder Pattern" (stores transcript history manually) to ensure exact reproduction of sessions.
    -   `GitGraphViz`: SVGs rendered based on commit history.
    -   `RemoteRepoView`: Handles "origin" interactions.

### Backend (Go 1.25 + go-git)
- **Standard Go Layout**: `cmd/server` (entry), `internal/` (logic).
- **Command Pattern**: All Git operations (add, commit, checkout) implement the `git.Command` interface in `internal/git/commands/`.
    -   *Rule*: New Git features MUST be implemented as new Command structs.
- **Session Management**: Each user (Alice, Bob) has a unique Session ID mapping to a temp directory.
    -   `Lock()`/`Unlock()` is critical for thread safety.

## 3. Directory Structure
```
gitgym/
├── .ai/                # AI Context & Prompts
├── backend/
│   ├── cmd/server/     # Main entry point
│   ├── internal/
│   │   ├── git/        # Core Engine
│   │   │   ├── commands/ # Individual Git Commands
│   │   │   └── engine.go # Dispatcher
│   │   └── server/     # HTTP Handlers
├── frontend/
│   └── src/
│       ├── components/ # UI Components
│       ├── context/    # Global State
│       ├── services/   # API Clients
│       └── types/      # Shared Types
└── docs/               # Architecture & Specs
```

## 4. Development Rules for AI Agents

### General
1.  **Task Boundary**: Always use `task_boundary` to track your progress and mode (PLANNING -> EXECUTION -> VERIFICATION).
2.  **Artifacts**: Use Artifacts (`implementation_plan.md`, `walkthrough.md`) for complex generic tasks.
3.  **No Procrastination**: If you see a Lint error, fix it immediately. If you see a "TODO", question if it should be done now.

### Coding Standards
-   **TypeScript**: Strict typing. No `any`. Use `interface` over `type` for objects.
-   **Go**: Handle all errors. Use `fmt.Errorf` with wrapping (`%w`).
-   **Testing**:
    -   Frontend: Check `frontend/tests/` (Playwright) for regressions.
    -   Backend: Add unit tests for every new Command.

### Commit Messages
-   Use clear English.
-   Format: `type: summary` (e.g., `feat: add git merge support`, `fix: resolve race condition in terminal`).
