<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Badge } from '$lib/components/ui/badge';
	import type { TreeNode } from '$lib/stores/work-graph';
	import type { WIPItem } from '$lib/stores/wip';
	import { getExpressiveStatus, computeAgentHealth, getContextPercent, getContextColor } from '$lib/stores/wip';

	export let tree: TreeNode[] = [];
	export let newIssueIds: Set<string> = new Set();
	export let wipItems: WIPItem[] = [];

	// Flatten tree for keyboard navigation
	let flattenedNodes: (TreeNode | WIPItem)[] = [];
	let selectedIndex = 0;
	
	// Track expanded details separately (fixes reactivity issues)
	let expandedDetails = new Set<string>();

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

	// Rebuild flattened list when tree or wipItems change
	$: {
		const treeNodes = flattenTree(tree);
		// Prepend WIP items to flattened list (WIP first, then main tree)
		flattenedNodes = [...wipItems, ...treeNodes];
		// Clamp selected index to valid range
		if (selectedIndex >= flattenedNodes.length) {
			selectedIndex = Math.max(0, flattenedNodes.length - 1);
		}
	}

	// Type guard to check if item is a WIPItem
	function isWIPItem(item: TreeNode | WIPItem): item is WIPItem {
		return 'type' in item && (item.type === 'running' || item.type === 'queued');
	}

	// Get ID from either WIPItem or TreeNode
	function getItemId(item: TreeNode | WIPItem): string {
		if (isWIPItem(item)) {
			return item.type === 'running' ? item.agent.id : item.issue.id;
		}
		return item.id;
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

	// Get status icon for running agents based on health (from WIP store)
	function getAgentStatusIcon(agent: any): { icon: string; color: string } {
		const health = computeAgentHealth(agent);
		
		if (health.status === 'critical') {
			return { icon: '🚨', color: 'text-red-500' };
		}
		if (health.status === 'warning') {
			return { icon: '⚠️', color: 'text-yellow-500' };
		}
		
		// Healthy - show activity-based icon
		if (agent.is_processing) {
			return { icon: '◉', color: 'text-blue-500 animate-pulse' };
		}
		if (agent.status === 'idle') {
			return { icon: '⏸', color: 'text-muted-foreground' };
		}
		return { icon: '▶', color: 'text-blue-500' };
	}

	// Keyboard navigation handlers
	function handleKeyDown(event: KeyboardEvent) {
		const current = flattenedNodes[selectedIndex];
		if (!current) return;

		const itemId = getItemId(current);
		const isWIP = isWIPItem(current);

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
			event.preventDefault();
			// Expand tree node if it has children (WIP items don't have tree expansion)
			if (!isWIP && current.children.length > 0) {
				toggleExpansion(current as TreeNode);
			}
			break;

		case 'Enter':
			event.preventDefault();
			// Toggle L1 details expansion (works for both WIP items and tree nodes)
			if (expandedDetails.has(itemId)) {
				expandedDetails.delete(itemId);
			} else {
				expandedDetails.add(itemId);
			}
			expandedDetails = expandedDetails; // Trigger reactivity
			break;

		case 'h':
		case 'ArrowLeft':
			event.preventDefault();
			// Collapse tree node if it has children and is expanded (WIP items don't have tree collapse)
			if (!isWIP && current.children.length > 0 && current.expanded) {
				toggleExpansion(current as TreeNode);
			} else if (!isWIP && current.parent_id) {
				// Jump to parent if no children to collapse
				const parentIdx = flattenedNodes.findIndex(n => !isWIPItem(n) && (n as TreeNode).id === current.parent_id);
				if (parentIdx !== -1) {
					selectedIndex = parentIdx;
					scrollToSelected();
				}
			}
			break;

		case 'Escape':
			event.preventDefault();
			if (expandedDetails.has(itemId)) {
				// Close L1 details
				expandedDetails.delete(itemId);
				expandedDetails = expandedDetails; // Trigger reactivity
			} else if (!isWIP && current.parent_id) {
				// Jump to parent
				const parentIdx = flattenedNodes.findIndex(n => !isWIPItem(n) && (n as TreeNode).id === current.parent_id);
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
		// Manually rebuild flattened nodes (include WIP items)
		const treeNodes = flattenTree(tree);
		flattenedNodes = [...wipItems, ...treeNodes];
		// Clamp selected index to valid range
		if (selectedIndex >= flattenedNodes.length) {
			selectedIndex = Math.max(0, flattenedNodes.length - 1);
		}
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
	{#each flattenedNodes as item, index (getItemId(item))}
		{@const itemId = getItemId(item)}
		{@const isWIP = isWIPItem(item)}
		<div
			data-testid="issue-row-{itemId}"
			data-node-index={index}
			class="node-row cursor-pointer select-none focus:outline-none"
			class:new-issue-highlight={!isWIP && newIssueIds.has(itemId)}
			role="treeitem"
			aria-selected={index === selectedIndex}
			tabindex="-1"
			on:click={() => selectNode(index)}
		>
		{#if isWIP}
			{#if item.type === 'running'}
				{@const agent = item.agent}
				{@const statusIcon = getAgentStatusIcon(agent)}
				{@const health = computeAgentHealth(agent)}
				{@const contextPct = getContextPercent(agent)}
				<!-- Running Agent - WIP Item -->
				<div class="flex items-center gap-3 py-2 px-3 rounded transition-colors {index === selectedIndex ? 'bg-accent' : ''}" style="padding-left: 12px">
					<!-- Expansion indicator placeholder (matches tree nodes) -->
					<span class="w-4"></span>
					
					<!-- Status icon with health indication -->
					<span class="{statusIcon.color} w-5 text-center">{statusIcon.icon}</span>
					
					<!-- Priority placeholder (w-8 matches tree badge width) -->
					<span class="w-8"></span>
					
					<!-- ID (min-w-[120px] matches tree) -->
					<span class="text-xs font-mono text-muted-foreground min-w-[120px]">
						{agent.beads_id || agent.id.slice(0, 15)}
					</span>
					
					<!-- Title (text-sm font-medium matches tree) -->
					<span class="flex-1 text-sm font-medium text-foreground truncate">
						{agent.task || agent.skill || 'Unknown task'}
					</span>
					
					<!-- Expressive status (replaces phase badge) -->
					<span class="text-xs text-muted-foreground italic min-w-[120px]">
						{getExpressiveStatus(agent)}
					</span>
					
					<!-- Health warning tooltip -->
					{#if health.status !== 'healthy'}
						<span class="text-xs {health.status === 'critical' ? 'text-red-500' : 'text-yellow-500'}" title={health.reasons.join(', ')}>
							{health.status === 'critical' ? '!' : '?'}
						</span>
					{/if}
					
					<!-- Runtime -->
					{#if agent.runtime}
						<span class="text-xs text-muted-foreground min-w-[40px] text-right">{agent.runtime}</span>
					{/if}
				</div>
				
				<!-- L1: Expanded details for running agents -->
				{#if expandedDetails.has(itemId)}
					<div class="expanded-details ml-14 pb-2 px-3 flex items-center gap-4 text-xs text-muted-foreground bg-muted/30 rounded mt-1 mb-2 p-3">
						<!-- Phase -->
						{#if agent.phase}
							<span class="flex items-center gap-1">
								<span class="text-foreground/60">Phase:</span>
								<span class="text-foreground">{agent.phase}</span>
							</span>
						{/if}
						
						<!-- Context % -->
						{#if contextPct !== null}
							<span class="flex items-center gap-1">
								<span class="text-foreground/60">Context:</span>
								<span class="{getContextColor(contextPct)}">{contextPct}%</span>
							</span>
						{/if}
						
						<!-- Skill -->
						{#if agent.skill}
							<span class="flex items-center gap-1">
								<span class="text-foreground/60">Skill:</span>
								<span>{agent.skill}</span>
							</span>
						{/if}
						
						<!-- Model (short form) -->
						{#if agent.model}
							<span class="flex items-center gap-1">
								<span class="text-foreground/60">Model:</span>
								<span>{agent.model.split('/').pop()?.split('-').slice(0, 2).join('-') || agent.model}</span>
							</span>
						{/if}
					</div>
				{/if}
			{:else}
				{@const issue = item.issue}
				<!-- Queued Issue - WIP Item (NO opacity-60) -->
				<div class="flex items-center gap-3 py-2 px-3 rounded transition-colors {index === selectedIndex ? 'bg-accent' : ''}" style="padding-left: 12px">
					<!-- Expansion indicator placeholder (matches tree nodes) -->
					<span class="w-4"></span>
					
					<!-- Status icon -->
					<span class="text-muted-foreground w-5">○</span>
					
					<!-- Priority badge -->
					<Badge variant={getPriorityVariant(issue.priority)} class="w-8 justify-center text-xs">
						P{issue.priority}
					</Badge>
					
					<!-- ID -->
					<span class="text-xs font-mono text-muted-foreground min-w-[120px]">
						{issue.id}
					</span>
					
					<!-- Title -->
					<span class="flex-1 text-sm font-medium text-foreground truncate">
						{issue.title}
					</span>
					
					<!-- Queue reason -->
					<span class="text-xs text-muted-foreground italic min-w-[80px]">
						queued
					</span>
					
					<!-- Type badge -->
					<Badge variant="outline" class="{getTypeBadge(issue.issue_type)} text-xs">
						{issue.issue_type}
					</Badge>
				</div>
				
				<!-- L1: Expanded details for queued issues -->
				{#if expandedDetails.has(itemId)}
					<div class="expanded-details ml-14 mt-1 mb-2 p-3 bg-muted/30 rounded text-sm">
						<div class="text-xs text-muted-foreground">
							Queued for daemon processing
						</div>
					</div>
				{/if}
			{/if}
		{:else}
			{@const node = item as TreeNode}
			<!-- Tree Node - L0: Row -->
			<div
				class="flex items-center gap-3 py-2 px-3 rounded transition-colors {index === selectedIndex ? 'bg-accent' : ''}"
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
				{#if expandedDetails.has(node.id)}
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
		{/if}
		</div>
	{/each}
</div>

<style>
	.work-graph-tree {
		/* Ensure keyboard focus works */
		min-height: 100%;
	}
	
	.new-issue-highlight {
		animation: highlight-fade 3s ease-out;
	}
	
	@keyframes highlight-fade {
		0% {
			background-color: rgba(59, 130, 246, 0.3); /* blue-500 with opacity */
			box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.5);
		}
		100% {
			background-color: transparent;
			box-shadow: 0 0 0 0 transparent;
		}
	}
</style>
