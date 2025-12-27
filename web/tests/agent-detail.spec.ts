import { test, expect } from '@playwright/test';

// Use existing dev server instead of building/previewing
test.use({
	baseURL: 'http://localhost:5188'
});

test.describe('Agent Detail Panel', () => {
	test('should show slide-out panel when clicking agent card', async ({ page }) => {
		// Navigate to dashboard (dev server)
		await page.goto('/');
		
		// Switch to historical mode to access agent-sections
		await page.getByTestId('mode-toggle').getByRole('button', { name: /History/ }).click();
		
		// Wait for agents to load
		await page.waitForSelector('[data-testid="agent-sections"]', { timeout: 10000 });
		
		// Find and click an agent card (buttons within the grid, not the section toggles)
		const agentCard = page.locator('.grid button.group').first();
		
		// If no agents exist, skip the test
		const cardCount = await agentCard.count();
		if (cardCount === 0) {
			test.skip();
			return;
		}
		
		// Click the agent card
		await agentCard.click();
		
		// Verify the slide-out panel appears
		await expect(page.getByRole('dialog')).toBeVisible();
		await expect(page.getByText('Agent Details')).toBeVisible();
		
		// Verify identifiers section is shown
		await expect(page.getByText('Identifiers')).toBeVisible();
		
		// Verify context section is shown
		await expect(page.getByText('Context')).toBeVisible();
		
		// Take a screenshot for visual verification
		await page.screenshot({ path: 'test-results/agent-detail-panel.png', fullPage: true });
	});
	
	test('should close panel when clicking backdrop', async ({ page }) => {
		await page.goto('/');
		// Switch to historical mode to access agent-sections
		await page.getByTestId('mode-toggle').getByRole('button', { name: /History/ }).click();
		await page.waitForSelector('[data-testid="agent-sections"]', { timeout: 10000 });
		
		const agentCard = page.locator('.grid button.group').first();
		const cardCount = await agentCard.count();
		if (cardCount === 0) {
			test.skip();
			return;
		}
		
		// Open the panel
		await agentCard.click();
		await expect(page.getByRole('dialog')).toBeVisible();
		
		// Click the backdrop (the button that covers the left side)
		await page.locator('button[aria-label="Close panel"]').click();
		
		// Panel should close
		await expect(page.getByRole('dialog')).not.toBeVisible();
	});
	
	test('should close panel when pressing Escape', async ({ page }) => {
		await page.goto('/');
		// Switch to historical mode to access agent-sections
		await page.getByTestId('mode-toggle').getByRole('button', { name: /History/ }).click();
		await page.waitForSelector('[data-testid="agent-sections"]', { timeout: 10000 });
		
		const agentCard = page.locator('.grid button.group').first();
		const cardCount = await agentCard.count();
		if (cardCount === 0) {
			test.skip();
			return;
		}
		
		// Open the panel
		await agentCard.click();
		await expect(page.getByRole('dialog')).toBeVisible();
		
		// Press Escape
		await page.keyboard.press('Escape');
		
		// Panel should close
		await expect(page.getByRole('dialog')).not.toBeVisible();
	});
	
	test('should show selected state on agent card', async ({ page }) => {
		await page.goto('/');
		// Switch to historical mode to access agent-sections
		await page.getByTestId('mode-toggle').getByRole('button', { name: /History/ }).click();
		await page.waitForSelector('[data-testid="agent-sections"]', { timeout: 10000 });
		
		const agentCard = page.locator('.grid button.group').first();
		const cardCount = await agentCard.count();
		if (cardCount === 0) {
			test.skip();
			return;
		}
		
		// Click the agent card
		await agentCard.click();
		
		// Verify selected styling (ring)
		await expect(agentCard).toHaveClass(/ring-2/);
	});
});
