# Refactoring Guide

Use this prompt/checklist when asked to refactor code.

## 1. Analysis
- [ ] Identify the specific "Smell" or issue (Duplication, Complexity, Stale Reference).
- [ ] Check existing tests. **If no tests exist for the target code, create a characterization test first.**

## 2. Planning
- [ ] Describe the intended structure.
- [ ] Verify that the architectural patterns (e.g., Recorder Pattern in Terminal) remain intact.
- [ ] List touched files.

## 3. Execution Rules
- **Incremental Steps**: Do not rewrite the entire file in one go if it handles multiple concerns.
- **Maintain Interfaces**: Do not change public APIs unless explicitly required.
- **Strict Linting**: Ensure no new lint errors are introduced.

## 4. Verification
- [ ] Run existing tests for regression.
- [ ] Manual verification steps (if UI is involved).
