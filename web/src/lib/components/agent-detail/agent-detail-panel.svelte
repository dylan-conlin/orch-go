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
			case 'awaiting-cleanup': return 'bg-amber-500';
			default: return 'bg-gray-500';
		}
	}

	function getStatusVariant(status: Agent['status']) {
		switch (status) {
			case 'active': return 'active';
			case 'completed': return 'completed';
			case 'abandoned': return 'abandoned';
			case 'awaiting-cleanup': return 'secondary';
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

	// Prevent body scroll when panel is open to avoid double scrollbar
	$effect(() => {
		if (!browser) return;
		
		if ($selectedAgent) {
			// Panel is open - disable body scroll
			document.body.style.overflow = 'hidden';
		} else {
			// Panel is closed - restore body scroll
			document.body.style.overflow = '';
		}
		
		// Cleanup when effect is destroyed (component unmounts)
		return () => {
			document.body.style.overflow = '';
		};
	});

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

		<!-- Tab Content -->
		<div class="flex-1 overflow-hidden flex flex-col">
			<!-- Activity Tab (for active agents) - handles its own scrolling -->
			{#if activeTab === 'activity'}
				<ActivityTab agent={$selectedAgent} />
			{/if}

			<!-- Synthesis Tab (for completed agents) -->
			{#if activeTab === 'synthesis'}
				<div class="flex-1 overflow-y-auto p-4">
					<SynthesisTab agent={$selectedAgent} />
				</div>
			{/if}

			<!-- Investigation Tab (for completed/abandoned agents) -->
			{#if activeTab === 'investigation'}
				<div class="flex-1 overflow-y-auto p-4">
					<InvestigationTab agent={$selectedAgent} />
				</div>
			{/if}
		</div>
	</div>
{/if}
