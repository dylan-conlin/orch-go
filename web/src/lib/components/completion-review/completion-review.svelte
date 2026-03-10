<script lang="ts">
	import { formatRelativeTime } from '$lib/stores/attention';
	import { daemon, type DaemonStatus } from '$lib/stores/daemon';
	import type { ReadyToCompleteItem } from './completion-review-types';

	let {
		items,
		apiBase,
		onRefresh,
	}: {
		items: ReadyToCompleteItem[];
		apiBase: string;
		onRefresh: () => void;
	} = $props();

	let expandedItems = new Set<string>();
	let acknowledging = new Set<string>();
	let acknowledgingAll = false;

	let safeItems = $derived(items.filter(item => item.escalation === 'safe'));
	let reviewItems = $derived(items.filter(item => item.escalation !== 'safe'));

	function toggleExpand(id: string) {
		if (expandedItems.has(id)) {
			expandedItems = new Set([...expandedItems].filter(i => i !== id));
		} else {
			expandedItems = new Set([...expandedItems, id]);
		}
	}

	function formatTokenTotal(total: number | null): string {
		if (total === null) return 'tokens unknown';
		if (total >= 1_000_000) return `${(total / 1_000_000).toFixed(1)}M tokens`;
		if (total >= 1_000) return `${(total / 1_000).toFixed(1)}k tokens`;
		return `${total} tokens`;
	}

	async function acknowledgeItem(beadsId: string): Promise<void> {
		if (acknowledging.has(beadsId)) return;
		acknowledging = new Set([...acknowledging, beadsId]);
		try {
			const response = await fetch(`${apiBase}/api/issues/close`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					beads_id: beadsId,
					reason: 'Acknowledged via dashboard completion review',
				}),
			});
			const data = await response.json();
			if (!data.success) {
				console.error(`Failed to close ${beadsId}:`, data.error);
			}
		} catch (err) {
			console.error(`Failed to acknowledge ${beadsId}:`, err);
		} finally {
			acknowledging = new Set([...acknowledging].filter(id => id !== beadsId));
			onRefresh();
		}
	}

	async function acknowledgeAll(): Promise<void> {
		if (acknowledgingAll || safeItems.length === 0) return;
		acknowledgingAll = true;
		try {
			const ids = safeItems.map(item => item.id);
			const response = await fetch(`${apiBase}/api/issues/close-batch`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					beads_ids: ids,
					reason: 'Batch acknowledged via dashboard completion review',
				}),
			});
			const data = await response.json();
			if (data.total_failed > 0) {
				const failed = data.results.filter((r: { success: boolean }) => !r.success);
				console.error('Some issues failed to close:', failed);
			}
		} catch (err) {
			console.error('Failed to batch acknowledge:', err);
		} finally {
			acknowledgingAll = false;
			onRefresh();
		}
	}

	async function resumeDaemon(): Promise<void> {
		try {
			await fetch(`${apiBase}/api/daemon/resume`, { method: 'POST' });
			await daemon.fetch();
		} catch (err) {
			console.error('Failed to resume daemon:', err);
		}
	}
</script>

