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

// Build a structural fingerprint of the tree to detect actual content changes.
// Ignores fields that change without meaningful structural impact (Date, Metadata, Path).
// Children are sorted by ID before fingerprinting so reordering doesn't trigger false updates.
function treeFingerprint(node: KnowledgeNode | null): string {
  if (!node) return '';
  const parts = [node.ID, node.Type, node.Title, node.Status || ''];
  if (node.Children?.length) {
    const childFps = node.Children.map(c => treeFingerprint(c));
    childFps.sort();
    parts.push(childFps.join(','));
  }
  return parts.join('|');
}

function createKnowledgeTreeStore() {
  const { subscribe, set, update } = writable<TreeResponse>({ tree: null });

  // Track last tree fingerprint to skip duplicate updates
  let lastFingerprint = '';

  return {
    subscribe,

    // Fetch tree from API
    async fetch(view: TreeView = 'knowledge'): Promise<void> {
      try {
        const url = `${API_BASE}/api/tree?view=${view}`;
        const response = await fetch(url);

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const tree = await response.json();
        lastFingerprint = treeFingerprint(tree);
        set({ tree });
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Unknown error';
        set({ tree: null, error: message });
      }
    },

    // Connect to SSE stream for live updates
    connectSSE(view: TreeView = 'knowledge'): void {
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
    }
  };
}

export const knowledgeTree = createKnowledgeTreeStore();
