<script lang="ts">
	export let value: string = '';
	export let onChange: (value: string) => void;
	export let placeholder: string = 'Filter by label...';

	let inputEl: HTMLInputElement;

	function handle(event: Event) {
		const target = event.target as HTMLInputElement;
		value = target.value;
		onChange(value);
	}

	function clear() {
		value = '';
		onChange('');
		inputEl?.focus();
	}

	// Expose focus for keyboard shortcut
	export function focus() {
		inputEl?.focus();
	}
</script>

<div class="relative flex items-center">
	<svg
		class="absolute left-2.5 w-3.5 h-3.5 text-muted-foreground pointer-events-none"
		xmlns="http://www.w3.org/2000/svg"
		viewBox="0 0 24 24"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
		stroke-linecap="round"
		stroke-linejoin="round"
	>
		<path d="M20.59 13.41l-7.17 7.17a2 2 0 0 1-2.83 0L2 12V2h10l8.59 8.59a2 2 0 0 1 0 2.82z" />
		<line x1="7" y1="7" x2="7.01" y2="7" />
	</svg>
	<input
		bind:this={inputEl}
		type="text"
		{value}
		{placeholder}
		on:input={handle}
		class="h-8 w-48 pl-8 pr-7 text-sm bg-background border border-border rounded-md
			text-foreground placeholder:text-muted-foreground
			focus:outline-none focus:ring-1 focus:ring-ring focus:border-ring
			transition-colors"
	/>
	{#if value}
		<button
			type="button"
			on:click={clear}
			class="absolute right-2 text-muted-foreground hover:text-foreground transition-colors"
			title="Clear filter"
		>
			<svg class="w-3.5 h-3.5" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
				<line x1="18" y1="6" x2="6" y2="18" />
				<line x1="6" y1="6" x2="18" y2="18" />
			</svg>
		</button>
	{/if}
</div>
