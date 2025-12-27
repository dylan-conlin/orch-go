<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";
	import { Circle } from "@lucide/svelte";
	import type { Snippet } from "svelte";

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: DropdownMenuPrimitive.RadioItemProps & { children?: Snippet } = $props();
</script>

<DropdownMenuPrimitive.RadioItem
	bind:ref
	class={cn(
		"relative flex cursor-default select-none items-center rounded-sm py-1.5 pl-8 pr-2 text-sm outline-none transition-colors focus:bg-accent focus:text-accent-foreground data-[disabled]:pointer-events-none data-[disabled]:opacity-50 data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground",
		className
	)}
	{...restProps}
>
	{#snippet children({ checked })}
		<span class="absolute left-2 flex h-3.5 w-3.5 items-center justify-center">
			{#if checked}
				<Circle class="h-2 w-2 fill-current" />
			{/if}
		</span>
		{@render children?.({ checked })}
	{/snippet}
</DropdownMenuPrimitive.RadioItem>
