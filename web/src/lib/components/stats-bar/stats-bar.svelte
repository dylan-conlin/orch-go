<script lang="ts">
	import { Button } from '$lib/components/ui/button';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import { SettingsPanel } from '$lib/components/settings-panel';
	import { DaemonConfigPanel } from '$lib/components/daemon-config-panel';
	import {
		activeAgents,
		deadAgents,
		connectionStatus,
		connectSSE,
		disconnectSSE
	} from '$lib/stores/agents';
	import { errorEvents } from '$lib/stores/agentlog';
	import { focus, getDriftEmoji } from '$lib/stores/focus';
	import { servers } from '$lib/stores/servers';
	import { beads, reviewQueue } from '$lib/stores/beads';
	import { daemon, getDaemonEmoji, getDaemonCapacity } from '$lib/stores/daemon';
	import { dashboardMode } from '$lib/stores/dashboard-mode';
	import { filters, orchestratorContext, type TimeFilter } from '$lib/stores/context';

	// Props for section state management (bind from parent)
	let { readyQueueExpanded = $bindable(false), reviewQueueExpanded = $bindable(false) }: { readyQueueExpanded?: boolean; reviewQueueExpanded?: boolean } = $props();

	// Time filter options
	const timeFilterOptions: { value: TimeFilter; label: string }[] = [
		{ value: '12h', label: '12h' },
		{ value: '24h', label: '24h' },
		{ value: '48h', label: '48h' },
		{ value: '7d', label: '7d' },
		{ value: 'all', label: 'All' },
	];

	function handleTimeFilterChange(event: Event) {
		const value = (event.target as HTMLSelectElement).value as TimeFilter;
		filters.setTimeFilter(value);
	}

	function toggleFollowOrchestrator() {
		filters.setFollowOrchestrator(!$filters.followOrchestrator);
	}

	function handleConnectClick() {
		if ($connectionStatus === 'disconnected') {
			connectSSE();
		} else {
			disconnectSSE();
		}
	}

</script>

