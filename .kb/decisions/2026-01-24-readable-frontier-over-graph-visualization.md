# Decision: Readable Frontier Over Graph Visualization

**Date:** 2026-01-24
**Status:** Accepted
**Context:** Decidability graph visualization shipped as Cytoscape.js force-directed graph, but didn't match the vision

## Decision

**Readable text-first frontier view is the primary decidability interface. Graph visualization is optional deep-dive.**

## Why

After seeing the Cytoscape graph in action, Dylan noted it felt "more graphical when what I wanted was readable first and graphical second."

The decidability model's own example was text:
```
Frontier Report:
  Ready frontier: [Work-D, Work-E] (daemon-traversable)
  Question-blocked: [Work-F, Work-G] waiting on Question-Q1
```

The graph requires:
- Exploration (zoom, pan, click)
- Visual decoding (colors, shapes)
- Interaction to understand

The frontier report provides:
- Scannable in 5 seconds
- Grouped by "who needs to act"
- Answers the question directly
- No interaction required

## What This Means

1. **`orch frontier` CLI command** becomes primary interface (orch-go-2fut9)
2. **Dashboard section** renders same data in readable form (orch-go-7ee90)
3. **Cytoscape graph** remains available as deep-dive exploration tool
4. **Decidability visualization epic** pivots from "build graph" to "make state readable"

## The Insight

The graph is a representation OF something. The something itself (decidability state: what's ready, what's blocked, who needs to act) is better expressed as structured text for daily use. Graph is for occasional exploration of relationships.

## References

- Conversation 2026-01-24 exploring why graph didn't feel right
- `.kb/models/decidability-graph.md` - the frontier report example was text
- orch-go-5ubkd - original epic (now pivoting)
