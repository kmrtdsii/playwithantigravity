# Chore/Maintenance Checklist

Use this checklist for maintenance tasks, dependency updates, or minor cleanups.

## 1. Scope
- [ ] **Goal**: Clearly define what is being cleaned up or updated.
- [ ] **Risk**: Is this purely cosmetic/internal, or could it affect runtime behavior?

## 2. Execution
- [ ] **Update**: Perform the update or cleanup.
- [ ] **Consistency**: Ensure the change is applied consistently across the codebase (e.g., if renaming a variable, rename it everywhere).

## 3. Verification
- [ ] **Build**: Ensure the project still compiles/builds.
- [ ] **Test**: Run all tests. Chores should generally *not* break tests unless the tests themselves are being updated.
- [ ] **Lint**: Ensure no new lint errors are introduced.
