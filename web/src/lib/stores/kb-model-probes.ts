// Stub store for kb-model-probes
// TODO: Implement full functionality for KB model probes integration

import { writable } from 'svelte/store';

export interface KBModelProbe {
	id: string;
	model: string;
	probe: string;
	status: string;
}

function createKBModelProbesStore() {
	const { subscribe, set, update } = writable<KBModelProbe[]>([]);

	return {
		subscribe,
		fetch: async () => {
			// Stub implementation - no-op for now
			console.warn('kb-model-probes store: fetch not implemented');
			return [];
		},
		set,
		update
	};
}

export const kbModelProbes = createKBModelProbesStore();
