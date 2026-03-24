<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { MarkdownContent } from '$lib/components/markdown-content';
	import { reviewQueue, type ReviewQueueIssue, type BriefResponse } from '$lib/stores/beads';
	import { agents, type Agent } from '$lib/stores/agents';

	const API_BASE = 'http://localhost:3348';

	export let expanded: boolean = true;

	// Join review queue items with agent data for synthesis enrichment
	$: agentByBeadsId = new Map<string, Agent>(
		$agents
			.filter((a): a is Agent & { beads_id: string } => !!a.beads_id)
			.map(a => [a.beads_id, a])
	);

	// Brief expansion state: which issue IDs have their brief expanded
	let expandedBriefs: Set<string> = new Set();
	// Brief content cache
	let briefCache: Map<string, BriefResponse> = new Map();
	// Loading state
	let briefLoading: Set<string> = new Set();

	function toggle() {
		expanded = !expanded;
	}

	async function toggleBrief(issueId: string) {
		if (expandedBriefs.has(issueId)) {
			expandedBriefs.delete(issueId);
			expandedBriefs = expandedBriefs; // trigger reactivity
			return;
		}

		// Fetch brief if not cached
		if (!briefCache.has(issueId)) {
			briefLoading.add(issueId);
			briefLoading = briefLoading;
			try {
				const response = await fetch(`${API_BASE}/api/briefs/${issueId}`);
				if (response.ok) {
					const data: BriefResponse = await response.json();
					briefCache.set(issueId, data);
					briefCache = briefCache;
				}
			} catch (e) {
				console.error('Failed to fetch brief:', e);
			} finally {
				briefLoading.delete(issueId);
				briefLoading = briefLoading;
			}
		}

		expandedBriefs.add(issueId);
		expandedBriefs = expandedBriefs;
	}

	async function markAsRead(issueId: string, event: MouseEvent) {
		event.stopPropagation();
		try {
			const response = await fetch(`${API_BASE}/api/briefs/${issueId}`, {
				method: 'POST',
			});
			if (response.ok) {
				// Update cache to reflect read state
				const cached = briefCache.get(issueId);
				if (cached) {
					briefCache.set(issueId, { ...cached, marked_read: true });
					briefCache = briefCache;
				}
			}
		} catch (e) {
			console.error('Failed to mark brief as read:', e);
		}
	}

	function getPreview(issues: ReviewQueueIssue[]): string {
		if (issues.length === 0) return '';

		const titles = issues.slice(0, 2).map((i) => {
			// Use synthesis TLDR if available, otherwise issue title
			const agent = agentByBeadsId.get(i.id);
			const text = agent?.synthesis?.tldr || i.title;
			return text.length > 30 ? text.substring(0, 30) + '...' : text;
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
						{@const synthesis = agentByBeadsId.get(issue.id)?.synthesis}
						{@const brief = briefCache.get(issue.id)}
						{@const isBriefExpanded = expandedBriefs.has(issue.id)}
						{@const isLoading = briefLoading.has(issue.id)}
						<div data-testid="review-issue-{issue.id}">
							<div class="flex items-center gap-1 sm:gap-2 rounded px-2 py-1.5 text-sm hover:bg-accent/50">
								<span class="flex-shrink-0 text-xs font-medium {getTierClass(issue.tier)}">
									{getTierLabel(issue.tier)}
								</span>
								{#if synthesis?.outcome}
									<Badge variant={synthesis.outcome === 'success' ? 'default' : 'secondary'} class="h-4 px-1 text-[10px] flex-shrink-0">
										{synthesis.outcome.split(/\s*\(/)[0].trim()}
									</Badge>
								{/if}
								{#if synthesis?.tldr}
									<span class="flex-1 truncate min-w-0" title={synthesis.tldr}>
										{synthesis.tldr.length > 60 ? synthesis.tldr.substring(0, 57) + '...' : synthesis.tldr}
									</span>
								{:else}
									<span class="flex-1 truncate min-w-0 italic text-muted-foreground" title={issue.title}>
										{issue.title}
									</span>
								{/if}
								{#if !synthesis}
									<Badge variant="outline" class="h-4 px-1 text-[10px] flex-shrink-0 text-muted-foreground/70 border-dashed">
										no synthesis
									</Badge>
								{/if}
								{#if issue.has_brief}
									<button
										class="h-5 px-1.5 text-[10px] rounded border flex-shrink-0 transition-colors
											{brief?.marked_read
												? 'border-emerald-500/30 text-emerald-500/60 bg-emerald-500/5'
												: 'border-blue-500/40 text-blue-500 bg-blue-500/10 hover:bg-blue-500/20'}"
										onclick={() => toggleBrief(issue.id)}
										data-testid="brief-toggle-{issue.id}"
									>
										{#if isLoading}
											...
										{:else if brief?.marked_read}
											read
										{:else}
											brief
										{/if}
									</button>
								{/if}
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

							{#if isBriefExpanded && brief}
								<div class="mx-2 mb-2 p-3 rounded-md border border-blue-500/20 bg-blue-500/5" data-testid="brief-content-{issue.id}">
									<div class="flex items-center justify-between mb-2">
										<span class="text-xs font-medium text-blue-400">Brief</span>
										{#if !brief.marked_read}
											<button
												class="text-xs px-2 py-0.5 rounded border border-emerald-500/40 text-emerald-500 hover:bg-emerald-500/10 transition-colors"
												onclick={(e) => markAsRead(issue.id, e)}
												data-testid="mark-read-{issue.id}"
											>
												Mark as read
											</button>
										{:else}
											<span class="text-xs text-emerald-500/60">Read</span>
										{/if}
									</div>
									<MarkdownContent content={brief.content} />
								</div>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		{/if}
	</div>
{/if}
