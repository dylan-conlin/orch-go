<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { knowledgeTree, type TreeView, type KnowledgeNode, type NodeType, type NodeAnimation } from '$lib/stores/knowledge-tree';
	import { timelineStore } from '$lib/stores/timeline';
	import { KnowledgeTree as KnowledgeTreeComponent } from '$lib/components/knowledge-tree';
	import { SessionGroup } from '$lib/components/timeline';
	import type { ConnectionStatus } from '$lib/services/sse-connection';

	// localStorage key for expansion state
	const EXPANSION_STATE_KEY = 'knowledge-tree-expansion';

	type ViewMode = 'knowledge' | 'work' | 'timeline';

	let currentView: ViewMode = 'knowledge';
	let treeView: TreeView = 'knowledge'; // For knowledge tree API (only 'knowledge' or 'work')
	let loading = true;
	let searchQuery = '';
	let selectedTypes: Set<NodeType> = new Set();
	let sseStatus: ConnectionStatus = 'disconnected';
	let sseStatusUnsubscribe: (() => void) | null = null;
	let scrollContainer: HTMLDivElement;
	let animationStates: Map<string, NodeAnimation> = new Map();

	// Load expansion state from localStorage
	function loadExpansionState(): Set<string> {
		if (typeof window === 'undefined') return new Set();
		try {
			const stored = localStorage.getItem(EXPANSION_STATE_KEY);
			return stored ? new Set(JSON.parse(stored)) : new Set();
		} catch (e) {
			console.error('Failed to load expansion state:', e);
			return new Set();
		}
	}

	// Save expansion state to localStorage
	function saveExpansionState(expandedIds: Set<string>) {
		if (typeof window === 'undefined') return;
		try {
			localStorage.setItem(EXPANSION_STATE_KEY, JSON.stringify(Array.from(expandedIds)));
		} catch (e) {
			console.error('Failed to save expansion state:', e);
		}
	}

	// Expansion state is fully owned by this component, not by the store.
	// This makes it immune to tree data replacements from SSE updates.
	let expandedNodes = loadExpansionState();

	// Subscribe to SSE connection status
	function subscribeSSEStatus() {
		if (sseStatusUnsubscribe) sseStatusUnsubscribe();
		const statusStore = knowledgeTree.getSSEStatus();
		if (statusStore) {
			sseStatusUnsubscribe = statusStore.subscribe(s => { sseStatus = s; });
		}
	}

	// Subscribe to animation states
	const animationStatesStore = knowledgeTree.getAnimationStates();
	const animationUnsubscribe = animationStatesStore.subscribe(states => {
		animationStates = states;
	});

	// Expanded sessions in timeline view
	let expandedSessions = new Set<string>();

	// Load initial tree
	onMount(async () => {
		await knowledgeTree.fetch(treeView);
		knowledgeTree.connectSSE(treeView);
		subscribeSSEStatus();
		loading = false;
	});

	// Cleanup on unmount
	onDestroy(() => {
		knowledgeTree.disconnectSSE();
		timelineStore.disconnectSSE();
		if (sseStatusUnsubscribe) sseStatusUnsubscribe();
		animationUnsubscribe();
	});

	// Handle view toggle - cycles through knowledge → work → timeline → knowledge
	async function handleViewToggle() {
		loading = true;

		// Cycle through views
		if (currentView === 'knowledge') {
			currentView = 'work';
			treeView = 'work';
		} else if (currentView === 'work') {
			currentView = 'timeline';
		} else {
			currentView = 'knowledge';
			treeView = 'knowledge';
		}

		// Disconnect previous SSE connections
		knowledgeTree.disconnectSSE();
		timelineStore.disconnectSSE();

		// Load data for new view
		if (currentView === 'timeline') {
			await timelineStore.fetch(undefined, 10);
			timelineStore.connectSSE(undefined, 10);
		} else {
			await knowledgeTree.fetch(treeView);
			knowledgeTree.connectSSE(treeView);
			subscribeSSEStatus();
		}

		loading = false;
	}

	// Toggle session expansion in timeline view
	function toggleSession(sessionID: string) {
		if (expandedSessions.has(sessionID)) {
			expandedSessions.delete(sessionID);
		} else {
			expandedSessions.add(sessionID);
		}
		expandedSessions = expandedSessions; // Trigger reactivity
	}

	// Handle node toggle - purely local state, independent of store
	function handleNodeToggle(nodeId: string) {
		if (expandedNodes.has(nodeId)) {
			expandedNodes.delete(nodeId);
		} else {
			expandedNodes.add(nodeId);
		}
		expandedNodes = expandedNodes; // Trigger Svelte reactivity
		saveExpansionState(expandedNodes);
	}

	// Sort children by ID for stable ordering across SSE updates.
	// Backend may return clusters in different order based on filesystem mod times.
	function stableSort(children: KnowledgeNode[]): KnowledgeNode[] {
		return [...children].sort((a, b) => a.ID.localeCompare(b.ID));
	}

	// Filter tree by search and type
	function filterTree(node: KnowledgeNode | null): KnowledgeNode | null {
		if (!node) return null;

		const matchesSearch = searchQuery === '' ||
			node.Title.toLowerCase().includes(searchQuery.toLowerCase());

		const matchesType = selectedTypes.size === 0 ||
			selectedTypes.has(node.Type);

		const filteredChildren = stableSort(
			node.Children
				?.map(child => filterTree(child))
				.filter(child => child !== null) as KnowledgeNode[] || []
		);

		if (matchesSearch && matchesType) {
			return { ...node, Children: filteredChildren };
		} else if (filteredChildren.length > 0) {
			return { ...node, Children: filteredChildren };
		}

		return null;
	}

	// Toggle type filter
	function toggleTypeFilter(type: NodeType) {
		if (selectedTypes.has(type)) {
			selectedTypes.delete(type);
		} else {
			selectedTypes.add(type);
		}
		selectedTypes = selectedTypes;
	}

	// Filtered tree derived from store data
	$: filteredTree = $knowledgeTree.tree ? filterTree($knowledgeTree.tree) : null;
	$: rootChildren = filteredTree?.Children || [];

	// SSE status indicator
	$: statusColor = sseStatus === 'connected' ? 'bg-green-500' : sseStatus === 'connecting' ? 'bg-yellow-500' : 'bg-red-500';
	$: statusLabel = sseStatus === 'connected' ? 'Live' : sseStatus === 'connecting' ? 'Connecting...' : 'Disconnected';

	// Available node types for filtering
	const nodeTypes: NodeType[] = [
		'investigation',
		'decision',
		'model',
		'guide',
		'issue',
		'probe',
		'cluster'
	];
