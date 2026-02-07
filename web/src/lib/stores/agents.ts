import { writable, derived } from 'svelte/store';
import { createSSEConnection, type SSEConnection } from '../services/sse-connection';

// Agent types matching orch-go registry
// 'dead' = no activity for 3+ minutes (crashed/stuck/killed) - needs investigation
// 'awaiting-cleanup' = completed but not closed via orch complete - needs cleanup
export type AgentState = 'active' | 'idle' | 'completed' | 'abandoned' | 'deleted' | 'dead' | 'awaiting-cleanup';

// Synthesis data from SYNTHESIS.md (D.E.K.N. format)
export interface Synthesis {
	tldr?: string;
	outcome?: string; // success, partial, blocked, failed
	recommendation?: string; // close, continue, escalate
	delta_summary?: string; // e.g., "3 files created, 2 modified, 5 commits"
	next_actions?: string[]; // Follow-up items
}

// Gap analysis data from spawn time (context quality)
export interface GapAnalysis {
	has_gaps: boolean;
	context_quality: number; // 0-100
	should_warn: boolean;
	match_count?: number;
	constraints?: number;
	decisions?: number;
	investigations?: number;
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
	is_stale?: boolean; // True if agent is older than beadsFetchThreshold (beads data not fetched)
	is_stalled?: boolean; // True if active agent has same phase for 15+ minutes (advisory)
	synthesis?: Synthesis; // Parsed SYNTHESIS.md for completed agents
	close_reason?: string; // Beads close reason, fallback for completed agents without synthesis
	gap_analysis?: GapAnalysis; // Context gap analysis from spawn time
	investigation_path?: string; // Path to investigation file from beads comments
	synthesis_content?: string; // Raw SYNTHESIS.md content for inline rendering
	investigation_content?: string; // Raw investigation file content for inline rendering
	// Real-time activity tracking
	current_activity?: {
		type: 'text' | 'tool' | 'reasoning' | 'step-start' | 'step-finish';
		text?: string;
		timestamp: number;
	};
}

// Display state for agent cards - derived from agent status + phase + activity
// Provides clearer visual distinction between different agent states
export type DisplayState = 'running' | 'ready-for-review' | 'idle' | 'waiting' | 'completed' | 'abandoned' | 'dead' | 'awaiting-cleanup';

/**
 * Compute the display state from agent status + phase + activity
 * This provides clearer visual distinction between:
 * - running: actively processing (is_processing=true)
 * - ready-for-review: phase=Complete but status still active
 * - idle: no activity for a while (60+ seconds)
 * - waiting: active but no activity yet
 * - completed: agent status is completed
 * - abandoned: agent status is abandoned
 */
export function computeDisplayState(agent: Agent): DisplayState {
	if (agent.status === 'completed') return 'completed';
	if (agent.status === 'abandoned') return 'abandoned';
	if (agent.status === 'dead') return 'dead';
	if (agent.status === 'awaiting-cleanup') return 'awaiting-cleanup';
	
	if (agent.status === 'active') {
		// Phase: Complete means agent reported done, waiting for orchestrator to close
		if (agent.phase?.toLowerCase() === 'complete') {
			return 'ready-for-review';
		}
		
		// Actively processing
		if (agent.is_processing) {
			return 'running';
		}
		
		// Check if idle for too long (no activity in 60+ seconds)
		if (agent.current_activity?.timestamp) {
			const idleMs = Date.now() - agent.current_activity.timestamp;
			if (idleMs > 60000) {
				return 'idle';
			}
		}
		
		return 'waiting';
	}
	
	return 'waiting';
}

