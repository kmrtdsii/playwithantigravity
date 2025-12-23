# Feature Implementation Guide

Use this prompt/checklist when implemented a new feature.

## 1. Requirement Analysis
- [ ] Understand the User Goal.
- [ ] Identify touched components (Frontend UI, Backend API, Database/Git State).
- [ ] **Check Existing Specs**: Does this conflict with `docs/specs/*.md`?

## 2. Architecture & Design
- **Backend**:
    - If adding a Git command: Create `internal/git/commands/<cmd>.go`. Implement `git.Command`.
    - Register in `internal/git/engine.go`.
- **Frontend**:
    - Add types in `types/gitTypes.ts`.
    - Update `gitService.ts`.
    - Update `GitAPIContext.tsx` if global state handling changes.

## 3. Implementation Plan
Create `implementation_plan.md` artifact detailing:
- [ ] Backend Changes
- [ ] Frontend Changes
- [ ] Verification Strategy

## 4. Verification
- [ ] Add Backend Unit Test (`_test.go`).
- [ ] Update E2E Spec if user flow changes.
