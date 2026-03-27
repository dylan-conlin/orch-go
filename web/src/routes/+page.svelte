<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { Badge } from '$lib/components/ui/badge';
	import { ReviewQueueSection } from '$lib/components/review-queue-section';
	import { QuestionsSection } from '$lib/components/questions-section';
	import {
		agents,
		trulyActiveAgents,
		needsReviewAgents,
		connectSSE,
		disconnectSSE,
		connectionStatus,
		setFilterQueryStringCallback
	} from '$lib/stores/agents';
	import { threads, threadDetail } from '$lib/stores/threads';
	import { briefs, type BriefListItem } from '$lib/stores/briefs';
	import { beads, readyIssues, reviewQueue } from '$lib/stores/beads';
	import type { BriefResponse } from '$lib/stores/beads';
	import { questions } from '$lib/stores/questions';
	import { filters, orchestratorContext, buildFilterQueryString } from '$lib/stores/context';

	// Thread expansion state
	let expandedThread: string | null = null;
	let homeBriefCache = new Map<string, BriefResponse>();
	let homeBriefLoadKey = '';

	const HOME_BRIEF_LIMIT = 2;
	const HOME_QUESTION_LIMIT = 3;

	type BriefSectionPreview = {
		label: string;
		text: string;
	};

	function toggleThread(slug: string) {
		if (expandedThread === slug) {
			expandedThread = null;
			threadDetail.clear();
		} else {
			expandedThread = slug;
			threadDetail.fetch(slug);
		}
	}

	function getStatusColor(status: string): string {
		switch (status) {
			case 'forming': return 'text-blue-500';
			case 'active': return 'text-green-500';
			case 'converged': return 'text-purple-500';
			case 'resolved': return 'text-muted-foreground';
			default: return 'text-foreground';
		}
	}

	function getStatusLabel(status: string): string {
		switch (status) {
			case 'forming': return 'forming';
			case 'active': return 'active';
			case 'converged': return 'converged';
			case 'resolved': return 'resolved';
			case 'subsumed': return 'subsumed';
			default: return status;
		}
	}

	function collapseWhitespace(value: string): string {
		return value.replace(/\s+/g, ' ').trim();
	}

	function trimPreview(value: string, maxLength: number): string {
		const collapsed = collapseWhitespace(value);
		if (collapsed.length <= maxLength) return collapsed;
		const trimmed = collapsed.slice(0, maxLength);
		const breakpoint = Math.max(trimmed.lastIndexOf('. '), trimmed.lastIndexOf('; '), trimmed.lastIndexOf(', '), trimmed.lastIndexOf(' '));
		return `${trimmed.slice(0, breakpoint > 60 ? breakpoint : maxLength).trimEnd()}...`;
	}

	function extractBriefSection(content: string, heading: string): string {
		const escapedHeading = heading.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
		const match = content.match(new RegExp(`(?:^|\\n)## ${escapedHeading}\\s*\\n([\\s\\S]*?)(?=\\n## |\\n# |$)`));
		return match?.[1] ? collapseWhitespace(match[1]) : '';
	}

	function getBriefPreviewSections(content: string): BriefSectionPreview[] {
		return [
			{ label: 'Frame', text: trimPreview(extractBriefSection(content, 'Frame'), 150) },
			{ label: 'Resolution', text: trimPreview(extractBriefSection(content, 'Resolution'), 170) },
			{ label: 'Tension', text: trimPreview(extractBriefSection(content, 'Tension'), 140) }
		].filter((section) => section.text.length > 0);
	}

	function getHomeBriefCandidates(items: BriefListItem[]): BriefListItem[] {
		const unread = items.filter((item) => !item.marked_read);
		const source = unread.length > 0 ? unread : items;
		return source.slice(0, HOME_BRIEF_LIMIT);
	}

	async function hydrateHomeBriefs(items: BriefListItem[], projectDir?: string) {
		const results = await Promise.all(
			items.map(async (item) => {
				if (homeBriefCache.has(item.beads_id)) {
					return [item.beads_id, homeBriefCache.get(item.beads_id)] as const;
				}
				const brief = await briefs.fetchBrief(item.beads_id, projectDir);
				return [item.beads_id, brief] as const;
			})
		);

		const nextCache = new Map(homeBriefCache);
		for (const [beadsId, brief] of results) {
			if (brief) {
				nextCache.set(beadsId, brief);
			}
		}
		homeBriefCache = nextCache;
	}

	// Status bucket ordering: active thinking first, then forming, then converged
	function statusOrder(status: string): number {
		switch (status) {
			case 'active': return 0;
			case 'forming': return 1;
			case 'converged': return 2;
			case 'subsumed': return 3;
			case 'resolved': return 4;
			default: return 5;
		}
	}

	// Derived: active threads (non-resolved), sorted by status bucket then updated desc
	$: activeThreads = $threads
		.filter(t => t.status !== 'resolved')
		.sort((a, b) => {
			const bucketDiff = statusOrder(a.status) - statusOrder(b.status);
			if (bucketDiff !== 0) return bucketDiff;
			return b.updated.localeCompare(a.updated);
		});
	$: unreadBriefCount = $briefs.filter(b => !b.marked_read).length;
	$: homeBriefItems = getHomeBriefCandidates($briefs);
	$: featuredQuestions = [
		...($questions?.open || []),
		...($questions?.investigating || [])
	].slice(0, HOME_QUESTION_LIMIT);

	// Section collapse state with localStorage persistence
	const STORAGE_KEY = 'orch-dashboard-sections';
	let sectionState = {
		reviewQueue: true,
		questions: true
	};
	let sectionStateLoaded = false;

	function loadSectionState() {
		if (typeof window === 'undefined') return;
		try {
			const stored = localStorage.getItem(STORAGE_KEY);
			if (stored) {
				const parsed = JSON.parse(stored);
				// Only load keys we still use
				if ('reviewQueue' in parsed) sectionState.reviewQueue = parsed.reviewQueue;
				if ('questions' in parsed) sectionState.questions = parsed.questions;
			}
		} catch (e) {
			console.warn('Failed to load section state:', e);
		}
		sectionStateLoaded = true;
	}

	function saveSectionState() {
		if (typeof window === 'undefined') return;
		if (!sectionStateLoaded) return;
		try {
			localStorage.setItem(STORAGE_KEY, JSON.stringify(sectionState));
		} catch (e) {
			console.warn('Failed to save section state:', e);
		}
	}

	$: if (sectionStateLoaded && sectionState) {
		saveSectionState();
	}

	onMount(() => {
		loadSectionState();

		// Start context SSE for real-time follow mode
		if ($filters.followOrchestrator) {
			orchestratorContext.startPolling();
		}

		// Set up filter query string callback for SSE-triggered fetches
		setFilterQueryStringCallback(() => buildFilterQueryString($filters));

		// Connect SSE for real-time agent counts
		connectSSE();

		// Fetch comprehension data in parallel
		Promise.all([
			threads.fetch(),
			briefs.fetch(),
			beads.fetch(),
			reviewQueue.fetch(),
			questions.fetch()
		]).catch(console.error);

		// Defer secondary data fetches
		const deferSecondaryFetches = () => {
			readyIssues.fetch();
		};

		if ('requestIdleCallback' in window) {
			requestIdleCallback(deferSecondaryFetches, { timeout: 2000 });
		} else {
			setTimeout(deferSecondaryFetches, 100);
		}

		// Refresh data every 60 seconds
		const refreshInterval = setInterval(() => {
			const projectDir = $filters.followOrchestrator ? $orchestratorContext.project_dir : undefined;
			Promise.all([
				threads.fetch(),
				briefs.fetch(projectDir),
				beads.fetch(projectDir),
				readyIssues.fetch(projectDir),
				reviewQueue.fetch(projectDir),
				questions.fetch()
			]).catch(console.error);
		}, 60000);

		const handleBeforeUnload = () => {
			disconnectSSE();
		};
		window.addEventListener('beforeunload', handleBeforeUnload);

		return () => {
			window.removeEventListener('beforeunload', handleBeforeUnload);
			clearInterval(refreshInterval);
		};
	});

	onDestroy(() => {
		disconnectSSE();
		orchestratorContext.stopPolling();
	});

	// React to followOrchestrator changes
	$: {
		if (typeof window !== 'undefined') {
			if ($filters.followOrchestrator) {
				orchestratorContext.startPolling();
			} else {
				orchestratorContext.stopPolling();
			}
		}
	}

	// Update project filter when orchestrator context changes
	$: {
		if ($filters.followOrchestrator && $orchestratorContext.project) {
			filters.setProjectFilter($orchestratorContext.project, $orchestratorContext.included_projects);
		}
	}

	// Refetch beads when orchestrator context changes
	$: {
		if (typeof window !== 'undefined' && $filters.followOrchestrator && $orchestratorContext.project_dir) {
			beads.fetch($orchestratorContext.project_dir).catch(console.error);
			readyIssues.fetch($orchestratorContext.project_dir).catch(console.error);
			reviewQueue.fetch($orchestratorContext.project_dir).catch(console.error);
		}
	}

	// Reactive query string for SSE-triggered agent refetches
	$: filterQueryString = buildFilterQueryString($filters);

	$: if (filterQueryString !== undefined && typeof window !== 'undefined') {
		if ($connectionStatus === 'connected') {
			agents.fetch(filterQueryString).catch(console.error);
		}
	}

	$: if (typeof window !== 'undefined') {
		const projectDir = $filters.followOrchestrator ? $orchestratorContext.project_dir : undefined;
		const nextKey = `${projectDir || ''}:${homeBriefItems.map((item) => item.beads_id).join(',')}`;
		if (homeBriefItems.length > 0 && nextKey !== homeBriefLoadKey) {
			homeBriefLoadKey = nextKey;
			void hydrateHomeBriefs(homeBriefItems, projectDir);
		}
	}
