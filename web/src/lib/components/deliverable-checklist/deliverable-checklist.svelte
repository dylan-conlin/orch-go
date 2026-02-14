<script lang="ts">
	// Stub component for deliverable checklist
	// TODO: Implement full functionality for tracking deliverables
	
	let {
		deliverables = [],
		issueId = ''
	}: {
		deliverables: string[];
		issueId: string;
	} = $props();
</script>

{#if deliverables && deliverables.length > 0}
	<div class="text-xs space-y-1">
		<div class="font-medium text-muted-foreground">Expected Deliverables:</div>
		<ul class="list-disc list-inside space-y-0.5">
			{#each deliverables as deliverable}
				<li class="text-muted-foreground">{deliverable}</li>
			{/each}
		</ul>
	</div>
{/if}
