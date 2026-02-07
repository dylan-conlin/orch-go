import { tv, type VariantProps } from 'tailwind-variants';
export { default as Badge } from './badge.svelte';

export const badgeVariants = tv({
	base: 'inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2',
	variants: {
		variant: {
			default: 'border-transparent bg-primary text-primary-foreground',
			secondary: 'border-transparent bg-secondary text-secondary-foreground',
			destructive: 'border-transparent bg-destructive text-destructive-foreground',
			outline: 'text-foreground',
			// Swarm-specific variants
			active: 'border-transparent bg-green-500/20 text-green-400',
			completed: 'border-transparent bg-blue-500/20 text-blue-400',
			abandoned: 'border-transparent bg-red-500/20 text-red-400',
			idle: 'border-transparent bg-yellow-500/20 text-yellow-400'
		}
	},
	defaultVariants: {
		variant: 'default'
	}
});

export type Variant = VariantProps<typeof badgeVariants>['variant'];
