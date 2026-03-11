import { writable } from 'svelte/store';

// API configuration
const API_BASE = 'http://localhost:3348';

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

// Context store - now uses SSE push for real-time updates
function createContextStore() {
	const { subscribe, set } = writable<OrchestratorContext>({});

	let eventSource: EventSource | null = null;
	let reconnectTimeout: ReturnType<typeof setTimeout> | null = null;
	let fallbackPollInterval: ReturnType<typeof setInterval> | null = null;

	return {
		subscribe,
		set,

		// Fetch current context from API (used for initial load and fallback)
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/context`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch context:', error);
				set({ error: String(error) });
			}
		},

		// Retry: force a fresh fetch (used by work-graph retry button)
		async retry(): Promise<void> {
			return this.fetch();
		},

		// Connect to SSE for real-time context push notifications.
		// Falls back to polling at a slow interval if SSE fails.
		connectSSE(): void {
			if (eventSource) return; // Already connected

			eventSource = new EventSource(`${API_BASE}/api/events/context`);

			eventSource.addEventListener('context.changed', (event: MessageEvent) => {
				try {
					const data = JSON.parse(event.data);
					set(data);
				} catch (e) {
					console.error('Failed to parse context event:', e);
				}
			});

			eventSource.onerror = () => {
				// SSE disconnected - clean up and fall back to polling
				eventSource?.close();
				eventSource = null;

				// Start fallback polling
				this.startFallbackPolling();

				// Try to reconnect SSE after delay
				if (reconnectTimeout) clearTimeout(reconnectTimeout);
				reconnectTimeout = setTimeout(() => {
					this.stopFallbackPolling();
					this.connectSSE();
				}, 5000);
			};
		},

		// Disconnect SSE and stop all polling
		disconnectSSE(): void {
			if (reconnectTimeout) {
				clearTimeout(reconnectTimeout);
				reconnectTimeout = null;
			}
			if (eventSource) {
				eventSource.close();
				eventSource = null;
			}
			this.stopFallbackPolling();
		},

		// Start slow fallback polling (30s) - only used when SSE is unavailable
		startFallbackPolling(): void {
			if (fallbackPollInterval) return;
			fallbackPollInterval = setInterval(() => {
				this.fetch();
			}, 30000);
		},

		// Stop fallback polling
		stopFallbackPolling(): void {
			if (fallbackPollInterval) {
				clearInterval(fallbackPollInterval);
				fallbackPollInterval = null;
			}
		},

		// Legacy: Start polling for context changes (kept for backwards compat)
		// Now just connects SSE + starts slow fallback
		startPolling(intervalMs: number = 2000): void {
			// Initial fetch for immediate data
			this.fetch();
			// Connect SSE for real-time updates
			this.connectSSE();
		},

		// Legacy: Stop polling (kept for backwards compat)
		stopPolling(): void {
			this.disconnectSSE();
		},
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
