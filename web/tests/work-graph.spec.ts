import { test, expect } from '@playwright/test';

test.describe('Work Graph Page', () => {
	test('should render work graph route', async ({ page }) => {
		await page.goto('/work-graph');
		
		// Should show page title
		await expect(page.getByRole('heading', { name: 'Work Graph' })).toBeVisible();
	});

	test('should fetch and display graph data', async ({ page }) => {
		// Mock the graph API response
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
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
							title: 'Child Task',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						}
					],
					edges: [
						{
							from: 'orch-go-1.1',  // child
							to: 'orch-go-1',      // parent
							type: 'parent-child'
						}
					],
					node_count: 2,
					edge_count: 1
				})
			});
		});

		await page.goto('/work-graph');
		
		// Should display the parent node
		await expect(page.getByText('Parent Epic')).toBeVisible();
		
		// Should display the child node
		await expect(page.getByText('Child Task')).toBeVisible();
	});
});

test.describe('Work Graph Tree Structure', () => {
	test('should display L0 view with status, priority, id, title, age, type', async ({ page }) => {
		const now = new Date().toISOString();
		
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-100',
							title: 'Test Issue',
							type: 'task',
							status: 'in_progress',
							priority: 1,
							source: 'beads',
							created_at: now
						}
					],
					edges: [],
					node_count: 1,
					edge_count: 0
				})
			});
		});

		await page.goto('/work-graph');
		
		const issueRow = page.locator('[data-testid="issue-row-orch-go-100"]');
		
		// Should show status icon
		await expect(issueRow.locator('[data-testid="status-icon"]')).toBeVisible();
		
		// Should show priority
		await expect(issueRow.locator('[data-testid="priority-badge"]')).toBeVisible();
		
		// Should show ID
		await expect(issueRow.getByText('orch-go-100')).toBeVisible();
		
		// Should show title
		await expect(issueRow.getByText('Test Issue')).toBeVisible();
		
		// Should show type badge
		await expect(issueRow.locator('[data-testid="type-badge"]')).toBeVisible();
	});
});

test.describe('Work Graph Keyboard Navigation', () => {
	test('should support j/k for up/down navigation', async ({ page }) => {
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-1',
							title: 'First Issue',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						},
						{
							id: 'orch-go-2',
							title: 'Second Issue',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						}
					],
					edges: [],
					node_count: 2,
					edge_count: 0
				})
			});
		});

		await page.goto('/work-graph');
		
		// Ensure container has focus
		await page.locator('.work-graph-tree').focus();
		
		// First item should be selected initially
		await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(/selected/);
		
		// Press j to move down
		await page.keyboard.press('j');
		await expect(page.locator('[data-testid="issue-row-orch-go-2"]')).toHaveClass(/selected/);
		
		// Press k to move up
		await page.keyboard.press('k');
		await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(/selected/);
	});

	test('should support Enter to expand L1 details', async ({ page }) => {
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-1',
							title: 'Test Issue',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads',
							description: 'Test description'
						}
					],
					edges: [],
					node_count: 1,
					edge_count: 0
				})
			});
		});

		await page.goto('/work-graph');
		
		// Ensure container has focus
		await page.locator('.work-graph-tree').focus();
		
		// Expanded details should not be visible initially
		await expect(page.getByText('Test description')).not.toBeVisible();
		
		// Press Enter to expand L1 details
		await page.keyboard.press('Enter');
		
		// L1 details should now be visible
		await expect(page.getByText('Test description')).toBeVisible();
	});

	test('should support Escape to collapse L1 details', async ({ page }) => {
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-1',
							title: 'Test Issue',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads',
							description: 'Test description'
						}
					],
					edges: [],
					node_count: 1,
					edge_count: 0
				})
			});
		});

		await page.goto('/work-graph');
		
		// Ensure container has focus
		await page.locator('.work-graph-tree').focus();
		
		// Expand first with Enter
		await page.keyboard.press('Enter');
		await expect(page.getByText('Test description')).toBeVisible();
		
		// Press Escape to collapse L1 details
		await page.keyboard.press('Escape');
		await expect(page.getByText('Test description')).not.toBeVisible();
	});

	test('should support g/G for top/bottom navigation', async ({ page }) => {
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{ id: 'orch-go-1', title: 'First', type: 'task', status: 'open', priority: 2, source: 'beads' },
						{ id: 'orch-go-2', title: 'Second', type: 'task', status: 'open', priority: 2, source: 'beads' },
						{ id: 'orch-go-3', title: 'Third', type: 'task', status: 'open', priority: 2, source: 'beads' }
					],
					edges: [],
					node_count: 3,
					edge_count: 0
				})
			});
		});

		await page.goto('/work-graph');
		
		// Ensure container has focus
		await page.locator('.work-graph-tree').focus();
		
		// Press Shift+G to go to bottom
		await page.keyboard.press('Shift+G');
		await expect(page.locator('[data-testid="issue-row-orch-go-3"]')).toHaveClass(/selected/);
		
		// Press g twice to go to top
		await page.keyboard.press('g');
		await page.keyboard.press('g');
		await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(/selected/);
	});
});

