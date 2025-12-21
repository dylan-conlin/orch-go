<script lang="ts">
	import type { Synthesis } from '$lib/stores/agents';
	import { Badge } from '$lib/components/ui/badge';

	export let synthesis: Synthesis;

	function getOutcomeVariant(outcome?: string): 'default' | 'secondary' | 'destructive' | 'outline' {
		switch (outcome?.toLowerCase()) {
			case 'success':
				return 'default';
			case 'partial':
				return 'secondary';
			case 'blocked':
			case 'failed':
				return 'destructive';
			default:
				return 'outline';
		}
	}

	function getRecommendationIcon(rec?: string): string {
		switch (rec?.toLowerCase()) {
			case 'close':
				return 'check';
			case 'continue':
			case 'spawn-follow-up':
				return 'arrow-right';
			case 'escalate':
				return 'alert';
			case 'resume':
				return 'play';
			default:
				return '';
		}
	}
</script>

<div class="mt-3 space-y-2 rounded-md border border-muted bg-muted/30 p-3 text-xs">
	<!-- TLDR (always shown if available) -->
	{#if synthesis.tldr}
		<p class="text-sm leading-snug text-foreground">
			{synthesis.tldr.length > 120 ? synthesis.tldr.slice(0, 117) + '...' : synthesis.tldr}
		</p>
	{/if}

	<!-- Outcome and Recommendation row -->
	{#if synthesis.outcome || synthesis.recommendation}
		<div class="flex items-center gap-2">
			{#if synthesis.outcome}
				<Badge variant={getOutcomeVariant(synthesis.outcome)} class="text-xs">
					{synthesis.outcome}
				</Badge>
			{/if}
			{#if synthesis.recommendation}
				<span class="text-muted-foreground">
					rec: <span class="font-medium text-foreground">{synthesis.recommendation}</span>
				</span>
			{/if}
		</div>
	{/if}

	<!-- Delta summary -->
	{#if synthesis.delta_summary}
		<div class="text-muted-foreground">
			<span class="font-medium">Delta:</span> {synthesis.delta_summary}
		</div>
	{/if}

	<!-- Next Actions (show at most 2 to keep condensed) -->
	{#if synthesis.next_actions && synthesis.next_actions.length > 0}
		<div class="space-y-1">
			<span class="font-medium text-muted-foreground">Next:</span>
			<ul class="ml-3 space-y-0.5">
				{#each synthesis.next_actions.slice(0, 2) as action}
					<li class="text-muted-foreground">
						{action.length > 60 ? action.slice(0, 57) + '...' : action}
					</li>
				{/each}
				{#if synthesis.next_actions.length > 2}
					<li class="text-muted-foreground/60">
						+{synthesis.next_actions.length - 2} more
					</li>
				{/if}
			</ul>
		</div>
	{/if}
</div>
