import { test, expect } from '@playwright/test';

test.describe('Attention Panel', () => {
	test('should show attention panel when there are attention items', async ({ page }) => {
		// Mock API responses to create attention conditions
		await page.route('**/api/agents', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify([
					{
						id: 'test-agent-1',
						session_id: 'ses_123',
						status: 'active',
						phase: 'Complete',
						skill: 'feature-impl',
						task: 'Test task',
						spawned_at: new Date().toISOString(),
						beads_id: 'test-001'
					}
				])
			});
		});

		await page.goto('/');
		
		// Wait for the page to fully load
		await page.waitForLoadState('networkidle');
		
		// Attention panel should be visible with blocking agents
		const attentionPanel = page.getByTestId('attention-panel');
		await expect(attentionPanel).toBeVisible();
		
		// Should show "Attention Required" header
		await expect(attentionPanel).toContainText('Attention Required');
	});

	test('should show blocking section for agents at Phase: Complete', async ({ page }) => {
		await page.route('**/api/agents', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify([
					{
						id: 'complete-agent',
						session_id: 'ses_complete',
						status: 'active',
						phase: 'Complete',
						skill: 'feature-impl',
						task: 'Completed task awaiting review',
						spawned_at: new Date().toISOString(),
						beads_id: 'ready-for-review'
					}
				])
			});
		});

		await page.goto('/');
		await page.waitForLoadState('networkidle');
		
		const blockingSection = page.getByTestId('blocking-section');
		await expect(blockingSection).toBeVisible();
		await expect(blockingSection).toContainText('Blocking');
		await expect(blockingSection).toContainText('ready-for-review');
	});

	test('should show usage warning when usage exceeds 80%', async ({ page }) => {
		await page.route('**/api/usage', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					account: 'test@example.com',
					five_hour_percent: 85,
					weekly_percent: 70,
					five_hour_reset: '2h 30m'
				})
			});
		});

		// Need at least one agent for the panel to show
		await page.route('**/api/agents', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify([])
			});
		});

		await page.goto('/');
		await page.waitForLoadState('networkidle');
		
		const usageWarning = page.getByTestId('usage-warning-section');
		await expect(usageWarning).toBeVisible();
		await expect(usageWarning).toContainText('Usage Warning');
		await expect(usageWarning).toContainText('85%');
	});

	test('should show decision section for blocked issues needing action', async ({ page }) => {
		await page.route('**/api/beads/blocked', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					issues: [
						{
							id: 'blocked-001',
							blocked_by: ['dep-001'],
							blocker_status: 'closed',
							days_blocked: 10,
							needs_action: true,
							action_reason: 'Blocker was closed, remove dependency'
						}
					]
				})
			});
		});

		await page.goto('/');
		await page.waitForLoadState('networkidle');
		
		const decisionSection = page.getByTestId('decision-section');
		await expect(decisionSection).toBeVisible();
		await expect(decisionSection).toContainText('Decision Needed');
		await expect(decisionSection).toContainText('blocked-001');
	});

	test('should hide attention panel when swarm is healthy', async ({ page }) => {
		// Mock all APIs to return "healthy" states
		await page.route('**/api/agents', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify([
					{
						id: 'working-agent',
						session_id: 'ses_working',
						status: 'active',
						phase: 'Implementing',
						skill: 'feature-impl',
						task: 'Working on something',
						spawned_at: new Date().toISOString()
					}
				])
			});
		});

		await page.route('**/api/usage', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					account: 'test@example.com',
					five_hour_percent: 30,
					weekly_percent: 50
				})
			});
		});

		await page.route('**/api/beads/blocked', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ issues: [] })
			});
		});

		await page.route('**/api/errors', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ errors: [] })
			});
		});

		await page.route('**/api/pending-reviews', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					agents: [],
					total_agents: 0,
					total_unreviewed: 0
				})
			});
		});

		await page.route('**/api/gaps', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ suggestions: [], recurring_patterns: 0 })
			});
		});

		await page.route('**/api/patterns', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ patterns: [] })
			});
		});

		await page.goto('/');
		await page.waitForLoadState('networkidle');
		
		// Attention panel should not be visible when everything is healthy
		const attentionPanel = page.getByTestId('attention-panel');
		await expect(attentionPanel).not.toBeVisible();
	});

	test('should always show active agents section', async ({ page }) => {
		await page.goto('/');
		
		const activeAgentsSection = page.getByTestId('active-agents-section');
		await expect(activeAgentsSection).toBeVisible();
		await expect(activeAgentsSection).toContainText('Active Agents');
	});
});

test.describe('Dashboard Layout', () => {
	test('should not have mode toggle', async ({ page }) => {
		await page.goto('/');
		
		// Mode toggle should NOT exist anymore
		const modeToggle = page.getByTestId('mode-toggle');
		await expect(modeToggle).not.toBeVisible();
	});

	test('should have unified layout without conditional modes', async ({ page }) => {
		await page.goto('/');
		
		// Stats bar should be visible
		const statsBar = page.getByTestId('stats-bar');
		await expect(statsBar).toBeVisible();
		
		// Active agents section should be visible
		const activeAgentsSection = page.getByTestId('active-agents-section');
		await expect(activeAgentsSection).toBeVisible();
	});

	test('should maintain 666px width usability', async ({ page }) => {
		// Set viewport to minimum supported width
		await page.setViewportSize({ width: 666, height: 900 });
		
		await page.goto('/');
		
		// Stats bar should fit without horizontal scroll
		const statsBar = page.getByTestId('stats-bar');
		await expect(statsBar).toBeVisible();
		
		// Check that there's no horizontal overflow
		const hasHorizontalScroll = await page.evaluate(() => {
			return document.documentElement.scrollWidth > document.documentElement.clientWidth;
		});
		expect(hasHorizontalScroll).toBe(false);
	});
});
