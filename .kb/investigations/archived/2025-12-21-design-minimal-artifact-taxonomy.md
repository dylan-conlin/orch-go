## Summary (D.E.K.N.)

**Delta:** Minimal artifact set is 5 types organized by temporal lifecycle: ephemeral (SPAWN_CONTEXT.md, SYNTHESIS.md), persistent (investigations, decisions), and operational (beads comments); with explicit promotion paths and handoff protocols.

**Evidence:** Synthesized 6 investigations covering workspace lifecycle, knowledge promotion, session boundaries, beads-kb-workspace relationships, multi-agent synthesis, and failure mode artifacts. Pattern analysis of 100+ workspaces, 140+ investigations, 33 abandon events.

**Knowledge:** The three-tier temporal model (ephemeral/persistent/operational) is the organizing principle; artifacts should live where their lifecycle dictates, not where work happens; promotion paths must be explicit but friction is intentional for curation.

**Next:** Promote to decision. Ready for implementation: add FAILURE_REPORT.md template, standardize SESSION_HANDOFF.md for orchestrator.

**Confidence:** High (90%) - Comprehensive synthesis of parallel investigations with strong evidence.

---

# Design: Minimal Artifact Set Specification

**Question:** What is the minimal set of artifacts needed for amnesia-resilient orchestration, and what are their relationships and lifecycle rules?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** Promote to decision
**Status:** Complete
**Confidence:** High (90%)

---

## Problem Framing

### Design Question

Given the constraint of session amnesia (Claude has no memory between sessions), what is the minimal set of artifacts that enables:
1. Zero-context-loss resumption for any Claude instance
2. Discoverable knowledge across agents and sessions
3. Clear lifecycle boundaries (creation → usage → archival/promotion)

### Success Criteria

- Any fresh Claude can resume orchestrator or worker work with artifacts alone
- `kb context` finds relevant knowledge regardless of which agent created it
- Artifacts don't accumulate unbounded without value
- Failure modes are captured, not just successes
- Multi-agent synthesis has clear protocol

### Constraints

- **Session amnesia** - Next Claude won't remember conversations
- **Local-first** - Files over databases, git over external services
- **Progressive disclosure** - TLDR first, details available
- **Evidence hierarchy** - Code is truth, artifacts are hypotheses

### Scope

**In scope:** Artifact types, locations, lifecycle rules, relationships, handoff protocols
**Out of scope:** Implementation details of tooling changes, specific template content beyond structure

---

## Findings from Input Investigations

### Finding 1: Three Temporal Categories Define Artifact Placement

**Evidence (from .1 - Workspace Lifecycle):**
Workspaces are created at spawn, never automatically deleted. Registry tracks state transitions (active→completed→abandoned→deleted) but "deleted" is soft-delete only. 150+ workspaces persist indefinitely. This is by design - workspaces serve as permanent artifacts for post-mortems.

**Evidence (from .4 - Beads-KB-Workspace Relationships):**
Three-layer architecture with explicit bidirectional links:
- Beads tracks WIP issues with comments (investigation_path, phase transitions)
- KB stores knowledge artifacts (investigations/decisions)
- Workspaces provide ephemeral agent execution context

**Synthesis:** Artifacts fall into three temporal categories:

| Category | Lifecycle | Location | Examples |
|----------|-----------|----------|----------|
| **Ephemeral** | Session-bound | `.orch/workspace/{name}/` | SPAWN_CONTEXT.md, SYNTHESIS.md |
| **Persistent** | Project-lifetime | `.kb/` | Investigations, decisions, guides |
| **Operational** | Work-in-progress | `.beads/` | Issues, comments, phase tracking |

**Significance:** Placement follows lifecycle, not workflow convenience. This explains why investigations go to `.kb/` (discoverable via `kb context`) rather than staying in workspaces (ephemeral, cleaned eventually).

---

### Finding 2: Explicit Promotion Paths with Intentional Friction

