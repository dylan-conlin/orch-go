import { writable, derived } from 'svelte/store';
import { agents, type Agent } from './agents';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Ready issue from /api/beads/ready (simplified for WIP display)
export interface ReadyIssue {
	id: string;
	title: string;
	priority: number;
	issue_type: string;
	created_at: string;
}

// WIP item can be either a running agent or a queued issue
export type WIPItem = 
	| { type: 'running'; agent: Agent }
	| { type: 'queued'; issue: ReadyIssue };

// WIP store state
interface WIPState {
	runningAgents: Agent[];
	queuedIssues: ReadyIssue[];
	loading: boolean;
	error: string | null;
}

// Create the WIP store
function createWIPStore() {
	const { subscribe, set, update } = writable<WIPState>({
		runningAgents: [],
		queuedIssues: [],
		loading: false,
		error: null
	});

	return {
		subscribe,
		
		// Fetch queued issues from beads/ready
		async fetchQueued(): Promise<void> {
			update(s => ({ ...s, loading: true, error: null }));
			try {
				const response = await fetch(`${API_BASE}/api/beads/ready`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				// Limit to top 5 for WIP display
				const issues = (data.issues || []).slice(0, 5);
				update(s => ({ ...s, queuedIssues: issues, loading: false }));
			} catch (error) {
				console.error('Failed to fetch queued issues:', error);
				update(s => ({ 
					...s, 
					error: error instanceof Error ? error.message : 'Failed to fetch', 
					loading: false 
				}));
			}
		},

		// Update running agents from the agents store
		setRunningAgents(agentList: Agent[]): void {
			const running = agentList.filter(a => 
				a.status === 'active' || a.status === 'idle'
			);
			update(s => ({ ...s, runningAgents: running }));
		},

		// Clear all state
		clear(): void {
			set({
				runningAgents: [],
				queuedIssues: [],
				loading: false,
				error: null
			});
		}
	};
}

export const wip = createWIPStore();

// Derived store that combines running agents + queued issues into a single list
// Only shows queued issues when there are running agents (otherwise tree below has same info)
export const wipItems = derived(wip, ($wip): WIPItem[] => {
	const items: WIPItem[] = [];
	
	// Running agents first
	for (const agent of $wip.runningAgents) {
		items.push({ type: 'running', agent });
	}
	
	// Only show queued issues if there are running agents
	// (otherwise the tree below already shows the same info)
	if ($wip.runningAgents.length > 0) {
		for (const issue of $wip.queuedIssues) {
			items.push({ type: 'queued', issue });
		}
	}
	
	return items;
});

// Derived store for summary stats
export const wipStats = derived(wip, ($wip) => ({
	running: $wip.runningAgents.length,
	queued: $wip.queuedIssues.length,
	total: $wip.runningAgents.length + $wip.queuedIssues.length
}));

// Health status for running agents
export type HealthStatus = 'healthy' | 'warning' | 'critical';

export interface AgentHealth {
	status: HealthStatus;
	reasons: string[];
}

/**
 * Compute health status for an agent based on various signals
 */
export function computeAgentHealth(agent: Agent): AgentHealth {
	const reasons: string[] = [];
	let status: HealthStatus = 'healthy';

	// Critical: stalled for 15+ minutes (from API)
	if (agent.is_stalled) {
		status = 'critical';
		reasons.push('Stalled (same phase 15+ min)');
	}

	// Warning: high context usage (if we have token data)
	// Note: tokens field may not always be present
	
	// Warning: long runtime without phase change
	// Parse runtime like "5m 30s" or "1h 20m"
	const runtimeMinutes = parseRuntimeMinutes(agent.runtime);
	if (runtimeMinutes && runtimeMinutes > 30 && !agent.phase?.toLowerCase().includes('complete')) {
		if (status !== 'critical') status = 'warning';
		reasons.push(`Long runtime (${agent.runtime})`);
	}

	return { status, reasons };
}

/**
 * Parse runtime string to minutes (e.g., "5m 30s" -> 5, "1h 20m" -> 80)
 */
function parseRuntimeMinutes(runtime?: string): number | null {
	if (!runtime) return null;
	
	let minutes = 0;
	const hourMatch = runtime.match(/(\d+)h/);
	const minMatch = runtime.match(/(\d+)m/);
	
	if (hourMatch) minutes += parseInt(hourMatch[1]) * 60;
	if (minMatch) minutes += parseInt(minMatch[1]);
	
	return minutes || null;
}

/**
 * Get expressive status text for an agent
 * Shows what the agent is currently doing in human-readable form
 */
export function getExpressiveStatus(agent: Agent): string {
	// If we have current activity, use it
	if (agent.current_activity?.text) {
		const text = agent.current_activity.text;
		// Truncate long activity text
		if (text.length > 40) {
			return text.slice(0, 37) + '...';
		}
		return text;
	}

	// Fall back to phase-based status
	if (agent.is_processing) {
		return 'Thinking...';
	}

	// Use phase if available
	if (agent.phase) {
		switch (agent.phase.toLowerCase()) {
			case 'planning':
				return 'Planning approach...';
			case 'implementation':
			case 'implementing':
				return 'Writing code...';
			case 'validation':
			case 'validating':
				return 'Running tests...';
			case 'complete':
				return 'Ready for review';
			default:
				return agent.phase;
		}
	}

	// Default based on status
	if (agent.status === 'idle') {
		return 'Waiting for input...';
	}

	return 'Working...';
}
