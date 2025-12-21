<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { SynthesisCard } from '$lib/components/synthesis-card';
	import type { Agent } from '$lib/stores/agents';

	export let agent: Agent;

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

	function getStatusColor(status: Agent['status']) {
		switch (status) {
			case 'active':
				return 'bg-green-500';
			case 'completed':
				return 'bg-blue-500';
			case 'abandoned':
				return 'bg-red-500';
			default:
				return 'bg-gray-500';
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

	function formatDate(isoDate: string): string {
		return new Date(isoDate).toLocaleDateString(undefined, {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit'
		});
	}
</script>

<div class="group relative rounded-lg border bg-card p-4 transition-all hover:border-primary/50 hover:shadow-md">
	<!-- Status indicator bar at top -->
	<div class={`absolute left-0 top-0 h-1 w-full rounded-t-lg ${getStatusColor(agent.status)}`}></div>

	<!-- Primary row: Status + Duration -->
	<div class="mt-1 flex items-center justify-between">
		<div class="flex items-center gap-2">
			<Badge variant={getStatusVariant(agent.status)} class="text-xs font-medium">
				{agent.status}
			</Badge>
			{#if agent.status === 'active'}
				<span class="flex items-center gap-1 text-xs text-muted-foreground">
					<span class="h-1.5 w-1.5 animate-pulse rounded-full bg-green-500"></span>
					{formatDuration(agent.spawned_at)}
				</span>
			{:else}
				<span class="text-xs text-muted-foreground">
					{formatDuration(agent.spawned_at)}
				</span>
			{/if}
		</div>
	</div>

	<!-- Secondary: Agent ID (prominent) -->
	<div class="mt-3">
		<p class="font-mono text-sm font-semibold text-foreground" title={agent.id}>
			{agent.id}
		</p>
	</div>

	<!-- Tertiary: Skill + Beads ID -->
	<div class="mt-2 flex flex-wrap items-center gap-2">
		{#if agent.skill}
			<Badge variant="outline" class="text-xs">
				{agent.skill}
			</Badge>
		{/if}
		{#if agent.beads_id}
			<span class="text-xs text-muted-foreground" title="Beads Issue">
				{agent.beads_id}
			</span>
		{/if}
	</div>

	<!-- Metadata: Spawned time -->
	<div class="mt-3 border-t border-border/50 pt-2">
		<span class="text-xs text-muted-foreground">
			Spawned {formatDate(agent.spawned_at)}
		</span>
	</div>

	<!-- Synthesis Card for completed agents -->
	{#if agent.status === 'completed' && agent.synthesis}
		<SynthesisCard synthesis={agent.synthesis} />
	{/if}
</div>
