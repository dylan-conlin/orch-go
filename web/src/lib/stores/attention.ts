import { writable, derived } from 'svelte/store';
import type { GraphNode } from './work-graph';

// Attention badge types for active work
export type AttentionBadgeType =
	| 'verify'      // Phase: Complete, needs orch complete
	| 'decide'      // Investigation has recommendation needing decision
	| 'escalate'    // Question needs human judgment
	| 'likely_done' // Commits suggest completion
	| 'unblocked'   // Blocker just closed, now actionable
	| 'stuck'       // Agent stuck >2h
	| 'crashed';    // Agent crashed without completing

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
	variant: string;
}> = {
	verify: { label: 'VERIFY', variant: 'attention_verify' },
	decide: { label: 'DECIDE', variant: 'attention_decide' },
	escalate: { label: 'ESCALATE', variant: 'attention_escalate' },
	likely_done: { label: 'LIKELY DONE', variant: 'attention_likely_done' },
	unblocked: { label: 'UNBLOCKED', variant: 'attention_unblocked' },
	stuck: { label: 'STUCK', variant: 'attention_stuck' },
	crashed: { label: 'CRASHED', variant: 'attention_crashed' },
	unverified: { label: 'UNVERIFIED', variant: 'attention_unverified' },
	needs_fix: { label: 'NEEDS FIX', variant: 'attention_needs_fix' },
};

// ============================================================================
// MOCK DATA FOR PROTOTYPING
// Replace with real API calls when backend is ready
// ============================================================================

const MOCK_ATTENTION_SIGNALS: AttentionSignal[] = [
	{
		issueId: 'orch-go-21180',
		badge: 'verify',
		reason: 'Phase: Complete reported',
		source: 'beads comment',
		timestamp: new Date(Date.now() - 15 * 60 * 1000).toISOString(), // 15m ago
	},
	{
		issueId: 'orch-go-20876',
		badge: 'likely_done',
		reason: '3 commits reference this issue',
		source: 'git commits',
		timestamp: new Date(Date.now() - 60 * 60 * 1000).toISOString(), // 1h ago
	},
	{
		issueId: 'orch-go-20999',
		badge: 'unblocked',
		reason: 'Blocker orch-go-20888 was closed',
		source: 'beads dependency',
		timestamp: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(), // 2h ago
	},
	{
		issueId: 'orch-go-19845',
		badge: 'verify',
		reason: 'Phase: Complete reported',
		source: 'beads comment',
		timestamp: new Date(Date.now() - 60 * 60 * 1000).toISOString(), // 1h ago
	},
];

const MOCK_COMPLETED_ISSUES: CompletedIssue[] = [
	{
		id: 'orch-go-20445',
		title: 'Implement SSE reconnection logic',
		type: 'task',
		status: 'closed',
		priority: 1,
		source: 'beads',
		completedAt: new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString(), // 2h ago
		verificationStatus: 'needs_fix',
		attentionBadge: 'needs_fix',
	},
	{
		id: 'orch-go-20512',
		title: 'Fix agent status polling',
		type: 'bug',
		status: 'closed',
		priority: 2,
		source: 'beads',
		completedAt: new Date(Date.now() - 1 * 60 * 60 * 1000).toISOString(), // 1h ago
		verificationStatus: 'unverified',
		attentionBadge: 'unverified',
	},
	{
		id: 'orch-go-20398',
		title: 'Add context % to agent display',
		type: 'task',
		status: 'closed',
		priority: 2,
		source: 'beads',
		completedAt: new Date(Date.now() - 5 * 60 * 60 * 1000).toISOString(), // 5h ago
		verificationStatus: 'verified',
		// No attentionBadge - verified correct
	},
	{
		id: 'orch-go-20401',
		title: 'Update spawn context template',
		type: 'task',
		status: 'closed',
		priority: 2,
		source: 'beads',
		completedAt: new Date(Date.now() - 8 * 60 * 60 * 1000).toISOString(), // 8h ago
		verificationStatus: 'verified',
		// No attentionBadge - verified correct
	},
];

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

		// Initialize with mock data (for prototyping)
		// Replace with real API call when backend is ready
		async fetch(): Promise<void> {
			update(s => ({ ...s, loading: true }));

			// Simulate API delay
			await new Promise(resolve => setTimeout(resolve, 100));

			// Load mock data
			const signalsMap = new Map<string, AttentionSignal>();
			for (const signal of MOCK_ATTENTION_SIGNALS) {
				signalsMap.set(signal.issueId, signal);
			}

			set({
				signals: signalsMap,
				completedIssues: MOCK_COMPLETED_ISSUES,
				loading: false,
			});
		},

		// Get attention signal for a specific issue
		getSignal(issueId: string): AttentionSignal | undefined {
			let signal: AttentionSignal | undefined;
			subscribe(s => {
				signal = s.signals.get(issueId);
			})();
			return signal;
		},

		// Mark a completed issue as verified
		markVerified(issueId: string): void {
			update(s => ({
				...s,
				completedIssues: s.completedIssues.map(issue =>
					issue.id === issueId
						? { ...issue, verificationStatus: 'verified' as const, attentionBadge: undefined }
						: issue
				),
			}));
		},

		// Mark a completed issue as needing fix
		markNeedsFix(issueId: string): void {
			update(s => ({
				...s,
				completedIssues: s.completedIssues.map(issue =>
					issue.id === issueId
						? { ...issue, verificationStatus: 'needs_fix' as const, attentionBadge: 'needs_fix' as const }
						: issue
				),
			}));
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
