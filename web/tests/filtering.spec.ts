import { test, expect } from '@playwright/test';

// NOTE: Filter bar is now always visible in the unified attention-first layout
// No mode switching needed - tests work directly on the main view
// Status filter was removed - sections (Working/Review/Problems/History) serve as the status filter

test.describe('Agent Filtering and Sorting', () => {
	test('should render filter bar', async ({ page }) => {
		await page.goto('/');
		
		const filterBar = page.getByTestId('filter-bar');
		await expect(filterBar).toBeVisible();
	});

	test('should have search input', async ({ page }) => {
		await page.goto('/');
		
		const searchInput = page.getByTestId('search-input');
		await expect(searchInput).toBeVisible();
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
		
		// Use data-testid for unambiguous selection (avoids conflict with empty state's "Clear filters" link)
		const clearButton = page.getByTestId('clear-filters-button');
		
		// Initially no clear button (default filters)
		await expect(clearButton).not.toBeVisible();
		
		// Change sort to trigger active filter state
		const sortSelect = page.getByTestId('sort-select');
		await sortSelect.selectOption('oldest');
		
		// Now clear button should appear
		await expect(clearButton).toBeVisible();
		
		// Click clear and verify filters reset
		await clearButton.click();
		await expect(sortSelect).toHaveValue('recent-activity');
		await expect(clearButton).not.toBeVisible();
	});

	test('should render agent sections', async ({ page }) => {
		await page.goto('/');
		
		// With progressive disclosure, we now have agent-sections container
		const agentSections = page.getByTestId('agent-sections');
		await expect(agentSections).toBeVisible();
	});

	test('should render working agents section', async ({ page }) => {
		await page.goto('/');
		
		// Working agents section should always be visible (even if empty)
		const workingSection = page.getByTestId('working-agents-section');
		await expect(workingSection).toBeVisible();
	});
});
