<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Badge } from '$lib/components/ui/badge';
	import type { TreeNode } from '$lib/stores/work-graph';

	export let tree: TreeNode[] = [];

	// Flatten tree for keyboard navigation
	let flattenedNodes: TreeNode[] = [];
	let selectedIndex = 0;

	// Flatten tree respecting expansion state
	function flattenTree(nodes: TreeNode[], result: TreeNode[] = []): TreeNode[] {
		for (const node of nodes) {
			result.push(node);
			if (node.expanded && node.children.length > 0) {
				flattenTree(node.children, result);
			}
		}
		return result;
	}

	// Rebuild flattened list when tree or expansion state changes
	$: {
		flattenedNodes = flattenTree(tree);
		// Clamp selected index to valid range
		if (selectedIndex >= flattenedNodes.length) {
			selectedIndex = Math.max(0, flattenedNodes.length - 1);
		}
	}

	// Get status icon
	function getStatusIcon(status: string): string {
		switch (status.toLowerCase()) {
			case 'in_progress': return '▶';
			case 'blocked': return '🚫';
			case 'open': return '○';
			case 'closed': return '✓';
			case 'complete': return '✓';
			default: return '•';
		}
	}

	// Get status color
	function getStatusColor(status: string): string {
		switch (status.toLowerCase()) {
			case 'in_progress': return 'text-blue-500';
			case 'blocked': return 'text-red-500';
			case 'open': return 'text-muted-foreground';
			case 'closed': return 'text-green-500';
			case 'complete': return 'text-green-500';
			default: return 'text-muted-foreground';
		}
	}

	// Get priority badge variant
	function getPriorityVariant(priority: number): 'destructive' | 'secondary' | 'outline' {
		if (priority === 0) return 'destructive';
		if (priority === 1) return 'secondary';
		return 'outline';
	}

	// Get type badge color
	function getTypeBadge(type: string): string {
		switch (type.toLowerCase()) {
			case 'epic': return 'bg-purple-500/10 text-purple-500';
			case 'feature': return 'bg-blue-500/10 text-blue-500';
			case 'bug': return 'bg-red-500/10 text-red-500';
			case 'task': return 'bg-green-500/10 text-green-500';
			case 'question': return 'bg-yellow-500/10 text-yellow-500';
			default: return 'bg-muted text-muted-foreground';
		}
	}

	// Get age from ID (creation timestamp encoded in beads IDs)
	function getAge(id: string): string {
		// Beads IDs have timestamp - could parse it, but for now just return placeholder
		// This would need actual created_at from API
		return ''; // TODO: Add created_at to GraphNode
	}

	// Keyboard navigation handlers
	function handleKeyDown(event: KeyboardEvent) {
		const current = flattenedNodes[selectedIndex];
		if (!current) return;

		switch (event.key) {
			case 'j':
			case 'ArrowDown':
				event.preventDefault();
				selectedIndex = Math.min(selectedIndex + 1, flattenedNodes.length - 1);
				scrollToSelected();
				break;

			case 'k':
			case 'ArrowUp':
				event.preventDefault();
				selectedIndex = Math.max(selectedIndex - 1, 0);
				scrollToSelected();
				break;

		case 'l':
		case 'ArrowRight':
		case 'Enter':
			event.preventDefault();
			// Toggle L1 details expansion
			current.details_expanded = !current.details_expanded;
			tree = [...tree]; // Create new reference to trigger reactivity
			break;

		case 'h':
		case 'ArrowLeft':
		case 'Escape':
			event.preventDefault();
			if (current.details_expanded) {
				// Close L1 details
				current.details_expanded = false;
				tree = [...tree]; // Create new reference to trigger reactivity
			} else if (current.parent_id) {
				// Jump to parent
				const parentIdx = flattenedNodes.findIndex(n => n.id === current.parent_id);
				if (parentIdx !== -1) {
					selectedIndex = parentIdx;
					scrollToSelected();
				}
			}
			break;

			case 'g':
				event.preventDefault();
				selectedIndex = 0;
				scrollToSelected();
				break;

			case 'G':
				event.preventDefault();
				selectedIndex = flattenedNodes.length - 1;
				scrollToSelected();
				break;
		}
	}

	// Scroll selected item into view
	function scrollToSelected() {
		const element = document.querySelector(`[data-node-index="${selectedIndex}"]`);
		if (element) {
			element.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
		}
	}

	// Focus management - auto-focus on mount
	let containerElement: HTMLDivElement;

	onMount(() => {
		// Use setTimeout to ensure DOM is fully rendered
		setTimeout(() => {
			containerElement?.focus();
		}, 100);
	});

	// Toggle expansion
	function toggleExpansion(node: TreeNode) {
		node.expanded = !node.expanded;
		tree = [...tree]; // Create new reference to trigger reactivity
	}

	// Select node on click
	function selectNode(index: number) {
		selectedIndex = index;
	}
