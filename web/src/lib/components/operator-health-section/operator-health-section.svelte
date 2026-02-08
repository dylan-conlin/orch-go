<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { operatorHealth, type HealthStatus } from '$lib/stores/operator-health';

	export let expanded = true;

	const statusRank: Record<HealthStatus, number> = {
		healthy: 0,
		unknown: 1,
		warning: 2,
		critical: 3
	};

	const statusLabel: Record<HealthStatus, string> = {
		healthy: 'Healthy',
		warning: 'Watch',
		critical: 'At Risk',
		unknown: 'Unknown'
	};

	function statusBadgeClass(status: HealthStatus): string {
		switch (status) {
			case 'healthy':
				return 'bg-green-500/20 text-green-700 border-green-500/30';
			case 'warning':
				return 'bg-amber-500/20 text-amber-700 border-amber-500/30';
			case 'critical':
				return 'bg-red-500/20 text-red-700 border-red-500/30';
			default:
				return 'bg-slate-500/20 text-slate-700 border-slate-500/30';
		}
	}

	function overallStatus(statuses: HealthStatus[]): HealthStatus {
		let current: HealthStatus = 'healthy';
		for (const status of statuses) {
			if (statusRank[status] > statusRank[current]) {
				current = status;
			}
		}
		return current;
	}

	function formatPercent(value: number): string {
		return `${(value * 100).toFixed(0)}%`;
	}

	function formatBytes(bytes: number): string {
		if (bytes < 0) return 'n/a';
		if (bytes < 1024) return `${bytes} B`;
		if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
		if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
		return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
	}

	$: statuses = [
		$operatorHealth.crash_free_streak.status,
		$operatorHealth.resource_ceilings.status,
		$operatorHealth.investigation_rate_30d.status,
		$operatorHealth.defect_class_clusters.status,
		$operatorHealth.agent_health_ratio_7d.status,
		$operatorHealth.process_census.status
	];

	$: overall = overallStatus(statuses);

	$: completionShare = formatPercent($operatorHealth.agent_health_ratio_7d.completion_share || 0);
</script>

