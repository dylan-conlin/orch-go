<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: DropdownMenuPrimitive.GroupHeadingProps = $props();
</script>

<DropdownMenuPrimitive.GroupHeading
	bind:ref
	class={cn("px-2 py-1.5 text-sm font-semibold", className)}
	{...restProps}
/>
