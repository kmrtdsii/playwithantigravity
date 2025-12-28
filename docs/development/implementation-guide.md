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

## 2.1 Recommended Workflow for AI Agents
Agents are encouraged to use **Flash Mode** (see `.ai/patterns/flash_mode.md`) to accelerate development, provided the final output strictly adheres to the Command Phasing pattern.

### The "Draft & Seal" Protocol
1.  **Draft phase (Flash Mode)**:
    -   Write the specific command logic rapidly in a single pass.
    -   Ignore strict linting or splitting into sub-functions temporarily.
    -   *Goal*: Get `go build` passing and the feature working.
2.  **Seal phase (Refinement)**:
    -   Refactor the working draft into `parseArgs`, `resolveContext`, `performAction`.
    -   Apply project lints and style guides.
    -   *Goal*: Compliance with Section 2.1 Command Pattern standards.

## 2.2 Backend Command Pattern (Standardization)
All commands in `internal/git/commands/` must follow the **Command Phasing** pattern to ensure consistency and testability.

### Phase 1: Parse Arguments (`parseArgs`)
-   **Input**: `[]string` args.
-   **Output**: `*Options` struct, `error`.
-   **Responsibility**:
    -   Parse flags (using manual switch or flag set, but consistent).
    -   Handle `--help` or `-h` by returning the specific error `fmt.Errorf("help requested")`. The `Execute` method must catch this.
    -   Validate basic inputs (e.g., missing required args).

### Phase 2: Resolve Context (`resolveContext`)
-   **Input**: `*git.Session`, `*Options`.
-   **Output**: `*Context` struct (e.g., `pushContext`), `error`.
-   **Responsibility**:
    -   Interact with `go-git` to resolve references, commits, or remotes.
    -   Perform "Read-Only" checks (e.g., does the branch exist? is the remote valid?).
    -   **No side effects** allowed here.

### Phase 3: Execute Action (`performX`)
-   **Input**: `*Context`, `*Options`.
-   **Output**: `string` (user message), `error`.
-   **Responsibility**:
    -   Perform the actual write operations (Commit, Push, Reset).
    -   Return the success message.

### Example `Execute` Implementation
```go
func (c *MyCommand) Execute(ctx context.Context, s *git.Session, args []string) (string, error) {
    s.Lock()
    defer s.Unlock()

    // 1. Parse
    opts, err := c.parseArgs(args)
    if err != nil {
        if err.Error() == "help requested" {
            return c.Help(), nil
        }
        return "", err
    }

    // 2. Resolve
    mCtx, err := c.resolveContext(s, opts)
    if err != nil {
        return "", err
    }

    // 3. Perform
    return c.performAction(s, mCtx)
}

## 2.3 Pragmatic Performance Guards (The "Cap" Rule)
When implementing potentially expensive operations (e.g., file system walks, graph traversals):
1.  **Hard Stop**: Always implement a hard limit (e.g., `MaxFileCount = 1000`) for synchronous operations.
2.  **Fail Gracefully**: If the limit is reached, return partial results with a clear indicator (e.g., `... (limit reached)`).
3.  **Defer Complexity**: Do not implement complex async indexers until the simple cap is proven insufficient. Avoid "Premature Optimization".

## 2.4 Configuration Hygiene
-   **Env Var First**: All filesystem paths and critical constants MUST be overridable via Environment Variables (e.g., `GITGYM_DATA_ROOT`).
-   **Safe Defaults**: Always provide a sensible default if the env var is unset.
-   **No Hardcoding**: Never hardcode absolute paths or "magic folders" deeper than the project root default.

## 3. Debugging Utilities

When debugging `go-git` operations, use these patterns in test files or temporary scripts:

### Inspecting Branches and Tags
```go
repo, _ := git.PlainOpen(path)

// List branches
iter, _ := repo.Branches()
_ = iter.ForEach(func(r *plumbing.Reference) error {
    fmt.Println(r.Name(), r.Hash())
    return nil
})

// List tags (including annotated)
tIter, _ := repo.Tags()
_ = tIter.ForEach(func(r *plumbing.Reference) error {
    fmt.Printf("Tag: %s, Hash: %s\n", r.Name().Short(), r.Hash())
    // Check if annotated
    if tagObj, err := repo.TagObject(r.Hash()); err == nil {
        fmt.Printf("  -> Annotated! Target: %s\n", tagObj.Target)
    }
    return nil
})

// Check HEAD
h, _ := repo.Head()
fmt.Println(h.Name(), h.Hash())
```

## 4. Verification Strategy
Detailed testing patterns are defined in [Testing Strategy](./testing-strategy.md).

-   **Backend Unit Tests**: Required for every new command logic.
-   **E2E Tests**: Use Playwright (`npm run test:e2e`) if the user flow is affected.

