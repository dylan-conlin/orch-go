<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { errorEvents } from '$lib/stores/agentlog';
	import { pendingReviews, type PendingReviewAgent, type PendingReviewItem } from '$lib/stores/pending-reviews';
	import { beads } from '$lib/stores/beads';
	import { createIssue } from '$lib/stores/agents';

	// State for issue creation (same as pending-reviews)
	let creatingIssue: { [key: string]: boolean } = {};
	let createdIssues: { [key: string]: string } = {};
	let dismissingItem: { [key: string]: boolean } = {};
	let dismissingAllLightTier: boolean = false;

	// Separate light-tier from standard agents
	$: lightTierAgents = $pendingReviews?.agents.filter(a => a.is_light_tier) ?? [];
	$: standardAgents = $pendingReviews?.agents.filter(a => !a.is_light_tier) ?? [];
	
	// Count total light-tier unreviewed items
	$: lightTierTotalUnreviewed = lightTierAgents.reduce((sum, agent) => 
		sum + getUnreviewedItems(agent).length, 0);

	// Calculate total attention items
	$: totalErrors = $errorEvents.length;
	$: totalReviews = $pendingReviews?.total_unreviewed ?? 0;
	$: totalBlocked = $beads?.blocked_issues ?? 0;
	$: totalAttentionItems = totalErrors + totalReviews + (totalBlocked > 0 ? 1 : 0);

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
				pendingReviews.markActedOn(agent.workspace_id, item.index);
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

	function formatUnixTime(timestamp: number): string {
		return new Date(timestamp * 1000).toLocaleTimeString();
	}
</script>

{#if totalAttentionItems > 0}
	<div class="rounded-lg border border-amber-500/30 bg-amber-500/5" data-testid="needs-attention-section">
		<div class="flex items-center gap-2 px-3 py-2 border-b">
			<span class="text-sm">⚠️</span>
			<span class="text-sm font-medium">Needs Attention</span>
			<Badge variant="secondary" class="h-5 px-1.5 text-xs bg-amber-500/20 text-amber-600">
				{totalAttentionItems}
			</Badge>
		</div>
		<div class="p-2 space-y-2">
			<!-- Errors Section -->
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

			<!-- Blocked Issues Section -->
			{#if totalBlocked > 0}
				<div class="rounded border bg-card p-2 border-orange-500/30">
					<div class="flex items-center gap-2">
						<span class="text-sm">🚧</span>
						<span class="text-xs font-medium text-orange-500">Blocked Issues</span>
						<Badge variant="outline" class="h-4 px-1 text-[10px]">{totalBlocked}</Badge>
						<span class="text-[10px] text-muted-foreground">Run <code class="bg-muted px-1 rounded">bd blocked</code> to see details</span>
					</div>
				</div>
			{/if}

			<!-- Pending Reviews Section (condensed) -->
			{#if totalReviews > 0}
				<div class="rounded border bg-card p-2">
					<div class="flex items-center gap-2 mb-2">
						<span class="text-sm">📋</span>
						<span class="text-xs font-medium">Pending Reviews</span>
						<Badge variant="outline" class="h-4 px-1 text-[10px]">{totalReviews}</Badge>
					</div>

					<!-- Light-tier agents: grouped into single summary -->
					{#if lightTierTotalUnreviewed > 0}
						<div class="rounded border bg-blue-500/5 border-blue-500/30 p-2 mb-2">
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-2">
									<Badge variant="secondary" class="text-[10px] bg-blue-500/20 text-blue-400 border-blue-500/30 h-4 px-1">
										⚡ light
									</Badge>
									<span class="text-[10px] text-muted-foreground">
										{lightTierTotalUnreviewed} from {lightTierAgents.length} agent{lightTierAgents.length === 1 ? '' : 's'}
									</span>
								</div>
								<Button
									variant="outline"
									size="sm"
									class="h-5 px-1.5 text-[10px]"
									onclick={handleDismissAllLightTier}
									disabled={dismissingAllLightTier}
								>
									{dismissingAllLightTier ? '...' : 'Dismiss All'}
								</Button>
							</div>
						</div>
					{/if}

					<!-- Standard agents: show full detail -->
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
														{creatingIssue[key] ? '...' : 'Issue'}
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
		</div>
	</div>
{/if}
