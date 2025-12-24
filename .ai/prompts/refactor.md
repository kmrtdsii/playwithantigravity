# Refactoring Checklist

Use this checklist when improving existing code integrity.

## 1. Analysis
- [ ] **Identify the Smell**: Is it duplication? Complexity? Poor naming?
- [ ] **Safety Net**: Check existing tests. **If no tests exist, create a characterization test first.**

## 2. Planning
- [ ] **Strategy**: Describe the intended structure vs current structure.
- [ ] **Architecture Check**: Ensure refactoring respects core architectural constraints (Consult `docs/architecture/`).
- [ ] **Scope**: List touched files to ensure the scope is manageable.

## 3. Execution Rules
- **Incremental Steps**: Make small, safe moves. Don't rewrite the world effectively in one go.
- **Maintain Interfaces**: Public APIs should remain stable unless a breaking change is intended and approved.
- **Strict Linting**: Fix existing lints in touched files; introduce no new ones.

## 4. Verification
- [ ] **Regression Testing**: Run existing tests to ensure behavior is unchanged.
- [ ] **Cleanup**: Remove dead code or old artifacts.
