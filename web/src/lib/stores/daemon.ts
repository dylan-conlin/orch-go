import { writable } from 'svelte/store';

// API configuration
const API_BASE = 'http://127.0.0.1:3348';

// Daemon status response from /api/daemon
export interface DaemonStatus {
	running: boolean;
	status?: string; // "running", "stalled", or undefined if not running
	last_poll?: string; // ISO 8601 timestamp
	last_poll_ago?: string; // Human-readable time since last poll
	last_spawn?: string; // ISO 8601 timestamp
	last_spawn_ago?: string; // Human-readable time since last spawn
	ready_count: number; // Issues ready to process
	capacity_max: number; // Maximum concurrent agents
	capacity_used: number; // Currently active agents
	capacity_free: number; // Available slots
	issues_per_hour?: number; // Processing rate (future)
}

// Daemon store
function createDaemonStore() {
	const { subscribe, set } = writable<DaemonStatus | null>(null);

	return {
		subscribe,
		set,
		// Fetch daemon status from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/daemon`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch daemon status:', error);
				// Set to not running on error
				set({
					running: false,
					ready_count: 0,
					capacity_max: 0,
					capacity_used: 0,
					capacity_free: 0
				});
			}
		}
	};
}

export const daemon = createDaemonStore();

// Helper to get status emoji
export function getDaemonEmoji(status: DaemonStatus | null): string {
	if (!status?.running) return '💤'; // Not running
	if (status.status === 'stalled') return '⚠️'; // Stalled
	if (status.capacity_free === 0) return '🔴'; // At capacity
	return '🟢'; // Running with capacity
}

// Helper to get status label
export function getDaemonLabel(status: DaemonStatus | null): string {
	if (!status?.running) return 'stopped';
	if (status.status === 'stalled') return 'stalled';
	return 'running';
}

// Helper to get capacity display
export function getDaemonCapacity(status: DaemonStatus | null): string {
	if (!status?.running) return '';
	return `${status.capacity_used}/${status.capacity_max}`;
}
