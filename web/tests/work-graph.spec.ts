import { test, expect } from '@playwright/test'

test.describe('Work Graph Page', () => {
  test('should render work graph route', async ({ page }) => {
    await page.goto('/work-graph')

    // Should show page title
    await expect(page.getByRole('heading', { name: 'Work Graph' })).toBeVisible()
  })

  test('should fetch and display graph data', async ({ page }) => {
    // Mock the graph API response
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-1',
              title: 'Parent Epic',
              type: 'epic',
              status: 'in_progress',
              priority: 1,
              source: 'beads',
            },
            {
              id: 'orch-go-1.1',
              title: 'Child Task',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
          ],
          edges: [
            {
              from: 'orch-go-1.1', // child
              to: 'orch-go-1', // parent
              type: 'parent-child',
            },
          ],
          node_count: 2,
          edge_count: 1,
        }),
      })
    })

    await page.goto('/work-graph')

    // Should display the parent node
    await expect(page.getByText('Parent Epic')).toBeVisible()

    // Should display the child node
    await expect(page.getByText('Child Task')).toBeVisible()
  })
})

test.describe('Work Graph Tree Structure', () => {
  test('should display L0 view with status, priority, id, title, age, type', async ({
    page,
  }) => {
    const now = new Date().toISOString()

    await page.route('**/api/focus**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ has_focus: false }),
      })
    })

    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: [], total: 0, sources: [], role: 'human' }),
      })
    })

    await page.route('**/api/agents**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      })
    })

    await page.route('**/api/beads/ready**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ issues: [] }),
      })
    })

    await page.route('**/api/daemon/status**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ running: false, paused: false, queue_length: 0 }),
      })
    })

    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-100',
              title: 'Test Issue',
              type: 'task',
              status: 'in_progress',
              priority: 1,
              source: 'beads',
              created_at: now,
            },
          ],
          edges: [],
          node_count: 1,
          edge_count: 0,
        }),
      })
    })

    await page.goto('/work-graph')

    const issueRow = page.locator('[data-testid="issue-row-orch-go-100"]')

    // Should show status icon
    await expect(issueRow.locator('[data-testid="status-icon"]')).toBeVisible()

    // Should show priority
    await expect(issueRow.locator('[data-testid="priority-badge"]')).toBeVisible()

    // Should show ID
    await expect(issueRow.getByText('orch-go-100')).toBeVisible()

    // Should show title
    await expect(issueRow.getByText('Test Issue')).toBeVisible()

    // Should show type badge
    await expect(issueRow.locator('[data-testid="type-badge"]')).toBeVisible()
  })

  test('should display status details in L1 expansion', async ({ page }) => {
    await page.route('**/api/focus**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ has_focus: false }),
      })
    })

    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: [], total: 0, sources: [], role: 'human' }),
      })
    })

    await page.route('**/api/agents**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      })
    })

    await page.route('**/api/beads/ready**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ issues: [] }),
      })
    })

    await page.route('**/api/daemon/status**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ running: false, paused: false, queue_length: 0 }),
      })
    })

    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-101',
              title: 'Issue With Details',
              type: 'bug',
              status: 'in_progress',
              priority: 1,
              source: 'beads',
              description: 'Reproduce and fix issue in work graph',
            },
          ],
          edges: [],
          node_count: 1,
          edge_count: 0,
        }),
      })
    })

    await page.goto('/work-graph')
    await expect(page.locator('[data-testid="issue-row-orch-go-101"]')).toBeVisible()
    await page.locator('[data-testid="issue-row-orch-go-101"]').click()
    await page.locator('.work-graph-tree').focus()

    // Expand L1 details
    await page.keyboard.press('Enter')

    const details = page.locator('[data-testid="issue-details-orch-go-101"]')
    await expect(details).toBeVisible()
    await expect(details.getByText('Issue summary:')).toBeVisible()
    await expect(details.getByText('Reproduce and fix issue in work graph')).toBeVisible()
    await expect(details.getByText('Dependency context:')).toBeVisible()
    await expect(details.getByText('Status:')).toBeVisible()
    await expect(details.getByText('in progress')).toBeVisible()
    await expect(details.getByText('Priority:')).toBeVisible()
    await expect(details.getByText('P1')).toBeVisible()
  })

  test('should show comprehension aids in L1 expansion', async ({ page }) => {
    await page.route('**/api/focus**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ has_focus: false }),
      })
    })

    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: [], total: 0, sources: [], role: 'human' }),
      })
    })

    await page.route('**/api/agents**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      })
    })

    await page.route('**/api/beads/ready**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ issues: [] }),
      })
    })

    await page.route('**/api/daemon/status**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ running: false, paused: false, queue_length: 0 }),
      })
    })

    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-300',
              title: 'Comprehension Epic',
              type: 'epic',
              status: 'open',
              priority: 1,
              source: 'beads',
              description:
                'Deliver comprehension improvements for operators. Include summaries, dependency context, and progress signals.',
            },
            {
              id: 'orch-go-300.1',
              title: 'Completed child',
              type: 'task',
              status: 'closed',
              priority: 2,
              source: 'beads',
            },
            {
              id: 'orch-go-300.2',
              title: 'Open child',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
            {
              id: 'orch-go-301',
              title: 'Upstream blocker',
              type: 'task',
              status: 'open',
              priority: 0,
              source: 'beads',
            },
          ],
          edges: [
            {
              from: 'orch-go-300.1',
              to: 'orch-go-300',
              type: 'parent-child',
            },
            {
              from: 'orch-go-300.2',
              to: 'orch-go-300',
              type: 'parent-child',
            },
            {
              from: 'orch-go-301',
              to: 'orch-go-300',
              type: 'blocks',
            },
          ],
          node_count: 4,
          edge_count: 3,
        }),
      })
    })

    await page.goto('/work-graph')
    await expect(page.locator('[data-testid="issue-row-orch-go-300"]')).toBeVisible()
    await page.locator('[data-testid="issue-row-orch-go-300"]').click()
    await page.locator('.work-graph-tree').focus()

    // Expand L1 details
    await page.keyboard.press('Enter')

    const details = page.locator('[data-testid="issue-details-orch-go-300"]')
    await expect(details).toBeVisible()
    await expect(details.getByText('Issue summary:')).toBeVisible()
    await expect(details.getByText('Dependency context:')).toBeVisible()
    await expect(details.getByText('Blocked by 1 upstream issue.')).toBeVisible()
    await expect(details.getByText('Progress & completeness:')).toBeVisible()
    await expect(details.getByText('1/2 done (50%)')).toBeVisible()
    await expect(details.getByText('Related issues:')).toBeVisible()
    await expect(details.getByText('Children (2):')).toBeVisible()
  })
})

