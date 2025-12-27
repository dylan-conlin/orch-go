<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { readyIssues, type ReadyIssue } from '$lib/stores/beads';

	export let expanded: boolean = false;

	function toggle() {
		expanded = !expanded;
	}

	function getPreview(issues: ReadyIssue[]): string {
		if (issues.length === 0) return '';
		
		const titles = issues.slice(0, 2).map(i => {
			const title = i.title;
			return title.length > 30 ? title.substring(0, 30) + '...' : title;
		});
		
		if (issues.length <= 2) {
			return titles.join(', ');
		}
		
		return `${titles.join(', ')} +${issues.length - 2}`;
	}

	function getPriorityClass(priority: number): string {
		switch (priority) {
			case 0: return 'text-red-500';
			case 1: return 'text-orange-500';
			case 2: return 'text-yellow-500';
			default: return 'text-muted-foreground';
		}
	}
</script>

{#if $readyIssues && $readyIssues.count > 0}
	<div class="rounded-lg border border-blue-500/30 bg-blue-500/5" data-testid="ready-queue-section">
		<button
			class="flex w-full items-center justify-between px-3 py-2 text-left hover:bg-accent/50 transition-colors"
			onclick={toggle}
			aria-expanded={expanded}
			data-testid="ready-queue-toggle"
		>
			<div class="flex items-center gap-2 min-w-0 flex-1">
				<span class="text-sm flex-shrink-0">📋</span>
				<span class="text-sm font-medium flex-shrink-0">Ready Queue</span>
				<Badge variant="secondary" class="h-5 px-1.5 text-xs flex-shrink-0">
					{$readyIssues.count}
				</Badge>
				{#if !expanded}
					<span class="text-xs text-muted-foreground truncate" data-testid="ready-queue-preview">
						— {getPreview($readyIssues.issues)}
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
			<div class="border-t p-2" data-testid="ready-queue-content">
				{#if $readyIssues.issues.length === 0}
					<p class="py-4 text-center text-sm text-muted-foreground">No ready issues</p>
				{:else}
					<div class="space-y-1">
						{#each $readyIssues.issues as issue (issue.id)}
							<div class="flex items-center gap-2 rounded px-2 py-1.5 text-sm hover:bg-accent/50" data-testid="ready-issue-{issue.id}">
								<!-- Priority indicator -->
								<span class="flex-shrink-0 text-xs font-medium {getPriorityClass(issue.priority)}">
									P{issue.priority}
								</span>
								<!-- Issue title (truncated) -->
								<span class="flex-1 truncate" title={issue.title}>
									{issue.title}
								</span>
								<!-- Issue type -->
								<Badge variant="outline" class="h-5 px-1.5 text-xs flex-shrink-0">
									{issue.issue_type}
								</Badge>
								<!-- Labels (show first 2, max) -->
								{#if issue.labels && issue.labels.length > 0}
									{#each issue.labels.slice(0, 2) as label}
										<Badge variant="secondary" class="h-5 px-1.5 text-xs flex-shrink-0">
											{label}
										</Badge>
									{/each}
									{#if issue.labels.length > 2}
										<span class="text-xs text-muted-foreground">+{issue.labels.length - 2}</span>
									{/if}
								{/if}
								<!-- Issue ID -->
								<span class="text-xs text-muted-foreground flex-shrink-0 font-mono">
									{issue.id}
								</span>
							</div>
						{/each}
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}
