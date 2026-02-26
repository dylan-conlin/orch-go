<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The orchestration system has 8 surfacing mechanisms, but 4 are passive (require human to invoke) and 4 are active (automatically surface); the gap is the "pressure visibility layer" - making failures visible so humans don't compensate.

**Evidence:** Cataloged all surfacing mechanisms: kb context (pre-spawn), SessionStart hooks (session init), kb reflect/daemon (pattern detection), beads ready/status (work surfacing), SPAWN_CONTEXT.md (agent context), orch status (swarm monitoring), SYNTHESIS.md (agent outputs), completion lifecycle (post-work). Only hooks, spawn context, and daemon are truly proactive.

**Knowledge:** Pressure Over Compensation requires visible failures, not just available data. The principle says "don't compensate for gaps" but the current system makes compensation frictionless because gaps are invisible until a human asks.

**Next:** Create epic for "Pressure Visibility System" with 3 children: (1) Gap Detection - detect when orchestrator lacks context it should have, (2) Failure Surfacing - show gaps prominently vs bury in logs, (3) System Learning Loop - convert gap observations into mechanism improvements.

**Confidence:** High (85%) - Clear mechanism inventory, principle application is design interpretation not fact

---

# Investigation: Pressure Over Compensation - Surfacing Mechanisms Audit

**Question:** What surfacing mechanisms exist in the orchestration system, and how should they be improved to apply the Pressure Over Compensation principle (letting failures create pressure for system improvement rather than human compensation)?

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** og-work-pressure-over-compensation-25dec
**Phase:** Complete
**Next Step:** Create epic with children for Pressure Visibility System
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Eight Distinct Surfacing Mechanisms Exist

**Evidence:** The orchestration system has 8 surfacing mechanisms, categorized by when they operate:

| Mechanism | When | Type | Purpose |
|-----------|------|------|---------|
| `kb context` | Pre-spawn | Passive | Surface relevant knowledge before agent work |
| SessionStart hooks | Session init | Active | Inject context at session beginning |
| `kb reflect` / daemon | Background | Active | Detect patterns requiring human attention |
| `bd ready` / `bd status` | On-demand | Passive | Surface available work |
| SPAWN_CONTEXT.md | Agent spawn | Active | Provide agent with required context |
| `orch status` | On-demand | Passive | Show swarm state |
| SYNTHESIS.md | Agent completion | Active | Externalize agent findings |
| Completion lifecycle | Post-work | Passive | Review completed work |

**Source:** 
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md generation
- `pkg/spawn/kbcontext.go` - KB context integration
- `pkg/daemon/reflect.go` - Reflection analysis
- `~/.claude/hooks/cdd-hooks.json` - Hook configuration
- `~/.kb/principles.md` - Surfacing Over Browsing principle

**Significance:** The system has substantial surfacing infrastructure, but 4/8 mechanisms are passive (require human to invoke). The gap isn't "no surfacing" - it's "surfacing doesn't create pressure."

---

### Finding 2: Passive vs Active Surfacing Creates Compensation Opportunity

**Evidence:** Passive mechanisms require human to remember to invoke them:

**Passive (Human must invoke):**
- `kb context "topic"` - Human must query before spawning
- `bd ready` - Human must check for work
- `orch status` - Human must check swarm state
- `orch complete` / review - Human must initiate completion

**Active (System invokes automatically):**
- SessionStart hooks - Run on every session
- SPAWN_CONTEXT.md - Generated at spawn time with kb context embedded
- `kb reflect` / daemon - Runs in background
- SYNTHESIS.md requirement - Gated by skill completion protocol

When passive mechanisms fail (human forgets to invoke), the human compensates by providing context directly. This is the exact anti-pattern the Pressure Over Compensation principle warns against.

**Source:**
- `pkg/spawn/kbcontext.go:100-146` - RunKBContextCheck is called during spawn, making it active
- `~/.claude/hooks/cdd-hooks.json` - SessionStart hooks are automatic
- Observation: bd ready, orch status, orch complete are manual commands

**Significance:** The pre-spawn kb context check (now automatic) was a direct response to this gap. But other mechanisms (completion review, status checking) remain passive.

---

### Finding 3: Gaps Are Invisible Until Human Notices

**Evidence:** Current gap detection is ad-hoc:

1. **No "context quality" feedback** - When kb context returns sparse results, agent proceeds anyway
2. **No "did you know?" prompts** - If agent should know something but kb context missed it, no warning
3. **No completion quality gate** - SYNTHESIS.md is required, but content quality isn't verified
4. **No "pattern detection" surfacing** - kb reflect runs but results sit in `~/.orch/reflect-suggestions.json` until human queries

