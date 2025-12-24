# Test Generation Prompt

## Goal
Generate robust unit or integration tests for the provided code.

## Context
- **Framework**: (Verify project tech stack, e.g., Jest/Vitest for TS, `testing` package for Go)
- **Standards**: Follow `.ai/guidelines/coding_standards.md` (e.g., naming `TestXxx`).

## Steps
1. **Analyze**: specific edge cases and happy paths.
2. **Mocking**: Identify external dependencies to mock.
3. **Implementation**: Write the test code.
4. **Coverage**: Aim for high branch coverage.

## Output Format
- Complete generic test file or test function.
