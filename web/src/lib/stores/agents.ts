import { writable, derived } from 'svelte/store';

// Agent types matching orch-go registry
export type AgentState = 'active' | 'completed' | 'abandoned' | 'deleted';

// Synthesis data from SYNTHESIS.md (D.E.K.N. format)
export interface Synthesis {
	tldr?: string;
	outcome?: string; // success, partial, blocked, failed
	recommendation?: string; // close, continue, escalate
	delta_summary?: string; // e.g., "3 files created, 2 modified, 5 commits"
	next_actions?: string[]; // Follow-up items
}

export interface Agent {
	id: string;
	session_id?: string;
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
	synthesis?: Synthesis; // Parsed SYNTHESIS.md for completed agents
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
	timestamp?: number;
}

// API configuration
const API_BASE = 'http://127.0.0.1:3333';

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
		},
		// Fetch agents from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/agents`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data || []);
			} catch (error) {
				console.error('Failed to fetch agents:', error);
				throw error;
			}
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

// SSE connection manager
let eventSource: EventSource | null = null;
let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;

export function connectSSE(): void {
	if (eventSource) {
		eventSource.close();
	}

	connectionStatus.set('connecting');

	eventSource = new EventSource(`${API_BASE}/api/events`);

	eventSource.onopen = () => {
		connectionStatus.set('connected');
		// Fetch agents on connection to get current state
		agents.fetch().catch(console.error);
	};

	eventSource.onerror = (error) => {
		console.error('SSE error:', error);
		connectionStatus.set('disconnected');
		eventSource?.close();
		eventSource = null;

		// Auto-reconnect after 5 seconds
		if (reconnectTimeout) {
			clearTimeout(reconnectTimeout);
		}
		reconnectTimeout = setTimeout(() => {
			connectSSE();
		}, 5000);
	};

	eventSource.onmessage = (event) => {
		try {
			const data = JSON.parse(event.data);
			handleSSEEvent(data);
		} catch (e) {
			// Non-JSON data, create simple event
			sseEvents.addEvent({
				type: 'raw',
				timestamp: Date.now()
			});
		}
	};

	// Handle specific event types if sent with event: prefix
	const eventTypes = ['session.status', 'session.created', 'session.deleted', 'agent.completed', 'agent.abandoned'];
	eventTypes.forEach((type) => {
		eventSource?.addEventListener(type, (event) => {
			try {
				const data = JSON.parse((event as MessageEvent).data);
				handleSSEEvent({ ...data, type });
			} catch (e) {
				console.error(`Failed to parse ${type} event:`, e);
			}
		});
	});

	// Handle custom events from our proxy
	eventSource.addEventListener('connected', (event) => {
		try {
			const data = JSON.parse((event as MessageEvent).data);
			sseEvents.addEvent({
				type: 'proxy.connected',
				timestamp: Date.now()
			});
			console.log('SSE proxy connected to:', data.source);
		} catch (e) {
			console.log('SSE connected');
		}
	});

	eventSource.addEventListener('disconnected', () => {
		connectionStatus.set('disconnected');
		sseEvents.addEvent({
			type: 'proxy.disconnected',
			timestamp: Date.now()
		});
	});

	eventSource.addEventListener('error', (event) => {
		try {
			const data = JSON.parse((event as MessageEvent).data);
			console.error('SSE proxy error:', data.error);
			sseEvents.addEvent({
				type: 'proxy.error',
				timestamp: Date.now()
			});
		} catch (e) {
			// Ignore parse errors for error events
		}
	});
}

function handleSSEEvent(data: any) {
	const sseEvent: SSEEvent = {
		type: data.type || 'unknown',
		properties: data.properties,
		timestamp: Date.now()
	};
	sseEvents.addEvent(sseEvent);

	// Handle session status changes - refresh agent list
	const refreshEvents = [
		'session.status',
		'session.created',
		'session.deleted',
		'agent.completed',
		'agent.abandoned'
	];
	if (refreshEvents.includes(data.type)) {
		agents.fetch().catch(console.error);
	}
}

export function disconnectSSE(): void {
	if (reconnectTimeout) {
		clearTimeout(reconnectTimeout);
		reconnectTimeout = null;
	}
	if (eventSource) {
		eventSource.close();
		eventSource = null;
	}
	connectionStatus.set('disconnected');
}
