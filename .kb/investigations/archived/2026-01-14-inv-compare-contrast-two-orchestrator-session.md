<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Interactive and spawned orchestrator sessions represent two fundamentally different orchestration paradigms: TEMPORAL (human continuity across breaks) vs HIERARCHICAL (agent delegation to autonomous orchestrators), with distinct state management, completion protocols, handoff mechanisms, and context models.

**Evidence:** Read cmd/orch/session.go (session.json state, orch session start/end commands, self-directed completion), pkg/spawn/orchestrator_context.go (workspace-based state, ORCHESTRATOR_CONTEXT.md generation, external completion protocol), pkg/session/session.go (checkpoint thresholds differ: 2h/3h/4h for agents vs 4h/6h/8h for orchestrators).

**Knowledge:** These architectures are COMPLEMENTARY by design, each optimized for different orchestration needs - interactive sessions for human-driven work with natural breaks, spawned orchestrators for autonomous multi-goal execution. Mixing them causes confusion (spawned orchestrators running `orch session end`, interactive sessions trying to wait for external completion).

**Next:** Update orchestrator-session-lifecycle model with enable/constrain pattern, add technical comparison table to spawned-orchestrator-pattern.md guide, consider tracking this architecture in .kb/models/ for queryable understanding.

**Promote to Decision:** recommend-no (this deepens existing documentation, not new architectural choice)

---

# Investigation: Compare and Contrast Two Orchestrator Session Architectures

