<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { MarkdownContent } from '$lib/components/markdown-content';
	import type { Agent } from '$lib/stores/agents';

	// Props
	interface Props {
		agent: Agent;
	}

	let { agent }: Props = $props();

	// Get outcome badge variant based on outcome type
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

	// Get outcome emoji for visual emphasis
	function getOutcomeEmoji(outcome?: string): string {
		switch (outcome?.toLowerCase()) {
			case 'success':
				return '✅';
			case 'partial':
				return '⚠️';
			case 'blocked':
				return '🚫';
			case 'failed':
				return '❌';
			default:
				return '📋';
		}
	}
</script>

<div class="p-4">
	{#if agent.synthesis_content}
		<!-- Header with outcome badge -->
		<div class="flex items-center justify-between mb-3">
			<h3 class="text-sm font-medium text-muted-foreground">Synthesis</h3>
			{#if agent.synthesis?.outcome}
				<Badge variant={getOutcomeVariant(agent.synthesis.outcome)} class="text-xs">
					{getOutcomeEmoji(agent.synthesis.outcome)} {agent.synthesis.outcome}
				</Badge>
			{/if}
		</div>

		<!-- Rendered markdown content -->
		<MarkdownContent content={agent.synthesis_content} />
	{:else if agent.close_reason}
		<!-- Fallback to close reason if no synthesis content -->
<div>
			<h3 class="text-sm font-medium text-muted-foreground mb-3">Close Reason</h3>
			<div class="rounded-lg border bg-muted/30 p-3">
				<p class="text-sm leading-relaxed">{agent.close_reason}</p>
			</div>
		</div>
	{:else}
		<!-- No synthesis data -->
		<div class="rounded-lg border border-dashed bg-muted/20 p-6 text-center">
			<span class="text-2xl mb-2 block">📄</span>
			<p class="text-sm text-muted-foreground">No synthesis data available</p>
			<p class="text-xs text-muted-foreground/70 mt-1">
				SYNTHESIS.md may not have been created yet
			</p>
		</div>
	{/if}
</div>
