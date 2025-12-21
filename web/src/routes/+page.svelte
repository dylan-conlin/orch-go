<script lang="ts">
	import { onMount } from 'svelte';
	import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '$lib/components/ui/card';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import {
		agents,
		activeAgents,
		completedAgents,
		abandonedAgents,
		sseEvents,
		connectionStatus,
		type Agent,
		type SSEEvent
	} from '$lib/stores/agents';

	// Mock data for development - will be replaced with SSE stream
	onMount(() => {
		// Simulate some agents
		agents.set([
			{
				id: 'og-feat-add-cli-20dec',
				beads_id: 'orch-go-abc',
				status: 'active',
				skill: 'feature-impl',
				spawned_at: new Date(Date.now() - 3600000).toISOString(),
				updated_at: new Date().toISOString()
			},
			{
				id: 'og-inv-explore-20dec',
				beads_id: 'orch-go-def',
				status: 'active',
				skill: 'investigation',
				spawned_at: new Date(Date.now() - 1800000).toISOString(),
				updated_at: new Date().toISOString()
			},
			{
				id: 'og-debug-fix-sse-19dec',
				beads_id: 'orch-go-ghi',
				status: 'completed',
				skill: 'systematic-debugging',
				spawned_at: new Date(Date.now() - 7200000).toISOString(),
				updated_at: new Date(Date.now() - 3600000).toISOString(),
				completed_at: new Date(Date.now() - 3600000).toISOString()
			}
		]);
		connectionStatus.set('disconnected');

		// TODO: Connect to SSE stream
		// const eventSource = new EventSource('/api/events');
		// eventSource.onmessage = (event) => { ... };
	});

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
</script>

<div class="space-y-8">
	<!-- Stats Overview -->
	<div class="grid gap-4 md:grid-cols-4">
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
				<Button variant="outline" size="sm" disabled>
					Connect SSE
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
						<span class="text-muted-foreground">{event.type}</span>
					</div>
				{:else}
					<p class="text-muted-foreground">
						Waiting for SSE connection...
					</p>
				{/each}
			</div>
		</CardContent>
	</Card>
</div>
