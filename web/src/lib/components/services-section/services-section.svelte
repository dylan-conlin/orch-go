<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { ServiceCard } from '$lib/components/service-card';
	import { services } from '$lib/stores/services';

	// Bind to allow parent to control expansion
	export let expanded = true;
</script>

{#if $services.total_count > 0}
	<div class="rounded-lg border-2 border-blue-500/30 bg-card" data-testid="services-section">
		<button
			class="flex w-full items-center justify-between px-3 py-2 text-left hover:bg-accent/50 transition-colors border-b border-blue-500/20"
			onclick={() => { expanded = !expanded; }}
			aria-expanded={expanded}
			data-testid="services-toggle"
		>
			<div class="flex items-center gap-2">
				<span class="text-sm">S</span>
				<span class="text-sm font-medium text-blue-400">Services</span>
				<Badge variant="secondary" class="h-5 px-1.5 text-xs bg-blue-600/50">
					{$services.total_count}
				</Badge>
				<Badge variant="secondary" class="h-5 px-1.5 text-xs bg-green-600/50">
					{$services.running_count} running
				</Badge>
				{#if $services.stopped_count > 0}
					<Badge variant="secondary" class="h-5 px-1.5 text-xs bg-red-600/50">
						{$services.stopped_count} stopped
					</Badge>
				{/if}
			</div>
			<span class="text-muted-foreground transition-transform {expanded ? 'rotate-180' : ''}">
				<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="6 9 12 15 18 9"></polyline>
				</svg>
			</span>
		</button>
		{#if expanded}
			<div class="p-2">
				<div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
					{#each $services.services as service (service.name)}
						<ServiceCard {service} project={$services.project} />
					{/each}
				</div>
			</div>
		{/if}
	</div>
{/if}
