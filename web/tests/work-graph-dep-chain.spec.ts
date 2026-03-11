import { test, expect } from '@playwright/test';

test.use({
	baseURL: 'http://localhost:5188'
});

const mockGraphResponse = {
	nodes: [
		{
			id: 'orch-go-1',
			title: 'Upstream Root',
			type: 'task',
			status: 'open',
			priority: 1,
			source: 'beads',
			layer: 0,
			created_at: '2026-02-19T10:00:00Z'
		},
		{
			id: 'orch-go-2',
			title: 'Midstream A',
			type: 'task',
			status: 'open',
			priority: 1,
			source: 'beads',
			layer: 1,
			created_at: '2026-02-19T10:00:00Z'
		},
		{
			id: 'orch-go-3',
			title: 'Midstream B',
			type: 'task',
			status: 'open',
			priority: 1,
			source: 'beads',
			layer: 1,
			created_at: '2026-02-19T10:00:00Z'
		},
		{
			id: 'orch-go-4',
			title: 'Downstream Gate',
			type: 'task',
			status: 'open',
			priority: 1,
			source: 'beads',
			layer: 2,
			created_at: '2026-02-19T10:00:00Z'
		}
	],
	edges: [
		{ from: 'orch-go-1', to: 'orch-go-2', type: 'blocks' },
		{ from: 'orch-go-1', to: 'orch-go-3', type: 'blocks' },
		{ from: 'orch-go-2', to: 'orch-go-4', type: 'blocks' },
		{ from: 'orch-go-3', to: 'orch-go-4', type: 'blocks' }
	],
	node_count: 4,
	edge_count: 4,
	project_dir: '/Users/dylanconlin/Documents/personal/orch-go'
};

async function stubWorkGraphApis(page: import('@playwright/test').Page) {
	await page.route('http://localhost:3348/api/context', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				project_dir: '/Users/dylanconlin/Documents/personal/orch-go',
				project: 'orch-go'
			})
		});
	});

	await page.route('http://localhost:3348/api/beads/graph**', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(mockGraphResponse)
		});
	});

	await page.route('http://localhost:3348/api/agents**', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify([])
		});
	});

	await page.route('http://localhost:3348/api/attention**', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				items: [],
				total: 0,
				sources: [],
				role: 'test',
				collected_at: new Date().toISOString()
			})
		});
	});

	await page.route('http://localhost:3348/api/focus', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({ has_focus: false, is_drifting: false })
		});
	});

	await page.route('http://localhost:3348/api/daemon', async (route) => {
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				running: false,
				ready_count: 0,
				capacity_max: 0,
				capacity_used: 0,
				capacity_free: 0
			})
		});
	});

	await page.route('http://localhost:3348/api/events', async (route) => {
		await route.fulfill({
			status: 200,
			headers: {
				'Content-Type': 'text/event-stream'
			},
			body: '\n'
		});
	});
}

test.describe('Work Graph dependency chain ordering', () => {
	test('renders upstream-first order and gate separator above convergence', async ({ page }) => {
		await stubWorkGraphApis(page);
		await page.goto('/work-graph');

		await expect(page.getByTestId('issue-row-orch-go-1')).toBeVisible();
		await expect(page.getByTestId('issue-row-orch-go-4')).toBeVisible();

		const rowIds = await page.locator('[data-testid^="issue-row-"]').evaluateAll((els) =>
			els.map((el) => el.getAttribute('data-testid'))
		);

		expect(rowIds[0]).toBe('issue-row-orch-go-1');
		expect(rowIds[rowIds.length - 1]).toBe('issue-row-orch-go-4');

		const idx2 = rowIds.indexOf('issue-row-orch-go-2');
		const idx3 = rowIds.indexOf('issue-row-orch-go-3');
		const idx4 = rowIds.indexOf('issue-row-orch-go-4');
		expect(idx4).toBeGreaterThan(idx2);
		expect(idx4).toBeGreaterThan(idx3);

		await expect(page.getByTestId('issue-row-orch-go-1')).toContainText('◆');
		await expect(page.getByTestId('issue-row-orch-go-4')).not.toContainText('◆');

		const gateLabel = page.getByText('gate', { exact: true }).first();
		await expect(gateLabel).toBeVisible();
		const nextGateRow = await gateLabel.evaluate((el) =>
			el.closest('div')?.nextElementSibling?.getAttribute('data-testid')
		);
		expect(nextGateRow).toBe('issue-row-orch-go-4');
	});
});
