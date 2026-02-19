<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { TabButton, ActivityTab, InvestigationTab, SynthesisTab, ScreenshotsTab } from '$lib/components/agent-detail';
	import { agents, type Agent } from '$lib/stores/agents';
	import type { TreeNode } from '$lib/stores/work-graph';
	import { createEventDispatcher, onMount } from 'svelte';
	import { fly } from 'svelte/transition';
	import { browser } from '$app/environment';

	// Tab types based on agent state
	type TabType = 'overview' | 'activity' | 'synthesis' | 'investigation' | 'screenshots';

	let {
		issue = $bindable(null as TreeNode | null),
		onClose = () => {}
	}: {
		issue: TreeNode | null;
		onClose?: () => void;
	} = $props();

	const dispatch = createEventDispatcher<{ close: void }>();

	// Active tab state - determined by agent status
	let activeTab: TabType = $state('overview');
	let lastIssueId: string | null = null;

	// Resolve agent for this issue (by beads_id or agent id fallback)
	let relatedAgent: Agent | null = $derived(() => {
		if (!issue) return null;
		return $agents.find((agent) => agent.beads_id === issue.id || agent.id === issue.id) || null;
	});

	function closePanel() {
		onClose?.();
		dispatch('close');
	}

	// Determine which tabs are visible based on agent status
	function getVisibleTabs(agent: Agent | null): TabType[] {
		if (!agent) return ['overview'];
		switch (agent.status) {
			case 'active':
				return ['overview', 'activity', 'screenshots'];
			case 'completed':
				return ['overview', 'synthesis', 'investigation', 'screenshots'];
			case 'abandoned':
				return ['overview', 'investigation', 'screenshots'];
			default:
				return ['overview', 'activity', 'screenshots'];
		}
	}

	// Get default tab for agent status
	function getDefaultTab(agent: Agent | null): TabType {
		if (!agent) return 'overview';
		switch (agent.status) {
			case 'active':
				return 'activity';
			case 'completed':
				return 'synthesis';
			case 'abandoned':
				return 'investigation';
			default:
				return 'activity';
		}
	}

	// Visible tabs derived from agent
	$effect(() => {
		const visibleTabs = getVisibleTabs(relatedAgent);
		if (!visibleTabs.includes(activeTab)) {
			activeTab = getDefaultTab(relatedAgent);
		}
	});

	// Reset tab to default when issue changes
	$effect(() => {
		const issueId = issue?.id ?? null;
		if (issueId && issueId !== lastIssueId) {
			activeTab = getDefaultTab(relatedAgent);
			lastIssueId = issueId;
		}
	});

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) {
			closePanel();
		}
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			closePanel();
		}
	}

	// Prevent body scroll when panel is open to avoid double scrollbar
	$effect(() => {
		if (!browser) return;
		if (issue) {
			document.body.style.overflow = 'hidden';
		} else {
			document.body.style.overflow = '';
		}
		return () => {
			document.body.style.overflow = '';
		};
	});

	onMount(() => {
		if (browser) {
			window.addEventListener('keydown', handleKeydown);
		}
		return () => {
			if (browser) {
				window.removeEventListener('keydown', handleKeydown);
			}
		};
	});
</script>

