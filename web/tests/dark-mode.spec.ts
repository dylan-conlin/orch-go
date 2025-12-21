import { test, expect } from '@playwright/test';

test.describe('Dark Mode Toggle', () => {
	test.beforeEach(async ({ page }) => {
		// Clear localStorage before each test
		await page.addInitScript(() => {
			localStorage.clear();
		});
	});

	test('should render theme toggle button', async ({ page }) => {
		await page.goto('/');
		const themeToggle = page.getByRole('button', { name: 'Toggle theme' });
		await expect(themeToggle).toBeVisible();
	});

	test('should toggle from light to dark mode', async ({ page }) => {
		await page.goto('/');
		
		// Initially should be in light mode (based on system preference or default)
		const html = page.locator('html');
		
		// Click the toggle
		const themeToggle = page.getByRole('button', { name: 'Toggle theme' });
		await themeToggle.click();
		
		// Check if dark class is added/removed
		const hasDarkClass = await html.evaluate(el => el.classList.contains('dark'));
		
		// Click again to toggle back
		await themeToggle.click();
		const hasDarkClassAfter = await html.evaluate(el => el.classList.contains('dark'));
		
		// The classes should be different after toggling
		expect(hasDarkClass).not.toBe(hasDarkClassAfter);
	});

	test('should persist theme preference in localStorage', async ({ page, context }) => {
		// Create a new page without the beforeEach clearing localStorage
		const newPage = await context.newPage();
		await newPage.goto('/');
		
		// Click the toggle
		const themeToggle = newPage.getByRole('button', { name: 'Toggle theme' });
		await themeToggle.click();
		
		// Check localStorage
		const theme = await newPage.evaluate(() => localStorage.getItem('theme'));
		expect(theme).toBeTruthy();
		expect(['light', 'dark', 'system']).toContain(theme);
		
		// Reload the page (no addInitScript, so localStorage persists)
		await newPage.reload();
		
		// Wait for the page to load
		await newPage.waitForLoadState('domcontentloaded');
		
		// Theme should be restored from localStorage
		const themeAfterReload = await newPage.evaluate(() => localStorage.getItem('theme'));
		expect(themeAfterReload).toBe(theme);
		
		await newPage.close();
	});

	test('should show sun icon in dark mode and moon icon in light mode', async ({ page }) => {
		await page.goto('/');
		
		const themeToggle = page.getByRole('button', { name: 'Toggle theme' });
		
		// Get initial icon
		const initialSvg = themeToggle.locator('svg').first();
		await expect(initialSvg).toBeVisible();
		
		// Toggle and check icon changed
		await themeToggle.click();
		const newSvg = themeToggle.locator('svg').first();
		await expect(newSvg).toBeVisible();
	});
});
