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

## 2. Frontend Testing (Playwright)
We use **Playwright** for End-to-End (E2E) testing.

### Philosophy
- **User-Centric**: Test flows (e.g., "User creates a branch"), not implementation details.
- **Visual Assertions**: Since GitGym is a visualization tool, check that the graph updates correctly using locator assertions.

### Structure
- `tests/e2e/`: Contains all E2E specs.
- `tests/fixtures/`: Reusable setups (e.g., "Repo with 10 commits").

### Running Tests
- **CI**: Runs automatically on PRs.
- **Local**:
  ```bash
  npm run test:e2e
  ```
