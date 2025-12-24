<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { SynthesisCard } from '$lib/components/synthesis-card';
	import type { Agent } from '$lib/stores/agents';
	import { selectedAgentId } from '$lib/stores/agents';

	export let agent: Agent;

	$: isSelected = $selectedAgentId === agent.id;

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
	 * - Completed agents: TLDR first sentence (truncated)
	 * - Active agents: task field, or cleaned workspace name
	 */
	function getDisplayTitle(agent: Agent): string {
		if (agent.status === 'completed' && agent.synthesis?.tldr) {
			return truncateTldr(agent.synthesis.tldr);
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
		// For completed agents with TLDR, always show workspace
		if (agent.status === 'completed' && agent.synthesis?.tldr) {
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
	class="group relative w-full cursor-pointer rounded border bg-card p-2 text-left transition-all hover:border-primary/50 hover:shadow-sm {agent.is_processing ? 'border-yellow-500 animate-pulse shadow-md shadow-yellow-500/20' : ''} {isSelected ? 'ring-2 ring-primary border-primary' : ''}"
>
	<!-- Status indicator bar at top - yellow when processing -->
	<div class={`absolute left-0 top-0 h-0.5 w-full rounded-t ${agent.is_processing ? 'bg-yellow-500' : getStatusColor(agent.status)}`}></div>

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
			{#if agent.is_processing}
				<span class="h-1.5 w-1.5 animate-pulse rounded-full bg-yellow-500" title="Generating response"></span>
			{:else if agent.status === 'active'}
				<span class="h-1 w-1 rounded-full bg-green-500"></span>
			{/if}
			{agent.runtime || formatDuration(agent.spawned_at)}
		</span>
	</div>

	<!-- Title (human-readable) -->
	<p class="mt-1 truncate text-xs font-medium" title={agent.synthesis?.tldr || agent.task || agent.id}>
		{getDisplayTitle(agent)}
	</p>

	<!-- Workspace ID (as subtitle when title differs) -->
	{#if shouldShowWorkspaceSubtitle(agent)}
		<p class="truncate font-mono text-[10px] text-muted-foreground" title={agent.id}>
			{agent.id}
		</p>
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
			<span class="text-[10px] text-muted-foreground" title="Beads Issue">
				{agent.beads_id}
			</span>
		{/if}
	</div>

	<!-- Current Activity (for active agents) -->
	{#if agent.status === 'active' && agent.current_activity}
		<div class="mt-1.5 border-t border-border/50 pt-1.5">
			<div class="flex items-start gap-1">
				<span class="text-[10px]">{getActivityIcon(agent.current_activity.type)}</span>
				<div class="flex-1 min-w-0">
					<p class="line-clamp-2 text-[10px] leading-tight text-muted-foreground">
						{agent.current_activity.text || 'Working...'}
					</p>
					<span class="text-[9px] text-muted-foreground/70">
						{formatActivityAge(agent.current_activity.timestamp)}
					</span>
				</div>
			</div>
		</div>
	{/if}

	<!-- Synthesis for completed agents -->
	{#if agent.status === 'completed' && agent.synthesis}
		<div class="mt-1.5 border-t border-border/50 pt-1.5">
			{#if agent.synthesis.tldr}
				<p class="text-[10px] leading-tight text-muted-foreground">
					{agent.synthesis.tldr}
				</p>
			{/if}
			{#if agent.synthesis.outcome}
				<Badge variant={agent.synthesis.outcome === 'success' ? 'default' : 'secondary'} class="mt-1 h-4 px-1 text-[10px]">
					{agent.synthesis.outcome}
				</Badge>
			{/if}
		</div>
	{/if}
</button>
