<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Epics conflate static grouping with dependency structure; labels should handle grouping while edges handle dependencies. Recommend 3-prefix taxonomy (`area:`, `effort:`, `status:`) with epic demotion (become `area:` labels with optional milestone) and multi-dimensional UI grouping.

**Evidence:** Analyzed 4 open epics, 50+ issues, existing label patterns (`subtype:*`, `triage:*`, `parked`). Prior decision on question subtypes via labels (2026-01-28) proves convention works. Work Graph already has filter infrastructure.

**Knowledge:** The core insight is evolving by distinction: epics conflate "these belong together" (grouping/labels) with "this blocks that" (dependency/edges). Labels enable dynamic, multi-dimensional views that adapt as work evolves. AI enforcement at spawn time ensures discipline without manual overhead.

**Next:** Implement in 3 phases: (1) Add label taxonomy + bd create enforcement, (2) Update Work Graph UI with grouping sections, (3) Migrate existing epics to area labels.

**Authority:** architectural - Cross-component decision affecting beads, Work Graph UI, spawn system, and orchestrator workflows. Requires orchestrator synthesis to balance implementation cost vs long-term maintainability.

---

# Design: Label-Based Issue Grouping

**Question:** How should the orchestration system group issues to replace rigid epic hierarchies with dynamic, AI-friendly labels?

**Started:** 2026-02-05
**Updated:** 2026-02-05
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** Create implementation issues from recommendations
**Status:** Complete

<!-- Lineage (fill only when applicable) -->

**Patches-Decision:** `.kb/decisions/2026-01-28-question-subtype-encoding-labels.md` (extends label convention pattern)
**Extracted-From:** N/A

## Prior Work

| Investigation                                                                | Relationship | Verified                    | Conflicts |
| ---------------------------------------------------------------------------- | ------------ | --------------------------- | --------- |
| `.kb/decisions/2026-01-28-question-subtype-encoding-labels.md`               | extends      | ✅ Labels work for subtypes | None      |
| `.kb/investigations/archived/2026-02-02-inv-audit-work-graph-design-docs.md` | informs      | ✅ Verified UI structure    | None      |
| `.kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md` | informs      | ✅ Attention signals exist  | None      |
| `.kb/models/decidability-graph.md`                                           | informs      | ✅ Authority labels exist   | None      |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## The Core Problem

Epics conflate two distinct concepts:

| Concept        | What It Means                  | Best Representation                 |
| -------------- | ------------------------------ | ----------------------------------- |
| **Grouping**   | "These issues belong together" | Labels (dynamic, multi-dimensional) |
| **Dependency** | "This issue blocks that one"   | Edges (explicit, ordered)           |

**Current state:**

- 4 open epics use parent-child edges for both grouping AND blocking
- Epic children are blocked until epic closes
- Epics are rigid - can't adapt as work evolves
- AI agents struggle with epic lifecycle (when to create, decompose, close)

**The distinction (Evolve by Distinction principle):**

- Labels handle grouping (queryable, multi-dimensional, dynamic)
- Edges handle dependencies (explicit blocking relationships)
- Issues can have multiple labels (cross-cutting concerns)
- Issues can't have multiple parents (single hierarchy)

---

## Findings

### Finding 1: Current label usage is sparse but effective

**Evidence:** Existing labels in the system:

- `triage:ready`, `triage:review` - Workflow state (5+ issues)
- `subtype:factual`, `subtype:judgment`, `subtype:framing` - Question subtypes
- `authority:implementation`, `authority:architectural`, `authority:strategic` - Decision authority
- `parked` - Deferred work (3 issues)
- `skill` - Skill-related work (1 issue)

**Source:**

- `bd list --status open` output showing labels in brackets
- `.kb/decisions/2026-01-28-question-subtype-encoding-labels.md` for subtype pattern

**Significance:** Labels are already used for categorization with convention-based prefixes. The pattern works and is AI-friendly. The gap is lack of systematic taxonomy for work domain/area.

---

### Finding 2: Epics currently serve as work areas, not just milestones

**Evidence:** 4 open epics:

- `orch-go-21308` (P1) - "Orchestration System Simplification (Quick Wins)" - 6 children
- `orch-go-21260` (P2) - "Code Extraction: Critical File Bloat" - 4 children
- `orch-go-21262` (P2) - "Decision Documentation Backlog" - unknown children
- `orch-go-kz7zr` (P3, parked) - "Governance Infrastructure for Human-AI Systems"

