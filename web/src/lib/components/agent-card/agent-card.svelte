<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { SynthesisCard } from '$lib/components/synthesis-card';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import type { Agent } from '$lib/stores/agents';
	import { selectedAgentId, computeDisplayState, type DisplayState } from '$lib/stores/agents';
	import { hotspots, getHotspotForAgent, type Hotspot } from '$lib/stores/hotspot';

	export let agent: Agent;

	// Check if agent is working in a hotspot area
	$: agentHotspot = getHotspotForAgent(agent.beads_id, agent.task, agent.skill, $hotspots);

	$: isSelected = $selectedAgentId === agent.id;
	$: contextIndicator = getContextQualityIndicator(agent);
	$: displayState = computeDisplayState(agent);

	function handleClick() {
		selectedAgentId.set(agent.id);
	}

	function getStatusVariant(status: Agent['status']) {
		switch (status) {
			case 'active':
				return 'active';
			case 'completed':
				return 'completed';
			case 'abandoned':
				return 'abandoned';
			default:
				return 'default';
		}
	}

	function getStatusColor(status: Agent['status']) {
		switch (status) {
			case 'active':
				return 'bg-green-500';
			case 'completed':
				return 'bg-blue-500';
			case 'abandoned':
				return 'bg-red-500';
			default:
				return 'bg-gray-500';
		}
	}

	function getPhaseVariant(phase?: string): 'default' | 'secondary' | 'outline' {
		if (!phase) return 'outline';
		switch (phase.toLowerCase()) {
			case 'complete':
				return 'default';
			case 'implementing':
				return 'secondary';
			default:
				return 'outline';
		}
	}

	/**
	 * Get context quality indicator based on gap analysis
	 * Returns emoji and color class for visual representation
	 */
	function getContextQualityIndicator(agent: Agent): { emoji: string; colorClass: string; label: string } | null {
		if (!agent.gap_analysis) return null;
		
		const quality = agent.gap_analysis.context_quality;
		
		if (quality === 0) {
			return { emoji: '🚨', colorClass: 'text-red-500', label: 'No context' };
		}
		if (quality < 20) {
			return { emoji: '⚠️', colorClass: 'text-red-500', label: `${quality}% context` };
		}
		if (quality < 40) {
			return { emoji: '⚠️', colorClass: 'text-yellow-500', label: `${quality}% context` };
		}
		// Good quality - no indicator needed
		return null;
	}

	function formatDuration(isoDate: string | undefined): string {
		if (!isoDate) return '-';
		const date = new Date(isoDate);
		if (isNaN(date.getTime())) return '-';
		const ms = Date.now() - date.getTime();
		const minutes = Math.floor(ms / 60000);
		const hours = Math.floor(minutes / 60);
		if (hours > 0) {
			return `${hours}h${minutes % 60}m`;
		}
		return `${minutes}m`;
	}

	function getActivityIcon(type?: string): string {
		switch (type) {
			case 'text':
				return '💬';
			case 'tool':
			case 'tool-invocation':
				return '🔧';
			case 'reasoning':
				return '🤔';
			case 'step-start':
				return '▶️';
			case 'step-finish':
				return '✓';
			default:
				return '📝';
		}
	}

	function formatActivityAge(timestamp?: number): string {
		if (!timestamp) return '';
		const seconds = Math.floor((Date.now() - timestamp) / 1000);
		if (seconds < 5) return 'now';
		if (seconds < 60) return `${seconds}s ago`;
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes}m ago`;
		const hours = Math.floor(minutes / 60);
		return `${hours}h ago`;
	}

	/**
	 * Clean workspace name into human-readable format
	 * e.g., "og-feat-improve-agent-card-24dec [orch-go-uu9v]" -> "Improve agent card"
	 */
	function cleanWorkspaceName(id: string): string {
		// Remove beads ID suffix like " [orch-go-uu9v]"
		let cleaned = id.replace(/\s*\[[^\]]+\]$/, '');
		
		// Remove common prefixes like og-, proj-, etc. and date suffixes
		cleaned = cleaned
			.replace(/^[a-z]+-/, '') // Remove project prefix (og-, sk-, etc.)
			.replace(/-\d{1,2}[a-z]{3}$/, '') // Remove date suffix (24dec, 5jan, etc.)
			.replace(/^(feat|fix|inv|debug|research|design)-/, '') // Remove skill prefixes
			.replace(/-/g, ' ') // Replace hyphens with spaces
			.trim();
		
		// Capitalize first letter
		return cleaned.charAt(0).toUpperCase() + cleaned.slice(1);
	}

	/**
	 * Extract first sentence from TLDR, truncated to ~50 chars
	 */
	function truncateTldr(tldr: string, maxLen: number = 50): string {
		// Get first sentence
		const firstSentence = tldr.split(/[.!?]/)[0].trim();
		
		if (firstSentence.length <= maxLen) {
			return firstSentence;
		}
		
		// Truncate at word boundary
		const truncated = firstSentence.substring(0, maxLen);
		const lastSpace = truncated.lastIndexOf(' ');
		if (lastSpace > maxLen * 0.6) {
			return truncated.substring(0, lastSpace) + '…';
		}
		return truncated + '…';
	}

	/**
	 * Get display title for agent card
	 * - Completed agents: TLDR first sentence (truncated), or close_reason fallback
	 * - Active agents: task field, or cleaned workspace name
	 */
	function getDisplayTitle(agent: Agent): string {
		if (agent.status === 'completed') {
			// First try synthesis TLDR
			if (agent.synthesis?.tldr) {
				return truncateTldr(agent.synthesis.tldr);
			}
			// Fallback to close_reason for light-tier agents
			if (agent.close_reason) {
				return truncateTldr(agent.close_reason, 60);
			}
		}
		
		if (agent.task) {
			// Task is usually a good title, truncate if needed
			return truncateTldr(agent.task, 60);
		}
		
		return cleanWorkspaceName(agent.id);
	}

	/**
	 * Check if we should show workspace as subtitle
	 * (when display title differs from workspace name)
	 */
	function shouldShowWorkspaceSubtitle(agent: Agent): boolean {
		// For completed agents with TLDR or close_reason, always show workspace
		if (agent.status === 'completed' && (agent.synthesis?.tldr || agent.close_reason)) {
			return true;
		}
		// For agents with task, show workspace if task is displayed
		if (agent.task) {
			return true;
		}
		return false;
	}
</script>

<button
	type="button"
	onclick={handleClick}
	class="group relative w-full cursor-pointer rounded border bg-card p-2 text-left transition-all duration-500 hover:border-primary/50 hover:shadow-sm {displayState === 'running' ? 'border-yellow-500 shadow-md shadow-yellow-500/20' : displayState === 'ready-for-review' ? 'border-blue-500 shadow-md shadow-blue-500/20' : displayState === 'idle' ? 'border-orange-500/50' : ''} {isSelected ? 'ring-2 ring-primary border-primary' : ''}"
>
	<!-- Status indicator bar at top - color reflects display state -->
	<div class={`absolute left-0 top-0 h-0.5 w-full rounded-t transition-colors duration-500 ${displayState === 'running' ? 'bg-yellow-500' : displayState === 'ready-for-review' ? 'bg-blue-500' : displayState === 'idle' ? 'bg-orange-500' : getStatusColor(agent.status)}`}></div>

	<!-- Header: Status + Phase + Duration -->
	<div class="flex items-center justify-between gap-1">
		<div class="flex items-center gap-1">
			<Badge variant={getStatusVariant(agent.status)} class="h-4 px-1.5 text-[10px]">
				{agent.status}
			</Badge>
			{#if agent.phase}
				<Badge variant={getPhaseVariant(agent.phase)} class="h-4 px-1 text-[10px]">
					{agent.phase}
				</Badge>
			{/if}
		</div>
		<span class="flex items-center gap-0.5 text-[10px] text-muted-foreground">
			{#if agentHotspot}
				<Tooltip.Root>
					<Tooltip.Trigger>
						<span class="text-orange-500 animate-pulse">🔥</span>
					</Tooltip.Trigger>
					<Tooltip.Content>
						<p class="font-medium text-orange-500">Hotspot Area</p>
						<p class="text-xs">{agentHotspot.path}</p>
						<p class="text-xs text-muted-foreground mt-1">
							{agentHotspot.type === 'fix-density' 
								? `${agentHotspot.score} fix commits` 
								: `${agentHotspot.score} investigations`}
						</p>
						<p class="text-xs text-orange-400 mt-1">{agentHotspot.recommendation}</p>
					</Tooltip.Content>
				</Tooltip.Root>
			{/if}
			{#if contextIndicator}
				<Tooltip.Root>
					<Tooltip.Trigger>
						<span class={contextIndicator.colorClass}>{contextIndicator.emoji}</span>
					</Tooltip.Trigger>
					<Tooltip.Content>
						<p>{contextIndicator.label}</p>
						{#if agent.gap_analysis}
							<p class="text-xs text-muted-foreground mt-1">
								Found: {agent.gap_analysis.match_count ?? 0} matches
								({agent.gap_analysis.constraints ?? 0} constraints,
								{agent.gap_analysis.decisions ?? 0} decisions,
								{agent.gap_analysis.investigations ?? 0} investigations)
							</p>
						{/if}
					</Tooltip.Content>
				</Tooltip.Root>
			{/if}
			{#if displayState === 'running'}
				<Tooltip.Root>
					<Tooltip.Trigger>
						<span class="h-1.5 w-1.5 animate-pulse rounded-full bg-yellow-500"></span>
					</Tooltip.Trigger>
					<Tooltip.Content>
						<p>Generating response</p>
					</Tooltip.Content>
				</Tooltip.Root>
			{:else if displayState === 'ready-for-review'}
				<Tooltip.Root>
					<Tooltip.Trigger>
						<span class="h-1.5 w-1.5 rounded-full bg-blue-500"></span>
					</Tooltip.Trigger>
					<Tooltip.Content>
						<p>Done - pending review</p>
					</Tooltip.Content>
				</Tooltip.Root>
			{:else if displayState === 'idle'}
				<Tooltip.Root>
					<Tooltip.Trigger>
						<span class="h-1.5 w-1.5 rounded-full bg-orange-500"></span>
					</Tooltip.Trigger>
					<Tooltip.Content>
						<p>Idle - no recent activity</p>
					</Tooltip.Content>
				</Tooltip.Root>
			{:else if agent.status === 'active'}
				<span class="h-1 w-1 rounded-full bg-green-500"></span>
			{/if}
			<Tooltip.Root>
				<Tooltip.Trigger>
					<span class="cursor-default">{agent.runtime || formatDuration(agent.spawned_at)}</span>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p>Spawned: {agent.spawned_at ? new Date(agent.spawned_at).toLocaleString() : 'Unknown'}</p>
					{#if agent.updated_at}
						<p class="text-xs text-muted-foreground">Last update: {new Date(agent.updated_at).toLocaleString()}</p>
					{/if}
				</Tooltip.Content>
			</Tooltip.Root>
		</span>
	</div>

	<!-- Title (human-readable) -->
	<Tooltip.Root>
		<Tooltip.Trigger>
			{#snippet child({ props })}
				<span {...props} class="mt-1 block w-full truncate text-left text-xs font-medium cursor-default">
					{getDisplayTitle(agent)}
				</span>
			{/snippet}
		</Tooltip.Trigger>
		<Tooltip.Content class="max-w-xs">
			<p>{agent.synthesis?.tldr || agent.task || agent.id}</p>
		</Tooltip.Content>
	</Tooltip.Root>

	<!-- Workspace ID (as subtitle when title differs) -->
	{#if shouldShowWorkspaceSubtitle(agent)}
		<Tooltip.Root>
			<Tooltip.Trigger>
				{#snippet child({ props })}
					<span {...props} class="block w-full truncate text-left font-mono text-[10px] text-muted-foreground cursor-default">
						{agent.id}
					</span>
				{/snippet}
			</Tooltip.Trigger>
			<Tooltip.Content>
				<p class="font-mono text-xs">{agent.id}</p>
			</Tooltip.Content>
		</Tooltip.Root>
	{/if}

	<!-- Project + Skill + Beads -->
	<div class="mt-1 flex flex-wrap items-center gap-1">
		{#if agent.project}
			<Badge variant="secondary" class="h-4 px-1 text-[10px]">
				{agent.project}
			</Badge>
		{/if}
		{#if agent.skill}
			<Badge variant="outline" class="h-4 px-1 text-[10px]">
				{agent.skill}
			</Badge>
		{/if}
		{#if agent.beads_id}
			<Tooltip.Root>
				<Tooltip.Trigger>
					<span class="text-[10px] text-muted-foreground cursor-default">
						{agent.beads_id}
					</span>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p>Beads Issue: {agent.beads_id}</p>
					<p class="text-xs text-muted-foreground">Run <code>bd show {agent.beads_id}</code> for details</p>
				</Tooltip.Content>
			</Tooltip.Root>
		{/if}
	</div>

	<!-- Current Activity Summary (for active agents) - always reserve space to prevent height jitter -->
	{#if agent.status === 'active'}
		<div class="mt-1.5 border-t border-border/50 pt-1.5">
			{#if displayState === 'ready-for-review'}
				<!-- Agent reported Phase: Complete, waiting for orchestrator to close -->
				<div class="flex items-center gap-1">
					<span class="text-[10px]">✅</span>
					<p class="flex-1 truncate text-[10px] text-blue-400 font-medium">
						Done - pending review
					</p>
					<Tooltip.Root>
						<Tooltip.Trigger>
							<span class="text-[9px] text-muted-foreground/70 shrink-0">
								{agent.runtime || formatDuration(agent.spawned_at)}
							</span>
						</Tooltip.Trigger>
						<Tooltip.Content>
							<p>Agent reported Phase: Complete</p>
							<p class="text-xs text-muted-foreground">Run <code>orch complete</code> to close</p>
						</Tooltip.Content>
					</Tooltip.Root>
				</div>
			{:else if displayState === 'idle'}
				<!-- Agent has been idle for a while - might be stuck or waiting for input -->
				<div class="flex items-center gap-1">
					<span class="text-[10px]">💤</span>
					<p class="flex-1 truncate text-[10px] text-orange-400">
						Idle
					</p>
					<Tooltip.Root>
						<Tooltip.Trigger>
							<span class="text-[9px] text-muted-foreground/70 shrink-0">
								{formatActivityAge(agent.current_activity?.timestamp)}
							</span>
						</Tooltip.Trigger>
						<Tooltip.Content>
							<p>No activity for a while</p>
							<p class="text-xs text-muted-foreground">May be stuck or waiting for input</p>
						</Tooltip.Content>
					</Tooltip.Root>
				</div>
			{:else if agent.current_activity}
				<div class="flex items-center gap-1">
					<span class="text-[10px]">{getActivityIcon(agent.current_activity.type)}</span>
					<p class="flex-1 truncate text-[10px] text-muted-foreground">
						{agent.current_activity.text || 'Working...'}
					</p>
					<span class="text-[9px] text-muted-foreground/70 shrink-0">
						{formatActivityAge(agent.current_activity.timestamp)}
					</span>
				</div>
			{:else}
				<!-- Placeholder to maintain consistent card height -->
				<div class="flex items-center gap-1">
					<span class="text-[10px] text-muted-foreground/50">⏳</span>
					<p class="flex-1 truncate text-[10px] text-muted-foreground/50">
						Starting up...
					</p>
				</div>
			{/if}
		</div>
	{/if}

	<!-- Synthesis outcome badge for completed agents (only if outcome exists) -->
	<!-- Note: TLDR/close_reason already shown in title - no need to duplicate -->
	{#if agent.status === 'completed' && agent.synthesis?.outcome}
		<div class="mt-1.5 border-t border-border/50 pt-1.5">
			<Badge variant={agent.synthesis.outcome === 'success' ? 'default' : 'secondary'} class="h-4 px-1 text-[10px]">
				{agent.synthesis.outcome}
			</Badge>
		</div>
	{/if}

	<!-- Abandoned agents footer - reserve space for consistency -->
	{#if agent.status === 'abandoned'}
		<div class="mt-1.5 border-t border-border/50 pt-1.5">
			<p class="text-[10px] leading-tight text-red-500/70">
				{agent.close_reason || 'Agent was abandoned'}
			</p>
		</div>
	{/if}
</button>
