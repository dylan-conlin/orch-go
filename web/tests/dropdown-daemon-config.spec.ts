import { test, expect } from '@playwright/test';

test.describe('Daemon Config Dropdown', () => {
	test('should render dropdown content when daemon indicator is clicked', async ({ page }) => {
		// Mock the daemon API response
		await page.route('**/api/daemon', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					running: true,
					status: 'running',
					capacity_used: 2,
					capacity_max: 3,
					capacity_free: 1,
					ready_count: 5,
					last_poll_ago: '2s ago'
				})
			});
		});

		// Mock the daemon config API response
		await page.route('**/api/config/daemon', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					poll_interval: 30,
					max_agents: 3,
					label: 'triage:ready',
					verbose: false,
					reflect_issues: true,
					working_directory: '/Users/test',
					path: '/usr/local/bin/orch'
				})
			});
		});

		// Mock the drift status API response
		await page.route('**/api/config/drift', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					in_sync: true,
					plist_path: '/Users/test/Library/LaunchAgents/com.orch.daemon.plist',
					plist_exists: true,
					config_path: '/Users/test/.orch/config.yaml'
				})
			});
		});

		await page.goto('/');
		
		// Wait for the daemon indicator to appear
		const daemonIndicator = page.getByTestId('daemon-indicator');
		await expect(daemonIndicator).toBeVisible();
		
		// Click the daemon indicator
		await daemonIndicator.click();
		
		// Wait a bit for dropdown to appear
		await page.waitForTimeout(500);
		
		// Check if dropdown content is visible
		// Look for the "Daemon Settings" text which should be in the dropdown
		const dropdownContent = page.locator('text=Daemon Settings');
		await expect(dropdownContent).toBeVisible({ timeout: 2000 });
		
		// Check if the poll interval input is visible
		const pollIntervalInput = page.locator('input#poll-interval');
		await expect(pollIntervalInput).toBeVisible();
	});
});
