import { writable } from 'svelte/store';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'http://localhost:3348';

// Service from /api/services
export interface Service {
	name: string;
	pid: number;
	status: string; // "running", "stopped", etc.
	restart_count: number;
	uptime: string; // Human-readable uptime (e.g., "2h 15m")
}

// API response structure
interface ServicesResponse {
	project: string;
	services: Service[];
	total_count: number;
	running_count: number;
	stopped_count: number;
}

// Services store
function createServicesStore() {
	const { subscribe, set } = writable<ServicesResponse>({
		project: '',
		services: [],
		total_count: 0,
		running_count: 0,
		stopped_count: 0
	});

	return {
		subscribe,
		set,
		// Fetch services from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/services`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data: ServicesResponse = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch services:', error);
				set({
					project: '',
					services: [],
					total_count: 0,
					running_count: 0,
					stopped_count: 0
				});
			}
		}
	};
}

export const services = createServicesStore();

// Helper to get status color
export function getStatusColor(status: string, pid: number): string {
	if (status === 'running' && pid !== 0) {
		return 'text-green-400';
	}
	return 'text-red-400';
}

// Helper to get status icon
export function getStatusIcon(status: string, pid: number): string {
	if (status === 'running' && pid !== 0) {
		return '●'; // Running
	}
	return '○'; // Stopped
}