<!-- Daemon Paused Banner -->
{#if $daemon?.verification?.is_paused}
	<div
		class="mx-2 mt-2 rounded-md border border-amber-500/40 bg-amber-500/10 px-3 py-2 flex items-center justify-between"
		data-testid="daemon-paused-banner"
	>
		<div class="flex items-center gap-2">
			<span class="text-amber-400 text-sm">⏸</span>
			<span class="text-sm text-amber-300">Daemon paused — {$daemon.verification.completions_since_verification} completions awaiting review</span>
		</div>
		<div class="flex items-center gap-2">
			{#if safeItems.length > 0}
				<button
					type="button"
					onclick={acknowledgeAll}
					disabled={acknowledgingAll}
					class="px-2.5 py-1 text-xs font-medium text-amber-200 border border-amber-500/40 rounded hover:bg-amber-500/20 transition-colors disabled:opacity-50"
					data-testid="acknowledge-all-button"
				>
					{acknowledgingAll ? 'Closing...' : `Close Safe (${safeItems.length})`}
				</button>
			{/if}
			<button
				type="button"
				onclick={resumeDaemon}
				class="px-2.5 py-1 text-xs font-medium text-emerald-300 border border-emerald-500/40 rounded hover:bg-emerald-500/20 transition-colors"
				data-testid="resume-daemon-button"
			>
				Resume
			</button>
		</div>
	</div>
{/if}

<div
	class="mx-2 {$daemon?.verification?.is_paused ? 'mt-1' : 'mt-2'} mb-2"
	data-testid="ready-to-complete-section"
>
	<!-- Section Header -->
	<div class="px-3 py-2 flex items-center justify-between gap-4">
		<div class="text-sm font-semibold text-emerald-400">Ready to Complete</div>
		<span class="text-xs text-muted-foreground">
			{items.length} awaiting review{#if reviewItems.length > 0} · {reviewItems.length} need{reviewItems.length === 1 ? 's' : ''} attention{/if}
		</span>
	</div>

	<div class="max-h-64 overflow-y-auto space-y-2">
		<!-- Needs Review Group -->
		{#if reviewItems.length > 0}
			<div class="rounded-md border border-amber-500/30 bg-amber-500/5" data-testid="needs-review-group">
				<div class="px-3 py-1.5 border-b border-amber-500/20 flex items-center justify-between">
					<span class="text-xs font-medium text-amber-400">Needs Review</span>
					<span class="text-xs text-amber-300/60">{reviewItems.length}</span>
				</div>
				{#each reviewItems as item (item.id)}
					<div
						class="px-3 py-2 border-b border-amber-500/10 last:border-b-0"
						data-testid={`ready-to-complete-row-${item.id}`}
					>
						<!-- Line 1: Outcome badge + TLDR -->
						<div class="flex items-start gap-2">
							{#if item.outcome}
								<span class="flex-shrink-0 mt-0.5 {item.outcome === 'failed' ? 'text-red-400' : item.outcome === 'partial' || item.outcome === 'blocked' ? 'text-amber-400' : 'text-green-400'} text-xs font-medium">
									{#if item.outcome === 'failed'}✗ failed{:else if item.outcome === 'partial'}~ partial{:else if item.outcome === 'blocked'}⊘ blocked{:else}✓ {item.outcome}{/if}
								</span>
							{/if}
							<span class="text-sm text-foreground flex-1">{item.tldr || item.title}</span>
							<button
								type="button"
								onclick={() => acknowledgeItem(item.id)}
								disabled={acknowledging.has(item.id) || acknowledgingAll}
								class="px-2 py-0.5 text-xs font-medium text-amber-300 border border-amber-500/30 rounded hover:bg-amber-500/20 transition-colors disabled:opacity-50 flex-shrink-0"
								data-testid={`acknowledge-button-${item.id}`}
							>
								{acknowledging.has(item.id) ? '...' : 'Close'}
							</button>
						</div>
						<!-- Line 2: Metadata -->
						<div class="mt-1 ml-[1.25rem] flex items-center gap-2 text-xs text-muted-foreground flex-wrap">
							{#if item.skill}<span>{item.skill}</span><span class="text-muted-foreground/40">·</span>{/if}
							<span class="font-mono">{item.id}</span>
							{#if item.deltaSummary}<span class="text-muted-foreground/40">·</span><span>{item.deltaSummary}</span>{/if}
							{#if item.runtime}<span class="text-muted-foreground/40">·</span><span>{item.runtime}</span>{/if}
							<span class="text-muted-foreground/40">·</span>
							<span>{formatRelativeTime(item.completionAt)}</span>
						</div>
						<!-- Line 3: Expandable next_actions -->
						{#if item.nextActions && item.nextActions.length > 0}
							<button
								type="button"
								onclick={() => toggleExpand(item.id)}
								class="mt-1 ml-[1.25rem] text-xs text-amber-300/70 hover:text-amber-300 flex items-center gap-1"
							>
								<span class="inline-block transition-transform {expandedItems.has(item.id) ? 'rotate-90' : ''}" style="font-size: 0.6em">▶</span>
								{item.nextActions.length} follow-up action{item.nextActions.length !== 1 ? 's' : ''}
							</button>
							{#if expandedItems.has(item.id)}
								<div class="mt-1 ml-[1.25rem] pl-3 border-l border-amber-500/20 space-y-1">
									{#each item.nextActions as action}
										<div class="text-xs text-muted-foreground">• {action}</div>
									{/each}
									{#if item.recommendation && item.recommendation !== 'close'}
										<div class="text-xs text-amber-300/80 mt-1">Recommendation: {item.recommendation}</div>
									{/if}
								</div>
							{/if}
						{/if}
					</div>
				{/each}
			</div>
		{/if}

		<!-- Safe to Close Group -->
		{#if safeItems.length > 0}
			<div class="rounded-md border border-emerald-500/30 bg-emerald-500/5" data-testid="safe-to-close-group">
				<div class="px-3 py-1.5 border-b border-emerald-500/20 flex items-center justify-between">
					<span class="text-xs font-medium text-emerald-400">Safe to Close</span>
					<div class="flex items-center gap-2">
						<span class="text-xs text-emerald-300/60">{safeItems.length}</span>
						{#if safeItems.length > 1}
							<button
								type="button"
								onclick={acknowledgeAll}
								disabled={acknowledgingAll}
								class="px-2 py-0.5 text-xs font-medium text-emerald-300 border border-emerald-500/30 rounded hover:bg-emerald-500/20 transition-colors disabled:opacity-50"
								data-testid="acknowledge-all-compact-button"
							>
								{acknowledgingAll ? 'Closing...' : 'Close All'}
							</button>
						{/if}
					</div>
				</div>
				{#each safeItems as item (item.id)}
					<div
						class="px-3 py-2 border-b border-emerald-500/10 last:border-b-0"
						data-testid={`ready-to-complete-row-${item.id}`}
					>
						<!-- Line 1: Outcome badge + TLDR -->
						<div class="flex items-start gap-2">
							<span class="flex-shrink-0 mt-0.5 text-emerald-400 text-xs font-medium">✓ success</span>
							<span class="text-sm text-foreground flex-1">{item.tldr || item.title}</span>
							<button
								type="button"
								onclick={() => acknowledgeItem(item.id)}
								disabled={acknowledging.has(item.id) || acknowledgingAll}
								class="px-2 py-0.5 text-xs font-medium text-emerald-300 border border-emerald-500/30 rounded hover:bg-emerald-500/20 transition-colors disabled:opacity-50 flex-shrink-0"
								data-testid={`acknowledge-button-${item.id}`}
							>
								{acknowledging.has(item.id) ? '...' : 'Close'}
							</button>
						</div>
						<!-- Line 2: Metadata -->
						<div class="mt-1 ml-[1.25rem] flex items-center gap-2 text-xs text-muted-foreground flex-wrap">
							{#if item.skill}<span>{item.skill}</span><span class="text-muted-foreground/40">·</span>{/if}
							<span class="font-mono">{item.id}</span>
							{#if item.deltaSummary}<span class="text-muted-foreground/40">·</span><span>{item.deltaSummary}</span>{/if}
							{#if item.runtime}<span class="text-muted-foreground/40">·</span><span>{item.runtime}</span>{/if}
							<span class="text-muted-foreground/40">·</span>
							<span>{formatRelativeTime(item.completionAt)}</span>
						</div>
						<!-- Expandable next_actions for safe items too -->
						{#if item.nextActions && item.nextActions.length > 0}
							<button
								type="button"
								onclick={() => toggleExpand(item.id)}
								class="mt-1 ml-[1.25rem] text-xs text-emerald-300/70 hover:text-emerald-300 flex items-center gap-1"
							>
								<span class="inline-block transition-transform {expandedItems.has(item.id) ? 'rotate-90' : ''}" style="font-size: 0.6em">▶</span>
								{item.nextActions.length} follow-up action{item.nextActions.length !== 1 ? 's' : ''}
							</button>
							{#if expandedItems.has(item.id)}
								<div class="mt-1 ml-[1.25rem] pl-3 border-l border-emerald-500/20 space-y-1">
									{#each item.nextActions as action}
										<div class="text-xs text-muted-foreground">• {action}</div>
									{/each}
								</div>
							{/if}
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>