</script>

<div class="space-y-3">
	<!-- ═══════════════════════════════════════════════════════════ -->
	<!-- COMPREHENSION LAYER -->
	<!-- ═══════════════════════════════════════════════════════════ -->

	<!-- Active Threads — the thinking spine -->
	<div class="rounded-lg border bg-card" data-testid="threads-section">
		<div class="flex items-center justify-between border-b px-3 py-2">
			<div class="flex items-center gap-2">
				<span class="text-sm font-semibold">Threads</span>
				{#if activeThreads.length > 0}
					<Badge variant="default" class="h-5 px-1.5 text-xs">{activeThreads.length}</Badge>
				{/if}
			</div>
			{#if $threads.length > activeThreads.length}
				<span class="text-xs text-muted-foreground">{$threads.length - activeThreads.length} resolved</span>
			{/if}
		</div>
		<div class="p-2">
			{#if activeThreads.length > 0}
				<div class="space-y-1">
					{#each activeThreads as t (t.name)}
						<button
							class="w-full text-left rounded-md border px-3 py-2 transition-colors hover:bg-accent/50 {expandedThread === t.name ? 'bg-accent/30 border-foreground/20' : 'border-transparent'}"
							onclick={() => toggleThread(t.name)}
						>
							<div class="flex items-center justify-between gap-2">
								<div class="flex items-center gap-2 min-w-0">
									<svg
										class="h-3 w-3 flex-shrink-0 transition-transform duration-150 text-muted-foreground {expandedThread === t.name ? 'rotate-90' : ''}"
										viewBox="0 0 16 16"
										fill="currentColor"
									>
										<path d="M6.22 3.22a.75.75 0 0 1 1.06 0l4.25 4.25a.75.75 0 0 1 0 1.06l-4.25 4.25a.75.75 0 0 1-1.06-1.06L9.94 8 6.22 4.28a.75.75 0 0 1 0-1.06Z"/>
									</svg>
									<span class="inline-flex items-center rounded px-1.5 py-0.5 text-[10px] font-medium leading-none {getStatusColor(t.status)} bg-current/10"
										style="background-color: color-mix(in srgb, currentColor 10%, transparent);"
									>{getStatusLabel(t.status)}</span>
									<span class="text-sm font-medium truncate">{t.title}</span>
								</div>
								<div class="flex items-center gap-2 flex-shrink-0">
									<span class="text-xs text-muted-foreground">{t.entry_count} entries</span>
									<span class="text-xs text-muted-foreground">{t.updated}</span>
								</div>
							</div>
							{#if t.latest_entry}
								<p
									class="ml-5 mt-1 text-sm leading-5 text-muted-foreground"
									style="display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden;"
									data-testid="thread-inline-entry-{t.name}"
								>{t.latest_entry}</p>
							{/if}
						</button>
						{#if expandedThread === t.name && $threadDetail}
							<div class="ml-6 border-l-2 border-muted pl-3 py-1 space-y-2">
								{#each $threadDetail.entries.slice().reverse().slice(0, 5) as entry (entry.date + entry.text.slice(0, 20))}
									<div>
										<span class="text-xs font-mono text-muted-foreground">{entry.date}</span>
										<p class="text-xs mt-0.5 whitespace-pre-wrap">{entry.text.slice(0, 300)}{entry.text.length > 300 ? '...' : ''}</p>
									</div>
								{/each}
								{#if $threadDetail.entries.length > 5}
									<p class="text-xs text-muted-foreground">... {$threadDetail.entries.length - 5} more entries</p>
								{/if}
							</div>
						{/if}
					{/each}
				</div>
			{:else}
				<div class="rounded border border-dashed p-4 text-center">
					<p class="text-sm text-muted-foreground">No active threads</p>
					<p class="mt-1 text-xs text-muted-foreground">
						Create with <code class="rounded bg-muted px-1">orch thread new "question"</code>
					</p>
				</div>
			{/if}
		</div>
	</div>

	<!-- New Evidence — inline brief and question content -->
	<div class="grid gap-2 lg:grid-cols-[minmax(0,1.25fr)_minmax(0,0.95fr)]">
		<div class="rounded-lg border bg-card px-3 py-3" data-testid="home-briefs-card">
			<div class="flex items-center justify-between gap-3">
				<div>
					<p class="text-sm font-medium">Recent Briefs</p>
					<p class="text-xs text-muted-foreground">Frame, resolution, and tension from the latest work.</p>
				</div>
				<div class="flex items-center gap-2">
					{#if unreadBriefCount > 0}
						<a href="/briefs" class="inline-flex items-center gap-1">
							<Badge variant="default" class="h-5 px-1.5 text-xs">{unreadBriefCount} unread</Badge>
						</a>
					{/if}
					<a href="/briefs" class="text-xs font-medium text-foreground hover:underline">Open queue</a>
				</div>
			</div>

			{#if homeBriefItems.length > 0}
				<div class="mt-3 space-y-2">
					{#each homeBriefItems as item (item.beads_id)}
						{@const brief = homeBriefCache.get(item.beads_id)}
						<div class="rounded-md border border-border/70 bg-background/40 px-3 py-2" data-testid="home-brief-{item.beads_id}">
							<div class="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
								<span class="font-mono text-[11px] text-foreground">{item.beads_id}</span>
								{#if item.thread_title}
									<span>{item.thread_title}</span>
								{/if}
								{#if item.has_tension}
									<Badge variant="outline" class="h-5 px-1.5 text-[10px]">tension</Badge>
								{/if}
							</div>
							{#if brief}
								<div class="mt-2 space-y-2">
									{#each getBriefPreviewSections(brief.content) as section (section.label)}
										<div class="grid gap-1 sm:grid-cols-[5.25rem_minmax(0,1fr)]" data-testid="home-brief-section-{item.beads_id}-{section.label.toLowerCase()}">
											<span class="text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">{section.label}</span>
											<p class="text-sm leading-5 text-foreground/90">{section.text}</p>
										</div>
									{/each}
								</div>
							{:else}
								<p class="mt-2 text-sm text-muted-foreground">Loading brief content...</p>
							{/if}
						</div>
					{/each}
				</div>
			{:else}
				<div class="mt-3 rounded border border-dashed p-4 text-center text-sm text-muted-foreground">
					No recent briefs yet.
				</div>
			{/if}
		</div>

		<div class="rounded-lg border bg-card px-3 py-3" data-testid="home-questions-card">
			<div class="flex items-center justify-between gap-3">
				<div>
					<p class="text-sm font-medium">Open Questions</p>
					<p class="text-xs text-muted-foreground">The tensions blocking clean movement right now.</p>
				</div>
				{#if $questions && $questions.open && $questions.open.length > 0}
					<Badge variant="destructive" class="h-5 px-1.5 text-xs">{$questions.open.length} open</Badge>
				{:else}
					<span class="text-xs text-muted-foreground">none</span>
				{/if}
			</div>

			{#if featuredQuestions.length > 0}
				<div class="mt-3 space-y-2">
					{#each featuredQuestions as question (question.id)}
						<div class="rounded-md border border-border/70 bg-background/40 px-3 py-2" data-testid="home-question-{question.id}">
							<p class="text-sm leading-5 text-foreground">{question.title}</p>
							<div class="mt-2 flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
								<span>{question.status === 'open' ? 'Needs answer' : 'Investigating'}</span>
								{#if question.blocking && question.blocking.length > 0}
									<Badge variant="outline" class="h-5 px-1.5 text-[10px]">Blocks {question.blocking.length}</Badge>
								{/if}
								<span class="font-mono text-[11px]">{question.id}</span>
							</div>
						</div>
					{/each}
				</div>
			{:else}
				<div class="mt-3 rounded border border-dashed p-4 text-center text-sm text-muted-foreground">
					No active tensions.
				</div>
			{/if}
		</div>
	</div>

	<!-- Questions detail (promoted above fold — always visible when blocking questions exist) -->
	<QuestionsSection
		bind:expanded={sectionState.questions}
	/>

	<!-- Review Queue (comprehension queue — completions awaiting review) -->
	<ReviewQueueSection
		bind:expanded={sectionState.reviewQueue}
	/>

	<!-- ═══════════════════════════════════════════════════════════ -->
	<!-- CONDENSED OPERATIONAL SUMMARY -->
	<!-- ═══════════════════════════════════════════════════════════ -->

	<div class="rounded-lg border bg-card px-3 py-2">
		<div class="flex items-center justify-between">
			<span class="text-xs text-muted-foreground">
				{$trulyActiveAgents.length} agent{$trulyActiveAgents.length === 1 ? '' : 's'} active
				· {$readyIssues?.count ?? 0} ready
				· {$needsReviewAgents.length} need review
			</span>
			<a href="/work-graph" class="text-xs font-medium text-foreground hover:underline">
				View Work →
			</a>
		</div>
	</div>
</div>
