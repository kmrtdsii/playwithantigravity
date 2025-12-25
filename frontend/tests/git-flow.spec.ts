import { test, expect } from '@playwright/test';

test.describe('GitGym Critical User Journey', () => {
    test.beforeEach(async ({ page }) => {
        // Ensure we start with a fresh session if possible, 
        // but since we can't easily reset backend state per test without API,
        // we'll assume a fresh load acts as a new user session or handles existing state gracefully.
        // For now, verified against "GitGym" title which implies app loaded.
        await page.goto('/');
    });

    test('should load the application and show correct title', async ({ page }) => {
        // Verify changes from "GitForge" to "GitGym"
        await expect(page).toHaveTitle(/GitGym/);

        // Check key elements exist
        await expect(page.getByText('User:')).toBeVisible();
    });

    test('should initialize repository via terminal', async ({ page }) => {
        // Wait for terminal to be ready
        const terminalWrapper = page.getByTestId('terminal-canvas-container');
        await expect(terminalWrapper).toBeVisible({ timeout: 30000 });

        // Type "git init"
        // Interacting with xterm canvas requires focus.
        await terminalWrapper.click();
        await page.keyboard.type('git init');
        await page.keyboard.press('Enter');

        // Verify output in terminal
        // Xterm renders text consistently so we can check for "Initialized"
        // Note: xterm canvas is hard to read directly, but we can check the accessible buffer if exposed,
        // or rely on side effects (like the graph empty state disappearing).

        // Verify Graph State: Empty state should be visible or consistent with 0 commits
        // Actually, git init -> 0 commits. The "Type git init to start" message might change to "No commits yet" or similar?
        // Wait, GitGraphViz logic: if (!state.initialized) show "Type git init".
        // After git init, state.initialized becomes true.
        // If 0 commits, computeLayout returns empty nodes.
        // So the "git init" message should DISAPPEAR.
        await expect(page.getByTestId('git-graph-empty')).not.toBeVisible({ timeout: 10000 });

        // Also check prompt update if possible (e.g. branch name in terminal)
        // But verifying graph update on commit is more robust.
    });

    test('should create a commit and render graph node', async ({ page }) => {
        // Assume shared session state or fast replay. 
        // For robustness, ensure we are in a repo.
        const terminalWrapper = page.getByTestId('terminal-canvas-container');
        await terminalWrapper.click();

        // 1. git init (idempotent-ish if already done, but safe to run)
        await page.keyboard.type('git init');
        await page.keyboard.press('Enter');
        await page.waitForTimeout(500); // Short debounce

        // 2. git commit -m "First Commit"
        // Note: GitGym usually requires "git add" first implicitly or explicitly.
        // Let's assume "--allow-empty" is supported or we just commit.
        // If not, we might need to touch a file. 
        // Standard git requires staged changes.
        // GitGym simplifications might allow direct commit? 
        // Let's try `git commit --allow-empty -m "Initial"`
        await page.keyboard.type('git commit --allow-empty -m "Initial"');
        await page.keyboard.press('Enter');

        // 3. Verify Graph Update
        // Should wait for a commit row to appear
        const commitRow = page.getByTestId('commit-row').first();
        await expect(commitRow).toBeVisible({ timeout: 10000 });

        // 4. Verify Message
        const message = commitRow.getByTestId('commit-message');
        await expect(message).toHaveText('Initial');
    });
});
