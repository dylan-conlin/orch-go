import { writable } from 'svelte/store';

// API configuration
const API_BASE = 'http://localhost:3348';

// Types matching the Go HarnessResponse
export interface PipelineComponent {
	name: string;
	type: 'hard' | 'soft' | 'human';
	measurement_status: 'flowing' | 'proxy_only' | 'unmeasured' | 'collecting';
	fire_rate?: number;
	block_rate?: number;
	bypass_rate?: number;
	fail_rate?: number;
	pass_rate?: number;
	bypassed?: number;
	blocked?: number;
	last_fired?: string;
	proxy_metric?: string;
	collecting_since?: string;
}

export interface PipelineStage {
	stage: string;
	components: PipelineComponent[];
}

export interface CompletionCoverage {
	total_completions: number;
	with_skill: number;
	with_outcome: number;
	with_duration: number;
	coverage_pct: number;
}

export interface FalsificationVerdict {
	criterion: string;
	status: 'falsified' | 'confirmed' | 'insufficient_data' | 'not_measurable';
	evidence: string;
	threshold: string;
}

export interface MeasurementCoverage {
	total_components: number;
	with_measurement: number;
	proxy_only: number;
	unmeasured: number;
}

export interface ExplorationMetrics {
	total_runs: number;
	completed_runs: number;
	total_findings: number;
	total_accepted: number;
	total_contested: number;
	total_rejected: number;
	total_gaps: number;
	avg_workers_per_run: number;
}

export interface HarnessData {
	generated_at: string;
	analysis_period: string;
	total_spawns: number;
	pipeline: PipelineStage[];
	completion_coverage: CompletionCoverage;
	falsification_verdicts: Record<string, FalsificationVerdict>;
	measurement_coverage: MeasurementCoverage;
	exploration_metrics?: ExplorationMetrics;
}

const emptyData: HarnessData = {
	generated_at: '',
	analysis_period: '',
	total_spawns: 0,
	pipeline: [],
	completion_coverage: {
		total_completions: 0,
		with_skill: 0,
		with_outcome: 0,
		with_duration: 0,
		coverage_pct: 0
	},
	falsification_verdicts: {},
	measurement_coverage: {
		total_components: 13,
		with_measurement: 0,
		proxy_only: 0,
		unmeasured: 13
	}
};

function createHarnessStore() {
	const { subscribe, set } = writable<HarnessData>(emptyData);

	return {
		subscribe,
		async fetch(days: number = 7): Promise<void> {
			try {
				const response = await fetch(`${API_BASE}/api/harness?days=${days}`);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data: HarnessData = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch harness data:', error);
				set(emptyData);
			}
		}
	};
}

export const harness = createHarnessStore();

// Display helpers

export function stageLabel(stage: string): string {
	const labels: Record<string, string> = {
		spawn: 'Spawn',
		authoring: 'Authoring',
		pre_commit: 'Pre-Commit',
		completion: 'Completion'
	};
	return labels[stage] || stage;
}

export function componentLabel(name: string): string {
	const labels: Record<string, string> = {
		triage_gate: 'Triage Gate',
		hotspot_gate: 'Hotspot Gate',
		verification_gate: 'Verification Gate',
		claude_md: 'CLAUDE.md',
		spawn_context: 'SPAWN_CONTEXT.md',
		kb_knowledge: '.kb/ Knowledge',
		accretion_gate: 'Accretion Gate',
		build_gate: 'Build Gate',
		verification_pipeline: 'Verify Pipeline',
		explain_back: 'Explain-Back'
	};
	return labels[name] || name;
}

export function typeColor(type: string): string {
	switch (type) {
		case 'hard': return 'text-blue-400 border-blue-500/50 bg-blue-500/10';
		case 'soft': return 'text-yellow-400 border-yellow-500/50 bg-yellow-500/10';
		case 'human': return 'text-green-400 border-green-500/50 bg-green-500/10';
		default: return 'text-muted-foreground border-border bg-muted';
	}
}

export function measurementIcon(status: string): string {
	switch (status) {
		case 'flowing': return '●';
		case 'collecting': return '◑';
		case 'proxy_only': return '◐';
		case 'unmeasured': return '○';
		default: return '?';
	}
}

export function measurementColor(status: string): string {
	switch (status) {
		case 'flowing': return 'text-green-400';
		case 'collecting': return 'text-blue-400';
		case 'proxy_only': return 'text-yellow-400';
		case 'unmeasured': return 'text-muted-foreground';
		default: return 'text-muted-foreground';
	}
}

