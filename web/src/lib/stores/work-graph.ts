import { writable } from 'svelte/store'
import { shallowEqual } from '$lib/utils/shallow-equal'

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348'

// Graph node from /api/beads/graph
export interface GraphNode {
  id: string
  title: string
  type: string // beads: task, bug, feature, epic, question; kb: investigation, decision
  status: string // open, in_progress, closed, blocked, Complete, Accepted, etc.
  priority: number // 0-4 for beads, 0 for kb artifacts
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

  // Sort children by ID (maintains creation order)
  for (const node of treeNodes.values()) {
    node.children.sort((a, b) => a.id.localeCompare(b.id))
  }

  // Sort roots by priority, then by ID
  roots.sort((a, b) => {
    if (a.priority !== b.priority) {
      return a.priority - b.priority
    }
    return a.id.localeCompare(b.id)
  })

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
// Note: This function does NOT handle 'dep-chain' mode - caller should use clusterByDepChain instead
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

// Cluster tree nodes by dependency chains (connected components)
// Groups nodes that are connected via blocks or parent-child edges
export function clusterByDepChain(nodes: TreeNode[], edges: GraphEdge[]): GroupSection[] {
  // Build adjacency list from edges (undirected - connection in either direction means same cluster)
  const adjacency = new Map<string, Set<string>>()
  const nodeMap = new Map<string, TreeNode>()
  
  // Initialize adjacency list and node map
  for (const node of nodes) {
    adjacency.set(node.id, new Set())
    nodeMap.set(node.id, node)
  }
  
  // Add edges for blocks and parent-child relationships
  for (const edge of edges) {
    // Only consider edges where both nodes are in our current node set
    if (!nodeMap.has(edge.from) || !nodeMap.has(edge.to)) continue
    
    if (edge.type === 'blocks' || edge.type === '' || edge.type === 'parent-child') {
      adjacency.get(edge.from)?.add(edge.to)
      adjacency.get(edge.to)?.add(edge.from)
    }
  }
  
  // Find connected components using BFS
  const visited = new Set<string>()
  const clusters: TreeNode[][] = []
  
  for (const node of nodes) {
    if (visited.has(node.id)) continue
    
    // BFS to find all nodes in this connected component
    const component: TreeNode[] = []
    const queue: string[] = [node.id]
    visited.add(node.id)
    
    while (queue.length > 0) {
      const currentId = queue.shift()!
      const currentNode = nodeMap.get(currentId)
      if (currentNode) {
        component.push(currentNode)
      }
      
      // Add neighbors to queue
      const neighbors = adjacency.get(currentId) || new Set()
      for (const neighborId of neighbors) {
        if (!visited.has(neighborId)) {
          visited.add(neighborId)
          queue.push(neighborId)
        }
      }
    }
    
    clusters.push(component)
  }
  
  // Build GroupSections
  const sections: GroupSection[] = []
  
  for (const cluster of clusters) {
    if (cluster.length === 0) continue
    
    // Find root node (node with no blockers/parents in this cluster, or alphabetically first)
    let rootNode = cluster[0]
    for (const node of cluster) {
      // A node is a root if it has no blocked_by relationships within this cluster
      const clusterIds = new Set(cluster.map(n => n.id))
      const hasBlockersInCluster = node.blocked_by.some(blockerId => clusterIds.has(blockerId))
      
      if (!hasBlockersInCluster) {
        // This is a potential root - choose alphabetically first among roots
        if (node.id < rootNode.id || rootNode.blocked_by.some(bid => clusterIds.has(bid))) {
          rootNode = node
        }
      }
    }
    
    // Sort cluster by topological order (dependencies first)
    // Simple approach: sort by number of dependencies within cluster, then alphabetically
    const clusterIds = new Set(cluster.map(n => n.id))
    const sortedCluster = cluster.sort((a, b) => {
      const aDepCount = a.blocked_by.filter(bid => clusterIds.has(bid)).length
      const bDepCount = b.blocked_by.filter(bid => clusterIds.has(bid)).length
      
      // Fewer dependencies first (leaves before roots in dependency tree)
      if (aDepCount !== bDepCount) {
        return aDepCount - bDepCount
      }
      
      // Then by priority
      if (a.priority !== b.priority) {
        return a.priority - b.priority
      }
      
      // Finally alphabetically
      return a.id.localeCompare(b.id)
    })
    
    sections.push({
      label: cluster.length === 1 ? 'Independent' : rootNode.title || rootNode.id,
      key: `dep-chain-${rootNode.id}`,
      nodes: sortedCluster,
      unlabeled: cluster.length === 1, // Single nodes are treated like "unlabeled"
    })
  }
  
  // Sort sections: multi-node clusters first (by size, desc), then independent nodes
  sections.sort((a, b) => {
    const aSize = a.nodes.length
    const bSize = b.nodes.length
    
    // Single-node sections go last
    if (aSize === 1 && bSize > 1) return 1
    if (bSize === 1 && aSize > 1) return -1
    
    // Multi-node sections sorted by size (largest first)
    if (aSize !== bSize) return bSize - aSize
    
    // Same size - sort by label
    return a.label.localeCompare(b.label)
  })
  
  // Merge all single-node clusters into one "Independent" section
  const independentNodes: TreeNode[] = []
  const clusteredSections: GroupSection[] = []
  
  for (const section of sections) {
    if (section.nodes.length === 1) {
      independentNodes.push(...section.nodes)
    } else {
      clusteredSections.push(section)
    }
  }
  
  // Add Independent section at the end if there are any
  if (independentNodes.length > 0) {
    clusteredSections.push({
      label: 'Independent',
      key: 'dep-chain-independent',
      nodes: independentNodes,
      unlabeled: true,
    })
  }
  
  return clusteredSections
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
