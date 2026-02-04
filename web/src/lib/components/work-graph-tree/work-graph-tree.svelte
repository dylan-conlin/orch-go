<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { cn } from '$lib/utils';
	import type { TreeNode, AttentionBadgeType } from '$lib/stores/work-graph';
	import { closeIssue } from '$lib/stores/work-graph';
	import type { WIPItem } from '$lib/stores/wip';
	import { getExpressiveStatus, computeAgentHealth, getContextPercent, getContextColor } from '$lib/stores/wip';
	import { attention, ATTENTION_BADGE_CONFIG, type CompletedIssue } from '$lib/stores/attention';
	import { DeliverableChecklist } from '$lib/components/deliverable-checklist';
	import { getExpectedDeliverables } from '$lib/stores/deliverables';
	import { IssueSidePanel } from '$lib/components/issue-side-panel';
	import { CloseIssueModal } from '$lib/components/close-issue-modal';
	import { orchestratorContext } from '$lib/stores/context';

	export let tree: TreeNode[] = [];
	export let newIssueIds: Set<string> = new Set();
	export let wipItems: WIPItem[] = [];
	export let completedIssues: CompletedIssue[] = [];
	export let onToggleExpansion: (nodeId: string, expanded: boolean) => void = () => {};
	export let onSetFocus: (beadsId: string, title: string) => void = () => {};

	// Get attention badge config for a badge type
	function getAttentionBadge(badge: AttentionBadgeType | 'unverified' | 'needs_fix' | undefined) {
		if (!badge) return null;
		return ATTENTION_BADGE_CONFIG[badge] || null;
	}

	// Flatten tree for keyboard navigation
	// Now includes completed-but-unverified issues as TreeNode-like objects
	let flattenedNodes: (TreeNode | WIPItem | CompletedIssue)[] = [];
	let selectedIndex = 0;
	let pendingVerification: CompletedIssue[] = [];
	let pinnedTreeIds = new Set<string>();

	// Track expanded details separately (fixes reactivity issues)
	let expandedDetails = new Set<string>();
	
	// Track selected issue for side panel
	let selectedIssueForPanel: TreeNode | null = null;
	// Track issue for close modal
	let issueToClose: TreeNode | null = null;
	let isClosing = false;

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

	// Type guard to check if item is a CompletedIssue
	function isCompletedIssue(item: TreeNode | WIPItem | CompletedIssue): item is CompletedIssue {
		return 'verificationStatus' in item;
	}

	// Rebuild flattened list when tree, wipItems, or completedIssues change
	$: {
		const treeNodes = flattenTree(tree);
		// Filter completed issues: only show unverified or needs_fix (verified = truly done)
		// Sort by urgency: needs_fix first (broken), then unverified (just needs review)
		pendingVerification = completedIssues
			.filter(issue => issue.verificationStatus !== 'verified')
			.sort((a, b) => {
				// needs_fix before unverified
				if (a.verificationStatus === 'needs_fix' && b.verificationStatus !== 'needs_fix') return -1;
				if (b.verificationStatus === 'needs_fix' && a.verificationStatus !== 'needs_fix') return 1;
				// then by priority
				return a.priority - b.priority;
			});

		// Track which tree nodes are also surfaced in WIP (for visual differentiation in the tree)
		const pinnedIds = new Set<string>();
		for (const item of wipItems) {
			if (item.type === 'running') {
				if (item.agent.beads_id) {
					pinnedIds.add(item.agent.beads_id);
				}
			} else {
				pinnedIds.add(item.issue.id);
			}
		}
		pinnedTreeIds = pinnedIds;

		// Order: WIP items first, then pending verification, then main tree
		flattenedNodes = [...wipItems, ...pendingVerification, ...treeNodes];
		// Clamp selected index to valid range
		if (selectedIndex >= flattenedNodes.length) {
			selectedIndex = Math.max(0, flattenedNodes.length - 1);
		}
	}

	// Type guard to check if item is a WIPItem
	function isWIPItem(item: TreeNode | WIPItem | CompletedIssue): item is WIPItem {
		return 'type' in item && (item.type === 'running' || item.type === 'queued');
	}

	// Get ID from WIPItem, TreeNode, or CompletedIssue
	function getItemId(item: TreeNode | WIPItem | CompletedIssue): string {
		if (isWIPItem(item)) {
			return item.type === 'running' ? item.agent.id : item.issue.id;
		}
		return item.id;
	}

	// Get stable key for Svelte each blocks (avoids collisions when same issue appears in multiple views)
	function getItemKey(item: TreeNode | WIPItem | CompletedIssue): string {
		if (isWIPItem(item)) {
			return item.type === 'running' ? `wip-running-${item.agent.id}` : `wip-queued-${item.issue.id}`;
		}
		if (isCompletedIssue(item)) return `completed-${item.id}`;
		return `tree-${item.id}`;
	}

	// Get stable test ID per row type (avoids collisions when issue appears in both WIP + tree)
	function getRowTestId(item: TreeNode | WIPItem | CompletedIssue): string {
		if (isWIPItem(item)) {
			return item.type === 'running'
				? `wip-row-${item.agent.beads_id || item.agent.id}`
				: `wip-row-${item.issue.id}`;
		}
		if (isCompletedIssue(item)) return `completed-row-${item.id}`;
		return `issue-row-${item.id}`;
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
			// Expand tree node if it has children (WIP items and completed issues don't have tree expansion)
			if (!isWIP && !isCompletedIssue(current) && (current as TreeNode).children.length > 0) {
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
			// Collapse tree node if it has children and is expanded (WIP items and completed issues don't have tree collapse)
			if (!isWIP && !isCompletedIssue(current) && (current as TreeNode).children.length > 0 && (current as TreeNode).expanded) {
				toggleExpansion(current as TreeNode);
			} else if (!isWIP && !isCompletedIssue(current) && (current as TreeNode).parent_id) {
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
			// Close side panel first if it's open
			if (selectedIssueForPanel) {
				selectedIssueForPanel = null;
			} else if (expandedDetails.has(itemId)) {
				// Close L1 details
				expandedDetails.delete(itemId);
				expandedDetails = expandedDetails; // Trigger reactivity
			} else if (!isWIP && !isCompletedIssue(current) && (current as TreeNode).parent_id) {
				// Jump to parent
				const parentIdx = flattenedNodes.findIndex(n => !isWIPItem(n) && (n as TreeNode).id === current.parent_id);
				if (parentIdx !== -1) {
					selectedIndex = parentIdx;
					scrollToSelected();
				}
			}
			break;

			case 'i':
			case 'o':
				event.preventDefault();
				// Open side panel for TreeNode (not for WIP items or completed issues)
				if (!isWIP && !isCompletedIssue(current)) {
					selectedIssueForPanel = current as TreeNode;
				}
				break;

			case 'v':
				event.preventDefault();
				// Mark completed issue as verified (only for UNVERIFIED issues)
				if (isCompletedIssue(current) && current.verificationStatus === 'unverified') {
					attention.markVerified(current.id);
				}
				break;

			case 'x':
				event.preventDefault();
				// For completed issues: mark as needs_fix
				// For regular tree nodes: open close modal
				if (isCompletedIssue(current) && current.verificationStatus === 'unverified') {
					attention.markNeedsFix(current.id);
				} else if (!isWIP && !isCompletedIssue(current)) {
					// Open close modal for regular tree nodes
					issueToClose = current as TreeNode;
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

			case 't':
				event.preventDefault();
				// Jump from WIP section item to its tree position
				if (isWIP) {
					const wipItem = current as WIPItem;
					const beadsId = wipItem.type === 'running' 
						? (wipItem.agent.beads_id || wipItem.agent.id) 
						: wipItem.issue.id;
					// Find matching tree node in flattenedNodes (tree nodes come after WIP and completed)
					const treeIdx = flattenedNodes.findIndex((n) => 
						!isWIPItem(n) && !isCompletedIssue(n) && (n as TreeNode).id === beadsId
					);
					if (treeIdx !== -1) {
						selectedIndex = treeIdx;
						scrollToSelected();
					}
				}
				break;

			case 'w':
				event.preventDefault();
				// Jump from tree item (if in WIP) to its WIP section position
				if (!isWIP && !isCompletedIssue(current)) {
					const nodeId = (current as TreeNode).id;
					if (pinnedTreeIds.has(nodeId)) {
						// Find matching WIP item in flattenedNodes
						const wipIdx = flattenedNodes.findIndex(n => {
							if (!isWIPItem(n)) return false;
							const wipItem = n as WIPItem;
							if (wipItem.type === 'running') {
								return (wipItem.agent.beads_id || wipItem.agent.id) === nodeId;
							} else {
								return wipItem.issue.id === nodeId;
							}
						});
						if (wipIdx !== -1) {
							selectedIndex = wipIdx;
							scrollToSelected();
						}
					}
				}
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
		// Notify parent component to update expansion state
		onToggleExpansion(node.id, node.expanded);
		// Manually rebuild flattened nodes (include WIP items)
		const treeNodes = flattenTree(tree);
		flattenedNodes = [...wipItems, ...pendingVerification, ...treeNodes];
		// Clamp selected index to valid range
		if (selectedIndex >= flattenedNodes.length) {
			selectedIndex = Math.max(0, flattenedNodes.length - 1);
		}
	}

	// Select node on click
	function selectNode(index: number) {
		selectedIndex = index;
	}
	
	// Close side panel
	function closeSidePanel() {
		selectedIssueForPanel = null;
	}
	
	// Open side panel for a node
	function openSidePanel(node: TreeNode) {
		selectedIssueForPanel = node;
	}

	// Handle close modal cancel
	function handleCloseModalCancel() {
		issueToClose = null;
	}

	// Handle close modal confirm
	async function handleCloseModalConfirm(event: CustomEvent<{ reason: string }>) {
		if (!issueToClose || isClosing) return;

		isClosing = true;
		const projectDir = $orchestratorContext?.project_dir;
		const result = await closeIssue(issueToClose.id, event.detail.reason, projectDir);

		if (!result.success) {
			console.error('Failed to close issue:', result.error);
			// TODO: Show error toast
		}

		issueToClose = null;
		isClosing = false;
	}
</script>

<div
	bind:this={containerElement}
	class="work-graph-tree h-full overflow-y-auto px-6 py-4 focus:outline-none"
	role="tree"
	tabindex="0"
	on:keydown={handleKeyDown}
>
	{#each flattenedNodes as item, index (getItemKey(item))}
		{@const itemId = getItemId(item)}
		{@const isWIP = isWIPItem(item)}
		{@const isCompleted = isCompletedIssue(item)}
		{@const depth = (!isWIP && !isCompleted) ? (item as TreeNode).depth : undefined}
		<div
			data-testid={getRowTestId(item)}
			data-node-index={index}
			data-depth={depth !== undefined ? String(depth) : null}
			class="node-row cursor-pointer select-none focus:outline-none"
			class:selected={index === selectedIndex}
			class:focused={index === selectedIndex}
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
				<div class="flex items-center gap-3 py-2 px-3 rounded transition-colors {index === selectedIndex ? 'bg-zinc-800' : ''}" style="padding-left: 12px">
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
					<div class="expanded-details ml-14 pb-2 px-3 text-xs text-muted-foreground bg-muted/30 rounded mt-1 mb-2 p-3 space-y-2">
						<!-- Agent metadata row -->
						<div class="flex items-center gap-4">
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
						
						<!-- Deliverables checklist (compact L1 view) -->
						<DeliverableChecklist 
							deliverables={getExpectedDeliverables('bug', agent.skill || 'feature-impl')} 
							mode="compact" 
						/>
					</div>
				{/if}
			{:else}
				{@const issue = item.issue}
				<!-- Queued Issue - WIP Item (NO opacity-60) -->
				<div class="flex items-center gap-3 py-2 px-3 rounded transition-colors {index === selectedIndex ? 'bg-zinc-800' : ''}" style="padding-left: 12px">
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
		{:else if isCompletedIssue(item)}
			{@const issue = item}
			{@const badgeConfig = getAttentionBadge(issue.attentionBadge)}
			<!-- Completed Issue (pending verification) - inline in main list -->
			<div
				class={cn(
					"flex items-center gap-3 py-2 px-3 rounded transition-colors",
					index === selectedIndex ? 'bg-zinc-800' : '',
					issue.verificationStatus === 'needs_fix' && "bg-red-950/20"
				)}
				style="padding-left: 12px"
			>
				<!-- Expansion indicator placeholder -->
				<span class="w-4"></span>

				<!-- Verification status icon -->
				<span class="w-5 text-center">
					{#if issue.verificationStatus === 'needs_fix'}
						<span class="text-red-500">✗</span>
					{:else}
						<span class="text-yellow-500">○</span>
					{/if}
				</span>

				<!-- Priority badge -->
				<Badge variant={getPriorityVariant(issue.priority)} class="w-8 justify-center text-xs">
					P{issue.priority}
				</Badge>

				<!-- ID -->
				<span class="text-xs font-mono text-muted-foreground min-w-[120px]">
					{issue.id}
				</span>

				<!-- Title -->
				<span
					class="flex-1 text-sm font-medium truncate"
					class:line-through={issue.verificationStatus === 'needs_fix'}
					class:text-muted-foreground={issue.verificationStatus === 'needs_fix'}
					class:text-foreground={issue.verificationStatus !== 'needs_fix'}
				>
					{issue.title}
				</span>

				<!-- Attention badge (UNVERIFIED or NEEDS FIX) -->
				{#if badgeConfig}
					<Badge variant={badgeConfig.variant} class="shrink-0">
						{badgeConfig.label}
					</Badge>
				{/if}

				<!-- Type badge -->
				<Badge variant="outline" class="{getTypeBadge(issue.type)} text-xs shrink-0">
					{issue.type}
				</Badge>
			</div>

			<!-- L1: Expanded details for completed issues -->
			{#if expandedDetails.has(itemId)}
				<div class="expanded-details ml-14 mt-1 mb-2 p-3 bg-muted/30 rounded text-sm space-y-2">
					<!-- Description -->
					{#if issue.description}
						<div>
							<span class="text-xs font-semibold uppercase text-foreground">Description:</span>
							<p class="mt-1 text-xs text-muted-foreground">{issue.description}</p>
						</div>
					{/if}

					<!-- Completion info -->
					<div class="flex items-center gap-4 text-xs">
						{#if issue.completedAt}
							<span class="flex items-center gap-1">
								<span class="text-foreground/60">Completed:</span>
								<span class="text-muted-foreground">{new Date(issue.completedAt).toLocaleString()}</span>
							</span>
						{/if}
						<span class="flex items-center gap-1">
							<span class="text-foreground/60">Status:</span>
							<span class={issue.verificationStatus === 'needs_fix' ? 'text-red-500' : 'text-yellow-500'}>
								{issue.verificationStatus}
							</span>
						</span>
					</div>

					<!-- Action hints -->
					<div class="text-xs text-muted-foreground border-t border-border pt-2 mt-2">
						{#if issue.verificationStatus === 'unverified'}
							Press <kbd class="px-1 py-0.5 bg-muted rounded text-foreground">v</kbd> to verify or <kbd class="px-1 py-0.5 bg-muted rounded text-foreground">x</kbd> to mark needs fix
						{:else if issue.verificationStatus === 'needs_fix'}
							Marked as needing fix — reopen or reassign this issue
						{/if}
					</div>
				</div>
			{/if}
		{:else}
			{@const node = item as TreeNode}
			{@const dimPinned = pinnedTreeIds.has(node.id) && index !== selectedIndex}
			<!-- Tree Node - L0: Row -->
			<div
				class="flex items-center gap-3 py-2 px-3 rounded transition-colors {index === selectedIndex ? 'bg-zinc-800' : ''} {node.absorbed_by ? 'opacity-50' : ''} {dimPinned ? 'opacity-60' : ''}"
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
					<span class="flex-1 text-sm font-medium text-foreground truncate">
						{node.title}
					</span>

					<!-- Attention badge (if any) -->
					{#if node.attentionBadge}
						{@const badgeConfig = getAttentionBadge(node.attentionBadge)}
						{#if badgeConfig}
							<Badge variant={badgeConfig.variant} class="shrink-0">
								{badgeConfig.label}
							</Badge>
						{/if}
					{/if}

					<!-- Type badge -->
					<Badge data-testid="type-badge" variant="outline" class="{getTypeBadge(node.type)} text-xs shrink-0">
						{node.type}
					</Badge>

						<!-- Set as Focus button for epics -->
						{#if node.type === 'epic'}
							<button
								type="button"
								class="text-xs text-blue-500 hover:text-blue-600 hover:underline px-1"
								onclick={() => onSetFocus(node.id, node.title)}
								title="Set this epic as your current focus"
							>
								Set Focus
							</button>
						{/if}

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

						{#if node.absorbed_by}
							<div class="mb-2">
								<span class="text-xs font-semibold uppercase text-purple-500">Absorbed by:</span>
								<span class="text-xs text-muted-foreground ml-1">⊃ {node.absorbed_by}</span>
							</div>
						{/if}

						{#if node.blocked_by.length === 0 && node.blocks.length === 0 && !node.absorbed_by}
							<div class="text-xs text-muted-foreground">No blocking relationships</div>
						{/if}
					</div>
				{/if}
		{/if}
		</div>
	{/each}
</div>

<!-- Issue Side Panel -->
{#if selectedIssueForPanel}
	<IssueSidePanel issue={selectedIssueForPanel} on:close={closeSidePanel} />
{/if}

<!-- Close Issue Modal -->
{#if issueToClose}
	<CloseIssueModal
		issueId={issueToClose.id}
		issueTitle={issueToClose.title}
		on:close={handleCloseModalCancel}
		on:confirm={handleCloseModalConfirm}
	/>
{/if}

<style>
	.work-graph-tree {
		/* Ensure keyboard focus works */
		min-height: 100%;
	}
	
	.new-issue-highlight {
		animation: highlight-fade 30s ease-out;
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
