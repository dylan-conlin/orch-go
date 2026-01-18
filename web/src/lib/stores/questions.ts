import { writable } from 'svelte/store';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Question from the API
export interface Question {
	id: string;
	title: string;
	status: string;
	priority: number;
	labels?: string[];
	created_at?: string;
	closed_at?: string;
	close_reason?: string;
	blocking?: string[]; // IDs of issues this question blocks
}

// Response from /api/questions
export interface QuestionsResponse {
	open: Question[];
	investigating: Question[];
	answered: Question[];
	total_count: number;
	error?: string;
}

// Questions store
function createQuestionsStore() {
	const { subscribe, set } = writable<QuestionsResponse | null>(null);

	return {
		subscribe,
		set,
		// Fetch questions from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/questions`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch questions:', error);
				set({
					open: [],
					investigating: [],
					answered: [],
					total_count: 0,
					error: String(error)
				});
			}
		}
	};
}

export const questions = createQuestionsStore();
