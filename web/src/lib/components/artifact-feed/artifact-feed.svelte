<script lang="ts">
	import { onMount } from 'svelte';
	import { kbArtifacts, type ArtifactFeedItem } from '$lib/stores/kb-artifacts';
	import { orchestratorContext } from '$lib/stores/context';
	import { ArtifactRow } from '$lib/components/artifact-row';
	import { ArtifactSidePanel } from '$lib/components/artifact-side-panel';

	let selectedArtifact: ArtifactFeedItem | null = null;
	let selectedIndex = -1;
	let currentSection: 'needs-decision' | 'recent' | 'browse' = 'needs-decision';
	let timeFilter = '7d';
	let browseType: string | null = null;

	// Get time filter from localStorage
	onMount(() => {
		const projectDir = $orchestratorContext?.project_dir;
		const saved = localStorage.getItem('artifact-feed-time-filter');
		const validFilters = new Set(['24h', '7d', '30d', 'all']);
		const savedFilter = saved && validFilters.has(saved) ? saved : null;
		timeFilter = savedFilter || kbArtifacts.getSince();

		if (savedFilter && savedFilter !== kbArtifacts.getSince()) {
			kbArtifacts.fetch(projectDir, savedFilter).catch(console.error);
		}

		if (!savedFilter && kbArtifacts.getSince() !== '7d') {
			kbArtifacts.fetch(projectDir).catch(console.error);
		}
	});

	// Save time filter to localStorage
	function setTimeFilter(filter: string) {
		timeFilter = filter;
		localStorage.setItem('artifact-feed-time-filter', filter);
		// Re-fetch with new filter
		const projectDir = $orchestratorContext?.project_dir;
		kbArtifacts.fetch(projectDir, filter);
	}

	// Artifacts filtered by browse type, or the default needs_decision + recent
	$: browseArtifacts = browseType && $kbArtifacts?.by_type?.[browseType]
		? $kbArtifacts.by_type[browseType]
		: null;

	// Flatten all artifacts into a single list for keyboard navigation
	$: allArtifacts = browseArtifacts
		? browseArtifacts
		: [
			...($kbArtifacts?.needs_decision ?? []),
			...($kbArtifacts?.recent ?? [])
		];

	function toggleBrowseType(type: string) {
		if (browseType === type) {
			browseType = null;
		} else {
			browseType = type;
		}
		selectedIndex = -1;
		selectedArtifact = null;
	}

	// Keyboard navigation
	function handleKeydown(event: KeyboardEvent) {
		const key = event.key;

		// j/k navigation
		if (key === 'j' && selectedIndex < allArtifacts.length - 1) {
			event.preventDefault();
			selectedIndex++;
			selectedArtifact = allArtifacts[selectedIndex];
		} else if (key === 'k' && selectedIndex > 0) {
			event.preventDefault();
			selectedIndex--;
			selectedArtifact = allArtifacts[selectedIndex];
		}

		// l/Enter - open side panel
		else if ((key === 'l' || key === 'Enter') && selectedArtifact) {
			event.preventDefault();
			// Side panel opens automatically via selectedArtifact binding
		}

		// h/Escape - close side panel
		else if (key === 'h' || key === 'Escape') {
			event.preventDefault();
			if (selectedArtifact) {
				selectedArtifact = null;
				selectedIndex = -1;
			}
		}

		// g - jump to top
		else if (key === 'g' && !event.shiftKey) {
			event.preventDefault();
			selectedIndex = 0;
			selectedArtifact = allArtifacts[0];
		}

		// G (Shift+g) - jump to bottom
		else if (key === 'G' || (key === 'g' && event.shiftKey)) {
			event.preventDefault();
			selectedIndex = allArtifacts.length - 1;
			selectedArtifact = allArtifacts[selectedIndex];
		}

		// 1/2/3 - jump to section
		else if (key === '1') {
			event.preventDefault();
			currentSection = 'needs-decision';
		} else if (key === '2') {
			event.preventDefault();
			currentSection = 'recent';
		} else if (key === '3') {
			event.preventDefault();
			currentSection = 'browse';
		}

		// C - copy path
		else if (key === 'C' || (key === 'c' && event.shiftKey)) {
			event.preventDefault();
			if (selectedArtifact) {
				navigator.clipboard.writeText(selectedArtifact.path);
			}
		}

		// i/o - toggle side panel
		else if (key === 'i' || key === 'o') {
			event.preventDefault();
			if (selectedIndex >= 0 && allArtifacts[selectedIndex]) {
				const current = allArtifacts[selectedIndex];
				if (selectedArtifact?.path === current.path) {
					selectedArtifact = null;
				} else {
					selectedArtifact = current;
				}
			}
		}
	}

	function handleArtifactClick(artifact: ArtifactFeedItem, index: number) {
		selectedArtifact = artifact;
		selectedIndex = index;
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="artifact-feed flex flex-col h-full min-h-0 overflow-y-auto">
	<!-- Browse by Type filter bar -->
	<div class="px-6 py-3 border-b border-border">
		<div class="flex flex-wrap gap-2 text-xs">
			{#if $kbArtifacts?.by_type}
				{@const types = [
					{ key: 'investigation', label: 'Investigations' },
					{ key: 'decision', label: 'Decisions' },
					{ key: 'model', label: 'Models' },
					{ key: 'guide', label: 'Guides' }
				]}
				{#each types as t (t.key)}
					{@const count = $kbArtifacts.by_type[t.key]?.length ?? 0}
					<button
						class="px-2 py-1 rounded-md border transition-colors {browseType === t.key
							? 'bg-accent text-accent-foreground border-accent'
							: 'border-border text-muted-foreground hover:text-foreground hover:border-foreground/30'}"
						on:click={() => toggleBrowseType(t.key)}
					>
						{t.label} ({count})
					</button>
				{/each}
			{/if}
		</div>
	</div>

	{#if browseArtifacts}
		<!-- Filtered by type -->
		<div class="px-6 py-4">
			<div class="flex items-center justify-between mb-3">
				<h2 class="text-sm font-semibold text-foreground uppercase">
					{browseType}s ({browseArtifacts.length})
				</h2>
				<button
					class="text-xs text-muted-foreground hover:text-foreground"
					on:click={() => { browseType = null; }}
				>
					Clear filter
				</button>
			</div>
			<div class="space-y-2">
				{#each browseArtifacts as artifact, i (artifact.path)}
					<ArtifactRow
						{artifact}
						selected={selectedIndex === i}
						on:click={() => handleArtifactClick(artifact, i)}
					/>
				{/each}
			</div>
		</div>
	{:else}
		<!-- Needs Decision Section -->
		{#if $kbArtifacts?.needs_decision && $kbArtifacts.needs_decision.length > 0}
			<div class="px-6 py-4 border-b border-border">
				<h2 class="text-sm font-semibold text-foreground mb-3">
					NEEDS DECISION ({$kbArtifacts.needs_decision.length})
				</h2>
				<div class="space-y-2">
					{#each $kbArtifacts.needs_decision as artifact, i (artifact.path)}
						<ArtifactRow
							{artifact}
							selected={selectedIndex === i}
							on:click={() => handleArtifactClick(artifact, i)}
						/>
					{/each}
				</div>
			</div>
		{/if}

		<!-- Recently Updated Section -->
		<div class="px-6 py-4 border-b border-border">
			<div class="flex items-center justify-between mb-3">
				<h2 class="text-sm font-semibold text-foreground">
					RECENTLY UPDATED ({$kbArtifacts?.recent.length ?? 0})
				</h2>
				<!-- Time Filter -->
				<select
					bind:value={timeFilter}
					on:change={(e) => setTimeFilter(e.currentTarget.value)}
					class="text-xs border border-border rounded px-2 py-1 bg-background text-foreground"
				>
					<option value="24h">24h</option>
					<option value="7d">7d</option>
					<option value="30d">30d</option>
					<option value="all">all</option>
				</select>
			</div>
			<div class="space-y-2">
				{#if $kbArtifacts?.recent && $kbArtifacts.recent.length > 0}
					{#each $kbArtifacts.recent as artifact, i (artifact.path)}
						{@const globalIndex = ($kbArtifacts.needs_decision?.length ?? 0) + i}
						<ArtifactRow
							{artifact}
							selected={selectedIndex === globalIndex}
							on:click={() => handleArtifactClick(artifact, globalIndex)}
						/>
					{/each}
				{:else}
					<p class="text-sm text-muted-foreground">No recent artifacts</p>
				{/if}
			</div>
		</div>
	{/if}
</div>

<!-- Side Panel -->
{#if selectedArtifact}
	<ArtifactSidePanel
		artifact={selectedArtifact}
		on:close={() => {
			selectedArtifact = null;
			selectedIndex = -1;
		}}
	/>
{/if}

<style>
	.artifact-feed {
		max-height: 100%;
		overscroll-behavior: contain;
	}
</style>
