import { writable, type Readable } from 'svelte/store'
import {
  createSSEConnection,
  type SSEConnection,
  type ConnectionStatus,
} from '../services/sse-connection'

// API configuration
const API_BASE = 'http://localhost:3348'

// Tree node type from pkg/tree/types.go
export type NodeType =
  | 'investigation'
  | 'decision'
  | 'model'
  | 'probe'
  | 'guide'
  | 'issue'
  | 'cluster'
  | 'postmortem'
  | 'handoff'

export type NodeStatus = 'complete' | 'triage' | 'in_progress' | 'closed' | 'open'

// Knowledge node from /api/tree
export interface KnowledgeNode {
  ID: string
  Type: NodeType
  Title: string
  Path: string
  Status: NodeStatus
  Date?: string // ISO timestamp
  Children: KnowledgeNode[]
  Metadata?: Record<string, any>
}

// Tree response
export interface TreeResponse {
  tree: KnowledgeNode | null
  error?: string
}

// View mode (only knowledge view exposed in UI, work view still available via CLI)
export type TreeView = 'knowledge'

// Sort modes matching backend SortMode enum
export type SortMode = 'recency' | 'connectivity' | 'alphabetical'

// Animation states for nodes
export type AnimationState = 'pulsing' | 'fading' | 'growing' | 'static'

export interface NodeAnimation {
  state: AnimationState
  parentId?: string // For growing nodes, which parent they split from
  startTime: number // Timestamp when animation started
}

// Create SSE connection for tree updates
let treeSSE: SSEConnection | null = null

// Build a structural fingerprint of the tree to detect actual content changes.
// Ignores fields that change without meaningful structural impact (Date, Metadata, Path).
// Children are sorted by ID before fingerprinting so reordering doesn't trigger false updates.
function treeFingerprint(node: KnowledgeNode | null): string {
  if (!node) return ''
  const parts = [node.ID, node.Type, node.Title, node.Status || '']
  if (node.Children?.length) {
    const childFps = node.Children.map((c) => treeFingerprint(c))
    childFps.sort()
    parts.push(childFps.join(','))
  }
  return parts.join('|')
}

// Detect transformations between old and new trees
function detectTransformations(
  oldTree: KnowledgeNode | null,
  newTree: KnowledgeNode | null,
): Map<string, NodeAnimation> {
  const animations = new Map<string, NodeAnimation>()
  if (!oldTree || !newTree) return animations

  // Build maps of old and new nodes for easy lookup
  const oldNodes = new Map<string, KnowledgeNode>()
  const newNodes = new Map<string, KnowledgeNode>()

  const indexNodes = (node: KnowledgeNode, map: Map<string, KnowledgeNode>) => {
    map.set(node.ID, node)
    node.Children?.forEach((child) => indexNodes(child, map))
  }

  indexNodes(oldTree, oldNodes)
  indexNodes(newTree, newNodes)

  // Find transformations: issue nodes that were active and are now closed with new artifact children
  oldNodes.forEach((oldNode, id) => {
    const newNode = newNodes.get(id)

    // Check if this is an issue node that completed
    if (oldNode.Type === 'issue' && newNode) {
      const wasActive = oldNode.Status === 'in_progress' || oldNode.Status === 'open'
      const nowClosed = newNode.Status === 'closed' || newNode.Status === 'complete'

      if (wasActive && nowClosed) {
        // Find new children that are knowledge artifacts
        const oldChildIds = new Set(oldNode.Children?.map((c) => c.ID) || [])
        const newArtifactChildren =
          newNode.Children?.filter(
            (child) =>
              !oldChildIds.has(child.ID) &&
              ['investigation', 'decision', 'model', 'guide'].includes(child.Type),
          ) || []

        if (newArtifactChildren.length > 0) {
          // Mark parent for fading
          animations.set(id, {
            state: 'fading',
            startTime: Date.now(),
          })

          // Mark new children for growing
          newArtifactChildren.forEach((child) => {
            animations.set(child.ID, {
              state: 'growing',
              parentId: id,
              startTime: Date.now(),
            })
          })
        }
      }
    }
  })

  // Find nodes that are actively running (for pulsing animation)
  newNodes.forEach((node, id) => {
    if (node.Type === 'issue' && node.Status === 'in_progress' && !animations.has(id)) {
      animations.set(id, {
        state: 'pulsing',
        startTime: Date.now(),
      })
    }
  })

  return animations
}

function createKnowledgeTreeStore() {
  const { subscribe, set, update } = writable<TreeResponse>({ tree: null })

  // Track last tree fingerprint to skip duplicate updates
  let lastFingerprint = ''

  // Track previous tree for transformation detection
  let previousTree: KnowledgeNode | null = null

  // Animation states for nodes
  const animationStates = writable<Map<string, NodeAnimation>>(new Map())

  return {
    subscribe,

    // Fetch tree from API
    async fetch(
      view: TreeView = 'knowledge',
      sortMode: SortMode = 'recency',
    ): Promise<void> {
      try {
        const url = `${API_BASE}/api/tree?view=${view}&sort=${sortMode}`
        const response = await fetch(url)

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`)
        }

        const tree = await response.json()
        lastFingerprint = treeFingerprint(tree)

        // Detect transformations and update animation states
        const newAnimations = detectTransformations(previousTree, tree)
        animationStates.set(newAnimations)
        previousTree = tree

        set({ tree })
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Unknown error'
        set({ tree: null, error: message })
      }
    },

    // Connect to SSE stream for live updates
    connectSSE(view: TreeView = 'knowledge', sortMode: SortMode = 'recency'): void {
      if (treeSSE) {
        treeSSE.disconnect()
      }

      const url = `${API_BASE}/api/events/tree?view=${view}&sort=${sortMode}`
      treeSSE = createSSEConnection(url, {
        eventListeners: {
          'tree-update': (event) => {
            try {
              const tree = JSON.parse(event.data)

              // Skip update if tree content hasn't changed
              const fp = treeFingerprint(tree)
              if (fp === lastFingerprint) return
              lastFingerprint = fp

              // Detect transformations and update animation states
              const newAnimations = detectTransformations(previousTree, tree)
              animationStates.set(newAnimations)
              previousTree = tree

              set({ tree })
            } catch (error) {
              console.error('Failed to parse tree update:', error)
            }
          },
        },
        // No onDisconnect handler - disconnects should not hide the tree.
        // Connection status is tracked via treeSSE.status store.
      })

      treeSSE.connect()
    },

    // Disconnect SSE
    disconnectSSE(): void {
      if (treeSSE) {
        treeSSE.disconnect()
        treeSSE = null
      }
    },

    // Get SSE connection status store (reactive)
    getSSEStatus(): Readable<ConnectionStatus> | null {
      return treeSSE?.status ?? null
    },

    // Get animation states store (reactive)
    getAnimationStates() {
      return animationStates
    },

    // Clear animation for a specific node
    clearAnimation(nodeId: string): void {
      animationStates.update((states) => {
        const newStates = new Map(states)
        newStates.delete(nodeId)
        return newStates
      })
    },
  }
}

export const knowledgeTree = createKnowledgeTreeStore()
