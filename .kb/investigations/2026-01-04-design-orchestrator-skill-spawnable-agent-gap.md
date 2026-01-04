<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The orchestrator IS already "spawnable" via `orch session start/end` - the constraint that prevented spawning was false. The gap is verification and pattern analysis, not the spawning mechanism itself.

**Evidence:** Compared worker spawn machinery (SPAWN_CONTEXT.md, orch complete, kb reflect) against orchestrator infrastructure (SESSION_CONTEXT.md, session.go, session-transition skill). Both have input context files and output artifacts. Orchestrators lack only verification and reflection automation.

**Knowledge:** Workers and orchestrators have symmetric structures: SPAWN_CONTEXT.md ↔ SESSION_CONTEXT.md, SYNTHESIS.md ↔ SESSION_HANDOFF.md. The "not spawnable" perception comes from missing `orch session complete` verification and `kb reflect` analysis for orchestrator sessions - both solvable gaps.

**Next:** Don't create new "spawnable orchestrator" mechanism. Instead: (1) add `orch session end --require-handoff` verification, (2) extend `kb reflect` to analyze SESSION_HANDOFF.md artifacts.

---

# Investigation: Orchestrator Skill as Spawnable Agent Gap

**Question:** What would it mean for orchestrator sessions to be "spawnable"? What infrastructure exists vs what's missing?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Workers have a five-layer spawn protocol

**Evidence:** The worker spawn machinery consists of five distinct layers:

| Layer | Component | Purpose |
|-------|-----------|---------|
| 1. Input Context | SPAWN_CONTEXT.md | Task, authority, deliverables, skill guidance |
| 2. Progress Tracking | `bd comment "Phase: X"` | Observable state transitions |
| 3. Output Artifacts | SYNTHESIS.md, investigation files | Structured knowledge externalization |
| 4. Completion Verification | `orch complete` | Validates outputs, tier requirements |
| 5. Pattern Analysis | `kb reflect` | Surfaces synthesis opportunities across sessions |

**Source:** 
- `pkg/spawn/context.go:28-272` (SPAWN_CONTEXT template)
- `cmd/orch/complete_cmd.go` (verification logic)
- `pkg/daemon/reflect.go` (kb reflect implementation)

**Significance:** This is the full definition of "spawnable." Any entity lacking these layers has reduced observability, verifiability, and knowledge accumulation.

---

### Finding 2: Orchestrators already have three of five layers

**Evidence:** Existing orchestrator session infrastructure provides:

| Layer | Worker | Orchestrator | Status |
|-------|--------|--------------|--------|
| 1. Input Context | SPAWN_CONTEXT.md | SESSION_CONTEXT.md | ✅ Exists |
| 2. Progress Tracking | `bd comment` | No equivalent | ❌ Gap |
| 3. Output Artifacts | SYNTHESIS.md | SESSION_HANDOFF.md | ✅ Exists |
| 4. Completion Verification | `orch complete` | No equivalent | ❌ Gap |
| 5. Pattern Analysis | `kb reflect` | Not for orchestrator sessions | ❌ Gap |

- `orch session start` creates SESSION_CONTEXT.md at `~/.orch/session/{date}/`
- `orch session end` logs session.ended event but doesn't create SESSION_HANDOFF.md automatically
- Session reflection prompts exist in orchestrator skill but aren't enforced

**Source:**
- `cmd/orch/session.go:80-115` (session start)
- `cmd/orch/session.go:300-356` (session end)
- `~/.orch/session/2026-01-01/SESSION_CONTEXT.md` (example artifact)

**Significance:** The orchestrator already has the input/output artifact pattern (layers 1 and 3). The gaps are progress tracking (layer 2), verification (layer 4), and pattern analysis (layer 5) - not the fundamental spawn structure.

---

### Finding 3: The "interactive" constraint was a false premise

**Evidence:** The SPAWN_CONTEXT.md states the questioned constraint:

> "The orchestrator is 'interactive' (human-facing). But is that actually preventing it from being spawnable? Workers are also interactive with their task - they just have structure around the interaction."

Analysis confirms this intuition:
- Workers ARE interactive - they read/write files, make decisions, ask questions
- The structure (SPAWN_CONTEXT → work → SYNTHESIS) doesn't prevent interactivity
- Orchestrators already have structure (SESSION_CONTEXT → work → SESSION_HANDOFF)
- The difference is enforcement/verification, not capability

**Source:** SPAWN_CONTEXT.md lines 18-21

**Significance:** The question "can orchestrators be spawnable?" is answered - they already are structurally spawnable. The real question is "should orchestrator outputs be verified like worker outputs?"

---

### Finding 4: The value proposition differs from workers

**Evidence:** The prior investigation (2025-12-26-inv-session-end-workflow-orchestrators.md) identified three orchestrator-specific checkpoints:

1. **Friction audit:** "What was harder than it should have been?"
2. **Gap capture:** "What knowledge should have been surfaced but wasn't?"
3. **System reaction check:** "Does this session suggest system improvements?"

These differ from worker checkpoints (SYNTHESIS.md focuses on Delta/Evidence/Knowledge/Next for the specific task). Orchestrator reflection is about the *system*, not the *task*.

**Source:** 
- `.kb/investigations/2025-12-26-inv-session-end-workflow-orchestrators.md:163-179`
- `.kb/investigations/2025-12-21-inv-orchestrator-session-boundaries.md:40-55`

**Significance:** Worker verification checks "did the task succeed?" Orchestrator verification should check "did the session improve the system?" This is a different artifact structure.

---

### Finding 5: kb reflect has the right architecture but wrong scope

**Evidence:** `kb reflect` already analyzes multiple artifact types:

```
Reflection types:
- synthesis: Investigations needing consolidation (3+ on same topic)
- promote: kn entries worth promoting to kb decisions
- stale: Decisions with no citations
- drift: Constraints contradicted by code
- open: Investigations with unimplemented recommendations
- refine: kn entries that refine principles
- skill-candidate: kn entry clusters warranting skill updates
```

SESSION_HANDOFF.md is not in this list, but the architecture (scan artifacts → detect patterns → surface recommendations) applies directly.

**Source:** `kb reflect --help` output, `pkg/daemon/reflect.go`

**Significance:** Adding orchestrator session analysis to kb reflect is an incremental extension, not a new system. The pattern detection machinery exists.

---

## Synthesis

**Key Insights:**

1. **Orchestrators are already structurally spawnable** - SESSION_CONTEXT.md and SESSION_HANDOFF.md parallel SPAWN_CONTEXT.md and SYNTHESIS.md. The spawn machinery exists; only verification and reflection are missing.

2. **The gap is observability, not capability** - Workers have visible phase transitions (`Phase: Planning` → `Phase: Complete`). Orchestrators work invisibly until session end. Adding phase tracking for orchestrators would require rethinking what "orchestrator phases" mean (not task phases, but focus/triage/synthesis phases).

3. **Verification requires different criteria** - Worker verification asks "did SYNTHESIS.md get created?" Orchestrator verification should ask "did the session capture learnings?" This maps to the three checkpoints: friction/gaps/system-reaction.

4. **Pattern analysis is the highest-value gap** - kb reflect analyzing SESSION_HANDOFF.md files would surface patterns like "orchestrator keeps hitting same friction" or "orchestrator never runs orch learn." This is more valuable than verification (which can be gamed).

**Answer to Investigation Question:**

**What would a spawned orchestrator session look like?**

It would look almost exactly like today's `orch session start/end` flow, with:
- SESSION_CONTEXT.md created at start (already happens)
- SESSION_HANDOFF.md created at end (happens manually, should be enforced)
- `orch session end --require-handoff` flag to gate completion
- kb reflect analyzing handoff artifacts for patterns