</script>

<div
	bind:this={containerElement}
	class="work-graph-tree h-full overflow-y-auto px-6 py-4 focus:outline-none"
	role="tree"
	tabindex="0"
	on:keydown={handleKeyDown}
>
	{#each flattenedNodes as node, index (node.id)}
		<div
			data-testid="issue-row-{node.id}"
			data-node-index={index}
			class="node-row cursor-pointer select-none"
			class:selected={index === selectedIndex}
			class:focused={index === selectedIndex}
			role="treeitem"
			aria-selected={index === selectedIndex}
			tabindex="-1"
			on:click={() => selectNode(index)}
			on:keydown={(e) => e.key === 'Enter' && selectNode(index)}
		>
			<!-- L0: Row -->
			<div
				class="flex items-center gap-3 py-2 px-3 rounded hover:bg-accent/50 transition-colors"
				class:bg-accent={index === selectedIndex}
				style="padding-left: {node.depth * 24 + 12}px"
			>
				<!-- Expansion indicator -->
				<span class="w-4 text-muted-foreground text-xs">
					{#if node.children.length > 0}
						{node.expanded ? '▼' : '▶'}
					{:else}
						<span class="opacity-0">•</span>
					{/if}
				</span>

				<!-- Status icon -->
				<span data-testid="status-icon" class="w-5 {getStatusColor(node.status)}">
					{getStatusIcon(node.status)}
				</span>

				<!-- Priority badge -->
				<Badge data-testid="priority-badge" variant={getPriorityVariant(node.priority)} class="w-8 justify-center text-xs">
					P{node.priority}
				</Badge>

				<!-- ID -->
				<span class="text-xs font-mono text-muted-foreground min-w-[120px]">
					{node.id}
				</span>

				<!-- Title -->
				<span class="flex-1 text-sm font-medium text-foreground">
					{node.title}
				</span>

				<!-- Type badge -->
				<Badge data-testid="type-badge" variant="outline" class="{getTypeBadge(node.type)} text-xs">
					{node.type}
				</Badge>

				<!-- Age (placeholder) -->
				{#if getAge(node.id)}
					<span class="text-xs text-muted-foreground min-w-[40px] text-right">
						{getAge(node.id)}
					</span>
				{/if}
			</div>

			<!-- L1: Expanded details -->
			{#if node.details_expanded}
				<div
					class="expanded-details ml-12 mt-1 mb-2 p-3 bg-muted/30 rounded text-sm"
					style="margin-left: {node.depth * 24 + 48}px"
				>
					<!-- Description preview -->
					{#if node.description}
						<div class="text-muted-foreground mb-2">
							<span class="text-xs font-semibold uppercase text-foreground">Description:</span>
							<p class="mt-1 text-xs">{node.description}</p>
						</div>
					{/if}

					<!-- Blocking relationships -->
					{#if node.blocked_by.length > 0}
						<div class="mb-2">
							<span class="text-xs font-semibold uppercase text-red-500">Blocked by:</span>
							<ul class="mt-1 space-y-1">
								{#each node.blocked_by as blocker}
									<li class="text-xs text-muted-foreground">→ {blocker}</li>
								{/each}
							</ul>
						</div>
					{/if}

					{#if node.blocks.length > 0}
						<div>
							<span class="text-xs font-semibold uppercase text-yellow-500">Blocks:</span>
							<ul class="mt-1 space-y-1">
								{#each node.blocks as blocked}
									<li class="text-xs text-muted-foreground">→ {blocked}</li>
								{/each}
							</ul>
						</div>
					{/if}

					{#if node.blocked_by.length === 0 && node.blocks.length === 0}
						<div class="text-xs text-muted-foreground">No blocking relationships</div>
					{/if}
				</div>
			{/if}
		</div>
	{/each}
</div>

<style>
	.node-row.selected {
		/* Selected row is already highlighted with bg-accent */
	}

	.work-graph-tree {
		/* Ensure keyboard focus works */
		min-height: 100%;
	}
</style>
