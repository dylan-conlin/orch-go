<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import type { Agent } from '$lib/stores/agents';

	export let title: string;
	export let icon: string;
	export let agents: Agent[];
	export let expanded: boolean = false;
	export let variant: 'active' | 'recent' | 'archive' = 'recent';

	function toggle() {
		expanded = !expanded;
	}

	function getVariantStyles(v: typeof variant) {
		switch (v) {
			case 'active':
				return 'border-green-500/30 bg-green-500/5';
			case 'recent':
				return 'border-blue-500/30 bg-blue-500/5';
			case 'archive':
				return 'border-gray-500/30 bg-gray-500/5';
		}
	}

	function getBadgeVariant(v: typeof variant): 'default' | 'secondary' | 'outline' {
		switch (v) {
			case 'active':
				return 'default';
			case 'recent':
				return 'secondary';
			case 'archive':
				return 'outline';
		}
	}

	/**
	 * Get a brief, human-readable task description for an agent
	 */
	function getAgentSummary(agent: Agent): string {
		// For completed agents with TLDR, use first sentence
		if (agent.synthesis?.tldr) {
			const firstSentence = agent.synthesis.tldr.split(/[.!?]/)[0].trim();
			return firstSentence.length > 40 ? firstSentence.substring(0, 40) + '…' : firstSentence;
		}

		// Fallback to close_reason for light-tier completed agents
		if (agent.close_reason) {
			const firstSentence = agent.close_reason.split(/[.!?]/)[0].trim();
			return firstSentence.length > 40 ? firstSentence.substring(0, 40) + '…' : firstSentence;
		}
		
		// Use task description if available
		if (agent.task) {
			return agent.task.length > 40 ? agent.task.substring(0, 40) + '…' : agent.task;
		}
		
		// Fall back to cleaned workspace name
		let cleaned = agent.id
			.replace(/\s*\[[^\]]+\]$/, '') // Remove beads ID suffix
			.replace(/^[a-z]+-/, '') // Remove project prefix
			.replace(/-\d{1,2}[a-z]{3}$/, '') // Remove date suffix
			.replace(/^(feat|fix|inv|debug|research|design)-/, '') // Remove skill prefixes
			.replace(/-/g, ' ')
			.trim();
		
		cleaned = cleaned.charAt(0).toUpperCase() + cleaned.slice(1);
		return cleaned.length > 40 ? cleaned.substring(0, 40) + '…' : cleaned;
	}

	/**
	 * Get preview text for collapsed section header
	 * Shows first 1-2 agent tasks when collapsed
	 */
	function getCollapsedPreview(agents: Agent[]): string {
		if (agents.length === 0) return '';
		
		const summaries = agents.slice(0, 2).map(getAgentSummary);
		
		if (agents.length <= 2) {
			return summaries.join(', ');
		}
		
		return `${summaries.join(', ')} +${agents.length - 2}`;
	}

	$: collapsedPreview = getCollapsedPreview(agents);
</script>

<div class="rounded-lg border {getVariantStyles(variant)}">
	<button
		class="flex w-full items-center justify-between px-3 py-2 text-left hover:bg-accent/50 transition-colors"
		onclick={toggle}
		aria-expanded={expanded}
		data-testid="section-toggle-{variant}"
	>
		<div class="flex items-center gap-2 min-w-0 flex-1">
			<span class="text-sm flex-shrink-0">{icon}</span>
			<span class="text-sm font-medium flex-shrink-0">{title}</span>
			<Badge variant={getBadgeVariant(variant)} class="h-5 px-1.5 text-xs flex-shrink-0">
				{agents.length}
			</Badge>
			{#if !expanded && collapsedPreview && agents.length > 0}
				<span class="text-xs text-muted-foreground truncate" data-testid="section-preview-{variant}">
					— {collapsedPreview}
				</span>
			{/if}
		</div>
		<span class="text-muted-foreground transition-transform flex-shrink-0 {expanded ? 'rotate-180' : ''}">
			<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<polyline points="6 9 12 15 18 9"></polyline>
			</svg>
		</span>
	</button>

	{#if expanded && agents.length > 0}
		<div class="border-t px-2 pb-2 pt-2" data-testid="section-content-{variant}">
			<slot />
		</div>
	{:else if expanded && agents.length === 0}
		<div class="border-t px-3 py-4 text-center text-sm text-muted-foreground" data-testid="section-empty-{variant}">
			No {title.toLowerCase()} agents
		</div>
	{/if}
</div>
