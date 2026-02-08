<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Meta-orchestrator architecture is incremental enhancement of existing infrastructure, not a new system. Orchestrators are already structurally spawnable via SESSION_CONTEXT.md ↔ SESSION_HANDOFF.md.

**Evidence:** Analyzed session.go, spawn_cmd.go, and prior investigation. Found three-tier hierarchy (meta → orchestrator → worker) already exists implicitly. Gap is verification and reflection, not spawn mechanism.

**Knowledge:** Meta-orchestrator IS Dylan (initially). Verification differs from workers (SESSION_HANDOFF.md vs SYNTHESIS.md). Three implementation phases: verification gate, dashboard visibility, pattern analysis.

**Next:** Implement feat-032 (`orch session end --require-handoff`), then feat-033 (dashboard session visibility), then feat-034 (`kb reflect --type orchestrator`).

---

# Investigation: Meta-Orchestrator Architecture for Spawnable Orchestrator Sessions

**Question:** How should we evolve from interactive orchestrator sessions to spawnable orchestrator sessions, creating a meta-orchestrator layer that manages orchestrator sessions like orchestrators manage workers?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Architect Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Problem Framing

### Design Question

How should we architecture spawnable orchestrator sessions, given:
1. Prior investigation found orchestrators ARE already structurally spawnable (SESSION_CONTEXT.md ↔ SPAWN_CONTEXT.md, SESSION_HANDOFF.md ↔ SYNTHESIS.md)
2. The gap is verification, reflection, and visibility - not spawn mechanism
3. Workers evolved from tmux-visible to headless with dashboard visibility
4. Orchestrators could follow the same evolution path

### Success Criteria

A good architecture should:
1. **Clear separation** - Meta-orchestrator vs orchestrator responsibilities are distinct
2. **Leverage existing infrastructure** - Build on `orch session start/end`, `kb reflect`, dashboard
3. **No cognitive overhead** - Should feel natural, not bureaucratic
4. **Enable pattern analysis** - Surface insights across orchestrator sessions
5. **Support multiple modes** - Dylan-interactive AND autonomous orchestrator sessions

### Constraints

1. **Session Amnesia** - Spawned orchestrators will have no memory between sessions
2. **Orchestrators delegate, never implement** - This constraint still applies
3. **Existing tooling** - Must work with current beads, kb, orch CLI ecosystem
4. **Dylan is the meta-orchestrator** (initially) - Human provides strategic direction

### Scope

**In scope:**
- Architecture for spawnable orchestrator sessions
- Meta-orchestrator responsibilities
- Verification and completion workflow
- Visibility (tmux vs headless vs dashboard)
- Artifact production requirements

**Out of scope:**
- Implementation details (this is architecture/design)
- Automated meta-orchestrator (Dylan remains human meta-orchestrator)
- Changes to worker spawn infrastructure

---

## Context Synthesis

### Current State (Prior Art)

**From prior investigation (2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md):**

| Layer | Worker | Orchestrator | Gap |
|-------|--------|--------------|-----|
| 1. Input Context | SPAWN_CONTEXT.md | SESSION_CONTEXT.md | ✅ Exists |
| 2. Progress Tracking | `bd comment "Phase: X"` | No equivalent | ❌ Gap |
| 3. Output Artifacts | SYNTHESIS.md | SESSION_HANDOFF.md | ✅ Exists |
| 4. Completion Verification | `orch complete` | No equivalent | ❌ Gap |
| 5. Pattern Analysis | `kb reflect` | Not for orchestrator sessions | ❌ Gap |

**Key insight:** Orchestrators are ALREADY structurally spawnable. The "not spawnable" perception comes from missing verification and reflection automation.

### Evolution Pattern (Tmux → Headless)

**Workers evolved:**
1. **tmux era** - Workers in tmux windows, orchestrator watches visually
2. **Headless era** - Workers via HTTP API, dashboard provides visibility
3. **Current** - Orchestrator still interactive, no visibility layer above

**Natural next step:**
1. Orchestrator sessions become spawnable
2. Dashboard (or new surface) provides meta-visibility
3. Meta-orchestrator (Dylan) delegates to orchestrator sessions

