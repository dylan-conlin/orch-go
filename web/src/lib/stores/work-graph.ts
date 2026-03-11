import { writable } from 'svelte/store'
import { shallowEqual } from '$lib/utils/shallow-equal'

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'http://localhost:3348'

// Graph node from /api/beads/graph
export interface GraphNode {
  id: string
  title: string
  type: string // beads: task, bug, feature, epic, question; kb: investigation, decision
  status: string // open, in_progress, closed, blocked, Complete, Accepted, etc.
  priority: number // 0-4 for beads, 0 for kb artifacts
  effective_priority?: number
  source: string // "beads" or "kb"
  date?: string // for kb artifacts
  created_at?: string // creation timestamp
  description?: string // issue description
  labels?: string[] // issue labels (area:*, effort:*, triage:*, etc.)
  layer?: number // Execution phase (0 = ready, N = blocked by layers 0..N-1)
  active_agent?: {
    phase?: string
    runtime?: string
    model?: string
  }
}

// Graph edge (dependency) from /api/beads/graph
export interface GraphEdge {
  from: string // ID of the issue that has the dependency
  to: string // ID of the issue being depended on
  type: string // dependency_type: blocks, parent-child, relates_to
}

// Work graph response from /api/beads/graph
export interface WorkGraphResponse {
  nodes: GraphNode[]
  edges: GraphEdge[]
  node_count: number
  edge_count: number
  project_dir?: string
  error?: string
}

// Attention badge type (matches attention store)
export type AttentionBadgeType =
  | 'verify' // Phase: Complete, needs orch complete
  | 'decide' // Investigation has recommendation needing decision
  | 'escalate' // Question needs human judgment
  | 'likely_done' // Commits suggest completion
  | 'recently_closed' // Recently closed, needs verification
  | 'unblocked' // Blocker just closed, now actionable
  | 'stuck' // Agent stuck >2h
  | 'crashed'
  | 'verify_failed' // Verification failed during auto-completion

// Tree node with hierarchy and expansion state
export interface TreeNode extends GraphNode {
  children: TreeNode[]
  depth: number
  expanded: boolean // Children expanded in tree
  details_expanded: boolean // L1 details expanded
  blocked_by: string[]
  blocks: string[]
  absorbed_by?: string // ID of the issue that absorbed this one (supersedes)
  parent_id?: string
  // Attention signal (if any)
  attentionBadge?: AttentionBadgeType
  attentionReason?: string
}