These epics represent work areas (dashboard, extraction, governance), not time-boxed milestones.

**Source:**

- `bd list --type epic` output
- `bd show orch-go-21260` showing blocks relationship

**Significance:** Epics are being used as area groupings. This confirms the value of area-based grouping, but suggests labels (`area:dashboard`, `area:bloat`) would be more flexible than epic type.

---

### Finding 3: Work Graph UI has filter infrastructure ready for labels

**Evidence:** Work Graph store (`web/src/lib/stores/work-graph.ts`) already supports:

- Scope filtering (`scope: "focus" | "open"`)
- Parent filtering (`parent` parameter for focus mode)
- Status-based tree building with priority sorting
- Attention badge overlays

**Source:**

- `web/src/lib/stores/work-graph.ts:88-112` - fetch with params
- `web/src/routes/work-graph/+page.svelte` - ViewToggle between issues/artifacts

**Significance:** The infrastructure for filtered views exists. Adding label-based grouping/sections would extend existing patterns, not require new architecture.

---

### Finding 4: Smart surfacing requires attention signals + labels

**Evidence:** Attention store (`attention.ts`) already computes:

- `verify` - Phase: Complete, needs orch complete
- `decide` - Investigation has recommendation
- `stuck` - Agent stuck >2h
- `recently_closed` - Recently closed, needs verification

**Source:**

- `.kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md`
- Attention badge types in `work-graph.ts:39-48`

**Significance:** "Smart features to surface competing/stale/duplicate work" can build on attention signals. Add: `stale` (no activity >30d), `duplicate-candidate` (similar titles via embedding). Labels enable grouping for these signals.

---

## Decision Forks

### Fork 1: Taxonomy Depth

**Question:** How many label prefixes, and what categories?

**Options:**

- **A: Shallow (2 prefixes)** - `area:*`, `effort:*` only
- **B: Medium (3 prefixes)** - `area:*`, `effort:*`, `status:*`
- **C: Deep (structured)** - Hierarchical namespacing (`area/subsystem:*`)

**Substrate says:**

- Principle: Session Amnesia - labels must be discoverable without memory
- Principle: Surfacing Over Browsing - fewer categories = easier to query
- Decision: Question subtypes use simple prefix pattern (`subtype:*`)

**Recommendation:** Option B (Medium - 3 prefixes)

**Reasoning:**

- `area:*` handles work domain (dashboard, spawn, beads)
- `effort:*` handles size estimation (small, medium, large) for AI spawning
- `status:*` handles meta-status beyond beads status (parked, blocked-external)
- 3 prefixes is learnable; 4+ creates friction

---

### Fork 2: Epic Migration Strategy

**Question:** What happens to existing epics?

**Options:**

- **A: Full deprecation** - Convert all epics to labels, remove epic type
- **B: Demotion** - Epics become optional "milestone" type for time-boxed releases
- **C: Coexistence** - Keep epics for explicit dependency structure, add labels for grouping

**Substrate says:**

- Principle: Evolve by Distinction - epics conflate grouping + dependency
- Principle: Coherence Over Patches - patches (adding labels alongside) vs coherent (clean separation)
- Current state: Only 4 open epics, manageable migration

**Recommendation:** Option B (Demotion)

**Reasoning:**

- Epics retain value for time-boxed milestones (e.g., "v2.0 release")
- Remove parent-child blocking - use `blocks` edge for actual dependencies
- Existing epics → `area:*` labels + optional milestone tag
- Don't add new epics for grouping; use labels instead

---

### Fork 3: UI Grouping Model

**Question:** How should Work Graph display label-based groups?

**Options:**

- **A: Filter-only** - Work Graph shows flat list, labels add filter dropdowns
- **B: Section-based** - Work Graph has collapsible sections by primary label (area)
- **C: Multi-dimension** - Toggle between groupings (priority, area, effort)

**Substrate says:**

- Principle: Surfacing Over Browsing - bring groups to user, don't require navigation
- Prior: ViewToggle already toggles issues/artifacts views
- Current: Tree view groups by parent-child hierarchy

**Recommendation:** Option C (Multi-dimension) with B as default

**Reasoning:**

- Default view: Sections by `area:*` (most common grouping need)
- Toggle: Group by priority (current) | area | effort
- Keyboard: `g` to cycle grouping modes
- Filter: Label filter in header for cross-cutting queries

