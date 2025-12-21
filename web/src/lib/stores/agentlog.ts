import { writable, derived } from 'svelte/store';

// Agent lifecycle event from events.jsonl
export interface AgentLogEvent {
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

// API configuration
const API_BASE = 'http://127.0.0.1:3333';

// Agentlog store
function createAgentlogStore() {
	const { subscribe, set, update } = writable<AgentLogEvent[]>([]);

	return {
		subscribe,
		set,
		update,
		addEvent: (event: AgentLogEvent) => {
			update((events) => {
				const newEvents = [...events, event];
				// Keep last 100 events
				return newEvents.slice(-100);
			});
		},
		clear: () => {
			set([]);
		},
		// Fetch initial events from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/agentlog`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data || []);
			} catch (error) {
				console.error('Failed to fetch agentlog:', error);
				throw error;
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

export function connectAgentlogSSE(): void {
	if (agentlogEventSource) {
		agentlogEventSource.close();
	}

	agentlogConnectionStatus.set('connecting');

	agentlogEventSource = new EventSource(`${API_BASE}/api/agentlog?follow=true`);

	agentlogEventSource.onopen = () => {
		agentlogConnectionStatus.set('connected');
		// Fetch initial events on connection
		agentlogEvents.fetch().catch(console.error);
	};

	agentlogEventSource.onerror = (error) => {
		console.error('Agentlog SSE error:', error);
		agentlogConnectionStatus.set('disconnected');
		agentlogEventSource?.close();
		agentlogEventSource = null;

		// Auto-reconnect after 5 seconds
		if (agentlogReconnectTimeout) {
			clearTimeout(agentlogReconnectTimeout);
		}
		agentlogReconnectTimeout = setTimeout(() => {
			connectAgentlogSSE();
		}, 5000);
	};

	// Handle agentlog events
	agentlogEventSource.addEventListener('agentlog', (event) => {
		try {
			const data = JSON.parse((event as MessageEvent).data);
			agentlogEvents.addEvent(data);
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
	if (agentlogReconnectTimeout) {
		clearTimeout(agentlogReconnectTimeout);
		agentlogReconnectTimeout = null;
	}
	if (agentlogEventSource) {
		agentlogEventSource.close();
		agentlogEventSource = null;
	}
	agentlogConnectionStatus.set('disconnected');
}
