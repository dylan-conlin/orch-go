<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import { agents, selectedAgentId, type Agent } from '$lib/stores/agents';
	import { derived } from 'svelte/store';
	import { slide } from 'svelte/transition';
	import { cubicOut } from 'svelte/easing';

	// Recent completions: agents completed in last 4 hours (immediate visibility)
	// Agents older than 4h but within 24h are still in the Recent section (CollapsibleSection)
	const RECENT_THRESHOLD_MS = 4 * 60 * 60 * 1000; // 4 hours

	// Derive recent completions from agents store
	const recentCompletions = derived(agents, ($agents) => {
		const now = Date.now();
		return $agents.filter((a) => {
			if (a.status !== 'completed') return false;
			// Check completed_at or updated_at
			const completedAt = a.completed_at 
				? new Date(a.completed_at).getTime() 
				: (a.updated_at ? new Date(a.updated_at).getTime() : 0);
			return now - completedAt < RECENT_THRESHOLD_MS;
		}).sort((a, b) => {
			// Most recent first
			const aTime = a.completed_at ? new Date(a.completed_at).getTime() : (a.updated_at ? new Date(a.updated_at).getTime() : 0);
			const bTime = b.completed_at ? new Date(b.completed_at).getTime() : (b.updated_at ? new Date(b.updated_at).getTime() : 0);
			return bTime - aTime;
		});
	});

	// Track expanded state
	let expanded = false;

	function handleClick(agent: Agent) {
		selectedAgentId.set(agent.id);
	}

	function formatTimeAgo(isoDate: string | undefined): string {
		if (!isoDate) return '';
		const date = new Date(isoDate);
		if (isNaN(date.getTime())) return '';
		const ms = Date.now() - date.getTime();
		const minutes = Math.floor(ms / 60000);
		const hours = Math.floor(minutes / 60);
		if (hours > 0) {
			return `${hours}h ago`;
		}
		if (minutes < 1) {
			return 'just now';
		}
		return `${minutes}m ago`;
	}

	function getOutcomeEmoji(outcome?: string): string {
		switch (outcome) {
			case 'success':
				return '✅';
			case 'partial':
				return '⚡';
			case 'blocked':
				return '🚧';
			case 'failed':
				return '❌';
			default:
				return '✅'; // Default to success for light-tier without explicit outcome
		}
	}

	function getOutcomeColor(outcome?: string): string {
		switch (outcome) {
			case 'success':
				return 'text-green-500';
			case 'partial':
				return 'text-yellow-500';
			case 'blocked':
				return 'text-orange-500';
			case 'failed':
				return 'text-red-500';
			default:
				return 'text-green-500';
		}
	}

	function truncateText(text: string, maxLen: number = 70): string {
		if (text.length <= maxLen) return text;
		const truncated = text.substring(0, maxLen);
		const lastSpace = truncated.lastIndexOf(' ');
		if (lastSpace > maxLen * 0.6) {
			return truncated.substring(0, lastSpace) + '…';
		}
		return truncated + '…';
	}

	function getDisplayText(agent: Agent): string {
		if (agent.synthesis?.tldr) {
			return truncateText(agent.synthesis.tldr);
		}
		if (agent.close_reason) {
			return truncateText(agent.close_reason);
		}
		if (agent.task) {
			return truncateText(agent.task);
		}
		// Fall back to cleaned workspace name
		return cleanWorkspaceName(agent.id);
	}

	function cleanWorkspaceName(id: string): string {
		return id
			.replace(/\s*\[[^\]]+\]$/, '')
			.replace(/^[a-z]+-/, '')
			.replace(/-\d{1,2}[a-z]{3}$/, '')
			.replace(/^(feat|fix|inv|debug|research|design)-/, '')
			.replace(/-/g, ' ')
			.trim();
	}

	// Get summary for collapsed header preview
	function getCollapsedPreview(): string {
		const wins = $recentCompletions;
		if (wins.length === 0) return '';
		
		const first = wins[0];
		const firstText = first.synthesis?.tldr 
			? truncateText(first.synthesis.tldr, 40)
			: first.close_reason 
				? truncateText(first.close_reason, 40)
				: first.task 
					? truncateText(first.task, 40)
					: cleanWorkspaceName(first.id);
		
		if (wins.length === 1) {
			return firstText;
		}
		
		return `${firstText} +${wins.length - 1}`;
	}

	let collapsedPreview = $derived(getCollapsedPreview());
</script>

{#if $recentCompletions.length > 0}
	<div class="rounded-lg border border-green-500/30 bg-green-500/5" data-testid="recent-completions-section">
		<button
			class="flex w-full items-center justify-between px-3 py-2 text-left rounded-lg transition-colors duration-150
				hover:bg-accent/30 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-inset"
			onclick={() => { expanded = !expanded; }}
			aria-expanded={expanded}
		>
			<div class="flex items-center gap-2 min-w-0 flex-1">
				<span class="text-base flex-shrink-0 transition-transform duration-200 {expanded ? 'scale-110' : ''}">✨</span>
				<span class="text-sm font-medium flex-shrink-0">Recently Completed</span>
				<Badge variant="default" class="h-5 px-1.5 text-xs flex-shrink-0 bg-green-500/80">
					{$recentCompletions.length}
				</Badge>
				<span class="text-[10px] text-muted-foreground flex-shrink-0">last 4h</span>
				{#if !expanded && collapsedPreview}
					<span class="text-xs text-muted-foreground truncate opacity-70">
						— {collapsedPreview}
					</span>
				{/if}
			</div>
			<span class="text-muted-foreground transition-transform duration-200 ease-out flex-shrink-0 {expanded ? 'rotate-180' : ''}">
				<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="6 9 12 15 18 9"></polyline>
				</svg>
			</span>
		</button>
		
		{#if expanded}
			<div 
				class="border-t border-green-500/20 p-2 space-y-1 max-h-64 overflow-y-auto"
				transition:slide={{ duration: 200, easing: cubicOut }}
			>
				{#each $recentCompletions as agent (agent.id)}
					<button
						type="button"
						class="flex w-full items-center gap-2 rounded px-2 py-1.5 text-left text-sm transition-colors
							hover:bg-accent/50 focus:outline-none focus-visible:ring-1 focus-visible:ring-primary"
						onclick={() => handleClick(agent)}
					>
						<Tooltip.Root>
							<Tooltip.Trigger>
								{#snippet child({ props })}
									<span {...props} class="text-sm flex-shrink-0 {getOutcomeColor(agent.synthesis?.outcome)}">
										{getOutcomeEmoji(agent.synthesis?.outcome)}
									</span>
								{/snippet}
							</Tooltip.Trigger>
							<Tooltip.Content>
								<p>Outcome: {agent.synthesis?.outcome || 'completed'}</p>
							</Tooltip.Content>
						</Tooltip.Root>
						
						<span class="flex-1 truncate text-xs" title={agent.synthesis?.tldr || agent.task || agent.id}>
							{getDisplayText(agent)}
						</span>
						
						{#if agent.project}
							<Badge variant="secondary" class="h-4 px-1 text-[10px] flex-shrink-0">
								{agent.project}
							</Badge>
						{/if}
						
						{#if agent.skill}
							<Badge variant="outline" class="h-4 px-1 text-[10px] flex-shrink-0">
								{agent.skill}
							</Badge>
						{/if}
						
						<span class="text-[10px] text-muted-foreground flex-shrink-0">
							{formatTimeAgo(agent.completed_at || agent.updated_at)}
						</span>
					</button>
				{/each}
			</div>
		{/if}
	</div>
{/if}
