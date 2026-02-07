<script lang="ts">
	import { shouldShowVersionMismatch, shouldShowStaleWarning, cacheValidation } from '$lib/stores/cache-validation';
	import { Button } from '$lib/components/ui/button';

	function handleReload() {
		window.location.reload();
	}

	function handleDismiss() {
		cacheValidation.dismissVersionMismatch();
	}
</script>

{#if $shouldShowVersionMismatch}
	<div class="fixed top-0 left-0 right-0 z-50 bg-amber-500 text-white px-4 py-3 shadow-lg">
		<div class="container mx-auto flex items-center justify-between">
			<div class="flex items-center gap-3">
				<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
				</svg>
				<span class="font-semibold">New version available</span>
			</div>
			<div class="flex items-center gap-2">
				<Button variant="secondary" size="sm" on:click={handleReload}>
					Reload Dashboard
				</Button>
				<button
					class="text-white hover:text-gray-200"
					on:click={handleDismiss}
					aria-label="Dismiss"
				>
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
						<path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
					</svg>
				</button>
			</div>
		</div>
	</div>
{/if}

{#if $shouldShowStaleWarning && !$shouldShowVersionMismatch}
	<div class="fixed top-0 left-0 right-0 z-50 bg-orange-500 text-white px-4 py-2 shadow-lg">
		<div class="container mx-auto flex items-center justify-center gap-2">
			<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
			</svg>
			<span class="text-sm">Data may be out of date (cache &gt; 60s old)</span>
		</div>
	</div>
{/if}