**What outputs should be verified at session end?**

1. SESSION_HANDOFF.md exists and has non-placeholder content
2. At least one of: `orch learn` run, `kn` command run, or explicit skip with reason
3. Git push completed (or explicit defer)

**How would kb reflect analyze orchestrator sessions?**

New reflection type: `kb reflect --type orchestrator-patterns`
- Scans `~/.orch/session/*/SESSION_HANDOFF.md`
- Detects recurring friction (same complaint in multiple sessions)
- Detects missing learnings (sessions without kn externalization)
- Surfaces sessions that never ended (started but no handoff)

**What's the equivalent of SPAWN_CONTEXT.md for orchestrator sessions?**

SESSION_CONTEXT.md already exists and serves this role. It could be enhanced with:
- Prior handoff context (currently done manually)
- Focus constraints
- Active agent summary at session start

**How does this relate to orch session start/end which already exists?**

`orch session start/end` IS the orchestrator spawn mechanism. The enhancement path is:
1. Make SESSION_HANDOFF.md creation automatic (or gated)
2. Add verification to `orch session end`
3. Add orchestrator session analysis to kb reflect

---

## Structured Uncertainty

**What's tested:**

- ✅ SESSION_CONTEXT.md is created by `orch session start` (verified: file exists at `~/.orch/session/2026-01-01/SESSION_CONTEXT.md`)
- ✅ `orch session end` logs events but doesn't verify artifacts (verified: read `cmd/orch/session.go:300-356`)
- ✅ kb reflect has extensible architecture for new reflection types (verified: read help output)
- ✅ Workers have five-layer spawn protocol (verified: traced through SPAWN_CONTEXT.md → orch complete → kb reflect)

**What's untested:**

- ⚠️ Whether orchestrators would actually fill in SESSION_HANDOFF.md if gated (behavioral assumption)
- ⚠️ Whether pattern analysis of handoffs would surface useful insights (value hypothesis)
- ⚠️ What "orchestrator phases" would look like for progress tracking (design question)

**What would change this:**

- If orchestrators consistently skip reflection even when gated → verification is theater, focus on pattern analysis instead
- If SESSION_HANDOFF.md content is too varied to pattern-match → need structured template with detectable fields
- If `orch session end` becomes too heavy → make verification opt-in, keep pattern analysis always-on

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Incremental enhancement of existing orch session** - Don't create new "spawnable orchestrator" mechanism. Enhance the existing `orch session` commands to add verification and analysis.

**Why this approach:**
- Builds on existing infrastructure (SESSION_CONTEXT.md, SESSION_HANDOFF.md)
- Aligns with the finding that orchestrators ARE already structurally spawnable
- Avoids creating parallel systems that fragment orchestrator attention

**Trade-offs accepted:**
- Doesn't add phase tracking for orchestrators (out of scope - different design question)
- Relies on SESSION_HANDOFF.md template consistency for pattern analysis
- Verification is optional, not gated by default

**Implementation sequence:**

1. **Add `orch session end --require-handoff` flag**
   - Gate session end on SESSION_HANDOFF.md existence
   - Create from template if missing, prompt to fill
   - Allow `--skip-handoff --reason "X"` for explicit bypass

2. **Add orchestrator session reflection to kb reflect**
   - New type: `kb reflect --type orchestrator`
   - Scans `~/.orch/session/*/SESSION_HANDOFF.md`
   - Detects: recurring friction, missing learnings, abandoned sessions

3. **Enhance SESSION_HANDOFF.md template with structured fields**
   - Friction section (detectable for patterns)
   - Learnings section (kn commands run)
   - System reaction section (skill/CLAUDE.md/plugin suggestions)

### Alternative Approaches Considered

**Option B: Create orch orchestrator spawn command**
- **Pros:** Cleaner separation, explicit "spawning an orchestrator"
- **Cons:** Creates parallel system, orchestrators already don't use `orch spawn` for themselves
- **When to use instead:** If orchestrator sessions need beads issue tracking (unclear value)

