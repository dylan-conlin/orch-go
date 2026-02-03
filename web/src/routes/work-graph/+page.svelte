<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
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

	let tree: TreeNode[] = [];
	let loading = true;
	let error: string | null = null;
	let currentView: 'issues' | 'artifacts' = 'issues';
	let refreshInterval: ReturnType<typeof setInterval> | null = null;
	let previousIssueIds = new Set<string>();
	let newIssueIds = new Set<string>();
	let completedIssues: CompletedIssue[] = [];

	// Fetch work graph and agents on mount, connect to SSE for real-time updates
	onMount(async () => {
		// Start orchestratorContext polling (2 seconds like old dashboard)
		orchestratorContext.startPolling(2000);

		const projectDir = $orchestratorContext?.project_dir;
		await Promise.all([
			workGraph.fetch(projectDir, 'open'),
			agents.fetch(),
			attention.fetch() // Fetch attention signals and completed issues
		]);

		// Fetch WIP and daemon data (non-blocking)
		wip.fetchQueued(projectDir).catch(console.error);
		daemon.fetch().catch(console.error);
		
		loading = false;
		
		// Initialize previousIssueIds from initial fetch
		if ($workGraph?.nodes) {
			previousIssueIds = new Set($workGraph.nodes.map(n => n.id));
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
	});

	// Subscribe to attention store for completed issues
	$: if ($attention) {
		completedIssues = $attention.completedIssues;
	}

	// Rebuild tree whenever graph data OR wip data OR attention changes, filtering out queued issues
	// Note: $wip dependency ensures filter re-runs when queued issues load
	$: if ($workGraph && !$workGraph.error && $wip) {
		// Build set of queued issue IDs for fast lookup
		// Handle case where queuedIssues might not be loaded yet
		const queuedIds = new Set(($wip.queuedIssues || []).map(issue => issue.id));

		// Filter out queued issues from nodes before building tree
		// Main tree shows 'everything NOT currently in the pipeline'
		const filteredNodes = $workGraph.nodes.filter(node => !queuedIds.has(node.id));

		tree = buildTree(filteredNodes, $workGraph.edges);

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
		
		// Track newly appeared issues for highlighting (using filtered nodes)
		if (filteredNodes) {
			const currentIssueIds = new Set(filteredNodes.map(n => n.id));
			
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
		}
	} else if ($workGraph?.error) {
		error = $workGraph.error;
		tree = [];
	}
	
	// Re-fetch workGraph and kbArtifacts when orchestrator project_dir changes
	$: {
		if (typeof window !== 'undefined' && $orchestratorContext.project_dir) {
			workGraph.fetch($orchestratorContext.project_dir, 'open').catch(console.error);
			// Also re-fetch kbArtifacts if we're in artifacts view
			if (currentView === 'artifacts' && $kbArtifacts) {
				kbArtifacts.fetch($orchestratorContext.project_dir, '7d').catch(console.error);
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
							Structure view - Navigate with j/k/l/h, expand with l/enter, collapse with h/esc
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
				<WorkGraphTree {tree} {newIssueIds} wipItems={$wipItems} {completedIssues} />
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


