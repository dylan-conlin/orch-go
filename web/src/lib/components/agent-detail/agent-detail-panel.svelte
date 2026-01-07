<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { TabButton, InvestigationTab, ActivityTab, SynthesisTab } from '$lib/components/agent-detail';
	import { selectedAgent, selectedAgentId } from '$lib/stores/agents';
	import type { Agent } from '$lib/stores/agents';
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { browser } from '$app/environment';
	
	// Tab types based on agent state
	type TabType = 'activity' | 'investigation' | 'synthesis';
	
	// Active tab state - will be determined by agent status
	let activeTab: TabType = $state('activity');
	
	// Determine which tabs are visible based on agent status
	function getVisibleTabs(agent: Agent | null): TabType[] {
		if (!agent) return [];
		switch (agent.status) {
			case 'active':
				return ['activity'];
			case 'completed':
				return ['synthesis', 'investigation'];
			case 'abandoned':
				return ['investigation'];
			default:
				return ['activity'];
		}
	}
	
	// Get default tab for agent status
	function getDefaultTab(agent: Agent | null): TabType {
		if (!agent) return 'activity';
		switch (agent.status) {
			case 'active':
				return 'activity';
			case 'completed':
				return 'synthesis';
			case 'abandoned':
				return 'investigation';
			default:
				return 'activity';
		}
	}
	
	// Visible tabs derived from agent
	$effect(() => {
		if ($selectedAgent) {
			const visibleTabs = getVisibleTabs($selectedAgent);
			// Reset to default tab if current tab isn't visible for this agent state
			if (!visibleTabs.includes(activeTab)) {
				activeTab = getDefaultTab($selectedAgent);
			}
		}
	});
	
	// Track which items were recently copied
	let copiedItem: string | null = $state(null);
	let copyTimeout: ReturnType<typeof setTimeout> | null = null;

	// Close panel handler
	function closePanel() {
		selectedAgentId.set(null);
	}

	// Handle escape key
	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			closePanel();
		}
	}

	// Handle click outside
	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			closePanel();
		}
	}

	// Copy to clipboard helper with visual feedback
	async function copyToClipboard(text: string, label: string) {
		try {
			await navigator.clipboard.writeText(text);
			// Set visual feedback
			copiedItem = label;
			if (copyTimeout) clearTimeout(copyTimeout);
			copyTimeout = setTimeout(() => {
				copiedItem = null;
			}, 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
		}
	}

	// Status helpers
	function getStatusColor(status: Agent['status']) {
		switch (status) {
			case 'active': return 'bg-green-500';
			case 'completed': return 'bg-blue-500';
			case 'abandoned': return 'bg-red-500';
			default: return 'bg-gray-500';
		}
	}

	function getStatusVariant(status: Agent['status']) {
		switch (status) {
			case 'active': return 'active';
			case 'completed': return 'completed';
			case 'abandoned': return 'abandoned';
			default: return 'default';
		}
	}

	// Format duration
	function formatDuration(isoDate: string | undefined): string {
		if (!isoDate) return '-';
		const date = new Date(isoDate);
		if (isNaN(date.getTime())) return '-';
		const ms = Date.now() - date.getTime();
		const minutes = Math.floor(ms / 60000);
		const hours = Math.floor(minutes / 60);
		if (hours > 0) {
			return `${hours}h ${minutes % 60}m`;
		}
		return `${minutes}m`;
	}

	// Format timestamp
	function formatTime(isoDate: string | undefined): string {
		if (!isoDate) return '-';
		const date = new Date(isoDate);
		if (isNaN(date.getTime())) return '-';
		return date.toLocaleTimeString();
	}

	onMount(() => {
		if (browser) {
			window.addEventListener('keydown', handleKeydown);
		}
		return () => {
			if (browser) {
				window.removeEventListener('keydown', handleKeydown);
			}
		};
	});
</script>

{#if browser && $selectedAgent}
	<!-- Backdrop -->
	<button 
		type="button"
		class="fixed inset-0 z-40 cursor-default border-none bg-background/80 backdrop-blur-sm"
		onclick={handleBackdropClick}
		aria-label="Close panel"
	></button>

	<!-- Slide-out Panel - 80-85% width for better content visibility -->
	<div 
		class="fixed right-0 top-0 z-50 flex h-full w-full flex-col border-l bg-card shadow-xl sm:w-[85vw] lg:w-[80vw] max-w-[1200px]"
		transition:fly={{ x: 500, duration: 200 }}
		role="dialog"
		aria-modal="true"
		aria-labelledby="agent-detail-title"
	>
		<!-- Header -->
		<div class="flex items-center justify-between border-b px-4 py-3">
			<div class="flex items-center gap-3">
				<div class={`h-3 w-3 rounded-full ${getStatusColor($selectedAgent.status)} ${$selectedAgent.status === 'active' && $selectedAgent.is_processing ? 'animate-pulse' : ''}`}></div>
				<h2 id="agent-detail-title" class="text-lg font-semibold">
					{$selectedAgent.task || $selectedAgent.id}
				</h2>
			</div>
			<Button variant="ghost" size="sm" onclick={closePanel} class="h-8 w-8 p-0">
				<span class="text-lg">×</span>
			</Button>
		</div>

		<!-- Status Bar -->
		<div class="border-b p-4">
			<div class="flex flex-wrap items-center gap-2">
				<Badge variant={getStatusVariant($selectedAgent.status)}>
					{$selectedAgent.status}
				</Badge>
				{#if $selectedAgent.phase}
					<Badge variant="outline">
						{$selectedAgent.phase}
					</Badge>
				{/if}
				{#if $selectedAgent.status === 'active' && $selectedAgent.is_processing}
					<Badge variant="secondary" class="animate-pulse">
						Processing
					</Badge>
				{/if}
				<span class="ml-auto text-sm text-muted-foreground">
					{$selectedAgent.runtime || formatDuration($selectedAgent.spawned_at)}
				</span>
			</div>
		</div>

		<!-- Tab Navigation -->
		<div class="border-b px-4">
			<div class="flex gap-1" role="tablist" aria-label="Agent detail tabs">
				{#if getVisibleTabs($selectedAgent).includes('activity')}
					<TabButton active={activeTab === 'activity'} onclick={() => activeTab = 'activity'}>
						Activity
					</TabButton>
				{/if}
				{#if getVisibleTabs($selectedAgent).includes('synthesis')}
					<TabButton active={activeTab === 'synthesis'} onclick={() => activeTab = 'synthesis'}>
						Synthesis
					</TabButton>
				{/if}
				{#if getVisibleTabs($selectedAgent).includes('investigation')}
					<TabButton active={activeTab === 'investigation'} onclick={() => activeTab = 'investigation'}>
						Investigation
					</TabButton>
				{/if}
			</div>
		</div>

		<!-- Tab Content - scrollable -->
		<div class="flex-1 overflow-y-auto">
		<!-- Activity Tab (for active agents) -->
		{#if activeTab === 'activity'}
			<ActivityTab agent={$selectedAgent} />
		{/if}

		<!-- Synthesis Tab (for completed agents) -->
		{#if activeTab === 'synthesis'}
			<SynthesisTab agent={$selectedAgent} />
		{/if}

		<!-- Investigation Tab (for completed/abandoned agents) -->
		{#if activeTab === 'investigation'}
			<InvestigationTab agent={$selectedAgent} />
		{/if}

		<!-- Quick Copy Section - Full-width clickable items -->
		<div class="border-b p-4">
			<h3 class="mb-3 text-sm font-medium text-muted-foreground">Quick Copy</h3>
			<div class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
				<!-- Workspace ID - clickable card -->
				<button
					type="button"
					class="group flex items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-left transition-all hover:bg-muted hover:border-primary/50 active:scale-[0.98]"
					onclick={() => copyToClipboard($selectedAgent?.id || '', 'workspace')}
				>
					<div class="flex-1 min-w-0">
						<span class="text-[10px] uppercase tracking-wide text-muted-foreground">Workspace</span>
						<p class="truncate font-mono text-xs">{$selectedAgent.id}</p>
					</div>
					<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors">
						{copiedItem === 'workspace' ? '✓' : '📋'}
					</span>
				</button>

				<!-- Session ID - clickable card -->
				{#if $selectedAgent.session_id}
					<button
						type="button"
						class="group flex items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-left transition-all hover:bg-muted hover:border-primary/50 active:scale-[0.98]"
						onclick={() => copyToClipboard($selectedAgent?.session_id || '', 'session')}
					>
						<div class="flex-1 min-w-0">
							<span class="text-[10px] uppercase tracking-wide text-muted-foreground">Session</span>
							<p class="truncate font-mono text-xs">{$selectedAgent.session_id.slice(0, 12)}...</p>
						</div>
						<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors">
							{copiedItem === 'session' ? '✓' : '📋'}
						</span>
					</button>
				{/if}

				<!-- Beads ID - clickable card -->
				{#if $selectedAgent.beads_id}
					<button
						type="button"
						class="group flex items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-left transition-all hover:bg-muted hover:border-primary/50 active:scale-[0.98]"
						onclick={() => copyToClipboard($selectedAgent?.beads_id || '', 'beads')}
					>
						<div class="flex-1 min-w-0">
							<span class="text-[10px] uppercase tracking-wide text-muted-foreground">Beads Issue</span>
							<p class="truncate font-mono text-xs">{$selectedAgent.beads_id}</p>
						</div>
						<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors">
							{copiedItem === 'beads' ? '✓' : '📋'}
						</span>
					</button>
				{/if}
			</div>
		</div>

			<!-- Context Section -->
			<div class="border-b p-4">
				<h3 class="mb-3 text-sm font-medium text-muted-foreground">Context</h3>
				<div class="space-y-3">
					{#if $selectedAgent.task}
						<div>
							<span class="text-xs text-muted-foreground">Task</span>
							<p class="text-sm">{$selectedAgent.task}</p>
						</div>
					{/if}
					<div class="flex flex-wrap gap-2">
						{#if $selectedAgent.project}
							<Badge variant="secondary">{$selectedAgent.project}</Badge>
						{/if}
						{#if $selectedAgent.skill}
							<Badge variant="outline">{$selectedAgent.skill}</Badge>
						{/if}
					</div>
					<div class="grid grid-cols-2 gap-2 text-xs text-muted-foreground">
						<div>
							<span class="block">Spawned</span>
							<span class="text-foreground">{formatTime($selectedAgent.spawned_at)}</span>
						</div>
						<div>
							<span class="block">Last Updated</span>
							<span class="text-foreground">{formatTime($selectedAgent.updated_at)}</span>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Quick Commands Footer - clickable command cards -->
		<div class="border-t p-4">
			<h3 class="mb-3 text-sm font-medium text-muted-foreground">Quick Commands</h3>
			<div class="grid gap-2 sm:grid-cols-2">
				{#if $selectedAgent.status === 'active'}
					<!-- Active agent commands -->
					<button
						type="button"
						class="group flex items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-left transition-all hover:bg-muted hover:border-primary/50 active:scale-[0.98]"
						onclick={() => copyToClipboard(`orch send ${$selectedAgent?.session_id} ""`, 'send')}
					>
						<span class="text-lg">💬</span>
						<div class="flex-1 min-w-0">
							<p class="text-xs font-medium">Send Message</p>
							<code class="text-[10px] text-muted-foreground">orch send ...</code>
						</div>
						<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors">
							{copiedItem === 'send' ? '✓' : '📋'}
						</span>
					</button>
					<button
						type="button"
						class="group flex items-center gap-2 rounded-lg border bg-red-500/10 px-3 py-2 text-left transition-all hover:bg-red-500/20 hover:border-red-500/50 active:scale-[0.98]"
						onclick={() => copyToClipboard(`orch abandon ${$selectedAgent?.id}`, 'abandon')}
					>
						<span class="text-lg">🛑</span>
						<div class="flex-1 min-w-0">
							<p class="text-xs font-medium">Abandon Agent</p>
							<code class="text-[10px] text-muted-foreground">orch abandon ...</code>
						</div>
						<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors">
							{copiedItem === 'abandon' ? '✓' : '📋'}
						</span>
					</button>
				{:else if $selectedAgent.status === 'completed'}
					<!-- Completed agent commands -->
					<button
						type="button"
						class="group flex items-center gap-2 rounded-lg border bg-green-500/10 px-3 py-2 text-left transition-all hover:bg-green-500/20 hover:border-green-500/50 active:scale-[0.98]"
						onclick={() => copyToClipboard(`orch complete ${$selectedAgent?.id}`, 'complete')}
					>
						<span class="text-lg">✅</span>
						<div class="flex-1 min-w-0">
							<p class="text-xs font-medium">Complete Agent</p>
							<code class="text-[10px] text-muted-foreground">orch complete ...</code>
						</div>
						<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors">
							{copiedItem === 'complete' ? '✓' : '📋'}
						</span>
					</button>
				{/if}
				
				<!-- Common commands -->
				{#if $selectedAgent.beads_id}
					<button
						type="button"
						class="group flex items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-left transition-all hover:bg-muted hover:border-primary/50 active:scale-[0.98]"
						onclick={() => copyToClipboard(`bd show ${$selectedAgent?.beads_id}`, 'show')}
					>
						<span class="text-lg">📋</span>
						<div class="flex-1 min-w-0">
							<p class="text-xs font-medium">Show Issue</p>
							<code class="text-[10px] text-muted-foreground">bd show ...</code>
						</div>
						<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors">
							{copiedItem === 'show' ? '✓' : '📋'}
						</span>
					</button>
				{/if}
			</div>
		</div>
	</div>
{/if}
