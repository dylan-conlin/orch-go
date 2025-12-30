<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { selectedAgent, selectedAgentId, sseEvents, createIssue } from '$lib/stores/agents';
	import type { Agent, SSEEvent } from '$lib/stores/agents';
	import ArtifactViewer from '$lib/components/artifact-viewer/artifact-viewer.svelte';
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { browser } from '$app/environment';
	
	// Issue creation state
	let creatingIssue = false;
	let issueCreationError: string | null = null;
	let createdIssueId: string | null = null;

	// Track which items were recently copied
	let copiedItem: string | null = null;
	let copyTimeout: ReturnType<typeof setTimeout> | null = null;
	
	// Collapsible sections state
	let showDetails = false;
	let showActivity = true;
	
	// Extract workspace name from agent ID (removes "[beads-id]" suffix if present)
	// Agent IDs have format "workspace-name [beads-id]" but artifact API expects just workspace name
	function extractWorkspaceName(agentId: string): string {
		// Look for "[beads-id]" pattern and strip it
		const bracketIndex = agentId.lastIndexOf(' [');
		if (bracketIndex !== -1 && agentId.endsWith(']')) {
			return agentId.substring(0, bracketIndex);
		}
		return agentId;
	}

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

	// Create follow-up issue from synthesis recommendation
	async function handleCreateIssue(action: string) {
		creatingIssue = true;
		issueCreationError = null;
		createdIssueId = null;
		
		try {
			// Clean up action text (remove bullet prefixes)
			const cleanAction = action.replace(/^[-*]\s*/, '').replace(/^\d+\.\s*/, '');
			
			// Create issue with context about the parent agent
			const parentContext = $selectedAgent?.beads_id 
				? `\n\nFollow-up from: ${$selectedAgent.beads_id}`
				: '';
			const description = `${cleanAction}${parentContext}`;
			
			const result = await createIssue(cleanAction.substring(0, 100), description, ['triage:ready']);
			if (result) {
				createdIssueId = result.id;
				// Auto-clear after 3 seconds
				setTimeout(() => {
					createdIssueId = null;
				}, 3000);
			}
		} catch (error) {
			issueCreationError = error instanceof Error ? error.message : 'Failed to create issue';
			// Auto-clear error after 5 seconds
			setTimeout(() => {
				issueCreationError = null;
			}, 5000);
		} finally {
			creatingIssue = false;
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
	function getActivityStyle(type?: string): string {
		switch (type) {
			case 'tool':
			case 'tool-invocation':
			case 'reasoning':
				return 'border-blue-500/20 bg-blue-500/5';
			case 'text':
				return 'border-muted-foreground/20 bg-muted/30';
			case 'step-start':
			case 'step-finish':
				return 'border-muted/50 bg-muted/10';
			default:
				return 'border-muted-foreground/20 bg-muted/20';
		}
	}

	// Filter SSE events for this agent's session
	$: agentEvents = $selectedAgent?.session_id 
		? $sseEvents.filter(e => {
			if (e.type !== 'message.part' && e.type !== 'message.part.updated') return false;
			const eventSessionId = e.properties?.part?.sessionID || e.properties?.sessionID;
			return eventSessionId === $selectedAgent?.session_id;
		}).slice(-50)
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
		<!-- Header - Full task title with wrap -->
		<div class="flex items-start justify-between border-b px-4 py-3 gap-2">
			<div class="flex items-start gap-3 flex-1 min-w-0">
				<div class={`h-3 w-3 rounded-full mt-1.5 shrink-0 ${getStatusColor($selectedAgent.status)} ${$selectedAgent.status === 'active' && $selectedAgent.is_processing ? 'animate-pulse' : ''}`}></div>
				<h2 id="agent-detail-title" class="text-lg font-semibold leading-snug">
					{$selectedAgent.task || $selectedAgent.id}
				</h2>
			</div>
			<Button variant="ghost" size="sm" onclick={closePanel} class="h-8 w-8 p-0 shrink-0">
				<span class="text-lg">×</span>
			</Button>
		</div>

		<!-- Compact Status Bar -->
		<div class="border-b px-4 py-2 flex flex-wrap items-center gap-2 text-sm">
			<Badge variant={getStatusVariant($selectedAgent.status)}>
				{$selectedAgent.status}
			</Badge>
			{#if $selectedAgent.phase}
				<Badge variant="outline" class="font-normal">
					{$selectedAgent.phase}
				</Badge>
			{/if}
			{#if $selectedAgent.skill}
				<span class="text-muted-foreground">{$selectedAgent.skill}</span>
			{/if}
			<span class="text-muted-foreground">•</span>
			<span class="text-muted-foreground">
				{$selectedAgent.runtime || formatDuration($selectedAgent.spawned_at)}
			</span>
			{#if $selectedAgent.beads_id}
				<span class="text-muted-foreground">•</span>
				<span class="font-mono text-xs text-muted-foreground">{$selectedAgent.beads_id}</span>
			{/if}
			{#if $selectedAgent.status === 'active' && $selectedAgent.is_processing}
				<Badge variant="secondary" class="animate-pulse ml-auto">
					Processing
				</Badge>
			{/if}
		</div>

		<!-- Main Content Area - scrollable -->
		<div class="flex-1 overflow-y-auto">
			<!-- Live Activity (for active agents) - Collapsible -->
			{#if $selectedAgent.status === 'active'}
				<div class="border-b">
					<button
						type="button"
						class="w-full px-4 py-2 flex items-center justify-between hover:bg-muted/50 transition-colors"
						onclick={() => showActivity = !showActivity}
					>
						<h3 class="text-sm font-medium text-muted-foreground">Live Activity</h3>
						<span class="text-muted-foreground text-sm">{showActivity ? '▼' : '▶'}</span>
					</button>
					
					{#if showActivity}
						<div class="px-4 pb-4">
							<!-- Current Activity -->
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
							{:else if $selectedAgent.last_activity}
								<div class="mb-3 rounded-lg border {getActivityStyle($selectedAgent.last_activity.type)} p-3">
									<div class="flex items-start gap-2">
										<span class="text-lg">{getActivityIcon($selectedAgent.last_activity.type)}</span>
										<div class="flex-1 min-w-0">
											<p class="text-sm font-medium">{$selectedAgent.last_activity.text || 'Working...'}</p>
											<span class="text-xs text-muted-foreground">
												{$selectedAgent.last_activity.type}
												<span class="text-muted-foreground/50">(last known)</span>
											</span>
										</div>
									</div>
								</div>
							{/if}

							<!-- Activity Log -->
							<div class="max-h-40 space-y-1 overflow-y-auto rounded border bg-muted/20 p-2 font-mono text-xs">
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
									{#if $selectedAgent.last_activity}
										<div class="flex items-start gap-2 py-1 text-muted-foreground">
											<span class="shrink-0">{getActivityIcon($selectedAgent.last_activity.type)}</span>
											<span class="flex-1 break-words leading-relaxed">
												{$selectedAgent.last_activity.text || 'Working...'} <span class="text-muted-foreground/50">(last known)</span>
											</span>
										</div>
									{:else}
										<p class="py-4 text-center text-muted-foreground">Waiting for activity...</p>
									{/if}
								{/each}
							</div>
						</div>
					{/if}
				</div>
			{/if}

			<!-- Artifact Viewer - Takes primary real estate for completed agents -->
			{#if $selectedAgent.status === 'completed' || $selectedAgent.phase === 'Complete'}
				<div class="p-4 flex-1">
					<h3 class="text-sm font-medium text-muted-foreground mb-3">Agent Output</h3>
					<div class="h-[calc(100vh-300px)] min-h-[300px]">
					<ArtifactViewer 
						workspaceId={extractWorkspaceName($selectedAgent.id)}
						beadsId={$selectedAgent.beads_id}
						skill={$selectedAgent.skill}
						closeReason={$selectedAgent.close_reason}
					/>
					</div>
				</div>
			{/if}

			<!-- Next Actions from Synthesis (inline for quick access) -->
			{#if $selectedAgent.synthesis?.next_actions && $selectedAgent.synthesis.next_actions.length > 0}
				<div class="border-t p-4">
					<div class="flex items-center justify-between mb-2">
						<h3 class="text-sm font-medium text-muted-foreground">Next Actions</h3>
						{#if issueCreationError}
							<span class="text-xs text-red-500">{issueCreationError}</span>
						{:else if createdIssueId}
							<span class="text-xs text-green-500">Created {createdIssueId}</span>
						{/if}
					</div>
					<ul class="space-y-1">
						{#each $selectedAgent.synthesis.next_actions as action}
							<li class="flex items-start gap-2 rounded p-1 hover:bg-muted/50 group">
								<span class="flex-1 text-sm">{action}</span>
								<button
									type="button"
									class="shrink-0 rounded border border-transparent px-2 py-0.5 text-[10px] text-muted-foreground opacity-0 transition-all hover:border-primary/50 hover:bg-primary/10 hover:text-foreground group-hover:opacity-100 disabled:opacity-50"
									onclick={() => handleCreateIssue(action)}
									disabled={creatingIssue}
								>
									{creatingIssue ? '...' : 'Create Issue'}
								</button>
							</li>
						{/each}
					</ul>
				</div>
			{/if}

			<!-- Close Reason fallback is now handled by ArtifactViewer when no synthesis exists -->
		</div>

		<!-- Collapsible Details Section at Bottom -->
		<div class="border-t">
			<button
				type="button"
				class="w-full px-4 py-2 flex items-center justify-between hover:bg-muted/50 transition-colors"
				onclick={() => showDetails = !showDetails}
			>
				<h3 class="text-sm font-medium text-muted-foreground">Details & Commands</h3>
				<span class="text-muted-foreground text-sm">{showDetails ? '▼' : '▶'}</span>
			</button>
			
			{#if showDetails}
				<div class="px-4 pb-4 space-y-4">
					<!-- Quick Copy -->
					<div>
						<h4 class="text-xs text-muted-foreground mb-2 uppercase tracking-wide">Quick Copy</h4>
						<div class="grid gap-2 sm:grid-cols-3">
							<!-- Workspace ID -->
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

							<!-- Session ID -->
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

							<!-- Beads ID -->
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

					<!-- Timestamps -->
					<div>
						<h4 class="text-xs text-muted-foreground mb-2 uppercase tracking-wide">Timestamps</h4>
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

					<!-- Quick Commands -->
					<div>
						<h4 class="text-xs text-muted-foreground mb-2 uppercase tracking-wide">Quick Commands</h4>
						<div class="grid gap-2 sm:grid-cols-2">
							{#if $selectedAgent.status === 'active'}
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
		</div>
	</div>
{/if}
