<script lang="ts">
	import { Sun, Moon } from '@lucide/svelte';
	import { theme, getEffective } from '$lib/stores/theme';
	import { Button } from '$lib/components/ui/button';

	let effectiveTheme = $derived(getEffective($theme));
</script>

<Button
	variant="ghost"
	size="sm"
	onclick={() => theme.toggle()}
	class="h-9 w-9 p-0"
	aria-label="Toggle theme"
>
	{#if effectiveTheme === 'dark'}
		<Sun class="h-4 w-4" />
	{:else}
		<Moon class="h-4 w-4" />
	{/if}
</Button>