test.describe('Work Graph Keyboard Navigation', () => {
  test('should support j/k for up/down navigation', async ({ page }) => {
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-1',
              title: 'First Issue',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
            {
              id: 'orch-go-2',
              title: 'Second Issue',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
          ],
          edges: [],
          node_count: 2,
          edge_count: 0,
        }),
      })
    })

    await page.goto('/work-graph')

    // Ensure container has focus
    await page.locator('.work-graph-tree').focus()

    // First item should be selected initially
    await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(
      /selected/,
    )

    // Press j to move down
    await page.keyboard.press('j')
    await expect(page.locator('[data-testid="issue-row-orch-go-2"]')).toHaveClass(
      /selected/,
    )

    // Press k to move up
    await page.keyboard.press('k')
    await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(
      /selected/,
    )
  })

  test('should support Enter to expand L1 details', async ({ page }) => {
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-1',
              title: 'Test Issue',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
              description: 'Test description',
            },
          ],
          edges: [],
          node_count: 1,
          edge_count: 0,
        }),
      })
    })

    await page.goto('/work-graph')

    // Ensure container has focus
    await page.locator('.work-graph-tree').focus()

    // Expanded details should not be visible initially
    await expect(page.getByText('Test description')).not.toBeVisible()

    // Press Enter to expand L1 details
    await page.keyboard.press('Enter')

    // L1 details should now be visible
    await expect(page.getByText('Test description')).toBeVisible()
  })

  test('should support Escape to collapse L1 details', async ({ page }) => {
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-1',
              title: 'Test Issue',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
              description: 'Test description',
            },
          ],
          edges: [],
          node_count: 1,
          edge_count: 0,
        }),
      })
    })

    await page.goto('/work-graph')

    // Ensure container has focus
    await page.locator('.work-graph-tree').focus()

    // Expand first with Enter
    await page.keyboard.press('Enter')
    await expect(page.getByText('Test description')).toBeVisible()

    // Press Escape to collapse L1 details
    await page.keyboard.press('Escape')
    await expect(page.getByText('Test description')).not.toBeVisible()
  })

  test('should support g/G for top/bottom navigation', async ({ page }) => {
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-1',
              title: 'First',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
            {
              id: 'orch-go-2',
              title: 'Second',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
            {
              id: 'orch-go-3',
              title: 'Third',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
          ],
          edges: [],
          node_count: 3,
          edge_count: 0,
        }),
      })
    })

    await page.goto('/work-graph')

    // Ensure container has focus
    await page.locator('.work-graph-tree').focus()

    // Press Shift+G to go to bottom
    await page.keyboard.press('Shift+G')
    await expect(page.locator('[data-testid="issue-row-orch-go-3"]')).toHaveClass(
      /selected/,
    )

    // Press g twice to go to top
    await page.keyboard.press('g')
    await page.keyboard.press('g')
    await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(
      /selected/,
    )
  })
})

