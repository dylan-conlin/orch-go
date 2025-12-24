import { writable } from 'svelte/store';

// API configuration
const API_BASE = 'http://127.0.0.1:3348';

// Beads stats response from /api/beads
export interface BeadsStats {
	total_issues: number;
	open_issues: number;
	in_progress_issues: number;
	blocked_issues: number;
	ready_issues: number;
	closed_issues: number;
	avg_lead_time_hours?: number;
	error?: string;
}

// Beads store
function createBeadsStore() {
	const { subscribe, set } = writable<BeadsStats | null>(null);

	return {
		subscribe,
		set,
		// Fetch beads stats from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/beads`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch beads stats:', error);
				set({
					total_issues: 0,
					open_issues: 0,
					in_progress_issues: 0,
					blocked_issues: 0,
					ready_issues: 0,
					closed_issues: 0,
					error: String(error)
				});
			}
		}
	};
}

export const beads = createBeadsStore();
