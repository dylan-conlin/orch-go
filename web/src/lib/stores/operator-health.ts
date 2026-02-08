import { writable } from 'svelte/store';

const API_BASE = 'https://localhost:3348';

export type HealthStatus = 'healthy' | 'warning' | 'critical' | 'unknown';

export interface OperatorInterventionSummary {
	timestamp: string;
	source: string;
	detail?: string;
	beads_id?: string;
}

export interface CrashFreeStreakMetric {
	status: HealthStatus;
	current_streak_days: number;
	current_streak_seconds: number;
	current_streak: string;
	target_days: number;
	progress_percent: number;
	last_intervention?: OperatorInterventionSummary;
}

export interface ResourceMetricValues {
	goroutines: number;
	heap_bytes: number;
	child_processes: number;
	open_file_descriptors: number;
}

export interface ResourceBreach {
	metric: string;
	baseline: number;
	current: number;
	threshold: number;
	ratio: number;
}

export interface ResourceCeilingsMetric {
	status: HealthStatus;
	baseline: ResourceMetricValues;
	current: ResourceMetricValues;
	ceiling_multiplier: number;
	breached: boolean;
	breaches?: ResourceBreach[];
	baseline_errors?: Record<string, string>;
	current_errors?: Record<string, string>;
}

export interface InvestigationRateMetric {
	status: HealthStatus;
	window_days: number;
	count: number;
	threshold: number;
	warning_from: number;
}

export interface DefectClassClusterItem {
	defect_class: string;
	count: number;
	window_days?: number;
}

export interface DefectClassClustersMetric {
	status: HealthStatus;
	window_days: number;
	top_classes: DefectClassClusterItem[];
	total_top_n: number;
}

export interface AgentHealthRatioMetric {
	status: HealthStatus;
	window_days: number;
	completions: number;
	abandonments: number;
	completion_share: number;
	completions_per_abandonment?: number;
}

export interface OrphanProcess {
	pid: number;
	ppid: number;
	command: string;
	args?: string;
}

export interface ProcessCensusMetric {
	status: HealthStatus;
	child_processes: number;
	orphaned_count: number;
	orphaned_processes?: OrphanProcess[];
}

export interface OperatorHealthState {
	generated_at: string;
	crash_free_streak: CrashFreeStreakMetric;
	resource_ceilings: ResourceCeilingsMetric;
	investigation_rate_30d: InvestigationRateMetric;
	defect_class_clusters: DefectClassClustersMetric;
	agent_health_ratio_7d: AgentHealthRatioMetric;
	process_census: ProcessCensusMetric;
	errors?: string[];
}

const defaultState: OperatorHealthState = {
	generated_at: '',
	crash_free_streak: {
		status: 'unknown',
		current_streak_days: 0,
		current_streak_seconds: 0,
		current_streak: 'No data',
		target_days: 7,
		progress_percent: 0
	},
	resource_ceilings: {
		status: 'unknown',
		baseline: {
			goroutines: 0,
			heap_bytes: 0,
			child_processes: 0,
			open_file_descriptors: 0
		},
		current: {
			goroutines: 0,
			heap_bytes: 0,
			child_processes: 0,
			open_file_descriptors: 0
		},
		ceiling_multiplier: 2,
		breached: false
	},
	investigation_rate_30d: {
		status: 'unknown',
		window_days: 30,
		count: 0,
		threshold: 50,
		warning_from: 40
	},
	defect_class_clusters: {
		status: 'unknown',
		window_days: 30,
		top_classes: [],
		total_top_n: 0
	},
	agent_health_ratio_7d: {
		status: 'unknown',
		window_days: 7,
		completions: 0,
		abandonments: 0,
		completion_share: 0
	},
	process_census: {
		status: 'unknown',
		child_processes: 0,
		orphaned_count: 0,
		orphaned_processes: []
	}
};

function createOperatorHealthStore() {
	const { subscribe, set } = writable<OperatorHealthState>(defaultState);

	return {
		subscribe,
		set,
		async fetch(projectDir?: string): Promise<void> {
			try {
				const url = projectDir
					? `${API_BASE}/api/operator-health?project=${encodeURIComponent(projectDir)}`
					: `${API_BASE}/api/operator-health`;
				const response = await fetch(url);
				if (!response.ok) {
					throw new Error(`HTTP ${response.status}: ${response.statusText}`);
				}
				const data: OperatorHealthState = await response.json();
				set(data);
			} catch (error) {
				console.error('Failed to fetch operator health:', error);
				set({
					...defaultState,
					errors: [error instanceof Error ? error.message : 'Unknown fetch error']
				});
			}
		}
	};
}

export const operatorHealth = createOperatorHealthStore();