The prior investigation (`2025-12-24-inv-orchestrator-skill-says-complete-agents.md`) found that even documented guidance fails when signal ratio is wrong. The same applies here: surfacing mechanisms exist but their outputs aren't visible enough to prevent compensation.

**Source:**
- `pkg/spawn/kbcontext.go:452-538` - FormatContextForSpawnWithLimit truncates but doesn't warn about gaps
- `pkg/daemon/reflect.go:103-125` - SaveSuggestions writes to file, no active notification
- Observation: kb reflect output shows 19+ synthesis opportunities, but this isn't surfaced to orchestrator

**Significance:** The Pressure Over Compensation principle requires failures to be felt. But current architecture makes gaps invisible - the human only notices when they have to paste context manually.

---

### Finding 4: The "Compensation Loop" Is Frictionless

**Evidence:** When surfacing fails, compensation is easier than fixing:

```
Gap occurs (orchestrator doesn't know something it should)
    ↓
Human notices (or agent asks, or work fails)
    ↓
Human pastes context (path of least resistance)
    ↓
Work proceeds (immediate success)
    ↓
Gap never surfaces as issue (no pressure to fix)
    ↓
Next session: same gap, same compensation
```

The Pressure Over Compensation principle (`~/.kb/decisions/2025-12-25-pressure-over-compensation.md`) explicitly addresses this:

> "Every time a human manually provides context, they're relieving pressure on the system, preventing the gap from being felt, ensuring the mechanism never gets built."

**Source:**
- `~/.kb/decisions/2025-12-25-pressure-over-compensation.md` - Principle decision
- Observation: kb context errors return nil, not an error (line 167 of kbcontext.go)

**Significance:** The architecture makes compensation frictionless. To apply the principle, we need to add friction to compensation and visibility to gaps.

---

### Finding 5: Three Missing Layers for Pressure Over Compensation

**Evidence:** Analyzing what the principle requires vs what exists:

| Required Layer | Purpose | Exists? |
|---------------|---------|---------|
| **Gap Detection** | Know when system lacks context it should have | Partial (kb context sparse warning, but buried) |
| **Failure Surfacing** | Make gaps visible and painful to ignore | No (failures are silent or logged only) |
| **System Learning** | Convert observed gaps into mechanism improvements | Partial (manual - create issues) |

The prior investigation `2025-12-21-inv-orchestrator-session-boundaries.md` noted:
> "The system lacks a unified 'session boundary approaching' signal. Workers complete abruptly; orchestrators handle boundaries manually."

This is another instance of the same pattern: passive mechanisms instead of active surfacing.

**Source:**
- `pkg/spawn/kbcontext.go` - Has truncation warning but no "gap detected" signal
- No existing "gap visibility" layer in codebase
- `~/.kb/principles.md` - Gate Over Remind principle applies here too

**Significance:** Three specific layers would operationalize Pressure Over Compensation: detect gaps, surface them visibly, convert to improvements. These could become an epic.

---

## Synthesis

**Key Insights:**

1. **Quantity of mechanisms ≠ effective surfacing** - The system has 8 surfacing mechanisms, but most are passive. Having `bd ready` doesn't help if no one runs it. Having `kb context` doesn't help if results aren't included in spawns.

2. **Compensation is frictionless, pressure is invisible** - The current architecture makes it easier to paste context manually than to fix the underlying gap. This is precisely the anti-pattern the principle warns against.

3. **The gap is a "Pressure Visibility Layer"** - What's missing isn't more surfacing mechanisms, but a layer that:
   - Detects when surfacing failed (gap detection)
   - Makes failures visible and painful (failure surfacing)
   - Converts observations into system improvements (learning loop)

4. **Gate Over Remind applies here too** - From `~/.kb/principles.md`: "Enforce knowledge capture through gates (cannot proceed without), not reminders (easily ignored)." The same applies to gap detection - reminders to check kb context fail; gates that block when context is sparse would work.

**Answer to Investigation Question:**

Eight surfacing mechanisms exist: kb context (pre-spawn), SessionStart hooks (session init), kb reflect/daemon (pattern detection), bd ready/status (work surfacing), SPAWN_CONTEXT.md (agent context), orch status (swarm monitoring), SYNTHESIS.md (agent outputs), and completion lifecycle (post-work).

To apply Pressure Over Compensation, the system needs three new layers:

1. **Gap Detection** - Detect when orchestrator/agent lacks context it should have
   - Sparse kb context results → warning flag
   - Agent asks question from spawn context → gap detected
   - Completion shows agent had to ask clarifying questions → gap detected

2. **Failure Surfacing** - Make gaps visible and painful to ignore
   - Don't just log gaps - display prominently
   - Block or warn when proceeding with sparse context
   - Show "you compensated here last session" reminders

