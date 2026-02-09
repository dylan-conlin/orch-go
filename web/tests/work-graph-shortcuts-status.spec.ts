import { test, expect } from '@playwright/test'

// Verification keyboard shortcuts (orch-go-21213) - now in Completed view tab
test.describe('Verification Keyboard Shortcuts', () => {
  test('should mark unverified issue as verified with v key in completed view', async ({
    page,
  }) => {
    // Track API calls
    let verifyApiCalled = false
    let verifyRequestBody: any = null

    // Mock the verify API
    await page.route('**/api/attention/verify', async (route) => {
      verifyApiCalled = true
      verifyRequestBody = JSON.parse(route.request().postData() || '{}')
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          issue_id: verifyRequestBody.issue_id,
          status: verifyRequestBody.status,
          verified_at: new Date().toISOString(),
        }),
      })
    })

    // Mock the attention API with a recently-closed (unverified) issue
    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: [
            {
              id: 'beads-recently-closed-orch-go-test-123',
              source: 'beads-recently-closed',
              concern: 'Verification',
              signal: 'recently-closed',
              subject: 'orch-go-test-123',
              summary: 'Closed 2h ago: Test completed issue',
              priority: 50,
              role: 'human',
              collected_at: new Date().toISOString(),
              metadata: {
                closed_at: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(),
                status: 'closed',
                issue_type: 'task',
                beads_priority: 2,
              },
            },
          ],
          total: 1,
          sources: ['beads-recently-closed'],
          role: 'human',
          collected_at: new Date().toISOString(),
        }),
      })
    })

    // Mock other required endpoints
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ nodes: [], edges: [], node_count: 0, edge_count: 0 }),
      })
    })

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

    await page.route('**/api/daemon**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ running: false, paused: false, queue_length: 0 }),
      })
    })

    await page.goto('/work-graph')
    await page.waitForTimeout(1000)

    // Switch to Completed view
    await page.getByRole('button', { name: /Completed/ }).click()
    await page.waitForTimeout(500)

    // Wait for the unverified issue to appear
    await expect(
      page.locator('[data-testid="completed-row-orch-go-test-123"]'),
    ).toBeVisible({ timeout: 5000 })

    // Ensure container has focus
    await page.locator('.completed-view').focus()

    // Press v to verify the issue
    await page.keyboard.press('v')

    // Wait for API call
    await page.waitForTimeout(200)

    // Verify the API was called with correct parameters
    expect(verifyApiCalled).toBe(true)
    expect(verifyRequestBody.issue_id).toBe('orch-go-test-123')
    expect(verifyRequestBody.status).toBe('verified')
  })

  test('should mark unverified issue as needs_fix with x key in completed view', async ({
    page,
  }) => {
    // Track API calls
    let verifyApiCalled = false
    let verifyRequestBody: any = null

    // Mock the verify API
    await page.route('**/api/attention/verify', async (route) => {
      verifyApiCalled = true
      verifyRequestBody = JSON.parse(route.request().postData() || '{}')
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          issue_id: verifyRequestBody.issue_id,
          status: verifyRequestBody.status,
          verified_at: new Date().toISOString(),
        }),
      })
    })

    // Mock the attention API with a recently-closed (unverified) issue
    await page.route('**/api/attention**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: [
            {
              id: 'beads-recently-closed-orch-go-test-456',
              source: 'beads-recently-closed',
              concern: 'Verification',
              signal: 'recently-closed',
              subject: 'orch-go-test-456',
              summary: 'Closed 1h ago: Another test issue',
              priority: 50,
              role: 'human',
              collected_at: new Date().toISOString(),
              metadata: {
                closed_at: new Date(Date.now() - 1 * 60 * 60 * 1000).toISOString(),
                status: 'closed',
                issue_type: 'bug',
                beads_priority: 1,
              },
            },
          ],
          total: 1,
          sources: ['beads-recently-closed'],
          role: 'human',
          collected_at: new Date().toISOString(),
        }),
      })
    })

    // Mock other required endpoints
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ nodes: [], edges: [], node_count: 0, edge_count: 0 }),
      })
    })

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

    await page.route('**/api/daemon**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ running: false, paused: false, queue_length: 0 }),
      })
    })

    await page.goto('/work-graph')
    await page.waitForTimeout(1000)

    // Switch to Completed view
    await page.getByRole('button', { name: /Completed/ }).click()
    await page.waitForTimeout(500)

    // Wait for the unverified issue to appear
    await expect(
      page.locator('[data-testid="completed-row-orch-go-test-456"]'),
    ).toBeVisible({ timeout: 5000 })

    // Ensure container has focus
    await page.locator('.completed-view').focus()

    // Press x to mark as needs_fix
    await page.keyboard.press('x')

    // Wait for API call
    await page.waitForTimeout(200)

    // Verify the API was called with correct parameters
    expect(verifyApiCalled).toBe(true)
    expect(verifyRequestBody.issue_id).toBe('orch-go-test-456')
    expect(verifyRequestBody.status).toBe('needs_fix')
  })
})

