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
		disconnectAgentlogSSE
	} from '$lib/stores/agentlog';

	// Filter and sort state
	let statusFilter: AgentState | 'all' = $state('all');
	let skillFilter: string = $state('all');
	let sortBy: 'newest' | 'oldest' | 'alphabetical' = $state('newest');

	// Get unique skills from agents
	let uniqueSkills = $derived(
		[...new Set($agents.map(a => a.skill).filter(Boolean))] as string[]
	);

	// Filtered and sorted agents
	let filteredAgents = $derived.by(() => {
		let result = $agents.filter(a => a.status !== 'deleted');

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
					return new Date(b.spawned_at).getTime() - new Date(a.spawned_at).getTime();
				case 'oldest':
					return new Date(a.spawned_at).getTime() - new Date(b.spawned_at).getTime();
				case 'alphabetical':
					return a.id.localeCompare(b.id);
				default:
					return 0;
			}
		});

		return result;
	});

	onMount(() => {
		// Fetch initial agents
		agents.fetch().catch((err) => {
			console.error('Initial fetch failed:', err);
		});

		// Fetch initial agentlog
		agentlogEvents.fetch().catch((err) => {
			console.error('Initial agentlog fetch failed:', err);
		});

		// Connect to SSE for real-time updates
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
	}

	let hasActiveFilters = $derived(
		statusFilter !== 'all' || skillFilter !== 'all' || sortBy !== 'newest'
	);
</script>

<div class="space-y-8">
	<!-- Stats Overview -->
	<div class="grid gap-4 md:grid-cols-5">
		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">Active Agents</CardTitle>
				<span class="text-2xl">🐝</span>
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">{$activeAgents.length}</div>
				<p class="text-xs text-muted-foreground">Currently working</p>
			</CardContent>
		</Card>
		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">Completed</CardTitle>
				<span class="text-2xl">✅</span>
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">{$completedAgents.length}</div>
				<p class="text-xs text-muted-foreground">Tasks finished</p>
			</CardContent>
		</Card>
		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">Abandoned</CardTitle>
				<span class="text-2xl">⚠️</span>
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">{$abandonedAgents.length}</div>
				<p class="text-xs text-muted-foreground">Stuck or failed</p>
			</CardContent>
		</Card>
		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">Agent Log</CardTitle>
				<span class="text-2xl">📋</span>
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">{$agentlogEvents.length}</div>
				<p class="text-xs text-muted-foreground">Lifecycle events</p>
			</CardContent>
		</Card>
		<Card>
			<CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
				<CardTitle class="text-sm font-medium">SSE Events</CardTitle>
				<span class="text-2xl">📡</span>
			</CardHeader>
			<CardContent>
				<div class="text-2xl font-bold">{$sseEvents.length}</div>
				<p class="text-xs text-muted-foreground">Last 100 events</p>
			</CardContent>
		</Card>
	</div>

	<!-- Swarm Map -->
	<Card>
		<CardHeader>
			<div class="flex items-center justify-between">
				<div>
					<CardTitle>Swarm Map</CardTitle>
					<CardDescription>Real-time view of all agent activity</CardDescription>
				</div>
				<Button
					variant={$connectionStatus === 'connected' ? 'destructive' : 'outline'}
					size="sm"
					onclick={handleConnectClick}
				>
					{#if $connectionStatus === 'connecting'}
						Connecting...
					{:else if $connectionStatus === 'connected'}
						Disconnect
					{:else}
						Connect SSE
					{/if}
				</Button>
			</div>
		</CardHeader>
		<CardContent>
			<!-- Filter Bar -->
			<div class="mb-4 flex flex-wrap items-center gap-3 rounded-lg border bg-muted/30 p-3" data-testid="filter-bar">
				<!-- Status Filter -->
				<div class="flex items-center gap-2">
					<label for="status-filter" class="text-sm font-medium text-muted-foreground">Status:</label>
					<select
						id="status-filter"
						bind:value={statusFilter}
						class="h-8 rounded-md border border-input bg-background px-2 text-sm"
						data-testid="status-filter"
					>
						<option value="all">All</option>
						<option value="active">Active</option>
						<option value="completed">Completed</option>
						<option value="abandoned">Abandoned</option>
					</select>
				</div>

				<!-- Skill Filter -->
				{#if uniqueSkills.length > 0}
					<div class="flex items-center gap-2">
						<label for="skill-filter" class="text-sm font-medium text-muted-foreground">Skill:</label>
						<select
							id="skill-filter"
							bind:value={skillFilter}
							class="h-8 rounded-md border border-input bg-background px-2 text-sm"
							data-testid="skill-filter"
						>
							<option value="all">All</option>
							{#each uniqueSkills as skill}
								<option value={skill}>{skill}</option>
							{/each}
						</select>
					</div>
				{/if}

				<!-- Sort -->
				<div class="flex items-center gap-2">
					<label for="sort-by" class="text-sm font-medium text-muted-foreground">Sort:</label>
					<select
						id="sort-by"
						bind:value={sortBy}
						class="h-8 rounded-md border border-input bg-background px-2 text-sm"
						data-testid="sort-select"
					>
						<option value="newest">Newest first</option>
						<option value="oldest">Oldest first</option>
						<option value="alphabetical">A-Z</option>
					</select>
				</div>

				<!-- Clear Filters -->
				{#if hasActiveFilters}
					<Button variant="ghost" size="sm" onclick={clearFilters} class="ml-auto text-xs">
						Clear filters
					</Button>
				{/if}

				<!-- Result count -->
				<span class="ml-auto text-xs text-muted-foreground" data-testid="filter-count">
					{filteredAgents.length} agent{filteredAgents.length === 1 ? '' : 's'}
				</span>
			</div>

			<!-- Agent Grid -->
			<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3" data-testid="agent-grid">
				{#each filteredAgents as agent (agent.id)}
					<AgentCard {agent} />
				{:else}
					<div class="col-span-full rounded-lg border border-dashed p-8 text-center">
						{#if hasActiveFilters}
							<p class="text-muted-foreground">No agents match the current filters</p>
							<Button variant="link" onclick={clearFilters} class="mt-2">
								Clear filters
							</Button>
						{:else}
							<p class="text-muted-foreground">No agents in the swarm</p>
							<p class="mt-1 text-sm text-muted-foreground">
								Agents will appear here when spawned via <code class="rounded bg-muted px-1">orch spawn</code>
							</p>
						{/if}
					</div>
				{/each}
			</div>
		</CardContent>
	</Card>

	<!-- Agent Lifecycle Events -->
	<Card>
		<CardHeader>
			<div class="flex items-center justify-between">
				<div>
					<CardTitle>Agent Lifecycle</CardTitle>
					<CardDescription>Events from ~/.orch/events.jsonl</CardDescription>
				</div>
				<Button
					variant={$agentlogConnectionStatus === 'connected' ? 'destructive' : 'outline'}
					size="sm"
					onclick={handleAgentlogConnectClick}
				>
					{#if $agentlogConnectionStatus === 'connecting'}
						Connecting...
					{:else if $agentlogConnectionStatus === 'connected'}
						Disconnect
					{:else}
						Follow Live
					{/if}
				</Button>
			</div>
		</CardHeader>
		<CardContent>
			<div class="max-h-80 overflow-y-auto rounded-lg bg-muted/50 p-4 font-mono text-xs">
				{#each $agentlogEvents.slice().reverse() as event, i (i)}
					<div class="border-b border-border py-2 last:border-0">
						<span class="mr-2">{getEventIcon(event.type)}</span>
						<span class="text-muted-foreground">[{formatUnixTime(event.timestamp)}]</span>
						<Badge variant="outline" class="ml-2 text-xs">
							{getEventLabel(event.type)}
						</Badge>
						{#if event.session_id}
							<span class="ml-2 font-medium">{event.session_id.slice(0, 12)}...</span>
						{/if}
						{#if event.data?.title}
							<span class="ml-2 text-muted-foreground">"{event.data.title}"</span>
						{/if}
						{#if event.data?.error}
							<span class="ml-2 text-red-500">{event.data.error}</span>
						{/if}
						{#if event.data?.status}
							<span class="ml-2 text-muted-foreground">→ {event.data.status}</span>
						{/if}
					</div>
				{:else}
					<p class="text-muted-foreground">
						{#if $agentlogConnectionStatus === 'connected'}
							Waiting for agent events...
						{:else if $agentlogConnectionStatus === 'connecting'}
							Connecting to agentlog stream...
						{:else}
							No agent lifecycle events yet. Spawn agents with <code class="rounded bg-muted px-1">orch spawn</code>
						{/if}
					</p>
				{/each}
			</div>
		</CardContent>
	</Card>

	<!-- Recent Events -->
	<Card>
		<CardHeader>
			<CardTitle>Recent Events</CardTitle>
			<CardDescription>SSE event stream from OpenCode</CardDescription>
		</CardHeader>
		<CardContent>
			<div class="max-h-64 overflow-y-auto rounded-lg bg-muted/50 p-4 font-mono text-xs">
				{#each $sseEvents.slice().reverse() as event, i (i)}
					<div class="border-b border-border py-2 last:border-0">
						<span class="text-muted-foreground">[{formatTime(event.timestamp)}]</span>
						<span class="ml-2">{event.type}</span>
						{#if event.properties?.sessionID}
							<span class="ml-2 text-muted-foreground">session: {event.properties.sessionID.slice(0, 8)}...</span>
						{/if}
					</div>
				{:else}
					<p class="text-muted-foreground">
						{#if $connectionStatus === 'connected'}
							Waiting for events...
						{:else if $connectionStatus === 'connecting'}
							Connecting to SSE...
						{:else}
							Click "Connect SSE" to start receiving events
						{/if}
					</p>
				{/each}
			</div>
		</CardContent>
	</Card>
</div>