<!-- Compact Stats Bar with Mode Toggle -->
<div class="flex flex-wrap items-center gap-x-2 sm:gap-x-4 gap-y-1.5 rounded-lg border bg-card px-2 sm:px-4 py-2" data-testid="stats-bar">
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

	<!-- Time Filter Dropdown -->
	<Tooltip.Root>
		<Tooltip.Trigger>
			{#snippet child({ props })}
				<div {...props} class="flex items-center gap-1">
					<span class="text-xs text-muted-foreground">Since:</span>
					<select
						value={$filters.since}
						onchange={handleTimeFilterChange}
						class="h-6 rounded border border-input bg-background px-1.5 text-xs cursor-pointer"
						data-testid="time-filter"
						aria-label="Time filter"
					>
						{#each timeFilterOptions as option}
							<option value={option.value}>{option.label}</option>
						{/each}
					</select>
				</div>
			{/snippet}
		</Tooltip.Trigger>
		<Tooltip.Content>
			<p>Filter agents by time window</p>
			<p class="text-xs text-muted-foreground">Reduces dashboard load time</p>
		</Tooltip.Content>
	</Tooltip.Root>

	<!-- Follow Orchestrator Toggle -->
	<Tooltip.Root>
		<Tooltip.Trigger>
			{#snippet child({ props })}
				<button
					{...props}
					class="flex items-center gap-1.5 px-2 py-1 rounded text-xs transition-colors {$filters.followOrchestrator ? 'bg-blue-500/20 text-blue-400' : 'bg-muted text-muted-foreground hover:text-foreground'}"
					onclick={toggleFollowOrchestrator}
					data-testid="follow-orchestrator-toggle"
				>
					<span>{$filters.followOrchestrator ? '👁️' : '👁️‍🗨️'}</span>
					<span class="hidden sm:inline">{$filters.followOrchestrator ? 'Following' : 'Follow'}</span>
				</button>
			{/snippet}
		</Tooltip.Trigger>
		<Tooltip.Content>
			{#if $filters.followOrchestrator}
				<p>Following orchestrator context</p>
				{#if $orchestratorContext.project}
					<p class="text-xs text-muted-foreground">Project: {$orchestratorContext.project}</p>
				{/if}
				<p class="text-xs text-muted-foreground mt-1">Click to show all projects</p>
			{:else}
				<p>Follow orchestrator</p>
				<p class="text-xs text-muted-foreground">Auto-filter by orchestrator's working directory</p>
			{/if}
		</Tooltip.Content>
	</Tooltip.Root>

	<!-- Secondary indicators group -->
	<div class="flex flex-wrap items-center gap-2 sm:gap-4">
		<!-- Errors indicator -->
		<Tooltip.Root>
			<Tooltip.Trigger>
				{#snippet child({ props })}
					<span {...props} class="inline-flex items-center gap-1 sm:gap-2 cursor-default">
						<span class="text-lg">❌</span>
						<span class="inline-flex items-baseline gap-1">
							<span class="text-xl font-bold" class:text-red-500={$errorEvents.length > 0}>{$errorEvents.length}</span>
							<span class="text-xs text-muted-foreground hidden sm:inline">errors</span>
						</span>
					</span>
				{/snippet}
			</Tooltip.Trigger>
			<Tooltip.Content>
				<p>{$errorEvents.length === 0 ? 'No errors logged' : `${$errorEvents.length} agent error${$errorEvents.length === 1 ? '' : 's'} logged`}</p>
			</Tooltip.Content>
		</Tooltip.Root>

		<!-- Active agents indicator (with dead count if any) -->
		<Tooltip.Root>
			<Tooltip.Trigger>
				{#snippet child({ props })}
					<span {...props} class="inline-flex items-center gap-1 sm:gap-2 cursor-default">
						<span class="text-lg">🟢</span>
						<span class="inline-flex items-baseline gap-1">
							<span class="text-xl font-bold" class:text-green-500={$activeAgents.length > 0}>{$activeAgents.length}</span>
							<span class="text-xs text-muted-foreground hidden sm:inline">active</span>
							{#if $deadAgents.length > 0}
								<span class="text-xs text-red-500 hidden sm:inline">(+{$deadAgents.length} need attention)</span>
							{/if}
						</span>
					</span>
				{/snippet}
			</Tooltip.Trigger>
			<Tooltip.Content>
				<p>{$activeAgents.length === 0 ? 'No active agents' : `${$activeAgents.length} agent${$activeAgents.length === 1 ? '' : 's'} running`}</p>
				{#if $deadAgents.length > 0}
					<p class="text-xs text-red-500 mt-1">{$deadAgents.length} dead agent{$deadAgents.length === 1 ? '' : 's'} - no activity for 3+ min</p>
				{/if}
			</Tooltip.Content>
		</Tooltip.Root>

		<!-- Focus indicator (only in historical mode or when drifting) -->
		{#if $focus?.has_focus && ($dashboardMode === 'historical' || $focus.is_drifting)}
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<span {...props} class="inline-flex items-center gap-1 sm:gap-2 cursor-default" data-testid="focus-indicator">
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
						<span {...props} class="inline-flex items-center gap-1 sm:gap-2 cursor-default" data-testid="servers-indicator">
							<span class="text-lg">{$servers.running_count > 0 ? '🖥️' : '💤'}</span>
							<span class="inline-flex items-baseline gap-1">
								<span class="text-xl font-bold" class:text-green-500={$servers.running_count > 0}>{$servers.running_count}</span>
								<span class="text-xs text-muted-foreground hidden sm:inline">/{$servers.total_count} servers</span>
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
							class="inline-flex items-center gap-1 sm:gap-2 cursor-pointer hover:bg-accent/50 rounded px-1 -mx-1 transition-colors"
							onclick={() => { readyQueueExpanded = !readyQueueExpanded; }}
							data-testid="beads-indicator"
						>
							<span class="text-lg">📋</span>
							<span class="inline-flex items-baseline gap-1">
								<span class="text-xl font-bold" class:text-green-500={$beads.ready_issues > 0}>{$beads.ready_issues}</span>
								<span class="text-xs text-muted-foreground hidden sm:inline">ready</span>
							</span>
							{#if $beads.blocked_issues > 0}
								<span class="text-xs text-red-500 hidden sm:inline">({$beads.blocked_issues} blocked)</span>
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

		<!-- Review queue indicator (clickable to toggle review queue section) -->
		{#if $reviewQueue}
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<button
							{...props}
							class="inline-flex items-center gap-1 sm:gap-2 cursor-pointer hover:bg-accent/50 rounded px-1 -mx-1 transition-colors"
							onclick={() => { reviewQueueExpanded = !reviewQueueExpanded; }}
							data-testid="review-queue-indicator"
						>
							<span class="text-lg">✅</span>
							<span class="inline-flex items-baseline gap-1">
								<span class="text-xl font-bold" class:text-emerald-500={$reviewQueue.count > 0}>{$reviewQueue.count}</span>
								<span class="text-xs text-muted-foreground hidden sm:inline">review</span>
							</span>
						</button>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p>{$reviewQueue.count} completion{$reviewQueue.count === 1 ? '' : 's'} awaiting review</p>
					<p class="text-xs text-muted-foreground mt-1">Click to {reviewQueueExpanded ? 'collapse' : 'expand'} review queue</p>
				</Tooltip.Content>
			</Tooltip.Root>
		{/if}

		<!-- Verification status indicator (uses daemon API completions_since_verification) -->
		{#if $daemon?.verification}
			{@const v = $daemon.verification}
			{@const vCount = v.completions_since_verification}
			{#if vCount > 0 || v.is_paused}
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props })}
						<span {...props} class="inline-flex items-center gap-1 sm:gap-2 cursor-default" data-testid="verification-indicator">
							<span class="text-lg">🛡️</span>
							<span class="inline-flex items-baseline gap-1">
								<span class="text-xl font-bold" class:text-amber-500={v.is_paused}>
									{vCount}
								</span>
								<span class="text-xs hidden sm:inline" class:text-amber-500={v.is_paused} class:text-muted-foreground={!v.is_paused}>to review{#if v.is_paused} (paused){/if}</span>
							</span>
						</span>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p>{vCount} completion{vCount === 1 ? '' : 's'} since last verification (threshold: {v.threshold})</p>
					{#if v.remaining_before_pause > 0}
						<p class="text-xs text-muted-foreground">{v.remaining_before_pause} remaining before daemon pauses</p>
					{:else if v.is_paused}
						<p class="text-xs text-amber-500">Daemon paused — verification needed</p>
					{/if}
					{#if v.last_verification_ago}
						<p class="text-xs text-muted-foreground">Last verified: {v.last_verification_ago} ago</p>
					{/if}
				</Tooltip.Content>
			</Tooltip.Root>
			{/if}
		{/if}

		<!-- Daemon indicator (clickable to show config) -->
		{#if $daemon}
			<DropdownMenu.Root>
				<DropdownMenu.Trigger>
					{#snippet child({ props })}
						<button
							{...props}
							class="inline-flex items-center gap-1 sm:gap-2 cursor-pointer hover:bg-accent/50 rounded px-1 -mx-1 transition-colors"
							data-testid="daemon-indicator"
						>
							<span class="text-lg">{getDaemonEmoji($daemon)}</span>
							<span class="inline-flex items-baseline gap-1">
								{#if $daemon.running}
									<span class="text-xl font-bold" class:text-green-500={$daemon.capacity_free > 0} class:text-red-500={$daemon.capacity_free === 0}>{getDaemonCapacity($daemon)}</span>
									<span class="text-xs text-muted-foreground hidden sm:inline">agents</span>
									{#if $daemon.ready_count > 0}
										<span class="text-xs text-muted-foreground hidden sm:inline">·</span>
										<span class="text-xs font-medium" class:text-amber-500={$daemon.capacity_free === 0} class:text-muted-foreground={$daemon.capacity_free > 0}>{$daemon.ready_count} queued</span>
									{/if}
								{:else}
									<span class="text-xs text-muted-foreground hidden sm:inline">daemon off</span>
								{/if}
							</span>
						</button>
					{/snippet}
				</DropdownMenu.Trigger>
				<DropdownMenu.Content class="w-80" align="end">
					<div class="p-2 space-y-3">
						<!-- Status summary at top -->
						<div class="pb-2 border-b">
							{#if $daemon.running}
								<p class="text-sm font-medium">Daemon {$daemon.status}</p>
								<p class="text-xs text-muted-foreground">
									{$daemon.capacity_used}/{$daemon.capacity_max} agents • {$daemon.ready_count} ready
								</p>
								{#if $daemon.last_poll_ago}
									<p class="text-xs text-muted-foreground">Last poll: {$daemon.last_poll_ago}</p>
								{/if}
							{:else}
								<p class="text-sm font-medium text-amber-500">Daemon not running</p>
								<p class="text-xs text-muted-foreground">Start with: orch daemon run</p>
							{/if}
						</div>
						<!-- Config editing panel -->
						<DaemonConfigPanel />
					</div>
				</DropdownMenu.Content>
			</DropdownMenu.Root>
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
