import { writable } from 'svelte/store';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Decision item from /api/decisions
export interface DecisionItem {
	id: string;
	beads_id?: string;
	title?: string;
	skill?: string;
	project?: string;
	escalation_level?: string;
	escalation_reason?: string;
	tldr?: string;
	recommendation?: string;
	has_web_changes?: boolean;
	next_actions?: string[];
	workspace_path?: string;
	investigation_path?: string;
	completed_at?: string;
}

// Decisions response from /api/decisions
export interface DecisionsState {
	absorb_knowledge: DecisionItem[];
	give_approvals: DecisionItem[];
	answer_questions: DecisionItem[];
	handle_failures: DecisionItem[];
	total_count: number;
	project_dir?: string;
	error?: string;
}

// Default empty state
const defaultState: DecisionsState = {
	absorb_knowledge: [],
	give_approvals: [],
	answer_questions: [],
	handle_failures: [],
	total_count: 0
};

// Decisions store
function createDecisionsStore() {
	const { subscribe, set } = writable<DecisionsState>(defaultState);

	return {
		subscribe,
		set,
		// Fetch decisions from orch-go API
		async fetch(projectDir?: string): Promise<void> {
			try {
				const url = projectDir
					? `${API_BASE}/api/decisions?project_dir=${encodeURIComponent(projectDir)}`
					: `${API_BASE}/api/decisions`;
				const response = await fetch(url);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch decisions:', error);
				set({
					...defaultState,
					error: error instanceof Error ? error.message : 'Unknown error'
				});
			}
		}
	};
}

export const decisions = createDecisionsStore();
