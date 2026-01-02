import { test, expect } from '@playwright/test';

test.describe('Dashboard Mode Toggle', () => {
	test('should switch between operational and historical modes', async ({ page }) => {
		await page.goto('/');
		
		const modeToggle = page.getByTestId('mode-toggle');
		await expect(modeToggle).toBeVisible();
		
		// Get both buttons
		const opsButton = modeToggle.getByRole('button', { name: /Ops/ });
		const historyButton = modeToggle.getByRole('button', { name: /History/ });
		
		await expect(opsButton).toBeVisible();
		await expect(historyButton).toBeVisible();
		
		// Ops button should be active by default (has bg-background class)
		await expect(opsButton).toHaveClass(/bg-background/);
		await expect(historyButton).not.toHaveClass(/bg-background/);
		
		// Should see operational mode content (active agents section is always visible in ops mode)
		await expect(page.getByTestId('active-agents-section')).toBeVisible();
		
		// Click History button
		await historyButton.click();
		
		// History button should now be active
		await expect(historyButton).toHaveClass(/bg-background/);
		await expect(opsButton).not.toHaveClass(/bg-background/);
		
		// Should see historical mode content (filter bar is only in historical mode)
		await expect(page.getByTestId('filter-bar')).toBeVisible();
		
		// Click Ops button to switch back
		await opsButton.click();
		
		// Ops button should be active again
		await expect(opsButton).toHaveClass(/bg-background/);
		await expect(historyButton).not.toHaveClass(/bg-background/);
		
		// Filter bar should be hidden (only in historical mode)
		await expect(page.getByTestId('filter-bar')).not.toBeVisible();
	});

	test('should persist mode selection in localStorage', async ({ page }) => {
		await page.goto('/');
		
		const modeToggle = page.getByTestId('mode-toggle');
		const historyButton = modeToggle.getByRole('button', { name: /History/ });
		
		// Click History button
		await historyButton.click();
		
		// Verify localStorage was updated
		const storedMode = await page.evaluate(() => {
			return localStorage.getItem('orch-dashboard-mode');
		});
		expect(storedMode).toBe('historical');
		
		// Reload the page
		await page.reload();
		
		// History button should still be active after reload
		await expect(historyButton).toHaveClass(/bg-background/);
	});
});
