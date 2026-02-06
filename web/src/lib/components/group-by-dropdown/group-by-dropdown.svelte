<script lang="ts">
	import type { GroupByMode } from '$lib/stores/work-graph';

	export let mode: GroupByMode = 'priority';
	export let onChange: (mode: GroupByMode) => void;

	const options: { value: GroupByMode; label: string }[] = [
		{ value: 'priority', label: 'Priority' },
		{ value: 'area', label: 'Area' },
		{ value: 'effort', label: 'Effort' },
	];

	function handle(event: Event) {
		const target = event.target as HTMLSelectElement;
		mode = target.value as GroupByMode;
		onChange(mode);
	}
</script>

<div class="flex items-center gap-1.5">
	<span class="text-xs text-muted-foreground">Group:</span>
	<select
		value={mode}
		on:change={handle}
		class="h-7 px-2 pr-6 text-xs bg-background border border-border rounded-md
			text-foreground appearance-none cursor-pointer
			focus:outline-none focus:ring-1 focus:ring-ring focus:border-ring
			transition-colors"
		data-testid="group-by-dropdown"
	>
		{#each options as opt}
			<option value={opt.value}>{opt.label}</option>
		{/each}
	</select>
</div>
