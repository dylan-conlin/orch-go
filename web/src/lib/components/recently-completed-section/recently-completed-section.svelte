<script lang="ts">
	import { onMount } from 'svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { cn } from '$lib/utils';
	import {
		attention,
		ATTENTION_BADGE_CONFIG,
		formatRelativeTime,
		type CompletedIssue
	} from '$lib/stores/attention';
	import { IssueSidePanel } from '$lib/components/issue-side-panel';

	export let completedIssues: CompletedIssue[] = [];

	let flattenedItems: CompletedIssue[] = [];
	let selectedIndex = 0;
	let expandedDetails = new Set<string>();
	let selectedIssueForPanel: CompletedIssue | null = null;
	let containerElement: HTMLDivElement;
	let copiedId: string | null = null;
	let copiedTimeout: ReturnType<typeof setTimeout> | null = null;

	function completedAtMs(issue: CompletedIssue): number {
		const ms = new Date(issue.completedAt).getTime();
		if (Number.isNaN(ms)) return 0;
		return ms;
	}

	// Sort by completion recency (most recent first)
	$: {
		flattenedItems = [...completedIssues]
			.sort((a, b) => {
				const timeDiff = completedAtMs(b) - completedAtMs(a);
				if (timeDiff !== 0) return timeDiff;
				return a.priority - b.priority;
			});
		if (selectedIndex >= flattenedItems.length) {
			selectedIndex = Math.max(0, flattenedItems.length - 1);
		}
	}

	$: statusCounts = {
		needs_fix: flattenedItems.filter(i => i.verificationStatus === 'needs_fix').length,
		unverified: flattenedItems.filter(i => i.verificationStatus === 'unverified').length,
		verified: flattenedItems.filter(i => i.verificationStatus === 'verified').length
	};

	onMount(() => {
		setTimeout(() => containerElement?.focus(), 100);
	});

	function getAttentionBadge(badge: 'unverified' | 'needs_fix' | undefined) {
		if (!badge) return null;
		return ATTENTION_BADGE_CONFIG[badge] || null;
	}

	function getPriorityVariant(priority: number): 'destructive' | 'secondary' | 'outline' {
		if (priority === 0) return 'destructive';
		if (priority === 1) return 'secondary';
		return 'outline';
	}

	function getTypeBadge(type: string): string {
		switch (type.toLowerCase()) {
			case 'epic': return 'bg-purple-500/10 text-purple-500';
			case 'feature': return 'bg-blue-500/10 text-blue-500';
			case 'bug': return 'bg-red-500/10 text-red-500';
			case 'task': return 'bg-green-500/10 text-green-500';
			case 'question': return 'bg-yellow-500/10 text-yellow-500';
			default: return 'bg-muted text-muted-foreground';
		}
	}

	async function copyToClipboard(id: string) {
		try {
			await navigator.clipboard.writeText(id);
			if (copiedTimeout) clearTimeout(copiedTimeout);
			copiedId = id;
			copiedTimeout = setTimeout(() => {
				copiedId = null;
				copiedTimeout = null;
			}, 1500);
		} catch (err) {
			console.error('Failed to copy to clipboard:', err);
		}
	}

	function scrollToSelected() {
		const element = document.querySelector(`[data-completed-index="${selectedIndex}"]`);
		if (element) {
			element.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
		}
	}

	function closeSidePanel() {
		selectedIssueForPanel = null;
		setTimeout(() => containerElement?.focus(), 0);
	}

	function handleKeyDown(event: KeyboardEvent) {
		const current = flattenedItems[selectedIndex];
		if (!current) return;

		switch (event.key) {
			case 'j':
			case 'ArrowDown':
				event.preventDefault();
				selectedIndex = Math.min(selectedIndex + 1, flattenedItems.length - 1);
				scrollToSelected();
				break;

			case 'k':
			case 'ArrowUp':
				event.preventDefault();
				selectedIndex = Math.max(selectedIndex - 1, 0);
				scrollToSelected();
				break;

			case 'Enter':
				event.preventDefault();
				if (expandedDetails.has(current.id)) {
					expandedDetails.delete(current.id);
				} else {
					expandedDetails.add(current.id);
				}
				expandedDetails = expandedDetails;
				break;

			case 'Escape':
				event.preventDefault();
				if (selectedIssueForPanel) {
					selectedIssueForPanel = null;
					setTimeout(() => containerElement?.focus(), 0);
				} else if (expandedDetails.has(current.id)) {
					expandedDetails.delete(current.id);
					expandedDetails = expandedDetails;
				}
				break;

			case 'i':
			case 'o':
				event.preventDefault();
				if (selectedIssueForPanel?.id === current.id) {
					selectedIssueForPanel = null;
					setTimeout(() => containerElement?.focus(), 0);
				} else {
					selectedIssueForPanel = current;
				}
				break;

			case 'v':
				event.preventDefault();
				if (current.verificationStatus === 'unverified') {
					attention.markVerified(current.id);
				}
				break;

			case 'x':
				event.preventDefault();
				if (current.verificationStatus === 'unverified') {
					attention.markNeedsFix(current.id);
				}
				break;

			case 'g':
				event.preventDefault();
				selectedIndex = 0;
				scrollToSelected();
				break;

			case 'G':
				event.preventDefault();
				selectedIndex = flattenedItems.length - 1;
				scrollToSelected();
				break;

			case 'c':
				event.preventDefault();
				copyToClipboard(current.id);
				break;
		}
	}
