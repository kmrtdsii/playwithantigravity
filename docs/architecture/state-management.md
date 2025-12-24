# State Management Architecture

GitGym employs a **Command-Query Separation** style architecture to ensure the visual graph always represents the true backend state.

## 1. The Core Principle
> **"The Backend is the Single Source of Truth."**

The Frontend **never** simulates the results of a command to update its local state (no Optimistic UI). It:
1.  Sends Command -> Backend.
2.  Backend executes -> Disk IO.
3.  Frontend Fetches State -> Backend.
4.  Frontend Replaces State -> UI Redraw.

This trade-off (latency vs correctness) is chosen because simulating Git's graph logic in JavaScript exactly matching `go-git` is error-prone.

## 2. Frontend State (`GitAPIContext`)
The state is managed via React Context (`src/context/GitAPIContext.tsx`).

### The `GitState` Object
Detailed in `src/types/gitTypes.ts`.
```typescript
interface GitState {
    commits: Commit[];      // All visible commits
    branches: Record<string, string>; // Ref -> SHA
    HEAD: { type: 'branch' | 'commit', ref: string };
    staging: string[];      // 'git status' file list
    // ...
}
```

### The Update Loop
1.  **`runCommand(cmd)`**: User types `git commit -m "msg"`.
2.  **`api.post('/command', { cmd })`**: Request sent.
3.  **Await Response**: Backend returns `output` (stdout text) AND the new `state` (JSON).
4.  **`setState(newState)` React updates, triggering `GitGraphViz` re-render.

## 3. Backend State (`internal/git/`)
The backend is stateless regarding HTTP requests but stateful regarding the filesystem.

### Session Manager
-   **Session ID**: Derived from Cookie/Header.
-   **Isolation**: Maps Session ID -> `/tmp/gitgym-sessions/<id>/`.
-   **Locking**: `Mutex` ensures concurrent requests (e.g., fast typing) don't corrupt the `.git` index.

### Command Pattern
Every Git operation is an immutable command struct.
-   **Input**: `[]string` args (parsed by `pflag` usually).
-   **Execution**: Manipulates `go-git` Repository object.
-   **Output**: Returns `stdout` string. The *State* is not returned by the command itself, but aggregated by the `Engine` after execution.
