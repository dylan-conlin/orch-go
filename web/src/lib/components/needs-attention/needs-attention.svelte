<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { errorEvents } from '$lib/stores/agentlog';
	import { pendingReviews, type PendingReviewAgent, type PendingReviewItem } from '$lib/stores/pending-reviews';
	import { beads } from '$lib/stores/beads';
	import { agents, activeAgents, createIssue } from '$lib/stores/agents';
	import { gaps } from '$lib/stores/gaps';
	import { onMount } from 'svelte';

	// Fetch gaps on mount
	onMount(() => {
		gaps.fetch();
	});

	// State for issue creation
	let creatingIssue: { [key: string]: boolean } = {};
	let createdIssues: { [key: string]: string } = {};
	let dismissingItem: { [key: string]: boolean } = {};
	let dismissingAllLightTier: boolean = false;

	// BLOCKING: Agents at Phase: Complete that need orch complete
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

	// PATTERNS: Recurring gaps that could use kn constrain
	$: patternSuggestions = $gaps?.suggestions ?? [];
	$: hasPatterns = patternSuggestions.length > 0;

	// Calculate total actionable items
	// Count by *category* not individual items (keeps count small and actionable)
	$: totalErrors = $errorEvents.length;
	$: totalBlocked = $beads?.blocked_issues ?? 0;
	$: totalBlockingAgents = completeAgents.length;
	$: standardAgentCount = standardAgents.filter(a => getUnreviewedItems(a).length > 0).length;
	
	// Decision items: blocked issues + standard agent reviews
	$: totalDecisions = (totalBlocked > 0 ? 1 : 0) + standardAgentCount;
	
	// Total attention items: count categories, not individual items
	// BLOCKING (complete agents), Errors, DECISION (blocked + standard agent count), PATTERNS
	$: totalAttentionItems = 
		(totalBlockingAgents > 0 ? 1 : 0) +  // BLOCKING section
		(totalErrors > 0 ? 1 : 0) +            // Errors section
		totalDecisions +                       // DECISION section
		(hasPatterns ? 1 : 0);                 // PATTERNS section

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

