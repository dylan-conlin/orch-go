<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { AgentCard } from '$lib/components/agent-card';
	import { AgentDetailPanel } from '$lib/components/agent-detail';
	import { CollapsibleSection } from '$lib/components/collapsible-section';
	import { ReadyQueueSection } from '$lib/components/ready-queue-section';
	// Note: PendingReviewsSection, UpNextSection, RecentWins removed - consolidated into NeedsAttention
	import { NeedsAttention } from '$lib/components/needs-attention';
	import {
		agents,
		activeAgents,
		recentAgents,
		archivedAgents,
		completedAgents,
		abandonedAgents,
		sseEvents,
		connectionStatus,
		connectSSE,
		disconnectSSE,
		type Agent,
		type AgentState
	} from '$lib/stores/agents';
	import {
		agentlogEvents,
		agentlogConnectionStatus,
		connectAgentlogSSE,
		disconnectAgentlogSSE,
		errorEvents
	} from '$lib/stores/agentlog';
	import { usage } from '$lib/stores/usage';
	import { focus, getDriftEmoji } from '$lib/stores/focus';
	import { servers } from '$lib/stores/servers';
	import { beads, readyIssues } from '$lib/stores/beads';
	import { daemon, getDaemonEmoji, getDaemonCapacity } from '$lib/stores/daemon';
	import { pendingReviews } from '$lib/stores/pending-reviews';
	// Note: dashboardMode store removed - single unified view now
	import { config } from '$lib/stores/config';
	import { patterns } from '$lib/stores/patterns';
	import { SettingsPanel } from '$lib/components/settings-panel';

	// Filter and sort state
	let statusFilter: AgentState | 'all' = 'all';
	let skillFilter: string = 'all';
	let projectFilter: string = 'all';
	let sortBy: 'recent-activity' | 'newest' | 'oldest' | 'alphabetical' | 'project' | 'phase' = 'recent-activity';
	let activeOnly: boolean = false;
	let searchQuery: string = '';

	// Section collapse state with localStorage persistence
	const STORAGE_KEY = 'orch-dashboard-sections';
	let sectionState = {
		active: true,   // Active always expanded by default
		recent: false,  // Recent collapsed by default
		archive: false, // Archive collapsed by default
		upNext: false,  // Up Next collapsed by default (auto-expands on P0/P1)
		readyQueue: false, // Ready queue collapsed by default
		pendingReviews: true, // Pending reviews expanded by default (actionable)
		sseStream: false // SSE Stream collapsed by default (low signal-to-noise for most users)
	};

	// Load section state from localStorage on mount
	function loadSectionState() {
		if (typeof window === 'undefined') return;
		try {
			const stored = localStorage.getItem(STORAGE_KEY);
			if (stored) {
				const parsed = JSON.parse(stored);
				sectionState = { ...sectionState, ...parsed };
			}
		} catch (e) {
			console.warn('Failed to load section state:', e);
		}
	}

	// Save section state to localStorage
	function saveSectionState() {
		if (typeof window === 'undefined') return;
		try {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(sectionState));
		} catch (e) {
			console.warn('Failed to save section state:', e);
		}
	}

	// Reactive saving when state changes
	$: if (typeof window !== 'undefined') {
		saveSectionState();
	}

	// Get unique skills from agents
	$: uniqueSkills = [...new Set($agents.map(a => a.skill).filter(Boolean))] as string[];

	// Get unique projects from agents
	$: uniqueProjects = [...new Set($agents.map(a => a.project).filter(Boolean))].sort() as string[];

	onMount(() => {
		// Load section state from localStorage (sync, instant)
		loadSectionState();

		// Connect to primary SSE immediately - this triggers agents.fetch() on connection
		// which is the most critical data for initial render
		connectSSE();

		// Fetch critical data in parallel using Promise.all
		// These affect the primary dashboard view and should load ASAP
		Promise.all([
			beads.fetch(),
			pendingReviews.fetch(),
			config.fetch()
		]).catch(console.error);

		// Defer secondary data fetches using requestIdleCallback or setTimeout fallback
		// These are "nice to have" data that can load after initial render
		const deferSecondaryFetches = () => {
			usage.fetch();
			focus.fetch();
			servers.fetch();
			readyIssues.fetch();
			daemon.fetch();
			patterns.fetch();
		};

		// Use requestIdleCallback for better performance, with setTimeout fallback
		if ('requestIdleCallback' in window) {
			requestIdleCallback(deferSecondaryFetches, { timeout: 2000 });
		} else {
			setTimeout(deferSecondaryFetches, 100);
		}

		// Defer agentlog SSE connection - it's for the event log panel, not critical
		const connectSecondarySSE = () => {
			connectAgentlogSSE();
		};
		if ('requestIdleCallback' in window) {
			requestIdleCallback(connectSecondarySSE, { timeout: 3000 });
		} else {
			setTimeout(connectSecondarySSE, 500);
		}

		// Refresh all data every 60 seconds (all fetches are already non-blocking)
		const refreshInterval = setInterval(() => {
			Promise.all([
				usage.fetch(),
				focus.fetch(),
				servers.fetch(),
				beads.fetch(),
				readyIssues.fetch(),
				daemon.fetch(),
				pendingReviews.fetch(),
				patterns.fetch()
			]).catch(console.error);
		}, 60000);

		// Clean up connections before page unload to avoid Firefox network errors
		const handleBeforeUnload = () => {
			disconnectSSE();
			disconnectAgentlogSSE();
		};
		window.addEventListener('beforeunload', handleBeforeUnload);

		return () => {
			window.removeEventListener('beforeunload', handleBeforeUnload);
			clearInterval(refreshInterval);
		};
	});

	onDestroy(() => {
		disconnectSSE();
		disconnectAgentlogSSE();
	});

	function handleConnectClick() {
		if ($connectionStatus === 'disconnected') {
			connectSSE();
		} else {
			disconnectSSE();
		}
	}

	function formatTime(timestamp?: number): string {
		if (!timestamp) return '';
		return new Date(timestamp).toLocaleTimeString();
	}

	function formatUnixTime(timestamp: number): string {
		return new Date(timestamp * 1000).toLocaleTimeString();
	}

	function getEventIcon(type: string): string {
		switch (type) {
			case 'session.spawned':
				return '🚀';
			case 'session.completed':
				return '✅';
			case 'session.error':
				return '❌';
			case 'session.status':
				return '📊';
			default:
				return '📝';
		}
	}

	function getEventLabel(type: string): string {
		switch (type) {
			case 'session.spawned':
				return 'Spawned';
			case 'session.completed':
				return 'Completed';
			case 'session.error':
				return 'Error';
			case 'session.status':
				return 'Status';
			default:
				return type;
		}
	}

	function handleAgentlogConnectClick() {
		if ($agentlogConnectionStatus === 'disconnected') {
			connectAgentlogSSE();
		} else {
			disconnectAgentlogSSE();
		}
	}

	function clearFilters() {
		statusFilter = 'all';
		skillFilter = 'all';
		projectFilter = 'all';
		sortBy = 'recent-activity';
		activeOnly = false;
		searchQuery = '';
	}

	$: hasActiveFilters = statusFilter !== 'all' || skillFilter !== 'all' || projectFilter !== 'all' || sortBy !== 'recent-activity' || activeOnly || searchQuery !== '';

	// Helper function to apply sorting to agent arrays
	// useStableSort: when true, uses spawned_at (immutable) instead of updated_at (volatile) 
	// to prevent constant reordering of active agents as they receive SSE updates
	// IMPORTANT: When useStableSort is true, we skip is_processing comparison to prevent
	// grid jostling when multiple agents toggle between busy/idle states rapidly
	function sortAgents(agentList: Agent[], useStableSort: boolean = false): Agent[] {
		return [...agentList].sort((a, b) => {
			switch (sortBy) {
				case 'recent-activity':
					// Only use is_processing for sort tiebreaker in non-stable sort mode
					// In stable sort mode, is_processing toggles rapidly via SSE causing grid jostling
					// The visual indicator (gold border) still shows processing state per-card
					if (!useStableSort && a.is_processing !== b.is_processing) {
						return a.is_processing ? -1 : 1;
					}
					// For stable sort (active agents), use spawned_at to maintain grid positions
					// For volatile sort (recent/archive), use updated_at for recency ordering
					if (useStableSort) {
						const bSpawned = b.spawned_at ? new Date(b.spawned_at).getTime() : 0;
						const aSpawned = a.spawned_at ? new Date(a.spawned_at).getTime() : 0;
						return bSpawned - aSpawned;
					}
					const bUpdated = b.updated_at ? new Date(b.updated_at).getTime() : 0;
					const aUpdated = a.updated_at ? new Date(a.updated_at).getTime() : 0;
					return bUpdated - aUpdated;
				case 'newest':
					const bTime = b.spawned_at ? new Date(b.spawned_at).getTime() : 0;
					const aTime = a.spawned_at ? new Date(a.spawned_at).getTime() : 0;
					return bTime - aTime;
				case 'oldest':
					const aTimeOld = a.spawned_at ? new Date(a.spawned_at).getTime() : 0;
					const bTimeOld = b.spawned_at ? new Date(b.spawned_at).getTime() : 0;
					return aTimeOld - bTimeOld;
				case 'alphabetical':
					return a.id.localeCompare(b.id);
				case 'project':
					const projectA = a.project || 'zzz';
					const projectB = b.project || 'zzz';
					if (projectA !== projectB) {
						return projectA.localeCompare(projectB);
					}
					// Same stable sort logic for project grouping
					if (useStableSort) {
						const bProjSpawned = b.spawned_at ? new Date(b.spawned_at).getTime() : 0;
						const aProjSpawned = a.spawned_at ? new Date(a.spawned_at).getTime() : 0;
						return bProjSpawned - aProjSpawned;
					}
					const bProjUpdated = b.updated_at ? new Date(b.updated_at).getTime() : 0;
					const aProjUpdated = a.updated_at ? new Date(a.updated_at).getTime() : 0;
					return bProjUpdated - aProjUpdated;
				case 'phase':
					const phaseOrder: Record<string, number> = {
						'Implementing': 1,
						'Implementation': 1,
						'Planning': 2,
						'Validating': 3,
						'Complete': 4,
					};
					const phaseA = phaseOrder[a.phase || ''] || 5;
					const phaseB = phaseOrder[b.phase || ''] || 5;
					if (phaseA !== phaseB) {
						return phaseA - phaseB;
					}
					// Same stable sort logic for phase grouping
					if (useStableSort) {
						const bPhaseSpawned = b.spawned_at ? new Date(b.spawned_at).getTime() : 0;
						const aPhaseSpawned = a.spawned_at ? new Date(a.spawned_at).getTime() : 0;
						return bPhaseSpawned - aPhaseSpawned;
					}
					const bPhaseUpdated = b.updated_at ? new Date(b.updated_at).getTime() : 0;
					const aPhaseUpdated = a.updated_at ? new Date(a.updated_at).getTime() : 0;
					return bPhaseUpdated - aPhaseUpdated;
				default:
					return 0;
			}
		});
	}

	// Apply skill filter to any agent list
	function applySkillFilter(agentList: Agent[]): Agent[] {
		if (skillFilter === 'all') return agentList;
		return agentList.filter(a => a.skill === skillFilter);
	}

	// Apply project filter to any agent list
	function applyProjectFilter(agentList: Agent[]): Agent[] {
		if (projectFilter === 'all') return agentList;
		return agentList.filter(a => a.project === projectFilter);
	}

	// Apply search filter - searches workspace name (id), beads_id, task, beads_title, skill
	function applySearchFilter(agentList: Agent[]): Agent[] {
		if (!searchQuery.trim()) return agentList;
		const query = searchQuery.toLowerCase().trim();
		return agentList.filter(a => {
			// Search across multiple fields
			const searchableFields = [
				a.id,              // workspace name
				a.beads_id,        // beads issue ID
				a.task,            // task description
				a.beads_title,     // beads issue title
				a.skill,           // skill type
				a.project          // project name
			];
			return searchableFields.some(field => 
				field && field.toLowerCase().includes(query)
			);
		});
	}

	// Apply all filters (skill + project + search)
	function applyFilters(agentList: Agent[]): Agent[] {
		return applySearchFilter(applyProjectFilter(applySkillFilter(agentList)));
	}

	// Progressive disclosure: sorted and filtered agents per section
	// Active and Recent use stable sort (spawned_at) to prevent jostling from SSE updates
	// Archive uses volatile sort (updated_at) since historical recency matters more there
	$: sortedActiveAgents = sortAgents(applyFilters($activeAgents), true);
	$: sortedRecentAgents = sortAgents(applyFilters($recentAgents), true);
	$: sortedArchivedAgents = sortAgents(applyFilters($archivedAgents), false);

	// Total visible agents across all sections (for filter count)
	$: totalVisibleAgents = sortedActiveAgents.length + sortedRecentAgents.length + sortedArchivedAgents.length;