**Evidence (from .2 - Knowledge Promotion Paths):**
Four documented promotion paths:
1. `kn constraint` → `.kb/principles.md` (when universal)
2. `kn decide` → `.kb/decisions/` (when architecturally significant)
3. Investigation → Decision (when recommendation accepted)
4. Investigation → Guide (when reusable pattern emerges)

CLI mechanisms exist (`kb promote`, `kb publish`) but low usage (39 kn entries → 1 kb decision). Global `~/.kb/decisions/` doesn't exist. This appears intentional - curation over accumulation.

**Synthesis:** Promotion is manual by design. The friction prevents knowledge accumulation without value. The criteria for promotion are:
- Must be tested (emerged from actual problems)
- Must be generative (guides future decisions)
- Must have teeth (violation causes real problems)

**Significance:** Don't automate promotion. The low rate is healthy curation, not a bug.

---

### Finding 3: Session Boundaries Need Explicit Handoff Protocols

**Evidence (from .3 - Orchestrator Session Boundaries):**
Three distinct session types with different boundary patterns:

| Session Type | Boundary Trigger | Handoff Mechanism |
|--------------|------------------|-------------------|
| **Worker** | `Phase: Complete` + `/exit` | SPAWN_CONTEXT.md → SYNTHESIS.md |
| **Orchestrator** | Context full OR explicit end | session-transition skill |
| **Cross-session** | End of working day | SESSION_HANDOFF.md (manual) |

Worker boundaries are strictly enforced via SPAWN_CONTEXT.md template. Orchestrator boundaries are state-detected but manual. Cross-session boundaries use SESSION_HANDOFF.md but inconsistently.

**Synthesis:** Worker handoff is solved (SYNTHESIS.md). Orchestrator handoff is the gap - needs standardized SESSION_HANDOFF.md protocol.

**Significance:** Adopt skillc's SESSION_HANDOFF.md pattern for orchestrator sessions.

---

### Finding 4: Multi-Agent Synthesis Works via Isolation + D.E.K.N.

**Evidence (from .5 - Multi-Agent Synthesis):**
- Workspace isolation prevents file-level conflicts (100+ workspaces, 0 git conflicts)
- Registry uses timestamp-based last-write-wins for concurrent access
- 52 SYNTHESIS.md files follow D.E.K.N. pattern consistently
- `orch review` aggregates multi-agent outputs into single view

Logical conflict detection (Agent A says "do X", Agent B says "do Y") is manual - no automation. This is acceptable because agents typically work on different issues.

**Synthesis:** Current architecture handles multi-agent synthesis well. SYNTHESIS.md + `orch review` + orchestrator judgment = sufficient.

**Significance:** No new artifacts needed for multi-agent synthesis. The pattern works.

---

### Finding 5: Failure Artifacts Are the Missing Piece

**Evidence (from .6 - Failure Mode Artifacts):**
Significant asymmetry: successful agents create SYNTHESIS.md, abandoned agents leave only SPAWN_CONTEXT.md. 33 abandonment events in events.jsonl with minimal data (agent_id, beads_id, window_id only).

Three failure modes identified:
1. Context exhaustion (~60%) - agent makes progress, then stops
2. External blocker (~25%) - agent asks question, never answered
3. Unresponsive/stuck (~15%) - no progress after spawn

No FAILURE_REPORT.md template exists. No `.orch/knowledge/spawning-lessons/` despite documentation mentioning it.

**Synthesis:** Add FAILURE_REPORT.md with layered capture:
1. **Always (automatic):** Enhanced `agent.abandoned` event
2. **Usually (encouraged):** `orch abandon --reason "..."` → beads comment
3. **Sometimes (for patterns):** Manual post-mortem in `.orch/knowledge/spawning-lessons/`

**Significance:** Failure capture is the main gap. Knowledge is lost when agents fail.

---

### Finding 6: D.E.K.N. is the Universal Handoff Structure

**Evidence (across all investigations):**
Every investigation uses D.E.K.N. (Delta, Evidence, Knowledge, Next) structure:
- SYNTHESIS.md template uses D.E.K.N. explicitly
- Investigation template uses D.E.K.N. in summary
- SESSION_HANDOFF.md pattern mirrors D.E.K.N. concepts

