<script lang="ts">
	export let currentView: 'issues' | 'artifacts' | 'completed' = 'issues';
	export let issueViewMode: 'tree' | 'phase' | 'status' = 'tree';
	export let completedCount: number = 0;
	export let onToggle: ((view: 'issues' | 'artifacts' | 'completed') => void) | undefined = undefined;
	export let onIssueViewModeChange: (mode: 'tree' | 'phase' | 'status') => void = () => {};

	function handleClick(view: 'issues' | 'artifacts' | 'completed') {
		currentView = view;
		onToggle?.(view);
	}

	function handleModeClick(mode: 'tree' | 'phase' | 'status') {
		issueViewMode = mode;
		onIssueViewModeChange(mode);
	}
</script>

<div class="flex items-center justify-center gap-3">
	<!-- Primary view toggle: Issues / Completed / Artifacts -->
	<div class="flex gap-1 border border-border rounded-md p-0.5">
		<button
			class="px-2 py-0.5 rounded text-xs font-medium transition-colors {currentView === 'issues'
				? 'bg-accent text-accent-foreground'
				: 'text-muted-foreground hover:text-foreground'}"
			on:click={() => handleClick('issues')}
		>
			Issues
		</button>
		<button
			class="px-2 py-0.5 rounded text-xs font-medium transition-colors flex items-center gap-1 {currentView === 'completed'
				? 'bg-accent text-accent-foreground'
				: 'text-muted-foreground hover:text-foreground'}"
			on:click={() => handleClick('completed')}
		>
			Completed
			{#if completedCount > 0}
				<span class="inline-flex items-center justify-center h-4 min-w-[16px] px-1 text-[10px] rounded-full {currentView === 'completed' ? 'bg-background/30 text-accent-foreground' : 'bg-yellow-500/20 text-yellow-500'}">
					{completedCount}
				</span>
			{/if}
		</button>
		<button
			class="px-2 py-0.5 rounded text-xs font-medium transition-colors {currentView === 'artifacts'
				? 'bg-accent text-accent-foreground'
				: 'text-muted-foreground hover:text-foreground'}"
			on:click={() => handleClick('artifacts')}
		>
			Artifacts
		</button>
	</div>

	<!-- Secondary view mode toggle (only visible when viewing issues) -->
	{#if currentView === 'issues'}
		<div class="flex gap-0.5 border border-border rounded-md p-0.5">
			<button
				class="px-1.5 py-0.5 rounded text-[11px] font-medium transition-colors {issueViewMode === 'tree'
					? 'bg-accent text-accent-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => handleModeClick('tree')}
				title="Tree View - Parent-child hierarchy"
			>
				Tree
			</button>
			<button
				class="px-1.5 py-0.5 rounded text-[11px] font-medium transition-colors {issueViewMode === 'phase'
					? 'bg-accent text-accent-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
				on:click={() => handleModeClick('phase')}
				title="Phase View - Grouped by execution layer"
			>
				Phase
			</button>
			<button
				class="px-1.5 py-0.5 rounded text-[11px] font-medium transition-colors {issueViewMode === 'status'
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
