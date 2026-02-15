<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { knowledgeTree, type TreeView, type KnowledgeNode, type NodeType } from '$lib/stores/knowledge-tree';
	import { KnowledgeTree as KnowledgeTreeComponent } from '$lib/components/knowledge-tree';
	
	let currentView: TreeView = 'knowledge';
	let loading = true;
	let searchQuery = '';
	let selectedTypes: Set<NodeType> = new Set();
	
	// Load initial tree
	onMount(async () => {
		await knowledgeTree.fetch(currentView);
		knowledgeTree.connectSSE(currentView);
		loading = false;
	});
	
	// Cleanup SSE on unmount
	onDestroy(() => {
		knowledgeTree.disconnectSSE();
	});
	
	// Handle view toggle
	async function handleViewToggle() {
		loading = true;
		currentView = currentView === 'knowledge' ? 'work' : 'knowledge';
		
		// Disconnect old SSE, fetch new tree, reconnect SSE
		knowledgeTree.disconnectSSE();
		await knowledgeTree.fetch(currentView);
		knowledgeTree.connectSSE(currentView);
		loading = false;
	}
	
	// Handle node toggle
	function handleNodeToggle(nodeId: string) {
		knowledgeTree.toggleNode(nodeId);
	}
	
	// Filter tree by search and type
	function filterTree(node: KnowledgeNode | null): KnowledgeNode | null {
		if (!node) return null;
		
		// Check if this node matches filters
		const matchesSearch = searchQuery === '' || 
			node.Title.toLowerCase().includes(searchQuery.toLowerCase());
		
		const matchesType = selectedTypes.size === 0 || 
			selectedTypes.has(node.Type);
		
		// Filter children recursively
		const filteredChildren = node.Children
			?.map(child => filterTree(child))
			.filter(child => child !== null) as KnowledgeNode[] || [];
		
		// Include node if it matches OR has matching children
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
		selectedTypes = selectedTypes; // Trigger reactivity
	}
	
	// Get filtered root nodes
	$: filteredTree = $knowledgeTree.tree ? filterTree($knowledgeTree.tree) : null;
	$: rootChildren = filteredTree?.Children || [];
	
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
				{currentView === 'knowledge' ? '📚 Knowledge' : '⚙️ Work'}
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
			
			<div class="text-xs text-muted-foreground ml-auto">
				{rootChildren.length} {currentView === 'knowledge' ? 'clusters' : 'issues'}
			</div>
		</div>
	</div>
	
	<!-- Content -->
	<div class="flex-1 overflow-auto">
		{#if loading}
			<div class="flex items-center justify-center h-full">
				<div class="text-muted-foreground">Loading tree...</div>
			</div>
		{:else if $knowledgeTree.error}
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
					/>
				{/each}
			</div>
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
