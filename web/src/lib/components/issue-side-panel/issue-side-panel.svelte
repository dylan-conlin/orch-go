<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { TreeNode } from '$lib/stores/work-graph';
	import { AttemptHistory } from '$lib/components/attempt-history';
	import { ActivityTab } from '$lib/components/agent-detail';
	import { CompletionDetails } from '$lib/components/completion-details';
	import { DeliverableChecklist } from '$lib/components/deliverable-checklist';
	import { MarkdownContent } from '$lib/components/markdown-content';
	import type { Agent } from '$lib/stores/agents';
	import { agents } from '$lib/stores/agents';
	import { Badge } from '$lib/components/ui/badge';

	export let issue: TreeNode;

	const dispatch = createEventDispatcher();

	// Tab state
	type TabId = 'description' | 'activity' | 'deliverables';
	let activeTab: TabId = 'description';
	let lastIssueId = '';
	let lastLifecycleTab: TabId = 'description';

	// Determine if issue is completed (show Completion tab)
	$: isCompleted = ['closed', 'complete', 'completed'].includes(issue.status.toLowerCase());

	function getLifecycleTab(status: string): TabId {
		const normalized = status.toLowerCase();
		if (normalized === 'in_progress') return 'activity';
		if (normalized === 'closed' || normalized === 'complete' || normalized === 'completed') return 'deliverables';
		return 'description';
	}

	function getActivityAgent(agentList: Agent[], beadsId: string): Agent | null {
		const relatedAgents = agentList
			.filter((agent) => agent.beads_id === beadsId)
			.sort((a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime());

		if (relatedAgents.length === 0) {
			return null;
		}

		const liveAgent = relatedAgents.find((agent) => agent.status === 'active' || agent.status === 'idle');
		return liveAgent || relatedAgents[0];
	}

	$: lifecycleTab = getLifecycleTab(issue.status);
	$: activityAgent = getActivityAgent($agents, issue.id);

	// Keep tab lifecycle-aware while still allowing manual tab switching.
	// We only auto-switch when issue identity changes or status transitions lifecycle stage.
	$: {
		if (issue.id !== lastIssueId) {
			lastIssueId = issue.id;
			activeTab = lifecycleTab;
			lastLifecycleTab = lifecycleTab;
		} else if (lifecycleTab !== lastLifecycleTab) {
			activeTab = lifecycleTab;
			lastLifecycleTab = lifecycleTab;
		}
	}

	// Get status badge variant
	function getStatusBadge(status: string): { variant: 'default' | 'secondary' | 'destructive' | 'outline'; text: string } {
		const normalized = status.toLowerCase();
		switch (normalized) {
			case 'in_progress':
				return { variant: 'default', text: 'In Progress' };
			case 'blocked':
				return { variant: 'destructive', text: 'Blocked' };
			case 'open':
				return { variant: 'outline', text: 'Open' };
			case 'closed':
			case 'complete':
				return { variant: 'secondary', text: 'Complete' };
			default:
				return { variant: 'outline', text: status };
		}
	}

	// Get type badge variant
	function getTypeBadge(type: string): { variant: 'default' | 'secondary' | 'destructive' | 'outline'; text: string } {
		const normalized = type.toLowerCase();
		switch (normalized) {
			case 'bug':
				return { variant: 'destructive', text: 'Bug' };
			case 'feature':
				return { variant: 'default', text: 'Feature' };
			case 'task':
				return { variant: 'secondary', text: 'Task' };
			case 'epic':
				return { variant: 'outline', text: 'Epic' };
			case 'question':
				return { variant: 'outline', text: 'Question' };
			case 'investigation':
				return { variant: 'secondary', text: 'Investigation' };
			case 'decision':
				return { variant: 'default', text: 'Decision' };
			default:
				return { variant: 'outline', text: type };
		}
	}

	// Format priority
	function getPriorityLabel(priority: number): string {
		switch (priority) {
			case 0: return 'P0 - Critical';
			case 1: return 'P1 - High';
			case 2: return 'P2 - Medium';
			case 3: return 'P3 - Low';
			case 4: return 'P4 - Backlog';
			default: return `P${priority}`;
		}
	}

	function handleClose() {
		dispatch('close');
	}

	// Tab order for cycling
	const tabOrder: TabId[] = ['description', 'activity', 'deliverables'];

	// Handle keyboard shortcuts
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' || event.key === 'h') {
			event.preventDefault();
			handleClose();
		}
		if (event.key === 'Tab') {
			event.preventDefault();
			const idx = tabOrder.indexOf(activeTab);
			if (event.shiftKey) {
				activeTab = tabOrder[(idx - 1 + tabOrder.length) % tabOrder.length];
			} else {
				activeTab = tabOrder[(idx + 1) % tabOrder.length];
			}
		}
	}
