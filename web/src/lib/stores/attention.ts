// Stub store for attention/completed issues tracking
// TODO: Implement full functionality for attention tracking

import { writable } from 'svelte/store';

export interface CompletedIssue {
	id: string;
	title: string;
	completed_at: string;
	project: string;
}

function createAttentionStore() {
	const { subscribe, set, update } = writable<CompletedIssue[]>([]);

	return {
		subscribe,
		fetch: async () => {
			// Stub implementation - no-op for now
			console.warn('attention store: fetch not implemented');
			return [];
		},
		set,
		update
	};
}

export const attention = createAttentionStore();

export function formatRelativeTime(timestamp: string): string {
	const now = new Date();
	const then = new Date(timestamp);
	const diffMs = now.getTime() - then.getTime();
	const diffMins = Math.floor(diffMs / 60000);
	
	if (diffMins < 1) return 'just now';
	if (diffMins < 60) return `${diffMins}m ago`;
	
	const diffHours = Math.floor(diffMins / 60);
	if (diffHours < 24) return `${diffHours}h ago`;
	
	const diffDays = Math.floor(diffHours / 24);
	return `${diffDays}d ago`;
}

// Attention badge configuration stub
// TODO: Implement proper attention badge types and styling
export const ATTENTION_BADGE_CONFIG = {
	warning: {
		color: 'text-yellow-600',
		bg: 'bg-yellow-100',
		label: 'Needs Attention'
	},
	error: {
		color: 'text-red-600',
		bg: 'bg-red-100',
		label: 'Error'
	},
	info: {
		color: 'text-blue-600',
		bg: 'bg-blue-100',
		label: 'Info'
	}
};
