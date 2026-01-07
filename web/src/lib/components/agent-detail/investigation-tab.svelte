<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import type { Agent } from '$lib/stores/agents';

	// Props
	interface Props {
		agent: Agent;
	}

	let { agent }: Props = $props();

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

	// Derive workspace path from agent id
	// Workspace is at .orch/workspace/{agent.id}/
	$effect(() => {
		// Reset copied state when agent changes
		copiedItem = null;
	});

	// Get project directory or fallback
	const projectDir = $derived(agent.project_dir || '(project directory)');
	
	// Construct workspace path
	const workspacePath = $derived(`${projectDir}/.orch/workspace/${agent.id}`);
	
	// Get primary artifact path if available
	const primaryArtifact = $derived(agent.primary_artifact);
	
	// Generate terminal commands
	const commands = $derived({
		openWorkspace: `cd "${workspacePath}"`,
		listFiles: `ls -la "${workspacePath}"`,
		viewSynthesis: `cat "${workspacePath}/SYNTHESIS.md"`,
		viewContext: `cat "${workspacePath}/SPAWN_CONTEXT.md"`,
		viewArtifact: primaryArtifact ? `cat "${primaryArtifact}"` : null
	});
</script>

<div class="p-4">
	<!-- Header -->
	<div class="mb-4 flex items-center justify-between">
		<h3 class="text-sm font-medium text-muted-foreground">Investigation Files</h3>
		{#if agent.status === 'abandoned'}
			<Badge variant="destructive">Abandoned</Badge>
		{/if}
	</div>

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

	<!-- Primary Artifact Path Card (if available) -->
	{#if primaryArtifact}
		<div class="mb-4">
			<h4 class="mb-2 text-xs font-medium uppercase tracking-wide text-muted-foreground">Primary Artifact</h4>
			<button
				type="button"
				class="group flex w-full items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-left transition-all hover:bg-muted hover:border-primary/50 active:scale-[0.99]"
				onclick={() => copyToClipboard(primaryArtifact, 'artifact')}
			>
				<div class="flex-1 min-w-0">
					<p class="truncate font-mono text-xs">{primaryArtifact}</p>
				</div>
				<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors shrink-0">
					{copiedItem === 'artifact' ? '✓ Copied' : '📋 Copy'}
				</span>
			</button>
		</div>
	{/if}

	<!-- Terminal Commands Section -->
	<div class="mt-6">
		<h4 class="mb-3 text-xs font-medium uppercase tracking-wide text-muted-foreground">Terminal Commands</h4>
		<div class="space-y-2">
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

			<!-- View Primary Artifact (if available) -->
			{#if commands.viewArtifact}
				<button
					type="button"
					class="group flex w-full items-center gap-3 rounded-lg border bg-blue-500/10 px-3 py-2 text-left transition-all hover:bg-blue-500/20 hover:border-blue-500/30 active:scale-[0.99]"
					onclick={() => copyToClipboard(commands.viewArtifact!, 'cmd-artifact')}
				>
					<span class="text-lg shrink-0">🔍</span>
					<div class="flex-1 min-w-0">
						<p class="text-xs font-medium text-foreground">View Primary Artifact</p>
						<code class="text-[10px] text-muted-foreground truncate block">{commands.viewArtifact}</code>
					</div>
					<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors shrink-0">
						{copiedItem === 'cmd-artifact' ? '✓' : '📋'}
					</span>
				</button>
			{/if}
		</div>
	</div>

	<!-- Empty State for agents without workspace info -->
	{#if !agent.project_dir && !agent.id}
		<div class="mt-4 rounded-lg border border-dashed bg-muted/20 p-6 text-center">
			<p class="text-sm text-muted-foreground">No workspace information available for this agent.</p>
		</div>
	{/if}
</div>
