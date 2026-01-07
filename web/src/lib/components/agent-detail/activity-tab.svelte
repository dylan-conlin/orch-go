<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import type { Agent, SSEEvent } from '$lib/stores/agents';
	import { sseEvents } from '$lib/stores/agents';
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
	let agentEvents = $derived(agent?.session_id 
		? $sseEvents.filter(e => {
			if (e.type !== 'message.part' && e.type !== 'message.part.updated') return false;
			const eventSessionId = e.properties?.part?.sessionID || e.properties?.sessionID;
			if (eventSessionId !== agent?.session_id) return false;
			const partType = e.properties?.part?.type;
			const category = getFilterCategory(partType);
			if (category && !enabledTypes.has(category)) return false;
			return true;
		}).slice(-EVENT_LIMIT)
		: []);

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
</script>

<div class="flex flex-col h-full">
	<!-- Header with controls -->
	<div class="p-3 border-b flex items-center justify-between gap-2 flex-wrap shrink-0">
		<div class="flex items-center gap-2">
			<span class="text-xs text-muted-foreground">{agentEvents.length} events</span>
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
		{#each agentEvents as event (event.id)}
			{@const part = event.properties?.part}
			{#if part}
				<div class="flex items-start gap-2 py-0.5 text-muted-foreground hover:text-foreground transition-colors">
					<span class="shrink-0 opacity-60">{getActivityIcon(part.type)}</span>
					<span class="flex-1 break-words leading-relaxed">
						{part.text || part.state?.title || (part.tool ? `Using ${part.tool}` : part.type)}
					</span>
				</div>
			{/if}
		{:else}
			<p class="py-4 text-center text-muted-foreground/50">Waiting for activity...</p>
		{/each}
	</div>
</div>
