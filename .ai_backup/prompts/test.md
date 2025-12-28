# Test Implementation Checklist

Use this checklist when your primary task is adding or fixing tests.

## 1. Strategy
- [ ] **Identify Gaps**: Which scenarios are missing coverage?
- [ ] **Type of Test**: Unit, Integration, or E2E? (Refer to `docs/` for strategy).

## 2. Implementation
- [ ] **Isolation**: Ensure the test tests *one thing*.
- [ ] **Readability**: The test name should describe the scenario and expected outcome.
- [ ] **Clean Code**: specific setup/teardown logic should be reusable if possible.

## 3. Verification
- [ ] **Green**: Ensure the new checks pass.
- [ ] **Red**: Temporarily break the code to ensure the test *fails* (avoids false positives).
- [ ] **Performance**: Ensure the test doesn't drastically slow down the suite.
