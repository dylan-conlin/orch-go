import { writable } from 'svelte/store';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Knowledge health item (synthesis, promote, stale, investigation-promotion)
export interface KBHealthItem {
	[key: string]: unknown; // Flexible structure for different item types
}

// Knowledge health category
export interface KBHealthCategory {
	count: number;
	items: KBHealthItem[];
}

// Knowledge health response from /api/kb-health
export interface KBHealthState {
	synthesis: KBHealthCategory;
	promote: KBHealthCategory;
	stale: KBHealthCategory;
	investigation_promotion: KBHealthCategory;
	total: number;
	last_updated: string;
	error?: string;
}

// Default empty state
const defaultState: KBHealthState = {
	synthesis: { count: 0, items: [] },
	promote: { count: 0, items: [] },
	stale: { count: 0, items: [] },
	investigation_promotion: { count: 0, items: [] },
	total: 0,
	last_updated: ''
};

// KB Health store
function createKBHealthStore() {
	const { subscribe, set } = writable<KBHealthState>(defaultState);

	return {
		subscribe,
		set,
		// Fetch kb health state from orch-go API
		async fetch(projectDir?: string): Promise<void> {
			try {
				const url = projectDir
					? `${API_BASE}/api/kb-health?project=${encodeURIComponent(projectDir)}`
					: `${API_BASE}/api/kb-health`;
				const response = await fetch(url);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch kb health:', error);
				set({
					...defaultState,
					error: error instanceof Error ? error.message : 'Unknown error'
				});
			}
		}
	};
}

export const kbHealth = createKBHealthStore();
