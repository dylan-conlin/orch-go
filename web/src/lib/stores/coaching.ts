import { writable } from 'svelte/store';
import type { Writable } from 'svelte/store';

// Worker health metrics for individual worker sessions
export interface WorkerHealthMetrics {
	session_id: string;
	// tool_failure_rate: consecutive tool failures (>=3 is warning)
	tool_failure_rate: number;
	// context_usage: estimated token usage percentage (>=80 is warning)
	context_usage: number;
	// time_in_phase: minutes since last phase change (>=15 is warning)
	time_in_phase: number;
	// commit_gap: minutes since last commit (>=30 is warning)
	commit_gap: number;
	// Derived health status: good/warning/critical
	health_status: 'good' | 'warning' | 'critical';
	// Last update timestamp
	last_updated: string;
}

export interface CoachingData {
	overall_status: 'good' | 'warning' | 'poor';
	status_message: string;
	last_coaching_time?: string;
	session: {
		session_id: string;
		started: string;
		duration_minutes: number;
	};
	// Worker health metrics keyed by session ID
	worker_health?: Record<string, WorkerHealthMetrics>;
}

const emptyCoaching: CoachingData = {
	overall_status: 'good',
	status_message: 'No metrics yet',
	session: {
		session_id: '',
		started: '',
		duration_minutes: 0
	}
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

/**
 * Get worker health metrics for a specific session ID.
 * Returns null if no health data available for the session.
 */
export function getWorkerHealthForSession(sessionId: string): WorkerHealthMetrics | null {
	let result: WorkerHealthMetrics | null = null;
	coaching.subscribe(data => {
		result = data.worker_health?.[sessionId] ?? null;
	})();
	return result;
}

/**
 * Check if a session has any health warnings or critical issues.
 */
export function hasHealthIssues(health: WorkerHealthMetrics | null): boolean {
	if (!health) return false;
	return health.health_status === 'warning' || health.health_status === 'critical';
}

/**
 * Get a formatted health summary string for display.
 */
export function getHealthSummary(health: WorkerHealthMetrics | null): string | null {
	if (!health || health.health_status === 'good') return null;

	const issues: string[] = [];

	if (health.tool_failure_rate >= 3) {
		issues.push(`${health.tool_failure_rate} tool failures`);
	}
	if (health.context_usage >= 80) {
		issues.push(`${health.context_usage}% context`);
	}
	if (health.time_in_phase >= 15) {
		issues.push(`${health.time_in_phase}m in phase`);
	}
	if (health.commit_gap >= 30) {
		issues.push(`${health.commit_gap}m since commit`);
	}

	return issues.length > 0 ? issues.join(', ') : null;
}
