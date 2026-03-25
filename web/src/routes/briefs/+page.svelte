<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { derived } from 'svelte/store';
	import { briefs, type BriefListItem } from '$lib/stores/briefs';
	import { orchestratorContext } from '$lib/stores/context';
	import { MarkdownContent } from '$lib/components/markdown-content';
	import type { BriefResponse } from '$lib/stores/beads';

	let pollInterval: ReturnType<typeof setInterval> | null = null;
	let filter: 'all' | 'unread' | 'read' = 'all';
	let expandedId: string | null = null;
	let briefCache: Map<string, BriefResponse> = new Map();
	let briefLoading: Set<string> = new Set();
	let currentProjectDir: string | undefined;

	// Derived store for project_dir to isolate reactivity
	const projectDir = derived(orchestratorContext, $ctx => $ctx.project_dir);

	$: unreadCount = $briefs.filter(b => !b.marked_read).length;
	$: readCount = $briefs.filter(b => b.marked_read).length;

	$: filtered = $briefs.filter(b => {
		if (filter === 'unread') return !b.marked_read;
		if (filter === 'read') return b.marked_read;
		return true;
	});

	// Re-fetch briefs when project_dir changes
	$: {
		if (typeof window !== 'undefined' && $projectDir && $projectDir !== currentProjectDir) {
			currentProjectDir = $projectDir;
			// Clear cache when switching projects
			briefCache = new Map();
			expandedId = null;
			briefs.fetch($projectDir);
		}
	}

	async function toggleExpand(item: BriefListItem) {
		if (expandedId === item.beads_id) {
			expandedId = null;
			return;
		}

		if (!briefCache.has(item.beads_id)) {
			briefLoading.add(item.beads_id);
			briefLoading = briefLoading;
			const data = await briefs.fetchBrief(item.beads_id, $projectDir);
			if (data) {
				briefCache.set(item.beads_id, data);
				briefCache = briefCache;
			}
			briefLoading.delete(item.beads_id);
			briefLoading = briefLoading;
		}

		expandedId = item.beads_id;
	}

	async function markAsRead(beadsId: string, event: MouseEvent) {
		event.stopPropagation();
		const success = await briefs.markAsRead(beadsId, $projectDir);
		if (success) {
			const cached = briefCache.get(beadsId);
			if (cached) {
				briefCache.set(beadsId, { ...cached, marked_read: true });
				briefCache = briefCache;
			}
		}
	}

	onMount(async () => {
		orchestratorContext.startPolling(10_000);
		const dir = $orchestratorContext?.project_dir;
		currentProjectDir = dir;
		await briefs.fetch(dir);
		pollInterval = setInterval(() => briefs.fetch($projectDir), 30_000);
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
	});
</script>

<div class="space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-lg font-semibold">Briefs</h1>
			<p class="text-xs text-muted-foreground">
				Reading queue — completion briefs from agent work
			</p>
		</div>
		<div class="flex items-center gap-3 text-xs">
			<span class="text-muted-foreground" data-testid="briefs-stats">
				{unreadCount} unread · {readCount} read · {$briefs.length} total
			</span>
			{#if $orchestratorContext?.project_dir}
				<span class="text-muted-foreground truncate max-w-xs">
					{$orchestratorContext.project_dir.split('/').pop()}
				</span>
			{/if}
		</div>
	</div>

	<div class="flex items-center gap-0.5 rounded-md border p-0.5" data-testid="briefs-filter">
		<button
			class="px-2 py-1 text-xs rounded transition-colors {filter === 'all' ? 'bg-foreground/10 text-foreground font-medium' : 'text-muted-foreground hover:text-foreground'}"
			onclick={() => { filter = 'all'; }}
			data-testid="filter-all"
		>All</button>
		<button
			class="px-2 py-1 text-xs rounded transition-colors {filter === 'unread' ? 'bg-blue-500/20 text-blue-400 font-medium' : 'text-muted-foreground hover:text-foreground'}"
			onclick={() => { filter = 'unread'; }}
			data-testid="filter-unread"
		>Unread{#if unreadCount > 0}<span class="ml-1 text-[10px]">({unreadCount})</span>{/if}</button>
		<button
			class="px-2 py-1 text-xs rounded transition-colors {filter === 'read' ? 'bg-emerald-500/20 text-emerald-400 font-medium' : 'text-muted-foreground hover:text-foreground'}"
			onclick={() => { filter = 'read'; }}
			data-testid="filter-read"
		>Read{#if readCount > 0}<span class="ml-1 text-[10px]">({readCount})</span>{/if}</button>
	</div>

	{#if filtered.length === 0}
		<div class="rounded-md border p-8 text-center" data-testid="briefs-empty">
			<p class="text-sm text-muted-foreground">
				{#if filter !== 'all'}
					No {filter} briefs.
				{:else}
					No briefs yet. Briefs are produced when agents complete work.
				{/if}
			</p>
		</div>
	{:else}
		<div class="space-y-1.5" data-testid="briefs-list">
			{#each filtered as item (item.beads_id)}
				{@const brief = briefCache.get(item.beads_id)}
				{@const isExpanded = expandedId === item.beads_id}
				{@const isLoading = briefLoading.has(item.beads_id)}
				<div
					class="rounded-md border border-border
						{isExpanded ? 'border-foreground/10' : ''}"
					data-testid="brief-item-{item.beads_id}"
				>
					<button
						class="flex w-full items-center justify-between gap-2 px-3 py-2.5 text-left cursor-pointer"
						onclick={() => toggleExpand(item)}
					>
						<div class="flex items-center gap-2 min-w-0 flex-1">
							{#if !item.marked_read}
								<span class="h-1.5 w-1.5 rounded-full bg-blue-500 flex-shrink-0"></span>
							{/if}
							<span class="text-sm font-mono truncate">{item.beads_id}</span>
							{#if isLoading}
								<span class="text-xs text-muted-foreground">loading...</span>
							{/if}
						</div>
						<div class="flex items-center gap-1.5 flex-shrink-0">
							{#if item.marked_read}
								<span class="px-1.5 py-0.5 text-[10px] rounded bg-emerald-500/20 text-emerald-400">read</span>
							{:else}
								<span class="px-1.5 py-0.5 text-[10px] rounded bg-blue-500/20 text-blue-400">unread</span>
							{/if}
						</div>
					</button>

					{#if isExpanded && brief}
						<div class="px-3 pb-3 pt-0 border-t space-y-2" data-testid="brief-content-{item.beads_id}">
							<div class="flex items-center justify-between pt-2">
								<span class="text-xs font-medium text-blue-400">Brief</span>
								{#if !brief.marked_read}
									<button
										class="text-xs px-2 py-0.5 rounded border border-emerald-500/40 text-emerald-500 hover:bg-emerald-500/10 transition-colors"
										onclick={(e) => markAsRead(item.beads_id, e)}
										data-testid="mark-read-{item.beads_id}"
									>
										Mark as read
									</button>
								{:else}
									<span class="text-xs text-emerald-500/60">Read</span>
								{/if}
							</div>
							<div class="prose prose-sm dark:prose-invert max-w-none">
								<MarkdownContent content={brief.content} />
							</div>
						</div>
					{/if}
				</div>
			{/each}
		</div>
	{/if}
</div>
