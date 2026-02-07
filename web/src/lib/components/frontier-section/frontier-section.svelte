<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { frontier, formatLeverage, type FrontierIssue, type BlockedIssue, type ActiveAgent } from '$lib/stores/frontier';

	export let expanded: boolean = true;
	export let maxItems: number = 8;

	function toggle() {
		expanded = !expanded;
	}

	// Truncate title to maxLen characters
	function truncateTitle(title: string, maxLen: number = 50): string {
		if (title.length <= maxLen) return title;
		return title.substring(0, maxLen - 3) + '...';
	}

	// Get preview of first few items
	function getPreview(): string {
		const items: string[] = [];

		if ($frontier.ready_total > 0) {
			items.push(`${$frontier.ready_total} ready`);
		}
		if ($frontier.blocked_total > 0) {
			items.push(`${$frontier.blocked_total} blocked`);
		}
		if ($frontier.active_total > 0) {
			items.push(`${$frontier.active_total} active`);
		}
		if ($frontier.stuck_total > 0) {
			items.push(`${$frontier.stuck_total} stuck`);
		}

		return items.join(', ');
	}

	// Check if there are any stuck agents (health warning)
	$: hasWarnings = $frontier.stuck_total > 0;

	// Total items across all categories
	$: totalItems = $frontier.ready_total + $frontier.blocked_total + $frontier.active_total + $frontier.stuck_total;
</script>

{#if totalItems > 0 || $frontier.error}
	<div
		class="rounded-lg border {hasWarnings ? 'border-yellow-500/30 bg-yellow-500/5' : 'border-purple-500/30 bg-purple-500/5'}"
		data-testid="frontier-section"
	>
		<button
			class="flex w-full items-center justify-between px-3 py-2 text-left hover:bg-accent/50 transition-colors"
			onclick={toggle}
			aria-expanded={expanded}
			data-testid="frontier-toggle"
		>
			<div class="flex items-center gap-2 min-w-0 flex-1">
				<span class="text-sm flex-shrink-0">🗺️</span>
				<span class="text-sm font-medium flex-shrink-0">Frontier</span>
				<Badge variant="secondary" class="h-5 px-1.5 text-xs flex-shrink-0">
					{$frontier.ready_total + $frontier.active_total}
				</Badge>
				{#if hasWarnings}
					<Badge variant="destructive" class="h-5 px-1.5 text-xs flex-shrink-0">
						{$frontier.stuck_total} stuck
					</Badge>
				{/if}
				{#if !expanded}
					<span class="text-xs text-muted-foreground truncate" data-testid="frontier-preview">
						— {getPreview()}
					</span>
				{/if}
			</div>
			<span class="text-muted-foreground transition-transform flex-shrink-0 {expanded ? 'rotate-180' : ''}">
				<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="6 9 12 15 18 9"></polyline>
				</svg>
			</span>
		</button>

		{#if expanded}
			<div class="border-t p-3 space-y-4 font-mono text-sm" data-testid="frontier-content">
				{#if $frontier.error}
					<div class="text-red-500">
						Error: {$frontier.error}
					</div>
				{:else}
					<!-- Health Warnings -->
					{#if $frontier.warnings && $frontier.warnings.length > 0}
						<div class="text-yellow-500 text-xs">
							{#each $frontier.warnings as warning}
								<div>⚠️ {warning}</div>
							{/each}
						</div>
					{/if}

					<!-- READY TO RELEASE -->
					<div>
						<div class="text-green-500 font-semibold mb-1">
							READY TO RELEASE ({$frontier.ready_total})
						</div>
						{#if $frontier.ready.length === 0}
							<div class="text-muted-foreground pl-4">(none)</div>
						{:else}
							<div class="pl-4 space-y-0.5">
								{#each $frontier.ready.slice(0, maxItems) as issue (issue.id)}
									<div class="flex gap-2">
										<span class="text-blue-400">{issue.id}</span>
										<span class="text-foreground truncate" title={issue.title}>
											{truncateTitle(issue.title)}
										</span>
									</div>
								{/each}
								{#if $frontier.ready_total > maxItems}
									<div class="text-muted-foreground">
										... and {$frontier.ready_total - maxItems} more
									</div>
								{/if}
							</div>
						{/if}
					</div>

					<!-- BLOCKED -->
					<div>
						<div class="text-red-400 font-semibold mb-1">
							BLOCKED ({$frontier.blocked_total})
						</div>
						{#if $frontier.blocked.length === 0}
							<div class="text-muted-foreground pl-4">(none)</div>
						{:else}
							<div class="pl-4 space-y-0.5">
								{#each $frontier.blocked.slice(0, maxItems) as blocked (blocked.id)}
									{@const leverage = formatLeverage(blocked)}
									<div class="flex gap-2">
										<span class="text-blue-400">{blocked.id}</span>
										<span class="text-foreground truncate" title={blocked.title}>
											{truncateTitle(blocked.title, 40)}
										</span>
										{#if leverage}
											<span class="text-yellow-400 flex-shrink-0">
												→ {leverage}
											</span>
										{/if}
									</div>
								{/each}
								{#if $frontier.blocked_total > maxItems}
									<div class="text-muted-foreground">
										... and {$frontier.blocked_total - maxItems} more
									</div>
								{/if}
							</div>
						{/if}
					</div>

					<!-- ACTIVE -->
					<div>
						<div class="text-blue-400 font-semibold mb-1">
							ACTIVE ({$frontier.active_total})
						</div>
						{#if $frontier.active.length === 0}
							<div class="text-muted-foreground pl-4">(none)</div>
						{:else}
							<div class="pl-4 space-y-0.5">
								{#each $frontier.active.slice(0, maxItems) as agent (agent.beads_id)}
									<div class="flex gap-2">
										<span class="text-blue-400">{agent.beads_id}</span>
										<span class="text-muted-foreground">[{agent.runtime}]</span>
										{#if agent.skill}
											<span class="text-purple-400">({agent.skill})</span>
										{/if}
									</div>
								{/each}
								{#if $frontier.active_total > maxItems}
									<div class="text-muted-foreground">
										... and {$frontier.active_total - maxItems} more
									</div>
								{/if}
							</div>
						{/if}
					</div>

					<!-- STUCK -->
					{#if $frontier.stuck_total > 0}
						<div>
							<div class="text-yellow-500 font-semibold mb-1">
								STUCK (&gt; 2h) ({$frontier.stuck_total})
							</div>
							<div class="pl-4 space-y-0.5">
								{#each $frontier.stuck.slice(0, maxItems) as agent (agent.beads_id)}
									<div class="flex gap-2">
										<span class="text-blue-400">{agent.beads_id}</span>
										<span class="text-muted-foreground">[{agent.runtime}]</span>
									</div>
								{/each}
								{#if $frontier.stuck_total > maxItems}
									<div class="text-muted-foreground">
										... and {$frontier.stuck_total - maxItems} more
									</div>
								{/if}
							</div>
						</div>
					{/if}
				{/if}
			</div>
		{/if}
	</div>
{/if}