// Bug fixes for Phase 1.1
test.describe('Bug Fixes - Phase 1.1', () => {
	// Bug 1: orch-go-21144 - Highlight makes text unreadable
	test('should use background highlight for selection', async ({ page }) => {
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-1',
							title: 'Test Issue',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						}
					],
					edges: [],
					node_count: 1,
					edge_count: 0
				})
			});
		});

		await page.goto('/work-graph');
		
		// Wait for the row to be visible
		const issueRow = page.locator('[data-testid="issue-row-orch-go-1"]');
		await expect(issueRow).toBeVisible();
		
		const rowContent = issueRow.locator('> div').first();
		
		// Selected item should have bg-accent class (background highlight)
		await expect(rowContent).toHaveClass(/bg-accent/);
		
		// Should NOT have border-primary (no border)
		await expect(rowContent).not.toHaveClass(/border-primary/);
	});

	// Bug 2: orch-go-21145 - Border and highlight out of sync
	test('should unify selection state between click and keyboard navigation', async ({ page }) => {
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-1',
							title: 'First Issue',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						},
						{
							id: 'orch-go-2',
							title: 'Second Issue',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						}
					],
					edges: [],
					node_count: 2,
					edge_count: 0
				})
			});
		});

		await page.goto('/work-graph');
		
		// Click on second item
		await page.locator('[data-testid="issue-row-orch-go-2"]').click();
		
		// Should have selected class (unified state)
		await expect(page.locator('[data-testid="issue-row-orch-go-2"]')).toHaveClass(/selected/);
		
		// Should have bg-accent styling (no border)
		const row2Content = page.locator('[data-testid="issue-row-orch-go-2"] > div').first();
		await expect(row2Content).toHaveClass(/bg-accent/);
		await expect(row2Content).not.toHaveClass(/border-primary/);
		
		// Navigate with keyboard
		await page.keyboard.press('k');
		
		// First item should now have selected class
		await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(/selected/);
		
		// Should have same bg-accent styling
		const row1Content = page.locator('[data-testid="issue-row-orch-go-1"] > div').first();
		await expect(row1Content).toHaveClass(/bg-accent/);
		await expect(row1Content).not.toHaveClass(/border-primary/);
		
		// Second item should no longer have selected class
		await expect(page.locator('[data-testid="issue-row-orch-go-2"]')).not.toHaveClass(/selected/);
	});

	// Bug 3: orch-go-21146 - Can't collapse epics with children
	test('should collapse/expand tree nodes with h/l keys', async ({ page }) => {
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
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
				})
			});
		});

		await page.goto('/work-graph');
		
		// Wait for tree to render
		await expect(page.getByText('Parent Epic')).toBeVisible();
		
		// Children should be visible initially (expanded by default)
		await expect(page.locator('[data-testid="issue-row-orch-go-1.1"]')).toBeVisible();
		await expect(page.locator('[data-testid="issue-row-orch-go-1.2"]')).toBeVisible();
		
		// Click on parent epic to select it
		await page.locator('[data-testid="issue-row-orch-go-1"]').click();
		await page.waitForTimeout(100);
		
		// Ensure container has focus before pressing h
		await page.locator('.work-graph-tree').focus();
		
		// Collapse with h key (while parent is selected)
		await page.keyboard.press('h');
		
		// Wait for DOM to update
		await page.waitForTimeout(500);
		
		// Children should now be hidden
		await expect(page.locator('[data-testid="issue-row-orch-go-1.1"]')).not.toBeVisible();
		await expect(page.locator('[data-testid="issue-row-orch-go-1.2"]')).not.toBeVisible();
		
		// Parent should still be visible
		await expect(page.getByText('Parent Epic')).toBeVisible();
		
		// Click on parent to ensure it's selected
		await page.locator('[data-testid="issue-row-orch-go-1"]').click();
		await page.waitForTimeout(100);
		
		// Ensure focus is still on container
		await page.locator('.work-graph-tree').focus();
		
		// Expand with l key
		await page.keyboard.press('l');
		
		// Children should be visible again
		await expect(page.locator('[data-testid="issue-row-orch-go-1.1"]')).toBeVisible();
		await expect(page.locator('[data-testid="issue-row-orch-go-1.2"]')).toBeVisible();
	});

	// Bug 4: orch-go-21150 - Selection highlight barely visible (regression fix)
	test('should have clearly visible selection with background highlight', async ({ page }) => {
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-1',
							title: 'Test Issue',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						}
					],
					edges: [],
					node_count: 1,
					edge_count: 0
				})
			});
		});

		await page.goto('/work-graph');
		
		// Wait for the row to be visible
		const issueRow = page.locator('[data-testid="issue-row-orch-go-1"]');
		await expect(issueRow).toBeVisible();
		
		const rowContent = issueRow.locator('> div').first();
		
		// Should have bg-accent for clear visibility (no border)
		await expect(rowContent).toHaveClass(/bg-accent/);
		await expect(rowContent).not.toHaveClass(/border-primary/);
	});

	// Bug 5: orch-go-21168 - Queued issues should appear in both WIP and tree (dual presence)
	test('should show queued issues in both WIP and main tree', async ({ page }) => {
		// Mock the beads/ready API (queued issues)
		await page.route('**/api/beads/ready**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					issues: [
						{
							id: 'orch-go-21164',
							title: 'Queued Issue',
							priority: 0,
							issue_type: 'task',
							created_at: '2026-02-02T10:00:00Z'
						}
					]
				})
			});
		});

		// Mock the beads/graph API (all issues)
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-21164',
							title: 'Queued Issue',
							type: 'task',
							status: 'open',
							priority: 0,
							source: 'beads'
						},
						{
							id: 'orch-go-100',
							title: 'Regular Issue',
							type: 'task',
							status: 'open',
							priority: 1,
							source: 'beads'
						}
					],
					edges: [],
					node_count: 2,
					edge_count: 0
				})
			});
		});

		// Mock agents API (no running agents)
		await page.route('**/api/agents**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					agents: [],
					count: 0
				})
			});
		});

		// Mock daemon/status API (WIP section needs this)
		await page.route('**/api/daemon/status**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					running: false,
					paused: false,
					queue_length: 1
				})
			});
		});

		await page.goto('/work-graph');

		// Wait for data to load
		await page.waitForTimeout(1500);

		// Regular issue should appear in the main tree
		await expect(page.locator('[data-testid="issue-row-orch-go-100"]')).toBeVisible();

		// Queued issue should appear in WIP section
		await expect(page.locator('[data-testid="wip-row-orch-go-21164"]')).toBeVisible();

		// And also remain visible in the main tree (context preserved)
		await expect(page.locator('[data-testid="issue-row-orch-go-21164"]')).toBeVisible();
	});
});

