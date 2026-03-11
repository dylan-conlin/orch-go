import { writable, derived } from 'svelte/store';

// API configuration
const API_BASE = 'http://localhost:3348';

// Cache validation state
interface CacheValidationState {
	currentVersion: string | null;
	lastCacheTime: Date | null;
	versionMismatchDetected: boolean;
	isStale: boolean;
}

function createCacheValidationStore() {
	const { subscribe, set, update } = writable<CacheValidationState>({
		currentVersion: null,
		lastCacheTime: null,
		versionMismatchDetected: false,
		isStale: false
	});

	return {
		subscribe,

		// Update cache state from API response headers
		updateFromResponse(response: Response): void {
			const version = response.headers.get('x-orch-version');
			const cacheTime = response.headers.get('x-cache-time');

			update(state => {
				const newState = { ...state };

				// Check for version mismatch
				if (version) {
					if (state.currentVersion && state.currentVersion !== version) {
						newState.versionMismatchDetected = true;
					}
					newState.currentVersion = version;
				}

				// Update cache time and check staleness
				if (cacheTime) {
					const timestamp = new Date(cacheTime);
					newState.lastCacheTime = timestamp;

					// Check if data is stale (>60 seconds old)
					const ageMs = Date.now() - timestamp.getTime();
					newState.isStale = ageMs > 60000;
				}

				return newState;
			});
		},

		// Dismiss version mismatch warning
		dismissVersionMismatch(): void {
			update(state => ({
				...state,
				versionMismatchDetected: false
			}));
		},

		// Reset state
		reset(): void {
			set({
				currentVersion: null,
				lastCacheTime: null,
				versionMismatchDetected: false,
				isStale: false
			});
		}
	};
}

export const cacheValidation = createCacheValidationStore();

// Derived stores for easy access
export const shouldShowVersionMismatch = derived(
	cacheValidation,
	$state => $state.versionMismatchDetected
);

export const shouldShowStaleWarning = derived(
	cacheValidation,
	$state => $state.isStale
);

// Helper function to wrap fetch calls with cache validation
export async function fetchWithCacheValidation(url: string, options?: RequestInit): Promise<Response> {
	const response = await fetch(url, options);
	cacheValidation.updateFromResponse(response);
	return response;
}
