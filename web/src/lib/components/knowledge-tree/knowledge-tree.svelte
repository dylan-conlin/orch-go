<script lang="ts">
	import type { KnowledgeNode, NodeType, NodeStatus } from '$lib/stores/knowledge-tree';
	
	export let node: KnowledgeNode;
	export let depth: number = 0;
	export let onToggle: (nodeId: string) => void;
	
	// Node icon by type (from design doc)
	function getNodeIcon(type: NodeType): string {
		switch (type) {
			case 'investigation': return '◉';
			case 'decision': return '★';
			case 'model': return '◆';
			case 'guide': return '◈';
			case 'issue': return '●';
			case 'cluster': return '📁';
			case 'probe': return '◇';
			case 'postmortem': return '📋';
			case 'handoff': return '🤝';
			default: return '◦';
		}
	}
	
	// Color by type (from design doc)
	function getNodeColor(type: NodeType): string {
		switch (type) {
			case 'investigation': return 'text-green-400';
			case 'decision': return 'text-yellow-400';
			case 'model': return 'text-purple-400';
			case 'guide': return 'text-cyan-400';
			case 'issue': return 'text-orange-400';
			case 'cluster': return 'text-gray-400';
			case 'probe': return 'text-pink-400';
			default: return 'text-gray-500';
		}
	}
	
	// Status badge color
	function getStatusColor(status: NodeStatus): string {
		switch (status) {
			case 'complete': return 'bg-green-500/20 text-green-400';
			case 'in_progress': return 'bg-blue-500/20 text-blue-400';
			case 'triage': return 'bg-yellow-500/20 text-yellow-400';
			case 'closed': return 'bg-gray-500/20 text-gray-400';
			case 'open': return 'bg-orange-500/20 text-orange-400';
			default: return 'bg-gray-500/20 text-gray-400';
		}
	}
	
	function handleClick() {
		if (node.Children && node.Children.length > 0) {
			onToggle(node.ID);
		}
	}
	
	$: hasChildren = node.Children && node.Children.length > 0;
	$: isExpanded = node.expanded !== false; // Default to expanded
	$: indentClass = `pl-${Math.min(depth * 4, 16)}`;
</script>

<div class="tree-node">
	<!-- Node row -->
	<button
		type="button"
		class="w-full text-left px-2 py-1 hover:bg-zinc-800/50 flex items-center gap-2 text-sm {indentClass}"
		onclick={handleClick}
		data-node-id={node.ID}
	>
		<!-- Expand/collapse indicator -->
		{#if hasChildren}
			<span class="text-xs text-gray-500 w-3">
				{isExpanded ? '▼' : '▶'}
			</span>
		{:else}
			<span class="w-3"></span>
		{/if}
		
		<!-- Node icon -->
		<span class="text-base {getNodeColor(node.Type)}">
			{getNodeIcon(node.Type)}
		</span>
		
		<!-- Title -->
		<span class="flex-1 truncate text-gray-200">
			{node.Title}
		</span>
		
		<!-- Status badge -->
		{#if node.Status && node.Status !== 'complete'}
			<span class="text-xs px-2 py-0.5 rounded {getStatusColor(node.Status)}">
				{node.Status}
			</span>
		{/if}
		
		<!-- Date -->
		{#if node.Date}
			<span class="text-xs text-gray-500">
				{new Date(node.Date).toLocaleDateString()}
			</span>
		{/if}
	</button>
	
	<!-- Children (recursive) -->
	{#if hasChildren && isExpanded}
		{#each node.Children as child (child.ID)}
			<svelte:self node={child} depth={depth + 1} {onToggle} />
		{/each}
	{/if}
</div>

<style>
	/* Tailwind pl-* classes for indentation */
	.pl-0 { padding-left: 0rem; }
	.pl-4 { padding-left: 1rem; }
	.pl-8 { padding-left: 2rem; }
	.pl-12 { padding-left: 3rem; }
	.pl-16 { padding-left: 4rem; }
</style>
