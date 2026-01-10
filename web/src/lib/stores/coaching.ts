import { writable } from 'svelte/store';
import type { Writable } from 'svelte/store';

export interface CoachingMetric {
	value: number;
	label: string;
	status: 'good' | 'warning' | 'poor';
}

export interface CoachingData {
	session: {
		session_id: string;
		started: string;
		duration_minutes: number;
	};
	metrics: {
		[key: string]: CoachingMetric;
	};
	coaching: string[];
}

const emptyCoaching: CoachingData = {
	session: {
		session_id: '',
		started: '',
		duration_minutes: 0
	},
	metrics: {},
	coaching: []
};

export const coaching: Writable<CoachingData> = writable(emptyCoaching);

let refreshInterval: ReturnType<typeof setInterval> | null = null;

export async function fetchCoaching() {
	try {
		const response = await fetch('https://localhost:3348/api/coaching');
		if (!response.ok) {
			console.error('Failed to fetch coaching metrics:', response.statusText);
			return;
		}
		const data: CoachingData = await response.json();
		coaching.set(data);
	} catch (err) {
		console.error('Error fetching coaching metrics:', err);
	}
}

export function startCoachingPolling(intervalMs: number = 30000) {
	// Fetch immediately
	fetchCoaching();

	// Then poll every intervalMs
	if (refreshInterval) {
		clearInterval(refreshInterval);
	}
	refreshInterval = setInterval(fetchCoaching, intervalMs);
}

export function stopCoachingPolling() {
	if (refreshInterval) {
		clearInterval(refreshInterval);
		refreshInterval = null;
	}
}