**UI Mockup Description:**

```
┌─────────────────────────────────────────────────────────────────┐
│ Work Graph                              [Group: Area ▾] Filter: │
├─────────────────────────────────────────────────────────────────┤
│ ▼ area:dashboard (5 issues)                                     │
│   ○ [P2] Recently-closed attention signals not appearing...     │
│   ○ [P2] Add comprehension features to Work Graph               │
│   ○ [P3] L1 expansion only shows blocking relationships         │
├─────────────────────────────────────────────────────────────────┤
│ ▼ area:spawn (3 issues)                                         │
│   ○ [P2] Skill-specific tier defaults                           │
│   ○ [P2] Wire up project config to spawn defaults               │
├─────────────────────────────────────────────────────────────────┤
│ ▼ area:beads (4 issues)                                         │
│   ○ [P2] bd label remove changes don't persist                  │
│   ...                                                           │
├─────────────────────────────────────────────────────────────────┤
│ ▼ [unlabeled] (12 issues)                                       │
│   ○ [P1] Rebase OpenCode fork onto upstream v1.1.52             │
│   ...                                                           │
└─────────────────────────────────────────────────────────────────┘
```

---

### Fork 4: Enforcement Strategy

**Question:** How do we ensure labels get applied consistently?

**Options:**

- **A: Soft enforcement** - Suggest labels, don't require
- **B: Gate at creation** - `bd create` fails without required labels
- **C: AI auto-tagging** - AI infers labels at spawn time

**Substrate says:**

- Principle: Gate Over Remind - gates prevent bypass under cognitive load
- Principle: Infrastructure Over Instruction - don't rely on agent memory
- Prior: `triage:review` pattern works for deferred validation

**Recommendation:** Hybrid of A + C (Soft + AI auto-tagging)

**Reasoning:**

- **At spawn time:** AI auto-suggests labels based on issue title/description
- **At bd create:** Warn if no area label, but don't block (allow `--no-label-check`)
- **At orchestrator review:** Surface unlabeled issues for triage
- **Dashboard:** Show "unlabeled" section prominently as attention signal

**Why not hard gate?**

- Sometimes work is exploratory (unclear area)
- Blocking on label would slow down rapid issue creation
- Triage pattern works: create now, label during review

---

## Synthesis

**Key Insights:**

1. **Grouping ≠ Dependency** - Epics conflate static grouping with blocking relationships. Labels handle grouping (dynamic, queryable, multi-dimensional). Edges handle dependencies (explicit blocking). This distinction is the core insight.

2. **Convention over Schema** - Following the question subtype decision, labels with prefix conventions (`area:*`, `effort:*`) provide flexibility without schema changes. The decidability graph already uses `authority:*` labels. Extend the pattern.

3. **Multi-dimensional Views** - Issues have multiple facets (area, priority, effort). The UI should let users toggle between groupings, not force a single hierarchy. This is the key advantage over epics.

4. **AI-Friendly Enforcement** - Gate enforcement at creation is too rigid. AI auto-tagging at spawn time + prominent "unlabeled" surfacing creates pressure without blocking work.

**Answer to Investigation Question:**

The orchestration system should adopt a 3-prefix label taxonomy (`area:*`, `effort:*`, `status:*`) with AI-assisted tagging at spawn time. Epics should be demoted to optional milestones, with existing epics migrated to `area:*` labels. The Work Graph UI should add multi-dimensional grouping (toggle between area, priority, effort views) with collapsible sections. Enforcement should be soft (warn on missing labels) with prominent surfacing of unlabeled issues.

This approach:

- Enables dynamic grouping that adapts as work evolves
- Is AI-friendly (clear conventions, auto-tagging)
- Builds on existing label infrastructure (no schema changes)
- Provides flexibility (multi-dimensional views)
- Maintains backward compatibility (epics still valid for milestones)

---

## Structured Uncertainty

**What's tested:**

- ✅ Labels work for categorization (verified: `subtype:*` pattern in production)
- ✅ Work Graph has filter infrastructure (verified: `scope`, `parent` params in store)
- ✅ Attention badges render on issues (verified: existing attention store integration)
- ✅ Only 4 open epics to migrate (verified: `bd list --type epic` output)
- ✅ ViewToggle pattern exists for view switching (verified: issues/artifacts toggle)

