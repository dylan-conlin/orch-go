<script lang="ts">
	import type { GroupByMode } from '$lib/stores/work-graph';
	
	type Props = {
		mode: GroupByMode;
		onChange: (mode: GroupByMode) => void;
	};
	
	let { mode, onChange }: Props = $props();
	
	function handleChange(event: Event) {
		const target = event.target as HTMLSelectElement;
		onChange(target.value as GroupByMode);
	}
</script>

<select
	value={mode}
	onchange={handleChange}
	class="px-2 py-1 text-sm border border-border rounded bg-background text-foreground hover:bg-accent transition-colors"
>
	<option value="priority">By Priority</option>
	<option value="area">By Area</option>
	<option value="effort">By Effort</option>
	<option value="dep-chain">By Dependency Chain</option>
</select>
