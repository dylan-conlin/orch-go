import { writable, derived } from 'svelte/store';
import type { GraphNode } from './work-graph';
import type { Variant } from '$lib/components/ui/badge';

// API configuration - HTTPS for HTTP/2 multiplexing (same as work-graph.ts)
const API_BASE = 'https://localhost:3348';

// Attention badge types for active work
export type AttentionBadgeType =
	| 'verify'         // Phase: Complete, needs orch complete
	| 'decide'         // Investigation has recommendation needing decision
	| 'escalate'       // Question needs human judgment
	| 'likely_done'    // Commits suggest completion
	| 'recently_closed' // Recently closed, needs verification
	| 'unblocked'      // Blocker just closed, now actionable
	| 'stuck'          // Agent stuck >2h
	| 'crashed';       // Agent crashed without completing

// Verification status for completed issues
export type VerificationStatus =
	| 'unverified'  // Completed but not human-verified
	| 'verified'    // Human verified as correct
	| 'needs_fix';  // Verified incorrect, needs rework

// Attention signal attached to an issue
export interface AttentionSignal {
	issueId: string;
	badge: AttentionBadgeType;
	reason: string;      // Human-readable explanation
	source: string;      // Where signal came from (e.g., "beads comment", "git commits")
	timestamp: string;   // When signal was detected
}

// Completed issue with verification tracking
export interface CompletedIssue extends GraphNode {
	completedAt: string;
	verificationStatus: VerificationStatus;
	attentionBadge?: 'unverified' | 'needs_fix'; // Only for issues needing attention
}

// Attention store state
interface AttentionState {
	// Map of issue ID -> attention signal (for active issues)
	signals: Map<string, AttentionSignal>;
	// Recently completed issues (last 24h)
	completedIssues: CompletedIssue[];
	// Loading state
	loading: boolean;
}

// Badge display configuration
export const ATTENTION_BADGE_CONFIG: Record<AttentionBadgeType | 'unverified' | 'needs_fix', {
	label: string;
	variant: Variant;
}> = {
	verify: { label: 'VERIFY', variant: 'attention_verify' },
	decide: { label: 'DECIDE', variant: 'attention_decide' },
	escalate: { label: 'ESCALATE', variant: 'attention_escalate' },
	likely_done: { label: 'LIKELY DONE', variant: 'attention_likely_done' },
	recently_closed: { label: 'RECENTLY CLOSED', variant: 'attention_recently_closed' },
	unblocked: { label: 'UNBLOCKED', variant: 'attention_unblocked' },
	stuck: { label: 'STUCK', variant: 'attention_stuck' },
	crashed: { label: 'CRASHED', variant: 'attention_crashed' },
	unverified: { label: 'UNVERIFIED', variant: 'attention_unverified' },
	needs_fix: { label: 'NEEDS FIX', variant: 'attention_needs_fix' },
};

// ============================================================================
// API Types - Match backend /api/attention response structure
// ============================================================================

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

// ============================================================================
// Mapping Functions
// ============================================================================

// Map backend signal types to frontend badge types
function mapSignalToBadge(signal: string): AttentionBadgeType | null {
	switch (signal) {
		case 'likely-done':
			return 'likely_done';
		case 'recently-closed':
			return 'recently_closed';
		case 'issue-ready':
			// issue-ready doesn't have a direct badge mapping yet
			// This is for actionable work, not attention needing human review
			return null;
		default:
			return null;
	}
}

// ============================================================================
// Store Implementation
// ============================================================================

