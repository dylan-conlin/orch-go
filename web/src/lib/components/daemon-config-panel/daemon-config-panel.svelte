<script lang="ts">
	import { onMount } from 'svelte';
	import { Settings, RefreshCw, Check, AlertTriangle, Save, X } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button';
	import { Badge } from '$lib/components/ui/badge';
	import { daemonConfig, driftStatus, type DaemonConfig, type DriftStatus } from '$lib/stores/daemonConfig';

	// Local form state for editing
	let formState = $state<DaemonConfig | null>(null);
	let isSaving = $state(false);
	let saveError = $state<string | null>(null);
	let saveSuccess = $state(false);
	let isRegenerating = $state(false);
	let hasChanges = $state(false);

	// Track if form has been modified
	$effect(() => {
		if (formState && $daemonConfig) {
			hasChanges = 
				formState.poll_interval !== $daemonConfig.poll_interval ||
				formState.max_agents !== $daemonConfig.max_agents ||
				formState.label !== $daemonConfig.label ||
				formState.verbose !== $daemonConfig.verbose ||
				formState.reflect_issues !== $daemonConfig.reflect_issues;
		}
	});

	// Initialize form state when config loads
	$effect(() => {
		if ($daemonConfig && !formState) {
			formState = { ...$daemonConfig };
		}
	});

	onMount(() => {
		daemonConfig.fetch();
		driftStatus.fetch();
	});

	// Reset form to current saved state
	function resetForm() {
		if ($daemonConfig) {
			formState = { ...$daemonConfig };
		}
		saveError = null;
		saveSuccess = false;
	}

	// Save changes to server
	async function handleSave() {
		if (!formState) return;
		
		isSaving = true;
		saveError = null;
		saveSuccess = false;

		try {
			const result = await daemonConfig.save({
				poll_interval: formState.poll_interval,
				max_agents: formState.max_agents,
				label: formState.label,
				verbose: formState.verbose,
				reflect_issues: formState.reflect_issues
			});

			if (result) {
				saveSuccess = true;
				// Update form state with response
				formState = {
					poll_interval: result.poll_interval,
					max_agents: result.max_agents,
					label: result.label,
					verbose: result.verbose,
					reflect_issues: result.reflect_issues,
					working_directory: result.working_directory,
					path: result.path
				};
				
				// Refetch drift status
				await driftStatus.fetch();
				
				// Clear success message after a delay
				setTimeout(() => { saveSuccess = false; }, 3000);
			}
		} catch (error) {
			saveError = error instanceof Error ? error.message : 'Save failed';
		} finally {
			isSaving = false;
		}
	}

	// Regenerate plist without changing config
	async function handleRegenerate() {
		isRegenerating = true;
		saveError = null;

		try {
			const result = await driftStatus.regenerate();
			if (result && !result.success) {
				saveError = result.error || 'Regeneration failed';
			}
		} catch (error) {
			saveError = error instanceof Error ? error.message : 'Regeneration failed';
		} finally {
			isRegenerating = false;
		}
	}

	// Validation helpers
	function validatePollInterval(value: number): string | null {
		if (value < 10) return 'Must be at least 10 seconds';
		if (value > 3600) return 'Must be at most 3600 seconds (1 hour)';
		return null;
	}

	function validateMaxAgents(value: number): string | null {
		if (value < 1) return 'Must be at least 1';
		if (value > 10) return 'Must be at most 10';
		return null;
	}

	function validateLabel(value: string): string | null {
		if (!value.trim()) return 'Label cannot be empty';
		return null;
	}

	// Computed validation errors using $derived
	let pollIntervalError = $derived(formState ? validatePollInterval(formState.poll_interval) : null);
	let maxAgentsError = $derived(formState ? validateMaxAgents(formState.max_agents) : null);
	let labelError = $derived(formState ? validateLabel(formState.label) : null);
	let isValid = $derived(!pollIntervalError && !maxAgentsError && !labelError);
</script>