</script>

<div class="knowledge-tree-page flex flex-col h-screen bg-background">
	<!-- Header -->
	<div class="border-b border-border px-4 py-3">
		<div class="flex items-center gap-4">
			<!-- View Toggle -->
			<button
				type="button"
				onclick={handleViewToggle}
				class="px-3 py-1.5 text-sm rounded border border-border hover:bg-zinc-800 transition-colors"
			>
				{currentView === 'knowledge' ? '📚 Knowledge' : currentView === 'work' ? '⚙️ Work' : '📅 Timeline'}
			</button>

			<!-- Search -->
			<input
				type="text"
				bind:value={searchQuery}
				placeholder="Search nodes..."
				class="px-3 py-1.5 text-sm bg-zinc-900 border border-border rounded flex-1 max-w-md"
			/>

			<!-- Type Filters -->
			<div class="flex gap-2 flex-wrap">
				{#each nodeTypes as type}
					<button
						type="button"
						onclick={() => toggleTypeFilter(type)}
						class="px-2 py-1 text-xs rounded border transition-colors {selectedTypes.has(type) ? 'bg-blue-500/20 border-blue-500/50 text-blue-400' : 'border-border hover:bg-zinc-800'}"
					>
						{type}
					</button>
				{/each}
			</div>

			<div class="flex items-center gap-2 ml-auto text-xs text-muted-foreground">
				{#if currentView === 'timeline'}
					<span>{$timelineStore.timeline?.total || 0} actions</span>
				{:else}
					<span>{rootChildren.length} {currentView === 'knowledge' ? 'clusters' : 'issues'}</span>
				{/if}
				<span class="flex items-center gap-1" title={statusLabel}>
					<span class="inline-block w-2 h-2 rounded-full {statusColor}"></span>
					<span class="text-[10px]">{statusLabel}</span>
				</span>
			</div>
		</div>
	</div>

	<!-- Content -->
	<div class="flex-1 overflow-auto" bind:this={scrollContainer}>
		{#if loading}
			<div class="flex items-center justify-center h-full">
				<div class="text-muted-foreground">Loading...</div>
			</div>
		{:else if currentView === 'timeline'}
			<!-- Timeline View -->
			{#if $timelineStore.error}
				<div class="flex items-center justify-center h-full">
					<div class="text-red-500">Error: {$timelineStore.error}</div>
				</div>
			{:else if !$timelineStore.timeline || $timelineStore.timeline.sessions.length === 0}
				<div class="flex items-center justify-center h-full">
					<div class="text-muted-foreground">No timeline data</div>
				</div>
			{:else}
				<div class="py-4 px-2">
					{#each $timelineStore.timeline.sessions as session (session.session_id)}
						<SessionGroup
							{session}
							expanded={expandedSessions.has(session.session_id)}
							onToggle={() => toggleSession(session.session_id)}
						/>
					{/each}
				</div>
			{/if}
		{:else}
			<!-- Tree View (knowledge or work) -->
			{#if $knowledgeTree.error && !$knowledgeTree.tree}
				<div class="flex items-center justify-center h-full">
					<div class="text-red-500">Error: {$knowledgeTree.error}</div>
				</div>
			{:else if rootChildren.length === 0}
				<div class="flex items-center justify-center h-full">
					<div class="text-muted-foreground">
						{searchQuery || selectedTypes.size > 0 ? 'No nodes match filters' : 'No nodes found'}
					</div>
				</div>
			{:else}
				<div class="py-2">
					{#each rootChildren as child (child.ID)}
						<KnowledgeTreeComponent
							node={child}
							depth={0}
							onToggle={handleNodeToggle}
							{expandedNodes}
							{animationStates}
						/>
					{/each}
				</div>
			{/if}
		{/if}
	</div>

	<!-- Footer -->
	<div class="h-9 px-2 flex items-center justify-center border-t border-zinc-800 bg-zinc-950 text-zinc-500 text-[11px] font-mono">
		<span class="tracking-wide">
			<span class="text-zinc-400">Tab</span> toggle view
			<span class="mx-3">·</span>
			<span class="text-zinc-400">/</span> search
			<span class="mx-3">·</span>
			<span class="text-zinc-400">click</span> expand/collapse
		</span>
	</div>
</div>
