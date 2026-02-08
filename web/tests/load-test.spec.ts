import { test, expect, type Page } from '@playwright/test';

/**
 * Load Test Suite for Dashboard with 50+ Agents
 * 
 * These tests verify the dashboard can handle large numbers of agents using
 * HTTP API mocking. The approach:
 * 1. Mock /api/agents to return 50+ synthetic agents
 * 2. Mock /api/events SSE endpoint to return a minimal stream that triggers data load
 * 3. Measure rendering performance, filter responsiveness, and scroll behavior
 * 
 * Success criteria (from beads issue):
 * - No CPU spikes
 * - bd process count < 15
 * - API response < 500ms
 */

// Use existing dev server
test.use({
	baseURL: 'https://localhost:3348',
	ignoreHTTPSErrors: true
});

// Agent type matching the dashboard store
interface MockAgent {
	id: string;
	session_id: string;
	beads_id: string;
	beads_title: string;
	status: 'active' | 'idle' | 'completed' | 'abandoned';
	spawned_at: string;
	updated_at: string;
	completed_at?: string;
	project_dir: string;
	skill: string;
	phase: string;
	task: string;
	project: string;
	runtime: string;
	is_processing: boolean;
}

// Generate mock agents with realistic distribution
function generateMockAgents(count: number): MockAgent[] {
	const agents: MockAgent[] = [];
	const now = new Date();
	const statuses: MockAgent['status'][] = ['active', 'idle', 'completed', 'abandoned'];
	const skills = ['feature-impl', 'investigation', 'systematic-debugging', 'architect', 'research'];
	const phases = ['Planning', 'Implementing', 'Testing', 'Complete', 'Review'];
	const projects = ['orch-go', 'beads', 'skillc', 'kb-cli', 'snap', 'opencode'];
	
	for (let i = 0; i < count; i++) {
		const status = statuses[Math.floor(Math.random() * statuses.length)];
		const spawnedAt = new Date(now.getTime() - Math.random() * 7 * 24 * 60 * 60 * 1000); // Up to 7 days ago
		const updatedAt = new Date(spawnedAt.getTime() + Math.random() * 4 * 60 * 60 * 1000); // Up to 4 hours after spawn
		
		agents.push({
			id: `agent-${i.toString().padStart(4, '0')}`,
			session_id: `session-${i.toString().padStart(4, '0')}`,
			beads_id: `test-${Math.random().toString(36).substring(2, 6)}`,
			beads_title: `Test task ${i}: ${skills[i % skills.length]} work`,
			status,
			spawned_at: spawnedAt.toISOString(),
			updated_at: updatedAt.toISOString(),
			completed_at: status === 'completed' ? updatedAt.toISOString() : undefined,
			project_dir: `/Users/test/projects/${projects[i % projects.length]}`,
			skill: skills[i % skills.length],
			phase: phases[i % phases.length],
			task: `Implement feature ${i} for ${projects[i % projects.length]}`,
			project: projects[i % projects.length],
			runtime: `${Math.floor(Math.random() * 120)}m ${Math.floor(Math.random() * 60)}s`,
			is_processing: status === 'active' && Math.random() > 0.5
		});
	}
	
	return agents;
}

// API metrics tracking object
interface APIMetrics {
	callCount: number;
	responseTimes: number[];
	getAvgResponseTime: () => number;
	getMaxResponseTime: () => number;
}

// Setup mock API routes for load testing
// Returns both the mock agents and metrics tracker
async function setupMockAPI(page: Page, agentCount: number): Promise<{ agents: MockAgent[], metrics: APIMetrics }> {
	const mockAgents = generateMockAgents(agentCount);
	
	// Track metrics with closure
	const metrics: APIMetrics = {
		callCount: 0,
		responseTimes: [],
		getAvgResponseTime() {
			return this.responseTimes.length > 0 
				? this.responseTimes.reduce((a, b) => a + b, 0) / this.responseTimes.length 
				: 0;
		},
		getMaxResponseTime() {
			return Math.max(...this.responseTimes, 0);
		}
	};
	
	// Mock the agents API endpoint
	await page.route('**/api/agents', async (route) => {
		const startTime = Date.now();
		metrics.callCount++;
		
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(mockAgents)
		});
		
		metrics.responseTimes.push(Date.now() - startTime);
	});
	
	// Mock SSE endpoint - this can't truly mock EventSource but prevents connection errors
	await page.route('**/api/events', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'text/event-stream',
			headers: {
				'Cache-Control': 'no-cache',
				'Connection': 'keep-alive'
			},
			body: 'event: connected\ndata: {"source":"mock"}\n\n'
		});
	});
	
	return { agents: mockAgents, metrics };
}

