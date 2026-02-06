import { test, expect } from '@playwright/test'

// Helper: mock attention API with completed issues
function mockAttentionWithIssues(items: any[]) {
  return {
    items,
    total: items.length,
    sources: ['beads-recently-closed'],
    role: 'human',
    collected_at: new Date().toISOString(),
  }
}

function makeCompletedItem(
  id: string,
  title: string,
  opts: { priority?: number; type?: string; verification_status?: string } = {},
) {
  return {
    id: `beads-recently-closed-${id}`,
    source: 'beads-recently-closed',
    concern: 'Verification',
    signal: 'recently-closed',
    subject: id,
    summary: `Closed 2h ago: ${title}`,
    priority: 50,
    role: 'human',
    collected_at: new Date().toISOString(),
    metadata: {
      closed_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
      status: 'closed',
      issue_type: opts.type || 'task',
      beads_priority: opts.priority ?? 2,
      verification_status: opts.verification_status,
    },
  }
}

test.describe('Completed View (Tab)', () => {
  test('should show Completed tab with count badge when issues exist', async ({
    page,
  }) => {
    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          mockAttentionWithIssues([
            makeCompletedItem('orch-go-001', 'First completed'),
            makeCompletedItem('orch-go-002', 'Second completed'),
            makeCompletedItem('orch-go-003', 'Third completed'),
          ]),
        ),
      })
    })

    await page.goto('/work-graph')
    await page.waitForTimeout(2000)

    // Completed tab should be visible with count badge
    const completedBtn = page.getByRole('button', { name: /Completed/ })
    await expect(completedBtn).toBeVisible()
    await expect(completedBtn.getByText('3')).toBeVisible()
  })

  test('should switch to completed view on tab click and show issues', async ({
    page,
  }) => {
    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          mockAttentionWithIssues([
            makeCompletedItem('orch-go-001', 'Completed issue one'),
            makeCompletedItem('orch-go-002', 'Completed issue two'),
          ]),
        ),
      })
    })

    await page.goto('/work-graph')
    await page.waitForTimeout(2000)

    // Click Completed tab
    await page.getByRole('button', { name: /Completed/ }).click()

    // Should show completed view container
    const view = page.locator('[data-testid="completed-view"]')
    await expect(view).toBeVisible()

    // Should show completed issues
    await expect(view.getByText('Completed issue one')).toBeVisible()
    await expect(view.getByText('Completed issue two')).toBeVisible()
  })

  test('should show empty state when no completed issues', async ({ page }) => {
    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(mockAttentionWithIssues([])),
      })
    })

    await page.goto('/work-graph')
    await page.waitForTimeout(2000)

    // Click Completed tab
    await page.getByRole('button', { name: /Completed/ }).click()

    // Should show empty state
    await expect(page.getByText('No recently completed issues')).toBeVisible()
  })

  test('should not show completed section above issues in tree view', async ({
    page,
  }) => {
    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          mockAttentionWithIssues([makeCompletedItem('orch-go-001', 'Completed issue')]),
        ),
      })
    })

    await page.goto('/work-graph')
    await page.waitForTimeout(2000)

    // The old recently-completed-section should NOT appear in the issues view
    const section = page.locator('[data-testid="recently-completed-section"]')
    await expect(section).not.toBeVisible()
  })

  test('should support keyboard navigation (j/k) in completed view', async ({ page }) => {
    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          mockAttentionWithIssues([
            makeCompletedItem('orch-go-001', 'First issue'),
            makeCompletedItem('orch-go-002', 'Second issue'),
          ]),
        ),
      })
    })

    await page.goto('/work-graph')
    await page.waitForTimeout(2000)

    // Switch to completed view
    await page.getByRole('button', { name: /Completed/ }).click()
    await page.waitForTimeout(500)

    // Focus the completed view container
    await page.locator('.completed-view').focus()

    // First item should be selected
    const firstRow = page.locator('[data-testid="completed-row-orch-go-001"]')
    await expect(firstRow).toHaveClass(/selected/)

    // Press j to move down
    await page.keyboard.press('j')
    const secondRow = page.locator('[data-testid="completed-row-orch-go-002"]')
    await expect(secondRow).toHaveClass(/selected/)

    // Press k to move back up
    await page.keyboard.press('k')
    await expect(firstRow).toHaveClass(/selected/)
  })

  test('should expand details with Enter key', async ({ page }) => {
    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          mockAttentionWithIssues([
            makeCompletedItem('orch-go-001', 'Issue with details'),
          ]),
        ),
      })
    })

    await page.goto('/work-graph')
    await page.waitForTimeout(2000)

    // Switch to completed view
    await page.getByRole('button', { name: /Completed/ }).click()
    await page.waitForTimeout(500)

    await page.locator('.completed-view').focus()

    // Details should not be visible initially
    const details = page.locator('.expanded-details')
    await expect(details).not.toBeVisible()

    // Press Enter to expand
    await page.keyboard.press('Enter')
    await expect(details).toBeVisible()

    // Press Enter again to collapse
    await page.keyboard.press('Enter')
    await expect(details).not.toBeVisible()
  })

  test('should cycle views with Tab key: issues -> completed -> artifacts', async ({
    page,
  }) => {
    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(
          mockAttentionWithIssues([makeCompletedItem('orch-go-001', 'Some issue')]),
        ),
      })
    })

    await page.goto('/work-graph')
    await page.waitForTimeout(2000)

    // Should start on issues view
    const issuesBtn = page.getByRole('button', { name: 'Issues' })
    await expect(issuesBtn).toHaveClass(/bg-accent/)

    // Press Tab -> should go to completed
    await page.keyboard.press('Tab')
    const completedBtn = page.getByRole('button', { name: /Completed/ })
    await expect(completedBtn).toHaveClass(/bg-accent/)

    // Press Tab -> should go to artifacts
    await page.keyboard.press('Tab')
    const artifactsBtn = page.getByRole('button', { name: 'Artifacts' })
    await expect(artifactsBtn).toHaveClass(/bg-accent/)

    // Press Tab -> should cycle back to issues
    await page.keyboard.press('Tab')
    await expect(issuesBtn).toHaveClass(/bg-accent/)
  })
})
