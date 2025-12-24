# Codebase Map

This document provides a high-level map of the GitGym codebase. Use this to locate logic quickly.

## 📂 Root
- `.ai/`: Antigravity Agent Brain (SOPs, Knowledge).
- `docs/`: Project Documentation (Architecture, Development Guides, Specs).
- `backend/`: Go Backend Service.
- `frontend/`: React Frontend Application.

## 🖥️ Frontend (`frontend/src`)
The frontend is a React 19 application built with Vite.

### Core Structure
- **`src/main.tsx`**: Application Entry Point.
- **`src/App.tsx`**: Main Layout & Routing.
- **`src/types/gitTypes.ts`**: **CRITICAL**. Defines the shared data model (`GitState`, `Commit`) between Frontend/Backend.

### Components (`src/components/`)
- **`layout/`**: Structural panels.
    - `FileExplorer.tsx`: Left pane (Projects, Branches).
    - `GitTerminal.tsx`: Xterm.js integration. **See "Recorder Pattern"**.
    - `RemoteRepoView.tsx`: Simulator for "Remote" repo visualization.
- **`visualization/`**: D3/SVG Rendering.
    - `GitGraphViz.tsx`: The main commit graph renderer. Uses purely deterministic layout calculation from `GitState`.

### State & Logic
- **`src/context/GitAPIContext.tsx`**: Global State Store.
    - Holds `serverState` (The Truth from Backend).
    - Exposes `runCommand` (The Action).
- **`src/services/gitService.ts`**: API Client Layer. Fetches state from Backend API.

## ⚙️ Backend (`backend/`)
The backend is a Go service using standard layout.

### Entry Point
- **`cmd/server/main.go`**: HTTP Server & Dependency Injection.

### Core Logic (`internal/`)
- **`internal/git/`**: The Git Engine.
    - **`engine.go`**: Dispatcher. Routes string commands `git commit ...` to specific Command structs.
    - **`commands/`**: **CRITICAL**. One file per Git Command (e.g., `clone.go`, `push.go`).
        - *Rule*: All business logic lives here.
- **`internal/state/`**: Session & Persistence.
    - **`session.go`**: Managing User Sessions (in-memory/temp dir).
    - **`actions.go`**: "IngestRemote" logic (Pseudo-Remote architecture).
- **`internal/server/`**: HTTP Handlers.
    - **`api.go`**: REST Endpoints mapping to Engine calls.
