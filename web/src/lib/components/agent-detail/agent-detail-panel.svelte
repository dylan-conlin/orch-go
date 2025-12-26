<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { selectedAgent, selectedAgentId, sseEvents } from '$lib/stores/agents';
	import type { Agent, SSEEvent } from '$lib/stores/agents';
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { browser } from '$app/environment';

	// Track which items were recently copied
	let copiedItem: string | null = null;
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

	// Activity styling - different emphasis levels based on activity type
	// High: tool usage, reasoning (active work worth highlighting)
	// Medium: text output (communication)
	// Low: step transitions (transient markers, not worth highlighting)
	function getActivityStyle(type?: string): string {
		switch (type) {
			case 'tool':
			case 'tool-invocation':
			case 'reasoning':
				// Active work - subtle blue tint (less attention-grabbing than gold)
				return 'border-blue-500/20 bg-blue-500/5';
			case 'text':
				// Communication - very subtle styling
				return 'border-muted-foreground/20 bg-muted/30';
			case 'step-start':
			case 'step-finish':
				// Transient states - minimal styling, nearly invisible
				return 'border-muted/50 bg-muted/10';
			default:
				// Default - neutral styling
				return 'border-muted-foreground/20 bg-muted/20';
		}
	}

	// Filter SSE events for this agent's session
	// Note: For message.part events, sessionID is nested at properties.part.sessionID
	// For session.* events, sessionID is at properties.sessionID
	$: agentEvents = $selectedAgent?.session_id 
		? $sseEvents.filter(e => {
			if (e.type !== 'message.part' && e.type !== 'message.part.updated') return false;
			// sessionID is inside the part object for message.part events
			const eventSessionId = e.properties?.part?.sessionID || e.properties?.sessionID;
			return eventSessionId === $selectedAgent?.session_id;
		}).slice(-50)  // Keep more events for the full activity log
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

	<!-- Slide-out Panel - 2/3 width for better content visibility -->
	<div 
		class="fixed right-0 top-0 z-50 flex h-full w-full flex-col border-l bg-card shadow-xl sm:w-[66vw] lg:w-[60vw] xl:w-[55vw]"
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

		<!-- Live Activity Stream (for active agents) - Primary location for activity -->
		{#if $selectedAgent.status === 'active'}
			<div class="border-b p-4">
				<div class="mb-3 flex items-center justify-between">
					<h3 class="text-sm font-medium text-muted-foreground">Live Activity</h3>
					{#if $selectedAgent.is_processing}
						<Badge variant="secondary" class="animate-pulse">
							Processing
						</Badge>
					{/if}
				</div>
				
				<!-- Current Activity - styling varies by activity type -->
				{#if $selectedAgent.current_activity}
					<div class="mb-3 rounded-lg border {getActivityStyle($selectedAgent.current_activity.type)} p-3">
						<div class="flex items-start gap-2">
							<span class="text-lg">{getActivityIcon($selectedAgent.current_activity.type)}</span>
							<div class="flex-1 min-w-0">
								<p class="text-sm font-medium">{$selectedAgent.current_activity.text || 'Working...'}</p>
								<span class="text-xs text-muted-foreground">
									{$selectedAgent.current_activity.type}
								</span>
							</div>
						</div>
					</div>
				{/if}

				<!-- Activity Log - scrollable with more height -->
				<div class="max-h-64 space-y-1 overflow-y-auto rounded border bg-muted/20 p-2 font-mono text-xs">
					{#each agentEvents.slice().reverse() as event (event.id)}
						{@const part = event.properties?.part}
						{#if part}
							<div class="flex items-start gap-2 py-1 text-muted-foreground hover:bg-muted/50 rounded px-1 transition-colors">
								<span class="shrink-0">{getActivityIcon(part.type)}</span>
								<span class="flex-1 break-words leading-relaxed">
									{part.text || part.state?.title || (part.tool ? `Using ${part.tool}` : part.type)}
								</span>
							</div>
						{/if}
					{:else}
						<p class="py-4 text-center text-muted-foreground">Waiting for activity...</p>
					{/each}
				</div>
			</div>
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

			<!-- Synthesis (for completed agents, with close_reason fallback) -->
			{#if $selectedAgent.status === 'completed' && ($selectedAgent.synthesis || $selectedAgent.close_reason)}
				<div class="border-b p-4">
					<h3 class="mb-3 text-sm font-medium text-muted-foreground">
						{$selectedAgent.synthesis ? 'Synthesis' : 'Completion Summary'}
					</h3>
					<div class="space-y-3">
						{#if $selectedAgent.synthesis?.tldr}
							<div>
								<span class="text-xs text-muted-foreground">TLDR</span>
								<p class="text-sm">{$selectedAgent.synthesis.tldr}</p>
							</div>
						{:else if $selectedAgent.close_reason}
							<div>
								<span class="text-xs text-muted-foreground">Close Reason</span>
								<p class="text-sm">{$selectedAgent.close_reason}</p>
							</div>
						{/if}

						{#if $selectedAgent.synthesis?.outcome}
							<div>
								<span class="text-xs text-muted-foreground">Outcome</span>
								<Badge variant={$selectedAgent.synthesis.outcome === 'success' ? 'default' : 'secondary'}>
									{$selectedAgent.synthesis.outcome}
								</Badge>
							</div>
						{/if}

						{#if $selectedAgent.synthesis?.recommendation}
							<div>
								<span class="text-xs text-muted-foreground">Recommendation</span>
								<p class="text-sm">{$selectedAgent.synthesis.recommendation}</p>
							</div>
						{/if}

						{#if $selectedAgent.synthesis?.delta_summary}
							<div>
								<span class="text-xs text-muted-foreground">Changes</span>
								<p class="text-sm">{$selectedAgent.synthesis.delta_summary}</p>
							</div>
						{/if}

						{#if $selectedAgent.synthesis?.next_actions && $selectedAgent.synthesis.next_actions.length > 0}
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
