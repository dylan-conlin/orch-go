import type { AgentLogEvent } from '$lib/stores/agentlog';
import type { DaemonStatus } from '$lib/stores/daemon';

export function formatEventAge(timestamp: number): string {
	const now = Math.floor(Date.now() / 1000);
	const diff = now - timestamp;
	if (diff < 60) return 'now';
	if (diff < 3600) return `${Math.floor(diff / 60)}m`;
	if (diff < 86400) return `${Math.floor(diff / 3600)}h`;
	return `${Math.floor(diff / 86400)}d`;
}

export function eventIcon(type: string): string {
	switch (type) {
		case 'session.spawned': return '⚡';
		case 'agent.completed':
		case 'session.auto_completed': return '✓';
		case 'session.error':
		case 'verification.failed': return '✗';
		case 'agent.abandoned': return '⊘';
		case 'agent.reworked': return '↻';
		default: return '•';
	}
}

export function eventColorClass(type: string): string {
	switch (type) {
		case 'session.spawned': return 'text-blue-400';
		case 'agent.completed':
		case 'session.auto_completed': return 'text-emerald-400';
		case 'session.error':
		case 'verification.failed': return 'text-red-400';
		case 'agent.abandoned': return 'text-amber-400';
		case 'agent.reworked': return 'text-purple-400';
		default: return 'text-muted-foreground';
	}
}

export function eventLabel(type: string): string {
	switch (type) {
		case 'session.spawned': return 'spawned';
		case 'agent.completed': return 'completed';
		case 'session.auto_completed': return 'auto-closed';
		case 'session.error': return 'error';
		case 'agent.abandoned': return 'abandoned';
		case 'agent.reworked': return 'rework';
		case 'verification.failed': return 'verify failed';
		default: return type;
	}
}

export function eventTarget(event: AgentLogEvent): string {
	const data = event.data || {};
	const id = data.beads_id || event.session_id || '';
	const skill = data.skill || '';
	if (event.type === 'session.spawned' && skill) {
		return `${id} (${skill})`;
	}
	return id;
}

export function daemonQueueSummary(status: DaemonStatus): string {
	const queued = status.queue?.queued ?? status.ready_count ?? 0;
	const reasons: string[] = [];
	if ((status.queue?.waiting_for_slots ?? 0) > 0) {
		reasons.push(`${status.queue?.waiting_for_slots} waiting for slots`);
	}
	if ((status.queue?.grace_period ?? 0) > 0) {
		reasons.push(`${status.queue?.grace_period} in grace period`);
	}
	if ((status.queue?.processed_cache ?? 0) > 0) {
		reasons.push(`${status.queue?.processed_cache} in processed cache`);
	}

	if (queued === 0 || reasons.length === 0) {
		return `${queued} queued`;
	}

	return `${queued} queued (${reasons.join(', ')})`;
}
