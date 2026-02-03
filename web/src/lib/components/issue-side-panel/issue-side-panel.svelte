<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import type { TreeNode } from '$lib/stores/work-graph';
	import { AttemptHistory } from '$lib/components/attempt-history';
	import { DeliverableChecklist } from '$lib/components/deliverable-checklist';
	import { MarkdownContent } from '$lib/components/markdown-content';
	import { Badge } from '$lib/components/ui/badge';

	export let issue: TreeNode;

	const dispatch = createEventDispatcher();

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

	// Handle keyboard shortcuts
	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' || event.key === 'h') {
			event.preventDefault();
			handleClose();
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
			class="text-muted-foreground hover:text-foreground transition-colors ml-4"
			aria-label="Close panel"
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

	<!-- Content -->
	<div class="flex-1 overflow-auto px-6 py-4 space-y-6">
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

		<!-- Deliverables -->
		{#if issue.source === 'beads'}
			<section>
				<h3 class="text-sm font-semibold text-foreground mb-2">Deliverables</h3>
				<DeliverableChecklist beadsId={issue.id} issueType={issue.type} compact={false} />
			</section>
		{/if}

		<!-- Attempt History -->
		{#if issue.source === 'beads'}
			<section>
				<h3 class="text-sm font-semibold text-foreground mb-2">Attempt History</h3>
				<AttemptHistory beadsId={issue.id} />
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
	</div>

	<!-- Footer -->
	<div class="border-t border-border px-6 py-3 text-xs text-muted-foreground">
		Press <kbd class="px-1 py-0.5 bg-muted rounded">h</kbd> or
		<kbd class="px-1 py-0.5 bg-muted rounded">Esc</kbd> to close
	</div>
</div>
