<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { MarkdownContent } from '$lib/components/markdown-content';

	export let beadsId: string;

	interface CompletionCommit {
		hash: string;
		subject: string;
		author: string;
		date: string;
	}

	interface CompletionDetailsResponse {
		beads_id: string;
		completion_message?: string;
		completion_date?: string;
		commits: CompletionCommit[];
		artifacts: string[];
		workspace_name?: string;
		error?: string;
	}

	let loading = true;
	let error: string | null = null;
	let details: CompletionDetailsResponse | null = null;

	// Fetch completion details from API
	async function fetchCompletionDetails() {
		loading = true;
		error = null;

		try {
			const response = await fetch(`https://localhost:3348/api/beads/${beadsId}/completion`);

			if (!response.ok) {
				throw new Error(`Failed to fetch completion details: ${response.statusText}`);
			}

			const data: CompletionDetailsResponse = await response.json();

			if (data.error) {
				error = data.error;
				details = null;
			} else {
				details = data;
			}
		} catch (e) {
			error = String(e);
			details = null;
		} finally {
			loading = false;
		}
	}

	// Fetch when beadsId changes
	$: if (beadsId) {
		fetchCompletionDetails();
	}

	// Format date as relative time
	function formatRelativeTime(dateStr: string): string {
		try {
			const date = new Date(dateStr);
			const now = new Date();
			const diffMs = now.getTime() - date.getTime();
			const diffMins = Math.floor(diffMs / 60000);
			const diffHours = Math.floor(diffMs / 3600000);
			const diffDays = Math.floor(diffMs / 86400000);

			if (diffMins < 1) return 'just now';
			if (diffMins < 60) return `${diffMins}m ago`;
			if (diffHours < 24) return `${diffHours}h ago`;
			if (diffDays < 7) return `${diffDays}d ago`;

			return date.toLocaleDateString();
		} catch {
			return dateStr;
		}
	}

	// Format commit date
	function formatCommitDate(dateStr: string): string {
		try {
			const date = new Date(dateStr);
			return date.toLocaleString(undefined, {
				month: 'short',
				day: 'numeric',
				hour: '2-digit',
				minute: '2-digit'
			});
		} catch {
			return dateStr;
		}
	}

	// Check if there's any completion data to show
	$: hasData = details && (
		details.completion_message ||
		(details.commits && details.commits.length > 0) ||
		(details.artifacts && details.artifacts.length > 0)
	);
</script>

<div class="completion-details">
	{#if loading}
		<div class="flex items-center justify-center py-4">
			<div class="text-sm text-muted-foreground">Loading completion details...</div>
		</div>
	{:else if error}
		<div class="text-sm text-muted-foreground py-4">
			Could not load completion details
		</div>
	{:else if !hasData}
		<div class="text-sm text-muted-foreground py-4 italic">
			No completion details available yet
		</div>
	{:else if details}
		<div class="space-y-5">
			<!-- Completion Message -->
			{#if details.completion_message}
				<div>
					<div class="flex items-center gap-2 mb-2">
						<h4 class="text-xs font-semibold text-muted-foreground uppercase tracking-wide">Completion Summary</h4>
						{#if details.completion_date}
							<span class="text-xs text-muted-foreground">
								{formatRelativeTime(details.completion_date)}
							</span>
						{/if}
					</div>
					<div class="bg-muted/50 rounded-md p-3 text-sm">
						<MarkdownContent content={details.completion_message} />
					</div>
				</div>
			{/if}

			<!-- Commits -->
			{#if details.commits && details.commits.length > 0}
				<div>
					<h4 class="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2">
						Commits ({details.commits.length})
					</h4>
					<div class="space-y-2">
						{#each details.commits as commit}
							<div class="flex items-start gap-2 text-sm">
								<code class="text-xs font-mono text-blue-500 bg-muted px-1.5 py-0.5 rounded shrink-0">
									{commit.hash}
								</code>
								<div class="flex-1 min-w-0">
									<div class="text-foreground truncate">{commit.subject}</div>
									<div class="text-xs text-muted-foreground">
										{commit.author} &middot; {formatCommitDate(commit.date)}
									</div>
								</div>
							</div>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Artifacts -->
			{#if details.artifacts && details.artifacts.length > 0}
				<div>
					<h4 class="text-xs font-semibold text-muted-foreground uppercase tracking-wide mb-2">
						Artifacts ({details.artifacts.length})
					</h4>
					<ul class="space-y-1">
						{#each details.artifacts as artifact}
							<li class="flex items-center gap-2 text-sm">
								<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-muted-foreground shrink-0">
									<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
									<polyline points="14 2 14 8 20 8" />
								</svg>
								<span class="font-mono text-xs text-foreground truncate">{artifact}</span>
							</li>
						{/each}
					</ul>
				</div>
			{/if}

			<!-- Workspace -->
			{#if details.workspace_name}
				<div class="text-xs text-muted-foreground pt-1 border-t border-border">
					Workspace: <span class="font-mono">{details.workspace_name}</span>
				</div>
			{/if}
		</div>
	{/if}
</div>