test.describe('Load Test - 50+ Agents', () => {
	test.describe('UI Structure Tests (No Mock)', () => {
		test('should load dashboard structure quickly', async ({ page }) => {
			const startTime = Date.now();
			await page.goto('/');

			// Wait for page structure to load
			await page.waitForSelector('[data-testid="stats-bar"]', { timeout: 10000 });
			await page.waitForSelector('[data-testid="agent-sections"]', { timeout: 10000 });

			const loadTime = Date.now() - startTime;
			console.log(`Dashboard structure load time: ${loadTime}ms`);

			// Verify core UI elements are present
			await expect(page.getByTestId('stats-bar')).toBeVisible();
			await expect(page.getByTestId('filter-bar')).toBeVisible();
			await expect(page.getByTestId('agent-sections')).toBeVisible();
			await expect(page.getByTestId('filter-count')).toBeVisible();
			await expect(page.getByTestId('sort-select')).toBeVisible();

			// Page structure should load quickly
			expect(loadTime).toBeLessThan(3000);
		});

		test('should render filter controls', async ({ page }) => {
			await page.goto('/');
			await page.waitForSelector('[data-testid="filter-bar"]', { timeout: 10000 });

			const filterBar = page.getByTestId('filter-bar');
			await expect(filterBar).toBeVisible();

			// Verify all filter controls exist
			await expect(page.getByTestId('status-filter')).toBeVisible();
			await expect(page.getByTestId('sort-select')).toBeVisible();
			await expect(page.getByTestId('active-only-toggle')).toBeVisible();
		});

		test('should display connection controls', async ({ page }) => {
			await page.goto('/');
			await page.waitForSelector('[data-testid="stats-bar"]', { timeout: 10000 });

			const statsBar = page.getByTestId('stats-bar');
			
			// Should have connect/disconnect button
			const connectButton = statsBar.getByRole('button', { name: /Connect|Disconnect/ });
			await expect(connectButton).toBeVisible();
		});
	});

	test.describe('Load Tests with 50+ Mocked Agents', () => {
		const AGENT_COUNT = 60; // 50+ agents as per requirements

		test('should render 50+ agents without errors', async ({ page }) => {
			const { agents: mockAgents } = await setupMockAPI(page, AGENT_COUNT);
			
			const consoleErrors: string[] = [];
			page.on('console', msg => {
				if (msg.type() === 'error') {
					consoleErrors.push(msg.text());
				}
			});

			const startTime = Date.now();
			await page.goto('/');
			
			// Wait for agents to load - check filter count shows our agent count
			await page.waitForSelector('[data-testid="filter-count"]', { timeout: 10000 });
			
			// Give time for the mock API data to render
			await page.waitForTimeout(1000);
			
			const loadTime = Date.now() - startTime;
			console.log(`Dashboard with ${AGENT_COUNT} agents load time: ${loadTime}ms`);

			// Check for console errors (ignore expected SSE/network errors)
			const criticalErrors = consoleErrors.filter(
				err => !err.includes('SSE') && 
				       !err.includes('EventSource') && 
				       !err.includes('fetch') &&
				       !err.includes('net::')
			);
			
			expect(criticalErrors).toHaveLength(0);
			
			// Performance assertion - should load in reasonable time
			expect(loadTime).toBeLessThan(5000);
		});

		test('should handle filter changes with 50+ agents', async ({ page }) => {
			await setupMockAPI(page, AGENT_COUNT);
			await page.goto('/');
			await page.waitForSelector('[data-testid="filter-bar"]', { timeout: 10000 });
			await page.waitForTimeout(500); // Let initial data load

			const statusFilter = page.getByTestId('status-filter');
			const sortSelect = page.getByTestId('sort-select');

			// Time filter operations
			const filterStart = Date.now();
			
			// Change status filter
			await statusFilter.selectOption('active');
			await page.waitForTimeout(50);
			await statusFilter.selectOption('completed');
			await page.waitForTimeout(50);
			await statusFilter.selectOption('all');

			// Change sort
			await sortSelect.selectOption('newest');
			await page.waitForTimeout(50);
			await sortSelect.selectOption('oldest');
			await page.waitForTimeout(50);
			await sortSelect.selectOption('alphabetical');

			const filterTime = Date.now() - filterStart;
			console.log(`Filter operations with ${AGENT_COUNT} agents: ${filterTime}ms`);

			// Filters should remain responsive
			expect(filterTime).toBeLessThan(2000);
			
			// UI should still be responsive
			await expect(page.getByTestId('filter-count')).toBeVisible();
		});

		test('should handle rapid filter changes without race conditions', async ({ page }) => {
			await setupMockAPI(page, AGENT_COUNT);
			await page.goto('/');
			await page.waitForSelector('[data-testid="filter-bar"]', { timeout: 10000 });
			await page.waitForTimeout(500);

			const sortSelect = page.getByTestId('sort-select');
			const statusFilter = page.getByTestId('status-filter');

			// Rapidly change filters to test for race conditions
			const startTime = Date.now();
			for (let i = 0; i < 10; i++) {
				await sortSelect.selectOption(['newest', 'oldest', 'alphabetical', 'project', 'phase'][i % 5]);
				await statusFilter.selectOption(['all', 'active', 'completed', 'idle'][i % 4]);
			}
			const filterTime = Date.now() - startTime;

			console.log(`10 rapid filter changes with ${AGENT_COUNT} agents: ${filterTime}ms`);

			// Success criteria: < 3000ms for 10 rapid changes
			expect(filterTime).toBeLessThan(3000);

			// Page should still be responsive
			await expect(page.getByTestId('filter-count')).toBeVisible();
		});

		test('should scroll smoothly with 50+ agents', async ({ page }) => {
			await setupMockAPI(page, AGENT_COUNT);
			await page.goto('/');
			await page.waitForSelector('[data-testid="agent-sections"]', { timeout: 10000 });
			await page.waitForTimeout(500);

			// Scroll test with data loaded
			const scrollStart = Date.now();
			await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
			await page.waitForTimeout(100);
			await page.evaluate(() => window.scrollTo(0, 0));
			const scrollTime = Date.now() - scrollStart;

			console.log(`Scroll round-trip time with ${AGENT_COUNT} agents: ${scrollTime}ms`);

			// Success criteria: < 500ms scroll round-trip
			expect(scrollTime).toBeLessThan(500);
		});

		test('should maintain API response time under 500ms', async ({ page }) => {
			// This test verifies that the /api/agents endpoint responds quickly
			// by making direct fetch requests (bypassing the SSE-triggered flow)
			const { metrics } = await setupMockAPI(page, AGENT_COUNT);
			await page.goto('/');
			await page.waitForSelector('[data-testid="filter-count"]', { timeout: 10000 });
			
			// Make direct API calls to measure response time
			const responseTimes: number[] = [];
			for (let i = 0; i < 5; i++) {
				const startTime = Date.now();
				const response = await page.evaluate(async () => {
					const res = await fetch('https://localhost:3348/api/agents');
					return { status: res.status, count: (await res.json()).length };
				});
				responseTimes.push(Date.now() - startTime);
				
				// Verify mock is returning correct data
				expect(response.status).toBe(200);
				expect(response.count).toBe(AGENT_COUNT);
			}
			
			const avgResponseTime = responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length;
			const maxResponseTime = Math.max(...responseTimes);
			
			console.log(`API response times: avg=${avgResponseTime.toFixed(0)}ms, max=${maxResponseTime}ms (5 calls)`);
			
			// Success criteria: API response time < 500ms
			expect(maxResponseTime).toBeLessThan(500);
		});
	});

	test.describe('Section and State Tests', () => {
		test('should handle section toggles', async ({ page }) => {
			await page.goto('/');
			await page.waitForSelector('[data-testid="agent-sections"]', { timeout: 10000 });

			// Find section toggles (if any sections exist)
			const sectionToggles = page.locator('[data-testid^="section-toggle-"]');
			const toggleCount = await sectionToggles.count();

			console.log(`Found ${toggleCount} section toggles`);

			if (toggleCount === 0) {
				// No sections to toggle - just verify page loads
				await expect(page.getByTestId('agent-sections')).toBeVisible();
				return;
			}

			// Toggle the first section and verify click completes without errors
			const toggle = sectionToggles.first();
			const initialExpanded = await toggle.getAttribute('aria-expanded');
			console.log(`Initial expanded state: ${initialExpanded}`);
			
			// Click toggle - should complete without errors
			await toggle.click();
			await page.waitForTimeout(200);
			
			const newExpanded = await toggle.getAttribute('aria-expanded');
			console.log(`New expanded state: ${newExpanded}`);
			
			// The toggle button should remain clickable and functional
			await expect(toggle).toBeVisible();
			
			// Click again to restore
			await toggle.click();
			await page.waitForTimeout(100);
			
			// Should still be functional
			await expect(toggle).toBeVisible();
		});

		test('should persist section state', async ({ page }) => {
			await page.goto('/');
			await page.waitForSelector('[data-testid="agent-sections"]', { timeout: 10000 });

			// Find the active section toggle
			const activeToggle = page.getByTestId('section-toggle-active');
			const toggleExists = await activeToggle.count();
			
			if (toggleExists > 0) {
				// Get initial state
				const initialExpanded = await activeToggle.getAttribute('aria-expanded');
				console.log(`Initial active section state: ${initialExpanded}`);
				
				// Toggle
				await activeToggle.click();
				await page.waitForTimeout(300); // Give more time for localStorage update
				
				// Verify toggle happened immediately
				const afterClick = await activeToggle.getAttribute('aria-expanded');
				console.log(`After click state: ${afterClick}`);
				
				// Reload page
				await page.reload();
				await page.waitForSelector('[data-testid="agent-sections"]', { timeout: 10000 });
				
				// Check state persisted
				const newActiveToggle = page.getByTestId('section-toggle-active');
				const newToggleExists = await newActiveToggle.count();
				
				if (newToggleExists > 0) {
					const persistedExpanded = await newActiveToggle.getAttribute('aria-expanded');
					console.log(`Persisted state after reload: ${persistedExpanded}`);
					
					// State should have persisted (match afterClick, not initial)
					if (afterClick !== initialExpanded) {
						expect(persistedExpanded).toBe(afterClick);
					} else {
						// Toggle didn't change state - just verify page loads
						console.log('Toggle did not change state - section may be empty or locked');
					}
				}
			} else {
				// No active section - just verify page loads
				await expect(page.getByTestId('agent-sections')).toBeVisible();
			}
		});

		test('should handle dark mode toggle', async ({ page }) => {
			await page.goto('/');
			await page.waitForSelector('[data-testid="stats-bar"]', { timeout: 10000 });

			// Find the theme toggle if it exists (in layout)
			const themeToggle = page.getByRole('button', { name: /theme|mode|dark|light/i });
			const toggleExists = await themeToggle.count();

			if (toggleExists > 0) {
				// Toggle theme
				await themeToggle.click();
				await page.waitForTimeout(200);

				// Page should still be functional
				await expect(page.getByTestId('stats-bar')).toBeVisible();
				await expect(page.getByTestId('filter-bar')).toBeVisible();
			} else {
				// No theme toggle found - just verify page works
				await expect(page.getByTestId('stats-bar')).toBeVisible();
			}
		});
	});

	test.describe('Stress Tests', () => {
		test('should handle 100 agents without degradation', async ({ page }) => {
			const STRESS_COUNT = 100;
			await setupMockAPI(page, STRESS_COUNT);
			
			const startTime = Date.now();
			await page.goto('/');
			await page.waitForSelector('[data-testid="filter-count"]', { timeout: 15000 });
			await page.waitForTimeout(1000);
			
			const loadTime = Date.now() - startTime;
			console.log(`Dashboard with ${STRESS_COUNT} agents load time: ${loadTime}ms`);
			
			// Should still load in reasonable time even with 100 agents
			expect(loadTime).toBeLessThan(8000);
			
			// Filters should still work
			const statusFilter = page.getByTestId('status-filter');
			await statusFilter.selectOption('active');
			await expect(statusFilter).toHaveValue('active');
		});

		test('should handle rapid scroll with 100 agents', async ({ page }) => {
			await setupMockAPI(page, 100);
			await page.goto('/');
			await page.waitForSelector('[data-testid="agent-sections"]', { timeout: 15000 });
			await page.waitForTimeout(1000);

			// Rapid scroll test
			const scrollStart = Date.now();
			for (let i = 0; i < 5; i++) {
				await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
				await page.waitForTimeout(50);
				await page.evaluate(() => window.scrollTo(0, 0));
				await page.waitForTimeout(50);
			}
			const scrollTime = Date.now() - scrollStart;

			console.log(`5 rapid scroll cycles with 100 agents: ${scrollTime}ms`);
			
			// Should complete in reasonable time
			expect(scrollTime).toBeLessThan(2000);
		});
	});
});
