<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { fetchArtifactContent, type ArtifactFeedItem } from '$lib/stores/kb-artifacts';
	import { orchestratorContext } from '$lib/stores/context';
	import { MarkdownContent } from '$lib/components/markdown-content';

	export let artifact: ArtifactFeedItem;

	const dispatch = createEventDispatcher();
	let content = '';
	let loading = true;
	let error: string | null = null;

	// React to artifact prop changes - fetch full content from API
	$: if (artifact) {
		loadArtifactContent(artifact);
	}

	async function loadArtifactContent(artifact: ArtifactFeedItem) {
		loading = true;
		error = null;

		try {
			const projectDir = $orchestratorContext?.project_dir;
			const response = await fetchArtifactContent(artifact.path, projectDir);

			if (response.error) {
				error = response.error;
				// Fall back to metadata-only view
				content = generateFallbackMarkdown(artifact);
			} else {
				content = response.content;
			}
		} catch (e) {
			error = String(e);
			content = generateFallbackMarkdown(artifact);
		} finally {
			loading = false;
		}
	}

	// Generate fallback content when API fails
	function generateFallbackMarkdown(artifact: ArtifactFeedItem): string {
		let md = `# ${artifact.title}\n\n`;

		md += `**Type:** ${artifact.type}\n\n`;

		if (artifact.status) {
			md += `**Status:** ${artifact.status}\n\n`;
		}

		if (artifact.date) {
			md += `**Date:** ${artifact.date}\n\n`;
		}

		md += `**Last Modified:** ${artifact.relative_time}\n\n`;

		md += `**Path:** \`${artifact.path}\`\n\n`;

		if (artifact.recommendation) {
			md += `> ⚠️ This investigation has a recommendation\n\n`;
		}

		if (artifact.summary) {
			md += `## Summary\n\n${artifact.summary}\n\n`;
		}

		md += `---\n\n`;
		md += `_Could not load full content. Open file directly: \`${artifact.path}\`_\n`;

		return md;
	}

	function handleClose() {
		dispatch('close');
	}
</script>

<div
	class="fixed top-0 right-0 h-screen w-1/2 bg-background border-l border-border shadow-lg z-50 flex flex-col"
	role="dialog"
	aria-modal="true"
>
	<!-- Header -->
	<div class="border-b border-border px-6 py-4 flex items-center justify-between">
		<h2 class="text-lg font-semibold text-foreground">Artifact Details</h2>
		<button
			on:click={handleClose}
			class="text-muted-foreground hover:text-foreground transition-colors"
		>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				width="20"
				height="20"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<line x1="18" y1="6" x2="6" y2="18" />
				<line x1="6" y1="6" x2="18" y2="18" />
			</svg>
		</button>
	</div>

	<!-- Content -->
	<div class="flex-1 overflow-auto px-6 py-4">
		{#if loading}
			<div class="flex items-center justify-center h-full">
				<div class="text-muted-foreground">Loading...</div>
			</div>
		{:else if error}
			<div class="text-red-500">Error: {error}</div>
		{:else}
			<MarkdownContent content={content} />
		{/if}
	</div>

	<!-- Footer -->
	<div class="border-t border-border px-6 py-3 text-xs text-muted-foreground">
		Press <kbd class="px-1 py-0.5 bg-muted rounded">h</kbd> or
		<kbd class="px-1 py-0.5 bg-muted rounded">Esc</kbd> to close
	</div>
</div>
