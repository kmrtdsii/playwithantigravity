# Technology Stack & Environment

## Core Environment
- **Operating System**: macOS (Dev), Linux (Prod).
- **Package Management**: Nix (via `flake.nix`).
- **Shell**: zsh.

## Backend
- **Language**: Go.
- **Framework**: Standard Library / internal packages.
- **Database**: (Check if any used, e.g., SQLite, Postgres).
- **Tools**: `exclude-go` (for excluding files).

## Frontend
- **Framework**: React.
- **Build Tool**: Vite.
- **Language**: TypeScript (`.ts`, `.tsx`).
- **Styling**: Vanilla CSS (CSS Variables), no Tailwind unless requested.

## Testing
- **Backend**: Go `testing` package.
- **Frontend**: Vitest / Playwright (E2E).

## AI Agent Tools
- **Antigravity**: Agentic coding environment.
- **Excludes**: `.ai` folder is for generic skills, `docs` for project context.
