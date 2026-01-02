import { writable, derived } from 'svelte/store';

// Agent lifecycle event from events.jsonl
export interface AgentLogEvent {
	id: string; // Unique ID for keyed rendering
	type: string; // session.spawned, session.completed, session.error, session.status
	session_id?: string;
	timestamp: number; // Unix timestamp
	data?: {
		prompt?: string;
		title?: string;
		error?: string;
		status?: string;
	};
}

// Counter for generating unique event IDs
let eventIdCounter = 0;

// Generate a unique event ID
function generateEventId(): string {
	return `evt-${Date.now()}-${++eventIdCounter}`;
}

// API configuration
const API_BASE = 'http://127.0.0.1:3348';

// Fetch state management - prevents race conditions during rapid reloads
let currentFetchController: AbortController | null = null;

// Agentlog store
function createAgentlogStore() {
	const { subscribe, set, update } = writable<AgentLogEvent[]>([]);

	return {
		subscribe,
		set,
		update,
		addEvent: (event: Omit<AgentLogEvent, 'id'>) => {
			update((events) => {
				const newEvents = [...events, { ...event, id: generateEventId() }];
				// Keep last 100 events
				return newEvents.slice(-100);
			});
		},
		clear: () => {
			set([]);
		},
		// Fetch initial events from orch-go API with abort support
		async fetch(): Promise<void> {
			// Cancel any in-flight request to prevent race conditions
			if (currentFetchController) {
				currentFetchController.abort();
			}
			currentFetchController = new AbortController();
			
			try {
				const response = await fetch(`${API_BASE}/api/agentlog`, {
					signal: currentFetchController.signal
				});
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				// Assign unique IDs to fetched events
				const eventsWithIds = (data || []).map((e: Omit<AgentLogEvent, 'id'>) => ({
					...e,
					id: generateEventId()
				}));
				set(eventsWithIds);
			} catch (error) {
				// Don't log abort errors - they're expected during cleanup
				if (error instanceof Error && error.name === 'AbortError') {
					return;
				}
				console.error('Failed to fetch agentlog:', error);
				throw error;
			} finally {
				currentFetchController = null;
			}
		},
		// Cancel pending operations - call on disconnect
		cancelPending(): void {
			if (currentFetchController) {
				currentFetchController.abort();
				currentFetchController = null;
			}
		}
	};
}

export const agentlogEvents = createAgentlogStore();

// Derived stores for filtered views
export const spawnedEvents = derived(agentlogEvents, ($events) =>
	$events.filter((e) => e.type === 'session.spawned')
);

export const completedEvents = derived(agentlogEvents, ($events) =>
	$events.filter((e) => e.type === 'session.completed')
);

export const errorEvents = derived(agentlogEvents, ($events) =>
	$events.filter((e) => e.type === 'session.error')
);

// Agentlog SSE connection status
export const agentlogConnectionStatus = writable<'connected' | 'disconnected' | 'connecting'>('disconnected');

// Agentlog SSE connection manager
let agentlogEventSource: EventSource | null = null;
let agentlogReconnectTimeout: ReturnType<typeof setTimeout> | null = null;
// Connection generation counter - prevents stale reconnect timers from firing
let connectionGeneration = 0;

export function connectAgentlogSSE(): void {
	// Increment generation to invalidate any pending reconnect timers
	const thisGeneration = ++connectionGeneration;
	
	// Clear any pending reconnect timer from previous connection
	if (agentlogReconnectTimeout) {
		clearTimeout(agentlogReconnectTimeout);
		agentlogReconnectTimeout = null;
	}
	
	if (agentlogEventSource) {
		agentlogEventSource.close();
		agentlogEventSource = null;
	}

	agentlogConnectionStatus.set('connecting');

	agentlogEventSource = new EventSource(`${API_BASE}/api/agentlog?follow=true`);

	agentlogEventSource.onopen = () => {
		// Ignore if this connection is stale (newer connection started)
		if (thisGeneration !== connectionGeneration) {
			agentlogEventSource?.close();
			return;
		}
		agentlogConnectionStatus.set('connected');
		// Fetch initial events on connection
		agentlogEvents.fetch().catch(console.error);
	};

	agentlogEventSource.onerror = () => {
		// Ignore if this connection is stale (newer connection started)
		if (thisGeneration !== connectionGeneration) {
			return;
		}
		
		// Don't log errors during page unload (expected behavior)
		agentlogConnectionStatus.set('disconnected');
		agentlogEventSource?.close();
		agentlogEventSource = null;

		// Auto-reconnect after 5 seconds
		// Use generation check to prevent stale timer from firing
		if (agentlogReconnectTimeout) {
			clearTimeout(agentlogReconnectTimeout);
		}
		agentlogReconnectTimeout = setTimeout(() => {
			// Only reconnect if no newer connection was started
			if (thisGeneration === connectionGeneration) {
				connectAgentlogSSE();
			}
		}, 5000);
	};

	// Handle agentlog events
	agentlogEventSource.addEventListener('agentlog', (event) => {
		try {
			const data = JSON.parse((event as MessageEvent).data);
			agentlogEvents.addEvent(data);

			// Trigger agent list refresh on relevant events (debounced to prevent race conditions)
			import('./agents').then(({ agents }) => {
				agents.fetchDebounced();
			});
		} catch (e) {
			console.error('Failed to parse agentlog event:', e);
		}
	});

	// Handle connected event
	agentlogEventSource.addEventListener('connected', (event) => {
		try {
			const data = JSON.parse((event as MessageEvent).data);
			console.log('Agentlog SSE connected to:', data.source);
		} catch (e) {
			console.log('Agentlog SSE connected');
		}
	});

	// Handle error event
	agentlogEventSource.addEventListener('error', (event) => {
		try {
			const data = JSON.parse((event as MessageEvent).data);
			console.error('Agentlog SSE error event:', data.error);
		} catch (e) {
			// Ignore parse errors for error events
		}
	});
}

export function disconnectAgentlogSSE(): void {
	// Increment generation to invalidate any pending reconnect timers
	connectionGeneration++;
	
	if (agentlogReconnectTimeout) {
		clearTimeout(agentlogReconnectTimeout);
		agentlogReconnectTimeout = null;
	}
	if (agentlogEventSource) {
		agentlogEventSource.close();
		agentlogEventSource = null;
	}
	// Cancel any pending fetch operations
	agentlogEvents.cancelPending();
	agentlogConnectionStatus.set('disconnected');
}
