<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import type { TreeNode } from '$lib/stores/work-graph';
	import type { Agent } from '$lib/stores/agents';
	import {
		type RunningAgentDetails,
		formatRelatedIssueList,
		formatStatusLabel,
		getAge,
		getAgentBadge,
		getAttentionBadge,
		getDependencyExplanation,
		getInProgressSubline,
		getIssueSummary,
		getPriorityVariant,
		getProgressSnapshot,
		getRelatedIssueLabel,
		getStatusColor,
		getStatusIcon,
		getTypeBadge,
	} from './work-graph-tree-helpers';

	export let node: TreeNode;
	export let index: number;
	export let selectedIndex: number;
	export let copiedId: string | null;
	export let expandedDetails: Set<string>;
	export let depPrefix: string | undefined;
	export let treeNodeIndex: Map<string, TreeNode>;
	export let runningAgentDetailsByIssueId: Map<string, RunningAgentDetails>;
	export let actionFeedback: Map<string, 'priority' | 'queue'>;
	export let agentsByBeadsId: Map<string, Agent>;
	export let onSelectNode: (index: number) => void;
	export let onCopyToClipboard: (id: string) => void;
	export let onOpenSidePanel: (node: TreeNode) => void;
	export let onSetFocus: (beadsId: string, title: string) => void;

	$: feedback = actionFeedback.get(node.id);
	$: inProgressSubline = getInProgressSubline(node, runningAgentDetailsByIssueId);
	$: progressSnapshot = getProgressSnapshot(node);
	$: nodeAgent = agentsByBeadsId.get(node.id);
	$: agentBadge = nodeAgent ? getAgentBadge(nodeAgent) : null;
</script>

<!-- Tree Node - L0: Row -->
<div
	class="flex items-center gap-3 py-2 px-1 rounded transition-colors {index === selectedIndex ? 'bg-zinc-800' : ''} {node.status.toLowerCase() === 'in_progress' ? 'border-l-2 border-blue-500/60 bg-blue-500/5' : 'border-l border-transparent'} {node.absorbed_by ? 'opacity-50' : ''} {feedback === 'priority' ? 'action-feedback-priority' : ''} {feedback === 'queue' ? 'action-feedback-queue' : ''}"
	style="padding-left: {depPrefix !== undefined ? '8px' : node.depth * 24 + 'px'}"
	role="button"
	tabindex="-1"
	onclick={() => {
		onSelectNode(index);
		onOpenSidePanel(node);
	}}
	onkeydown={(event) => {
		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			onSelectNode(index);
			onOpenSidePanel(node);
		}
	}}
