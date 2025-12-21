<script lang="ts">
	import '../app.css';
	import { connectionStatus } from '$lib/stores/agents';
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
</script>

<div class="min-h-screen bg-background">
	<!-- Header -->
	<header class="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
		<div class="container flex h-14 items-center">
			<div class="flex items-center space-x-4">
				<a href="/" class="flex items-center space-x-2">
					<span class="text-xl">🐝</span>
					<span class="font-bold">Swarm Dashboard</span>
				</a>
			</div>
			<div class="flex flex-1 items-center justify-end space-x-4">
				<div class="flex items-center space-x-2 text-sm text-muted-foreground">
					<span class="flex items-center gap-2">
						<span class={`h-2 w-2 rounded-full ${statusColor}`}></span>
						{$connectionStatus}
					</span>
				</div>
			</div>
		</div>
	</header>

	<!-- Main content -->
	<main class="container py-6">
		{@render children()}
	</main>
</div>
