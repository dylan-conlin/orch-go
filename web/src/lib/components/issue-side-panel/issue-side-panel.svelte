<script lang="ts">
	// Stub component for issue side panel
	// TODO: Implement full functionality for issue details panel
	
	import type { TreeNode } from '$lib/stores/work-graph';
	
	let {
		issue = $bindable(null as TreeNode | null),
		onClose = () => {}
	}: {
		issue: TreeNode | null;
		onClose: () => void;
	} = $props();
</script>

{#if issue}
	<div class="fixed right-0 top-0 h-full w-96 bg-background border-l shadow-lg p-4 overflow-y-auto z-40">
		<div class="flex justify-between items-center mb-4">
			<h2 class="text-lg font-semibold">Issue Details</h2>
			<button onclick={onClose} class="text-muted-foreground hover:text-foreground">✕</button>
		</div>
		
		<div class="space-y-4">
			<div>
				<div class="text-sm font-medium text-muted-foreground">ID</div>
				<div class="text-sm">{issue.id}</div>
			</div>
			
			<div>
				<div class="text-sm font-medium text-muted-foreground">Title</div>
				<div class="text-sm">{issue.title}</div>
			</div>
			
			{#if issue.status}
				<div>
					<div class="text-sm font-medium text-muted-foreground">Status</div>
					<div class="text-sm">{issue.status}</div>
				</div>
			{/if}
			
			{#if issue.priority !== undefined}
				<div>
					<div class="text-sm font-medium text-muted-foreground">Priority</div>
					<div class="text-sm">P{issue.priority}</div>
				</div>
			{/if}
		</div>
		
		<!-- TODO: Add more issue details, comments, metadata -->
	</div>
{/if}
