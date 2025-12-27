<script lang="ts">
	import { DropdownMenu as DropdownMenuPrimitive } from "bits-ui";

	let {
		ref = $bindable(null),
		...restProps
	}: DropdownMenuPrimitive.RadioGroupProps = $props();
</script>

<DropdownMenuPrimitive.RadioGroup bind:ref {...restProps} />
