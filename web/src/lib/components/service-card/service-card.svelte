<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import type { Service } from '$lib/stores/services';
	import { getStatusColor, getStatusIcon } from '$lib/stores/services';
	import ServiceLogViewer from '$lib/components/service-log-viewer/service-log-viewer.svelte';

	export let service: Service;
	export let project: string;

	let showLogViewer = false;

	function getStatusVariant(status: string, pid: number) {
		if (status === 'running' && pid !== 0) {
			return 'default';
		}
		return 'secondary';
	}

	function getProjectIcon(project: string): string {
		const icons: Record<string, string> = {
			'orch-go': '🎯',
			'orch-knowledge': '📚',
			'skillc': '⚡',
			'beads': '📿',
			'kb-cli': '📖',
			'opencode': '💻',
		};
		return icons[project] || '📁';
	}
</script>

<div
	class="group relative w-full rounded-lg border-2 border-blue-500/50 bg-card p-3 transition-all hover:border-blue-500 hover:shadow-md hover:shadow-blue-500/20"
>
	<!-- Service icon badge -->
	<div class="absolute -right-1 -top-1 flex h-5 w-5 items-center justify-center rounded-full bg-blue-600 text-[10px] shadow-sm">
		<Tooltip.Root>
			<Tooltip.Trigger>
				<span>S</span>
			</Tooltip.Trigger>
			<Tooltip.Content>
				<p class="font-medium">Service</p>
				<p class="text-xs text-muted-foreground">Overmind-managed process</p>
			</Tooltip.Content>
		</Tooltip.Root>
	</div>

	<!-- Header: Project icon + Status -->
	<div class="flex items-center justify-between gap-2">
		<div class="flex items-center gap-2">
			<span class="text-lg">{getProjectIcon(project)}</span>
			<Badge variant={getStatusVariant(service.status, service.pid)} class="h-5 px-2 text-xs {service.status === 'running' && service.pid !== 0 ? 'bg-green-600/80 hover:bg-green-600' : 'bg-red-600/80 hover:bg-red-600'}">
				<span class="mr-1">{getStatusIcon(service.status, service.pid)}</span>
				{service.status}
			</Badge>
		</div>
		<div class="flex items-center gap-1 text-xs text-muted-foreground">
			<Tooltip.Root>
				<Tooltip.Trigger>
					<span class="cursor-default font-mono">{service.uptime}</span>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p>Uptime: {service.uptime}</p>
				</Tooltip.Content>
			</Tooltip.Root>
		</div>
	</div>

	<!-- Service name (main focus) -->
	<div class="mt-2">
		<p class="text-sm font-semibold leading-tight">
			{service.name}
		</p>
	</div>

	<!-- PID + Project info -->
	<div class="mt-2 flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground">
		<Tooltip.Root>
			<Tooltip.Trigger>
				<span class="font-mono cursor-default">PID: {service.pid}</span>
			</Tooltip.Trigger>
			<Tooltip.Content>
				<p class="font-mono text-xs">Process ID: {service.pid}</p>
			</Tooltip.Content>
		</Tooltip.Root>
		<span class="text-muted-foreground/50">|</span>
		<Badge variant="secondary" class="h-4 px-1.5 text-[10px]">
			{project}
		</Badge>
	</div>

	<!-- Restart count + View Events button -->
	<div class="mt-2 border-t border-blue-500/20 pt-2">
		<div class="flex items-center justify-between gap-2">
			{#if service.restart_count > 0}
				<div class="flex items-center gap-2 text-xs">
					<Tooltip.Root>
						<Tooltip.Trigger>
							<span class="flex items-center gap-1 cursor-default text-yellow-400">
								<span class="text-sm">⟳</span>
								{service.restart_count} restart{service.restart_count === 1 ? '' : 's'}
							</span>
						</Tooltip.Trigger>
						<Tooltip.Content>
							<p>Service has restarted {service.restart_count} time{service.restart_count === 1 ? '' : 's'} since monitor started</p>
						</Tooltip.Content>
					</Tooltip.Root>
				</div>
			{:else}
				<div></div>
			{/if}
			<button
				on:click={() => showLogViewer = true}
				class="text-xs text-blue-400 hover:text-blue-300 transition-colors px-2 py-1 rounded hover:bg-accent"
			>
				📊 Events
			</button>
		</div>
	</div>
</div>

<!-- Log viewer modal -->
{#if showLogViewer}
	<ServiceLogViewer serviceName={service.name} onClose={() => showLogViewer = false} />
{/if}
