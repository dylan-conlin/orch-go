import { test, expect } from '@playwright/test';

test.describe('Collapsible Section Persistence', () => {
	test.beforeEach(async ({ page }) => {
		// Clear localStorage before each test
		await page.goto('/');
		await page.evaluate(() => {
			localStorage.removeItem('orch-dashboard-sections');
		});
	});

	test('should persist section collapse state across page refresh', async ({ page }) => {
		// Switch to Historical mode where collapsible sections are visible
		const modeToggle = page.getByTestId('mode-toggle');
		const historyButton = modeToggle.getByRole('button', { name: /History/ });
		await historyButton.click();
		
		// Wait for the page to settle
		await page.waitForTimeout(500);
		
		// Verify localStorage starts with default values or is empty
		// (We don't need to assert on this, just ensure it's cleared)
		
		// The SSE stream section should exist in Historical mode
		const sseToggle = page.getByTestId('sse-stream-toggle');
		await expect(sseToggle).toBeVisible();
		
		// SSE Stream is collapsed by default - click to expand it
		await sseToggle.click();
		await page.waitForTimeout(200);
		
		// Verify localStorage was updated with expanded state
		const storedState = await page.evaluate(() => {
			const stored = localStorage.getItem('orch-dashboard-sections');
			return stored ? JSON.parse(stored) : null;
		});
		expect(storedState).toBeTruthy();
		expect(storedState.sseStream).toBe(true);
		
		// Reload the page
		await page.reload();
		
		// Switch back to Historical mode (mode persists separately)
		await modeToggle.getByRole('button', { name: /History/ }).click();
		await page.waitForTimeout(500);
		
		// Verify the section is still expanded after reload
		// The SSE stream toggle should show expanded state (has rotate-180 on the chevron)
		const chevron = sseToggle.locator('span.transition-transform');
		await expect(chevron).toHaveClass(/rotate-180/);
		
		// Verify localStorage still has the correct state
		const finalState = await page.evaluate(() => {
			const stored = localStorage.getItem('orch-dashboard-sections');
			return stored ? JSON.parse(stored) : null;
		});
		expect(finalState.sseStream).toBe(true);
	});

	test('should not overwrite stored state on initial page load', async ({ page }) => {
		// Pre-set localStorage with custom section state
		await page.evaluate(() => {
			localStorage.setItem('orch-dashboard-sections', JSON.stringify({
				active: false,    // Changed from default true
				recent: true,     // Changed from default false
				archive: true,    // Changed from default false
				upNext: true,     // Changed from default false
				readyQueue: true, // Changed from default false
				sseStream: true,  // Changed from default false
				orchestratorSessions: false // Changed from default true
			}));
		});
		
		// Reload the page to apply the preset localStorage
		await page.reload();
		
		// Switch to Historical mode
		const modeToggle = page.getByTestId('mode-toggle');
		await modeToggle.getByRole('button', { name: /History/ }).click();
		await page.waitForTimeout(500);
		
		// Verify localStorage was NOT overwritten with defaults
		const storedState = await page.evaluate(() => {
			const stored = localStorage.getItem('orch-dashboard-sections');
			return stored ? JSON.parse(stored) : null;
		});
		
		// The pre-set values should be preserved
		expect(storedState.active).toBe(false);
		expect(storedState.recent).toBe(true);
		expect(storedState.archive).toBe(true);
		expect(storedState.sseStream).toBe(true);
	});

	test('should save state changes after initial load', async ({ page }) => {
		// Start fresh
		await page.evaluate(() => {
			localStorage.removeItem('orch-dashboard-sections');
		});
		await page.reload();
		
		// Switch to Historical mode
		const modeToggle = page.getByTestId('mode-toggle');
		await modeToggle.getByRole('button', { name: /History/ }).click();
		await page.waitForTimeout(500);
		
		// Toggle SSE stream section
		const sseToggle = page.getByTestId('sse-stream-toggle');
		await sseToggle.click();
		await page.waitForTimeout(200);
		
		// Verify the change was saved
		let storedState = await page.evaluate(() => {
			const stored = localStorage.getItem('orch-dashboard-sections');
			return stored ? JSON.parse(stored) : null;
		});
		expect(storedState.sseStream).toBe(true);
		
		// Toggle it back
		await sseToggle.click();
		await page.waitForTimeout(200);
		
		// Verify the change was saved again
		storedState = await page.evaluate(() => {
			const stored = localStorage.getItem('orch-dashboard-sections');
			return stored ? JSON.parse(stored) : null;
		});
		expect(storedState.sseStream).toBe(false);
	});
});