</script>

<div class="space-y-4">
	<!-- Compact Stats Bar -->
	<div class="flex flex-wrap items-center gap-x-5 gap-y-2 rounded-xl border bg-card/80 backdrop-blur-sm px-4 py-2.5 shadow-sm" data-testid="stats-bar">
		<!-- Primary indicators group -->
		<div class="flex items-center gap-5">
			<!-- Errors indicator -->
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<span {...props} class="inline-flex items-center gap-2 cursor-default rounded-lg px-2 py-1 transition-colors hover:bg-accent/30">
							<span class="text-lg">❌</span>
							<span class="inline-flex items-baseline gap-1">
								<span class="text-xl font-bold tabular-nums" class:text-red-500={$errorEvents.length > 0}>{$errorEvents.length}</span>
								<span class="text-xs text-muted-foreground">errors</span>
							</span>
						</span>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p>{$errorEvents.length === 0 ? 'No errors logged' : `${$errorEvents.length} agent error${$errorEvents.length === 1 ? '' : 's'} logged`}</p>
				</Tooltip.Content>
			</Tooltip.Root>

			<!-- Active agents indicator -->
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<span {...props} class="inline-flex items-center gap-2 cursor-default rounded-lg px-2 py-1 transition-colors hover:bg-accent/30">
							<span class="text-lg">🟢</span>
							<span class="inline-flex items-baseline gap-1">
								<span class="text-xl font-bold tabular-nums" class:text-green-500={$activeAgents.length > 0}>{$activeAgents.length}</span>
								<span class="text-xs text-muted-foreground">active</span>
							</span>
						</span>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p>{$activeAgents.length === 0 ? 'No active agents' : `${$activeAgents.length} agent${$activeAgents.length === 1 ? '' : 's'} running`}</p>
				</Tooltip.Content>
			</Tooltip.Root>

			<!-- Focus indicator (only when drifting - attention signal) -->
			{#if $focus?.has_focus && $focus.is_drifting}
				<Tooltip.Root>
					<Tooltip.Trigger>
						{#snippet child({ props })}
							<span {...props} class="inline-flex items-center gap-2 cursor-default" data-testid="focus-indicator">
								<span class="text-lg">{getDriftEmoji($focus)}</span>
								<span class="inline-flex items-baseline gap-1">
									<span class="text-xs truncate max-w-32" class:text-red-500={$focus.is_drifting} class:text-green-500={!$focus.is_drifting}>
										{$focus.is_drifting ? 'drifting' : 'focused'}
									</span>
								</span>
							</span>
						{/snippet}
					</Tooltip.Trigger>
					<Tooltip.Content>
						<p class="font-medium">{$focus.goal || 'Focus set'}</p>
						{#if $focus.is_drifting}
							<p class="text-xs text-muted-foreground mt-1">Current work may not align with focus</p>
						{/if}
					</Tooltip.Content>
				</Tooltip.Root>
			{/if}

			<!-- Servers indicator -->
			{#if $servers && $servers.running_count > 0}
				<Tooltip.Root>
					<Tooltip.Trigger>
						{#snippet child({ props })}
							<span {...props} class="inline-flex items-center gap-2 cursor-default" data-testid="servers-indicator">
								<span class="text-lg">{$servers.running_count > 0 ? '🖥️' : '💤'}</span>
								<span class="inline-flex items-baseline gap-1">
									<span class="text-xl font-bold" class:text-green-500={$servers.running_count > 0}>{$servers.running_count}</span>
									<span class="text-xs text-muted-foreground">/{$servers.total_count} servers</span>
								</span>
							</span>
						{/snippet}
					</Tooltip.Trigger>
					<Tooltip.Content>
						<p>{$servers.running_count} running, {$servers.stopped_count} stopped</p>
						<p class="text-xs text-muted-foreground mt-1">Local development servers</p>
					</Tooltip.Content>
				</Tooltip.Root>
			{/if}

			<!-- Beads indicator (clickable to toggle ready queue section) -->
			{#if $beads}
				<Tooltip.Root>
					<Tooltip.Trigger>
						{#snippet child({ props })}
							<button
								{...props}
								class="inline-flex items-center gap-2 cursor-pointer hover:bg-accent/50 rounded px-1 -mx-1 transition-colors"
								onclick={() => { sectionState.readyQueue = !sectionState.readyQueue; }}
								data-testid="beads-indicator"
							>
								<span class="text-lg">📋</span>
								<span class="inline-flex items-baseline gap-1">
									<span class="text-xl font-bold" class:text-green-500={$beads.ready_issues > 0}>{$beads.ready_issues}</span>
									<span class="text-xs text-muted-foreground">ready</span>
								</span>
								{#if $beads.blocked_issues > 0}
									<span class="text-xs text-red-500">({$beads.blocked_issues} blocked)</span>
								{/if}
							</button>
						{/snippet}
					</Tooltip.Trigger>
					<Tooltip.Content>
						<p>{$beads.ready_issues} ready to work on</p>
						<p class="text-xs text-muted-foreground">{$beads.blocked_issues} blocked • {$beads.open_issues} total open</p>
						<p class="text-xs text-muted-foreground mt-1">Click to {sectionState.readyQueue ? 'collapse' : 'expand'} queue</p>
					</Tooltip.Content>
				</Tooltip.Root>
			{/if}

			<!-- Daemon indicator -->
			{#if $daemon}
				<Tooltip.Root>
					<Tooltip.Trigger>
						{#snippet child({ props })}
							<span {...props} class="inline-flex items-center gap-2 cursor-default" data-testid="daemon-indicator">
								<span class="text-lg">{getDaemonEmoji($daemon)}</span>
								<span class="inline-flex items-baseline gap-1">
									{#if $daemon.running}
										<span class="text-xl font-bold" class:text-green-500={$daemon.capacity_free > 0} class:text-red-500={$daemon.capacity_free === 0}>{getDaemonCapacity($daemon)}</span>
										<span class="text-xs text-muted-foreground">slots</span>
									{:else}
										<span class="text-xs text-muted-foreground">daemon</span>
									{/if}
								</span>
							</span>
						{/snippet}
					</Tooltip.Trigger>
					<Tooltip.Content>
						{#if $daemon.running}
							<p class="font-medium">Daemon {$daemon.status}</p>
							<p class="text-xs text-muted-foreground">
								{$daemon.capacity_used}/{$daemon.capacity_max} agents • {$daemon.ready_count} ready
							</p>
							{#if $daemon.last_poll_ago}
								<p class="text-xs text-muted-foreground">Last poll: {$daemon.last_poll_ago}</p>
							{/if}
							{#if $daemon.last_spawn_ago}
								<p class="text-xs text-muted-foreground">Last spawn: {$daemon.last_spawn_ago}</p>
							{/if}
						{:else}
							<p>Daemon not running</p>
							<p class="text-xs text-muted-foreground">Start with: orch daemon run</p>
						{/if}
					</Tooltip.Content>
				</Tooltip.Root>
			{/if}
		</div>
		<!-- Connection button and settings - pushed to end -->
		<div class="ml-auto flex items-center gap-1">
			<SettingsPanel />
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<Button
							{...props}
							variant={$connectionStatus === 'connected' ? 'destructive' : 'outline'}
							size="sm"
							onclick={handleConnectClick}
							class="h-7 text-xs"
						>
							{#if $connectionStatus === 'connecting'}
								...
							{:else if $connectionStatus === 'connected'}
								Disconnect
							{:else}
								Connect
							{/if}
						</Button>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>
					{#if $connectionStatus === 'connected'}
						<p>Disconnect from SSE stream</p>
						<p class="text-xs text-muted-foreground">Stop receiving real-time agent updates</p>
					{:else if $connectionStatus === 'connecting'}
						<p>Connecting to SSE stream...</p>
					{:else}
						<p>Connect to SSE stream</p>
						<p class="text-xs text-muted-foreground">Receive real-time agent updates</p>
					{/if}
				</Tooltip.Content>
			</Tooltip.Root>
		</div>
	</div>

	<!-- ATTENTION-FIRST LAYOUT: Single unified view -->
	
	<!-- 🔔 Attention Panel (PRIMARY - top, prominent) -->
	<NeedsAttention />
	
	<!-- 🟢 Active Agents (always visible) -->
	<div class="rounded-lg border bg-card border-green-500/30" data-testid="active-agents-section">
		<div class="flex items-center gap-2 px-3 py-2 border-b">
			<span class="text-sm">🟢</span>
			<span class="text-sm font-medium">Active Agents</span>
			<Badge variant="default" class="h-5 px-1.5 text-xs">
				{sortedActiveAgents.length}
			</Badge>
		</div>
		<div class="p-2">
			{#if sortedActiveAgents.length > 0}
				<div class="grid gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
					{#each sortedActiveAgents as agent, i (`${agent.id}-${agent.session_id ?? i}`)}
						<AgentCard {agent} />
					{/each}
				</div>
			{:else}
				<div class="rounded-lg border-2 border-dashed border-muted-foreground/20 p-8 text-center">
					<div class="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-full bg-muted/50">
						<span class="text-2xl opacity-50">🤖</span>
					</div>
					<p class="text-sm font-medium text-muted-foreground">No active agents</p>
					<p class="mt-2 text-xs text-muted-foreground">
						Spawn with <code class="rounded-md bg-muted px-2 py-0.5 font-mono text-xs">orch spawn</code>
					</p>
				</div>
			{/if}
		</div>
	</div>

	<!-- 📋 Ready Queue (collapsed by default, secondary) -->
	<ReadyQueueSection
		bind:expanded={sectionState.readyQueue}
	/>

	<!-- 📦 Recent & Archive (collapsed by default, secondary) -->
	{#if sortedRecentAgents.length > 0 || sortedArchivedAgents.length > 0}
		<div class="space-y-2">
			<!-- Recent Section (idle/completed within 24h) -->
			{#if sortedRecentAgents.length > 0}
				<CollapsibleSection
					title="Recent"
					icon="🕐"
					agents={sortedRecentAgents}
					bind:expanded={sectionState.recent}
					variant="recent"
				>
					<div class="grid gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
						{#each sortedRecentAgents as agent, i (`${agent.id}-${agent.session_id ?? i}`)}
							<AgentCard {agent} />
						{/each}
					</div>
				</CollapsibleSection>
			{/if}

			<!-- Archive Section (older than 24h) -->
			{#if sortedArchivedAgents.length > 0}
				<CollapsibleSection
					title="Archive"
					icon="📦"
					agents={sortedArchivedAgents}
					bind:expanded={sectionState.archive}
					variant="archive"
				>
					<div class="grid gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
						{#each sortedArchivedAgents as agent, i (`${agent.id}-${agent.session_id ?? i}`)}
							<AgentCard {agent} />
						{/each}
					</div>
				</CollapsibleSection>
			{/if}
		</div>
	{/if}
</div>

<!-- Agent Detail Slide-out Panel -->
<AgentDetailPanel />
