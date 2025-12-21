<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { SynthesisCard } from '$lib/components/synthesis-card';
	import {
		agents,
		activeAgents,
		completedAgents,
		abandonedAgents,
		sseEvents,
		connectionStatus,
		connectSSE,
		disconnectSSE,
		type Agent
	} from '$lib/stores/agents';
	import {
		agentlogEvents,
		agentlogConnectionStatus,
		connectAgentlogSSE,
		disconnectAgentlogSSE,
		type AgentLogEvent
	} from '$lib/stores/agentlog';

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

	function getStatusVariant(status: Agent['status']) {
		switch (status) {
			case 'active':
				return 'active';
			case 'completed':
				return 'completed';
			case 'abandoned':
				return 'abandoned';
			default:
				return 'default';
		}
	}

	function formatDuration(isoDate: string): string {
		const ms = Date.now() - new Date(isoDate).getTime();
		const minutes = Math.floor(ms / 60000);
		const hours = Math.floor(minutes / 60);
		if (hours > 0) {
			return `${hours}h ${minutes % 60}m`;
		}
		return `${minutes}m`;
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
			<div class="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
				{#each $agents.filter(a => a.status !== 'deleted') as agent (agent.id)}
					<div class="rounded-lg border p-4 transition-all hover:border-primary/50 hover:shadow-md">
						<div class="flex items-start justify-between">
							<div class="space-y-1">
								<div class="flex items-center gap-2">
									<Badge variant={getStatusVariant(agent.status)}>
										{agent.status}
									</Badge>
									{#if agent.skill}
										<Badge variant="outline" class="text-xs">
											{agent.skill}
										</Badge>
									{/if}
								</div>
								<p class="font-mono text-sm font-medium">{agent.id}</p>
								{#if agent.beads_id}
									<p class="text-xs text-muted-foreground">
										{agent.beads_id}
									</p>
								{/if}
							</div>
						</div>
						<div class="mt-4 flex items-center justify-between text-xs text-muted-foreground">
							<span>Duration: {formatDuration(agent.spawned_at)}</span>
							{#if agent.status === 'active'}
								<span class="flex items-center gap-1">
									<span class="h-2 w-2 animate-pulse rounded-full bg-green-500"></span>
									Running
								</span>
							{/if}
						</div>

						<!-- Synthesis Card for completed agents -->
						{#if agent.status === 'completed' && agent.synthesis}
							<SynthesisCard synthesis={agent.synthesis} />
						{/if}
					</div>
				{:else}
					<div class="col-span-full rounded-lg border border-dashed p-8 text-center">
						<p class="text-muted-foreground">No agents in the swarm</p>
						<p class="mt-1 text-sm text-muted-foreground">
							Agents will appear here when spawned via <code class="rounded bg-muted px-1">orch spawn</code>
						</p>
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
