<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import type { Agent, SSEEvent } from '$lib/stores/agents';
	import { sseEvents, sessionHistory } from '$lib/stores/agents';
	import { onMount, tick } from 'svelte';

	// Props
	interface Props {
		agent: Agent;
	}

	let { agent }: Props = $props();

	// Event limit per agent
	const EVENT_LIMIT = 500;

	// Auto-scroll state - persisted in localStorage
	let autoScroll = $state(true);
	let scrollContainer: HTMLDivElement | null = null;

	// Message type filter state - which types to show
	type MessageType = 'text' | 'tool' | 'reasoning' | 'step';
	let enabledTypes = $state<Set<MessageType>>(new Set(['text', 'tool', 'reasoning', 'step']));

	// Loading state for historical events
	let historyLoading = $state(false);
	let historyError = $state<string | null>(null);
	let historicalEvents = $state<SSEEvent[]>([]);

	// Track current session to detect agent changes
	let currentSessionId = $state<string | null>(null);

	// Load auto-scroll preference from localStorage on mount
	onMount(() => {
		const stored = localStorage.getItem('activityTab.autoScroll');
		if (stored !== null) {
			autoScroll = stored === 'true';
		}
	});

	// Fetch historical events when session changes
	$effect(() => {
		const sessionId = agent?.session_id;
		if (sessionId && sessionId !== currentSessionId) {
			currentSessionId = sessionId;
			fetchHistoricalEvents(sessionId);
		}
	});

	async function fetchHistoricalEvents(sessionId: string) {
		if (!sessionId) return;
		
		// Check cache first
		const cached = sessionHistory.getState(sessionId);
		if (cached?.loaded) {
			historicalEvents = cached.events;
			historyLoading = false;
			historyError = null;
			return;
		}
		
		historyLoading = true;
		historyError = null;
		
		try {
			const events = await sessionHistory.fetchHistory(sessionId);
			historicalEvents = events;
			historyError = null;
		} catch (error) {
			historyError = error instanceof Error ? error.message : 'Failed to load history';
			historicalEvents = [];
		} finally {
			historyLoading = false;
		}
	}

	// Save auto-scroll preference when it changes
	$effect(() => {
		localStorage.setItem('activityTab.autoScroll', String(autoScroll));
	});

	// Activity icon helper
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

	// Format tool name with capitalized first letter (e.g., "bash" -> "Bash")
	function formatToolName(tool: string): string {
		if (!tool) return 'Tool';
		return tool.charAt(0).toUpperCase() + tool.slice(1);
	}

	// Extract the most relevant argument from tool input for display
	// Returns a short string suitable for inline display
	function extractToolArg(input: unknown): string {
		if (!input || typeof input !== 'object') return '';
		const inp = input as Record<string, unknown>;
		
		// Bash: show command
		if (inp.command && typeof inp.command === 'string') {
			return inp.command;
		}
		// Read/Write/Edit: show file path
		if (inp.filePath && typeof inp.filePath === 'string') {
			return inp.filePath;
		}
		// Glob: show pattern
		if (inp.pattern && typeof inp.pattern === 'string') {
			return inp.pattern;
		}
		// Grep: show pattern (search term)
		if (inp.pattern && typeof inp.pattern === 'string') {
			return inp.pattern;
		}
		// WebFetch: show URL
		if (inp.url && typeof inp.url === 'string') {
			return inp.url;
		}
		// Task: show description
		if (inp.description && typeof inp.description === 'string') {
			return inp.description;
		}
		// Generic: try common field names
		const commonFields = ['name', 'path', 'query', 'text', 'content', 'selector'];
		for (const field of commonFields) {
			if (inp[field] && typeof inp[field] === 'string') {
				return inp[field] as string;
			}
		}
		return '';
	}

	// Truncate text with ellipsis, respecting max length
	function truncate(text: string, maxLen: number): string {
		if (!text || text.length <= maxLen) return text;
		return text.slice(0, maxLen - 1) + '…';
	}

	// Part type extracted from SSEEvent for type safety
	type Part = NonNullable<NonNullable<SSEEvent['properties']>['part']>;
	
	// Format a tool call for display: "ToolName(arg)" or "ToolName" if no args
	// Full arg is available via title attribute for tooltip
	function formatToolCall(part: Part | undefined): { display: string; full: string } {
		if (!part) return { display: 'Tool', full: '' };
		
		const toolName = formatToolName(part.tool || 'tool');
		const arg = extractToolArg(part.state?.input);
		const title = part.state?.title;
		
		// If we have a title, use it as the full description
		const fullDescription = title || arg || '';
		
		if (!arg) {
			return { display: toolName, full: fullDescription };
		}
		
		// Truncate arg for display (keep it readable at 666px width)
		// Use shorter truncation - most screen space is limited
		const truncatedArg = truncate(arg, 60);
		const display = `${toolName}(${truncatedArg})`;
		
		return { display, full: arg };
	}

	// Map event type to filter category
	function getFilterCategory(type?: string): MessageType | null {
		switch (type) {
			case 'text': return 'text';
			case 'tool':
			case 'tool-invocation': return 'tool';
			case 'reasoning': return 'reasoning';
			case 'step-start':
			case 'step-finish': return 'step';
			default: return null;
		}
	}

	// Toggle a message type filter
	function toggleType(type: MessageType) {
		const newSet = new Set(enabledTypes);
		if (newSet.has(type)) {
			newSet.delete(type);
		} else {
			newSet.add(type);
		}
		enabledTypes = newSet;
	}

	// Filter events by type and session
	function filterEvents(events: SSEEvent[], sessionId: string | undefined): SSEEvent[] {
		if (!sessionId) return [];
		return events.filter(e => {
			if (e.type !== 'message.part' && e.type !== 'message.part.updated') return false;
			const eventSessionId = e.properties?.part?.sessionID || e.properties?.sessionID;
			if (eventSessionId !== sessionId) return false;
			const partType = e.properties?.part?.type;
			const category = getFilterCategory(partType);
			if (category && !enabledTypes.has(category)) return false;
			return true;
		});
	}

	// Filter SSE events for this agent's session (real-time events)
	let sseFilteredEvents = $derived(filterEvents($sseEvents, agent?.session_id));

	// Filter historical events
	let historyFilteredEvents = $derived(filterEvents(historicalEvents, agent?.session_id));

	// Merge historical and SSE events, deduplicating by ID
	// Historical events come first, SSE events appended (real-time updates)
	let mergedEvents = $derived(() => {
		const seenIds = new Set<string>();
		const merged: SSEEvent[] = [];
		
		// Add historical events first
		for (const event of historyFilteredEvents) {
			if (event.id && !seenIds.has(event.id)) {
				seenIds.add(event.id);
				merged.push(event);
			}
		}
		
		// Add SSE events (real-time), deduplicating against historical
		for (const event of sseFilteredEvents) {
			if (event.id && !seenIds.has(event.id)) {
				seenIds.add(event.id);
				merged.push(event);
			}
		}
		
		// Sort by timestamp if available, then limit
		merged.sort((a, b) => (a.timestamp || 0) - (b.timestamp || 0));
		
		return merged.slice(-EVENT_LIMIT);
	});

	// Use the merged events for display
	let agentEvents = $derived(mergedEvents());

	// Auto-scroll to bottom when new events arrive
	$effect(() => {
		if (autoScroll && scrollContainer && agentEvents.length > 0) {
			tick().then(() => {
				if (scrollContainer) {
					scrollContainer.scrollTop = scrollContainer.scrollHeight;
				}
			});
		}
	});

	// Handle manual scroll - disable auto-scroll if user scrolls up
	function handleScroll(event: Event) {
		const target = event.target as HTMLDivElement;
		const isAtBottom = target.scrollHeight - target.scrollTop - target.clientHeight < 50;
		if (!isAtBottom && autoScroll) {
			autoScroll = false;
		} else if (isAtBottom && !autoScroll) {
			autoScroll = true;
		}
	}

	// Expand/collapse state for tool results - keyed by event ID
	let expandedResults = $state<Map<string, boolean>>(new Map());
	
	// Track currently focused/hovered event ID for keyboard shortcuts
	let focusedEventId = $state<string | null>(null);
	
	// Toggle expand/collapse for a specific tool result
	function toggleExpand(eventId: string) {
		const newMap = new Map(expandedResults);
		newMap.set(eventId, !newMap.get(eventId));
		expandedResults = newMap;
	}
	
	// Truncate tool output to first N lines
	function truncateOutput(output: string, maxLines: number = 3): { preview: string; hasMore: boolean; totalLines: number } {
		if (!output) return { preview: '', hasMore: false, totalLines: 0 };
		const lines = output.split('\n');
		const totalLines = lines.length;
		const hasMore = totalLines > maxLines;
		const preview = lines.slice(0, maxLines).join('\n');
		return { preview, hasMore, totalLines };
	}
	
	// Handle keyboard shortcuts
	function handleActivityKeydown(event: KeyboardEvent) {
		// Ctrl+O: Toggle expand/collapse for focused tool result
		if (event.ctrlKey && event.key === 'o' && focusedEventId) {
			event.preventDefault();
			toggleExpand(focusedEventId);
		}
	}
	
	// Set up keyboard event listener on mount
	onMount(() => {
		window.addEventListener('keydown', handleActivityKeydown);
		return () => {
			window.removeEventListener('keydown', handleActivityKeydown);
		};
	});
	
	// Message input state
	let messageInput = $state('');
	let isSending = $state(false);
	let sendError = $state<string | null>(null);

	// Determine if input should be disabled
	let isInputDisabled = $derived(
		agent.status !== 'active' || !agent.session_id || isSending
	);

	// Send message to agent
	async function sendMessage() {
		if (!messageInput.trim() || !agent.session_id || isInputDisabled) {
			return;
		}

		const message = messageInput.trim();
		messageInput = ''; // Clear input immediately
		sendError = null;
		isSending = true;

		try {
			// Get OpenCode server URL from agents store (API_BASE)
			const serverURL = 'http://localhost:4096';
			
			// POST to OpenCode prompt_async endpoint
			const response = await fetch(`${serverURL}/session/${agent.session_id}/prompt_async`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					parts: [{ type: 'text', text: message }],
					agent: 'build',
				}),
			});

			if (!response.ok) {
				const errorText = await response.text();
				throw new Error(`Failed to send message: ${response.status} ${errorText}`);
			}

			// Message sent successfully - SSE will update feed with response
		} catch (error) {
			sendError = error instanceof Error ? error.message : 'Failed to send message';
			messageInput = message; // Restore message on error
		} finally {
			isSending = false;
		}
	}

	// Handle keyboard events for Enter to send, Shift+Enter for newline
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' && !event.shiftKey) {
			event.preventDefault();
			sendMessage();
		}
	}
