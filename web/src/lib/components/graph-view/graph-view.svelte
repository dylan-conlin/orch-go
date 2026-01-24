<script lang="ts">
	import { onMount, onDestroy, tick } from 'svelte';
	import { mode, getEffective } from '$lib/stores/theme';
	import type { Core, ElementDefinition, StylesheetJsonBlock, LayoutOptions } from 'cytoscape';

	// Dynamic import to avoid SSR issues - cytoscape requires DOM
	let cytoscapeLib: typeof import('cytoscape').default | null = null;

	interface GraphNode {
		id: string;
		title: string;
		type: 'task' | 'question' | 'epic' | 'feature' | 'bug' | 'investigation' | 'decision';
		status: string;
		priority: number;
		source?: 'beads' | 'kb';
		date?: string;
	}

	interface GraphEdge {
		from: string;
		to: string;
		type: string;
		authority?: 'daemon' | 'orchestrator' | 'human';
	}

	interface GraphData {
		nodes: GraphNode[];
		edges: GraphEdge[];
		node_count: number;
		edge_count: number;
	}

	// Scope controls what issues are shown
	type GraphScope = 'focus' | 'open' | 'all';
	let scope: GraphScope = 'focus';

	let container: HTMLDivElement;
	let cy: Core | null = null;
	let loading = true;
	let error: string | null = null;
	let hoveredNode: GraphNode | null = null;
	let debugState = 'initializing';
	let nodeCount = 0;
	let edgeCount = 0;

	// Colors by node type
	const typeColors: Record<string, string> = {
		// Beads issue types
		task: '#3b82f6',          // blue-500
		question: '#f59e0b',      // amber-500
		epic: '#a855f7',          // purple-500
		feature: '#22c55e',       // green-500
		bug: '#ef4444',           // red-500
		// KB artifact types
		investigation: '#06b6d4', // cyan-500
		decision: '#f97316'       // orange-500
	};

	// Get opacity based on status
	function getStatusOpacity(status: string): number {
		switch (status) {
			case 'open':
			case 'in_progress':
				return 1;
			case 'blocked':
				return 0.5;
			case 'closed':
				return 0.3;
			default:
				return 1;
		}
	}

	// Get edge style based on authority
	function getEdgeStyle(authority?: string): 'solid' | 'dashed' {
		switch (authority) {
			case 'orchestrator':
				return 'dashed';
			case 'human':
				return 'solid'; // We'll use red color instead
			default:
				return 'solid';
		}
	}

	function getEdgeColor(authority?: string): string {
		switch (authority) {
			case 'human':
				return '#ef4444'; // red-500
			default:
				return '#6b7280'; // gray-500
		}
	}

	const API_BASE = 'https://localhost:3348';

	async function fetchGraphData(): Promise<GraphData> {
		const response = await fetch(`${API_BASE}/api/beads/graph?scope=${scope}`);
		if (!response.ok) {
			throw new Error(`Failed to fetch graph: ${response.statusText}`);
		}
		return response.json();
	}

	function setScope(newScope: GraphScope) {
		scope = newScope;
		initGraph();
	}

	function buildCytoscapeElements(data: GraphData): ElementDefinition[] {
		const elements: ElementDefinition[] = [];

		// Add nodes
		for (const node of data.nodes) {
			// Shorter label for kb artifacts (just the topic part)
			let label = node.id;
			if (node.source === 'kb' && node.id.includes('-inv-')) {
				// Extract topic from "2026-01-22-inv-topic-here" -> "topic-here"
				const parts = node.id.split('-inv-');
				if (parts.length > 1) {
					label = parts[1].substring(0, 20) + (parts[1].length > 20 ? '...' : '');
				}
			}

			elements.push({
				data: {
					id: node.id,
					label: label,
					title: node.title,
					type: node.type,
					status: node.status,
					priority: node.priority,
					source: node.source || 'beads',
					color: typeColors[node.type] || '#6b7280',
					opacity: getStatusOpacity(node.status)
				}
			});
		}

		// Add edges
		for (const edge of data.edges) {
			elements.push({
				data: {
					id: `${edge.from}-${edge.to}`,
					source: edge.from,
					target: edge.to,
					type: edge.type,
					authority: edge.authority,
					lineStyle: getEdgeStyle(edge.authority),
					color: getEdgeColor(edge.authority)
				}
			});
		}

		return elements;
	}

	function getStylesheet(isDark: boolean): StylesheetJsonBlock[] {
		const textColor = isDark ? '#e5e5e5' : '#171717';
		const bgColor = isDark ? '#171717' : '#ffffff';

		return [
			{
				selector: 'node',
				style: {
					'background-color': 'data(color)',
					'background-opacity': 'data(opacity)',
					'label': 'data(label)',
					'text-valign': 'bottom',
					'text-halign': 'center',
					'text-margin-y': 5,
					'font-size': '10px',
					'color': textColor,
					'width': 30,
					'height': 30,
					'border-width': 2,
					'border-color': isDark ? '#404040' : '#d4d4d4'
				}
			},
			{
				selector: 'node:selected',
				style: {
					'border-width': 3,
					'border-color': '#3b82f6'
				}
			},
			{
				selector: 'edge',
				style: {
					'width': 2,
					'line-color': 'data(color)',
					'target-arrow-color': 'data(color)',
					'target-arrow-shape': 'triangle',
					'curve-style': 'bezier',
					'line-style': 'data(lineStyle)' as any
				}
			},
			{
				selector: 'edge[authority = "orchestrator"]',
				style: {
					'line-style': 'dashed'
				}
			},
			{
				selector: 'edge[authority = "human"]',
				style: {
					'line-color': '#ef4444',
					'target-arrow-color': '#ef4444'
				}
			},
			// KB artifacts get square shape to distinguish from beads issues
			{
				selector: 'node[source = "kb"]',
				style: {
					'shape': 'rectangle',
					'width': 35,
					'height': 25
				}
			},
			// Reference edges get dotted style
			{
				selector: 'edge[type = "references"]',
				style: {
					'line-style': 'dotted',
					'line-color': '#06b6d4',
					'target-arrow-color': '#06b6d4'
				}
			}
		];
	}

	function getLayout(): LayoutOptions {
		return {
			name: 'cose',
			animate: true,
			animationDuration: 500,
			nodeRepulsion: () => 8000,
			idealEdgeLength: () => 100,
			edgeElasticity: () => 100,
			nestingFactor: 1.2,
			gravity: 0.25,
			numIter: 1000,
			initialTemp: 200,
			coolingFactor: 0.95,
			minTemp: 1.0
		} as LayoutOptions;
	}

	async function initGraph() {
		if (!container || !cytoscapeLib) {
			console.warn('GraphView: container or cytoscape not ready', { container: !!container, cytoscapeLib: !!cytoscapeLib });
			debugState = `guard failed: container=${!!container}, lib=${!!cytoscapeLib}`;
			return;
		}

		loading = true;
		error = null;
		debugState = 'fetching data';

		try {
			const data = await fetchGraphData();
			nodeCount = data.node_count;
			edgeCount = data.edge_count;
			debugState = `data: ${data.nodes.length} nodes, ${data.edges.length} edges`;
			const elements = buildCytoscapeElements(data);
			const isDark = getEffective($mode) === 'dark';

			cy = cytoscapeLib({
				container,
				elements,
				style: getStylesheet(isDark),
				layout: getLayout(),
				minZoom: 0.1,
				maxZoom: 3,
				wheelSensitivity: 0.3
			});

			// Add hover interactions
			cy.on('mouseover', 'node', (event) => {
				const node = event.target;
				hoveredNode = {
					id: node.data('id'),
					title: node.data('title'),
					type: node.data('type'),
					status: node.data('status'),
					priority: node.data('priority'),
					source: node.data('source')
				};
			});

			cy.on('mouseout', 'node', () => {
				hoveredNode = null;
			});

			// Click to navigate (could link to beads issue)
			cy.on('tap', 'node', (event) => {
				const id = event.target.data('id');
				console.log('Clicked node:', id);
			});

			loading = false;
		} catch (e) {
			console.error('GraphView initGraph error:', e);
			error = e instanceof Error ? e.message : 'Unknown error';
			loading = false;
		}
	}

	function updateTheme() {
		if (!cy) return;
		const isDark = getEffective($mode) === 'dark';
		cy.style(getStylesheet(isDark));
	}

	function fitGraph() {
		cy?.fit(undefined, 50);
	}

	function resetLayout() {
		if (!cy) return;
		cy.layout(getLayout()).run();
	}

	onMount(async () => {
		debugState = 'loading cytoscape';
		try {
			// Dynamic import to avoid SSR issues - cytoscape requires DOM
			const module = await import('cytoscape');
			cytoscapeLib = module.default;
		} catch (importError) {
			console.error('Failed to import cytoscape:', importError);
			debugState = `cytoscape import failed: ${importError}`;
			error = `Failed to load cytoscape: ${importError}`;
			loading = false;
			return;
		}
		debugState = 'cytoscape loaded, waiting for tick';
		// Wait for DOM to be fully updated with refs
		await tick();
		debugState = 'tick complete, checking container';
		// Add a small delay to ensure container has dimensions
		await new Promise(resolve => setTimeout(resolve, 50));
		debugState = `container: ${!!container}, lib: ${!!cytoscapeLib}`;
		console.log('GraphView onMount complete, container:', !!container, 'cytoscapeLib:', !!cytoscapeLib);
		if (container && cytoscapeLib) {
			debugState = 'calling initGraph';
			initGraph();
		} else {
			console.error('GraphView: container or cytoscapeLib not available after tick');
			error = 'Failed to initialize: container not available';
			loading = false;
		}
	});

	onDestroy(() => {
		cy?.destroy();
		cy = null;
	});

	// React to theme changes
	$: if (cy && $mode) {
		updateTheme();
	}
