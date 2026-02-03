import { writable } from 'svelte/store';
import { shallowEqual } from '$lib/utils/shallow-equal';

// API configuration - HTTPS for HTTP/2 multiplexing
const API_BASE = 'https://localhost:3348';

// Graph node from /api/beads/graph
export interface GraphNode {
	id: string;
	title: string;
	type: string;     // beads: task, bug, feature, epic, question; kb: investigation, decision
	status: string;   // open, in_progress, closed, blocked, Complete, Accepted, etc.
	priority: number; // 0-4 for beads, 0 for kb artifacts
	source: string;   // "beads" or "kb"
	date?: string;    // for kb artifacts
	created_at?: string; // creation timestamp
	description?: string; // issue description
}

// Graph edge (dependency) from /api/beads/graph
export interface GraphEdge {
	from: string; // ID of the issue that has the dependency
	to: string;   // ID of the issue being depended on
	type: string; // dependency_type: blocks, parent-child, relates_to
}

// Work graph response from /api/beads/graph
export interface WorkGraphResponse {
	nodes: GraphNode[];
	edges: GraphEdge[];
	node_count: number;
	edge_count: number;
	project_dir?: string;
	error?: string;
}

// Attention badge type (matches attention store)
export type AttentionBadgeType =
	| 'verify'      // Phase: Complete, needs orch complete
	| 'decide'      // Investigation has recommendation needing decision
	| 'escalate'    // Question needs human judgment
	| 'likely_done' // Commits suggest completion
	| 'unblocked'   // Blocker just closed, now actionable
	| 'stuck'       // Agent stuck >2h
	| 'crashed';    // Agent crashed without completing

// Tree node with hierarchy and expansion state
export interface TreeNode extends GraphNode {
	children: TreeNode[];
	depth: number;
	expanded: boolean; // Children expanded in tree
	details_expanded: boolean; // L1 details expanded
	blocked_by: string[];
	blocks: string[];
	parent_id?: string;
	// Attention signal (if any)
	attentionBadge?: AttentionBadgeType;
	attentionReason?: string;
}

// Work graph store with AbortController support to prevent race conditions
function createWorkGraphStore() {
	const { subscribe, set, update } = writable<WorkGraphResponse | null>(null);
	
	// Track in-flight requests to cancel stale ones
	let currentAbortController: AbortController | null = null;
	let fetchSequence = 0; // Sequence guard for additional safety
	let currentData: WorkGraphResponse | null = null; // Track current data for shallow equality

	return {
		subscribe,
		set,
		update,
		
		// Cancel any pending fetch - useful when project context changes
		cancelPending(): void {
			if (currentAbortController) {
				currentAbortController.abort();
				currentAbortController = null;
			}
		},
		
		// Fetch work graph from orch-go API
		// projectDir: Optional project directory to query (for following orchestrator context)
		// scope: "focus" (default) or "open" (all open issues)
		async fetch(projectDir?: string, scope: string = 'open'): Promise<void> {
			// Cancel any pending request before starting new one
			if (currentAbortController) {
				currentAbortController.abort();
			}
			
			// Create new abort controller for this request
			const abortController = new AbortController();
			currentAbortController = abortController;
			
			// Increment sequence for this fetch
			const thisSequence = ++fetchSequence;
			
			try {
				const params = new URLSearchParams();
				if (projectDir) {
					params.set('project_dir', projectDir);
				}
				params.set('scope', scope);
				const url = `${API_BASE}/api/beads/graph${params.toString() ? '?' + params.toString() : ''}`;
				const response = await fetch(url, { signal: abortController.signal });
				
				// Sequence guard: ignore response if newer fetch started
				if (thisSequence !== fetchSequence) {
					return; // Stale response, discard
				}
				
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data = await response.json();
				
				// Final sequence check before setting state
				if (thisSequence === fetchSequence) {
					// Only update if data actually changed (reduces reactive cascades)
					if (!shallowEqual(currentData, data)) {
						currentData = data;
						set(data);
					}
				}
			} catch (error) {
				// Ignore abort errors - they're intentional
				if (error instanceof Error && error.name === 'AbortError') {
					return;
				}
				
				// Only set error if this is still the current request
				if (thisSequence === fetchSequence) {
					console.error('Failed to fetch work graph:', error);
					const errorData = {
						nodes: [],
						edges: [],
						node_count: 0,
						edge_count: 0,
						error: String(error)
					};
					currentData = errorData;
					set(errorData);
				}
			} finally {
				// Clear controller if this was the current one
				if (currentAbortController === abortController) {
					currentAbortController = null;
				}
			}
		}
	};
}

// Parse hierarchy from beads IDs
// orch-go-X.1 is child of orch-go-X
// orch-go-X.1.2 is child of orch-go-X.1
export function parseParentId(id: string): string | undefined {
	const parts = id.split('.');
	if (parts.length <= 1) {
		return undefined; // No parent (top-level)
	}
	// Remove last part to get parent ID
	return parts.slice(0, -1).join('.');
}

// Build tree structure from flat nodes
export function buildTree(nodes: GraphNode[], edges: GraphEdge[]): TreeNode[] {
	// Create tree nodes with initial state
	const treeNodes: Map<string, TreeNode> = new Map();
	
	for (const node of nodes) {
		const parentId = parseParentId(node.id);
		treeNodes.set(node.id, {
			...node,
			children: [],
			depth: 0,
			expanded: true, // Children expanded by default for Phase 1
			details_expanded: false, // L1 details collapsed by default
			blocked_by: [],
			blocks: [],
			parent_id: parentId
		});
	}

	// Build blocking relationships from edges
	for (const edge of edges) {
		if (edge.type === 'blocks') {
			const fromNode = treeNodes.get(edge.from);
			const toNode = treeNodes.get(edge.to);
			if (fromNode && toNode) {
				// edge.from blocks edge.to
				toNode.blocked_by.push(edge.from);
				fromNode.blocks.push(edge.to);
			}
		}
	}

	// Apply parent-child edges from API (set via 'bd update --parent')
	// These override ID-pattern hierarchy when explicit parent-child edges exist
	for (const edge of edges) {
		if (edge.type === '' || edge.type === 'parent-child') {
			const childNode = treeNodes.get(edge.from);
			const parentNode = treeNodes.get(edge.to);
			if (childNode && parentNode) {
				// edge.from is child, edge.to is parent
				childNode.parent_id = edge.to;
			}
		}
	}

	// Build parent-child hierarchy
	const roots: TreeNode[] = [];
	
	for (const node of treeNodes.values()) {
		if (node.parent_id) {
			const parent = treeNodes.get(node.parent_id);
			if (parent) {
				parent.children.push(node);
				node.depth = parent.depth + 1;
			} else {
				// Parent doesn't exist in dataset, treat as root
				roots.push(node);
			}
		} else {
			// No parent, it's a root
			roots.push(node);
		}
	}

	// Sort children by ID (maintains creation order)
	for (const node of treeNodes.values()) {
		node.children.sort((a, b) => a.id.localeCompare(b.id));
	}

	// Sort roots by priority, then by ID
	roots.sort((a, b) => {
		if (a.priority !== b.priority) {
			return a.priority - b.priority;
		}
		return a.id.localeCompare(b.id);
	});

	return roots;
}

export const workGraph = createWorkGraphStore();
