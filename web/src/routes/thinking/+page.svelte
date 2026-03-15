<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import { digestProducts, digestStats, type DigestProduct } from '$lib/stores/digest';

	let pollInterval: ReturnType<typeof setInterval> | null = null;
	let stateFilter: string = '';
	let typeFilter: string = '';
	let expandedId: string | null = null;

	function typeLabel(type: string): string {
		switch (type) {
			case 'thread_progression': return 'Thread';
			case 'model_update': return 'Model';
			case 'model_probe': return 'Probe';
			case 'decision_brief': return 'Decision';
			default: return type;
		}
	}

	function typeBadgeColor(type: string): string {
		switch (type) {
			case 'thread_progression': return 'bg-blue-500/20 text-blue-400';
			case 'model_update': return 'bg-purple-500/20 text-purple-400';
			case 'model_probe': return 'bg-cyan-500/20 text-cyan-400';
			case 'decision_brief': return 'bg-amber-500/20 text-amber-400';
			default: return 'bg-foreground/10 text-muted-foreground';
		}
	}

	function significanceBadge(sig: string): string {
		switch (sig) {
			case 'high': return 'bg-red-500/20 text-red-400';
			case 'medium': return 'bg-yellow-500/20 text-yellow-400';
			case 'low': return 'bg-foreground/10 text-muted-foreground';
			default: return 'bg-foreground/10 text-muted-foreground';
		}
	}

	function stateBadge(state: string): string {
		switch (state) {
			case 'new': return 'bg-blue-500/20 text-blue-400';
			case 'read': return 'bg-foreground/10 text-muted-foreground';
			case 'starred': return 'bg-yellow-500/20 text-yellow-400';
			case 'archived': return 'bg-foreground/5 text-muted-foreground/50';
			default: return 'bg-foreground/10 text-muted-foreground';
		}
	}

	function changeTypeLabel(ct: string): string {
		switch (ct) {
			case 'content_added': return 'content added';
			case 'created': return 'new';
			case 'modified': return 'modified';
			case 'completed': return 'completed';
			case 'resolved': return 'resolved';
			default: return ct;
		}
	}

	function formatTime(iso: string): string {
		if (!iso) return '';
		const d = new Date(iso);
		const now = new Date();
		const diffMs = now.getTime() - d.getTime();
		const diffMins = Math.floor(diffMs / 60000);
		if (diffMins < 1) return 'just now';
		if (diffMins < 60) return `${diffMins}m ago`;
		const diffHours = Math.floor(diffMins / 60);
		if (diffHours < 24) return `${diffHours}h ago`;
		const diffDays = Math.floor(diffHours / 24);
		if (diffDays < 7) return `${diffDays}d ago`;
		return d.toLocaleDateString();
	}

	async function reload() {
		await digestProducts.fetch(stateFilter || undefined, typeFilter || undefined);
	}

	async function markRead(product: DigestProduct) {
		if (product.state === 'new') {
			await digestProducts.updateState(product.id, 'read');
			await digestStats.fetch();
		}
	}

	async function toggleStar(product: DigestProduct) {
		const newState = product.state === 'starred' ? 'read' : 'starred';
		await digestProducts.updateState(product.id, newState);
		await digestStats.fetch();
	}

	async function archive(product: DigestProduct) {
		await digestProducts.updateState(product.id, 'archived');
		await digestStats.fetch();
	}

	async function archiveAllRead() {
		const count = await digestProducts.archiveRead();
		if (count > 0) {
			await reload();
			await digestStats.fetch();
		}
	}

	function toggleExpand(product: DigestProduct) {
		if (expandedId === product.id) {
			expandedId = null;
		} else {
			expandedId = product.id;
			markRead(product);
		}
	}

	onMount(async () => {
		await reload();
		pollInterval = setInterval(reload, 60_000);
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
	});
</script>

