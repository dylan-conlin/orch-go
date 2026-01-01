<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { selectedAgent, selectedAgentId, sseEvents, createIssue } from '$lib/stores/agents';
	import type { Agent, SSEEvent } from '$lib/stores/agents';
	import ArtifactViewer from '$lib/components/artifact-viewer/artifact-viewer.svelte';
	import { onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { browser } from '$app/environment';
	
	// Tab types
	type TabId = 'issue' | 'context' | 'activity' | 'deliverables';
	
	// Tab persistence key
	const TAB_STORAGE_KEY = 'orch-agent-detail-tab';
	
	// Issue creation state
	let creatingIssue = false;
	let issueCreationError: string | null = null;
	let createdIssueId: string | null = null;

	// Track which items were recently copied
	let copiedItem: string | null = null;
	let copyTimeout: ReturnType<typeof setTimeout> | null = null;
	
	// Collapsible details section state (for commands at bottom)
	let showDetails = false;
	
	// Active tab - will be set based on agent status and persisted preference
	let activeTab: TabId = 'activity';
	
	// Load persisted tab preference
	function loadTabPreference(): TabId | null {
		if (!browser) return null;
		try {
			const stored = localStorage.getItem(TAB_STORAGE_KEY);
			if (stored && ['issue', 'context', 'activity', 'deliverables'].includes(stored)) {
				return stored as TabId;
			}
		} catch (e) {
			console.warn('Failed to load tab preference:', e);
		}
		return null;
	}
	
	// Save tab preference
	function saveTabPreference(tab: TabId) {
		if (!browser) return;
		try {
			localStorage.setItem(TAB_STORAGE_KEY, tab);
		} catch (e) {
			console.warn('Failed to save tab preference:', e);
		}
	}
	
	// Get default tab based on agent status
	function getDefaultTab(agent: Agent | null): TabId {
		if (!agent) return 'activity';
		
		// Default to Deliverables for completed agents
		if (agent.status === 'completed' || agent.phase === 'Complete') {
			return 'deliverables';
		}
		
		// Default to Activity for active agents
		return 'activity';
	}
	
	// Set tab and persist
	function setTab(tab: TabId) {
		activeTab = tab;
		saveTabPreference(tab);
	}
	
	// Tab definitions
	const tabs: { id: TabId; label: string; icon: string }[] = [
		{ id: 'issue', label: 'Issue', icon: '📋' },
		{ id: 'context', label: 'Context', icon: '📦' },
		{ id: 'activity', label: 'Activity', icon: '⚡' },
		{ id: 'deliverables', label: 'Deliverables', icon: '📄' },
	];
	
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
	
	// Get gap quality color
	function getGapQualityColor(quality: number): string {
		if (quality >= 80) return 'text-green-500';
		if (quality >= 50) return 'text-yellow-500';
		return 'text-red-500';
	}
	
	// Get gap quality label
	function getGapQualityLabel(quality: number): string {
		if (quality >= 80) return 'Good';
		if (quality >= 50) return 'Fair';
		return 'Limited';
	}

	// Filter SSE events for this agent's session
	$: agentEvents = $selectedAgent?.session_id 
		? $sseEvents.filter(e => {
			if (e.type !== 'message.part' && e.type !== 'message.part.updated') return false;
			const eventSessionId = e.properties?.part?.sessionID || e.properties?.sessionID;
			return eventSessionId === $selectedAgent?.session_id;
		}).slice(-50)
		: [];
	
	// Initialize tab when agent changes
	$: if ($selectedAgent) {
		const savedTab = loadTabPreference();
		// Use saved preference if available, otherwise use default based on status
		activeTab = savedTab || getDefaultTab($selectedAgent);
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

		<!-- Tab Navigation -->
		<div class="border-b px-2" data-testid="agent-detail-tabs">
			<div class="flex gap-1">
				{#each tabs as tab}
					<button
						type="button"
						class="flex items-center gap-1.5 px-3 py-2 text-sm font-medium transition-colors border-b-2 -mb-px"
						class:text-primary={activeTab === tab.id}
						class:border-primary={activeTab === tab.id}
						class:text-muted-foreground={activeTab !== tab.id}
						class:border-transparent={activeTab !== tab.id}
						class:hover:text-foreground={activeTab !== tab.id}
						onclick={() => setTab(tab.id)}
						data-testid={`tab-${tab.id}`}
					>
						<span>{tab.icon}</span>
						<span>{tab.label}</span>
					</button>
				{/each}
			</div>
		</div>

		<!-- Main Content Area - scrollable -->
		<div class="flex-1 overflow-y-auto">
			<!-- Issue Tab -->
			{#if activeTab === 'issue'}
				<div class="p-4 space-y-4">
					{#if $selectedAgent.beads_id}
						<div class="space-y-3">
							<!-- Issue ID and Title -->
							<div>
								<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">Issue</h3>
								<div class="rounded-lg border bg-muted/20 p-3">
									<div class="flex items-start gap-2">
										<span class="font-mono text-sm font-medium text-primary">{$selectedAgent.beads_id}</span>
									</div>
									{#if $selectedAgent.beads_title}
										<p class="mt-2 text-sm">{$selectedAgent.beads_title}</p>
									{/if}
								</div>
							</div>
							
							<!-- Task Description -->
							{#if $selectedAgent.task}
								<div>
									<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">Task</h3>
									<p class="text-sm leading-relaxed">{$selectedAgent.task}</p>
								</div>
							{/if}
							
							<!-- Quick Commands for Issue -->
							<div>
								<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">Commands</h3>
								<div class="flex flex-wrap gap-2">
									<button
										type="button"
										class="group flex items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-left transition-all hover:bg-muted hover:border-primary/50 active:scale-[0.98]"
										onclick={() => copyToClipboard(`bd show ${$selectedAgent?.beads_id}`, 'show')}
									>
										<span class="text-sm">📋</span>
										<code class="text-xs text-muted-foreground">bd show {$selectedAgent.beads_id}</code>
										<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors">
											{copiedItem === 'show' ? '✓' : '📋'}
										</span>
									</button>
								</div>
							</div>
						</div>
					{:else}
						<div class="text-center py-8 text-muted-foreground">
							<p class="text-sm">No beads issue linked</p>
							<p class="text-xs mt-1 opacity-75">This agent was spawned without issue tracking</p>
						</div>
					{/if}
				</div>
			{/if}

			<!-- Context Tab -->
			{#if activeTab === 'context'}
				<div class="p-4 space-y-4">
					<!-- Gap Analysis -->
					{#if $selectedAgent.gap_analysis}
						<div>
							<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">Context Quality</h3>
							<div class="rounded-lg border bg-muted/20 p-3 space-y-3">
								<div class="flex items-center justify-between">
									<span class="text-sm">Quality Score</span>
									<span class={`text-lg font-bold ${getGapQualityColor($selectedAgent.gap_analysis.context_quality)}`}>
										{$selectedAgent.gap_analysis.context_quality}%
										<span class="text-xs font-normal ml-1">({getGapQualityLabel($selectedAgent.gap_analysis.context_quality)})</span>
									</span>
								</div>
								
								{#if $selectedAgent.gap_analysis.match_count !== undefined}
									<div class="grid grid-cols-3 gap-2 text-center">
										{#if $selectedAgent.gap_analysis.constraints !== undefined}
											<div class="rounded bg-muted/50 p-2">
												<div class="text-lg font-bold">{$selectedAgent.gap_analysis.constraints}</div>
												<div class="text-xs text-muted-foreground">Constraints</div>
											</div>
										{/if}
										{#if $selectedAgent.gap_analysis.decisions !== undefined}
											<div class="rounded bg-muted/50 p-2">
												<div class="text-lg font-bold">{$selectedAgent.gap_analysis.decisions}</div>
												<div class="text-xs text-muted-foreground">Decisions</div>
											</div>
										{/if}
										{#if $selectedAgent.gap_analysis.investigations !== undefined}
											<div class="rounded bg-muted/50 p-2">
												<div class="text-lg font-bold">{$selectedAgent.gap_analysis.investigations}</div>
												<div class="text-xs text-muted-foreground">Investigations</div>
											</div>
										{/if}
									</div>
								{/if}
								
								{#if $selectedAgent.gap_analysis.has_gaps && $selectedAgent.gap_analysis.should_warn}
									<div class="flex items-start gap-2 text-yellow-600 dark:text-yellow-500 text-xs">
										<span>⚠️</span>
										<span>Limited context was available when this agent was spawned</span>
									</div>
								{/if}
							</div>
						</div>
					{:else}
						<div class="text-center py-4 text-muted-foreground">
							<p class="text-sm">No context analysis available</p>
						</div>
					{/if}
					
					<!-- Project and Skill Info -->
					<div>
						<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">Spawn Info</h3>
						<div class="rounded-lg border bg-muted/20 p-3">
							<dl class="grid grid-cols-2 gap-3 text-sm">
								{#if $selectedAgent.project}
									<div>
										<dt class="text-muted-foreground text-xs">Project</dt>
										<dd class="font-medium">{$selectedAgent.project}</dd>
									</div>
								{/if}
								{#if $selectedAgent.skill}
									<div>
										<dt class="text-muted-foreground text-xs">Skill</dt>
										<dd class="font-medium">{$selectedAgent.skill}</dd>
									</div>
								{/if}
								<div>
									<dt class="text-muted-foreground text-xs">Spawned</dt>
									<dd class="font-medium">{formatTime($selectedAgent.spawned_at)}</dd>
								</div>
								<div>
									<dt class="text-muted-foreground text-xs">Duration</dt>
									<dd class="font-medium">{$selectedAgent.runtime || formatDuration($selectedAgent.spawned_at)}</dd>
								</div>
								{#if $selectedAgent.project_dir}
									<div class="col-span-2">
										<dt class="text-muted-foreground text-xs">Directory</dt>
										<dd class="font-mono text-xs truncate">{$selectedAgent.project_dir}</dd>
									</div>
								{/if}
							</dl>
						</div>
					</div>
				</div>
			{/if}

			<!-- Activity Tab -->
			{#if activeTab === 'activity'}
				<div class="p-4">
					{#if $selectedAgent.status === 'active'}
						<!-- Live Activity for Active Agents -->
						<div class="space-y-3">
							<!-- Current Activity -->
							{#if $selectedAgent.current_activity}
								<div class="rounded-lg border {getActivityStyle($selectedAgent.current_activity.type)} p-3">
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
								<div class="rounded-lg border {getActivityStyle($selectedAgent.last_activity.type)} p-3">
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
							<div>
								<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">Activity Stream</h3>
								<div class="max-h-[calc(100vh-400px)] space-y-1 overflow-y-auto rounded border bg-muted/20 p-2 font-mono text-xs">
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
						</div>
					{:else}
						<!-- Completed Agent Activity Summary -->
						<div class="space-y-3">
							{#if $selectedAgent.last_activity}
								<div>
									<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">Final Activity</h3>
									<div class="rounded-lg border {getActivityStyle($selectedAgent.last_activity.type)} p-3">
										<div class="flex items-start gap-2">
											<span class="text-lg">{getActivityIcon($selectedAgent.last_activity.type)}</span>
											<div class="flex-1 min-w-0">
												<p class="text-sm font-medium">{$selectedAgent.last_activity.text || 'Completed'}</p>
												<span class="text-xs text-muted-foreground">{$selectedAgent.last_activity.type}</span>
											</div>
										</div>
									</div>
								</div>
							{/if}
							
							<div>
								<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">Timeline</h3>
								<div class="rounded-lg border bg-muted/20 p-3">
									<dl class="space-y-2 text-sm">
										<div class="flex justify-between">
											<dt class="text-muted-foreground">Started</dt>
											<dd>{formatTime($selectedAgent.spawned_at)}</dd>
										</div>
										{#if $selectedAgent.completed_at}
											<div class="flex justify-between">
												<dt class="text-muted-foreground">Completed</dt>
												<dd>{formatTime($selectedAgent.completed_at)}</dd>
											</div>
										{:else if $selectedAgent.abandoned_at}
											<div class="flex justify-between">
												<dt class="text-muted-foreground">Abandoned</dt>
												<dd class="text-red-500">{formatTime($selectedAgent.abandoned_at)}</dd>
											</div>
										{/if}
										<div class="flex justify-between">
											<dt class="text-muted-foreground">Duration</dt>
											<dd class="font-medium">{$selectedAgent.runtime || formatDuration($selectedAgent.spawned_at)}</dd>
										</div>
									</dl>
								</div>
							</div>
						</div>
					{/if}
				</div>
			{/if}

			<!-- Deliverables Tab -->
			{#if activeTab === 'deliverables'}
				<div class="p-4 flex-1">
					{#if $selectedAgent.status === 'completed' || $selectedAgent.phase === 'Complete'}
						<div class="h-[calc(100vh-320px)] min-h-[300px]">
							<ArtifactViewer 
								workspaceId={extractWorkspaceName($selectedAgent.id)}
								beadsId={$selectedAgent.beads_id}
								skill={$selectedAgent.skill}
								closeReason={$selectedAgent.close_reason}
							/>
						</div>
						
						<!-- Next Actions from Synthesis -->
						{#if $selectedAgent.synthesis?.next_actions && $selectedAgent.synthesis.next_actions.length > 0}
							<div class="mt-4 pt-4 border-t">
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
					{:else}
						<!-- Active agent - show what will be available -->
						<div class="text-center py-8 text-muted-foreground">
							<p class="text-sm">Deliverables will appear here when the agent completes</p>
							<p class="text-xs mt-2 opacity-75">
								This agent is currently <Badge variant="outline" class="mx-1">{$selectedAgent.phase || $selectedAgent.status}</Badge>
							</p>
						</div>
					{/if}
				</div>
			{/if}
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
									onclick={() => copyToClipboard(`bd show ${$selectedAgent?.beads_id}`, 'show-cmd')}
								>
									<span class="text-lg">📋</span>
									<div class="flex-1 min-w-0">
										<p class="text-xs font-medium">Show Issue</p>
										<code class="text-[10px] text-muted-foreground">bd show ...</code>
									</div>
									<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors">
										{copiedItem === 'show-cmd' ? '✓' : '📋'}
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
