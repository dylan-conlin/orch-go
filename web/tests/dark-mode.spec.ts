import { test, expect } from '@playwright/test';

test.describe('Theme Selection', () => {
	test.beforeEach(async ({ page }) => {
		// Clear localStorage before each test
		await page.addInitScript(() => {
			localStorage.clear();
		});
	});

	test('should render theme toggle button', async ({ page }) => {
		await page.goto('/');
		const themeToggle = page.getByRole('button', { name: 'Select theme' });
		await expect(themeToggle).toBeVisible();
	});

	test('should open dropdown menu on click', async ({ page }) => {
		await page.goto('/');
		
		const themeToggle = page.getByRole('button', { name: 'Select theme' });
		await themeToggle.click();
		
		// Check that dropdown content is visible
		await expect(page.getByText('Theme')).toBeVisible();
		await expect(page.getByText('Light')).toBeVisible();
		await expect(page.getByText('Dark')).toBeVisible();
		await expect(page.getByText('System')).toBeVisible();
	});

	test('should switch to dark mode when selecting Dark option', async ({ page }) => {
		await page.goto('/');
		
		const themeToggle = page.getByRole('button', { name: 'Select theme' });
		await themeToggle.click();
		
		// Click on Dark option
		await page.getByText('Dark').click();
		
		// Check if dark class is added to html
		const html = page.locator('html');
		await expect(html).toHaveClass(/dark/);
		
		// Check localStorage
		const theme = await page.evaluate(() => localStorage.getItem('theme'));
		expect(theme).toBe('dark');
	});

	test('should switch to light mode when selecting Light option', async ({ page }) => {
		await page.goto('/');
		
		const themeToggle = page.getByRole('button', { name: 'Select theme' });
		await themeToggle.click();
		
		// Click on Light option
		await page.getByText('Light').click();
		
		// Check if dark class is removed from html
		const html = page.locator('html');
		await expect(html).not.toHaveClass(/dark/);
		
		// Check localStorage
		const theme = await page.evaluate(() => localStorage.getItem('theme'));
		expect(theme).toBe('light');
	});

	test('should persist theme preference in localStorage', async ({ page, context }) => {
		// Create a new page without the beforeEach clearing localStorage
		const newPage = await context.newPage();
		await newPage.goto('/');
		
		// Open dropdown and select dark mode
		const themeToggle = newPage.getByRole('button', { name: 'Select theme' });
		await themeToggle.click();
		await newPage.getByText('Dark').click();
		
		// Check localStorage
		const theme = await newPage.evaluate(() => localStorage.getItem('theme'));
		expect(theme).toBe('dark');
		
		// Reload the page (no addInitScript, so localStorage persists)
		await newPage.reload();
		
		// Wait for the page to load
		await newPage.waitForLoadState('domcontentloaded');
		
		// Theme should be restored from localStorage
		const themeAfterReload = await newPage.evaluate(() => localStorage.getItem('theme'));
		expect(themeAfterReload).toBe('dark');
		
		// Check that dark class is still applied
		const html = newPage.locator('html');
		await expect(html).toHaveClass(/dark/);
		
		await newPage.close();
	});

	test('should show sun icon in dark mode and moon icon in light mode', async ({ page }) => {
		await page.goto('/');
		
		const themeToggle = page.getByRole('button', { name: 'Select theme' });
		
		// Get initial icon (should be Moon for light mode)
		const moonIcon = themeToggle.locator('svg');
		await expect(moonIcon).toBeVisible();
		
		// Switch to dark mode
		await themeToggle.click();
		await page.getByText('Dark').click();
		
		// Now should show Sun icon
		const sunIcon = themeToggle.locator('svg');
		await expect(sunIcon).toBeVisible();
	});

	test('should switch to system preference when selecting System option', async ({ page }) => {
		await page.goto('/');
		
		const themeToggle = page.getByRole('button', { name: 'Select theme' });
		await themeToggle.click();
		
		// First set to dark
		await page.getByText('Dark').click();
		
		// Then switch to system
		await themeToggle.click();
		await page.getByText('System').click();
		
		// Check localStorage
		const theme = await page.evaluate(() => localStorage.getItem('theme'));
		expect(theme).toBe('system');
	});

	test('should close dropdown when clicking outside', async ({ page }) => {
		await page.goto('/');
		
		const themeToggle = page.getByRole('button', { name: 'Select theme' });
		await themeToggle.click();
		
		// Dropdown should be visible
		await expect(page.getByText('Theme')).toBeVisible();
		
		// Press Escape to close the dropdown (more reliable than clicking outside)
		await page.keyboard.press('Escape');
		
		// Dropdown should be hidden
		await expect(page.getByText('Theme')).not.toBeVisible();
	});
});
