<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Three distinct context templates exist for worker/orchestrator/meta-orchestrator roles with proper routing in pkg/spawn/context.go, and beads guidance in SPAWN_CONTEXT.md is complementary (not duplicative) with bd prime hook.

**Evidence:** Analyzed pkg/spawn/context.go:486-497 routing logic, SpawnContextTemplate (304 lines), OrchestratorContextTemplate (127 lines), and MetaOrchestratorContextTemplate (187 lines).

**Knowledge:** Workers get beads tracking, orchestrators get SESSION_HANDOFF.md requirement and WAIT behavior, meta-orchestrators get interactive framing with no required artifacts - role separation is clean.

**Next:** Update epic with Probe 3 findings; investigate remaining duplication (orchestrator skill's embedded beads guidance) in separate probe.

**Promote to Decision:** recommend-no - This documents existing state, no architectural decision needed.

---

# Investigation: Audit Spawn Context MD Content

**Question:** What does SPAWN_CONTEXT.md contain for orchestrator vs worker vs meta-orchestrator roles?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** og-inv-audit-spawn-context-16jan-1218
**Phase:** Complete
**Next Step:** None - investigation complete
**Status:** Complete

**Extracted-From:** Epic `.orch/epics/context-injection-architecture.md` - Probe 3

---

## Findings

### Finding 1: Three Distinct Context Templates with Proper Routing

**Evidence:** The spawn package contains three separate templates:
- `SpawnContextTemplate` (context.go:40-304) → SPAWN_CONTEXT.md
- `OrchestratorContextTemplate` (orchestrator_context.go:19-127) → ORCHESTRATOR_CONTEXT.md
- `MetaOrchestratorContextTemplate` (meta_orchestrator_context.go:21-187) → META_ORCHESTRATOR_CONTEXT.md

Routing logic in `WriteContext()` (context.go:486-497):
```go
func WriteContext(cfg *Config) error {
    if cfg.IsMetaOrchestrator {
        return WriteMetaOrchestratorContext(cfg)
    }
    if cfg.IsOrchestrator {
        return WriteOrchestratorContext(cfg)
    }
    // ... worker context generation
}
```

**Source:** pkg/spawn/context.go:486-497, pkg/spawn/orchestrator_context.go, pkg/spawn/meta_orchestrator_context.go

**Significance:** Role detection is working correctly. Meta-orchestrator check happens first (more specific), then orchestrator, then default to worker. This answers Discovery Question C from the epic.

---

### Finding 2: Role-Specific Content Matrix

**Evidence:** Full content analysis of all three templates:

| Content Section | Worker | Orchestrator | Meta-Orchestrator |
|-----------------|--------|--------------|-------------------|
| **Context File Name** | SPAWN_CONTEXT.md | ORCHESTRATOR_CONTEXT.md | META_ORCHESTRATOR_CONTEXT.md |
| **Task Framing** | "TASK: {task}" | "Session Goal: {goal}" | "Role: managing orchestrator sessions" |
| **First Actions** | 3 actions: bd comment, read context, begin planning | 5 actions: read skill, orch status, bd ready, fill handoff | 3 actions: orch status, orch review, ask Dylan |
| **Beads Tracking** | ✅ Full bd comment instructions | ❌ None | ❌ None |
| **Beads ID** | ✅ Included ({{.BeadsID}}) | ❌ Not written | ❌ Not written |
| **Completion Artifact** | SYNTHESIS.md | SESSION_HANDOFF.md | None required |
| **Completion Signal** | `bd comment "Phase: Complete"` + `/exit` | Write handoff + WAIT | Stay interactive |
| **Git Push** | ❌ NEVER (workers commit locally) | Not mentioned | Not mentioned |
| **Authority Section** | Implementation details only | Spawn workers, manage beads issues | Spawn orchestrators, review handoffs |
| **Escalation Triggers** | Architectural decisions, scope unclear | Strategic direction, scope larger than expected | Strategic direction, resource constraints |
| **Server Context** | Conditional (IncludeServers) | Conditional | Conditional |
| **KB Context** | ✅ If provided | ✅ If provided | ✅ If provided |
| **Registered Projects** | ❌ Not included | ✅ For cross-project work | ✅ For cross-project work |
| **Prior Handoff** | ❌ | ❌ | ✅ Automatic via findPriorMetaOrchestratorHandoff() |
| **Skill Content** | ✅ Embedded | ✅ Embedded | ✅ Embedded |
| **Tier System** | light/full (affects SYNTHESIS.md) | "orchestrator" tier | "orchestrator" tier |

**Source:** Template analysis across all three files

**Significance:** Role boundaries are cleanly separated. Workers focus on task execution with beads tracking. Orchestrators coordinate workers and produce session handoffs. Meta-orchestrators stay interactive and manage orchestrator sessions.

---

### Finding 3: Beads Guidance is Complementary, Not Duplicative (between SPAWN_CONTEXT and bd prime)

**Evidence:** Compared `bd prime` output (~3KB) with SPAWN_CONTEXT.md beads section:

**bd prime hook content:**
- Session close protocol (git status, add, commit, push, bd sync)
- Core rules (when to use beads vs TodoWrite)
- Essential commands reference (bd ready, create, close, update, etc.)
- Dependencies & Blocking commands
- Sync & Collaboration workflows

**SPAWN_CONTEXT.md beads section (lines 206-236):**
- Phase reporting via `bd comment {{.BeadsID}}`
- When to comment (phase transitions, milestones, blockers)
- Why beads comments matter (searchable history)
- Worker constraint: NEVER run `bd close`

**Key difference:** bd prime is a general command reference for any session. SPAWN_CONTEXT.md beads section is spawned-worker-specific progress tracking protocol tied to a specific beads ID.

**Source:**
- `bd prime` output analysis
- pkg/spawn/context.go:206-236 (BEADS PROGRESS TRACKING section)

**Significance:** The claimed "duplication" between bd prime and SPAWN_CONTEXT.md is actually complementary content serving different purposes. However, note that Probe 1 also identified beads guidance in the orchestrator skill - that potential duplication wasn't tested here.

---

### Finding 4: Workspace Artifacts and Metadata by Role

**Evidence:** Files created in workspace directory by role:

| File | Worker | Orchestrator | Meta-Orchestrator |
|------|--------|--------------|-------------------|
| SPAWN_CONTEXT.md | ✅ | ❌ | ❌ |
| ORCHESTRATOR_CONTEXT.md | ❌ | ✅ | ❌ |
| META_ORCHESTRATOR_CONTEXT.md | ❌ | ❌ | ✅ |
| SESSION_HANDOFF.md | ❌ | ✅ (pre-filled) | ❌ |
| SESSION_HANDOFF.template.md | ❌ | ✅ (if exists) | ❌ |
| .beads_id | ✅ | ❌ | ❌ |
| .spawn_mode | ✅ | ❌ | ❌ |
| .spawn_time | ✅ | ✅ | ✅ |
| .tier | ✅ | ✅ (value: orchestrator) | ✅ (value: orchestrator) |
| .orchestrator | ❌ | ✅ | ❌ |
| .meta-orchestrator | ❌ | ❌ | ✅ |
| .workspace_name | ❌ | ✅ | ✅ |
| screenshots/ | ✅ | ✅ | ✅ |

**Source:**
- pkg/spawn/context.go:509-552 (WriteContext)
- pkg/spawn/orchestrator_context.go:191-257 (WriteOrchestratorContext)
- pkg/spawn/meta_orchestrator_context.go:250-301 (WriteMetaOrchestratorContext)

**Significance:** Workers have beads integration (.beads_id). Orchestrators use workspace name for lookup instead. All roles have spawn time and tier for verification. Marker files enable programmatic role detection.

---

### Finding 5: NoTrack Mode Strips Beads Instructions from Skill Content

**Evidence:** When `NoTrack=true` (ad-hoc spawns), the `StripBeadsInstructions()` function (context.go:315-406) removes:
- Code blocks containing `bd comment` or `bd close` commands
- "Report via Beads" sections
- Lines containing `<beads-id>` placeholders
- Completion criteria mentioning beads reporting

This prevents confusing workers with beads commands that won't work.

**Source:** pkg/spawn/context.go:315-406, 447-451

**Significance:** Ad-hoc spawns (--no-track) receive cleaned skill content. This is a thoughtful design that prevents errors from agents trying to use beads commands without a beads ID.

---

## Synthesis

**Key Insights:**

1. **Clean Role Separation** - Each role has its own context file, completion protocol, and authority boundaries. Workers execute with beads tracking, orchestrators coordinate and handoff, meta-orchestrators stay interactive.

2. **Beads is Worker-Only** - Only workers receive beads ID and tracking instructions. Orchestrators manage sessions (not issues), meta-orchestrators manage orchestrators (not issues or sessions).

3. **Progressive Documentation Encouraged** - Orchestrator SESSION_HANDOFF.md is pre-created with metadata to encourage filling as work progresses, not at the end.

4. **Cross-Project Support** - Orchestrators and meta-orchestrators receive RegisteredProjects context for cross-project work. Workers don't need this (they work in one project).

5. **NoTrack is Well-Handled** - Ad-hoc spawns get cleaned skill content without beads instructions, preventing confusion.

**Answer to Investigation Question:**

The three roles receive fundamentally different context:

- **Workers** receive SPAWN_CONTEXT.md with task focus, full beads tracking (bd comment for phases), SYNTHESIS.md requirement (full tier), and "never git push" constraint. They report Phase: Complete and call /exit.

- **Orchestrators** receive ORCHESTRATOR_CONTEXT.md with session goal focus, no beads tracking (they manage sessions not issues), SESSION_HANDOFF.md requirement (pre-filled), registered projects for cross-project work, and WAIT behavior (level above completes them).

- **Meta-Orchestrators** receive META_ORCHESTRATOR_CONTEXT.md with interactive framing, no beads tracking, no required artifacts, prior handoff context injection, registered projects, and explicit level constraints (never drop to orchestrator/worker level).

---

## Structured Uncertainty

**What's tested:**

- ✅ Three templates exist with distinct content (verified: read all three files)
- ✅ Routing logic prioritizes meta-orchestrator > orchestrator > worker (verified: read WriteContext)
- ✅ bd prime and SPAWN_CONTEXT.md beads sections are complementary (verified: compared outputs)
- ✅ NoTrack mode strips beads instructions from skill content (verified: read StripBeadsInstructions)

**What's untested:**

- ⚠️ Orchestrator skill's embedded beads guidance (mentioned in Probe 1, not examined here)
- ⚠️ Actual token counts for each template type (not measured)
- ⚠️ Whether hooks also fire for orchestrator/meta-orchestrator spawns (may add more context)

**What would change this:**

- If hooks add beads content for orchestrator spawns, there would be context waste
- If skill content has significant beads overlap with templates, there's duplication
- If token counts exceed budgets, templates need trimming

---

## Implementation Recommendations

**Purpose:** Not applicable - this is a discovery investigation, not a design investigation.

### Findings to Feed Back to Epic

1. **Discovery Question C (ANSWERED):** SPAWN_CONTEXT.md content varies correctly by role via three separate templates with proper routing.

2. **Beads Duplication Clarification:** bd prime + SPAWN_CONTEXT.md are complementary, not duplicative. However, orchestrator skill's embedded beads guidance (from Probe 1) still needs investigation.

3. **New Finding:** NoTrack mode has thoughtful handling that strips beads instructions from skill content.

### Recommended Next Probes

1. **Probe 4 (Usage Analysis):** Which injected content is actually referenced in sessions? May reveal unused sections.

2. **Probe 5 (Token Measurement):** Measure actual token counts for each template type to establish baselines.

3. **Probe 6 (Hook Firing for Orchestrators):** Do SessionStart hooks also fire for orchestrator/meta-orchestrator spawns? If so, what content overlaps?

---

## References

**Files Examined:**
- pkg/spawn/context.go - Worker template and routing logic
- pkg/spawn/orchestrator_context.go - Orchestrator template
- pkg/spawn/meta_orchestrator_context.go - Meta-orchestrator template
- pkg/spawn/config.go - Config struct and tier defaults

**Commands Run:**
```bash
# Check bd prime output size and content
bd prime 2>/dev/null | wc -c  # Result: 2961 bytes
bd prime 2>/dev/null | head -60
```

**Related Artifacts:**
- **Epic:** .orch/epics/context-injection-architecture.md - This is Probe 3
- **Investigation:** .kb/investigations/2026-01-16-inv-audit-sessionstart-hooks-claude-code.md - Probe 1 (hook audit)

---

## Investigation History

**2026-01-16 14:15:** Investigation started
- Initial question: What does SPAWN_CONTEXT.md contain for each role?
- Context: Epic Probe 3 - need role-specific content matrix

**2026-01-16 14:30:** Core templates analyzed
- Found three distinct templates with proper routing
- Documented full content matrix

**2026-01-16 14:45:** Beads duplication analyzed
- Compared bd prime with SPAWN_CONTEXT.md beads section
- Found complementary (not duplicative) content

**2026-01-16 14:50:** Investigation completed
- Status: Complete
- Key outcome: Clean role separation exists; bd prime + SPAWN_CONTEXT.md are complementary
