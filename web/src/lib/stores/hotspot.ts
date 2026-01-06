import { writable, derived } from 'svelte/store';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Hotspot represents a detected area needing architect attention
export interface Hotspot {
	path: string;          // File path or topic name
	type: string;          // "fix-density" or "investigation-cluster"
	score: number;         // Number of occurrences
	details?: string;      // Additional context
	related_files?: string[];  // Files affected (for investigation clusters)
	recommendation: string;    // Suggested action
}

// HotspotReport is the complete analysis output
export interface HotspotReport {
	generated_at: string;
	analysis_period: string;
	fix_threshold: number;
	inv_threshold: number;
	hotspots: Hotspot[];
	total_fix_commits: number;
	total_investigations: number;
	has_architect_work: boolean;
}

// Create the hotspot store
function createHotspotStore() {
	const { subscribe, set } = writable<HotspotReport | null>(null);

	return {
		subscribe,
		set,
		// Fetch hotspot data from orch-go API
		async fetch(): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/hotspot`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch hotspot data:', error);
				set({
					generated_at: new Date().toISOString(),
					analysis_period: 'Error',
					fix_threshold: 5,
					inv_threshold: 3,
					hotspots: [],
					total_fix_commits: 0,
					total_investigations: 0,
					has_architect_work: false
				});
			}
		}
	};
}

export const hotspots = createHotspotStore();

// Derived store: map of file paths to their hotspot data for quick lookup
export const hotspotsByPath = derived(hotspots, ($hotspots) => {
	const pathMap = new Map<string, Hotspot>();
	if ($hotspots?.hotspots) {
		for (const hotspot of $hotspots.hotspots) {
			pathMap.set(hotspot.path, hotspot);
		}
	}
	return pathMap;
});

// Helper function to check if a file path is in a hotspot area
// This does partial matching - if the agent's workspace contains files that are hotspots
export function isInHotspotArea(filePath: string, hotspotPaths: Map<string, Hotspot>): Hotspot | null {
	// Direct match
	if (hotspotPaths.has(filePath)) {
		return hotspotPaths.get(filePath)!;
	}
	
	// Check if the file path contains any hotspot path
	for (const [hotspotPath, hotspot] of hotspotPaths) {
		if (filePath.includes(hotspotPath) || hotspotPath.includes(filePath)) {
			return hotspot;
		}
	}
	
	return null;
}

// Check if an agent's beads ID or task relates to hotspot areas
// This is a heuristic based on keyword matching
export function getHotspotForAgent(
	beadsId: string | undefined,
	task: string | undefined,
	skill: string | undefined,
	hotspotReport: HotspotReport | null
): Hotspot | null {
	if (!hotspotReport?.hotspots?.length) {
		return null;
	}

	// Combine searchable text from agent
	const searchText = [beadsId, task, skill]
		.filter(Boolean)
		.join(' ')
		.toLowerCase();

	if (!searchText) {
		return null;
	}

	// Check each hotspot for keyword match
	for (const hotspot of hotspotReport.hotspots) {
		// Extract key terms from the hotspot path
		const pathTerms = hotspot.path
			.split(/[\/\-_.]/)
			.filter(term => term.length > 2)
			.map(term => term.toLowerCase());

		// Check if any path term appears in the agent's context
		for (const term of pathTerms) {
			if (searchText.includes(term)) {
				return hotspot;
			}
		}
	}

	return null;
}
