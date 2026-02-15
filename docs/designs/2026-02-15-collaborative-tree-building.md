# Design: Collaborative Tree-Building as Primary Work Interface

**Date:** 2026-02-15
**Status:** Proposed
**Owner:** Orchestrator + Dylan (design session)
**Context:** The knowledge tree (`orch tree`) shipped as a static CLI renderer. Dylan's vision is fundamentally different: the tree as a live, shared workspace where Dylan and the AI orchestrator grow branches together in real time. Issues transform into knowledge artifacts. The tree IS the project, not a report about it.

## Problem Statement

The knowledge tree today is a **snapshot renderer** — run a command, see static output, mentally decide what to do, run separate commands to do it. The work of creating issues, completing agents, and making decisions happens outside the tree.

Dylan's vision: the tree is the **medium you work through**. When the orchestrator creates an issue, a new node appears. When an agent completes, the node transforms. When a decision is made, it attaches to its parent investigation. The tree grows in real time as the session unfolds.

**Dylan's words:** "I'd be interacting with the AI orchestrator and as we created issues etc, we could see these 'grow' on the tree and transform into knowledge/decisions/designs in real time."

**And:** "This UI design could entirely transform how I work if we do this right."

## Core Insight

In Dylan's system, work and knowledge aren't separate things:
- An investigation IS work that BECOMES knowledge
- A decision IS knowledge that SPAWNS work
- The current separation (beads for work, `.kb/` for knowledge) is an implementation detail, not a real boundary

The tree should reflect this unity — one living structure where nodes transform from work into knowledge.

## Architecture: Shared Data, Separate Interfaces

```
pkg/tree/ (extraction + relationship graph)
    ↓
orch serve /api/tree (JSON endpoint)
    ↓                         ↓
CLI: orch tree           Dashboard: /knowledge-tree
(AI orchestrator)        (Dylan)
    ↓                         ↓
Text output for          Live visual tree with
spawn context,           animations, pulsing
triage, orientation      agents, click-to-expand
    ↓                         ↓
    └──── SSE pushes updates to dashboard ────┘
         when beads/kb state changes
```

**Key principle:** Same data, same tree, different rendering. The orchestrator acts via CLI, Dylan observes and browses via dashboard. Both see the same structure.

**Decision:** `kb-cf50a7` — Shared data layer, separate CLI and dashboard interfaces.

## Design Decisions

### 1. Node Transformation: Split-and-Grow

When work becomes knowledge (agent completes investigation, decision accepted), the visualization uses **split-and-grow** animation:

1. The `●` issue node fades/shrinks (but remains visible as a small gray node)
2. A new `◉` investigation / `★` decision / `◆` model node grows from it
3. A branch line connects them — "this knowledge came from this work"

**Why split-and-grow over morph-in-place:** Preserves history (you can still see the issue that produced the knowledge) while making the transformation visible. Naturally builds the lineage edges that make the tree useful.

**Node lifecycle:**
```
● Issue created (appears)
● Agent spawned (starts pulsing)
● Agent complete (stops pulsing, solid)
● → ◉/★/◆ (split-and-grow: issue fades, artifact grows with connecting edge)
```

### 2. Quick Decisions in the Tree

`kb quick decide/constrain/tried` entries currently live in `.kb/quick/` JSONL — invisible in the tree. They should be visible.

**Attachment:** Quick decisions attach to their related parent artifact via a new `--related` flag:

```bash
kb quick decide "serialize verification" \
  --reason "..." \
  --related decisions/2026-02-14-verifiability-first-hard-constraint.md
```

**Two views of the same node:**
- **Knowledge view:** Quick decision appears as child of its parent artifact
- **Session timeline:** Quick decision appears chronologically in session history

**Decision:** `kb-bcd869` — Add `--related` flag to `kb quick` commands for tree lineage.

### 3. Active Agent Visualization

Agents currently running show as **pulsing** nodes in the dashboard tree. The pulse communicates "something is happening here" without requiring text status. When the agent completes, the pulse stops and the node solidifies, then triggers split-and-grow if it produced a knowledge artifact.

### 4. Session Timeline Mode

The dashboard supports three views (user rotates between them):

| View | Question it answers | Primary nodes |
|------|---------------------|---------------|
| **Knowledge** (`orch tree`) | "What do we understand?" | Investigations, decisions, models |
| **Work** (`orch tree --work`) | "What are we doing and why?" | Issues grouped by state |
| **Timeline** (new) | "What happened this session?" | Chronological session actions |

**Timeline example:**
```
Session: Feb 15 — "Verifiability design review"
│
├─ 14:30  ★ Accepted: verifiability-first-hard-constraint
│         ├─ kb-676bdb  Three-tier verification granularity
│         ├─ kb-96dec5  Serialize all verification
│         ├─ kb-f26074  Trend-based override threshold
│         └─ kb-d2334a  Human observation meta-verification
│
├─ 15:10  ● Created: orch-go-3wu  Tree CLI MVP
│         → Completed → Split: ◉ pkg/tree/
│
├─ 16:20  ● Created: orch-go-2m1  Health smells          (pulsing)
│         ● Created: orch-go-786  Relationship parsing   (pulsing)
│
├─ 16:40  Released: e09, 8he, e9z, 2qj, 4tz, cjd
│
└─ 17:15  ★ kb-cf50a7  Shared data architecture
```

### 5. Session Identity: Hybrid Model

Sessions are identified by **OpenCode session ID** (zero friction, already exists) with optional enrichment via `orch session label "description"`.

- Default: correlate all actions by OpenCode `ses_xxxxx` ID
- Optional: `orch session label "verifiability design review"` adds human-readable name
- Dashboard timeline shows label if exists, falls back to session ID + time range
- Tool-agnostic: if we switch from OpenCode, just change where session ID comes from

