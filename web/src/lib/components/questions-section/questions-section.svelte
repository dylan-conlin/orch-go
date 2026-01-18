<script lang="ts">
	import { Badge } from '$lib/components/ui/badge';
	import { questions, type Question } from '$lib/stores/questions';

	export let expanded: boolean = false;

	function toggle() {
		expanded = !expanded;
	}

	function getPreview(questionsList: Question[]): string {
		if (questionsList.length === 0) return '';

		const titles = questionsList.slice(0, 2).map(q => {
			const title = q.title;
			return title.length > 30 ? title.substring(0, 30) + '...' : title;
		});

		if (questionsList.length <= 2) {
			return titles.join(', ');
		}

		return `${titles.join(', ')} +${questionsList.length - 2}`;
	}

	function getStatusClass(status: string): string {
		switch (status) {
			case 'open': return 'text-red-500';
			case 'in_progress':
			case 'investigating': return 'text-yellow-500';
			case 'closed':
			case 'answered': return 'text-green-500';
			default: return 'text-muted-foreground';
		}
	}

	function getStatusIcon(status: string): string {
		switch (status) {
			case 'open': return '?'; // Open circle with question
			case 'in_progress':
			case 'investigating': return '~'; // In progress
			case 'closed':
			case 'answered': return '+'; // Completed/answered
			default: return '?';
		}
	}

	function formatAge(createdAt?: string): string {
		if (!createdAt) return '';
		const created = new Date(createdAt);
		const now = new Date();
		const diffMs = now.getTime() - created.getTime();
		const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
		const diffDays = Math.floor(diffHours / 24);

		if (diffDays > 0) {
			return `${diffDays}d`;
		} else if (diffHours > 0) {
			return `${diffHours}h`;
		} else {
			const diffMinutes = Math.floor(diffMs / (1000 * 60));
			return `${diffMinutes}m`;
		}
	}

	// Calculate total count across all statuses
	$: totalCount = $questions ? ($questions.open.length + $questions.investigating.length + $questions.answered.length) : 0;

	// Get all open questions for preview (most urgent)
	$: openQuestions = $questions?.open || [];
</script>