</script>

<div
	bind:this={containerElement}
	class="completed-view h-full overflow-y-auto px-0 py-2 focus:outline-none"
	role="tree"
	tabindex="0"
	onkeydown={handleKeyDown}
	data-testid="completed-view"
>
	{#if flattenedItems.length === 0}
		<div class="flex items-center justify-center h-full">
			<div class="text-muted-foreground">No recently completed issues</div>
		</div>
	{:else}
		<!-- Summary bar -->
		<div class="flex items-center gap-3 px-4 py-2 mb-2 text-sm text-muted-foreground">
			<span>{flattenedItems.length} completed</span>
			{#if statusCounts.needs_fix > 0}
				<Badge variant="destructive" class="h-5 px-2 text-xs">
					{statusCounts.needs_fix} needs fix
				</Badge>
			{/if}
			{#if statusCounts.unverified > 0}
				<Badge variant="secondary" class="h-5 px-2 text-xs">
					{statusCounts.unverified} unverified
				</Badge>
			{/if}
			{#if statusCounts.verified > 0}
				<Badge variant="outline" class="h-5 px-2 text-xs text-green-500 border-green-500/30">
					{statusCounts.verified} verified
				</Badge>
			{/if}
		</div>

		{#each flattenedItems as issue, index (issue.id)}
			{@const badgeConfig = getAttentionBadge(issue.attentionBadge)}
			<div
				data-testid="completed-row-{issue.id}"
				data-completed-index={index}
				class="node-row cursor-pointer select-none focus:outline-none"
				class:selected={index === selectedIndex}
				class:focused={index === selectedIndex}
				role="treeitem"
				aria-selected={index === selectedIndex}
				tabindex="-1"
				onclick={() => { selectedIndex = index; }}
			>
				<div
					class={cn(
						"flex items-center gap-3 py-2 px-1 rounded transition-colors",
						index === selectedIndex ? 'bg-zinc-800' : '',
						issue.verificationStatus === 'needs_fix' && "bg-red-950/20"
					)}
					style="padding-left: 0"
				>
					<!-- Expansion indicator placeholder -->
					<span class="w-4"></span>

					<!-- Verification status icon -->
					<span class="w-5 text-center">
						{#if issue.verificationStatus === 'needs_fix'}
							<span class="text-red-500">✗</span>
						{:else if issue.verificationStatus === 'verified'}
							<span class="text-green-500">✓</span>
						{:else}
							<span class="text-yellow-500">○</span>
						{/if}
					</span>

					<!-- Priority badge -->
					<Badge variant={getPriorityVariant(issue.priority)} class="w-8 justify-center text-xs">
						P{issue.priority}
					</Badge>

					<!-- ID -->
					<span
						class="text-xs font-mono min-w-[120px] cursor-pointer hover:text-foreground transition-colors {copiedId === issue.id ? 'text-green-500' : 'text-muted-foreground'}"
						onclick={(e) => { e.stopPropagation(); copyToClipboard(issue.id); }}
						onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); e.preventDefault(); copyToClipboard(issue.id); }}}
						role="button"
						tabindex="-1"
						title="Click to copy"
					>
						{copiedId === issue.id ? 'Copied!' : issue.id}
					</span>

					<!-- Title -->
					<span
						class="flex-1 text-sm font-medium truncate"
						class:line-through={issue.verificationStatus === 'needs_fix'}
						class:text-muted-foreground={issue.verificationStatus === 'needs_fix' || issue.verificationStatus === 'verified'}
						class:text-foreground={issue.verificationStatus === 'unverified'}
					>
						{issue.title}
					</span>

					<!-- Attention badge -->
					{#if badgeConfig}
						<Badge variant={badgeConfig.variant} class="shrink-0">
							{badgeConfig.label}
						</Badge>
					{/if}

					<!-- Type badge -->
					<Badge variant="outline" class="{getTypeBadge(issue.type)} text-xs shrink-0">
						{issue.type}
					</Badge>

					<!-- Relative completion timestamp -->
					<span class="text-[11px] text-muted-foreground shrink-0 min-w-[45px] text-right">
						{issue.completedAt ? formatRelativeTime(issue.completedAt) : ''}
					</span>
				</div>

				<!-- L1: Expanded details -->
				{#if expandedDetails.has(issue.id)}
					<div class="expanded-details ml-14 mt-1 mb-2 p-3 bg-muted/30 rounded text-sm space-y-2">
						{#if issue.description}
							<div>
								<span class="text-xs font-semibold uppercase text-foreground">Description:</span>
								<p class="mt-1 text-xs text-muted-foreground">{issue.description}</p>
							</div>
						{/if}

						<div class="flex items-center gap-4 text-xs">
							{#if issue.completedAt}
								<span class="flex items-center gap-1">
									<span class="text-foreground/60">Completed:</span>
									<span class="text-muted-foreground">{new Date(issue.completedAt).toLocaleString()}</span>
								</span>
							{/if}
							<span class="flex items-center gap-1">
								<span class="text-foreground/60">Status:</span>
								<span class={issue.verificationStatus === 'needs_fix' ? 'text-red-500' : issue.verificationStatus === 'verified' ? 'text-green-500' : 'text-yellow-500'}>
									{issue.verificationStatus}
								</span>
							</span>
						</div>

						<div class="text-xs text-muted-foreground border-t border-border pt-2 mt-2">
							{#if issue.verificationStatus === 'unverified'}
								Press <kbd class="px-1 py-0.5 bg-muted rounded text-foreground">v</kbd> to verify or <kbd class="px-1 py-0.5 bg-muted rounded text-foreground">x</kbd> to mark needs fix
							{:else if issue.verificationStatus === 'needs_fix'}
								Marked as needing fix — reopen or reassign this issue
							{:else if issue.verificationStatus === 'verified'}
								Verified — this issue has been confirmed complete
							{/if}
						</div>
					</div>
				{/if}
			</div>
		{/each}
	{/if}
</div>

<!-- Issue Side Panel -->
{#if selectedIssueForPanel}
	<IssueSidePanel issue={{ ...selectedIssueForPanel, depth: 0, expanded: false, children: [], parent_id: undefined, blocked_by: [], blocks: [] }} on:close={closeSidePanel} />
{/if}
