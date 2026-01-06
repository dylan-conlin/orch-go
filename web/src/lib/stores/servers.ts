import { writable } from 'svelte/store';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Port info for a server
export interface ServerPortInfo {
	service: string;
	port: number;
	purpose?: string;
}

// Project server info
export interface ServerProjectInfo {
	project: string;
	ports: ServerPortInfo[];
	running: boolean;
	session?: string;
}

// Servers response from /api/servers
export interface ServersInfo {
	projects: ServerProjectInfo[];
	total_count: number;
	running_count: number;
	stopped_count: number;
	error?: string;
}

// Servers store
function createServersStore() {
	const { subscribe, set } = writable<ServersInfo | null>(null);

	return {
		subscribe,
		set,
		// Fetch servers from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/servers`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch servers:', error);
				set({
					projects: [],
					total_count: 0,
					running_count: 0,
					stopped_count: 0,
					error: String(error)
				});
			}
		}
	};
}

export const servers = createServersStore();
