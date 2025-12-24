# GitGym Implementation Guide

> [!NOTE]
> This guide details the specific patterns and file locations for implementing features in GitGym.

## 1. Adding a New Git Command
To implement a new Git feature (e.g., `git merge` support):

### Backend (`/backend`)
1.  **Create Command**: Add a new file `internal/git/commands/<command_name>.go`.
2.  **Implement Interface**: Must satisfy the `git.Command` interface.
    ```go
    type Command interface {
        Execute(ctx context.Context, s *state.Session, args []string) (string, error)
    }
    ```
3.  **Register**: Add the command to the dispatcher in `internal/git/engine.go`.
4.  **Test**: Create `internal/git/commands/<command_name>_test.go`.

### Frontend (`/frontend`)
1.  **Types**: Update `src/types/gitTypes.ts` if the state shape changes.
2.  **Service**: Update `src/services/gitService.ts` to call the new API endpoint (if applicable).
3.  **Context**: Update `src/context/GitAPIContext.tsx` if global state handling needs to change significantly (prefer avoiding this).

## 2. Refactoring Guidelines
-   **Terminal**: Respect the "Recorder Pattern" in `GitTerminal`. Do not bypass the transcript history mechanism.
-   **State**: Always rely on backend-provided state. Do not perform optimistic updates that might diverge from `go-git`'s reality.

## 3. Verification Strategy
Detailed testing patterns are defined in [Testing Strategy](./testing-strategy.md).

-   **Backend Unit Tests**: Required for every new command logic.
-   **E2E Tests**: Use Playwright (`npm run test:e2e`) if the user flow is affected.
