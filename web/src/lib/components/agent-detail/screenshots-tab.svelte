<script lang="ts">
	import type { Agent } from '$lib/stores/agents';
	import { onMount } from 'svelte';

	// Props
	interface Props {
		agent: Agent;
	}

	let { agent }: Props = $props();

	// State
	let screenshots = $state<string[]>([]);
	let loading = $state(true);
	let error = $state<string | null>(null);
	let expandedImage = $state<string | null>(null);

	// Fetch screenshots for the agent
	async function fetchScreenshots() {
		if (!agent.project_dir || !agent.id) {
			loading = false;
			return;
		}

		try {
			const response = await fetch(
				`https://localhost:3348/api/screenshots?agent_id=${encodeURIComponent(agent.id)}&project_dir=${encodeURIComponent(agent.project_dir)}`
			);

			if (!response.ok) {
				throw new Error(`HTTP ${response.status}: ${response.statusText}`);
			}

			const data = await response.json();
			if (data.error) {
				throw new Error(data.error);
			}

			screenshots = data.screenshots || [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load screenshots';
			console.error('Failed to fetch screenshots:', err);
		} finally {
			loading = false;
		}
	}

	// Build full path for screenshot image
	function getScreenshotPath(filename: string): string {
		return `https://localhost:3348/api/file?path=${encodeURIComponent(
			`${agent.project_dir}/.orch/workspace/${agent.id}/screenshots/${filename}`
		)}`;
	}

	// Get image source from file API response
	async function getImageSrc(filename: string): Promise<string> {
		const response = await fetch(getScreenshotPath(filename));
		const data = await response.json();
		
		if (data.error) {
			throw new Error(data.error);
		}

		// File API returns base64-encoded content for binary files
		// For now, construct data URL directly (we'll load images via API)
		return getScreenshotPath(filename);
	}

	// Close expanded image on Escape key
	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape' && expandedImage) {
			expandedImage = null;
		}
	}

	// Load screenshots when component mounts
	onMount(() => {
		fetchScreenshots();
		window.addEventListener('keydown', handleKeydown);
		return () => {
			window.removeEventListener('keydown', handleKeydown);
		};
	});
</script>

<div class="p-4">
	{#if loading}
		<!-- Loading state -->
		<div class="flex items-center justify-center p-8">
			<div class="text-sm text-muted-foreground">Loading screenshots...</div>
		</div>
	{:else if error}
		<!-- Error state -->
		<div class="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
			<p class="text-sm text-destructive">Error: {error}</p>
		</div>
	{:else if screenshots.length === 0}
		<!-- Empty state -->
		<div class="rounded-lg border border-dashed bg-muted/20 p-6 text-center">
			<span class="text-2xl mb-2 block">📸</span>
			<p class="text-sm text-muted-foreground">No screenshots available</p>
			<p class="text-xs text-muted-foreground/70 mt-1">
				Screenshots are captured during agent validation and stored in workspace/screenshots/
			</p>
		</div>
	{:else}
		<!-- Screenshots grid -->
		<div class="space-y-3">
			<div class="flex items-center justify-between">
				<h3 class="text-sm font-medium text-muted-foreground">
					Screenshots ({screenshots.length})
				</h3>
			</div>

			<!-- Thumbnail grid -->
			<div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
				{#each screenshots as filename}
					<button
						type="button"
						class="relative aspect-video rounded-lg border bg-muted/30 overflow-hidden hover:border-primary transition-colors cursor-pointer group"
						onclick={() => (expandedImage = filename)}
					>
						<img
							src={getScreenshotPath(filename)}
							alt={filename}
							class="w-full h-full object-cover"
							loading="lazy"
						/>
						<!-- Hover overlay -->
						<div
							class="absolute inset-0 bg-background/80 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center"
						>
							<span class="text-xs font-medium">Click to expand</span>
						</div>
					</button>
				{/each}
			</div>

			<!-- Image filenames -->
			<div class="mt-2 space-y-1">
				{#each screenshots as filename}
					<div class="text-xs text-muted-foreground font-mono truncate">
						{filename}
					</div>
				{/each}
			</div>
		</div>
	{/if}
</div>

<!-- Expanded image modal -->
{#if expandedImage}
	<div
		class="fixed inset-0 z-50 bg-background/95 backdrop-blur-sm flex items-center justify-center p-4"
		onclick={() => (expandedImage = null)}
		role="button"
		tabindex="-1"
	>
		<div class="relative max-w-6xl max-h-[90vh] w-full">
			<!-- Close button -->
			<button
				type="button"
				class="absolute top-4 right-4 z-10 rounded-lg bg-background/80 p-2 text-foreground hover:bg-background transition-colors"
				onclick={(e) => { e.stopPropagation(); expandedImage = null; }}
			>
				<span class="text-xl">×</span>
			</button>

			<!-- Image -->
			<img
				src={getScreenshotPath(expandedImage)}
				alt={expandedImage}
				class="w-full h-full object-contain rounded-lg border shadow-lg"
				onclick={(e) => e.stopPropagation()}
			/>

			<!-- Filename -->
			<div class="mt-3 text-center">
				<p class="text-sm text-muted-foreground font-mono">{expandedImage}</p>
			</div>
		</div>
	</div>
{/if}
