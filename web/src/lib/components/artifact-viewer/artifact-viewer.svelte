<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchArtifact, type Artifact } from '$lib/stores/agents';
	
	// Props
	export let workspaceId: string;
	export let beadsId: string | undefined = undefined;
	export let skill: string | undefined = undefined;
	
	// State
	type ArtifactType = 'synthesis' | 'investigation' | 'decision';
	let activeTab: ArtifactType = 'synthesis';
	let artifacts: Map<ArtifactType, Artifact | null> = new Map();
	let loading: Map<ArtifactType, boolean> = new Map();
	let availableTabs: ArtifactType[] = [];
	
	// Determine which tabs to show based on skill type
	function getTabsForSkill(skillName?: string): ArtifactType[] {
		// Default tabs to try
		const tabs: ArtifactType[] = ['synthesis'];
		
		// Add investigation tab for investigation skill
		if (skillName === 'investigation' || skillName === 'systematic-debugging' || skillName === 'research') {
			tabs.push('investigation');
		}
		
		// Add decision tab for architect skill
		if (skillName === 'architect' || skillName === 'design-session') {
			tabs.push('decision');
		}
		
		// If no specific skill, try all tabs
		if (!skillName) {
			return ['synthesis', 'investigation', 'decision'];
		}
		
		return tabs;
	}
	
	// Load artifact content
	async function loadArtifact(type: ArtifactType) {
		if (loading.get(type) || artifacts.get(type)) {
			return; // Already loading or loaded
		}
		
		loading.set(type, true);
		loading = loading; // Trigger reactivity
		
		try {
			const artifact = await fetchArtifact(workspaceId, type, beadsId);
			artifacts.set(type, artifact);
			artifacts = artifacts; // Trigger reactivity
			
			// Update available tabs based on which artifacts have content
			updateAvailableTabs();
		} finally {
			loading.set(type, false);
			loading = loading; // Trigger reactivity
		}
	}
	
	// Update list of tabs that have content
	function updateAvailableTabs() {
		availableTabs = [];
		for (const [type, artifact] of artifacts.entries()) {
			if (artifact && artifact.content && !artifact.error) {
				availableTabs.push(type);
			}
		}
		
		// If current tab is not available, switch to first available
		if (availableTabs.length > 0 && !availableTabs.includes(activeTab)) {
			activeTab = availableTabs[0];
		}
	}
	
	// Get tab label
	function getTabLabel(type: ArtifactType): string {
		switch (type) {
			case 'synthesis': return 'Synthesis';
			case 'investigation': return 'Investigation';
			case 'decision': return 'Decision';
			default: return type;
		}
	}
	
	// Simple markdown-to-HTML conversion
	function renderMarkdown(content: string): string {
		if (!content) return '';
		
		// Escape HTML
		let html = content
			.replace(/&/g, '&amp;')
			.replace(/</g, '&lt;')
			.replace(/>/g, '&gt;');
		
		// Headers
		html = html.replace(/^### (.+)$/gm, '<h3 class="text-md font-semibold mt-4 mb-2">$1</h3>');
		html = html.replace(/^## (.+)$/gm, '<h2 class="text-lg font-semibold mt-4 mb-2">$1</h2>');
		html = html.replace(/^# (.+)$/gm, '<h1 class="text-xl font-bold mt-4 mb-2">$1</h1>');
		
		// Bold and italic
		html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
		html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');
		
		// Inline code
		html = html.replace(/`([^`]+)`/g, '<code class="bg-muted px-1 py-0.5 rounded text-sm font-mono">$1</code>');
		
		// Code blocks
		html = html.replace(/```(\w+)?\n([\s\S]+?)```/g, '<pre class="bg-muted p-3 rounded my-2 overflow-x-auto text-sm font-mono"><code>$2</code></pre>');
		
		// Lists
		html = html.replace(/^- (.+)$/gm, '<li class="ml-4">$1</li>');
		html = html.replace(/(<li[^>]*>.*<\/li>\n?)+/g, '<ul class="list-disc my-2">$&</ul>');
		
		// Paragraphs (lines separated by blank lines)
		html = html.split(/\n\n+/).map(p => {
			// Don't wrap already-wrapped elements
			if (p.startsWith('<h') || p.startsWith('<ul') || p.startsWith('<pre') || p.startsWith('<li')) {
				return p;
			}
			return `<p class="my-2">${p.replace(/\n/g, '<br/>')}</p>`;
		}).join('\n');
		
		return html;
	}
	
	// Initialize: try to load all possible artifacts
	onMount(() => {
		const tabsToTry = getTabsForSkill(skill);
		
		// Load all possible artifacts in parallel
		Promise.all(tabsToTry.map(type => loadArtifact(type)));
	});
</script>

<div class="flex flex-col h-full">
	<!-- Tab bar -->
	{#if availableTabs.length > 0}
		<div class="flex border-b mb-2">
			{#each availableTabs as tab}
				<button
					type="button"
					class="px-4 py-2 text-sm font-medium transition-colors border-b-2 -mb-px"
					class:text-primary={activeTab === tab}
					class:border-primary={activeTab === tab}
					class:text-muted-foreground={activeTab !== tab}
					class:border-transparent={activeTab !== tab}
					class:hover:text-foreground={activeTab !== tab}
					onclick={() => activeTab = tab}
				>
					{getTabLabel(tab)}
				</button>
			{/each}
		</div>
	{/if}
	
	<!-- Content area -->
	<div class="flex-1 overflow-y-auto">
		{#if loading.get(activeTab)}
			<div class="flex items-center justify-center h-32 text-muted-foreground">
				<div class="animate-pulse">Loading {getTabLabel(activeTab).toLowerCase()}...</div>
			</div>
		{:else if artifacts.get(activeTab)?.error}
			<div class="text-center py-8 text-muted-foreground">
				<p class="text-sm">{getTabLabel(activeTab)} not available</p>
				<p class="text-xs mt-1 opacity-75">{artifacts.get(activeTab)?.error}</p>
			</div>
		{:else if artifacts.get(activeTab)?.content}
			<div class="prose prose-sm dark:prose-invert max-w-none">
				{@html renderMarkdown(artifacts.get(activeTab)?.content || '')}
			</div>
			{#if artifacts.get(activeTab)?.path}
				<div class="mt-4 pt-2 border-t text-xs text-muted-foreground">
					<span class="font-mono">{artifacts.get(activeTab)?.path}</span>
				</div>
			{/if}
		{:else if availableTabs.length === 0}
			<div class="text-center py-8 text-muted-foreground">
				<p class="text-sm">No artifacts available</p>
				<p class="text-xs mt-1 opacity-75">This agent hasn't produced any viewable artifacts yet</p>
			</div>
		{/if}
	</div>
</div>
