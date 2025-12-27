<script lang="ts">
	import { Settings, Bell, FileText, Terminal } from '@lucide/svelte';
	import { config, type ConfigInfo } from '$lib/stores/config';
	import { Button } from '$lib/components/ui/button';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu';
	import * as Tooltip from '$lib/components/ui/tooltip';

	// Track saving state for visual feedback
	let isSaving = $state(false);

	// Handle toggle for boolean settings
	async function handleToggle(key: 'auto_export_transcript' | 'notifications_enabled') {
		isSaving = true;
		await config.toggle(key);
		isSaving = false;
	}
</script>

<DropdownMenu.Root>
	<DropdownMenu.Trigger>
		{#snippet child({ props })}
			<Tooltip.Root>
				<Tooltip.Trigger>
					{#snippet child({ props: tooltipProps })}
						<Button {...props} {...tooltipProps} variant="ghost" size="sm" class="h-9 w-9 p-0" aria-label="Settings">
							<Settings class="h-4 w-4" />
						</Button>
					{/snippet}
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p>Configuration Settings</p>
					<p class="text-xs text-muted-foreground">~/.orch/config.yaml</p>
				</Tooltip.Content>
			</Tooltip.Root>
		{/snippet}
	</DropdownMenu.Trigger>
	<DropdownMenu.Content align="end" class="w-64">
		<DropdownMenu.Group>
			<DropdownMenu.GroupHeading>
				<div class="flex items-center gap-2">
					<Settings class="h-4 w-4" />
					<span>Configuration</span>
				</div>
			</DropdownMenu.GroupHeading>
		</DropdownMenu.Group>

		{#if $config}
			<!-- Notifications toggle -->
			<DropdownMenu.Item
				class="flex items-center justify-between cursor-pointer"
				onclick={() => handleToggle('notifications_enabled')}
			>
				<div class="flex items-center gap-2">
					<Bell class="h-4 w-4" />
					<span>Desktop Notifications</span>
				</div>
				<span class="text-xs px-2 py-0.5 rounded {$config.notifications_enabled ? 'bg-green-500/20 text-green-500' : 'bg-muted text-muted-foreground'}">
					{$config.notifications_enabled ? 'ON' : 'OFF'}
				</span>
			</DropdownMenu.Item>

			<!-- Auto export transcript toggle -->
			<DropdownMenu.Item
				class="flex items-center justify-between cursor-pointer"
				onclick={() => handleToggle('auto_export_transcript')}
			>
				<div class="flex items-center gap-2">
					<FileText class="h-4 w-4" />
					<span>Auto Export Transcript</span>
				</div>
				<span class="text-xs px-2 py-0.5 rounded {$config.auto_export_transcript ? 'bg-green-500/20 text-green-500' : 'bg-muted text-muted-foreground'}">
					{$config.auto_export_transcript ? 'ON' : 'OFF'}
				</span>
			</DropdownMenu.Item>

			<DropdownMenu.Separator />

			<!-- Backend display (read-only for now) -->
			<DropdownMenu.Item class="flex items-center justify-between cursor-default" disabled>
				<div class="flex items-center gap-2">
					<Terminal class="h-4 w-4" />
					<span>Backend</span>
				</div>
				<span class="text-xs text-muted-foreground">{$config.backend}</span>
			</DropdownMenu.Item>

			{#if $config.config_path}
				<DropdownMenu.Separator />
				<div class="px-2 py-1.5 text-xs text-muted-foreground truncate" title={$config.config_path}>
					{$config.config_path}
				</div>
			{/if}
		{:else}
			<DropdownMenu.Item disabled>
				<span class="text-muted-foreground">Loading...</span>
			</DropdownMenu.Item>
		{/if}
	</DropdownMenu.Content>
</DropdownMenu.Root>
