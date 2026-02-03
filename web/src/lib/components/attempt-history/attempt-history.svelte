<script lang="ts">
	import { onMount } from 'svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { orchestratorContext } from '$lib/stores/context';

	export let beadsId: string;

	interface AttemptHistoryEntry {
		attempt_number: number;
		timestamp: string;
		outcome: string;
		phase: string;
		artifacts: string[];
		workspace_name: string;
	}

	interface AttemptHistoryResponse {
		beads_id: string;
		attempts: AttemptHistoryEntry[];
		count: number;
		error?: string;
	}

	let loading = true;
	let error: string | null = null;
	let attempts: AttemptHistoryEntry[] = [];

	// Fetch attempt history from API
	async function fetchAttemptHistory() {
		loading = true;
		error = null;

		try {
			const response = await fetch(`http://localhost:3348/api/beads/${beadsId}/attempts`);
			
			if (!response.ok) {
				throw new Error(`Failed to fetch attempt history: ${response.statusText}`);
			}

			const data: AttemptHistoryResponse = await response.json();

			if (data.error) {
				error = data.error;
				attempts = [];
			} else {
				attempts = data.attempts || [];
			}
		} catch (e) {
			error = String(e);
			attempts = [];
		} finally {
			loading = false;
		}
	}

	// Fetch on mount and when beadsId changes
	$: if (beadsId) {
		fetchAttemptHistory();
	}

	// Get outcome badge variant
	function getOutcomeBadge(outcome: string): { variant: 'default' | 'secondary' | 'destructive' | 'outline'; text: string } {
		switch (outcome.toLowerCase()) {
			case 'success':
				return { variant: 'default', text: '✓ Success' };
			case 'failed':
				return { variant: 'destructive', text: '✗ Failed' };
			case 'died':
				return { variant: 'destructive', text: '💀 Died' };
			case 'closed→reopened':
				return { variant: 'secondary', text: '↻ Reopened' };
			case 'in_progress':
				return { variant: 'outline', text: '▶ In Progress' };
			default:
				return { variant: 'outline', text: outcome };
		}
	}

	// Format timestamp as relative time
	function formatRelativeTime(timestamp: string): string {
		try {
			const date = new Date(timestamp);
			const now = new Date();
			const diffMs = now.getTime() - date.getTime();
			const diffMins = Math.floor(diffMs / 60000);
			const diffHours = Math.floor(diffMs / 3600000);
			const diffDays = Math.floor(diffMs / 86400000);

			if (diffMins < 1) return 'just now';
			if (diffMins < 60) return `${diffMins}m ago`;
			if (diffHours < 24) return `${diffHours}h ago`;
			if (diffDays < 7) return `${diffDays}d ago`;
			
			// Fall back to date string
			return date.toLocaleDateString();
		} catch (e) {
			return timestamp;
		}
	}
</script>

<div class="attempt-history">
	{#if loading}
		<div class="flex items-center justify-center py-4">
			<div class="text-sm text-muted-foreground">Loading attempt history...</div>
		</div>
	{:else if error}
		<div class="text-sm text-red-500 py-4">
			Error loading attempt history: {error}
		</div>
	{:else if attempts.length === 0}
		<div class="text-sm text-muted-foreground py-4">
			No attempts recorded yet
		</div>
	{:else}
		<div class="space-y-4">
			{#each attempts as attempt}
				<div class="border-l-2 border-border pl-4 py-2 relative">
					<!-- Timeline dot -->
					<div class="absolute left-[-5px] top-3 w-2 h-2 rounded-full bg-border"></div>
					
					<!-- Attempt header -->
					<div class="flex items-center justify-between mb-2">
						<div class="flex items-center gap-2">
							<span class="text-sm font-semibold text-foreground">
								Attempt #{attempt.attempt_number}
							</span>
							<Badge variant={getOutcomeBadge(attempt.outcome).variant}>
								{getOutcomeBadge(attempt.outcome).text}
							</Badge>
							{#if attempt.phase}
								<span class="text-xs text-muted-foreground">
									Phase: {attempt.phase}
								</span>
							{/if}
						</div>
						<span class="text-xs text-muted-foreground">
							{formatRelativeTime(attempt.timestamp)}
						</span>
					</div>
					
					<!-- Workspace name -->
					<div class="text-xs text-muted-foreground mb-1">
						Workspace: {attempt.workspace_name}
					</div>
					
					<!-- Artifacts -->
					{#if attempt.artifacts && attempt.artifacts.length > 0}
						<div class="mt-2">
							<div class="text-xs font-medium text-muted-foreground mb-1">
								Artifacts produced:
							</div>
							<ul class="text-xs text-muted-foreground space-y-1">
								{#each attempt.artifacts as artifact}
									<li class="pl-2">
										<span class="text-foreground font-mono">
											{artifact}
										</span>
									</li>
								{/each}
							</ul>
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>
