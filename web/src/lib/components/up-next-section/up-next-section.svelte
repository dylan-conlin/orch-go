<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { readyIssues, type ReadyIssue } from '$lib/stores/beads';
	import { focus } from '$lib/stores/focus';

	interface Props {
		expanded?: boolean;
		maxItems?: number;
	}
	let { expanded = false, maxItems = 5 }: Props = $props();

	// Track if auto-expand has been triggered (to avoid overriding user collapse)
	let autoExpandTriggered = false;

	function toggle() {
		expanded = !expanded;
		// If user manually collapses, don't auto-expand again until urgent items change
		if (!expanded) {
			autoExpandTriggered = true;
		}
	}

	// Filter and sort to top priority items
	let priorityItems = $derived(($readyIssues?.issues ?? [])
		.slice()
		.sort((a, b) => a.priority - b.priority)
		.slice(0, maxItems));

	// Check if any P0 or P1 items exist
	let hasUrgent = $derived(priorityItems.some(i => i.priority <= 1));

	// Auto-expand if P0 or P1 items exist (only if not already triggered)
	$effect(() => {
		if (hasUrgent && !expanded && !autoExpandTriggered) {
			expanded = true;
			autoExpandTriggered = true;
		}
	});

	// Reset auto-expand trigger when urgent items are resolved
	$effect(() => {
		if (!hasUrgent) {
			autoExpandTriggered = false;
		}
	});

	// Focus alignment check
	let focusGoal = $derived($focus?.goal?.toLowerCase() ?? '');
	
	function isFocusAligned(issue: ReadyIssue): boolean {
		if (!focusGoal) return false;
		const titleMatch = issue.title.toLowerCase().includes(focusGoal);
		const labelMatch = (issue.labels ?? []).some(l => 
			focusGoal.includes(l.toLowerCase()) || l.toLowerCase().includes(focusGoal)
		);
		return titleMatch || labelMatch;
	}

	function getPriorityClass(priority: number): string {
		switch (priority) {
			case 0: return 'text-red-500 font-bold';
			case 1: return 'text-orange-500 font-semibold';
			case 2: return 'text-yellow-500';
			default: return 'text-muted-foreground';
		}
	}

	function getPriorityBgClass(priority: number): string {
		switch (priority) {
			case 0: return 'bg-red-500/10';
			case 1: return 'bg-orange-500/10';
			default: return '';
		}
	}

	function getAge(createdAt?: string): string {
		if (!createdAt) return '';
		
		const created = new Date(createdAt);
		const now = new Date();
		const diffMs = now.getTime() - created.getTime();
		const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
		const diffDays = Math.floor(diffHours / 24);
		
		if (diffDays > 0) return `${diffDays}d`;
		if (diffHours > 0) return `${diffHours}h`;
		return '<1h';
	}

	function getPreview(issues: ReadyIssue[]): string {
		if (issues.length === 0) return '';
		
		const titles = issues.slice(0, 2).map(i => {
			const title = i.title;
			return title.length > 25 ? title.substring(0, 25) + '...' : title;
		});
		
		if (issues.length <= 2) {
			return titles.join(', ');
		}
		
		return `${titles.join(', ')} +${issues.length - 2}`;
	}
</script>

{#if $readyIssues && priorityItems.length > 0}
	<div 
		class="rounded-lg border {hasUrgent ? 'border-red-500/30 bg-red-500/5' : 'border-blue-500/30 bg-blue-500/5'}"
		data-testid="up-next-section"
	>
		<button
			class="flex w-full items-center justify-between px-3 py-2 text-left hover:bg-accent/50 transition-colors"
			onclick={toggle}
			aria-expanded={expanded}
			data-testid="up-next-toggle"
		>
			<div class="flex items-center gap-2 min-w-0 flex-1">
				<span class="text-sm flex-shrink-0">🎯</span>
				<span class="text-sm font-medium flex-shrink-0">Up Next</span>
				<Badge variant="secondary" class="h-5 px-1.5 text-xs flex-shrink-0">
					{priorityItems.length}
				</Badge>
				{#if hasUrgent}
					<Badge variant="destructive" class="h-5 px-1.5 text-xs flex-shrink-0">
						{priorityItems.filter(i => i.priority <= 1).length} urgent
					</Badge>
				{/if}
				{#if !expanded}
					<span class="text-xs text-muted-foreground truncate" data-testid="up-next-preview">
						— {getPreview(priorityItems)}
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
			<div class="border-t p-2" data-testid="up-next-content">
				<div class="space-y-1">
					{#each priorityItems as issue (issue.id)}
						<div 
							class="flex items-center gap-2 rounded px-2 py-1.5 text-sm hover:bg-accent/50 {getPriorityBgClass(issue.priority)}"
							data-testid="up-next-issue-{issue.id}"
						>
							<!-- Focus alignment indicator -->
							{#if isFocusAligned(issue)}
								<span class="flex-shrink-0 text-xs" title="Focus aligned">⭐</span>
							{/if}
							
							<!-- Priority indicator -->
							<span class="flex-shrink-0 text-xs font-medium w-6 {getPriorityClass(issue.priority)}">
								P{issue.priority}
							</span>
							
							<!-- Issue title (truncated) -->
							<span class="flex-1 truncate" title={issue.title}>
								{issue.title}
							</span>
							
							<!-- Age indicator -->
							{#if issue.created_at}
								<span class="text-xs text-muted-foreground flex-shrink-0" title="Age">
									{getAge(issue.created_at)}
								</span>
							{/if}
							
							<!-- Labels (show first skill label if exists) -->
							{#if issue.labels && issue.labels.length > 0}
								{@const skillLabel = issue.labels.find(l => l.startsWith('skill:'))}
								{#if skillLabel}
									<Badge variant="outline" class="h-5 px-1.5 text-xs flex-shrink-0">
										{skillLabel.replace('skill:', '')}
									</Badge>
								{/if}
							{/if}
							
							<!-- Issue ID -->
							<span class="text-xs text-muted-foreground flex-shrink-0 font-mono">
								{issue.id}
							</span>
						</div>
					{/each}
				</div>
				
				{#if $readyIssues.count > maxItems}
					<p class="mt-2 text-xs text-muted-foreground text-center">
						+{$readyIssues.count - maxItems} more in queue
					</p>
				{/if}
			</div>
		{/if}
	</div>
{/if}
