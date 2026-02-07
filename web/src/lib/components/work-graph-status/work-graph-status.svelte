<script lang="ts">
	import { onMount } from 'svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { cn } from '$lib/utils';
	import type { StatusGroup, TreeNode, AttentionBadgeType } from '$lib/stores/work-graph';
	import { ATTENTION_BADGE_CONFIG } from '$lib/stores/attention';
	import { IssueSidePanel } from '$lib/components/issue-side-panel';

	export let groups: StatusGroup[] = [];
	export let newIssueIds: Set<string> = new Set();

	// Track expansion state for groups
	let groupExpansion = new Map<string, boolean>();
	
	// Track selected item for navigation
	let selectedGroupIndex = 0;
	let selectedNodeIndex = 0;
	
	// Track expanded details
	let expandedDetails = new Set<string>();
	
	// Track selected issue for side panel
	let selectedIssueForPanel: TreeNode | null = null;

	// Initialize group expansion
	$: {
		for (const group of groups) {
			if (!groupExpansion.has(group.status)) {
				groupExpansion.set(group.status, true);
			}
		}
	}

	// Get attention badge config
	function getAttentionBadge(badge: AttentionBadgeType | undefined) {
		if (!badge) return null;
		return ATTENTION_BADGE_CONFIG[badge] || null;
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

	// Toggle group expansion
	function toggleGroup(status: string) {
		const current = groupExpansion.get(status) ?? true;
		groupExpansion.set(status, !current);
		groupExpansion = groupExpansion; // Trigger reactivity
	}

	// Keyboard navigation
	function handleKeyDown(event: KeyboardEvent) {
		const currentGroup = groups[selectedGroupIndex];
		if (!currentGroup) return;
		const isGroupExpanded = groupExpansion.get(currentGroup.status) ?? true;

		switch (event.key) {
			case 'j':
			case 'ArrowDown':
				event.preventDefault();
				if (isGroupExpanded && selectedNodeIndex < currentGroup.nodes.length - 1) {
					selectedNodeIndex++;
				} else if (selectedGroupIndex < groups.length - 1) {
					selectedGroupIndex++;
					selectedNodeIndex = 0;
				}
				scrollToSelected();
				break;

			case 'k':
			case 'ArrowUp':
				event.preventDefault();
				if (selectedNodeIndex > 0) {
					selectedNodeIndex--;
				} else if (selectedGroupIndex > 0) {
					selectedGroupIndex--;
					const prevGroup = groups[selectedGroupIndex];
					const isPrevExpanded = groupExpansion.get(prevGroup.status) ?? true;
					selectedNodeIndex = isPrevExpanded ? prevGroup.nodes.length - 1 : 0;
				}
				scrollToSelected();
				break;

			case 'l':
			case 'ArrowRight':
				event.preventDefault();
				// Expand group if collapsed
				if (!isGroupExpanded) {
					toggleGroup(currentGroup.status);
				}
				break;

			case 'h':
			case 'ArrowLeft':
				event.preventDefault();
				// Collapse group if expanded
				if (isGroupExpanded) {
					toggleGroup(currentGroup.status);
				}
				break;

			case 'Enter':
				event.preventDefault();
				// Toggle details expansion
				const node = currentGroup.nodes[selectedNodeIndex];
				if (node) {
					if (expandedDetails.has(node.id)) {
						expandedDetails.delete(node.id);
					} else {
						expandedDetails.add(node.id);
					}
					expandedDetails = expandedDetails;
				}
				break;

			case 'i':
			case 'o':
				event.preventDefault();
				// Open side panel
				const selectedNode = currentGroup.nodes[selectedNodeIndex];
				if (selectedNode) {
					selectedIssueForPanel = selectedNode;
				}
				break;

			case 'Escape':
				event.preventDefault();
				if (selectedIssueForPanel) {
					selectedIssueForPanel = null;
				}
				break;

			case 'g':
				event.preventDefault();
				selectedGroupIndex = 0;
				selectedNodeIndex = 0;
				scrollToSelected();
				break;

			case 'G':
				event.preventDefault();
				selectedGroupIndex = groups.length - 1;
				const lastGroup = groups[selectedGroupIndex];
				const isLastExpanded = groupExpansion.get(lastGroup.status) ?? true;
				selectedNodeIndex = isLastExpanded ? lastGroup.nodes.length - 1 : 0;
				scrollToSelected();
				break;
		}
	}

	// Scroll selected item into view
	function scrollToSelected() {
		const element = document.querySelector(`[data-status-selected="true"]`);
		if (element) {
			element.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
		}
	}

	// Focus management
	let containerElement: HTMLDivElement;

	onMount(() => {
		setTimeout(() => {
			containerElement?.focus();
		}, 100);
	});

	// Select node on click
	function selectNode(groupIdx: number, nodeIdx: number) {
		selectedGroupIndex = groupIdx;
		selectedNodeIndex = nodeIdx;
	}

	// Close side panel
	function closeSidePanel() {
		selectedIssueForPanel = null;
	}
</script>

<div
	bind:this={containerElement}
	class="work-graph-status h-full overflow-y-auto px-6 py-4 focus:outline-none"
	role="tree"
	tabindex="0"
	onkeydown={handleKeyDown}
>
	{#each groups as group, groupIdx}
		{@const isExpanded = groupExpansion.get(group.status) ?? true}
		{@const isGroupSelected = groupIdx === selectedGroupIndex}
		
		<!-- Group Header -->
		<div
			class="group-header flex items-center gap-3 py-3 px-3 mb-2 rounded-lg border border-border cursor-pointer hover:bg-accent/50 transition-colors {isGroupSelected && selectedNodeIndex === -1 ? 'bg-accent' : ''}"
			role="treeitem"
			aria-expanded={isExpanded}
			onclick={() => toggleGroup(group.status)}
		>
			<!-- Expansion indicator -->
			<span class="w-4 text-muted-foreground text-sm">
				{isExpanded ? '▼' : '▶'}
			</span>
			
			<!-- Status icon -->
			<span class="{group.color}">
				{group.icon}
			</span>
			
			<!-- Group label -->
			<span class="font-semibold text-foreground">
				{group.label}
			</span>
			
			<!-- Count badge -->
			<Badge variant="outline" class="ml-auto text-xs">
				{group.count}
			</Badge>
		</div>
		
		<!-- Group Nodes -->
		{#if isExpanded && group.nodes.length > 0}
			<div class="group-nodes ml-6 mb-4">
				{#each group.nodes as node, nodeIdx}
					{@const isNodeSelected = isGroupSelected && nodeIdx === selectedNodeIndex}
					<div
						data-status-selected={isNodeSelected}
						class="node-row cursor-pointer select-none"
						class:new-issue-highlight={newIssueIds.has(node.id)}
						role="treeitem"
						aria-selected={isNodeSelected}
						tabindex="-1"
						onclick={() => selectNode(groupIdx, nodeIdx)}
					>
						<!-- Node Row -->
						<div
							class="flex items-center gap-3 py-2 px-3 rounded transition-colors {isNodeSelected ? 'bg-zinc-800' : ''}"
						>
							<!-- Tree indent placeholder -->
							<span class="w-4 text-muted-foreground text-xs">
								├─
							</span>
							
							<!-- Status icon -->
							<span class="w-5 {getStatusColor(node.status)}">
								{getStatusIcon(node.status)}
							</span>
							
							<!-- Priority badge -->
							<Badge variant={getPriorityVariant(node.priority)} class="w-8 justify-center text-xs">
								P{node.priority}
							</Badge>
							
							<!-- ID -->
							<span class="text-xs font-mono text-muted-foreground min-w-[120px]">
								{node.id}
							</span>
							
							<!-- Title -->
							<span class="flex-1 text-sm font-medium text-foreground truncate">
								{node.title}
							</span>
							
							<!-- Attention badge -->
							{#if node.attentionBadge}
								{@const badgeConfig = getAttentionBadge(node.attentionBadge)}
								{#if badgeConfig}
									<Badge variant={badgeConfig.variant} class="shrink-0">
										{badgeConfig.label}
									</Badge>
								{/if}
							{/if}
							
							<!-- Type badge -->
							<Badge variant="outline" class="{getTypeBadge(node.type)} text-xs shrink-0">
								{node.type}
							</Badge>
						</div>
						
						<!-- Expanded details -->
						{#if expandedDetails.has(node.id)}
							<div class="expanded-details ml-12 mt-1 mb-2 p-3 bg-muted/30 rounded text-sm">
								<!-- Description -->
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
		{:else if isExpanded && group.nodes.length === 0}
			<div class="ml-6 mb-4 py-2 px-3 text-sm text-muted-foreground italic">
				No issues
			</div>
		{/if}
	{/each}
	
	{#if groups.length === 0}
		<div class="flex items-center justify-center h-full">
			<div class="text-muted-foreground">No issues to display</div>
		</div>
	{/if}
</div>

<!-- Issue Side Panel -->
{#if selectedIssueForPanel}
	<IssueSidePanel issue={selectedIssueForPanel} on:close={closeSidePanel} />
{/if}

<style>
	.work-graph-status {
		min-height: 100%;
	}
	
	.new-issue-highlight {
		animation: highlight-fade 30s ease-out;
	}
	
	@keyframes highlight-fade {
		0% {
			background-color: rgba(59, 130, 246, 0.3);
			box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.5);
		}
		100% {
			background-color: transparent;
			box-shadow: 0 0 0 0 transparent;
		}
	}
</style>
