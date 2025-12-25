import { writable, derived } from 'svelte/store';

// Agent types matching orch-go registry
export type AgentState = 'active' | 'idle' | 'completed' | 'abandoned' | 'deleted';

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
	beads_title?: string;
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
	// New fields from enhanced API
	phase?: string; // "Planning", "Implementing", "Complete", etc.
	task?: string; // Task description from beads issue
	project?: string; // Project name (orch-go, skillc, etc.)
	runtime?: string; // Formatted duration
	is_processing?: boolean; // True if actively generating response
	synthesis?: Synthesis; // Parsed SYNTHESIS.md for completed agents
	close_reason?: string; // Beads close reason, fallback for completed agents without synthesis
	// Real-time activity tracking
	current_activity?: {
		type: 'text' | 'tool' | 'reasoning' | 'step-start' | 'step-finish';
		text?: string;
		timestamp: number;
	};
}

// SSE Event types from OpenCode
export interface SSEEvent {
	id: string; // Unique ID for keyed rendering
	type: string;
	properties?: {
		sessionID?: string;
		status?: {
			type: string;
		};
		message?: unknown;
		part?: {
			type: string;
			text?: string;
			tool?: string;
			function?: string;
			state?: {
				title?: string;
				status?: string;
				input?: unknown;
				output?: string;
			};
		};
	};
	timestamp?: number;
}

// Counter for generating unique event IDs
let sseEventIdCounter = 0;

// Generate a unique SSE event ID
function generateSSEEventId(): string {
	return `sse-${Date.now()}-${++sseEventIdCounter}`;
}

// API configuration
const API_BASE = 'http://127.0.0.1:3348';

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

export const idleAgents = derived(agents, ($agents) =>
	$agents.filter((a) => a.status === 'idle')
);

export const completedAgents = derived(agents, ($agents) =>
	$agents.filter((a) => a.status === 'completed')
);

export const abandonedAgents = derived(agents, ($agents) =>
	$agents.filter((a) => a.status === 'abandoned')
);

// Time-based thresholds for progressive disclosure
const RECENT_THRESHOLD_MS = 24 * 60 * 60 * 1000; // 24 hours

// Progressive disclosure groups
// Active: status === 'active' (agents actively processing)
// Recent: idle/completed within 24 hours
// Archive: idle/completed older than 24 hours
export const recentAgents = derived(agents, ($agents) => {
	const now = Date.now();
	return $agents.filter((a) => {
		if (a.status === 'active' || a.status === 'deleted') return false;
		const updatedAt = a.updated_at ? new Date(a.updated_at).getTime() : 0;
		return now - updatedAt < RECENT_THRESHOLD_MS;
	});
});

export const archivedAgents = derived(agents, ($agents) => {
	const now = Date.now();
	return $agents.filter((a) => {
		if (a.status === 'active' || a.status === 'deleted') return false;
		const updatedAt = a.updated_at ? new Date(a.updated_at).getTime() : 0;
		return now - updatedAt >= RECENT_THRESHOLD_MS;
	});
});

// SSE event stream
function createSSEStore() {
	const { subscribe, update } = writable<SSEEvent[]>([]);

	return {
		subscribe,
		addEvent: (event: Omit<SSEEvent, 'id'>) => {
			update((events) => {
				const newEvents = [...events, { ...event, id: generateSSEEventId() }];
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

// Selected agent for detail panel
export const selectedAgentId = writable<string | null>(null);

// Derived store for the selected agent
export const selectedAgent = derived([agents, selectedAgentId], ([$agents, $selectedAgentId]) => {
	if (!$selectedAgentId) return null;
	return $agents.find((a) => a.id === $selectedAgentId) || null;
});

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

	eventSource.onerror = () => {
		// Don't log errors during page unload (expected behavior)
		connectionStatus.set('disconnected');
		eventSource?.close();
		eventSource = null;

		// Auto-reconnect after 5 seconds (unless page is unloading)
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
	sseEvents.addEvent({
		type: data.type || 'unknown',
		properties: data.properties,
		timestamp: Date.now()
	});

	// Handle message.part events - update agent activity and processing state in real-time
	if (data.type === 'message.part' && data.properties) {
		const sessionID = data.properties.sessionID;
		const part = data.properties.part;
		
		if (sessionID && part && part.type) {
			// Update agent activity and set is_processing=true (agent is actively responding)
			agents.update((agentList) => {
				return agentList.map((agent) => {
					// Match by session_id
					if (agent.session_id === sessionID) {
						return {
							...agent,
							is_processing: true, // Agent is actively generating response
							current_activity: {
								type: part.type,
								text: part.text || extractActivityText(part),
								timestamp: Date.now()
							}
						};
					}
					return agent;
				});
			});
		}
	}

	// Handle session.status events - update is_processing based on busy/idle state
	if (data.type === 'session.status' && data.properties) {
		const sessionID = data.properties.sessionID;
		const statusType = data.properties.status?.type;
		
		if (sessionID && statusType) {
			// Update is_processing based on status type
			// "busy" = processing, "idle" = not processing
			const isProcessing = statusType === 'busy';
			
			agents.update((agentList) => {
				return agentList.map((agent) => {
					if (agent.session_id === sessionID) {
						return {
							...agent,
							is_processing: isProcessing,
							// Clear activity when idle (agent finished)
							current_activity: isProcessing ? agent.current_activity : undefined
						};
					}
					return agent;
				});
			});
		}
	}

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

// Extract displayable text from message part
function extractActivityText(part: any): string {
	// For tool invocations, show tool name and function
	if (part.type === 'tool-invocation' || part.type === 'tool') {
		if (part.tool) {
			return `Using ${part.tool}${part.function ? `.${part.function}` : ''}`;
		}
		return 'Using tool';
	}
	
	// For step-start/finish, show step info
	if (part.type === 'step-start') {
		return 'Starting step...';
	}
	if (part.type === 'step-finish') {
		return 'Completed step';
	}
	
	// For reasoning, show preview
	if (part.type === 'reasoning' && part.text) {
		return part.text.substring(0, 100);
	}
	
	// Default: show part type
	return part.type.replace(/-/g, ' ');
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
