import { writable, get } from 'svelte/store';
import { browser } from '$app/environment';
import { goto } from '$app/navigation';

// Dashboard mode: operational (daily) vs historical (full archive)
export type DashboardMode = 'operational' | 'historical';

const STORAGE_KEY = 'orch-dashboard-mode';

// Map URL tab values to dashboard modes
const TAB_TO_MODE: Record<string, DashboardMode> = {
	ops: 'operational',
	operational: 'operational',
	history: 'historical',
	historical: 'historical'
};

const MODE_TO_TAB: Record<DashboardMode, string> = {
	operational: 'ops',
	historical: 'history'
};

// Get mode from URL query params (only works in browser)
function getModeFromURL(): DashboardMode | null {
	if (!browser) return null;
	try {
		const url = new URL(window.location.href);
		const tab = url.searchParams.get('tab');
		if (tab && tab in TAB_TO_MODE) {
			return TAB_TO_MODE[tab];
		}
	} catch (e) {
		console.warn('Failed to parse URL for tab param:', e);
	}
	return null;
}

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

// Update URL with current mode (without navigation, just replaceState)
function updateURL(mode: DashboardMode) {
	if (!browser) return;
	try {
		const url = new URL(window.location.href);
		url.searchParams.set('tab', MODE_TO_TAB[mode]);
		// Use replaceState to avoid cluttering history
		goto(url.pathname + url.search, { replaceState: true, noScroll: true, keepFocus: true });
	} catch (e) {
		console.warn('Failed to update URL with tab param:', e);
	}
}

// Create store with localStorage and URL persistence
function createDashboardModeStore() {
	// Start with default, will be synced from URL/localStorage on init()
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
				// Update URL
				updateURL(value);
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
					// Update URL
					updateURL(newValue);
				}
				return newValue;
			});
		},
		// Initialize from URL or localStorage (call in onMount)
		// URL takes precedence over localStorage for deep linking
		init: () => {
			if (browser) {
				// URL param takes precedence for deep linking
				const urlMode = getModeFromURL();
				if (urlMode) {
					store.set(urlMode);
					// Also persist to localStorage so it persists after URL is removed
					try {
						localStorage.setItem(STORAGE_KEY, urlMode);
					} catch (e) {
						console.warn('Failed to save dashboard mode to localStorage:', e);
					}
					return;
				}
				// Fall back to localStorage
				const stored = getStoredMode();
				store.set(stored);
				// Set initial URL to reflect current state
				updateURL(stored);
			}
		}
	};
}

export const dashboardMode = createDashboardModeStore();
