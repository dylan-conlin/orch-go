// Stub store for deliverables tracking
// TODO: Implement full functionality for deliverables/checkpoints

import { writable } from 'svelte/store';

export interface Deliverable {
	id: string;
	issueId: string;
	title: string;
	completed: boolean;
	created_at: string;
}

function createDeliverablesStore() {
	const { subscribe, set, update } = writable<Deliverable[]>([]);

	return {
		subscribe,
		fetch: async () => {
			// Stub implementation - no-op for now
			console.warn('deliverables store: fetch not implemented');
			return [];
		},
		set,
		update
	};
}

export const deliverables = createDeliverablesStore();
