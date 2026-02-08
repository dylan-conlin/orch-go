import { test, expect } from '@playwright/test';

test.use({ ignoreHTTPSErrors: true });

/**
 * Test suite for the verification flow.
 * Tests that verified items are properly hidden from the Work Graph UI.
 */

test.describe('Verification Flow', () => {
	// Sample tree node to ensure WorkGraphTree component renders
	const sampleTreeNode = {
		id: 'test-open-issue',
		title: 'Sample open issue',
		type: 'task',
		status: 'open',
		priority: 2,
		source: 'beads'
	};

	// Common mock setup
	const mockEndpoints = async (page: any, attentionItems: any[] = [], treeNodes: any[] = [sampleTreeNode]) => {
		await page.route('**/api/attention**', async (route: any) => {
			if (route.request().method() === 'POST') {
				// Mock POST /api/attention/verify
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						issue_id: 'test-completed-1',
						status: 'verified',
						verified_at: new Date().toISOString()
					})
				});
			} else {
				// Mock GET /api/attention
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						items: attentionItems,
						total: attentionItems.length,
						sources: ['beads-recently-closed'],
						role: 'human',
						collected_at: new Date().toISOString()
					})
				});
			}
		});

		// Return proper WorkGraphResponse format with a tree node
		await page.route('**/api/beads/graph**', async (route: any) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: treeNodes,
					edges: [],
					node_count: treeNodes.length,
					edge_count: 0
				})
			});
		});

		await page.route('**/api/beads/ready**', async (route: any) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ issues: [] })
			});
		});

		await page.route('**/api/agents**', async (route: any) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify([])
			});
		});

		await page.route('**/api/daemon**', async (route: any) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ enabled: false, running: false })
			});
		});

		await page.route('**/api/orchestrator/context**', async (route: any) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ project_dir: '/test/project' })
			});
		});
	};

	test('should display NEEDS REVIEW badge for unverified completed issues', async ({ page }) => {
		const completedIssue = {
			id: 'beads-recently-closed-test-1',
			source: 'beads-recently-closed',
			concern: 'Verification',
			signal: 'recently-closed',
			subject: 'test-completed-1',
			summary: 'Closed 1h ago: Test completed issue',
			priority: 50,
			role: 'human',
			collected_at: new Date().toISOString(),
			metadata: {
				status: 'closed',
				beads_priority: 1,
				issue_type: 'task',
				closed_at: new Date(Date.now() - 60 * 60 * 1000).toISOString()
			}
		};

		await mockEndpoints(page, [completedIssue]);
		await page.goto('https://localhost:3348/work-graph');
		
		// Wait for loading to complete
		await expect(page.locator('text=Loading work graph')).not.toBeVisible({ timeout: 10000 });
		
		// Wait for the completed issue row to appear
		// The completed issues use data-testid="completed-row-{id}"
		const completedRow = page.locator('[data-testid="completed-row-test-completed-1"]');
		await expect(completedRow).toBeVisible({ timeout: 10000 });
		
		// Verify the NEEDS REVIEW badge is displayed within the row
		const badge = completedRow.locator('text=NEEDS REVIEW');
		await expect(badge).toBeVisible({ timeout: 5000 });
	});

	test('should hide verified items from the list after pressing v', async ({ page }) => {
		// Track API calls
		let verifyCallMade = false;

		// First load page with an unverified item
		const completedIssue = {
			id: 'beads-recently-closed-test-1',
			source: 'beads-recently-closed',
			concern: 'Verification',
			signal: 'recently-closed',
			subject: 'test-completed-1',
			summary: 'Closed 1h ago: Test completed issue',
			priority: 50,
			role: 'human',
			collected_at: new Date().toISOString(),
			metadata: {
				status: 'closed',
				beads_priority: 1,
				issue_type: 'task',
				closed_at: new Date(Date.now() - 60 * 60 * 1000).toISOString()
			}
		};

		await page.route('**/api/attention**', async (route) => {
			if (route.request().method() === 'POST') {
				verifyCallMade = true;
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						issue_id: 'test-completed-1',
						status: 'verified',
						verified_at: new Date().toISOString()
					})
				});
			} else {
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify({
						items: [completedIssue],
						total: 1,
						sources: ['beads-recently-closed'],
						role: 'human',
						collected_at: new Date().toISOString()
					})
				});
			}
		});

		// Return proper WorkGraphResponse format with a sample tree node
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [sampleTreeNode],
					edges: [],
					node_count: 1,
					edge_count: 0
				})
			});
		});

		await page.route('**/api/beads/ready**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ issues: [] })
			});
		});

		await page.route('**/api/agents**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify([])
			});
		});

		await page.route('**/api/daemon**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ enabled: false, running: false })
			});
		});

		await page.route('**/api/orchestrator/context**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ project_dir: '/test/project' })
			});
		});

		await page.goto('https://localhost:3348/work-graph');

		// Wait for loading to complete
		await expect(page.locator('text=Loading work graph')).not.toBeVisible({ timeout: 10000 });

		// Wait for the item to be visible
		const completedRow = page.locator('[data-testid="completed-row-test-completed-1"]');
		await expect(completedRow).toBeVisible({ timeout: 10000 });

		// Focus on the work graph tree
		const workGraph = page.locator('.work-graph-tree');
		await workGraph.click();

		// Press 'v' to verify the item (it should be selected since it's the first item)
		await page.keyboard.press('v');
		await page.waitForTimeout(500);

		// The item should now be hidden (verified items are filtered out)
		await expect(completedRow).not.toBeVisible({ timeout: 5000 });

		// Verify the API was called
		expect(verifyCallMade).toBe(true);
	});

	test('verified items should not appear in the list from backend', async ({ page }) => {
		// Backend returns no completed items (simulating all items are verified)
		await mockEndpoints(page, []);
		await page.goto('https://localhost:3348/work-graph');
		
		// Wait for loading to complete
		await expect(page.locator('text=Loading work graph')).not.toBeVisible({ timeout: 10000 });

		// No completed issues should be visible
		const completedRows = page.locator('[data-testid^="completed-row-"]');
		await expect(completedRows).toHaveCount(0);
	});

	test('needs_fix items should display with NEEDS FIX badge', async ({ page }) => {
		// Item with needs_fix verification status from backend
		const needsFixIssue = {
			id: 'beads-recently-closed-test-fix',
			source: 'beads-recently-closed',
			concern: 'Verification',
			signal: 'recently-closed',
			subject: 'test-needs-fix-1',
			summary: 'Closed 1h ago: Test needs fix issue',
			priority: 50,
			role: 'human',
			collected_at: new Date().toISOString(),
			metadata: {
				status: 'closed',
				beads_priority: 1,
				issue_type: 'bug',
				closed_at: new Date(Date.now() - 60 * 60 * 1000).toISOString(),
				verification_status: 'needs_fix'
			}
		};

		await mockEndpoints(page, [needsFixIssue]);
		await page.goto('https://localhost:3348/work-graph');

		// Wait for loading to complete
		await expect(page.locator('text=Loading work graph')).not.toBeVisible({ timeout: 10000 });

		// Wait for the completed issue row to appear
		const completedRow = page.locator('[data-testid="completed-row-test-needs-fix-1"]');
		await expect(completedRow).toBeVisible({ timeout: 10000 });
	});
});
