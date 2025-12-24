# Feature Implementation Checklist

Use this checklist to ensure a structured approach to feature development.

## 1. Requirement Analysis
- [ ] **Goal**: Understand the user's intent clearly.
- [ ] **Docs First**: Check `docs/` for existing specifications or constraints.
- [ ] **Impact Analysis**: Identify which layers (Frontend, Backend, DB/State) are touched.

## 2. Architecture & Design
- [ ] **Consult Project Guides**: Refer to `docs/development/implementation-guide.md` (or equivalent) for specific file paths and patterns.
- [ ] **Consistency**: Ensure the new design aligns with existing patterns (e.g., Command Pattern, Flux Architecture).

## 3. Implementation Plan
Create `implementation_plan.md` artifact detailing:
- [ ] **Proposed Changes**: List detailed file paths and logical changes.
- [ ] **Verification Strategy**: How will we prove it works? (Unit tests, E2E, manual steps).

## 4. Verification
- [ ] **Unit Tests**: Implement tests for core logic.
- [ ] **Regression**: Ensure existing functionality is not broken.
- [ ] **Documentation**: Update `docs/` if the feature introduces new architecture.
