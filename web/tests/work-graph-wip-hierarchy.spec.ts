import { test, expect } from '@playwright/test'

// WIP Section Integration (Bug: orch-go-21169)
test.describe('WIP Section Integration', () => {
  test('should navigate WIP items with j/k keys before main tree', async ({ page }) => {
    // Mock beads/ready API for queued issues
    await page.route('**/api/beads/ready**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          issues: [
            {
              id: 'orch-go-queued-1',
              title: 'Queued Issue 1',
              priority: 0,
              issue_type: 'bug',
              created_at: '2026-02-02T10:00:00Z',
            },
          ],
        }),
      })
    })

    // Mock agents API for running agents
    await page.route('**/api/agents**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          agents: [
            {
              id: 'agent-1',
              beads_id: 'orch-go-running-1',
              task: 'Running Task 1',
              status: 'active',
              phase: 'Implementation',
              runtime: '5m 30s',
              is_processing: true,
              is_stalled: false,
              spawned_at: '2026-02-02T10:00:00Z',
              updated_at: '2026-02-02T10:05:00Z',
            },
          ],
        }),
      })
    })

    // Mock graph API for main tree
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-tree-1',
              title: 'Tree Issue 1',
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

    // Mock daemon API
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

    // Wait for tree container to render
    await expect(page.locator('.work-graph-tree')).toBeVisible()

    // Wait for WIP items to be added to flattened nodes (check for data-node-index="0")
    await expect(page.locator('[data-node-index="0"]')).toBeVisible({ timeout: 10000 })

    // Ensure container has focus
    await page.locator('.work-graph-tree').focus()

    // First WIP item (running agent) should be focused initially
    let focusedRow = page.locator('[data-node-index="0"].focused')
    await expect(focusedRow).toBeVisible()
    await expect(focusedRow.getByText('Running Task 1')).toBeVisible()

    // Press j to move to second WIP item (queued issue)
    await page.keyboard.press('j')
    focusedRow = page.locator('[data-node-index="1"].focused')
    await expect(focusedRow).toBeVisible()
    await expect(focusedRow.getByText('Queued Issue 1')).toBeVisible()

    // Press j again to move to main tree
    await page.keyboard.press('j')
    focusedRow = page.locator('[data-node-index="2"].focused')
    await expect(focusedRow).toBeVisible()
    await expect(focusedRow.getByText('Tree Issue 1')).toBeVisible()

    // Press k to move back to WIP item
    await page.keyboard.press('k')
    focusedRow = page.locator('[data-node-index="1"].focused')
    await expect(focusedRow).toBeVisible()
    await expect(focusedRow.getByText('Queued Issue 1')).toBeVisible()
  })

  test('should NOT apply greyed-out styling to WIP items', async ({ page }) => {
    // Mock APIs
    await page.route('**/api/beads/ready**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          issues: [
            {
              id: 'orch-go-queued-1',
              title: 'Queued Issue',
              priority: 0,
              issue_type: 'bug',
              created_at: '2026-02-02T10:00:00Z',
            },
          ],
        }),
      })
    })

    await page.route('**/api/agents**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          agents: [
            {
              id: 'agent-1',
              beads_id: 'orch-go-running-1',
              task: 'Running Task',
              status: 'active',
              phase: 'Implementation',
              runtime: '5m',
              is_processing: true,
              is_stalled: false,
              spawned_at: '2026-02-02T10:00:00Z',
              updated_at: '2026-02-02T10:05:00Z',
            },
          ],
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

    // Wait for tree container and WIP items to render
    await expect(page.locator('.work-graph-tree')).toBeVisible()
    await expect(page.locator('[data-node-index="0"]')).toBeVisible({ timeout: 10000 })

    // Check that WIP items do NOT have opacity-60 class
    const runningRow = page.locator('[data-node-index="0"]')
    await expect(runningRow).not.toHaveClass(/opacity-60/)

    const queuedRow = page.locator('[data-node-index="1"]')
    await expect(queuedRow).not.toHaveClass(/opacity-60/)
  })

  test('should toggle L1 details for WIP items with Enter key', async ({ page }) => {
    // Mock APIs
    await page.route('**/api/agents**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          agents: [
            {
              id: 'agent-1',
              beads_id: 'orch-go-running-1',
              task: 'Running Task with Details',
              status: 'active',
              phase: 'Implementation',
              skill: 'feature-impl',
              model: 'anthropic/claude-sonnet-4',
              runtime: '5m',
              is_processing: true,
              is_stalled: false,
              spawned_at: '2026-02-02T10:00:00Z',
              updated_at: '2026-02-02T10:05:00Z',
            },
          ],
        }),
      })
    })

    await page.route('**/api/beads/ready**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ issues: [] }),
      })
    })

    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ nodes: [], edges: [], node_count: 0, edge_count: 0 }),
      })
    })

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

    // Wait for tree container and WIP items to render
    await expect(page.locator('.work-graph-tree')).toBeVisible()
    await expect(page.locator('[data-node-index="0"]')).toBeVisible({ timeout: 10000 })

    // Ensure container has focus
    await page.locator('.work-graph-tree').focus()

    // L1 details should not be visible initially
    const expandedDetails = page.locator('.expanded-details')
    await expect(expandedDetails).not.toBeVisible()

    // Press Enter to expand L1 details
    await page.keyboard.press('Enter')

    // L1 details should now be visible with agent info
    await expect(expandedDetails).toBeVisible()
    await expect(expandedDetails.getByText(/Phase:/)).toBeVisible()
    await expect(expandedDetails.getByText(/Skill:/)).toBeVisible()
  })

  test('should open side panel for running and queued WIP items with i/o keys', async ({ page }) => {
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
        body: JSON.stringify([
          {
            id: 'agent-1',
            beads_id: 'orch-go-running-1',
            task: 'Running Task 1',
            status: 'active',
            phase: 'Implementation',
            runtime: '5m',
            is_processing: true,
            is_stalled: false,
            spawned_at: '2026-02-02T10:00:00Z',
            updated_at: '2026-02-02T10:05:00Z',
          },
        ]),
      })
    })

    await page.route('**/api/beads/ready**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          issues: [
            {
              id: 'orch-go-queued-1',
              title: 'Queued Issue 1',
              priority: 0,
              issue_type: 'task',
              created_at: '2026-02-02T10:00:00Z',
            },
          ],
        }),
      })
    })

    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-running-1',
              title: 'Running Tree Node',
              type: 'task',
              status: 'in_progress',
              priority: 1,
              source: 'beads',
            },
            {
              id: 'orch-go-queued-1',
              title: 'Queued Tree Node',
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

    await page.route('**/api/daemon/status**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ running: true, paused: false, queue_length: 1 }),
      })
    })

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
    await expect(page.locator('[data-testid="wip-row-orch-go-running-1"]')).toBeVisible()
    await expect(page.locator('[data-testid="wip-row-orch-go-queued-1"]')).toBeVisible()

    await page.locator('.work-graph-tree').focus()

    // Running WIP item is selected by default. Press i to open panel.
    await page.keyboard.press('i')
    const panel = page.locator('[role="dialog"]')
    await expect(panel).toBeVisible()
    await expect(panel.getByRole('heading', { name: 'Running Tree Node' })).toBeVisible()

    // Close panel, move to queued WIP item, and open with o.
    await page.keyboard.press('Escape')
    await expect(panel).not.toBeVisible()
    await page.keyboard.press('j')
    await page.keyboard.press('o')

    await expect(panel).toBeVisible()
    await expect(panel.getByRole('heading', { name: 'Queued Tree Node' })).toBeVisible()
    await expect(panel.getByRole('heading', { name: 'Running Tree Node' })).not.toBeVisible()
  })
})