**Question:** How do the two orchestrator session architectures (interactive via `orch session start/end` vs spawned via `orch spawn orchestrator`) differ in terms of lifecycle, handoff mechanisms, context management, state tracking, and completion protocols? What does each enable and constrain?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** og-inv-compare-contrast-two-14jan-be43
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Extracted-From:** N/A
**Supersedes:** Deepens `.kb/investigations/2026-01-13-inv-analyze-orchestrator-session-management-architecture.md` (doesn't supersede - complementary)
**Superseded-By:** N/A

---

## Findings

### Finding 1: Fundamentally Different State Management Models

**Evidence:**

**Interactive Sessions (Temporal Model):**
- State stored globally in `~/.orch/session.json`
- Single active session at a time (new session replaces old)
- State includes: goal, start time, window name, spawn records
- No workspace artifacts required during session
- State is EPHEMERAL until `orch session end` creates handoff

```go
// pkg/session/session.go:93-110
type Session struct {
    Goal       string       `json:"goal"`
    StartedAt  time.Time    `json:"started_at"`
    WindowName string       `json:"window_name,omitempty"`
    Spawns     []SpawnRecord `json:"spawns"`
}
```

**Spawned Orchestrators (Hierarchical Model):**
- State stored in workspace: `.orch/workspace/{name}/`
- Multiple concurrent orchestrators supported
- State includes: ORCHESTRATOR_CONTEXT.md, SESSION_HANDOFF.md, .tier file, .orchestrator marker
- Workspace artifacts exist from spawn time
- State is PERSISTENT from creation

```go
// pkg/spawn/orchestrator_context.go:240-256
// Creates: ORCHESTRATOR_CONTEXT.md, .orchestrator marker, .tier,
// .workspace_name, pre-filled SESSION_HANDOFF.md
```

**Source:**
- `pkg/session/session.go:93-110` - Session struct definition
- `pkg/spawn/orchestrator_context.go:191-257` - WriteOrchestratorContext function
- `cmd/orch/session.go:68-143` - Session start command

**Significance:** These represent fundamentally different state models - global singleton (interactive) vs instance-per-workspace (spawned). This explains why you can have many spawned orchestrators but only one interactive session. The models are complementary by design.

---

### Finding 2: Opposite Completion Protocols

**Evidence:**

**Interactive Sessions (Self-Directed Completion):**
- Orchestrator runs `orch session end` themselves
- Creates SESSION_HANDOFF.md at end time
- Updates `latest` symlink for resume discovery
- Session.json cleared after end
- No external verification required

```go
// cmd/orch/session.go:447-542 - session end command
// Creates {project}/.orch/session/{timestamp}/SESSION_HANDOFF.md
// Updates symlink: latest -> {timestamp}
// Clears session.json
```

**Spawned Orchestrators (External Completion):**
- Writes SESSION_HANDOFF.md and WAITS
- MUST NOT run `/exit` or `orch session end`
- Level above (meta-orchestrator) runs `orch complete {workspace}`
- Completion verifies SESSION_HANDOFF.md exists
- Workspace status updated to "completed"

```go
// pkg/spawn/orchestrator_context.go:82-88
// "**Do NOT use `/exit` or `orch session end`** - spawned orchestrators
// wait for the level above to complete them."
```

**Source:**
- `pkg/spawn/orchestrator_context.go:73-88` - Completion protocol in template
- `cmd/orch/session.go:447-542` - Session end implementation
- `pkg/verify/check.go` - VerifyCompletionWithTier for orchestrator tier

**Significance:** The protocols are OPPOSITE - interactive self-completes, spawned waits for external completion. Mixing them breaks the hierarchy: a spawned orchestrator running `orch session end` would bypass meta-orchestrator verification. An interactive session waiting for external completion would block forever.

---

### Finding 3: Different Handoff Templates Serve Different Purposes

**Evidence:**

**Interactive Session Handoff (Reflective, End-of-Session):**
- Created AT END via `orch session end`
- Template focused on resume context
- Sections: Summary, What Was Accomplished, Active Work, Pending Work, Recommendations, Context
- Purpose: Enable next session to resume from where human left off

```go
// cmd/orch/session.go:666-752 - createSessionHandoffDirectory
// Creates SESSION_HANDOFF.md with reflective template
// Human fills at end based on memory of session
```

**Spawned Orchestrator Handoff (Progressive, Throughout Session):**
- Pre-created AT SPAWN with metadata
- Template encourages filling AS YOU WORK
- Sections: TLDR, Spawns table, Evidence, Knowledge, Friction, Focus Progress, Next, Unexplored Questions, Session Metadata
- Purpose: Signal completion to level above with comprehensive work record

```go
// pkg/spawn/orchestrator_context.go:358-522 - PreFilledSessionHandoffTemplate
// Pre-fills: Orchestrator name, Focus, Duration start, Outcome placeholder
// Agent fills progressively during work
```

**Template Size Comparison:**
- Interactive: ~40 lines, 6 sections
- Spawned: ~165 lines, 12 sections

**Source:**
- `pkg/spawn/orchestrator_context.go:358-522` - PreFilledSessionHandoffTemplate
- `cmd/orch/session.go:666-752` - Interactive handoff creation

**Significance:** The templates reflect different documentation philosophies:
- Interactive: Human memory-based reflection at session end
- Spawned: Agent-based progressive capture throughout work
The spawned template is more comprehensive because agents can't recall - they must document as they go.

---

### Finding 4: Different Checkpoint Thresholds Reflect Different Context Degradation Patterns

**Evidence:**

The system uses different checkpoint thresholds based on session type:

```go
// pkg/session/session.go:67-85
func DefaultAgentThresholds() CheckpointThresholds {
    return CheckpointThresholds{
        Warning: 2 * time.Hour,  // Agents: 2h
        Strong:  3 * time.Hour,  // Agents: 3h
        Max:     4 * time.Hour,  // Agents: 4h
    }
}

func DefaultOrchestratorThresholds() CheckpointThresholds {
    return CheckpointThresholds{
        Warning: 4 * time.Hour,  // Orchestrators: 4h
        Strong:  6 * time.Hour,  // Orchestrators: 6h
        Max:     8 * time.Hour,  // Orchestrators: 8h
    }
}
```

**Why the difference:**
- Agents accumulate implementation context (code, debugging state) which degrades quickly
- Orchestrators coordinate work (spawn, complete, synthesize) which doesn't accumulate as much context
- Orchestrators delegate implementation work, so their context is higher-level and persists longer

**Source:**
- `pkg/session/session.go:28-85` - Checkpoint threshold definitions with rationale

**Significance:** The system explicitly models that orchestrator sessions can run TWICE as long before context exhaustion becomes a concern. This affects both interactive sessions (when to run `orch session end`) and spawned orchestrators (when to surface warnings to meta-orchestrator).

---

### Finding 5: Context Injection Differs Between Architectures

**Evidence:**

**Interactive Sessions (Auto-Injection via Hooks):**
- SessionStart hook discovers latest handoff
- Injects into NEW Claude session automatically
- Path: `{project}/.orch/session/latest/SESSION_HANDOFF.md`
- Hook implementation: `~/.claude/hooks/session-start.sh`

```go
// cmd/orch/session.go:336-392 - findLatestHandoff
// Walks up tree to find project, checks for latest symlink
// Returns handoff content for injection
```

**Spawned Orchestrators (Embedded Context):**
- Full ORCHESTRATOR_CONTEXT.md created at spawn
- Skill content embedded directly (300-2000+ lines)
- KB context queried and embedded
- No runtime discovery needed - everything pre-loaded

```go
// pkg/spawn/orchestrator_context.go:144-188 - GenerateOrchestratorContext
// Embeds: SessionGoal, SkillContent, KBContext, ServerContext, RegisteredProjects
```

**Source:**
- `cmd/orch/session.go:336-392` - findLatestHandoff for resume
- `pkg/spawn/orchestrator_context.go:144-188` - GenerateOrchestratorContext
- `.kb/guides/session-resume-protocol.md` - Resume protocol documentation

**Significance:** Interactive sessions rely on DISCOVERY at runtime (hook finds handoff). Spawned orchestrators get EMBEDDED context at spawn time. This explains why spawned orchestrators have richer initial context but interactive sessions have more flexibility.

---

### Finding 6: Beads Integration Differs Significantly

**Evidence:**

**Interactive Sessions:**
- Session.json tracks spawns but NOT as beads issues
- Spawn records stored locally: `[]SpawnRecord` with beads_id reference
- Session itself is NOT a beads issue
- Session registry (`~/.orch/sessions.json`) tracks active sessions

**Spawned Orchestrators:**
- Workspace does NOT have `.beads_id` file (explicit design decision)
- Can be spawned WITH `--issue` flag for tracking epic progress
- Tracked via session registry, not beads
- Comments in code: "Orchestrators do NOT write .beads_id - they don't use beads tracking"

```go
// pkg/spawn/orchestrator_context.go:253-254
// Note: Orchestrators do NOT write .beads_id - they don't use beads tracking
// SESSION_HANDOFF.md is the completion signal, not Phase: Complete
```

**Source:**
- `pkg/spawn/orchestrator_context.go:253-254` - Explicit beads exclusion
- `pkg/session/session.go:113-129` - SpawnRecord includes beads_id for spawned workers
- `.kb/models/orchestrator-session-lifecycle.md:79-98` - Session registry vs beads explanation

**Significance:** Both architectures explicitly exclude orchestrators from beads tracking because:
- Orchestrators manage sessions (conversations), not tasks (work items)
- Beads tracks "what needs doing"; session registry tracks "who's managing work"
- This is a deliberate architectural choice, not an oversight

---

## Synthesis

**Key Insights:**

1. **Two Complementary Paradigms** - Interactive sessions solve TEMPORAL orchestration (human continuity across breaks), spawned orchestrators solve HIERARCHICAL orchestration (delegate to autonomous agents). Neither can replace the other because they serve fundamentally different needs.

2. **State Models Enable Different Capabilities** - Global singleton (interactive) enables one human to resume context across breaks. Instance-per-workspace (spawned) enables multiple concurrent orchestrators working on different goals. The constraint of "one interactive session" is a feature, not a bug - it models human attention.

3. **Completion Protocols Reflect Agency Models** - Self-directed completion (interactive) respects human agency - Dylan decides when to stop. External completion (spawned) enables hierarchical oversight - meta-orchestrator verifies work before closing. Mixing these would break the agency model.

4. **Handoff Templates Optimize for Documentation Capabilities** - Reflective templates (interactive) work because humans can remember. Progressive templates (spawned) compensate for agent amnesia. Both produce SESSION_HANDOFF.md but with different assumptions about the producer.

5. **Checkpoint Thresholds Model Context Degradation** - The 2x longer thresholds for orchestrators (4h/6h/8h vs 2h/3h/4h) reflect that coordination context persists better than implementation context. This is evidence-based from observing 5h+ sessions with quality degradation.

**Answer to Investigation Question:**

**How do they differ?**

| Aspect | Interactive (`orch session start/end`) | Spawned (`orch spawn orchestrator`) |
|--------|----------------------------------------|-------------------------------------|
| **State Location** | Global singleton (`~/.orch/session.json`) | Per-workspace (`.orch/workspace/{name}/`) |
| **Concurrency** | One at a time | Multiple concurrent |
| **Completion** | Self-directed (`orch session end`) | External (`orch complete` by level above) |
| **Handoff Creation** | At session end (reflective) | At spawn time (progressive) |
| **Context Injection** | Runtime discovery via hooks | Embedded at spawn time |
| **Beads Integration** | Tracks spawns, not itself | Neither tracks spawns nor itself |
| **Checkpoint Thresholds** | 4h/6h/8h (orchestrator) | 4h/6h/8h (orchestrator) |
| **Primary User** | Human (Dylan) | Meta-orchestrator (Dylan or another agent) |

**What does each enable?**

| Architecture | Enables | Because |
|--------------|---------|---------|
| Interactive | Human continuity across breaks | Session state persists, handoff auto-injected on resume |
| Interactive | Natural work sessions | Self-completion matches human agency model |
| Interactive | Goal refinement through conversation | Goal can evolve during session, not fixed at start |
| Spawned | Concurrent multi-goal orchestration | Multiple workspaces, each with distinct goal |
| Spawned | Hierarchical delegation | Meta-orchestrator can spawn, observe, complete |
| Spawned | Autonomous overnight processing | Spawned agent works without human oversight |

**What does each constrain?**

| Architecture | Constrains | Because |
|--------------|------------|---------|
| Interactive | Single active session | Global singleton state model |
| Interactive | Requires human to end session | Self-completion doesn't happen automatically |
| Interactive | Context dependent on hooks working | Discovery-based injection can fail |
| Spawned | Can't self-complete | External completion protocol |
| Spawned | Fixed goal at spawn time | ORCHESTRATOR_CONTEXT.md is immutable |
| Spawned | Requires meta-orchestrator oversight | Someone must run `orch complete` |

---

## Structured Uncertainty

**What's tested:**

- ✅ State locations differ (verified: read session.go, orchestrator_context.go)
- ✅ Completion protocols are opposite (verified: read templates and commands)
- ✅ Handoff templates differ significantly (verified: compared line counts and sections)
- ✅ Checkpoint thresholds are 2x longer for orchestrators (verified: read session.go:67-85)
- ✅ Beads integration excluded for orchestrators (verified: read comment at orchestrator_context.go:253)

**What's untested:**

- ⚠️ Whether context injection hooks work reliably across all environments (not tested in this investigation)
- ⚠️ Whether 4h/6h/8h thresholds are optimal for orchestrators (based on prior evidence, not new testing)
- ⚠️ Whether users actually understand when to use each architecture (assumed confusion, not validated)

**What would change this:**

- If hooks reliably fail → resume protocol would need rework
- If orchestrator context degradation matches agent degradation → thresholds should be unified
- If users consistently pick the right architecture → guidance already sufficient

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable improvements.

### Recommended Approach: Update Existing Documentation

**Add enable/constrain pattern to model** - Update `.kb/models/orchestrator-session-lifecycle.md` to use the enable/constrain query pattern established for models.

**Why this approach:**
- Model exists but doesn't explicitly state what each architecture enables/constrains
- Enable/constrain pattern makes knowledge queryable for strategic questions
- No new infrastructure needed, just documentation enhancement

**Trade-offs accepted:**
- Documentation-only fix doesn't add automated guardrails
- Users still need to read docs to avoid confusion
- Worth it because this is a guidance problem, not an architecture problem

**Implementation sequence:**
1. Add "What This Enables" section to model with table from synthesis above
2. Add "What This Constrains" section to model with table from synthesis above
3. Update spawned-orchestrator-pattern.md guide with technical comparison table

### Alternative Approaches Considered

**Option B: Add runtime validation to prevent mixing**
- **Pros:** Automated guardrails prevent confusion
- **Cons:** Adds complexity, may block valid edge cases
- **When to use instead:** If documentation proves insufficient after 30 days

**Option C: Unify architectures into single mechanism**
- **Pros:** Eliminates confusion, one thing to learn
- **Cons:** Loses either temporal OR hierarchical orchestration capability
- **When to use instead:** Never - architectures solve genuinely different problems

**Rationale for recommendation:** The architectures are sound; the gap is documentation clarity. Option A addresses this directly with minimal risk.

---

### Implementation Details

**What to implement first:**
- Add enable/constrain tables to orchestrator-session-lifecycle model
- Update spawned-orchestrator-pattern.md with technical comparison

**Things to watch out for:**
- ⚠️ Don't conflate the two architectures in any updated docs
- ⚠️ Ensure spawned orchestrator guidance explicitly states "do NOT use orch session end"
- ⚠️ Ensure interactive session guidance explicitly states "this is NOT for spawned orchestrators"

**Areas needing further investigation:**
- Hook reliability across environments (potential gap in resume protocol)
- Whether 4h/6h/8h thresholds need calibration based on usage data

**Success criteria:**
- ✅ Model has clear enable/constrain sections
- ✅ Guide has technical comparison table
- ✅ Users can answer "which architecture should I use?" without confusion

---

## References

**Files Examined:**
- `cmd/orch/session.go` - Session start/end/resume commands
- `pkg/session/session.go` - Session state management and checkpoint thresholds
- `pkg/spawn/orchestrator_context.go` - ORCHESTRATOR_CONTEXT.md template and generation
- `.kb/models/orchestrator-session-lifecycle.md` - Existing model
- `.kb/guides/orchestrator-session-management.md` - Existing guide
- `.kb/guides/spawned-orchestrator-pattern.md` - Existing spawned orchestrator guide
- `.kb/investigations/2026-01-13-inv-analyze-orchestrator-session-management-architecture.md` - Prior investigation

**Commands Run:**
```bash
# Pattern search for ORCHESTRATOR_CONTEXT references
rg "ORCHESTRATOR_CONTEXT" --type go

# Find orchestrator-related files
fd orchestrator .kb/
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-13-inv-analyze-orchestrator-session-management-architecture.md` - Prior analysis (this investigation deepens)
- **Model:** `.kb/models/orchestrator-session-lifecycle.md` - Target for enable/constrain update
- **Guide:** `.kb/guides/spawned-orchestrator-pattern.md` - Target for comparison table update

---

## Self-Review

- [x] Real test performed (code analysis of both architectures)
- [x] Conclusion from evidence (tables based on code)
- [x] Question answered (comprehensive comparison with enable/constrain)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled (summary complete)
- [x] Scope verified (searched for all relevant files)
- [x] Discovered work tracked (documentation updates recommended)

**Self-Review Status:** PASSED

---

## Leave it Better

```bash
kb quick constrain "Interactive sessions are global singleton - only one active at a time" --reason "session.json state model"
```

---

## Investigation History

**2026-01-14 14:30:** Investigation started
- Initial question: Deep comparison of interactive vs spawned orchestrator architectures
- Context: Task spawned to produce thorough analysis with recommendations

**2026-01-14 14:45:** Key findings emerged
- State models fundamentally different (singleton vs instance)
- Completion protocols opposite (self vs external)
- Handoff templates serve different documentation philosophies

**2026-01-14 15:00:** Synthesis completed
- Status: Complete
- Key outcome: Architectures are COMPLEMENTARY (temporal vs hierarchical orchestration), documentation should add enable/constrain pattern for queryability
