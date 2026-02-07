import { writable } from 'svelte/store';

// API configuration
const API_BASE = 'https://localhost:3348';

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

// Context store
function createContextStore() {
	const { subscribe, set } = writable<OrchestratorContext>({});

	let pollInterval: ReturnType<typeof setInterval> | null = null;

	return {
		subscribe,
		set,

		// Fetch current context from API
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

		// Start polling for context changes (500ms default)
		startPolling(intervalMs: number = 500): void {
			if (pollInterval) return;
			
			// Initial fetch
			this.fetch();
			
			// Poll at interval
			pollInterval = setInterval(() => {
				this.fetch();
			}, intervalMs);
		},

		// Stop polling
		stopPolling(): void {
			if (pollInterval) {
				clearInterval(pollInterval);
				pollInterval = null;
			}
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
	
	if (state.project) {
		params.set('project', state.project);
	}
	
	const query = params.toString();
	return query ? `?${query}` : '';
}
