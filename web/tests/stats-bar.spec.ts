import { test, expect } from '@playwright/test';

test.describe('Stats Bar Visibility', () => {
	test('should render stats bar with all counts', async ({ page }) => {
		await page.goto('/');
		
		const statsBar = page.getByTestId('stats-bar');
		await expect(statsBar).toBeVisible();
		
		// Check for active count
		await expect(statsBar.getByText('active')).toBeVisible();
		
		// Check for recent count (progressive disclosure)
		await expect(statsBar.getByText('recent')).toBeVisible();
		
		// Check for archive count (progressive disclosure)
		await expect(statsBar.getByText('archive')).toBeVisible();
		
		// Check for error count
		await expect(statsBar.getByText('errors')).toBeVisible();
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
