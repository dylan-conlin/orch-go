import { writable } from 'svelte/store';
import { shallowEqual } from '$lib/utils/shallow-equal';

// API configuration
const API_BASE = 'https://localhost:3348';

// Connection status
export type ConnectionStatus = 'connected' | 'disconnected' | 'retrying';

export interface ConnectionState {
	status: ConnectionStatus;
	error?: string;
	lastErrorLogged?: boolean; // Track if we've already logged this error
}

// Context response from /api/context
export interface OrchestratorContext {
	cwd?: string;
	project_dir?: string;
	project?: string;
	included_projects?: string[];
	error?: string;
}

// Time filter options
export type TimeFilter = '12h' | '24h' | '48h' | '7d' | 'all';

// Filter state
export interface FilterState {
	// Time-based filtering
	since: TimeFilter;
	// Project-based filtering (from orchestrator context)
	project?: string;
	includedProjects?: string[];
	// Whether to auto-follow the orchestrator's context
	followOrchestrator: boolean;
}

// Default filter state
const defaultFilterState: FilterState = {
	since: '12h',
	followOrchestrator: true,
};

// Load filter state from localStorage
function loadFilterState(): FilterState {
	if (typeof window === 'undefined') return defaultFilterState;
	try {
		const stored = localStorage.getItem('orch-dashboard-filters');
		if (stored) {
			return { ...defaultFilterState, ...JSON.parse(stored) };
		}
	} catch (e) {
		console.warn('Failed to load filter state:', e);
	}
	return defaultFilterState;
}

// Save filter state to localStorage
function saveFilterState(state: FilterState) {
	if (typeof window === 'undefined') return;
	try {
		localStorage.setItem('orch-dashboard-filters', JSON.stringify(state));
	} catch (e) {
		console.warn('Failed to save filter state:', e);
	}
}

// Connection status store (shared across all API calls)
function createConnectionStore() {
	const { subscribe, set, update } = writable<ConnectionState>({
		status: 'connected',
		lastErrorLogged: false
	});

	return {
		subscribe,
		
		setConnected(): void {
			update(state => ({
				...state,
				status: 'connected',
				error: undefined,
				lastErrorLogged: false
			}));
		},
		
		setDisconnected(error: string): void {
			update(state => {
				// Only log error once
				if (!state.lastErrorLogged) {
					console.error('Backend unavailable:', error);
				}
				return {
					status: 'disconnected',
					error,
					lastErrorLogged: true
				};
			});
		},
		
		setRetrying(): void {
			update(state => ({
				...state,
				status: 'retrying'
			}));
		}
	};
}

// Create connection status store first so it can be referenced
export const connectionStatus = createConnectionStore();

