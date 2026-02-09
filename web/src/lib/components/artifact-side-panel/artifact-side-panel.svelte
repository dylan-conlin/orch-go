<script lang="ts">
		import { createEventDispatcher } from 'svelte';
		import { fetchArtifactContent, kbArtifacts, type ArtifactFeedItem } from '$lib/stores/kb-artifacts';
		import { orchestratorContext } from '$lib/stores/context';
		import { MarkdownContent } from '$lib/components/markdown-content';

		export let artifact: ArtifactFeedItem;

		const dispatch = createEventDispatcher();
		let content = '';
		let loading = true;
		let error: string | null = null;
		let timelineExpanded = false;
		let timelineLoading = false;
		let probeMeta = new Map<string, { verdict: ProbeVerdict | null; claim: string }>();
		let metaLoadRequest = 0;
		let previousArtifactPath = '';
		let knownArtifacts = new Map<string, ArtifactFeedItem>();
		let probeModelName: string | null = null;
		let modelName: string | null = null;
		let modelArtifact: ArtifactFeedItem | null = null;
		let modelProbes: ArtifactFeedItem[] = [];
		let probeIndex = -1;
		let prevProbe: ArtifactFeedItem | null = null;
		let nextProbe: ArtifactFeedItem | null = null;
		let currentProbeMeta: { verdict: ProbeVerdict | null; claim: string } | null = null;

		type ProbeVerdict = 'confirms' | 'extends' | 'contradicts';

		// Build a deduplicated map of known artifacts so lineage links can jump between model/probes.
		$: knownArtifacts = (() => {
			const map = new Map<string, ArtifactFeedItem>();

			for (const item of $kbArtifacts?.needs_decision ?? []) {
				map.set(item.path, item);
			}

			for (const item of $kbArtifacts?.recent ?? []) {
				map.set(item.path, item);
			}

			for (const items of Object.values($kbArtifacts?.by_type ?? {})) {
				for (const item of items) {
					map.set(item.path, item);
				}
			}

			return map;
		})();

		$: probeModelName = getProbeModelName(artifact.path);
		$: modelName = probeModelName ?? getModelName(artifact.path);
		$: modelArtifact = probeModelName ? getModelArtifact(probeModelName, knownArtifacts) : null;
		$: modelProbes = modelName
			? Array.from(knownArtifacts.values())
				.filter((item) => getProbeModelName(item.path) === modelName)
				.sort((a, b) => getArtifactTimestamp(b) - getArtifactTimestamp(a))
			: [];
		$: probeIndex = probeModelName ? modelProbes.findIndex((item) => item.path === artifact.path) : -1;
		$: prevProbe = probeIndex > 0 ? modelProbes[probeIndex - 1] : null;
		$: nextProbe = probeIndex >= 0 && probeIndex < modelProbes.length - 1 ? modelProbes[probeIndex + 1] : null;
		$: currentProbeMeta = probeModelName
			? (probeMeta.get(artifact.path) ?? {
					verdict: extractProbeVerdict(content),
					claim: extractQuestionExcerpt(content),
				})
			: null;

		// Reset local panel-only state whenever selected artifact changes.
		$: if (artifact?.path && artifact.path !== previousArtifactPath) {
			previousArtifactPath = artifact.path;
			timelineExpanded = false;
		}

		// React to artifact prop changes - fetch full content from API
		$: if (artifact) {
			loadArtifactContent(artifact);
		}

		$: if (!probeModelName && modelName && modelProbes.length > 0) {
			loadProbeMeta(modelProbes);
		}

	async function loadArtifactContent(artifact: ArtifactFeedItem) {
		loading = true;
		error = null;

		try {
			const projectDir = $orchestratorContext?.project_dir;
			const response = await fetchArtifactContent(artifact.path, projectDir);

			if (response.error) {
				error = response.error;
				// Fall back to metadata-only view
				content = generateFallbackMarkdown(artifact);
			} else {
				content = response.content;
			}
		} catch (e) {
			error = String(e);
			content = generateFallbackMarkdown(artifact);
		} finally {
			loading = false;
		}
	}

	// Generate fallback content when API fails
	function generateFallbackMarkdown(artifact: ArtifactFeedItem): string {
		let md = `# ${artifact.title}\n\n`;

		md += `**Type:** ${artifact.type}\n\n`;

		if (artifact.status) {
			md += `**Status:** ${artifact.status}\n\n`;
		}

		if (artifact.date) {
			md += `**Date:** ${artifact.date}\n\n`;
		}

		md += `**Last Modified:** ${artifact.relative_time}\n\n`;

		md += `**Path:** \`${artifact.path}\`\n\n`;

		if (artifact.recommendation) {
			md += `> ⚠️ This investigation has a recommendation\n\n`;
		}

		if (artifact.summary) {
			md += `## Summary\n\n${artifact.summary}\n\n`;
		}

		md += `---\n\n`;
		md += `_Could not load full content. Open file directly: \`${artifact.path}\`_\n`;

		return md;
	}

		function handleClose() {
			dispatch('close');
		}

		function getModelName(path: string): string | null {
			const match = normalizePath(path).match(/^\.kb\/models\/([^/]+)\.md$/);
			if (!match) return null;
			return match[1];
		}

		function getProbeModelName(path: string): string | null {
			const match = normalizePath(path).match(/^\.kb\/models\/([^/]+)\/probes\/[^/]+\.md$/);
			if (!match) return null;
			return match[1];
		}

		function getProbeSlug(path: string): string {
			const match = normalizePath(path).match(/\/probes\/([^/]+)\.md$/);
			if (!match) return path;
			return match[1];
		}

		function normalizePath(path: string): string {
			return path.replaceAll('\\\\', '/');
		}

		function getModelArtifact(model: string, artifacts: Map<string, ArtifactFeedItem>): ArtifactFeedItem | null {
			for (const item of artifacts.values()) {
				if (getModelName(item.path) === model) {
					return item;
				}
			}

			return null;
		}

		function getArtifactTimestamp(item: ArtifactFeedItem): number {
			const modified = Date.parse(item.modified_at);
			if (!Number.isNaN(modified)) {
				return modified;
			}

			const dated = Date.parse(item.date);
			if (!Number.isNaN(dated)) {
				return dated;
			}

			return 0;
		}

		function extractProbeVerdict(markdown: string): ProbeVerdict | null {
			const match = markdown.match(/\*\*Verdict:\*\*\s*(confirms|extends|contradicts)/i);
			if (!match) return null;
			return match[1].toLowerCase() as ProbeVerdict;
		}

		function extractQuestionExcerpt(markdown: string): string {
			const section = markdown.match(/(?:^|\n)##\s+Question\s*\n([\s\S]*?)(?:\n##\s+|\n#\s+|$)/i);
			if (!section) return '';

			for (const raw of section[1].split('\n')) {
				const line = sanitizeExcerptLine(raw);
				if (line) {
					return line;
				}
			}

			return '';
		}

		function sanitizeExcerptLine(line: string): string {
			const trimmed = line.trim();
			if (!trimmed) return '';
			if (trimmed.startsWith('<!--') || trimmed.startsWith('##')) return '';

			const withoutListPrefix = trimmed.replace(/^[-*]\s+/, '');
			const withoutLinks = withoutListPrefix.replace(/\[([^\]]+)\]\([^\)]+\)/g, '$1');
			const withoutBold = withoutLinks.replace(/\*\*([^*]+)\*\*/g, '$1');
			const withoutInlineCode = withoutBold.replace(/`([^`]+)`/g, '$1');
			return withoutInlineCode.trim();
		}

		function getVerdictClass(verdict: ProbeVerdict | null): string {
			if (verdict === 'confirms') return 'bg-green-500/10 text-green-500 border-green-500/20';
			if (verdict === 'extends') return 'bg-blue-500/10 text-blue-500 border-blue-500/20';
			if (verdict === 'contradicts') return 'bg-red-500/10 text-red-500 border-red-500/20';
			return 'bg-muted text-muted-foreground border-border';
		}

		function getVerdictLabel(verdict: ProbeVerdict | null): string {
			if (!verdict) return 'unknown';
			return verdict;
		}

		function selectArtifact(next: ArtifactFeedItem | null) {
			if (!next) return;
			artifact = next;
			dispatch('artifact-select', { artifact: next });
		}

		async function loadProbeMeta(probes: ArtifactFeedItem[]) {
			const pending = probes.filter((probe) => !probeMeta.has(probe.path));
			if (pending.length === 0) {
				return;
			}

			metaLoadRequest += 1;
			const requestID = metaLoadRequest;
			timelineLoading = true;

			try {
				const projectDir = $orchestratorContext?.project_dir;
				const pairs = await Promise.all(
					pending.map(async (probe) => {
						const response = await fetchArtifactContent(probe.path, projectDir);
						if (response.error) {
							return [probe.path, { verdict: null, claim: '' }] as const;
						}

						return [
							probe.path,
							{
								verdict: extractProbeVerdict(response.content),
								claim: extractQuestionExcerpt(response.content),
							},
						] as const;
					}),
				);

				if (requestID !== metaLoadRequest) {
					return;
				}

				const next = new Map(probeMeta);
				for (const [path, meta] of pairs) {
					next.set(path, meta);
				}
				probeMeta = next;
			} finally {
				if (requestID === metaLoadRequest) {
					timelineLoading = false;
				}
			}
		}
</script>

<div
	class="fixed top-0 right-0 h-screen w-1/2 bg-background border-l border-border shadow-lg z-50 flex flex-col"
	role="dialog"
	aria-modal="true"
>
	<!-- Header -->
	<div class="border-b border-border px-6 py-4 flex items-center justify-between">
		<div class="min-w-0">
			<h2 class="text-lg font-semibold text-foreground">Artifact Details</h2>
			{#if probeModelName}
				<div class="mt-1 text-xs text-muted-foreground truncate" data-testid="probe-lineage-breadcrumb">
					Model: {probeModelName} &gt; Probe: {getProbeSlug(artifact.path)}
				</div>
			{/if}
		</div>
		<button
			on:click={handleClose}
			class="text-muted-foreground hover:text-foreground transition-colors"
		>
			<svg
				xmlns="http://www.w3.org/2000/svg"
				width="20"
				height="20"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<line x1="18" y1="6" x2="6" y2="18" />
				<line x1="6" y1="6" x2="18" y2="18" />
			</svg>
		</button>
	</div>

	<!-- Content -->
	<div class="flex-1 overflow-auto px-6 py-4">
		{#if loading}
			<div class="flex items-center justify-center h-full">
				<div class="text-muted-foreground">Loading...</div>
			</div>
		{:else if error}
			<div class="text-red-500">Error: {error}</div>
		{:else}
			{#if probeModelName}
				<div class="mb-4 rounded-md border border-border bg-muted/20 p-3 space-y-3" data-testid="probe-lineage-card">
					<div class="flex flex-wrap items-center gap-2">
						<span class="text-xs uppercase tracking-wide text-muted-foreground">Probe lineage</span>
						<span
							class="text-xs px-2 py-0.5 rounded border {getVerdictClass(currentProbeMeta?.verdict ?? null)}"
							data-testid="probe-verdict-chip"
						>
							{getVerdictLabel(currentProbeMeta?.verdict ?? null)}
						</span>
					</div>

					{#if currentProbeMeta?.claim}
						<p class="text-sm text-foreground" data-testid="probe-claim-excerpt">{currentProbeMeta.claim}</p>
					{/if}

					{#if modelArtifact}
						<button
							type="button"
							on:click={() => selectArtifact(modelArtifact)}
							class="text-xs text-blue-500 hover:text-blue-400 underline-offset-2 hover:underline"
							data-testid="probe-parent-model-link"
						>
							Open parent model
						</button>
					{/if}

					<div class="grid grid-cols-2 gap-2">
						<button
							type="button"
							on:click={() => selectArtifact(prevProbe)}
							disabled={!prevProbe}
							class="text-xs border border-border rounded px-2 py-1 text-left disabled:opacity-40 disabled:cursor-not-allowed hover:border-foreground/30"
							data-testid="probe-prev-link"
						>
							Prev probe
						</button>
						<button
							type="button"
							on:click={() => selectArtifact(nextProbe)}
							disabled={!nextProbe}
							class="text-xs border border-border rounded px-2 py-1 text-left disabled:opacity-40 disabled:cursor-not-allowed hover:border-foreground/30"
							data-testid="probe-next-link"
						>
							Next probe
						</button>
					</div>
				</div>
			{:else if modelName && modelProbes.length > 0}
				<div class="mb-4 rounded-md border border-border bg-muted/20 p-3" data-testid="model-probe-timeline-card">
					<button
						type="button"
						on:click={() => (timelineExpanded = !timelineExpanded)}
						class="w-full text-left flex items-center justify-between"
						data-testid="model-probe-timeline-toggle"
					>
						<span class="text-sm font-medium text-foreground">Probe timeline ({modelProbes.length})</span>
						<span class="text-xs text-muted-foreground">{timelineExpanded ? 'Hide' : 'Show'}</span>
					</button>

					{#if timelineExpanded}
						<div class="mt-3 space-y-2 max-h-64 overflow-y-auto pr-1" data-testid="model-probe-timeline-list">
							{#if timelineLoading}
								<p class="text-xs text-muted-foreground">Loading probe metadata...</p>
							{/if}
							{#each modelProbes as probe (probe.path)}
								{@const meta = probeMeta.get(probe.path)}
								<button
									type="button"
									on:click={() => selectArtifact(probe)}
									class="w-full text-left rounded border border-border px-2 py-2 hover:border-foreground/30"
								>
									<div class="flex items-center gap-2">
										<span class="text-xs px-2 py-0.5 rounded border {getVerdictClass(meta?.verdict ?? null)}">
											{getVerdictLabel(meta?.verdict ?? null)}
										</span>
										<span class="text-sm text-foreground truncate">{probe.title}</span>
									</div>
									<div class="mt-1 text-xs text-muted-foreground">{probe.date || probe.relative_time}</div>
								</button>
							{/each}
						</div>
					{/if}
				</div>
			{/if}

			<MarkdownContent content={content} />
		{/if}
	</div>

	<!-- Footer -->
	<div class="border-t border-border px-6 py-3 text-xs text-muted-foreground">
		Press <kbd class="px-1 py-0.5 bg-muted rounded">h</kbd> or
		<kbd class="px-1 py-0.5 bg-muted rounded">Esc</kbd> to close
	</div>
</div>
