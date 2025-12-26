<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { connectionStatus } from '$lib/stores/agents';
	import { usage } from '$lib/stores/usage';
	import { theme } from '$lib/stores/theme';
	import { ThemeToggle } from '$lib/components/theme-toggle';
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();

	function getUsageColor(percent: number): 'green' | 'yellow' | 'red' {
		if (percent < 60) return 'green';
		if (percent < 80) return 'yellow';
		return 'red';
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
	});
</script>

<div class="min-h-screen bg-background">
	<!-- Compact Header -->
	<header class="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
		<div class="container flex h-10 items-center">
			<div class="flex items-center gap-2">
				<a href="/" class="flex items-center gap-1.5">
					<span class="text-base">🐝</span>
					<span class="text-sm font-semibold">Swarm</span>
				</a>
			</div>
			<div class="flex flex-1 items-center justify-end gap-3">
				{#if $usage && !$usage.error}
					<div class="flex items-center gap-2 text-xs" title="Claude Max usage limits">
						<span
							class="font-medium cursor-help"
							class:text-green-600={getUsageColor($usage.five_hour_percent) === 'green'}
							class:text-yellow-600={getUsageColor($usage.five_hour_percent) === 'yellow'}
							class:text-red-600={getUsageColor($usage.five_hour_percent) === 'red'}
							title="5-hour rolling usage: {$usage.five_hour_percent.toFixed(0)}% of limit{$usage.five_hour_reset ? ` • Resets in ${$usage.five_hour_reset}` : ''}"
						>
							{$usage.five_hour_percent.toFixed(0)}%{#if $usage.five_hour_reset} <span class="text-muted-foreground font-normal">({$usage.five_hour_reset})</span>{/if}
						</span>
						<span class="text-muted-foreground">|</span>
						<span
							class="font-medium cursor-help"
							class:text-green-600={getUsageColor($usage.weekly_percent) === 'green'}
							class:text-yellow-600={getUsageColor($usage.weekly_percent) === 'yellow'}
							class:text-red-600={getUsageColor($usage.weekly_percent) === 'red'}
							title="Weekly usage: {$usage.weekly_percent.toFixed(0)}% of limit{$usage.weekly_reset ? ` • Resets in ${$usage.weekly_reset}` : ''}"
						>
							{$usage.weekly_percent.toFixed(0)}%{#if $usage.weekly_reset} <span class="text-muted-foreground font-normal">({$usage.weekly_reset})</span>{/if}
						</span>
						{#if $usage.account_name || $usage.account}
							<span class="text-muted-foreground" title="Active Claude Max account">@{$usage.account_name || $usage.account.split('@')[0]}</span>
						{/if}
					</div>
				{/if}
				<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
					<span class={`h-1.5 w-1.5 rounded-full ${statusColor}`}></span>
					{$connectionStatus}
				</div>
				<ThemeToggle />
			</div>
		</div>
	</header>

	<!-- Main content with reduced padding -->
	<main class="container py-3">
		{@render children()}
	</main>
</div>