// WIP Section Integration (Bug: orch-go-21169)
test.describe('WIP Section Integration', () => {
	test('should navigate WIP items with j/k keys before main tree', async ({ page }) => {
		// Mock beads/ready API for queued issues
		await page.route('**/api/beads/ready**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					issues: [
						{
							id: 'orch-go-queued-1',
							title: 'Queued Issue 1',
							priority: 0,
							issue_type: 'bug',
							created_at: '2026-02-02T10:00:00Z'
						}
					]
				})
			});
		});

		// Mock agents API for running agents
		await page.route('**/api/agents**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					agents: [
						{
							id: 'agent-1',
							beads_id: 'orch-go-running-1',
							task: 'Running Task 1',
							status: 'active',
							phase: 'Implementation',
							runtime: '5m 30s',
							is_processing: true,
							is_stalled: false,
							spawned_at: '2026-02-02T10:00:00Z',
							updated_at: '2026-02-02T10:05:00Z'
						}
					]
				})
			});
		});

		// Mock graph API for main tree
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-tree-1',
							title: 'Tree Issue 1',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						}
					],
					edges: [],
					node_count: 1,
					edge_count: 0
				})
			});
		});

		// Mock daemon API
		await page.route('**/api/daemon**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					running: true,
					capacity_max: 3,
					capacity_used: 1,
					capacity_free: 2
				})
			});
		});

		await page.goto('/work-graph');
		
		// Wait for tree container to render
		await expect(page.locator('.work-graph-tree')).toBeVisible();
		
		// Wait for WIP items to be added to flattened nodes (check for data-node-index="0")
		await expect(page.locator('[data-node-index="0"]')).toBeVisible({ timeout: 10000 });
		
		// Ensure container has focus
		await page.locator('.work-graph-tree').focus();
		
		// First WIP item (running agent) should be focused initially
		let focusedRow = page.locator('[data-node-index="0"].focused');
		await expect(focusedRow).toBeVisible();
		await expect(focusedRow.getByText('Running Task 1')).toBeVisible();
		
		// Press j to move to second WIP item (queued issue)
		await page.keyboard.press('j');
		focusedRow = page.locator('[data-node-index="1"].focused');
		await expect(focusedRow).toBeVisible();
		await expect(focusedRow.getByText('Queued Issue 1')).toBeVisible();
		
		// Press j again to move to main tree
		await page.keyboard.press('j');
		focusedRow = page.locator('[data-node-index="2"].focused');
		await expect(focusedRow).toBeVisible();
		await expect(focusedRow.getByText('Tree Issue 1')).toBeVisible();
		
		// Press k to move back to WIP item
		await page.keyboard.press('k');
		focusedRow = page.locator('[data-node-index="1"].focused');
		await expect(focusedRow).toBeVisible();
		await expect(focusedRow.getByText('Queued Issue 1')).toBeVisible();
	});

	test('should NOT apply greyed-out styling to WIP items', async ({ page }) => {
		// Mock APIs
		await page.route('**/api/beads/ready**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					issues: [
						{
							id: 'orch-go-queued-1',
							title: 'Queued Issue',
							priority: 0,
							issue_type: 'bug',
							created_at: '2026-02-02T10:00:00Z'
						}
					]
				})
			});
		});

		await page.route('**/api/agents**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					agents: [
						{
							id: 'agent-1',
							beads_id: 'orch-go-running-1',
							task: 'Running Task',
							status: 'active',
							phase: 'Implementation',
							runtime: '5m',
							is_processing: true,
							is_stalled: false,
							spawned_at: '2026-02-02T10:00:00Z',
							updated_at: '2026-02-02T10:05:00Z'
						}
					]
				})
			});
		});

		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ nodes: [], edges: [], node_count: 0, edge_count: 0 })
			});
		});

		await page.route('**/api/daemon**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					running: true,
					capacity_max: 3,
					capacity_used: 1,
					capacity_free: 2
				})
			});
		});

		await page.goto('/work-graph');
		
		// Wait for tree container and WIP items to render
		await expect(page.locator('.work-graph-tree')).toBeVisible();
		await expect(page.locator('[data-node-index="0"]')).toBeVisible({ timeout: 10000 });
		
		// Check that WIP items do NOT have opacity-60 class
		const runningRow = page.locator('[data-node-index="0"]');
		await expect(runningRow).not.toHaveClass(/opacity-60/);
		
		const queuedRow = page.locator('[data-node-index="1"]');
		await expect(queuedRow).not.toHaveClass(/opacity-60/);
	});

	test('should toggle L1 details for WIP items with Enter key', async ({ page }) => {
		// Mock APIs
		await page.route('**/api/agents**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					agents: [
						{
							id: 'agent-1',
							beads_id: 'orch-go-running-1',
							task: 'Running Task with Details',
							status: 'active',
							phase: 'Implementation',
							skill: 'feature-impl',
							model: 'anthropic/claude-sonnet-4',
							runtime: '5m',
							is_processing: true,
							is_stalled: false,
							spawned_at: '2026-02-02T10:00:00Z',
							updated_at: '2026-02-02T10:05:00Z'
						}
					]
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

		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ nodes: [], edges: [], node_count: 0, edge_count: 0 })
			});
		});

		await page.route('**/api/daemon**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					running: true,
					capacity_max: 3,
					capacity_used: 1,
					capacity_free: 2
				})
			});
		});

		await page.goto('/work-graph');
		
		// Wait for tree container and WIP items to render
		await expect(page.locator('.work-graph-tree')).toBeVisible();
		await expect(page.locator('[data-node-index="0"]')).toBeVisible({ timeout: 10000 });
		
		// Ensure container has focus
		await page.locator('.work-graph-tree').focus();
		
		// L1 details should not be visible initially
		const expandedDetails = page.locator('.expanded-details');
		await expect(expandedDetails).not.toBeVisible();
		
		// Press Enter to expand L1 details
		await page.keyboard.press('Enter');
		
		// L1 details should now be visible with agent info
		await expect(expandedDetails).toBeVisible();
		await expect(expandedDetails.getByText(/Phase:/)).toBeVisible();
		await expect(expandedDetails.getByText(/Skill:/)).toBeVisible();
	});
});

