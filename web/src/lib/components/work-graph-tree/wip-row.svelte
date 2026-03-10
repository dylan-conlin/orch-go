<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import type { TreeNode } from '$lib/stores/work-graph';
	import type { WIPItem } from '$lib/stores/wip';
	import { computeAgentHealth, getContextPercent, getContextColor, getExpressiveStatus } from '$lib/stores/wip';
	import { DeliverableChecklist } from '$lib/components/deliverable-checklist';
	import { getExpectedDeliverables } from '$lib/stores/deliverables';
	import {
		type RunningAgentDetails,
		getAgentStatusIcon,
		getAttentionBadge,
		getInProgressSubline,
		getPanelIssue,
		getPriorityVariant,
		getTypeBadge,
		shortenModel,
	} from './work-graph-tree-helpers';

	export let item: WIPItem;
	export let index: number;
	export let selectedIndex: number;
	export let itemId: string;
	export let copiedId: string | null;
	export let expandedDetails: Set<string>;
	export let pinnedTreeIds: Set<string>;
	export let treeNodeIndex: Map<string, TreeNode>;
	export let runningAgentDetailsByIssueId: Map<string, RunningAgentDetails>;
	export let onSelectNode: (index: number) => void;
	export let onCopyToClipboard: (id: string) => void;
	export let onOpenSidePanel: (node: TreeNode) => void;

	function handleClick() {
		onSelectNode(index);
		const panelIssue = getPanelIssue(item, treeNodeIndex);
		if (panelIssue) {
			onOpenSidePanel(panelIssue);
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Enter' || event.key === ' ') {
			event.preventDefault();
			handleClick();
		}
	}
</script>

