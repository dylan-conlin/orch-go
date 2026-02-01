<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { wip, wipItems, wipStats, type WIPItem } from '$lib/stores/wip';
	import { agents } from '$lib/stores/agents';
	import { Badge } from '$lib/components/ui/badge';

	// Refresh interval for queued issues (30s)
	let refreshInterval: ReturnType<typeof setInterval>;

	onMount(async () => {
		// Initial fetch of queued issues
		await wip.fetchQueued();
		
		// Set up refresh interval
		refreshInterval = setInterval(() => {
			wip.fetchQueued();
		}, 30000);
	});

	onDestroy(() => {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}
	});

	// Sync running agents from the agents store
	$: wip.setRunningAgents($agents);

	// Get status icon for running agents
	function getAgentStatusIcon(status: string): string {
		switch (status) {
			case 'active': return '▶';
			case 'idle': return '⏸';
			default: return '•';
		}
	}

	// Get priority badge variant
	function getPriorityVariant(priority: number): 'destructive' | 'secondary' | 'outline' {
		if (priority === 0) return 'destructive';
		if (priority === 1) return 'secondary';
		return 'outline';
	}

	// Get type badge color
	function getTypeBadge(type: string): string {
		switch (type.toLowerCase()) {
			case 'epic': return 'bg-purple-500/10 text-purple-500';
			case 'feature': return 'bg-blue-500/10 text-blue-500';
			case 'bug': return 'bg-red-500/10 text-red-500';
			case 'task': return 'bg-green-500/10 text-green-500';
			case 'question': return 'bg-yellow-500/10 text-yellow-500';
			default: return 'bg-muted text-muted-foreground';
		}
	}
</script>

{#if $wipStats.running > 0 || $wipStats.queued > 0}
	<div class="wip-section border-b-2 border-primary/30 bg-primary/5">
		<!-- Header -->
		<div class="flex items-center justify-between px-6 py-2 border-b border-border/50">
			<div class="flex items-center gap-2">
				<span class="text-sm font-medium text-foreground">Work in Progress</span>
				{#if $wipStats.running > 0}
					<Badge variant="secondary" class="text-xs">
						{$wipStats.running} running
					</Badge>
				{/if}
				{#if $wipStats.queued > 0}
					<Badge variant="outline" class="text-xs">
						{$wipStats.queued} queued
					</Badge>
				{/if}
			</div>
		</div>

		<!-- Items -->
		<div class="px-6 py-2 space-y-1">
			{#each $wipItems as item}
				{#if item.type === 'running'}
					<!-- Running Agent -->
					<div class="flex items-center gap-3 py-1.5 px-2 rounded text-sm">
						<span class="text-blue-500 w-4">{getAgentStatusIcon(item.agent.status)}</span>
						<span class="font-mono text-xs text-muted-foreground min-w-[100px]">
							{item.agent.beads_id || item.agent.id.slice(0, 15)}
						</span>
						<span class="flex-1 text-foreground truncate">
							{item.agent.task || item.agent.skill || 'Unknown task'}
						</span>
						{#if item.agent.phase}
							<Badge variant="outline" class="text-xs bg-blue-500/10 text-blue-500">
								{item.agent.phase}
							</Badge>
						{/if}
						{#if item.agent.runtime}
							<span class="text-xs text-muted-foreground">{item.agent.runtime}</span>
						{/if}
					</div>
				{:else}
					<!-- Queued Issue -->
					<div class="flex items-center gap-3 py-1.5 px-2 rounded text-sm opacity-60">
						<span class="text-muted-foreground w-4">○</span>
						<Badge variant={getPriorityVariant(item.issue.priority)} class="w-8 justify-center text-xs">
							P{item.issue.priority}
						</Badge>
						<span class="font-mono text-xs text-muted-foreground min-w-[100px]">
							{item.issue.id}
						</span>
						<span class="flex-1 text-foreground truncate">
							{item.issue.title}
						</span>
						<Badge variant="outline" class="{getTypeBadge(item.issue.issue_type)} text-xs">
							{item.issue.issue_type}
						</Badge>
					</div>
				{/if}
			{/each}
		</div>
	</div>
{/if}
