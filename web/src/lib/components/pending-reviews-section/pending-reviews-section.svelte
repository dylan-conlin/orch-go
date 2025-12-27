<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { pendingReviews, type PendingReviewAgent, type PendingReviewItem } from '$lib/stores/pending-reviews';
	import { createIssue } from '$lib/stores/agents';

	export let expanded: boolean = true;

	// State for issue creation
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

	function toggle() {
		expanded = !expanded;
	}

	function getItemKey(workspaceId: string, index: number): string {
		return `${workspaceId}-${index}`;
	}

	async function handleCreateIssue(agent: PendingReviewAgent, item: PendingReviewItem) {
		const key = getItemKey(agent.workspace_id, item.index);
		creatingIssue[key] = true;
		creatingIssue = creatingIssue;
		
		try {
			// Clean up action text (remove bullet prefixes)
			const cleanAction = item.text.replace(/^[-*]\s*/, '').replace(/^\d+\.\s*/, '');
			
			// Create issue with context about the parent agent
			const parentContext = agent.beads_id 
				? `\n\nFollow-up from: ${agent.beads_id}`
				: '';
			const description = `${cleanAction}${parentContext}`;
			
			const result = await createIssue(cleanAction.substring(0, 100), description, ['triage:ready']);
			if (result) {
				createdIssues[key] = result.id;
				createdIssues = createdIssues;
				// Mark as acted on in the store
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
			// Dismiss all unreviewed items from light-tier agents
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

	// Get unreviewed items for an agent
	function getUnreviewedItems(agent: PendingReviewAgent): PendingReviewItem[] {
		return agent.items.filter(item => !item.reviewed && !item.dismissed && !item.acted_on);
	}

	// Format workspace name for display
	function formatWorkspaceName(workspaceId: string): string {
		return workspaceId
			.replace(/^[a-z]+-/, '') // Remove project prefix
			.replace(/-\d{1,2}[a-z]{3}$/, '') // Remove date suffix
			.replace(/-/g, ' ')
			.trim();
	}
</script>

{#if $pendingReviews && $pendingReviews.total_unreviewed > 0}
	<div class="rounded-lg border border-amber-500/30 bg-amber-500/5">
		<button
			class="flex w-full items-center justify-between px-3 py-2 text-left hover:bg-accent/50 transition-colors"
			onclick={toggle}
			aria-expanded={expanded}
			data-testid="pending-reviews-toggle"
		>
			<div class="flex items-center gap-2 min-w-0 flex-1">
				<span class="text-sm flex-shrink-0">📋</span>
				<span class="text-sm font-medium flex-shrink-0">Pending Reviews</span>
				<Badge variant="secondary" class="h-5 px-1.5 text-xs flex-shrink-0">
					{$pendingReviews.total_unreviewed}
				</Badge>
				{#if !expanded}
					<span class="text-xs text-muted-foreground truncate">
						— {$pendingReviews.total_agents} agent{$pendingReviews.total_agents === 1 ? '' : 's'} with recommendations
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
			<div class="border-t px-3 py-2 space-y-3" data-testid="pending-reviews-content">
				<!-- Light-tier agents: grouped into single summary card -->
				{#if lightTierTotalUnreviewed > 0}
					<div class="rounded border bg-card p-3 border-blue-500/30 bg-blue-500/5">
						<div class="flex items-center justify-between mb-2">
							<div class="flex items-center gap-2">
								<Badge variant="secondary" class="text-xs bg-blue-500/20 text-blue-400 border-blue-500/30">
									⚡ light tier
								</Badge>
								<span class="text-sm text-muted-foreground">
									{lightTierTotalUnreviewed} recommendation{lightTierTotalUnreviewed === 1 ? '' : 's'} from {lightTierAgents.length} agent{lightTierAgents.length === 1 ? '' : 's'}
								</span>
							</div>
							<Button
								variant="outline"
								size="sm"
								class="h-6 px-2 text-xs"
								onclick={handleDismissAllLightTier}
								disabled={dismissingAllLightTier}
							>
								{dismissingAllLightTier ? 'Dismissing...' : 'Dismiss All'}
							</Button>
						</div>
						<p class="text-xs text-muted-foreground">
							Light tier spawns have minimal synthesis - these can usually be dismissed in bulk.
						</p>
					</div>
				{/if}

				<!-- Standard agents: show full detail -->
				{#each standardAgents as agent (agent.workspace_id)}
					{@const unreviewedItems = getUnreviewedItems(agent)}
					{#if unreviewedItems.length > 0}
						<div class="rounded border bg-card p-3">
							<div class="flex items-center gap-2 mb-2">
								<span class="text-sm font-medium">{formatWorkspaceName(agent.workspace_id)}</span>
								<Badge variant="outline" class="text-xs">
									{unreviewedItems.length} recommendation{unreviewedItems.length === 1 ? '' : 's'}
								</Badge>
								{#if agent.beads_id}
									<span class="text-xs text-muted-foreground">{agent.beads_id}</span>
								{/if}
							</div>

							{#if agent.tldr}
								<p class="text-xs text-muted-foreground mb-2 line-clamp-2">{agent.tldr}</p>
							{/if}

							<div class="space-y-1">
								{#each unreviewedItems as item (item.index)}
									{@const key = getItemKey(agent.workspace_id, item.index)}
									<div class="flex items-start gap-2 rounded p-1.5 hover:bg-muted/50 group text-sm">
										<span class="flex-1 text-xs">{item.text}</span>
										<div class="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
											{#if createdIssues[key]}
												<span class="text-xs text-green-500 px-2 py-0.5">{createdIssues[key]}</span>
											{:else}
												<Button
													variant="outline"
													size="sm"
													class="h-6 px-2 text-xs"
													onclick={() => handleCreateIssue(agent, item)}
													disabled={creatingIssue[key]}
												>
													{creatingIssue[key] ? '...' : 'Create Issue'}
												</Button>
												<Button
													variant="ghost"
													size="sm"
													class="h-6 px-2 text-xs text-muted-foreground hover:text-foreground"
													onclick={() => handleDismiss(agent, item)}
													disabled={dismissingItem[key]}
												>
													{dismissingItem[key] ? '...' : 'Dismiss'}
												</Button>
											{/if}
										</div>
									</div>
								{/each}
							</div>
						</div>
					{/if}
				{/each}
			</div>
		{/if}
	</div>
{/if}