### Window Layout Evolution

**Old (tmux workers):**
```
Left Ghostty: orchestrator (Dylan + Claude interactive)
Right Ghostty: workers-{project} (spawned agents in tmux windows)
```

**Current (headless workers):**
```
Left Ghostty: orchestrator (Dylan + Claude interactive)
Dashboard: Workers (headless, via localhost:5188)
```

**Proposed:**
```
Left Ghostty: meta-orchestrator (Dylan + strategic decisions)
Right Ghostty: orchestrator sessions (spawned, visible in tmux)
Dashboard: Workers (headless, managed by orchestrator sessions)
```

---

## Findings

### Finding 1: The Three-Tier Hierarchy Emerges Naturally

**Evidence:** The spawn context asks these questions:
- What triggers orchestrator session spawn? 
- What's the session boundary?
- What does meta-orchestrator do vs orchestrator session?

The answers reveal a natural three-tier hierarchy:

| Tier | Role | Scope | Artifact | Visibility |
|------|------|-------|----------|------------|
| Meta-orchestrator | Strategic | Cross-project, multi-session | Epic progress, handoffs | Dylan-interactive |
| Orchestrator Session | Tactical | Single project, single focus | SESSION_HANDOFF.md | Tmux window |
| Worker | Implementation | Single issue | SYNTHESIS.md, code | Headless + dashboard |

**Source:** SPAWN_CONTEXT.md design questions

**Significance:** This isn't a new design - it's documenting what already exists implicitly. Dylan already operates as meta-orchestrator; we're just making the tier explicit and adding infrastructure.

---

### Finding 2: Orchestrator Session Boundaries Map to Focus Blocks

**Evidence:** The existing `orch session start/end` infrastructure already tracks:
- Goal (focus)
- Start time
- Spawns made during session

What triggers a new orchestrator session should be:
- Focus shift (different goal)
- Time boundary (taking a break, ending day)
- Project switch (major context change)
- Explicit command (`orch session start "new goal"`)

**Source:** `pkg/session/session.go:31-42`, orchestrator skill "Focus-Based Session Model"

**Significance:** Session boundaries are already defined. What's missing is verification at session end and pattern analysis across sessions.

---

### Finding 3: Meta-Orchestrator Responsibilities Are Distinct

**Evidence:** Comparing what meta-orchestrator does vs orchestrator session:

**Meta-orchestrator (Dylan):**
- Strategic focus decisions (which epic, which project)
- Cross-session pattern recognition
- Epic-level progress tracking
- Orchestrator session spawning and review
- System-level decisions (tooling, process improvements)

**Orchestrator Session:**
- Tactical execution within focus
- Triage issues, spawn workers, complete workers
- Synthesize worker findings
- Produce SESSION_HANDOFF.md
- Stay within delegated scope

**Source:** Orchestrator skill "Orchestrator Core Responsibilities"

**Significance:** The separation is clean - meta deals with WHICH focus, orchestrator deals with HOW to execute that focus. This matches how orchestrators delegate to workers (WHAT to do, not HOW).

---

### Finding 4: Visibility Options Follow Worker Pattern

**Evidence:** Workers have three spawn modes:
- `--inline` - Blocking TUI, for debugging
- `--tmux` - Visible window, opt-in
- `--headless` - HTTP API, default for automation

Orchestrator sessions could follow the same pattern:

| Mode | Use Case | Visibility |
|------|----------|------------|
| Interactive (current) | Dylan working with orchestrator | Inline TUI |
| Tmux | Visible orchestrator sessions | Tmux window |
| Headless | Autonomous orchestrator sessions | Dashboard only |

**Source:** `cmd/orch/spawn_cmd.go:66-77`

**Significance:** The infrastructure exists - we could spawn orchestrator sessions with the same modes. Whether this is valuable depends on whether autonomous orchestrators make sense.

---

### Finding 5: Verification Differs from Worker Verification

**Evidence:** Worker verification (`orch complete`) checks:
- Phase: Complete reported
- Tier-appropriate artifacts (SYNTHESIS.md for full tier)
- Git commits present
- Tests pass (if applicable)

