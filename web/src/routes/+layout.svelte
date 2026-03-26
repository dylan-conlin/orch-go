<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { connectionStatus } from '$lib/stores/agents';
	import { usage } from '$lib/stores/usage';
	import { theme, mode, getEffective } from '$lib/stores/theme';
	import { page } from '$app/stores';
	import { ThemeToggle } from '$lib/components/theme-toggle';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();

	function getUsageColor(percent: number | null): 'green' | 'yellow' | 'red' | 'unavailable' {
		if (percent === null) return 'unavailable';
		if (percent < 60) return 'green';
		if (percent < 80) return 'yellow';
		return 'red';
	}

	// Format percent for display - returns "N/A" when null
	function formatPercent(percent: number | null): string {
		if (percent === null) return 'N/A';
		return `${percent.toFixed(0)}%`;
	}

	let statusColor = $derived.by(() => {
		switch ($connectionStatus) {
			case 'connected':
				return 'bg-green-500';
			case 'connecting':
				return 'bg-yellow-500';
			default:
				return 'bg-red-500';
		}
	});

	onMount(() => {
		theme.init();
		// Apply dark class based on mode
		const effectiveMode = getEffective($mode);
		if (effectiveMode === 'dark') {
			document.documentElement.classList.add('dark');
		} else {
			document.documentElement.classList.remove('dark');
		}
	});
</script>

<Tooltip.Provider>
	<div class="min-h-screen bg-background">
		<!-- Compact Header -->
		<header class="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
			<div class="container flex h-10 items-center gap-2">
				<div class="flex items-center gap-2 sm:gap-4">
					<a href="/" class="flex items-center gap-1.5 flex-shrink-0">
						<span class="text-base">🐝</span>
						<span class="text-sm font-semibold hidden sm:inline">Swarm</span>
					</a>
					<nav class="flex items-center gap-0.5 sm:gap-1" aria-label="Main navigation">
						<a
							href="/"
							class="px-1.5 sm:px-2 py-1 text-xs font-medium transition-colors {$page.url.pathname === '/' ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'}"
							aria-current={$page.url.pathname === '/' ? 'page' : undefined}
						><span class="sm:hidden">T</span><span class="hidden sm:inline">Threads</span></a>
						<a
							href="/briefs"
							class="px-1.5 sm:px-2 py-1 text-xs font-medium transition-colors {$page.url.pathname === '/briefs' ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'}"
							aria-current={$page.url.pathname === '/briefs' ? 'page' : undefined}
						><span class="sm:hidden">B</span><span class="hidden sm:inline">Briefs</span></a>
						<a
							href="/knowledge-tree"
							class="px-1.5 sm:px-2 py-1 text-xs font-medium transition-colors {$page.url.pathname === '/knowledge-tree' ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'}"
							aria-current={$page.url.pathname === '/knowledge-tree' ? 'page' : undefined}
						><span class="sm:hidden">KB</span><span class="hidden sm:inline">Knowledge</span></a>
						<a
							href="/work-graph"
							class="px-1.5 sm:px-2 py-1 text-xs font-medium transition-colors {$page.url.pathname === '/work-graph' ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'}"
							aria-current={$page.url.pathname === '/work-graph' ? 'page' : undefined}
						><span class="sm:hidden">Work</span><span class="hidden sm:inline">Work</span></a>
						<a
							href="/harness"
							class="px-1.5 sm:px-2 py-1 text-xs font-medium transition-colors {$page.url.pathname === '/harness' ? 'text-foreground' : 'text-muted-foreground hover:text-foreground'}"
							aria-current={$page.url.pathname === '/harness' ? 'page' : undefined}
						><span class="sm:hidden">H</span><span class="hidden sm:inline">Harness</span></a>
					</nav>
				</div>
				<div class="flex flex-1 items-center justify-end gap-1.5 sm:gap-3 min-w-0">
					{#if $usage && !$usage.error}
						<Tooltip.Root>
							<Tooltip.Trigger>
								{#snippet child({ props })}
									<span {...props} class="inline-flex items-center gap-1 sm:gap-2 text-xs cursor-default">
										<span
											class="font-medium"
											class:text-green-600={getUsageColor($usage.five_hour_percent) === 'green'}
											class:text-yellow-600={getUsageColor($usage.five_hour_percent) === 'yellow'}
											class:text-red-600={getUsageColor($usage.five_hour_percent) === 'red'}
											class:text-muted-foreground={getUsageColor($usage.five_hour_percent) === 'unavailable'}
										>
											{formatPercent($usage.five_hour_percent)}{#if $usage.five_hour_reset}<span class="text-muted-foreground font-normal hidden sm:inline"> ({$usage.five_hour_reset})</span>{/if}
										</span>
										<span class="text-muted-foreground hidden sm:inline">|</span>
										<span
											class="font-medium hidden sm:inline"
											class:text-green-600={getUsageColor($usage.weekly_percent) === 'green'}
											class:text-yellow-600={getUsageColor($usage.weekly_percent) === 'yellow'}
											class:text-red-600={getUsageColor($usage.weekly_percent) === 'red'}
											class:text-muted-foreground={getUsageColor($usage.weekly_percent) === 'unavailable'}
										>
											{formatPercent($usage.weekly_percent)}{#if $usage.weekly_reset}<span class="text-muted-foreground font-normal"> ({$usage.weekly_reset})</span>{/if}
										</span>
										{#if $usage.account_name || $usage.account}
											<span class="text-muted-foreground hidden sm:inline">@{$usage.account_name || $usage.account.split('@')[0]}</span>
										{/if}
									</span>
								{/snippet}
							</Tooltip.Trigger>
							<Tooltip.Content>
								<p class="font-medium">Claude Max Usage</p>
								<p class="text-xs text-muted-foreground mt-1">
									5-hour: {formatPercent($usage.five_hour_percent)}{$usage.five_hour_reset ? ` • Resets in ${$usage.five_hour_reset}` : ''}
								</p>
								<p class="text-xs text-muted-foreground">
									Weekly: {formatPercent($usage.weekly_percent)}{$usage.weekly_reset ? ` • Resets in ${$usage.weekly_reset}` : ''}
								</p>
							</Tooltip.Content>
						</Tooltip.Root>
					{/if}
					<Tooltip.Root>
						<Tooltip.Trigger>
							{#snippet child({ props })}
								<span {...props} class="inline-flex items-center gap-1.5 text-xs text-muted-foreground cursor-default">
									<span class={`h-1.5 w-1.5 rounded-full ${statusColor}`}></span>
									<span class="hidden sm:inline">{$connectionStatus}</span>
								</span>
							{/snippet}
						</Tooltip.Trigger>
						<Tooltip.Content>
							{#if $connectionStatus === 'connected'}
								<p>Connected to SSE stream</p>
								<p class="text-xs text-muted-foreground">Receiving real-time updates</p>
							{:else if $connectionStatus === 'connecting'}
								<p>Connecting to SSE stream...</p>
							{:else}
								<p>Disconnected from SSE stream</p>
								<p class="text-xs text-muted-foreground">Click Connect to resume updates</p>
							{/if}
						</Tooltip.Content>
					</Tooltip.Root>
					<ThemeToggle />
				</div>
			</div>
		</header>

		<!-- Main content with reduced padding -->
		<main class="container py-3">
			{@render children()}
		</main>
	</div>
</Tooltip.Provider>
