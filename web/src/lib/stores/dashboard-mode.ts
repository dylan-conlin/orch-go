import { writable } from 'svelte/store';

// Dashboard mode: operational (daily) vs historical (full archive)
export type DashboardMode = 'operational' | 'historical';

const STORAGE_KEY = 'orch-dashboard-mode';

// Create store with localStorage persistence
function createDashboardModeStore() {
	// Load initial value from localStorage
	let initialValue: DashboardMode = 'operational'; // Default to operational
	if (typeof window !== 'undefined') {
		try {
			const stored = localStorage.getItem(STORAGE_KEY);
			if (stored === 'operational' || stored === 'historical') {
				initialValue = stored;
			}
		} catch (e) {
			console.warn('Failed to load dashboard mode from localStorage:', e);
		}
	}

	const { subscribe, set, update } = writable<DashboardMode>(initialValue);

	return {
		subscribe,
		set: (value: DashboardMode) => {
			set(value);
			// Persist to localStorage
			if (typeof window !== 'undefined') {
				try {
					localStorage.setItem(STORAGE_KEY, value);
				} catch (e) {
					console.warn('Failed to save dashboard mode to localStorage:', e);
				}
			}
		},
		toggle: () => {
			update((current) => {
				const newValue = current === 'operational' ? 'historical' : 'operational';
				// Persist to localStorage
				if (typeof window !== 'undefined') {
					try {
						localStorage.setItem(STORAGE_KEY, newValue);
					} catch (e) {
						console.warn('Failed to save dashboard mode to localStorage:', e);
					}
				}
				return newValue;
			});
		}
	};
}

export const dashboardMode = createDashboardModeStore();
