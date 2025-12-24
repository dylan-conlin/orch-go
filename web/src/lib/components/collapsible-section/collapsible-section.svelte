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
</script>

<div class="rounded-lg border {getVariantStyles(variant)}">
	<button
		class="flex w-full items-center justify-between px-3 py-2 text-left hover:bg-accent/50 transition-colors"
		onclick={toggle}
		aria-expanded={expanded}
		data-testid="section-toggle-{variant}"
	>
		<div class="flex items-center gap-2">
			<span class="text-sm">{icon}</span>
			<span class="text-sm font-medium">{title}</span>
			<Badge variant={getBadgeVariant(variant)} class="h-5 px-1.5 text-xs">
				{agents.length}
			</Badge>
		</div>
		<span class="text-muted-foreground transition-transform {expanded ? 'rotate-180' : ''}">
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
