<script lang="ts">
	export let currentView: 'issues' | 'artifacts' = 'issues';
	export let issueViewMode: 'tree' | 'phase' | 'status' = 'tree';
	export let onToggle: (view: 'issues' | 'artifacts') => void;
	export let onIssueViewModeChange: (mode: 'tree' | 'phase' | 'status') => void = () => {};

	function handleClick(view: 'issues' | 'artifacts') {
		currentView = view;
		onToggle(view);
	}

	function handleModeClick(mode: 'tree' | 'phase' | 'status') {
		issueViewMode = mode;
		onIssueViewModeChange(mode);
	}
</script>

<div class="flex items-center gap-4">
	<!-- Primary view toggle: Issues vs Artifacts -->
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

	<!-- Secondary view mode toggle (only visible when viewing issues) -->
	{#if currentView === 'issues'}
		<div class="flex gap-1 border border-border rounded-md p-1">
			<button
				class="px-2 py-1 rounded text-xs font-medium transition-colors {issueViewMode === 'tree'
					? 'bg-accent text-accent-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => handleModeClick('tree')}
				title="Tree View - Parent-child hierarchy"
			>
				Tree
			</button>
			<button
				class="px-2 py-1 rounded text-xs font-medium transition-colors {issueViewMode === 'phase'
					? 'bg-accent text-accent-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => handleModeClick('phase')}
				title="Phase View - Grouped by execution layer"
			>
				Phase
			</button>
			<button
				class="px-2 py-1 rounded text-xs font-medium transition-colors {issueViewMode === 'status'
					? 'bg-accent text-accent-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => handleModeClick('status')}
				title="Status View - Grouped by status"
			>
				Status
			</button>
		</div>
	{/if}
</div>
