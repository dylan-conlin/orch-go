import { writable } from 'svelte/store';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Focus response from /api/focus
export interface FocusInfo {
	goal?: string;
	beads_id?: string;
	set_at?: string;
	is_drifting: boolean;
	has_focus: boolean;
}

// Set focus request
interface SetFocusRequest {
	goal?: string;
	beads_id?: string;
}

// Set focus response
interface SetFocusResponse {
	success: boolean;
	error?: string;
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
		},
		
		// Set a new focus
		async setFocus(goal?: string, beadsId?: string): Promise<{ success: boolean; error?: string }> {
			try {
				const request: SetFocusRequest = {};
				if (goal) request.goal = goal;
				if (beadsId) request.beads_id = beadsId;
				
				const response = await fetch(`${API_BASE}/api/focus`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(request),
				});
				
				if (!response.ok) {
					const text = await response.text();
					return { success: false, error: `HTTP ${response.status}: ${text}` };
				}
				
				const data: SetFocusResponse = await response.json();
				
				if (data.success) {
					// Refresh the focus state
					await this.fetch();
				}
				
				return data;
			} catch (error) {
				return { success: false, error: String(error) };
			}
		},
		
		// Clear the current focus
		async clearFocus(): Promise<{ success: boolean; error?: string }> {
			try {
				const response = await fetch(`${API_BASE}/api/focus`, {
					method: 'DELETE',
				});
				
				if (!response.ok) {
					const text = await response.text();
					return { success: false, error: `HTTP ${response.status}: ${text}` };
				}
				
				const data: SetFocusResponse = await response.json();
				
				if (data.success) {
					// Clear the local state
					set({ has_focus: false, is_drifting: false });
				}
				
				return data;
			} catch (error) {
				return { success: false, error: String(error) };
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
