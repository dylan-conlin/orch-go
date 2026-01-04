<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { SettingsPanel } from '$lib/components/settings-panel';
	import {
		activeAgents,
		connectionStatus,
		connectSSE,
		disconnectSSE
	} from '$lib/stores/agents';
	import { errorEvents } from '$lib/stores/agentlog';
	import { focus, getDriftEmoji } from '$lib/stores/focus';
	import { servers } from '$lib/stores/servers';
	import { beads } from '$lib/stores/beads';
	import { daemon, getDaemonEmoji, getDaemonCapacity } from '$lib/stores/daemon';
	import { dashboardMode } from '$lib/stores/dashboard-mode';

	// Props for section state management (bind:readyQueueExpanded from parent)
	let { readyQueueExpanded = $bindable(false) }: { readyQueueExpanded?: boolean } = $props();

	function handleConnectClick() {
		if ($connectionStatus === 'disconnected') {
			connectSSE();
		} else {
			disconnectSSE();
		}
	}
</script>

<!-- Compact Stats Bar with Mode Toggle -->
<div class="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-lg border bg-card px-4 py-2" data-testid="stats-bar">
	<!-- Mode Toggle -->
	<div class="flex items-center gap-1 rounded-md bg-muted p-0.5" data-testid="mode-toggle">
		<button
			class="px-2 py-1 rounded text-xs font-medium transition-colors {$dashboardMode === 'operational' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground'}"
			onclick={() => dashboardMode.set('operational')}
		>
			⚡ Ops
		</button>
		<button
			class="px-2 py-1 rounded text-xs font-medium transition-colors {$dashboardMode === 'historical' ? 'bg-background shadow-sm text-foreground' : 'text-muted-foreground hover:text-foreground'}"
			onclick={() => dashboardMode.set('historical')}
		>
			📦 History
		</button>
	</div>

	<!-- Secondary indicators group -->
	<div class="flex items-center gap-4">
		<!-- Errors indicator -->
		<Tooltip.Root>
			<Tooltip.Trigger>
				{#snippet child({ props })}
					<span {...props} class="inline-flex items-center gap-2 cursor-default">
						<span class="text-lg">❌</span>
						<span class="inline-flex items-baseline gap-1">
							<span class="text-xl font-bold" class:text-red-500={$errorEvents.length > 0}>{$errorEvents.length}</span>
							<span class="text-xs text-muted-foreground">errors</span>
						</span>
					</span>
				{/snippet}
			</Tooltip.Trigger>
			<Tooltip.Content>
				<p>{$errorEvents.length === 0 ? 'No errors logged' : `${$errorEvents.length} agent error${$errorEvents.length === 1 ? '' : 's'} logged`}</p>
			</Tooltip.Content>
		</Tooltip.Root>

		<!-- Active agents indicator -->
		<Tooltip.Root>
			<Tooltip.Trigger>
				{#snippet child({ props })}
					<span {...props} class="inline-flex items-center gap-2 cursor-default">
						<span class="text-lg">🟢</span>
						<span class="inline-flex items-baseline gap-1">
							<span class="text-xl font-bold" class:text-green-500={$activeAgents.length > 0}>{$activeAgents.length}</span>
							<span class="text-xs text-muted-foreground">active</span>
						</span>
					</span>
				{/snippet}
			</Tooltip.Trigger>
			<Tooltip.Content>
				<p>{$activeAgents.length === 0 ? 'No active agents' : `${$activeAgents.length} agent${$activeAgents.length === 1 ? '' : 's'} running`}</p>
			</Tooltip.Content>
		</Tooltip.Root>

		<!-- Focus indicator (only in historical mode or when drifting) -->
		{#if $focus?.has_focus && ($dashboardMode === 'historical' || $focus.is_drifting)}
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<span {...props} class="inline-flex items-center gap-2 cursor-default" data-testid="focus-indicator">
							<span class="text-lg">{getDriftEmoji($focus)}</span>
							<span class="inline-flex items-baseline gap-1">
								<span class="text-xs truncate max-w-32" class:text-red-500={$focus.is_drifting} class:text-green-500={!$focus.is_drifting}>
									{$focus.is_drifting ? 'drifting' : 'focused'}
								</span>
							</span>
						</span>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p class="font-medium">{$focus.goal || 'Focus set'}</p>
					{#if $focus.is_drifting}
						<p class="text-xs text-muted-foreground mt-1">Current work may not align with focus</p>
					{/if}
				</Tooltip.Content>
			</Tooltip.Root>
		{/if}

		<!-- Servers indicator (only in historical mode) -->
		{#if $servers && $dashboardMode === 'historical'}
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<span {...props} class="inline-flex items-center gap-2 cursor-default" data-testid="servers-indicator">
							<span class="text-lg">{$servers.running_count > 0 ? '🖥️' : '💤'}</span>
							<span class="inline-flex items-baseline gap-1">
								<span class="text-xl font-bold" class:text-green-500={$servers.running_count > 0}>{$servers.running_count}</span>
								<span class="text-xs text-muted-foreground">/{$servers.total_count} servers</span>
							</span>
						</span>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p>{$servers.running_count} running, {$servers.stopped_count} stopped</p>
					<p class="text-xs text-muted-foreground mt-1">Local development servers</p>
				</Tooltip.Content>
			</Tooltip.Root>
		{/if}

		<!-- Beads indicator (clickable to toggle ready queue section) -->
		{#if $beads}
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<button
							{...props}
							class="inline-flex items-center gap-2 cursor-pointer hover:bg-accent/50 rounded px-1 -mx-1 transition-colors"
							onclick={() => { readyQueueExpanded = !readyQueueExpanded; }}
							data-testid="beads-indicator"
						>
							<span class="text-lg">📋</span>
							<span class="inline-flex items-baseline gap-1">
								<span class="text-xl font-bold" class:text-green-500={$beads.ready_issues > 0}>{$beads.ready_issues}</span>
								<span class="text-xs text-muted-foreground">ready</span>
							</span>
							{#if $beads.blocked_issues > 0}
								<span class="text-xs text-red-500">({$beads.blocked_issues} blocked)</span>
							{/if}
						</button>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p>{$beads.ready_issues} ready to work on</p>
					<p class="text-xs text-muted-foreground">{$beads.blocked_issues} blocked • {$beads.open_issues} total open</p>
					<p class="text-xs text-muted-foreground mt-1">Click to {readyQueueExpanded ? 'collapse' : 'expand'} queue</p>
				</Tooltip.Content>
			</Tooltip.Root>
		{/if}

		<!-- Daemon indicator -->
		{#if $daemon}
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<span {...props} class="inline-flex items-center gap-2 cursor-default" data-testid="daemon-indicator">
							<span class="text-lg">{getDaemonEmoji($daemon)}</span>
							<span class="inline-flex items-baseline gap-1">
								{#if $daemon.running}
									<span class="text-xl font-bold" class:text-green-500={$daemon.capacity_free > 0} class:text-red-500={$daemon.capacity_free === 0}>{getDaemonCapacity($daemon)}</span>
									<span class="text-xs text-muted-foreground">slots</span>
								{:else}
									<span class="text-xs text-muted-foreground">daemon</span>
								{/if}
							</span>
						</span>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>
					{#if $daemon.running}
						<p class="font-medium">Daemon {$daemon.status}</p>
						<p class="text-xs text-muted-foreground">
							{$daemon.capacity_used}/{$daemon.capacity_max} agents • {$daemon.ready_count} ready
						</p>
						{#if $daemon.last_poll_ago}
							<p class="text-xs text-muted-foreground">Last poll: {$daemon.last_poll_ago}</p>
						{/if}
						{#if $daemon.last_spawn_ago}
							<p class="text-xs text-muted-foreground">Last spawn: {$daemon.last_spawn_ago}</p>
						{/if}
					{:else}
						<p>Daemon not running</p>
						<p class="text-xs text-muted-foreground">Start with: orch daemon run</p>
					{/if}
				</Tooltip.Content>
			</Tooltip.Root>
		{/if}
	</div>
	<!-- Connection button and settings - pushed to end -->
	<div class="ml-auto flex items-center gap-1">
		<SettingsPanel />
		<Tooltip.Root>
			<Tooltip.Trigger>
				{#snippet child({ props })}
					<Button
						{...props}
						variant={$connectionStatus === 'connected' ? 'destructive' : 'outline'}
						size="sm"
						onclick={handleConnectClick}
						class="h-7 text-xs"
					>
						{#if $connectionStatus === 'connecting'}
							...
						{:else if $connectionStatus === 'connected'}
							Disconnect
						{:else}
							Connect
						{/if}
					</Button>
				{/snippet}
			</Tooltip.Trigger>
			<Tooltip.Content>
				{#if $connectionStatus === 'connected'}
					<p>Disconnect from SSE stream</p>
					<p class="text-xs text-muted-foreground">Stop receiving real-time agent updates</p>
				{:else if $connectionStatus === 'connecting'}
					<p>Connecting to SSE stream...</p>
				{:else}
					<p>Connect to SSE stream</p>
					<p class="text-xs text-muted-foreground">Receive real-time agent updates</p>
				{/if}
			</Tooltip.Content>
		</Tooltip.Root>
	</div>
</div>
