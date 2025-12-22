# GitGym 🏋️‍♂️

GitGym is an interactive, visual sandbox for learning and experimenting with Git commands. It provides a real-time visualization of the Git graph structure alongside a functional terminal interface, allowing users to see exactly how their commands affect the repository state.

![GitGym Interface](frontend/public/vite.svg) <!-- You might want to replace this with a real screenshot later -->

## 🚀 Features

- **Interactive Terminal**: A fully functional web-based terminal to run Git commands.
- **Real-time Visualization**: Instantly see commits, branches, and HEAD movement as you type.
- **Sandboxed Environment**: Experiment safely without affecting your actual projects.
- **Command Support**:
  - `git init`, `status`, `add`, `commit`
  - `git branch`, `checkout`, `switch`
  - `git log`, `diff`
  - `git tag`, `reset`, `clean`
  - `git merge`, `rebase` (basic support)

## 🏗 Architecture

GitGym is built with a modern, modular stack designed for maintainability and performance.

### Frontend (`/frontend`)
- **Framework**: React 19 + TypeScript
- **Build Tool**: Vite
- **Terminal**: Xterm.js
- **Visualization**: Custom SVG-based graph renderer
- **Testing**: Playwright (E2E)

### Backend (`/backend`)
- **Language**: Go (Golang) 1.22+
- **Core Library**: `go-git` (pure Go implementation of Git)
- **Design Pattern**: Command Pattern (encapsulating Git operations)
- **API**: RESTful endpoints for session management and command execution

### Infrastructure
- **Docker**: Full containerization of frontend (Nginx) and backend services.
- **Dev Container**: Ready-to-use development environment for VS Code.

## 🛠 Getting Started

### Prerequisites
- Docker & Docker Compose

### Quick Start
1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/playwithantigravity.git
   cd playwithantigravity
   ```

2. Start the application:
   ```bash
   docker compose up --build
   ```

3. Open your browser:
   - Navigate to [http://localhost](http://localhost) to access GitGym.

### Development Setup
We recommend using **VS Code Dev Containers** for the best experience.
1. Open the project in VS Code.
2. Click "Reopen in Container" when prompted.
3. The environment comes pre-configured with Go, Node.js, and all extensions.

#### Environment Automation (Optional but Recommended)
If you prefer developing outside of a Dev Container, we use `nix` and `direnv` to automate the development environment.
1.  **Install Nix & direnv**: Ensure both are installed on your system.
2.  **Hook direnv to your shell**:
    - For `bash`, add `eval "$(direnv hook bash)"` to your `~/.bashrc`.
    - For `zsh`, add `eval "$(direnv hook zsh)"` to your `~/.zshrc`.
3.  **Allow the environment**: Run `direnv allow` in the project root.
4.  **VS Code Integration**: Install the `mkhl.direnv` extension to make the environment available to the IDE and Antigravity.

## 🧪 Testing

### Backend Tests
Run integration tests for the Git engine and API server:
```bash
cd backend
go test -v ./...
```

### Frontend E2E Tests
Run End-to-End tests using Playwright (requires Dev Container or local Node environment):
```bash
cd frontend
npm run test:e2e
```

## 📂 Project Structure

```
.
├── backend/            # Go backend service
│   ├── cmd/
│   │   └── server/     # Entry point
│   │       └── main.go
│   ├── internal/
│   │   ├── git/        # Core Git logic (commands, session, types)
│   │   └── server/     # HTTP handlers and router
│   └── go.mod
├── frontend/           # React frontend
│   ├── src/
│   │   ├── components/ # UI Components (Terminal, GraphViz)
│   │   ├── context/    # State management
│   │   └── services/   # API abstraction
│   └── tests/          # Playwright E2E specs
├── .devcontainer/      # VS Code Dev Container config
└── docker-compose.yml  # Production/Staging orchestration
```

## 📄 License
MIT
