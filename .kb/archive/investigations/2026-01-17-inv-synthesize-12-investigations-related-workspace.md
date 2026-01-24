<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** A formal "Workspace Lifecycle Model" now exists at `.kb/models/workspace-lifecycle-model.md`, synthesizing 13 investigations into a unified framework covering three workspace types, four lifecycle states, and three storage locations.

**Evidence:** Analyzed 13 workspace investigations (Dec 2025 - Jan 2026); verified current model captures all key patterns; identified 3 gaps and 2 remaining ambiguities.

**Knowledge:** Workspaces divide into three distinct types (Worker, Spawned Orchestrator, Interactive Session) with different locations, naming conventions, and completion artifacts; archival remains the only unautomated lifecycle step.

**Next:** Close - model is formalized and complete; remaining gap (auto-archival) is a feature request, not a model gap.

**Promote to Decision:** recommend-yes - The workspace lifecycle model establishes authoritative patterns for workspace creation, naming, and cleanup.

---

# Investigation: Synthesize 12 Investigations Related Workspace

**Question:** What is the complete workspace lifecycle model, how do interactive and spawned workspaces differ, and what is the consistent strategy for naming and cleanup?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Workspace Taxonomy Has Three Distinct Types

**Evidence:** Across 13 investigations, workspaces consistently fall into three categories with distinct characteristics:

| Type | Location | Naming | Completion Artifact | Beads Tracked |
|------|----------|--------|---------------------|---------------|
| **Worker** | `{project}/.orch/workspace/og-{skill}-{slug}-{date}-{hex}/` | 4-char hex suffix | SYNTHESIS.md (full tier) or none (light tier) | Yes |
| **Spawned Orchestrator** | `{project}/.orch/workspace/og-orch-{slug}-{date}-{hex}/` | 4-char hex suffix | SESSION_HANDOFF.md | No |
| **Interactive Session** | `~/.orch/session/{date}/` | Date-based (no suffix) | SESSION_HANDOFF.md | No |

**Source:**
- `2026-01-06-inv-workspace-session-architecture.md` - Established three-layer model
- `2026-01-05-inv-orchestrator-workspaces-clear-visual-distinction.md` - og-orch-* naming
- `2026-01-09-inv-create-orchestrator-workspace-session-start.md` - Interactive session location

**Significance:** The confusion between "interactive" and "spawned" workspaces is now definitively resolved. Interactive sessions use global ~/.orch/session/ with date-based directories; spawned agents use project-local .orch/workspace/ with unique hex suffixes.

---

### Finding 2: Lifecycle Has Four States with Clear Transitions

**Evidence:** The workspace lifecycle follows a consistent flow:

```
Spawn → Execute → Complete → Archive
  │        │          │          │
  └──┬─────┴─────┬────┴─────┬────┘
     │           │          │
  [Create]   [Active]  [Resolved]  [Cleaned]
```

State indicators:
- **Created:** Directory exists with context file (SPAWN_CONTEXT.md or *_CONTEXT.md)
- **Active:** `.session_id` points to live OpenCode session
- **Resolved:** Completion artifact exists (SYNTHESIS.md or SESSION_HANDOFF.md)
- **Archived:** Moved to `{project}/.orch/workspace/archived/`

**Source:**
- `2025-12-21-inv-workspace-lifecycle-when-workspaces-created.md` - Creation and "deletion" behavior
- `2026-01-06-inv-define-workspace-cleanup-strategy-context.md` - Archival implementation
- `2026-01-07-inv-address-340-active-workspaces-completion.md` - Archival gap analysis

**Significance:** The lifecycle is well-defined. The gap is in automation: archival requires manual `orch clean --stale` invocation.

---

### Finding 3: Naming Collision Bug Was Fixed with Hex Suffix

**Evidence:** Prior to fix (commit Jan 5, 2026), workspace names used format `{proj}-{skill}-{slug}-{date}` without uniqueness, causing same-day sessions to overwrite each other's artifacts.

Fix added:
- 4-character random hex suffix via `crypto/rand`
- Workspace existence check before spawn
- 65,536 possible suffixes per day per task

**Source:**
- `2026-01-05-debug-orchestrator-workspace-name-collision-bug.md` - Root cause and fix

**Significance:** Naming collision is a **solved problem** for spawned workspaces. Interactive sessions still use date-based directories (one per day by design).

