<script lang="ts">
	// Stub component for label filtering
	// TODO: Implement full functionality for label filtering
	
	let { selectedLabels = $bindable([] as string[]) }: { selectedLabels: string[] } = $props();
	
	// Placeholder labels
	const availableLabels = ['bug', 'feature', 'enhancement', 'documentation'];
</script>

<div class="flex gap-2 flex-wrap">
	{#each availableLabels as label}
		<label class="flex items-center gap-1 text-sm">
			<input
				type="checkbox"
				value={label}
				checked={selectedLabels.includes(label)}
				onchange={(e) => {
					if (e.currentTarget.checked) {
						selectedLabels = [...selectedLabels, label];
					} else {
						selectedLabels = selectedLabels.filter(l => l !== label);
					}
				}}
			/>
			{label}
		</label>
	{/each}
</div>
