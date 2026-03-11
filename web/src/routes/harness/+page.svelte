<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		harness,
		stageLabel,
		stageDescription,
		componentLabel,
		typeColor,
		measurementIcon,
		measurementColor,
		verdictIcon,
		verdictColor,
		verdictPlainLanguage,
		formatRate,
		harnessSummary,
		coverageFraming,
		type PipelineComponent,
		type ExplorationMetrics
	} from '$lib/stores/harness';

	let pollInterval: ReturnType<typeof setInterval> | null = null;
	let expandedCard: string | null = null;
	let days = 7;

	function toggleCard(name: string) {
		expandedCard = expandedCard === name ? null : name;
	}

	function rateBar(rate: number | undefined | null): string {
		if (rate === undefined || rate === null) return 'w-0';
		const pct = Math.min(rate * 100, 100);
		if (pct < 5) return 'w-1';
		if (pct < 15) return 'w-3';
		if (pct < 30) return 'w-6';
		if (pct < 50) return 'w-10';
		if (pct < 75) return 'w-16';
		return 'w-20';
	}

	function coverageColor(pct: number): string {
		if (pct >= 95) return 'text-green-400';
		if (pct >= 80) return 'text-yellow-400';
		return 'text-muted-foreground';
	}

	function fireRateLabel(comp: PipelineComponent): string {
		if (comp.fire_rate === undefined || comp.fire_rate === null) return '';
		const pct = (comp.fire_rate * 100).toFixed(0);
		return `fires on ${pct}% of spawns`;
	}

	onMount(async () => {
		await harness.fetch(days);
		pollInterval = setInterval(() => harness.fetch(days), 60_000);
	});

	onDestroy(() => {
		if (pollInterval) clearInterval(pollInterval);
	});
</script>

