<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { errorEvents } from '$lib/stores/agentlog';
	import { pendingReviews, type PendingReviewAgent, type PendingReviewItem } from '$lib/stores/pending-reviews';
	import { beads, blockedIssues, type BlockedIssue } from '$lib/stores/beads';
	import { agents, activeAgents, createIssue } from '$lib/stores/agents';
	import { gaps } from '$lib/stores/gaps';
	import { onMount } from 'svelte';

	// Fetch gaps and blocked issues on mount
	onMount(() => {
		gaps.fetch();
		blockedIssues.fetch();
	});

	// State for issue creation
	let creatingIssue: { [key: string]: boolean } = {};
	let createdIssues: { [key: string]: string } = {};
	let dismissingItem: { [key: string]: boolean } = {};
	let dismissingAllLightTier: boolean = false;

	// Track collapsed state for light-tier section
	let lightTierExpanded = false;

	// 🔴 BLOCKING: Agents at Phase: Complete that need orch complete
	$: completeAgents = $activeAgents.filter(a => 
		a.phase?.toLowerCase() === 'complete'
	);

	// Separate light-tier from standard agents for pending reviews
	$: lightTierAgents = $pendingReviews?.agents.filter(a => a.is_light_tier) ?? [];
	$: standardAgents = $pendingReviews?.agents.filter(a => !a.is_light_tier) ?? [];
	
	// Count total light-tier unreviewed items
	$: lightTierTotalUnreviewed = lightTierAgents.reduce((sum, agent) => 
		sum + getUnreviewedItems(agent).length, 0);

	// Standard reviews that need decision
	$: standardReviewCount = standardAgents.reduce((sum, agent) => 
		sum + getUnreviewedItems(agent).length, 0);

	// ⚠️ DECISION NEEDED: Blocked issues that actually need intervention
	// Only count issues where needs_action is true (blocked by closed/abandoned, or >7 days)
	$: actionableBlocked = ($blockedIssues?.issues ?? []).filter(i => i.needs_action);
	$: totalBlocked = actionableBlocked.length;

	// 📊 PATTERNS: Recurring gaps that could use kn constrain
	$: patternSuggestions = $gaps?.suggestions ?? [];
	$: hasPatterns = patternSuggestions.length > 0;

	// Total errors
	$: totalErrors = $errorEvents.length;

	// Calculate total attention items (keep it small and actionable)
	// Count the number of CATEGORIES that need attention, not individual items
	$: totalAttentionItems = 
		(completeAgents.length > 0 ? 1 : 0) +      // BLOCKING category
		(totalErrors > 0 ? 1 : 0) +                 // ERRORS category
		(totalBlocked > 0 ? 1 : 0) +                // Blocked issues
		(hasPatterns ? 1 : 0);                      // PATTERNS category

	// Helper to check if we have anything to show (excluding light-tier)
	$: hasAttentionItems = totalAttentionItems > 0;

	function getItemKey(workspaceId: string, index: number): string {
		return `${workspaceId}-${index}`;
	}

	async function handleCreateIssue(agent: PendingReviewAgent, item: PendingReviewItem) {
		const key = getItemKey(agent.workspace_id, item.index);
		creatingIssue[key] = true;
		creatingIssue = creatingIssue;
		
		try {
			const cleanAction = item.text.replace(/^[-*]\s*/, '').replace(/^\d+\.\s*/, '');
			const parentContext = agent.beads_id 
				? `\n\nFollow-up from: ${agent.beads_id}`
				: '';
			const description = `${cleanAction}${parentContext}`;
			
			const result = await createIssue(cleanAction.substring(0, 100), description, ['triage:ready']);
			if (result) {
				createdIssues[key] = result.id;
				createdIssues = createdIssues;
				await pendingReviews.markActedOn(agent.workspace_id, item.index);
			}
		} catch (error) {
			console.error('Failed to create issue:', error);
		} finally {
			creatingIssue[key] = false;
			creatingIssue = creatingIssue;
		}
	}

	async function handleDismiss(agent: PendingReviewAgent, item: PendingReviewItem) {
		const key = getItemKey(agent.workspace_id, item.index);
		dismissingItem[key] = true;
		dismissingItem = dismissingItem;
		
		try {
			await pendingReviews.dismiss(agent.workspace_id, item.index);
		} finally {
			dismissingItem[key] = false;
			dismissingItem = dismissingItem;
		}
	}

	async function handleDismissAllLightTier() {
		dismissingAllLightTier = true;
		
		try {
			const promises = lightTierAgents.flatMap(agent => 
				getUnreviewedItems(agent).map(item => 
					pendingReviews.dismiss(agent.workspace_id, item.index)
				)
			);
			await Promise.all(promises);
		} finally {
			dismissingAllLightTier = false;
		}
	}

	function getUnreviewedItems(agent: PendingReviewAgent): PendingReviewItem[] {
		return agent.items.filter(item => !item.reviewed && !item.dismissed && !item.acted_on);
	}

	function formatWorkspaceName(workspaceId: string): string {
		return workspaceId
			.replace(/^[a-z]+-/, '')
			.replace(/-\d{1,2}[a-z]{3}$/, '')
			.replace(/-/g, ' ')
			.trim();
	}

	function formatRuntime(agent: { runtime?: string; spawned_at?: string }): string {
		if (agent.runtime) return agent.runtime;
		if (!agent.spawned_at) return '';
		const ms = Date.now() - new Date(agent.spawned_at).getTime();
		const minutes = Math.floor(ms / 60000);
		if (minutes < 60) return `${minutes}m`;
		const hours = Math.floor(minutes / 60);
		return `${hours}h ${minutes % 60}m`;
	}

	function formatUnixTime(timestamp: number): string {
		return new Date(timestamp * 1000).toLocaleTimeString();
	}

	function copyCommand(cmd: string) {
		navigator.clipboard.writeText(cmd);
	}
