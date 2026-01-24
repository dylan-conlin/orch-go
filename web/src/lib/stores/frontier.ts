import { writable } from 'svelte/store';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Issue in the frontier
export interface FrontierIssue {
	id: string;
	title: string;
	issue_type: string;
	priority: number;
}

// Blocked issue with leverage info
export interface BlockedIssue {
	id: string;
	title: string;
	issue_type: string;
	priority: number;
	would_unblock?: string[];
	total_leverage: number;
}

// Active agent
export interface ActiveAgent {
	beads_id: string;
	title?: string;
	runtime: string;
	skill?: string;
}

// Frontier response from /api/frontier
export interface FrontierState {
	warnings?: string[];
	ready: FrontierIssue[];
	ready_total: number;
	blocked: BlockedIssue[];
	blocked_total: number;
	active: ActiveAgent[];
	active_total: number;
	stuck: ActiveAgent[];
	stuck_total: number;
	error?: string;
}

// Default empty state
const defaultState: FrontierState = {
	ready: [],
	ready_total: 0,
	blocked: [],
	blocked_total: 0,
	active: [],
	active_total: 0,
	stuck: [],
	stuck_total: 0
};

// Frontier store
function createFrontierStore() {
	const { subscribe, set } = writable<FrontierState>(defaultState);

	return {
		subscribe,
		set,
		// Fetch frontier state from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/frontier`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch frontier state:', error);
				set({
					...defaultState,
					error: error instanceof Error ? error.message : 'Unknown error'
				});
			}
		}
	};
}

export const frontier = createFrontierStore();

// Helper to format leverage info
export function formatLeverage(blocked: BlockedIssue): string {
	if (blocked.total_leverage === 0) {
		return '';
	}

	if (!blocked.would_unblock || blocked.would_unblock.length === 0) {
		return `unblocks ${blocked.total_leverage} (transitive)`;
	}

	if (blocked.would_unblock.length === 1) {
		return `unblocks: ${blocked.would_unblock[0]}`;
	}

	if (blocked.would_unblock.length <= 3) {
		return `unblocks: ${blocked.would_unblock.join(', ')}`;
	}

	return `unblocks: ${blocked.would_unblock.slice(0, 2).join(', ')}... (+${blocked.would_unblock.length - 2} more)`;
}
