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
		
		// First item should be focused initially
		await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(/focused/);
		
		// Press j to move down
		await page.keyboard.press('j');
		await expect(page.locator('[data-testid="issue-row-orch-go-2"]')).toHaveClass(/focused/);
		
		// Press k to move up
		await page.keyboard.press('k');
		await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(/focused/);
	});

	test('should support l/enter to expand items', async ({ page }) => {
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
		
		// Expanded details should not be visible initially
		await expect(page.getByText('Test description')).not.toBeVisible();
		
		// Press l to expand
		await page.keyboard.press('l');
		
		// L1 details should now be visible
		await expect(page.getByText('Test description')).toBeVisible();
	});

	test('should support h/esc to collapse items', async ({ page }) => {
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
		
		// Expand first
		await page.keyboard.press('l');
		await expect(page.getByText('Test description')).toBeVisible();
		
		// Press h to collapse
		await page.keyboard.press('h');
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
		
		// Press Shift+G to go to bottom
		await page.keyboard.press('Shift+G');
		await expect(page.locator('[data-testid="issue-row-orch-go-3"]')).toHaveClass(/focused/);
		
		// Press g twice to go to top
		await page.keyboard.press('g');
		await page.keyboard.press('g');
		await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(/focused/);
	});
});
