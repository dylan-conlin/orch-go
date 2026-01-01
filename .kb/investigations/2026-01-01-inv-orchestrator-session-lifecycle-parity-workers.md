<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Workers get full lifecycle machinery (SPAWN_CONTEXT.md, phase tracking, beads comments, required SYNTHESIS.md, orch complete verification gates); orchestrators have partial parity (orch session start/status/end exists) but lack verification gates, required artifacts, and friction capture at session-end.

**Evidence:** Analyzed pkg/spawn/context.go (800+ line spawn template), pkg/verify/check.go (verification gates), cmd/orch/session.go (orchestrator session commands). Workers have 5 verification checks at completion; orchestrators have 2 optional prompts (Knowledge/Next).

**Knowledge:** The gap is NOT in ceremony (orchestrators have session commands) but in gates and artifacts. Workers cannot complete without Phase: Complete + SYNTHESIS.md; orchestrators can `orch session end --no-handoff` with no required output. This matches "Gate Over Remind" principle - without gates, compliance drops under cognitive load.

**Next:** Create epic with children implementing: (1) ORCH_SESSION_CONTEXT.md template auto-created at session start, (2) required artifacts at session end (friction + knowledge capture), (3) verification gates in `orch session end` that match worker parity.

---

# Investigation: Orchestrator Session Lifecycle Parity Workers

**Question:** What specific lifecycle machinery do workers have that orchestrators lack, and what should parity look like given that orchestrators are fundamentally different (composite, multi-spawn, human-collaborative) from workers (atomic, single-spawn)?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** og-work-orchestrator-session-lifecycle-01jan
**Phase:** Complete
**Next Step:** Create epic with implementation children
**Status:** Complete

**Supersedes:** N/A (synthesizes prior work, does not replace)

---

## Findings

### Finding 1: Workers Have 5-Layer Lifecycle Machinery

**Evidence:** Workers receive comprehensive lifecycle support through:

1. **SPAWN_CONTEXT.md (800+ lines)** - Contains task, authority levels, deliverables, beads ID, skill content, kb context, project ecosystem, behavioral patterns warning, tier, scope estimation
2. **Workspace Artifacts** - `.orch/workspace/{name}/` with SPAWN_CONTEXT.md, .tier file, .spawn_time file, SYNTHESIS.md (required for full tier)
3. **Phase Tracking** - `bd comment "Phase: X"` at transitions (Planning → Implementing → Testing → Complete)
4. **Verification Gates** - `orch complete` checks:
   - Phase: Complete reported via beads
   - SYNTHESIS.md exists (for full tier)
   - Investigation file filled (if applicable)
   - Git commits present
   - Tests pass (if applicable)
5. **Failure Report** - FAILURE_REPORT.md template required when abandoning, gates on fill-out before respawning

**Source:** 
- `pkg/spawn/context.go:20-280` - SPAWN_CONTEXT.md template
- `pkg/verify/check.go:145-300` - Verification result structure and synthesis parsing
- `pkg/spawn/context.go:1135-1237` - Failure report gate

**Significance:** Workers cannot proceed without gates. The machinery is enforced, not advisory.

---

### Finding 2: Orchestrators Have Partial Machinery - Sessions Without Gates

**Evidence:** Orchestrator session machinery exists but is advisory:

1. **Session Start** (`orch session start "goal"`) - Creates session ID, sets focus, records start time
2. **Session Status** (`orch session status`) - Shows duration, spawns, goal, active agents
3. **Session End** (`orch session end`) - Prompts for Knowledge and Next, generates handoff, warns about in-progress agents

**Missing:**
- No ORCH_SESSION_CONTEXT.md auto-created (orchestrator gets context via OpenCode plugin injection)
- No required artifacts (SYNTHESIS.md equivalent for orchestrator)
- No verification gates (can use `--no-handoff` to skip everything)
- No friction capture in prompts
- No gate on session reflection (Session Reflection in orchestrator skill is "soft gate with explicit skip")

**Source:** 
- `cmd/orch/session.go:64-200` - Session start implementation
- `cmd/orch/session.go:302-446` - Session end (2 prompts: Knowledge, Next)

**Significance:** The orchestrator skill says "run at least one of orch learn, kn command, or explicit skip" as a gate, but `orch session end` doesn't enforce this - it's a reminder in documentation, not a CLI gate.

---