**Synthesis:** D.E.K.N. is the proven handoff structure. Apply to all artifacts:
- **Delta:** What changed/was discovered
- **Evidence:** How we know (primary sources)
- **Knowledge:** What it means (insights, constraints)
- **Next:** What should happen (recommendation)

**Significance:** Standardize on D.E.K.N. as the universal artifact structure for session handoff.

---

## Synthesis: Minimal Artifact Taxonomy

### The Five Essential Artifacts

| Artifact | Location | Creator | Temporal | Purpose |
|----------|----------|---------|----------|---------|
| **SPAWN_CONTEXT.md** | `.orch/workspace/{name}/` | `orch spawn` | Ephemeral | Agent initialization: skill, task, beads ID, deliverables |
| **SYNTHESIS.md** | `.orch/workspace/{name}/` | Worker agent | Ephemeral | Session outcome: D.E.K.N. summary, delta, recommendation |
| **Investigation** | `.kb/investigations/` | Worker agent | Persistent | Deep research: question → findings → answer |
| **Decision** | `.kb/decisions/` | Orchestrator | Persistent | Architectural choice: promoted from investigation |
| **Beads Comments** | `.beads/` (via `bd comment`) | Any agent | Operational | Phase tracking, investigation_path, blockers |

### Supplementary Artifacts

| Artifact | Location | Creator | Temporal | Purpose |
|----------|----------|---------|----------|---------|
| **SESSION_HANDOFF.md** | `.orch/` | Orchestrator | Persistent | Cross-session context: priority work, decisions, questions |
| **FAILURE_REPORT.md** | `.orch/workspace/{name}/` | Orchestrator on abandon | Ephemeral | Failure context: failure mode, what tried, retry guidance |
| **kn entries** | `.kn/entries.jsonl` | Any agent | Operational | Quick decisions, constraints, tried/failed, questions |

### Artifact Relationships

```
┌─────────────────────────────────────────────────────────────┐
│ OPERATIONAL (Work Tracking)                                  │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ .beads/                                                  │ │
│ │  └── issues.jsonl                                        │ │
│ │       └── comments[]                                     │ │
│ │            ├── "Phase: X"                                │ │
│ │            ├── "investigation_path: Y"  ──────────┐      │ │
│ │            └── "agent_metadata: {...}"            │      │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                                                       │
         ┌─────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────────────┐
│ PERSISTENT (Knowledge)                                       │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ .kb/                                                     │ │
│ │  ├── investigations/                                     │ │
│ │  │    └── YYYY-MM-DD-*.md  ◄─── linked_issues (bidir)   │ │
│ │  ├── decisions/                                          │ │
│ │  │    └── YYYY-MM-DD-*.md  ◄─── promoted from inv       │ │
│ │  └── guides/                                             │ │
│ │       └── *.md  ◄─── evolved from investigations        │ │
│ └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
         ▲
         │
         │ (creates, references)
         │
┌─────────────────────────────────────────────────────────────┐
│ EPHEMERAL (Session)                                          │
│ ┌─────────────────────────────────────────────────────────┐ │
│ │ .orch/workspace/{name}/                                  │ │
│ │  ├── SPAWN_CONTEXT.md  ──► contains beads_id            │ │
│ │  ├── SYNTHESIS.md  ──► references investigation_path    │ │
│ │  └── FAILURE_REPORT.md  (on abandon)                    │ │
│ └─────────────────────────────────────────────────────────┘ │
│                                                              │
│ .orch/                                                       │
│  └── SESSION_HANDOFF.md  (orchestrator cross-session)       │
└─────────────────────────────────────────────────────────────┘
```

### Lifecycle Rules

**Creation:**
- `SPAWN_CONTEXT.md`: At `orch spawn`, before agent execution
- `SYNTHESIS.md`: By agent, before `/exit`
- `Investigation`: By agent via `kb create investigation`, during work
- `Decision`: By orchestrator via promotion, after accepting recommendation
- `Beads comments`: By agent via `bd comment`, at phase transitions
- `SESSION_HANDOFF.md`: By orchestrator, at session end or context exhaustion
- `FAILURE_REPORT.md`: By orchestrator on `orch abandon --reason`