// Parent-child edge support (orch-go-21194)
test.describe('Parent-Child Edge Support', () => {
	test('should nest children under parents using parent-child edges from API', async ({ page }) => {
		// Mock all required APIs for work-graph page
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

		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-21193',
							title: 'Parent Epic',
							type: 'epic',
							status: 'in_progress',
							priority: 1,
							source: 'beads'
						},
						{
							id: 'orch-go-21172',
							title: 'Child Task',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						}
					],
					edges: [
						{
							from: 'orch-go-21172',  // child
							to: 'orch-go-21193',    // parent
							type: ''                 // empty type for parent-child
						}
					],
					node_count: 2,
					edge_count: 1
				})
			});
		});

		await page.goto('/work-graph');
		
		// Wait for tree to render
		await expect(page.getByText('Parent Epic')).toBeVisible();
		
		// Child should be visible initially (expanded by default)
		await expect(page.locator('[data-testid="issue-row-orch-go-21172"]')).toBeVisible();
		
		// Child should be indented (depth > 0)
		const childRow = page.locator('[data-testid="issue-row-orch-go-21172"]');
		await expect(childRow).toHaveAttribute('data-depth', '1');
		
		// Ensure container has focus
		await page.locator('.work-graph-tree').focus();
		
		// Collapse parent with h key
		await page.keyboard.press('h');
		
		// Wait for DOM to update
		await page.waitForTimeout(500);
		
		// Child should now be hidden
		await expect(page.locator('[data-testid="issue-row-orch-go-21172"]')).not.toBeVisible();
		
		// Parent should still be visible
		await expect(page.getByText('Parent Epic')).toBeVisible();
		
		// Expand parent with l key
		await page.keyboard.press('l');
		
		// Child should be visible again
		await expect(page.locator('[data-testid="issue-row-orch-go-21172"]')).toBeVisible();
	});

	test('should support explicit parent-child type edges', async ({ page }) => {
		// Mock all required APIs
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

		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-100',
							title: 'Parent Issue',
							type: 'feature',
							status: 'open',
							priority: 1,
							source: 'beads'
						},
						{
							id: 'orch-go-200',
							title: 'Child Issue',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						}
					],
					edges: [
						{
							from: 'orch-go-200',     // child
							to: 'orch-go-100',       // parent
							type: 'parent-child'     // explicit type
						}
					],
					node_count: 2,
					edge_count: 1
				})
			});
		});

		await page.goto('/work-graph');
		
		// Wait for tree to render
		await expect(page.getByText('Parent Issue')).toBeVisible();
		
		// Child should be nested under parent
		const childRow = page.locator('[data-testid="issue-row-orch-go-200"]');
		await expect(childRow).toBeVisible();
		await expect(childRow).toHaveAttribute('data-depth', '1');
	});

	test('should combine ID pattern hierarchy with edge-based hierarchy', async ({ page }) => {
		// Mock all required APIs
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

		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-1',
							title: 'Root Epic',
							type: 'epic',
							status: 'open',
							priority: 1,
							source: 'beads'
						},
						{
							id: 'orch-go-1.1',
							title: 'ID Pattern Child',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						},
						{
							id: 'orch-go-500',
							title: 'Edge-based Child',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						}
					],
					edges: [
						{
							from: 'orch-go-500',  // edge-based child
							to: 'orch-go-1',      // parent
							type: ''
						}
					],
					node_count: 3,
					edge_count: 1
				})
			});
		});

		await page.goto('/work-graph');
		
		// Wait for tree to render
		await expect(page.getByText('Root Epic')).toBeVisible();
		
		// Both children should be visible and nested at depth 1
		const idPatternChild = page.locator('[data-testid="issue-row-orch-go-1.1"]');
		await expect(idPatternChild).toBeVisible();
		await expect(idPatternChild).toHaveAttribute('data-depth', '1');
		
		const edgeBasedChild = page.locator('[data-testid="issue-row-orch-go-500"]');
		await expect(edgeBasedChild).toBeVisible();
		await expect(edgeBasedChild).toHaveAttribute('data-depth', '1');
	});
});

