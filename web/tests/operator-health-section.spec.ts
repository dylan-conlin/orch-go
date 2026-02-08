import { expect, test } from '@playwright/test';

test.describe('Operator Health Section', () => {
	test('renders operator health signals from API', async ({ page }) => {
		await page.route('**/api/operator-health**', async (route) => {
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({
					generated_at: new Date().toISOString(),
					crash_free_streak: {
						status: 'warning',
						current_streak_days: 2,
						current_streak_seconds: 172800,
						current_streak: '2d 0h 0m',
						target_days: 7,
						progress_percent: 28.5
					},
					resource_ceilings: {
						status: 'healthy',
						baseline: {
							goroutines: 10,
							heap_bytes: 1024,
							child_processes: 2,
							open_file_descriptors: 20
						},
						current: {
							goroutines: 12,
							heap_bytes: 1536,
							child_processes: 3,
							open_file_descriptors: 24
						},
						ceiling_multiplier: 2,
						breached: false
					},
					investigation_rate_30d: {
						status: 'critical',
						window_days: 30,
						count: 63,
						threshold: 50,
						warning_from: 40
					},
					defect_class_clusters: {
						status: 'warning',
						window_days: 30,
						total_top_n: 2,
						top_classes: [
							{ defect_class: 'resource-leak', count: 8, window_days: 30 },
							{ defect_class: 'state-drift', count: 6, window_days: 30 }
						]
					},
					agent_health_ratio_7d: {
						status: 'warning',
						window_days: 7,
						completions: 12,
						abandonments: 5,
						completion_share: 0.705,
						completions_per_abandonment: 2.4
					},
					process_census: {
						status: 'critical',
						child_processes: 6,
						orphaned_count: 2,
						orphaned_processes: [
							{ pid: 1234, ppid: 1, command: 'bun' },
							{ pid: 5678, ppid: 1, command: 'vite' }
						]
					}
				})
			});
		});

		await page.goto('/');

		const section = page.getByTestId('operator-health-section');
		await expect(section).toBeVisible();
		await expect(section).toContainText('Operator Health');
		await expect(section).toContainText('63');
		await expect(section).toContainText('resource-leak');
		await expect(section).toContainText('2 orphan process(es) with PPID=1');
	});
});
