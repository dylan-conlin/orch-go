<script lang="ts">
	import { cn } from '$lib/utils';
	import { buttonVariants, type Variant, type Size } from './index.js';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	type $$Props = HTMLButtonAttributes & {
		variant?: Variant;
		size?: Size;
	};

	let className: $$Props['class'] = undefined;
	export { className as class };
	export let variant: Variant = 'default';
	export let size: Size = 'default';
</script>

<button
	class={cn(buttonVariants({ variant, size }), className)}
	{...$$restProps}
>
	<slot />
</button>
