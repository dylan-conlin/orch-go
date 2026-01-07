<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import type { Agent, SSEEvent } from '$lib/stores/agents';
	import { sseEvents } from '$lib/stores/agents';
	import { onMount, tick } from 'svelte';

	// Props
	interface Props {
		agent: Agent;
	}

	let { agent }: Props = $props();

	// Event limit - increased from 50 to 100
	const EVENT_LIMIT = 100;

	// Auto-scroll state - persisted in localStorage
	let autoScroll = $state(true);
	let scrollContainer: HTMLDivElement | null = null;

	// Message type filter state - which types to show
	type MessageType = 'text' | 'tool' | 'reasoning' | 'step';
	let enabledTypes = $state<Set<MessageType>>(new Set(['text', 'tool', 'reasoning', 'step']));

	// Load auto-scroll preference from localStorage on mount
	onMount(() => {
		const stored = localStorage.getItem('activityTab.autoScroll');
		if (stored !== null) {
			autoScroll = stored === 'true';
		}
	});

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

	// Filter SSE events for this agent's session
	// Note: For message.part events, sessionID is nested at properties.part.sessionID
	let agentEvents = $derived(agent?.session_id 
		? $sseEvents.filter(e => {
			// Only include message.part events
			if (e.type !== 'message.part' && e.type !== 'message.part.updated') return false;
			// Check session ID match
			const eventSessionId = e.properties?.part?.sessionID || e.properties?.sessionID;
			if (eventSessionId !== agent?.session_id) return false;
			// Apply type filter
			const partType = e.properties?.part?.type;
			const category = getFilterCategory(partType);
			if (category && !enabledTypes.has(category)) return false;
			return true;
		}).slice(-EVENT_LIMIT)
		: []);

	// Auto-scroll to bottom when new events arrive
	$effect(() => {
		if (autoScroll && scrollContainer && agentEvents.length > 0) {
			// Use tick to ensure DOM is updated before scrolling
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
		// If user scrolled away from bottom, disable auto-scroll
		// If they scroll back to bottom, re-enable it
		if (!isAtBottom && autoScroll) {
			autoScroll = false;
		} else if (isAtBottom && !autoScroll) {
			autoScroll = true;
		}
	}
</script>

<div class="p-4">
	<!-- Header with controls -->
	<div class="mb-3 flex items-center justify-between gap-2 flex-wrap">
		<div class="flex items-center gap-2">
			<h3 class="text-sm font-medium text-muted-foreground">Live Activity</h3>
			{#if agent.is_processing}
				<Badge variant="secondary" class="animate-pulse">
					Processing
				</Badge>
			{/if}
		</div>
		
		<!-- Controls -->
		<div class="flex items-center gap-2">
			<!-- Message type filters -->
			<div class="flex items-center gap-1 border rounded-md p-0.5">
				<button
					type="button"
					class="px-2 py-0.5 text-xs rounded transition-colors {enabledTypes.has('text') ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'}"
					onclick={() => toggleType('text')}
					title="Show text messages"
				>
					💬
				</button>
				<button
					type="button"
					class="px-2 py-0.5 text-xs rounded transition-colors {enabledTypes.has('tool') ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'}"
					onclick={() => toggleType('tool')}
					title="Show tool invocations"
				>
					🔧
				</button>
				<button
					type="button"
					class="px-2 py-0.5 text-xs rounded transition-colors {enabledTypes.has('reasoning') ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'}"
					onclick={() => toggleType('reasoning')}
					title="Show reasoning"
				>
					🤔
				</button>
				<button
					type="button"
					class="px-2 py-0.5 text-xs rounded transition-colors {enabledTypes.has('step') ? 'bg-muted text-foreground' : 'text-muted-foreground hover:text-foreground'}"
					onclick={() => toggleType('step')}
					title="Show step transitions"
				>
					▶️
				</button>
			</div>
			
			<!-- Auto-scroll toggle -->
			<button
				type="button"
				class="px-2 py-0.5 text-xs rounded border transition-colors {autoScroll ? 'bg-primary/10 text-primary border-primary/50' : 'text-muted-foreground hover:text-foreground'}"
				onclick={() => autoScroll = !autoScroll}
				title="Toggle auto-scroll"
			>
				{autoScroll ? '⬇️ Auto' : '⬇️'}
			</button>
		</div>
	</div>
	
	<!-- Current Activity - styling varies by activity type -->
	{#if agent.current_activity}
		<div class="mb-3 rounded-lg border {getActivityStyle(agent.current_activity.type)} p-3">
			<div class="flex items-start gap-2">
				<span class="text-lg">{getActivityIcon(agent.current_activity.type)}</span>
				<div class="flex-1 min-w-0">
					<p class="text-sm font-medium">{agent.current_activity.text || 'Working...'}</p>
					<span class="text-xs text-muted-foreground">
						{agent.current_activity.type}
					</span>
				</div>
			</div>
		</div>
	{/if}

	<!-- Activity Log - scrollable with more height -->
	<div 
		bind:this={scrollContainer}
		onscroll={handleScroll}
		class="max-h-96 space-y-1 overflow-y-auto rounded border bg-muted/20 p-2 font-mono text-xs"
	>
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
	
	<!-- Event count indicator -->
	<div class="mt-2 text-xs text-muted-foreground text-right">
		{agentEvents.length} / {EVENT_LIMIT} events
	</div>
</div>