**Decision:** `kb-cf9d8d` — Hybrid session identity.

### 6. Latency

SSE (Server-Sent Events) from `orch serve` to dashboard. When beads state changes (issue created, agent status update, completion), push tree update. Expected 1-2 second latency. Upgrade to websocket later if needed.

`orch serve` already has SSE infrastructure for the existing dashboard.

### 7. Dashboard Interactivity

**Now:** Read-only. Dylan browses, expands/collapses, filters, searches. All mutations happen through orchestrator CLI.

**Later:** Interactive. Click a health smell → "spawn into this." Click a node → action menu (close, create child issue, add label). This is a separate design session when the read-only version is proven.

## Data Flow: How the Tree Updates in Real Time

```
1. Orchestrator runs: bd create "new issue" -l triage:ready
2. Beads writes to .beads/ database
3. orch serve detects state change (filesystem watch or polling)
4. orch serve re-extracts tree via pkg/tree/
5. orch serve pushes SSE event: {type: "tree-update", diff: {...}}
6. Dashboard receives SSE, applies diff to rendered tree
7. New ● node appears with grow-in animation
8. Dylan sees it in real time
```

**For agent status updates:**
```
1. Agent reports phase change via bd comment
2. orch serve detects comment (polling agent status)
3. SSE push: {type: "agent-status", id: "orch-go-xxx", phase: "Complete"}
4. Dashboard: pulsing stops, node solidifies
5. If agent produced artifact: trigger split-and-grow animation
```

## Prerequisite: Uncategorized Investigation Audit

The tree is only useful if its structure is meaningful. Currently ~800 investigations sit in an "uncategorized" bucket, making the tree noisy. 

**Issue:** `orch-go-5ts` — Audit and recommend archive vs cluster labels.
**Strategy:** Archive spiral-era fossils, auto-cluster going forward with `area:` labels at creation time.

## Implementation Phases

### Phase 1: Tree API Endpoint
- [ ] Create `GET /api/tree` endpoint in `orch serve`
- [ ] Return full tree as JSON (same structure as `orch tree --format json`)
- [ ] Support query params: `?view=knowledge|work`, `?cluster=X`, `?depth=N`
- [ ] Add SSE channel for tree updates
- [ ] Detect beads/kb state changes (filesystem watch or polling interval)
- [ ] Push tree diffs on state change

### Phase 2: Dashboard Knowledge + Work Views
- [ ] Create `/knowledge-tree` route in dashboard (Svelte)
- [ ] Implement tree component with expand/collapse
- [ ] Render node types with correct icons (◉ ★ ◆ ◈ ● ◇)
- [ ] Color coding by type
- [ ] Pulsing animation for active agents
- [ ] Subscribe to SSE for live updates
- [ ] Grow-in animation for new nodes
- [ ] Filter by type, area, status
- [ ] Search
- [ ] View toggle: knowledge / work
- [ ] Save tree state to localStorage

### Phase 3: Node Transformation Animations
- [ ] Split-and-grow animation when issue produces artifact
- [ ] Detect artifact creation (new .kb/ file linked to beads issue)
- [ ] Fade issue node, grow artifact node with connecting edge
- [ ] Smooth transitions for state changes (triage → in-progress → complete)

### Phase 4: Session Timeline View
- [ ] Add `orch session label` command
- [ ] Correlate actions to OpenCode session ID
- [ ] Create timeline view in dashboard (chronological session actions)
- [ ] Group actions by session
- [ ] Show quick decisions, issue creation, completions, releases
- [ ] View toggle: knowledge / work / timeline

### Phase 5: `--related` Flag for kb quick
- [ ] Add `--related <path>` flag to `kb quick decide/constrain/tried`
- [ ] Store relationship in JSONL entry
- [ ] Include related quick entries in tree extraction (as children of parent artifact)
- [ ] Surface in both knowledge view and timeline view

## Success Criteria

- [ ] Dylan can watch the tree grow in real time during an orchestrator session
- [ ] Node transformation (split-and-grow) is visible when agents produce artifacts
- [ ] Active agents pulse, completed agents solidify
- [ ] Session timeline shows what happened in the current session
- [ ] Quick decisions appear attached to their parent artifacts
- [ ] Dashboard loads and renders full tree interactively in <1 second
- [ ] Three views (knowledge, work, timeline) are accessible and useful
- [ ] AI orchestrator uses CLI tree for orientation, triage, and spawn context
- [ ] Dylan says "this is how I work now"

## Open Questions

1. **Tree diff format for SSE.** Full tree re-render on every change, or incremental diff? Full is simpler, diff is more efficient. Start with full, optimize later.

2. **Animation performance with large trees.** 800+ nodes with animations could be heavy. May need virtualization (only render visible nodes). Test with real data.

3. **Cross-project trees.** Dylan works across orch-go, orch-knowledge, opencode, snap. Should the dashboard show trees from multiple projects? Or one project at a time? Start with single project, add multi-project later.

4. **Offline/async sessions.** When daemon runs overnight and 6 agents complete, the timeline shows a gap. Should it show daemon actions as a "daemon session"? Or just show completions without session context?

## References

- Prior design: `docs/designs/2026-02-14-knowledge-lineage-tree-visualization.md` (data model, extraction algorithm, CLI design)
- Architecture decision: `kb-cf50a7` (shared data, separate interfaces)
- Session identity decision: `kb-cf9d8d` (hybrid OpenCode + optional label)
- `--related` flag decision: `kb-bcd869`
- Uncategorized audit: `orch-go-5ts`
- Existing tree code: `pkg/tree/`
- Dashboard SSE infrastructure: `orch serve` existing implementation
