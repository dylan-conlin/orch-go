import { writable } from 'svelte/store';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Daemon config response from GET /api/config/daemon
export interface DaemonConfig {
	poll_interval: number;
	max_agents: number;
	label: string;
	verbose: boolean;
	reflect_issues: boolean;
	working_directory: string;
	path: string[];
}

// Daemon config update response (includes regeneration status)
export interface DaemonConfigUpdateResponse extends DaemonConfig {
	plist_regenerated: boolean;
	daemon_kicked: boolean;
	regenerate_error?: string;
}

// Drift status response from GET /api/config/drift
export interface DriftStatus {
	in_sync: boolean;
	plist_path: string;
	plist_exists: boolean;
	config_path: string;
	drift_details?: string;
}

// Regenerate response from POST /api/config/regenerate
export interface RegenerateResponse {
	success: boolean;
	plist_path: string;
	daemon_kicked: boolean;
	error?: string;
}

// Daemon config update request - all fields optional
export interface DaemonConfigUpdateRequest {
	poll_interval?: number;
	max_agents?: number;
	label?: string;
	verbose?: boolean;
	reflect_issues?: boolean;
	working_directory?: string;
}

// Daemon config store
function createDaemonConfigStore() {
	const { subscribe, set, update } = writable<DaemonConfig | null>(null);

	return {
		subscribe,
		set,
		update,
		// Fetch daemon config from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/config/daemon`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch daemon config:', error);
				// Set default state so UI can still render
				set({
					poll_interval: 60,
					max_agents: 3,
					label: 'triage:ready',
					verbose: true,
					reflect_issues: false,
					working_directory: '',
					path: []
				});
			}
		},
		// Update daemon config on the server
		async save(updates: DaemonConfigUpdateRequest): Promise<DaemonConfigUpdateResponse | null> {
			try {
				const response = await fetch(`${API_BASE}/api/config/daemon`, {
					method: 'PUT',
					headers: {
						'Content-Type': 'application/json'
					},
					body: JSON.stringify(updates)
				});
				if (!response.ok) {
					const text = await response.text();
					throw new Error(text || `HTTP ${response.status}`);
				}
				const data: DaemonConfigUpdateResponse = await response.json();
				// Update local state with new config
				set({
					poll_interval: data.poll_interval,
					max_agents: data.max_agents,
					label: data.label,
					verbose: data.verbose,
					reflect_issues: data.reflect_issues,
					working_directory: data.working_directory,
					path: data.path
				});
				return data;
			} catch (error) {
				console.error('Failed to save daemon config:', error);
				throw error;
			}
		}
	};
}

// Drift status store
function createDriftStore() {
	const { subscribe, set } = writable<DriftStatus | null>(null);

	return {
		subscribe,
		set,
		// Fetch drift status from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/config/drift`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch drift status:', error);
				set(null);
			}
		},
		// Regenerate plist from config
		async regenerate(): Promise<RegenerateResponse | null> {
			try {
				const response = await fetch(`${API_BASE}/api/config/regenerate`, {
					method: 'POST'
				});
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data: RegenerateResponse = await response.json();
				// Refetch drift status after regeneration
				await this.fetch();
				return data;
			} catch (error) {
				console.error('Failed to regenerate plist:', error);
				throw error;
			}
		}
	};
}

export const daemonConfig = createDaemonConfigStore();
export const driftStatus = createDriftStore();
