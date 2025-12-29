import { writable } from 'svelte/store';

// API configuration
const API_BASE = 'http://localhost:3348';

// Gap suggestion from recurring patterns
export interface GapSuggestion {
	query: string;
	count: number;
	priority: string;
	suggestion: string;
}

// Gaps API response
export interface GapsResponse {
	total_gaps: number;
	recurring_patterns: number;
	by_skill?: Record<string, { gaps: number; total: number; rate: number }>;
	recent_gaps: number;
	suggestions: GapSuggestion[];
	error?: string;
}

// Gaps store
function createGapsStore() {
	const { subscribe, set } = writable<GapsResponse | null>(null);

	return {
		subscribe,
		set,
		// Fetch gaps from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/gaps`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch gaps:', error);
				set({
					total_gaps: 0,
					recurring_patterns: 0,
					recent_gaps: 0,
					suggestions: [],
					error: String(error)
				});
			}
		}
	};
}

export const gaps = createGapsStore();
