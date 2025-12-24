import { test, expect } from '@playwright/test';

test.describe('Agent Filtering and Sorting', () => {
	test('should render filter bar', async ({ page }) => {
		await page.goto('/');
		
		const filterBar = page.getByTestId('filter-bar');
		await expect(filterBar).toBeVisible();
	});

	test('should have status filter dropdown', async ({ page }) => {
		await page.goto('/');
		
		const statusFilter = page.getByTestId('status-filter');
		await expect(statusFilter).toBeVisible();
		
		// Check default value is "all"
		await expect(statusFilter).toHaveValue('all');
		
		// Check options exist
		const options = statusFilter.locator('option');
		await expect(options).toHaveCount(5); // All, Active, Idle, Completed, Abandoned
	});

	test('should have sort dropdown', async ({ page }) => {
		await page.goto('/');
		
		const sortSelect = page.getByTestId('sort-select');
		await expect(sortSelect).toBeVisible();
		
		// Check default value is "recent-activity"
		await expect(sortSelect).toHaveValue('recent-activity');
		
		// Check options exist
		const options = sortSelect.locator('option');
		await expect(options).toHaveCount(6); // Recent Activity, Newest, Oldest, By Project, By Phase, A-Z
	});

	test('should display agent count', async ({ page }) => {
		await page.goto('/');
		
		const filterCount = page.getByTestId('filter-count');
		await expect(filterCount).toBeVisible();
		
		// Should show "X agents" or "X agent"
		await expect(filterCount).toContainText(/\d+ agents?/);
	});

	test('should change status filter', async ({ page }) => {
		await page.goto('/');
		
		const statusFilter = page.getByTestId('status-filter');
		
		// Change to "active"
		await statusFilter.selectOption('active');
		await expect(statusFilter).toHaveValue('active');
		
		// Change to "completed"
		await statusFilter.selectOption('completed');
		await expect(statusFilter).toHaveValue('completed');
		
		// Change back to "all"
		await statusFilter.selectOption('all');
		await expect(statusFilter).toHaveValue('all');
	});

	test('should change sort order', async ({ page }) => {
		await page.goto('/');
		
		const sortSelect = page.getByTestId('sort-select');
		
		// Change to "oldest"
		await sortSelect.selectOption('oldest');
		await expect(sortSelect).toHaveValue('oldest');
		
		// Change to "alphabetical"
		await sortSelect.selectOption('alphabetical');
		await expect(sortSelect).toHaveValue('alphabetical');
		
		// Change back to "newest"
		await sortSelect.selectOption('newest');
		await expect(sortSelect).toHaveValue('newest');
	});

	test('should show clear filters button when filters are active', async ({ page }) => {
		await page.goto('/');
		
		// Get the clear button in the filter bar specifically
		const filterBar = page.getByTestId('filter-bar');
		const clearButton = filterBar.getByRole('button', { name: 'Clear' });
		
		// Initially no clear button (default filters)
		await expect(clearButton).not.toBeVisible();
		
		// Change status filter
		const statusFilter = page.getByTestId('status-filter');
		await statusFilter.selectOption('active');
		
		// Now clear button should appear
		await expect(clearButton).toBeVisible();
		
		// Click clear and verify filters reset
		await clearButton.click();
		await expect(statusFilter).toHaveValue('all');
		await expect(clearButton).not.toBeVisible();
	});

	test('should render agent sections', async ({ page }) => {
		await page.goto('/');
		
		// With progressive disclosure, we now have agent-sections container
		const agentSections = page.getByTestId('agent-sections');
		await expect(agentSections).toBeVisible();
	});
});
