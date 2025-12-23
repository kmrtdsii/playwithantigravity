# GitGym 🏋️‍♂️

GitGym is an interactive, visual sandbox for learning and experimenting with Git commands. It provides a real-time visualization of the Git graph structure alongside a functional terminal interface, allowing users to see exactly how their commands affect the repository state.

> [!TIP]
> **For AI Agents (Gemini/Claude)**: Please refer to [.ai/context.md](.ai/context.md) for architectural guidelines and project context.

![GitGym Interface](frontend/public/vite.svg)

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

## 📚 Documentation

- **Architecture**: [docs/architecture/](docs/architecture/)
- **Specifications & User Flows**: [docs/specs/](docs/specs/)
- **Setup Guide**: [docs/setup/](docs/setup/)
- **AI Context**: [.ai/](.ai/)

## 🏗 Architecture

GitGym is built with a modern, modular stack designed for maintainability and performance.

### Frontend (`/frontend`)
- **Framework**: React 19 + TypeScript
- **State**: `GitAPIContext` (Flux-like)
- **Terminal**: Xterm.js with "Recorder Pattern"
- **Testing**: Playwright (E2E)

### Backend (`/backend`)
- **Language**: Go 1.25+
- **Core**: `go-git`
- **Pattern**: Command Pattern encapsulated commands
- **API**: RESTful

## 🛠 Getting Started

### Quick Start
1. Clone the repository:
   ```bash
   git clone https://github.com/kurobon/gitgym.git
   cd gitgym
   ```
2. Start with Docker:
   ```bash
   docker compose up --build
   ```
3. Open [http://localhost](http://localhost).

### Development Setup
See [docs/setup/git_environment.md](docs/setup/git_environment.md).

## 📂 Project Structure

```
gitgym/
├── .ai/                # AI Context & Prompts
├── backend/            # Go backend service
│   ├── cmd/server/     # Entry point
│   └── internal/       # Application logic
├── frontend/           # React frontend
│   └── src/            # Source code
├── docs/               # Documentation
├── .devcontainer/      # DevContainer config
└── docker-compose.yml  # Orchestration
```

## 🧪 Testing

- **Backend**: `cd backend && go test ./...`
- **Frontend E2E**: `cd frontend && npm run test:e2e`

## 📄 License
MIT
