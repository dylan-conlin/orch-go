import { writable } from 'svelte/store';

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
	let currentBackoff = 1000; // Start at 1s
	const maxBackoff = 30000; // Cap at 30s
	const baseBackoff = 1000; // Base backoff 1s
	let isDisconnected = false; // Track connection state for backoff
	
	async function fetchWithRetry(): Promise<void> {
		try {
			const response = await fetch(`${API_BASE}/api/context`);
			if (!response.ok) {
				throw new Error(`HTTP ${response.status}`);
			}
			const data = await response.json();
			set(data);
			
			// Success - reset backoff and mark connected
			currentBackoff = baseBackoff;
			isDisconnected = false;
			connectionStatus.setConnected();
		} catch (error) {
			// Connection failed - mark disconnected
			isDisconnected = true;
			connectionStatus.setDisconnected(String(error));
			set({ error: String(error) });
			
			// Don't throw - let polling continue with backoff
		}
	}

	function scheduleNextPoll(): void {
		if (pollTimeout) return;
		
		pollTimeout = setTimeout(async () => {
			pollTimeout = null;
			
			// If disconnected, mark as retrying before fetch
			if (isDisconnected) {
				connectionStatus.setRetrying();
			}
			
			await fetchWithRetry();
			
			// If still disconnected after fetch, increase backoff exponentially
			if (isDisconnected) {
				currentBackoff = Math.min(currentBackoff * 2, maxBackoff);
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
			if (pollTimeout) return;
			
			// Set initial backoff based on interval
			currentBackoff = intervalMs;
			
			// Initial fetch
			this.fetch();
			
			// Start polling loop
			scheduleNextPoll();
		},

		// Stop polling
		stopPolling(): void {
			if (pollTimeout) {
				clearTimeout(pollTimeout);
				pollTimeout = null;
			}
		},
		
		// Manual retry - resets backoff
		async retry(): Promise<void> {
			currentBackoff = baseBackoff;
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
