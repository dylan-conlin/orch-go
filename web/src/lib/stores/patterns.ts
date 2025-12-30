import { writable } from 'svelte/store';

// API configuration
const API_BASE = 'http://localhost:3348';

// A detected behavioral pattern
export interface BehavioralPattern {
	type: string;           // Pattern type (e.g., "repeated_empty_read", "repeated_error")
	description: string;    // Human-readable description
	severity: string;       // "info", "warning", "critical"
	count: number;          // How many times this pattern occurred
	suggestion?: string;    // Suggested action
	context?: Record<string, string>; // Common context (tier, skill, etc.)
	tool?: string;          // Tool that triggered the pattern
	target?: string;        // Target of the action (file path, command, etc.)
}

// Patterns API response
export interface PatternsResponse {
	total_events: number;   // Total action events in log
	total_patterns: number; // Number of detected patterns
	patterns: BehavioralPattern[];
	summary: string;        // Brief summary of log state
	error?: string;
}

// Patterns store
function createPatternsStore() {
	const { subscribe, set } = writable<PatternsResponse | null>(null);

	return {
		subscribe,
		set,
		// Fetch patterns from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/patterns`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch patterns:', error);
				set({
					total_events: 0,
					total_patterns: 0,
					patterns: [],
					summary: '',
					error: String(error)
				});
			}
		}
	};
}

export const patterns = createPatternsStore();

// Helper to get severity color class
export function getSeverityColor(severity: string): string {
	switch (severity) {
		case 'critical':
			return 'text-red-500 border-red-500/50';
		case 'warning':
			return 'text-orange-500 border-orange-500/50';
		case 'info':
		default:
			return 'text-blue-500 border-blue-500/50';
	}
}

// Helper to get severity icon
export function getSeverityIcon(severity: string): string {
	switch (severity) {
		case 'critical':
			return '●';
		case 'warning':
			return '◐';
		case 'info':
		default:
			return '○';
	}
}
