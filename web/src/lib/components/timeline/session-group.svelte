<script lang="ts">
	import type { SessionGroup } from '$lib/stores/timeline';
	import TimelineAction from './timeline.svelte';

	export let session: SessionGroup;
	export let expanded: boolean = true;
	export let onToggle: () => void;

	// Format date range
	function formatDateRange(start: string, end: string): string {
		const startDate = new Date(start);
		const endDate = new Date(end);
		
		const startTime = startDate.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
		const endTime = endDate.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
		
		// If same day, show date once
		if (startDate.toDateString() === endDate.toDateString()) {
			const date = startDate.toLocaleDateString([], { month: 'short', day: 'numeric' });
			return `${date}, ${startTime} - ${endTime}`;
		}
		
		// Different days
		const startFull = startDate.toLocaleDateString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
		const endFull = endDate.toLocaleDateString([], { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
		return `${startFull} - ${endFull}`;
	}

	$: timeRange = formatDateRange(session.start_time, session.end_time);
	$: displayName = session.label || session.session_id;
</script>

<div class="session-group mb-6">
	<!-- Session header -->
	<button
		type="button"
		onclick={onToggle}
		class="w-full text-left px-2 py-2 hover:bg-zinc-800/50 flex items-center gap-2 rounded border-b border-zinc-800"
	>
		<!-- Expand/collapse indicator -->
		<span class="text-xs text-gray-500 w-3">
			{expanded ? '▼' : '▶'}
		</span>

		<!-- Session name/label -->
		<div class="flex-1 min-w-0">
			<div class="text-gray-200 font-medium truncate">
				{displayName}
			</div>
			<div class="text-xs text-gray-500">
				{timeRange} · {session.action_count} {session.action_count === 1 ? 'action' : 'actions'}
			</div>
		</div>
	</button>

	<!-- Actions (when expanded) -->
	{#if expanded}
		<div class="mt-2 pl-4 border-l-2 border-zinc-800 ml-2">
			{#each session.actions as action (action.timestamp + action.type + action.title)}
				<TimelineAction {action} />
			{/each}
		</div>
	{/if}
</div>
