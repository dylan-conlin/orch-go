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

/**
 * Get context usage percent for agent
 * Stub implementation - returns 0 for now
 */
export function getContextPercent(agent: any): number {
	// TODO: Implement actual context percent calculation
	return 0;
}

/**
 * Get context usage color based on percent
 * Stub implementation - returns 'green' for now
 */
export function getContextColor(percent: number): string {
	if (percent < 60) return 'text-green-600';
	if (percent < 80) return 'text-yellow-600';
	return 'text-red-600';
}

/**
 * Get expressive status string for agent
 * Stub implementation - returns empty string for now
 */
export function getExpressiveStatus(agent: any): string {
	// TODO: Implement actual expressive status generation
	return '';
}
