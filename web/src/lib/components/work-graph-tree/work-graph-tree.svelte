<script lang="ts">
	import { onMount } from 'svelte';
	import type { TreeNode, GroupSection, GroupByMode, GraphEdge } from '$lib/stores/work-graph';
	import { closeIssue, updateIssue, buildDependencyView, flattenDepChain } from '$lib/stores/work-graph';
	import type { WIPItem } from '$lib/stores/wip';
	import { IssueSidePanel } from '$lib/components/issue-side-panel';
	import { CloseIssueModal } from '$lib/components/close-issue-modal';
	import { orchestratorContext } from '$lib/stores/context';
	import type { Agent } from '$lib/stores/agents';
	import {
		type GroupHeader,
		type DepSectionHeader,
		type RunningAgentDetails,
		flattenVisibleTree,
		findNodeById,
		getIndependentPreview,
		getItemId,
		getItemKey,
		getPanelIssue,
		getRowTestId,
		isGroupHeader,
		isDepSectionHeader,
		isWIPItem,
		sortNodesByPriorityAndRecency,
	} from './work-graph-tree-helpers';
	import WipRow from './wip-row.svelte';
	import TreeNodeRow from './tree-node-row.svelte';

	export let tree: TreeNode[] = [];
	export let groups: GroupSection[] = [];
	export let groupMode: GroupByMode = 'priority';
	export let edges: GraphEdge[] = [];
	export let wipItems: WIPItem[] = [];
	export let excludeIds: Set<string> = new Set();
	export let agentsByBeadsId: Map<string, Agent> = new Map();
	export let onToggleExpansion: (nodeId: string, expanded: boolean) => void = () => {};
	export let onSetFocus: (beadsId: string, title: string) => void = () => {};

	// Track which group sections are collapsed
	let collapsedGroups = new Set<string>();

	// Track which dependency chain sections are collapsed
	let collapsedChains = new Set<string>();

	// Flow connector prefix map for dependency chain nodes
	let depPrefixMap = new Map<string, string>();
	// Gate tracking: items that are convergence points in dependency chains
	let depGateIds = new Set<string>();
	let depGateSeparatorBefore = new Set<string>();
	let showAllIndependent = false;
	let independentHiddenCount = 0;
	let independentHasOverflow = false;

	function toggleGroup(key: string) {
		if (collapsedGroups.has(key)) {
			collapsedGroups.delete(key);
		} else {
			collapsedGroups.add(key);
		}
		collapsedGroups = collapsedGroups; // trigger reactivity
	}

	function toggleChain(key: string) {
		if (collapsedChains.has(key)) {
			collapsedChains.delete(key);
		} else {
			collapsedChains.add(key);
		}
		collapsedChains = collapsedChains; // trigger reactivity
	}

	// Whether we're in grouped mode (non-priority has label sections)
	$: isGrouped = groups.length > 0 && groupMode !== 'priority';

	// Flatten tree for keyboard navigation
	let flattenedNodes: (TreeNode | WIPItem | GroupHeader | DepSectionHeader)[] = [];
	let selectedIndex = 0;
	let pinnedTreeIds = new Set<string>();
	let runningAgentDetailsByIssueId = new Map<string, RunningAgentDetails>();
	let visibleWipItems: WIPItem[] = [];

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

	function buildFlattenedItems(): (TreeNode | WIPItem | GroupHeader | DepSectionHeader)[] {
		independentHiddenCount = 0;
		independentHasOverflow = false;
		const excludedIds = new Set([...pinnedTreeIds, ...excludeIds]);
		const items: (TreeNode | WIPItem | GroupHeader | DepSectionHeader)[] = [...visibleWipItems];

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
					items.push(...flattenVisibleTree(group.nodes, excludedIds));
				}
			}
			depPrefixMap = new Map();
			depGateIds = new Set();
			depGateSeparatorBefore = new Set();
			return items;
		}

		const hasBlockingEdges = edges.some(
			(e) => e.type === 'blocks' && treeNodeIndex.has(e.from) && treeNodeIndex.has(e.to),
		);
		const shouldUseDependencyView = groupMode === 'dep-chain' || (hasBlockingEdges && !isGrouped);

		if (shouldUseDependencyView) {
			const dv = buildDependencyView(treeNodeIndex, edges);
			const prefixes = new Map<string, string>();
			const gateIds = new Set<string>();
			const gateSepBefore = new Set<string>();

			for (const chain of dv.chains) {
				items.push({
					_depSectionHeader: true,
					key: chain.id,
					label: chain.label,
					count: chain.size,
					type: 'chain' as const,
				});

				if (!collapsedChains.has(chain.id)) {
					const flatItems = flattenDepChain(chain, excludedIds);
					let firstGateInChain = true;
					for (const fi of flatItems) {
						if (fi.isGate) {
							gateIds.add(fi.node.id);
							if (firstGateInChain) {
								gateSepBefore.add(fi.node.id);
								firstGateInChain = false;
							}
						}
						prefixes.set(fi.node.id, fi.prefix);
						items.push(fi.node);
					}
				}
			}

			const availableIndependent = dv.independentNodes.filter((node) => !excludedIds.has(node.id));
			if (availableIndependent.length > 0) {
				items.push({
					_depSectionHeader: true,
					key: 'independent',
					label: 'Independent Issues',
					count: availableIndependent.length,
					type: 'independent' as const,
				});

				if (!collapsedChains.has('independent')) {
					const ordered = sortNodesByPriorityAndRecency(availableIndependent);
					const preview = getIndependentPreview(ordered);
					const visible = showAllIndependent ? ordered : preview.visible;
					independentHiddenCount = showAllIndependent ? 0 : preview.hidden;
					independentHasOverflow = preview.hidden > 0;
					for (const node of visible) {
						items.push(node);
					}
				} else {
					independentHiddenCount = 0;
					independentHasOverflow = false;
				}
			}

			depPrefixMap = prefixes;
			depGateIds = gateIds;
			depGateSeparatorBefore = gateSepBefore;
			return items;
		}

		items.push(...flattenVisibleTree(tree, excludedIds));
		depPrefixMap = new Map();
		depGateIds = new Set();
		depGateSeparatorBefore = new Set();
		return items;
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
		const filteredWipItems: WIPItem[] = [];
		for (const item of wipItems) {
			const itemId = item.type === 'running' ? item.agent.beads_id : item.issue.id;
			if (itemId && excludeIds.has(itemId)) {
				continue;
			}
			filteredWipItems.push(item);
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
		visibleWipItems = filteredWipItems;
		pinnedTreeIds = pinnedIds;
		runningAgentDetailsByIssueId = runningDetails;
		// Ensure reactive dependencies for flattened list rebuilds
		tree;
		groups;
		edges;
		isGrouped;
		treeNodeIndex;
		collapsedGroups;
		collapsedChains;
		showAllIndependent;
		excludeIds;

		flattenedNodes = buildFlattenedItems();
		// Clamp selected index to valid range
		if (selectedIndex >= flattenedNodes.length) {
			selectedIndex = Math.max(0, flattenedNodes.length - 1);
		}
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

		// Handle dependency section header interactions
		if (isDepSectionHeader(current)) {
			if (event.key === 'Enter' || event.key === 'l' || event.key === 'h' || event.key === 'ArrowRight' || event.key === 'ArrowLeft') {
				event.preventDefault();
				toggleChain(current.key);
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
				const panelIssue = getPanelIssue(current, treeNodeIndex);
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
		flattenedNodes = buildFlattenedItems();
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
		if (selectedIssueForPanel?.id === node.id) {
			selectedIssueForPanel = null;
			return;
		}
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
		{@const isWIP = !isGroupHeader(item) && !isDepSectionHeader(item) && isWIPItem(item)}
		{@const depth = !isGroupHeader(item) && !isDepSectionHeader(item) && !isWIP ? (item as TreeNode).depth : undefined}
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
		{:else if isDepSectionHeader(item)}
			<!-- Dependency Chain Section Header -->
			<div
				data-testid={getRowTestId(item)}
				data-node-index={index}
				class="dep-section-header cursor-pointer select-none focus:outline-none border-t border-border/40 mt-1 first:mt-0 first:border-t-0"
				class:selected={index === selectedIndex}
				class:focused={index === selectedIndex}
				role="treeitem"
				aria-selected={index === selectedIndex}
				tabindex="-1"
				onclick={() => { selectNode(index); toggleChain(item.key); }}
			>
				<div class="flex items-center gap-2 py-2 px-2 {index === selectedIndex ? 'bg-zinc-800' : ''}">
					<span class="w-4 text-muted-foreground text-xs">
						{collapsedChains.has(item.key) ? '▶' : '▼'}
					</span>
					{#if item.type === 'chain'}
						<span class="text-xs text-blue-400/60 font-mono">▸</span>
					{/if}
					<span class="text-xs font-semibold tracking-wider {item.type === 'independent' ? 'text-muted-foreground uppercase' : 'text-blue-400'} truncate max-w-md">
						{item.label}
					</span>
					<span class="text-xs text-muted-foreground">
						({item.count})
					</span>
					{#if item.type === 'independent' && independentHasOverflow}
						<button
							class="ml-auto text-[11px] text-muted-foreground hover:text-foreground"
							onclick={(event) => {
								event.stopPropagation();
								showAllIndependent = !showAllIndependent;
							}}
						>
							{showAllIndependent ? 'Show less' : `Show all (${independentHiddenCount} more)`}
						</button>
					{/if}
				</div>
			</div>
		{:else}
		<!-- Gate separator: visual signal that everything above must complete before this closes -->
		{#if !isWIP && depGateSeparatorBefore.has((item as TreeNode).id) && depPrefixMap.get((item as TreeNode).id) !== ''}
			<div class="flex items-center gap-2 py-1.5 px-3 select-none">
				<div class="flex-1 border-t border-dashed border-zinc-600"></div>
				<span class="text-[10px] uppercase tracking-widest text-zinc-500 font-medium">gate</span>
				<div class="flex-1 border-t border-dashed border-zinc-600"></div>
			</div>
		{/if}
		<div
			data-testid={getRowTestId(item)}
			data-node-index={index}
			data-depth={depth !== undefined ? String(depth) : null}
			class="node-row cursor-pointer select-none focus:outline-none"
			class:selected={index === selectedIndex}
			class:focused={index === selectedIndex}

			role="treeitem"
			aria-selected={index === selectedIndex}
			tabindex="-1"
			onclick={() => selectNode(index)}
		>
		{#if isWIP}
			<WipRow
				{item}
				{index}
				{selectedIndex}
				{itemId}
				{copiedId}
				{expandedDetails}
				{pinnedTreeIds}
				{treeNodeIndex}
				{runningAgentDetailsByIssueId}
				onSelectNode={selectNode}
				onCopyToClipboard={copyToClipboard}
				onOpenSidePanel={openSidePanel}
			/>
		{:else}
			<TreeNodeRow
				node={item as TreeNode}
				{index}
				{selectedIndex}
				{copiedId}
				{expandedDetails}
				depPrefix={depPrefixMap.get((item as TreeNode).id)}
				{treeNodeIndex}
				{runningAgentDetailsByIssueId}
				{actionFeedback}
				{agentsByBeadsId}
				onSelectNode={selectNode}
				onCopyToClipboard={copyToClipboard}
				onOpenSidePanel={openSidePanel}
				{onSetFocus}
			/>
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
</style>
