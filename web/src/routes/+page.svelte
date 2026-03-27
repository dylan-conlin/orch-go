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
	import { briefs } from '$lib/stores/briefs';
	import { beads, readyIssues, reviewQueue } from '$lib/stores/beads';
	import { questions } from '$lib/stores/questions';
	import { filters, orchestratorContext, buildFilterQueryString } from '$lib/stores/context';

	// Thread expansion state
	let expandedThread: string | null = null;

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
								<p class="mt-1 text-xs text-muted-foreground truncate ml-5">{t.latest_entry}</p>
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

	<!-- New Evidence — unread briefs + open questions -->
	<div class="grid gap-2 sm:grid-cols-2">
		<!-- Unread Briefs -->
		<div class="rounded-lg border bg-card px-3 py-2">
			<div class="flex items-center justify-between">
				<span class="text-sm font-medium">Unread Briefs</span>
				{#if unreadBriefCount > 0}
					<a href="/briefs" class="inline-flex items-center gap-1">
						<Badge variant="default" class="h-5 px-1.5 text-xs">{unreadBriefCount}</Badge>
					</a>
				{:else}
					<span class="text-xs text-muted-foreground">all read</span>
				{/if}
			</div>
		</div>

		<!-- Open Questions -->
		<div class="rounded-lg border bg-card px-3 py-2">
			<div class="flex items-center justify-between">
				<span class="text-sm font-medium">Open Questions</span>
				{#if $questions && $questions.open && $questions.open.length > 0}
					<Badge variant="destructive" class="h-5 px-1.5 text-xs">{$questions.open.length} blocking</Badge>
				{:else}
					<span class="text-xs text-muted-foreground">none</span>
				{/if}
			</div>
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
