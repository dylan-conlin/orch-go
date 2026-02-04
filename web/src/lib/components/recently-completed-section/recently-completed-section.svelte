<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { attention, ATTENTION_BADGE_CONFIG, type CompletedIssue } from '$lib/stores/attention';
	import { cn } from '$lib/utils';

	export let completedIssues: CompletedIssue[] = [];
	export let onSelectItem: (index: number) => void = () => {};
	export let selectedIndex: number = -1;
	export let startIndex: number = 0; // Starting index in flattened list

	let expanded = false;

	// Sort by urgency: needs_fix first, then by priority
	$: sortedIssues = [...completedIssues]
		.filter(issue => issue.verificationStatus !== 'verified')
		.sort((a, b) => {
			// needs_fix before unverified
			if (a.verificationStatus === 'needs_fix' && b.verificationStatus !== 'needs_fix') return -1;
			if (b.verificationStatus === 'needs_fix' && a.verificationStatus !== 'needs_fix') return 1;
			// then by priority
			return a.priority - b.priority;
		});

	// Count by status for grouping display
	$: statusCounts = {
		needs_fix: sortedIssues.filter(i => i.verificationStatus === 'needs_fix').length,
		unverified: sortedIssues.filter(i => i.verificationStatus === 'unverified').length
	};

	function toggle() {
		expanded = !expanded;
	}

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

	// Handle keyboard shortcuts within section
	function handleKeydown(event: KeyboardEvent, issue: CompletedIssue) {
		if (event.key === 'v' && issue.verificationStatus === 'unverified') {
			event.preventDefault();
			attention.markVerified(issue.id);
		} else if (event.key === 'x' && issue.verificationStatus === 'unverified') {
			event.preventDefault();
			attention.markNeedsFix(issue.id);
		}
	}
</script>

{#if sortedIssues.length > 0}
	<div
		class="recently-completed-section mb-4 rounded-lg border border-zinc-700 bg-zinc-900/50"
		data-testid="recently-completed-section"
	>
		<!-- Section Header -->
		<button
			class="flex w-full items-center justify-between px-4 py-3 text-left hover:bg-zinc-800/50 transition-colors rounded-t-lg"
			onclick={toggle}
			data-testid="recently-completed-toggle"
		>
			<div class="flex items-center gap-3">
				<span class="text-sm">✓</span>
				<span class="text-sm font-medium text-foreground">Recently Completed</span>
				<Badge variant="secondary" class="h-5 px-2 text-xs">
					{sortedIssues.length}
				</Badge>
				{#if statusCounts.needs_fix > 0}
					<Badge variant="destructive" class="h-5 px-2 text-xs">
						{statusCounts.needs_fix} needs fix
					</Badge>
				{/if}
				{#if !expanded && sortedIssues.length > 0}
					<span class="text-xs text-muted-foreground truncate max-w-[200px]">
						— {sortedIssues[0].title}
						{#if sortedIssues.length > 1}
							+{sortedIssues.length - 1}
						{/if}
					</span>
				{/if}
			</div>
			<span class="text-muted-foreground transition-transform {expanded ? 'rotate-180' : ''}">
				<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="6 9 12 15 18 9"></polyline>
				</svg>
			</span>
		</button>

		<!-- Section Content -->
		{#if expanded}
			<div class="border-t border-zinc-700 py-2" data-testid="recently-completed-content">
				{#each sortedIssues as issue, index (issue.id)}
					{@const badgeConfig = getAttentionBadge(issue.attentionBadge)}
					{@const itemIndex = startIndex + index}
					<div
						class={cn(
							"flex items-center gap-3 py-2 px-4 cursor-pointer transition-colors",
							selectedIndex === itemIndex && "bg-zinc-800",
							issue.verificationStatus === 'needs_fix' && "bg-red-950/20"
						)}
						data-testid="completed-row-{issue.id}"
						data-node-index={itemIndex}
						onclick={() => onSelectItem(itemIndex)}
						onkeydown={(e) => handleKeydown(e, issue)}
						role="treeitem"
						tabindex="-1"
					>
						<!-- Verification status icon -->
						<span class="w-5 text-center">
							{#if issue.verificationStatus === 'needs_fix'}
								<span class="text-red-500">✗</span>
							{:else}
								<span class="text-yellow-500">○</span>
							{/if}
						</span>

						<!-- Priority badge -->
						<Badge variant={getPriorityVariant(issue.priority)} class="w-8 justify-center text-xs">
							P{issue.priority}
						</Badge>

						<!-- ID -->
						<span class="text-xs font-mono min-w-[120px] text-muted-foreground">
							{issue.id}
						</span>

						<!-- Title -->
						<span
							class="flex-1 text-sm font-medium truncate"
							class:line-through={issue.verificationStatus === 'needs_fix'}
							class:text-muted-foreground={issue.verificationStatus === 'needs_fix'}
							class:text-foreground={issue.verificationStatus !== 'needs_fix'}
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
					</div>
				{/each}
			</div>
		{/if}
	</div>
{/if}