</script>

<div class="relative h-full w-full min-h-[600px] rounded-lg border bg-card">
	<!-- Controls -->
	<div class="absolute top-2 right-2 z-10 flex gap-2">
		<!-- Scope toggle -->
		<div class="flex rounded border bg-background">
			<button
				onclick={() => setScope('focus')}
				class="px-2 py-1 text-xs transition-colors {scope === 'focus' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'}"
				title="Focus: in_progress + blockers + P0/P1"
			>
				Focus
			</button>
			<button
				onclick={() => setScope('open')}
				class="px-2 py-1 text-xs border-l transition-colors {scope === 'open' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'}"
				title="All open issues"
			>
				Open
			</button>
			<button
				onclick={() => setScope('all')}
				class="px-2 py-1 text-xs border-l transition-colors {scope === 'all' ? 'bg-primary text-primary-foreground' : 'hover:bg-accent'}"
				title="All issues including closed"
			>
				All
			</button>
		</div>
		<button
			onclick={fitGraph}
			class="rounded border bg-background px-2 py-1 text-xs hover:bg-accent"
			title="Fit to view"
		>
			Fit
		</button>
		<button
			onclick={resetLayout}
			class="rounded border bg-background px-2 py-1 text-xs hover:bg-accent"
			title="Reset layout"
		>
			Reset
		</button>
		<button
			onclick={initGraph}
			class="rounded border bg-background px-2 py-1 text-xs hover:bg-accent"
			title="Refresh data"
		>
			Refresh
		</button>
	</div>

	<!-- Legend -->
	<div class="absolute top-2 left-2 z-10 rounded border bg-background/90 p-2 text-xs">
		<div class="font-medium mb-1">Beads Issues</div>
		<div class="flex flex-wrap gap-x-3 gap-y-1">
			<span class="flex items-center gap-1">
				<span class="inline-block h-3 w-3 rounded-full" style="background-color: {typeColors.task}"></span>
				Task
			</span>
			<span class="flex items-center gap-1">
				<span class="inline-block h-3 w-3 rounded-full" style="background-color: {typeColors.feature}"></span>
				Feature
			</span>
			<span class="flex items-center gap-1">
				<span class="inline-block h-3 w-3 rounded-full" style="background-color: {typeColors.epic}"></span>
				Epic
			</span>
			<span class="flex items-center gap-1">
				<span class="inline-block h-3 w-3 rounded-full" style="background-color: {typeColors.question}"></span>
				Question
			</span>
			<span class="flex items-center gap-1">
				<span class="inline-block h-3 w-3 rounded-full" style="background-color: {typeColors.bug}"></span>
				Bug
			</span>
		</div>
		<div class="font-medium mt-2 mb-1">KB Artifacts</div>
		<div class="flex flex-wrap gap-x-3 gap-y-1">
			<span class="flex items-center gap-1">
				<span class="inline-block h-3 w-3 rounded-sm" style="background-color: {typeColors.investigation}"></span>
				Investigation
			</span>
			<span class="flex items-center gap-1">
				<span class="inline-block h-3 w-3 rounded-sm" style="background-color: {typeColors.decision}"></span>
				Decision
			</span>
		</div>
		<div class="font-medium mt-2 mb-1">Status</div>
		<div class="flex flex-wrap gap-x-3 gap-y-1 text-muted-foreground">
			<span>Solid = Open/Active</span>
			<span>Dimmed = Blocked</span>
			<span>Faded = Closed</span>
		</div>
		{#if nodeCount > 0}
			<div class="mt-2 pt-2 border-t text-muted-foreground">
				{nodeCount} nodes, {edgeCount} edges
			</div>
		{/if}
	</div>

	<!-- Hover tooltip -->
	{#if hoveredNode}
		<div class="absolute bottom-2 left-2 z-10 rounded border bg-background/95 p-2 text-xs max-w-md">
			<div class="font-medium">{hoveredNode.id}</div>
			{#if hoveredNode.title}
				<div class="text-muted-foreground mt-1 line-clamp-2">{hoveredNode.title}</div>
			{/if}
			<div class="flex gap-2 mt-1 text-muted-foreground">
				{#if hoveredNode.source === 'kb'}
					<span class="text-cyan-500">{hoveredNode.type}</span>
				{:else}
					<span class="capitalize">{hoveredNode.type}</span>
					<span>•</span>
					<span class="capitalize">{hoveredNode.status}</span>
					{#if hoveredNode.priority !== undefined && hoveredNode.priority <= 1}
						<span>•</span>
						<span class="text-red-500">P{hoveredNode.priority}</span>
					{/if}
				{/if}
			</div>
		</div>
	{/if}

	<!-- Loading/Error states -->
	{#if loading}
		<div class="absolute inset-0 flex items-center justify-center bg-background/80">
			<div class="text-center">
				<div class="text-muted-foreground">Loading graph...</div>
				<div class="text-xs text-muted-foreground mt-2">Debug: {debugState}</div>
			</div>
		</div>
	{:else if error}
		<div class="absolute inset-0 flex items-center justify-center bg-background/80">
			<div class="text-center">
				<div class="text-red-500">{error}</div>
				<button
					onclick={initGraph}
					class="mt-2 rounded border px-3 py-1 text-sm hover:bg-accent"
				>
					Retry
				</button>
			</div>
		</div>
	{/if}

	<!-- Cytoscape container -->
	<div bind:this={container} class="h-full w-full"></div>
</div>
