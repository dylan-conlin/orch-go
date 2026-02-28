<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { derived } from 'svelte/store';
	import { workGraph, buildTree, groupTreeNodes, type TreeNode, type GroupSection, type GroupByMode } from '$lib/stores/work-graph';
	import { orchestratorContext } from '$lib/stores/context';
	import { agents, connectSSE, disconnectSSE, sseEvents, connectionStatus, type Agent } from '$lib/stores/agents';
	import {
		agentlogEvents,
		connectAgentlogSSE,
		disconnectAgentlogSSE,
		type AgentLogEvent,
	} from '$lib/stores/agentlog';
	import { WorkGraphTree } from '$lib/components/work-graph-tree';
	import { GroupByDropdown } from '$lib/components/group-by-dropdown';
	import { wip, wipItems } from '$lib/stores/wip';
	import { daemon, type DaemonStatus } from '$lib/stores/daemon';
	import { attention, formatRelativeTime } from '$lib/stores/attention';
	import { focus, type FocusInfo } from '$lib/stores/focus';

	const WORK_GRAPH_POLL_INTERVAL_MS = 30000;
	const WORK_GRAPH_MAX_BACKOFF_MS = 120000;
	const CONTEXT_POLL_INTERVAL_MS = 15000;
	const EVENT_DRIVEN_REFRESH_THROTTLE_MS = 3000;
	const EVENT_DRIVEN_REFRESH_TYPES = new Set([
		'session.created',
		'session.deleted',
		'agent.completed',
		'agent.abandoned',
	]);
	const AGENTLOG_EVENT_DRIVEN_REFRESH_TYPES = new Set([
		'session.spawned',
		'session.completed',
		'session.error',
		'session.auto_completed',
		'agent.completed',
		'agent.abandoned',
	]);
	
	// Derived store for project_dir to isolate reactivity
	// Only triggers reactive blocks when project_dir changes, not other context fields
	const projectDir = derived(orchestratorContext, $ctx => $ctx.project_dir);

	// Per-project seen issues tracking to prevent false highlights on project switch
	const SEEN_ISSUES_KEY = 'work-graph-seen-issues';
	
	interface SeenIssuesState {
		byProject: Record<string, {
			issueIds: string[];
			firstSeenAt: string; // ISO timestamp
		}>;
	}
	
	function loadSeenIssues(): SeenIssuesState {
		if (typeof window === 'undefined') return { byProject: {} };
		try {
			const stored = localStorage.getItem(SEEN_ISSUES_KEY);
			if (stored) {
				return JSON.parse(stored);
			}
		} catch (e) {
			console.error('Failed to load seen issues from localStorage:', e);
		}
		return { byProject: {} };
	}
	
	function saveSeenIssues(state: SeenIssuesState): void {
		if (typeof window === 'undefined') return;
		try {
			localStorage.setItem(SEEN_ISSUES_KEY, JSON.stringify(state));
		} catch (e) {
			console.error('Failed to save seen issues to localStorage:', e);
		}
	}

	let tree: TreeNode[] = [];
	let loading = true;
	let error: string | null = null;
	let refreshTimeout: ReturnType<typeof setTimeout> | null = null;
	let isRefreshCycleInFlight = false;
	let isRefreshPolling = false;
	let refreshBackoffMs = WORK_GRAPH_POLL_INTERVAL_MS;
	let lastEventDrivenRefreshAt = 0;
	let lastProcessedSSEEventId: string | null = null;
	let lastProcessedAgentlogEventId: string | null = null;
	let agentlogRealtimeStartUnix = 0;
	let seenIssuesState: SeenIssuesState = { byProject: {} };
	let currentProjectDir: string | undefined = undefined;
	let projectChangeDebounceTimeout: ReturnType<typeof setTimeout> | null = null;
	let previousIssueIds = new Set<string>();
	let focusedBeadsId: string | undefined = undefined; // Current focus beads ID for auto-scoping
	let groupByMode: GroupByMode = 'priority';

	interface ReadyToCompleteItem {
		id: string;
		title: string;
		type: string;
		priority: number;
		runtime?: string;
		tokenTotal: number | null;
		completionAt: string;
	}

	let readyToCompleteItems: ReadyToCompleteItem[] = [];
	let readyToCompleteIds = new Set<string>();
	
	// Persist groupBy mode in localStorage
	const GROUP_BY_KEY = 'work-graph-group-by';
	if (typeof window !== 'undefined') {
		const stored = localStorage.getItem(GROUP_BY_KEY);
		if (stored === 'priority' || stored === 'area' || stored === 'effort' || stored === 'dep-chain') {
			groupByMode = stored;
		}
	}
	
	// Track expansion state separately to preserve across tree rebuilds
	let expansionState = new Map<string, boolean>();
	
	// Debounce timeout for tree rebuild to batch rapid store updates
	let rebuildDebounceTimeout: ReturnType<typeof setTimeout> | null = null;
	let hasRenderedTree = false; // Skip debounce until first tree render completes

	async function runRefreshCycle(): Promise<boolean> {
		const projectDir = $orchestratorContext?.project_dir;

		const requests: Promise<void>[] = [
			workGraph.fetch(projectDir, 'open', focusedBeadsId),
			wip.fetchQueued(projectDir),
			daemon.fetch(),
			attention.fetch(projectDir),
			agents.fetch(), // Refresh agents to detect phase transitions (Phase: Complete via bd comment)
		];

		const results = await Promise.allSettled(requests);
		return results.every((result) => result.status === 'fulfilled');
	}

	function scheduleNextRefresh(delayMs: number): void {
		if (!isRefreshPolling || refreshTimeout) return;

		refreshTimeout = setTimeout(async () => {
			refreshTimeout = null;
			if (!isRefreshPolling) return;

			if (isRefreshCycleInFlight) {
				scheduleNextRefresh(refreshBackoffMs);
				return;
			}

			isRefreshCycleInFlight = true;
			try {
				const ok = await runRefreshCycle();
				if (ok) {
					refreshBackoffMs = WORK_GRAPH_POLL_INTERVAL_MS;
				} else {
					refreshBackoffMs = Math.min(refreshBackoffMs * 2, WORK_GRAPH_MAX_BACKOFF_MS);
				}
			} finally {
				isRefreshCycleInFlight = false;
				scheduleNextRefresh(refreshBackoffMs);
			}
		}, delayMs);
	}

	function startRefreshPolling(): void {
		if (isRefreshPolling) return;
		isRefreshPolling = true;
		refreshBackoffMs = WORK_GRAPH_POLL_INTERVAL_MS;
		scheduleNextRefresh(refreshBackoffMs);
	}

	async function triggerEventDrivenRefresh(): Promise<void> {
		if (!isRefreshPolling || isRefreshCycleInFlight) return;

		const now = Date.now();
		if (now - lastEventDrivenRefreshAt < EVENT_DRIVEN_REFRESH_THROTTLE_MS) {
			return;
		}
		lastEventDrivenRefreshAt = now;

		if (refreshTimeout) {
			clearTimeout(refreshTimeout);
			refreshTimeout = null;
		}

		isRefreshCycleInFlight = true;
		try {
			const ok = await runRefreshCycle();
			if (ok) {
				refreshBackoffMs = WORK_GRAPH_POLL_INTERVAL_MS;
			} else {
				refreshBackoffMs = Math.min(refreshBackoffMs * 2, WORK_GRAPH_MAX_BACKOFF_MS);
			}
		} finally {
			isRefreshCycleInFlight = false;
			scheduleNextRefresh(refreshBackoffMs);
		}
	}

	// Fetch work graph and agents on mount, connect to SSE for real-time updates
	onMount(async () => {
		// Load seen issues from localStorage
		seenIssuesState = loadSeenIssues();
		
		// Start orchestratorContext polling (slower cadence to avoid backend saturation)
		orchestratorContext.startPolling(CONTEXT_POLL_INTERVAL_MS);

		const projectDir = $orchestratorContext?.project_dir;
		currentProjectDir = projectDir;
		
		// Fetch focus first to get the beads_id for auto-scoping
		await focus.fetch();
		const focusBeadsId = $focus?.beads_id;
		focusedBeadsId = focusBeadsId;
		
		await Promise.all([
			workGraph.fetch(projectDir, 'open', focusBeadsId),
			agents.fetch(),
			attention.fetch(projectDir) // Fetch attention signals and completed issues (filtered by project)
		]);

		// Fetch WIP and daemon data (non-blocking)
		wip.fetchQueued(projectDir).catch(console.error);
		daemon.fetch().catch(console.error);
		
		loading = false;
		
		// Initialize previousIssueIds from stored state OR initial fetch
		if (projectDir && seenIssuesState.byProject[projectDir]) {
			// Use stored state for this project
			previousIssueIds = new Set(seenIssuesState.byProject[projectDir].issueIds);
		} else if ($workGraph?.nodes) {
			// First time seeing this project - store all current issues as "seen"
			previousIssueIds = new Set($workGraph.nodes.map(n => n.id));
			if (projectDir) {
				seenIssuesState.byProject[projectDir] = {
					issueIds: Array.from(previousIssueIds),
					firstSeenAt: new Date().toISOString()
				};
				saveSeenIssues(seenIssuesState);
			}
		}
		
		// Connect to SSE for real-time agent updates (WIP section)
		connectSSE();
		agentlogRealtimeStartUnix = Math.floor(Date.now() / 1000);
		connectAgentlogSSE();

		// Keep a low-frequency poll as fallback; rely on SSE for responsive refreshes
		startRefreshPolling();
	});

	$: if ($sseEvents.length > 0 && !loading) {
		const latestEvent = $sseEvents[$sseEvents.length - 1];
		if (latestEvent.id !== lastProcessedSSEEventId) {
			lastProcessedSSEEventId = latestEvent.id;
			if (EVENT_DRIVEN_REFRESH_TYPES.has(latestEvent.type)) {
				triggerEventDrivenRefresh().catch(console.error);
			}
		}
	}

	$: if ($agentlogEvents.length > 0 && !loading) {
		const latestAgentlogEvent = $agentlogEvents[$agentlogEvents.length - 1] as AgentLogEvent;
		if (latestAgentlogEvent.id !== lastProcessedAgentlogEventId) {
			lastProcessedAgentlogEventId = latestAgentlogEvent.id;
			if (
				latestAgentlogEvent.timestamp >= agentlogRealtimeStartUnix &&
				AGENTLOG_EVENT_DRIVEN_REFRESH_TYPES.has(latestAgentlogEvent.type)
			) {
				triggerEventDrivenRefresh().catch(console.error);
			}
		}
	}

	// Subscribe to focus changes and update focusedBeadsId for auto-scoping
	$: if ($focus?.beads_id) {
		focusedBeadsId = $focus.beads_id;
	} else {
		focusedBeadsId = undefined;
	}

	// Sync running agents from agents store to WIP store
	$: wip.setRunningAgents($agents);

	// Disconnect SSE and stop polling on unmount
	onDestroy(() => {
		disconnectSSE();
		disconnectAgentlogSSE();
		orchestratorContext.stopPolling();
		isRefreshPolling = false;
		if (refreshTimeout) {
			clearTimeout(refreshTimeout);
			refreshTimeout = null;
		}
		isRefreshCycleInFlight = false;
		if (projectChangeDebounceTimeout) {
			clearTimeout(projectChangeDebounceTimeout);
			projectChangeDebounceTimeout = null;
		}
		if (rebuildDebounceTimeout) {
			clearTimeout(rebuildDebounceTimeout);
			rebuildDebounceTimeout = null;
		}
		// Cancel any pending workGraph fetches
		workGraph.cancelPending();
	});

	// Build a dedicated review queue for agents that reported Phase: Complete
	// while their beads issue is still in_progress (not closed yet).
	$: {
		const queueByIssue = new Map<string, ReadyToCompleteItem>();
		const nodesById = new Map(($workGraph?.nodes || []).map((node) => [node.id, node]));

		for (const agent of $agents || []) {
			const beadsId = agent.beads_id;
			if (!beadsId) continue;
			// Include 'completed' status because determineAgentStatus() returns 'completed'
			// for Phase: Complete agents whose sessions are still alive (Priority 3 in cascade).
			// Closed issues are excluded below via the issueNode.status check (workGraph scope=open
			// means closed issues aren't in the graph at all).
			if (agent.status !== 'active' && agent.status !== 'awaiting-cleanup' && agent.status !== 'completed') continue;
			if (!agent.phase?.toLowerCase().startsWith('complete')) continue;

			const issueNode = nodesById.get(beadsId);
			if (!issueNode || issueNode.source !== 'beads') continue;
			if (issueNode.status.toLowerCase() !== 'in_progress') continue;

			const completionAt = agent.phase_reported_at || agent.updated_at || agent.spawned_at;
			if (!completionAt) continue;

			const candidate: ReadyToCompleteItem = {
				id: beadsId,
				title: issueNode.title,
				type: issueNode.type,
				priority: issueNode.priority,
				runtime: agent.runtime,
				tokenTotal: getAgentTokenTotal(agent),
				completionAt,
			};

			const existing = queueByIssue.get(beadsId);
			if (!existing || completionMs(candidate) > completionMs(existing)) {
				queueByIssue.set(beadsId, candidate);
			}
		}

		readyToCompleteItems = Array.from(queueByIssue.values()).sort((a, b) => {
			const ageDiff = completionMs(a) - completionMs(b); // oldest completion first
			if (ageDiff !== 0) return ageDiff;
			if (a.priority !== b.priority) return a.priority - b.priority;
			return a.id.localeCompare(b.id);
		});
	}

	$: {
		readyToCompleteIds = new Set(readyToCompleteItems.map((item) => item.id));
	}


	// Rebuild tree and phases whenever graph data OR attention changes
	// Debounced to batch rapid updates and reduce CPU during polling
	// Skip debounce until first tree render completes for immediate display
	$: if ($workGraph && !$workGraph.error) {
		// Cancel any pending rebuild
		if (rebuildDebounceTimeout) {
			clearTimeout(rebuildDebounceTimeout);
		}
		
		// Debounce rebuild to batch rapid updates (50ms is fast but still batches)
		const executeRebuild = () => {
			rebuildDebounceTimeout = null;

			// Build tree from full open set
			tree = buildTree($workGraph.nodes, $workGraph.edges);
			
			// Mark that we've completed first render (enable debouncing for subsequent updates)
			hasRenderedTree = true;

			// Apply stored expansion state to preserve user's collapse/expand choices
			const applyExpansionState = (nodes: TreeNode[]) => {
				for (const node of nodes) {
					// If we have stored expansion state for this node, apply it
					// Otherwise keep the default from buildTree (which is expanded: true)
					if (expansionState.has(node.id)) {
						node.expanded = expansionState.get(node.id)!;
					} else {
						// First time seeing this node, store its default state
						expansionState.set(node.id, node.expanded);
					}
					// Recursively apply to children
					if (node.children.length > 0) {
						applyExpansionState(node.children);
					}
				}
			};
			applyExpansionState(tree);

			// Attach attention badges to tree nodes
			if ($attention?.signals) {
				const attachBadges = (nodes: TreeNode[]) => {
					for (const node of nodes) {
						const signal = $attention.signals.get(node.id);
						if (signal) {
							node.attentionBadge = signal.badge;
							node.attentionReason = signal.reason;
						}
						if (node.children.length > 0) {
							attachBadges(node.children);
						}
					}
				};
				attachBadges(tree);
			}

			error = null;
			
			if ($workGraph.nodes) {
				const currentIssueIds = new Set($workGraph.nodes.map(n => n.id));
				const projectDir = $orchestratorContext?.project_dir;

				// Update previousIssueIds for next comparison
				previousIssueIds = currentIssueIds;
				
				// Persist seen issues to localStorage for this project
				if (projectDir) {
					const existingFirstSeen = seenIssuesState.byProject[projectDir]?.firstSeenAt;
					seenIssuesState.byProject[projectDir] = {
						issueIds: Array.from(currentIssueIds),
						firstSeenAt: existingFirstSeen || new Date().toISOString()
					};
					saveSeenIssues(seenIssuesState);
				}
			}
		};
		
		// Execute immediately until first tree render, then debounce subsequent updates
		if (hasRenderedTree) {
			rebuildDebounceTimeout = setTimeout(executeRebuild, 50); // 50ms batches rapid updates
		} else {
			executeRebuild(); // Immediate for first render
		}
	} else if ($workGraph?.error) {
		error = $workGraph.error;
		tree = [];
	}
	
	// Re-fetch workGraph when orchestrator project_dir changes
	// Uses derived store to isolate reactivity (only fires when project_dir changes)
	// Uses debounce + abort to prevent flip-flopping between old/new project data
	$: {
		if (typeof window !== 'undefined' && $projectDir) {
			const newProjectDir = $projectDir;
			
			// Only react to actual project changes (not other context changes)
			if (newProjectDir !== currentProjectDir) {
				// Cancel any pending debounced fetch
				if (projectChangeDebounceTimeout) {
					clearTimeout(projectChangeDebounceTimeout);
				}
				
				// Cancel any in-flight workGraph requests immediately
				workGraph.cancelPending();
				
				// Update state synchronously to prevent stale comparisons
				currentProjectDir = newProjectDir;

				// Load seen issues for this project from localStorage
				if (seenIssuesState.byProject[newProjectDir]) {
					previousIssueIds = new Set(seenIssuesState.byProject[newProjectDir].issueIds);
				} else {
					// New project we haven't seen before - will be populated on first fetch
					previousIssueIds = new Set<string>();
				}
				
				// Debounce the actual fetch to wait for stable project value
				// 300ms prevents rapid flip-flopping while still feeling responsive
				projectChangeDebounceTimeout = setTimeout(() => {
					projectChangeDebounceTimeout = null;
					workGraph.fetch(newProjectDir, 'open', focusedBeadsId).catch(console.error);
					attention.fetch(newProjectDir).catch(console.error); // Re-fetch attention for new project
				}, 300);
			}
		}
	}

	// Manual retry handler
	async function handleRetry() {
		await orchestratorContext.retry();
	}

	// Handle expansion state updates from tree component
	function handleToggleExpansion(nodeId: string, expanded: boolean) {
		expansionState.set(nodeId, expanded);
	}

	// Keyboard navigation for Tab to toggle views
	// Handle clear focus button click
	async function handleClearFocus() {
		const result = await focus.clearFocus();
		if (result.success) {
			// Refresh the work graph without focus filter
			focusedBeadsId = undefined;
			const projectDir = $orchestratorContext?.project_dir;
			workGraph.fetch(projectDir, 'open').catch(console.error);
		} else {
			console.error('Failed to clear focus:', result.error);
		}
	}

	// Handle setting focus on an epic
	async function handleSetFocus(beadsId: string, title: string) {
		const result = await focus.setFocus(title, beadsId);
		if (result.success) {
			// Update local state and refresh graph with new focus
			focusedBeadsId = beadsId;
			const projectDir = $orchestratorContext?.project_dir;
			workGraph.fetch(projectDir, 'open', beadsId).catch(console.error);
		} else {
			console.error('Failed to set focus:', result.error);
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		// 'G' (shift+g) to cycle group mode
		if (event.key === 'G' && event.shiftKey) {
			const active = document.activeElement;
			if (active?.tagName !== 'INPUT' && active?.tagName !== 'TEXTAREA') {
				event.preventDefault();
				event.stopPropagation();
				const idx = groupOrder.indexOf(groupByMode);
				handleGroupByChange(groupOrder[(idx + 1) % groupOrder.length]);
			}
		}
	}
	
	// Get help text based on current view mode
	$: filteredTree = tree;

	// Compute group sections from filtered tree
	$: groupSections = (groupByMode === 'area' || groupByMode === 'effort')
		? groupTreeNodes(filteredTree, groupByMode)
		: [] as GroupSection[];

	function handleGroupByChange(mode: GroupByMode) {
		groupByMode = mode;
		if (typeof window !== 'undefined') {
			localStorage.setItem(GROUP_BY_KEY, mode);
		}
	}

	function getAgentTokenTotal(agent: Agent): number | null {
		const tokens = agent.tokens;
		if (!tokens) return null;

		const total =
			tokens.total_tokens ??
			(tokens.input_tokens || 0) +
				(tokens.output_tokens || 0) +
				(tokens.cache_read_tokens || 0);

		if (!Number.isFinite(total) || total <= 0) {
			return null;
		}

		return total;
	}

	function formatTokenTotal(total: number | null): string {
		if (total === null) return 'tokens unknown';
		if (total >= 1_000_000) return `${(total / 1_000_000).toFixed(1)}M tokens`;
		if (total >= 1_000) return `${(total / 1_000).toFixed(1)}k tokens`;
		return `${total} tokens`;
	}

	function completionMs(item: ReadyToCompleteItem): number {
		const ms = new Date(item.completionAt).getTime();
		if (Number.isNaN(ms)) return 0;
		return ms;
	}

	// Cycle group mode order for 'g' shortcut
	const groupOrder: GroupByMode[] = ['priority', 'area', 'effort', 'dep-chain'];

	function daemonQueueSummary(status: DaemonStatus): string {
		const queued = status.queue?.queued ?? status.ready_count ?? 0;
		const reasons: string[] = [];
		if ((status.queue?.waiting_for_slots ?? 0) > 0) {
			reasons.push(`${status.queue?.waiting_for_slots} waiting for slots`);
		}
		if ((status.queue?.grace_period ?? 0) > 0) {
			reasons.push(`${status.queue?.grace_period} in grace period`);
		}
		if ((status.queue?.processed_cache ?? 0) > 0) {
			reasons.push(`${status.queue?.processed_cache} in processed cache`);
		}

		if (queued === 0 || reasons.length === 0) {
			return `${queued} queued`;
		}

		return `${queued} queued (${reasons.join(', ')})`;
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="work-graph-container flex flex-col h-[calc(100vh-4rem)] overflow-hidden bg-background">
	<!-- Backend Error Banner -->
	{#if $connectionStatus.status === 'disconnected'}
		<div 
			class="bg-red-500/10 border-b border-red-500/20 px-4 py-3 flex items-center justify-between"
			data-testid="backend-error-banner"
		>
			<div class="flex-1 min-w-0">
				<p class="text-sm text-red-600 dark:text-red-400">
					<span class="font-semibold">Backend not running.</span>
					<span class="ml-2">Start with: <code class="bg-red-500/20 px-1 rounded text-xs">orch serve</code></span>
				</p>
			</div>
			<button
				type="button"
				onclick={handleRetry}
				class="ml-4 px-3 py-1 text-xs font-medium text-red-600 dark:text-red-400 border border-red-500/30 rounded hover:bg-red-500/10 transition-colors whitespace-nowrap"
				data-testid="retry-button"
			>
				Retry
			</button>
		</div>
	{/if}

	<!-- Header -->
	<div class="border-b border-border px-2 py-2">
		<div class="flex items-center gap-6">
			<GroupByDropdown
				mode={groupByMode}
				onChange={handleGroupByChange}
			/>
			<div class="flex items-center gap-4 text-sm text-muted-foreground ml-auto">
				{#if $daemon}
					<span class="truncate max-w-[40rem]">
						Daemon: {$daemon.running ? ($daemon.status || 'running') : 'stopped'}
						{#if $daemon.running}
							· {$daemon.capacity_used}/{$daemon.capacity_max} slots
							{#if $daemon.last_poll_ago}
								· last poll {$daemon.last_poll_ago}
							{/if}
							· {daemonQueueSummary($daemon)}
							{#if $daemon.verification && ($daemon.verification.completions_since_verification > 0 || $daemon.verification.is_paused)}
								· <span class:text-amber-400={$daemon.verification.is_paused}>{$daemon.verification.completions_since_verification} to review{#if $daemon.verification.is_paused} (paused){/if}</span>
							{/if}
						{/if}
					</span>
				{/if}
				{#if $workGraph}
					{#if readyToCompleteItems.length > 0}
						<span class="text-emerald-400">{readyToCompleteItems.length} ready to complete</span>
					{/if}
					{#if ($wipItems?.length ?? 0) > 0}
						<span class="text-blue-400">{$wipItems.length} wip</span>
					{/if}
					<span>{$workGraph.node_count} issues</span>
					<span>{$workGraph.edge_count} edges</span>
				{/if}
				{#if $orchestratorContext?.project_dir}
					<span class="truncate max-w-xs">
						{$orchestratorContext.project_dir.split('/').pop()}
					</span>
				{/if}
			</div>
		</div>
	</div>

	<!-- Focus Breadcrumb -->
	{#if $focus?.has_focus && $focus?.beads_id}
		<div class="bg-blue-500/10 border-b border-blue-500/20 px-4 py-2 flex items-center justify-between">
			<div class="flex items-center gap-2">
				<span class="text-blue-500">🎯</span>
				<span class="text-sm text-blue-600 dark:text-blue-400 font-medium">
					Focus: {$focus.beads_id}
				</span>
				{#if $focus.goal}
					<span class="text-sm text-blue-500/80">
						{$focus.goal}
					</span>
				{/if}
			</div>
			<button
				type="button"
				onclick={handleClearFocus}
				class="text-xs text-blue-500 hover:text-blue-600 hover:underline"
			>
				Clear Focus
			</button>
		</div>
	{/if}

	<!-- Content -->
	<div class="flex-1 min-h-0 overflow-hidden">
		{#if loading}
				<div class="flex items-center justify-center h-full">
					<div class="text-muted-foreground">Loading work graph...</div>
				</div>
			{:else if error}
				<div class="flex items-center justify-center h-full">
					<div class="text-red-500">Error: {error}</div>
				</div>
		{:else if filteredTree.length === 0 && readyToCompleteItems.length === 0 && ($wipItems?.length ?? 0) === 0}
			<div class="flex items-center justify-center h-full">
				<div class="text-muted-foreground">
					No open issues found
				</div>
			</div>
			{:else}
				<div class="h-full min-h-0 flex flex-col">
					{#if readyToCompleteItems.length > 0}
						<!-- Daemon Paused Banner -->
						{#if $daemon?.verification?.is_paused}
							<div
								class="mx-2 mt-2 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 flex items-center justify-between"
								data-testid="daemon-paused-banner"
							>
								<div class="flex items-center gap-2">
									<span class="text-amber-400 text-sm">⏸</span>
									<span class="text-sm text-amber-300">Daemon paused — {$daemon.verification.completions_since_verification} completions awaiting review</span>
								</div>
								<div class="flex items-center gap-2">
									<button
										type="button"
										onclick={acknowledgeAll}
										disabled={acknowledgingAll}
										class="px-2.5 py-1 text-xs font-medium text-amber-200 border border-amber-500/40 rounded hover:bg-amber-500/20 transition-colors disabled:opacity-50"
										data-testid="acknowledge-all-button"
									>
										{acknowledgingAll ? 'Closing...' : `Close All (${readyToCompleteItems.length})`}
									</button>
									<button
										type="button"
										onclick={resumeDaemon}
										class="px-2.5 py-1 text-xs font-medium text-emerald-300 border border-emerald-500/40 rounded hover:bg-emerald-500/20 transition-colors"
										data-testid="resume-daemon-button"
									>
										Resume
									</button>
								</div>
							</div>
						{/if}

						<div
							class="mx-2 {$daemon?.verification?.is_paused ? 'mt-1' : 'mt-2'} mb-2 rounded-md border border-emerald-500/30 bg-emerald-500/5"
							data-testid="ready-to-complete-section"
						>
							<div class="px-3 py-2 border-b border-emerald-500/20 flex items-center justify-between gap-4">
								<div class="text-sm font-semibold text-emerald-400">Ready to Complete</div>
								<div class="flex items-center gap-3">
									<span class="text-xs text-emerald-300/80">{readyToCompleteItems.length} awaiting review · oldest first</span>
									{#if !$daemon?.verification?.is_paused && readyToCompleteItems.length > 1}
										<button
											type="button"
											onclick={acknowledgeAll}
											disabled={acknowledgingAll}
											class="px-2 py-0.5 text-xs font-medium text-emerald-300 border border-emerald-500/30 rounded hover:bg-emerald-500/20 transition-colors disabled:opacity-50"
											data-testid="acknowledge-all-compact-button"
										>
											{acknowledgingAll ? 'Closing...' : 'Close All'}
										</button>
									{/if}
								</div>
							</div>
							<div class="max-h-48 overflow-y-auto">
								{#each readyToCompleteItems as item (item.id)}
									<div
										class="px-3 py-2 border-b border-emerald-500/10 last:border-b-0"
										data-testid={`ready-to-complete-row-${item.id}`}
									>
										<div class="flex items-center gap-3 text-xs">
											<span class="font-mono text-emerald-300 min-w-[120px]">{item.id}</span>
											<span class="text-foreground text-sm flex-1 truncate">{item.title}</span>
											<span class="text-muted-foreground whitespace-nowrap">{item.runtime || 'runtime unknown'}</span>
											<span class="text-muted-foreground whitespace-nowrap">{formatTokenTotal(item.tokenTotal)}</span>
											<span class="text-emerald-200/80 whitespace-nowrap">completed {formatRelativeTime(item.completionAt)}</span>
											<button
												type="button"
												onclick={() => acknowledgeItem(item.id)}
												disabled={acknowledging.has(item.id) || acknowledgingAll}
												class="px-2 py-0.5 text-xs font-medium text-emerald-300 border border-emerald-500/30 rounded hover:bg-emerald-500/20 transition-colors disabled:opacity-50 flex-shrink-0"
												data-testid={`acknowledge-button-${item.id}`}
											>
												{acknowledging.has(item.id) ? '...' : 'Close'}
											</button>
										</div>
										{#if item.tldr || item.deltaSummary}
											<div class="mt-1 ml-[132px] flex items-center gap-3 text-xs text-muted-foreground">
												{#if item.tldr}
													<span class="truncate max-w-[400px]" title={item.tldr}>{item.tldr}</span>
												{/if}
												{#if item.deltaSummary}
													<span class="text-emerald-400/60 whitespace-nowrap">{item.deltaSummary}</span>
												{/if}
											</div>
										{/if}
									</div>
								{/each}
							</div>
						</div>
					{/if}

					{#if filteredTree.length > 0}
						<div class="flex-1 min-h-0">
							<WorkGraphTree
								tree={filteredTree}
								groups={groupSections}
								groupMode={groupByMode}
								edges={$workGraph?.edges || []}
							excludeIds={readyToCompleteIds}
								wipItems={$wipItems}
								onToggleExpansion={handleToggleExpansion}
								onSetFocus={handleSetFocus}
							/>
						</div>
					{/if}
				</div>
			{/if}
	</div>

	<!-- Keyboard Shortcuts Footer -->
	<div class="h-9 px-2 flex items-center justify-center border-t border-zinc-800 bg-zinc-950 text-zinc-500 text-[11px] font-mono">
		<span class="tracking-wide">
			<span class="text-zinc-400">j/k</span> navigate
			<span class="mx-3">·</span>
			<span class="text-zinc-400">h/l</span> collapse/expand
			<span class="mx-3">·</span>
			<span class="text-zinc-400">enter</span> details
			<span class="mx-3">·</span>
			<span class="text-zinc-400">i</span> side panel
			<span class="mx-3">·</span>
			<span class="text-zinc-400">v</span> verify
			<span class="mx-3">·</span>
			<span class="text-zinc-400">x</span> close
			<span class="mx-3">·</span>
			<span class="text-zinc-400">c</span> copy ID
			<span class="mx-3">·</span>
			<span class="text-zinc-400">t/w</span> WIP↔tree
			<span class="mx-3">·</span>
			<span class="text-zinc-400">G</span> cycle groups
		</span>
	</div>
</div>
