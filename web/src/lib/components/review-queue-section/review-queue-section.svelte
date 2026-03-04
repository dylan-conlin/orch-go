<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { reviewQueue, type ReviewQueueIssue } from '$lib/stores/beads';

	export let expanded: boolean = true;

	function toggle() {
		expanded = !expanded;
	}

	function getPreview(issues: ReviewQueueIssue[]): string {
		if (issues.length === 0) return '';

		const titles = issues.slice(0, 2).map((i) => {
			const title = i.title;
			return title.length > 30 ? title.substring(0, 30) + '...' : title;
		});

		if (issues.length <= 2) {
			return titles.join(', ');
		}

		return `${titles.join(', ')} +${issues.length - 2}`;
	}

	function getTierLabel(tier: number): string {
		switch (tier) {
			case 1: return 'T1';
			case 2: return 'T2';
			case 3: return 'T3';
			default: return 'T?';
		}
	}

	function getTierClass(tier: number): string {
		switch (tier) {
			case 1: return 'text-orange-500';
			case 2: return 'text-yellow-500';
			case 3: return 'text-muted-foreground';
			default: return 'text-muted-foreground';
		}
	}

	function getGateStatus(issue: ReviewQueueIssue): string {
		if (issue.tier === 1) {
			if (!issue.gate1 && !issue.gate2) return 'needs both gates';
			if (!issue.gate1) return 'needs comprehension';
			if (!issue.gate2) return 'needs behavioral';
		} else if (issue.tier === 2) {
			if (!issue.gate1) return 'needs comprehension';
		}
		return '';
	}

	$: sortedIssues = ($reviewQueue?.issues || []).slice().sort((a, b) => {
		if (a.tier !== b.tier) return a.tier - b.tier;
		return a.id.localeCompare(b.id);
	});
</script>

{#if $reviewQueue && $reviewQueue.count > 0}
	<div class="rounded-lg border border-emerald-500/40 bg-emerald-500/5" data-testid="review-queue-section">
		<button
			class="flex w-full items-center justify-between px-3 py-2 text-left hover:bg-accent/50 transition-colors"
			onclick={toggle}
			aria-expanded={expanded}
			data-testid="review-queue-toggle"
		>
			<div class="flex items-center gap-2 min-w-0 flex-1">
				<span class="text-sm flex-shrink-0">✅</span>
				<span class="text-sm font-medium flex-shrink-0">Review Queue</span>
				<Badge variant="secondary" class="h-5 px-1.5 text-xs flex-shrink-0 bg-emerald-500/20 text-emerald-600">
					{$reviewQueue.count}
				</Badge>
				<span class="text-xs text-emerald-500/80 truncate">completions awaiting review</span>
				{#if !expanded}
					<span class="text-xs text-muted-foreground truncate" data-testid="review-queue-preview">
						— {getPreview(sortedIssues)}
					</span>
				{/if}
			</div>
			<span class="text-muted-foreground transition-transform flex-shrink-0 {expanded ? 'rotate-180' : ''}">
				<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="6 9 12 15 18 9"></polyline>
				</svg>
			</span>
		</button>

		{#if expanded}
			<div class="border-t p-2" data-testid="review-queue-content">
				<div class="space-y-1">
					{#each sortedIssues as issue (issue.id)}
						<div class="flex items-center gap-1 sm:gap-2 rounded px-2 py-1.5 text-sm hover:bg-accent/50" data-testid="review-issue-{issue.id}">
							<span class="flex-shrink-0 text-xs font-medium {getTierClass(issue.tier)}">
								{getTierLabel(issue.tier)}
							</span>
							<span class="flex-1 truncate min-w-0" title={issue.title}>
								{issue.title}
							</span>
							<Badge variant="outline" class="h-5 px-1.5 text-xs flex-shrink-0">
								{issue.issue_type}
							</Badge>
							{#if getGateStatus(issue)}
								<span class="text-xs text-amber-500 flex-shrink-0 hidden sm:inline">{getGateStatus(issue)}</span>
							{/if}
							<span class="text-xs text-muted-foreground flex-shrink-0 font-mono hidden sm:inline">
								{issue.id}
							</span>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	</div>
{/if}
