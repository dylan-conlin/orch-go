import { test, expect } from '@playwright/test';

test.describe('Stats Bar Visibility', () => {
	test('should render stats bar with all counts', async ({ page }) => {
		await page.goto('/');
		
		const statsBar = page.getByTestId('stats-bar');
		await expect(statsBar).toBeVisible();
		
		// Check for error count
		await expect(statsBar.getByText('errors')).toBeVisible();
		
		// Check for connection button
		await expect(statsBar.getByRole('button', { name: /Connect|Disconnect/ })).toBeVisible();
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

	test('should display beads indicator when available', async ({ page }) => {
		// Mock the beads API response
		await page.route('**/api/beads', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					total_issues: 100,
					open_issues: 20,
					in_progress_issues: 5,
					blocked_issues: 3,
					ready_issues: 12,
					closed_issues: 75
				})
			});
		});

		await page.goto('/');
		
		const statsBar = page.getByTestId('stats-bar');
		const beadsIndicator = statsBar.getByTestId('beads-indicator');
		
		// Beads indicator should be visible
		await expect(beadsIndicator).toBeVisible();
		
		// Should show ready count
		await expect(beadsIndicator).toContainText('12');
		await expect(beadsIndicator).toContainText('ready');
		
		// Should show blocked count in parentheses
		await expect(beadsIndicator).toContainText('3 blocked');
	});

	test('should hide blocked count when zero', async ({ page }) => {
		// Mock the beads API response with no blocked issues
		await page.route('**/api/beads', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					total_issues: 50,
					open_issues: 10,
					in_progress_issues: 2,
					blocked_issues: 0,
					ready_issues: 8,
					closed_issues: 40
				})
			});
		});

		await page.goto('/');
		
		const statsBar = page.getByTestId('stats-bar');
		const beadsIndicator = statsBar.getByTestId('beads-indicator');
		
		// Beads indicator should be visible
		await expect(beadsIndicator).toBeVisible();
		
		// Should show ready count
		await expect(beadsIndicator).toContainText('8');
		await expect(beadsIndicator).toContainText('ready');
		
		// Should NOT show blocked (since it's 0)
		await expect(beadsIndicator).not.toContainText('blocked');
	});
});