**Archival/Cleanup:**
- Workspaces: Persist indefinitely (by design, for post-mortems)
- Investigations/Decisions: Never archived (cumulative knowledge)
- Beads issues: Close via `bd close`, but comments persist
- SESSION_HANDOFF.md: Overwritten each session (one active copy)
- kn entries: Archived via `kn archive` after promotion

**Promotion:**
- kn entry → Decision: `kb promote <kn-id>` (manual, when architecturally significant)
- Investigation → Decision: Manual (read investigation, accept recommendation, `kb create decision`)
- kn constraint → Principle: Manual (edit `~/.kb/principles.md`) when universal
- Investigation → Guide: Manual when reusable pattern emerges

### Handoff Protocols

**Worker Session End:**
```
1. Create SYNTHESIS.md (D.E.K.N. structure)
2. Run: `bd comment <beads-id> "Phase: Complete - [summary]"`
3. Run: `/exit`
```

**Orchestrator Session End:**
```
1. Update SESSION_HANDOFF.md with:
   - TLDR (current state)
   - Priority Work (what's next)
   - Key Decisions (made this session)
   - Open Questions
2. Commit SESSION_HANDOFF.md
```

**Agent Abandonment:**
```
1. Orchestrator runs: `orch abandon <id> --reason "[reason]"`
2. System creates beads comment: "Phase: Abandoned - [reason]"
3. (Optional) Create FAILURE_REPORT.md in workspace
4. (For patterns) Create `.orch/knowledge/spawning-lessons/` post-mortem
```

---

## Implementation Recommendations

### Recommended Approach ⭐

**Adopt Five Essential + Three Supplementary Artifacts** - Minimal set with explicit lifecycle rules.

**Why this approach:**
- Based on Session Amnesia principle (foundational)
- Three-tier temporal model provides clear placement rules
- D.E.K.N. standardizes handoff across all artifacts
- Failure capture addresses main gap discovered

**Trade-offs accepted:**
- Promotion paths remain manual (intentional friction for curation)
- Workspaces persist indefinitely (trade disk space for post-mortem value)
- Orchestrator handoff requires discipline (SESSION_HANDOFF.md not automated)

**Implementation sequence:**
1. Create `FAILURE_REPORT.md` template in `.orch/templates/`
2. Modify `orch abandon` to accept `--reason` flag and create beads comment
3. Adopt SESSION_HANDOFF.md template in `.orch/templates/`
4. Update orchestrator skill to mandate SESSION_HANDOFF.md at session end
5. Create `.orch/knowledge/spawning-lessons/` directory with README

### Alternative Approaches Considered

