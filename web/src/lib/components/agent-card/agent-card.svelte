<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { SynthesisCard } from '$lib/components/synthesis-card';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import type { Agent } from '$lib/stores/agents';
	import { selectedAgentId, computeDisplayState, type DisplayState } from '$lib/stores/agents';
	import { hotspots, getHotspotForAgent, type Hotspot } from '$lib/stores/hotspot';
	import { coaching, type WorkerHealthMetrics } from '$lib/stores/coaching';

	export let agent: Agent;

	// Worker health metrics for this agent (derived from session_id)
	$: workerHealth = agent.session_id && $coaching.worker_health ? $coaching.worker_health[agent.session_id] : null;

	// Health indicator for the agent card
	$: healthIndicator = getHealthIndicator(workerHealth);

	/**
	 * Get health indicator based on worker health metrics
	 */
	function getHealthIndicator(health: WorkerHealthMetrics | null): { emoji: string; colorClass: string; label: string; details: string[] } | null {
		if (!health || health.health_status === 'good') return null;

		const details: string[] = [];
		let emoji = '⚠️';
		let colorClass = 'text-yellow-500';

		// Collect all issues
		if (health.tool_failure_rate >= 5) {
			details.push(`${health.tool_failure_rate} consecutive tool failures`);
		} else if (health.tool_failure_rate >= 3) {
			details.push(`${health.tool_failure_rate} tool failures`);
		}

		if (health.context_usage >= 90) {
			details.push(`${health.context_usage}% context used`);
		} else if (health.context_usage >= 80) {
			details.push(`${health.context_usage}% context`);
		}

		if (health.time_in_phase >= 30) {
			details.push(`${health.time_in_phase}m in current phase`);
		} else if (health.time_in_phase >= 15) {
			details.push(`${health.time_in_phase}m in phase`);
		}

		if (health.commit_gap >= 60) {
			details.push(`${health.commit_gap}m since last commit`);
		} else if (health.commit_gap >= 30) {
			details.push(`${health.commit_gap}m since commit`);
		}

		// Set severity indicators
		if (health.health_status === 'critical') {
			emoji = '🚨';
			colorClass = 'text-red-500';
		}

		const label = health.health_status === 'critical' ? 'Critical health issues' : 'Health warnings';

		return details.length > 0 ? { emoji, colorClass, label, details } : null;
	}

	// Track beads ID copy state
	let copiedBeadsId = false;

	async function copyBeadsId(e: MouseEvent) {
		e.stopPropagation(); // Prevent card click
		if (!agent.beads_id) return;
		try {
			await navigator.clipboard.writeText(agent.beads_id);
			copiedBeadsId = true;
			setTimeout(() => copiedBeadsId = false, 2000);
		} catch (err) {
			console.error('Failed to copy:', err);
		}
	}

	// Check if agent is working in a hotspot area
	$: agentHotspot = getHotspotForAgent(agent.beads_id, agent.task, agent.skill, $hotspots);

	$: isSelected = $selectedAgentId === agent.id;
	$: contextIndicator = getContextQualityIndicator(agent);
	$: displayState = computeDisplayState(agent);

	function handleClick() {
		selectedAgentId.set(agent.id);
	}

	/**
	 * Get user-friendly death reason message and icon
	 */
	function getDeathReasonInfo(reason?: string): { message: string; icon: string; color: string } {
		switch (reason) {
			case 'server_restart':
				return {
					message: 'Server restarted while agent was running.',
					icon: '🔄',
					color: 'text-blue-500'
				};
			case 'context_exhausted':
				return {
					message: 'Agent exhausted context/token limit.',
					icon: '📊',
					color: 'text-purple-500'
				};
			case 'auth_failed':
				return {
					message: 'Authentication failed (Claude Max limit?).',
					icon: '🔒',
					color: 'text-yellow-500'
				};
			case 'error':
				return {
					message: 'Agent encountered an unrecoverable error.',
					icon: '❌',
					color: 'text-red-500'
				};
			case 'timeout':
				return {
					message: 'No activity for 3+ minutes (timeout).',
					icon: '⏱️',
					color: 'text-orange-500'
				};
			default:
				return {
					message: 'Unknown cause (check logs).',
					icon: '💀',
					color: 'text-red-500'
				};
		}
	}

	/**
	 * Get short label for death reason (for status footer display)
	 */
	function getDeathReasonLabel(reason?: string): string {
		switch (reason) {
			case 'server_restart':
				return 'server restart';
			case 'context_exhausted':
				return 'context limit';
			case 'auth_failed':
				return 'auth failed';
			case 'error':
				return 'error';
			case 'timeout':
				return 'timeout';
			default:
				return 'crashed/stuck';
		}
	}

	function getStatusVariant(status: Agent['status']) {
		switch (status) {
			case 'active':
				return 'active';
			case 'completed':
				return 'completed';
			case 'abandoned':
				return 'abandoned';
			case 'awaiting-cleanup':
				return 'secondary'; // Amber/warning style
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
			case 'awaiting-cleanup':
				return 'bg-amber-500'; // Distinct from dead (red) and completed (blue)
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
	 * Get expressive status text based on current activity
	 * Returns text like "Hatching... (thought for 8s)" or "Running Bash..." or "Reading files..."
	 */
	function getExpressiveStatus(activity?: Agent['current_activity']): string {
		if (!activity) return 'Starting up';
		
		const elapsedSeconds = activity.timestamp ? Math.floor((Date.now() - activity.timestamp) / 1000) : 0;
		
		switch (activity.type) {
			case 'reasoning':
				// "Hatching..." with thinking duration
				return `Hatching... (thought for ${elapsedSeconds}s)`;
			case 'tool':
			case 'tool-invocation':
				// Extract tool name from text like "Using bash" -> "Running Bash..."
				if (activity.text?.toLowerCase().includes('bash')) {
					return 'Running Bash...';
				}
				if (activity.text?.toLowerCase().includes('read') || activity.text?.toLowerCase().includes('reading')) {
					return 'Reading files...';
				}
				if (activity.text?.toLowerCase().includes('edit') || activity.text?.toLowerCase().includes('editing')) {
					return 'Editing files...';
				}
				if (activity.text?.toLowerCase().includes('write') || activity.text?.toLowerCase().includes('writing')) {
					return 'Writing files...';
				}
				if (activity.text?.toLowerCase().includes('grep') || activity.text?.toLowerCase().includes('search')) {
					return 'Searching code...';
				}
				// Fallback: use the activity text as-is or generic tool message
				return activity.text || 'Using tool...';
			case 'text':
				return 'Responding...';
			case 'step-start':
				return 'Starting step...';
			case 'step-finish':
				return 'Finishing step...';
			default:
				return activity.text || 'Processing...';
		}
	}

	/**
	 * Format time since a date for elapsed display
	 */
	function formatElapsedTime(isoDate: string | undefined): string {
		if (!isoDate) return 'unknown time';
		const date = new Date(isoDate);
		if (isNaN(date.getTime())) return 'unknown time';
		const ms = Date.now() - date.getTime();
		const minutes = Math.floor(ms / 60000);
		if (minutes < 1) return 'less than a minute';
		if (minutes < 60) return `${minutes} minute${minutes === 1 ? '' : 's'}`;
		const hours = Math.floor(minutes / 60);
		const remainingMins = minutes % 60;
		if (remainingMins === 0) return `${hours} hour${hours === 1 ? '' : 's'}`;
		return `${hours}h ${remainingMins}m`;
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

	/**
	 * Get short outcome for badge display
	 * Extracts the core outcome (success, partial, blocked, failed) without parenthetical details
	 */
	function getShortOutcome(outcome: string): string {
		// Extract just the first word before any parenthetical details
		// e.g., "success (fix already implemented by prior agents)" -> "success"
		return outcome.split(/\s*\(/)[0].trim();
	}

	/**
	 * Check if outcome has additional details beyond the short version
	 */
	function hasOutcomeDetails(outcome: string): boolean {
		return outcome.includes('(') || outcome.length > 20;
	}

	/**
	 * Format model name for badge display (shortened versions)
	 */
	function formatModelBadge(model: string): string {
		const modelAbbreviations: Record<string, string> = {
			'gemini-3-flash-preview': 'flash3',
			'gemini-2.5-flash': 'flash-2.5',
			'gemini-2.5-pro': 'pro-2.5',
			'claude-opus-4-5-20251101': 'opus-4.5',
			'claude-sonnet-4-5-20250929': 'sonnet-4.5',
			'claude-haiku-4-5-20251001': 'haiku-4.5',
			'gpt-5': 'gpt5',
			'gpt-5.2': 'gpt5-latest',
			'gpt-5-mini': 'gpt5-mini',
			'o3': 'o3',
			'o3-mini': 'o3-mini',
			'deepseek-chat': 'deepseek',
			'deepseek-reasoner': 'deepseek-r1'
		};
		
		return modelAbbreviations[model] || model.substring(0, 12);
	}
</script>

<button
	type="button"
	onclick={handleClick}
	class="group relative w-full cursor-pointer rounded border bg-card p-2 text-left transition-all duration-500 hover:border-primary/50 hover:shadow-sm {displayState === 'running' ? 'border-yellow-500 shadow-md shadow-yellow-500/20' : displayState === 'ready-for-review' ? 'border-blue-500 shadow-md shadow-blue-500/20' : displayState === 'dead' ? 'border-red-500 shadow-md shadow-red-500/20' : displayState === 'awaiting-cleanup' ? 'border-amber-500 shadow-md shadow-amber-500/20' : agent.is_stalled ? 'border-orange-500 shadow-md shadow-orange-500/20' : displayState === 'idle' ? 'border-orange-500/50' : ''} {isSelected ? 'ring-2 ring-primary border-primary' : ''}"
>
	<!-- Status indicator bar at top - color reflects display state -->
	<div class={`absolute left-0 top-0 h-0.5 w-full rounded-t transition-colors duration-500 ${displayState === 'running' ? 'bg-yellow-500' : displayState === 'ready-for-review' ? 'bg-blue-500' : displayState === 'dead' ? 'bg-red-500' : displayState === 'awaiting-cleanup' ? 'bg-amber-500' : agent.is_stalled ? 'bg-orange-500' : displayState === 'idle' ? 'bg-orange-500' : getStatusColor(agent.status)}`}></div>

	<!-- Header: Status + Phase + Duration -->
	<div class="flex items-center justify-between gap-1">
		<div class="flex items-center gap-1">
			<Badge variant={getStatusVariant(agent.status)} class="h-4 px-1.5 text-[10px]">
				{agent.status}
			</Badge>
			{#if agent.is_stale}
				<Tooltip.Root>
					<Tooltip.Trigger>
						<Badge variant="outline" class="h-4 px-1 text-[10px] text-muted-foreground">
							📦
						</Badge>
					</Tooltip.Trigger>
					<Tooltip.Content>
						<p class="font-medium">Stale Agent</p>
						<p class="text-xs text-muted-foreground">
							Last updated > 2h ago.<br />
							Phase and task data may be outdated.
						</p>
					</Tooltip.Content>
				</Tooltip.Root>
			{:else if agent.phase}
				<Badge variant={getPhaseVariant(agent.phase)} class="h-4 px-1 text-[10px]">
					{agent.phase}
				</Badge>
			{/if}
		</div>
		<span class="flex items-center gap-0.5 text-[10px] text-muted-foreground">
			{#if healthIndicator}
				<Tooltip.Root>
					<Tooltip.Trigger>
						<span class={healthIndicator.colorClass}>{healthIndicator.emoji}</span>
					</Tooltip.Trigger>
					<Tooltip.Content>
						<p class={`font-medium ${healthIndicator.colorClass}`}>{healthIndicator.label}</p>
						{#each healthIndicator.details as detail}
							<p class="text-xs text-muted-foreground">{detail}</p>
						{/each}
					</Tooltip.Content>
				</Tooltip.Root>
			{/if}
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
		{#if displayState === 'dead'}
			{@const deathInfo = getDeathReasonInfo(agent.death_reason)}
			<Tooltip.Root>
				<Tooltip.Trigger>
					<span class={deathInfo.color}>{deathInfo.icon}</span>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p class="font-medium {deathInfo.color}">Dead Agent</p>
					<p class="text-xs text-muted-foreground mb-1">
						{deathInfo.message}
					</p>
					{#if agent.death_reason === 'timeout'}
						<p class="text-xs text-muted-foreground opacity-70">
							Agent may have crashed, been killed, or is stuck.
						</p>
					{/if}
				</Tooltip.Content>
			</Tooltip.Root>
		{:else if displayState === 'awaiting-cleanup'}
			<Tooltip.Root>
				<Tooltip.Trigger>
					<span class="text-amber-500">🧹</span>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p class="font-medium text-amber-500">Awaiting Cleanup</p>
					<p class="text-xs text-muted-foreground">
						Agent completed work but wasn't formally closed.<br />
						Run <code class="bg-muted px-1 rounded">orch complete</code> to clean up.
					</p>
				</Tooltip.Content>
			</Tooltip.Root>
		{:else if agent.is_stalled}
			<Tooltip.Root>
				<Tooltip.Trigger>
					<span class="text-orange-500">⏱️</span>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p class="font-medium text-orange-500">Stalled Agent</p>
					<p class="text-xs text-muted-foreground">
						Same phase ({agent.phase || 'unknown'}) for 15+ minutes.<br />
						May be stuck, blocked, or waiting for input.
					</p>
				</Tooltip.Content>
			</Tooltip.Root>
		{:else if displayState === 'running'}
			<Tooltip.Root>
				<Tooltip.Trigger>
					<span class="h-1.5 w-1.5 animate-pulse rounded-full bg-yellow-500"></span>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p class="font-medium text-yellow-500">{getExpressiveStatus(agent.current_activity)}</p>
					<p class="text-xs text-muted-foreground">Agent is actively generating a response</p>
				</Tooltip.Content>
			</Tooltip.Root>
		{:else if displayState === 'ready-for-review'}
			<Tooltip.Root>
				<Tooltip.Trigger>
					<span class="h-1.5 w-1.5 rounded-full bg-blue-500"></span>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p class="font-medium text-blue-500">Ready for Review</p>
					<p class="text-xs text-muted-foreground">
						Agent reported Phase: Complete.<br />
						Run <code class="bg-muted px-1 rounded">orch complete</code> to close.
					</p>
				</Tooltip.Content>
			</Tooltip.Root>
		{:else if displayState === 'idle'}
			<Tooltip.Root>
				<Tooltip.Trigger>
					<span class="h-1.5 w-1.5 rounded-full bg-orange-500/70"></span>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p class="font-medium text-orange-500">Idle</p>
					<p class="text-xs text-muted-foreground">
						No activity for 60+ seconds.<br />
						Agent may be waiting for input or processing slowly.
					</p>
				</Tooltip.Content>
			</Tooltip.Root>
		{:else if agent.status === 'active'}
			<Tooltip.Root>
				<Tooltip.Trigger>
					<span class="h-1 w-1 rounded-full bg-green-500"></span>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p class="font-medium text-green-500">Active</p>
					<p class="text-xs text-muted-foreground">Agent is running normally</p>
				</Tooltip.Content>
			</Tooltip.Root>
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

	<!-- Project + Skill + Model + Beads -->
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
		{#if agent.model}
			<Tooltip.Root>
				<Tooltip.Trigger>
					<Badge variant="outline" class="h-4 px-1 text-[10px] text-purple-600 dark:text-purple-400 border-purple-300 dark:border-purple-700">
						{formatModelBadge(agent.model)}
					</Badge>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p class="text-xs">Model: {agent.model}</p>
				</Tooltip.Content>
			</Tooltip.Root>
		{/if}
		{#if agent.beads_id}
			<Tooltip.Root>
				<Tooltip.Trigger>
					<button
						type="button"
						onclick={copyBeadsId}
						class="text-[10px] text-muted-foreground hover:text-foreground transition-colors"
					>
						{copiedBeadsId ? '✓ copied' : agent.beads_id}
					</button>
				</Tooltip.Trigger>
				<Tooltip.Content>
					<p>Click to copy</p>
				</Tooltip.Content>
			</Tooltip.Root>
		{/if}
	</div>

	<!-- Current Activity Summary (for active agents and dead agents) - always reserve space to prevent height jitter -->
	{#if agent.status === 'active' || agent.status === 'dead'}
		<div class="mt-1.5 border-t border-border/50 pt-1.5">
			{#if displayState === 'dead' || agent.status === 'dead'}
				<!-- Agent is dead - no heartbeat for 3+ minutes -->
				{@const deathInfo = getDeathReasonInfo(agent.death_reason)}
				<div class="flex items-center gap-1">
					<span class="text-[10px]">{deathInfo.icon}</span>
					<p class="flex-1 truncate text-[10px] text-red-400 font-medium">
						No activity for {formatElapsedTime(agent.updated_at)}
					</p>
					<Tooltip.Root>
						<Tooltip.Trigger>
							<span class="text-[9px] text-red-400/70 shrink-0">
								{getDeathReasonLabel(agent.death_reason)}
							</span>
						</Tooltip.Trigger>
						<Tooltip.Content>
							<p class="font-medium text-red-500">Agent Unresponsive</p>
							<p class="text-xs text-muted-foreground mb-1">
								{deathInfo.message}
							</p>
							{#if agent.death_reason === 'timeout'}
								<p class="text-xs text-muted-foreground opacity-70">
									Agent may have crashed, been killed, or is stuck.
								</p>
							{/if}
							<p class="text-xs text-muted-foreground mt-1">
								Consider running <code class="bg-muted px-1 rounded">orch abandon</code>
							</p>
						</Tooltip.Content>
					</Tooltip.Root>
				</div>
			{:else if agent.is_stalled}
				<!-- Agent is stalled - same phase for 15+ minutes -->
				<div class="flex items-center gap-1">
					<span class="text-[10px]">⏱️</span>
					<p class="flex-1 truncate text-[10px] text-orange-400 font-medium">
						Stuck at {agent.phase || 'current phase'} for 15+ min
					</p>
					<Tooltip.Root>
						<Tooltip.Trigger>
							<span class="text-[9px] text-orange-400/70 shrink-0">
								may need attention
							</span>
						</Tooltip.Trigger>
						<Tooltip.Content>
							<p class="font-medium text-orange-500">Progress Stalled</p>
							<p class="text-xs text-muted-foreground">
								Agent has been at phase "{agent.phase}" for 15+ minutes.<br />
								May be blocked, waiting for input, or stuck in a loop.
							</p>
							<p class="text-xs text-muted-foreground mt-1">
								Check the agent's output for blockers or errors.
							</p>
						</Tooltip.Content>
					</Tooltip.Root>
				</div>
			{:else if displayState === 'ready-for-review'}
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
						Idle - no activity for {formatActivityAge(agent.current_activity?.timestamp)}
					</p>
					<Tooltip.Root>
						<Tooltip.Trigger>
							<span class="text-[9px] text-muted-foreground/70 shrink-0">
								waiting
							</span>
						</Tooltip.Trigger>
						<Tooltip.Content>
							<p class="font-medium text-orange-500">Agent Idle</p>
							<p class="text-xs text-muted-foreground">
								No activity for 60+ seconds.<br />
								Agent may be waiting for input, thinking,<br />
								or processing a slow operation.
							</p>
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
			{#if hasOutcomeDetails(agent.synthesis.outcome)}
				<Tooltip.Root>
					<Tooltip.Trigger>
						<Badge variant={getShortOutcome(agent.synthesis.outcome) === 'success' ? 'default' : 'secondary'} class="h-4 px-1 text-[10px]">
							{getShortOutcome(agent.synthesis.outcome)}
						</Badge>
					</Tooltip.Trigger>
					<Tooltip.Content class="max-w-xs">
						<p>{agent.synthesis.outcome}</p>
					</Tooltip.Content>
				</Tooltip.Root>
			{:else}
				<Badge variant={agent.synthesis.outcome === 'success' ? 'default' : 'secondary'} class="h-4 px-1 text-[10px]">
					{agent.synthesis.outcome}
				</Badge>
			{/if}
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