{#if item.type === 'running'}
	{@const agent = item.agent}
	{@const displayId = agent.beads_id || agent.id.slice(0, 15)}
	{@const relatedNodeId = agent.beads_id && pinnedTreeIds.has(agent.beads_id) ? agent.beads_id : null}
	{@const relatedNode = relatedNodeId ? treeNodeIndex.get(relatedNodeId) : null}
	{@const statusIcon = getAgentStatusIcon(agent)}
	{@const health = computeAgentHealth(agent)}
	{@const contextPct = getContextPercent(agent)}
	{@const inProgressSubline = relatedNode
		? getInProgressSubline(relatedNode, runningAgentDetailsByIssueId)
		: (agent.phase || agent.runtime || agent.model
			? { text: `${agent.phase || 'active'} · ${agent.runtime || 'runtime unknown'} · ${shortenModel(agent.model)}`, tone: 'text-blue-500/90' }
			: null)}
	<!-- Running Agent - WIP Item -->
	<div
		class="flex items-start gap-3 py-2 px-1 rounded transition-colors {index === selectedIndex ? 'bg-zinc-800' : ''}"
		style="padding-left: 0"
		role="button"
		tabindex="-1"
		onclick={handleClick}
		onkeydown={handleKeydown}
	>
		<!-- Expansion indicator placeholder (matches tree nodes) -->
		<span class="w-4"></span>

		<!-- Status icon with health indication -->
		<span class="{statusIcon.color} w-5 text-center">{statusIcon.icon}</span>

		<!-- Priority badge -->
		{#if relatedNode}
			<Badge variant={getPriorityVariant(relatedNode.priority)} class="w-8 justify-center text-xs">
				P{relatedNode.priority}
			</Badge>
		{:else}
			<span class="w-8"></span>
		{/if}

		<!-- ID (min-w-[120px] matches tree) -->
		<span
			class="text-xs font-mono min-w-[120px] cursor-pointer hover:text-foreground transition-colors {copiedId === displayId ? 'text-green-500' : 'text-muted-foreground'}"
			onclick={(e) => { e.stopPropagation(); onCopyToClipboard(displayId); }}
			onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); e.preventDefault(); onCopyToClipboard(displayId); }}}
			role="button"
			tabindex="-1"
			title="Click to copy"
		>
			{copiedId === displayId ? 'Copied!' : displayId}
		</span>

		<!-- Title + in-progress details -->
		<div class="flex-1 min-w-0">
			<span class="block text-sm font-medium text-foreground truncate">
				{relatedNode?.title || agent.task || agent.skill || 'Unknown task'}
			</span>
			{#if inProgressSubline}
				<span class="block text-[11px] leading-4 truncate {inProgressSubline.tone}">
					{inProgressSubline.text}
				</span>
			{/if}
			<span class="block text-[11px] leading-4 truncate text-muted-foreground">
				{getExpressiveStatus(agent)}
			</span>
		</div>

		<!-- Attention badge (if any) -->
		{#if relatedNode?.attentionBadge}
			{@const badgeConfig = getAttentionBadge(relatedNode.attentionBadge)}
			{#if badgeConfig}
				<Badge variant={badgeConfig.variant} class="shrink-0">
					{badgeConfig.label}
				</Badge>
			{/if}
		{/if}

		<!-- Type badge -->
		{#if relatedNode}
			<Badge variant="outline" class="{getTypeBadge(relatedNode.type)} text-xs shrink-0">
				{relatedNode.type}
			</Badge>
		{/if}

			<!-- Health warning tooltip -->
			{#if health.status !== 'healthy'}
				<span class="text-xs {health.status === 'critical' ? 'text-red-500' : 'text-yellow-500'}" title={health.reasons.join(', ')}>
					{health.status === 'critical' ? '!' : '?'}
				</span>
			{/if}
		</div>

		<!-- L1: Expanded details for running agents -->
		{#if expandedDetails.has(itemId)}
			<div class="expanded-details ml-14 pb-2 px-3 text-xs text-muted-foreground bg-muted/30 rounded mt-1 mb-2 p-3 space-y-2">
				<!-- Agent metadata row -->
				<div class="flex items-center gap-4">
					<!-- Phase -->
					{#if agent.phase}
						<span class="flex items-center gap-1">
							<span class="text-foreground/60">Phase:</span>
							<span class="text-foreground">{agent.phase}</span>
						</span>
					{/if}

					<!-- Context % -->
					{#if contextPct !== null}
						<span class="flex items-center gap-1">
							<span class="text-foreground/60">Context:</span>
							<span class="{getContextColor(contextPct)}">{contextPct}%</span>
						</span>
					{/if}

					<!-- Skill -->
					{#if agent.skill}
						<span class="flex items-center gap-1">
							<span class="text-foreground/60">Skill:</span>
							<span>{agent.skill}</span>
						</span>
					{/if}

					<!-- Model (short form) -->
					{#if agent.model}
						<span class="flex items-center gap-1">
							<span class="text-foreground/60">Model:</span>
							<span>{agent.model.split('/').pop()?.split('-').slice(0, 2).join('-') || agent.model}</span>
						</span>
					{/if}
				</div>

				<!-- Deliverables checklist (compact L1 view) -->
				<DeliverableChecklist
					deliverables={getExpectedDeliverables('bug', agent.skill || 'feature-impl')}
					mode="compact"
				/>
			</div>
		{/if}
{:else}
	{@const issue = item.issue}
	<!-- Queued Issue - WIP Item (NO opacity-60) -->
	<div
		class="flex items-center gap-3 py-2 px-1 rounded transition-colors {index === selectedIndex ? 'bg-zinc-800' : ''}"
		style="padding-left: 0"
		role="button"
		tabindex="-1"
		onclick={handleClick}
		onkeydown={handleKeydown}
	>
		<!-- Expansion indicator placeholder (matches tree nodes) -->
		<span class="w-4"></span>

		<!-- Status icon -->
		<span class="text-muted-foreground w-5">○</span>

		<!-- Priority badge -->
		<Badge variant={getPriorityVariant(issue.priority)} class="w-8 justify-center text-xs">
			P{issue.priority}
		</Badge>
		{#if issue.effective_priority !== undefined && issue.effective_priority !== issue.priority}
			<Badge variant="outline" class="text-xs">
				Eff P{issue.effective_priority}
			</Badge>
		{/if}

		<!-- ID -->
		<span
			class="text-xs font-mono min-w-[120px] cursor-pointer hover:text-foreground transition-colors {copiedId === issue.id ? 'text-green-500' : 'text-muted-foreground'}"
			onclick={(e) => { e.stopPropagation(); onCopyToClipboard(issue.id); }}
			onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.stopPropagation(); e.preventDefault(); onCopyToClipboard(issue.id); }}}
			role="button"
			tabindex="-1"
			title="Click to copy"
		>
			{copiedId === issue.id ? 'Copied!' : issue.id}
		</span>

		<!-- Title -->
		<span class="flex-1 text-sm font-medium text-foreground truncate">
			{issue.title}
		</span>

		<!-- Queue reason -->
		<span class="text-xs text-muted-foreground italic min-w-[80px]">
			queued
		</span>

		<!-- Type badge -->
		<Badge variant="outline" class="{getTypeBadge(issue.issue_type)} text-xs">
			{issue.issue_type}
		</Badge>
	</div>

	<!-- L1: Expanded details for queued issues -->
	{#if expandedDetails.has(itemId)}
		<div class="expanded-details ml-14 mt-1 mb-2 p-3 bg-muted/30 rounded text-sm">
			<div class="text-xs text-muted-foreground">
				Queued for daemon processing
			</div>
		</div>
	{/if}
{/if}
