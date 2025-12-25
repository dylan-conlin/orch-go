<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { AgentCard } from '$lib/components/agent-card';
	import { AgentDetailPanel } from '$lib/components/agent-detail';
	import { CollapsibleSection } from '$lib/components/collapsible-section';
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
	import { usage, getUsageColor, getUsageEmoji } from '$lib/stores/usage';
	import { focus, getDriftEmoji } from '$lib/stores/focus';
	import { servers } from '$lib/stores/servers';
	import { beads } from '$lib/stores/beads';

	// Filter and sort state
	let statusFilter: AgentState | 'all' = 'all';
	let skillFilter: string = 'all';
	let projectFilter: string = 'all';
	let sortBy: 'recent-activity' | 'newest' | 'oldest' | 'alphabetical' | 'project' | 'phase' = 'recent-activity';
	let activeOnly: boolean = false;

	// Section collapse state with localStorage persistence
	const STORAGE_KEY = 'orch-dashboard-sections';
	let sectionState = {
		active: true,   // Active always expanded by default
		recent: false,  // Recent collapsed by default
		archive: false  // Archive collapsed by default
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
		// Load section state from localStorage
		loadSectionState();

		// Connect to SSE - this will trigger initial fetch when connection opens
		// Removes race condition from parallel fetch + SSE connect
		connectSSE();
		connectAgentlogSSE();

		// Fetch usage, focus, servers, and beads data
		usage.fetch();
		focus.fetch();
		servers.fetch();
		beads.fetch();

		// Refresh usage, focus, servers, and beads every 60 seconds
		const usageInterval = setInterval(() => {
			usage.fetch();
			focus.fetch();
			servers.fetch();
			beads.fetch();
		}, 60000);

		// Clean up connections before page unload to avoid Firefox network errors
		const handleBeforeUnload = () => {
			disconnectSSE();
			disconnectAgentlogSSE();
		};
		window.addEventListener('beforeunload', handleBeforeUnload);

		return () => {
			window.removeEventListener('beforeunload', handleBeforeUnload);
			clearInterval(usageInterval);
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
	}

	$: hasActiveFilters = statusFilter !== 'all' || skillFilter !== 'all' || projectFilter !== 'all' || sortBy !== 'recent-activity' || activeOnly;

	// Helper function to apply sorting to agent arrays
	// useStableSort: when true, uses spawned_at (immutable) instead of updated_at (volatile) 
	// to prevent constant reordering of active agents as they receive SSE updates
	function sortAgents(agentList: Agent[], useStableSort: boolean = false): Agent[] {
		return [...agentList].sort((a, b) => {
			switch (sortBy) {
				case 'recent-activity':
					if (a.is_processing !== b.is_processing) {
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

	// Apply all filters (skill + project)
	function applyFilters(agentList: Agent[]): Agent[] {
		return applyProjectFilter(applySkillFilter(agentList));
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

<div class="space-y-3">
	<!-- Compact Stats Bar -->
	<div class="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-lg border bg-card px-4 py-2" data-testid="stats-bar">
		<!-- Usage progress bars -->
		<div class="flex items-center gap-4">
			{#if $usage}
				<!-- Hourly usage -->
				<div class="flex items-center gap-2" title="5-hour session limit{$usage.five_hour_reset ? ` • Resets in ${$usage.five_hour_reset}` : ''}">
					<span class="text-sm text-muted-foreground">Hourly</span>
					<div class="relative h-2 w-20 overflow-hidden rounded-full bg-muted">
						<div 
							class="h-full transition-all duration-300" 
							class:bg-green-500={$usage.five_hour_percent < 60}
							class:bg-yellow-500={$usage.five_hour_percent >= 60 && $usage.five_hour_percent < 80}
							class:bg-red-500={$usage.five_hour_percent >= 80}
							style="width: {Math.min($usage.five_hour_percent, 100)}%"
						></div>
					</div>
					<span class="text-xs font-medium" class:text-green-500={$usage.five_hour_percent < 60} class:text-yellow-500={$usage.five_hour_percent >= 60 && $usage.five_hour_percent < 80} class:text-red-500={$usage.five_hour_percent >= 80}>
						{Math.round($usage.five_hour_percent)}%
					</span>
					{#if $usage.five_hour_reset}
						<span class="text-xs text-muted-foreground">({$usage.five_hour_reset})</span>
					{/if}
				</div>
				<div class="h-4 w-px bg-border"></div>
				<!-- Weekly usage -->
				<div class="flex items-center gap-2" title="7-day weekly limit{$usage.weekly_reset ? ` • Resets in ${$usage.weekly_reset}` : ''}">
					<span class="text-sm text-muted-foreground">Weekly</span>
					<div class="relative h-2 w-20 overflow-hidden rounded-full bg-muted">
						<div 
							class="h-full transition-all duration-300" 
							class:bg-green-500={$usage.weekly_percent < 60}
							class:bg-yellow-500={$usage.weekly_percent >= 60 && $usage.weekly_percent < 80}
							class:bg-red-500={$usage.weekly_percent >= 80}
							style="width: {Math.min($usage.weekly_percent, 100)}%"
						></div>
					</div>
					<span class="text-xs font-medium" class:text-green-500={$usage.weekly_percent < 60} class:text-yellow-500={$usage.weekly_percent >= 60 && $usage.weekly_percent < 80} class:text-red-500={$usage.weekly_percent >= 80}>
						{Math.round($usage.weekly_percent)}%
					</span>
					{#if $usage.weekly_reset}
						<span class="text-xs text-muted-foreground">({$usage.weekly_reset})</span>
					{/if}
				</div>
			{:else}
				<span class="text-xs text-muted-foreground">Loading usage...</span>
			{/if}
			<div class="h-4 w-px bg-border"></div>
			<div class="flex items-center gap-2">
				<span class="text-lg">❌</span>
				<div class="flex items-baseline gap-1">
					<span class="text-xl font-bold" class:text-red-500={$errorEvents.length > 0}>{$errorEvents.length}</span>
					<span class="text-xs text-muted-foreground">errors</span>
				</div>
			</div>
		</div>
		<!-- Secondary indicators group -->
		<div class="flex items-center gap-4">
			{#if $focus?.has_focus}
				<div class="flex items-center gap-2" data-testid="focus-indicator" title={$focus.goal || 'Focus set'}>
					<span class="text-lg">{getDriftEmoji($focus)}</span>
					<div class="flex items-baseline gap-1">
						<span class="text-xs truncate max-w-32" class:text-red-500={$focus.is_drifting} class:text-green-500={!$focus.is_drifting}>
							{$focus.is_drifting ? 'drifting' : 'focused'}
						</span>
					</div>
				</div>
			{/if}
			{#if $servers}
				<div class="flex items-center gap-2" data-testid="servers-indicator" title="{$servers.running_count} running, {$servers.stopped_count} stopped">
					<span class="text-lg">{$servers.running_count > 0 ? '🖥️' : '💤'}</span>
					<div class="flex items-baseline gap-1">
						<span class="text-xl font-bold" class:text-green-500={$servers.running_count > 0}>{$servers.running_count}</span>
						<span class="text-xs text-muted-foreground">/{$servers.total_count} servers</span>
					</div>
				</div>
			{/if}
			{#if $beads}
				<div class="flex items-center gap-2" data-testid="beads-indicator" title="{$beads.ready_issues} ready, {$beads.blocked_issues} blocked, {$beads.open_issues} open">
					<span class="text-lg">📋</span>
					<div class="flex items-baseline gap-1">
						<span class="text-xl font-bold" class:text-green-500={$beads.ready_issues > 0}>{$beads.ready_issues}</span>
						<span class="text-xs text-muted-foreground">ready</span>
					</div>
					{#if $beads.blocked_issues > 0}
						<span class="text-xs text-red-500">({$beads.blocked_issues} blocked)</span>
					{/if}
				</div>
			{/if}
		</div>
		<!-- Connection button - pushed to end -->
		<div class="ml-auto flex items-center gap-2">
			<Button
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
		</div>
	</div>

	<!-- Swarm Map (Primary Focus) -->
	<div class="rounded-lg border bg-card">
		<div class="flex items-center justify-between border-b px-3 py-2">
			<div class="flex items-center gap-2">
				<h2 class="text-sm font-semibold">Swarm Map</h2>
				<span class="text-xs text-muted-foreground">Real-time agent activity</span>
			</div>
		</div>
		<div class="p-2">
			<!-- Compact Filter Bar -->
			<div class="mb-2 flex flex-wrap items-center gap-2 text-xs" data-testid="filter-bar">
				<label class="flex items-center gap-1 cursor-pointer">
					<input
						type="checkbox"
						bind:checked={activeOnly}
						class="h-3 w-3 rounded border-input"
						data-testid="active-only-toggle"
					/>
					<span class="text-xs">Active Only</span>
				</label>

				<div class="h-4 w-px bg-border"></div>

				<select
					id="status-filter"
					bind:value={statusFilter}
					class="h-6 rounded border border-input bg-background px-1.5 text-xs"
					data-testid="status-filter"
					disabled={activeOnly}
				>
					<option value="all">All status</option>
					<option value="active">Active</option>
					<option value="idle">Idle</option>
					<option value="completed">Completed</option>
					<option value="abandoned">Abandoned</option>
				</select>

				{#if uniqueSkills.length > 0}
					<select
						id="skill-filter"
						bind:value={skillFilter}
						class="h-6 rounded border border-input bg-background px-1.5 text-xs"
						data-testid="skill-filter"
					>
						<option value="all">All skills</option>
						{#each uniqueSkills as skill}
							<option value={skill}>{skill}</option>
						{/each}
					</select>
				{/if}

				{#if uniqueProjects.length > 0}
					<select
						id="project-filter"
						bind:value={projectFilter}
						class="h-6 rounded border border-input bg-background px-1.5 text-xs"
						data-testid="project-filter"
					>
						<option value="all">All projects</option>
						{#each uniqueProjects as project}
							<option value={project}>{project}</option>
						{/each}
					</select>
				{/if}

				<select
					id="sort-by"
					bind:value={sortBy}
					class="h-6 rounded border border-input bg-background px-1.5 text-xs"
					data-testid="sort-select"
				>
					<option value="recent-activity">Recent Activity</option>
					<option value="newest">Newest Spawned</option>
					<option value="oldest">Oldest Spawned</option>
					<option value="project">By Project</option>
					<option value="phase">By Phase</option>
					<option value="alphabetical">A-Z</option>
				</select>

				{#if hasActiveFilters}
					<button onclick={clearFilters} class="text-xs text-muted-foreground hover:text-foreground">
						Clear
					</button>
				{/if}

				<span class="ml-auto text-muted-foreground" data-testid="filter-count">
					{totalVisibleAgents} agent{totalVisibleAgents === 1 ? '' : 's'}
				</span>
			</div>

			<!-- Progressive Disclosure: Collapsible Sections -->
			<div class="space-y-2" data-testid="agent-sections">
				{#if activeOnly}
					<!-- Active Only mode: show flat grid of active agents -->
					<div class="grid gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5" data-testid="agent-grid">
						{#each sortedActiveAgents as agent (agent.id)}
							<AgentCard {agent} />
						{:else}
							<div class="col-span-full rounded border border-dashed p-6 text-center">
								<p class="text-sm text-muted-foreground">No active agents</p>
								<p class="mt-1 text-xs text-muted-foreground">
									Spawn with <code class="rounded bg-muted px-1">orch spawn</code>
								</p>
							</div>
						{/each}
					</div>
				{:else}
					<!-- Progressive disclosure mode: collapsible sections -->
					<!-- Active Section (always visible when agents exist) -->
					{#if sortedActiveAgents.length > 0 || (sortedRecentAgents.length === 0 && sortedArchivedAgents.length === 0)}
						<CollapsibleSection
							title="Active"
							icon="🟢"
							agents={sortedActiveAgents}
							bind:expanded={sectionState.active}
							variant="active"
						>
							<div class="grid gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
								{#each sortedActiveAgents as agent (agent.id)}
									<AgentCard {agent} />
								{/each}
							</div>
						</CollapsibleSection>
					{/if}

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
								{#each sortedRecentAgents as agent (agent.id)}
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
								{#each sortedArchivedAgents as agent (agent.id)}
									<AgentCard {agent} />
								{/each}
							</div>
						</CollapsibleSection>
					{/if}

					<!-- Empty state when no agents at all -->
					{#if sortedActiveAgents.length === 0 && sortedRecentAgents.length === 0 && sortedArchivedAgents.length === 0}
						<div class="rounded border border-dashed p-6 text-center">
							{#if hasActiveFilters}
								<p class="text-sm text-muted-foreground">No agents match filters</p>
								<Button variant="link" onclick={clearFilters} class="mt-1 h-auto p-0 text-xs">
									Clear filters
								</Button>
							{:else}
								<p class="text-sm text-muted-foreground">No agents in the swarm</p>
								<p class="mt-1 text-xs text-muted-foreground">
									Spawn with <code class="rounded bg-muted px-1">orch spawn</code>
								</p>
							{/if}
						</div>
					{/if}
				{/if}
			</div>
		</div>
	</div>

	<!-- Event Panels (side by side on larger screens) -->
	<div class="grid gap-2 lg:grid-cols-2">
		<!-- Agent Lifecycle Events -->
		<div class="rounded-lg border bg-card">
			<div class="flex items-center justify-between border-b px-3 py-1.5">
				<div class="flex items-center gap-2">
					<h3 class="text-xs font-semibold">Agent Lifecycle</h3>
					<span class="text-xs text-muted-foreground">~/.orch/events.jsonl</span>
				</div>
				<Button
					variant={$agentlogConnectionStatus === 'connected' ? 'destructive' : 'ghost'}
					size="sm"
					onclick={handleAgentlogConnectClick}
					class="h-5 px-2 text-xs"
				>
					{#if $agentlogConnectionStatus === 'connecting'}
						...
					{:else if $agentlogConnectionStatus === 'connected'}
						Stop
					{:else}
						Follow
					{/if}
				</Button>
			</div>
			<div class="max-h-64 overflow-y-auto p-2 font-mono text-sm">
				{#each $agentlogEvents.slice().reverse().slice(0, 20) as event (event.id)}
					<div class="flex items-center gap-1 py-0.5 text-muted-foreground">
						<span>{getEventIcon(event.type)}</span>
						<span class="opacity-60">{formatUnixTime(event.timestamp)}</span>
						<Badge variant="outline" class="h-4 px-1 text-xs">
							{getEventLabel(event.type)}
						</Badge>
						{#if event.session_id}
							<span class="font-medium text-foreground">{event.session_id.slice(0, 8)}</span>
						{/if}
						{#if event.data?.error}
							<span class="text-red-500 font-semibold">{event.data.error}</span>
						{/if}
					</div>
				{:else}
					<p class="py-2 text-center text-muted-foreground">
						{#if $agentlogConnectionStatus === 'connected'}
							Waiting...
						{:else}
							No events
						{/if}
					</p>
				{/each}
			</div>
		</div>

		<!-- SSE Events -->
		<div class="rounded-lg border bg-card">
			<div class="flex items-center justify-between border-b px-3 py-1.5">
				<div class="flex items-center gap-2">
					<h3 class="text-xs font-semibold">SSE Stream</h3>
					<span class="text-xs text-muted-foreground">OpenCode events</span>
				</div>
				<span class="text-xs text-muted-foreground">{$sseEvents.length} events</span>
			</div>
			<div class="max-h-64 overflow-y-auto p-2 font-mono text-sm">
				{#each $sseEvents.slice().reverse().slice(0, 20) as event (event.id)}
					<div class="flex items-center gap-1 py-0.5 text-muted-foreground">
						<span class="opacity-60">{formatTime(event.timestamp)}</span>
						<span class="text-foreground">{event.type}</span>
						{#if event.properties?.sessionID}
							<span class="opacity-60">{event.properties.sessionID.slice(0, 8)}</span>
						{/if}
					</div>
				{:else}
					<p class="py-2 text-center text-muted-foreground">
						{#if $connectionStatus === 'connected'}
							Waiting...
						{:else}
							Click Connect
						{/if}
					</p>
				{/each}
			</div>
		</div>
	</div>
</div>

<!-- Agent Detail Slide-out Panel -->
<AgentDetailPanel />
