import { test, expect } from '@playwright/test';

test.describe('Race Condition Fix', () => {
	test('should load agents without network errors on initial page load', async ({ page }) => {
		const consoleErrors: string[] = [];
		
		// Capture console errors
		page.on('console', msg => {
			if (msg.type() === 'error') {
				consoleErrors.push(msg.text());
			}
		});

		// Navigate to the page
		await page.goto('http://localhost:5188');

		// Wait for the stats bar to be visible (indicates page loaded)
		await page.waitForSelector('[data-testid="stats-bar"]', { timeout: 5000 });

		// Wait a bit for any delayed fetches to complete
		await page.waitForTimeout(2000);

		// Check that no "Failed to fetch agents" errors occurred
		const fetchErrors = consoleErrors.filter(err => 
			err.includes('Failed to fetch agents') || 
			err.includes('NetworkError')
		);

		expect(fetchErrors).toHaveLength(0);
	});

	test('should handle multiple page reloads without race condition errors', async ({ page }) => {
		const allErrors: string[] = [];

		page.on('console', msg => {
			if (msg.type() === 'error') {
				allErrors.push(msg.text());
			}
		});

		// Reload page 3 times to test consistency
		for (let i = 0; i < 3; i++) {
			await page.goto('http://localhost:5188');
			await page.waitForSelector('[data-testid="stats-bar"]', { timeout: 5000 });
			await page.waitForTimeout(1000);
		}

		// Check that no fetch-related errors occurred across all reloads
		const fetchErrors = allErrors.filter(err => 
			err.includes('Failed to fetch') || 
			err.includes('NetworkError')
		);

		expect(fetchErrors).toHaveLength(0);
	});

	test('should display agent data after SSE connection establishes', async ({ page }) => {
		await page.goto('http://localhost:5188');

		// Wait for stats bar
		await page.waitForSelector('[data-testid="stats-bar"]', { timeout: 5000 });

		// Check that agent count is displayed (verifies data loaded)
		const filterCount = await page.locator('[data-testid="filter-count"]');
		await expect(filterCount).toBeVisible();

		// Verify count shows a number (not empty/error state)
		const countText = await filterCount.textContent();
		expect(countText).toMatch(/\d+ agents?/);
	});

	test('should show agents in grid after SSE connects', async ({ page }) => {
		await page.goto('http://localhost:5188');

		// Wait for stats bar
		await page.waitForSelector('[data-testid="stats-bar"]', { timeout: 5000 });

		// Wait for agent grid to load
		const agentGrid = await page.locator('[data-testid="agent-grid"]');
		await expect(agentGrid).toBeVisible();

		// Should either show agents or "no agents" message (not empty/broken)
		const agentCount = await agentGrid.locator('.agent-card').count();
		const emptyMessageCount = await agentGrid.locator('text=No agents').count();

		expect(agentCount > 0 || emptyMessageCount > 0).toBe(true);
	});
});