<div class="space-y-4">
	<!-- Header -->
	<div class="flex items-center justify-between">
		<div>
			<h1 class="text-lg font-semibold">Thinking Products</h1>
			<p class="text-xs text-muted-foreground">
				Digest of knowledge base changes — threads, models, probes, decisions
			</p>
		</div>
		<div class="flex items-center gap-3 text-xs">
			<span class="text-muted-foreground">
				{$digestStats.unread} unread · {$digestStats.starred} starred · {$digestStats.total} total
			</span>
		</div>
	</div>

	<!-- Filters -->
	<div class="flex items-center gap-2 flex-wrap">
		<div class="flex items-center gap-0.5 rounded-md border p-0.5">
			<button
				class="px-2 py-1 text-xs rounded transition-colors {stateFilter === '' ? 'bg-foreground/10 text-foreground font-medium' : 'text-muted-foreground hover:text-foreground'}"
				on:click={() => { stateFilter = ''; reload(); }}
			>All</button>
			<button
				class="px-2 py-1 text-xs rounded transition-colors {stateFilter === 'new' ? 'bg-blue-500/20 text-blue-400 font-medium' : 'text-muted-foreground hover:text-foreground'}"
				on:click={() => { stateFilter = 'new'; reload(); }}
			>Unread{#if $digestStats.unread > 0}<span class="ml-1 text-[10px]">({$digestStats.unread})</span>{/if}</button>
			<button
				class="px-2 py-1 text-xs rounded transition-colors {stateFilter === 'starred' ? 'bg-yellow-500/20 text-yellow-400 font-medium' : 'text-muted-foreground hover:text-foreground'}"
				on:click={() => { stateFilter = 'starred'; reload(); }}
			>Starred{#if $digestStats.starred > 0}<span class="ml-1 text-[10px]">({$digestStats.starred})</span>{/if}</button>
			<button
				class="px-2 py-1 text-xs rounded transition-colors {stateFilter === 'read' ? 'bg-foreground/10 text-foreground font-medium' : 'text-muted-foreground hover:text-foreground'}"
				on:click={() => { stateFilter = 'read'; reload(); }}
			>Read</button>
			<button
				class="px-2 py-1 text-xs rounded transition-colors {stateFilter === 'archived' ? 'bg-foreground/10 text-foreground font-medium' : 'text-muted-foreground hover:text-foreground'}"
				on:click={() => { stateFilter = 'archived'; reload(); }}
			>Archived</button>
		</div>

		<div class="flex items-center gap-0.5 rounded-md border p-0.5">
			<button
				class="px-2 py-1 text-xs rounded transition-colors {typeFilter === '' ? 'bg-foreground/10 text-foreground font-medium' : 'text-muted-foreground hover:text-foreground'}"
				on:click={() => { typeFilter = ''; reload(); }}
			>All types</button>
			<button
				class="px-2 py-1 text-xs rounded transition-colors {typeFilter === 'thread_progression' ? 'bg-blue-500/20 text-blue-400 font-medium' : 'text-muted-foreground hover:text-foreground'}"
				on:click={() => { typeFilter = 'thread_progression'; reload(); }}
			>Threads</button>
			<button
				class="px-2 py-1 text-xs rounded transition-colors {typeFilter === 'model_update' ? 'bg-purple-500/20 text-purple-400 font-medium' : 'text-muted-foreground hover:text-foreground'}"
				on:click={() => { typeFilter = 'model_update'; reload(); }}
			>Models</button>
			<button
				class="px-2 py-1 text-xs rounded transition-colors {typeFilter === 'model_probe' ? 'bg-cyan-500/20 text-cyan-400 font-medium' : 'text-muted-foreground hover:text-foreground'}"
				on:click={() => { typeFilter = 'model_probe'; reload(); }}
			>Probes</button>
			<button
				class="px-2 py-1 text-xs rounded transition-colors {typeFilter === 'decision_brief' ? 'bg-amber-500/20 text-amber-400 font-medium' : 'text-muted-foreground hover:text-foreground'}"
				on:click={() => { typeFilter = 'decision_brief'; reload(); }}
			>Decisions</button>
		</div>

		{#if stateFilter === 'read' || stateFilter === ''}
			<button
				class="ml-auto px-2 py-1 text-xs rounded border text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
				on:click={archiveAllRead}
			>Archive read</button>
		{/if}
	</div>

	<!-- Product List -->
	{#if $digestProducts.products.length === 0}
		<div class="rounded-md border p-8 text-center">
			<p class="text-sm text-muted-foreground">
				{#if stateFilter || typeFilter}
					No products match the current filters.
				{:else}
					No thinking products yet. The daemon produces these from KB changes.
				{/if}
			</p>
		</div>
	{:else}
		<div class="space-y-1.5">
			{#each $digestProducts.products as product (product.id)}
				<button
					class="w-full text-left rounded-md border p-3 transition-colors hover:bg-accent/50 {product.state === 'new' ? 'border-blue-500/30 bg-blue-500/5' : ''} {product.state === 'archived' ? 'opacity-50' : ''} {expandedId === product.id ? 'ring-1 ring-foreground/20' : ''}"
					on:click={() => toggleExpand(product)}
				>
					<div class="flex items-start justify-between gap-2">
						<div class="flex-1 min-w-0">
							<div class="flex items-center gap-1.5 mb-1">
								{#if product.state === 'new'}
									<span class="h-1.5 w-1.5 rounded-full bg-blue-500 flex-shrink-0"></span>
								{/if}
								<span class="text-xs font-medium truncate">{product.title}</span>
							</div>
							<p class="text-[11px] text-muted-foreground line-clamp-2">{product.summary}</p>
						</div>
						<div class="flex items-center gap-1.5 flex-shrink-0">
							<span class="px-1.5 py-0.5 text-[10px] rounded {typeBadgeColor(product.type)}">{typeLabel(product.type)}</span>
							<span class="px-1.5 py-0.5 text-[10px] rounded {significanceBadge(product.significance)}">{product.significance}</span>
							<span class="text-[10px] text-muted-foreground tabular-nums">{formatTime(product.created_at)}</span>
						</div>
					</div>

					<!-- Expanded details -->
					{#if expandedId === product.id}
						<div class="mt-3 pt-3 border-t space-y-2">
							<div class="text-[11px] space-y-1">
								<div class="flex justify-between">
									<span class="text-muted-foreground">Source</span>
									<span class="font-mono text-[10px]">{product.source.path}</span>
								</div>
								<div class="flex justify-between">
									<span class="text-muted-foreground">Change</span>
									<span>{changeTypeLabel(product.source.change_type)}</span>
								</div>
								{#if product.source.delta_words}
									<div class="flex justify-between">
										<span class="text-muted-foreground">Delta</span>
										<span>{product.source.delta_words > 0 ? '+' : ''}{product.source.delta_words} words</span>
									</div>
								{/if}
								<div class="flex justify-between">
									<span class="text-muted-foreground">State</span>
									<span class="px-1.5 py-0.5 rounded {stateBadge(product.state)}">{product.state}</span>
								</div>
							</div>

							<!-- Actions -->
							<div class="flex items-center gap-2 pt-1">
								<button
									class="px-2 py-1 text-[11px] rounded border transition-colors {product.state === 'starred' ? 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30' : 'text-muted-foreground hover:text-foreground hover:bg-accent/50'}"
									on:click|stopPropagation={() => toggleStar(product)}
								>{product.state === 'starred' ? 'Unstar' : 'Star'}</button>
								{#if product.state !== 'archived'}
									<button
										class="px-2 py-1 text-[11px] rounded border text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
										on:click|stopPropagation={() => archive(product)}
									>Archive</button>
								{/if}
							</div>
						</div>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>
