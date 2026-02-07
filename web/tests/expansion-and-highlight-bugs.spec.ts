import { test, expect } from '@playwright/test';

test.describe('Bug: Expansion state resets on tree rebuild (orch-go-21194)', () => {
	test('should preserve expansion state across poll/rebuild', async ({ page }) => {
		const graphData = {
			nodes: [
				{ id: 'test-epic-1', title: 'Test Parent Epic', type: 'epic', status: 'in_progress', priority: 1, source: 'beads' },
				{ id: 'test-epic-1.1', title: 'Test Child Task 1', type: 'task', status: 'open', priority: 2, source: 'beads' },
				{ id: 'test-epic-1.2', title: 'Test Child Task 2', type: 'task', status: 'open', priority: 2, source: 'beads' }
			],
			edges: [
				{ from: 'test-epic-1.1', to: 'test-epic-1', type: 'parent-child' },
				{ from: 'test-epic-1.2', to: 'test-epic-1', type: 'parent-child' }
			],
			node_count: 3,
			edge_count: 2
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
		await expect(page.getByText('Test Parent Epic')).toBeVisible({ timeout: 30000 });

		// Children should be visible initially (expanded by default)
		await expect(page.locator('[data-testid="issue-row-test-epic-1.1"]')).toBeVisible();
		await expect(page.locator('[data-testid="issue-row-test-epic-1.2"]')).toBeVisible();

		// Click on parent row to select it
		const parentRow = page.locator('[data-testid="issue-row-test-epic-1"]');
		await parentRow.click();
		await page.waitForTimeout(200);
		
		// Verify parent is selected (has selected class)
		await expect(parentRow).toHaveAttribute('aria-selected', 'true');
		
		// Press h to collapse - this requires focus on the tree container
		await page.keyboard.press('h');
		await page.waitForTimeout(500);

		// Children should now be hidden (collapsed)
		await expect(page.locator('[data-testid="issue-row-test-epic-1.1"]')).not.toBeVisible({ timeout: 5000 });
		await expect(page.locator('[data-testid="issue-row-test-epic-1.2"]')).not.toBeVisible();

		// Wait for polling to trigger (5+ seconds)
		await page.waitForTimeout(5500);

		// After poll, children should STILL be hidden (expansion state preserved)
		await expect(page.locator('[data-testid="issue-row-test-epic-1.1"]')).not.toBeVisible();
		await expect(page.locator('[data-testid="issue-row-test-epic-1.2"]')).not.toBeVisible();

		// Parent should still be visible
		await expect(page.getByText('Test Parent Epic')).toBeVisible();
	});
});

test.describe('Bug: Blue highlight on epic children when expanded (orch-go-21195)', () => {
	test('should not highlight issues on initial load', async ({ page }) => {
		const graphData = {
			nodes: [
				{ id: 'test-issue-1', title: 'Initial Issue Only', type: 'task', status: 'open', priority: 1, source: 'beads' }
			],
			edges: [],
			node_count: 1,
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
		await expect(page.getByText('Initial Issue Only')).toBeVisible({ timeout: 30000 });

		// Initial issue should NOT have the new-issue-highlight class
		const issueRow = page.locator('[data-testid="issue-row-test-issue-1"]');
		const classes = await issueRow.getAttribute('class');
		expect(classes).not.toContain('new-issue-highlight');
	});

	test('should not highlight children (they were always in API data)', async ({ page }) => {
		const graphData = {
			nodes: [
				{ id: 'test-parent', title: 'Parent With Child', type: 'epic', status: 'in_progress', priority: 1, source: 'beads' },
				{ id: 'test-child', title: 'Child Issue', type: 'task', status: 'open', priority: 2, source: 'beads' }
			],
			edges: [
				{ from: 'test-child', to: 'test-parent', type: 'parent-child' }
			],
			node_count: 2,
			edge_count: 1
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
		await expect(page.getByText('Parent With Child')).toBeVisible({ timeout: 30000 });
		await expect(page.getByText('Child Issue')).toBeVisible();

		// Child should NOT be highlighted on initial load
		const childRow = page.locator('[data-testid="issue-row-test-child"]');
		const childClasses = await childRow.getAttribute('class');
		expect(childClasses).not.toContain('new-issue-highlight');
	});
});
