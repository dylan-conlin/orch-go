import { writable } from 'svelte/store';

// API configuration
const API_BASE = 'https://localhost:3348';

// Types matching the Go HarnessResponse
export interface PipelineComponent {
	name: string;
	type: 'hard' | 'soft' | 'human';
	measurement_status: 'flowing' | 'proxy_only' | 'unmeasured';
	fire_rate?: number;
	block_rate?: number;
	bypass_rate?: number;
	fail_rate?: number;
	pass_rate?: number;
	bypassed?: number;
	blocked?: number;
	last_fired?: string;
	proxy_metric?: string;
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

export interface HarnessData {
	generated_at: string;
	analysis_period: string;
	total_spawns: number;
	pipeline: PipelineStage[];
	completion_coverage: CompletionCoverage;
	falsification_verdicts: Record<string, FalsificationVerdict>;
	measurement_coverage: MeasurementCoverage;
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
		case 'proxy_only': return '◐';
		case 'unmeasured': return '○';
		default: return '?';
	}
}

export function measurementColor(status: string): string {
	switch (status) {
		case 'flowing': return 'text-green-400';
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
