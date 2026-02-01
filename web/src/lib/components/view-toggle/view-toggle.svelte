<script lang="ts">
	export let currentView: 'issues' | 'artifacts' = 'issues';
	export let onToggle: (view: 'issues' | 'artifacts') => void;

	function handleClick(view: 'issues' | 'artifacts') {
		currentView = view;
		onToggle(view);
	}
</script>

<div class="flex gap-2 border border-border rounded-md p-1">
	<button
		class="px-3 py-1 rounded text-sm font-medium transition-colors {currentView === 'issues'
			? 'bg-accent text-accent-foreground'
			: 'text-muted-foreground hover:text-foreground'}"
		on:click={() => handleClick('issues')}
	>
		Issues
	</button>
	<button
		class="px-3 py-1 rounded text-sm font-medium transition-colors {currentView === 'artifacts'
			? 'bg-accent text-accent-foreground'
			: 'text-muted-foreground hover:text-foreground'}"
		on:click={() => handleClick('artifacts')}
	>
		Artifacts
	</button>
</div>
