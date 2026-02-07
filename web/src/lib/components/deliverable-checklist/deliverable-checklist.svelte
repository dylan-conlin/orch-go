<script lang="ts">
	import type { Deliverable } from '$lib/stores/deliverables';
	import { getCompletionStats } from '$lib/stores/deliverables';

	// Props
	export let deliverables: Deliverable[] = [];
	export let mode: 'compact' | 'full' = 'compact'; // L1 (compact) or L2 (full)

	// Compute stats
	$: stats = getCompletionStats(deliverables);

	// Get icon for status
	function getStatusIcon(status: Deliverable['status']): string {
		switch (status) {
			case 'complete':
				return '✓';
			case 'incomplete':
				return '○';
			case 'skipped':
				return '✗';
			default:
				return '?';
		}
	}

	// Get color class for status
	function getStatusColor(status: Deliverable['status']): string {
		switch (status) {
			case 'complete':
				return 'text-green-500';
			case 'incomplete':
				return 'text-muted-foreground';
			case 'skipped':
				return 'text-yellow-500';
			default:
				return 'text-muted-foreground';
		}
	}
</script>

{#if mode === 'compact'}
	<!-- L1: Compact view - just icons (✓ ✓ ○ ○) -->
	<div class="flex items-center gap-1" data-testid="deliverables-compact">
		<span class="text-xs text-foreground/60 mr-1">Deliverables:</span>
		{#each deliverables as deliverable}
			<span
				class="{getStatusColor(deliverable.status)} text-sm"
				title="{deliverable.label} - {deliverable.status}"
			>
				{getStatusIcon(deliverable.status)}
			</span>
		{/each}
		{#if deliverables.length === 0}
			<span class="text-xs text-muted-foreground italic">None defined</span>
		{/if}
	</div>
{:else}
	<!-- L2: Full view - labels, artifact links, override reasons -->
	<div class="deliverables-full space-y-2" data-testid="deliverables-full">
		<div class="text-xs font-semibold uppercase text-foreground/80 mb-2">
			Deliverables ({stats.complete}/{stats.total})
		</div>

		{#if deliverables.length === 0}
			<div class="text-xs text-muted-foreground italic">No deliverables defined for this issue type</div>
		{:else}
			<ul class="space-y-1">
				{#each deliverables as deliverable}
					<li class="flex items-start gap-2 text-xs">
						<!-- Status icon -->
						<span class="{getStatusColor(deliverable.status)} shrink-0 mt-0.5">
							{getStatusIcon(deliverable.status)}
						</span>

						<!-- Label and details -->
						<div class="flex-1 min-w-0">
							<span
								class="text-foreground"
								class:line-through={deliverable.status === 'skipped'}
							>
								{deliverable.label}
							</span>

							<!-- Artifact link (if provided) -->
							{#if deliverable.artifact_link}
								<a
									href={deliverable.artifact_link}
									class="ml-2 text-blue-500 hover:underline"
									target="_blank"
									rel="noopener noreferrer"
								>
									[view]
								</a>
							{/if}

							<!-- Override reason (if skipped) -->
							{#if deliverable.status === 'skipped' && deliverable.override_reason}
								<div class="mt-0.5 text-muted-foreground italic text-xs">
									Skipped: {deliverable.override_reason}
								</div>
							{/if}
						</div>
					</li>
				{/each}
			</ul>

			<!-- Summary stats -->
			<div class="mt-3 pt-2 border-t border-border text-xs text-muted-foreground">
				<div class="flex items-center justify-between">
					<span>Completion:</span>
					<span class="font-medium text-foreground">{stats.percentage}%</span>
				</div>
				{#if stats.skipped > 0}
					<div class="flex items-center justify-between mt-1">
						<span>Skipped:</span>
						<span class="text-yellow-500">{stats.skipped}</span>
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}
