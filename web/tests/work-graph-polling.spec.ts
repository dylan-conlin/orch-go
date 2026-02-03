import { test, expect } from '@playwright/test';

// Phase 1.2: Polling and Refresh (orch-go-21164, orch-go-21171)
test.describe('Work Graph Polling and Refresh', () => {
	test('should start orchestratorContext polling on mount', async ({ page }) => {
		let contextFetchCount = 0;
		
		// Mock context API to track fetch count
		await page.route('**/api/context', async (route) => {
			contextFetchCount++;
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					project_dir: '/test/project',
					project: 'test-project'
				})
			});
		});
		
		// Mock graph API
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [],
					edges: [],
					node_count: 0,
					edge_count: 0
				})
			});
		});
		
		await page.goto('/work-graph');
		
		// Wait for initial mount
		await page.waitForTimeout(100);
		
		// Initial fetch should have happened
		expect(contextFetchCount).toBeGreaterThanOrEqual(1);
		
		// Wait for at least one polling cycle (2 seconds + buffer)
		await page.waitForTimeout(2500);
		
		// Should have polled multiple times
		expect(contextFetchCount).toBeGreaterThanOrEqual(2);
	});
	
	test('should re-fetch workGraph when project_dir changes', async ({ page }) => {
		let graphFetchCount = 0;
		let currentProjectDir = '/test/project1';
		
		// Mock context API that allows changing project_dir
		await page.route('**/api/context', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					project_dir: currentProjectDir,
					project: 'test-project'
				})
			});
		});
		
		// Mock graph API to track fetch count and project_dir
		await page.route('**/api/beads/graph**', async (route) => {
			graphFetchCount++;
			const url = new URL(route.request().url());
			const projectDirParam = url.searchParams.get('project_dir');
			
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [],
					edges: [],
					node_count: 0,
					edge_count: 0,
					project_dir: projectDirParam
				})
			});
		});
		
		await page.goto('/work-graph');
		await page.waitForTimeout(100);
		
		const initialFetchCount = graphFetchCount;
		
		// Change project_dir in context
		currentProjectDir = '/test/project2';
		
		// Wait for reactive block to trigger (context polls every 2 seconds)
		await page.waitForTimeout(2500);
		
		// Should have fetched again with new project_dir
		expect(graphFetchCount).toBeGreaterThan(initialFetchCount);
	});
	
	test('should re-fetch kbArtifacts when project_dir changes', async ({ page }) => {
		let artifactsFetchCount = 0;
		let currentProjectDir = '/test/project1';
		
		// Mock context API
		await page.route('**/api/context', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					project_dir: currentProjectDir,
					project: 'test-project'
				})
			});
		});
		
		// Mock graph API
		await page.route('**/api/beads/graph**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [],
					edges: [],
					node_count: 0,
					edge_count: 0
				})
			});
		});
		
		// Mock artifacts API to track fetch count
		await page.route('**/api/kb/artifacts**', async (route) => {
			artifactsFetchCount++;
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					needs_decision: [],
					recent: []
				})
			});
		});
		
		await page.goto('/work-graph');
		
		// Switch to artifacts view to trigger initial fetch
		await page.keyboard.press('Tab');
		await page.waitForTimeout(500);
		
		const initialFetchCount = artifactsFetchCount;
		
		// Change project_dir in context
		currentProjectDir = '/test/project2';
		
		// Wait for reactive block to trigger
		await page.waitForTimeout(2500);
		
		// Should have fetched again with new project_dir
		expect(artifactsFetchCount).toBeGreaterThan(initialFetchCount);
	});
	
	test('should poll workGraph periodically at 5-second intervals', async ({ page }) => {
		let graphFetchCount = 0;
		
		// Mock context API
		await page.route('**/api/context', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					project_dir: '/test/project',
					project: 'test-project'
				})
			});
		});
		
		// Mock graph API to track fetch count
		await page.route('**/api/beads/graph**', async (route) => {
			graphFetchCount++;
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes: [],
					edges: [],
					node_count: 0,
					edge_count: 0
				})
			});
		});
		
		await page.goto('/work-graph');
		await page.waitForTimeout(100);
		
		const initialFetchCount = graphFetchCount;
		
		// Wait for at least one 5-second polling cycle + buffer
		await page.waitForTimeout(5500);
		
		// Should have polled at least once more
		expect(graphFetchCount).toBeGreaterThan(initialFetchCount);
	});
	
	test('should highlight newly appeared issues with 30-second visual signal', async ({ page }) => {
		let returnNewIssue = false;
		
		// Mock context API
		await page.route('**/api/context', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					project_dir: '/test/project',
					project: 'test-project'
				})
			});
		});
		
		// Mock agents API
		await page.route('**/api/agents**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify([])
			});
		});
		
		// Mock attention API
		await page.route('**/api/attention**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					signals: [],
					completedIssues: []
				})
			});
		});
		
		// Mock WIP queued API (beads/ready endpoint)
		await page.route('**/api/beads/ready**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ issues: [] })
			});
		});
		
		// Mock daemon API
		await page.route('**/api/daemon**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					enabled: false,
					running: false
				})
			});
		});
		
		// Mock graph API to return different data
		await page.route('**/api/beads/graph**', async (route) => {
			const nodes = [
				{
					id: 'orch-go-1',
					title: 'Existing Issue',
					type: 'task',
					status: 'open',
					priority: 2,
					source: 'beads'
				}
			];
			
			// Add new issue after flag is set
			if (returnNewIssue) {
				nodes.push({
					id: 'orch-go-2',
					title: 'New Issue',
					type: 'task',
					status: 'open',
					priority: 2,
					source: 'beads'
				});
			}
			
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					nodes,
					edges: [],
					node_count: nodes.length,
					edge_count: 0
				})
			});
		});
		
		await page.goto('/work-graph');
		
		// Wait for initial render
		await expect(page.getByText('Existing Issue')).toBeVisible();
		
		// New issue should not be visible yet
		await expect(page.getByText('New Issue')).not.toBeVisible();
		
		// Trigger new issue to appear
		returnNewIssue = true;
		
		// Wait for polling cycle (5 seconds + buffer for React/processing)
		await page.waitForTimeout(6000);
		
		// New issue should now be visible with highlight class
		const newIssueRow = page.locator('[data-testid="issue-row-orch-go-2"]');
		await expect(newIssueRow).toBeVisible();
		// Wait a bit for React/Svelte to update the class
		await page.waitForTimeout(500);
		await expect(newIssueRow).toHaveClass(/new-issue-highlight/);
		
		// Verify highlight is still present after 15 seconds
		await page.waitForTimeout(15000);
		await expect(newIssueRow).toHaveClass(/new-issue-highlight/);
		
		// Wait for animation duration to complete (30 seconds total)
		await page.waitForTimeout(15500);
		
		// Highlight class should be removed after 30 seconds
		await expect(newIssueRow).not.toHaveClass(/new-issue-highlight/);
	});
});
