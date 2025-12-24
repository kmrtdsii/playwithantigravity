# GitGym Application Overview

GitGym is an interactive, visual sandbox for learning and experimenting with Git commands. It simulates a Git environment where users can execute commands in a terminal and see real-time updates in a graphical interface, covering both local and remote repository interactions.

## 🏗 Architecture

GitGym operates as a single-page application (SPA) backed by a stateful Go service.

```mermaid
graph TD
    User[User] -->|Browser| Frontend
    Frontend[React Frontend] -->|REST API| Backend[Go Backend]
    Backend -->|go-git| LocalMem[Session Memory (Local Repo)]
    Backend -->|go-git| RemoteFS[Pseudo-Remote (Persistent FS)]
    
    subgraph "Backend Service"
        API[API Handler]
        Session[Session Manager]
        Cmd[Command Pattern]
        
        API --> Session
        Session --> Cmd
        Cmd --> LocalMem
        Cmd --> RemoteFS
    end
```

### 1. Frontend (`/frontend`)
Built with **React 19** and **TypeScript**.
- **Visualizations**:
  - **Local Graph**: Shows the commit history, branches, and tags of the user's "local" repository.
  - **Remote Graph**: Visualizes the state of the simulated remote server (`origin`).
  - **Ghost Mode**: Displays "potential commits" (simulated nodes) for dry-run operations.
- **Terminal**: Interactive implementation using `Xterm.js`. It intercepts shell commands and sends them to the backend for processing.
- **Workspace**: Allows context switching between different "projects" (repositories).

### 2. Backend (`/backend`)
Built with **Go (Golang)** using `go-git/v5`.
- **Session Management**: Isolates user environments. Each session maintains its own set of in-memory repositories.
- **Command Pattern**: Git commands (`clone`, `pull`, `push`, etc.) are encapsulated as distinct command objects in `internal/git/commands`.
- **Pseudo-Remote Architecture**:
  - Simulates a remote server (e.g., GitHub) using **local bare repositories** stored in `.gitgym-data/remotes`.
  - **Isolation**: Prevents operations from touching actual upstream servers.
  - **Single Remote Policy**: Enforces a strict one-to-one mapping between a configured URL and the simulated remote path. Switching URLs wipes the previous remote data.

## 🚀 Key Features

### Simulated Environment
- **In-Memory & Persistent Hybrid**: "Local" repositories are often ephemeral/in-memory, while "Remote" repositories are persistent on disk to allow realistic `clone`/`push` lifecycles across sessions.
- **Interception**: High-level Git commands are intercepted and executed via API calls rather than shelling out to a system `git` binary.

### "Ghost Mode" & Simulation
- **Dry-Run**: Commands like `merge`, `rebase`, and `push` can run in simulation mode.
- **Visualization**: The frontend renders these simulated future states as "Ghost Nodes" (dashed outlines), allowing users to verify operations before committing.

### Pseudo-Remote Workflow
1.  **Ingest**: User provides a real GitHub URL.
2.  **Mirror**: Backend clones a **bare** copy to local storage (`.gitgym-data/remotes/<hash>`).
3.  **Interaction**: User runs `git clone <url>` in the terminal. The backend redirects this to the local bare path.
4.  **Push/Pull**: All subsequent network operations happen locally against this bare repository.

## 🛠 Tech Stack

| Component | Technology | Description |
|-----------|------------|-------------|
| **Frontend** | React 19, TypeScript, Vite | Modern UI library with strict typing. |
| **State** | React Context (Flux-like) | Manages global git state and command outputs. |
| **Terminal** | Xterm.js | Web-based terminal emulator. |
| **Backend** | Go 1.25+ | High-performance, statically typed backend. |
| **Git Core** | go-git (Pure Go) | Git implementation without C dependencies. |
| **Container** | Docker | Unified development and deployment environment. |
