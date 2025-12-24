<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { selectedAgent, selectedAgentId, sseEvents } from '$lib/stores/agents';
	import type { Agent, SSEEvent } from '$lib/stores/agents';
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { browser } from '$app/environment';

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

	// Copy to clipboard helper
	async function copyToClipboard(text: string, label: string) {
		try {
			await navigator.clipboard.writeText(text);
			// Could add a toast notification here
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

	// Activity icon
	function getActivityIcon(type?: string): string {
		switch (type) {
			case 'text': return '💬';
			case 'tool':
			case 'tool-invocation': return '🔧';
			case 'reasoning': return '🤔';
			case 'step-start': return '▶️';
			case 'step-finish': return '✓';
			default: return '📝';
		}
	}

	// Filter SSE events for this agent's session
	$: agentEvents = $selectedAgent?.session_id 
		? $sseEvents.filter(e => 
			e.properties?.sessionID === $selectedAgent?.session_id && 
			e.type === 'message.part'
		).slice(-20)
		: [];

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

	<!-- Slide-out Panel -->
	<div 
		class="fixed right-0 top-0 z-50 flex h-full w-full flex-col border-l bg-card shadow-xl sm:w-[450px] lg:w-[500px]"
		transition:fly={{ x: 500, duration: 200 }}
		role="dialog"
		aria-modal="true"
		aria-labelledby="agent-detail-title"
	>
		<!-- Header -->
		<div class="flex items-center justify-between border-b px-4 py-3">
			<div class="flex items-center gap-2">
				<div class={`h-3 w-3 rounded-full ${getStatusColor($selectedAgent.status)}`}></div>
				<h2 id="agent-detail-title" class="text-lg font-semibold">Agent Details</h2>
			</div>
			<Button variant="ghost" size="sm" onclick={closePanel} class="h-8 w-8 p-0">
				<span class="text-lg">×</span>
			</Button>
		</div>

		<!-- Content - scrollable -->
		<div class="flex-1 overflow-y-auto">
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
					{#if $selectedAgent.is_processing}
						<Badge variant="secondary" class="animate-pulse">
							Processing
						</Badge>
					{/if}
					<span class="ml-auto text-sm text-muted-foreground">
						{$selectedAgent.runtime || formatDuration($selectedAgent.spawned_at)}
					</span>
				</div>
			</div>

			<!-- Identifiers Section -->
			<div class="border-b p-4">
				<h3 class="mb-3 text-sm font-medium text-muted-foreground">Identifiers</h3>
				<div class="space-y-2">
					<!-- Workspace ID -->
					<div class="flex items-center justify-between rounded bg-muted/50 px-3 py-2">
						<div>
							<span class="text-xs text-muted-foreground">Workspace</span>
							<p class="font-mono text-sm">{$selectedAgent.id}</p>
						</div>
						<Button 
							variant="ghost" 
							size="sm" 
							class="h-7 text-xs"
							onclick={() => copyToClipboard($selectedAgent?.id || '', 'Workspace ID')}
						>
							Copy
						</Button>
					</div>

					<!-- Session ID -->
					{#if $selectedAgent.session_id}
						<div class="flex items-center justify-between rounded bg-muted/50 px-3 py-2">
							<div>
								<span class="text-xs text-muted-foreground">Session ID</span>
								<p class="font-mono text-sm">{$selectedAgent.session_id}</p>
							</div>
							<Button 
								variant="ghost" 
								size="sm" 
								class="h-7 text-xs"
								onclick={() => copyToClipboard($selectedAgent?.session_id || '', 'Session ID')}
							>
								Copy
							</Button>
						</div>
					{/if}

					<!-- Beads ID -->
					{#if $selectedAgent.beads_id}
						<div class="flex items-center justify-between rounded bg-muted/50 px-3 py-2">
							<div>
								<span class="text-xs text-muted-foreground">Beads Issue</span>
								<p class="font-mono text-sm">{$selectedAgent.beads_id}</p>
							</div>
							<Button 
								variant="ghost" 
								size="sm" 
								class="h-7 text-xs"
								onclick={() => copyToClipboard($selectedAgent?.beads_id || '', 'Beads ID')}
							>
								Copy
							</Button>
						</div>
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

			<!-- Live Output (for active agents) -->
			{#if $selectedAgent.status === 'active'}
				<div class="border-b p-4">
					<h3 class="mb-3 text-sm font-medium text-muted-foreground">Live Activity</h3>
					
					{#if $selectedAgent.current_activity}
						<div class="mb-3 rounded bg-muted/50 p-2">
							<div class="flex items-center gap-2">
								<span>{getActivityIcon($selectedAgent.current_activity.type)}</span>
								<span class="text-sm">{$selectedAgent.current_activity.text || 'Working...'}</span>
							</div>
						</div>
					{/if}

					<div class="max-h-40 space-y-1 overflow-y-auto font-mono text-xs">
						{#each agentEvents.slice().reverse() as event (event.id)}
							{@const part = event.properties?.part}
							{#if part}
								<div class="flex items-start gap-1 py-0.5 text-muted-foreground">
									<span>{getActivityIcon(part.type)}</span>
									<span class="flex-1 break-all">
										{part.text || (part.tool ? `Using ${part.tool}` : part.type)}
									</span>
								</div>
							{/if}
						{:else}
							<p class="text-muted-foreground">Waiting for activity...</p>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Synthesis (for completed agents) -->
			{#if $selectedAgent.status === 'completed' && $selectedAgent.synthesis}
				<div class="border-b p-4">
					<h3 class="mb-3 text-sm font-medium text-muted-foreground">Synthesis</h3>
					<div class="space-y-3">
						{#if $selectedAgent.synthesis.tldr}
							<div>
								<span class="text-xs text-muted-foreground">TLDR</span>
								<p class="text-sm">{$selectedAgent.synthesis.tldr}</p>
							</div>
						{/if}

						{#if $selectedAgent.synthesis.outcome}
							<div>
								<span class="text-xs text-muted-foreground">Outcome</span>
								<Badge variant={$selectedAgent.synthesis.outcome === 'success' ? 'default' : 'secondary'}>
									{$selectedAgent.synthesis.outcome}
								</Badge>
							</div>
						{/if}

						{#if $selectedAgent.synthesis.recommendation}
							<div>
								<span class="text-xs text-muted-foreground">Recommendation</span>
								<p class="text-sm">{$selectedAgent.synthesis.recommendation}</p>
							</div>
						{/if}

						{#if $selectedAgent.synthesis.delta_summary}
							<div>
								<span class="text-xs text-muted-foreground">Changes</span>
								<p class="text-sm">{$selectedAgent.synthesis.delta_summary}</p>
							</div>
						{/if}

						{#if $selectedAgent.synthesis.next_actions && $selectedAgent.synthesis.next_actions.length > 0}
							<div>
								<span class="text-xs text-muted-foreground">Next Actions</span>
								<ul class="mt-1 list-inside list-disc text-sm">
									{#each $selectedAgent.synthesis.next_actions as action}
										<li>{action}</li>
									{/each}
								</ul>
							</div>
						{/if}
					</div>
				</div>
			{/if}
		</div>

		<!-- Actions Footer -->
		<div class="border-t p-4">
			<div class="flex flex-wrap gap-2">
				{#if $selectedAgent.status === 'active'}
					<!-- Active agent actions -->
					<Button 
						variant="outline" 
						size="sm"
						onclick={() => copyToClipboard(`orch send ${$selectedAgent?.session_id} ""`, 'Command')}
					>
						Copy Send Command
					</Button>
					<Button 
						variant="outline" 
						size="sm"
						onclick={() => copyToClipboard(`orch abandon ${$selectedAgent?.id}`, 'Command')}
					>
						Copy Abandon Command
					</Button>
				{:else if $selectedAgent.status === 'completed'}
					<!-- Completed agent actions -->
					<Button 
						variant="outline" 
						size="sm"
						onclick={() => copyToClipboard(`orch complete ${$selectedAgent?.id}`, 'Command')}
					>
						Copy Complete Command
					</Button>
				{/if}
				
				<!-- Common actions -->
				{#if $selectedAgent.beads_id}
					<Button 
						variant="outline" 
						size="sm"
						onclick={() => copyToClipboard(`bd show ${$selectedAgent?.beads_id}`, 'Command')}
					>
						Copy Show Issue Command
					</Button>
				{/if}
			</div>
		</div>
	</div>
{/if}
