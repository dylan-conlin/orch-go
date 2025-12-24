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

	function getPhaseVariant(phase?: string): 'default' | 'secondary' | 'outline' {
		if (!phase) return 'outline';
		switch (phase.toLowerCase()) {
			case 'complete':
				return 'default';
			case 'implementing':
				return 'secondary';
			default:
				return 'outline';
		}
	}

	function formatDuration(isoDate: string | undefined): string {
		if (!isoDate) return '-';
		const date = new Date(isoDate);
		if (isNaN(date.getTime())) return '-';
		const ms = Date.now() - date.getTime();
		const minutes = Math.floor(ms / 60000);
		const hours = Math.floor(minutes / 60);
		if (hours > 0) {
			return `${hours}h${minutes % 60}m`;
		}
		return `${minutes}m`;
	}

	function getActivityIcon(type?: string): string {
		switch (type) {
			case 'text':
				return '💬';
			case 'tool':
			case 'tool-invocation':
				return '🔧';
			case 'reasoning':
				return '🤔';
			case 'step-start':
				return '▶️';
			case 'step-finish':
				return '✓';
			default:
				return '📝';
		}
	}

	function formatActivityAge(timestamp?: number): string {
		if (!timestamp) return '';
		const seconds = Math.floor((Date.now() - timestamp) / 1000);
		if (seconds < 5) return 'now';
		if (seconds < 60) return `${seconds}s ago`;
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes}m ago`;
		const hours = Math.floor(minutes / 60);
		return `${hours}h ago`;
	}
</script>

<div class="group relative rounded border bg-card p-2 transition-all hover:border-primary/50 hover:shadow-sm">
	<!-- Status indicator bar at top -->
	<div class={`absolute left-0 top-0 h-0.5 w-full rounded-t ${getStatusColor(agent.status)}`}></div>

	<!-- Header: Status + Phase + Duration -->
	<div class="flex items-center justify-between gap-1">
		<div class="flex items-center gap-1">
			<Badge variant={getStatusVariant(agent.status)} class="h-4 px-1.5 text-[10px]">
				{agent.status}
			</Badge>
			{#if agent.phase}
				<Badge variant={getPhaseVariant(agent.phase)} class="h-4 px-1 text-[10px]">
					{agent.phase}
				</Badge>
			{/if}
		</div>
		<span class="flex items-center gap-0.5 text-[10px] text-muted-foreground">
			{#if agent.is_processing}
				<span class="h-1.5 w-1.5 animate-pulse rounded-full bg-yellow-500" title="Generating response"></span>
			{:else if agent.status === 'active'}
				<span class="h-1 w-1 rounded-full bg-green-500"></span>
			{/if}
			{agent.runtime || formatDuration(agent.spawned_at)}
		</span>
	</div>

	<!-- Agent ID -->
	<p class="mt-1 truncate font-mono text-xs font-medium" title={agent.id}>
		{agent.id}
	</p>

	<!-- Task (from beads issue) -->
	{#if agent.task}
		<p class="mt-0.5 truncate text-[10px] text-muted-foreground" title={agent.task}>
			{agent.task}
		</p>
	{/if}

	<!-- Project + Skill + Beads -->
	<div class="mt-1 flex flex-wrap items-center gap-1">
		{#if agent.project}
			<Badge variant="secondary" class="h-4 px-1 text-[10px]">
				{agent.project}
			</Badge>
		{/if}
		{#if agent.skill}
			<Badge variant="outline" class="h-4 px-1 text-[10px]">
				{agent.skill}
			</Badge>
		{/if}
		{#if agent.beads_id}
			<span class="text-[10px] text-muted-foreground" title="Beads Issue">
				{agent.beads_id}
			</span>
		{/if}
	</div>

	<!-- Current Activity (for active agents) -->
	{#if agent.status === 'active' && agent.current_activity}
		<div class="mt-1.5 border-t border-border/50 pt-1.5">
			<div class="flex items-start gap-1">
				<span class="text-[10px]">{getActivityIcon(agent.current_activity.type)}</span>
				<div class="flex-1 min-w-0">
					<p class="line-clamp-2 text-[10px] leading-tight text-muted-foreground">
						{agent.current_activity.text || 'Working...'}
					</p>
					<span class="text-[9px] text-muted-foreground/70">
						{formatActivityAge(agent.current_activity.timestamp)}
					</span>
				</div>
			</div>
		</div>
	{/if}

	<!-- Compact Synthesis for completed agents -->
	{#if agent.status === 'completed' && agent.synthesis}
		<div class="mt-1.5 border-t border-border/50 pt-1.5">
			{#if agent.synthesis.tldr}
				<p class="line-clamp-2 text-[10px] leading-tight text-muted-foreground">
					{agent.synthesis.tldr}
				</p>
			{/if}
			{#if agent.synthesis.outcome}
				<Badge variant={agent.synthesis.outcome === 'success' ? 'default' : 'secondary'} class="mt-1 h-4 px-1 text-[10px]">
					{agent.synthesis.outcome}
				</Badge>
			{/if}
		</div>
	{/if}
</div>
