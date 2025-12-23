# GitGym - AI Coding Context

> This file provides context for AI coding assistants (Claude, Cursor, Copilot, etc.)

## Project Overview

GitGym is an interactive Git learning sandbox with real-time visualization.

**Architecture:**
- **Frontend**: React 19 + TypeScript + Vite (port 5173)
- **Backend**: Go 1.25 + go-git (port 8080)
- **Pattern**: Command Pattern for Git operations

## Quick Commands

- **[E2E Spec](docs/E2E_SPEC.md)**: Follow this spec for manual verification and UI testing.

```bash
# Frontend
cd frontend && npm run dev      # Start dev server
cd frontend && npm run build    # Production build
cd frontend && npm run lint     # ESLint check
cd frontend && npm run test:e2e # Playwright tests

# Backend
cd backend && go test ./...     # Run tests
cd backend && go run cmd/server/main.go  # Start server

# Docker
docker compose up --build       # Full stack
```

## Project Structure

```
gitgym/
├── backend/
│   ├── cmd/server/main.go           # Entry point
│   └── internal/
│       ├── git/
│       │   ├── commands/            # Command Pattern (add.go, commit.go, ...)
│       │   ├── engine.go            # Command dispatcher
│       │   └── session.go           # Session management
│       └── server/handlers.go       # HTTP handlers
├── frontend/
│   └── src/
│       ├── components/
│       │   ├── layout/              # UI layout components
│       │   ├── terminal/            # xterm.js terminal
│       │   └── visualization/       # Git graph SVG
│       ├── context/GitAPIContext.tsx # State management
│       ├── services/gitService.ts   # API client
│       └── types/gitTypes.ts        # TypeScript types
└── docker-compose.yml
```

## Coding Conventions

### TypeScript/React
- Use functional components with hooks
- Prefer `const` over `let`
- Use TypeScript strict mode (no `@ts-ignore`)
- CSS: CSS Variables for theming (`var(--bg-primary)`)

### Go
- Follow Go Standard Project Layout
- Use docstrings for exported functions
- Commands implement `git.Command` interface
- Session locking: `s.Lock()` / `defer s.Unlock()`

## Key Patterns

### Adding a new Git command (Backend)
```go
// backend/internal/git/commands/newcmd.go
func init() {
    git.RegisterCommand("newcmd", func() git.Command { return &NewCommand{} })
}

type NewCommand struct{}

func (c *NewCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
    s.Lock()
    defer s.Unlock()
    // Implementation
}

func (c *NewCommand) Help() string { return "usage: git newcmd" }
```

### State Flow (Frontend)
```
GitTerminal → runCommand() → gitService.execCommand() → Backend API
                                      ↓
                    GitAPIContext.state ← fetchState()
                                      ↓
                    GitGraphViz re-renders with new commits
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/ping` | GET | Health check |
| `/api/init` | POST | Create session |
| `/api/exec` | POST | Execute command |
| `/api/graph` | GET | Get Git state |
| `/api/remote` | GET | Get remote state |

## Testing

- **E2E**: `frontend/tests/*.spec.ts` (Playwright)
- **Backend**: `backend/internal/**/*_test.go`
- Test IDs: Use unique `data-testid` attributes