{#if totalAttentionItems > 0 || lightTierTotalUnreviewed > 0}
	<div class="rounded-lg border border-amber-500/30 bg-amber-500/5" data-testid="needs-attention-section">
		<div class="flex items-center gap-2 px-3 py-2 border-b">
			<span class="text-sm">⚠️</span>
			<span class="text-sm font-medium">Needs Attention</span>
			<Badge variant="secondary" class="h-5 px-1.5 text-xs bg-amber-500/20 text-amber-600">
				{totalAttentionItems}
			</Badge>
		</div>
		<div class="p-2 space-y-2">
			<!-- 🔴 BLOCKING: Agents at Phase: Complete -->
			{#if completeAgents.length > 0}
				<div class="rounded border bg-card p-2 border-red-500/30">
					<div class="flex items-center gap-2 mb-2">
						<span class="text-sm">🔴</span>
						<span class="text-xs font-semibold text-red-500">BLOCKING</span>
						<span class="text-xs text-muted-foreground">
							{completeAgents.length} agent{completeAgents.length === 1 ? '' : 's'} at Phase: Complete need review
						</span>
					</div>
					<div class="space-y-1">
						{#each completeAgents as agent (agent.id)}
							<div class="flex items-center justify-between gap-2 rounded p-1.5 hover:bg-muted/50 group">
								<div class="flex items-center gap-2 min-w-0">
									<span class="text-[10px] font-mono text-muted-foreground">{agent.beads_id || agent.id.slice(0, 12)}</span>
									<span class="text-xs truncate">{agent.task || formatWorkspaceName(agent.id)}</span>
									<span class="text-[10px] text-muted-foreground">{formatRuntime(agent)}</span>
								</div>
								<Button
									variant="outline"
									size="sm"
									class="h-5 px-2 text-[10px] shrink-0"
									onclick={() => copyCommand(`orch complete ${agent.beads_id || agent.id}`)}
								>
									→ orch complete
								</Button>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<!-- ❌ Errors Section -->
			{#if totalErrors > 0}
				<div class="rounded border bg-card p-2 border-red-500/30">
					<div class="flex items-center gap-2 mb-1">
						<span class="text-sm">❌</span>
						<span class="text-xs font-medium text-red-500">Errors ({totalErrors})</span>
					</div>
					<div class="space-y-0.5 max-h-24 overflow-y-auto">
						{#each $errorEvents.slice().reverse().slice(0, 5) as event (event.id)}
							<div class="flex items-center gap-1 text-xs text-muted-foreground">
								<span class="opacity-60">{formatUnixTime(event.timestamp)}</span>
								{#if event.session_id}
									<span class="font-mono">{event.session_id.slice(0, 8)}</span>
								{/if}
								{#if event.data?.error}
									<span class="text-red-500 truncate">{event.data.error}</span>
								{/if}
							</div>
						{/each}
						{#if totalErrors > 5}
							<div class="text-[10px] text-muted-foreground">+{totalErrors - 5} more errors</div>
						{/if}
					</div>
				</div>
			{/if}

			<!-- ⚠️ DECISION NEEDED: Blocked Issues + Standard Reviews -->
			{#if totalDecisions > 0}
				<div class="rounded border bg-card p-2 border-orange-500/30">
					<div class="flex items-center gap-2 mb-2">
						<span class="text-sm">⚠️</span>
						<span class="text-xs font-semibold text-orange-500">DECISION NEEDED</span>
						<Badge variant="outline" class="h-4 px-1 text-[10px]">{totalDecisions}</Badge>
					</div>

					<!-- Blocked Issues -->
					{#if totalBlocked > 0}
						<div class="flex items-center gap-2 p-1.5 rounded hover:bg-muted/50 mb-1">
							<span class="text-xs">🚧</span>
							<span class="text-xs">{totalBlocked} blocked issue{totalBlocked === 1 ? '' : 's'}</span>
							<Button
								variant="ghost"
								size="sm"
								class="h-5 px-2 text-[10px] ml-auto"
								onclick={() => copyCommand('bd blocked')}
							>
								→ bd blocked
							</Button>
						</div>
					{/if}

					<!-- Standard Reviews (not light-tier) -->
					{#each standardAgents as agent (agent.workspace_id)}
						{@const unreviewedItems = getUnreviewedItems(agent)}
						{#if unreviewedItems.length > 0}
							<div class="rounded border bg-card/50 p-1.5 mb-1">
								<div class="flex items-center gap-1 mb-1">
									<span class="text-[10px] font-medium truncate">{formatWorkspaceName(agent.workspace_id)}</span>
									<Badge variant="outline" class="h-4 px-1 text-[10px]">{unreviewedItems.length}</Badge>
								</div>

								<div class="space-y-0.5">
									{#each unreviewedItems.slice(0, 2) as item (item.index)}
										{@const key = getItemKey(agent.workspace_id, item.index)}
										<div class="flex items-start gap-1 rounded p-1 hover:bg-muted/50 group text-xs">
											<span class="flex-1 text-[10px] truncate">{item.text}</span>
											<div class="flex gap-0.5 opacity-0 group-hover:opacity-100 transition-opacity">
												{#if createdIssues[key]}
													<span class="text-[10px] text-green-500 px-1">{createdIssues[key]}</span>
												{:else}
													<Button
														variant="outline"
														size="sm"
														class="h-4 px-1 text-[9px]"
														onclick={() => handleCreateIssue(agent, item)}
														disabled={creatingIssue[key]}
													>
														{creatingIssue[key] ? '...' : '→ Issue'}
													</Button>
													<Button
														variant="ghost"
														size="sm"
														class="h-4 px-1 text-[9px]"
														onclick={() => handleDismiss(agent, item)}
														disabled={dismissingItem[key]}
													>
														✕
													</Button>
												{/if}
											</div>
										</div>
									{/each}
									{#if unreviewedItems.length > 2}
										<div class="text-[10px] text-muted-foreground pl-1">+{unreviewedItems.length - 2} more</div>
									{/if}
								</div>
							</div>
						{/if}
					{/each}
				</div>
			{/if}

			<!-- 📊 PATTERN: Recurring gaps -->
			{#if hasPatterns}
				<div class="rounded border bg-card p-2 border-blue-500/30">
					<div class="flex items-center gap-2 mb-2">
						<span class="text-sm">📊</span>
						<span class="text-xs font-semibold text-blue-500">PATTERN</span>
						<span class="text-xs text-muted-foreground">Recurring gaps ({$gaps?.recurring_patterns || 0})</span>
					</div>
					<div class="space-y-1">
						{#each patternSuggestions.slice(0, 3) as suggestion (suggestion.query)}
							<div class="flex items-center justify-between gap-2 rounded p-1.5 hover:bg-muted/50">
								<div class="flex items-center gap-2 min-w-0">
									<span class="text-[10px] text-muted-foreground">{suggestion.count}×</span>
									<span class="text-xs truncate">"{suggestion.query}"</span>
								</div>
								<Button
									variant="ghost"
									size="sm"
									class="h-5 px-2 text-[10px] shrink-0"
									onclick={() => copyCommand(suggestion.suggestion)}
								>
									→ {suggestion.suggestion.split(' ').slice(0, 2).join(' ')}
								</Button>
							</div>
						{/each}
						{#if patternSuggestions.length > 3}
							<div class="text-[10px] text-muted-foreground pl-1">
								+{patternSuggestions.length - 3} more patterns • <code class="bg-muted px-1 rounded">orch learn</code>
							</div>
						{/if}
					</div>
				</div>
			{/if}

			<!-- ⚡ Light-tier summary (collapsed/dismissible) -->
			{#if lightTierTotalUnreviewed > 0}
				<div class="rounded border bg-muted/30 p-2 border-muted-foreground/20">
					<div class="flex items-center justify-between">
						<div class="flex items-center gap-2">
							<Badge variant="secondary" class="text-[10px] bg-blue-500/20 text-blue-400 border-blue-500/30 h-4 px-1">
								⚡ light
							</Badge>
							<span class="text-[10px] text-muted-foreground">
								{lightTierTotalUnreviewed} stale recommendations from {lightTierAgents.length} agent{lightTierAgents.length === 1 ? '' : 's'}
							</span>
						</div>
						<Button
							variant="ghost"
							size="sm"
							class="h-5 px-2 text-[10px]"
							onclick={handleDismissAllLightTier}
							disabled={dismissingAllLightTier}
						>
							{dismissingAllLightTier ? '...' : 'Dismiss All'}
						</Button>
					</div>
				</div>
			{/if}
		</div>
	</div>
{/if}
