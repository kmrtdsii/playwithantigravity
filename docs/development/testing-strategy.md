# Testing Strategy

> [!NOTE]
> This document outlines the testing standards for GitGym. All new features must adhere to these patterns.

## 1. Backend Testing (Go)
We use the standard `testing` package with `testify/assert` for assertions.

### Unit Tests
- **Location**: Co-located with code (e.g., `checkout.go` -> `checkout_test.go`).
- **Scope**: Test the `Execute` method of commands.
- **Mocking**: Use `test/mock` packages if you need to mock the filesystem or external git binary (though we prefer using real temp directories for robustness).

### Test Data
- Use `internal/git/test_utils.go` (if available) or helper functions to set up a clean git environment for each test.
- **Do not** rely on shared global state. Each test must spin up its own temp git repo.

```go
func TestCheckout(t *testing.T) {
    repo, worktree := setupTestRepo(t) // Helper creates temp dir
    cmd := &CheckoutCommand{}
    // Act
    _, err := cmd.Execute(ctx, session, []string{"checkout", "-b", "new-branch"})
    // Assert
    assert.NoError(t, err)
    // Verify side effects
    branch, _ := repo.Head()
    assert.Equal(t, "refs/heads/new-branch", branch.Name().String())
}
```

### Simulating Remotes
Since GitGym uses "Simulated Remotes" (local directories in `.gitgym-data`), tests involving network commands (`clone`, `push`, `pull`, `fetch`) must:
1.  **Initialize Remote**: Create a bare repo to act as remote.
2.  **Register Remote**: Use `s.Manager.SharedRemotes` or manually map it in `s.Repos` if testing session-local remotes.
3.  **Panic Guards**: When inspecting references from `go-git` (e.g., `ref.Hash()`), ALWAYS check for `nil` or error first. `go-git` can return nil references in edge cases which cause immediate panics.

#### Example: Remote Setup
```go
// Setup "remote"
remoteSt := memory.NewStorage()
remoteRepo, _ := gogit.Init(remoteSt, nil) // Bare

// Setup "local" and link
s.Repos["local"] = localRepo
_, _ = localRepo.CreateRemote(&config.RemoteConfig{
    Name: "origin",
    URLs: []string{"/remote-path"}, // Internal logic maps this
})
```

## 2. Frontend Testing (Playwright)
We use **Playwright** for End-to-End (E2E) testing.

### Philosophy
- **User-Centric**: Test flows (e.g., "User creates a branch"), not implementation details.
- **Visual Assertions**: Since GitGym is a visualization tool, check that the graph updates correctly using locator assertions.
- **Selectors**: Always use `data-testid` attributes (`page.getByTestId('...')`) for reliable element targeting. Do not use CSS classes or text content which may change.

### Structure
- `tests/e2e/`: Contains all E2E specs.
- `tests/fixtures/`: Reusable setups (e.g., "Repo with 10 commits").

### Running Tests
- **CI**: Runs automatically on PRs.
- **Local**:
  ```bash
  npm run test:e2e
  ```

## 3. Visual Verification (Multimodal)
*See `.ai/guidelines/multimodal_debugging.md`*

For UI-heavy components (like `GitGraphViz.tsx`), standard E2E assertions (`toBeVisible`) are insufficient.
-   **Screenshot Analysis**: Agents should capture screenshots of the graph rendering to verify complex topological sorting or layout issues.
-   **Design-to-Code**: Compare implementation against design mocks (if provided) using visual inspection.

## 4. Adaptive Verification (Scripts)
*See `.ai/patterns/tool_composition.md`*

For complex refactors or audits where standard tests are too slow or rigid, Agents should generate **Adaptive Verification Scripts**.
-   **Use Case**: "Find all functions that ignore errors" or "Verify migration of 50 files".
-   **Method**: Write a temporary Go/Python script to perform the check, run it, and analyze output.
-   **Safety**: Scripts must be read-only or strictly scoped.

## 5. Agent Verification Workflow
As an Antigravity Agent, you must verify your work holistically before requesting user review.
-   **Unified Script**: Run `scripts/test-all.sh` (if available) or manually run:
    1.  Backend Tests + Lint
    2.  Frontend Lint
    3.  Frontend Check
-   **Pattern**:
    1.  Make changes.
    2.  Run `test-all.sh`.
    3.  Fix **ALL** issues.
    4.  Only then create `walkthrough.md`.