<div class="rounded-lg border-2 border-slate-500/30 bg-card" data-testid="operator-health-section">
	<button
		class="flex w-full items-center justify-between border-b border-slate-500/20 px-3 py-2 text-left hover:bg-accent/50 transition-colors"
		onclick={() => {
			expanded = !expanded;
		}}
		aria-expanded={expanded}
		data-testid="operator-health-toggle"
	>
		<div class="flex min-w-0 items-center gap-2">
			<span class="text-sm">🛡</span>
			<span class="text-sm font-medium">Operator Health</span>
			<Badge variant="secondary" class={`h-5 px-1.5 text-xs border ${statusBadgeClass(overall)}`}>
				{statusLabel[overall]}
			</Badge>
			{#if $operatorHealth.generated_at}
				<span class="truncate text-xs text-muted-foreground">
					Updated {new Date($operatorHealth.generated_at).toLocaleTimeString()}
				</span>
			{/if}
		</div>
		<span class="text-muted-foreground transition-transform {expanded ? 'rotate-180' : ''}">
			<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<polyline points="6 9 12 15 18 9"></polyline>
			</svg>
		</span>
	</button>

	{#if expanded}
		<div class="grid gap-2 p-2 sm:grid-cols-2">
			<div class="rounded border border-border/60 p-2">
				<div class="mb-1 flex items-center justify-between">
					<span class="text-xs font-medium">Crash-Free Streak</span>
					<Badge variant="outline" class={`h-5 px-1.5 text-xs border ${statusBadgeClass($operatorHealth.crash_free_streak.status)}`}>
						{statusLabel[$operatorHealth.crash_free_streak.status]}
					</Badge>
				</div>
				<p class="text-sm font-semibold">{$operatorHealth.crash_free_streak.current_streak}</p>
				<p class="text-xs text-muted-foreground">
					{$operatorHealth.crash_free_streak.current_streak_days} day streak · target {$operatorHealth.crash_free_streak.target_days} days
				</p>
				{#if $operatorHealth.crash_free_streak.last_intervention}
					<p class="mt-1 text-xs text-muted-foreground">
						Last recovery: {$operatorHealth.crash_free_streak.last_intervention.source}
					</p>
				{/if}
			</div>

			<div class="rounded border border-border/60 p-2">
				<div class="mb-1 flex items-center justify-between">
					<span class="text-xs font-medium">Resource Ceilings</span>
					<Badge variant="outline" class={`h-5 px-1.5 text-xs border ${statusBadgeClass($operatorHealth.resource_ceilings.status)}`}>
						{statusLabel[$operatorHealth.resource_ceilings.status]}
					</Badge>
				</div>
				<p class="text-xs text-muted-foreground">
					Goroutines {$operatorHealth.resource_ceilings.current.goroutines} / {$operatorHealth.resource_ceilings.baseline.goroutines}
				</p>
				<p class="text-xs text-muted-foreground">
					Heap {formatBytes($operatorHealth.resource_ceilings.current.heap_bytes)} / {formatBytes($operatorHealth.resource_ceilings.baseline.heap_bytes)}
				</p>
				<p class="text-xs text-muted-foreground">
					Child procs {$operatorHealth.resource_ceilings.current.child_processes} / {$operatorHealth.resource_ceilings.baseline.child_processes}
				</p>
				<p class="text-xs text-muted-foreground">
					FDs {$operatorHealth.resource_ceilings.current.open_file_descriptors} / {$operatorHealth.resource_ceilings.baseline.open_file_descriptors}
				</p>
				{#if $operatorHealth.resource_ceilings.breached}
					<p class="mt-1 text-xs text-red-600">
						{$operatorHealth.resource_ceilings.breaches?.length || 0} metric(s) over {$operatorHealth.resource_ceilings.ceiling_multiplier}x baseline
					</p>
				{/if}
			</div>

			<div class="rounded border border-border/60 p-2">
				<div class="mb-1 flex items-center justify-between">
					<span class="text-xs font-medium">Investigations (30d)</span>
					<Badge variant="outline" class={`h-5 px-1.5 text-xs border ${statusBadgeClass($operatorHealth.investigation_rate_30d.status)}`}>
						{statusLabel[$operatorHealth.investigation_rate_30d.status]}
					</Badge>
				</div>
				<p class="text-sm font-semibold">{$operatorHealth.investigation_rate_30d.count}</p>
				<p class="text-xs text-muted-foreground">
					Warning from {$operatorHealth.investigation_rate_30d.warning_from} · fire-fighting threshold {$operatorHealth.investigation_rate_30d.threshold}
				</p>
			</div>

			<div class="rounded border border-border/60 p-2">
				<div class="mb-1 flex items-center justify-between">
					<span class="text-xs font-medium">Defect Clusters (30d)</span>
					<Badge variant="outline" class={`h-5 px-1.5 text-xs border ${statusBadgeClass($operatorHealth.defect_class_clusters.status)}`}>
						{statusLabel[$operatorHealth.defect_class_clusters.status]}
					</Badge>
				</div>
				{#if $operatorHealth.defect_class_clusters.top_classes.length > 0}
					<div class="space-y-0.5">
						{#each $operatorHealth.defect_class_clusters.top_classes.slice(0, 4) as item}
							<p class="text-xs text-muted-foreground">
								<span class="font-medium text-foreground">{item.defect_class}</span> · {item.count}
							</p>
						{/each}
					</div>
				{:else}
					<p class="text-xs text-muted-foreground">No clustered defect class signal.</p>
				{/if}
			</div>

			<div class="rounded border border-border/60 p-2">
				<div class="mb-1 flex items-center justify-between">
					<span class="text-xs font-medium">Agent Health Ratio (7d)</span>
					<Badge variant="outline" class={`h-5 px-1.5 text-xs border ${statusBadgeClass($operatorHealth.agent_health_ratio_7d.status)}`}>
						{statusLabel[$operatorHealth.agent_health_ratio_7d.status]}
					</Badge>
				</div>
				<p class="text-xs text-muted-foreground">
					{$operatorHealth.agent_health_ratio_7d.completions} completions · {$operatorHealth.agent_health_ratio_7d.abandonments} abandonments
				</p>
				<p class="text-xs text-muted-foreground">Completion share {completionShare}</p>
				{#if $operatorHealth.agent_health_ratio_7d.completions_per_abandonment !== undefined}
					<p class="text-xs text-muted-foreground">
						{$operatorHealth.agent_health_ratio_7d.completions_per_abandonment.toFixed(2)} completions per abandonment
					</p>
				{/if}
			</div>

			<div class="rounded border border-border/60 p-2">
				<div class="mb-1 flex items-center justify-between">
					<span class="text-xs font-medium">Process Census</span>
					<Badge variant="outline" class={`h-5 px-1.5 text-xs border ${statusBadgeClass($operatorHealth.process_census.status)}`}>
						{statusLabel[$operatorHealth.process_census.status]}
					</Badge>
				</div>
				<p class="text-xs text-muted-foreground">
					{$operatorHealth.process_census.child_processes} active child process(es)
				</p>
				<p class="text-xs text-muted-foreground">
					{$operatorHealth.process_census.orphaned_count} orphan process(es) with PPID=1
				</p>
				{#if $operatorHealth.process_census.orphaned_processes && $operatorHealth.process_census.orphaned_processes.length > 0}
					<div class="mt-1 space-y-0.5">
						{#each $operatorHealth.process_census.orphaned_processes.slice(0, 3) as process}
							<p class="text-xs text-red-600 break-all">PID {process.pid} · {process.command}</p>
						{/each}
					</div>
				{/if}
			</div>
		</div>

		{#if $operatorHealth.errors && $operatorHealth.errors.length > 0}
			<div class="px-2 pb-2">
				<p class="rounded border border-amber-500/30 bg-amber-500/10 p-2 text-xs text-amber-700">
					Partial data: {$operatorHealth.errors.join(' | ')}
				</p>
			</div>
		{/if}
	{/if}
</div>
