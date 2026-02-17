import { writable } from 'svelte/store';
import type { AttentionBadgeType } from './work-graph';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

export interface CompletedIssue {
	id: string;
	title: string;
	completed_at: string;
	project: string;
}

export interface AttentionSignal {
	badge: AttentionBadgeType;
	reason: string;
}

export interface AttentionStoreData {
	completedIssues: CompletedIssue[];
	signals: Map<string, AttentionSignal>;
}

// API response types (matching backend)
interface AttentionItemResponse {
	id: string;
	source: string;
	concern: string;
	signal: string;
	subject: string;
	summary: string;
	priority: number;
	role: string;
	action_hint?: string;
	collected_at: string;
	metadata?: Record<string, any>;
}

interface AttentionAPIResponse {
	items: AttentionItemResponse[];
	total: number;
	sources: string[];
	role: string;
	collected_at: string;
}

function createAttentionStore() {
	const { subscribe, set, update } = writable<AttentionStoreData>({
		completedIssues: [],
		signals: new Map()
	});

	return {
		subscribe,
		
		async fetch(projectDir?: string): Promise<void> {
			try {
				const url = new URL(`${API_BASE}/api/attention`);
				if (projectDir) {
					url.searchParams.set('project', projectDir);
				}

				const response = await fetch(url.toString());
				if (!response.ok) {
					console.error('Failed to fetch attention signals:', response.statusText);
					return;
				}

				const data: AttentionAPIResponse = await response.json();
				
				// Transform API response into store data
				const completedIssues: CompletedIssue[] = [];
				const signals = new Map<string, AttentionSignal>();

			for (const item of data.items) {
				// Map signal type to badge type
				const badge = mapSignalToBadge(item);
				
				// Only add to signals map if badge is not null
				// (filters out informational signals like 'issue-ready' that don't need visual badges)
				if (badge !== null) {
					signals.set(item.subject, {
						badge,
						reason: item.summary
					});
				}

				// Add to completedIssues if recently-closed
				if (item.signal === 'recently-closed') {
					completedIssues.push({
						id: item.subject,
						title: item.summary,
						completed_at: item.collected_at,
						project: projectDir || ''
					});
				}
			}

				set({ completedIssues, signals });
			} catch (err) {
				console.error('Error fetching attention signals:', err);
			}
		},
		
		set,
		update
	};
}

/**
 * Map attention signal types to badge types based on signal and metadata
 */
function mapSignalToBadge(item: AttentionItemResponse): AttentionBadgeType | null {
	// Handle recently-closed with verification status
	if (item.signal === 'recently-closed') {
		const verificationStatus = item.metadata?.verification_status;
		if (verificationStatus === 'verified') {
			return 'recently_closed';
		}
		if (verificationStatus === 'needs_fix') {
			return 'verify_failed';
		}
		// Default to 'verify' for unverified or no status
		return 'verify';
	}

	// Direct signal to badge mappings
	switch (item.signal) {
		case 'likely-done':
			return 'likely_done';
		case 'stuck':
			return 'stuck';
		case 'unblocked':
			return 'unblocked';
		case 'verify':
			return 'verify';
		case 'verify-failed':
			return 'verify_failed';
		default:
			// Return null for unmapped signals (e.g., 'issue-ready', 'stale', etc.)
			// These are informational signals that don't need visual badges
			return null;
	}
}

export const attention = createAttentionStore();

export function formatRelativeTime(timestamp: string): string {
	const now = new Date();
	const then = new Date(timestamp);
	const diffMs = now.getTime() - then.getTime();
	const diffMins = Math.floor(diffMs / 60000);
	
	if (diffMins < 1) return 'just now';
	if (diffMins < 60) return `${diffMins}m ago`;
	
	const diffHours = Math.floor(diffMins / 60);
	if (diffHours < 24) return `${diffHours}h ago`;
	
	const diffDays = Math.floor(diffHours / 24);
	return `${diffDays}d ago`;
}

// Attention badge configuration
export const ATTENTION_BADGE_CONFIG: Record<AttentionBadgeType, {
	color: string;
	bg: string;
	label: string;
}> = {
	verify: {
		color: 'text-blue-600',
		bg: 'bg-blue-100',
		label: 'Verify'
	},
	decide: {
		color: 'text-purple-600',
		bg: 'bg-purple-100',
		label: 'Decide'
	},
	escalate: {
		color: 'text-orange-600',
		bg: 'bg-orange-100',
		label: 'Escalate'
	},
	likely_done: {
		color: 'text-green-600',
		bg: 'bg-green-100',
		label: 'Likely Done'
	},
	recently_closed: {
		color: 'text-gray-600',
		bg: 'bg-gray-100',
		label: 'Recently Closed'
	},
	unblocked: {
		color: 'text-teal-600',
		bg: 'bg-teal-100',
		label: 'Unblocked'
	},
	stuck: {
		color: 'text-red-600',
		bg: 'bg-red-100',
		label: 'Stuck'
	},
	crashed: {
		color: 'text-red-700',
		bg: 'bg-red-200',
		label: 'Crashed'
	},
	verify_failed: {
		color: 'text-yellow-700',
		bg: 'bg-yellow-100',
		label: 'Verify Failed'
	}
};
