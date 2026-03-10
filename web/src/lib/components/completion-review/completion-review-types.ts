export type EscalationLevel = 'safe' | 'review' | 'blocked';

export interface ReadyToCompleteItem {
	id: string;
	title: string;
	type: string;
	priority: number;
	skill?: string;
	outcome?: string;
	recommendation?: string;
	nextActions?: string[];
	runtime?: string;
	tokenTotal: number | null;
	completionAt: string;
	tldr?: string;
	deltaSummary?: string;
	escalation: EscalationLevel;
}

// Map server escalation level (5-tier) to client EscalationLevel (3-tier).
export function mapServerEscalation(serverLevel?: string): EscalationLevel | undefined {
	if (!serverLevel) return undefined;
	switch (serverLevel) {
		case 'block':
		case 'failed':
			return 'blocked';
		case 'review':
			return 'review';
		case 'none':
		case 'info':
			return 'safe';
		default:
			return undefined;
	}
}

export function computeEscalation(item: {
	serverEscalation?: string;
	outcome?: string;
	recommendation?: string;
	nextActions?: string[];
	skill?: string;
}): EscalationLevel {
	const mapped = mapServerEscalation(item.serverEscalation);
	if (mapped !== undefined) return mapped;
	if (item.outcome === 'failed' || item.outcome === 'blocked') return 'blocked';
	if (item.outcome === 'partial') return 'review';
	if (item.recommendation === 'escalate') return 'blocked';
	if (item.recommendation === 'continue' || item.recommendation === 'resume') return 'review';
	const knowledgeSkills = new Set(['investigation', 'architect', 'research', 'design-session', 'codebase-audit', 'issue-creation']);
	if (item.skill && knowledgeSkills.has(item.skill)) return 'review';
	return 'safe';
}
