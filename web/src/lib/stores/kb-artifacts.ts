import { writable } from 'svelte/store';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Artifact from /api/kb/artifacts
export interface ArtifactFeedItem {
	path: string;           // Relative path from project root
	title: string;          // From frontmatter or filename
	type: string;           // investigation, decision, model, guide, principle
	status: string;         // Status field from frontmatter
	date: string;           // Date from frontmatter or filename
	summary: string;        // First paragraph or summary from frontmatter
	recommendation: boolean; // True if investigation has recommendation section
	modified_at: string;    // File modification time (ISO 8601)
	relative_time: string;  // Human-readable relative time (e.g., "2h ago")
}

// Artifacts response from /api/kb/artifacts
export interface KBArtifactsResponse {
	needs_decision: ArtifactFeedItem[];
	recent: ArtifactFeedItem[];
	by_type: Record<string, ArtifactFeedItem[]>;
	project_dir?: string;
	error?: string;
}

// KB artifacts store
function createKBArtifactsStore() {
	const { subscribe, set, update } = writable<KBArtifactsResponse | null>(null);

	return {
		subscribe,
		set,
		update,
		// Fetch KB artifacts from orch-go API
		// projectDir: Optional project directory to query
		// since: Time filter for "recently updated" (e.g., "7d", "24h", "30d", "all")
		async fetch(projectDir?: string, since: string = '7d'): Promise<void> {
			try {
				const params = new URLSearchParams();
				if (projectDir) {
					params.set('project_dir', projectDir);
				}
				params.set('since', since);
				const url = `${API_BASE}/api/kb/artifacts${params.toString() ? '?' + params.toString() : ''}`;
				const response = await fetch(url);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch KB artifacts:', error);
				set({
					needs_decision: [],
					recent: [],
					by_type: {},
					error: String(error)
				});
			}
		}
	};
}

export const kbArtifacts = createKBArtifactsStore();
