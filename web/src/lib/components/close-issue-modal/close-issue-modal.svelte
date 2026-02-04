<script lang="ts">
	import { createEventDispatcher } from 'svelte';

	export let issueId: string;
	export let issueTitle: string = '';

	const dispatch = createEventDispatcher<{
		close: void;
		confirm: { reason: string };
	}>();

	let reason = '';
	let inputElement: HTMLInputElement;

	// Focus input on mount
	$: if (inputElement) {
		setTimeout(() => inputElement?.focus(), 50);
	}

	function handleSubmit(e: Event) {
		e.preventDefault();
		dispatch('confirm', { reason: reason.trim() });
	}

	function handleCancel() {
		dispatch('close');
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			e.stopPropagation();
			handleCancel();
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<!-- Backdrop -->
<div
	class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center"
	role="dialog"
	aria-modal="true"
	aria-labelledby="close-issue-title"
	on:click|self={handleCancel}
>
	<!-- Modal -->
	<div class="bg-background border border-border rounded-lg shadow-xl w-full max-w-md mx-4 p-6">
		<h2 id="close-issue-title" class="text-lg font-semibold text-foreground mb-2">
			Close Issue
		</h2>
		
		<p class="text-sm text-muted-foreground mb-4">
			{#if issueTitle}
				<span class="font-mono text-xs">{issueId}</span>: {issueTitle}
			{:else}
				<span class="font-mono">{issueId}</span>
			{/if}
		</p>

		<form on:submit={handleSubmit}>
			<label for="close-reason" class="block text-sm font-medium text-foreground mb-2">
				Reason (optional)
			</label>
			<input
				bind:this={inputElement}
				bind:value={reason}
				id="close-reason"
				type="text"
				placeholder="e.g., Completed, duplicate of X, wont-fix"
				class="w-full px-3 py-2 bg-muted border border-border rounded text-foreground text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary focus:border-transparent"
			/>

			<div class="flex justify-end gap-3 mt-6">
				<button
					type="button"
					on:click={handleCancel}
					class="px-4 py-2 text-sm font-medium text-muted-foreground hover:text-foreground transition-colors"
				>
					Cancel
				</button>
				<button
					type="submit"
					class="px-4 py-2 text-sm font-medium bg-primary text-primary-foreground rounded hover:bg-primary/90 transition-colors"
				>
					Close Issue
				</button>
			</div>
		</form>
	</div>
</div>