**Option C: Add orchestrator phase tracking**
- **Pros:** Full symmetry with workers, observable progress
- **Cons:** What ARE orchestrator phases? Triage/spawn/synthesis? Unclear, needs separate design
- **When to use instead:** If invisible orchestrator work becomes a problem

**Option D: Focus only on kb reflect without verification**
- **Pros:** Lowest friction, always-on analysis
- **Cons:** Garbage in → garbage out (analysis only as good as handoff content)
- **When to use instead:** If verification proves too heavy for orchestrator flow

**Rationale for recommendation:** Option A (incremental enhancement) provides the highest-value gaps (verification + analysis) with lowest disruption. It treats orchestrators as "already spawnable but unverified" rather than "needs to become spawnable."

---

### Implementation Details

**What to implement first:**
- `orch session end --require-handoff` - establishes the artifact production habit
- SESSION_HANDOFF.md template with structured sections - enables pattern detection

**Things to watch out for:**
- ⚠️ Don't make `--require-handoff` the default immediately - orchestrators will rebel
- ⚠️ SESSION_HANDOFF.md needs to be lightweight enough to fill in 2 minutes
- ⚠️ kb reflect orchestrator analysis needs clear actionable output (not just "you had friction")

**Areas needing further investigation:**
- What should orchestrator "phase tracking" look like? (separate design session)
- Should SESSION_HANDOFF.md go in project `.orch/` or global `~/.orch/`? (currently global)
- How to handle cross-project orchestrator sessions? (orchestration home question)

**Success criteria:**
- ✅ `orch session end --require-handoff` gates on artifact existence
- ✅ `kb reflect --type orchestrator` surfaces at least one actionable pattern
- ✅ Dylan stops manually prompting the same reflection questions every session

---

## References

**Files Examined:**
- `cmd/orch/session.go` - Session start/end implementation
- `pkg/session/session.go` - Session state management
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template (for comparison)
- `.kb/investigations/2025-12-21-inv-orchestrator-session-boundaries.md` - Prior boundary investigation
- `.kb/investigations/2025-12-26-inv-session-end-workflow-orchestrators.md` - Prior session-end investigation
- `~/.orch/session/2026-01-01/SESSION_CONTEXT.md` - Example session context
- `~/.orch/session/2025-12-29/SESSION_HANDOFF.md` - Example session handoff

**Commands Run:**
```bash
# Check kb reflect capabilities
kb reflect --help

# Verify session infrastructure
ls -la ~/.orch/session/

# Check session status command
orch session status
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-21-inv-orchestrator-session-boundaries.md` - Foundational boundaries analysis
- **Investigation:** `.kb/investigations/2025-12-26-inv-session-end-workflow-orchestrators.md` - Session reflection checkpoints
- **Prior Decision:** "Session boundaries have three distinct patterns" (from kn context)

---

## Investigation History

**2026-01-04 08:50:** Investigation started
- Initial question: What would it mean for orchestrator sessions to be "spawnable"?
- Context: Gap identified - orchestrator skill runs interactively without spawn infrastructure

**2026-01-04 09:00:** Context gathering complete
- Found: SESSION_CONTEXT.md and SESSION_HANDOFF.md already exist
- Found: `orch session start/end` provides session lifecycle
- Found: Missing verification and pattern analysis, not spawn structure

**2026-01-04 09:15:** Key insight emerged
- The constraint "orchestrators are interactive so can't be spawned" is false
- Orchestrators ARE structurally spawnable - they just lack verification
- The question shifts from "how to spawn orchestrators" to "how to verify orchestrator sessions"

**2026-01-04 09:30:** Investigation completed
- Status: Complete
- Key outcome: Recommend incremental enhancement of `orch session` with verification and kb reflect analysis, not new spawn mechanism
