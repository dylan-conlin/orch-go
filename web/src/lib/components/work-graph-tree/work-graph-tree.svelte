<script lang="ts">
	import { onMount } from 'svelte';
	import { Badge } from '$lib/components/ui/badge';
	import type { TreeNode, GroupSection, GroupByMode } from '$lib/stores/work-graph';
	import { closeIssue, updateIssue } from '$lib/stores/work-graph';
	import type { WIPItem } from '$lib/stores/wip';
	import { computeAgentHealth, getContextPercent, getContextColor, getExpressiveStatus } from '$lib/stores/wip';
	import { DeliverableChecklist } from '$lib/components/deliverable-checklist';
	import { getExpectedDeliverables } from '$lib/stores/deliverables';
	import { IssueSidePanel } from '$lib/components/issue-side-panel';
	import { CloseIssueModal } from '$lib/components/close-issue-modal';
	import { orchestratorContext } from '$lib/stores/context';
	import {
		type GroupHeader,
		type RunningAgentDetails,
		flattenVisibleTree,
		findNodeById,
		formatRelatedIssueList,
		formatStatusLabel,
		getAgentStatusIcon,
		getAttentionBadge,
		getDependencyExplanation,
		getInProgressSubline,
		getIssueSummary,
		getItemId,
		getItemKey,
		getPriorityVariant,
		getProgressSnapshot,
		getRelatedIssueLabel,
		getRowTestId,
		getStatusColor,
		getStatusIcon,
		getTypeBadge,
		isGroupHeader,
		isWIPItem,
		shortenModel,
	} from './work-graph-tree-helpers';

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

	// Flatten tree for keyboard navigation
	let flattenedNodes: (TreeNode | WIPItem | GroupHeader)[] = [];
	let selectedIndex = 0;
	let pinnedTreeIds = new Set<string>();
	let runningAgentDetailsByIssueId = new Map<string, RunningAgentDetails>();

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
	let treeNodeIndex = new Map<string, TreeNode>();

	$: {
		const index = new Map<string, TreeNode>();
		const walk = (nodes: TreeNode[]) => {
			for (const node of nodes) {
				index.set(node.id, node);
				if (node.children.length > 0) {
					walk(node.children);
				}
			}
		};
		walk(tree);
		treeNodeIndex = index;
	}

	// Keep side panel issue in sync with latest tree snapshot so lifecycle tab auto-switching
	// follows status changes while the panel remains open.
	$: if (selectedIssueForPanel) {
		const latestIssue = findNodeById(tree, selectedIssueForPanel.id);
		if (latestIssue && latestIssue !== selectedIssueForPanel) {
			selectedIssueForPanel = latestIssue;
		}
	}

	// Rebuild flattened list when tree, groups, or wipItems change
		$: {
		// Track which tree nodes are already surfaced in WIP (hide duplicates from tree)
		const pinnedIds = new Set<string>();
		const runningDetails = new Map<string, { phase?: string; runtime?: string; model?: string; skill?: string }>();
		for (const item of wipItems) {
			if (item.type === 'running') {
				if (item.agent.beads_id) {
					pinnedIds.add(item.agent.beads_id);
					runningDetails.set(item.agent.beads_id, {
						phase: item.agent.phase,
						runtime: item.agent.runtime,
						model: item.agent.model,
						skill: item.agent.skill,
					});
				}
			} else {
				pinnedIds.add(item.issue.id);
			}
		}
		pinnedTreeIds = pinnedIds;
		runningAgentDetailsByIssueId = runningDetails;

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
					items.push(...flattenVisibleTree(group.nodes, pinnedIds));
				}
			}
		} else {
			items.push(...flattenVisibleTree(tree, pinnedIds));
		}

		flattenedNodes = items;
		// Clamp selected index to valid range
		if (selectedIndex >= flattenedNodes.length) {
			selectedIndex = Math.max(0, flattenedNodes.length - 1);
		}
	}

	function getPanelIssue(item: TreeNode | WIPItem | GroupHeader): TreeNode | null {
		if (isGroupHeader(item)) return null;
		if (!isWIPItem(item)) return item;

		const relatedIssueId = item.type === 'running' ? item.agent.beads_id : item.issue.id;
		if (!relatedIssueId) return null;

		return treeNodeIndex.get(relatedIssueId) || null;
	}

	// Get age from ID (creation timestamp encoded in beads IDs)
	function getAge(id: string): string {
		// Beads IDs have timestamp - could parse it, but for now just return placeholder
		// This would need actual created_at from API
		return ''; // TODO: Add created_at to GraphNode
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
				const panelIssue = getPanelIssue(current);
				if (panelIssue) {
					// Toggle: close if same issue is already open
					if (selectedIssueForPanel?.id === panelIssue.id) {
						selectedIssueForPanel = null;
					} else {
						selectedIssueForPanel = panelIssue;
					}
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
					items.push(...flattenVisibleTree(group.nodes, pinnedTreeIds));
				}
			}
		} else {
			items.push(...flattenVisibleTree(tree, pinnedTreeIds));
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
				{@const displayId = agent.beads_id || agent.id.slice(0, 15)}
				{@const relatedNodeId = agent.beads_id && pinnedTreeIds.has(agent.beads_id) ? agent.beads_id : null}
				{@const relatedNode = relatedNodeId ? treeNodeIndex.get(relatedNodeId) : null}
				{@const statusIcon = getAgentStatusIcon(agent)}
				{@const health = computeAgentHealth(agent)}
				{@const contextPct = getContextPercent(agent)}
				{@const inProgressSubline = relatedNode
					? getInProgressSubline(relatedNode, runningAgentDetailsByIssueId)
					: (agent.phase || agent.runtime || agent.model
						? { text: `${agent.phase || 'active'} · ${agent.runtime || 'runtime unknown'} · ${shortenModel(agent.model)}`, tone: 'text-blue-500/90' }
						: null)}
				<!-- Running Agent - WIP Item -->
			<div class="flex items-start gap-3 py-2 px-1 rounded transition-colors {index === selectedIndex ? 'bg-zinc-800' : ''}" style="padding-left: 0">
				<!-- Expansion indicator placeholder (matches tree nodes) -->
				<span class="w-4"></span>

				<!-- Status icon with health indication -->
				<span class="{statusIcon.color} w-5 text-center">{statusIcon.icon}</span>

				<!-- Priority badge -->
				{#if relatedNode}
					<Badge variant={getPriorityVariant(relatedNode.priority)} class="w-8 justify-center text-xs">
						P{relatedNode.priority}
					</Badge>
				{:else}
					<span class="w-8"></span>
				{/if}

				<!-- ID (min-w-[120px] matches tree) -->
				<span
					class="text-xs font-mono min-w-[120px] cursor-pointer hover:text-foreground transition-colors {copiedId === displayId ? 'text-green-500' : 'text-muted-foreground'}"
					onclick={(e) => { e.stopPropagation(); copyToClipboard(displayId); }}
					onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); e.preventDefault(); copyToClipboard(displayId); }}}
					role="button"
					tabindex="-1"
					title="Click to copy"
				>
					{copiedId === displayId ? 'Copied!' : displayId}
				</span>

				<!-- Title + in-progress details -->
				<div class="flex-1 min-w-0">
					<span class="block text-sm font-medium text-foreground truncate">
						{relatedNode?.title || agent.task || agent.skill || 'Unknown task'}
					</span>
					{#if inProgressSubline}
						<span class="block text-[11px] leading-4 truncate {inProgressSubline.tone}">
							{inProgressSubline.text}
						</span>
					{/if}
					<span class="block text-[11px] leading-4 truncate text-muted-foreground">
						{getExpressiveStatus(agent)}
					</span>
				</div>

				<!-- Attention badge (if any) -->
				{#if relatedNode?.attentionBadge}
					{@const badgeConfig = getAttentionBadge(relatedNode.attentionBadge)}
					{#if badgeConfig}
						<span class="shrink-0 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {badgeConfig.bg} {badgeConfig.color}" title={relatedNode.attentionReason || badgeConfig.label}>
							{badgeConfig.label}
						</span>
					{/if}
				{/if}

				<!-- Type badge -->
				{#if relatedNode}
					<Badge variant="outline" class="{getTypeBadge(relatedNode.type)} text-xs shrink-0">
						{relatedNode.type}
					</Badge>
				{/if}
					
					<!-- Health warning tooltip -->
					{#if health.status !== 'healthy'}
						<span class="text-xs {health.status === 'critical' ? 'text-red-500' : 'text-yellow-500'}" title={health.reasons.join(', ')}>
							{health.status === 'critical' ? '!' : '?'}
						</span>
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
			{@const feedback = actionFeedback.get(node.id)}
			{@const inProgressSubline = getInProgressSubline(node, runningAgentDetailsByIssueId)}
			{@const progressSnapshot = getProgressSnapshot(node)}
			<!-- Tree Node - L0: Row -->
			<div
			class="flex items-center gap-3 py-2 px-1 rounded transition-colors {index === selectedIndex ? 'bg-zinc-800' : ''} {node.status.toLowerCase() === 'in_progress' ? 'border-l border-blue-500/30' : 'border-l border-transparent'} {node.absorbed_by ? 'opacity-50' : ''} {feedback === 'priority' ? 'action-feedback-priority' : ''} {feedback === 'queue' ? 'action-feedback-queue' : ''}"
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

					<!-- Title + in_progress details -->
					<div class="flex-1 min-w-0">
						<span class="block text-sm font-medium text-foreground truncate">
							{node.title}
						</span>
						{#if inProgressSubline}
							<span class="block text-[11px] leading-4 truncate {inProgressSubline.tone}">
								{inProgressSubline.text}
							</span>
						{/if}
					</div>

					{#if progressSnapshot}
						<div class="hidden xl:flex items-center gap-2 min-w-[120px]">
							<div class="w-14 h-1.5 bg-muted rounded-full overflow-hidden">
								<div class="h-full bg-blue-500 transition-all" style="width: {progressSnapshot.percent}%"></div>
							</div>
							<span class="text-[11px] text-muted-foreground tabular-nums">
								{progressSnapshot.done}/{progressSnapshot.total}
							</span>
						</div>
					{/if}

					<!-- Attention badge (if any) -->
					{#if node.attentionBadge}
						{@const badgeConfig = getAttentionBadge(node.attentionBadge)}
						{#if badgeConfig}
							<span class="shrink-0 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {badgeConfig.bg} {badgeConfig.color}" title={node.attentionReason || badgeConfig.label}>
								{badgeConfig.label}
							</span>
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
					{@const dependencyExplanation = getDependencyExplanation(node, treeNodeIndex)}
					{@const progressInDetails = getProgressSnapshot(node)}
					{@const parentLabel = node.parent_id ? getRelatedIssueLabel(node.parent_id, treeNodeIndex) : null}
					{@const directChildLabels = node.children.map((child) => getRelatedIssueLabel(child.id, treeNodeIndex))}
					<div
						data-testid={`issue-details-${node.id}`}
						class="expanded-details ml-12 mt-1 mb-2 p-3 bg-muted/30 rounded text-sm"
						style="margin-left: {node.depth * 24 + 48}px"
					>
						<div class="mb-3">
							<span class="text-xs font-semibold uppercase text-foreground">Issue summary:</span>
							<p class="mt-1 text-xs text-muted-foreground">{getIssueSummary(node)}</p>
						</div>

						<div class="mb-3">
							<span class="text-xs font-semibold uppercase text-foreground">Dependency context:</span>
							<p class="mt-1 text-xs {dependencyExplanation.tone}">{dependencyExplanation.headline}</p>
							<p class="mt-1 text-[11px] text-muted-foreground">{dependencyExplanation.detail}</p>
						</div>

						{#if progressInDetails}
							<div class="mb-3">
								<div class="flex items-center justify-between gap-2">
									<span class="text-xs font-semibold uppercase text-foreground">Progress & completeness:</span>
									<span class="text-xs text-muted-foreground tabular-nums">{progressInDetails.done}/{progressInDetails.total} done ({progressInDetails.percent}%)</span>
								</div>
								<div class="mt-1 h-1.5 bg-muted rounded-full overflow-hidden">
									<div class="h-full bg-blue-500 transition-all" style="width: {progressInDetails.percent}%"></div>
								</div>
								<p class="mt-1 text-[11px] text-muted-foreground">
									{#if progressInDetails.visible === progressInDetails.total}
										All {progressInDetails.total} related issues are visible in this branch.
									{:else}
										{progressInDetails.visible} of {progressInDetails.total} related issues are visible (expand children to inspect all).
									{/if}
								</p>
							</div>
						{/if}

						<div class="mb-2">
							<span class="text-xs font-semibold uppercase text-foreground">Related issues:</span>
							<div class="mt-1 space-y-1 text-xs text-muted-foreground">
								{#if parentLabel}
									<div>Parent: {parentLabel}</div>
								{/if}
								{#if directChildLabels.length > 0}
									<div>Children ({directChildLabels.length}): {formatRelatedIssueList(node.children.map((child) => child.id), treeNodeIndex)}</div>
								{/if}
								{#if node.blocked_by.length > 0}
									<div>Upstream blockers: {formatRelatedIssueList(node.blocked_by, treeNodeIndex)}</div>
								{/if}
								{#if node.blocks.length > 0}
									<div>Downstream dependents: {formatRelatedIssueList(node.blocks, treeNodeIndex)}</div>
								{/if}
								{#if node.absorbed_by}
									<div>Absorbed by: {getRelatedIssueLabel(node.absorbed_by, treeNodeIndex)}</div>
								{/if}
								{#if !parentLabel && directChildLabels.length === 0 && node.blocked_by.length === 0 && node.blocks.length === 0 && !node.absorbed_by}
									<div>No directly related issues in the current scope.</div>
								{/if}
							</div>
						</div>

						<!-- Status details -->
						<div class="flex flex-wrap items-center gap-4 text-xs border-t border-border pt-2 mt-2">
							<span class="flex items-center gap-1">
								<span class="text-foreground/60">Status:</span>
								<span class={getStatusColor(node.status)}>{formatStatusLabel(node.status)}</span>
							</span>
							<span class="flex items-center gap-1">
								<span class="text-foreground/60">Priority:</span>
								<span class="text-muted-foreground">P{node.priority}</span>
							</span>
							<span class="flex items-center gap-1">
								<span class="text-foreground/60">Type:</span>
								<span class="text-muted-foreground">{node.type}</span>
							</span>
							<span class="flex items-center gap-1">
								<span class="text-foreground/60">Source:</span>
								<span class="text-muted-foreground">{node.source}</span>
							</span>
						</div>
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
