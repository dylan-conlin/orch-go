<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { decisions } from '$lib/stores/decisions';
	import { kbHealth } from '$lib/stores/kb-health';

	export let expanded: boolean = true;

	function toggle() {
		expanded = !expanded;
	}

	// Calculate total count across all categories
	$: totalCount = $decisions 
		? ($decisions.absorb_knowledge.length + 
		   $decisions.give_approvals.length + 
		   $decisions.answer_questions.length + 
		   $decisions.handle_failures.length +
		   ($kbHealth?.total ?? 0))
		: 0;

	function formatAge(timestamp?: string): string {
		if (!timestamp) return '';
		const created = new Date(timestamp);
		const now = new Date();
		const diffMs = now.getTime() - created.getTime();
		const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
		const diffDays = Math.floor(diffHours / 24);

		if (diffDays > 0) {
			return `${diffDays}d ago`;
		} else if (diffHours > 0) {
			return `${diffHours}h ago`;
		} else {
			const diffMinutes = Math.floor(diffMs / (1000 * 60));
			return `${diffMinutes}m ago`;
		}
	}
</script>

{#if totalCount > 0}
	<div class="rounded-lg border border-indigo-500/30 bg-indigo-500/5" data-testid="decision-center-section">
		<button
			class="flex w-full items-center justify-between px-3 py-2 text-left hover:bg-accent/50 transition-colors border-b"
			onclick={toggle}
			aria-expanded={expanded}
			data-testid="decision-center-toggle"
		>
			<div class="flex items-center gap-2 min-w-0 flex-1">
				<span class="text-sm flex-shrink-0">🎯</span>
				<span class="text-sm font-medium flex-shrink-0">Strategic Center</span>
				<Badge variant="secondary" class="h-5 px-1.5 text-xs flex-shrink-0">
					{totalCount}
				</Badge>
			</div>
			<span class="text-muted-foreground transition-transform flex-shrink-0 {expanded ? 'rotate-180' : ''}">
				<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="6 9 12 15 18 9"></polyline>
				</svg>
			</span>
		</button>

		{#if expanded}
			<div class="p-2 space-y-2">
				<!-- Absorb Knowledge -->
				{#if $decisions.absorb_knowledge.length > 0}
					<div class="rounded border bg-card p-2">
						<div class="flex items-center gap-2 mb-2">
							<span class="text-sm">📚</span>
							<span class="text-xs font-medium">Absorb Knowledge</span>
							<Badge variant="outline" class="h-5 px-1.5 text-xs">
								{$decisions.absorb_knowledge.length}
							</Badge>
						</div>
						<div class="space-y-1">
							{#each $decisions.absorb_knowledge as item}
								<div class="rounded border border-border/50 p-2 hover:bg-accent/30 transition-colors">
									<div class="flex items-start justify-between gap-2">
										<div class="flex-1 min-w-0">
											<div class="text-xs font-medium truncate">
												{item.title || item.id}
											</div>
											{#if item.tldr}
												<div class="text-[10px] text-muted-foreground mt-0.5 line-clamp-2">
													{item.tldr}
												</div>
											{/if}
											{#if item.skill}
												<Badge variant="outline" class="h-4 px-1 text-[10px] mt-1">
													{item.skill}
												</Badge>
											{/if}
										</div>
										{#if item.completed_at}
											<span class="text-[10px] text-muted-foreground flex-shrink-0">
												{formatAge(item.completed_at)}
											</span>
										{/if}
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Give Approvals -->
				{#if $decisions.give_approvals.length > 0}
					<div class="rounded border bg-card p-2 border-amber-500/30">
						<div class="flex items-center gap-2 mb-2">
							<span class="text-sm">✅</span>
							<span class="text-xs font-medium text-amber-600">Give Approvals</span>
							<Badge variant="secondary" class="h-5 px-1.5 text-xs bg-amber-500/20 text-amber-600">
								{$decisions.give_approvals.length}
							</Badge>
						</div>
						<div class="space-y-1">
							{#each $decisions.give_approvals as item}
								<div class="rounded border border-amber-500/30 p-2 hover:bg-amber-500/10 transition-colors">
									<div class="flex items-start justify-between gap-2">
										<div class="flex-1 min-w-0">
											<div class="text-xs font-medium truncate">
												{item.title || item.id}
											</div>
											<div class="text-[10px] text-amber-600 mt-0.5">
												Visual verification needed
											</div>
											{#if item.skill}
												<Badge variant="outline" class="h-4 px-1 text-[10px] mt-1">
													{item.skill}
												</Badge>
											{/if}
										</div>
										{#if item.completed_at}
											<span class="text-[10px] text-muted-foreground flex-shrink-0">
												{formatAge(item.completed_at)}
											</span>
										{/if}
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Answer Questions -->
				{#if $decisions.answer_questions.length > 0}
					<div class="rounded border bg-card p-2 border-purple-500/30">
						<div class="flex items-center gap-2 mb-2">
							<span class="text-sm">❓</span>
							<span class="text-xs font-medium text-purple-600">Answer Questions</span>
							<Badge variant="secondary" class="h-5 px-1.5 text-xs bg-purple-500/20 text-purple-600">
								{$decisions.answer_questions.length}
							</Badge>
						</div>
						<div class="space-y-1">
							{#each $decisions.answer_questions as item}
								<div class="rounded border border-purple-500/30 p-2 hover:bg-purple-500/10 transition-colors">
									<div class="flex items-start justify-between gap-2">
										<div class="flex-1 min-w-0">
											<div class="text-xs font-medium truncate">
												{item.title || item.id}
											</div>
											{#if item.next_actions && item.next_actions.length > 0}
												<div class="text-[10px] text-purple-600 mt-0.5">
													Blocks: {item.next_actions.join(', ')}
												</div>
											{/if}
										</div>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Handle Failures -->
				{#if $decisions.handle_failures.length > 0}
					<div class="rounded border bg-card p-2 border-red-500/30">
						<div class="flex items-center gap-2 mb-2">
							<span class="text-sm">❌</span>
							<span class="text-xs font-medium text-red-600">Handle Failures</span>
							<Badge variant="destructive" class="h-5 px-1.5 text-xs">
								{$decisions.handle_failures.length}
							</Badge>
						</div>
						<div class="space-y-1">
							{#each $decisions.handle_failures as item}
								<div class="rounded border border-red-500/30 p-2 hover:bg-red-500/10 transition-colors">
									<div class="flex items-start justify-between gap-2">
										<div class="flex-1 min-w-0">
											<div class="text-xs font-medium truncate">
												{item.title || item.id}
											</div>
											{#if item.escalation_reason}
												<div class="text-[10px] text-red-600 mt-0.5 line-clamp-2">
													{item.escalation_reason}
												</div>
											{/if}
											{#if item.skill}
												<Badge variant="outline" class="h-4 px-1 text-[10px] mt-1">
													{item.skill}
												</Badge>
											{/if}
										</div>
										{#if item.completed_at}
											<span class="text-[10px] text-muted-foreground flex-shrink-0">
												{formatAge(item.completed_at)}
											</span>
										{/if}
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Tend Knowledge -->
				{#if $kbHealth && $kbHealth.total > 0}
					<div class="rounded border bg-card p-2 border-blue-500/30">
						<div class="flex items-center gap-2 mb-2">
							<span class="text-sm">🌱</span>
							<span class="text-xs font-medium text-blue-600">Tend Knowledge</span>
							<Badge variant="secondary" class="h-5 px-1.5 text-xs bg-blue-500/20 text-blue-600">
								{$kbHealth.total}
							</Badge>
						</div>
						<div class="space-y-1">
							<!-- Synthesis Opportunities -->
							{#if $kbHealth.synthesis.count > 0}
								<div class="text-[10px] text-muted-foreground mb-1">Synthesis ({$kbHealth.synthesis.count})</div>
								{#each $kbHealth.synthesis.items as item}
									<div class="rounded border border-border/50 p-2 hover:bg-accent/30 transition-colors">
										<div class="text-xs font-medium">
											{item.topic || 'Unknown topic'}
										</div>
										<div class="text-[10px] text-muted-foreground mt-0.5">
											{item.count} investigations
										</div>
									</div>
								{/each}
							{/if}

							<!-- Pending Promotions -->
							{#if $kbHealth.promote.count > 0}
								<div class="text-[10px] text-muted-foreground mb-1 {$kbHealth.synthesis.count > 0 ? 'mt-2' : ''}">Promote ({$kbHealth.promote.count})</div>
								{#each $kbHealth.promote.items as item}
									<div class="rounded border border-border/50 p-2 hover:bg-accent/30 transition-colors">
										<div class="text-xs font-medium truncate">
											{item.value || 'Unknown entry'}
										</div>
										<Badge variant="outline" class="h-4 px-1 text-[10px] mt-0.5">
											{item.type}
										</Badge>
									</div>
								{/each}
							{/if}

							<!-- Stale Decisions -->
							{#if $kbHealth.stale.count > 0}
								<div class="text-[10px] text-muted-foreground mb-1 {$kbHealth.synthesis.count > 0 || $kbHealth.promote.count > 0 ? 'mt-2' : ''}">Stale ({$kbHealth.stale.count})</div>
								{#each $kbHealth.stale.items as item}
									<div class="rounded border border-border/50 p-2 hover:bg-accent/30 transition-colors">
										<div class="text-xs font-medium truncate">
											{item.title || item.path}
										</div>
										<div class="text-[10px] text-muted-foreground mt-0.5">
											{item.age_days}d without citations
										</div>
									</div>
								{/each}
							{/if}

							<!-- Investigation Promotions -->
							{#if $kbHealth.investigation_promotion.count > 0}
								<div class="text-[10px] text-muted-foreground mb-1 {$kbHealth.synthesis.count > 0 || $kbHealth.promote.count > 0 || $kbHealth.stale.count > 0 ? 'mt-2' : ''}">
									Investigation Promotions ({$kbHealth.investigation_promotion.count})
								</div>
								{#each $kbHealth.investigation_promotion.items as item}
									<div class="rounded border border-border/50 p-2 hover:bg-accent/30 transition-colors">
										<div class="text-xs font-medium truncate">
											{item.title || item.path}
										</div>
										<Badge variant="outline" class="h-4 px-1 text-[10px] mt-0.5 border-green-500/30 text-green-600">
											{item.suggestion}
										</Badge>
									</div>
								{/each}
							{/if}

							<!-- Recurring Defect Classes -->
							{#if $kbHealth.defect_class.count > 0}
								<div class="text-[10px] text-muted-foreground mb-1 {$kbHealth.synthesis.count > 0 || $kbHealth.promote.count > 0 || $kbHealth.stale.count > 0 || $kbHealth.investigation_promotion.count > 0 ? 'mt-2' : ''}">
									Recurring Defects ({$kbHealth.defect_class.count})
								</div>
								{#each $kbHealth.defect_class.items as item}
									<div class="rounded border border-border/50 p-2 hover:bg-accent/30 transition-colors">
										<div class="text-xs font-medium truncate">
											{item.defect_class || 'Unknown defect class'}
										</div>
										<div class="text-[10px] text-muted-foreground mt-0.5">
											{item.count} investigations in {item.window_days || 30}d
										</div>
									</div>
								{/each}
							{/if}
						</div>
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}
