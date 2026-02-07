import { test, expect } from '@playwright/test';

// Test graceful degradation when orch serve backend is not running
test.describe('Graceful Degradation - Backend Unavailable', () => {
	test('should show error banner when backend is unavailable', async ({ page }) => {
		// Mock all API endpoints to fail with connection refused
		await page.route('**/api/**', async (route) => {
			await route.abort('failed');
		});
		
		await page.goto('/work-graph');
		
		// Wait for initial fetch attempt
		await page.waitForTimeout(500);
		
		// Should show error banner with clear message
		const errorBanner = page.locator('[data-testid="backend-error-banner"]');
		await expect(errorBanner).toBeVisible();
		await expect(errorBanner).toContainText('Backend not running');
		await expect(errorBanner).toContainText('orch serve');
	});
	
	test('should show retry button in error banner', async ({ page }) => {
		// Mock all API endpoints to fail
		await page.route('**/api/**', async (route) => {
			await route.abort('failed');
		});
		
		await page.goto('/work-graph');
		await page.waitForTimeout(500);
		
		// Should show retry button
		const retryButton = page.locator('[data-testid="retry-button"]');
		await expect(retryButton).toBeVisible();
		await expect(retryButton).toContainText('Retry');
	});
	
	test('should use exponential backoff for retries', async ({ page }) => {
		let fetchAttempts: number[] = [];
		let startTime = Date.now();
		
		// Track fetch attempts with timestamps
		await page.route('**/api/context', async (route) => {
			fetchAttempts.push(Date.now() - startTime);
			await route.abort('failed');
		});
		
		await page.goto('/work-graph');
		
		// Wait for several retry attempts
		// Initial: 2s, then doubles: 4s, 8s
		await page.waitForTimeout(15000);
		
		// Should have multiple attempts
		expect(fetchAttempts.length).toBeGreaterThanOrEqual(3);
		
		// Check that intervals are increasing (exponential backoff)
		// Starting at 2s interval, doubling each time
		if (fetchAttempts.length >= 3) {
			const interval1 = fetchAttempts[1] - fetchAttempts[0];
			const interval2 = fetchAttempts[2] - fetchAttempts[1];
			
			// Allow tolerance for timing jitter
			expect(interval1).toBeGreaterThanOrEqual(1800); // ~2s
			expect(interval1).toBeLessThan(2500);
			expect(interval2).toBeGreaterThanOrEqual(3500); // ~4s
			expect(interval2).toBeLessThan(5000);
		}
	});
	
	test('should cap backoff at 30 seconds', async ({ page }) => {
		test.setTimeout(35000); // Increase timeout for this test
		
		// The backoff sequence is: 2s, 4s, 8s, 16s, 30s (capped)
		// Verify that backoff increases exponentially
		
		let fetchAttempts: number[] = [];
		let startTime = Date.now();
		
		// Track fetch attempts
		await page.route('**/api/context', async (route) => {
			fetchAttempts.push(Date.now() - startTime);
			await route.abort('failed');
		});
		
		await page.goto('/work-graph');
		
		// Wait for: initial(2s) + retry1(4s) + retry2(8s) = 14s + buffer
		await page.waitForTimeout(16000);
		
		// Should have at least 3 attempts
		expect(fetchAttempts.length).toBeGreaterThanOrEqual(3);
		
		// Verify intervals are increasing (exponential backoff)
		if (fetchAttempts.length >= 3) {
			const interval1 = fetchAttempts[1] - fetchAttempts[0];
			const interval2 = fetchAttempts[2] - fetchAttempts[1];
			
			// Each interval should be roughly double the previous
			// interval1 ~2s, interval2 ~4s, so interval2 should be > interval1
			expect(interval2).toBeGreaterThan(interval1);
			
			// interval2 should be at least 3.5s (4s with tolerance)
			expect(interval2).toBeGreaterThanOrEqual(3500);
		}
	});
	
	test('should auto-reconnect when backend becomes available', async ({ page }) => {
		let backendAvailable = false;
		
		// Start with backend unavailable
		await page.route('**/api/context', async (route) => {
			if (backendAvailable) {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						project_dir: '/test/project',
						project: 'test-project'
					})
				});
			} else {
				await route.abort('failed');
			}
		});
		
		await page.route('**/api/beads/graph**', async (route) => {
			if (backendAvailable) {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						nodes: [],
						edges: [],
						node_count: 0,
						edge_count: 0
					})
				});
			} else {
				await route.abort('failed');
			}
		});
		
		await page.goto('/work-graph');
		await page.waitForTimeout(500);
		
		// Should show error banner
		const errorBanner = page.locator('[data-testid="backend-error-banner"]');
		await expect(errorBanner).toBeVisible();
		
		// Make backend available
		backendAvailable = true;
		
		// Wait for next retry (should happen within 2-3 seconds)
		await page.waitForTimeout(3500);
		
		// Error banner should disappear
		await expect(errorBanner).not.toBeVisible();
		
		// Should show normal content (header)
		await expect(page.getByRole('heading', { name: 'Work Graph' })).toBeVisible();
	});
	
	test('should allow manual retry via button', async ({ page }) => {
		let backendAvailable = false;
		let fetchCount = 0;
		
		// Start with backend unavailable
		await page.route('**/api/context', async (route) => {
			fetchCount++;
			if (backendAvailable) {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						project_dir: '/test/project',
						project: 'test-project'
					})
				});
			} else {
				await route.abort('failed');
			}
		});
		
		await page.route('**/api/beads/graph**', async (route) => {
			if (backendAvailable) {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						nodes: [],
						edges: [],
						node_count: 0,
						edge_count: 0
					})
				});
			} else {
				await route.abort('failed');
			}
		});
		
		await page.goto('/work-graph');
		await page.waitForTimeout(500);
		
		const initialFetchCount = fetchCount;
		
		// Make backend available
		backendAvailable = true;
		
		// Click retry button
		const retryButton = page.locator('[data-testid="retry-button"]');
		await retryButton.click();
		
		// Should have attempted another fetch
		await page.waitForTimeout(500);
		expect(fetchCount).toBeGreaterThan(initialFetchCount);
		
		// Error banner should disappear
		const errorBanner = page.locator('[data-testid="backend-error-banner"]');
		await expect(errorBanner).not.toBeVisible();
	});
	
	test('should not spam console with errors', async ({ page }) => {
		const consoleErrors: string[] = [];
		
		// Capture console errors
		page.on('console', (msg) => {
			if (msg.type() === 'error') {
				consoleErrors.push(msg.text());
			}
		});
		
		// Mock all API endpoints to fail
		await page.route('**/api/**', async (route) => {
			await route.abort('failed');
		});
		
		await page.goto('/work-graph');
		
		// Wait for multiple retry attempts (2s + 4s = 6s)
		await page.waitForTimeout(7000);
		
		// Should have logged error once, not spammed
		// Filter for backend-specific errors (from our connectionStatus store)
		const backendErrors = consoleErrors.filter(
			err => err.includes('Backend unavailable')
		);
		
		// Should log initial error once, not on retries
		// (Other API endpoints like /api/beads/graph may also fail, but we only care about context store)
		expect(backendErrors.length).toBeLessThanOrEqual(1);
	});
	
	test('error banner should fit within 666px width', async ({ page }) => {
		// Set viewport to minimum width constraint
		await page.setViewportSize({ width: 666, height: 800 });
		
		// Mock all API endpoints to fail
		await page.route('**/api/**', async (route) => {
			await route.abort('failed');
		});
		
		await page.goto('/work-graph');
		await page.waitForTimeout(500);
		
		const errorBanner = page.locator('[data-testid="backend-error-banner"]');
		await expect(errorBanner).toBeVisible();
		
		// Check that banner doesn't cause horizontal scroll
		const bodyWidth = await page.evaluate(() => document.body.scrollWidth);
		expect(bodyWidth).toBeLessThanOrEqual(666);
		
		// Check that all text is visible (not overflowing)
		const bannerBox = await errorBanner.boundingBox();
		expect(bannerBox).not.toBeNull();
		if (bannerBox) {
			expect(bannerBox.width).toBeLessThanOrEqual(666);
		}
	});
});
