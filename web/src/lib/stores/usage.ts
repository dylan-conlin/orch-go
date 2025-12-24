import { writable } from 'svelte/store';

// API configuration
const API_BASE = 'http://127.0.0.1:3348';

// Usage response from /api/usage
export interface UsageInfo {
	account: string;
	account_name?: string; // Account name from accounts.yaml (e.g., "personal", "work")
	five_hour_percent: number;
	weekly_percent: number;
	weekly_opus_percent?: number;
	error?: string;
}

// Usage store
function createUsageStore() {
	const { subscribe, set } = writable<UsageInfo | null>(null);

	return {
		subscribe,
		set,
		// Fetch usage from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/usage`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch usage:', error);
				set({ account: '', five_hour_percent: 0, weekly_percent: 0, error: String(error) });
			}
		}
	};
}

export const usage = createUsageStore();

// Helper to get color class based on percentage
// green <60%, yellow 60-80%, red >80%
export function getUsageColor(percent: number): 'green' | 'yellow' | 'red' {
	if (percent >= 80) return 'red';
	if (percent >= 60) return 'yellow';
	return 'green';
}

// Helper to get emoji based on percentage
export function getUsageEmoji(percent: number): string {
	if (percent >= 80) return '🔴';
	if (percent >= 60) return '🟡';
	return '🟢';
}