<div class="space-y-4">
	<!-- Header + Summary -->
	<div class="space-y-2">
		<div class="flex items-center justify-between">
			<div>
				<h1 class="text-lg font-semibold">Harness Pipeline</h1>
				<p class="text-xs text-muted-foreground">
					{$harness.analysis_period} · {$harness.total_spawns} spawns
				</p>
			</div>
			<div class="flex items-center gap-2">
				<select
					class="text-xs bg-background border rounded px-2 py-1"
					bind:value={days}
					on:change={() => harness.fetch(days)}
				>
					<option value={7}>7 days</option>
					<option value={14}>14 days</option>
					<option value={30}>30 days</option>
				</select>
				<div class="flex items-center gap-3 text-xs text-muted-foreground">
					<span class="inline-flex items-center gap-1"><span class="text-green-400">●</span> measured</span>
					<span class="inline-flex items-center gap-1"><span class="text-blue-400">◑</span> collecting</span>
					<span class="inline-flex items-center gap-1"><span class="text-yellow-400">◐</span> proxy</span>
					<span class="inline-flex items-center gap-1"><span class="text-muted-foreground">○</span> unmeasured</span>
				</div>
			</div>
		</div>

		<!-- Top-level summary: "Is the harness working?" -->
		<div class="rounded-md border p-3 bg-card text-xs text-muted-foreground">
			<p>{harnessSummary($harness)}</p>
			<p class="mt-1 opacity-70">The harness is a set of gates and context layers that check agent work at each stage. This page shows which parts are measured, what the data says, and where the gaps are.</p>
		</div>
	</div>

	<!-- Pipeline Visualization -->
	<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
		{#each $harness.pipeline as stage}
			<div class="space-y-2">
				<div>
					<div class="flex items-center gap-2">
						<h2 class="text-sm font-medium">{stageLabel(stage.stage)}</h2>
						<span class="text-xs text-muted-foreground">
							{stage.components.length}
						</span>
					</div>
					<p class="text-[10px] text-muted-foreground mt-0.5">{stageDescription(stage.stage)}</p>
				</div>
				<div class="space-y-1.5">
					{#each stage.components as comp}
						<button
							class="w-full text-left rounded-md border p-2 transition-colors hover:bg-accent/50 {typeColor(comp.type)} {expandedCard === comp.name ? 'ring-1 ring-foreground/20' : ''}"
							on:click={() => toggleCard(comp.name)}
						>
							<div class="flex items-center justify-between">
								<div class="flex items-center gap-1.5">
									<span class="{measurementColor(comp.measurement_status)} text-xs">{measurementIcon(comp.measurement_status)}</span>
									<span class="text-xs font-medium">{componentLabel(comp.name)}</span>
								</div>
								<span class="text-[10px] uppercase opacity-60">{comp.type}</span>
							</div>

							<!-- Fire rate with label -->
							{#if comp.fire_rate !== undefined && comp.fire_rate !== null}
								<div class="mt-1.5 flex items-center gap-2">
									<div class="flex-1 h-1 rounded-full bg-foreground/10 overflow-hidden">
										<div class="h-full rounded-full bg-current {rateBar(comp.fire_rate)}" style="width: {Math.min((comp.fire_rate || 0) * 100, 100)}%"></div>
									</div>
									<span class="text-[10px] tabular-nums opacity-70">{fireRateLabel(comp)}</span>
								</div>
							{/if}

							<!-- Proxy-only / collecting / unmeasured explanation -->
							{#if comp.measurement_status === 'proxy_only' && comp.proxy_metric}
								<p class="mt-1.5 text-[10px] opacity-50 italic">Proxy: {comp.proxy_metric}</p>
							{:else if comp.measurement_status === 'collecting'}
								<p class="mt-1.5 text-[10px] text-blue-400/70 italic">Collecting data{comp.collecting_since ? ` (since ${comp.collecting_since})` : ''}</p>
							{:else if comp.measurement_status === 'unmeasured' && !comp.fire_rate}
								<p class="mt-1.5 text-[10px] opacity-50 italic">No measurement data</p>
							{/if}

							<!-- Expanded details -->
							{#if expandedCard === comp.name}
								<div class="mt-2 pt-2 border-t border-current/20 space-y-1 text-[11px]">
									{#if comp.fire_rate !== undefined}
										<div class="flex justify-between">
											<span class="opacity-70">Fire rate</span>
											<span class="tabular-nums">{formatRate(comp.fire_rate)}</span>
										</div>
									{/if}
									{#if comp.block_rate !== undefined}
										<div class="flex justify-between">
											<span class="opacity-70">Block rate</span>
											<span class="tabular-nums">{formatRate(comp.block_rate)}</span>
										</div>
									{/if}
									{#if comp.bypass_rate !== undefined}
										<div class="flex justify-between">
											<span class="opacity-70">Bypass rate</span>
											<span class="tabular-nums">{formatRate(comp.bypass_rate)}</span>
										</div>
									{/if}
									{#if comp.bypassed}
										<div class="flex justify-between">
											<span class="opacity-70">Bypassed</span>
											<span class="tabular-nums">{comp.bypassed}</span>
										</div>
									{/if}
									{#if comp.blocked}
										<div class="flex justify-between">
											<span class="opacity-70">Blocked</span>
											<span class="tabular-nums">{comp.blocked}</span>
										</div>
									{/if}
									{#if comp.last_fired}
										<div class="flex justify-between">
											<span class="opacity-70">Last fired</span>
											<span class="tabular-nums">{new Date(comp.last_fired).toLocaleDateString()}</span>
										</div>
									{/if}
								</div>
							{/if}
						</button>
					{/each}
				</div>
			</div>
		{/each}
	</div>

	<!-- Exploration Metrics (only shown when exploration events exist) -->
	{#if $harness.exploration_metrics}
		{@const em = $harness.exploration_metrics}
		<div class="space-y-2">
			<div>
				<h2 class="text-sm font-medium">Exploration Runs</h2>
				<p class="text-[10px] text-muted-foreground mt-0.5">Parallel decomposition with judge synthesis — analysis quality metrics</p>
			</div>
			<div class="rounded-md border p-3 bg-card">
				<div class="grid grid-cols-4 gap-3 text-center mb-3">
					<div>
						<div class="text-lg font-semibold">{em.total_runs}</div>
						<div class="text-[10px] text-muted-foreground">Runs</div>
					</div>
					<div>
						<div class="text-lg font-semibold text-green-400">{em.completed_runs}</div>
						<div class="text-[10px] text-muted-foreground">Completed</div>
					</div>
					<div>
						<div class="text-lg font-semibold">{em.total_findings}</div>
						<div class="text-[10px] text-muted-foreground">Findings</div>
					</div>
					<div>
						<div class="text-lg font-semibold">{em.avg_workers_per_run.toFixed(1)}</div>
						<div class="text-[10px] text-muted-foreground">Avg Workers</div>
					</div>
				</div>
				<!-- Judge verdict breakdown -->
				{#if em.total_findings > 0}
					<div class="space-y-1">
						<div class="text-[10px] text-muted-foreground mb-1">Judge Verdicts</div>
						<div class="h-3 rounded-full bg-foreground/10 overflow-hidden flex">
							<div
								class="h-full bg-green-500"
								title="{em.total_accepted} accepted"
								style="width: {(em.total_accepted / em.total_findings) * 100}%"
							></div>
							<div
								class="h-full bg-yellow-500"
								title="{em.total_contested} contested"
								style="width: {(em.total_contested / em.total_findings) * 100}%"
							></div>
							<div
								class="h-full bg-red-500"
								title="{em.total_rejected} rejected"
								style="width: {(em.total_rejected / em.total_findings) * 100}%"
							></div>
						</div>
						<div class="flex justify-between text-[10px] text-muted-foreground">
							<span class="flex items-center gap-1"><span class="inline-block w-2 h-2 rounded-full bg-green-500"></span> {em.total_accepted} accepted</span>
							<span class="flex items-center gap-1"><span class="inline-block w-2 h-2 rounded-full bg-yellow-500"></span> {em.total_contested} contested</span>
							<span class="flex items-center gap-1"><span class="inline-block w-2 h-2 rounded-full bg-red-500"></span> {em.total_rejected} rejected</span>
						</div>
					</div>
					{#if em.total_gaps > 0}
						<p class="mt-2 text-[10px] text-yellow-400/70">{em.total_gaps} coverage gap{em.total_gaps > 1 ? 's' : ''} identified by judges</p>
					{/if}
				{/if}
			</div>
		</div>
	{/if}

	<!-- Bottom sections: Verdicts + Coverage -->
	<div class="grid grid-cols-1 lg:grid-cols-2 gap-3">
		<!-- Falsification Verdicts (plain language) -->
		<div class="space-y-2">
			<div>
				<h2 class="text-sm font-medium">Can we disprove the harness works?</h2>
				<p class="text-[10px] text-muted-foreground mt-0.5">Each row tests a way the harness could fail. Green = that failure mode is ruled out.</p>
			</div>
			<div class="space-y-1.5">
				{#each Object.entries($harness.falsification_verdicts) as [key, verdict]}
					<div class="rounded-md border p-2 {verdictColor(verdict.status)}">
						<div class="flex items-center justify-between">
							<div class="flex items-center gap-1.5">
								<span class="text-sm">{verdictIcon(verdict.status)}</span>
								<span class="text-xs font-medium">{verdictPlainLanguage(key, verdict.status)}</span>
							</div>
						</div>
						<p class="mt-1 text-[11px] opacity-70">{verdict.evidence}</p>
					</div>
				{/each}
			</div>
		</div>

		<!-- Measurement Coverage + Event Data Quality -->
		<div class="space-y-3">
			<!-- Measurement Coverage -->
			<div class="space-y-2">
				<h2 class="text-sm font-medium">Measurement Coverage</h2>
				<div class="rounded-md border p-3 bg-card">
					<p class="text-[10px] text-muted-foreground mb-2">{coverageFraming($harness.measurement_coverage)}</p>
					<div class="grid grid-cols-4 gap-2 text-center">
						<div>
							<div class="text-lg font-semibold">{$harness.measurement_coverage.total_components}</div>
							<div class="text-[10px] text-muted-foreground">Total</div>
						</div>
						<div>
							<div class="text-lg font-semibold text-green-400">{$harness.measurement_coverage.with_measurement}</div>
							<div class="text-[10px] text-muted-foreground">Measured</div>
						</div>
						<div>
							<div class="text-lg font-semibold text-yellow-400">{$harness.measurement_coverage.proxy_only}</div>
							<div class="text-[10px] text-muted-foreground">Proxy</div>
						</div>
						<div>
							<div class="text-lg font-semibold text-muted-foreground">{$harness.measurement_coverage.unmeasured}</div>
							<div class="text-[10px] text-muted-foreground">Unmeasured</div>
						</div>
					</div>
					<!-- Coverage bar -->
					<div class="mt-2 h-2 rounded-full bg-foreground/10 overflow-hidden flex">
						{#if $harness.measurement_coverage.total_components > 0}
							<div
								class="h-full bg-green-500"
								style="width: {($harness.measurement_coverage.with_measurement / $harness.measurement_coverage.total_components) * 100}%"
							></div>
							<div
								class="h-full bg-yellow-500"
								style="width: {($harness.measurement_coverage.proxy_only / $harness.measurement_coverage.total_components) * 100}%"
							></div>
						{/if}
					</div>
				</div>
			</div>

			<!-- Event Data Quality (renamed from Completion Coverage) -->
			<div class="space-y-2">
				<div>
					<h2 class="text-sm font-medium">Event Data Quality</h2>
					<p class="text-[10px] text-muted-foreground mt-0.5">How complete are completion event fields — not agent success rate</p>
				</div>
				<div class="rounded-md border p-3 bg-card">
					<div class="flex items-center justify-between mb-2">
						<span class="text-xs text-muted-foreground">{$harness.completion_coverage.total_completions} completions logged</span>
						<span class="text-sm font-semibold {coverageColor($harness.completion_coverage.coverage_pct)}">
							{$harness.completion_coverage.coverage_pct.toFixed(0)}% fields filled
						</span>
					</div>
					<div class="space-y-1 text-[11px]">
						<div class="flex justify-between">
							<span class="text-muted-foreground">With skill name</span>
							<span class="tabular-nums">{$harness.completion_coverage.with_skill}/{$harness.completion_coverage.total_completions}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-muted-foreground">With outcome</span>
							<span class="tabular-nums">{$harness.completion_coverage.with_outcome}/{$harness.completion_coverage.total_completions}</span>
						</div>
						<div class="flex justify-between">
							<span class="text-muted-foreground">With duration</span>
							<span class="tabular-nums">{$harness.completion_coverage.with_duration}/{$harness.completion_coverage.total_completions}</span>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
</div>