// Context store with exponential backoff
function createContextStore() {
	const { subscribe, set } = writable<OrchestratorContext>({});

	let pollTimeout: ReturnType<typeof setTimeout> | null = null;
	let currentAbortController: AbortController | null = null;
	let fetchInFlight: Promise<void> | null = null;
	let isPolling = false;
	let pollInterval = 2000;
	let currentBackoff = pollInterval;
	const maxBackoff = 30000; // Cap at 30s
	let isDisconnected = false; // Track connection state for backoff
	let currentData: OrchestratorContext = {}; // Track current data for shallow equality
	
	async function fetchWithRetry(): Promise<void> {
		if (fetchInFlight) {
			await fetchInFlight;
			return;
		}

		fetchInFlight = (async () => {
			const abortController = new AbortController();
			currentAbortController = abortController;

			try {
				const response = await fetch(`${API_BASE}/api/context`, {
					signal: abortController.signal,
				});
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}`);
				}
				const data = await response.json();
				
				// Only update if data actually changed (reduces reactive cascades)
				if (!shallowEqual(currentData, data)) {
					currentData = data;
					set(data);
				}
				
				// Success - use configured poll interval and mark connected
				currentBackoff = pollInterval;
				isDisconnected = false;
				connectionStatus.setConnected();
			} catch (error) {
				// Abort is expected on unmount/stop; don't mark disconnected
				if (error instanceof Error && error.name === 'AbortError') {
					return;
				}

				// Connection failed - mark disconnected
				isDisconnected = true;
				connectionStatus.setDisconnected(String(error));
				const errorData = { error: String(error) };
				currentData = errorData;
				set(errorData);
				
				// Don't throw - let polling continue with backoff
			} finally {
				if (currentAbortController === abortController) {
					currentAbortController = null;
				}
			}
		})();

		try {
			await fetchInFlight;
		} finally {
			fetchInFlight = null;
		}
	}

	function scheduleNextPoll(): void {
		if (!isPolling || pollTimeout) return;
		
		pollTimeout = setTimeout(async () => {
			pollTimeout = null;
			if (!isPolling) return;
			
			// If disconnected, mark as retrying before fetch
			if (isDisconnected) {
				connectionStatus.setRetrying();
			}
			
			await fetchWithRetry();
			
			// If still disconnected after fetch, increase backoff exponentially
			if (isDisconnected) {
				currentBackoff = Math.min(currentBackoff * 2, maxBackoff);
			} else {
				currentBackoff = pollInterval;
			}
			
			// Schedule next poll
			scheduleNextPoll();
		}, currentBackoff);
	}

	return {
		subscribe,
		set,

		// Fetch current context from API
		async fetch(): Promise<void> {
			await fetchWithRetry();
		},

		// Start polling for context changes with exponential backoff
		startPolling(intervalMs: number = 2000): void {
			pollInterval = intervalMs;

			if (isPolling) return;
			isPolling = true;
			
			// Set initial poll delay based on configured interval
			currentBackoff = pollInterval;
			
			// Initial fetch
			this.fetch().catch(() => {
				// Errors are handled in fetchWithRetry
			});
			
			// Start polling loop
			scheduleNextPoll();
		},

		// Stop polling
		stopPolling(): void {
			isPolling = false;
			if (pollTimeout) {
				clearTimeout(pollTimeout);
				pollTimeout = null;
			}
			if (currentAbortController) {
				currentAbortController.abort();
				currentAbortController = null;
			}
		},
		
		// Manual retry - resets backoff
		async retry(): Promise<void> {
			currentBackoff = pollInterval;
			await fetchWithRetry();
		}
	};
}

// Filter state store
function createFilterStore() {
	const initial = loadFilterState();
	const { subscribe, set, update } = writable<FilterState>(initial);

	return {
		subscribe,

		// Set time filter
		setTimeFilter(since: TimeFilter): void {
			update((state) => {
				const newState = { ...state, since };
				saveFilterState(newState);
				return newState;
			});
		},

		// Set project filter (from orchestrator context)
		setProjectFilter(project?: string, includedProjects?: string[]): void {
			update((state) => {
				const newState = { ...state, project, includedProjects };
				saveFilterState(newState);
				return newState;
			});
		},

		// Toggle follow orchestrator mode
		setFollowOrchestrator(follow: boolean): void {
			update((state) => {
				const newState = { ...state, followOrchestrator: follow };
				saveFilterState(newState);
				return newState;
			});
		},

		// Clear project filter
		clearProjectFilter(): void {
			update((state) => {
				const newState = { ...state, project: undefined, includedProjects: undefined };
				saveFilterState(newState);
				return newState;
			});
		},

		// Reset to defaults
		reset(): void {
			set(defaultFilterState);
			saveFilterState(defaultFilterState);
		},
	};
}

export const orchestratorContext = createContextStore();
export const filters = createFilterStore();

// Build API query string from filter state
export function buildFilterQueryString(state: FilterState): string {
	const params = new URLSearchParams();

	if (state.since && state.since !== 'all') {
		params.set('since', state.since);
	}

	// Multi-project support: serialize includedProjects as comma-separated values
	// This enables "orch-go special case" where orchestrator coordinates across 6 repos
	if (state.includedProjects && state.includedProjects.length > 0) {
		params.set('project', state.includedProjects.join(','));
	} else if (state.project) {
		// Fallback to single project if no includedProjects
		params.set('project', state.project);
	}

	const query = params.toString();
	return query ? `?${query}` : '';
}
