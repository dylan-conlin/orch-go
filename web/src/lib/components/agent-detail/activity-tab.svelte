<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import MarkdownContent from '$lib/components/markdown-content/markdown-content.svelte';
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

	// Group related events (tool + step-finish) for nested display
	// Returns array of groups where each group has:
	// - primary: The main event to display (tool, text, reasoning, etc.)
	// - related: Array of related events (step-finish, step-start) to nest under primary
	type EventGroup = {
		primary: SSEEvent;
		related: SSEEvent[];
	};
	
	function groupToolEvents(events: SSEEvent[]): EventGroup[] {
		const groups: EventGroup[] = [];
		let i = 0;
		
		while (i < events.length) {
			const event = events[i];
			const part = event.properties?.part;
			
			if (!part) {
				i++;
				continue;
			}
			
			// Check if this is a tool event
			if (part.type === 'tool' || part.type === 'tool-invocation') {
				const group: EventGroup = {
					primary: event,
					related: []
				};
				
				// Look ahead for related step events
				let j = i + 1;
				while (j < events.length) {
					const nextEvent = events[j];
					const nextPart = nextEvent.properties?.part;
					
					// Stop if we hit another tool or text event (start of new group)
					if (nextPart?.type === 'tool' || nextPart?.type === 'tool-invocation' || nextPart?.type === 'text') {
						break;
					}
					
					// Include step-start and step-finish as related events
					if (nextPart?.type === 'step-start' || nextPart?.type === 'step-finish') {
						group.related.push(nextEvent);
						j++;
					} else {
						break;
					}
				}
				
				groups.push(group);
				i = j; // Skip past related events
			} else {
				// Non-tool events get their own group (no related events)
				groups.push({
					primary: event,
					related: []
				});
				i++;
			}
		}
		
		return groups;
	}
	
	// Grouped events for rendering
	let groupedEvents = $derived(groupToolEvents(agentEvents));

	// Auto-scroll to bottom when new events arrive
	$effect(() => {
		if (autoScroll && scrollContainer && groupedEvents.length > 0) {
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

	// Image attachment state
	type PendingImage = {
		id: string;
		file: File;
		dataUrl: string;
		base64: string;
		mediaType: string;
	};
	let pendingImages = $state<PendingImage[]>([]);
	let isDragging = $state(false);

	// Determine if input should be disabled
	let isInputDisabled = $derived(
		agent.status !== 'active' || !agent.session_id || isSending
	);

	// Send message to agent
	async function sendMessage() {
		if ((!messageInput.trim() && pendingImages.length === 0) || !agent.session_id || isInputDisabled) {
			return;
		}

		const message = messageInput.trim();
		const imagesToSend = [...pendingImages];
		messageInput = ''; // Clear input immediately
		pendingImages = []; // Clear images immediately
		sendError = null;
		isSending = true;

		try {
			// Get OpenCode server URL from agents store (API_BASE)
			const serverURL = 'http://localhost:4096';
			
			// Build parts array with text and images
			const parts: Array<{ type: string; text?: string; source?: { type: string; media_type: string; data: string } }> = [];
			
			// Add text part if present
			if (message) {
				parts.push({ type: 'text', text: message });
			}
			
			// Add image parts
			for (const img of imagesToSend) {
				parts.push({
					type: 'image',
					source: {
						type: 'base64',
						media_type: img.mediaType,
						data: img.base64
					}
				});
			}
			
			// POST to OpenCode prompt_async endpoint
			const response = await fetch(`${serverURL}/session/${agent.session_id}/prompt_async`, {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					parts,
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
			pendingImages = imagesToSend; // Restore images on error
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

	// Convert image file to base64 and create pending image
	async function processImageFile(file: File): Promise<void> {
		if (!file.type.startsWith('image/')) {
			sendError = 'Only image files are supported';
			return;
		}

		// Check file size (limit to 5MB)
		if (file.size > 5 * 1024 * 1024) {
			sendError = 'Image too large (max 5MB)';
			return;
		}

		try {
			const dataUrl = await readFileAsDataURL(file);
			const base64 = dataUrl.split(',')[1];
			
			const pendingImage: PendingImage = {
				id: `img-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
				file,
				dataUrl,
				base64,
				mediaType: file.type
			};
			
			pendingImages = [...pendingImages, pendingImage];
			sendError = null;
		} catch (error) {
			sendError = error instanceof Error ? error.message : 'Failed to process image';
		}
	}

	// Read file as data URL
	function readFileAsDataURL(file: File): Promise<string> {
		return new Promise((resolve, reject) => {
			const reader = new FileReader();
			reader.onload = () => resolve(reader.result as string);
			reader.onerror = () => reject(new Error('Failed to read file'));
			reader.readAsDataURL(file);
		});
	}

	// Remove pending image
	function removePendingImage(id: string) {
		pendingImages = pendingImages.filter(img => img.id !== id);
	}

	// Handle clipboard paste
	async function handlePaste(event: ClipboardEvent) {
		const items = event.clipboardData?.items;
		if (!items) return;

		for (const item of Array.from(items)) {
			if (item.type.startsWith('image/')) {
				event.preventDefault();
				const file = item.getAsFile();
				if (file) {
					await processImageFile(file);
				}
			}
		}
	}

	// Handle drag over
	function handleDragOver(event: DragEvent) {
		event.preventDefault();
		isDragging = true;
	}

	// Handle drag leave
	function handleDragLeave(event: DragEvent) {
		event.preventDefault();
		isDragging = false;
	}

	// Handle file drop
	async function handleDrop(event: DragEvent) {
		event.preventDefault();
		isDragging = false;

		const files = event.dataTransfer?.files;
		if (!files || files.length === 0) return;

		for (const file of Array.from(files)) {
			if (file.type.startsWith('image/')) {
				await processImageFile(file);
			}
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
			{#each groupedEvents as group (group.primary.id)}
				{@const part = group.primary.properties?.part}
				{#if part}
					<!-- Tool events: collapsible group with primary tool call and related step events -->
					{#if part.type === 'tool' || part.type === 'tool-invocation'}
						{@const toolDisplay = formatToolCall(part)}
						{@const isExpanded = expandedResults.get(group.primary.id) || false}
						{@const hasOutput = part.state?.output}
						{@const truncated = hasOutput && part.state?.output ? truncateOutput(part.state.output, 3) : null}
						
						<div class="flex flex-col gap-1 py-1">
							<!-- Tool header - clickable to expand/collapse -->
							<button
								type="button"
								onclick={() => toggleExpand(group.primary.id)}
								onfocus={() => focusedEventId = group.primary.id}
								onmouseenter={() => focusedEventId = group.primary.id}
								onmouseleave={() => focusedEventId = null}
								class="flex items-start gap-2 hover:bg-muted/10 rounded px-1 py-0.5 text-left w-full transition-colors"
								title="{isExpanded ? 'Click to collapse' : 'Click to expand'} (or Ctrl+O)"
							>
								<span class="shrink-0 text-muted-foreground/50">{isExpanded ? '▼' : '▶'}</span>
								<span
									class="flex-1 break-words leading-relaxed font-mono"
									title={toolDisplay.full || undefined}
								>
									<span class="text-blue-400 font-semibold">{formatToolName(part.tool || 'tool')}</span><!--
									-->{#if part.state?.status === 'pending' || part.state?.status === 'running'}<!--
										--><span class="text-yellow-400/80 ml-1 animate-pulse">…</span><!--
									-->{:else if part.state?.status === 'error'}<!--
										--><span class="text-red-400 ml-1">✗</span><!--
									-->{:else if part.state?.status === 'completed'}<!--
										--><span class="text-green-400/60 ml-1">✓</span><!--
									-->{/if}<!--
									-->{#if toolDisplay.full}<span class="text-muted-foreground/60 font-normal">({truncate(extractToolArg(part.state?.input), 60)})</span>{/if}
								</span>
							</button>
							
							<!-- Expanded content: tool output and related step events -->
							{#if isExpanded}
								<div class="ml-8 flex flex-col gap-1">
									<!-- Tool output -->
									{#if hasOutput && truncated && part.state}
										<div class="text-xs">
											<pre class="font-mono text-muted-foreground/50 whitespace-pre-wrap break-words bg-black/10 rounded p-2">{part.state.output}</pre>
										</div>
									{/if}
									
									<!-- Related step events (step-start, step-finish) -->
									{#each group.related as relatedEvent}
										{@const relatedPart = relatedEvent.properties?.part}
										{#if relatedPart}
											<div class="flex items-start gap-2 text-muted-foreground/50 text-xs">
												<span class="shrink-0 opacity-50">{getActivityIcon(relatedPart.type)}</span>
												<span class="flex-1 font-mono">
													{relatedPart.text || relatedPart.state?.title || relatedPart.type}
												</span>
											</div>
										{/if}
									{/each}
								</div>
							{/if}
						</div>
					{:else if part.type === 'image'}
						<!-- Image events: display inline -->
						<div class="flex flex-col gap-1 py-1">
							{#if part.source && part.source.type === 'base64'}
								<div class="flex items-start gap-2">
									<span class="shrink-0 opacity-60">📷</span>
									<div class="flex-1">
										<img 
											src="data:{part.source.media_type};base64,{part.source.data}" 
											alt="User uploaded"
											class="max-w-md max-h-64 rounded border"
										/>
									</div>
								</div>
							{/if}
						</div>
					{:else}
						<!-- Non-tool events: render with hierarchy based on type -->
						<div class="flex flex-col gap-1 py-1">
							{#if part.type === 'reasoning'}
								<!-- Reasoning: muted color, bullet prefix, standard font (not monospace), markdown rendered -->
								<div class="flex items-start gap-2 text-muted-foreground/50">
									<span class="shrink-0">•</span>
									<div class="flex-1 break-words leading-relaxed font-sans text-sm activity-reasoning">
										<MarkdownContent content={part.text || part.state?.title || part.type || ''} />
									</div>
								</div>
							{:else}
								<!-- Text and other events: highest contrast, standard font, markdown rendered -->
								<div class="flex items-start gap-2 text-foreground">
									<span class="shrink-0 opacity-50">{getActivityIcon(part.type)}</span>
									<div class="flex-1 break-words leading-relaxed font-sans activity-text">
										<MarkdownContent content={part.text || part.state?.title || part.type || ''} />
									</div>
								</div>
							{/if}
						</div>
					{/if}
				{/if}
			{:else}
				<p class="py-4 text-center text-muted-foreground/50">Waiting for activity...</p>
			{/each}
		{/if}
	</div>

	<!-- Message Input -->
	<div 
		class="p-3 border-t shrink-0 relative"
		role="region"
		aria-label="Message input with image upload"
		ondragover={handleDragOver}
		ondragleave={handleDragLeave}
		ondrop={handleDrop}
	>
		<!-- Drag overlay -->
		{#if isDragging}
			<div class="absolute inset-0 z-50 bg-primary/10 border-2 border-dashed border-primary rounded flex items-center justify-center">
				<div class="text-center">
					<p class="text-lg mb-1">📷</p>
					<p class="text-sm text-primary font-medium">Drop image here</p>
				</div>
			</div>
		{/if}

		{#if sendError}
			<div class="mb-2 text-xs text-red-500">
				{sendError}
			</div>
		{/if}

		<!-- Image previews -->
		{#if pendingImages.length > 0}
			<div class="mb-2 flex flex-wrap gap-2">
				{#each pendingImages as img (img.id)}
					<div class="relative group">
						<img 
							src={img.dataUrl} 
							alt="Pending upload"
							class="h-20 w-20 object-cover rounded border"
						/>
						<button
							type="button"
							onclick={() => removePendingImage(img.id)}
							class="absolute -top-1 -right-1 w-5 h-5 rounded-full bg-destructive text-destructive-foreground text-xs flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
							title="Remove image"
						>
							×
						</button>
					</div>
				{/each}
			</div>
		{/if}

		<div class="flex gap-2 items-end">
			<textarea
				bind:value={messageInput}
				onkeydown={handleKeydown}
				onpaste={handlePaste}
				disabled={isInputDisabled}
				placeholder={isInputDisabled ? 'Agent not active' : 'Send a message... (Enter to send, Shift+Enter for newline, Cmd+V to paste image)'}
				class="flex-1 min-h-[40px] max-h-[120px] px-3 py-2 text-sm rounded border bg-background resize-none disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-primary"
				rows="1"
			></textarea>
			<button
				type="button"
				onclick={sendMessage}
				disabled={isInputDisabled || (!messageInput.trim() && pendingImages.length === 0)}
				class="px-4 py-2 text-sm rounded bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
				title="Send message"
			>
				{isSending ? 'Sending...' : 'Send'}
			</button>
		</div>
		<p class="mt-1 text-xs text-muted-foreground">
			Enter to send, Shift+Enter for newline, Cmd+V to paste images, or drag & drop
		</p>
	</div>
</div>

<style>
	/* Compact markdown styles for activity feed - override default MarkdownContent spacing */
	.activity-text :global(.markdown-content p),
	.activity-reasoning :global(.markdown-content p) {
		margin-bottom: 0.25rem;
	}
	
	.activity-text :global(.markdown-content p:last-child),
	.activity-reasoning :global(.markdown-content p:last-child) {
		margin-bottom: 0;
	}
	
	.activity-text :global(.markdown-content ul),
	.activity-text :global(.markdown-content ol),
	.activity-reasoning :global(.markdown-content ul),
	.activity-reasoning :global(.markdown-content ol) {
		margin-top: 0.25rem;
		margin-bottom: 0.25rem;
		padding-left: 1.25rem;
	}
	
	.activity-text :global(.markdown-content li),
	.activity-reasoning :global(.markdown-content li) {
		margin-bottom: 0.125rem;
	}
	
	.activity-text :global(.markdown-content pre),
	.activity-reasoning :global(.markdown-content pre) {
		margin-top: 0.25rem;
		margin-bottom: 0.25rem;
		padding: 0.5rem;
		font-size: 0.75rem;
	}
	
	.activity-text :global(.markdown-content code),
	.activity-reasoning :global(.markdown-content code) {
		font-size: 0.75rem;
		padding: 0.125rem 0.25rem;
	}
	
	/* Muted styles for reasoning content */
	.activity-reasoning :global(.markdown-content),
	.activity-reasoning :global(.markdown-content p),
	.activity-reasoning :global(.markdown-content li),
	.activity-reasoning :global(.markdown-content span) {
		color: inherit;
	}
</style>
