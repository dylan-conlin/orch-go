import { writable, type Readable } from 'svelte/store';
import { createSSEConnection, type SSEConnection, type ConnectionStatus } from '../services/sse-connection';

// API configuration
const API_BASE = 'http://localhost:3348';

// Action types (from pkg/timeline/types.go)
export type ActionType =
  | 'issue_created'
  | 'issue_completed'
  | 'issue_closed'
  | 'issue_released'
  | 'agent_spawned'
  | 'agent_completed'
  | 'decision_made'
  | 'quick_decision'
  | 'session_started'
  | 'session_ended'
  | 'session_labeled';

// Timeline action
export interface TimelineAction {
  type: ActionType;
  timestamp: string; // ISO timestamp
  session_id: string;
  title: string;
  beads_id?: string;
  path?: string;
  metadata?: Record<string, any>;
  artifact_id?: string; // For split-and-grow animations
}

// Session group
export interface SessionGroup {
  session_id: string;
  label?: string; // Human-readable session name
  start_time: string; // ISO timestamp
  end_time: string; // ISO timestamp
  actions: TimelineAction[];
  action_count: number;
}

// Timeline response
export interface TimelineResponse {
  sessions: SessionGroup[];
  total: number;
}

// Store state
interface TimelineState {
  timeline: TimelineResponse | null;
  error?: string;
}

// Create SSE connection for timeline updates
let timelineSSE: SSEConnection | null = null;

function createTimelineStore() {
  const { subscribe, set, update } = writable<TimelineState>({ timeline: null });

  return {
    subscribe,

    // Fetch timeline from API
    async fetch(sessionID?: string, limit: number = 10): Promise<void> {
      try {
        let url = `${API_BASE}/api/timeline?limit=${limit}`;
        if (sessionID) {
          url += `&session=${sessionID}`;
        }

        const response = await fetch(url);

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const timeline = await response.json();
        set({ timeline });
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Unknown error';
        set({ timeline: null, error: message });
      }
    },

    // Connect to SSE stream for live updates
    connectSSE(sessionID?: string, limit: number = 10): void {
      if (timelineSSE) {
        timelineSSE.disconnect();
      }

      let url = `${API_BASE}/api/events/timeline?limit=${limit}`;
      if (sessionID) {
        url += `&session=${sessionID}`;
      }

      timelineSSE = createSSEConnection(url, {
        eventListeners: {
          'timeline-update': (event) => {
            try {
              const timeline = JSON.parse(event.data);
              set({ timeline });
            } catch (error) {
              console.error('Failed to parse timeline update:', error);
            }
          }
        }
      });

      timelineSSE.connect();
    },

    // Disconnect SSE
    disconnectSSE(): void {
      if (timelineSSE) {
        timelineSSE.disconnect();
        timelineSSE = null;
      }
    },

    // Get SSE connection status store (reactive)
    getSSEStatus(): Readable<ConnectionStatus> | null {
      return timelineSSE?.status ?? null;
    }
  };
}

export const timelineStore = createTimelineStore();
