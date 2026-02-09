	<script lang="ts">
		import { createEventDispatcher } from 'svelte';
		import { onMount } from 'svelte';
	import { AlertTriangle, Check, Plus, ChevronDown } from '@lucide/svelte';
	import { Badge } from '$lib/components/ui/badge';
	import {
		kbModelProbes,
		type ModelProbe,
		type ModelProbeItem,
		type ModelProbeStatus,
		type ProbeVerdict,
	} from '$lib/stores/kb-model-probes';
		import { orchestratorContext } from '$lib/stores/context';

		const dispatch = createEventDispatcher();

	let expanded = new Set<string>();

	onMount(() => {
		if ($kbModelProbes) {
			return;
		}
		const projectDir = $orchestratorContext?.project_dir;
		kbModelProbes.fetch(projectDir).catch(console.error);
	});

	$: models = $kbModelProbes?.models ?? [];
	$: queue = ($kbModelProbes?.queue ?? []).filter((item) => isActionable(item));

	function isActionable(item: ModelProbe): boolean {
		if (item.merged) {
			return false;
		}
		return item.verdict === 'extends' || item.verdict === 'contradicts';
	}

	function toggle(name: string) {
		if (expanded.has(name)) {
			expanded.delete(name);
			expanded = new Set(expanded);
			return;
		}
		expanded.add(name);
		expanded = new Set(expanded);
	}

	function isExpanded(name: string): boolean {
		return expanded.has(name);
	}

	function statusClass(status: ModelProbeStatus): string {
		if (status === 'needs_review') {
			return 'bg-amber-500/20 text-amber-300 border-amber-500/30';
		}
		if (status === 'stale') {
			return 'bg-zinc-500/20 text-zinc-300 border-zinc-500/30';
		}
		if (status === 'well_validated') {
			return 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30';
		}
		return 'bg-blue-500/20 text-blue-300 border-blue-500/30';
	}

	function statusLabel(status: ModelProbeStatus): string {
		return status.replace('_', ' ');
	}

	function verdictMeta(verdict: ProbeVerdict): {
		icon: typeof Check;
		className: string;
		label: string;
	} {
		if (verdict === 'confirms') {
			return {
				icon: Check,
				className: 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30',
				label: 'confirms',
			};
		}
		if (verdict === 'extends') {
			return {
				icon: Plus,
				className: 'bg-blue-500/20 text-blue-300 border-blue-500/30',
				label: 'extends',
			};
		}
		return {
			icon: AlertTriangle,
			className: 'bg-red-500/20 text-red-300 border-red-500/30',
			label: 'contradicts',
		};
	}

	function count(model: ModelProbeItem, verdict: ProbeVerdict): number {
		return model.probe_counts?.[verdict] ?? 0;
	}

	function formatProbeDate(value?: string): string {
		if (!value) {
			return 'no probes yet';
		}
		const date = new Date(value);
		if (Number.isNaN(date.getTime())) {
			return value;
		}
		return date.toLocaleDateString(undefined, {
			month: 'short',
			day: 'numeric',
			year: 'numeric',
		});
	}

	function probeTitle(item: ModelProbe): string {
		if (item.title) {
			return item.title;
		}
		const file = item.probe_path.split('/').pop() || item.probe_path;
		return file.replace(/\.md$/, '').replace(/-/g, ' ');
	}

		function copyMergeCommand(path: string) {
			navigator.clipboard.writeText(`kb merge-probe ${path}`);
		}

		function modelPath(name: string): string {
			return `.kb/models/${name}.md`;
		}

		function select(path: string) {
			dispatch('select', { path });
		}
	</script>