// Keyboard shortcuts for reprioritize and queue toggle (orch-go-fuqss)
test.describe('Reprioritize and Queue Toggle Shortcuts', () => {
  test('should enter priority mode with p key and set priority with 0-4', async ({ page }) => {
    // Track API calls
    let updateApiCalled = false
    let updateRequestBody: any = null

    // Mock the update API
    await page.route('**/api/beads/update', async (route) => {
      updateApiCalled = true
      updateRequestBody = JSON.parse(route.request().postData() || '{}')
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: updateRequestBody.id,
          success: true,
        }),
      })
    })

    // Mock graph API to return updated data
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

    // Mock other required APIs
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

    await page.goto('/work-graph')

    // Wait for tree to render
    await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toBeVisible()

    // Ensure container has focus
    await page.locator('.work-graph-tree').focus()

    // Priority mode indicator should not be visible
    await expect(page.getByText('Priority Mode: Press 0-4')).not.toBeVisible()

    // Press p to enter priority mode
    await page.keyboard.press('p')

    // Priority mode indicator should appear
    await expect(page.getByText('Priority Mode: Press 0-4')).toBeVisible()

    // Press 1 to set priority to P1
    await page.keyboard.press('1')

    // Wait for API call
    await page.waitForTimeout(200)

    // Priority mode indicator should disappear
    await expect(page.getByText('Priority Mode: Press 0-4')).not.toBeVisible()

    // Verify the API was called with correct parameters
    expect(updateApiCalled).toBe(true)
    expect(updateRequestBody.id).toBe('orch-go-1')
    expect(updateRequestBody.priority).toBe(1)
  })

  test('should toggle triage:ready label with q key', async ({ page }) => {
    // Track API calls
    let updateApiCalled = false
    let updateRequestBody: any = null

    // Mock the update API
    await page.route('**/api/beads/update', async (route) => {
      updateApiCalled = true
      updateRequestBody = JSON.parse(route.request().postData() || '{}')
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          id: updateRequestBody.id,
          success: true,
        }),
      })
    })

    // Mock graph API
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
              labels: [],
            },
          ],
          edges: [],
          node_count: 1,
          edge_count: 0,
        }),
      })
    })

    // Mock other required APIs
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

    await page.goto('/work-graph')

    // Wait for tree to render
    await expect(page.locator('[data-testid="issue-row-orch-go-1"]')).toBeVisible()

    // Ensure container has focus
    await page.locator('.work-graph-tree').focus()

    // Press q to toggle triage:ready label
    await page.keyboard.press('q')

    // Wait for API call
    await page.waitForTimeout(200)

    // Verify the API was called with correct parameters
    expect(updateApiCalled).toBe(true)
    expect(updateRequestBody.id).toBe('orch-go-1')
    expect(updateRequestBody.add_labels).toEqual(['triage:ready'])
  })

  test('should NOT work on WIP items', async ({ page }) => {
    // Track API calls
    let updateApiCalled = false

    // Mock the update API
    await page.route('**/api/beads/update', async (route) => {
      updateApiCalled = true
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ id: 'should-not-be-called', success: true }),
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

    // Mock graph API
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [],
          edges: [],
          node_count: 0,
          edge_count: 0,
        }),
      })
    })

    // Mock other required APIs
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
        body: JSON.stringify({
          running: true,
          capacity_max: 3,
          capacity_used: 1,
          capacity_free: 2,
        }),
      })
    })

    await page.route('**/api/daemon/status**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ running: true, paused: false, queue_length: 0 }),
      })
    })

    await page.goto('/work-graph')

    // Wait for work-graph-tree container
    await expect(page.locator('.work-graph-tree')).toBeVisible()

    // Wait for WIP item to render
    await expect(page.locator('[data-node-index="0"]')).toBeVisible({ timeout: 10000 })

    // Ensure container has focus
    await page.locator('.work-graph-tree').focus()

    // WIP item should be selected (first item)
    await expect(page.getByText('Running Task')).toBeVisible()

    // Try pressing p (should not work on WIP item)
    await page.keyboard.press('p')
    await page.waitForTimeout(100)

    // Priority mode indicator should NOT appear
    await expect(page.getByText('Priority Mode: Press 0-4')).not.toBeVisible()

    // Try pressing q (should not work on WIP item)
    await page.keyboard.press('q')
    await page.waitForTimeout(200)

    // Update API should NOT have been called
    expect(updateApiCalled).toBe(false)
  })
})

// Status View tests (orch-go-21209)
test.describe('Status View', () => {
  test('should switch to status view and display status groups', async ({ page }) => {
    // Mock the graph API response (matching pattern from working tests)
    await page.route('**/api/beads/graph**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          nodes: [
            {
              id: 'orch-go-1',
              title: 'Ready Issue',
              type: 'task',
              status: 'open',
              priority: 1,
              source: 'beads',
            },
            {
              id: 'orch-go-2',
              title: 'In Progress Issue',
              type: 'task',
              status: 'in_progress',
              priority: 2,
              source: 'beads',
            },
            {
              id: 'orch-go-3',
              title: 'Blocked Issue',
              type: 'task',
              status: 'blocked',
              priority: 1,
              source: 'beads',
            },
            {
              id: 'orch-go-4',
              title: 'Done Issue',
              type: 'task',
              status: 'closed',
              priority: 2,
              source: 'beads',
            },
          ],
          edges: [],
          node_count: 4,
          edge_count: 0,
        }),
      })
    })

    await page.goto('/work-graph')

    // Wait for tree view to load (default view shows issue titles)
    await expect(page.getByText('Ready Issue')).toBeVisible()

    // Click on Status view button
    await page.getByRole('button', { name: 'Status' }).click()

    // Wait for status view container to appear
    await expect(page.locator('.work-graph-status')).toBeVisible()

    // Should show status group headers (use first() to handle multiple matches)
    await expect(page.getByText('Ready', { exact: true }).first()).toBeVisible()
    await expect(page.getByText('In Progress').first()).toBeVisible()
    await expect(page.getByText('Blocked').first()).toBeVisible()
    await expect(page.getByText('Done').first()).toBeVisible()

    // Verify issues are still visible under their groups
    await expect(page.getByText('Ready Issue')).toBeVisible()
    await expect(page.getByText('In Progress Issue')).toBeVisible()
    await expect(page.getByText('Blocked Issue')).toBeVisible()
    await expect(page.getByText('Done Issue')).toBeVisible()
  })
})
