<script lang="ts">
	import type { TimelineAction, ActionType } from '$lib/stores/timeline';

	export let action: TimelineAction;

	// Action icon by type
	function getActionIcon(type: ActionType): string {
		switch (type) {
			case 'issue_created': return '●';
			case 'issue_completed': return '✓';
			case 'issue_closed': return '✓';
			case 'issue_released': return '↗';
			case 'agent_spawned': return '▶';
			case 'agent_completed': return '■';
			case 'decision_made': return '★';
			case 'quick_decision': return '◆';
			case 'session_started': return '⚡';
			case 'session_ended': return '⏸';
			case 'session_labeled': return '🏷';
			default: return '◦';
		}
	}

	// Color by type
	function getActionColor(type: ActionType): string {
		switch (type) {
			case 'issue_created': return 'text-orange-400';
			case 'issue_completed': return 'text-green-400';
			case 'issue_closed': return 'text-green-400';
			case 'issue_released': return 'text-blue-400';
			case 'agent_spawned': return 'text-cyan-400';
			case 'agent_completed': return 'text-cyan-500';
			case 'decision_made': return 'text-yellow-400';
			case 'quick_decision': return 'text-purple-400';
			case 'session_started': return 'text-white';
			case 'session_ended': return 'text-gray-500';
			case 'session_labeled': return 'text-blue-300';
			default: return 'text-gray-500';
		}
	}

	// Format timestamp
	function formatTimestamp(timestamp: string): string {
		const date = new Date(timestamp);
		return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
	}

	$: icon = getActionIcon(action.type);
	$: color = getActionColor(action.type);
	$: time = formatTimestamp(action.timestamp);
</script>

<div class="timeline-action flex items-start gap-3 px-2 py-1.5 hover:bg-zinc-800/30 rounded text-sm">
	<!-- Time -->
	<span class="text-xs text-gray-500 font-mono w-14 flex-shrink-0 text-right">
		{time}
	</span>

	<!-- Icon -->
	<span class="text-base {color} flex-shrink-0 w-5 text-center">
		{icon}
	</span>

	<!-- Title -->
	<div class="flex-1 min-w-0">
		<span class="text-gray-200 truncate block">
			{action.title}
		</span>
		{#if action.beads_id}
			<span class="text-xs text-gray-500 font-mono">
				{action.beads_id}
			</span>
		{/if}
	</div>
</div>
