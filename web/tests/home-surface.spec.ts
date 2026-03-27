import { test, expect, type Page } from '@playwright/test';

test.use({
	baseURL: 'http://localhost:4173'
});

async function mockHomeSurface(page: Page) {
	await page.route('**/api/threads', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify([
				{
					name: 'threads-as-primary-artifact-thinking',
					title: 'Threads as primary artifact',
					status: 'active',
					created: '2026-03-24',
					updated: '2026-03-26',
					latest_entry: 'Reading through briefs now reveals the actual argument instead of asking Dylan to infer product identity from counts and badges.',
					entry_count: 4,
					filename: '2026-03-24-threads-as-primary-artifact-thinking.md'
				}
			])
		});
	});

	await page.route('**/api/briefs', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify([
				{
					beads_id: 'orch-go-uiv9d',
					marked_read: false,
					thread_title: 'Threads as primary artifact',
					has_tension: true
				}
			])
		});
	});

	await page.route('**/api/briefs/orch-go-uiv9d**', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				beads_id: 'orch-go-uiv9d',
				marked_read: false,
				content: `# Brief: orch-go-uiv9d

## Frame

The home surface still feels like monitoring because it asks the reader to decode status chrome instead of reading the substance directly.

## Resolution

Render the frame, resolution, and tension excerpts inline so the page teaches the product through actual prose above the fold.

## Tension

How much content can the surface expose before it becomes a wall of text instead of an invitation to read?`
			})
		});
	});

	await page.route('**/api/questions', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				open: [
					{
						id: 'orch-go-f8y50',
						title: 'What is the minimum comprehension surface that unmistakably feels like the product?',
						status: 'open',
						priority: 1,
						blocking: ['orch-go-3l45p']
					}
				],
				investigating: [],
				answered: [],
				total_count: 1
			})
		});
	});

	await page.route('**/api/beads', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				total_issues: 0,
				open_issues: 0,
				in_progress_issues: 0,
				blocked_issues: 0,
				ready_issues: 0,
				closed_issues: 0
			})
		});
	});

	await page.route('**/api/beads/ready', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ issues: [], count: 0 })
		});
	});

	await page.route('**/api/beads/review-queue', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ issues: [], count: 0 })
		});
	});

	await page.route('**/api/context', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				project_dir: '/Users/dylanconlin/Documents/personal/orch-go',
				project: 'orch-go',
				included_projects: ['orch-go']
			})
		});
	});

	await page.route('**/api/agents**', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify([])
		});
	});

	await page.route('**/api/events/context', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'text/event-stream',
			headers: {
				'Cache-Control': 'no-cache',
				Connection: 'keep-alive'
			},
			body: ''
		});
	});

	await page.route('**/api/events', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'text/event-stream',
			headers: {
				'Cache-Control': 'no-cache',
				Connection: 'keep-alive'
			},
			body: 'event: connected\ndata: {"source":"mock"}\n\n'
		});
	});

	await page.route('**/api/usage', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				account: 'work',
				five_hour_percent: 10,
				weekly_percent: 20
			})
		});
	});
}

test('renders thread, brief, and question content inline on the home surface', async ({ page }) => {
	await mockHomeSurface(page);

	await page.goto('/');

	await expect(page.getByTestId('thread-inline-entry-threads-as-primary-artifact-thinking')).toContainText('Reading through briefs now reveals the actual argument');
	await expect(page.getByTestId('home-briefs-card')).toContainText('The home surface still feels like monitoring');
	await expect(page.getByTestId('home-brief-section-orch-go-uiv9d-resolution')).toContainText('Render the frame, resolution, and tension excerpts inline');
	await expect(page.getByTestId('home-questions-card')).toContainText('What is the minimum comprehension surface that unmistakably feels like the product?');
});
