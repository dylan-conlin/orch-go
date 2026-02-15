import { writable } from 'svelte/store';
import { createSSEConnection, type SSEConnection } from '../services/sse-connection';

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

function createKnowledgeTreeStore() {
  const { subscribe, set, update } = writable<TreeResponse>({ tree: null });

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
              set({ tree });
            } catch (error) {
              console.error('Failed to parse tree update:', error);
            }
          }
        },
        onDisconnect: () => {
          update(state => ({ ...state, error: 'Disconnected from tree updates' }));
        }
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