</script>

<div class="flex flex-col h-full">
	<!-- Header with controls -->
	<div class="p-3 border-b flex items-center justify-between gap-2 flex-wrap shrink-0">
		<div class="flex items-center gap-2">
			<span class="text-xs text-muted-foreground">{agentEvents.length} events</span>
			{#if historyLoading}
				<Badge variant="outline" class="text-xs">
					Loading history...
				</Badge>
			{/if}
			{#if historyError}
				<Badge variant="destructive" class="text-xs">
					{historyError}
				</Badge>
			{/if}
			{#if agent.is_processing}
				<Badge variant="secondary" class="animate-pulse text-xs">
					Processing
				</Badge>
			{/if}
		</div>
		
		<!-- Controls -->
		<div class="flex items-center gap-2">
			<!-- Message type filters -->
			<div class="flex items-center gap-0.5 border rounded p-0.5">
				<button
					type="button"
					class="px-1.5 py-0.5 text-xs rounded transition-colors {enabledTypes.has('text') ? 'bg-muted' : 'opacity-40 hover:opacity-100'}"
					onclick={() => toggleType('text')}
					title="Text"
				>💬</button>
				<button
					type="button"
					class="px-1.5 py-0.5 text-xs rounded transition-colors {enabledTypes.has('tool') ? 'bg-muted' : 'opacity-40 hover:opacity-100'}"
					onclick={() => toggleType('tool')}
					title="Tools"
				>🔧</button>
				<button
					type="button"
					class="px-1.5 py-0.5 text-xs rounded transition-colors {enabledTypes.has('reasoning') ? 'bg-muted' : 'opacity-40 hover:opacity-100'}"
					onclick={() => toggleType('reasoning')}
					title="Reasoning"
				>🤔</button>
				<button
					type="button"
					class="px-1.5 py-0.5 text-xs rounded transition-colors {enabledTypes.has('step') ? 'bg-muted' : 'opacity-40 hover:opacity-100'}"
					onclick={() => toggleType('step')}
					title="Steps"
				>▶️</button>
			</div>
			
			<!-- Auto-scroll toggle -->
			<button
				type="button"
				class="px-1.5 py-0.5 text-xs rounded border transition-colors {autoScroll ? 'bg-primary/10 text-primary border-primary/50' : 'opacity-40 hover:opacity-100'}"
				onclick={() => autoScroll = !autoScroll}
				title="Auto-scroll"
			>⬇️</button>
		</div>
	</div>
	
	<!-- Activity Log - terminal style, new messages at bottom -->
	<div 
		bind:this={scrollContainer}
		onscroll={handleScroll}
		class="flex-1 overflow-y-auto bg-black/20 p-2 font-mono text-xs"
	>
		{#if historyLoading && agentEvents.length === 0}
			<p class="py-4 text-center text-muted-foreground/50">Loading activity history...</p>
		{:else}
			{#each agentEvents as event (event.id)}
				{@const part = event.properties?.part}
				{#if part}
					<div class="flex flex-col gap-1 py-0.5">
						<div class="flex items-start gap-2 text-muted-foreground hover:text-foreground transition-colors">
							<span class="shrink-0 opacity-60">{getActivityIcon(part.type)}</span>
							{#if part.type === 'tool' || part.type === 'tool-invocation'}
								{@const toolDisplay = formatToolCall(part)}
								<span 
									class="flex-1 break-words leading-relaxed font-mono"
									title={toolDisplay.full || undefined}
								>
									<span class="text-blue-400">{formatToolName(part.tool || 'tool')}</span>{#if toolDisplay.full}<span class="text-muted-foreground/70">({truncate(extractToolArg(part.state?.input), 60)})</span>{/if}
								</span>
							{:else}
								<span class="flex-1 break-words leading-relaxed">
									{part.text || part.state?.title || part.type}
								</span>
							{/if}
						</div>
						
						<!-- Tool result output -->
						{#if (part.type === 'tool' || part.type === 'tool-invocation') && part.state?.output}
							{@const isExpanded = expandedResults.get(event.id) || false}
							{@const truncated = truncateOutput(part.state.output, 3)}
							<div class="ml-6 text-xs">
								<button
									type="button"
									onclick={() => toggleExpand(event.id)}
									onfocus={() => focusedEventId = event.id}
									onmouseenter={() => focusedEventId = event.id}
									onmouseleave={() => focusedEventId = null}
									class="w-full text-left hover:bg-muted/20 rounded p-1 transition-colors focus:ring-1 focus:ring-primary"
									title="Click to expand/collapse (or Ctrl+O)"
								>
									<pre class="font-mono text-muted-foreground/80 whitespace-pre-wrap break-words">{isExpanded ? part.state.output : truncated.preview}</pre>
									{#if truncated.hasMore}
										<div class="text-muted-foreground/50 mt-1">
											{isExpanded ? '▲ Click to collapse' : `▼ ... +${truncated.totalLines - 3} lines (click to expand)`}
										</div>
									{/if}
								</button>
							</div>
						{/if}
					</div>
				{/if}
			{:else}
				<p class="py-4 text-center text-muted-foreground/50">Waiting for activity...</p>
			{/each}
		{/if}
	</div>

	<!-- Message Input -->
	<div class="p-3 border-t shrink-0">
		{#if sendError}
			<div class="mb-2 text-xs text-red-500">
				{sendError}
			</div>
		{/if}
		<div class="flex gap-2 items-end">
			<textarea
				bind:value={messageInput}
				onkeydown={handleKeydown}
				disabled={isInputDisabled}
				placeholder={isInputDisabled ? 'Agent not active' : 'Send a message... (Enter to send, Shift+Enter for newline)'}
				class="flex-1 min-h-[40px] max-h-[120px] px-3 py-2 text-sm rounded border bg-background resize-none disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-primary"
				rows="1"
			></textarea>
			<button
				type="button"
				onclick={sendMessage}
				disabled={isInputDisabled || !messageInput.trim()}
				class="px-4 py-2 text-sm rounded bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
				title="Send message"
			>
				{isSending ? 'Sending...' : 'Send'}
			</button>
		</div>
		<p class="mt-1 text-xs text-muted-foreground">
			Enter to send, Shift+Enter for newline
		</p>
	</div>
</div>
