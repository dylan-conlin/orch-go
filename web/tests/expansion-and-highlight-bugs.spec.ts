import { test, expect, type Page } from '@playwright/test';

// Helper to setup common API mocks
async function setupAPIMocks(page: Page) {
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
			body: JSON.stringify({ agents: [], count: 0 })
		});
	});

	await page.route('**/api/daemon/status**', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ running: false, paused: false, queue_length: 0 })
		});
	});

	await page.route('**/api/context**', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ project_dir: '/test/project' })
		});
	});

	await page.route('**/api/attention**', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				items: [],
				total: 0, sources: [], role: "human", collected_at: "2026-01-01T00:00:00Z"
			})
		});
	});
}

test.describe('Bug: Expansion state resets on tree rebuild (orch-go-21194)', () => {
	test('should preserve expansion state across poll/rebuild', async ({ page }) => {

		// Initial graph with parent and children
		const graphData = {
			nodes: [
				{
					id: 'orch-go-1',
					title: 'Parent Epic',
					type: 'epic',
					status: 'in_progress',
					priority: 1,
					source: 'beads'
				},
				{
					id: 'orch-go-1.1',
					title: 'Child Task 1',
					type: 'task',
					status: 'open',
					priority: 2,
					source: 'beads'
				},
				{
					id: 'orch-go-1.2',
					title: 'Child Task 2',
					type: 'task',
					status: 'open',
					priority: 2,
					source: 'beads'
				}
			],
			edges: [],
			node_count: 3,
			edge_count: 0
		};

		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify(graphData)
			});
		});

		await page.goto('/work-graph');

		// Wait for tree to render
		await expect(page.getByText('Parent Epic')).toBeVisible();

		// Children should be visible initially (expanded by default)
		await expect(page.locator('[data-testid="issue-row-orch-go-1.1"]')).toBeVisible();
		await expect(page.locator('[data-testid="issue-row-orch-go-1.2"]')).toBeVisible();

		// Click on parent to select it
		await page.locator('[data-testid="issue-row-orch-go-1"]').click();
		await page.waitForTimeout(100);
		
		// Ensure container has focus
		await page.locator('.work-graph-tree').focus();
		
		// Collapse the parent with h key
		await page.keyboard.press('h');

		// Wait for DOM to update
		await page.waitForTimeout(300);

		// Children should now be hidden
		await expect(page.locator('[data-testid="issue-row-orch-go-1.1"]')).not.toBeVisible();
		await expect(page.locator('[data-testid="issue-row-orch-go-1.2"]')).not.toBeVisible();

		// Simulate a poll by triggering a graph refetch (wait 5+ seconds for the polling interval)
		await page.waitForTimeout(5500);

		// After poll, children should STILL be hidden (expansion state preserved)
		await expect(page.locator('[data-testid="issue-row-orch-go-1.1"]')).not.toBeVisible();
		await expect(page.locator('[data-testid="issue-row-orch-go-1.2"]')).not.toBeVisible();

		// Parent should still be visible
		await expect(page.getByText('Parent Epic')).toBeVisible();
	});
});

test.describe('Bug: Blue highlight on epic children when expanded (orch-go-21195)', () => {
	test('should not highlight children when expanding parent (they are not new)', async ({ page }) => {

		// Initial graph with parent and children
		const graphData = {
			nodes: [
				{
					id: 'orch-go-1',
					title: 'Parent Epic',
					type: 'epic',
					status: 'in_progress',
					priority: 1,
					source: 'beads'
				},
				{
					id: 'orch-go-1.1',
					title: 'Child Task 1',
					type: 'task',
					status: 'open',
					priority: 2,
					source: 'beads'
				},
				{
					id: 'orch-go-1.2',
					title: 'Child Task 2',
					type: 'task',
					status: 'open',
					priority: 2,
					source: 'beads'
				}
			],
			edges: [],
			node_count: 3,
			edge_count: 0
		};

		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify(graphData)
			});
		});

		await page.goto('/work-graph');

		// Wait for tree to render
		await expect(page.getByText('Parent Epic')).toBeVisible();

		// Wait for initial "new" highlights to clear (30 second timeout in code)
		// For testing, we'll just wait a bit and check that no highlight appears when expanding
		await page.waitForTimeout(1000);

		// Click on parent to select it
		await page.locator('[data-testid="issue-row-orch-go-1"]').click();
		await page.waitForTimeout(100);
		
		// Ensure container has focus
		await page.locator('.work-graph-tree').focus();
		
		// Collapse the parent with h key
		await page.keyboard.press('h');
		await page.waitForTimeout(300);

		// Children should be hidden
		await expect(page.locator('[data-testid="issue-row-orch-go-1.1"]')).not.toBeVisible();
		await expect(page.locator('[data-testid="issue-row-orch-go-1.2"]')).not.toBeVisible();

		// Expand the parent with l key
		await page.keyboard.press('l');
		await page.waitForTimeout(300);

		// Children should be visible again
		await expect(page.locator('[data-testid="issue-row-orch-go-1.1"]')).toBeVisible();
		await expect(page.locator('[data-testid="issue-row-orch-go-1.2"]')).toBeVisible();

		// Children should NOT have the "new issue" blue highlight
		// The highlight is applied via a bg-blue class when newIssueIds.has(node.id)
		const child1Row = page.locator('[data-testid="issue-row-orch-go-1.1"]');
		const child2Row = page.locator('[data-testid="issue-row-orch-go-1.2"]');

		// Check that they don't have the blue highlight class
		// (Looking for bg-blue-500/10 or similar classes that indicate new issue highlight)
		const child1Classes = await child1Row.getAttribute('class');
		const child2Classes = await child2Row.getAttribute('class');

		expect(child1Classes).not.toContain('bg-blue');
		expect(child2Classes).not.toContain('bg-blue');
	});

	test('should highlight truly new issues from API, not children from expansion', async ({ page }) => {

		// Initial graph with one issue
		let graphData = {
			nodes: [
				{
					id: 'orch-go-1',
					title: 'Existing Issue',
					type: 'task',
					status: 'open',
					priority: 1,
					source: 'beads'
				}
			],
			edges: [],
			node_count: 1,
			edge_count: 0
		};

		let callCount = 0;
		await page.route('**/api/beads/graph**', async (route) => {
			callCount++;
			if (callCount === 1) {
				// First call: return initial data
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify(graphData)
				});
			} else {
				// Subsequent calls: return data with new issue
				graphData = {
					nodes: [
						...graphData.nodes,
						{
							id: 'orch-go-2',
							title: 'Truly New Issue',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						}
					],
					edges: [],
					node_count: 2,
					edge_count: 0
				};
				await route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify(graphData)
				});
			}
		});

		await page.goto('/work-graph');

		// Wait for tree to render
		await expect(page.getByText('Existing Issue')).toBeVisible();

		// Wait for polling to trigger (5 seconds)
		await page.waitForTimeout(5500);

		// New issue should appear
		await expect(page.getByText('Truly New Issue')).toBeVisible();

		// The truly new issue SHOULD have the blue highlight
		const newIssueRow = page.locator('[data-testid="issue-row-orch-go-2"]');
		const newIssueClasses = await newIssueRow.getAttribute('class');

		// This should have the bg-blue highlight (unlike children from expansion)
		expect(newIssueClasses).toContain('bg-blue');
	});
});
