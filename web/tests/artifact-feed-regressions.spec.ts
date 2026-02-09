import { expect, test, type Page } from '@playwright/test'

type ArtifactRequest = {
  projectDir: string | null
  since: string | null
}

function makeArtifact(index: number, titlePrefix: string) {
  return {
    path: `.kb/investigations/${titlePrefix.toLowerCase()}-${index}.md`,
    title: `${titlePrefix} ${index}`,
    type: 'investigation',
    status: 'Active',
    date: '2026-02-08',
    summary: 'Regression fixture',
    recommendation: false,
    modified_at: '2026-02-08T10:00:00Z',
    relative_time: '1h ago',
  }
}

async function mockWorkGraphShell(page: Page, getProjectDir: () => string) {
  await page.route('**/api/context', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        project_dir: getProjectDir(),
        project: 'test-project',
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

test.describe('Artifact Feed regressions', () => {
  test('artifacts pane is user-scrollable when list overflows', async ({ page }) => {
    let currentProjectDir = '/test/project-a'
    await mockWorkGraphShell(page, () => currentProjectDir)

    await page.route('**/api/kb/artifacts**', async (route) => {
      const needsDecision = Array.from({ length: 25 }, (_, i) =>
        makeArtifact(i, 'Needs Decision'),
      )
      const recent = Array.from({ length: 25 }, (_, i) => makeArtifact(i, 'Recent Item'))
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ needs_decision: needsDecision, recent, by_type: {} }),
      })
    })

    await page.goto('/work-graph')
    await expect(page.getByText('project-a')).toBeVisible()
    await page.getByRole('button', { name: 'Artifacts' }).click()
    await expect(page.getByText('RECENTLY UPDATED')).toBeVisible()

    const feed = page.locator('.artifact-feed')
    const overflowY = await feed.evaluate((el) => getComputedStyle(el).overflowY)
    expect(overflowY).toBe('auto')

    const metrics = await feed.evaluate((el) => ({
      scrollHeight: el.scrollHeight,
      clientHeight: el.clientHeight,
      scrollTop: el.scrollTop,
    }))
    expect(metrics.scrollHeight).toBeGreaterThan(metrics.clientHeight)

    await feed.hover()
    await page.mouse.wheel(0, 800)
    await page.waitForTimeout(100)

    const scrolledTop = await feed.evaluate((el) => el.scrollTop)
    expect(scrolledTop).toBeGreaterThan(metrics.scrollTop)
  })

  test('selected time filter persists during background refresh', async ({ page }) => {
    test.setTimeout(50000)

    let currentProjectDir = '/test/project-a'
    const artifactRequests: ArtifactRequest[] = []
    await mockWorkGraphShell(page, () => currentProjectDir)

    await page.route('**/api/kb/artifacts**', async (route) => {
      const url = new URL(route.request().url())
      const since = url.searchParams.get('since')
      const projectDir = url.searchParams.get('project_dir')
      artifactRequests.push({ since, projectDir })

      const title = since === '24h' ? 'Recent 24h Artifact' : 'Recent 7d Artifact'
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          needs_decision: [],
          recent: [makeArtifact(1, title)],
          by_type: {},
        }),
      })
    })

    await page.goto('/work-graph')
    await expect(page.getByText('project-a')).toBeVisible()
    await page.getByRole('button', { name: 'Artifacts' }).click()
    await expect(page.getByText('RECENTLY UPDATED')).toBeVisible()

    await page.locator('select').selectOption('24h')
    await expect(page.getByText('Recent 24h Artifact')).toBeVisible()

    // Wait for work-graph polling cycle (30s) to trigger a background artifacts refresh.
    await page.waitForTimeout(31000)

    expect(artifactRequests.length).toBeGreaterThanOrEqual(2)
    const lastRequest = artifactRequests[artifactRequests.length - 1]
    expect(lastRequest.since).toBe('24h')

    // Also ensure project change refresh keeps the active filter.
    currentProjectDir = '/test/project-b'
    await page.waitForTimeout(15500)

    const latestRequest = artifactRequests[artifactRequests.length - 1]
    expect(latestRequest.since).toBe('24h')
    expect(latestRequest.projectDir).toBe('/test/project-b')
  })
})