<div class="space-y-4">
	<!-- Header with drift status -->
	<div class="flex items-center justify-between">
		<div class="flex items-center gap-2">
			<Settings class="h-4 w-4" />
			<span class="font-medium">Daemon Settings</span>
		</div>
		{#if $driftStatus}
			{#if $driftStatus.in_sync}
				<Badge variant="outline" class="text-green-600 border-green-600/30">
					<Check class="h-3 w-3 mr-1" />
					plist in sync
				</Badge>
			{:else}
				<Badge variant="outline" class="text-amber-600 border-amber-600/30">
					<AlertTriangle class="h-3 w-3 mr-1" />
					drift detected
				</Badge>
			{/if}
		{/if}
	</div>

	{#if formState}
		<div class="space-y-3">
			<!-- Poll Interval -->
			<div class="space-y-1">
				<label for="poll-interval" class="text-sm text-muted-foreground">
					Poll Interval (seconds)
				</label>
				<input
					id="poll-interval"
					type="number"
					min="10"
					max="3600"
					bind:value={formState.poll_interval}
					class="w-full h-8 px-2 text-sm rounded border border-input bg-background focus:outline-none focus:ring-1 focus:ring-ring {pollIntervalError ? 'border-red-500' : ''}"
				/>
				{#if pollIntervalError}
					<p class="text-xs text-red-500">{pollIntervalError}</p>
				{/if}
			</div>

			<!-- Max Agents -->
			<div class="space-y-1">
				<label for="max-agents" class="text-sm text-muted-foreground">
					Max Agents
				</label>
				<input
					id="max-agents"
					type="number"
					min="1"
					max="10"
					bind:value={formState.max_agents}
					class="w-full h-8 px-2 text-sm rounded border border-input bg-background focus:outline-none focus:ring-1 focus:ring-ring {maxAgentsError ? 'border-red-500' : ''}"
				/>
				{#if maxAgentsError}
					<p class="text-xs text-red-500">{maxAgentsError}</p>
				{/if}
			</div>

			<!-- Label -->
			<div class="space-y-1">
				<label for="label" class="text-sm text-muted-foreground">
					Issue Label
				</label>
				<input
					id="label"
					type="text"
					bind:value={formState.label}
					placeholder="triage:ready"
					class="w-full h-8 px-2 text-sm rounded border border-input bg-background focus:outline-none focus:ring-1 focus:ring-ring {labelError ? 'border-red-500' : ''}"
				/>
				{#if labelError}
					<p class="text-xs text-red-500">{labelError}</p>
				{/if}
			</div>

			<!-- Toggle Options -->
			<div class="flex flex-col gap-2">
				<label class="flex items-center gap-2 cursor-pointer">
					<input
						type="checkbox"
						bind:checked={formState.verbose}
						class="h-4 w-4 rounded border-input"
					/>
					<span class="text-sm">Verbose logging</span>
				</label>
				<label class="flex items-center gap-2 cursor-pointer">
					<input
						type="checkbox"
						bind:checked={formState.reflect_issues}
						class="h-4 w-4 rounded border-input"
					/>
					<span class="text-sm">Create issues from kb reflect</span>
				</label>
			</div>

			<!-- Working Directory (read-only) -->
			<div class="space-y-1">
				<label class="text-sm text-muted-foreground">Working Directory</label>
				<p class="text-xs font-mono bg-muted px-2 py-1 rounded truncate" title={formState.working_directory}>
					{formState.working_directory || '(default)'}
				</p>
			</div>

			<!-- Action Buttons -->
			<div class="flex gap-2 pt-2">
				<Button
					variant="default"
					size="sm"
					onclick={handleSave}
					disabled={!hasChanges || !isValid || isSaving}
					class="flex-1"
				>
					{#if isSaving}
						<RefreshCw class="h-3 w-3 mr-1 animate-spin" />
						Saving...
					{:else if saveSuccess}
						<Check class="h-3 w-3 mr-1" />
						Saved
					{:else}
						<Save class="h-3 w-3 mr-1" />
						Save & Restart Daemon
					{/if}
				</Button>
				{#if hasChanges}
					<Button
						variant="ghost"
						size="sm"
						onclick={resetForm}
						disabled={isSaving}
					>
						<X class="h-3 w-3" />
					</Button>
				{/if}
			</div>

			<!-- Drift Status Actions -->
			{#if $driftStatus && !$driftStatus.in_sync}
				<div class="pt-2 border-t">
					<div class="flex items-center justify-between">
						<span class="text-xs text-muted-foreground">
							{$driftStatus.drift_details || 'plist differs from config'}
						</span>
						<Button
							variant="outline"
							size="sm"
							onclick={handleRegenerate}
							disabled={isRegenerating}
						>
							{#if isRegenerating}
								<RefreshCw class="h-3 w-3 mr-1 animate-spin" />
								Regenerating...
							{:else}
								<RefreshCw class="h-3 w-3 mr-1" />
								Regenerate
							{/if}
						</Button>
					</div>
				</div>
			{/if}

			<!-- Error Display -->
			{#if saveError}
				<div class="p-2 text-xs text-red-500 bg-red-500/10 rounded">
					{saveError}
				</div>
			{/if}
		</div>
	{:else}
		<div class="py-4 text-center text-sm text-muted-foreground">
			Loading configuration...
		</div>
	{/if}
</div>
