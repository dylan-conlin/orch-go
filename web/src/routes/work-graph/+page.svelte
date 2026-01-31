<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { workGraph, buildTree, type TreeNode } from '$lib/stores/work-graph';
	import { orchestratorContext } from '$lib/stores/context';
	import { WorkGraphTree } from '$lib/components/work-graph-tree';

	let tree: TreeNode[] = [];
	let loading = true;
	let error: string | null = null;

	// Fetch work graph on mount
	onMount(async () => {
		const projectDir = $orchestratorContext?.project_dir;
		await workGraph.fetch(projectDir, 'open');
		loading = false;
	});

	// Rebuild tree whenever graph data changes
	$: if ($workGraph && !$workGraph.error) {
		tree = buildTree($workGraph.nodes, $workGraph.edges);
		error = null;
	} else if ($workGraph?.error) {
		error = $workGraph.error;
		tree = [];
	}

	// Keyboard navigation is handled by WorkGraphTree component
</script>

<div class="work-graph-container flex flex-col h-screen bg-background">
	<!-- Header -->
	<div class="border-b border-border px-6 py-4">
		<div class="flex items-center justify-between">
			<div>
				<h1 class="text-2xl font-semibold text-foreground">Work Graph</h1>
				<p class="text-sm text-muted-foreground mt-1">
					Structure view - Navigate with j/k/l/h, expand with l/enter, collapse with h/esc
				</p>
			</div>
			{#if $workGraph}
				<div class="flex gap-4 text-sm text-muted-foreground">
					<span>{$workGraph.node_count} issues</span>
					<span>{$workGraph.edge_count} edges</span>
				</div>
			{/if}
		</div>
	</div>

	<!-- Content -->
	<div class="flex-1 overflow-hidden">
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
			<WorkGraphTree {tree} />
		{/if}
	</div>
</div>

<style>
	.work-graph-container {
		max-height: 100vh;
	}
</style>