---

### Finding 4: Three Storage Locations Serve Different Purposes

**Evidence:** Workspaces exist in three locations with different semantics:

| Location | Scope | Purpose |
|----------|-------|---------|
| `{project}/.orch/workspace/` | Project-local | Worker and spawned orchestrator execution |
| `{project}/.orch/workspace/archived/` | Project-local | Completed workspaces after cleanup |
| `~/.orch/session/` | Global | Interactive human sessions |

**Source:**
- `2025-12-21-inv-beads-kb-workspace-relationships-how.md` - Three-layer artifact system
- `2026-01-09-inv-create-orchestrator-workspace-session-start.md` - Interactive session location

**Significance:** The location distinction is intentional: workers need project-local workspaces to run tests and access codebase; interactive sessions use global location because humans work across projects.

---

### Finding 5: Cleanup Strategy Exists But Is Manual

**Evidence:** Two cleanup mechanisms exist:

1. **`orch clean --stale`** - Archives workspaces older than N days (default: 7)
   - Uses file-based completion detection (SYNTHESIS.md, .tier, .beads_id)
   - Fast (sub-second for 300+ workspaces)
   - Moves to `archived/` subdirectory

2. **`orch doctor --sessions`** - Cross-references three layers
   - Workspaces ↔ OpenCode sessions ↔ Orchestrator registry
   - Detects orphaned workspaces, orphaned sessions, zombie sessions
   - Diagnostic only, no automatic cleanup

**Source:**
- `2026-01-06-inv-define-workspace-cleanup-strategy-context.md` - Implementation
- `2026-01-06-inv-add-orch-doctor-sessions-workspace.md` - Cross-reference enhancement
- `2026-01-07-inv-address-340-active-workspaces-completion.md` - Archival gap

**Significance:** The tools exist but aren't automated. Workspace accumulation (340+ observed) is a recurring issue requiring periodic manual cleanup.

---

### Finding 6: Additional Workspace Metadata Files

**Evidence:** Beyond core files, workspaces track additional state:

| File | Purpose | Added By |
|------|---------|----------|
| `.tier` | light/full/orchestrator - verification rules | Spawn |
| `.session_id` | OpenCode session ID link | Spawn |
| `.beads_id` | Beads issue tracking link | Spawn |
| `.spawn_time` | Nanosecond timestamp for age calculation | Spawn |
| `.review-state.json` | Synthesis recommendation review state | `orch review done` |

**Source:**
- `2025-12-26-inv-add-review-state-tracking-workspace.md` - ReviewState implementation
- `2026-01-06-inv-workspace-session-architecture.md` - Metadata file documentation

**Significance:** Workspaces are self-describing via metadata files. This enables file-based status detection without API calls.

---

## Synthesis

**Key Insights:**

1. **Three Types, Three Locations, One Lifecycle** - Despite surface differences, all workspace types follow the same Spawn→Execute→Complete→Archive lifecycle. The variation is in location (project-local vs global), naming (hex suffix vs date-based), and completion artifact (SYNTHESIS.md vs SESSION_HANDOFF.md).

2. **Interactive vs Spawned is Now Clear** - Interactive sessions (`~/.orch/session/{date}/`) are for human-driven work across projects with daily continuity. Spawned workspaces (`{project}/.orch/workspace/og-{skill}-*/`) are for autonomous agents with goal-atomic execution.

3. **Archival is the Only Manual Step** - Creation, execution, and completion are fully automated. Only archival requires explicit action (`orch clean --stale`). This is the remaining lifecycle gap.

4. **File-Based State Detection is Intentional** - Workspaces are designed to be self-describing via metadata files (.tier, .session_id, .beads_id, .spawn_time). This enables fast bulk operations without API calls.

**Answer to Investigation Question:**

The **Workspace Lifecycle Model** is:
- **Three types:** Worker (project-local, hex suffix, SYNTHESIS.md), Spawned Orchestrator (project-local, hex suffix, SESSION_HANDOFF.md), Interactive Session (global, date-based, SESSION_HANDOFF.md)
- **Four states:** Spawn → Execute → Complete → Archive
- **Naming strategy:** `og-{skill}-{slug}-{date}-{hex}` for spawned; `{date}` for interactive
- **Cleanup strategy:** `orch clean --stale` archives after 7 days; `orch doctor --sessions` diagnoses orphans