function createAttentionStore() {
	const { subscribe, set, update } = writable<AttentionState>({
		signals: new Map(),
		completedIssues: [],
		loading: false,
	});

	return {
		subscribe,

		// Fetch attention signals from /api/attention endpoint
		async fetch(): Promise<void> {
			update(s => ({ ...s, loading: true }));

			try {
				// Call /api/attention endpoint on orch-go server
				const response = await fetch(`${API_BASE}/api/attention?role=human`);
				
				if (!response.ok) {
					console.error('Failed to fetch attention signals:', response.statusText);
					// Set empty state on error
					set({
						signals: new Map(),
						completedIssues: [],
						loading: false,
					});
					return;
				}

				const data: AttentionAPIResponse = await response.json();

			// Map API response to store state
			const signalsMap = new Map<string, AttentionSignal>();
			const completedIssuesList: CompletedIssue[] = [];
			
			for (const item of data.items) {
				// Map backend signal types to frontend badge types
				const badge = mapSignalToBadge(item.signal);
				if (!badge) {
					// Skip signals that don't map to known badge types
					continue;
				}

					// For recently-closed signals, create CompletedIssue entries
				if (item.signal === 'recently-closed' && item.metadata) {
					const completedIssue: CompletedIssue = {
						id: item.subject,
						title: item.summary.split(': ').slice(1).join(': '), // Remove "Closed Xh ago:" prefix
						description: '',
						status: item.metadata.status || 'closed',
						priority: item.metadata.beads_priority || 0,
						type: item.metadata.issue_type || 'task',
						source: 'beads',
						completedAt: item.metadata.closed_at || item.collected_at,
						verificationStatus: 'unverified',
						attentionBadge: 'unverified',
					};
					completedIssuesList.push(completedIssue);
				}

				const signal: AttentionSignal = {
					issueId: item.subject,
					badge: badge,
					reason: item.metadata?.reason || item.summary,
					source: item.source,
					timestamp: item.collected_at,
				};

				signalsMap.set(signal.issueId, signal);
			}

			set({
				signals: signalsMap,
				completedIssues: completedIssuesList,
				loading: false,
			});
			} catch (error) {
				console.error('Error fetching attention signals:', error);
				// Set empty state on error
				set({
					signals: new Map(),
					completedIssues: [],
					loading: false,
				});
			}
		},

		// Get attention signal for a specific issue
		getSignal(issueId: string): AttentionSignal | undefined {
			let signal: AttentionSignal | undefined;
			subscribe(s => {
				signal = s.signals.get(issueId);
			})();
			return signal;
		},

		// Mark a completed issue as verified (calls API and updates local state)
		async markVerified(issueId: string): Promise<boolean> {
			try {
				const response = await fetch(`${API_BASE}/api/attention/verify`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ issue_id: issueId, status: 'verified' }),
				});

				if (!response.ok) {
					console.error('Failed to mark verified:', response.statusText);
					return false;
				}

				// Update local state
				update(s => ({
					...s,
					completedIssues: s.completedIssues.map(issue =>
						issue.id === issueId
							? { ...issue, verificationStatus: 'verified' as const, attentionBadge: undefined }
							: issue
					),
				}));
				return true;
			} catch (error) {
				console.error('Error marking verified:', error);
				return false;
			}
		},

		// Mark a completed issue as needing fix (calls API and updates local state)
		async markNeedsFix(issueId: string): Promise<boolean> {
			try {
				const response = await fetch(`${API_BASE}/api/attention/verify`, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ issue_id: issueId, status: 'needs_fix' }),
				});

				if (!response.ok) {
					console.error('Failed to mark needs_fix:', response.statusText);
					return false;
				}

				// Update local state
				update(s => ({
					...s,
					completedIssues: s.completedIssues.map(issue =>
						issue.id === issueId
							? { ...issue, verificationStatus: 'needs_fix' as const, attentionBadge: 'needs_fix' as const }
							: issue
					),
				}));
				return true;
			} catch (error) {
				console.error('Error marking needs_fix:', error);
				return false;
			}
		},

		// Clear all state
		clear(): void {
			set({
				signals: new Map(),
				completedIssues: [],
				loading: false,
			});
		},
	};
}

export const attention = createAttentionStore();

// Derived store: count of items needing attention
export const attentionCounts = derived(attention, ($attention) => {
	const activeSignals = $attention.signals.size;
	const unverifiedCompleted = $attention.completedIssues.filter(
		i => i.verificationStatus !== 'verified'
	).length;

	return {
		active: activeSignals,
		completed: unverifiedCompleted,
		total: activeSignals + unverifiedCompleted,
	};
});

// Helper: format relative time
export function formatRelativeTime(timestamp: string): string {
	const now = Date.now();
	const then = new Date(timestamp).getTime();
	const diffMs = now - then;
	const diffMins = Math.floor(diffMs / 60000);
	const diffHours = Math.floor(diffMs / 3600000);
	const diffDays = Math.floor(diffMs / 86400000);

	if (diffMins < 1) return 'just now';
	if (diffMins < 60) return `${diffMins}m ago`;
	if (diffHours < 24) return `${diffHours}h ago`;
	return `${diffDays}d ago`;
}
