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
							from: 'orch-go-1',
							to: 'orch-go-1.1',
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
		
		// First item should be focused initially
		await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(/focused/);
		
		// Press j to move down
		await page.keyboard.press('j');
		await expect(page.locator('[data-testid="issue-row-orch-go-2"]')).toHaveClass(/focused/);
		
		// Press k to move up
		await page.keyboard.press('k');
		await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(/focused/);
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
		await expect(page.locator('[data-testid="issue-row-orch-go-3"]')).toHaveClass(/focused/);
		
		// Press g twice to go to top
		await page.keyboard.press('g');
		await page.keyboard.press('g');
		await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(/focused/);
	});
});

// Bug fixes for Phase 1.1
test.describe('Bug Fixes - Phase 1.1', () => {
	// Bug 1: orch-go-21144 - Highlight makes text unreadable
	test('should use border-only selection for better readability', async ({ page }) => {
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
		
		// Selected item should have border-primary class (visible border)
		await expect(rowContent).toHaveClass(/border-primary/);
		
		// Should have border-2 class (2px border width)
		await expect(rowContent).toHaveClass(/border-2/);
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
		
		// Should have both focused and selected classes
		await expect(page.locator('[data-testid="issue-row-orch-go-2"]')).toHaveClass(/focused/);
		await expect(page.locator('[data-testid="issue-row-orch-go-2"]')).toHaveClass(/selected/);
		
		// Navigate with keyboard
		await page.keyboard.press('k');
		
		// First item should now have both classes
		await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(/focused/);
		await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(/selected/);
		
		// Second item should no longer have either class
		await expect(page.locator('[data-testid="issue-row-orch-go-2"]')).not.toHaveClass(/focused/);
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
		
		// Ensure focus is still on container
		await page.locator('.work-graph-tree').focus();
		
		// Expand with l key
		await page.keyboard.press('l');
		
		// Children should be visible again
		await expect(page.locator('[data-testid="issue-row-orch-go-1.1"]')).toBeVisible();
		await expect(page.locator('[data-testid="issue-row-orch-go-1.2"]')).toBeVisible();
	});
});
