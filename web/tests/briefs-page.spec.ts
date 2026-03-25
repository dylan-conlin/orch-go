import { test, expect } from '@playwright/test';

test.describe('Briefs Page', () => {
	test('should navigate to /briefs from nav', async ({ page }) => {
		await page.goto('/');
		const briefsLink = page.getByRole('link', { name: /Briefs/ });
		await expect(briefsLink).toBeVisible();
		await briefsLink.click();
		await expect(page).toHaveURL('/briefs');
	});

	test('should render briefs page header', async ({ page }) => {
		await page.goto('/briefs');
		await expect(page.getByRole('heading', { name: 'Briefs' })).toBeVisible();
		await expect(page.getByText('Reading queue')).toBeVisible();
	});

	test('should render stats bar', async ({ page }) => {
		await page.goto('/briefs');
		const stats = page.getByTestId('briefs-stats');
		await expect(stats).toBeVisible();
		await expect(stats).toContainText('total');
	});

	test('should render filter buttons', async ({ page }) => {
		await page.goto('/briefs');
		const filterBar = page.getByTestId('briefs-filter');
		await expect(filterBar).toBeVisible();
		await expect(page.getByTestId('filter-all')).toBeVisible();
		await expect(page.getByTestId('filter-unread')).toBeVisible();
		await expect(page.getByTestId('filter-read')).toBeVisible();
	});

	test('should show empty state or briefs list', async ({ page }) => {
		await page.goto('/briefs');
		// Either empty state or list should be visible
		const empty = page.getByTestId('briefs-empty');
		const list = page.getByTestId('briefs-list');
		await expect(empty.or(list)).toBeVisible();
	});
});
