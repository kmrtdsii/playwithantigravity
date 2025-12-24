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
        // Use .xterm class which wrapper for the terminal
        const terminal = page.locator('.xterm');
        await expect(terminal).toBeVisible({ timeout: 30000 });

        // Type "git init"
        // Interacting with xterm canvas requires focus.
        await terminal.click();
        await page.keyboard.type('git init');
        await page.keyboard.press('Enter');

        // Verify output in terminal (by checking the DOM for xterm rows containing text)
        // Xterm renders text in localized rows.
        await expect(page.locator('.xterm-rows')).toContainText(/(initialized .* Git repository|Git repository .* initialized)/i, { timeout: 10000 });

        // Verify Graph Update
        // Note: git init creates an empty repo with 0 commits, so the graph nodes (and HEAD badge) 
        // will NOT be rendered until a commit is made.
        // We verify success via terminal output (already done above).
    });
});