// Parent-child edge support (orch-go-21194)
test.describe('Parent-Child Edge Support', () => {
  test('should nest children under parents using parent-child edges from API', async ({
    page,
  }) => {
    // Mock all required APIs for work-graph page
    await page.route('**/api/beads/ready**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ issues: [] }),
      })
    })

    await page.route('**/api/agents**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ agents: [], count: 0 }),
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
              id: 'orch-go-21193',
              title: 'Parent Epic',
              type: 'epic',
              status: 'in_progress',
              priority: 1,
              source: 'beads',
            },
            {
              id: 'orch-go-21172',
              title: 'Child Task',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
          ],
          edges: [
            {
              from: 'orch-go-21172', // child
              to: 'orch-go-21193', // parent
              type: '', // empty type for parent-child
            },
          ],
          node_count: 2,
          edge_count: 1,
        }),
      })
    })

    await page.goto('/work-graph')

    // Wait for tree to render
    await expect(page.getByText('Parent Epic')).toBeVisible()

    // Child should be visible initially (expanded by default)
    await expect(page.locator('[data-testid="issue-row-orch-go-21172"]')).toBeVisible()

    // Child should be indented (depth > 0)
    const childRow = page.locator('[data-testid="issue-row-orch-go-21172"]')
    await expect(childRow).toHaveAttribute('data-depth', '1')

    // Ensure container has focus
    await page.locator('.work-graph-tree').focus()

    // Collapse parent with h key
    await page.keyboard.press('h')

    // Wait for DOM to update
    await page.waitForTimeout(500)

    // Child should now be hidden
    await expect(
      page.locator('[data-testid="issue-row-orch-go-21172"]'),
    ).not.toBeVisible()

    // Parent should still be visible
    await expect(page.getByText('Parent Epic')).toBeVisible()

    // Expand parent with l key
    await page.keyboard.press('l')

    // Child should be visible again
    await expect(page.locator('[data-testid="issue-row-orch-go-21172"]')).toBeVisible()
  })

  test('should support explicit parent-child type edges', async ({ page }) => {
    // Mock all required APIs
    await page.route('**/api/beads/ready**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ issues: [] }),
      })
    })

    await page.route('**/api/agents**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ agents: [], count: 0 }),
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
              title: 'Parent Issue',
              type: 'feature',
              status: 'open',
              priority: 1,
              source: 'beads',
            },
            {
              id: 'orch-go-200',
              title: 'Child Issue',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
          ],
          edges: [
            {
              from: 'orch-go-200', // child
              to: 'orch-go-100', // parent
              type: 'parent-child', // explicit type
            },
          ],
          node_count: 2,
          edge_count: 1,
        }),
      })
    })

    await page.goto('/work-graph')

    // Wait for tree to render
    await expect(page.getByText('Parent Issue')).toBeVisible()

    // Child should be nested under parent
    const childRow = page.locator('[data-testid="issue-row-orch-go-200"]')
    await expect(childRow).toBeVisible()
    await expect(childRow).toHaveAttribute('data-depth', '1')
  })

  test('should combine ID pattern hierarchy with edge-based hierarchy', async ({
    page,
  }) => {
    // Mock all required APIs
    await page.route('**/api/beads/ready**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ issues: [] }),
      })
    })

    await page.route('**/api/agents**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ agents: [], count: 0 }),
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
              id: 'orch-go-1',
              title: 'Root Epic',
              type: 'epic',
              status: 'open',
              priority: 1,
              source: 'beads',
            },
            {
              id: 'orch-go-1.1',
              title: 'ID Pattern Child',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
            {
              id: 'orch-go-500',
              title: 'Edge-based Child',
              type: 'task',
              status: 'open',
              priority: 2,
              source: 'beads',
            },
          ],
          edges: [
            {
              from: 'orch-go-500', // edge-based child
              to: 'orch-go-1', // parent
              type: '',
            },
          ],
          node_count: 3,
          edge_count: 1,
        }),
      })
    })

    await page.goto('/work-graph')

    // Wait for tree to render
    await expect(page.getByText('Root Epic')).toBeVisible()

    // Both children should be visible and nested at depth 1
    const idPatternChild = page.locator('[data-testid="issue-row-orch-go-1.1"]')
    await expect(idPatternChild).toBeVisible()
    await expect(idPatternChild).toHaveAttribute('data-depth', '1')

    const edgeBasedChild = page.locator('[data-testid="issue-row-orch-go-500"]')
    await expect(edgeBasedChild).toBeVisible()
    await expect(edgeBasedChild).toHaveAttribute('data-depth', '1')
  })
})