// SSE Event types from OpenCode
export interface SSEEvent {
	id: string; // Unique ID for keyed rendering
	type: string;
	properties?: {
		sessionID?: string; // Present on session.* events
		status?: {
			type: string;
		};
		message?: unknown;
		part?: {
			type: string;
			text?: string;
			tool?: string;
			function?: string;
			sessionID?: string; // Present on message.part events
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

// Counter for generating unique event IDs (fallback for events without part.id)
let sseEventIdCounter = 0;

// Generate a unique SSE event ID
function generateSSEEventId(): string {
	return `sse-${Date.now()}-${++sseEventIdCounter}`;
}

// Extract stable ID from SSE event data
// For message.part and message.part.updated events, use the part.id for deduplication
function extractEventId(data: any): string | null {
	if (data?.properties?.part?.id) {
		return data.properties.part.id;
	}
	return null;
}

// API configuration - HTTPS for HTTP/2 multiplexing (fixes connection pool exhaustion)
const API_BASE = 'https://localhost:3348';

// Fetch state management - prevents race conditions during rapid reloads
let currentFetchController: AbortController | null = null;
let fetchDebounceTimer: ReturnType<typeof setTimeout> | null = null;
// Tracks if a fetch is currently in-flight. When true, new fetchDebounced() calls
// set needsRefetch flag instead of starting new requests (prevents request storm).
let isFetching = false;
// When true, a fetch was requested while one was in-flight. After the current
// fetch completes, we'll trigger another debounced fetch to get the latest data.
let needsRefetch = false;
// Debounce interval for SSE-triggered refetches. 500ms strikes a balance between:
// - Responsiveness: User sees updates within 500ms (imperceptible delay)
// - Performance: Collapses rapid SSE events into single request
// - CPU: With 3 agents sending events, 500ms debounce reduces refetches by ~80%
const FETCH_DEBOUNCE_MS = 500;

// Processing state debounce - tracks pending "idle" state transitions per session
// When an agent goes idle, we delay the visual update to prevent rapid flapping
// between busy/idle states (which causes gold border flashing)
const processingClearTimers: Map<string, ReturnType<typeof setTimeout>> = new Map();
// Delay before clearing is_processing state. 5000ms (5 seconds) keeps the gold
// "processing" indicator visible between rapid tool calls, preventing the visually
// distracting flashing that occurs with shorter delays. The agent typically switches
// between busy/idle states every few hundred milliseconds during active work.
const PROCESSING_CLEAR_DELAY_MS = 5000;

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
		// Fetch agents from orch-go API with in-flight tracking
		// Only one fetch runs at a time. If called while fetching, queues a re-fetch
		// after the current one completes (prevents request storm from SSE events).
		// Optional queryString parameter for time/project filtering (e.g., "?since=12h&project=orch-go")
		async fetch(queryString: string = ''): Promise<void> {
			// If already fetching, mark that we need another fetch after this one
			if (isFetching) {
				needsRefetch = true;
				return;
			}
			
			isFetching = true;
			needsRefetch = false;
			currentFetchController = new AbortController();
			
			try {
				const response = await fetch(`${API_BASE}/api/agents${queryString}`, {
					signal: currentFetchController.signal
				});
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				// Transform API response: current_activity comes as string from backend,
				// but frontend expects object {type, text, timestamp}
				const transformed = (data || []).map((agent: Agent & { current_activity?: string | Agent['current_activity'], last_activity_at?: string }) => {
					if (typeof agent.current_activity === 'string' && agent.current_activity) {
						return {
							...agent,
							current_activity: {
								type: 'text' as const,
								text: agent.current_activity,
								timestamp: agent.last_activity_at ? new Date(agent.last_activity_at).getTime() : Date.now()
							}
						};
					}
					return agent;
				});
				set(transformed);
			} catch (error) {
				// Don't log abort errors - they're expected during cleanup
				if (error instanceof Error && error.name === 'AbortError') {
					return;
				}
				console.error('Failed to fetch agents:', error);
				throw error;
			} finally {
				currentFetchController = null;
				isFetching = false;
				
				// If events arrived while we were fetching, do another fetch
				// This ensures we don't miss updates without creating request storms
				if (needsRefetch) {
					needsRefetch = false;
					this.fetchDebounced();
				}
			}
		},
		// Debounced fetch - prevents multiple rapid fetches from SSE events
		fetchDebounced(): void {
			if (fetchDebounceTimer) {
				clearTimeout(fetchDebounceTimer);
			}
			fetchDebounceTimer = setTimeout(() => {
				fetchDebounceTimer = null;
				// Use filter query string if available
				const queryString = getFilterQueryString ? getFilterQueryString() : '';
				this.fetch(queryString).catch(console.error);
			}, FETCH_DEBOUNCE_MS);
		},
		// Cancel pending operations - call on disconnect
		cancelPending(): void {
			if (currentFetchController) {
				currentFetchController.abort();
				currentFetchController = null;
			}
			if (fetchDebounceTimer) {
				clearTimeout(fetchDebounceTimer);
				fetchDebounceTimer = null;
			}
			// Reset in-flight tracking state
			isFetching = false;
			needsRefetch = false;
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

// Dead agents: no activity for 3+ minutes (crashed/stuck/killed)
// These need immediate attention - surfaced in "Needs Attention" section
export const deadAgents = derived(agents, ($agents) =>
	$agents.filter((a) => a.status === 'dead')
);

// Awaiting cleanup: completed work but needs orch complete to close
// These are less urgent than dead agents - agent did its job, just needs cleanup
export const awaitingCleanupAgents = derived(agents, ($agents) =>
	$agents.filter((a) => a.status === 'awaiting-cleanup')
);

// Stalled agents: active with same phase for 15+ minutes (may be stuck)
// Advisory only - surfaced in "Needs Attention" section with orange indicator
// See .kb/investigations/2026-01-08-inv-design-stalled-agent-detection-agents.md
export const stalledAgents = derived(agents, ($agents) =>
	$agents.filter((a) => a.status === 'active' && a.is_stalled === true)
);

// Needs Review: agents at Phase: Complete that haven't been closed yet
// These are waiting for orchestrator to run `orch complete`
export const needsReviewAgents = derived(agents, ($agents) =>
	$agents.filter((a) => 
		a.status === 'active' && 
		a.phase?.toLowerCase() === 'complete'
	)
);

// Truly active: running agents that are NOT in needs-review state
// These are the agents consuming capacity
export const trulyActiveAgents = derived(agents, ($agents) =>
	$agents.filter((a) => 
		a.status === 'active' && 
		a.phase?.toLowerCase() !== 'complete'
	)
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

// SSE event stream with deduplication
// Events with the same part.id are updated in place rather than added as duplicates
function createSSEStore() {
	const { subscribe, update } = writable<SSEEvent[]>([]);

	return {
		subscribe,
		// Add or update an event - data is the raw parsed SSE event from OpenCode
		addOrUpdateEvent: (data: any) => {
			// Extract stable ID from part.id if available (for message.part events)
			const partId = extractEventId(data);
			
			update((events) => {
				const eventWithId: SSEEvent = {
					id: partId || generateSSEEventId(),
					type: data.type || 'unknown',
					properties: data.properties,
					timestamp: Date.now()
				};
				
				// If we have a stable part.id, try to update existing event
				if (partId) {
					const existingIndex = events.findIndex(e => e.id === partId);
					if (existingIndex !== -1) {
						// Update in place - this replaces the old version with new data
						const newEvents = [...events];
						newEvents[existingIndex] = eventWithId;
						return newEvents;
					}
				}
				
				// New event - add to list
				const newEvents = [...events, eventWithId];
				// Keep last 1000 events (global across all agents)
				return newEvents.slice(-1000);
			});
		},
		// Legacy addEvent for backwards compatibility
		addEvent: (event: Omit<SSEEvent, 'id'>) => {
			update((events) => {
				const newEvents = [...events, { ...event, id: generateSSEEventId() }];
				// Keep last 1000 events (global across all agents)
				return newEvents.slice(-1000);
			});
		},
		clear: () => {
			update(() => []);
		}
	};
}

export const sseEvents = createSSEStore();

// Per-session event history store for hybrid SSE + API architecture.
// Stores historical events fetched from API separately from real-time SSE events.
// This enables:
// 1. Complete session history on-demand (survives page refresh)
// 2. Per-agent history without global buffer dilution
// 3. Seamless merge with real-time SSE events
interface SessionHistory {
	events: SSEEvent[];
	loading: boolean;
	loaded: boolean;
	error: string | null;
}

function createSessionHistoryStore() {
	const { subscribe, update } = writable<Map<string, SessionHistory>>(new Map());
	
	return {
		subscribe,
		// Fetch historical events for a session from the orch-go API
		async fetchHistory(sessionId: string): Promise<SSEEvent[]> {
			// Check if already loaded or loading
			let currentState: Map<string, SessionHistory> = new Map();
			update(state => {
				currentState = state;
				return state;
			});
			
			const existing = currentState.get(sessionId);
			if (existing?.loaded || existing?.loading) {
				return existing.events;
			}
			
			// Mark as loading
			update(state => {
				const newState = new Map(state);
				newState.set(sessionId, { events: [], loading: true, loaded: false, error: null });
				return newState;
			});
			
			try {
				const response = await fetch(`${API_BASE}/api/session/${sessionId}/messages`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const events: SSEEvent[] = await response.json();
				
				// Store the fetched events
				update(state => {
					const newState = new Map(state);
					newState.set(sessionId, { events, loading: false, loaded: true, error: null });
					return newState;
				});
				
				return events;
			} catch (error) {
				const errorMsg = error instanceof Error ? error.message : 'Unknown error';
				update(state => {
					const newState = new Map(state);
					newState.set(sessionId, { events: [], loading: false, loaded: false, error: errorMsg });
					return newState;
				});
				console.error('Failed to fetch session history:', error);
				return [];
			}
		},
		// Get current state for a session
		getState(sessionId: string): SessionHistory | undefined {
			let state: SessionHistory | undefined;
			update(current => {
				state = current.get(sessionId);
				return current;
			});
			return state;
		},
		// Clear history for a session (useful when session is deleted)
		clearSession(sessionId: string): void {
			update(state => {
				const newState = new Map(state);
				newState.delete(sessionId);
				return newState;
			});
		},
		// Clear all history
		clear(): void {
			update(() => new Map());
		}
	};
}

export const sessionHistory = createSessionHistoryStore();

// Connection status
export const connectionStatus = writable<'connected' | 'disconnected' | 'connecting'>('disconnected');

// Selected agent for detail panel
export const selectedAgentId = writable<string | null>(null);

// Derived store for the selected agent
export const selectedAgent = derived([agents, selectedAgentId], ([$agents, $selectedAgentId]) => {
	if (!$selectedAgentId) return null;
	return $agents.find((a) => a.id === $selectedAgentId) || null;
});

// SSE connection manager - uses shared service for connection lifecycle
let sseConnection: SSEConnection | null = null;

// Build event listeners for the SSE connection
function buildSSEEventListeners(): Record<string, (event: MessageEvent) => void> {
	const listeners: Record<string, (event: MessageEvent) => void> = {};

	// Handle specific event types if sent with event: prefix
	const eventTypes = ['session.status', 'session.created', 'session.deleted', 'agent.completed', 'agent.abandoned'];
	eventTypes.forEach((type) => {
		listeners[type] = (event: MessageEvent) => {
			try {
				const data = JSON.parse(event.data);
				handleSSEEvent({ ...data, type });
			} catch (e) {
				console.error(`Failed to parse ${type} event:`, e);
			}
		};
	});

	// Handle custom events from our proxy
	listeners['connected'] = (event: MessageEvent) => {
		try {
			const data = JSON.parse(event.data);
			sseEvents.addEvent({
				type: 'proxy.connected',
				timestamp: Date.now()
			});
			console.log('SSE proxy connected to:', data.source);
		} catch (e) {
			console.log('SSE connected');
		}
	};

	listeners['disconnected'] = () => {
		connectionStatus.set('disconnected');
		sseEvents.addEvent({
			type: 'proxy.disconnected',
			timestamp: Date.now()
		});
	};

	listeners['error'] = (event: MessageEvent) => {
		try {
			const data = JSON.parse(event.data);
			console.error('SSE proxy error:', data.error);
			sseEvents.addEvent({
				type: 'proxy.error',
				timestamp: Date.now()
			});
		} catch (e) {
			// Ignore parse errors for error events
		}
	};

	return listeners;
}

// Callback to get current filter query string for fetches
// Set by the page component to provide dynamic filter state
let getFilterQueryString: (() => string) | null = null;

export function setFilterQueryStringCallback(callback: () => string): void {
	getFilterQueryString = callback;
}

export function connectSSE(): void {
	// Create connection if not exists
	if (!sseConnection) {
		sseConnection = createSSEConnection(`${API_BASE}/api/events`, {
			onOpen: () => {
				connectionStatus.set('connected');
				// Fetch agents on connection to get current state (with filters if available)
				const queryString = getFilterQueryString ? getFilterQueryString() : '';
				agents.fetch(queryString).catch(console.error);
			},
			onMessage: (event: MessageEvent) => {
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
			},
			onDisconnect: () => {
				connectionStatus.set('disconnected');
			},
			eventListeners: buildSSEEventListeners(),
			reconnectDelayMs: 5000,
			autoReconnect: true
		});

		// Sync the connection status from the service to our local store
		sseConnection.status.subscribe((status) => {
			connectionStatus.set(status);
		});
	}

	// Mark as connecting and initiate connection
	connectionStatus.set('connecting');
	sseConnection.connect();
}

function handleSSEEvent(data: any) {
	// Use addOrUpdateEvent for proper deduplication of streaming events
	// Events with the same part.id will update in place rather than duplicate
	sseEvents.addOrUpdateEvent(data);

	// Handle message.part and message.part.updated events - update agent activity in real-time
	// message.part: text streaming (incremental text chunks)
	// message.part.updated: tool state changes (running -> completed)
	if ((data.type === 'message.part' || data.type === 'message.part.updated') && data.properties) {
		const part = data.properties.part;
		// sessionID is nested inside the part object
		const sessionID = part?.sessionID;
		
		if (sessionID && part && part.type) {
			// Update agent activity and set is_processing=true (agent is actively responding)
			// Only update active agents to prevent stale pulsing on completed agents
			// (late SSE events can arrive after agent status changes to completed)
			agents.update((agentList) => {
				return agentList.map((agent) => {
					// Match by session_id AND only update active agents
					if (agent.session_id === sessionID && agent.status === 'active') {
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
	// Uses debounced clear to prevent rapid flapping between busy/idle states
	if (data.type === 'session.status' && data.properties) {
		const sessionID = data.properties.sessionID;
		const statusType = data.properties.status?.type;
		
		if (sessionID && statusType) {
			const isProcessing = statusType === 'busy';
			
			if (isProcessing) {
				// Setting to busy: immediate, and cancel any pending clear timer
				const pendingClear = processingClearTimers.get(sessionID);
				if (pendingClear) {
					clearTimeout(pendingClear);
					processingClearTimers.delete(sessionID);
				}
				
				agents.update((agentList) => {
					return agentList.map((agent) => {
						if (agent.session_id === sessionID && agent.status === 'active') {
							return { ...agent, is_processing: true };
						}
						return agent;
					});
				});
			} else {
				// Setting to idle: debounced to prevent rapid flapping
				// Cancel any existing timer first
				const existingTimer = processingClearTimers.get(sessionID);
				if (existingTimer) {
					clearTimeout(existingTimer);
				}
				
				// Set new timer for delayed clear
				const timer = setTimeout(() => {
					processingClearTimers.delete(sessionID);
					agents.update((agentList) => {
						return agentList.map((agent) => {
							if (agent.session_id === sessionID) {
								return {
									...agent,
									is_processing: false
									// Note: Don't clear current_activity here - the last activity
									// should persist so users can see what the agent was doing.
									// Clearing it causes "Starting up..." to show for completed agents.
								};
							}
							return agent;
						});
					});
				}, PROCESSING_CLEAR_DELAY_MS);
				
				processingClearTimers.set(sessionID, timer);
			}
		}
	}

	// Handle lifecycle events - refresh agent list (debounced to prevent race conditions)
	// Note: session.status is NOT included because it's already handled via local state
	// updates above (lines 510-562). Including it would cause redundant fetches since
	// session.status fires on every busy/idle toggle (high frequency).
	const refreshEvents = [
		'session.created',
		'session.deleted',
		'agent.completed',
		'agent.abandoned'
	];
	if (refreshEvents.includes(data.type)) {
		agents.fetchDebounced();
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

// Create a new beads issue (for follow-ups from synthesis recommendations)
export async function createIssue(title: string, description?: string, labels?: string[]): Promise<{ id: string; title: string } | null> {
	try {
		const response = await fetch(`${API_BASE}/api/issues`, {
			method: 'POST',
			headers: {
				'Content-Type': 'application/json',
			},
			body: JSON.stringify({
				title,
				description,
				labels: labels || ['triage:ready'],
				issue_type: 'task',
			}),
		});
		
		if (!response.ok) {
			const errorData = await response.json();
			throw new Error(errorData.error || `HTTP ${response.status}`);
		}
		
		const data = await response.json();
		if (data.success) {
			return { id: data.id, title: data.title };
		}
		throw new Error(data.error || 'Unknown error');
	} catch (error) {
		console.error('Failed to create issue:', error);
		throw error;
	}
}

export function disconnectSSE(): void {
	if (sseConnection) {
		sseConnection.disconnect();
	}
	// Cancel any pending fetch operations
	agents.cancelPending();
	// Clear all pending processing clear timers to prevent memory leaks
	processingClearTimers.forEach((timer) => clearTimeout(timer));
	processingClearTimers.clear();
	connectionStatus.set('disconnected');
}