</script>

<svelte:window on:keydown={handleKeydown} />

<div
	class="fixed top-0 right-0 h-screen w-1/2 bg-background border-l border-border shadow-lg z-50 flex flex-col"
	role="dialog"
	aria-modal="true"
	aria-labelledby="issue-title"
>
	<!-- Header -->
	<div class="border-b border-border px-6 py-4 flex items-center justify-between">
		<div class="flex-1 min-w-0">
			<div class="flex items-center gap-2 mb-2">
				<span class="text-sm font-mono text-muted-foreground">{issue.id}</span>
				<Badge variant={getStatusBadge(issue.status).variant}>
					{getStatusBadge(issue.status).text}
				</Badge>
				<Badge variant={getTypeBadge(issue.type).variant}>
					{getTypeBadge(issue.type).text}
				</Badge>
				{#if issue.priority !== undefined}
					<span class="text-xs text-muted-foreground">
						{getPriorityLabel(issue.priority)}
					</span>
				{/if}
			</div>
			<h2 id="issue-title" class="text-lg font-semibold text-foreground truncate">
				{issue.title}
			</h2>
		</div>
		<button
			on:click={handleClose}
			on:mousedown|preventDefault
			class="text-muted-foreground hover:text-foreground transition-colors ml-4"
			aria-label="Close panel"
			tabindex="-1"
		>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				width="20"
				height="20"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<line x1="18" y1="6" x2="6" y2="18" />
				<line x1="6" y1="6" x2="18" y2="18" />
			</svg>
		</button>
	</div>

	<!-- Lifecycle Tabs -->
	<div class="border-b border-border px-6 flex gap-0" role="tablist" aria-label="Issue lifecycle tabs">
		<button
			class="px-4 py-2.5 text-sm font-medium transition-colors relative
				{activeTab === 'description'
					? 'text-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
			on:click={() => activeTab = 'description'}
			on:mousedown|preventDefault
			role="tab"
			aria-selected={activeTab === 'description'}
			tabindex="-1"
		>
			Description
			{#if activeTab === 'description'}
				<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-foreground"></div>
			{/if}
		</button>
		<button
			class="px-4 py-2.5 text-sm font-medium transition-colors relative
				{activeTab === 'activity'
					? 'text-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
			on:click={() => activeTab = 'activity'}
			on:mousedown|preventDefault
			role="tab"
			aria-selected={activeTab === 'activity'}
			tabindex="-1"
		>
			Activity
			{#if activeTab === 'activity'}
				<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-foreground"></div>
			{/if}
		</button>
		<button
			class="px-4 py-2.5 text-sm font-medium transition-colors relative
				{activeTab === 'deliverables'
					? 'text-foreground'
					: 'text-muted-foreground hover:text-foreground'}"
			on:click={() => activeTab = 'deliverables'}
			on:mousedown|preventDefault
			role="tab"
			aria-selected={activeTab === 'deliverables'}
			tabindex="-1"
		>
			Deliverables
			{#if activeTab === 'deliverables'}
				<div class="absolute bottom-0 left-0 right-0 h-0.5 bg-foreground"></div>
			{/if}
		</button>
	</div>

	<!-- Content -->
	<div class="flex-1 min-h-0 {activeTab === 'activity' ? 'flex flex-col overflow-hidden' : 'overflow-auto px-6 py-4 space-y-6'}">
		{#if activeTab === 'description'}
			<!-- Description -->
			{#if issue.description}
				<section>
					<h3 class="text-sm font-semibold text-foreground mb-2">Description</h3>
					<div class="prose prose-sm dark:prose-invert max-w-none">
						<MarkdownContent content={issue.description} />
					</div>
				</section>
			{:else}
				<section>
					<div class="text-sm text-muted-foreground italic">
						No description available
					</div>
				</section>
			{/if}

			<!-- Metadata -->
			<section>
				<h3 class="text-sm font-semibold text-foreground mb-2">Metadata</h3>
				<dl class="grid grid-cols-2 gap-2 text-sm">
					<div>
						<dt class="text-muted-foreground">ID</dt>
						<dd class="font-mono text-foreground">{issue.id}</dd>
					</div>
					<div>
						<dt class="text-muted-foreground">Source</dt>
						<dd class="text-foreground">{issue.source}</dd>
					</div>
					{#if issue.created_at}
						<div>
							<dt class="text-muted-foreground">Created</dt>
							<dd class="text-foreground">{new Date(issue.created_at).toLocaleDateString()}</dd>
						</div>
					{/if}
					{#if issue.date}
						<div>
							<dt class="text-muted-foreground">Date</dt>
							<dd class="text-foreground">{issue.date}</dd>
						</div>
					{/if}
				</dl>
			</section>

			<!-- Dependencies -->
			{#if issue.blocked_by?.length > 0 || issue.blocks?.length > 0}
				<section>
					<h3 class="text-sm font-semibold text-foreground mb-2">Dependencies</h3>
					{#if issue.blocked_by?.length > 0}
						<div class="mb-2">
							<div class="text-xs text-muted-foreground mb-1">Blocked by:</div>
							<div class="flex flex-wrap gap-1">
								{#each issue.blocked_by as blockerId}
									<Badge variant="destructive">{blockerId}</Badge>
								{/each}
							</div>
						</div>
					{/if}
					{#if issue.blocks?.length > 0}
						<div>
							<div class="text-xs text-muted-foreground mb-1">Blocks:</div>
							<div class="flex flex-wrap gap-1">
								{#each issue.blocks as blockedId}
									<Badge variant="outline">{blockedId}</Badge>
								{/each}
							</div>
						</div>
					{/if}
				</section>
			{/if}

			<!-- Attention Signal -->
			{#if issue.attentionBadge}
				<section>
					<h3 class="text-sm font-semibold text-foreground mb-2">Attention</h3>
					<div class="flex items-start gap-2 p-3 bg-muted rounded-md">
						<Badge variant="outline">{issue.attentionBadge}</Badge>
						{#if issue.attentionReason}
							<div class="text-sm text-muted-foreground">
								{issue.attentionReason}
							</div>
						{/if}
					</div>
				</section>
			{/if}
		{:else if activeTab === 'activity'}
			{#if activityAgent}
				<ActivityTab agent={activityAgent} showComposer={false} />
			{:else}
				<section class="rounded-md border border-dashed border-border p-4 bg-muted/20 mx-6 my-4">
					<h3 class="text-sm font-semibold text-foreground mb-1">No activity stream available</h3>
					<p class="text-sm text-muted-foreground">
						No agent session is currently linked to <span class="font-mono">{issue.id}</span>.
						{#if issue.status.toLowerCase() === 'in_progress'}
							Waiting for live messages from the active worker.
						{/if}
					</p>
				</section>
			{/if}
		{:else if activeTab === 'deliverables'}
			{#if issue.source === 'beads'}
				<section>
					<h3 class="text-sm font-semibold text-foreground mb-2">Deliverable Checklist</h3>
					<DeliverableChecklist beadsId={issue.id} issueType={issue.type} compact={false} />
				</section>

				<section>
					<h3 class="text-sm font-semibold text-foreground mb-2">Artifacts, Commits, and Synthesis</h3>
					{#if isCompleted}
						<CompletionDetails beadsId={issue.id} />
					{:else}
						<div class="text-sm text-muted-foreground italic">
							Completion artifacts appear after this issue reaches Complete.
						</div>
					{/if}
				</section>

				<section>
					<h3 class="text-sm font-semibold text-foreground mb-2">Attempt History</h3>
					<AttemptHistory beadsId={issue.id} />
				</section>
			{:else}
				<section class="rounded-md border border-dashed border-border p-4 bg-muted/20">
					<h3 class="text-sm font-semibold text-foreground mb-1">Deliverables unavailable</h3>
					<p class="text-sm text-muted-foreground">
						Deliverables are only tracked for Beads issues.
					</p>
				</section>
			{/if}
		{/if}
	</div>

	<!-- Footer -->
	<div class="border-t border-border px-6 py-3 text-xs text-muted-foreground">
		Press <kbd class="px-1 py-0.5 bg-muted rounded">h</kbd> or
		<kbd class="px-1 py-0.5 bg-muted rounded">Esc</kbd> to close
	</div>
</div>
