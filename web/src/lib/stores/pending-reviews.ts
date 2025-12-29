import { writable } from 'svelte/store';

// API configuration
const API_BASE = 'http://localhost:3348';

// Single recommendation item
export interface PendingReviewItem {
	workspace_id: string;
	beads_id: string;
	index: number;
	text: string;
	reviewed: boolean;
	acted_on: boolean;
	dismissed: boolean;
}

// Agent with pending reviews
export interface PendingReviewAgent {
	workspace_id: string;
	workspace_path: string;
	beads_id: string;
	tldr?: string;
	total_recommendations: number;
	unreviewed_count: number;
	items: PendingReviewItem[];
	is_light_tier?: boolean; // True if this was a light tier spawn (no synthesis by design)
}

// API response
export interface PendingReviewsResponse {
	agents: PendingReviewAgent[];
	total_agents: number;
	total_unreviewed: number;
}

// Pending reviews store
function createPendingReviewsStore() {
	const { subscribe, set, update } = writable<PendingReviewsResponse | null>(null);

	return {
		subscribe,
		set,
		update,
		// Fetch pending reviews from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/pending-reviews`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch pending reviews:', error);
				set({
					agents: [],
					total_agents: 0,
					total_unreviewed: 0
				});
			}
		},
		// Dismiss a recommendation
		async dismiss(workspaceId: string, index: number): Promise<boolean> {
			try {
				const response = await fetch(`${API_BASE}/api/dismiss-review`, {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
					},
					body: JSON.stringify({
						workspace_id: workspaceId,
						index: index
					})
				});
				
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				
				const result = await response.json();
				if (result.success) {
					// Update local state to reflect dismissal
					update(state => {
						if (!state) return state;
						return {
							...state,
							total_unreviewed: state.total_unreviewed - 1,
							agents: state.agents.map(agent => {
								if (agent.workspace_id !== workspaceId) return agent;
								return {
									...agent,
									unreviewed_count: agent.unreviewed_count - 1,
									items: agent.items.map(item => {
										if (item.index !== index) return item;
										return {
											...item,
											dismissed: true,
											reviewed: true
										};
									})
								};
							}).filter(agent => agent.unreviewed_count > 0)
						};
					});
					return true;
				}
				return false;
			} catch (error) {
				console.error('Failed to dismiss recommendation:', error);
				return false;
			}
		},
		// Mark a recommendation as acted on (issue created)
		async markActedOn(workspaceId: string, index: number): Promise<boolean> {
			try {
				const response = await fetch(`${API_BASE}/api/act-on-review`, {
					method: 'POST',
					headers: {
						'Content-Type': 'application/json',
					},
					body: JSON.stringify({
						workspace_id: workspaceId,
						index: index
					})
				});
				
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				
				const result = await response.json();
				if (result.success) {
					// Update local state to reflect the action
					update(state => {
						if (!state) return state;
						return {
							...state,
							total_unreviewed: state.total_unreviewed - 1,
							agents: state.agents.map(agent => {
								if (agent.workspace_id !== workspaceId) return agent;
								return {
									...agent,
									unreviewed_count: agent.unreviewed_count - 1,
									items: agent.items.map(item => {
										if (item.index !== index) return item;
										return {
											...item,
											acted_on: true,
											reviewed: true
										};
									})
								};
							}).filter(agent => agent.unreviewed_count > 0)
						};
					});
					return true;
				}
				return false;
			} catch (error) {
				console.error('Failed to mark recommendation as acted on:', error);
				return false;
			}
		}
	};
}

export const pendingReviews = createPendingReviewsStore();
