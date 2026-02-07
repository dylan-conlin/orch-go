import { writable, derived } from 'svelte/store';
import { createSSEConnection, type SSEConnection } from '../services/sse-connection';

// Service lifecycle event from events.jsonl
export interface ServiceLogEvent {
	id: string; // Unique ID for keyed rendering
	type: string; // service.started, service.crashed, service.restarted
	session_id?: string; // Service name used as session ID
	timestamp: number; // Unix timestamp
	data?: {
		service_name?: string;
		project_path?: string;
		old_pid?: number;
		new_pid?: number;
		pid?: number;
		restart_count?: number;
		auto_restart?: boolean;
	};
}

// Counter for generating unique event IDs
let eventIdCounter = 0;

// Generate a unique event ID
function generateEventId(): string {
	return `svc-evt-${Date.now()}-${++eventIdCounter}`;
}

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Fetch state management - prevents race conditions during rapid reloads
let currentFetchController: AbortController | null = null;

// Servicelog store
function createServicelogStore() {
	const { subscribe, set, update } = writable<ServiceLogEvent[]>([]);

	return {
		subscribe,
		set,
		update,
		addEvent: (event: Omit<ServiceLogEvent, 'id'>) => {
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
				const response = await fetch(`${API_BASE}/api/events/services`, {
					signal: currentFetchController.signal
				});
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				// Assign unique IDs to fetched events
				const eventsWithIds = (data || []).map((e: Omit<ServiceLogEvent, 'id'>) => ({
					...e,
					id: generateEventId()
				}));
				set(eventsWithIds);
			} catch (error) {
				// Don't log abort errors - they're expected during cleanup
				if (error instanceof Error && error.name === 'AbortError') {
					return;
				}
				console.error('Failed to fetch servicelog:', error);
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

export const servicelogEvents = createServicelogStore();

// Derived stores for filtered views
export const crashedEvents = derived(servicelogEvents, ($events) =>
	$events.filter((e) => e.type === 'service.crashed')
);

export const restartedEvents = derived(servicelogEvents, ($events) =>
	$events.filter((e) => e.type === 'service.restarted')
);

export const startedEvents = derived(servicelogEvents, ($events) =>
	$events.filter((e) => e.type === 'service.started')
);

// Servicelog SSE connection status
export const servicelogConnectionStatus = writable<'connected' | 'disconnected' | 'connecting'>('disconnected');

// Servicelog SSE connection manager - uses shared service for connection lifecycle
let servicelogConnection: SSEConnection | null = null;

// Build event listeners for the servicelog SSE connection
function buildServicelogEventListeners(): Record<string, (event: MessageEvent) => void> {
	return {
		'servicelog': (event: MessageEvent) => {
			try {
				const data = JSON.parse(event.data);
				servicelogEvents.addEvent(data);
				// Note: Service events might want to trigger a services.fetch() to update service cards
				// But we'll keep it simple for now and let the periodic refresh handle it
			} catch (e) {
				console.error('Failed to parse servicelog event:', e);
			}
		},
		'connected': (event: MessageEvent) => {
			try {
				const data = JSON.parse(event.data);
				console.log('Servicelog SSE connected to:', data.source);
			} catch (e) {
				console.log('Servicelog SSE connected');
			}
		},
		'error': (event: MessageEvent) => {
			try {
				const data = JSON.parse(event.data);
				console.error('Servicelog SSE error event:', data.error);
			} catch (e) {
				// Ignore parse errors for error events
			}
		}
	};
}

export function connectServicelogSSE(): void {
	// Create connection if not exists
	if (!servicelogConnection) {
		servicelogConnection = createSSEConnection(`${API_BASE}/api/events/services?follow=true`, {
			onOpen: () => {
				servicelogConnectionStatus.set('connected');
				// Fetch initial events on connection
				servicelogEvents.fetch().catch(console.error);
			},
			onDisconnect: () => {
				servicelogConnectionStatus.set('disconnected');
			},
			eventListeners: buildServicelogEventListeners(),
			reconnectDelayMs: 5000,
			autoReconnect: true
		});

		// Sync the connection status from the service to our local store
		servicelogConnection.status.subscribe((status) => {
			servicelogConnectionStatus.set(status);
		});
	}

	// Mark as connecting and initiate connection
	servicelogConnectionStatus.set('connecting');
	servicelogConnection.connect();
}

export function disconnectServicelogSSE(): void {
	if (servicelogConnection) {
		servicelogConnection.disconnect();
	}
	// Cancel any pending fetch operations
	servicelogEvents.cancelPending();
	servicelogConnectionStatus.set('disconnected');
}