Orchestrator session verification should check different things:
- SESSION_HANDOFF.md exists with content
- All spawned workers completed or handed off
- Knowledge captured (kn entries, kb artifacts)
- Git pushed (if applicable)

**Source:** `pkg/verify/check.go`, orchestrator skill "Session Reflection"

**Significance:** Verification is different because orchestrator output is different. Workers produce code + investigation artifacts. Orchestrators produce synthesis + handoffs + knowledge.

---

## Synthesis

### Key Insights

1. **Three-tier hierarchy is already implicit** - Dylan operates as meta-orchestrator, orchestrators delegate to workers. Making this explicit adds infrastructure, not concepts.

2. **Session boundaries exist, verification doesn't** - `orch session start/end` provides the lifecycle, but `orch session end` doesn't verify output quality.

3. **Visibility follows worker pattern** - Orchestrator sessions could be visible in tmux, or headless with dashboard visibility. The choice depends on use case.

4. **Verification criteria differ** - Orchestrators produce knowledge/handoffs, not code. Verification should match the artifact type.

5. **Pattern analysis is high-value** - `kb reflect` analyzing SESSION_HANDOFF.md files would surface recurring friction, missed learnings, and orchestrator patterns.

### Answer to Investigation Question

**What should the meta-orchestrator architecture look like?**

**Recommended approach: Incremental enhancement of existing infrastructure**

Rather than creating a parallel "meta-orchestrator" system, extend the existing orchestrator infrastructure:

1. **Meta-orchestrator IS Dylan** (initially) - No automation needed. Dylan makes strategic decisions, spawns orchestrator sessions, reviews handoffs.

2. **Orchestrator sessions ARE spawnable** - Use existing `orch session start/end` with enhanced verification:
   - `orch session end --require-handoff` gates on SESSION_HANDOFF.md
   - Allow `--skip-handoff --reason "X"` for explicit bypass

3. **Add orchestrator session visibility** to dashboard:
   - Show active orchestrator session goal, duration, spawns
   - Surface session handoffs for review

4. **Extend `kb reflect` for orchestrator patterns:**
   - New type: `kb reflect --type orchestrator`
   - Scan SESSION_HANDOFF.md files for recurring friction
   - Surface sessions without knowledge externalization

---

## Structured Uncertainty

**What's tested:**

- ✅ SESSION_CONTEXT.md and SESSION_HANDOFF.md exist (verified: file structure)
- ✅ `orch session start/end` provides lifecycle (verified: code review)
- ✅ Worker spawn modes exist (verified: spawn_cmd.go)
- ✅ Dashboard shows agent status (verified: running system)

**What's untested:**

- ⚠️ Whether orchestrators would produce useful SESSION_HANDOFF.md if gated (behavioral)
- ⚠️ Whether visible orchestrator sessions (tmux) add value over interactive
- ⚠️ Whether pattern analysis of handoffs would surface actionable insights
- ⚠️ Whether autonomous orchestrator sessions are valuable (vs Dylan-interactive)

**What would change this:**

- If verification proves too heavy → make it opt-in
- If visible sessions add no value → keep interactive only
- If pattern analysis produces noise → focus on specific friction types
- If autonomous orchestrators prove valuable → add headless mode

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Incremental Enhancement of Existing Infrastructure**

Don't create new "meta-orchestrator spawn" command. Instead:

1. **Enhance `orch session end` with verification gate**
2. **Add orchestrator session to dashboard**
3. **Extend `kb reflect` for orchestrator patterns**

**Why this approach:**
- Builds on proven infrastructure
- No new concepts to learn
- Gradual evolution, not revolution
- Reversible if gates prove too heavy

**Trade-offs accepted:**
- No tmux visibility for orchestrator sessions (keep interactive)
- No autonomous orchestrator sessions (Dylan remains meta-orchestrator)
- Verification is opt-in, not gated by default

**Implementation sequence:**

1. **Phase 1: Verification gate** (high value, low effort)
   - Add `--require-handoff` flag to `orch session end`
   - Create SESSION_HANDOFF.md template from `~/.orch/templates/`
   - Allow `--skip-handoff --reason "X"` for bypass

