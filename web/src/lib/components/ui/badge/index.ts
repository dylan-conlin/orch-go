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
			idle: 'border-transparent bg-yellow-500/20 text-yellow-400',
			// Attention badge variants (inline signals on issues)
			attention_verify: 'border-transparent bg-yellow-900/50 text-yellow-400 text-[10px] px-1.5 py-0',
			attention_decide: 'border-transparent bg-yellow-900/50 text-yellow-400 text-[10px] px-1.5 py-0',
			attention_escalate: 'border-transparent bg-orange-900/50 text-orange-400 text-[10px] px-1.5 py-0',
			attention_likely_done: 'border-transparent bg-green-900/50 text-green-400 text-[10px] px-1.5 py-0',
			attention_unblocked: 'border-transparent bg-green-900/50 text-green-400 text-[10px] px-1.5 py-0',
			attention_stuck: 'border-transparent bg-red-900/50 text-red-400 text-[10px] px-1.5 py-0',
			attention_crashed: 'border-transparent bg-red-900/50 text-red-400 text-[10px] px-1.5 py-0',
			// Completed issue verification badges
			attention_unverified: 'border-transparent bg-yellow-900/50 text-yellow-400 text-[10px] px-1.5 py-0',
			attention_needs_fix: 'border-transparent bg-red-900/50 text-red-400 text-[10px] px-1.5 py-0'
		}
	},
	defaultVariants: {
		variant: 'default'
	}
});

export type Variant = VariantProps<typeof badgeVariants>['variant'];
