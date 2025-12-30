import { test, expect } from '@playwright/test';

// NOTE: Dashboard mode toggle has been removed in favor of unified attention-first layout
// These tests verify the new single-view dashboard behavior

test.describe('Dashboard Single View (Mode Toggle Removed)', () => {
	test('should not have mode toggle', async ({ page }) => {
		await page.goto('/');
		
		const modeToggle = page.getByTestId('mode-toggle');
		await expect(modeToggle).not.toBeVisible();
	});

	test('should show unified view without URL mode switching', async ({ page }) => {
		// Old URL params should be ignored (or redirect gracefully)
		await page.goto('/?tab=ops');
		
		// Mode toggle should not exist
		const modeToggle = page.getByTestId('mode-toggle');
		await expect(modeToggle).not.toBeVisible();
		
		// Active agents section should be visible (always)
		await expect(page.getByTestId('active-agents-section')).toBeVisible();
	});

	test('should ignore historical mode URL param', async ({ page }) => {
		await page.goto('/?tab=history');
		
		// Mode toggle should not exist
		const modeToggle = page.getByTestId('mode-toggle');
		await expect(modeToggle).not.toBeVisible();
		
		// Active agents section should be visible (always)
		await expect(page.getByTestId('active-agents-section')).toBeVisible();
	});

	test('should maintain backward compatibility for localStorage', async ({ page }) => {
		await page.goto('/');
		
		// Set old localStorage value
		await page.evaluate(() => {
			localStorage.setItem('orch-dashboard-mode', 'historical');
		});
		
		await page.reload();
		
		// Should still work (ignore the stored mode)
		await expect(page.getByTestId('active-agents-section')).toBeVisible();
	});
});