// Bug fixes for Phase 1.1
test.describe('Bug Fixes - Phase 1.1', () => {
  // Bug 1: orch-go-21144 - Highlight makes text unreadable
  test('should use background highlight for selection', async ({ page }) => {
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-1',
              title: 'Test Issue',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
          ],
          edges: [],
          node_count: 1,
          edge_count: 0,
        }),
      })
    })

    await page.goto('/work-graph')

    // Wait for the row to be visible
    const issueRow = page.locator('[data-testid="issue-row-orch-go-1"]')
    await expect(issueRow).toBeVisible()

    const rowContent = issueRow.locator('> div').first()

    // Selected item should have bg-accent class (background highlight)
    await expect(rowContent).toHaveClass(/bg-accent/)

    // Should NOT have border-primary (no border)
    await expect(rowContent).not.toHaveClass(/border-primary/)
  })

  // Bug 2: orch-go-21145 - Border and highlight out of sync
  test('should unify selection state between click and keyboard navigation', async ({
    page,
  }) => {
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-1',
              title: 'First Issue',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
            {
              id: 'orch-go-2',
              title: 'Second Issue',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
          ],
          edges: [],
          node_count: 2,
          edge_count: 0,
        }),
      })
    })

    await page.goto('/work-graph')

    // Click on second item
    await page.locator('[data-testid="issue-row-orch-go-2"]').click()

    // Should have selected class (unified state)
    await expect(page.locator('[data-testid="issue-row-orch-go-2"]')).toHaveClass(
      /selected/,
    )

    // Should have bg-accent styling (no border)
    const row2Content = page.locator('[data-testid="issue-row-orch-go-2"] > div').first()
    await expect(row2Content).toHaveClass(/bg-accent/)
    await expect(row2Content).not.toHaveClass(/border-primary/)

    // Navigate with keyboard
    await page.keyboard.press('k')

    // First item should now have selected class
    await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toHaveClass(
      /selected/,
    )

    // Should have same bg-accent styling
    const row1Content = page.locator('[data-testid="issue-row-orch-go-1"] > div').first()
    await expect(row1Content).toHaveClass(/bg-accent/)
    await expect(row1Content).not.toHaveClass(/border-primary/)

    // Second item should no longer have selected class
    await expect(page.locator('[data-testid="issue-row-orch-go-2"]')).not.toHaveClass(
      /selected/,
    )
  })

  // Bug 3: orch-go-21146 - Can't collapse epics with children
  test('should collapse/expand tree nodes with h/l keys', async ({ page }) => {
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-1',
              title: 'Parent Epic',
              type: 'epic',
              status: 'in_progress',
              priority: 1,
              source: 'beads',
            },
            {
              id: 'orch-go-1.1',
              title: 'Child Task 1',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
            {
              id: 'orch-go-1.2',
              title: 'Child Task 2',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
          ],
          edges: [],
          node_count: 3,
          edge_count: 0,
        }),
      })
    })

    await page.goto('/work-graph')

    // Wait for tree to render
    await expect(page.getByText('Parent Epic')).toBeVisible()

    // Children should be visible initially (expanded by default)
    await expect(page.locator('[data-testid="issue-row-orch-go-1.1"]')).toBeVisible()
    await expect(page.locator('[data-testid="issue-row-orch-go-1.2"]')).toBeVisible()

    // Click on parent epic to select it
    await page.locator('[data-testid="issue-row-orch-go-1"]').click()
    await page.waitForTimeout(100)

    // Ensure container has focus before pressing h
    await page.locator('.work-graph-tree').focus()

    // Collapse with h key (while parent is selected)
    await page.keyboard.press('h')

    // Wait for DOM to update
    await page.waitForTimeout(500)

    // Children should now be hidden
    await expect(page.locator('[data-testid="issue-row-orch-go-1.1"]')).not.toBeVisible()
    await expect(page.locator('[data-testid="issue-row-orch-go-1.2"]')).not.toBeVisible()

    // Parent should still be visible
    await expect(page.getByText('Parent Epic')).toBeVisible()

    // Click on parent to ensure it's selected
    await page.locator('[data-testid="issue-row-orch-go-1"]').click()
    await page.waitForTimeout(100)

    // Ensure focus is still on container
    await page.locator('.work-graph-tree').focus()

    // Expand with l key
    await page.keyboard.press('l')

    // Children should be visible again
    await expect(page.locator('[data-testid="issue-row-orch-go-1.1"]')).toBeVisible()
    await expect(page.locator('[data-testid="issue-row-orch-go-1.2"]')).toBeVisible()
  })

  // Bug 4: orch-go-21150 - Selection highlight barely visible (regression fix)
  test('should have clearly visible selection with background highlight', async ({
    page,
  }) => {
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-1',
              title: 'Test Issue',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
          ],
          edges: [],
          node_count: 1,
          edge_count: 0,
        }),
      })
    })

    await page.goto('/work-graph')

    // Wait for the row to be visible
    const issueRow = page.locator('[data-testid="issue-row-orch-go-1"]')
    await expect(issueRow).toBeVisible()

    const rowContent = issueRow.locator('> div').first()

    // Should have bg-accent for clear visibility (no border)
    await expect(rowContent).toHaveClass(/bg-accent/)
    await expect(rowContent).not.toHaveClass(/border-primary/)
  })

  test('should hide queued issues from tree when pinned in WIP', async ({ page }) => {
    await page.route('**/api/focus**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ has_focus: false }),
      })
    })

    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ items: [], total: 0, sources: [], role: 'human' }),
      })
    })

    await page.route('**/api/agents**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ agents: [], count: 0 }),
      })
    })

    // Mock the beads/ready API (queued issues)
    await page.route('**/api/beads/ready**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          issues: [
            {
              id: 'orch-go-21164',
              title: 'Queued Issue',
              priority: 0,
              issue_type: 'task',
              created_at: '2026-02-02T10:00:00Z',
            },
          ],
        }),
      })
    })

    // Mock the beads/graph API (includes same queued issue as tree node)
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-21164',
              title: 'Queued Issue',
              type: 'task',
              status: 'open',
              priority: 0,
              source: 'beads',
            },
            {
              id: 'orch-go-100',
              title: 'Regular Issue',
              type: 'task',
              status: 'open',
              priority: 1,
              source: 'beads',
            },
          ],
          edges: [],
          node_count: 2,
          edge_count: 0,
        }),
      })
    })

    // Mock agents API (no running agents)
    await page.route('**/api/agents**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          agents: [],
          count: 0,
        }),
      })
    })

    // Mock daemon API (WIP section needs queue context)
    await page.route('**/api/daemon**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          running: true,
          capacity_max: 3,
          capacity_used: 1,
          capacity_free: 2,
        }),
      })
    })

    await page.goto('/work-graph')

    // Wait for data to load
    await page.waitForTimeout(1500)

    // Non-pinned issue stays in the tree
    await expect(page.locator('[data-testid="issue-row-orch-go-100"]')).toBeVisible()

    // Queued issue should appear in WIP section
    await expect(page.locator('[data-testid="wip-row-orch-go-21164"]')).toBeVisible()

    // Duplicate tree row should be hidden when issue is pinned in WIP
    await expect(page.locator('[data-testid="issue-row-orch-go-21164"]')).toHaveCount(0)
  })
})
