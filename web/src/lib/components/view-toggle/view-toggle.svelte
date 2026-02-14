<script lang="ts">
	// Stub component for view toggle
	// TODO: Implement full functionality for view switching
	
	type ViewType = 'issues' | 'artifacts' | 'completed';
	
	let { currentView = $bindable('issues' as ViewType) }: { currentView: ViewType } = $props();
</script>

<div class="flex gap-2">
	<button
		class="px-3 py-1 text-sm rounded {currentView === 'issues' ? 'bg-primary text-primary-foreground' : 'bg-secondary'}"
		onclick={() => currentView = 'issues'}
	>
		Issues
	</button>
	<button
		class="px-3 py-1 text-sm rounded {currentView === 'artifacts' ? 'bg-primary text-primary-foreground' : 'bg-secondary'}"
		onclick={() => currentView = 'artifacts'}
	>
		Artifacts
	</button>
	<button
		class="px-3 py-1 text-sm rounded {currentView === 'completed' ? 'bg-primary text-primary-foreground' : 'bg-secondary'}"
		onclick={() => currentView = 'completed'}
	>
		Completed
	</button>
</div>
