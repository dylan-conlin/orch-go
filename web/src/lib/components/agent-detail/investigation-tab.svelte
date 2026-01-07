<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import type { Agent } from '$lib/stores/agents';

	// Props
	interface Props {
		agent: Agent;
	}

	let { agent }: Props = $props();

	// State for file content fetching
	let fileContent = $state<string | null>(null);
	let fileError = $state<string | null>(null);
	let isLoading = $state(false);

	// Track which items were recently copied
	let copiedItem: string | null = $state(null);
	let copyTimeout: ReturnType<typeof setTimeout> | null = null;

	// Copy to clipboard helper with visual feedback
	async function copyToClipboard(text: string, label: string) {
		try {
			await navigator.clipboard.writeText(text);
			copiedItem = label;
			if (copyTimeout) clearTimeout(copyTimeout);
			copyTimeout = setTimeout(() => {
				copiedItem = null;
			}, 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
		}
	}

	// Fetch investigation file content
	async function fetchInvestigationFile(path: string) {
		isLoading = true;
		fileError = null;
		
		try {
			const response = await fetch(`https://localhost:3348/api/file?path=${encodeURIComponent(path)}`);
			const data = await response.json();
			
			if (data.error) {
				fileError = data.error;
				fileContent = null;
			} else {
				fileContent = data.content;
			}
		} catch (err) {
			fileError = err instanceof Error ? err.message : 'Failed to fetch file';
			fileContent = null;
		} finally {
			isLoading = false;
		}
	}

	// Get project directory or fallback
	const projectDir = $derived(agent.project_dir || '(project directory)');
	
	// Construct workspace path
	const workspacePath = $derived(`${projectDir}/.orch/workspace/${agent.id}`);
	
	// Get investigation path if available
	const investigationPath = $derived(agent.investigation_path);
	
	// Generate terminal commands
	const commands = $derived({
		openWorkspace: `cd "${workspacePath}"`,
		listFiles: `ls -la "${workspacePath}"`,
		viewSynthesis: `cat "${workspacePath}/SYNTHESIS.md"`,
		viewContext: `cat "${workspacePath}/SPAWN_CONTEXT.md"`,
		viewInvestigation: investigationPath ? `cat "${investigationPath}"` : null
	});

	// Auto-fetch when investigation_path changes
	$effect(() => {
		if (investigationPath) {
			fetchInvestigationFile(investigationPath);
		} else {
			fileContent = null;
			fileError = null;
		}
	});

	// Reset state when agent changes
	$effect(() => {
		if (agent.id) {
			copiedItem = null;
		}
	});

	// Extract filename from path for display
	function getFilename(path: string): string {
		const parts = path.split('/');
		return parts[parts.length - 1] || path;
	}
</script>

<div class="p-4">
	<!-- Header -->
	<div class="mb-4 flex items-center justify-between">
		<h3 class="text-sm font-medium text-muted-foreground">Investigation</h3>
		{#if agent.status === 'abandoned'}
			<Badge variant="destructive">Abandoned</Badge>
		{/if}
	</div>

	<!-- Investigation File Content (if available) -->
	{#if investigationPath}
		<div class="mb-4">
			<div class="flex items-center justify-between mb-2">
				<h4 class="text-xs font-medium uppercase tracking-wide text-muted-foreground">Investigation File</h4>
				<button
					type="button"
					class="text-xs text-muted-foreground hover:text-foreground transition-colors"
					onclick={() => copyToClipboard(investigationPath, 'path')}
				>
					{copiedItem === 'path' ? '✓ Copied' : '📋 Copy path'}
				</button>
			</div>
			
			<p class="text-xs text-muted-foreground mb-2 truncate" title={investigationPath}>
				{getFilename(investigationPath)}
			</p>

			{#if isLoading}
				<div class="rounded-lg border bg-muted/30 p-4">
					<p class="text-sm text-muted-foreground animate-pulse">Loading investigation file...</p>
				</div>
			{:else if fileError}
				<div class="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
					<p class="text-sm text-destructive">{fileError}</p>
					<button
						type="button"
						class="mt-2 text-xs text-muted-foreground hover:text-foreground underline"
						onclick={() => fetchInvestigationFile(investigationPath)}
					>
						Retry
					</button>
				</div>
			{:else if fileContent}
				<div class="rounded-lg border bg-muted/20 overflow-hidden">
					<div class="max-h-[400px] overflow-y-auto p-3">
						<pre class="text-xs leading-relaxed whitespace-pre-wrap font-mono text-foreground/90">{fileContent}</pre>
					</div>
				</div>
			{/if}
		</div>
	{:else}
		<!-- No Investigation File Message -->
		<div class="mb-4 rounded-lg border border-dashed bg-muted/20 p-4 text-center">
			<span class="text-lg mb-2 block">🔬</span>
			<p class="text-sm text-muted-foreground">No investigation file reported</p>
			<p class="text-xs text-muted-foreground/70 mt-1">
				Agent has not reported an investigation_path via beads comment
			</p>
		</div>
	{/if}

	<!-- Workspace Path Card -->
	<div class="mb-4">
		<h4 class="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Workspace</h4>
		<button
			type="button"
			class="group flex w-full items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-left transition-all hover:bg-muted hover:border-primary/50 active:scale-[0.99]"
			onclick={() => copyToClipboard(workspacePath, 'workspace')}
		>
			<div class="flex-1 min-w-0">
				<p class="truncate font-mono text-xs">{workspacePath}</p>
			</div>
			<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors shrink-0">
				{copiedItem === 'workspace' ? '✓ Copied' : '📋 Copy'}
			</span>
		</button>
	</div>

	<!-- Terminal Commands Section -->
	<div class="mt-6">
		<h4 class="mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">Terminal Commands</h4>
		<div class="space-y-2">
			<!-- View Investigation (if available) -->
			{#if commands.viewInvestigation}
				<button
					type="button"
					class="group flex w-full items-center gap-3 rounded-lg border bg-blue-500/10 px-3 py-2 text-left transition-all hover:bg-blue-500/20 hover:border-blue-500/30 active:scale-[0.99]"
					onclick={() => copyToClipboard(commands.viewInvestigation!, 'cmd-investigation')}
				>
					<span class="text-lg shrink-0">🔬</span>
					<div class="flex-1 min-w-0">
						<p class="text-xs font-medium text-foreground">View Investigation</p>
						<code class="text-[10px] text-muted-foreground truncate block">{commands.viewInvestigation}</code>
					</div>
					<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors shrink-0">
						{copiedItem === 'cmd-investigation' ? '✓' : '📋'}
					</span>
				</button>
			{/if}

			<!-- Open Workspace -->
			<button
				type="button"
				class="group flex w-full items-center gap-3 rounded-lg border bg-muted/20 px-3 py-2 text-left transition-all hover:bg-muted/40 hover:border-primary/30 active:scale-[0.99]"
				onclick={() => copyToClipboard(commands.openWorkspace, 'cmd-cd')}
			>
				<span class="text-lg shrink-0">📂</span>
				<div class="flex-1 min-w-0">
					<p class="text-xs font-medium text-foreground">Open Workspace</p>
					<code class="text-[10px] text-muted-foreground truncate block">{commands.openWorkspace}</code>
				</div>
				<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors shrink-0">
					{copiedItem === 'cmd-cd' ? '✓' : '📋'}
				</span>
			</button>

			<!-- List Files -->
			<button
				type="button"
				class="group flex w-full items-center gap-3 rounded-lg border bg-muted/20 px-3 py-2 text-left transition-all hover:bg-muted/40 hover:border-primary/30 active:scale-[0.99]"
				onclick={() => copyToClipboard(commands.listFiles, 'cmd-ls')}
			>
				<span class="text-lg shrink-0">📋</span>
				<div class="flex-1 min-w-0">
					<p class="text-xs font-medium text-foreground">List Files</p>
					<code class="text-[10px] text-muted-foreground truncate block">{commands.listFiles}</code>
				</div>
				<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors shrink-0">
					{copiedItem === 'cmd-ls' ? '✓' : '📋'}
				</span>
			</button>

			<!-- View Synthesis -->
			<button
				type="button"
				class="group flex w-full items-center gap-3 rounded-lg border bg-muted/20 px-3 py-2 text-left transition-all hover:bg-muted/40 hover:border-primary/30 active:scale-[0.99]"
				onclick={() => copyToClipboard(commands.viewSynthesis, 'cmd-synth')}
			>
				<span class="text-lg shrink-0">📄</span>
				<div class="flex-1 min-w-0">
					<p class="text-xs font-medium text-foreground">View Synthesis</p>
					<code class="text-[10px] text-muted-foreground truncate block">{commands.viewSynthesis}</code>
				</div>
				<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors shrink-0">
					{copiedItem === 'cmd-synth' ? '✓' : '📋'}
				</span>
			</button>

			<!-- View Context -->
			<button
				type="button"
				class="group flex w-full items-center gap-3 rounded-lg border bg-muted/20 px-3 py-2 text-left transition-all hover:bg-muted/40 hover:border-primary/30 active:scale-[0.99]"
				onclick={() => copyToClipboard(commands.viewContext, 'cmd-ctx')}
			>
				<span class="text-lg shrink-0">📝</span>
				<div class="flex-1 min-w-0">
					<p class="text-xs font-medium text-foreground">View Spawn Context</p>
					<code class="text-[10px] text-muted-foreground truncate block">{commands.viewContext}</code>
				</div>
				<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors shrink-0">
					{copiedItem === 'cmd-ctx' ? '✓' : '📋'}
				</span>
			</button>
		</div>
	</div>

	<!-- Empty State for agents without workspace info -->
	{#if !agent.project_dir && !agent.id}
		<div class="mt-4 rounded-lg border border-dashed bg-muted/20 p-6 text-center">
			<p class="text-sm text-muted-foreground">No workspace information available for this agent.</p>
		</div>
	{/if}
</div>
