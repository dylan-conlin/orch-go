// Deliverables tracking for work graph issues
// Defines expected artifacts per issue type + skill combination

export interface Deliverable {
	id: string;              // Unique identifier (e.g., "code_committed", "tests_pass")
	label: string;           // Display name (e.g., "Code committed")
	status: 'complete' | 'incomplete' | 'skipped'; // Completion state
	artifact_link?: string;  // Optional link to artifact (commit, file, etc.)
	override_reason?: string; // If skipped, reason logged
}

export interface DeliverableSet {
	expected: Deliverable[];  // Expected deliverables for this type+skill
	actual: Deliverable[];    // Actual deliverables (may include extras)
}

// Schema: Expected deliverables per issue type + skill combination
// Based on design doc examples
export const DELIVERABLE_SCHEMAS: Record<string, Record<string, string[]>> = {
	// Issue type → Skill → Deliverable IDs
	bug: {
		'feature-impl': ['code_committed', 'tests_pass', 'visual_verification', 'synthesis'],
		'systematic-debugging': ['root_cause_identified', 'fix_committed', 'tests_pass', 'synthesis'],
	},
	task: {
		'feature-impl': ['code_committed', 'tests_pass', 'synthesis'],
	},
	feature: {
		'feature-impl': ['code_committed', 'tests_pass', 'visual_verification', 'synthesis'],
	},
	investigation: {
		investigation: ['investigation_artifact', 'recommendation'],
	},
	question: {
		'design-session': ['design_brief', 'mockups', 'decision_or_epic'],
		architect: ['investigation_with_recommendation', 'decision_record'],
	},
};

// Deliverable metadata (labels, descriptions)
export const DELIVERABLE_METADATA: Record<string, { label: string; description: string }> = {
	code_committed: {
		label: 'Code committed',
		description: 'Changes committed to git',
	},
	tests_pass: {
		label: 'Tests passing',
		description: 'Test suite passes without failures',
	},
	visual_verification: {
		label: 'Visual verification',
		description: 'UI changes verified visually (screenshot/manual check)',
	},
	synthesis: {
		label: 'SYNTHESIS.md',
		description: 'Session synthesis document created',
	},
	root_cause_identified: {
		label: 'Root cause identified',
		description: 'Investigation identified root cause',
	},
	fix_committed: {
		label: 'Fix committed',
		description: 'Bug fix committed to git',
	},
	investigation_artifact: {
		label: 'Investigation artifact',
		description: 'Investigation file created in .kb/investigations/',
	},
	recommendation: {
		label: 'Recommendation',
		description: 'Investigation includes actionable recommendation',
	},
	design_brief: {
		label: 'Design brief',
		description: 'Design document created',
	},
	mockups: {
		label: 'Mockups',
		description: 'UI mockups created (if applicable)',
	},
	decision_or_epic: {
		label: 'Decision or Epic created',
		description: 'Design session resulted in decision record or epic',
	},
	investigation_with_recommendation: {
		label: 'Investigation with recommendation',
		description: 'Investigation file with recommendation created',
	},
	decision_record: {
		label: 'Decision record',
		description: 'Decision documented in .kb/decisions/',
	},
};

// Get expected deliverables for an issue type + skill combination
export function getExpectedDeliverables(issueType: string, skill?: string): Deliverable[] {
	const skillSchemas = DELIVERABLE_SCHEMAS[issueType];
	if (!skillSchemas || !skill) {
		return [];
	}

	const deliverableIds = skillSchemas[skill];
	if (!deliverableIds) {
		return [];
	}

	return deliverableIds.map((id) => ({
		id,
		label: DELIVERABLE_METADATA[id]?.label || id,
		status: 'incomplete' as const,
	}));
}

// Compute completion stats for a deliverable set
export function getCompletionStats(deliverables: Deliverable[]): {
	total: number;
	complete: number;
	incomplete: number;
	skipped: number;
	percentage: number;
} {
	const total = deliverables.length;
	const complete = deliverables.filter((d) => d.status === 'complete').length;
	const incomplete = deliverables.filter((d) => d.status === 'incomplete').length;
	const skipped = deliverables.filter((d) => d.status === 'skipped').length;
	const percentage = total > 0 ? Math.round((complete / total) * 100) : 0;

	return { total, complete, incomplete, skipped, percentage };
}