>
		{#if depPrefix !== undefined}
			{#if depPrefix === ''}
				<!-- Flow origin: unblocked root of dependency chain -->
				<span class="text-blue-400 font-mono text-sm select-none w-4 text-center" title="Flow origin (unblocked)">◆</span>
			{:else}
				<!-- Flow connector: directional arrow shows dependency flow -->
				<span class="text-zinc-600 font-mono text-xs whitespace-pre select-none leading-none">{depPrefix}</span>
			{/if}
		{:else}
			<!-- Expansion indicator (non-dep mode) -->
			<span class="w-4 text-muted-foreground text-xs">
				{#if node.children.length > 0}
					{node.expanded ? '▼' : '▶'}
				{:else}
					<span class="opacity-0">•</span>
				{/if}
			</span>
		{/if}

		<!-- Status icon -->
		<span data-testid="status-icon" class="w-5 {getStatusColor(node.status)}">
			{getStatusIcon(node.status)}
		</span>

		<!-- Priority badge -->
		<Badge data-testid="priority-badge" variant={getPriorityVariant(node.priority)} class="w-8 justify-center text-xs">
			P{node.priority}
		</Badge>
		{#if node.effective_priority !== undefined && node.effective_priority !== node.priority}
			<Badge variant="outline" class="text-xs">
				Eff P{node.effective_priority}
			</Badge>
		{/if}

		<!-- ID -->
		<span
			class="text-xs font-mono min-w-[120px] cursor-pointer hover:text-foreground transition-colors {copiedId === node.id ? 'text-green-500' : 'text-muted-foreground'}"
			onclick={(e) => { e.stopPropagation(); onCopyToClipboard(node.id); }}
			onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); e.preventDefault(); onCopyToClipboard(node.id); }}}
			role="button"
			tabindex="-1"
			title="Click to copy"
		>
			{copiedId === node.id ? 'Copied!' : node.id}
		</span>

		<!-- Title + in_progress details -->
		<div class="flex-1 min-w-0">
			<span class="block text-sm font-medium text-foreground truncate">
				{node.title}
			</span>
			{#if inProgressSubline}
				<span class="block text-[11px] leading-4 truncate {inProgressSubline.tone}">
					{inProgressSubline.text}
				</span>
			{/if}
		</div>

		{#if progressSnapshot}
			<div class="hidden xl:flex items-center gap-2 min-w-[120px]">
				<div class="w-14 h-1.5 bg-muted rounded-full overflow-hidden">
					<div class="h-full bg-blue-500 transition-all" style="width: {progressSnapshot.percent}%"></div>
				</div>
				<span class="text-[11px] text-muted-foreground tabular-nums">
					{progressSnapshot.done}/{progressSnapshot.total}
				</span>
			</div>
		{/if}

		<!-- Agent badge (if any) -->
		{#if agentBadge}
			<span
				class="inline-flex items-center gap-1 px-1.5 py-0.5 text-[10px] font-medium rounded border shrink-0 {agentBadge.color}"
				title="Agent: {nodeAgent?.skill || nodeAgent?.status || 'unknown'}"
			>
				<span class={nodeAgent?.is_processing ? 'animate-pulse' : ''}>{agentBadge.icon}</span>
				{agentBadge.label}
			</span>
		{/if}

		<!-- Attention badge (if any) -->
		{#if node.attentionBadge}
			{@const badgeConfig = getAttentionBadge(node.attentionBadge)}
			{#if badgeConfig}
				<Badge variant={badgeConfig.variant} class="shrink-0">
					{badgeConfig.label}
				</Badge>
			{/if}
		{/if}

		<!-- Type badge -->
		<Badge data-testid="type-badge" variant="outline" class="{getTypeBadge(node.type)} text-xs shrink-0">
			{node.type}
		</Badge>

			<!-- Set as Focus button for epics -->
			{#if node.type === 'epic'}
			<button
				type="button"
				class="text-xs text-blue-500 hover:text-blue-600 hover:underline px-1"
				onclick={(event) => {
					event.stopPropagation();
					onSetFocus(node.id, node.title);
				}}
				title="Set this epic as your current focus"
			>
					Set Focus
				</button>
			{/if}

		<!-- Age (placeholder) -->
		{#if getAge(node.id)}
			<span class="text-xs text-muted-foreground min-w-[40px] text-right">
				{getAge(node.id)}
			</span>
		{/if}
	</div>

	<!-- L1: Expanded details -->
	{#if expandedDetails.has(node.id)}
		{@const dependencyExplanation = getDependencyExplanation(node, treeNodeIndex)}
		{@const progressInDetails = getProgressSnapshot(node)}
		{@const parentLabel = node.parent_id ? getRelatedIssueLabel(node.parent_id, treeNodeIndex) : null}
		{@const directChildLabels = node.children.map((child) => getRelatedIssueLabel(child.id, treeNodeIndex))}
		<div
			data-testid={`issue-details-${node.id}`}
			class="expanded-details ml-12 mt-1 mb-2 p-3 bg-muted/30 rounded text-sm"
			style="margin-left: {depPrefix !== undefined ? '48px' : node.depth * 24 + 48 + 'px'}"
		>
			<div class="mb-3">
				<span class="text-xs font-semibold uppercase text-foreground">Issue summary:</span>
				<p class="mt-1 text-xs text-muted-foreground">{getIssueSummary(node)}</p>
			</div>

			<div class="mb-3">
				<span class="text-xs font-semibold uppercase text-foreground">Dependency context:</span>
				<p class="mt-1 text-xs {dependencyExplanation.tone}">{dependencyExplanation.headline}</p>
				<p class="mt-1 text-[11px] text-muted-foreground">{dependencyExplanation.detail}</p>
			</div>

			{#if progressInDetails}
				<div class="mb-3">
					<div class="flex items-center justify-between gap-2">
						<span class="text-xs font-semibold uppercase text-foreground">Progress & completeness:</span>
						<span class="text-xs text-muted-foreground tabular-nums">{progressInDetails.done}/{progressInDetails.total} done ({progressInDetails.percent}%)</span>
					</div>
					<div class="mt-1 h-1.5 bg-muted rounded-full overflow-hidden">
						<div class="h-full bg-blue-500 transition-all" style="width: {progressInDetails.percent}%"></div>
					</div>
					<p class="mt-1 text-[11px] text-muted-foreground">
						{#if progressInDetails.visible === progressInDetails.total}
							All {progressInDetails.total} related issues are visible in this branch.
						{:else}
							{progressInDetails.visible} of {progressInDetails.total} related issues are visible (expand children to inspect all).
						{/if}
					</p>
				</div>
			{/if}

			<div class="mb-2">
				<span class="text-xs font-semibold uppercase text-foreground">Related issues:</span>
				<div class="mt-1 space-y-1 text-xs text-muted-foreground">
					{#if parentLabel}
						<div>Parent: {parentLabel}</div>
					{/if}
					{#if directChildLabels.length > 0}
						<div>Children ({directChildLabels.length}): {formatRelatedIssueList(node.children.map((child) => child.id), treeNodeIndex)}</div>
					{/if}
					{#if node.blocked_by.length > 0}
						<div>Upstream blockers: {formatRelatedIssueList(node.blocked_by, treeNodeIndex)}</div>
					{/if}
					{#if node.blocks.length > 0}
						<div>Downstream dependents: {formatRelatedIssueList(node.blocks, treeNodeIndex)}</div>
					{/if}
					{#if node.absorbed_by}
						<div>Absorbed by: {getRelatedIssueLabel(node.absorbed_by, treeNodeIndex)}</div>
					{/if}
					{#if !parentLabel && directChildLabels.length === 0 && node.blocked_by.length === 0 && node.blocks.length === 0 && !node.absorbed_by}
						<div>No directly related issues in the current scope.</div>
					{/if}
				</div>
			</div>

			<!-- Status details -->
			<div class="flex flex-wrap items-center gap-4 text-xs border-t border-border pt-2 mt-2">
				<span class="flex items-center gap-1">
					<span class="text-foreground/60">Status:</span>
					<span class={getStatusColor(node.status)}>{formatStatusLabel(node.status)}</span>
				</span>
				<span class="flex items-center gap-1">
					<span class="text-foreground/60">Priority:</span>
					<span class="text-muted-foreground">P{node.priority}</span>
				</span>
				<span class="flex items-center gap-1">
					<span class="text-foreground/60">Type:</span>
					<span class="text-muted-foreground">{node.type}</span>
				</span>
				<span class="flex items-center gap-1">
					<span class="text-foreground/60">Source:</span>
					<span class="text-muted-foreground">{node.source}</span>
				</span>
			</div>
		</div>
	{/if}
<style>
	.action-feedback-priority {
		animation: priority-flash 1s ease-out;
	}

	.action-feedback-queue {
		animation: queue-flash 1s ease-out;
	}

	@keyframes priority-flash {
		0% {
			background-color: rgba(168, 85, 247, 0.4);
			box-shadow: 0 0 0 2px rgba(168, 85, 247, 0.6);
		}
		100% {
			background-color: transparent;
			box-shadow: 0 0 0 0 transparent;
		}
	}

	@keyframes queue-flash {
		0% {
			background-color: rgba(34, 197, 94, 0.4);
			box-shadow: 0 0 0 2px rgba(34, 197, 94, 0.6);
		}
		100% {
			background-color: transparent;
			box-shadow: 0 0 0 0 transparent;
		}
	}
</style>
