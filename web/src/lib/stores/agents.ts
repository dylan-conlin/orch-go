import { writable, derived } from 'svelte/store';

// Agent types matching orch-go registry
// 'dead' = session has no activity for >3 minutes (killed/crashed/stuck)
// 'stalled' = untracked agent with no phase comments after >1 minute
export type AgentState = 'active' | 'idle' | 'completed' | 'abandoned' | 'deleted' | 'dead' | 'stalled';

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

// Token usage stats from OpenCode session
export interface TokenStats {
	input_tokens: number;
	output_tokens: number;
	reasoning_tokens?: number;
	cache_read_tokens?: number;
	total_tokens: number; // input + output + reasoning
}

// LastActivity from API response (initial load)
export interface LastActivity {
	type: string; // "text", "tool", "reasoning", "step-start", "step-finish"
	text?: string; // Activity description
	timestamp?: number; // Unix timestamp in milliseconds
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
	gap_analysis?: GapAnalysis; // Context gap analysis from spawn time
	last_activity?: LastActivity; // Last activity from API (initial load)
	// Real-time activity tracking (from SSE, takes precedence over last_activity)
	current_activity?: {
		type: 'text' | 'tool' | 'reasoning' | 'step-start' | 'step-finish';
		text?: string;
		timestamp: number;
	};
	tokens?: TokenStats; // Token usage for the session
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

// API configuration
const API_BASE = 'http://localhost:3348';

// Fetch state management - prevents race conditions during rapid reloads
let currentFetchController: AbortController | null = null;
let fetchDebounceTimer: ReturnType<typeof setTimeout> | null = null;
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
		// Fetch agents from orch-go API with abort support
		async fetch(): Promise<void> {
			// If a fetch is already in progress, skip this request.
			// The in-progress fetch will complete and update the store.
			// This prevents race conditions where SSE events trigger fetchDebounced
			// while the initial fetch is still running (~800ms for 1800+ agents).
			if (currentFetchController) {
				return;
			}
			currentFetchController = new AbortController();
			
			try {
				const response = await fetch(`${API_BASE}/api/agents`, {
					signal: currentFetchController.signal
				});
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data || []);
			} catch (error) {
				// Don't log abort errors - they're expected during cleanup
				if (error instanceof Error && error.name === 'AbortError') {
					return;
				}
				console.error('Failed to fetch agents:', error);
				throw error;
			} finally {
				currentFetchController = null;
			}
		},
		// Debounced fetch - prevents multiple rapid fetches from SSE events
		fetchDebounced(): void {
			if (fetchDebounceTimer) {
				clearTimeout(fetchDebounceTimer);
			}
			fetchDebounceTimer = setTimeout(() => {
				fetchDebounceTimer = null;
				this.fetch().catch(console.error);
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
		}
	};
}

export const agents = createAgentStore();

// Derived stores for filtered views
// Working agents = agents actively doing work (active or idle status).
// This matches the CLI semantics where an agent is "working" if it:
// - Has an active OpenCode session (session_id exists)
// - Not completed (status !== 'completed')
// - Not dead (status !== 'dead' - no activity for >3 min)
// - Not stalled (status !== 'stalled' - untracked with no phase for >1 min)
//
// Note: The API sets status='idle' for sessions that aren't actively processing
// (isProcessing=false) because calling IsSessionProcessing per-session caused
// 125% CPU. However, the SSE stream updates is_processing in real-time via
// session.status events. Agents with status='idle' but is_processing=true
// should still show as working (they ARE processing, API just doesn't know yet).
//
// We include status='idle' agents because they have active sessions - they're
// "working" even if momentarily between tasks.
export const workingAgents = derived(agents, ($agents) =>
	$agents.filter((a) => a.status === 'active' || a.status === 'idle')
);

// Needs attention agents = agents that have gone dead or stalled.
// These need human intervention to resume or clean up.
// - 'dead' = session has no activity for >3 minutes (killed/crashed/stuck)
// - 'stalled' = untracked agent with no phase comments after >1 minute
export const needsAttentionAgents = derived(agents, ($agents) =>
	$agents.filter((a) => a.status === 'dead' || a.status === 'stalled')
);

// Active agents = combined working + needs attention (for backwards compatibility)
// This includes all agents that have sessions and aren't completed.
export const activeAgents = derived(agents, ($agents) =>
	$agents.filter((a) => a.status === 'active' || a.status === 'idle' || a.status === 'dead' || a.status === 'stalled')
);

// Note: idleAgents is now effectively a subset of activeAgents since both
// 'idle' and 'active' status agents are shown in the Active Agents section.
// This store is kept for backwards compatibility but may be deprecated.
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
// Active: status === 'active' OR status === 'idle' OR status === 'dead' OR status === 'stalled' (agents with sessions)
// Recent: completed within 24 hours (not in active section)
// Archive: completed older than 24 hours
export const recentAgents = derived(agents, ($agents) => {
	const now = Date.now();
	return $agents.filter((a) => {
		// Exclude agents shown in active section (active + idle + dead + stalled)
		if (a.status === 'active' || a.status === 'idle' || a.status === 'dead' || a.status === 'stalled' || a.status === 'deleted') return false;
		const updatedAt = a.updated_at ? new Date(a.updated_at).getTime() : 0;
		return now - updatedAt < RECENT_THRESHOLD_MS;
	});
});