{#if $kbModelProbes && ($kbModelProbes.summary.models_total > 0 || $kbModelProbes.error)}
	<section class="px-6 py-4 border-b border-border" data-testid="model-probe-section">
		<div class="flex items-center justify-between mb-3">
			<h2 class="text-sm font-semibold text-foreground">MODELS &amp; PROBES</h2>
			<div class="flex flex-wrap items-center gap-1.5 text-[11px]">
				<Badge variant="outline" class="h-5 px-1.5 text-[10px]">
					{$kbModelProbes.summary.models_total} models
				</Badge>
				<Badge variant="outline" class="h-5 px-1.5 text-[10px]">
					{$kbModelProbes.summary.probes_total} probes
				</Badge>
			</div>
		</div>

		{#if $kbModelProbes.error}
			<p class="text-sm text-muted-foreground">Model/probe data unavailable</p>
		{:else}
			<div class="grid grid-cols-1 gap-3 lg:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
				<div class="space-y-2">
					{#each models as model (model.name)}
						<div class="rounded-lg border border-border bg-card p-3">
							<div class="flex items-start justify-between gap-2">
								<div class="min-w-0 flex-1">
									<div class="flex items-center gap-2 min-w-0">
										<button
											type="button"
											onclick={() => select(model.path)}
											class="text-sm font-medium text-foreground truncate hover:text-blue-400"
										>
											{model.name}
										</button>
										<Badge variant="outline" class={statusClass(model.status)}>
											{statusLabel(model.status)}
										</Badge>
									</div>
									<div class="mt-2 flex flex-wrap gap-1.5 text-[11px]">
										{#if model.unmerged_count > 0}
											<Badge variant="outline" class="bg-amber-500/20 text-amber-300 border-amber-500/30">
												+{model.unmerged_count} unmerged
											</Badge>
										{/if}
										<Badge variant="outline" class="h-5 px-1.5 text-[10px]">
											last probe {formatProbeDate(model.last_probe_at)}
										</Badge>
										<Badge variant="outline" class="h-5 px-1.5 text-[10px]">C {count(model, 'confirms')}</Badge>
										<Badge variant="outline" class="h-5 px-1.5 text-[10px]">E {count(model, 'extends')}</Badge>
										<Badge variant="outline" class="h-5 px-1.5 text-[10px]">X {count(model, 'contradicts')}</Badge>
									</div>
								</div>
								<button
									type="button"
									onclick={() => toggle(model.name)}
									class="inline-flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground"
								>
									<span>{isExpanded(model.name) ? 'hide timeline' : 'show timeline'}</span>
									<ChevronDown size={14} class={isExpanded(model.name) ? 'rotate-180 transition-transform' : 'transition-transform'} />
								</button>
							</div>

							{#if isExpanded(model.name)}
								<div class="mt-3 border-t border-border/60 pt-2 space-y-2">
									{#if model.probes.length === 0}
										<p class="text-xs text-muted-foreground">No probes yet</p>
									{:else}
										{#each model.probes as probe (probe.probe_path)}
											{@const meta = verdictMeta(probe.verdict)}
											<button
												type="button"
												onclick={() => select(probe.probe_path)}
												class="w-full text-left rounded border border-border/70 bg-background/40 p-2 hover:border-foreground/40"
											>
												<div class="flex items-center justify-between gap-2">
													<Badge variant="outline" class={meta.className}>
														<svelte:component this={meta.icon} size={11} />
														<span>{meta.label}</span>
													</Badge>
													<span class="text-xs text-muted-foreground">{formatProbeDate(probe.date)}</span>
												</div>
												<p class="text-xs text-foreground mt-1 truncate">{probeTitle(probe)}</p>
											</button>
										{/each}
									{/if}
								</div>
							{/if}
						</div>
					{/each}
				</div>

				<div class="rounded-lg border border-border bg-card p-3 h-fit" data-testid="model-probe-queue">
					<div class="flex items-center justify-between mb-2">
						<h3 class="text-sm font-medium text-foreground">Merge Review Queue</h3>
						<Badge variant="secondary" class="h-5 px-1.5 text-xs">{queue.length}</Badge>
					</div>

					{#if queue.length === 0}
						<p class="text-xs text-muted-foreground">No actionable probes</p>
					{:else}
						<div class="space-y-2">
							{#each queue as item (item.probe_path)}
								{@const meta = verdictMeta(item.verdict)}
								<div class="rounded border border-border/70 bg-background/40 p-2">
									<div class="flex items-start justify-between gap-2">
										<div class="min-w-0">
											<div class="flex items-center gap-1.5">
												<Badge variant="outline" class={meta.className}>
													<svelte:component this={meta.icon} size={11} />
													<span>{meta.label}</span>
												</Badge>
												<button
													type="button"
													onclick={() => select(modelPath(item.model))}
													class="text-xs text-muted-foreground truncate hover:text-blue-400"
												>
													{item.model}
												</button>
											</div>
											<button
												type="button"
												onclick={() => select(item.probe_path)}
												class="text-sm text-foreground mt-1 truncate hover:text-blue-400"
											>
												{probeTitle(item)}
											</button>
											{#if item.claim}
												<p class="text-xs text-muted-foreground mt-1 line-clamp-2">{item.claim}</p>
											{/if}
											<p class="text-xs text-muted-foreground mt-1">{formatProbeDate(item.date)}</p>
										</div>
										<div class="flex flex-col items-end gap-1 shrink-0">
											<button
												type="button"
												onclick={() => select(item.probe_path)}
												class="text-xs text-muted-foreground hover:text-foreground"
											>
												review probe
											</button>
											<button
												type="button"
												onclick={() => copyMergeCommand(item.probe_path)}
												class="text-xs text-muted-foreground hover:text-foreground"
											>
												copy merge
											</button>
										</div>
									</div>
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</section>
{/if}