**What's untested:**

- ⚠️ Multi-dimensional grouping UI usability (needs prototype/user testing)
- ⚠️ AI auto-tagging accuracy for area inference (needs testing on real spawns)
- ⚠️ Performance of label-based grouping at scale (61 issues currently, may be fine)
- ⚠️ Whether 3 prefixes is enough or too many (may need to evolve)
- ⚠️ Epic → label migration friction for in-flight work

**What would change this:**

- If AI auto-tagging has <80% accuracy → add validation step before applying
- If 3 prefixes cause confusion → reduce to 2 (`area:*`, `effort:*`)
- If unlabeled pile grows despite surfacing → add gentle gate (warning + confirm)
- If multi-dimensional views rarely used → simplify to area-only sections

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation           | Authority      | Rationale                                                         |
| ------------------------ | -------------- | ----------------------------------------------------------------- |
| Label taxonomy design    | architectural  | Cross-component: affects beads, UI, spawn, orchestrator workflows |
| Epic migration path      | architectural  | Cross-component: affects existing issues, workflows               |
| Work Graph UI changes    | implementation | Within existing component, extends patterns                       |
| AI auto-tagging in spawn | implementation | Within spawn component, extends existing context injection        |

### Recommended Approach ⭐

**3-Prefix Taxonomy with Phased Migration** - Adopt `area:*`, `effort:*`, `status:*` labels with AI-assisted enforcement and incremental epic demotion.

**Why this approach:**

- Builds on proven pattern (subtype labels already work)
- Minimal schema changes (labels already exist in beads)
- Multi-dimensional views address core grouping need
- AI enforcement reduces manual overhead
- Phased migration reduces disruption

**Trade-offs accepted:**

- Some unlabeled issues will exist (acceptable: surface them prominently)
- Migration takes multiple phases (acceptable: incremental is safer)
- 3 prefixes may need adjustment (acceptable: convention can evolve)

**Implementation sequence:**

### Phase 1: Label Taxonomy + Basic Enforcement (1-2 days)

1. Document label conventions in CLAUDE.md
2. Add `--suggest-labels` to `bd create` (AI inference from title)
3. Add warning when creating issue without `area:*` label
4. Add `bd label list-conventions` to show valid prefixes

### Phase 2: Work Graph UI Updates (2-3 days)

1. Add grouping dropdown to header (Priority | Area | Effort)
2. Implement collapsible sections by selected grouping
3. Add "unlabeled" section at bottom with attention styling
4. Add `g` keyboard shortcut to cycle groupings
5. Add label filter to header

### Phase 3: Epic Migration (1 day)

1. For each open epic, create equivalent `area:*` label
2. Remove parent-child blocking edges (convert to `blocks` if needed)
3. Keep epic as milestone reference (close when all labeled issues done)
4. Update spawn guidance to use labels, not epics for grouping

### Phase 4: Smart Surfacing (future, optional)

1. Add `stale` attention badge (no activity >30d)
2. Add `duplicate-candidate` detection (embedding similarity)
3. Add `competing` detection (same area + similar title)

### Alternative Approaches Considered

**Option B: Hard Gate Enforcement**

- **Pros:** Guaranteed label coverage
- **Cons:** Blocks rapid issue creation, friction for exploratory work
- **When to use instead:** If unlabeled pile becomes unmanageable (>20% of issues)

**Option C: Full Epic Deprecation**

- **Pros:** Cleaner separation of concerns
- **Cons:** Loses milestone grouping for releases
- **When to use instead:** If epic maintenance burden outweighs milestone value

**Rationale for recommendation:** Phased approach reduces risk, AI enforcement provides discipline without friction, multi-dimensional UI addresses the core need for flexible grouping.

---

### Implementation Details

**What to implement first:**

- Label conventions in CLAUDE.md (zero-effort, immediate value)
- Warning on `bd create` without area label (quick, high signal)
- `unlabeled` surfacing in UI (makes the problem visible)

**Things to watch out for:**

- ⚠️ Don't create labels for every concept - start narrow, expand if needed
- ⚠️ AI label inference may need training on Dylan's domain vocabulary
- ⚠️ Epic migration should preserve blocking relationships where they're real dependencies

**Areas needing further investigation:**

- Embedding-based duplicate detection (quality/performance tradeoffs)
- Label autocomplete UX in bd CLI
- Cross-project label consistency (if using labels in multiple beads repos)

