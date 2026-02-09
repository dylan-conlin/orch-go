import { expect, test } from '@playwright/test'

async function mockWorkGraphShell(page: import('@playwright/test').Page) {
  await page.route('**/api/context', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        project_dir: '/test/model-probe',
        project: 'model-probe-test',
      }),
    })
  })

  await page.route('**/api/beads/graph**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ nodes: [], edges: [], node_count: 0, edge_count: 0 }),
    })
  })

  await page.route('**/api/agents**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify([]),
    })
  })

  await page.route('**/api/attention**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ signals: [], completedIssues: [] }),
    })
  })

  await page.route('**/api/beads/ready**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ issues: [] }),
    })
  })

  await page.route('**/api/daemon**', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ running: false, enabled: false }),
    })
  })

  await page.route('**/api/focus', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ has_focus: false, is_drifting: false }),
    })
  })
}

test.describe('Model probe dashboard section', () => {
  test('renders model cards and queue with timeline collapsed by default', async ({
    page,
  }) => {
    await mockWorkGraphShell(page)

    await page.route('**/api/kb/artifacts**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ needs_decision: [], recent: [], by_type: {} }),
      })
    })

    await page.route('**/api/kb/model-probes**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          summary: {
            models_total: 1,
            probes_total: 2,
            needs_review: 1,
            stale: 0,
            well_validated: 0,
          },
          queue: [
            {
              probe_path: '.kb/models/daemon/probes/queue-probe.md',
              title: 'Queue Probe',
              model: 'daemon',
              verdict: 'contradicts',
              date: '2026-02-09',
              claim: 'Claim from question',
              merged: false,
            },
          ],
          models: [
            {
              name: 'daemon',
              path: '.kb/models/daemon.md',
              last_updated: '2026-02-09',
              status: 'needs_review',
              probe_counts: { confirms: 0, extends: 1, contradicts: 1 },
              unmerged_count: 2,
              last_probe_at: '2026-02-09',
              probes: [
                {
                  probe_path: '.kb/models/daemon/probes/timeline-probe.md',
                  title: 'Timeline Probe Hidden',
                  model: 'daemon',
                  verdict: 'extends',
                  date: '2026-02-08',
                  claim: 'Timeline claim',
                  merged: false,
                },
              ],
            },
          ],
        }),
      })
    })

    await page.goto('/work-graph')
    await expect(page.getByText('model-probe')).toBeVisible()
    await page.getByRole('button', { name: 'Artifacts' }).click()

    await expect(page.locator('[data-testid="model-probe-section"]')).toBeVisible()
    await expect(page.locator('[data-testid="model-probe-queue"]')).toContainText(
      'Queue Probe',
    )
    await expect(page.getByText('Timeline Probe Hidden')).toHaveCount(0)

    await page.getByRole('button', { name: 'show timeline' }).first().click()
    await expect(page.getByText('Timeline Probe Hidden')).toBeVisible()
  })

  test('stays single-column at 666px without horizontal overflow', async ({ page }) => {
    await page.setViewportSize({ width: 666, height: 900 })
    await mockWorkGraphShell(page)

    await page.route('**/api/kb/artifacts**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ needs_decision: [], recent: [], by_type: {} }),
      })
    })

    await page.route('**/api/kb/model-probes**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          summary: {
            models_total: 1,
            probes_total: 1,
            needs_review: 1,
            stale: 0,
            well_validated: 0,
          },
          queue: [
            {
              probe_path: '.kb/models/daemon/probes/queue-probe.md',
              model: 'daemon',
              verdict: 'extends',
              date: '2026-02-09',
              claim: 'Claim from question',
              merged: false,
            },
          ],
          models: [
            {
              name: 'daemon',
              path: '.kb/models/daemon.md',
              last_updated: '2026-02-09',
              status: 'needs_review',
              probe_counts: { confirms: 0, extends: 1, contradicts: 0 },
              unmerged_count: 1,
              last_probe_at: '2026-02-09',
              probes: [],
            },
          ],
        }),
      })
    })

    await page.goto('/work-graph')
    await expect(page.getByText('model-probe')).toBeVisible()
    await page.getByRole('button', { name: 'Artifacts' }).click()
    await expect(page.locator('[data-testid="model-probe-section"]')).toBeVisible()

    const bodyWidth = await page.evaluate(() => document.body.scrollWidth)
    expect(bodyWidth).toBeLessThanOrEqual(666)

    const columns = page.locator('[data-testid="model-probe-section"] .grid > div')
    const modelsColumn = await columns.nth(0).boundingBox()
    const queueColumn = await columns.nth(1).boundingBox()
    expect(modelsColumn).not.toBeNull()
    expect(queueColumn).not.toBeNull()

    if (modelsColumn && queueColumn) {
      expect(Math.abs(modelsColumn.x - queueColumn.x)).toBeLessThan(4)
      expect(queueColumn.y).toBeGreaterThan(modelsColumn.y)
    }
  })
})
