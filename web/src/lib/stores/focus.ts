import { writable } from 'svelte/store';

// API configuration
const API_BASE = 'http://localhost:3348';

// Focus response from /api/focus
export interface FocusInfo {
	goal?: string;
	beads_id?: string;
	set_at?: string;
	is_drifting: boolean;
	has_focus: boolean;
}

// Focus store
function createFocusStore() {
	const { subscribe, set } = writable<FocusInfo | null>(null);

	return {
		subscribe,
		set,
		// Fetch focus from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/focus`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch focus:', error);
				set({ has_focus: false, is_drifting: false });
			}
		}
	};
}

export const focus = createFocusStore();

// Helper to get drift indicator emoji
export function getDriftEmoji(focusInfo: FocusInfo | null): string {
	if (!focusInfo || !focusInfo.has_focus) return '';
	return focusInfo.is_drifting ? '⚠️' : '🎯';
}

// Helper to get drift indicator color
export function getDriftColor(focusInfo: FocusInfo | null): 'red' | 'green' | 'gray' {
	if (!focusInfo || !focusInfo.has_focus) return 'gray';
	return focusInfo.is_drifting ? 'red' : 'green';
}
