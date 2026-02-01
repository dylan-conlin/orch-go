<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { wip, wipItems, wipStats, getExpressiveStatus, computeAgentHealth, getContextPercent, getContextColor, type WIPItem } from '$lib/stores/wip';
	import { agents, type Agent } from '$lib/stores/agents';
	import { daemon } from '$lib/stores/daemon';
	import { Badge } from '$lib/components/ui/badge';

	// Refresh interval for queued issues and daemon status (30s)
	let refreshInterval: ReturnType<typeof setInterval>;

	onMount(async () => {
		// Initial fetch of queued issues and daemon status
		await Promise.all([
			wip.fetchQueued(),
			daemon.fetch()
		]);
		
		// Set up refresh interval
		refreshInterval = setInterval(() => {
			wip.fetchQueued();
			daemon.fetch();
		}, 30000);
	});

	onDestroy(() => {
		if (refreshInterval) {
			clearInterval(refreshInterval);
		}
	});

	// Sync running agents from the agents store
	$: wip.setRunningAgents($agents);

	// Get status icon for running agents based on health
	function getAgentStatusIcon(agent: Agent): { icon: string; color: string } {
		const health = computeAgentHealth(agent);
		
		if (health.status === 'critical') {
			return { icon: '🚨', color: 'text-red-500' };
		}
		if (health.status === 'warning') {
			return { icon: '⚠️', color: 'text-yellow-500' };
		}
		
		// Healthy - show activity-based icon
		if (agent.is_processing) {
			return { icon: '◉', color: 'text-blue-500 animate-pulse' };
		}
		if (agent.status === 'idle') {
			return { icon: '⏸', color: 'text-muted-foreground' };
		}
		return { icon: '▶', color: 'text-blue-500' };
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

{#if $wipStats.running > 0}
	<div class="wip-section border-b-2 border-primary/30 bg-primary/5">
		<!-- Header -->
		<div class="flex items-center justify-between px-6 py-2 border-b border-border/50">
			<div class="flex items-center gap-2">
				<span class="text-sm font-medium text-foreground">Work in Progress</span>
				<Badge variant="secondary" class="text-xs">
					{$wipStats.running} running
				</Badge>
				{#if $wipStats.queued > 0}
					<Badge variant="outline" class="text-xs">
						+{$wipStats.queued} queued
					</Badge>
				{/if}
			</div>
			<!-- Daemon status -->
			{#if $daemon}
				<div class="flex items-center gap-2 text-xs text-muted-foreground">
					{#if $daemon.running}
						<span class="text-green-500">●</span>
						<span>{$daemon.capacity_used}/{$daemon.capacity_max} slots</span>
					{:else}
						<span class="text-yellow-500">●</span>
						<span>daemon stopped</span>
					{/if}
				</div>
			{/if}
		</div>

		<!-- Items - matching work-graph-tree row styling -->
		<div class="px-6 py-4">
			{#each $wipItems as item}
				{#if item.type === 'running'}
					{@const statusIcon = getAgentStatusIcon(item.agent)}
					{@const health = computeAgentHealth(item.agent)}
					{@const contextPct = getContextPercent(item.agent)}
					<!-- Running Agent - L0 row -->
					<div class="running-agent">
						<div class="flex items-center gap-3 py-2 px-3 rounded">
							<!-- Status icon with health indication -->
							<span class="{statusIcon.color} w-5 text-center">{statusIcon.icon}</span>
							
							<!-- Priority placeholder (w-8 matches tree badge width) -->
							<span class="w-8"></span>
							
							<!-- ID (min-w-[120px] matches tree) -->
							<span class="text-xs font-mono text-muted-foreground min-w-[120px]">
								{item.agent.beads_id || item.agent.id.slice(0, 15)}
							</span>
							
							<!-- Title (text-sm font-medium matches tree) -->
							<span class="flex-1 text-sm font-medium text-foreground truncate">
								{item.agent.task || item.agent.skill || 'Unknown task'}
							</span>
							
							<!-- Expressive status (replaces phase badge) -->
							<span class="text-xs text-muted-foreground italic min-w-[120px]">
								{getExpressiveStatus(item.agent)}
							</span>
							
							<!-- Health warning tooltip -->
							{#if health.status !== 'healthy'}
								<span class="text-xs {health.status === 'critical' ? 'text-red-500' : 'text-yellow-500'}" title={health.reasons.join(', ')}>
									{health.status === 'critical' ? '!' : '?'}
								</span>
							{/if}
							
							<!-- Runtime -->
							{#if item.agent.runtime}
								<span class="text-xs text-muted-foreground min-w-[40px] text-right">{item.agent.runtime}</span>
							{/if}
						</div>
						
						<!-- L1: Auto-expanded details for running agents -->
						<div class="ml-14 pb-2 px-3 flex items-center gap-4 text-xs text-muted-foreground">
							<!-- Phase -->
							{#if item.agent.phase}
								<span class="flex items-center gap-1">
									<span class="text-foreground/60">Phase:</span>
									<span class="text-foreground">{item.agent.phase}</span>
								</span>
							{/if}
							
							<!-- Context % -->
							{#if contextPct !== null}
								<span class="flex items-center gap-1">
									<span class="text-foreground/60">Context:</span>
									<span class="{getContextColor(contextPct)}">{contextPct}%</span>
								</span>
							{/if}
							
							<!-- Skill -->
							{#if item.agent.skill}
								<span class="flex items-center gap-1">
									<span class="text-foreground/60">Skill:</span>
									<span>{item.agent.skill}</span>
								</span>
							{/if}
							
							<!-- Model (short form) -->
							{#if item.agent.model}
								<span class="flex items-center gap-1">
									<span class="text-foreground/60">Model:</span>
									<span>{item.agent.model.split('/').pop()?.split('-').slice(0, 2).join('-') || item.agent.model}</span>
								</span>
							{/if}
						</div>
					</div>
				{:else}
					<!-- Queued Issue - matches tree row structure -->
					<div class="flex items-center gap-3 py-2 px-3 rounded opacity-60">
						<!-- Status icon -->
						<span class="text-muted-foreground w-5">○</span>
						
						<!-- Priority badge -->
						<Badge variant={getPriorityVariant(item.issue.priority)} class="w-8 justify-center text-xs">
							P{item.issue.priority}
						</Badge>
						
						<!-- ID -->
						<span class="text-xs font-mono text-muted-foreground min-w-[120px]">
							{item.issue.id}
						</span>
						
						<!-- Title -->
						<span class="flex-1 text-sm font-medium text-foreground truncate">
							{item.issue.title}
						</span>
						
						<!-- Type badge -->
						<Badge variant="outline" class="{getTypeBadge(item.issue.issue_type)} text-xs">
							{item.issue.issue_type}
						</Badge>
					</div>
				{/if}
			{/each}
		</div>
	</div>
{/if}