{#if browser && issue}
	<!-- Backdrop -->
	<button
		type="button"
		class="fixed inset-0 z-40 cursor-default border-none bg-background/80 backdrop-blur-sm"
		onclick={handleBackdropClick}
		aria-label="Close panel"
	></button>

	<!-- Slide-out Panel -->
	<div
		class="fixed right-0 top-0 z-50 flex h-full w-full flex-col border-l bg-card shadow-xl sm:w-[85vw] lg:w-[80vw] max-w-[1200px]"
		transition:fly={{ x: 500, duration: 200 }}
		role="dialog"
		aria-modal="true"
		aria-labelledby="issue-detail-title"
	>
		<!-- Header -->
		<div class="flex items-center justify-between border-b px-4 py-3">
			<div class="min-w-0">
				<h2 id="issue-detail-title" class="text-lg font-semibold truncate">
					{issue.title}
				</h2>
				<div class="mt-1 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
					<span class="font-mono">{issue.id}</span>
					{#if issue.status}
						<Badge variant="outline" class="text-xs">
							{issue.status}
						</Badge>
					{/if}
						<Badge variant="secondary" class="text-xs">P{issue.priority}</Badge>
						{#if issue.effective_priority !== undefined && issue.effective_priority !== issue.priority}
							<Badge variant="outline" class="text-xs">Eff P{issue.effective_priority}</Badge>
						{/if}
					{#if issue.layer !== undefined}
						<Badge variant="outline" class="text-xs">Layer {issue.layer}</Badge>
					{/if}
					<Badge variant="outline" class="text-xs">{issue.type}</Badge>
				</div>
			</div>
			<Button variant="ghost" size="sm" onclick={closePanel} class="h-8 w-8 p-0">
				<span class="text-lg">×</span>
			</Button>
		</div>

		<!-- Tab Navigation -->
		<div class="border-b px-4">
			<div class="flex gap-1" role="tablist" aria-label="Issue detail tabs">
				{#if getVisibleTabs(relatedAgent).includes('overview')}
					<TabButton active={activeTab === 'overview'} onclick={() => activeTab = 'overview'}>
						Overview
					</TabButton>
				{/if}
				{#if getVisibleTabs(relatedAgent).includes('activity')}
					<TabButton active={activeTab === 'activity'} onclick={() => activeTab = 'activity'}>
						Activity
					</TabButton>
				{/if}
				{#if getVisibleTabs(relatedAgent).includes('synthesis')}
					<TabButton active={activeTab === 'synthesis'} onclick={() => activeTab = 'synthesis'}>
						Synthesis
					</TabButton>
				{/if}
				{#if getVisibleTabs(relatedAgent).includes('investigation')}
					<TabButton active={activeTab === 'investigation'} onclick={() => activeTab = 'investigation'}>
						Investigation
					</TabButton>
				{/if}
				{#if getVisibleTabs(relatedAgent).includes('screenshots')}
					<TabButton active={activeTab === 'screenshots'} onclick={() => activeTab = 'screenshots'}>
						Screenshots
					</TabButton>
				{/if}
			</div>
		</div>

		<!-- Tab Content -->
		<div class="flex-1 overflow-hidden flex flex-col">
			{#if activeTab === 'overview'}
				<div class="flex-1 overflow-y-auto p-4 space-y-4">
					<div class="grid gap-4 sm:grid-cols-2">
						<div>
							<p class="text-xs text-muted-foreground">Status</p>
							<p class="text-sm">{issue.status || 'unknown'}</p>
						</div>
						<div>
							<p class="text-xs text-muted-foreground">Source</p>
							<p class="text-sm">{issue.source}</p>
						</div>
						<div>
							<p class="text-xs text-muted-foreground">Priority</p>
							<p class="text-sm">P{issue.priority}</p>
						</div>
						{#if issue.effective_priority !== undefined && issue.effective_priority !== issue.priority}
							<div>
								<p class="text-xs text-muted-foreground">Effective Priority</p>
								<p class="text-sm">P{issue.effective_priority}</p>
							</div>
						{/if}
						{#if issue.layer !== undefined}
							<div>
								<p class="text-xs text-muted-foreground">Topological Layer</p>
								<p class="text-sm">{issue.layer}</p>
							</div>
						{/if}
						{#if issue.created_at}
							<div>
								<p class="text-xs text-muted-foreground">Created</p>
								<p class="text-sm">{new Date(issue.created_at).toLocaleString()}</p>
							</div>
						{/if}
						{#if issue.labels && issue.labels.length > 0}
							<div>
								<p class="text-xs text-muted-foreground">Labels</p>
								<div class="mt-1 flex flex-wrap gap-1">
									{#each issue.labels as label}
										<Badge variant="outline" class="text-xs">{label}</Badge>
									{/each}
								</div>
							</div>
						{/if}
					</div>

					<div>
						<p class="text-xs text-muted-foreground mb-1">Description</p>
						{#if issue.description}
							<p class="text-sm whitespace-pre-wrap">{issue.description}</p>
						{:else}
							<p class="text-sm text-muted-foreground">No description provided.</p>
						{/if}
					</div>

					<div class="grid gap-4 sm:grid-cols-2">
						<div>
							<p class="text-xs text-muted-foreground">Blocked By</p>
							<p class="text-sm">{issue.blocked_by?.length ? issue.blocked_by.join(', ') : 'none'}</p>
						</div>
						<div>
							<p class="text-xs text-muted-foreground">Blocks</p>
							<p class="text-sm">{issue.blocks?.length ? issue.blocks.join(', ') : 'none'}</p>
						</div>
					</div>

					{#if relatedAgent}
						<div class="rounded-lg border bg-muted/20 p-3">
							<p class="text-xs text-muted-foreground mb-2">Linked Agent</p>
							<div class="grid gap-2 sm:grid-cols-2 text-sm">
								<div>
									<span class="text-xs text-muted-foreground">Agent ID</span>
									<p class="font-mono text-xs">{relatedAgent.id}</p>
								</div>
								{#if relatedAgent.session_id}
									<div>
										<span class="text-xs text-muted-foreground">Session</span>
										<p class="font-mono text-xs">{relatedAgent.session_id}</p>
									</div>
								{/if}
								{#if relatedAgent.phase}
									<div>
										<span class="text-xs text-muted-foreground">Phase</span>
										<p class="text-sm">{relatedAgent.phase}</p>
									</div>
								{/if}
								{#if relatedAgent.runtime}
									<div>
										<span class="text-xs text-muted-foreground">Runtime</span>
										<p class="text-sm">{relatedAgent.runtime}</p>
									</div>
								{/if}
							</div>
						</div>
					{/if}
				</div>
			{/if}

			{#if activeTab === 'activity' && relatedAgent}
				<ActivityTab agent={relatedAgent} />
			{/if}

			{#if activeTab === 'synthesis' && relatedAgent}
				<div class="flex-1 overflow-y-auto p-4">
					<SynthesisTab agent={relatedAgent} />
				</div>
			{/if}

			{#if activeTab === 'investigation' && relatedAgent}
				<div class="flex-1 overflow-y-auto p-4">
					<InvestigationTab agent={relatedAgent} />
				</div>
			{/if}

			{#if activeTab === 'screenshots' && relatedAgent}
				<div class="flex-1 overflow-y-auto">
					<ScreenshotsTab agent={relatedAgent} />
				</div>
			{/if}
		</div>
	</div>
{/if}
