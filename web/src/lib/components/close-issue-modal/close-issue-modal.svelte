<script lang="ts">
	// Stub component for close issue modal
	// TODO: Implement full functionality for closing issues with reason
	
	let {
		issueId = '',
		onClose = () => {},
		onConfirm = (reason: string) => {}
	}: {
		issueId: string;
		onClose: () => void;
		onConfirm: (reason: string) => void;
	} = $props();
	
	let reason = $state('');
</script>

<div class="fixed inset-0 z-50 bg-background/80 backdrop-blur-sm flex items-center justify-center">
	<div class="bg-background border rounded-lg p-6 max-w-md w-full shadow-lg">
		<h2 class="text-lg font-semibold mb-4">Close Issue</h2>
		
		<p class="text-sm text-muted-foreground mb-4">
			Closing issue: <span class="font-mono">{issueId}</span>
		</p>
		
		<div class="space-y-4">
			<div>
				<label class="text-sm font-medium">Reason</label>
				<textarea
					bind:value={reason}
					class="w-full mt-1 px-3 py-2 border rounded-md text-sm"
					rows="3"
					placeholder="Enter reason for closing..."
				></textarea>
			</div>
			
			<div class="flex gap-2 justify-end">
				<button
					onclick={onClose}
					class="px-4 py-2 text-sm border rounded-md hover:bg-secondary"
				>
					Cancel
				</button>
				<button
					onclick={() => onConfirm(reason)}
					class="px-4 py-2 text-sm bg-primary text-primary-foreground rounded-md hover:bg-primary/90"
				>
					Close Issue
				</button>
			</div>
		</div>
	</div>
</div>
