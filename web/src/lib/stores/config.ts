import { writable } from 'svelte/store';

// API configuration
const API_BASE = 'http://127.0.0.1:3348';

// Config response from /api/config
export interface ConfigInfo {
	backend: string;
	auto_export_transcript: boolean;
	notifications_enabled: boolean;
	config_path?: string;
}

// Config update request - all fields optional (only update what's provided)
export interface ConfigUpdateRequest {
	backend?: string;
	auto_export_transcript?: boolean;
	notifications_enabled?: boolean;
}

// Config store
function createConfigStore() {
	const { subscribe, set, update } = writable<ConfigInfo | null>(null);

	return {
		subscribe,
		set,
		update,
		// Fetch config from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/config`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch config:', error);
				// Set a default state so the UI can still render
				set({
					backend: 'opencode',
					auto_export_transcript: false,
					notifications_enabled: true
				});
			}
		},
		// Update config on the server
		async save(updates: ConfigUpdateRequest): Promise<boolean> {
			try {
				const response = await fetch(`${API_BASE}/api/config`, {
					method: 'PUT',
					headers: {
						'Content-Type': 'application/json'
					},
					body: JSON.stringify(updates)
				});
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
				return true;
			} catch (error) {
				console.error('Failed to save config:', error);
				return false;
			}
		},
		// Toggle a boolean setting
		async toggle(key: 'auto_export_transcript' | 'notifications_enabled'): Promise<boolean> {
			let newValue: boolean | undefined;
			update(current => {
				if (current) {
					newValue = !current[key];
				}
				return current;
			});
			if (newValue !== undefined) {
				return await this.save({ [key]: newValue });
			}
			return false;
		}
	};
}

export const config = createConfigStore();