</script>

{#if hasAttentionItems || lightTierTotalUnreviewed > 0}
	<div class="rounded-lg border border-amber-500/30 bg-amber-500/5" data-testid="needs-attention-section">
		<div class="flex items-center gap-2 px-3 py-2 border-b border-amber-500/20">
			<span class="text-sm">⚠️</span>
			<span class="text-sm font-medium">Needs Attention</span>
			{#if totalAttentionItems > 0}
				<Badge variant="secondary" class="h-5 px-1.5 text-xs bg-amber-500/20 text-amber-600">
					{totalAttentionItems}
				</Badge>
			{/if}
		</div>
		<div class="p-2 space-y-2">
			<!-- 🔴 BLOCKING: Agents at Phase: Complete need immediate review -->
			{#if completeAgents.length > 0}
				<div class="rounded border bg-card p-2.5 border-red-500/30" data-testid="blocking-section">
					<div class="flex items-center gap-2 mb-2">
						<span class="text-sm">🔴</span>
						<span class="text-xs font-semibold text-red-500 uppercase tracking-wide">Blocking</span>
						<Badge variant="outline" class="h-4 px-1.5 text-[10px] border-red-500/50 text-red-500">
							{completeAgents.length}
						</Badge>
						<span class="text-[10px] text-muted-foreground ml-1">
							— agents waiting for review
						</span>
					</div>
					<div class="space-y-1.5">
						{#each completeAgents.slice(0, 5) as agent (agent.id)}
							<div class="flex items-center justify-between gap-2 rounded-md px-2 py-1.5 hover:bg-muted/50 group transition-colors">
								<div class="flex items-center gap-2 min-w-0 flex-1">
									<code class="text-[10px] font-mono text-muted-foreground bg-muted px-1 py-0.5 rounded shrink-0">
										{agent.beads_id || agent.id.slice(0, 12)}
									</code>
									<span class="text-xs truncate flex-1">{agent.task || formatWorkspaceName(agent.id)}</span>
									<span class="text-[10px] text-muted-foreground shrink-0">{formatRuntime(agent)}</span>
								</div>
								<Tooltip.Root>
									<Tooltip.Trigger>
										{#snippet child({ props })}
											<Button
												{...props}
												variant="outline"
												size="sm"
												class="h-6 px-2 text-[10px] shrink-0 opacity-70 group-hover:opacity-100 transition-opacity"
												onclick={() => copyCommand(`orch complete ${agent.beads_id || agent.id}`)}
											>
												→ complete
											</Button>
										{/snippet}
									</Tooltip.Trigger>
									<Tooltip.Content side="left">
										<p class="text-xs">Click to copy command</p>
										<code class="text-[10px] text-muted-foreground">orch complete {agent.beads_id || agent.id}</code>
									</Tooltip.Content>
								</Tooltip.Root>
							</div>
						{/each}
						{#if completeAgents.length > 5}
							<div class="text-[10px] text-muted-foreground pl-2">
								+{completeAgents.length - 5} more agents waiting
							</div>
						{/if}
					</div>
				</div>
			{/if}

			<!-- ❌ Errors Section (if any) -->
			{#if totalErrors > 0}
				<div class="rounded border bg-card p-2.5 border-red-500/30" data-testid="errors-section">
					<div class="flex items-center gap-2 mb-1.5">
						<span class="text-sm">❌</span>
						<span class="text-xs font-medium text-red-500">Errors</span>
						<Badge variant="outline" class="h-4 px-1.5 text-[10px] border-red-500/50 text-red-500">
							{totalErrors}
						</Badge>
					</div>
					<div class="space-y-0.5 max-h-20 overflow-y-auto">
						{#each $errorEvents.slice().reverse().slice(0, 3) as event (event.id)}
							<div class="flex items-center gap-1.5 text-xs text-muted-foreground px-1">
								<span class="opacity-60 text-[10px] tabular-nums">{formatUnixTime(event.timestamp)}</span>
								{#if event.session_id}
									<code class="font-mono text-[10px]">{event.session_id.slice(0, 8)}</code>
								{/if}
								{#if event.data?.error}
									<span class="text-red-500 truncate text-[10px]">{event.data.error}</span>
								{/if}
							</div>
						{/each}
						{#if totalErrors > 3}
							<div class="text-[10px] text-muted-foreground pl-1">+{totalErrors - 3} more</div>
						{/if}
					</div>
				</div>
			{/if}

			<!-- ⚠️ DECISION NEEDED: Blocked issues requiring human decision -->
			{#if totalBlocked > 0}
				<div class="rounded border bg-card p-2.5 border-orange-500/30" data-testid="decision-section">
					<div class="flex items-center gap-2 mb-2">
						<span class="text-sm">⚠️</span>
						<span class="text-xs font-semibold text-orange-500 uppercase tracking-wide">Decision Needed</span>
						<Badge variant="outline" class="h-4 px-1.5 text-[10px] border-orange-500/50 text-orange-500">
							{totalBlocked}
						</Badge>
						<span class="text-[10px] text-muted-foreground ml-1">
							— blocked issue{totalBlocked === 1 ? '' : 's'} need{totalBlocked === 1 ? 's' : ''} intervention
						</span>
					</div>
					<div class="space-y-1.5">
						{#each actionableBlocked.slice(0, 5) as issue (issue.id)}
							<div class="flex items-center justify-between gap-2 rounded-md px-2 py-1.5 hover:bg-muted/50 group transition-colors">
								<div class="flex items-center gap-2 min-w-0 flex-1">
									<code class="text-[10px] font-mono text-muted-foreground bg-muted px-1 py-0.5 rounded shrink-0">
										{issue.id}
									</code>
									<span class="text-xs truncate flex-1" title={issue.action_reason}>{issue.action_reason}</span>
									{#if issue.days_blocked > 7}
										<Badge variant="outline" class="h-4 px-1.5 text-[10px] border-orange-500/30 text-orange-500 shrink-0">
											{issue.days_blocked}d
										</Badge>
									{/if}
								</div>
								<Tooltip.Root>
									<Tooltip.Trigger>
										{#snippet child({ props })}
											<Button
												{...props}
												variant="outline"
												size="sm"
												class="h-6 px-2 text-[10px] shrink-0 opacity-70 group-hover:opacity-100 transition-opacity"
												onclick={() => {
													if (issue.blocker_status === 'closed') {
														copyCommand(`bd dep remove ${issue.id} ${issue.blocked_by[0]}`);
													} else {
														copyCommand(`bd show ${issue.id}`);
													}
												}}
											>
												{issue.blocker_status === 'closed' ? '→ remove dep' : '→ show'}
											</Button>
										{/snippet}
									</Tooltip.Trigger>
									<Tooltip.Content side="left">
										<p class="text-xs">Click to copy command</p>
										<code class="text-[10px] text-muted-foreground">
											{issue.blocker_status === 'closed' 
												? `bd dep remove ${issue.id} ${issue.blocked_by[0]}`
												: `bd show ${issue.id}`}
										</code>
									</Tooltip.Content>
								</Tooltip.Root>
							</div>
						{/each}
						{#if actionableBlocked.length > 5}
							<div class="text-[10px] text-muted-foreground pl-2">
								+{actionableBlocked.length - 5} more issues needing attention
							</div>
						{/if}
					</div>
				</div>
			{/if}

			<!-- 📊 PATTERN: Recurring gaps that suggest constraints needed -->
			{#if hasPatterns}
				<div class="rounded border bg-card p-2.5 border-blue-500/30" data-testid="pattern-section">
					<div class="flex items-center gap-2 mb-2">
						<span class="text-sm">📊</span>
						<span class="text-xs font-semibold text-blue-500 uppercase tracking-wide">Pattern</span>
						<Badge variant="outline" class="h-4 px-1.5 text-[10px] border-blue-500/50 text-blue-500">
							{$gaps?.recurring_patterns || 0}
						</Badge>
						<span class="text-[10px] text-muted-foreground ml-1">
							— recurring gaps
						</span>
					</div>
					<div class="space-y-1.5">
						{#each patternSuggestions.slice(0, 3) as suggestion (suggestion.query)}
							<div class="flex items-center justify-between gap-2 rounded-md px-2 py-1.5 hover:bg-muted/50 group transition-colors">
								<div class="flex items-center gap-2 min-w-0 flex-1">
									<Badge variant="secondary" class="h-4 px-1.5 text-[10px] shrink-0">
										{suggestion.count}×
									</Badge>
									<span class="text-xs truncate flex-1">"{suggestion.query}"</span>
								</div>
								<Tooltip.Root>
									<Tooltip.Trigger>
										{#snippet child({ props })}
											<Button
												{...props}
												variant="ghost"
												size="sm"
												class="h-6 px-2 text-[10px] shrink-0 opacity-70 group-hover:opacity-100 transition-opacity"
												onclick={() => copyCommand(suggestion.suggestion)}
											>
												→ constrain
											</Button>
										{/snippet}
									</Tooltip.Trigger>
									<Tooltip.Content side="left">
										<p class="text-xs">Click to copy command</p>
										<code class="text-[10px] text-muted-foreground whitespace-pre-wrap max-w-64">{suggestion.suggestion}</code>
									</Tooltip.Content>
								</Tooltip.Root>
							</div>
						{/each}
						{#if patternSuggestions.length > 3}
							<div class="flex items-center justify-between pl-2">
								<span class="text-[10px] text-muted-foreground">
									+{patternSuggestions.length - 3} more patterns
								</span>
								<Button
									variant="ghost"
									size="sm"
									class="h-5 px-2 text-[10px]"
									onclick={() => copyCommand('orch learn')}
								>
									orch learn →
								</Button>
							</div>
						{/if}
					</div>
				</div>
			{/if}

			<!-- ⚡ Light-tier stale recommendations (collapsed by default, separate from main attention count) -->
			{#if lightTierTotalUnreviewed > 0}
				<div class="rounded border bg-muted/20 border-muted-foreground/10" data-testid="light-tier-section">
					<button
						class="flex items-center justify-between w-full px-2.5 py-1.5 text-left hover:bg-muted/30 transition-colors rounded"
						onclick={() => { lightTierExpanded = !lightTierExpanded; }}
					>
						<div class="flex items-center gap-2">
							<Badge variant="secondary" class="text-[10px] bg-slate-500/20 text-slate-500 border-slate-500/30 h-4 px-1.5">
								⚡ light
							</Badge>
							<span class="text-[10px] text-muted-foreground">
								{lightTierTotalUnreviewed} stale from {lightTierAgents.length} agent{lightTierAgents.length === 1 ? '' : 's'}
							</span>
						</div>
						<div class="flex items-center gap-2">
							<Button
								variant="ghost"
								size="sm"
								class="h-5 px-2 text-[10px]"
								onclick={(e: Event) => { e.stopPropagation(); handleDismissAllLightTier(); }}
								disabled={dismissingAllLightTier}
							>
								{dismissingAllLightTier ? '...' : 'Dismiss All'}
							</Button>
							<span class="text-muted-foreground transition-transform {lightTierExpanded ? 'rotate-180' : ''}">
								<svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
									<polyline points="6 9 12 15 18 9"></polyline>
								</svg>
							</span>
						</div>
					</button>
					{#if lightTierExpanded}
						<div class="px-2.5 pb-2 pt-1 space-y-1 border-t border-muted-foreground/10">
							{#each lightTierAgents as agent (agent.workspace_id)}
								{@const unreviewedItems = getUnreviewedItems(agent)}
								{#if unreviewedItems.length > 0}
									<div class="text-[10px] text-muted-foreground">
										<span class="font-medium">{formatWorkspaceName(agent.workspace_id)}</span>
										<span class="opacity-60"> ({unreviewedItems.length})</span>
									</div>
								{/if}
							{/each}
						</div>
					{/if}
				</div>
			{/if}
		</div>
	</div>
{/if}
