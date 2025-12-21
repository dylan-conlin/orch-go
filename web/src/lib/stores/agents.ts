import { writable, derived } from 'svelte/store';

// Agent types matching orch-go registry
export type AgentState = 'active' | 'completed' | 'abandoned' | 'deleted';

export interface Agent {
	id: string;
	beads_id?: string;
	window_id?: string;
	window?: string;
	status: AgentState;
	spawned_at: string;
	updated_at: string;
	completed_at?: string;
	abandoned_at?: string;
	deleted_at?: string;
	project_dir?: string;
	skill?: string;
	primary_artifact?: string;
	is_interactive?: boolean;
}

// SSE Event types from OpenCode
export interface SSEEvent {
	type: string;
	properties?: {
		sessionID?: string;
		status?: {
			type: string;
		};
		message?: unknown;
	};
}

// Agent store
function createAgentStore() {
	const { subscribe, set, update } = writable<Agent[]>([]);

	return {
		subscribe,
		set,
		update,
		addAgent: (agent: Agent) => {
			update((agents) => [...agents, agent]);
		},
		updateAgent: (id: string, changes: Partial<Agent>) => {
			update((agents) =>
				agents.map((a) => (a.id === id ? { ...a, ...changes } : a))
			);
		},
		removeAgent: (id: string) => {
			update((agents) => agents.filter((a) => a.id !== id));
		}
	};
}

export const agents = createAgentStore();

// Derived stores for filtered views
export const activeAgents = derived(agents, ($agents) =>
	$agents.filter((a) => a.status === 'active')
);

export const completedAgents = derived(agents, ($agents) =>
	$agents.filter((a) => a.status === 'completed')
);

export const abandonedAgents = derived(agents, ($agents) =>
	$agents.filter((a) => a.status === 'abandoned')
);

// SSE event stream
function createSSEStore() {
	const { subscribe, update } = writable<SSEEvent[]>([]);

	return {
		subscribe,
		addEvent: (event: SSEEvent) => {
			update((events) => {
				const newEvents = [...events, event];
				// Keep last 100 events
				return newEvents.slice(-100);
			});
		},
		clear: () => {
			update(() => []);
		}
	};
}

export const sseEvents = createSSEStore();

// Connection status
export const connectionStatus = writable<'connected' | 'disconnected' | 'connecting'>('disconnected');
