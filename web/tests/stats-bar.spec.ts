import { test, expect } from '@playwright/test';

test.describe('Stats Bar Visibility', () => {
	test('should render stats bar with all counts', async ({ page }) => {
		await page.goto('/');
		
		const statsBar = page.getByTestId('stats-bar');
		await expect(statsBar).toBeVisible();
		
		// Check for error count (abbreviated label "err" on larger screens)
		await expect(statsBar.getByText(/err|❌/)).toBeVisible();
		
		// Check for connection button (get the actual button text, not the tooltip trigger)
		await expect(statsBar.getByRole('button', { name: /Connect|Disconnect/ }).first()).toBeVisible();
	});
	
	test('should display error count', async ({ page }) => {
		await page.goto('/');
		
		const statsBar = page.getByTestId('stats-bar');
		// Find the error indicator by its emoji
		const errorSection = statsBar.locator('span:has-text("❌")').first();
		
		// Error count should be visible (will be 0 initially)
		await expect(errorSection).toBeVisible();
		
		// Should show a number
		await expect(errorSection).toContainText(/\d+/);
	});
	
	test('should have connection controls', async ({ page }) => {
		await page.goto('/');
		
		const statsBar = page.getByTestId('stats-bar');
		// Get the actual button (first one), not the tooltip trigger wrapper
		const connectButton = statsBar.getByRole('button', { name: /Connect|Disconnect/ }).first();
		
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
		await expect(beadsIndicator).toContainText(/rdy|ready/); // Abbreviated on narrow screens
		
		// Should show blocked count (abbreviated "+3blk" format)
		await expect(beadsIndicator).toContainText(/\+3|3.*blk|blocked/);
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
		await expect(beadsIndicator).toContainText(/rdy|ready/); // Abbreviated on narrow screens
		
		// Should NOT show blocked (since it's 0)
		await expect(beadsIndicator).not.toContainText('blk');
		await expect(beadsIndicator).not.toContainText('blocked');
	});

	test('should display stats bar correctly at 666px width', async ({ page }) => {
		// Set viewport to 666px width (minimum supported width per constraint)
		await page.setViewportSize({ width: 666, height: 800 });
		
		await page.goto('/');
		
		const statsBar = page.getByTestId('stats-bar');
		await expect(statsBar).toBeVisible();
		
		// Stats bar should not cause horizontal scroll
		const hasHorizontalScroll = await page.evaluate(() => {
			return document.documentElement.scrollWidth > document.documentElement.clientWidth;
		});
		expect(hasHorizontalScroll).toBe(false);
		
		// All key metrics should still be visible (by emoji)
		await expect(statsBar.locator('text=❌')).toBeVisible();
		await expect(statsBar.locator('text=🟢').first()).toBeVisible();
		await expect(statsBar.locator('text=📋')).toBeVisible();
		
		// Connection button should be visible
		await expect(statsBar.getByRole('button', { name: /Connect|Disconnect/ }).first()).toBeVisible();
	});
});
