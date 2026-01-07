<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import type { Agent, Synthesis } from '$lib/stores/agents';
	import { createIssue } from '$lib/stores/agents';

	// Props
	interface Props {
		agent: Agent;
	}

	let { agent }: Props = $props();

	// Issue creation state
	let creatingIssue = $state(false);
	let issueCreationError = $state<string | null>(null);
	let createdIssueId = $state<string | null>(null);

	// Get outcome badge variant based on outcome type
	function getOutcomeVariant(outcome?: string): 'default' | 'secondary' | 'destructive' | 'outline' {
		switch (outcome?.toLowerCase()) {
			case 'success':
				return 'default';
			case 'partial':
				return 'secondary';
			case 'blocked':
			case 'failed':
				return 'destructive';
			default:
				return 'outline';
		}
	}

	// Get outcome emoji for visual emphasis
	function getOutcomeEmoji(outcome?: string): string {
		switch (outcome?.toLowerCase()) {
			case 'success':
				return '✅';
			case 'partial':
				return '⚠️';
			case 'blocked':
				return '🚫';
			case 'failed':
				return '❌';
			default:
				return '📋';
		}
	}

	// Get recommendation styling
	function getRecommendationStyle(recommendation?: string): string {
		switch (recommendation?.toLowerCase()) {
			case 'close':
				return 'text-green-600 dark:text-green-400';
			case 'spawn-follow-up':
			case 'continue':
				return 'text-blue-600 dark:text-blue-400';
			case 'escalate':
				return 'text-yellow-600 dark:text-yellow-400';
			case 'resume':
				return 'text-purple-600 dark:text-purple-400';
			default:
				return 'text-muted-foreground';
		}
	}

	// Create follow-up issue from next action
	async function handleCreateIssue(action: string) {
		creatingIssue = true;
		issueCreationError = null;
		createdIssueId = null;
		
		try {
			// Clean up action text (remove bullet prefixes)
			const cleanAction = action.replace(/^[-*]\s*/, '').replace(/^\d+\.\s*/, '');
			
			// Create issue with context about the parent agent
			const parentContext = agent.beads_id 
				? `\n\nFollow-up from: ${agent.beads_id}`
				: '';
			const description = `${cleanAction}${parentContext}`;
			
			const result = await createIssue(cleanAction.substring(0, 100), description, ['triage:ready']);
			if (result) {
				createdIssueId = result.id;
				// Auto-clear after 3 seconds
				setTimeout(() => {
					createdIssueId = null;
				}, 3000);
			}
		} catch (error) {
			issueCreationError = error instanceof Error ? error.message : 'Failed to create issue';
			// Auto-clear error after 5 seconds
			setTimeout(() => {
				issueCreationError = null;
			}, 5000);
		} finally {
			creatingIssue = false;
		}
	}
</script>

<div class="p-4 space-y-4">
	<!-- Header with outcome badge -->
	<div class="flex items-center justify-between">
		<h3 class="text-sm font-medium text-muted-foreground">
			{agent.synthesis ? 'Synthesis' : 'Completion Summary'}
		</h3>
		{#if agent.synthesis?.outcome}
			<Badge variant={getOutcomeVariant(agent.synthesis.outcome)} class="text-xs">
				{getOutcomeEmoji(agent.synthesis.outcome)} {agent.synthesis.outcome}
			</Badge>
		{/if}
	</div>

	<!-- TLDR or Close Reason fallback -->
	{#if agent.synthesis?.tldr || agent.close_reason}
		<div class="rounded-lg border bg-muted/30 p-3">
			<span class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
				{agent.synthesis?.tldr ? 'TLDR' : 'Close Reason'}
			</span>
			<p class="mt-1 text-sm leading-relaxed">
				{agent.synthesis?.tldr || agent.close_reason}
			</p>
		</div>
	{/if}

	<!-- D.E.K.N. Sections (only show if synthesis exists) -->
	{#if agent.synthesis}
		<!-- Delta Section - What Changed -->
		{#if agent.synthesis.delta_summary}
			<div class="space-y-1">
				<div class="flex items-center gap-2">
					<span class="text-lg">📝</span>
					<span class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
						Delta (What Changed)
					</span>
				</div>
				<p class="text-sm text-foreground pl-7">
					{agent.synthesis.delta_summary}
				</p>
			</div>
		{/if}

		<!-- Evidence Section - placeholder for future expansion -->
		<!-- Note: The current Synthesis interface doesn't include evidence data.
		     This section is here as a placeholder for when the backend
		     starts including evidence in the synthesis parsing. -->

		<!-- Knowledge Section - placeholder for future expansion -->
		<!-- Note: The current Synthesis interface doesn't include knowledge data.
		     This section is here as a placeholder for when the backend
		     starts including knowledge in the synthesis parsing. -->

		<!-- Next Section - Recommendation and Actions -->
		{#if agent.synthesis.recommendation || (agent.synthesis.next_actions && agent.synthesis.next_actions.length > 0)}
			<div class="space-y-2">
				<div class="flex items-center gap-2">
					<span class="text-lg">➡️</span>
					<span class="text-xs font-medium text-muted-foreground uppercase tracking-wide">
						Next (What Should Happen)
					</span>
				</div>
				
				<!-- Recommendation -->
				{#if agent.synthesis.recommendation}
					<div class="pl-7 flex items-center gap-2">
						<span class="text-xs text-muted-foreground">Recommendation:</span>
						<span class="text-sm font-medium {getRecommendationStyle(agent.synthesis.recommendation)}">
							{agent.synthesis.recommendation}
						</span>
					</div>
				{/if}
				
				<!-- Next Actions with Create Issue buttons -->
				{#if agent.synthesis.next_actions && agent.synthesis.next_actions.length > 0}
					<div class="pl-7">
						<div class="flex items-center justify-between mb-1">
							<span class="text-xs text-muted-foreground">Actions:</span>
							{#if issueCreationError}
								<span class="text-xs text-red-500">{issueCreationError}</span>
							{:else if createdIssueId}
								<span class="text-xs text-green-500">Created {createdIssueId}</span>
							{/if}
						</div>
						<ul class="space-y-1">
							{#each agent.synthesis.next_actions as action}
								<li class="flex items-start gap-2 rounded p-1.5 hover:bg-muted/50 group transition-colors">
									<span class="text-muted-foreground shrink-0">•</span>
									<span class="flex-1 text-sm">{action}</span>
									<button
										type="button"
										class="shrink-0 rounded border border-transparent px-2 py-0.5 text-[10px] text-muted-foreground opacity-0 transition-all hover:border-primary/50 hover:bg-primary/10 hover:text-foreground group-hover:opacity-100 disabled:opacity-50"
										onclick={() => handleCreateIssue(action)}
										disabled={creatingIssue}
									>
										{creatingIssue ? '...' : 'Create Issue'}
									</button>
								</li>
							{/each}
						</ul>
					</div>
				{/if}
			</div>
		{/if}
	{:else if !agent.close_reason}
		<!-- No synthesis and no close_reason - show placeholder -->
		<div class="rounded-lg border border-dashed bg-muted/20 p-6 text-center">
			<span class="text-2xl mb-2 block">📄</span>
			<p class="text-sm text-muted-foreground">
				No synthesis data available for this agent.
			</p>
			<p class="text-xs text-muted-foreground/70 mt-1">
				SYNTHESIS.md may not have been created or parsed yet.
			</p>
		</div>
	{/if}
</div>
