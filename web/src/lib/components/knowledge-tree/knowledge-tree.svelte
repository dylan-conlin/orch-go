<script lang="ts">
	import { fade, scale } from 'svelte/transition';
	import type { KnowledgeNode, NodeType, NodeStatus, NodeAnimation } from '$lib/stores/knowledge-tree';

	export let node: KnowledgeNode;
	export let depth: number = 0;
	export let onToggle: (nodeId: string) => void;
	export let expandedNodes: Set<string>;
	export let animationStates: Map<string, NodeAnimation> = new Map();

	// Get animation state for this node
	$: animation = animationStates.get(node.ID);
	$: animationState = animation?.state || 'static';
	$: isPulsing = animationState === 'pulsing';
	$: isFading = animationState === 'fading';
	$: isGrowing = animationState === 'growing';

	// Node icon by type (from design doc)
	function getNodeIcon(type: NodeType): string {
		switch (type) {
			case 'investigation': return '◉';
			case 'decision': return '★';
			case 'model': return '◆';
			case 'guide': return '◈';
			case 'issue': return '●';
			case 'cluster': return '📁';
			case 'probe': return '◇';
			case 'postmortem': return '📋';
			case 'handoff': return '🤝';
			default: return '◦';
		}
	}

	// Color by type (from design doc)
	function getNodeColor(type: NodeType): string {
		switch (type) {
			case 'investigation': return 'text-green-400';
			case 'decision': return 'text-yellow-400';
			case 'model': return 'text-purple-400';
			case 'guide': return 'text-cyan-400';
			case 'issue': return 'text-orange-400';
			case 'cluster': return 'text-gray-400';
			case 'probe': return 'text-pink-400';
			default: return 'text-gray-500';
		}
	}

	// Status badge color
	function getStatusColor(status: NodeStatus): string {
		switch (status) {
			case 'complete': return 'bg-green-500/20 text-green-400';
			case 'in_progress': return 'bg-blue-500/20 text-blue-400';
			case 'triage': return 'bg-yellow-500/20 text-yellow-400';
			case 'closed': return 'bg-gray-500/20 text-gray-400';
			case 'open': return 'bg-orange-500/20 text-orange-400';
			default: return 'bg-gray-500/20 text-gray-400';
		}
	}

	function handleClick() {
		if (node.Children && node.Children.length > 0) {
			onToggle(node.ID);
		}
	}

	$: hasChildren = node.Children && node.Children.length > 0;
	$: isExpanded = expandedNodes.has(node.ID);
	$: indentClass = `pl-${Math.min(depth * 4, 16)}`;
</script>

<div 
	class="tree-node"
	class:fading-node={isFading}
	class:growing-node={isGrowing}
>
	<!-- Node row -->
	<button
		type="button"
		class="node-button w-full text-left px-2 py-1 hover:bg-zinc-800/50 flex items-center gap-2 text-sm {indentClass}"
		class:pulsing={isPulsing}
		class:fading={isFading}
		onclick={handleClick}
		data-node-id={node.ID}
		in:scale={{ duration: isGrowing ? 600 : 300, start: isGrowing ? 0.5 : 0.8 }}
	>
		<!-- Expand/collapse indicator -->
		{#if hasChildren}
			<span class="text-xs text-gray-500 w-3 transition-transform duration-200">
				{isExpanded ? '▼' : '▶'}
			</span>
		{:else}
			<span class="w-3"></span>
		{/if}

		<!-- Node icon -->
		<span 
			class="text-base {getNodeColor(node.Type)} transition-all duration-300"
			class:icon-pulse={isFading}
		>
			{getNodeIcon(node.Type)}
		</span>

		<!-- Title -->
		<span class="flex-1 truncate text-gray-200 transition-opacity duration-300">
			{node.Title}
		</span>

		<!-- Status badge -->
		{#if node.Status && node.Status !== 'complete'}
			<span 
				class="text-xs px-2 py-0.5 rounded {getStatusColor(node.Status)} transition-all duration-300"
				in:fade={{ duration: 200 }}
			>
				{node.Status}
			</span>
		{/if}

		<!-- Date (hide Go zero-time "0001-01-01T00:00:00Z" and missing dates) -->
		{#if node.Date && new Date(node.Date).getFullYear() > 1}
			<span class="text-xs text-gray-500">
				{new Date(node.Date).toLocaleDateString()}
			</span>
		{/if}
	</button>

	<!-- Children (recursive) -->
	{#if hasChildren && isExpanded}
		{#each node.Children as child (child.ID)}
			<svelte:self node={child} depth={depth + 1} {onToggle} {expandedNodes} {animationStates} />
		{/each}
	{/if}
</div>

<style>
	/* Tailwind pl-* classes for indentation */
	.pl-0 { padding-left: 0rem; }
	.pl-4 { padding-left: 1rem; }
	.pl-8 { padding-left: 2rem; }
	.pl-12 { padding-left: 3rem; }
	.pl-16 { padding-left: 4rem; }

	/* Pulsing animation for active agents (status: in_progress) */
	@keyframes pulse-node {
		0%, 100% { 
			opacity: 1;
			transform: scale(1);
		}
		50% { 
			opacity: 0.7;
			transform: scale(1.02);
		}
	}

	.pulsing {
		animation: pulse-node 2s ease-in-out infinite;
	}

	/* Split-and-grow transformation */
	/* Issue node fades and shrinks when transforming */
	.fading {
		animation: fade-shrink 0.8s ease forwards;
		position: relative;
	}

	@keyframes fade-shrink {
		0% {
			opacity: 1;
			transform: scale(1);
		}
		100% {
			opacity: 0.3;
			transform: scale(0.8);
			color: #9ca3af; /* gray-400 */
		}
	}

	/* Growing node appears with scale transition (handled by Svelte's in:scale) */
	.growing-node {
		position: relative;
	}

	/* Connecting line from parent to child during split-and-grow */
	.growing-node::before {
		content: '';
		position: absolute;
		left: 0.75rem;
		top: -0.5rem;
		width: 2px;
		height: 0.5rem;
		background: linear-gradient(to bottom, #6b7280, transparent);
		animation: line-grow 0.6s ease;
	}

	@keyframes line-grow {
		0% {
			height: 0;
			opacity: 0;
		}
		100% {
			height: 0.5rem;
			opacity: 1;
		}
	}

	/* Icon transformation effect */
	.icon-transform {
		animation: icon-pulse 0.5s ease;
	}

	@keyframes icon-pulse {
		0%, 100% {
			transform: scale(1);
		}
		50% {
			transform: scale(1.3);
		}
	}

	/* Smooth status transitions (triage → in_progress → complete) */
	.node-button {
		transition: 
			background-color 0.3s ease,
			border-color 0.3s ease,
			opacity 0.3s ease,
			transform 0.3s ease;
	}

	/* Artifact nodes grow in smoothly (handled by Svelte's scale transition) */
</style>
