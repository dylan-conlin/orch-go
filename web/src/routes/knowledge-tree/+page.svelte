<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { knowledgeTree, type TreeView, type KnowledgeNode, type NodeType, type NodeAnimation } from '$lib/stores/knowledge-tree';
	import { timelineStore } from '$lib/stores/timeline';
	import { KnowledgeTree as KnowledgeTreeComponent } from '$lib/components/knowledge-tree';
	import { SessionGroup } from '$lib/components/timeline';
	import type { ConnectionStatus } from '$lib/services/sse-connection';

	// localStorage keys
	const EXPANSION_STATE_KEY = 'knowledge-tree-expansion';
	const VIEW_STATE_KEY = 'knowledge-tree-view';

	type ViewMode = 'knowledge' | 'timeline';

	// Load initial view from URL hash or localStorage
	function loadInitialView(): ViewMode {
		if (typeof window === 'undefined') return 'knowledge';
		
		// 1. Check URL hash first (enables bookmarking)
		const hash = window.location.hash.slice(1); // Remove #
		if (hash === 'knowledge' || hash === 'timeline') {
			return hash as ViewMode;
		}
		// Legacy: redirect 'work' to 'knowledge'
		if (hash === 'work') {
			return 'knowledge';
		}
		
		// 2. Fall back to localStorage
		try {
			const stored = localStorage.getItem(VIEW_STATE_KEY);
			if (stored === 'knowledge' || stored === 'timeline') {
				return stored as ViewMode;
			}
			// Legacy: redirect 'work' to 'knowledge'
			if (stored === 'work') {
				return 'knowledge';
			}
		} catch (e) {
			console.error('Failed to load view state:', e);
		}
		
		// 3. Default to knowledge view
		return 'knowledge';
	}

	// Save view to both URL hash and localStorage
	function saveView(view: ViewMode) {
		if (typeof window === 'undefined') return;
		
		// Update URL hash (enables bookmarking)
		window.location.hash = view;
		
		// Also save to localStorage as fallback
		try {
			localStorage.setItem(VIEW_STATE_KEY, view);
		} catch (e) {
			console.error('Failed to save view state:', e);
		}
	}

	let currentView: ViewMode = loadInitialView();
	let treeView: TreeView = 'knowledge'; // Tree API always uses knowledge view
	let loading = true;
	let searchQuery = '';
	let selectedTypes: Set<NodeType> = new Set();
	let sseStatus: ConnectionStatus = 'disconnected';
	let sseStatusUnsubscribe: (() => void) | null = null;
	let scrollContainer: HTMLDivElement;
	let searchInput: HTMLInputElement;
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
		// Load data for initial view
		if (currentView === 'timeline') {
			await timelineStore.fetch(undefined, 10);
			timelineStore.connectSSE(undefined, 10);
		} else {
			await knowledgeTree.fetch(treeView);
			knowledgeTree.connectSSE(treeView);
			subscribeSSEStatus();
		}
		
		// Listen for hash changes (browser back/forward)
		window.addEventListener('hashchange', handleHashChange);
		
		loading = false;
	});

	// Keyboard shortcuts
	function handleKeydown(e: KeyboardEvent) {
		// Don't capture when typing in an input
		const target = e.target as HTMLElement;
		const isInput = target.tagName === 'INPUT' || target.tagName === 'TEXTAREA';

		if (e.key === '/' && !isInput) {
			e.preventDefault();
			searchInput?.focus();
		} else if (e.key === 'Escape' && isInput) {
			(target as HTMLInputElement).blur();
		} else if (e.key === 'Tab' && !isInput) {
			e.preventDefault();
			handleViewToggle();
		}
	}

	// Cleanup on unmount
	onDestroy(() => {
		knowledgeTree.disconnectSSE();
		timelineStore.disconnectSSE();
		if (sseStatusUnsubscribe) sseStatusUnsubscribe();
		animationUnsubscribe();
		if (typeof window !== 'undefined') {
			window.removeEventListener('hashchange', handleHashChange);
		}
	});

	// Handle view toggle - cycles through knowledge → timeline → knowledge
	async function handleViewToggle() {
		loading = true;

		// Cycle between knowledge and timeline
		currentView = currentView === 'knowledge' ? 'timeline' : 'knowledge';

		// Save view to hash and localStorage
		saveView(currentView);

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

	// Handle browser back/forward navigation via hash changes
	async function handleHashChange() {
		const newView = loadInitialView();
		if (newView !== currentView) {
			loading = true;
			currentView = newView;

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

	// Clear all filters
	function clearFilters() {
		searchQuery = '';
		selectedTypes = new Set();
	}

	// Count nodes by type in the unfiltered tree
	function countNodesByType(node: KnowledgeNode | null): Map<NodeType, number> {
		const counts = new Map<NodeType, number>();
		if (!node) return counts;

		function walk(n: KnowledgeNode) {
			counts.set(n.Type, (counts.get(n.Type) || 0) + 1);
			n.Children?.forEach(walk);
		}
		walk(node);
		return counts;
	}

	// Filtered tree derived from store data
	// NOTE: searchQuery and selectedTypes must appear directly in the reactive
	// expression so Svelte tracks them as dependencies (accessing them only inside
	// filterTree via closure is invisible to the compiler).
	$: filteredTree = (searchQuery, selectedTypes, $knowledgeTree.tree) ? filterTree($knowledgeTree.tree) : null;
	$: rootChildren = filteredTree?.Children || [];

	// Filter state
	$: hasActiveFilters = searchQuery !== '' || selectedTypes.size > 0;
	$: typeCounts = countNodesByType($knowledgeTree.tree);

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

<svelte:window on:keydown={handleKeydown} />

<div class="knowledge-tree-page flex flex-col -mx-4 -mt-3" style="height: calc(100vh - 2.5rem);">
	<!-- Header -->
	<div class="border-b border-border px-4 py-3">
		<div class="flex items-center gap-4">
			<!-- View Toggle -->
			<button
				type="button"
				onclick={handleViewToggle}
				class="px-3 py-1.5 text-sm rounded border border-border hover:bg-zinc-800 transition-colors"
			>
				{currentView === 'knowledge' ? '📚 Knowledge' : '📅 Timeline'}
			</button>

			<!-- Search -->
			<input
				type="text"
				bind:value={searchQuery}
				bind:this={searchInput}
				placeholder="Search nodes... ( / )"
				class="px-3 py-1.5 text-sm bg-zinc-900 border border-border rounded flex-1 max-w-md"
			/>

			<!-- Type Filters -->
			<div class="flex gap-2 flex-wrap items-center">
				{#each nodeTypes as type}
					{@const count = typeCounts.get(type) || 0}
					{#if count > 0}
						<button
							type="button"
							onclick={() => toggleTypeFilter(type)}
							class="px-2 py-1 text-xs rounded border transition-colors {selectedTypes.has(type) ? 'bg-blue-500/20 border-blue-500/50 text-blue-400' : 'border-border hover:bg-zinc-800'}"
						>
							{type} <span class="opacity-50">{count}</span>
						</button>
					{/if}
				{/each}
				{#if hasActiveFilters}
					<button
						type="button"
						onclick={clearFilters}
						class="px-2 py-1 text-xs rounded border border-red-500/30 text-red-400 hover:bg-red-500/10 transition-colors"
					>
						clear
					</button>
				{/if}
			</div>

			<div class="flex items-center gap-2 ml-auto text-xs text-muted-foreground">
				{#if currentView === 'timeline'}
					<span>{$timelineStore.timeline?.total || 0} actions</span>
				{:else}
					<span>{rootChildren.length} clusters</span>
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
