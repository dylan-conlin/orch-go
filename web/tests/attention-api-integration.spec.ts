import { test, expect } from '@playwright/test';

/**
 * Test suite for attention API integration.
 * Verifies that the attention store correctly fetches and maps data from /api/attention endpoint.
 */

test.describe('Attention API Integration', () => {
	test('should fetch and map likely-done signals from API', async ({ page }) => {
		// Mock the /api/attention endpoint with likely-done signal
		await page.route('**/api/attention**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					items: [
						{
							id: 'git-orch-go-20876',
							source: 'git',
							concern: 'Observability',
							signal: 'likely-done',
							subject: 'orch-go-20876',
							summary: 'in_progress: Implement SSE reconnection (commits: 3)',
							priority: 90,
							role: 'human',
							action_hint: 'orch complete orch-go-20876',
							collected_at: new Date().toISOString(),
							metadata: {
								commit_count: 3,
								last_commit_at: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
								issue_status: 'in_progress',
								reason: '3 commits found, no active workspace'
							}
						}
					],
					total: 1,
					sources: ['git'],
					role: 'human',
					collected_at: new Date().toISOString()
				})
			});
		});

		// Mock other required endpoints
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify([])
			});
		});

		await page.route('**/api/beads/ready**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ issues: [] })
			});
		});

		await page.route('**/api/daemon**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ enabled: false, running: false })
			});
		});

		// Navigate to the page
		await page.goto('http://localhost:5188');

		// Wait for attention data to load
		await page.waitForTimeout(500);

		// Verify the attention badge is displayed
		const badge = page.locator('text=LIKELY DONE').first();
		await expect(badge).toBeVisible({ timeout: 5000 });
	});

	test('should handle API errors gracefully', async ({ page }) => {
		// Mock the /api/attention endpoint to return error
		await page.route('**/api/attention**', async (route) => {
			await route.fulfill({
				status: 500,
				contentType: 'application/json',
				body: JSON.stringify({ error: 'Internal server error' })
			});
		});

		// Mock other required endpoints
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify([])
			});
		});

		await page.route('**/api/beads/ready**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ issues: [] })
			});
		});

		await page.route('**/api/daemon**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ enabled: false, running: false })
			});
		});

		// Navigate to the page
		await page.goto('http://localhost:5188');

		// Wait a bit for the page to load
		await page.waitForTimeout(500);

		// Page should still load without crashing
		await expect(page.locator('body')).toBeVisible();
	});

	test('should map multiple signal types correctly', async ({ page }) => {
		// Mock the /api/attention endpoint with multiple signal types
		await page.route('**/api/attention**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					items: [
						{
							id: 'git-orch-go-20876',
							source: 'git',
							concern: 'Observability',
							signal: 'likely-done',
							subject: 'orch-go-20876',
							summary: 'in_progress: Implement feature X (commits: 5)',
							priority: 80,
							role: 'human',
							action_hint: 'orch complete orch-go-20876',
							collected_at: new Date().toISOString(),
							metadata: {
								commit_count: 5,
								issue_status: 'in_progress'
							}
						},
						{
							id: 'beads-orch-go-20877',
							source: 'beads',
							concern: 'Actionability',
							signal: 'issue-ready',
							subject: 'orch-go-20877',
							summary: 'task: Add new dashboard widget',
							priority: 1,
							role: 'human',
							action_hint: 'orch spawn orch-go-20877',
							collected_at: new Date().toISOString(),
							metadata: {
								status: 'open',
								issue_type: 'task'
							}
						}
					],
					total: 2,
					sources: ['git', 'beads'],
					role: 'human',
					collected_at: new Date().toISOString()
				})
			});
		});

		// Mock other required endpoints
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify([])
			});
		});

		await page.route('**/api/beads/ready**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ issues: [] })
			});
		});

		await page.route('**/api/daemon**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ enabled: false, running: false })
			});
		});

		// Navigate to the page
		await page.goto('http://localhost:5188');

		// Wait for attention data to load
		await page.waitForTimeout(500);

		// Verify LIKELY DONE badge is displayed
		const likelyDoneBadge = page.locator('text=LIKELY DONE').first();
		await expect(likelyDoneBadge).toBeVisible({ timeout: 5000 });
	});
});
