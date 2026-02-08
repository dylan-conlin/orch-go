<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { cn } from '$lib/utils';
	import type { TreeNode, AttentionBadgeType, GroupSection, GroupByMode } from '$lib/stores/work-graph';
	import { closeIssue, updateIssue } from '$lib/stores/work-graph';
	import type { WIPItem } from '$lib/stores/wip';
	import { getExpressiveStatus, computeAgentHealth, getContextPercent, getContextColor } from '$lib/stores/wip';
	import { ATTENTION_BADGE_CONFIG } from '$lib/stores/attention';
	import { DeliverableChecklist } from '$lib/components/deliverable-checklist';
	import { getExpectedDeliverables } from '$lib/stores/deliverables';
	import { IssueSidePanel } from '$lib/components/issue-side-panel';
	import { CloseIssueModal } from '$lib/components/close-issue-modal';
	import { orchestratorContext } from '$lib/stores/context';

	export let tree: TreeNode[] = [];
	export let groups: GroupSection[] = [];
	export let groupMode: GroupByMode = 'priority';
	export let newIssueIds: Set<string> = new Set();
	export let wipItems: WIPItem[] = [];
	export let onToggleExpansion: (nodeId: string, expanded: boolean) => void = () => {};
	export let onSetFocus: (beadsId: string, title: string) => void = () => {};

	// Track which group sections are collapsed
	let collapsedGroups = new Set<string>();

	function toggleGroup(key: string) {
		if (collapsedGroups.has(key)) {
			collapsedGroups.delete(key);
		} else {
			collapsedGroups.add(key);
		}
		collapsedGroups = collapsedGroups; // trigger reactivity
	}

	// Whether we're in grouped mode (non-priority has label sections)
	$: isGrouped = groups.length > 0 && groupMode !== 'priority';

	// Get attention badge config for a badge type
	function getAttentionBadge(badge: AttentionBadgeType | 'unverified' | 'needs_fix' | undefined) {
		if (!badge) return null;
		return ATTENTION_BADGE_CONFIG[badge] || null;
	}

	// Group header sentinel for keyboard navigation
	interface GroupHeader {
		_groupHeader: true;
		key: string;
		label: string;
		count: number;
		unlabeled: boolean;
	}

	function isGroupHeader(item: any): item is GroupHeader {
		return item && '_groupHeader' in item;
	}

	// Flatten tree for keyboard navigation
		let flattenedNodes: (TreeNode | WIPItem | GroupHeader)[] = [];
		let selectedIndex = 0;
		let pinnedTreeIds = new Set<string>();

	// Track expanded details separately (fixes reactivity issues)
	let expandedDetails = new Set<string>();
	
	// Track copied ID for visual feedback
	let copiedId: string | null = null;
	let copiedTimeout: ReturnType<typeof setTimeout> | null = null;
	
	// Track priority mode (p key pressed, waiting for 0-4)
	let priorityMode = false;
	let priorityModeTimeout: ReturnType<typeof setTimeout> | null = null;
	
	// Track action feedback (flash on successful update)
	let actionFeedback = new Map<string, 'priority' | 'queue'>(); // itemId -> action type
	let actionFeedbackTimeout: ReturnType<typeof setTimeout> | null = null;
	
	// Copy ID to clipboard with visual feedback
	async function copyToClipboard(id: string) {
		try {
			await navigator.clipboard.writeText(id);
			// Clear any existing timeout
			if (copiedTimeout) {
				clearTimeout(copiedTimeout);
			}
			// Show "Copied!" feedback
			copiedId = id;
			// Clear after 1.5 seconds
			copiedTimeout = setTimeout(() => {
				copiedId = null;
				copiedTimeout = null;
			}, 1500);
		} catch (err) {
			console.error('Failed to copy to clipboard:', err);
		}
	}
	
	// Enter priority mode (waiting for 0-4 key)
	function enterPriorityMode() {
		priorityMode = true;
		// Auto-exit priority mode after 3 seconds
		if (priorityModeTimeout) {
			clearTimeout(priorityModeTimeout);
		}
		priorityModeTimeout = setTimeout(() => {
			priorityMode = false;
			priorityModeTimeout = null;
		}, 3000);
	}
	
	// Exit priority mode
	function exitPriorityMode() {
		priorityMode = false;
		if (priorityModeTimeout) {
			clearTimeout(priorityModeTimeout);
			priorityModeTimeout = null;
		}
	}
	
	// Set priority for a tree node
	async function setPriority(node: TreeNode, priority: number) {
		const projectDir = $orchestratorContext?.project_dir;
		const result = await updateIssue(node.id, { priority, project_dir: projectDir });
		
		if (result.success) {
			// Show success feedback
			showActionFeedback(node.id, 'priority');
		} else {
			console.error('Failed to update priority:', result.error);
			// TODO: Show error toast
		}
	}
	
	// Toggle triage:ready label for a tree node
	async function toggleTriageReady(node: TreeNode) {
		const projectDir = $orchestratorContext?.project_dir;
		const hasLabel = (node.labels || []).includes('triage:ready');
		
		const result = await updateIssue(node.id, {
			add_labels: hasLabel ? undefined : ['triage:ready'],
			remove_labels: hasLabel ? ['triage:ready'] : undefined,
			project_dir: projectDir,
		});
		
		if (result.success) {
			// Show success feedback
			showActionFeedback(node.id, 'queue');
		} else {
			console.error('Failed to toggle triage:ready:', result.error);
			// TODO: Show error toast
		}
	}
	
	// Show action feedback (brief flash)
	function showActionFeedback(itemId: string, action: 'priority' | 'queue') {
		// Clear any existing timeout
		if (actionFeedbackTimeout) {
			clearTimeout(actionFeedbackTimeout);
		}
		
		// Set feedback
		actionFeedback.set(itemId, action);
		actionFeedback = actionFeedback; // Trigger reactivity
		
		// Clear after 1 second
		actionFeedbackTimeout = setTimeout(() => {
			actionFeedback.delete(itemId);
			actionFeedback = actionFeedback; // Trigger reactivity
			actionFeedbackTimeout = null;
		}, 1000);
	}
	
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

	// Rebuild flattened list when tree, groups, or wipItems change
		$: {
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

		// Build flat list based on whether we're in grouped mode
		const items: (TreeNode | WIPItem | GroupHeader)[] = [...wipItems];

		if (isGrouped && groups.length > 0) {
			for (const group of groups) {
				items.push({
					_groupHeader: true,
					key: group.key,
					label: group.label,
					count: group.nodes.length,
					unlabeled: group.unlabeled,
				});
				if (!collapsedGroups.has(group.key)) {
					items.push(...flattenTree(group.nodes));
				}
			}
		} else {
			items.push(...flattenTree(tree));
		}

		flattenedNodes = items;
		// Clamp selected index to valid range
		if (selectedIndex >= flattenedNodes.length) {
			selectedIndex = Math.max(0, flattenedNodes.length - 1);
		}
	}

	// Type guard to check if item is a WIPItem
	function isWIPItem(item: TreeNode | WIPItem | GroupHeader): item is WIPItem {
		return 'type' in item && (item.type === 'running' || item.type === 'queued');
	}

	// Get ID from WIPItem, TreeNode, or GroupHeader
	function getItemId(item: TreeNode | WIPItem | GroupHeader): string {
		if (isGroupHeader(item)) return item.key;
		if (isWIPItem(item)) {
			return item.type === 'running' ? item.agent.id : item.issue.id;
		}
		return item.id;
	}

	// Get stable key for Svelte each blocks (avoids collisions when same issue appears in multiple views)
	function getItemKey(item: TreeNode | WIPItem | GroupHeader): string {
		if (isGroupHeader(item)) return `group-${item.key}`;
		if (isWIPItem(item)) {
			return item.type === 'running' ? `wip-running-${item.agent.id}` : `wip-queued-${item.issue.id}`;
		}
		return `tree-${item.id}`;
	}

	// Get stable test ID per row type (avoids collisions when issue appears in both WIP + tree)
	function getRowTestId(item: TreeNode | WIPItem | GroupHeader): string {
		if (isGroupHeader(item)) return `group-header-${item.key}`;
		if (isWIPItem(item)) {
			return item.type === 'running'
				? `wip-row-${item.agent.beads_id || item.agent.id}`
				: `wip-row-${item.issue.id}`;
		}
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
	async function handleKeyDown(event: KeyboardEvent) {
		const current = flattenedNodes[selectedIndex];
		if (!current) return;

		// Handle group header interactions
		if (isGroupHeader(current)) {
			if (event.key === 'Enter' || event.key === 'l' || event.key === 'h' || event.key === 'ArrowRight' || event.key === 'ArrowLeft') {
				event.preventDefault();
				toggleGroup(current.key);
				return;
			}
			if (event.key === 'j' || event.key === 'ArrowDown') {
				event.preventDefault();
				selectedIndex = Math.min(selectedIndex + 1, flattenedNodes.length - 1);
				scrollToSelected();
				return;
			}
			if (event.key === 'k' || event.key === 'ArrowUp') {
				event.preventDefault();
				selectedIndex = Math.max(selectedIndex - 1, 0);
				scrollToSelected();
				return;
			}
			return;
		}

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
			if (!isWIP && (current as TreeNode).children.length > 0) {
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
			// Collapse tree node if it has children and is expanded
			if (!isWIP && (current as TreeNode).children.length > 0 && (current as TreeNode).expanded) {
				toggleExpansion(current as TreeNode);
			} else if (!isWIP && (current as TreeNode).parent_id) {
				// Jump to parent if no children to collapse
				const parentIdx = flattenedNodes.findIndex(n => !isWIPItem(n) && (n as TreeNode).id === (current as TreeNode).parent_id);
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
			} else if (!isWIP && (current as TreeNode).parent_id) {
				// Jump to parent
				const parentIdx = flattenedNodes.findIndex(n => !isWIPItem(n) && (n as TreeNode).id === (current as TreeNode).parent_id);
				if (parentIdx !== -1) {
					selectedIndex = parentIdx;
					scrollToSelected();
				}
			}
			break;

			case 'i':
			case 'o':
				event.preventDefault();
				// Open side panel for TreeNode (not for WIP items)
				if (!isWIP) {
					selectedIssueForPanel = current as TreeNode;
				}
				break;

			case 'x':
				event.preventDefault();
				// Open close modal for regular tree nodes
				if (!isWIP) {
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
					// Find matching tree node in flattenedNodes
					const treeIdx = flattenedNodes.findIndex((n) => 
						!isWIPItem(n) && (n as TreeNode).id === beadsId
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
				if (!isWIP) {
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
			
			case 'c':
				event.preventDefault();
				// Copy selected item's ID to clipboard
				const id = getItemId(current);
				copyToClipboard(id);
				break;
			
			case 'p':
				event.preventDefault();
				// Enter priority mode (only for TreeNode, not WIP)
				if (!isWIP) {
					enterPriorityMode();
				}
				break;
			
			case '0':
			case '1':
			case '2':
			case '3':
			case '4':
				event.preventDefault();
				// Set priority (only when in priority mode and on TreeNode)
				if (priorityMode && !isWIP) {
					const priority = parseInt(event.key, 10);
					await setPriority(current as TreeNode, priority);
					exitPriorityMode();
				}
				break;
			
			case 'q':
				event.preventDefault();
				// Toggle triage:ready label (only for TreeNode, not WIP)
				if (!isWIP) {
					await toggleTriageReady(current as TreeNode);
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
		// Manually rebuild flattened nodes (include WIP items + groups)
		const items: (TreeNode | WIPItem | GroupHeader)[] = [...wipItems];
		if (isGrouped && groups.length > 0) {
			for (const group of groups) {
				items.push({
					_groupHeader: true,
					key: group.key,
					label: group.label,
					count: group.nodes.length,
					unlabeled: group.unlabeled,
				});
				if (!collapsedGroups.has(group.key)) {
					items.push(...flattenTree(group.nodes));
				}
			}
		} else {
			items.push(...flattenTree(tree));
		}
		flattenedNodes = items;
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
		// Restore focus to tree container for keyboard navigation
		setTimeout(() => containerElement?.focus(), 0);
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
		// Restore focus to tree container for keyboard navigation
		setTimeout(() => containerElement?.focus(), 0);
	}
</script>

<div
	bind:this={containerElement}
	class="work-graph-tree h-full overflow-y-auto px-0 py-2 focus:outline-none relative"
	role="tree"
	tabindex="0"
	onkeydown={handleKeyDown}
>
	<!-- Priority Mode Indicator -->
	{#if priorityMode}
		<div class="absolute top-2 right-4 z-10 bg-purple-500/90 text-white text-xs font-semibold px-3 py-1 rounded shadow-lg">
			Priority Mode: Press 0-4
		</div>
	{/if}
	{#each flattenedNodes as item, index (getItemKey(item))}
		{@const itemId = getItemId(item)}
		{@const isWIP = !isGroupHeader(item) && isWIPItem(item)}
		{@const depth = !isGroupHeader(item) && !isWIP ? (item as TreeNode).depth : undefined}
		{#if isGroupHeader(item)}
			<!-- Group Section Header -->
			<div
				data-testid={getRowTestId(item)}
				data-node-index={index}
				class="group-header cursor-pointer select-none focus:outline-none border-t border-border/40 mt-1 first:mt-0 first:border-t-0"
				class:selected={index === selectedIndex}
				class:focused={index === selectedIndex}
				role="treeitem"
				aria-selected={index === selectedIndex}
				tabindex="-1"
				onclick={() => { selectNode(index); toggleGroup(item.key); }}
			>
				<div class="flex items-center gap-2 py-2 px-2 {index === selectedIndex ? 'bg-zinc-800' : ''} {item.unlabeled ? 'bg-yellow-500/5' : ''}">
					<span class="w-4 text-muted-foreground text-xs">
						{collapsedGroups.has(item.key) ? '▶' : '▼'}
					</span>
					<span class="text-xs font-semibold uppercase tracking-wider {item.unlabeled ? 'text-yellow-500' : 'text-muted-foreground'}">
						{item.label}
					</span>
					<span class="text-xs text-muted-foreground">
						({item.count})
					</span>
					{#if item.unlabeled}
						<span class="text-xs text-yellow-500/70 italic">needs labeling</span>
					{/if}
				</div>
			</div>
		{:else}
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
			onclick={() => selectNode(index)}
		>
		{#if isWIP}
			{#if item.type === 'running'}
				{@const agent = item.agent}
				{@const statusIcon = getAgentStatusIcon(agent)}
				{@const health = computeAgentHealth(agent)}
				{@const contextPct = getContextPercent(agent)}
				<!-- Running Agent - WIP Item -->
			<div class="flex items-center gap-3 py-2 px-1 rounded transition-colors {index === selectedIndex ? 'bg-zinc-800' : ''}" style="padding-left: 0">
				<!-- Status icon with health indication -->
				<span class="{statusIcon.color} w-5 text-center">{statusIcon.icon}</span>
					
					<!-- Priority placeholder (w-8 matches tree badge width) -->
					<span class="w-8"></span>
					
					<!-- ID (min-w-[120px] matches tree) -->
					<span 
						class="text-xs font-mono min-w-[120px] cursor-pointer hover:text-foreground transition-colors {copiedId === (agent.beads_id || agent.id.slice(0, 15)) ? 'text-green-500' : 'text-muted-foreground'}"
						onclick={(e) => { e.stopPropagation(); copyToClipboard(agent.beads_id || agent.id.slice(0, 15)); }}
						onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); e.preventDefault(); copyToClipboard(agent.beads_id || agent.id.slice(0, 15)); }}}
						role="button"
						tabindex="-1"
						title="Click to copy"
					>
						{copiedId === (agent.beads_id || agent.id.slice(0, 15)) ? 'Copied!' : (agent.beads_id || agent.id.slice(0, 15))}
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
				<div class="flex items-center gap-3 py-2 px-1 rounded transition-colors {index === selectedIndex ? 'bg-zinc-800' : ''}" style="padding-left: 0">
					<!-- Expansion indicator placeholder (matches tree nodes) -->
					<span class="w-4"></span>
					
					<!-- Status icon -->
					<span class="text-muted-foreground w-5">○</span>
					
					<!-- Priority badge -->
					<Badge variant={getPriorityVariant(issue.priority)} class="w-8 justify-center text-xs">
						P{issue.priority}
					</Badge>
					
					<!-- ID -->
					<span 
						class="text-xs font-mono min-w-[120px] cursor-pointer hover:text-foreground transition-colors {copiedId === issue.id ? 'text-green-500' : 'text-muted-foreground'}"
						onclick={(e) => { e.stopPropagation(); copyToClipboard(issue.id); }}
						onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); e.preventDefault(); copyToClipboard(issue.id); }}}
						role="button"
						tabindex="-1"
						title="Click to copy"
					>
						{copiedId === issue.id ? 'Copied!' : issue.id}
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
			{@const dimPinned = pinnedTreeIds.has(node.id) && index !== selectedIndex}
			{@const feedback = actionFeedback.get(node.id)}
			<!-- Tree Node - L0: Row -->
			<div
			class="flex items-center gap-3 py-2 px-1 rounded transition-colors {index === selectedIndex ? 'bg-zinc-800' : ''} {node.absorbed_by ? 'opacity-50' : ''} {dimPinned ? 'opacity-60' : ''} {feedback === 'priority' ? 'action-feedback-priority' : ''} {feedback === 'queue' ? 'action-feedback-queue' : ''}"
			style="padding-left: {node.depth * 24}px"
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
					<span 
						class="text-xs font-mono min-w-[120px] cursor-pointer hover:text-foreground transition-colors {copiedId === node.id ? 'text-green-500' : 'text-muted-foreground'}"
						onclick={(e) => { e.stopPropagation(); copyToClipboard(node.id); }}
						onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); e.preventDefault(); copyToClipboard(node.id); }}}
						role="button"
						tabindex="-1"
						title="Click to copy"
					>
						{copiedId === node.id ? 'Copied!' : node.id}
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
		{/if}
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
	
	.action-feedback-priority {
		animation: priority-flash 1s ease-out;
	}
	
	.action-feedback-queue {
		animation: queue-flash 1s ease-out;
	}
	
	@keyframes priority-flash {
		0% {
			background-color: rgba(168, 85, 247, 0.4); /* purple-500 with opacity */
			box-shadow: 0 0 0 2px rgba(168, 85, 247, 0.6);
		}
		100% {
			background-color: transparent;
			box-shadow: 0 0 0 0 transparent;
		}
	}
	
	@keyframes queue-flash {
		0% {
			background-color: rgba(34, 197, 94, 0.4); /* green-500 with opacity */
			box-shadow: 0 0 0 2px rgba(34, 197, 94, 0.6);
		}
		100% {
			background-color: transparent;
			box-shadow: 0 0 0 0 transparent;
		}
	}
</style>
