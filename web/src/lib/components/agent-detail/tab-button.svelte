<script lang="ts">
	interface Props {
		active?: boolean;
		onclick?: () => void;
		children?: import('svelte').Snippet;
	}

	let { active = false, onclick, children }: Props = $props();
</script>

<button
	type="button"
	class="relative px-4 py-2 text-sm font-medium transition-colors rounded-t-md {active
		? 'text-foreground bg-background border border-b-0 border-border -mb-px'
		: 'text-muted-foreground hover:text-foreground hover:bg-muted/50'}"
	{onclick}
	role="tab"
	aria-selected={active}
>
	{@render children?.()}
</button>
