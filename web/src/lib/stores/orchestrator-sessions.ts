import { writable, derived } from 'svelte/store';

// API configuration
const API_BASE = 'http://localhost:3348';

// Orchestrator session from /api/orchestrator-sessions
export interface OrchestratorSession {
	workspace_name: string;
	session_id?: string;
	goal: string;
	duration: string;
	duration_seconds: number;
	project: string;
	project_dir: string;
	status: string;
	spawn_time: string;
	child_agent_count: number;
}

// API response structure
interface OrchestratorSessionsResponse {
	sessions: OrchestratorSession[];
	count: number;
}

// Orchestrator sessions store
function createOrchestratorSessionsStore() {
	const { subscribe, set } = writable<OrchestratorSession[]>([]);

	return {
		subscribe,
		set,
		// Fetch orchestrator sessions from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/orchestrator-sessions`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data: OrchestratorSessionsResponse = await response.json();
				set(data.sessions || []);
			} catch (error) {
				console.error('Failed to fetch orchestrator sessions:', error);
				set([]);
			}
		}
	};
}

export const orchestratorSessions = createOrchestratorSessionsStore();

// Derived store for active orchestrator sessions only
export const activeOrchestratorSessions = derived(orchestratorSessions, ($sessions) =>
	$sessions.filter((s) => s.status === 'active')
);

// Helper to get project icon (uses same logic as agents)
export function getProjectIcon(project: string): string {
	const icons: Record<string, string> = {
		'orch-go': '🎯',
		'orch-knowledge': '📚',
		'skillc': '⚡',
		'beads': '📿',
		'kb-cli': '📖',
		'opencode': '💻',
	};
	return icons[project] || '📁';
}
