import { test, expect } from '@playwright/test';

test.describe('Recently Completed Section', () => {
	test('should render section header with count badge', async ({ page }) => {
		// Mock only the attention API with completed issues
		await page.route('**/api/attention**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					items: [
						{
							id: 'beads-recently-closed-orch-go-001',
							source: 'beads-recently-closed',
							concern: 'Verification',
							signal: 'recently-closed',
							subject: 'orch-go-001',
							summary: 'Closed 2h ago: First completed',
							priority: 50,
							role: 'human',
							collected_at: new Date().toISOString(),
							metadata: {
								closed_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
								status: 'closed',
								issue_type: 'task',
								beads_priority: 2
							}
						},
						{
							id: 'beads-recently-closed-orch-go-002',
							source: 'beads-recently-closed',
							concern: 'Verification',
							signal: 'recently-closed',
							subject: 'orch-go-002',
							summary: 'Closed 1h ago: Second completed',
							priority: 50,
							role: 'human',
							collected_at: new Date().toISOString(),
							metadata: {
								closed_at: new Date(Date.now() - 1 * 60 * 60 * 1000).toISOString(),
								status: 'closed',
								issue_type: 'bug',
								beads_priority: 1
							}
						},
						{
							id: 'beads-recently-closed-orch-go-003',
							source: 'beads-recently-closed',
							concern: 'Verification',
							signal: 'recently-closed',
							subject: 'orch-go-003',
							summary: 'Closed 30m ago: Third completed',
							priority: 50,
							role: 'human',
							collected_at: new Date().toISOString(),
							metadata: {
								closed_at: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
								status: 'closed',
								issue_type: 'task',
								beads_priority: 2
							}
						}
					],
					total: 3,
					sources: ['beads-recently-closed'],
					role: 'human',
					collected_at: new Date().toISOString()
				})
			});
		});

		await page.goto('/work-graph');
		await expect(page.getByRole('heading', { name: 'Work Graph' })).toBeVisible();
		await page.waitForTimeout(2000);

		// Section should be visible with count badge
		const section = page.locator('[data-testid="recently-completed-section"]');
		await expect(section).toBeVisible({ timeout: 10000 });
		
		// Should show count in header
		await expect(section.getByText('3')).toBeVisible();
	});

	test('should be collapsed by default', async ({ page }) => {
		await page.route('**/api/attention**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					items: [
						{
							id: 'beads-recently-closed-orch-go-001',
							source: 'beads-recently-closed',
							concern: 'Verification',
							signal: 'recently-closed',
							subject: 'orch-go-001',
							summary: 'Closed 2h ago: Completed issue',
							priority: 50,
							role: 'human',
							collected_at: new Date().toISOString(),
							metadata: {
								closed_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
								status: 'closed',
								issue_type: 'task',
								beads_priority: 2
							}
						}
					],
					total: 1,
					sources: ['beads-recently-closed'],
					role: 'human',
					collected_at: new Date().toISOString()
				})
			});
		});

		await page.goto('/work-graph');
		await expect(page.getByRole('heading', { name: 'Work Graph' })).toBeVisible();
		await page.waitForTimeout(2000);

		// Section header should be visible
		const section = page.locator('[data-testid="recently-completed-section"]');
		await expect(section).toBeVisible({ timeout: 10000 });

		// Content should be hidden (collapsed by default)
		const content = page.locator('[data-testid="recently-completed-content"]');
		await expect(content).not.toBeVisible();
	});

	test('should expand on click to show completed issues', async ({ page }) => {
		await page.route('**/api/attention**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					items: [
						{
							id: 'beads-recently-closed-orch-go-001',
							source: 'beads-recently-closed',
							concern: 'Verification',
							signal: 'recently-closed',
							subject: 'orch-go-001',
							summary: 'Closed 2h ago: Completed issue',
							priority: 50,
							role: 'human',
							collected_at: new Date().toISOString(),
							metadata: {
								closed_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
								status: 'closed',
								issue_type: 'task',
								beads_priority: 2
							}
						}
					],
					total: 1,
					sources: ['beads-recently-closed'],
					role: 'human',
					collected_at: new Date().toISOString()
				})
			});
		});

		await page.goto('/work-graph');
		await expect(page.getByRole('heading', { name: 'Work Graph' })).toBeVisible();
		await page.waitForTimeout(2000);

		// Click to expand
		const sectionHeader = page.locator('[data-testid="recently-completed-toggle"]');
		await expect(sectionHeader).toBeVisible({ timeout: 10000 });
		await sectionHeader.click();

		// Content should now be visible
		const content = page.locator('[data-testid="recently-completed-content"]');
		await expect(content).toBeVisible();

		// Issue should be shown
		await expect(content.getByText('Completed issue')).toBeVisible();
	});

	test('should have hard visual delimiter from open issues', async ({ page }) => {
		await page.route('**/api/attention**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					items: [
						{
							id: 'beads-recently-closed-orch-go-001',
							source: 'beads-recently-closed',
							concern: 'Verification',
							signal: 'recently-closed',
							subject: 'orch-go-001',
							summary: 'Closed 2h ago: Completed issue',
							priority: 50,
							role: 'human',
							collected_at: new Date().toISOString(),
							metadata: {
								closed_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
								status: 'closed',
								issue_type: 'task',
								beads_priority: 2
							}
						}
					],
					total: 1,
					sources: ['beads-recently-closed'],
					role: 'human',
					collected_at: new Date().toISOString()
				})
			});
		});

		await page.goto('/work-graph');
		await expect(page.getByRole('heading', { name: 'Work Graph' })).toBeVisible();
		await page.waitForTimeout(2000);

		// Section should have border styling for visual delimiter
		const section = page.locator('[data-testid="recently-completed-section"]');
		await expect(section).toBeVisible({ timeout: 10000 });
		await expect(section).toHaveClass(/border/);
	});

	test('should not render section when no completed issues', async ({ page }) => {
		await page.route('**/api/attention**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					items: [],
					total: 0,
					sources: [],
					role: 'human',
					collected_at: new Date().toISOString()
				})
			});
		});

		await page.goto('/work-graph');
		await expect(page.getByRole('heading', { name: 'Work Graph' })).toBeVisible();
		await page.waitForTimeout(2000);

		// Section should not exist when no completed issues
		const section = page.locator('[data-testid="recently-completed-section"]');
		await expect(section).not.toBeVisible();
	});
});
