import { writable } from 'svelte/store';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Daily cost from /api/usage/cost
export interface DailyCost {
	date: string; // YYYY-MM-DD
	total_cost: number; // Total cost for the day in USD
	count: number; // Number of sessions included
}

// Cost response from /api/usage/cost
export interface CostInfo {
	current_month_cost: number; // Total cost for current month in USD
	current_month_date: string; // YYYY-MM format
	daily_costs: DailyCost[]; // Daily costs for last 30 days
	budget_color: 'green' | 'yellow' | 'red'; // Budget status color
	budget_emoji: string; // Emoji for budget status
	error?: string;
}

// Cost store
function createCostStore() {
	const { subscribe, set } = writable<CostInfo | null>(null);

	return {
		subscribe,
		set,
		// Fetch cost from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/usage/cost`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch cost:', error);
				set({
					current_month_cost: 0,
					current_month_date: new Date().toISOString().slice(0, 7), // YYYY-MM
					daily_costs: [],
					budget_color: 'green',
					budget_emoji: '🟢',
					error: String(error)
				});
			}
		}
	};
}

export const cost = createCostStore();

// Helper to format cost as currency
export function formatCost(cost: number): string {
	return new Intl.NumberFormat('en-US', {
		style: 'currency',
		currency: 'USD',
		minimumFractionDigits: 2,
		maximumFractionDigits: 2
	}).format(cost);
}

// Helper to format cost briefly (e.g., "$12.34")
export function formatCostBrief(cost: number): string {
	return `$${cost.toFixed(2)}`;
}

// Helper to get color class based on budget
export function getBudgetColor(cost: number): 'green' | 'yellow' | 'red' {
	if (cost < 100) return 'green';
	if (cost < 180) return 'yellow';
	return 'red';
}

// Helper to get emoji based on budget
export function getBudgetEmoji(cost: number): string {
	if (cost < 100) return '🟢';
	if (cost < 180) return '🟡';
	return '🔴';
}