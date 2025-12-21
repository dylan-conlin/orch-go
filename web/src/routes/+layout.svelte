<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { connectionStatus } from '$lib/stores/agents';
	import { theme } from '$lib/stores/theme';
	import { ThemeToggle } from '$lib/components/theme-toggle';
	import type { Snippet } from 'svelte';

	let { children }: { children: Snippet } = $props();

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
