import { writable, type Readable } from 'svelte/store';
import { createSSEConnection, type SSEConnection, type ConnectionStatus } from '../services/sse-connection';

// API configuration
const API_BASE = 'https://localhost:3348';

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
  | 'handoff';

export type NodeStatus =
  | 'complete'
  | 'triage'
  | 'in_progress'
  | 'closed'
  | 'open';

// Knowledge node from /api/tree
export interface KnowledgeNode {
  ID: string;
  Type: NodeType;
  Title: string;
  Path: string;
  Status: NodeStatus;
  Date?: string; // ISO timestamp
  Children: KnowledgeNode[];
  Metadata?: Record<string, any>;
  // UI state
  expanded?: boolean;
}

// Tree response
export interface TreeResponse {
  tree: KnowledgeNode | null;
  error?: string;
}

// View mode
export type TreeView = 'knowledge' | 'work';

// Create SSE connection for tree updates
let treeSSE: SSEConnection | null = null;

// Build a structural fingerprint of the tree (ignoring UI state like expanded)
// to detect when content actually changed vs just a re-send
function treeFingerprint(node: KnowledgeNode | null): string {
  if (!node) return '';
  const parts = [node.ID, node.Type, node.Title, node.Status || ''];
  if (node.Children?.length) {
    parts.push(node.Children.map(c => treeFingerprint(c)).join(','));
  }
  return parts.join('|');
}

function createKnowledgeTreeStore() {
  const { subscribe, set, update } = writable<TreeResponse>({ tree: null });

  // Track last tree fingerprint to skip duplicate updates
  let lastFingerprint = '';

  // Apply expansion state to tree nodes
  const applyExpansionState = (node: KnowledgeNode | null, expandedIds: Set<string>): void => {
    if (!node) return;

    // Set expanded state based on whether this node's ID is in the expandedIds set
    node.expanded = expandedIds.has(node.ID);

    // Recursively apply to children
    if (node.Children) {
      for (const child of node.Children) {
        applyExpansionState(child, expandedIds);
      }
    }
  };

  return {
    subscribe,

    // Fetch tree from API
    async fetch(view: TreeView = 'knowledge', expandedIds?: Set<string>): Promise<void> {
      try {
        const url = `${API_BASE}/api/tree?view=${view}`;
        const response = await fetch(url);

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const tree = await response.json();

        // Apply expansion state if provided
        if (expandedIds) {
          applyExpansionState(tree, expandedIds);
        }

        lastFingerprint = treeFingerprint(tree);
        set({ tree });
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Unknown error';
        set({ tree: null, error: message });
      }
    },

    // Connect to SSE stream for live updates
    connectSSE(view: TreeView = 'knowledge', expandedIds?: Set<string>): void {
      if (treeSSE) {
        treeSSE.disconnect();
      }

      const url = `${API_BASE}/api/events/tree?view=${view}`;
      treeSSE = createSSEConnection(url, {
        eventListeners: {
          'tree-update': (event) => {
            try {
              const tree = JSON.parse(event.data);

              // Skip update if tree content hasn't changed
              const fp = treeFingerprint(tree);
              if (fp === lastFingerprint) return;
              lastFingerprint = fp;

              // Apply expansion state to preserve UI state across SSE updates
              if (expandedIds) {
                applyExpansionState(tree, expandedIds);
              }

              set({ tree });
            } catch (error) {
              console.error('Failed to parse tree update:', error);
            }
          }
        }
        // No onDisconnect handler - disconnects should not hide the tree.
        // Connection status is tracked via treeSSE.status store.
      });

      treeSSE.connect();
    },

    // Disconnect SSE
    disconnectSSE(): void {
      if (treeSSE) {
        treeSSE.disconnect();
        treeSSE = null;
      }
    },

    // Get SSE connection status store (reactive)
    getSSEStatus(): Readable<ConnectionStatus> | null {
      return treeSSE?.status ?? null;
    },

    // Toggle node expansion
    toggleNode(nodeId: string): void {
      update(state => {
        if (!state.tree) return state;

        const toggleInTree = (node: KnowledgeNode): boolean => {
          if (node.ID === nodeId) {
            node.expanded = !node.expanded;
            return true;
          }
          for (const child of node.Children || []) {
            if (toggleInTree(child)) return true;
          }
          return false;
        };

        toggleInTree(state.tree);
        return { ...state };
      });
    }
  };
}

export const knowledgeTree = createKnowledgeTreeStore();
