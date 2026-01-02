import { writable } from 'svelte/store';
import { browser } from '$app/environment';

// Dashboard mode: operational (daily) vs historical (full archive)
export type DashboardMode = 'operational' | 'historical';

const STORAGE_KEY = 'orch-dashboard-mode';

// Get stored value from localStorage (only works in browser)
function getStoredMode(): DashboardMode {
	if (!browser) return 'operational';
	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored === 'operational' || stored === 'historical') {
			return stored;
		}
	} catch (e) {
		console.warn('Failed to load dashboard mode from localStorage:', e);
	}
	return 'operational';
}

// Create store with localStorage persistence
function createDashboardModeStore() {
	// Start with default, will be synced from localStorage on init()
	const store = writable<DashboardMode>('operational');
	const { subscribe, update } = store;

	return {
		subscribe,
		set: (value: DashboardMode) => {
			store.set(value);
			// Persist to localStorage
			if (browser) {
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
				if (browser) {
					try {
						localStorage.setItem(STORAGE_KEY, newValue);
					} catch (e) {
						console.warn('Failed to save dashboard mode to localStorage:', e);
					}
				}
				return newValue;
			});
		},
		// Initialize from localStorage (call in onMount)
		init: () => {
			if (browser) {
				const stored = getStoredMode();
				store.set(stored);
			}
		}
	};
}

export const dashboardMode = createDashboardModeStore();
