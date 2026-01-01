<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { selectedAgent, selectedAgentId, sseEvents, createIssue, fetchIssueDetails, fetchDeliverables, fetchSpawnContext } from '$lib/stores/agents';
	import type { Agent, SSEEvent, IssueDetail, Deliverables, SpawnContext } from '$lib/stores/agents';
	import ArtifactViewer from '$lib/components/artifact-viewer/artifact-viewer.svelte';
	import { onMount, tick } from 'svelte';
	import { fly } from 'svelte/transition';
	import { browser } from '$app/environment';
	import { marked } from 'marked';
	
	// Tab types
	type TabId = 'issue' | 'context' | 'activity' | 'deliverables';
	
	// Tab persistence key
	const TAB_STORAGE_KEY = 'orch-agent-detail-tab';
	
	// Issue creation state
	let creatingIssue = $state(false);
	let issueCreationError = $state<string | null>(null);
	let createdIssueId = $state<string | null>(null);
	
	// Issue tab state
	let issueDetails = $state<IssueDetail | null>(null);
	let issueLoading = $state(false);
	let issueError = $state<string | null>(null);
	let lastFetchedBeadsId = $state<string | null>(null);
	
	// Deliverables tab state
	let deliverables = $state<Deliverables | null>(null);
	let deliverablesLoading = $state(false);
	let lastFetchedWorkspaceId = $state<string | null>(null);
	
	// Context tab state (spawn context)
	let spawnContext = $state<SpawnContext | null>(null);
	let spawnContextLoading = $state(false);
	let lastFetchedContextWorkspaceId = $state<string | null>(null);

	// Track which items were recently copied
	let copiedItem = $state<string | null>(null);
	let copyTimeout: ReturnType<typeof setTimeout> | null = null;
	
	// Collapsible details section state (for commands at bottom)
	let showDetails = $state(false);
	
	// Active tab - will be set based on agent status and persisted preference
	let activeTab = $state<TabId>('activity');
	
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

	// Activity icon - human-readable icons based on tool/action type
	function getActivityIcon(type?: string, toolName?: string): string {
		// Tool-specific icons for common tools
		if (toolName) {
			switch (toolName) {
				case 'edit': return '✏️';
				case 'read': return '📖';
				case 'write': return '📝';
				case 'bash': return '💻';
				case 'glob': return '🔍';
				case 'grep': return '🔎';
				case 'task': return '📋';
				case 'todoread':
				case 'todowrite': return '✅';
				case 'webfetch': return '🌐';
				default: return '🔧';
			}
		}
		
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

	// Human-readable label for activity events
	function getHumanReadableLabel(part: any): string {
		if (!part) return 'Unknown activity';
		
		const toolName = part.tool || part.toolName;
		const state = part.state;
		
		// Tool-specific labels
		if (toolName) {
			switch (toolName) {
				case 'edit': {
					const filePath = state?.input?.filePath || part.input?.filePath;
					if (filePath) {
						const shortPath = filePath.split('/').slice(-2).join('/');
						return `Edit: ${shortPath}`;
					}
					return 'Edit file';
				}
				case 'read': {
					const filePath = state?.input?.filePath || part.input?.filePath;
					if (filePath) {
						const shortPath = filePath.split('/').slice(-2).join('/');
						return `Read: ${shortPath}`;
					}
					return 'Read file';
				}
				case 'write': {
					const filePath = state?.input?.filePath || part.input?.filePath;
					if (filePath) {
						const shortPath = filePath.split('/').slice(-2).join('/');
						return `Write: ${shortPath}`;
					}
					return 'Write file';
				}
				case 'bash': {
					const command = state?.input?.command || part.input?.command;
					if (command) {
						const shortCmd = command.length > 50 ? command.substring(0, 50) + '...' : command;
						return `Run: ${shortCmd}`;
					}
					return 'Run command';
				}
				case 'glob': {
					const pattern = state?.input?.pattern || part.input?.pattern;
					if (pattern) return `Search: ${pattern}`;
					return 'Search files';
				}
				case 'grep': {
					const pattern = state?.input?.pattern || part.input?.pattern;
					if (pattern) return `Grep: ${pattern}`;
					return 'Search content';
				}
				case 'task': {
					const description = state?.input?.description || part.input?.description;
					if (description) return `Task: ${description}`;
					return 'Spawn task';
				}
				case 'todoread': return 'Read todos';
				case 'todowrite': return 'Update todos';
				case 'webfetch': {
					const url = state?.input?.url || part.input?.url;
					if (url) {
						try {
							const hostname = new URL(url).hostname;
							return `Fetch: ${hostname}`;
						} catch {
							return `Fetch: ${url.substring(0, 30)}...`;
						}
					}
					return 'Fetch URL';
				}
				default:
					if (state?.title) return state.title;
					return `Using ${toolName}`;
			}
		}
		
		// Type-specific labels
		switch (part.type) {
			case 'text':
				return part.text?.substring(0, 80) || 'Thinking...';
			case 'reasoning':
				return part.text?.substring(0, 80) || 'Reasoning...';
			case 'step-start':
				return 'Step started';
			case 'step-finish':
				return 'Step completed';
			default:
				return part.text || state?.title || part.type?.replace(/-/g, ' ') || 'Working...';
		}
	}

	// Get status indicator for tool execution
	function getToolStatus(part: any): { text: string; class: string } | null {
		const state = part?.state;
		if (!state?.status) return null;
		
		switch (state.status) {
			case 'running':
				return { text: '...', class: 'text-yellow-500 animate-pulse' };
			case 'completed':
				return { text: '✓', class: 'text-green-500' };
			case 'error':
				return { text: '✗', class: 'text-red-500' };
			default:
				return null;
		}
	}

	// Group events - combine related events into expandable blocks
	interface GroupedEvent {
		id: string;
		type: 'tool' | 'text' | 'step';
		primary: SSEEvent;
		related: SSEEvent[];
		toolName?: string;
		label: string;
		expanded: boolean;
		timestamp: number;
		status?: { text: string; class: string } | undefined;
	}
	
	// Expanded state for grouped events
	let expandedGroups = $state<Set<string>>(new Set());
	
	function toggleGroup(id: string) {
		if (expandedGroups.has(id)) {
			expandedGroups.delete(id);
		} else {
			expandedGroups.add(id);
		}
		expandedGroups = new Set(expandedGroups); // Trigger reactivity with new Set
	}

	// Group related events together
	function groupEvents(events: SSEEvent[]): GroupedEvent[] {
		const groups: GroupedEvent[] = [];
		const processed = new Set<string>();
		
		for (let i = 0; i < events.length; i++) {
			const event = events[i];
			if (processed.has(event.id)) continue;
			
			const part = event.properties?.part;
			if (!part) continue;
			
			const toolName = part.tool || (part as any).toolName;
			const isToolEvent = part.type === 'tool' || part.type === 'tool-invocation';
			
			if (isToolEvent && toolName) {
				// Group tool invocation with its result (if available)
				const related: SSEEvent[] = [];
				
				// Look for related events (next few events with same tool)
				for (let j = i + 1; j < Math.min(i + 3, events.length); j++) {
					const nextEvent = events[j];
					const nextPart = nextEvent.properties?.part;
					if (nextPart && (nextPart.tool === toolName || nextPart.type === 'step-finish')) {
						related.push(nextEvent);
						processed.add(nextEvent.id);
					}
				}
				
				groups.push({
					id: event.id,
					type: 'tool',
					primary: event,
					related,
					toolName,
					label: getHumanReadableLabel(part),
					expanded: expandedGroups.has(event.id),
					timestamp: event.timestamp || Date.now(),
					status: getToolStatus(part) || undefined
				});
			} else if (part.type === 'text' || part.type === 'reasoning') {
				groups.push({
					id: event.id,
					type: 'text',
					primary: event,
					related: [],
					label: getHumanReadableLabel(part),
					expanded: expandedGroups.has(event.id),
					timestamp: event.timestamp || Date.now()
				});
			} else if (part.type === 'step-start' || part.type === 'step-finish') {
				// Skip standalone step events - they'll be grouped with tools
				if (!processed.has(event.id)) {
					groups.push({
						id: event.id,
						type: 'step',
						primary: event,
						related: [],
						label: getHumanReadableLabel(part),
						expanded: false,
						timestamp: event.timestamp || Date.now()
					});
				}
			}
			
			processed.add(event.id);
		}
		
		return groups;
	}

	// Render markdown safely
	function renderMarkdown(text: string): string {
		if (!text) return '';
		try {
			return marked.parse(text, { async: false }) as string;
		} catch {
			return text;
		}
	}

	// Reference to the activity container for auto-scrolling
	let activityContainer: HTMLDivElement;
	let shouldAutoScroll = $state(true);

	// Auto-scroll to bottom when new events arrive
	async function scrollToBottom() {
		if (activityContainer && shouldAutoScroll) {
			await tick();
			activityContainer.scrollTop = activityContainer.scrollHeight;
		}
	}

	// Check if user has scrolled up (disable auto-scroll if so)
	function handleActivityScroll() {
		if (!activityContainer) return;
		const { scrollTop, scrollHeight, clientHeight } = activityContainer;
		// If within 50px of bottom, enable auto-scroll
		shouldAutoScroll = scrollHeight - scrollTop - clientHeight < 50;
	}

	// Filter SSE events for this agent's session
	let agentEvents = $derived($selectedAgent?.session_id 
		? $sseEvents.filter(e => {
			if (e.type !== 'message.part' && e.type !== 'message.part.updated') return false;
			const eventSessionId = e.properties?.part?.sessionID || e.properties?.sessionID;
			return eventSessionId === $selectedAgent?.session_id;
		}).slice(-50)
		: []);

	// Watch for new events and auto-scroll
	$effect(() => {
		if (agentEvents.length > 0 && activityContainer) {
			scrollToBottom();
		}
	});

	// Compute grouped events reactively
	let groupedEvents = $derived(groupEvents(agentEvents));
	
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
	
	// Initialize tab when agent changes
	$effect(() => {
		if ($selectedAgent) {
			const savedTab = loadTabPreference();
			// Use saved preference if available, otherwise use default based on status
			activeTab = savedTab || getDefaultTab($selectedAgent);
		}
	});
	
	// Fetch issue details when beads_id changes (or when switching to Issue tab)
	$effect(() => {
		if ($selectedAgent?.beads_id && $selectedAgent.beads_id !== lastFetchedBeadsId) {
			loadIssueDetails($selectedAgent.beads_id, $selectedAgent.project_dir);
		}
	});
	
	async function loadIssueDetails(beadsId: string, projectDir?: string) {
		issueLoading = true;
		issueError = null;
		lastFetchedBeadsId = beadsId;
		
		try {
			issueDetails = await fetchIssueDetails(beadsId, projectDir);
			if (issueDetails.error) {
				issueError = issueDetails.error;
			}
		} catch (error) {
			issueError = error instanceof Error ? error.message : 'Failed to load issue details';
		} finally {
			issueLoading = false;
		}
	}
	
	// Fetch deliverables when switching to deliverables tab or agent changes
	$effect(() => {
		if (activeTab === 'deliverables' && $selectedAgent?.id) {
			const workspaceId = extractWorkspaceName($selectedAgent.id);
			if (workspaceId !== lastFetchedWorkspaceId) {
				loadDeliverables(workspaceId, $selectedAgent.spawned_at, $selectedAgent.project_dir, $selectedAgent.beads_id);
			}
		}
	});
	
	async function loadDeliverables(workspaceId: string, spawnedAt?: string, projectDir?: string, beadsId?: string) {
		deliverablesLoading = true;
		lastFetchedWorkspaceId = workspaceId;
		
		try {
			deliverables = await fetchDeliverables(workspaceId, spawnedAt, projectDir, beadsId);
		} catch (error) {
			console.error('Failed to load deliverables:', error);
		} finally {
			deliverablesLoading = false;
		}
	}
	
	// Fetch spawn context when switching to context tab or agent changes
	$effect(() => {
		if (activeTab === 'context' && $selectedAgent?.id) {
			const workspaceId = extractWorkspaceName($selectedAgent.id);
			if (workspaceId !== lastFetchedContextWorkspaceId) {
				loadSpawnContext(workspaceId);
			}
		}
	});
	
	async function loadSpawnContext(workspaceId: string) {
		spawnContextLoading = true;
		lastFetchedContextWorkspaceId = workspaceId;
		
		try {
			spawnContext = await fetchSpawnContext(workspaceId);
		} catch (error) {
			console.error('Failed to load spawn context:', error);
		} finally {
			spawnContextLoading = false;
		}
	}
	
	// Format commit timestamp to relative time
	function formatCommitTime(timestamp: string): string {
		try {
			const date = new Date(timestamp);
			const now = new Date();
			const diffMs = now.getTime() - date.getTime();
			const diffMins = Math.floor(diffMs / 60000);
			const diffHours = Math.floor(diffMins / 60);
			
			if (diffMins < 1) return 'just now';
			if (diffMins < 60) return `${diffMins}m ago`;
			if (diffHours < 24) return `${diffHours}h ago`;
			return date.toLocaleDateString();
		} catch {
			return timestamp;
		}
	}
	
	// Helper to get priority label and color
	function getPriorityInfo(priority: number): { label: string; color: string } {
		switch (priority) {
			case 0: return { label: 'P0 Critical', color: 'text-red-500' };
			case 1: return { label: 'P1 High', color: 'text-orange-500' };
			case 2: return { label: 'P2 Normal', color: 'text-blue-500' };
			case 3: return { label: 'P3 Low', color: 'text-gray-500' };
			default: return { label: `P${priority}`, color: 'text-gray-500' };
		}
	}
	
	// Helper to get status badge variant
	function getIssueStatusVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
		switch (status?.toLowerCase()) {
			case 'open': return 'default';
			case 'in_progress': return 'secondary';
			case 'closed': return 'outline';
			case 'blocked': return 'destructive';
			default: return 'default';
		}
	}
	
	// Format relative time
	function formatRelativeTime(isoDate: string | undefined): string {
		if (!isoDate) return '';
		const date = new Date(isoDate);
		if (isNaN(date.getTime())) return '';
		
		const now = Date.now();
		const diff = now - date.getTime();
		const minutes = Math.floor(diff / 60000);
		const hours = Math.floor(minutes / 60);
		const days = Math.floor(hours / 24);
		
		if (days > 0) return `${days}d ago`;
		if (hours > 0) return `${hours}h ago`;
		if (minutes > 0) return `${minutes}m ago`;
		return 'just now';
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

	<!-- Slide-out Panel - wider for better content visibility -->
	<div 
		class="fixed right-0 top-0 z-50 flex h-full w-full flex-col border-l bg-card shadow-xl sm:w-[80vw] lg:w-[75vw] xl:w-[70vw]"
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
						{#if issueLoading}
							<div class="flex items-center justify-center py-8">
								<div class="animate-pulse text-muted-foreground">Loading issue details...</div>
							</div>
						{:else if issueError}
							<div class="rounded-lg border border-destructive/50 bg-destructive/10 p-4">
								<p class="text-sm text-destructive">{issueError}</p>
								<button 
									type="button"
									class="mt-2 text-xs text-primary hover:underline"
									onclick={() => loadIssueDetails($selectedAgent?.beads_id || '', $selectedAgent?.project_dir)}
								>
									Retry
								</button>
							</div>
						{:else if issueDetails}
							<div class="space-y-4">
								<!-- Issue Header -->
								<div class="rounded-lg border bg-muted/20 p-4">
									<div class="flex flex-wrap items-center gap-2 mb-2">
										<span class="font-mono text-sm font-medium text-primary">{issueDetails.id}</span>
										<Badge variant={getIssueStatusVariant(issueDetails.status)}>
											{issueDetails.status}
										</Badge>
										{#if issueDetails.priority !== undefined}
											{@const priorityInfo = getPriorityInfo(issueDetails.priority)}
											<span class={`text-xs font-medium ${priorityInfo.color}`}>{priorityInfo.label}</span>
										{/if}
										{#if issueDetails.issue_type}
											<Badge variant="outline" class="text-xs">{issueDetails.issue_type}</Badge>
										{/if}
									</div>
									<h3 class="text-lg font-semibold">{issueDetails.title}</h3>
									{#if issueDetails.labels && issueDetails.labels.length > 0}
										<div class="flex flex-wrap gap-1 mt-2">
											{#each issueDetails.labels as label}
												<span class="inline-flex items-center px-2 py-0.5 rounded text-xs bg-muted text-muted-foreground">
													{label}
												</span>
											{/each}
										</div>
									{/if}
								</div>
								
								<!-- Description -->
								{#if issueDetails.description}
									<div>
										<h4 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">Description</h4>
										<div class="rounded-lg border bg-muted/10 p-3 prose prose-sm dark:prose-invert max-w-none prose-headings:text-sm prose-p:text-sm prose-p:leading-relaxed">
											{@html marked(issueDetails.description)}
										</div>
									</div>
								{/if}
								
								<!-- Parent/Child Relationships -->
								{#if issueDetails.parent || (issueDetails.children && issueDetails.children.length > 0)}
									<div>
										<h4 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">Relationships</h4>
										<div class="rounded-lg border bg-muted/10 p-3 space-y-2">
											{#if issueDetails.parent}
												<div class="flex items-center gap-2 text-sm">
													<span class="text-muted-foreground">Parent:</span>
													<span class="font-mono text-xs text-primary">{issueDetails.parent.id}</span>
													{#if issueDetails.parent.title}
														<span class="truncate">{issueDetails.parent.title}</span>
													{/if}
													{#if issueDetails.parent.status}
														<Badge variant="outline" class="text-xs shrink-0">{issueDetails.parent.status}</Badge>
													{/if}
												</div>
											{/if}
											{#if issueDetails.children && issueDetails.children.length > 0}
												<div class="space-y-1">
													<span class="text-sm text-muted-foreground">Children ({issueDetails.children.length}):</span>
													{#each issueDetails.children as child}
														<div class="flex items-center gap-2 text-sm pl-4">
															<span class="font-mono text-xs text-primary">{child.id}</span>
															{#if child.title}
																<span class="truncate flex-1">{child.title}</span>
															{/if}
															{#if child.status}
																<Badge variant="outline" class="text-xs shrink-0">{child.status}</Badge>
															{/if}
														</div>
													{/each}
												</div>
											{/if}
										</div>
									</div>
								{/if}
								
								<!-- Comments Timeline -->
								{#if issueDetails.comments && issueDetails.comments.length > 0}
									<div>
										<h4 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">
											Timeline ({issueDetails.comments.length} comments)
										</h4>
										<div class="rounded-lg border bg-muted/10 overflow-hidden">
											<div class="divide-y divide-border/50">
												{#each issueDetails.comments as comment}
													<div 
														class="p-3 text-sm {comment.is_phase ? 'bg-blue-500/5' : ''} {comment.is_phase || comment.is_blocked || comment.is_question ? 'border-l-2' : ''} {comment.is_phase && !comment.is_blocked && !comment.is_question ? 'border-l-blue-500' : ''} {comment.is_blocked ? 'border-l-red-500' : ''} {comment.is_question ? 'border-l-yellow-500' : ''}"
													>
														<div class="flex items-center justify-between mb-1">
															<div class="flex items-center gap-2">
																{#if comment.is_phase}
																	<span class="text-blue-500 font-medium">Phase: {comment.phase}</span>
																{:else if comment.is_blocked}
																	<span class="text-red-500 font-medium">BLOCKED</span>
																{:else if comment.is_question}
																	<span class="text-yellow-500 font-medium">QUESTION</span>
																{:else}
																	<span class="text-muted-foreground">{comment.author || 'Agent'}</span>
																{/if}
															</div>
															<span class="text-xs text-muted-foreground">
																{formatRelativeTime(comment.created_at)}
															</span>
														</div>
														<p class="text-muted-foreground whitespace-pre-wrap break-words">
															{comment.text}
														</p>
													</div>
												{/each}
											</div>
										</div>
									</div>
								{:else}
									<div class="text-center py-4 text-muted-foreground">
										<p class="text-sm">No comments yet</p>
									</div>
								{/if}
								
								<!-- Quick Commands -->
								<div>
									<h4 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">Commands</h4>
									<div class="flex flex-wrap gap-2">
										<button
											type="button"
											class="group flex items-center gap-2 rounded-lg border bg-muted/30 px-3 py-2 text-left transition-all hover:bg-muted hover:border-primary/50 active:scale-[0.98]"
											onclick={() => copyToClipboard(`bd show ${$selectedAgent?.beads_id}`, 'show')}
										>
											<span class="text-sm">📋</span>
											<code class="text-xs text-muted-foreground">bd show {$selectedAgent?.beads_id}</code>
											<span class="text-xs text-muted-foreground group-hover:text-foreground transition-colors">
												{copiedItem === 'show' ? '✓' : '📋'}
											</span>
										</button>
									</div>
								</div>
							</div>
						{/if}
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
					<!-- Spawn Metadata Summary -->
					<div>
						<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2">Spawn Metadata</h3>
						<div class="rounded-lg border bg-muted/20 p-3">
							<dl class="grid grid-cols-2 gap-3 text-sm">
								{#if spawnContext?.metadata.task || $selectedAgent.task}
									<div class="col-span-2">
										<dt class="text-muted-foreground text-xs">Task</dt>
										<dd class="font-medium">{spawnContext?.metadata.task || $selectedAgent.task}</dd>
									</div>
								{/if}
								{#if $selectedAgent.skill || spawnContext?.metadata.skill}
									<div>
										<dt class="text-muted-foreground text-xs">Skill</dt>
										<dd class="font-medium">{$selectedAgent.skill || spawnContext?.metadata.skill}</dd>
									</div>
								{/if}
								{#if $selectedAgent.beads_id || spawnContext?.metadata.beads_id}
									<div>
										<dt class="text-muted-foreground text-xs">Beads Issue</dt>
										<dd class="font-mono text-xs">{$selectedAgent.beads_id || spawnContext?.metadata.beads_id}</dd>
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
								{#if spawnContext?.metadata.spawn_tier}
									<div>
										<dt class="text-muted-foreground text-xs">Spawn Tier</dt>
										<dd class="font-medium capitalize">{spawnContext.metadata.spawn_tier}</dd>
									</div>
								{/if}
								{#if spawnContext?.metadata.session_scope}
									<div>
										<dt class="text-muted-foreground text-xs">Session Scope</dt>
										<dd class="font-medium">{spawnContext.metadata.session_scope}</dd>
									</div>
								{/if}
								{#if spawnContext?.workspace_path}
									<div class="col-span-2">
										<dt class="text-muted-foreground text-xs">Workspace Path</dt>
										<dd class="font-mono text-xs truncate">{spawnContext.workspace_path}</dd>
									</div>
								{/if}
								{#if spawnContext?.metadata.project_dir}
									<div class="col-span-2">
										<dt class="text-muted-foreground text-xs">Project Directory</dt>
										<dd class="font-mono text-xs truncate">{spawnContext.metadata.project_dir}</dd>
									</div>
								{/if}
							</dl>
						</div>
					</div>
					
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
					{/if}
					
					<!-- SPAWN_CONTEXT.md Content -->
					<div>
						<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2 flex items-center gap-2">
							<span>📦</span> SPAWN_CONTEXT.md
						</h3>
						{#if spawnContextLoading}
							<div class="flex items-center justify-center py-8">
								<div class="animate-pulse text-muted-foreground">Loading spawn context...</div>
							</div>
						{:else if spawnContext?.error}
							<div class="rounded-lg border border-muted bg-muted/10 p-4 text-center">
								<p class="text-sm text-muted-foreground">{spawnContext.error}</p>
							</div>
						{:else if spawnContext?.content}
							<div class="rounded-lg border bg-muted/10 overflow-hidden">
								<div class="max-h-[500px] overflow-y-auto p-4 prose prose-sm dark:prose-invert max-w-none 
									prose-headings:text-sm prose-headings:font-semibold prose-headings:mt-4 prose-headings:mb-2
									prose-p:text-sm prose-p:leading-relaxed prose-p:my-1
									prose-ul:my-1 prose-li:my-0.5
									prose-code:text-xs prose-code:bg-muted prose-code:px-1 prose-code:py-0.5 prose-code:rounded
									prose-pre:bg-muted/50 prose-pre:text-xs">
									{@html marked(spawnContext.content)}
								</div>
							</div>
						{:else}
							<div class="text-center py-8 text-muted-foreground">
								<p class="text-sm">No spawn context available</p>
								<p class="text-xs mt-1 opacity-75">This agent may not have a workspace yet</p>
							</div>
						{/if}
					</div>
				</div>
			{/if}

			<!-- Activity Tab -->
			{#if activeTab === 'activity'}
				<div class="p-4 flex flex-col h-full">
					{#if $selectedAgent.status === 'active'}
						<!-- Live Activity for Active Agents - Claude Code style -->
						<div class="flex flex-col h-full">
							<!-- Activity Stream - chronological order (oldest at top, newest at bottom) -->
							<div class="flex-1">
								<div 
									bind:this={activityContainer}
									onscroll={handleActivityScroll}
									class="h-[calc(100vh-320px)] space-y-1 overflow-y-auto rounded border bg-muted/20 p-2 text-sm"
								>
									{#each groupedEvents as group (group.id)}
										{#if group.type === 'tool'}
											<!-- Tool invocation - expandable block -->
											<button
												type="button"
												class="w-full flex items-start gap-2 py-1.5 px-2 text-left rounded transition-colors hover:bg-muted/50 border-l-2 {group.status?.class || 'border-blue-500/30'}"
												onclick={() => toggleGroup(group.id)}
											>
												<span class="shrink-0 mt-0.5">{getActivityIcon(group.primary.properties?.part?.type, group.toolName)}</span>
												<span class="flex-1 break-words leading-relaxed font-medium">{group.label}</span>
												{#if group.status}
													<span class="{group.status.class} shrink-0">{group.status.text}</span>
												{/if}
												<span class="shrink-0 text-muted-foreground text-xs">{group.expanded ? '▼' : '▶'}</span>
											</button>
											
											{#if group.expanded && group.related.length > 0}
												<div class="ml-6 pl-2 border-l border-muted-foreground/20 space-y-1 py-1">
													{#each group.related as related}
														{@const relatedPart = related.properties?.part}
														{#if relatedPart}
															<div class="text-xs text-muted-foreground py-0.5">
																{relatedPart.state?.output 
																	? relatedPart.state.output.substring(0, 200) + (relatedPart.state.output.length > 200 ? '...' : '')
																	: getHumanReadableLabel(relatedPart)}
															</div>
														{/if}
													{/each}
												</div>
											{/if}
										{:else if group.type === 'text'}
											<!-- Text/reasoning message - render markdown -->
											<div class="flex items-start gap-2 py-1.5 px-2 rounded hover:bg-muted/50">
												<span class="shrink-0 mt-0.5">{getActivityIcon(group.primary.properties?.part?.type)}</span>
												<div class="flex-1 break-words leading-relaxed prose prose-sm dark:prose-invert max-w-none prose-p:my-1 prose-headings:my-1">
													{@html renderMarkdown(group.primary.properties?.part?.text || '')}
												</div>
											</div>
										{:else}
											<!-- Step events - compact display -->
											<div class="flex items-start gap-2 py-0.5 px-2 text-xs text-muted-foreground">
												<span class="shrink-0">{getActivityIcon(group.primary.properties?.part?.type)}</span>
												<span class="flex-1 break-words leading-relaxed">{group.label}</span>
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
								
								<!-- Scroll indicator when auto-scroll is disabled -->
								{#if !shouldAutoScroll}
									<button
										type="button"
										class="w-full py-1 text-xs text-center text-primary hover:underline"
										onclick={() => { shouldAutoScroll = true; scrollToBottom(); }}
									>
										↓ New messages - click to scroll to bottom
									</button>
								{/if}
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
				<div class="p-4 flex-1 overflow-y-auto">
					{#if $selectedAgent.status === 'completed' || $selectedAgent.phase === 'Complete'}
						<div class="space-y-4">
							<!-- Synthesis Section (Primary) -->
							<div>
								<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2 flex items-center gap-2">
									<span>📝</span> Synthesis
								</h3>
								<div class="rounded-lg border bg-muted/20 overflow-hidden">
									<div class="max-h-[300px] overflow-y-auto">
										<ArtifactViewer 
											workspaceId={extractWorkspaceName($selectedAgent.id)}
											beadsId={$selectedAgent.beads_id}
											skill={$selectedAgent.skill}
											closeReason={$selectedAgent.close_reason}
										/>
									</div>
								</div>
							</div>
							
							<!-- Delta Section -->
							{#if deliverables && (deliverables.file_delta.created.length > 0 || deliverables.file_delta.modified.length > 0 || deliverables.file_delta.deleted.length > 0)}
								<div>
									<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2 flex items-center gap-2">
										<span>📊</span> File Changes
										<span class="text-[10px] font-normal">
											({deliverables.file_delta.created.length + deliverables.file_delta.modified.length + deliverables.file_delta.deleted.length} files)
										</span>
									</h3>
									<div class="rounded-lg border bg-muted/20 p-3 space-y-2">
										{#if deliverables.file_delta.created.length > 0}
											<div>
												<span class="text-xs text-green-500 font-medium">+ Created ({deliverables.file_delta.created.length})</span>
												<ul class="mt-1 space-y-0.5">
													{#each deliverables.file_delta.created.slice(0, 10) as file}
														<li class="text-xs font-mono text-green-600 dark:text-green-400 truncate">{file}</li>
													{/each}
													{#if deliverables.file_delta.created.length > 10}
														<li class="text-xs text-muted-foreground">...and {deliverables.file_delta.created.length - 10} more</li>
													{/if}
												</ul>
											</div>
										{/if}
										{#if deliverables.file_delta.modified.length > 0}
											<div>
												<span class="text-xs text-yellow-500 font-medium">~ Modified ({deliverables.file_delta.modified.length})</span>
												<ul class="mt-1 space-y-0.5">
													{#each deliverables.file_delta.modified.slice(0, 10) as file}
														<li class="text-xs font-mono text-yellow-600 dark:text-yellow-400 truncate">{file}</li>
													{/each}
													{#if deliverables.file_delta.modified.length > 10}
														<li class="text-xs text-muted-foreground">...and {deliverables.file_delta.modified.length - 10} more</li>
													{/if}
												</ul>
											</div>
										{/if}
										{#if deliverables.file_delta.deleted.length > 0}
											<div>
												<span class="text-xs text-red-500 font-medium">- Deleted ({deliverables.file_delta.deleted.length})</span>
												<ul class="mt-1 space-y-0.5">
													{#each deliverables.file_delta.deleted.slice(0, 10) as file}
														<li class="text-xs font-mono text-red-600 dark:text-red-400 truncate">{file}</li>
													{/each}
													{#if deliverables.file_delta.deleted.length > 10}
														<li class="text-xs text-muted-foreground">...and {deliverables.file_delta.deleted.length - 10} more</li>
													{/if}
												</ul>
											</div>
										{/if}
									</div>
								</div>
							{/if}
							
							<!-- Commits Section -->
							{#if deliverables && deliverables.commits.length > 0}
								<div>
									<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2 flex items-center gap-2">
										<span>📦</span> Commits ({deliverables.commits.length})
									</h3>
									<div class="rounded-lg border bg-muted/20 divide-y divide-muted-foreground/10">
										{#each deliverables.commits as commit}
											<div class="p-2 hover:bg-muted/30 transition-colors">
												<div class="flex items-start gap-2">
													<span class="font-mono text-xs text-primary shrink-0">{commit.hash}</span>
													<span class="text-sm flex-1 truncate">{commit.message}</span>
													<span class="text-xs text-muted-foreground shrink-0">{formatCommitTime(commit.timestamp)}</span>
												</div>
												<div class="mt-1 flex items-center gap-2 text-xs text-muted-foreground">
													<span>{commit.author}</span>
													{#if commit.files_changed > 0}
														<span>•</span>
														<span>{commit.files_changed} file{commit.files_changed > 1 ? 's' : ''}</span>
													{/if}
												</div>
											</div>
										{/each}
									</div>
								</div>
							{/if}
							
							<!-- Artifacts Section -->
							{#if deliverables && deliverables.artifacts.length > 0}
								<div>
									<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-2 flex items-center gap-2">
										<span>📋</span> Artifacts ({deliverables.artifacts.length})
									</h3>
									<div class="rounded-lg border bg-muted/20 divide-y divide-muted-foreground/10">
										{#each deliverables.artifacts as artifact}
											<div class="p-2 flex items-center gap-2">
												<Badge variant="outline" class="shrink-0 capitalize">{artifact.type}</Badge>
												<span class="text-sm font-mono truncate flex-1">{artifact.name}</span>
											</div>
										{/each}
									</div>
								</div>
							{/if}
							
							<!-- Loading state -->
							{#if deliverablesLoading}
								<div class="text-center py-4 text-muted-foreground">
									<span class="animate-pulse">Loading deliverables...</span>
								</div>
							{/if}
						
							<!-- Next Actions from Synthesis -->
							{#if $selectedAgent.synthesis?.next_actions && $selectedAgent.synthesis.next_actions.length > 0}
								<div class="pt-4 border-t">
									<div class="flex items-center justify-between mb-2">
										<h3 class="text-xs font-medium text-muted-foreground uppercase tracking-wide flex items-center gap-2">
											<span>🎯</span> Next Actions
										</h3>
										{#if issueCreationError}
											<span class="text-xs text-red-500">{issueCreationError}</span>
										{:else if createdIssueId}
											<span class="text-xs text-green-500">Created {createdIssueId}</span>
										{/if}
									</div>
									<ul class="space-y-1">
										{#each $selectedAgent.synthesis.next_actions as action}
											<li class="flex items-start gap-2 rounded p-1.5 hover:bg-muted/50 group border border-transparent hover:border-muted-foreground/20">
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
						</div>
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
