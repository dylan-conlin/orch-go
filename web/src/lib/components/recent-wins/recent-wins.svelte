<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { agents, type Agent } from '$lib/stores/agents';
	import { derived } from 'svelte/store';

	// Recent wins: completed agents in last 24 hours
	const RECENT_THRESHOLD_MS = 24 * 60 * 60 * 1000; // 24 hours

	// Derive recent wins from agents store
	const recentWins = derived(agents, ($agents) => {
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
			const aTime = a.completed_at ? new Date(a.completed_at).getTime() : 0;
			const bTime = b.completed_at ? new Date(b.completed_at).getTime() : 0;
			return bTime - aTime;
		});
	});

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
				return '✅';
		}
	}

	function truncateText(text: string, maxLen: number = 60): string {
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
		return agent.id;
	}
</script>

{#if $recentWins.length > 0}
	<div class="rounded-lg border border-green-500/30 bg-green-500/5" data-testid="recent-wins-section">
		<div class="flex items-center gap-2 px-3 py-2 border-b">
			<span class="text-sm">🏆</span>
			<span class="text-sm font-medium">Recent Wins</span>
			<Badge variant="secondary" class="h-5 px-1.5 text-xs">
				{$recentWins.length}
			</Badge>
			<span class="text-xs text-muted-foreground">last 24h</span>
		</div>
		<div class="p-2 space-y-1 max-h-48 overflow-y-auto">
			{#each $recentWins as agent (agent.id)}
				<div class="flex items-center gap-2 rounded px-2 py-1.5 text-sm hover:bg-accent/50">
					<span class="text-sm flex-shrink-0">{getOutcomeEmoji(agent.synthesis?.outcome)}</span>
					<span class="flex-1 truncate text-xs" title={agent.synthesis?.tldr || agent.task || agent.id}>
						{getDisplayText(agent)}
					</span>
					{#if agent.project}
						<Badge variant="secondary" class="h-4 px-1 text-[10px] flex-shrink-0">
							{agent.project}
						</Badge>
					{/if}
					<span class="text-[10px] text-muted-foreground flex-shrink-0">
						{formatTimeAgo(agent.completed_at || agent.updated_at)}
					</span>
				</div>
			{/each}
		</div>
	</div>
{/if}