The remaining gap is **automated archival** - recommend adding to `orch complete` or daemon poll.

---

## Structured Uncertainty

**What's tested:**

- ✅ Three workspace types exist with documented locations (verified: read 13 investigations)
- ✅ Hex suffix prevents naming collisions (verified: code trace in collision bug investigation)
- ✅ `orch clean --stale` archives old workspaces (verified: investigation shows 132 archivable)
- ✅ Metadata files enable file-based detection (verified: code in completion verification)

**What's untested:**

- ⚠️ Auto-archival in daemon/complete (recommended but not implemented)
- ⚠️ Cross-project workspace aggregation (archived investigation incomplete)
- ⚠️ Performance at >1000 workspaces (not load tested)

**What would change this:**

- Finding would be wrong if a fourth workspace type exists (not found in codebase)
- Finding would be incomplete if new metadata files are added without documentation
- Recommendation would change if manual archival is intentional (not just missing feature)

---

## Implementation Recommendations

**Purpose:** Formalize the workspace lifecycle model and address the archival gap.

### Recommended Approach ⭐

**Accept current model + automate archival** - The existing workspace-lifecycle-model.md is accurate; the only action needed is automating the archival step.

**Why this approach:**
- Model synthesis is complete (all 13 investigations reviewed)
- Existing model at `.kb/models/workspace-lifecycle-model.md` already captures key patterns
- Archival is the only lifecycle step requiring human intervention

**Trade-offs accepted:**
- Manual `orch clean --stale` continues to work (not breaking change)
- Auto-archival may archive workspaces user wants to keep (mitigate with --no-archive flag)

**Implementation sequence:**
1. Review and enhance existing model (minor updates only)
2. Create beads issue for auto-archival feature
3. Implement in `orch complete` post-completion hook

### Alternative Approaches Considered

**Option B: Redesign workspace system**
- **Pros:** Could unify all three types
- **Cons:** Massive scope; current design works well; investigations found no fundamental flaws
- **When to use instead:** If fundamental problems emerge with current design

**Option C: Add scheduled cleanup (cron/daemon)**
- **Pros:** Fully automated, no user action needed
- **Cons:** May clean during active work; complexity for marginal benefit
- **When to use instead:** If `orch complete` integration is problematic

---

## References

**Investigations Examined:**
1. `2025-12-21-inv-workspace-lifecycle-when-workspaces-created.md` - Lifecycle fundamentals
2. `2025-12-21-inv-beads-kb-workspace-relationships-how.md` - Three-layer artifact system
3. `2025-12-26-inv-add-review-state-tracking-workspace.md` - ReviewState metadata
4. `2025-12-26-inv-implement-multi-project-workspace-aggregation.md` - Archived/incomplete
5. `2026-01-05-inv-orchestrator-workspaces-clear-visual-distinction.md` - og-orch-* naming
6. `2026-01-05-debug-orchestrator-workspace-name-collision-bug.md` - Hex suffix fix
7. `2026-01-06-inv-define-workspace-cleanup-strategy-context.md` - Cleanup implementation
8. `2026-01-06-inv-workspace-session-architecture.md` - Three-layer model
9. `2026-01-06-inv-add-orch-attach-workspace-command.md` - Partial name matching
10. `2026-01-06-inv-extend-orch-resume-work-workspace.md` - Resume by workspace
11. `2026-01-06-inv-add-orch-doctor-sessions-workspace.md` - Cross-reference enhancement
12. `2026-01-07-inv-address-340-active-workspaces-completion.md` - Archival gap analysis
13. `2026-01-09-inv-create-orchestrator-workspace-session-start.md` - Interactive session clarification

**Model Updated:**
- `.kb/models/workspace-lifecycle-model.md` - Now verified as authoritative

---

## Investigation History

**2026-01-17:** Investigation started
- Initial question: Synthesize 12 workspace investigations into coherent model
- Context: kb reflect identified 8 workspace-related investigations for synthesis

**2026-01-17:** Found 13 investigations (not 12)
- Added 2026-01-09 interactive session investigation
- Discovered existing model at `.kb/models/workspace-lifecycle-model.md`

**2026-01-17:** Investigation completed
- Status: Complete
- Key outcome: Workspace lifecycle model is formalized and verified; archival automation is the remaining gap
