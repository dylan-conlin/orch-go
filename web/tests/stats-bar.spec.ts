import { test, expect } from '@playwright/test';

test.describe('Stats Bar Visibility', () => {
	test('should render stats bar with all counts', async ({ page }) => {
		await page.goto('/');
		
		const statsBar = page.getByTestId('stats-bar');
		await expect(statsBar).toBeVisible();
		
		// Check for active count
		await expect(statsBar.getByText('active')).toBeVisible();
		
		// Check for completed count
		await expect(statsBar.getByText('done')).toBeVisible();
		
		// Check for abandoned count
		await expect(statsBar.getByText('stuck')).toBeVisible();
		
		// Check for error count (new feature)
		await expect(statsBar.getByText('errors')).toBeVisible();
		
		// Check for events count
		await expect(statsBar.getByText('events')).toBeVisible();
	});
	
	test('should display error count', async ({ page }) => {
		await page.goto('/');
		
		const statsBar = page.getByTestId('stats-bar');
		const errorSection = statsBar.locator('text=errors').locator('..');
		
		// Error count should be visible (will be 0 initially)
		await expect(errorSection).toBeVisible();
		
		// Should show a number
		await expect(errorSection).toContainText(/\d+/);
	});
	
	test('should have connection controls', async ({ page }) => {
		await page.goto('/');
		
		const statsBar = page.getByTestId('stats-bar');
		const connectButton = statsBar.getByRole('button', { name: /Connect|Disconnect/ });
		
		await expect(connectButton).toBeVisible();
	});
});
