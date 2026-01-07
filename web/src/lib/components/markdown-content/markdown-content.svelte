<script lang="ts">
	import { marked } from 'marked';

	interface Props {
		content: string;
		class?: string;
	}

	let { content, class: className = '' }: Props = $props();

	// Configure marked for safe rendering
	marked.setOptions({
		gfm: true, // GitHub Flavored Markdown
		breaks: true, // Convert \n to <br>
	});

	// Render markdown to HTML
	const html = $derived(marked.parse(content) as string);
</script>

<div class="markdown-content prose prose-sm dark:prose-invert max-w-none {className}">
	{@html html}
</div>

<style>
	/* Custom markdown styles that work with dark mode */
	.markdown-content :global(h1) {
		@apply text-lg font-semibold mt-4 mb-2 text-foreground;
	}
	.markdown-content :global(h2) {
		@apply text-base font-semibold mt-3 mb-2 text-foreground;
	}
	.markdown-content :global(h3) {
		@apply text-sm font-semibold mt-2 mb-1 text-foreground;
	}
	.markdown-content :global(p) {
		@apply text-sm leading-relaxed mb-2 text-foreground/90;
	}
	.markdown-content :global(ul),
	.markdown-content :global(ol) {
		@apply text-sm pl-4 mb-2;
	}
	.markdown-content :global(li) {
		@apply mb-1;
	}
	.markdown-content :global(code) {
		@apply text-xs bg-muted px-1.5 py-0.5 rounded font-mono;
	}
	.markdown-content :global(pre) {
		@apply text-xs bg-muted p-3 rounded-lg overflow-x-auto mb-2;
	}
	.markdown-content :global(pre code) {
		@apply bg-transparent p-0;
	}
	.markdown-content :global(blockquote) {
		@apply border-l-2 border-muted-foreground/30 pl-3 italic text-muted-foreground;
	}
	.markdown-content :global(a) {
		@apply text-primary underline underline-offset-2;
	}
	.markdown-content :global(strong) {
		@apply font-semibold text-foreground;
	}
	.markdown-content :global(hr) {
		@apply my-4 border-muted;
	}
	.markdown-content :global(table) {
		@apply w-full text-sm border-collapse mb-2;
	}
	.markdown-content :global(th),
	.markdown-content :global(td) {
		@apply border border-muted px-2 py-1 text-left;
	}
	.markdown-content :global(th) {
		@apply bg-muted/50 font-medium;
	}
</style>