export const archivedAgents = derived(agents, ($agents) => {
	const now = Date.now();
	return $agents.filter((a) => {
		// Exclude agents shown in active section (active + idle + dead + stalled)
		if (a.status === 'active' || a.status === 'idle' || a.status === 'dead' || a.status === 'stalled' || a.status === 'deleted') return false;
		const updatedAt = a.updated_at ? new Date(a.updated_at).getTime() : 0;
		return now - updatedAt >= RECENT_THRESHOLD_MS;
	});
});

// Token usage aggregation across active agents
// Returns total tokens from all active sessions for the stats bar display
export interface TotalTokens {
	total: number;
	input: number;
	output: number;
	agentCount: number; // Number of agents contributing to the total
}

export const totalTokens = derived(activeAgents, ($activeAgents): TotalTokens => {
	let total = 0;
	let input = 0;
	let output = 0;
	let agentCount = 0;

	for (const agent of $activeAgents) {
		if (agent.tokens) {
			total += agent.tokens.total_tokens || 0;
			input += agent.tokens.input_tokens || 0;
			output += agent.tokens.output_tokens || 0;
			agentCount++;
		}
	}

	return { total, input, output, agentCount };
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
				// Keep last 100 events
				return newEvents.slice(-100);
			});
		},
		// Legacy addEvent for backwards compatibility
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
// Connection generation counter - prevents stale reconnect timers from firing
let connectionGeneration = 0;