// Verification keyboard shortcuts (orch-go-21213)
test.describe('Verification Keyboard Shortcuts', () => {
	test('should mark unverified issue as verified with v key', async ({ page }) => {
		// Track API calls
		let verifyApiCalled = false;
		let verifyRequestBody: any = null;

		// Mock the verify API
		await page.route('**/api/attention/verify', async (route) => {
			verifyApiCalled = true;
			verifyRequestBody = JSON.parse(route.request().postData() || '{}');
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					issue_id: verifyRequestBody.issue_id,
					status: verifyRequestBody.status,
					verified_at: new Date().toISOString()
				})
			});
		});

		// Mock the attention API with a recently-closed (unverified) issue
		await page.route('**/api/attention**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					items: [
						{
							id: 'beads-recently-closed-orch-go-test-123',
							source: 'beads-recently-closed',
							concern: 'Verification',
							signal: 'recently-closed',
							subject: 'orch-go-test-123',
							summary: 'Closed 2h ago: Test completed issue',
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

		// Mock other required endpoints
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ nodes: [], edges: [], node_count: 0, edge_count: 0 })
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
				body: JSON.stringify({ agents: [], count: 0 })
			});
		});

		await page.route('**/api/daemon**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ running: false, paused: false, queue_length: 0 })
			});
		});

		await page.goto('/work-graph');

		// Wait for the tree to render
		await expect(page.locator('.work-graph-tree')).toBeVisible();
		await page.waitForTimeout(500);

		// Wait for the unverified issue to appear
		await expect(page.locator('[data-testid="issue-row-orch-go-test-123"]')).toBeVisible({ timeout: 5000 });

		// Ensure container has focus
		await page.locator('.work-graph-tree').focus();

		// Press v to verify the issue
		await page.keyboard.press('v');

		// Wait for API call
		await page.waitForTimeout(200);

		// Verify the API was called with correct parameters
		expect(verifyApiCalled).toBe(true);
		expect(verifyRequestBody.issue_id).toBe('orch-go-test-123');
		expect(verifyRequestBody.status).toBe('verified');
	});

	test('should mark unverified issue as needs_fix with x key', async ({ page }) => {
		// Track API calls
		let verifyApiCalled = false;
		let verifyRequestBody: any = null;

		// Mock the verify API
		await page.route('**/api/attention/verify', async (route) => {
			verifyApiCalled = true;
			verifyRequestBody = JSON.parse(route.request().postData() || '{}');
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					issue_id: verifyRequestBody.issue_id,
					status: verifyRequestBody.status,
					verified_at: new Date().toISOString()
				})
			});
		});

		// Mock the attention API with a recently-closed (unverified) issue
		await page.route('**/api/attention**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					items: [
						{
							id: 'beads-recently-closed-orch-go-test-456',
							source: 'beads-recently-closed',
							concern: 'Verification',
							signal: 'recently-closed',
							subject: 'orch-go-test-456',
							summary: 'Closed 1h ago: Another test issue',
							priority: 50,
							role: 'human',
							collected_at: new Date().toISOString(),
							metadata: {
								closed_at: new Date(Date.now() - 1 * 60 * 60 * 1000).toISOString(),
								status: 'closed',
								issue_type: 'bug',
								beads_priority: 1
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

		// Mock other required endpoints
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ nodes: [], edges: [], node_count: 0, edge_count: 0 })
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
				body: JSON.stringify({ agents: [], count: 0 })
			});
		});

		await page.route('**/api/daemon**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ running: false, paused: false, queue_length: 0 })
			});
		});

		await page.goto('/work-graph');

		// Wait for the tree to render
		await expect(page.locator('.work-graph-tree')).toBeVisible();
		await page.waitForTimeout(500);

		// Wait for the unverified issue to appear
		await expect(page.locator('[data-testid="issue-row-orch-go-test-456"]')).toBeVisible({ timeout: 5000 });

		// Ensure container has focus
		await page.locator('.work-graph-tree').focus();

		// Press x to mark as needs_fix
		await page.keyboard.press('x');

		// Wait for API call
		await page.waitForTimeout(200);

		// Verify the API was called with correct parameters
		expect(verifyApiCalled).toBe(true);
		expect(verifyRequestBody.issue_id).toBe('orch-go-test-456');
		expect(verifyRequestBody.status).toBe('needs_fix');
	});

	test('should not trigger verification for non-completed issues', async ({ page }) => {
		// Track API calls
		let verifyApiCalled = false;

		// Mock the verify API
		await page.route('**/api/attention/verify', async (route) => {
			verifyApiCalled = true;
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ issue_id: 'test', status: 'verified', verified_at: new Date().toISOString() })
			});
		});

		// Mock attention API with no completed issues
		await page.route('**/api/attention**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ items: [], total: 0, sources: [], role: 'human', collected_at: new Date().toISOString() })
			});
		});

		// Mock graph API with regular tree node
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-regular-1',
							title: 'Regular Issue',
							type: 'task',
							status: 'open',
							priority: 2,
							source: 'beads'
						}
					],
					edges: [],
					node_count: 1,
					edge_count: 0
				})
			});
		});

		// Mock other endpoints
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

		await page.route('**/api/daemon**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ running: false, paused: false, queue_length: 0 })
			});
		});

		await page.goto('/work-graph');

		// Wait for the tree to render
		await expect(page.locator('.work-graph-tree')).toBeVisible();
		await expect(page.locator('[data-testid="issue-row-orch-go-regular-1"]')).toBeVisible({ timeout: 5000 });

		// Ensure container has focus
		await page.locator('.work-graph-tree').focus();

		// Press v on a regular tree node (should not trigger verify)
		await page.keyboard.press('v');
		await page.waitForTimeout(200);

		// Verify the API was NOT called
		expect(verifyApiCalled).toBe(false);
	});
});

