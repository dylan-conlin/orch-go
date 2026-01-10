<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { AgentCard } from '$lib/components/agent-card';
	import { AgentDetailPanel } from '$lib/components/agent-detail';
	import { CollapsibleSection } from '$lib/components/collapsible-section';
	// PendingReviewsSection removed - not actively used
	import { ReadyQueueSection } from '$lib/components/ready-queue-section';
	import { UpNextSection } from '$lib/components/up-next-section';
	import { RecentWins } from '$lib/components/recent-wins';
	import { NeedsAttention } from '$lib/components/needs-attention';
	import { StatsBar } from '$lib/components/stats-bar';
	import { CacheValidationBanner } from '$lib/components/cache-validation-banner';
	import {
		agents,
		activeAgents,
		needsReviewAgents,
		trulyActiveAgents,
		recentAgents,
		archivedAgents,
		deadAgents,
		sseEvents,
		connectionStatus,
		connectSSE,
		disconnectSSE,
		setFilterQueryStringCallback,
		type Agent,
		type AgentState
	} from '$lib/stores/agents';
	import {
		agentlogEvents,
		agentlogConnectionStatus,
		connectAgentlogSSE,
		disconnectAgentlogSSE
	} from '$lib/stores/agentlog';
	import {
		servicelogEvents,
		servicelogConnectionStatus,
		connectServicelogSSE,
		disconnectServicelogSSE
	} from '$lib/stores/servicelog';
	import { usage } from '$lib/stores/usage';
	import { focus } from '$lib/stores/focus';
	import { servers } from '$lib/stores/servers';
	import { beads, readyIssues } from '$lib/stores/beads';
	import { daemon } from '$lib/stores/daemon';
	// pendingReviews store removed - not actively used
	import { dashboardMode } from '$lib/stores/dashboard-mode';
	import { config } from '$lib/stores/config';
	import { hotspots } from '$lib/stores/hotspot';
	import { orchestratorSessions } from '$lib/stores/orchestrator-sessions';
	import { OrchestratorSessionsSection } from '$lib/components/orchestrator-sessions-section';
	import { services } from '$lib/stores/services';
	import { ServicesSection } from '$lib/components/services-section';
	import { filters, orchestratorContext, buildFilterQueryString } from '$lib/stores/context';

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
		needsReview: true, // Needs Review expanded by default (high importance)
		recent: false,  // Recent collapsed by default
		archive: false, // Archive collapsed by default
		upNext: false,  // Up Next collapsed by default (auto-expands on P0/P1)
		readyQueue: false, // Ready queue collapsed by default
		// pendingReviews removed - not actively used
		sseStream: false, // SSE Stream collapsed by default (low signal-to-noise for most users)
		orchestratorSessions: true, // Orchestrator sessions expanded by default (important visibility)
		services: true // Services expanded by default (important visibility)
	};
	
	// Track whether component has mounted and loaded initial state
	// Prevents reactive save from overwriting stored preferences during initialization
	let sectionStateLoaded = false;

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
		// Mark as loaded AFTER updating sectionState to avoid triggering save
		sectionStateLoaded = true;
	}

	// Save section state to localStorage
	function saveSectionState() {
		if (typeof window === 'undefined') return;
		if (!sectionStateLoaded) return; // Don't save until initial state is loaded
		try {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(sectionState));
		} catch (e) {
			console.warn('Failed to save section state:', e);
		}
	}

	// Reactive saving when state changes (only after initial load)
	$: if (sectionStateLoaded && sectionState) {
		saveSectionState();
	}

	// Get unique skills from agents
	$: uniqueSkills = [...new Set($agents.map(a => a.skill).filter(Boolean))] as string[];

	// Get unique projects from agents
	$: uniqueProjects = [...new Set($agents.map(a => a.project).filter(Boolean))].sort() as string[];

	onMount(() => {
		// Initialize dashboard mode from localStorage (must be in onMount for SSR)
		dashboardMode.init();

		// Load section state from localStorage (sync, instant)
		loadSectionState();

		// Start context polling if followOrchestrator is enabled
		if ($filters.followOrchestrator) {
			orchestratorContext.startPolling(2000); // Poll every 2 seconds
		}

		// Set up filter query string callback for SSE-triggered fetches
		setFilterQueryStringCallback(() => buildFilterQueryString($filters));

		// Connect to primary SSE immediately - this triggers agents.fetch() on connection
		// which is the most critical data for initial render
		connectSSE();

		// Connect to servicelog SSE for real-time service crash/restart notifications
		connectServicelogSSE();

		// Fetch critical data in parallel using Promise.all
		// These affect the primary dashboard view and should load ASAP
		// Note: beads.fetch() is called without projectDir initially - will be refetched
		// when orchestrator context is loaded (see reactive block below)
		Promise.all([
			beads.fetch(),
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
			hotspots.fetch();
			orchestratorSessions.fetch();
			services.fetch();
		};

		// Use requestIdleCallback for better performance, with setTimeout fallback
		if ('requestIdleCallback' in window) {
			requestIdleCallback(deferSecondaryFetches, { timeout: 2000 });
		} else {
			setTimeout(deferSecondaryFetches, 100);
		}

		// NOTE: Agentlog SSE connection is NO LONGER auto-connected on page load.
		// This fixes connection pool exhaustion where two SSE connections (primary events 
		// + agentlog) consumed 2 of 6 HTTP/1.1 connections per origin, blocking API fetches.
		// Users can manually connect via the "Follow" button in the Agent Lifecycle panel.
		// See: .kb/investigations/2026-01-05-inv-dashboard-connection-pool-exhaustion-sse.md

		// Refresh all data every 60 seconds (all fetches are already non-blocking)
		const refreshInterval = setInterval(() => {
			// Get current project_dir from context (if following orchestrator)
			const projectDir = $filters.followOrchestrator ? $orchestratorContext.project_dir : undefined;
			
			Promise.all([
				usage.fetch(),
				focus.fetch(),
				servers.fetch(),
				beads.fetch(projectDir),
				readyIssues.fetch(projectDir),
				daemon.fetch(),
				hotspots.fetch(),
				orchestratorSessions.fetch(),
				services.fetch()
			]).catch(console.error);
		}, 60000);

		// Clean up connections before page unload to avoid Firefox network errors
		const handleBeforeUnload = () => {
			disconnectSSE();
			disconnectAgentlogSSE();
			disconnectServicelogSSE();
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
		disconnectServicelogSSE();
		orchestratorContext.stopPolling();
	});

	// React to followOrchestrator changes - start/stop context polling
	$: {
		if (typeof window !== 'undefined') {
			if ($filters.followOrchestrator) {
				orchestratorContext.startPolling(2000);
			} else {
				orchestratorContext.stopPolling();
			}
		}
	}

	// Build query string from current filter state
	// Updates project filter when orchestrator context changes (if following)
	$: {
		if ($filters.followOrchestrator && $orchestratorContext.project) {
			// Auto-update project filter from orchestrator context
			filters.setProjectFilter($orchestratorContext.project, $orchestratorContext.included_projects);
		}
	}

	// Refetch beads when orchestrator context changes (if following)
	// This makes the beads stats and ready queue follow which project the orchestrator is working in
	$: {
		if (typeof window !== 'undefined' && $filters.followOrchestrator && $orchestratorContext.project_dir) {
			beads.fetch($orchestratorContext.project_dir).catch(console.error);
			readyIssues.fetch($orchestratorContext.project_dir).catch(console.error);
		}
	}

	// Reactive query string based on filter state
	$: filterQueryString = buildFilterQueryString($filters);

	// Refetch agents when filters change (debounced via the store)
	$: if (filterQueryString !== undefined && typeof window !== 'undefined') {
		// Only refetch if we're connected (SSE connection triggers initial fetch)
		if ($connectionStatus === 'connected') {
			agents.fetch(filterQueryString).catch(console.error);
		}
	}

	function formatTime(timestamp?: number): string {
		if (!timestamp) return '';
		return new Date(timestamp).toLocaleTimeString();
	}

	function formatUnixTime(timestamp: number): string {
		return new Date(timestamp * 1000).toLocaleTimeString();
	}

	const eventIcons: Record<string, string> = { 'session.spawned': '🚀', 'session.completed': '✅', 'session.error': '❌', 'session.status': '📊' };
	const eventLabels: Record<string, string> = { 'session.spawned': 'Spawned', 'session.completed': 'Completed', 'session.error': 'Error', 'session.status': 'Status' };
	function getEventIcon(type: string): string { return eventIcons[type] || '📝'; }
	function getEventLabel(type: string): string { return eventLabels[type] || type; }

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

	// Apply all filters (skill + project)
	function applyFilters(agentList: Agent[]): Agent[] {
		return applyProjectFilter(applySkillFilter(agentList));
	}

	// Progressive disclosure: sorted and filtered agents per section
	// Active (truly running, excludes needs-review) and Recent use stable sort (spawned_at) to prevent jostling from SSE updates
	// Archive uses volatile sort (updated_at) since historical recency matters more there
	$: sortedActiveAgents = sortAgents(applyFilters($trulyActiveAgents), true);
	$: sortedNeedsReviewAgents = sortAgents(applyFilters($needsReviewAgents), true);
	$: sortedDeadAgents = sortAgents(applyFilters($deadAgents), true);
	$: sortedRecentAgents = sortAgents(applyFilters($recentAgents), true);
	$: sortedArchivedAgents = sortAgents(applyFilters($archivedAgents), false);

	// Total visible agents across all sections (for filter count)
	$: totalVisibleAgents = sortedActiveAgents.length + sortedNeedsReviewAgents.length + sortedDeadAgents.length + sortedRecentAgents.length + sortedArchivedAgents.length;
</script>

<!-- Cache Validation Banner (fixed at top) -->
<CacheValidationBanner />

<div class="space-y-3">
	<!-- Stats Bar Component -->
	<StatsBar bind:readyQueueExpanded={sectionState.readyQueue} />

	<!-- Orchestrator Sessions (always visible at top when active) -->
	<OrchestratorSessionsSection
		bind:expanded={sectionState.orchestratorSessions}
	/>

	<!-- Services (overmind-managed processes) -->
	<ServicesSection
		bind:expanded={sectionState.services}
	/>

	{#if $dashboardMode === 'operational'}
		<!-- OPERATIONAL MODE: Focused daily coordination view -->
		
		<!-- Up Next (priority queue visibility) -->
		<UpNextSection
			bind:expanded={sectionState.upNext}
		/>
		
		<!-- Active Agents (truly running, excludes needs-review) -->
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
					<div class="rounded border border-dashed p-6 text-center">
						<p class="text-sm text-muted-foreground">No active agents</p>
						<p class="mt-1 text-xs text-muted-foreground">
							Spawn with <code class="rounded bg-muted px-1">orch spawn</code>
						</p>
					</div>
				{/if}
			</div>
		</div>

		<!-- Needs Review (Phase: Complete, awaiting orch complete) -->
		{#if sortedNeedsReviewAgents.length > 0}
			<div class="rounded-lg border bg-card border-amber-500/30" data-testid="needs-review-section">
				<button
					class="flex w-full items-center justify-between px-3 py-2 text-left hover:bg-accent/50 transition-colors border-b"
					onclick={() => { sectionState.needsReview = !sectionState.needsReview; }}
					aria-expanded={sectionState.needsReview}
				>
					<div class="flex items-center gap-2">
						<span class="text-sm">✅</span>
						<span class="text-sm font-medium">Needs Review</span>
						<Badge variant="secondary" class="h-5 px-1.5 text-xs bg-amber-500/20 text-amber-600">
							{sortedNeedsReviewAgents.length}
						</Badge>
					</div>
					<span class="text-muted-foreground transition-transform {sectionState.needsReview ? 'rotate-180' : ''}">
						<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
							<polyline points="6 9 12 15 18 9"></polyline>
						</svg>
					</span>
				</button>
				{#if sectionState.needsReview}
					<div class="p-2">
						<div class="grid gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
							{#each sortedNeedsReviewAgents as agent, i (`${agent.id}-${agent.session_id ?? i}`)}
								<AgentCard {agent} />
							{/each}
						</div>
						<p class="mt-2 text-xs text-muted-foreground text-center">
							Run <code class="rounded bg-muted px-1">orch complete</code> to review
						</p>
					</div>
				{/if}
			</div>
		{/if}

		<!-- Needs Attention (consolidated errors, pending reviews, blocked) -->
		<NeedsAttention />

		<!-- Recent Wins (completed in last 24h) -->
		<RecentWins />

		<!-- Ready Queue (collapsed by default) -->
		<ReadyQueueSection
			bind:expanded={sectionState.readyQueue}
		/>

	{:else}
		<!-- HISTORICAL MODE: Full archive with SSE stream and filters -->

		<!-- Up Next (priority queue visibility) -->
		<UpNextSection
			bind:expanded={sectionState.upNext}
		/>

		<!-- Ready Queue Section (dedicated collapsible section) -->
		<ReadyQueueSection
			bind:expanded={sectionState.readyQueue}
		/>

		<!-- Pending Reviews Section removed - not actively used -->

		<!-- Swarm Map (Primary Focus) -->
		<div class="rounded-lg border bg-card">
			<div class="flex items-center justify-between border-b px-3 py-2">
				<div class="flex items-center gap-2">
					<h2 class="text-sm font-semibold">Swarm Map</h2>
					<span class="text-xs text-muted-foreground">Full archive ({$agents.length} agents)</span>
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
					<button onclick={clearFilters} class="text-xs text-muted-foreground hover:text-foreground" data-testid="clear-filters-button">
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
						{#each sortedActiveAgents as agent, i (`${agent.id}-${agent.session_id ?? i}`)}
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
						<!-- Active Section (truly running, excludes needs-review) -->
						{#if sortedActiveAgents.length > 0 || (sortedNeedsReviewAgents.length === 0 && sortedRecentAgents.length === 0 && sortedArchivedAgents.length === 0)}
							<CollapsibleSection
								title="Active"
								icon="🟢"
								agents={sortedActiveAgents}
								bind:expanded={sectionState.active}
								variant="active"
							>
								<div class="grid gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
									{#each sortedActiveAgents as agent, i (`${agent.id}-${agent.session_id ?? i}`)}
										<AgentCard {agent} />
									{/each}
								</div>
							</CollapsibleSection>
						{/if}

						<!-- Needs Review Section (Phase: Complete, awaiting orch complete) -->
						{#if sortedNeedsReviewAgents.length > 0}
							<CollapsibleSection
								title="Needs Review"
								icon="✅"
								agents={sortedNeedsReviewAgents}
								bind:expanded={sectionState.needsReview}
								variant="needs-review"
							>
								<div class="grid gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
									{#each sortedNeedsReviewAgents as agent, i (`${agent.id}-${agent.session_id ?? i}`)}
										<AgentCard {agent} />
									{/each}
								</div>
								<p class="mt-2 text-xs text-muted-foreground text-center">
									Run <code class="rounded bg-muted px-1">orch complete</code> to review
								</p>
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

						<!-- Empty state when no agents at all -->
						{#if sortedActiveAgents.length === 0 && sortedNeedsReviewAgents.length === 0 && sortedRecentAgents.length === 0 && sortedArchivedAgents.length === 0}
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
					<Tooltip.Root>
					<Tooltip.Trigger>
						{#snippet child({ props })}
							<Button
								{...props}
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
						{/snippet}
					</Tooltip.Trigger>
					<Tooltip.Content>
						{#if $agentlogConnectionStatus === 'connected'}
							<p>Stop following agent lifecycle events</p>
						{:else if $agentlogConnectionStatus === 'connecting'}
							<p>Connecting to event stream...</p>
						{:else}
							<p>Follow agent lifecycle events</p>
							<p class="text-xs text-muted-foreground">Watch spawns, completions, and errors</p>
						{/if}
					</Tooltip.Content>
				</Tooltip.Root>
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

			<!-- SSE Events (collapsible) -->
			<div class="rounded-lg border bg-card">
				<button
					class="flex w-full items-center justify-between px-3 py-1.5 text-left hover:bg-accent/50 transition-colors border-b"
					onclick={() => { sectionState.sseStream = !sectionState.sseStream; }}
					aria-expanded={sectionState.sseStream}
					data-testid="sse-stream-toggle"
				>
					<div class="flex items-center gap-2">
						<h3 class="text-xs font-semibold">SSE Stream</h3>
						<span class="text-xs text-muted-foreground">OpenCode events</span>
					</div>
					<div class="flex items-center gap-2">
						<span class="text-xs text-muted-foreground">{$sseEvents.length} events</span>
						<span class="text-muted-foreground transition-transform {sectionState.sseStream ? 'rotate-180' : ''}">
							<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
								<polyline points="6 9 12 15 18 9"></polyline>
							</svg>
						</span>
					</div>
				</button>
				{#if sectionState.sseStream}
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
				{/if}
			</div>
		</div>
	{/if}
</div>

<!-- Agent Detail Slide-out Panel -->
<AgentDetailPanel />