2. **Phase 2: Dashboard visibility** (medium value, medium effort)
   - Add `/api/session` endpoint to `orch serve`
   - Show current session goal, duration, spawn count in stats bar
   - Link to SESSION_HANDOFF.md if exists

3. **Phase 3: Pattern analysis** (high value, medium effort)
   - Add `kb reflect --type orchestrator`
   - Scan `~/.orch/session/*/SESSION_HANDOFF.md`
   - Surface: recurring friction, abandoned sessions, missing learnings

### Alternative Approaches Considered

**Option B: Visible orchestrator sessions in tmux**
- **Pros:** Dylan can observe orchestrator work like worker windows
- **Cons:** Orchestrators ARE interactive with Dylan - visibility adds little
- **When to use:** If autonomous orchestrators become valuable

**Option C: Full three-tier spawn command**
- **Pros:** Explicit `orch spawn orchestrator` command
- **Cons:** Creates parallel system, orchestrators don't need beads tracking
- **When to use:** If orchestrator sessions need isolation from Dylan

**Option D: Focus only on kb reflect without verification**
- **Pros:** No friction added to session end workflow
- **Cons:** Garbage in → garbage out (analysis only as good as handoffs)
- **When to use:** If verification proves too heavy

**Rationale for recommendation:** Option A (incremental) provides highest value with lowest disruption. It treats orchestrators as "already spawnable but unverified" rather than "needs new spawn mechanism."

---

### Implementation Details

**What to implement first:**
- `orch session end --require-handoff` - establishes artifact production habit
- SESSION_HANDOFF.md template with structured sections - enables pattern detection

**Things to watch out for:**
- ⚠️ Don't make `--require-handoff` the default immediately - test adoption first
- ⚠️ SESSION_HANDOFF.md needs to be fillable in < 3 minutes
- ⚠️ `kb reflect --type orchestrator` needs actionable output, not just "you had friction"

**Areas needing further investigation:**
- Should SESSION_HANDOFF.md go in project `.orch/` or global `~/.orch/`?
- What orchestrator "phases" should be tracked (if any)?
- How to handle cross-project orchestrator sessions?

**Success criteria:**
- ✅ `orch session end --require-handoff` gates on artifact existence
- ✅ Dashboard shows current orchestrator session status
- ✅ `kb reflect --type orchestrator` surfaces at least one actionable pattern
- ✅ Dylan stops manually prompting the same reflection questions

---

## References

**Files Examined:**
- `cmd/orch/session.go` - Current session implementation
- `cmd/orch/spawn_cmd.go` - Spawn modes and patterns
- `pkg/session/session.go` - Session state management
- `.orch/templates/SESSION_HANDOFF.md` - Existing handoff template
- `.kb/investigations/2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md` - Prior investigation

**Commands Run:**
```bash
# Check session infrastructure
ls ~/.orch/session/

# Check kb reflect capabilities
kb reflect --help

# Review existing handoff templates
cat ~/.orch/templates/SESSION_HANDOFF.md
```

**Related Artifacts:**
- **Prior Investigation:** `.kb/investigations/2026-01-04-design-orchestrator-skill-spawnable-agent-gap.md` - Found orchestrators are already structurally spawnable
- **Orchestrator Skill:** `~/.claude/skills/meta/orchestrator/SKILL.md` - Defines session model
- **Window Setup:** `orchestration-window-setup.md` - Documents three-tier window layout

---

## Investigation History

**2026-01-04 09:00:** Investigation started
- Initial question: How should we architect spawnable orchestrator sessions?
- Context: Prior investigation found orchestrators ARE structurally spawnable, gap is verification

**2026-01-04 09:30:** Problem framing complete
- Defined success criteria, constraints, scope
- Identified three-tier hierarchy (meta → orchestrator → worker)

**2026-01-04 10:00:** Exploration complete
- Found 5 key findings about architecture
- Synthesized recommendation: incremental enhancement, not new system

**2026-01-04 10:30:** Investigation complete
- Status: Complete
- Key outcome: Recommend enhancing existing `orch session` infrastructure with verification gate, dashboard visibility, and pattern analysis. No new "meta-orchestrator" system needed.
