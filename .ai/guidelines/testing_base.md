# GenAI Native Testing Strategy

> [!NOTE]
> This document outlines the testing strategy optimized for AI-driven development. AI agents perform best when tests are stable, deterministic, and self-documenting.

## 1. Stability over implementation
Tests should verify *intent*, not *implementation*.
-   **Stable Selectors**: Never use CSS classes (e.g., `.xterm`, `.btn-primary`) for logic. They change often.
    -   **Good**: `data-testid="terminal-input"`, `role="button" name="submit"`
    -   **Bad**: `div > span.icon`, `.xterm-rows`
-   **Visual Regression**: For complex UIs like the Git Graph, use snapshot testing cautiously; prefer verifying the *existence* of DOM nodes (nodes, edges) over exact pixel matching unless strictly necessary.

## 2. Determinism & Isolation
AI Agents get confused by flaky tests that fail intermittently.
-   **Atomic Resets**: Every test must start from a clean slate.
    -   Use API hooks (e.g., `request.post('/api/reset')`) in `beforeEach`.
    -   Do not rely on the browser's previous state.
-   **Explicit Waits**: Avoid fixed `setTimeout`. Wait for specific UI states (`toBeVisible()`, `toHaveText()`).

## 3. Test as Documentation
The test file itself is a "specification" for the AI.
-   **Descriptive Names**: `test('git init creates 0.git directory')` is better than `test('init works')`.
-   **Step-by-Step flow**: Keep tests linear. Intermediate assertions help the AI pinpoint exactly *where* a complex flow failed.

## 4. Playwright Standards
-   Use `page.locator` with `data-testid` preferred.
-   Use `await expect(...)` for all assertions.
-   Keep tests independent (no shared mutable variables outside `test.describe`).
