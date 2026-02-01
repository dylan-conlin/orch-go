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