**Option B: Collapse workspace artifacts into .kb/**
- **Pros:** Single location for all knowledge
- **Cons:** Loses ephemeral/persistent distinction; breaks `kb context` with session noise
- **When to use instead:** Never - the temporal distinction is essential

**Option C: Automate all promotion paths**
- **Pros:** Less manual work
- **Cons:** Removes curation; would flood decisions with noise
- **When to use instead:** Never - friction is intentional

**Option D: Remove SYNTHESIS.md requirement**
- **Pros:** Simpler agent protocol
- **Cons:** Loses structured handoff; orchestrator can't review efficiently
- **When to use instead:** Never - SYNTHESIS.md is critical for orchestrator workflow

**Rationale for recommendation:** This is the minimal set that satisfies amnesia-resilience. Adding fewer artifacts loses essential context; adding more creates maintenance burden without value.

---

### Implementation Details

**What to implement first:**
1. FAILURE_REPORT.md template (addresses main gap)
2. `orch abandon --reason` enhancement (low effort, high value)
3. SESSION_HANDOFF.md template (orchestrator gap)

**Things to watch out for:**
- ⚠️ FAILURE_REPORT.md should be lightweight (not as detailed as SYNTHESIS.md)
- ⚠️ SESSION_HANDOFF.md can grow stale - needs explicit update discipline
- ⚠️ Don't over-automate promotion - manual curation is a feature

**Areas needing further investigation:**
- Whether SYNTHESIS.md should be archived before workspace cleanup (vs relying on kb investigation)
- Versioning strategy for SESSION_HANDOFF.md (timestamp? git history?)
- Cross-project orchestration (multiple repos, multiple handoffs)

**Success criteria:**
- ✅ Fresh Claude can resume orchestrator work via SESSION_HANDOFF.md alone
- ✅ Fresh Claude can resume worker work via SPAWN_CONTEXT.md + beads comments
- ✅ `kb context` finds all investigations regardless of which agent created them
- ✅ Abandoned agents have at least failure mode and reason in beads
- ✅ No artifact type has unbounded growth without explicit cleanup

---

## Self-Review Checklist

### Phase-Specific Checks

| Phase | Check | Status |
|-------|-------|--------|
| **Problem Framing** | Success criteria defined? | ✅ |
| **Exploration** | 2+ approaches compared? | ✅ (4 alternatives) |
| **Synthesis** | Clear recommendation with reasoning? | ✅ |
| **Externalization** | Investigation produced? Feature list reviewed? | ✅ |

### Self-Review

- [x] **Question clear** - Minimal artifact set for amnesia-resilient orchestration
- [x] **Criteria defined** - Zero-context-loss resumption, discoverable knowledge, clear lifecycle
- [x] **2+ approaches explored** - 4 alternatives considered
- [x] **Trade-offs documented** - Manual promotion, persistent workspaces, discipline required
- [x] **Recommendation clear** - 5 essential + 3 supplementary artifacts
- [x] **Principle cited** - Session Amnesia (foundational), Progressive Disclosure, Evidence Hierarchy
- [x] **Investigation produced** - This file

**Self-Review Status:** PASSED

---

## References

**Files Examined (Input Investigations):**
- `.kb/investigations/2025-12-21-inv-workspace-lifecycle-when-workspaces-created.md` - Workspace creation/cleanup
- `.kb/investigations/2025-12-21-inv-knowledge-promotion-paths.md` - Promotion mechanisms and usage
- `.kb/investigations/2025-12-21-inv-orchestrator-session-boundaries.md` - Session types and handoffs
- `.kb/investigations/2025-12-21-inv-beads-kb-workspace-relationships-how.md` - Three-layer architecture
- `.kb/investigations/2025-12-21-inv-multi-agent-synthesis-when-multiple.md` - Conflict handling
- `.kb/investigations/2025-12-21-inv-failure-mode-artifacts.md` - Failure capture gap

**Supporting Context:**
- `~/.kb/principles.md` - Session Amnesia, Progressive Disclosure, Evidence Hierarchy
- `.orch/templates/SYNTHESIS.md` - D.E.K.N. template structure
- `.kb/investigations/2025-12-21-design-deep-pattern-analysis-orchestration-artifacts.md` - Deep pattern analysis

**Related Artifacts:**
- **Epic:** `orch-go-4kwt` - Amnesia-Resilient Artifact Architecture
- **Parent investigation:** Deep Pattern Analysis (.kb/investigations/2025-12-21-design-deep-pattern-analysis-orchestration-artifacts.md)

---

## Investigation History

**2025-12-21 22:01:** Investigation started
- Initial question: What is the minimal artifact set for amnesia-resilient orchestration?
- Context: Synthesis of 6 parallel investigations from orch-go-4kwt epic

**2025-12-21 22:30:** Completed exploration phase
- Read all 6 input investigations
- Identified three-tier temporal model as organizing principle
- Found failure artifacts as main gap

**2025-12-21 23:00:** Synthesis completed
- Defined 5 essential + 3 supplementary artifacts
- Documented relationships and lifecycle rules
- Created implementation recommendations

**2025-12-21 23:15:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Minimal artifact taxonomy with clear lifecycle rules; FAILURE_REPORT.md and SESSION_HANDOFF.md are main additions needed
