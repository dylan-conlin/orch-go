<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { MarkdownContent } from '$lib/components/markdown-content';
	import type { Agent } from '$lib/stores/agents';

	// Props
	interface Props {
		agent: Agent;
	}

	let { agent }: Props = $props();

	// Track copy state
	let copiedPath = $state(false);

	// Copy to clipboard helper
	async function copyPath() {
		if (!agent.investigation_path) return;
		try {
			await navigator.clipboard.writeText(agent.investigation_path);
			copiedPath = true;
			setTimeout(() => copiedPath = false, 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
		}
	}

	// Get investigation content from API
	const investigationContent = $derived(agent.investigation_content);

	// Extract filename from path for display
	function getFilename(path: string): string {
		const parts = path.split('/');
		return parts[parts.length - 1] || path;
	}
</script>

<div>
	{#if agent.investigation_path && investigationContent}
		<!-- Header with file name and copy button -->
		<div class="flex items-center justify-between mb-3">
			<div class="flex items-center gap-2 min-w-0">
				<p class="text-xs text-muted-foreground truncate" title={agent.investigation_path}>
					{getFilename(agent.investigation_path)}
				</p>
				{#if agent.status === 'abandoned'}
					<Badge variant="destructive" class="shrink-0">Abandoned</Badge>
				{/if}
			</div>
			<button
				type="button"
				class="text-xs text-muted-foreground hover:text-foreground transition-colors shrink-0"
				onclick={copyPath}
			>
				{copiedPath ? '✓ Copied' : '📋 Copy path'}
			</button>
		</div>

		<!-- Content -->
		<MarkdownContent content={investigationContent} />
	{:else if agent.investigation_path}
		<!-- Path exists but no content -->
		<div class="rounded-lg border border-dashed bg-muted/20 p-6 text-center">
			<span class="text-2xl mb-2 block">🔬</span>
			<p class="text-sm text-muted-foreground">Investigation file found but content not available</p>
			<p class="text-xs text-muted-foreground/70 mt-2 font-mono">{agent.investigation_path}</p>
		</div>
	{:else}
		<!-- No investigation file -->
		<div class="rounded-lg border border-dashed bg-muted/20 p-6 text-center">
			<span class="text-2xl mb-2 block">🔬</span>
			<p class="text-sm text-muted-foreground">No investigation file reported</p>
			<p class="text-xs text-muted-foreground/70 mt-1">
				Agent has not reported an investigation_path
			</p>
		</div>
	{/if}
</div>