### Finding 3: The "Gate Over Remind" Principle Applies

**Evidence:** From `~/.kb/principles.md:146-163`:

> **Gate Over Remind:** Enforce knowledge capture through gates (cannot proceed without), not reminders (easily ignored).
> 
> **Why:** Reminders fail under cognitive load. When deep in a complex problem, "remember to update kn" gets crowded out. Gates make capture unavoidable.

The orchestrator Session Reflection (lines 1121-1152 in orchestrator skill) describes three checkpoints:
1. Friction Audit - What was harder than it should have been?
2. Gap Capture - What knowledge should have been surfaced but wasn't?
3. System Reaction Check - Does this session suggest system improvements?

But `orch session end` only prompts for Knowledge and Next - not Friction or System Reaction. And these prompts can be skipped by pressing Enter.

**Source:**
- `~/.kb/principles.md:146-163` - Gate Over Remind principle
- `~/.claude/skills/meta/orchestrator/SKILL.md:1121-1152` - Session Reflection (in skill, not CLI)

**Significance:** There's a gap between what the skill describes and what the CLI enforces. Workers have CLI-enforced gates; orchestrators have skill-described reminders.

---

### Finding 4: Prior Investigations Already Designed Components

**Evidence:** Several investigations have designed pieces of this parity:

1. **2025-12-29-inv-unified-session-model-design.md** - Defined "focus block" as orchestrator session unit, designed `orch session start/end/resume`, implemented MVP
2. **2026-01-01-inv-session-end-reflection-ritual.md** - Designed Friction section for SYNTHESIS.md, proposed Friction + System Reaction prompts for `orch session end`
3. **2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md** - Identified that action outcomes are ephemeral and untracked, proposed action logging
4. **2025-12-21-inv-orchestrator-session-boundaries.md** - Found three session types with different boundary patterns; orchestrator boundaries are state-detected, not protocol-driven

**Source:** Read all four investigations in full during context gathering.

**Significance:** The design work is done. What's missing is implementation that adds gates, not just ceremonies.

---

### Finding 5: Fundamental Difference - Composite vs Atomic

**Evidence:** Workers are atomic (spawn → task → complete). Orchestrators are composite (focus set → multiple spawns → synthesis → handoff).

| Aspect | Worker | Orchestrator |
|--------|--------|--------------|
| **Unit** | Single spawn | Focus block (hours/days) |
| **Tracking** | Beads issue per spawn | Session state + spawns |
| **Artifacts** | SYNTHESIS.md | SESSION_HANDOFF.md |
| **Gates** | `orch complete` verifies | None - advisory prompts |
| **Context** | SPAWN_CONTEXT.md | OpenCode plugin injection |
| **Friction capture** | Leave it Better (kn commands) | Session Reflection (skill guidance) |

**Source:** 
- `pkg/spawn/context.go` - Worker context template
- `cmd/orch/session.go` - Orchestrator session commands
- Orchestrator skill "Session Reflection" section

**Significance:** Parity doesn't mean identical structure. It means shared primitives: explicit start/end, required artifacts, verification gates. The orchestrator version is more lightweight but should still have gates.

---

### Finding 6: What "Parity" Should Mean

**Evidence:** Based on the design-session skill's decision tree:

```
Can we list the specific tasks needed?
├── YES → Do we understand all the tasks well enough to implement?
│   ├── YES → Epic with children
│   └── NO → Investigation (to clarify unknowns)
└── NO → Investigation
```

The tasks are clear:
1. **ORCH_SESSION_CONTEXT.md** - Create template auto-generated at session start (parity with SPAWN_CONTEXT.md)
2. **Friction prompts** - Add Friction + System Reaction prompts to `orch session end` (per prior investigation)
3. **Verification gates** - Gate on at least one output (handoff saved, or explicit skip with --no-handoff + reason)
4. **Session workspace** - Create `~/.orch/session/{date}/` with SESSION_CONTEXT.md and SESSION_HANDOFF.md

This is an epic with clear children.

**Source:** Analysis of findings 1-5.

**Significance:** Output type is EPIC with children, not Investigation or Decision.

---

## Synthesis

**Key Insights:**

1. **Machinery exists, gates don't** - Orchestrator session commands exist (`orch session start/status/end`) but lack the verification gates that workers have. The skill describes reflection; the CLI doesn't enforce it.