// Work graph store with AbortController support to prevent race conditions
function createWorkGraphStore() {
  const { subscribe, set, update } = writable<WorkGraphResponse | null>(null)

  // Track in-flight requests to cancel stale ones
  let currentAbortController: AbortController | null = null
  let fetchSequence = 0 // Sequence guard for additional safety
  let currentData: WorkGraphResponse | null = null // Track current data for shallow equality

  return {
    subscribe,
    set,
    update,

    // Cancel any pending fetch - useful when project context changes
    cancelPending(): void {
      if (currentAbortController) {
        currentAbortController.abort()
        currentAbortController = null
      }
    },

    // Fetch work graph from orch-go API
    // projectDir: Optional project directory to query (for following orchestrator context)
    // scope: "focus" (default) or "open" (all open issues)
    async fetch(
      projectDir?: string,
      scope: string = 'open',
      parent?: string,
    ): Promise<void> {
      // Cancel any pending request before starting new one
      if (currentAbortController) {
        currentAbortController.abort()
      }

      // Create new abort controller for this request
      const abortController = new AbortController()
      currentAbortController = abortController

      // Increment sequence for this fetch
      const thisSequence = ++fetchSequence

      try {
        const params = new URLSearchParams()
        if (projectDir) {
          params.set('project_dir', projectDir)
        }
        params.set('scope', scope)
        if (parent) {
          params.set('parent', parent)
        }
        const url = `${API_BASE}/api/beads/graph${params.toString() ? '?' + params.toString() : ''}`
        const response = await fetch(url, { signal: abortController.signal })

        // Sequence guard: ignore response if newer fetch started
        if (thisSequence !== fetchSequence) {
          return // Stale response, discard
        }

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`)
        }
        const data = await response.json()

        // Final sequence check before setting state
        if (thisSequence === fetchSequence) {
          // Only update if data actually changed (reduces reactive cascades)
          if (!shallowEqual(currentData, data)) {
            currentData = data
            set(data)
          }
        }
      } catch (error) {
        // Ignore abort errors - they're intentional
        if (error instanceof Error && error.name === 'AbortError') {
          return
        }

        // Only set error if this is still the current request
        if (thisSequence === fetchSequence) {
          console.error('Failed to fetch work graph:', error)
          const errorData = {
            nodes: [],
            edges: [],
            node_count: 0,
            edge_count: 0,
            error: String(error),
          }
          currentData = errorData
          set(errorData)
        }
      } finally {
        // Clear controller if this was the current one
        if (currentAbortController === abortController) {
          currentAbortController = null
        }
      }
    },
  }
}

// Parse hierarchy from beads IDs
// orch-go-X.1 is child of orch-go-X
// orch-go-X.1.2 is child of orch-go-X.1
export function parseParentId(id: string): string | undefined {
  const parts = id.split('.')
  if (parts.length <= 1) {
    return undefined // No parent (top-level)
  }
  // Remove last part to get parent ID
  return parts.slice(0, -1).join('.')
}

// Build tree structure from flat nodes
export function buildTree(nodes: GraphNode[], edges: GraphEdge[]): TreeNode[] {
  // Create tree nodes with initial state
  const treeNodes: Map<string, TreeNode> = new Map()

  for (const node of nodes) {
    const parentId = parseParentId(node.id)
    treeNodes.set(node.id, {
      ...node,
      children: [],
      depth: 0,
      expanded: true, // Children expanded by default for Phase 1
      details_expanded: false, // L1 details collapsed by default
      blocked_by: [],
      blocks: [],
      parent_id: parentId,
    })
  }

  // Build blocking relationships from edges
  for (const edge of edges) {
    if (edge.type === 'blocks') {
      const fromNode = treeNodes.get(edge.from)
      const toNode = treeNodes.get(edge.to)
      if (fromNode && toNode) {
        // edge.from blocks edge.to
        toNode.blocked_by.push(edge.from)
        fromNode.blocks.push(edge.to)
      }
    }
  }

  // Build supersedes (absorbed-by) relationships from edges
  for (const edge of edges) {
    if (edge.type === 'supersedes') {
      const absorbedNode = treeNodes.get(edge.from)
      if (absorbedNode) {
        // edge.from is absorbed by edge.to
        absorbedNode.absorbed_by = edge.to
      }
    }
  }
  // Apply parent-child edges from API (set via 'bd update --parent')
  // These override ID-pattern hierarchy when explicit parent-child edges exist
  for (const edge of edges) {
    if (edge.type === '' || edge.type === 'parent-child') {
      const childNode = treeNodes.get(edge.from)
      const parentNode = treeNodes.get(edge.to)
      if (childNode && parentNode) {
        // edge.from is child, edge.to is parent
        childNode.parent_id = edge.to
      }
    }
  }

  // Build parent-child hierarchy
  const roots: TreeNode[] = []

  for (const node of treeNodes.values()) {
    if (node.parent_id) {
      const parent = treeNodes.get(node.parent_id)
      if (parent) {
        parent.children.push(node)
        node.depth = parent.depth + 1
      } else {
        // Parent doesn't exist in dataset, treat as root
        roots.push(node)
      }
    } else {
      // No parent, it's a root
      roots.push(node)
    }
  }

  const getEffectivePriority = (node: TreeNode): number =>
    node.effective_priority ?? node.priority
  const getLayer = (node: TreeNode): number => node.layer ?? Number.POSITIVE_INFINITY
  const compareTreeOrder = (a: TreeNode, b: TreeNode): number => {
    const priorityDiff = getEffectivePriority(a) - getEffectivePriority(b)
    if (priorityDiff !== 0) return priorityDiff
    const layerDiff = getLayer(a) - getLayer(b)
    if (layerDiff !== 0) return layerDiff
    return a.id.localeCompare(b.id)
  }

  // Sort children by effective priority, then topological layer
  for (const node of treeNodes.values()) {
    node.children.sort(compareTreeOrder)
  }

  // Sort roots by effective priority, then topological layer
  roots.sort(compareTreeOrder)

  return roots
}

// Grouping mode for Work Graph
export type GroupByMode = 'priority' | 'area' | 'effort' | 'dep-chain'

// A group section for rendering
export interface GroupSection {
  label: string // Display label (e.g., "area:dashboard", "P1", "unlabeled")
  key: string // Unique key for the group
  nodes: TreeNode[]
  unlabeled: boolean // Whether this is the catch-all unlabeled group
}

// Group tree nodes by the selected mode
// Priority groups by priority level (P0, P1, P2, P3, P4)
// Area/Effort groups by matching label prefix, with unlabeled at bottom
export function groupTreeNodes(nodes: TreeNode[], mode: GroupByMode): GroupSection[] {
  if (mode === 'priority') {
    const groups = new Map<number, TreeNode[]>()
    for (const node of nodes) {
      const p = node.priority
      if (!groups.has(p)) groups.set(p, [])
      groups.get(p)!.push(node)
    }
    // Sort by priority number
    return Array.from(groups.entries())
      .sort(([a], [b]) => a - b)
      .map(([p, items]) => ({
        label: `P${p}`,
        key: `priority-${p}`,
        nodes: items,
        unlabeled: false,
      }))
  }

  // Label-based grouping (area: or effort:)
  const prefix = mode + ':'
  const groups = new Map<string, TreeNode[]>()
  const unlabeled: TreeNode[] = []

  for (const node of nodes) {
    const match = (node.labels ?? []).find((l) => l.startsWith(prefix))
    if (match) {
      if (!groups.has(match)) groups.set(match, [])
      groups.get(match)!.push(node)
    } else {
      unlabeled.push(node)
    }
  }

  // Sort groups alphabetically by label
  const sections: GroupSection[] = Array.from(groups.entries())
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([label, items]) => ({
      label,
      key: `label-${label}`,
      nodes: items,
      unlabeled: false,
    }))

  // Unlabeled section at bottom
  if (unlabeled.length > 0) {
    sections.push({
      label: 'unlabeled',
      key: 'unlabeled',
      nodes: unlabeled,
      unlabeled: true,
    })
  }

  return sections
}

// Filter tree nodes by label text. Keeps nodes (and their ancestors) where any label
// contains the filter text (case-insensitive). When a child matches, its parent is kept
// to preserve tree structure.
export function filterTreeByLabel(nodes: TreeNode[], filter: string): TreeNode[] {
  if (!filter) return nodes
  const lower = filter.toLowerCase()

  function matches(node: TreeNode): boolean {
    return (node.labels ?? []).some((l) => l.toLowerCase().includes(lower))
  }

  function filterNodes(nodes: TreeNode[]): TreeNode[] {
    const result: TreeNode[] = []
    for (const node of nodes) {
      const filtered = filterNodes(node.children)
      if (matches(node) || filtered.length > 0) {
        result.push({ ...node, children: filtered })
      }
    }
    return result
  }

  return filterNodes(nodes)
}

// === Dependency Chain Visualization ===

export interface DepNode {
  node: TreeNode
  depChildren: DepNode[]
  depDepth: number
}

export interface DepChain {
  id: string
  label: string
  roots: DepNode[]
  size: number
}

export interface FlatDepItem {
  node: TreeNode
  prefix: string
  depDepth: number
  isGate?: boolean
}

export interface DepView {
  chains: DepChain[]
  independentNodes: TreeNode[]
}

/**
 * Build dependency chains from blocking edges.
 * Groups connected issues into chains via BFS, finds roots (no blockers),
 * and builds trees following blocker→blocked direction.
 */
export function buildDependencyView(
  treeNodeIndex: Map<string, TreeNode>,
  edges: GraphEdge[],
): DepView {
  const parseCreatedAt = (value?: string): number => {
    if (!value) return 0
    const ms = new Date(value).getTime()
    return Number.isNaN(ms) ? 0 : ms
  }

  const compareNodes = (a: TreeNode, b: TreeNode): number => {
    if (a.priority !== b.priority) {
      return a.priority - b.priority
    }
    const dateDiff = parseCreatedAt(b.created_at) - parseCreatedAt(a.created_at)
    if (dateDiff !== 0) return dateDiff
    return a.id.localeCompare(b.id)
  }

  const blocksMap = new Map<string, string[]>()
  const blockedByMap = new Map<string, string[]>()
  const involvedNodes = new Set<string>()

  for (const edge of edges) {
    if (edge.type !== 'blocks') continue
    if (!treeNodeIndex.has(edge.from) || !treeNodeIndex.has(edge.to)) continue

    involvedNodes.add(edge.from)
    involvedNodes.add(edge.to)

    // edges are directional: edge.from depends on edge.to
    // build maps so blockers (upstream) point to dependents (downstream)
    if (!blocksMap.has(edge.to)) blocksMap.set(edge.to, [])
    blocksMap.get(edge.to)!.push(edge.from)
    if (!blockedByMap.has(edge.from)) blockedByMap.set(edge.from, [])
    blockedByMap.get(edge.from)!.push(edge.to)
  }

  if (involvedNodes.size === 0) {
    return {
      chains: [],
      independentNodes: Array.from(treeNodeIndex.values()).sort(compareNodes),
    }
  }

  // Find connected components using BFS
  const visited = new Set<string>()
  const components: Set<string>[] = []

  for (const nodeId of involvedNodes) {
    if (visited.has(nodeId)) continue
    const component = new Set<string>()
    const queue = [nodeId]
    while (queue.length > 0) {
      const current = queue.shift()!
      if (visited.has(current)) continue
      visited.add(current)
      component.add(current)

      for (const blocked of blocksMap.get(current) || []) {
        if (!visited.has(blocked)) queue.push(blocked)
      }
      for (const blocker of blockedByMap.get(current) || []) {
        if (!visited.has(blocker)) queue.push(blocker)
      }
    }
    if (component.size > 0) components.push(component)
  }

  // Build dependency trees for each component
  const chains: DepChain[] = []

  for (const component of components) {
    const roots: string[] = []
    for (const id of component) {
      const blockers = (blockedByMap.get(id) || []).filter((b) => component.has(b))
      if (blockers.length === 0) roots.push(id)
    }
    if (roots.length === 0) roots.push([...component][0])

    const buildDepNode = (nodeId: string, depth: number, seen: Set<string>): DepNode => {
      seen.add(nodeId)
      const blocked = (blocksMap.get(nodeId) || [])
        .filter((b) => component.has(b) && !seen.has(b))
        .sort((a, b) => {
          const nodeA = treeNodeIndex.get(a)
          const nodeB = treeNodeIndex.get(b)
          if (nodeA && nodeB) {
            // Topological: sort by layer first (upstream items before downstream)
            const layerA = nodeA.layer ?? Number.POSITIVE_INFINITY
            const layerB = nodeB.layer ?? Number.POSITIVE_INFINITY
            if (layerA !== layerB) return layerA - layerB
            return compareNodes(nodeA, nodeB)
          }
          return a.localeCompare(b)
        })
      return {
        node: treeNodeIndex.get(nodeId)!,
        depChildren: blocked.map((childId) => buildDepNode(childId, depth + 1, seen)),
        depDepth: depth,
      }
    }

    const seen = new Set<string>()
    const rootNodes: DepNode[] = []
    for (const rootId of roots.sort()) {
      if (seen.has(rootId)) continue
      rootNodes.push(buildDepNode(rootId, 0, seen))
    }

    const labelRoot = roots
      .map((id) => treeNodeIndex.get(id))
      .filter((node): node is TreeNode => Boolean(node))
      .sort((a, b) => {
        if (a.depth !== b.depth) return a.depth - b.depth
        return compareNodes(a, b)
      })[0]
    chains.push({
      id: `chain-${roots[0]}`,
      label: labelRoot?.title || roots[0],
      roots: rootNodes,
      size: component.size,
    })
  }

  chains.sort((a, b) => {
    if (a.size !== b.size) return b.size - a.size
    return a.id.localeCompare(b.id)
  })

  const independentNodes: TreeNode[] = []
  for (const [, node] of treeNodeIndex) {
    if (!involvedNodes.has(node.id)) {
      independentNodes.push(node)
    }
  }
  independentNodes.sort(compareNodes)

  return { chains, independentNodes }
}

/**
 * Flatten a dependency chain into items with flow connector prefixes.
 * Root nodes are flush left (◆ marker added by renderer), children get ├─▸ or └─▸ prefixes.
 * Top-to-bottom = dependency flow direction (upstream first, downstream last).
 * Gate items (single leaf at max depth) are marked for separator rendering.
 */
export function flattenDepChain(chain: DepChain, pinnedIds: Set<string>): FlatDepItem[] {
  const items: FlatDepItem[] = []
  const leafDepths: { nodeId: string; depth: number }[] = []

  function collect(depNode: DepNode) {
    if (pinnedIds.has(depNode.node.id)) return

    const visibleChildren = depNode.depChildren.filter((c) => !pinnedIds.has(c.node.id))
    if (visibleChildren.length === 0) {
      leafDepths.push({ nodeId: depNode.node.id, depth: depNode.depDepth })
      return
    }

    for (const child of visibleChildren) {
      collect(child)
    }
  }

  for (const root of chain.roots) {
    collect(root)
  }

  // Detect gate items: single leaf at maximum depth = convergence point
  // Gate = everything above must complete before this item closes
  const maxLeafDepth = leafDepths.reduce((max, l) => Math.max(max, l.depth), 0)
  let gateId: string | undefined
  if (maxLeafDepth > 0) {
    const leavesAtMax = leafDepths.filter((l) => l.depth === maxLeafDepth)
    if (leavesAtMax.length === 1) {
      gateId = leavesAtMax[0].nodeId
    }
  }

  const gatePathIds = new Set<string>()
  if (gateId) {
    const markGatePath = (depNode: DepNode): boolean => {
      if (pinnedIds.has(depNode.node.id)) return false
      const visibleChildren = depNode.depChildren.filter((c) => !pinnedIds.has(c.node.id))
      let onPath = depNode.node.id === gateId
      for (const child of visibleChildren) {
        if (markGatePath(child)) {
          onPath = true
        }
      }
      if (onPath) {
        gatePathIds.add(depNode.node.id)
      }
      return onPath
    }

    for (const root of chain.roots) {
      markGatePath(root)
    }
  }

  function walk(depNode: DepNode, ancestorIsLast: boolean[]) {
    if (pinnedIds.has(depNode.node.id)) return

    let prefix = ''
    if (depNode.depDepth > 0) {
      for (let i = 0; i < ancestorIsLast.length - 1; i++) {
        prefix += ancestorIsLast[i] ? '    ' : '│   '
      }
      const isLast = ancestorIsLast[ancestorIsLast.length - 1]
      // Flow connectors: directional arrow shows blocking flows downstream
      prefix += isLast ? '└─▸ ' : '├─▸ '
    }

    const visibleChildren = depNode.depChildren.filter((c) => !pinnedIds.has(c.node.id))
    let orderedChildren = visibleChildren

    if (gatePathIds.size > 0 && orderedChildren.length > 1) {
      const gateChildIndex = orderedChildren.findIndex((child) => gatePathIds.has(child.node.id))
      if (gateChildIndex !== -1 && gateChildIndex !== orderedChildren.length - 1) {
        orderedChildren = [...orderedChildren]
        const [gateChild] = orderedChildren.splice(gateChildIndex, 1)
        orderedChildren.push(gateChild)
      }
    }

    items.push({ node: depNode.node, prefix, depDepth: depNode.depDepth })

    for (let i = 0; i < orderedChildren.length; i++) {
      const childIsLast = i === orderedChildren.length - 1
      walk(orderedChildren[i], [...ancestorIsLast, childIsLast])
    }
  }

  for (const root of chain.roots) {
    walk(root, [])
  }

  if (gateId) {
    const gateItem = items.find((item) => item.node.id === gateId)
    if (gateItem) {
      gateItem.isGate = true
    }
  }

  return items
}

export const workGraph = createWorkGraphStore()

// Close issue request/response types
interface CloseIssueRequest {
  id: string
  reason?: string
  project_dir?: string
}

interface CloseIssueResponse {
  id: string
  success: boolean
  error?: string
}

/**
 * Close a beads issue via the API.
 * @param id - The issue ID to close
 * @param reason - Optional reason for closing
 * @param projectDir - Optional project directory
 * @returns Promise with the result
 */
export async function closeIssue(
  id: string,
  reason?: string,
  projectDir?: string,
): Promise<{ success: boolean; error?: string }> {
  try {
    const request: CloseIssueRequest = {
      id,
      reason,
      project_dir: projectDir,
    }

    const response = await fetch(`${API_BASE}/api/beads/close`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    })

    if (!response.ok) {
      const text = await response.text()
      return { success: false, error: `HTTP ${response.status}: ${text}` }
    }

    const data: CloseIssueResponse = await response.json()

    if (!data.success) {
      return { success: false, error: data.error || 'Unknown error' }
    }

    // Trigger a refresh of the work graph
    workGraph.fetch(projectDir, 'open').catch(console.error)

    return { success: true }
  } catch (error) {
    return { success: false, error: String(error) }
  }
}

// Update issue request/response types
interface UpdateIssueRequest {
  id: string
  priority?: number
  add_labels?: string[]
  remove_labels?: string[]
  project_dir?: string
}

interface UpdateIssueResponse {
  id: string
  success: boolean
  error?: string
}

/**
 * Update a beads issue via the API.
 * @param id - The issue ID to update
 * @param options - Update options (priority, add_labels, remove_labels, project_dir)
 * @returns Promise with the result
 */
export async function updateIssue(
  id: string,
  options: {
    priority?: number
    add_labels?: string[]
    remove_labels?: string[]
    project_dir?: string
  },
): Promise<{ success: boolean; error?: string }> {
  try {
    const request: UpdateIssueRequest = {
      id,
      ...options,
    }

    const response = await fetch(`${API_BASE}/api/beads/update`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    })

    if (!response.ok) {
      const text = await response.text()
      return { success: false, error: `HTTP ${response.status}: ${text}` }
    }

    const data: UpdateIssueResponse = await response.json()

    if (!data.success) {
      return { success: false, error: data.error || 'Unknown error' }
    }

    // Trigger a refresh of the work graph
    workGraph.fetch(options.project_dir, 'open').catch(console.error)

    return { success: true }
  } catch (error) {
    return { success: false, error: String(error) }
  }
}
