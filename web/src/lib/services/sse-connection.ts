import { writable, type Writable } from 'svelte/store';

export type ConnectionStatus = 'connected' | 'disconnected' | 'connecting';

export interface SSEConnectionOptions {
	/** Callback invoked when connection is established */
	onOpen?: () => void;
	/** Callback invoked when a message is received */
	onMessage?: (event: MessageEvent) => void;
	/** Callback invoked when connection is lost (before reconnect) */
	onDisconnect?: () => void;
	/** Event listeners to register - maps event name to handler */
	eventListeners?: Record<string, (event: MessageEvent) => void>;
	/** Auto-reconnect delay in ms (default: 5000) */
	reconnectDelayMs?: number;
	/** Whether to auto-reconnect on error (default: true) */
	autoReconnect?: boolean;
}

export interface SSEConnection {
	/** Connect to the SSE endpoint */
	connect: () => void;
	/** Disconnect from the SSE endpoint and cleanup */
	disconnect: () => void;
	/** Connection status store (reactive) */
	status: Writable<ConnectionStatus>;
	/** Check if currently connected */
	isConnected: () => boolean;
}

/**
 * Create a managed SSE connection with automatic reconnection and stale connection handling.
 * 
 * Key features:
 * - Generation counter prevents stale reconnect timers from firing
 * - Auto-reconnect with configurable delay
 * - Clean disconnect that cancels pending reconnects
 * - Reactive connection status via Svelte store
 * 
 * @example
 * const connection = createSSEConnection('http://localhost:3348/api/events', {
 *   onOpen: () => agents.fetch(),
 *   onMessage: (event) => handleSSEEvent(JSON.parse(event.data)),
 *   eventListeners: {
 *     'session.status': (event) => console.log('Status:', event.data)
 *   }
 * });
 * 
 * // Connect on mount
 * onMount(() => connection.connect());
 * 
 * // Disconnect on cleanup
 * onDestroy(() => connection.disconnect());
 */
export function createSSEConnection(
	url: string,
	options: SSEConnectionOptions = {}
): SSEConnection {
	const {
		onOpen,
		onMessage,
		onDisconnect,
		eventListeners = {},
		reconnectDelayMs = 5000,
		autoReconnect = true
	} = options;

	// Connection status store
	const status = writable<ConnectionStatus>('disconnected');

	// Internal state
	let eventSource: EventSource | null = null;
	let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
	// Generation counter - prevents stale reconnect timers from firing
	// Incremented on each connect/disconnect to invalidate pending timers
	let connectionGeneration = 0;

	function connect(): void {
		// Increment generation to invalidate any pending reconnect timers
		const thisGeneration = ++connectionGeneration;

		// Clear any pending reconnect timer from previous connection
		if (reconnectTimeout) {
			clearTimeout(reconnectTimeout);
			reconnectTimeout = null;
		}

		// Close existing connection
		if (eventSource) {
			eventSource.close();
			eventSource = null;
		}

		status.set('connecting');

		eventSource = new EventSource(url);

		eventSource.onopen = () => {
			// Ignore if this connection is stale (newer connection started)
			if (thisGeneration !== connectionGeneration) {
				eventSource?.close();
				return;
			}
			status.set('connected');
			onOpen?.();
		};

		eventSource.onerror = () => {
			// Ignore if this connection is stale (newer connection started)
			if (thisGeneration !== connectionGeneration) {
				return;
			}

			// Don't log errors during page unload (expected behavior)
			status.set('disconnected');
			eventSource?.close();
			eventSource = null;
			onDisconnect?.();

			// Auto-reconnect after delay (unless disabled or page is unloading)
			// Use generation check to prevent stale timer from firing
			if (autoReconnect) {
				if (reconnectTimeout) {
					clearTimeout(reconnectTimeout);
				}
				reconnectTimeout = setTimeout(() => {
					// Only reconnect if no newer connection was started
					if (thisGeneration === connectionGeneration) {
						connect();
					}
				}, reconnectDelayMs);
			}
		};

		// Handle generic messages
		if (onMessage) {
			eventSource.onmessage = onMessage;
		}

		// Register custom event listeners
		for (const [eventType, handler] of Object.entries(eventListeners)) {
			eventSource.addEventListener(eventType, handler as EventListener);
		}
	}

	function disconnect(): void {
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
		status.set('disconnected');
	}

	function isConnected(): boolean {
		return eventSource !== null && eventSource.readyState === EventSource.OPEN;
	}

	return {
		connect,
		disconnect,
		status,
		isConnected
	};
}