// Status View tests (orch-go-21209)
test.describe('Status View', () => {
	test('should switch to status view and display status groups', async ({ page }) => {
		// Mock the graph API response (matching pattern from working tests)
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [
						{
							id: 'orch-go-1',
							title: 'Ready Issue',
							type: 'task',
							status: 'open',
							priority: 1,
							source: 'beads'
						},
						{
							id: 'orch-go-2',
							title: 'In Progress Issue',
							type: 'task',
							status: 'in_progress',
							priority: 2,
							source: 'beads'
						},
						{
							id: 'orch-go-3',
							title: 'Blocked Issue',
							type: 'task',
							status: 'blocked',
							priority: 1,
							source: 'beads'
						},
						{
							id: 'orch-go-4',
							title: 'Done Issue',
							type: 'task',
							status: 'closed',
							priority: 2,
							source: 'beads'
						}
					],
					edges: [],
					node_count: 4,
					edge_count: 0
				})
			});
		});

		await page.goto('/work-graph');
		
		// Wait for tree view to load (default view shows issue titles)
		await expect(page.getByText('Ready Issue')).toBeVisible();
		
		// Click on Status view button
		await page.getByRole('button', { name: 'Status' }).click();
		
		// Wait for status view container to appear
		await expect(page.locator('.work-graph-status')).toBeVisible();
		
		// Should show status group headers (use first() to handle multiple matches)
		await expect(page.getByText('Ready', { exact: true }).first()).toBeVisible();
		await expect(page.getByText('In Progress').first()).toBeVisible();
		await expect(page.getByText('Blocked').first()).toBeVisible();
		await expect(page.getByText('Done').first()).toBeVisible();
		
		// Verify issues are still visible under their groups
		await expect(page.getByText('Ready Issue')).toBeVisible();
		await expect(page.getByText('In Progress Issue')).toBeVisible();
		await expect(page.getByText('Blocked Issue')).toBeVisible();
		await expect(page.getByText('Done Issue')).toBeVisible();
	});
});
