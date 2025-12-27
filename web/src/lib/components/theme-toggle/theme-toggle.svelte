<script lang="ts">
	import { Sun, Moon, Monitor } from '@lucide/svelte';
	import { theme, getEffective, type Theme } from '$lib/stores/theme';
	import { Button } from '$lib/components/ui/button';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';

	let effectiveTheme = $derived(getEffective($theme));

	const themeOptions: { value: Theme; label: string; icon: typeof Sun }[] = [
		{ value: 'light', label: 'Light', icon: Sun },
		{ value: 'dark', label: 'Dark', icon: Moon },
		{ value: 'system', label: 'System', icon: Monitor },
	];
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button
				{...props}
				variant="ghost"
				size="sm"
				class="h-9 w-9 p-0"
				aria-label="Select theme"
			>
				{#if effectiveTheme === 'dark'}
					<Sun class="h-4 w-4" />
				{:else}
					<Moon class="h-4 w-4" />
				{/if}
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end">
		<DropdownMenu.Group>
			<DropdownMenu.GroupHeading>Theme</DropdownMenu.GroupHeading>
		</DropdownMenu.Group>
		<DropdownMenu.Separator />
		<DropdownMenu.RadioGroup value={$theme} onValueChange={(value) => theme.set(value as Theme)}>
			{#each themeOptions as option}
				<DropdownMenu.RadioItem value={option.value}>
					<option.icon class="mr-2 h-4 w-4" />
					<span>{option.label}</span>
				</DropdownMenu.RadioItem>
			{/each}
		</DropdownMenu.RadioGroup>
	</DropdownMenu.Content>
</DropdownMenu.Root>
