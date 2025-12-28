# Bugfix Checklist

Use this checklist when addressing a reported bug or issue.

## 1. Reproduction & Analysis
- [ ] **Reproduce**: Create a minimal reproduction case (test or script) that fails.
- [ ] **Analyze Root Cause**: Understand *why* it failed. Don't just patch the symptom.
- [ ] **Check Docs**: Verify if this is unexpected behavior or a documented limitation.

## 2. Planning
- [ ] **Plan**: Outline the fix. If complex, create `implementation_plan.md`.
- [ ] **Impact Assessment**: What else could break? (Side effects).

## 3. Execution
- [ ] **Fix**: Apply the minimum necessary change to fix the root cause.
- [ ] **Test**: Ensure the reproduction case now passes.
- [ ] **Regression**: Run related test suites to ensure no regressions.

## 4. Documentation
- [ ] **Update Docs**: If the fix changes behavior or assumptions, update `docs/`.
- [ ] **Post-Mortem**: If critical, document what went wrong and how to prevent it in `.ai/knowledge/`.