{#if $questions && totalCount > 0}
	<div class="rounded-lg border border-purple-500/30 bg-purple-500/5" data-testid="questions-section">
		<button
			class="flex w-full items-center justify-between px-3 py-2 text-left hover:bg-accent/50 transition-colors"
			onclick={toggle}
			aria-expanded={expanded}
			data-testid="questions-toggle"
		>
			<div class="flex items-center gap-2 min-w-0 flex-1">
				<span class="text-sm flex-shrink-0">?</span>
				<span class="text-sm font-medium flex-shrink-0">Questions</span>
				<Badge variant="secondary" class="h-5 px-1.5 text-xs flex-shrink-0">
					{totalCount}
				</Badge>
				{#if openQuestions.length > 0}
					<Badge variant="destructive" class="h-5 px-1.5 text-xs flex-shrink-0">
						{openQuestions.length} open
					</Badge>
				{/if}
				{#if !expanded && openQuestions.length > 0}
					<span class="text-xs text-muted-foreground truncate" data-testid="questions-preview">
						-- {getPreview(openQuestions)}
					</span>
				{/if}
			</div>
			<span class="text-muted-foreground transition-transform flex-shrink-0 {expanded ? 'rotate-180' : ''}">
				<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
					<polyline points="6 9 12 15 18 9"></polyline>
				</svg>
			</span>
		</button>

		{#if expanded}
			<div class="border-t p-2" data-testid="questions-content">
				<!-- Open Questions (needs answer) - Red/Urgent -->
				{#if $questions.open.length > 0}
					<div class="mb-3">
						<div class="flex items-center gap-2 mb-1.5 px-2">
							<span class="text-red-500 text-xs font-medium">? Open (needs answer)</span>
							<Badge variant="destructive" class="h-4 px-1 text-xs">{$questions.open.length}</Badge>
						</div>
						<div class="space-y-1">
							{#each $questions.open as question (question.id)}
								<div class="flex items-center gap-2 rounded px-2 py-1.5 text-sm hover:bg-accent/50 border-l-2 border-red-500/50" data-testid="question-{question.id}">
									<span class="flex-shrink-0 text-xs font-medium {getStatusClass(question.status)}">
										{getStatusIcon(question.status)}
									</span>
									<span class="flex-1 truncate" title={question.title}>
										{question.title}
									</span>
									{#if question.blocking && question.blocking.length > 0}
										<Badge variant="outline" class="h-5 px-1.5 text-xs flex-shrink-0 border-red-500/50 text-red-500">
											Blocks {question.blocking.length}
										</Badge>
									{/if}
									<span class="text-xs text-muted-foreground flex-shrink-0">
										{formatAge(question.created_at)}
									</span>
									<span class="text-xs text-muted-foreground flex-shrink-0 font-mono">
										{question.id}
									</span>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Investigating Questions - Yellow/Warning -->
				{#if $questions.investigating.length > 0}
					<div class="mb-3">
						<div class="flex items-center gap-2 mb-1.5 px-2">
							<span class="text-yellow-500 text-xs font-medium">~ Investigating</span>
							<Badge variant="secondary" class="h-4 px-1 text-xs bg-yellow-500/20 text-yellow-600">{$questions.investigating.length}</Badge>
						</div>
						<div class="space-y-1">
							{#each $questions.investigating as question (question.id)}
								<div class="flex items-center gap-2 rounded px-2 py-1.5 text-sm hover:bg-accent/50 border-l-2 border-yellow-500/50" data-testid="question-{question.id}">
									<span class="flex-shrink-0 text-xs font-medium {getStatusClass(question.status)}">
										{getStatusIcon(question.status)}
									</span>
									<span class="flex-1 truncate" title={question.title}>
										{question.title}
									</span>
									{#if question.blocking && question.blocking.length > 0}
										<Badge variant="outline" class="h-5 px-1.5 text-xs flex-shrink-0 border-yellow-500/50 text-yellow-600">
											Blocks {question.blocking.length}
										</Badge>
									{/if}
									<span class="text-xs text-muted-foreground flex-shrink-0">
										{formatAge(question.created_at)}
									</span>
									<span class="text-xs text-muted-foreground flex-shrink-0 font-mono">
										{question.id}
									</span>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Answered Questions (recently closed) - Green/Success -->
				{#if $questions.answered.length > 0}
					<div>
						<div class="flex items-center gap-2 mb-1.5 px-2">
							<span class="text-green-500 text-xs font-medium">+ Answered (last 7 days)</span>
							<Badge variant="secondary" class="h-4 px-1 text-xs bg-green-500/20 text-green-600">{$questions.answered.length}</Badge>
						</div>
						<div class="space-y-1">
							{#each $questions.answered as question (question.id)}
								<div class="flex items-center gap-2 rounded px-2 py-1.5 text-sm hover:bg-accent/50 border-l-2 border-green-500/50 opacity-75" data-testid="question-{question.id}">
									<span class="flex-shrink-0 text-xs font-medium {getStatusClass(question.status)}">
										{getStatusIcon(question.status)}
									</span>
									<span class="flex-1 truncate" title={question.title}>
										{question.title}
									</span>
									{#if question.close_reason}
										<span class="text-xs text-green-600 truncate max-w-48" title={question.close_reason}>
											{question.close_reason.length > 30 ? question.close_reason.substring(0, 30) + '...' : question.close_reason}
										</span>
									{/if}
									<span class="text-xs text-muted-foreground flex-shrink-0 font-mono">
										{question.id}
									</span>
								</div>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Empty state when all categories are empty -->
				{#if $questions.open.length === 0 && $questions.investigating.length === 0 && $questions.answered.length === 0}
					<p class="py-4 text-center text-sm text-muted-foreground">No questions</p>
				{/if}
			</div>
		{/if}
	</div>
{/if}