export function verdictIcon(status: string): string {
	switch (status) {
		case 'falsified': return '✓';
		case 'confirmed': return '✗';
		case 'insufficient_data': return '…';
		case 'not_measurable': return '?';
		default: return '?';
	}
}

export function verdictColor(status: string): string {
	switch (status) {
		case 'falsified': return 'text-green-400 border-green-500/50 bg-green-500/10';
		case 'confirmed': return 'text-red-400 border-red-500/50 bg-red-500/10';
		case 'insufficient_data': return 'text-muted-foreground border-border bg-muted';
		case 'not_measurable': return 'text-muted-foreground border-border bg-muted/50';
		default: return 'text-muted-foreground border-border bg-muted';
	}
}

export function verdictLabel(status: string): string {
	switch (status) {
		case 'falsified': return 'FALSIFIED';
		case 'confirmed': return 'CONFIRMED';
		case 'insufficient_data': return 'INSUFFICIENT DATA';
		case 'not_measurable': return 'NOT MEASURABLE';
		default: return status.toUpperCase();
	}
}

export function formatRate(rate: number | undefined | null): string {
	if (rate === undefined || rate === null) return '—';
	return `${(rate * 100).toFixed(1)}%`;
}

// Plain-language verdict descriptions (eliminates double negatives)
export function verdictPlainLanguage(key: string, status: string): string {
	const plain: Record<string, Record<string, string>> = {
		gates_are_irrelevant: {
			falsified: 'Gates are relevant',
			confirmed: 'Gates rarely fire — may be irrelevant',
			insufficient_data: 'Not enough data yet'
		},
		gates_are_ceremony: {
			falsified: 'Gates slow accretion',
			confirmed: 'Gates ship but accretion unchanged',
			insufficient_data: 'Collecting data (checkpoint Mar 24)'
		},
		soft_harness_is_inert: {
			falsified: 'Soft harness affects behavior',
			confirmed: 'Soft harness has no measurable effect',
			not_measurable: 'No controlled experiment yet'
		},
		framework_is_anecdotal: {
			falsified: 'Framework works in a second system',
			confirmed: 'No benefit in second system',
			not_measurable: 'No second system instrumented'
		}
	};
	return plain[key]?.[status] || `${key}: ${status}`;
}

// Generate a top-level summary from harness data
export function harnessSummary(data: HarnessData): string {
	if (data.total_spawns === 0) return 'No spawn data yet.';

	const parts: string[] = [];

	// Gates status from verdicts
	const gatesVerdict = data.falsification_verdicts['gates_are_irrelevant'];
	if (gatesVerdict?.status === 'falsified') {
		const spawnStage = data.pipeline.find(s => s.stage === 'spawn');
		const totalFireRate = spawnStage?.components.reduce((sum, c) => sum + (c.fire_rate || 0), 0) || 0;
		parts.push(`Gates are active — ${formatRate(totalFireRate / (spawnStage?.components.length || 1))} average fire rate across ${data.total_spawns} spawns`);
	} else if (gatesVerdict?.status === 'confirmed') {
		parts.push('Gates are deployed but rarely fire');
	} else {
		parts.push('Collecting gate data');
	}

	// Accretion verdict
	const accretionVerdict = data.falsification_verdicts['gates_are_ceremony'];
	if (accretionVerdict?.status === 'insufficient_data') {
		parts.push('Accretion impact pending (verdict Mar 24)');
	} else if (accretionVerdict?.status === 'falsified') {
		parts.push('Accretion is slowing');
	}

	return parts.join('. ') + '.';
}

// Stage description for context
export function stageDescription(stage: string): string {
	const descriptions: Record<string, string> = {
		spawn: 'Gates that check work before agents start',
		authoring: 'Context that shapes agent behavior — no direct measurement',
		pre_commit: 'Gates that check code before it lands',
		completion: 'Verification after agent work finishes'
	};
	return descriptions[stage] || '';
}

// Coverage framing text
export function coverageFraming(coverage: MeasurementCoverage): string {
	const measuredPct = Math.round(((coverage.with_measurement + coverage.proxy_only) / coverage.total_components) * 100);
	if (coverage.unmeasured === 0) return 'All components measured.';
	if (measuredPct < 50) return `Less than half measured — ${coverage.unmeasured} components have no data.`;
	if (measuredPct < 80) return `${coverage.unmeasured} components still unmeasured.`;
	return `Most components covered. ${coverage.unmeasured} remaining.`;
}