export function connectSSE(): void {
	// Increment generation to invalidate any pending reconnect timers
	const thisGeneration = ++connectionGeneration;
	
	// Clear any pending reconnect timer from previous connection
	if (reconnectTimeout) {
		clearTimeout(reconnectTimeout);
		reconnectTimeout = null;
	}
	
	if (eventSource) {
		eventSource.close();
		eventSource = null;
	}

	connectionStatus.set('connecting');

	eventSource = new EventSource(`${API_BASE}/api/events`);

	eventSource.onopen = () => {
		// Ignore if this connection is stale (newer connection started)
		if (thisGeneration !== connectionGeneration) {
			eventSource?.close();
			return;
		}
		connectionStatus.set('connected');
		// Fetch agents on reconnection to refresh state.
		// On initial load, this will skip since fetch was already triggered before SSE connect
		// (to avoid Chrome's 6-connection-per-host limit blocking the fetch).
		// On reconnection after disconnect, this ensures data is refreshed.
		agents.fetch().catch(console.error);
	};

	eventSource.onerror = () => {
		// Ignore if this connection is stale (newer connection started)
		if (thisGeneration !== connectionGeneration) {
			return;
		}
		
		// Don't log errors during page unload (expected behavior)
		connectionStatus.set('disconnected');
		eventSource?.close();
		eventSource = null;

		// Auto-reconnect after 5 seconds (unless page is unloading)
		// Use generation check to prevent stale timer from firing
		if (reconnectTimeout) {
			clearTimeout(reconnectTimeout);
		}
		reconnectTimeout = setTimeout(() => {
			// Only reconnect if no newer connection was started
			if (thisGeneration === connectionGeneration) {
				connectSSE();
			}
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
									is_processing: false,
									// Clear activity when idle (agent finished)
									current_activity: undefined
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

	// Handle session status changes - refresh agent list (debounced to prevent race conditions)
	const refreshEvents = [
		'session.status',
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

// Artifact types for the artifact viewer
export interface Artifact {
	type: 'synthesis' | 'investigation' | 'decision';
	content: string;
	path?: string;
	workspace_id?: string;
	error?: string;
}

// Issue detail types for the Issue tab
export interface IssueComment {
	id: number;
	author: string;
	text: string;
	created_at: string;
	// Parsed fields for timeline display
	is_phase?: boolean;    // True if this is a Phase: comment
	phase?: string;        // The phase name (e.g., "Planning", "Implementing", "Complete")
	is_blocked?: boolean;  // True if this is a BLOCKED: comment
	is_question?: boolean; // True if this is a QUESTION: comment
}

export interface IssueRelationship {
	id: string;
	title: string;
	status: string;
}

export interface IssueDetail {
	id: string;
	title: string;
	description?: string;
	status: string;
	priority: number;
	issue_type?: string;
	labels?: string[];
	close_reason?: string;
	created_at?: string;
	updated_at?: string;
	closed_at?: string;
	parent?: IssueRelationship;
	children?: IssueRelationship[];
	comments: IssueComment[];
	error?: string;
}

// Fetch issue details with comments timeline
export async function fetchIssueDetails(
	beadsId: string,
	projectDir?: string
): Promise<IssueDetail> {
	try {
		const params = new URLSearchParams({ id: beadsId });
		if (projectDir) {
			params.set('project', projectDir);
		}
		
		const response = await fetch(`${API_BASE}/api/beads/issue?${params}`);
		const data = await response.json();
		
		if (!response.ok || data.error) {
			return {
				id: beadsId,
				title: '',
				status: '',
				priority: 0,
				comments: [],
				error: data.error || `HTTP ${response.status}`,
			};
		}
		
		return data;
	} catch (error) {
		return {
			id: beadsId,
			title: '',
			status: '',
			priority: 0,
			comments: [],
			error: error instanceof Error ? error.message : 'Failed to fetch issue details',
		};
	}
}

// Fetch artifact content for an agent
export async function fetchArtifact(
	workspaceId: string, 
	artifactType: 'synthesis' | 'investigation' | 'decision',
	beadsId?: string
): Promise<Artifact> {
	try {
		const params = new URLSearchParams({
			workspace: workspaceId,
			type: artifactType,
		});
		if (beadsId) {
			params.set('beads_id', beadsId);
		}
		
		const response = await fetch(`${API_BASE}/api/agents/artifact?${params}`);
		const data = await response.json();
		
		if (!response.ok || data.error) {
			return {
				type: artifactType,
				content: '',
				workspace_id: workspaceId,
				error: data.error || `HTTP ${response.status}`,
			};
		}
		
		return {
			type: data.type || artifactType,
			content: data.content || '',
			path: data.path,
			workspace_id: data.workspace_id || workspaceId,
		};
	} catch (error) {
		return {
			type: artifactType,
			content: '',
			workspace_id: workspaceId,
			error: error instanceof Error ? error.message : 'Failed to fetch artifact',
		};
	}
}

// Deliverables types for the Deliverables tab
export interface DeliverableCommit {
	hash: string;
	message: string;
	author: string;
	timestamp: string;
	files_changed: number;
}

export interface FileDeltaSummary {
	created: string[];
	modified: string[];
	deleted: string[];
}

export interface ArtifactLink {
	type: 'synthesis' | 'investigation' | 'decision';
	path: string;
	name: string;
}

export interface Deliverables {
	workspace_id: string;
	commits: DeliverableCommit[];
	file_delta: FileDeltaSummary;
	artifacts: ArtifactLink[];
	error?: string;
}

// Fetch deliverables for an agent
export async function fetchDeliverables(
	workspaceId: string,
	spawnedAt?: string,
	projectDir?: string,
	beadsId?: string
): Promise<Deliverables> {
	try {
		const params = new URLSearchParams({ workspace: workspaceId });
		if (spawnedAt) {
			params.set('spawned_at', spawnedAt);
		}
		if (projectDir) {
			params.set('project_dir', projectDir);
		}
		if (beadsId) {
			params.set('beads_id', beadsId);
		}
		
		const response = await fetch(`${API_BASE}/api/agents/deliverables?${params}`);
		const data = await response.json();
		
		if (!response.ok || data.error) {
			return {
				workspace_id: workspaceId,
				commits: [],
				file_delta: { created: [], modified: [], deleted: [] },
				artifacts: [],
				error: data.error || `HTTP ${response.status}`,
			};
		}
		
		return data;
	} catch (error) {
		return {
			workspace_id: workspaceId,
			commits: [],
			file_delta: { created: [], modified: [], deleted: [] },
			artifacts: [],
			error: error instanceof Error ? error.message : 'Failed to fetch deliverables',
		};
	}
}

// Spawn context types for the Context tab
export interface SpawnMetadata {
	task?: string;          // Task description from TASK: line
	skill?: string;         // Skill name
	beads_id?: string;      // Beads issue ID
	project_dir?: string;   // PROJECT_DIR value
	spawn_tier?: string;    // SPAWN TIER: light/full
	session_scope?: string; // SESSION SCOPE value
}

export interface SpawnContext {
	content: string;           // Raw SPAWN_CONTEXT.md markdown content
	workspace_path?: string;   // Full path to workspace directory
	metadata: SpawnMetadata;   // Extracted spawn metadata
	error?: string;            // Error message if not found
}

// Fetch spawn context for an agent
export async function fetchSpawnContext(workspaceId: string): Promise<SpawnContext> {
	try {
		const params = new URLSearchParams({ workspace: workspaceId });
		
		const response = await fetch(`${API_BASE}/api/agents/spawn-context?${params}`);
		const data = await response.json();
		
		if (!response.ok || data.error) {
			return {
				content: '',
				metadata: {},
				error: data.error || `HTTP ${response.status}`,
			};
		}
		
		return data;
	} catch (error) {
		return {
			content: '',
			metadata: {},
			error: error instanceof Error ? error.message : 'Failed to fetch spawn context',
		};
	}
}

export function disconnectSSE(): void {
	// Increment generation to invalidate any pending reconnect timers
	connectionGeneration++;
	
	if (reconnectTimeout) {
		clearTimeout(reconnectTimeout);
		reconnectTimeout = null;
	}
	if (eventSource) {
		eventSource.close();
		eventSource = null;
	}
	// Cancel any pending fetch operations
	agents.cancelPending();
	// Clear all pending processing clear timers to prevent memory leaks
	processingClearTimers.forEach((timer) => clearTimeout(timer));
	processingClearTimers.clear();
	connectionStatus.set('disconnected');
}