2. **Prior designs are ready for implementation** - The 2026-01-01-inv-session-end-reflection-ritual investigation already designed the Friction section and prompts. The unified-session-model-design defined the architecture. What's needed is implementation.

3. **Gate Over Remind is the principle** - Workers have gates (`orch complete` verification). Orchestrators have reminders (skill documentation). Parity means adding gates, not just ceremonies.

4. **Parity means shared primitives, not identical structure** - Orchestrators are composite (multi-spawn); workers are atomic (single-spawn). But both should have: explicit start/end, required artifacts, verification gates.

**Answer to Investigation Question:**

**What workers have that orchestrators lack:**

| Worker Machinery | Orchestrator Equivalent | Gap |
|-----------------|------------------------|-----|
| SPAWN_CONTEXT.md (800+ lines) | OpenCode plugin injection | No equivalent template file |
| Workspace artifacts (.tier, .spawn_time) | Session state in ~/.orch/session.json | Different but functional |
| Phase tracking via beads | N/A (orchestrators don't have phases) | Not applicable (different model) |
| `orch complete` verification gates | `orch session end` prompts (optional) | **GAP: No gates, just prompts** |
| Required SYNTHESIS.md | Optional SESSION_HANDOFF.md | **GAP: No required artifact** |
| FAILURE_REPORT.md gate | N/A | **GAP: No failure capture** |
| Leave it Better (kn commands) | Session Reflection (skill guidance) | **GAP: Reminder, not gate** |

**What parity should look like:**

1. **ORCH_SESSION_CONTEXT.md** - Template created at `orch session start` with goal, constraints, focus issue, prior handoff context
2. **Required prompts** - Add Friction + System Reaction to existing Knowledge + Next prompts
3. **Verification gate** - `orch session end` cannot complete without either:
   - Handoff saved (all 4 prompts answered, or default values)
   - Explicit `--skip-reflection` with documented reason
4. **Session workspace** - `~/.orch/session/{date}/` with SESSION_CONTEXT.md and SESSION_HANDOFF.md

---

## Structured Uncertainty

**What's tested:**

- ✅ Worker SPAWN_CONTEXT.md is 800+ lines with comprehensive context (verified: read pkg/spawn/context.go)
- ✅ `orch session end` prompts for Knowledge and Next only (verified: read cmd/orch/session.go:366-395)
- ✅ `--no-handoff` flag skips all prompts (verified: read cmd/orch/session.go:350-354)
- ✅ Session Reflection has 3 checkpoints in skill, but skill != CLI enforcement (verified: read orchestrator skill)

**What's untested:**

- ⚠️ Whether orchestrators would comply with gates (behavior not tested)
- ⚠️ Whether ORCH_SESSION_CONTEXT.md adds value vs plugin injection (hypothesis)
- ⚠️ Whether friction capture at session-end is timely enough (may need progressive capture)

**What would change this:**

- If orchestrators already have high handoff compliance without gates, gates may be unnecessary overhead
- If OpenCode plugin injection is sufficient context, ORCH_SESSION_CONTEXT.md may be over-engineering
- If friction capture needs to happen during session (not just at end), the design needs adjustment

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach: Epic with Children

**Orchestrator Session Lifecycle Parity Epic** - Implement gates and artifacts to match worker lifecycle rigor, while respecting orchestrator's composite nature.

**Why this approach:**
- Prior investigations have already designed the components
- Clear scope: 4 discrete tasks that build on existing session commands
- Addresses the core gap (gates vs reminders) while respecting orchestrator differences

**Trade-offs accepted:**
- Adds ceremony to session end (acceptable: "Gate Over Remind" principle)
- Creates new artifact (SESSION_CONTEXT.md) that may duplicate plugin injection (acceptable: file-based is more discoverable)

**Implementation sequence:**
1. **Session Context Template** - Create ORCH_SESSION_CONTEXT.md at `orch session start`
2. **Enhanced Prompts** - Add Friction + System Reaction to `orch session end`
3. **Verification Gate** - Require either handoff saved or `--skip-reflection` with reason
4. **Session Workspace** - Formalize `~/.orch/session/{date}/` structure

### Alternative Approaches Considered

**Option B: Minimal - Just Add Gate Flag**
- **Pros:** Smallest change, least disruption
- **Cons:** Doesn't add friction capture, doesn't create discoverable context file
- **When to use instead:** If orchestrators are already highly compliant with handoffs

**Option C: Full Worker Parity - Add Beads Tracking**
- **Pros:** True parity with workers (beads issue per orchestrator session)
- **Cons:** Over-engineering - orchestrators are composite, not atomic; beads tracking doesn't fit
- **When to use instead:** Never - fundamentally different session model

**Rationale for recommendation:** Option A provides the gates that workers have while respecting orchestrator's composite nature. It's based on prior design work and addresses the core principle violation (reminders instead of gates).

---

### Implementation Details

**What to implement first:**

1. **Enhanced `orch session end` prompts** (lowest risk, highest value)
   - Add Friction prompt after Knowledge prompt
   - Add System Reaction prompt after Next prompt
   - Include all 4 in handoff markdown template

2. **Verification gate** (core parity improvement)
   - Require at least one of: handoff saved OR `--skip-reflection --reason "why"`
   - Default: gate on handoff saved
   - Escape hatch: explicit skip with documented reason

3. **Session workspace formalization** (structure improvement)
   - Create `~/.orch/session/{date}/SESSION_CONTEXT.md` at session start
   - Save `SESSION_HANDOFF.md` to same directory at session end

4. **ORCH_SESSION_CONTEXT.md template** (context parity)
   - Include: goal, constraints, prior handoff context, focus issue, active agents

**Things to watch out for:**

- ⚠️ Don't make prompts so verbose they create fatigue (keep brief like current)
- ⚠️ Gate shouldn't block emergency session ends (--force flag exists)
- ⚠️ SESSION_CONTEXT.md might duplicate OpenCode plugin injection (test if valuable)

**Success criteria:**

- ✅ `orch session end` prompts for Friction and System Reaction (in addition to Knowledge and Next)
- ✅ Cannot end session without handoff OR explicit skip with reason
- ✅ SESSION_HANDOFF.md includes Friction and System Reaction sections
- ✅ At least one friction item leading to system improvement (validates the flow)

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - Worker SPAWN_CONTEXT.md template (800+ lines)
- `pkg/verify/check.go` - Worker verification gates
- `cmd/orch/session.go` - Orchestrator session commands
- `~/.claude/skills/meta/orchestrator/SKILL.md` - Session Reflection section
- `~/.kb/principles.md` - Gate Over Remind principle

**Prior Investigations:**
- `.kb/investigations/2025-12-29-inv-unified-session-model-design.md` - Designed session commands
- `.kb/investigations/2026-01-01-inv-session-end-reflection-ritual.md` - Designed friction capture
- `.kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md` - Action tracking gap
- `.kb/investigations/2025-12-21-inv-orchestrator-session-boundaries.md` - Session boundary patterns

**Commands Run:**
```bash
# Get KB context
kb context "orchestrator session lifecycle parity workers"

# Check related issues
bd ready
bd list --labels triage:review
```

**Related Artifacts:**
- **Investigation:** This investigation provides design for orchestrator session parity epic
- **Epic:** orch-go-amfa (Unified Orchestrator Session Model) - prior work
- **Principle:** Gate Over Remind (`~/.kb/principles.md:146-163`)

---

## Investigation History

**2026-01-01 09:00:** Investigation started
- Initial question: What lifecycle machinery do workers have that orchestrators lack?
- Context: Task spawned from design-session skill for parity analysis

**2026-01-01 09:30:** Context gathering complete
- Read 4 prior investigations on orchestrator sessions
- Read pkg/spawn/context.go (worker template)
- Read cmd/orch/session.go (orchestrator session commands)
- Read orchestrator skill Session Reflection section
- Read principles.md for Gate Over Remind

**2026-01-01 10:00:** Design synthesis complete
- Identified core gap: gates vs reminders
- Determined output type: Epic with children (4 tasks)
- Aligned with Gate Over Remind principle

**2026-01-01 10:15:** Investigation completed
- Status: Complete
- Key outcome: Epic with 4 children: (1) Session Context Template, (2) Enhanced Prompts, (3) Verification Gate, (4) Session Workspace

---

## Self-Review

- [x] Real test performed (read actual code files, not "analyzed logic")
- [x] Conclusion from evidence (based on codebase and prior investigation analysis)
- [x] Question answered (what workers have, what orchestrators lack, what parity should look like)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled

**Self-Review Status:** PASSED
