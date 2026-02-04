<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { derived } from 'svelte/store';
	import { workGraph, buildTree, type TreeNode, type AttentionBadgeType } from '$lib/stores/work-graph';
	import { kbArtifacts } from '$lib/stores/kb-artifacts';
	import { orchestratorContext, connectionStatus } from '$lib/stores/context';
	import { agents, connectSSE, disconnectSSE } from '$lib/stores/agents';
	import { WorkGraphTree } from '$lib/components/work-graph-tree';
	import { WIPSection } from '$lib/components/wip-section';
	import { ViewToggle } from '$lib/components/view-toggle';
	import { ArtifactFeed } from '$lib/components/artifact-feed';
	import { wip, wipItems } from '$lib/stores/wip';
	import { daemon } from '$lib/stores/daemon';
	import { attention, type CompletedIssue } from '$lib/stores/attention';
	
	// Derived store for project_dir to isolate reactivity
	// Only triggers reactive blocks when project_dir changes, not other context fields
	const projectDir = derived(orchestratorContext, $ctx => $ctx.project_dir);

	// Per-project seen issues tracking to prevent false highlights on project switch
	const SEEN_ISSUES_KEY = 'work-graph-seen-issues';
	
	interface SeenIssuesState {
		byProject: Record<string, {
			issueIds: string[];
			firstSeenAt: string; // ISO timestamp
		}>;
	}
	
	function loadSeenIssues(): SeenIssuesState {
		if (typeof window === 'undefined') return { byProject: {} };
		try {
			const stored = localStorage.getItem(SEEN_ISSUES_KEY);
			if (stored) {
				return JSON.parse(stored);
			}
		} catch (e) {
			console.error('Failed to load seen issues from localStorage:', e);
		}
		return { byProject: {} };
	}
	
	function saveSeenIssues(state: SeenIssuesState): void {
		if (typeof window === 'undefined') return;
		try {
			localStorage.setItem(SEEN_ISSUES_KEY, JSON.stringify(state));
		} catch (e) {
			console.error('Failed to save seen issues to localStorage:', e);
		}
	}

	let tree: TreeNode[] = [];
	let loading = true;
	let error: string | null = null;
	let currentView: 'issues' | 'artifacts' = 'issues';
	let refreshInterval: ReturnType<typeof setInterval> | null = null;
	let seenIssuesState: SeenIssuesState = { byProject: {} };
	let currentProjectDir: string | undefined = undefined;
	let projectChangeDebounceTimeout: ReturnType<typeof setTimeout> | null = null;
	let previousIssueIds = new Set<string>();
	let newIssueIds = new Set<string>();
	let completedIssues: CompletedIssue[] = [];
	
	// Track expansion state separately to preserve across tree rebuilds
	let expansionState = new Map<string, boolean>();
	
	// Debounce timeout for tree rebuild to batch rapid store updates
	let rebuildDebounceTimeout: ReturnType<typeof setTimeout> | null = null;
	let hasRenderedTree = false; // Skip debounce until first tree render completes

	// Fetch work graph and agents on mount, connect to SSE for real-time updates
	onMount(async () => {
		// Load seen issues from localStorage
		seenIssuesState = loadSeenIssues();
		
		// Start orchestratorContext polling (2 seconds like old dashboard)
		orchestratorContext.startPolling(2000);

		const projectDir = $orchestratorContext?.project_dir;
		currentProjectDir = projectDir;
		
		await Promise.all([
			workGraph.fetch(projectDir, 'open'),
			agents.fetch(),
			attention.fetch() // Fetch attention signals and completed issues
		]);

		// Fetch WIP and daemon data (non-blocking)
		wip.fetchQueued(projectDir).catch(console.error);
		daemon.fetch().catch(console.error);
		
		loading = false;
		
		// Initialize previousIssueIds from stored state OR initial fetch
		if (projectDir && seenIssuesState.byProject[projectDir]) {
			// Use stored state for this project
			previousIssueIds = new Set(seenIssuesState.byProject[projectDir].issueIds);
		} else if ($workGraph?.nodes) {
			// First time seeing this project - store all current issues as "seen"
			previousIssueIds = new Set($workGraph.nodes.map(n => n.id));
			if (projectDir) {
				seenIssuesState.byProject[projectDir] = {
					issueIds: Array.from(previousIssueIds),
					firstSeenAt: new Date().toISOString()
				};
				saveSeenIssues(seenIssuesState);
			}
		}
		
		// Connect to SSE for real-time agent updates (WIP section)
		connectSSE();
		
		// Poll workGraph periodically (5 seconds for faster updates)
		refreshInterval = setInterval(() => {
			const projectDir = $orchestratorContext?.project_dir;
			workGraph.fetch(projectDir, 'open').catch(console.error);
			wip.fetchQueued(projectDir).catch(console.error);
			daemon.fetch().catch(console.error);
			// Also poll kbArtifacts if in artifacts view
			if (currentView === 'artifacts' && $kbArtifacts) {
				kbArtifacts.fetch(projectDir, '7d').catch(console.error);
			}
		}, 5000);
	});

	// Sync running agents from agents store to WIP store
	$: wip.setRunningAgents($agents);

	// Disconnect SSE and stop polling on unmount
	onDestroy(() => {
		disconnectSSE();
		orchestratorContext.stopPolling();
		if (refreshInterval) {
			clearInterval(refreshInterval);
			refreshInterval = null;
		}
		if (projectChangeDebounceTimeout) {
			clearTimeout(projectChangeDebounceTimeout);
			projectChangeDebounceTimeout = null;
		}
		if (rebuildDebounceTimeout) {
			clearTimeout(rebuildDebounceTimeout);
			rebuildDebounceTimeout = null;
		}
		// Cancel any pending workGraph fetches
		workGraph.cancelPending();
	});

	// Subscribe to attention store for completed issues
	$: if ($attention) {
		completedIssues = $attention.completedIssues;
	}

	// Rebuild tree whenever graph data OR wip data OR attention changes, filtering out queued issues
	// Debounced to batch rapid updates and reduce CPU during polling
	// Skip debounce until first tree render completes for immediate display
	// Note: $wip dependency ensures filter re-runs when queued issues load
	$: if ($workGraph && !$workGraph.error && $wip) {
		// Cancel any pending rebuild
		if (rebuildDebounceTimeout) {
			clearTimeout(rebuildDebounceTimeout);
		}
		
		// Debounce rebuild to batch rapid updates (50ms is fast but still batches)
		const executeRebuild = () => {
			rebuildDebounceTimeout = null;
			
			// Build set of queued issue IDs for fast lookup
			// Handle case where queuedIssues might not be loaded yet
			const queuedIds = new Set(($wip.queuedIssues || []).map(issue => issue.id));

			// Filter out queued issues from nodes before building tree
			// Main tree shows 'everything NOT currently in the pipeline'
			const filteredNodes = $workGraph.nodes.filter(node => !queuedIds.has(node.id));

			tree = buildTree(filteredNodes, $workGraph.edges);
			
			// Mark that we've completed first render (enable debouncing for subsequent updates)
			hasRenderedTree = true;

			// Apply stored expansion state to preserve user's collapse/expand choices
			const applyExpansionState = (nodes: TreeNode[]) => {
				for (const node of nodes) {
					// If we have stored expansion state for this node, apply it
					// Otherwise keep the default from buildTree (which is expanded: true)
					if (expansionState.has(node.id)) {
						node.expanded = expansionState.get(node.id)!;
					} else {
						// First time seeing this node, store its default state
						expansionState.set(node.id, node.expanded);
					}
					// Recursively apply to children
					if (node.children.length > 0) {
						applyExpansionState(node.children);
					}
				}
			};
			applyExpansionState(tree);

			// Attach attention badges to tree nodes
			if ($attention?.signals) {
				const attachBadges = (nodes: TreeNode[]) => {
					for (const node of nodes) {
						const signal = $attention.signals.get(node.id);
						if (signal) {
							node.attentionBadge = signal.badge;
							node.attentionReason = signal.reason;
						}
						if (node.children.length > 0) {
							attachBadges(node.children);
						}
					}
				};
				attachBadges(tree);
			}

			error = null;
			
			// Track newly appeared issues for highlighting (use API nodes, not filtered nodes)
			// This prevents highlighting children when expanding parents (they were always in the API data)
			if ($workGraph.nodes) {
				const currentIssueIds = new Set($workGraph.nodes.map(n => n.id));
				const projectDir = $orchestratorContext?.project_dir;
				
				// Find issues that are new (in current but not in previous)
				for (const id of currentIssueIds) {
					if (!previousIssueIds.has(id) && !newIssueIds.has(id)) {
						newIssueIds.add(id);
						newIssueIds = newIssueIds; // Trigger reactivity
						// Remove highlight after 30 seconds
						setTimeout(() => {
							newIssueIds.delete(id);
							newIssueIds = newIssueIds; // Trigger reactivity
						}, 30000);
					}
				}
				
				// Update previousIssueIds for next comparison
				previousIssueIds = currentIssueIds;
				
				// Persist seen issues to localStorage for this project
				if (projectDir) {
					const existingFirstSeen = seenIssuesState.byProject[projectDir]?.firstSeenAt;
					seenIssuesState.byProject[projectDir] = {
						issueIds: Array.from(currentIssueIds),
						firstSeenAt: existingFirstSeen || new Date().toISOString()
					};
					saveSeenIssues(seenIssuesState);
				}
			}
		};
		
		// Execute immediately until first tree render, then debounce subsequent updates
		if (hasRenderedTree) {
			rebuildDebounceTimeout = setTimeout(executeRebuild, 50); // 50ms batches rapid updates
		} else {
			executeRebuild(); // Immediate for first render
		}
	} else if ($workGraph?.error) {
		error = $workGraph.error;
		tree = [];
	}
	
	// Re-fetch workGraph and kbArtifacts when orchestrator project_dir changes
	// Uses derived store to isolate reactivity (only fires when project_dir changes)
	// Uses debounce + abort to prevent flip-flopping between old/new project data
	$: {
		if (typeof window !== 'undefined' && $projectDir) {
			const newProjectDir = $projectDir;
			
			// Only react to actual project changes (not other context changes)
			if (newProjectDir !== currentProjectDir) {
				// Cancel any pending debounced fetch
				if (projectChangeDebounceTimeout) {
					clearTimeout(projectChangeDebounceTimeout);
				}
				
				// Cancel any in-flight workGraph requests immediately
				workGraph.cancelPending();
				
				// Update state synchronously to prevent stale comparisons
				currentProjectDir = newProjectDir;
				
				// Clear current highlights - they belong to the old project
				newIssueIds = new Set<string>();
				
				// Load seen issues for this project from localStorage
				if (seenIssuesState.byProject[newProjectDir]) {
					previousIssueIds = new Set(seenIssuesState.byProject[newProjectDir].issueIds);
				} else {
					// New project we haven't seen before - will be populated on first fetch
					previousIssueIds = new Set<string>();
				}
				
				// Debounce the actual fetch to wait for stable project value
				// 300ms prevents rapid flip-flopping while still feeling responsive
				projectChangeDebounceTimeout = setTimeout(() => {
					projectChangeDebounceTimeout = null;
					workGraph.fetch(newProjectDir, 'open').catch(console.error);
					// Also re-fetch kbArtifacts if we're in artifacts view
					if (currentView === 'artifacts' && $kbArtifacts) {
						kbArtifacts.fetch(newProjectDir, '7d').catch(console.error);
					}
				}, 300);
			}
		}
	}

	// Handle view toggle
	async function handleViewToggle(view: 'issues' | 'artifacts') {
		currentView = view;
		
		// Fetch artifacts when switching to artifacts view
		if (view === 'artifacts' && !$kbArtifacts) {
			const projectDir = $orchestratorContext?.project_dir;
			await kbArtifacts.fetch(projectDir, '7d');
		}
	}
	
	// Manual retry handler
	async function handleRetry() {
		await orchestratorContext.retry();
	}

	// Handle expansion state updates from tree component
	function handleToggleExpansion(nodeId: string, expanded: boolean) {
		expansionState.set(nodeId, expanded);
	}

	// Keyboard navigation for Tab to toggle views
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Tab' && !event.shiftKey) {
			event.preventDefault();
			currentView = currentView === 'issues' ? 'artifacts' : 'issues';
			handleViewToggle(currentView);
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="work-graph-container flex flex-col h-[calc(100vh-4rem)] overflow-hidden bg-background">
	<!-- Backend Error Banner -->
	{#if $connectionStatus.status === 'disconnected'}
		<div 
			class="bg-red-500/10 border-b border-red-500/20 px-4 py-3 flex items-center justify-between"
			data-testid="backend-error-banner"
		>
			<div class="flex-1 min-w-0">
				<p class="text-sm text-red-600 dark:text-red-400">
					<span class="font-semibold">Backend not running.</span>
					<span class="ml-2">Start with: <code class="bg-red-500/20 px-1 rounded text-xs">orch serve</code></span>
				</p>
			</div>
			<button
				type="button"
				onclick={handleRetry}
				class="ml-4 px-3 py-1 text-xs font-medium text-red-600 dark:text-red-400 border border-red-500/30 rounded hover:bg-red-500/10 transition-colors whitespace-nowrap"
				data-testid="retry-button"
			>
				Retry
			</button>
		</div>
	{/if}

	<!-- Header -->
	<div class="border-b border-border px-6 py-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-4">
				<div>
					<h1 class="text-2xl font-semibold text-foreground">Work Graph</h1>
					<p class="text-sm text-muted-foreground mt-1">
						{#if currentView === 'issues'}
							Structure view - Navigate with j/k, expand with l/enter, collapse with h/esc, close with x
						{:else}
							Artifact view - Navigate with j/k, open with l/enter, Tab to toggle
						{/if}
					</p>
				</div>
				<ViewToggle {currentView} onToggle={handleViewToggle} />
			</div>
			<div class="flex gap-4 text-sm text-muted-foreground">
				{#if currentView === 'issues' && $workGraph}
					<span>{$workGraph.node_count} issues</span>
					<span>{$workGraph.edge_count} edges</span>
				{:else if currentView === 'artifacts' && $kbArtifacts}
					<span>
						{($kbArtifacts.needs_decision?.length ?? 0) + ($kbArtifacts.recent?.length ?? 0)} artifacts
					</span>
				{/if}
				{#if $orchestratorContext?.project_dir}
					<span class="truncate max-w-xs">
						{$orchestratorContext.project_dir.split('/').pop()}
					</span>
				{/if}
			</div>
		</div>
	</div>

	<!-- Content -->
	<div class="flex-1 overflow-hidden">
		{#if currentView === 'issues'}
			{#if loading}
				<div class="flex items-center justify-center h-full">
					<div class="text-muted-foreground">Loading work graph...</div>
				</div>
			{:else if error}
				<div class="flex items-center justify-center h-full">
					<div class="text-red-500">Error: {error}</div>
				</div>
			{:else if tree.length === 0}
				<div class="flex items-center justify-center h-full">
					<div class="text-muted-foreground">No open issues found</div>
				</div>
			{:else}
				<WorkGraphTree 
					{tree} 
					{newIssueIds} 
					wipItems={$wipItems} 
					{completedIssues}
					onToggleExpansion={handleToggleExpansion}
				/>
			{/if}
		{:else}
			{#if $kbArtifacts?.error}
				<div class="flex items-center justify-center h-full">
					<div class="text-red-500">Error: {$kbArtifacts.error}</div>
				</div>
			{:else if !$kbArtifacts}
				<div class="flex items-center justify-center h-full">
					<div class="text-muted-foreground">Loading artifacts...</div>
				</div>
			{:else}
				<ArtifactFeed />
			{/if}
		{/if}
	</div>
</div>


