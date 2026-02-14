// Stub store for work-in-progress tracking
// TODO: Implement full functionality for WIP tracking

import { writable, derived } from 'svelte/store';

export interface WIPItem {
	id: string;
	title: string;
	status: string;
	project: string;
}

function createWIPStore() {
	const { subscribe, set, update } = writable<WIPItem[]>([]);

	return {
		subscribe,
		fetch: async () => {
			// Stub implementation - no-op for now
			console.warn('wip store: fetch not implemented');
			return [];
		},
		set,
		update
	};
}

export const wip = createWIPStore();
export const wipItems = derived(wip, ($wip) => $wip);

/**
 * Compute agent health status based on WIP item state
 * Stub implementation - returns 'healthy' for now
 */
export function computeAgentHealth(wipItem: WIPItem): 'healthy' | 'warning' | 'error' {
	// TODO: Implement actual health computation based on agent state
	console.warn('computeAgentHealth: not implemented, returning healthy');
	return 'healthy';
}
