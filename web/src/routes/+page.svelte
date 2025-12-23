<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { AgentCard } from '$lib/components/agent-card';
	import {
		agents,
		activeAgents,
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

	// Filter and sort state
	let statusFilter: AgentState | 'all' = 'all';
	let skillFilter: string = 'all';
	let sortBy: 'newest' | 'oldest' | 'alphabetical' = 'newest';
	let activeOnly: boolean = false;

	// Get unique skills from agents
	$: uniqueSkills = [...new Set($agents.map(a => a.skill).filter(Boolean))] as string[];

	// Filtered and sorted agents
	$: filteredAgents = (() => {
		let result = $agents.filter(a => a.status !== 'deleted');

		// Apply active-only filter
		if (activeOnly) {
			result = result.filter(a => a.status === 'active');
		}

		// Apply status filter
		if (statusFilter !== 'all') {
			result = result.filter(a => a.status === statusFilter);
		}

		// Apply skill filter
		if (skillFilter !== 'all') {
			result = result.filter(a => a.skill === skillFilter);
		}

		// Apply sorting
		result = [...result].sort((a, b) => {
			switch (sortBy) {
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
				default:
					return 0;
			}
		});

		return result;
	})();

	onMount(() => {
		// Connect to SSE - this will trigger initial fetch when connection opens
		// Removes race condition from parallel fetch + SSE connect
		connectSSE();
		connectAgentlogSSE();
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
		sortBy = 'newest';
		activeOnly = false;
	}

	$: hasActiveFilters = statusFilter !== 'all' || skillFilter !== 'all' || sortBy !== 'newest' || activeOnly;
</script>

<div class="space-y-3">
	<!-- Compact Stats Bar -->
	<div class="flex items-center gap-4 rounded-lg border bg-card px-4 py-2" data-testid="stats-bar">
		<div class="flex items-center gap-2">
			<span class="text-lg">🐝</span>
			<div class="flex items-baseline gap-1">
				<span class="text-xl font-bold">{$activeAgents.length}</span>
				<span class="text-xs text-muted-foreground">active</span>
			</div>
		</div>
		<div class="h-4 w-px bg-border"></div>
		<div class="flex items-center gap-2">
			<span class="text-lg">✅</span>
			<div class="flex items-baseline gap-1">
				<span class="text-xl font-bold">{$completedAgents.length}</span>
				<span class="text-xs text-muted-foreground">done</span>
			</div>
		</div>
		<div class="h-4 w-px bg-border"></div>
		<div class="flex items-center gap-2">
			<span class="text-lg">⚠️</span>
			<div class="flex items-baseline gap-1">
				<span class="text-xl font-bold">{$abandonedAgents.length}</span>
				<span class="text-xs text-muted-foreground">stuck</span>
			</div>
		</div>
		<div class="h-4 w-px bg-border"></div>
		<div class="flex items-center gap-2">
			<span class="text-lg">❌</span>
			<div class="flex items-baseline gap-1">
				<span class="text-xl font-bold" class:text-red-500={$errorEvents.length > 0}>{$errorEvents.length}</span>
				<span class="text-xs text-muted-foreground">errors</span>
			</div>
		</div>
		<div class="h-4 w-px bg-border"></div>
		<div class="flex items-center gap-2">
			<span class="text-lg">📋</span>
			<div class="flex items-baseline gap-1">
				<span class="text-xl font-bold">{$agentlogEvents.length}</span>
				<span class="text-xs text-muted-foreground">events</span>
			</div>
		</div>
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

				<select
					id="sort-by"
					bind:value={sortBy}
					class="h-6 rounded border border-input bg-background px-1.5 text-xs"
					data-testid="sort-select"
				>
					<option value="newest">Newest</option>
					<option value="oldest">Oldest</option>
					<option value="alphabetical">A-Z</option>
				</select>

				{#if hasActiveFilters}
					<button onclick={clearFilters} class="text-xs text-muted-foreground hover:text-foreground">
						Clear
					</button>
				{/if}

				<span class="ml-auto text-muted-foreground" data-testid="filter-count">
					{filteredAgents.length} agent{filteredAgents.length === 1 ? '' : 's'}
				</span>
			</div>

			<!-- Dense Agent Grid -->
			<div class="grid gap-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5" data-testid="agent-grid">
				{#each filteredAgents as agent, idx (idx)}
					<AgentCard {agent} />
				{:else}
					<div class="col-span-full rounded border border-dashed p-6 text-center">
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
				{/each}
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
				{#each $agentlogEvents.slice().reverse().slice(0, 20) as event, i (i)}
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
				{#each $sseEvents.slice().reverse().slice(0, 20) as event, i (i)}
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
