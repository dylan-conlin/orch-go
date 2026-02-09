<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { ArtifactFeedItem } from '$lib/stores/kb-artifacts';

	export let artifact: ArtifactFeedItem;
	export let selected = false;

	const dispatch = createEventDispatcher();

	let copiedPath = false;
	let copiedTimeout: ReturnType<typeof setTimeout> | null = null;

	async function copyPath(event: MouseEvent) {
		event.stopPropagation();
		try {
			await navigator.clipboard.writeText(artifact.path);
			copiedPath = true;
			if (copiedTimeout) clearTimeout(copiedTimeout);
			copiedTimeout = setTimeout(() => { copiedPath = false; }, 1500);
		} catch (err) {
			console.error('Failed to copy path:', err);
		}
	}

	function getTypeIcon(type: string): string {
		switch (type) {
			case 'investigation':
				return '🔍';
			case 'decision':
				return '⚖️';
			case 'model':
				return '📊';
			case 'guide':
				return '📖';
			case 'principle':
				return '⭐';
			default:
				return '📄';
		}
	}

	function getStatusBadgeClass(status: string): string {
		switch (status.toLowerCase()) {
			case 'active':
				return 'bg-blue-500/10 text-blue-500 border-blue-500/20';
			case 'complete':
				return 'bg-green-500/10 text-green-500 border-green-500/20';
			case 'proposed':
				return 'bg-yellow-500/10 text-yellow-500 border-yellow-500/20';
			case 'accepted':
				return 'bg-green-500/10 text-green-500 border-green-500/20';
			case 'superseded':
				return 'bg-gray-500/10 text-gray-500 border-gray-500/20';
			default:
				return 'bg-muted text-muted-foreground border-border';
		}
	}
</script>

<button
	class="w-full text-left px-4 py-3 rounded-md border border-border hover:border-accent transition-colors {selected
		? 'bg-accent/10 border-accent'
		: 'bg-background'}"
	on:click={() => dispatch('click')}
>
	<div class="flex items-start gap-3">
		<span class="text-lg">{getTypeIcon(artifact.type)}</span>
		<div class="flex-1 min-w-0">
			<div class="flex items-center gap-2 mb-1">
				<span class="text-sm font-medium text-foreground truncate">{artifact.title}</span>
				{#if artifact.status}
					<span
						class="text-xs px-2 py-0.5 rounded border {getStatusBadgeClass(artifact.status)}"
					>
						{artifact.status}
					</span>
				{/if}
				{#if artifact.recommendation}
					<span class="text-xs text-blue-500">Has recommendation</span>
				{/if}
			</div>
			{#if artifact.summary}
				<p class="text-xs text-muted-foreground line-clamp-2">{artifact.summary}</p>
			{/if}
			<div class="flex items-center gap-2 mt-2 text-xs text-muted-foreground">
				{#if artifact.date}
					<span>{artifact.date}</span>
					<span>·</span>
				{/if}
				<span>{artifact.relative_time}</span>
				<span>·</span>
				<span
					class="font-mono truncate max-w-[200px] cursor-pointer hover:text-foreground transition-colors {copiedPath ? 'text-green-500' : ''}"
					on:click={copyPath}
					on:keydown={(e) => { if (e.key === 'Enter') copyPath(e); }}
					role="button"
					tabindex="-1"
					title="Click to copy full path: {artifact.path}"
				>
					{copiedPath ? 'Copied!' : artifact.path}
				</span>
			</div>
		</div>
	</div>
</button>

<style>
	.line-clamp-2 {
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
</style>
