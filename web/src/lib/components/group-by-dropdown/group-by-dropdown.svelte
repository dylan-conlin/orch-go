<script lang="ts">
	// Stub component for group-by dropdown
	// TODO: Implement full functionality for grouping options
	
	import type { GroupByMode } from '$lib/stores/work-graph';
	
	let { groupBy = $bindable('none' as GroupByMode) }: { groupBy: GroupByMode } = $props();
</script>

<select bind:value={groupBy} class="px-2 py-1 text-sm border rounded">
	<option value="none">No Grouping</option>
	<option value="priority">By Priority</option>
	<option value="status">By Status</option>
	<option value="label">By Label</option>
</select>