3. **System Learning Loop** - Convert gap observations into mechanism improvements
   - Automatically suggest beads issues for recurring gaps
   - Track which topics frequently lack context
   - Recommend kb entries or hook additions

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The mechanism inventory is concrete - I examined actual code and configuration. The principle application is design interpretation based on the decision document. The gap analysis connects these but hasn't been tested.

**What's certain:**

- ✅ 8 surfacing mechanisms exist as cataloged
- ✅ 4 are passive, 4 are active
- ✅ Compensation is currently frictionless (no gates prevent it)
- ✅ Pressure Over Compensation principle explicitly addresses this pattern

**What's uncertain:**

- ⚠️ Whether gap detection is technically feasible without agent introspection
- ⚠️ How much friction to add (too much = unusable, too little = ignored)
- ⚠️ Whether this requires new tooling or just better hook configuration

**What would increase confidence to Very High (95%+):**

- Prototype gap detection in a real spawn/complete cycle
- Test with actual orchestrator sessions to see where compensation occurs
- Get Dylan's feedback on the three-layer model

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Create "Pressure Visibility System" Epic** - Build three layers that operationalize the Pressure Over Compensation principle.

**Why this approach:**
- Directly addresses the gap between principle and practice
- Structured as buildable increments (detect → surface → learn)
- Leverages existing mechanisms (kb context, hooks, beads)

**Trade-offs accepted:**
- Adds friction to orchestration workflow (by design - that's the pressure)
- May slow down initially while system learns (necessary for long-term improvement)

**Implementation sequence:**
1. **Gap Detection** - Start here because you can't surface what you can't detect
2. **Failure Surfacing** - Once detected, make gaps visible
3. **System Learning Loop** - Convert observations into improvements

### Alternative Approaches Considered

**Option B: Just add more hooks**
- **Pros:** Simpler, uses existing infrastructure
- **Cons:** Doesn't address the detection problem - hooks inject context but don't detect gaps
- **When to use instead:** If gap detection proves too complex

**Option C: Accept passive mechanisms**
- **Pros:** No new work, system already has surfacing
- **Cons:** Doesn't apply Pressure Over Compensation - compensation remains frictionless
- **When to use instead:** If the principle is deprioritized

**Rationale for recommendation:** Option A directly operationalizes the principle with structured, buildable components. Options B and C don't address the core gap.

---

### Implementation Details

**What to implement first:**
- Gap Detection at spawn time (when kb context returns sparse results)
- This is highest leverage - catches gaps before agent work begins

**Things to watch out for:**
- ⚠️ Don't block spawns completely - add warnings first, gates later
- ⚠️ Gap detection heuristics may have false positives - need tuning
- ⚠️ Failure surfacing must be visible but not intrusive

**Areas needing further investigation:**
- How to detect gaps during agent work (not just pre-spawn)?
- How to track compensation patterns across sessions?
- What constitutes "sparse" results for kb context?

**Success criteria:**
- ✅ Orchestrator sees gap warnings when context is sparse
- ✅ Compensation patterns are tracked and visible
- ✅ At least 1 gap converts to system improvement per week (beads issue, kn entry, hook)

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template
- `pkg/spawn/kbcontext.go` - KB context integration
- `pkg/daemon/reflect.go` - Reflection analysis
- `~/.claude/hooks/cdd-hooks.json` - Hook configuration
- `~/.kb/principles.md` - Principles document
- `~/.kb/decisions/2025-12-25-pressure-over-compensation.md` - Principle decision

**Commands Run:**
```bash
# Check surfacing mechanisms
kb context "surfacing"
kb reflect --format json

# Check work surfacing
bd ready
orch status

# Check daemon reflection
cat ~/.orch/reflect-suggestions.json
```

**Related Artifacts:**
- **Decision:** `~/.kb/decisions/2025-12-25-pressure-over-compensation.md` - The principle this operationalizes
- **Investigation:** `2025-12-24-inv-orchestrator-skill-says-complete-agents.md` - Signal ratio analysis
- **Investigation:** `2025-12-21-inv-orchestrator-session-boundaries.md` - Session boundary gaps

---

## Investigation History

**2025-12-25 ~15:00:** Investigation started
- Initial question: What surfacing mechanisms exist and how to improve them for Pressure Over Compensation?
- Context: Spawned as design-session for principle application

**2025-12-25 ~15:30:** Found 8 surfacing mechanisms
- Categorized by when they operate and active vs passive
- Identified that 4/8 are passive (require human invocation)

**2025-12-25 ~15:45:** Identified the "Pressure Visibility Layer" gap
- Three missing layers: gap detection, failure surfacing, system learning
- Connected to Gate Over Remind principle

**2025-12-25 ~16:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Recommend epic with 3 children for Pressure Visibility System