**Success criteria:**

- ✅ >80% of new issues have `area:*` label within 2 weeks
- ✅ Work Graph grouping view is used (measure toggle frequency)
- ✅ "Unlabeled" section shrinks over time
- ✅ Dylan reports reduced friction finding related work
- ✅ AI agents can create appropriately labeled issues without human fixup

---

## Label Taxonomy Reference

### Proposed Prefixes

| Prefix    | Values                                            | Purpose         | Example          |
| --------- | ------------------------------------------------- | --------------- | ---------------- |
| `area:`   | dashboard, spawn, beads, cli, skill, kb, opencode | Work domain     | `area:dashboard` |
| `effort:` | small, medium, large                              | Size estimation | `effort:small`   |
| `status:` | parked, blocked-external, needs-review            | Meta-status     | `status:parked`  |

### Existing Labels (Keep)

| Label                      | Purpose                   | Notes                   |
| -------------------------- | ------------------------- | ----------------------- |
| `triage:ready`             | Ready for daemon pickup   | Existing pattern        |
| `triage:review`            | Needs orchestrator review | Existing pattern        |
| `subtype:factual`          | Factual question          | Per decision 2026-01-28 |
| `subtype:judgment`         | Judgment question         | Per decision 2026-01-28 |
| `subtype:framing`          | Framing question          | Per decision 2026-01-28 |
| `authority:implementation` | Implementation decision   | Per decidability model  |
| `authority:architectural`  | Architectural decision    | Per decidability model  |
| `authority:strategic`      | Strategic decision        | Per decidability model  |

### Area Values (Initial Set)

| Value       | Covers                                 | Examples           |
| ----------- | -------------------------------------- | ------------------ |
| `dashboard` | Work Graph, Activity Feed, WIP section | UI components      |
| `spawn`     | Agent spawning, tier system, workspace | Orchestration      |
| `beads`     | Issue tracking, labels, dependencies   | bd CLI             |
| `cli`       | orch commands, completion, status      | Terminal interface |
| `skill`     | Skill system, skillc, templates        | Process docs       |
| `kb`        | Knowledge artifacts, investigations    | Knowledge system   |
| `opencode`  | Fork maintenance, session management   | AI backend         |

---

## References

**Files Examined:**

- `web/src/lib/stores/work-graph.ts` - Work Graph store, filter/grouping infrastructure
- `web/src/routes/work-graph/+page.svelte` - Work Graph page, ViewToggle pattern
- `.kb/decisions/2026-01-28-question-subtype-encoding-labels.md` - Label convention precedent
- `.kb/models/decidability-graph.md` - Authority labels, question subtypes
- `.kb/investigations/archived/2026-02-02-inv-audit-work-graph-design-docs.md` - UI implementation status
- `.kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md` - Attention signals

**Commands Run:**

```bash
# Check open issues and labels
bd list --status open -n 50

# Check epics
bd list --type epic

# Show epic structure
bd show orch-go-21260
bd graph orch-go-21260

# Get ready queue
bd ready -n 50
```

**Related Artifacts:**

- **Decision:** `.kb/decisions/2026-01-28-question-subtype-encoding-labels.md` - Establishes label convention pattern
- **Model:** `.kb/models/decidability-graph.md` - Authority labels, decidability
- **Investigation:** `.kb/investigations/2026-02-02-design-work-graph-unified-attention-model.md` - Attention surface design

---

## Investigation History

**2026-02-05 19:00:** Investigation started

- Initial question: How to replace rigid epics with dynamic label-based grouping?
- Context: Dylan uses Work Graph daily, epics are hard for AI to manage, issues pile up without dynamic grouping

**2026-02-05 19:15:** Context gathered

- Read existing work graph investigations, decidability graph, label decision
- Analyzed 4 open epics, existing label patterns
- Identified core distinction: grouping vs dependency

**2026-02-05 19:30:** Design forks navigated

- Fork 1: 3-prefix taxonomy (area, effort, status)
- Fork 2: Epic demotion (not deprecation)
- Fork 3: Multi-dimensional UI grouping
- Fork 4: Soft enforcement + AI auto-tagging

**2026-02-05 19:45:** Investigation completed

- Status: Complete
- Key outcome: 3-prefix label taxonomy with phased migration, multi-dimensional UI, AI enforcement
