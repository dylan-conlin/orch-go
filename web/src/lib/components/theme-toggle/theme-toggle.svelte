<script lang="ts">
	import { Sun, Moon, Monitor, Palette, Check } from '@lucide/svelte';
	import { theme, mode, getEffective, type Mode } from '$lib/stores/theme';
	import { Button } from '$lib/components/ui/button';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';

	let effectiveTheme = $derived(getEffective($mode));

	const modeOptions: { value: Mode; label: string; icon: typeof Sun }[] = [
		{ value: 'light', label: 'Light', icon: Sun },
		{ value: 'dark', label: 'Dark', icon: Moon },
		{ value: 'system', label: 'System', icon: Monitor }
	];

	const themes = theme.all();
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Button {...props} variant="ghost" size="sm" class="h-9 w-9 p-0" aria-label="Select theme">
				{#if effectiveTheme === 'dark'}
					<Moon class="h-4 w-4" />
				{:else}
					<Sun class="h-4 w-4" />
				{/if}
			</Button>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="max-h-96 overflow-y-auto">
		<DropdownMenu.Group>
			<DropdownMenu.GroupHeading>Mode</DropdownMenu.GroupHeading>
		</DropdownMenu.Group>
		<DropdownMenu.RadioGroup value={$mode} onValueChange={(value) => mode.set(value as Mode)}>
			{#each modeOptions as option}
				<DropdownMenu.RadioItem value={option.value}>
					<option.icon class="mr-2 h-4 w-4" />
					<span>{option.label}</span>
				</DropdownMenu.RadioItem>
			{/each}
		</DropdownMenu.RadioGroup>

		<DropdownMenu.Separator />

		<DropdownMenu.Group>
			<DropdownMenu.GroupHeading>
				<div class="flex items-center gap-2">
					<Palette class="h-4 w-4" />
					<span>Theme</span>
				</div>
			</DropdownMenu.GroupHeading>
		</DropdownMenu.Group>
		<DropdownMenu.RadioGroup value={$theme} onValueChange={(value) => theme.set(value as string)}>
			{#each themes as themeName}
				<DropdownMenu.RadioItem value={themeName}>
					<span class="capitalize">{themeName.replace(/-/g, ' ')}</span>
				</DropdownMenu.RadioItem>
			{/each}
		</DropdownMenu.RadioGroup>
	</DropdownMenu.Content>
</DropdownMenu.Root>
