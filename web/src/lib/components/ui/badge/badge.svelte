<script lang="ts">
	import { cn } from '$lib/utils';
	import { badgeVariants, type Variant } from './index.js';
	import type { HTMLAttributes } from 'svelte/elements';

	type $$Props = HTMLAttributes<HTMLDivElement> & {
		variant?: Variant;
	};

	let className: $$Props['class'] = undefined;
	export { className as class };
	export let variant: Variant = 'default';
</script>

<div class={cn(badgeVariants({ variant }), className)} {...$$restProps}>
	<slot />
</div>
