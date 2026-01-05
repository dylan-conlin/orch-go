<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import type { OrchestratorSession } from '$lib/stores/orchestrator-sessions';
	import { getProjectIcon } from '$lib/stores/orchestrator-sessions';

	export let session: OrchestratorSession;

	function getStatusVariant(status: string) {
		switch (status) {
			case 'active':
				return 'default';
			case 'completed':
				return 'secondary';
			default:
				return 'outline';
		}
	}

	/**
	 * Truncate goal text to specified length
	 */
	function truncateGoal(goal: string, maxLen: number = 60): string {
		if (goal.length <= maxLen) return goal;
		const truncated = goal.substring(0, maxLen);
		const lastSpace = truncated.lastIndexOf(' ');
		if (lastSpace > maxLen * 0.6) {
			return truncated.substring(0, lastSpace) + '...';
		}
		return truncated + '...';
	}

	/**
	 * Extract readable name from workspace name
	 * e.g., "og-orch-ship-feature-05jan" -> "Ship feature"
	 */
	function extractWorkspaceDisplay(workspaceName: string): string {
		return workspaceName
			.replace(/^[a-z]+-orch-/, '') // Remove prefix like og-orch-
			.replace(/-\d{1,2}[a-z]{3}$/, '') // Remove date suffix
			.replace(/-/g, ' ')
			.trim()
			.split(' ')
			.map(word => word.charAt(0).toUpperCase() + word.slice(1))
			.join(' ');
	}
</script>

<div
	class="group relative w-full rounded-lg border-2 border-purple-500/50 bg-card p-3 transition-all hover:border-purple-500 hover:shadow-md hover:shadow-purple-500/20"
>
	<!-- Orchestrator icon badge -->
	<div class="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-purple-600 text-[10px] shadow-sm">
		<Tooltip.Root>
			<Tooltip.Trigger>
				<span>O</span>
			</Tooltip.Trigger>
			<Tooltip.Content>
				<p class="font-medium">Orchestrator Session</p>
				<p class="text-xs text-muted-foreground">Coordinates worker agents</p>
			</Tooltip.Content>
		</Tooltip.Root>
	</div>

	<!-- Header: Project icon + Status -->
	<div class="flex items-center justify-between gap-2">
		<div class="flex items-center gap-2">
			<span class="text-lg">{getProjectIcon(session.project)}</span>
			<Badge variant={getStatusVariant(session.status)} class="h-5 px-2 text-xs bg-purple-600/80 hover:bg-purple-600">
				{session.status}
			</Badge>
		</div>
		<div class="flex items-center gap-1 text-xs text-muted-foreground">
			<Tooltip.Root>
				<Tooltip.Trigger>
					<span class="cursor-default font-mono">{session.duration}</span>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p>Started: {new Date(session.spawn_time).toLocaleString()}</p>
				</Tooltip.Content>
			</Tooltip.Root>
		</div>
	</div>

	<!-- Goal (main focus) -->
	<div class="mt-2">
		<Tooltip.Root>
			<Tooltip.Trigger>
				{#snippet child({ props })}
					<p {...props} class="text-sm font-semibold leading-tight cursor-default">
						{truncateGoal(session.goal)}
					</p>
				{/snippet}
			</Tooltip.Trigger>
			<Tooltip.Content class="max-w-sm">
				<p class="font-medium">Goal</p>
				<p class="text-sm">{session.goal}</p>
			</Tooltip.Content>
		</Tooltip.Root>
	</div>

	<!-- Workspace + Project info -->
	<div class="mt-2 flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground">
		<Tooltip.Root>
			<Tooltip.Trigger>
				<span class="font-mono cursor-default">{extractWorkspaceDisplay(session.workspace_name)}</span>
			</Tooltip.Trigger>
			<Tooltip.Content>
				<p class="font-mono text-xs">{session.workspace_name}</p>
			</Tooltip.Content>
		</Tooltip.Root>
		<span class="text-muted-foreground/50">|</span>
		<Badge variant="secondary" class="h-4 px-1.5 text-[10px]">
			{session.project}
		</Badge>
	</div>

	<!-- Child agents count (if any) -->
	{#if session.child_agent_count > 0}
		<div class="mt-2 border-t border-purple-500/20 pt-2">
			<div class="flex items-center gap-2 text-xs">
				<Tooltip.Root>
					<Tooltip.Trigger>
						<span class="flex items-center gap-1 cursor-default text-purple-400">
							<span class="text-sm">&#8627;</span>
							{session.child_agent_count} active agent{session.child_agent_count === 1 ? '' : 's'}
						</span>
					</Tooltip.Trigger>
					<Tooltip.Content>
						<p>Worker agents spawned in {session.project}</p>
					</Tooltip.Content>
				</Tooltip.Root>
			</div>
		</div>
	{/if}
</div>
