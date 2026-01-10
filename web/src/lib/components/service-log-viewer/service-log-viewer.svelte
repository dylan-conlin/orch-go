<script lang="ts">
	import { servicelogEvents, type ServiceLogEvent } from '$lib/stores/servicelog';
	import { Badge } from '$lib/components/ui/badge';

	export let serviceName: string;
	export let onClose: () => void;

	// Filter events for this specific service
	$: filteredEvents = $servicelogEvents.filter(
		(e) => e.data?.service_name === serviceName
	).reverse(); // Most recent first

	// Format timestamp
	function formatTimestamp(timestamp: number): string {
		const date = new Date(timestamp * 1000);
		const now = new Date();
		const diffMs = now.getTime() - date.getTime();
		const diffMins = Math.floor(diffMs / 60000);

		if (diffMins < 1) return 'just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		if (diffMins < 1440) return `${Math.floor(diffMins / 60)}h ago`;
		return date.toLocaleDateString();
	}

	// Get event badge variant
	function getEventVariant(type: string): 'default' | 'secondary' | 'destructive' {
		if (type === 'service.crashed') return 'destructive';
		if (type === 'service.restarted') return 'default';
		return 'secondary';
	}

	// Get event icon
	function getEventIcon(type: string): string {
		if (type === 'service.crashed') return '💥';
		if (type === 'service.restarted') return '🔄';
		if (type === 'service.started') return '▶️';
		return '📝';
	}

	// Get event title
	function getEventTitle(event: ServiceLogEvent): string {
		if (event.type === 'service.crashed') {
			return `Service crashed (PID ${event.data?.old_pid} → ${event.data?.new_pid})`;
		}
		if (event.type === 'service.restarted') {
			const auto = event.data?.auto_restart ? 'auto-' : '';
			return `Service ${auto}restarted (restart #${event.data?.restart_count || 1})`;
		}
		if (event.type === 'service.started') {
			return `Service started (PID ${event.data?.pid || event.data?.new_pid})`;
		}
		return event.type;
	}
</script>

<!-- Modal overlay -->
<div
	class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
	on:click={onClose}
	on:keydown={(e) => e.key === 'Escape' && onClose()}
	role="button"
	tabindex="0"
>
	<!-- Modal content -->
	<div
		class="relative w-full max-w-3xl max-h-[80vh] m-4 rounded-lg border-2 border-blue-500/50 bg-card shadow-xl overflow-hidden"
		on:click|stopPropagation
		on:keydown|stopPropagation
		role="dialog"
		tabindex="-1"
	>
		<!-- Header -->
		<div class="flex items-center justify-between border-b border-blue-500/20 bg-card px-4 py-3">
			<div class="flex items-center gap-2">
				<span class="text-lg">📊</span>
				<h2 class="text-lg font-semibold">Service Events: {serviceName}</h2>
				<Badge variant="secondary" class="h-5 px-2 text-xs">
					{filteredEvents.length} event{filteredEvents.length === 1 ? '' : 's'}
				</Badge>
			</div>
			<button
				on:click={onClose}
				class="rounded-lg p-1 hover:bg-accent transition-colors text-xl leading-none"
				aria-label="Close"
			>
				×
			</button>
		</div>

		<!-- Events list -->
		<div class="overflow-y-auto max-h-[calc(80vh-5rem)] p-4">
			{#if filteredEvents.length === 0}
				<div class="text-center text-muted-foreground py-8">
					<p>No events recorded for this service yet.</p>
					<p class="text-sm mt-2">Events will appear here when the service crashes, restarts, or starts.</p>
				</div>
			{:else}
				<div class="space-y-2">
					{#each filteredEvents as event (event.id)}
						<div class="rounded-lg border border-border bg-card/50 p-3 hover:bg-accent/50 transition-colors">
							<div class="flex items-start justify-between gap-2">
								<div class="flex items-start gap-2 flex-1">
									<span class="text-lg mt-0.5">{getEventIcon(event.type)}</span>
									<div class="flex-1 min-w-0">
										<div class="flex items-center gap-2 flex-wrap">
											<Badge variant={getEventVariant(event.type)} class="h-5 px-2 text-xs">
												{event.type.replace('service.', '')}
											</Badge>
											<span class="text-sm text-muted-foreground">
												{formatTimestamp(event.timestamp)}
											</span>
										</div>
										<p class="text-sm mt-1">
											{getEventTitle(event)}
										</p>
										{#if event.data?.project_path}
											<p class="text-xs text-muted-foreground mt-1 font-mono truncate">
												{event.data.project_path}
											</p>
										{/if}
									</div>
								</div>
							</div>
						</div>
					{/each}
				</div>
			{/if}
		</div>

		<!-- Footer -->
		<div class="border-t border-blue-500/20 bg-card px-4 py-3">
			<p class="text-xs text-muted-foreground">
				Events are recorded when services crash, restart, or start. Real-time updates via SSE.
			</p>
		</div>
	</div>
</div>
