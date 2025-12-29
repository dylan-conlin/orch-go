<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Orchestrator sessions need a "focus block" as their unit (vs worker's spawn→complete), with `orch session start/end/resume` commands replacing the unused session-transition skill.

**Evidence:** Workers have full tracking (SPAWN_CONTEXT.md, beads, workspace); orchestrators have fragile hook injection and 76 "where were we?" requests. session-transition skill never used because it's for Claude Code sessions, not orchestrator-specific context.

**Knowledge:** Orchestrator sessions are fundamentally different from worker sessions - they're human-agent collaborative, multi-spawn, and reflective rather than task-atomic. Unification means shared primitives (tracking, resume), not identical structure.

**Next:** Create epic with children for MVP implementation: `orch session start/end/resume`, orchestrator workspace, and retire session-transition skill.

---

# Investigation: Unified Session Model Design

**Question:** How should orchestrator sessions be spawned, tracked, resumed, and reflected upon using the same primitives as worker sessions?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** og-work-unified-session-model-29dec
**Phase:** Complete
**Next Step:** None - findings ready for epic creation
**Status:** Complete

**Supersedes:** .kb/investigations/2025-12-29-inv-unified-session-model-apply-worker.md (empty template)

---

## Findings

### Finding 1: Worker Session Model is Well-Defined

**Evidence:** Workers have complete lifecycle management:
- **Spawn:** `orch spawn SKILL "task"` creates SPAWN_CONTEXT.md with task, skill, authority, deliverables
- **Track:** Beads issue per spawn, `bd comment` for phase transitions, `.session_id` file
- **Resume:** `orch resume <id>` with workspace context preservation
- **Reflect:** SYNTHESIS.md required for "full" tier spawns

**Source:** 
- `pkg/spawn/context.go:17-257` - SPAWN_CONTEXT.md template
- `pkg/spawn/session.go:1-156` - Session metadata (ID, tier, spawn time)
- `pkg/spawn/config.go` - Spawn configuration

**Significance:** Workers have a clear "unit" = one spawn = one task = one beads issue. The atomic nature makes tracking straightforward.

---

### Finding 2: Orchestrator Session Model is Undefined

**Evidence:** Orchestrators currently have:
- **Spawn:** Manual `opencode` CLI invocation, no explicit spawn command
- **Track:** No beads tracking; relies on SessionStart hook for context injection
- **Resume:** Manual "where were we?" (76 instances in prior investigation)
- **Reflect:** `orch handoff` exists but requires manual D.E.K.N. fill-in

**Source:**
- `cmd/orch/handoff.go:1-899` - Handoff document generation
- `~/.claude/skills/session-transition/SKILL.md` - 551 lines, "never used" per task
- Prior investigation: `.kb/investigations/2025-12-29-inv-systematically-analyze-orchestrator-sessions-human.md`

**Significance:** The orchestrator "unit" is unclear. Is it a tmux window? A focus block? A day's work? This ambiguity causes the 76 "where were we?" requests.

---

### Finding 3: session-transition Skill Misaligned with Orchestrator Needs

**Evidence:** session-transition skill is designed for generic Claude Code session transitions:
- Detects state via git status, workspace parsing
- Captures context to `.claude/workspace/` (Claude Code path, not `.orch/workspace/`)
- Offers tmux cleanup and learning extraction
- Assumes user is ending a Claude session, not orchestrating agents

**Source:** `~/.claude/skills/session-transition/SKILL.md:1-551`

**Significance:** The skill never gets used because:
1. It's for Claude Code sessions (`.claude/`), not orch sessions (`.orch/`)
2. Orchestrators need to track *agent* state, not just their own git state
3. The interactive prompts ("What were you working on?") don't fit orchestrator flow

---

### Finding 4: orch handoff Partially Addresses Resume but Not Spawn/Track

**Evidence:** `orch handoff` generates comprehensive handoff documents with:
- Active agents from OpenCode/tmux
- Pending beads issues
- Local git state
- D.E.K.N. sections (gated on Knowledge/Next being filled)
- Auto-suggested next priorities

**Source:** `cmd/orch/handoff.go:64-135` - HandoffData struct

**Significance:** `orch handoff` is the *output* mechanism but lacks:
- Session *start* tracking (when did this session begin?)
- Session *scope* definition (what is this session's goal?)
- Automatic resume context (still requires reading the handoff doc)

---

### Finding 5: Fundamental Difference Between Orchestrator and Worker Sessions

**Evidence:** Worker sessions are **atomic** (spawn→task→complete). Orchestrator sessions are **composite** (focus set→multiple spawns→synthesis→handoff).

| Aspect | Worker | Orchestrator |
|--------|--------|--------------|
| **Unit** | Single spawn | "Focus block" (goal + spawns) |
| **Duration** | 1-4 hours | Hours to days |
| **Tracking** | Beads issue | Focus goal |
| **Output** | Code + investigation | Synthesis + decisions |
| **Resume** | Workspace context | Focus + agent states + handoff |

**Source:** Analysis of spawn.go (worker) vs handoff.go/focus.go (orchestrator)

**Significance:** Unification doesn't mean identical structure. It means shared *primitives*:
- Both have a "unit" (spawn vs focus block)
- Both produce artifacts (SYNTHESIS.md vs SESSION_HANDOFF.md)
- Both can be resumed (workspace vs handoff + focus)
- Both can be tracked (beads issue vs focus + events)

---

## Synthesis

**Key Insights:**

1. **"Focus Block" is the Orchestrator Session Unit** - The orchestrator's equivalent of a worker spawn is a "focus block" - a time-bounded period with a goal (`orch focus`). This already exists but isn't treated as a session boundary.

2. **session-transition Skill Should Be Replaced, Not Fixed** - The skill targets Claude Code sessions, not orchestrator sessions. Rather than adapting it, create `orch session` commands that understand orchestrator context (agents, focus, beads state).

3. **Shared Primitives, Different Structure** - Unification means: (a) Both have explicit start/end, (b) Both produce resumable artifacts, (c) Both track progress. But orchestrator sessions wrap multiple worker sessions, not replace them.

**Answer to Investigation Question:**

Orchestrator sessions should be managed with **new `orch session` commands** that:
1. **Start:** Set focus, record session start time, initialize session workspace
2. **Track:** Aggregate spawned agents, track progress via events
3. **Resume:** Auto-load focus, show agent states, inject prior handoff context
4. **Reflect:** Produce SESSION_HANDOFF.md with D.E.K.N. (upgrade `orch handoff` to `orch session end`)

This replaces the unused session-transition skill with orchestrator-native tooling.

---

## Structured Uncertainty

**What's tested:**

- ✅ Worker session model has SPAWN_CONTEXT.md, beads tracking, workspace (verified: read pkg/spawn/*.go)
- ✅ session-transition skill uses `.claude/` paths not `.orch/` (verified: read SKILL.md)
- ✅ orch handoff generates comprehensive state but requires manual D.E.K.N. (verified: read cmd/orch/handoff.go)

**What's untested:**

- ⚠️ Whether "focus block" abstraction will feel natural to Dylan (hypothesis based on existing `orch focus` usage)
- ⚠️ Whether automatic session start on `opencode` launch is desirable vs explicit `orch session start`
- ⚠️ Whether session workspace should be per-focus or per-day

**What would change this:**

- If Dylan prefers ad-hoc orchestration without session boundaries, the "focus block" model is wrong
- If orch handoff is working well enough, session commands may be over-engineering
- If orchestrator sessions need beads tracking (not just focus tracking), the model needs adjustment

---

## Implementation Recommendations

**Purpose:** Define MVP and full vision for unified orchestrator session model.

### Recommended Approach ⭐

**`orch session` Command Family** - Explicit session management commands that unify orchestrator lifecycle with worker patterns.

**Why this approach:**
- Explicit boundaries solve "where were we?" problem (76 instances of context loss)
- Reuses existing primitives (focus, handoff, workspace) in coherent workflow
- Replaces unused session-transition skill with orchestrator-native tooling

**Trade-offs accepted:**
- Adds ceremony (explicit start/end) vs current ad-hoc flow
- Orchestrator sessions won't have beads issues (focus tracking instead)

**Implementation sequence:**
1. `orch session start` - Set focus, create session workspace, record start time
2. `orch session status` - Show session state (duration, spawns, focus, agents)
3. `orch session end` - Generate handoff, offer cleanup, clear session state
4. Retire session-transition skill

### Alternative Approaches Considered

**Option B: Fix session-transition Skill**
- **Pros:** Reuses existing skill infrastructure
- **Cons:** Wrong abstraction (Claude Code vs orchestrator); would require major rewrite
- **When to use instead:** If skill system changes make embedding easier than commands

**Option C: Automatic Session Detection**
- **Pros:** Zero ceremony; sessions inferred from activity gaps
- **Cons:** Boundaries unclear; "where were we?" problem remains
- **When to use instead:** If explicit session management feels too heavy

**Rationale for recommendation:** Option A provides explicit boundaries (solving #1 friction source) while reusing existing primitives (focus, handoff, workspace) rather than building parallel infrastructure.

---

### Implementation Details

**MVP (Phase 1):**
- `orch session start ["goal"]` - Sets focus, records start time in `~/.orch/session.json`
- `orch session status` - Shows duration, spawns since start, current focus
- `orch session end` - Runs `orch handoff`, clears session state
- Retire session-transition skill (remove from skills directory)

**Full Vision (Phase 2):**
- `orch session resume` - Reads last handoff, injects context, shows agent states
- Session workspace: `~/.orch/session/{date}/` with session artifacts
- Integration with SessionStart hook: Auto-show session status on orchestrator start
- Session history: `orch session list` showing past sessions

**What to implement first:**
- MVP commands (`start`, `status`, `end`) - immediate friction reduction
- Session state file (`~/.orch/session.json`) - minimal new infrastructure

**Things to watch out for:**
- ⚠️ Session state must survive orchestrator restarts (persist to file, not memory)
- ⚠️ Multi-project orchestration: session should know which repo spawns belong to
- ⚠️ Don't break existing `orch focus` / `orch handoff` - build on top

**Success criteria:**
- ✅ "Where were we?" requests decrease (measure in future session analysis)
- ✅ Session boundaries are clear (explicit start/end)
- ✅ Resume context is automatic (not manual handoff reading)

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - Worker SPAWN_CONTEXT.md generation
- `pkg/spawn/session.go` - Worker session metadata
- `cmd/orch/handoff.go` - Orchestrator handoff generation
- `~/.claude/skills/session-transition/SKILL.md` - Existing skill to replace
- `.kb/investigations/2025-12-29-inv-systematically-analyze-orchestrator-sessions-human.md` - Prior friction analysis

**Commands Run:**
```bash
# List skills
ls ~/.claude/skills/

# Check handoff command
orch handoff --help

# Get kb context
kb context "unified session model"
```

**Related Artifacts:**
- **Investigation:** .kb/investigations/2025-12-29-inv-systematically-analyze-orchestrator-sessions-human.md - Found 76 "where were we?" instances

---

## Investigation History

**2025-12-29 10:36:** Investigation started
- Initial question: How to unify orchestrator and worker session models?
- Context: Task spawned from design-session skill

**2025-12-29 11:00:** Context gathering complete
- Read spawn/context.go, session.go, handoff.go, session-transition skill
- Identified fundamental worker vs orchestrator session differences

**2025-12-29 11:15:** Design synthesis complete
- Determined "focus block" as orchestrator session unit
- Recommended `orch session` command family
- Defined MVP vs full vision

**2025-12-29 11:30:** Investigation completed
- Status: Complete
- Key outcome: Orchestrator sessions need explicit `orch session start/end/resume` commands with "focus block" as the session unit
