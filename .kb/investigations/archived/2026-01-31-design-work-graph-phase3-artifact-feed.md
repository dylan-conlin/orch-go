# Design: Work Graph Phase 3 - Artifact Feed

**Date:** 2026-01-31
**Status:** Complete
**Owner:** Dylan + Claude
**Parent Issue:** orch-go-21155
**Parent Epic:** orch-go-21121

---

## Summary (D.E.K.N.)

**Delta:** Artifact Feed is a toggle view alongside Issues in Work Graph, showing knowledge outputs (investigations, decisions, models, guides) organized by "needs decision" vs "recently updated". Work in Progress section stays visible across both views.

**Evidence:** Design session identified that users don't want to browse completed issues - they want to browse consequential artifacts. Transition between views should be seamless, synced to orchestrator context.

**Knowledge:** Artifacts have two attention modes: needs decision (investigations with recommendations, proposed decisions) vs needs awareness (recently created/updated). Keyboard navigation should be consistent across both views.

**Next:** Implementation of Phase 3 features.

---

## Design Decisions

### 1. View Toggle, Not Separate Tab

**Decision:** Artifacts is a toggle within Work Graph, not a separate top-level tab.

```
Work Graph              [Issues]  [Artifacts]     orch-go
                             ↑ toggle               ↑ from context
```

**Rationale:**
- Seamless transition between task tracking and knowledge browsing
- Same spatial layout, content switches
- Both sync to orchestrator context (project_dir)

---

### 2. Work in Progress Stays Visible

**Decision:** The pinned Work in Progress section remains visible when in Artifacts view.

**Rationale:**
- Context continuity - often reviewing artifacts an agent just produced
- Lightweight - doesn't take much space
- One monitoring surface - don't have to switch back to check agent status
- Matches mental model - system activity applies to both views

**Structure:**

```
┌─────────────────────────────────────────────────────────────┐
│ Work Graph              [Issues]  [Artifacts]     orch-go   │
├─────────────────────────────────────────────────────────────┤
│ WORK IN PROGRESS (always visible)                           │
│  ▶ ...21154  Design: Agent Overlay        Complete          │
│  ▶ ...21155  Design: Artifact Feed        Running...        │
├─────────────────────────────────────────────────────────────┤
│ (below here changes based on toggle)                        │
└─────────────────────────────────────────────────────────────┘
```

---

### 3. Artifact Types

| Artifact Type | Location | Status Signals |
|---------------|----------|----------------|
| Investigations | `.kb/investigations/` | Status field (Active, Complete), has recommendation |
| Decisions | `.kb/decisions/` | Status field (Proposed, Accepted, Superseded) |
| Models | `.kb/models/` | (reference material, no status) |
| Guides | `.kb/guides/` | (reference material, no status) |
| Principles | `.kb/principles.md` | (single file, rarely changes) |

---

### 4. Attention Categories

**Two categories:**

| Category | What It Contains | Trigger |
|----------|------------------|---------|
| **Needs Decision** | Actionable items requiring human input | Investigation with recommendation, Decision with status: Proposed |
| **Recently Updated** | Awareness items | Created or modified within time filter |

**"Needs Decision" criteria:**

| Artifact | Needs Decision When |
|----------|---------------------|
| Investigation | Status: Active AND has recommendation section |
| Investigation | Status: Active AND stale (>7d no update) |
| Decision | Status: Proposed |

---

### 5. Artifacts View Layout

```
┌─────────────────────────────────────────────────────────────┐
│ ARTIFACTS                                         orch-go   │
│                                                             │
│ NEEDS DECISION (2)                                          │
│  → Investigation: semantic-metadata-encoding                │
│    Has recommendation · Active · 2h ago                     │
│  → Decision: deliverables-schema-approach                   │
│    Proposed · Waiting for acceptance · 1d ago               │
│                                                             │
│ RECENTLY UPDATED (5)                              [7d ▾]    │
│  → Investigation: disabled-gate-patterns     Complete  2h   │
│  → Model: issue-lifecycle                              1d   │
│  → Guide: agent-spawning-patterns                      2d   │
│  → Decision: override-with-logging           Accepted  3d   │
│  → Investigation: work-graph-phase1          Complete  5d   │
│                                                             │
│ BROWSE BY TYPE                                              │
│  Investigations (12)  Decisions (8)  Models (3)  Guides (2) │
└─────────────────────────────────────────────────────────────┘
```

---

### 6. Information Hierarchy

| Level | Content |
|-------|---------|
| **L0 (row)** | Type icon, title (from filename/frontmatter), status badge, age |
| **L1 (expand)** | Summary/first paragraph, recommendation (if investigation), related issue |
| **L2 (side panel)** | Full artifact content rendered, linked issues, metadata |

---

### 7. Keyboard Navigation

Consistent with Issues view:

| Key | Action |
|-----|--------|
| `j/k` | Move selection up/down |
| `l/Enter` | Open side panel (L2 detail) |
| `h/Esc` | Close side panel, or go back |
| `g/G` | Jump to top/bottom |
| `Tab` | Toggle between Issues ↔ Artifacts |
| `C` | Copy selected path (artifact) or ID (issue) |
| `1/2/3` | Jump to section (Needs Decision / Recent / Browse) |

---

### 8. Time Filter

- Default: 7 days
- Options: 24h, 7d, 30d, all
- Persisted in localStorage (same pattern as dashboard filters)
- Applies to "Recently Updated" section

---

### 9. Context Sync

- Both views read from `orchestratorContext` store
- `project_dir` determines which `.kb/` to browse
- Switching tmux sessions updates both views
- Uses existing `/api/context` polling mechanism

---

## Out of Scope (Future)

- **cmd-k search** - Unified search across issues, artifacts, agents (Phase 4+)
- **Global/cross-project artifacts** - Browse `~/.kb/` or domain-level
- **Inline editing** - Edit artifacts directly in dashboard
- **Artifact creation** - Create new investigations/decisions from UI

---

## Implementation Notes

### Data Requirements

- Parse `.kb/` directory structure
- Extract frontmatter from markdown files (status, date, title)
- Detect "has recommendation" in investigations
- Track file modification times for "recently updated"

### New API Endpoint

`/api/kb/artifacts?project_dir=X&since=7d`

Returns:
```json
{
  "needs_decision": [...],
  "recent": [...],
  "by_type": {
    "investigations": [...],
    "decisions": [...],
    "models": [...],
    "guides": [...]
  }
}
```

### Components to Build

1. `ArtifactFeed` - main view with three sections
2. `ArtifactRow` - row with L0/L1 content
3. `ArtifactSidePanel` - L2 detail with rendered markdown
4. `TimeFilter` - dropdown with persistence
5. `ViewToggle` - Issues/Artifacts toggle in header

### Keyboard Nav Extension

- Add `Tab` handler to toggle views
- Add `1/2/3` handlers for section jump
- Ensure focus management works across view switches

---

## Related Artifacts

- Phase 2 design: `.kb/investigations/2026-01-31-design-work-graph-phase2-agent-overlay.md`
- Original work graph design: `.kb/investigations/2026-01-30-design-work-graph-dashboard-tab.md`
- Phase 3 issue: `orch-go-21155`
- Parent epic: `orch-go-21121`
