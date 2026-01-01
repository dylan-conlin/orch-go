<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Designed a unified friction-capture protocol for session-end: workers report friction via SYNTHESIS.md "Friction" section, orchestrators capture via structured `orch session end` prompts, both feed into `kb learn` for system improvement.

**Evidence:** Current state analysis shows: workers have "Leave it Better" (knowledge externalization) but no friction reporting; orchestrators have Session Reflection (3 checkpoints) but no structured capture format; `orch session end` prompts for Knowledge/Next but not friction.

**Knowledge:** Friction is distinct from knowledge - "what was hard" vs "what was learned". Workers can surface friction that only they experience (tool failures, missing context) but can't act on it; orchestrators can act but don't always know. System pressure requires friction to flow upstream.

**Next:** Implement in sequence: (1) Add Friction section to SYNTHESIS.md template, (2) Enhance `orch session end` to prompt for friction + system reaction, (3) Update orchestrator Session Reflection to reference new flow.

---

# Investigation: Session End Reflection Ritual

**Question:** How should friction capture work as a standard session-end ritual for both orchestrators and workers? What format, what mechanism for upstream reporting, should it be gated, and how does it integrate with `orch session end` and `orch complete`?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** og-work-session-end-reflection-01jan
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-26-inv-session-end-workflow-orchestrators.md (extends, does not replace)
**Superseded-By:** N/A

---

## Findings

### Finding 1: Workers have Leave it Better, but it captures knowledge not friction

**Evidence:** The "Leave it Better" phase in worker skills focuses on externalizing knowledge via `kn` commands:
- `kn decide` - choices made
- `kn tried` - failed approaches
- `kn constrain` - constraints discovered
- `kn question` - open questions

This is about **what was learned**, not **what was hard**. There's no section for "this tool was broken", "this context was missing", "this spawn prompt was unclear".

**Source:** 
- `/Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md:275-291`
- `/Users/dylanconlin/.claude/skills/worker/feature-impl/SKILL.md:366-402`

**Significance:** Workers experience friction (tooling failures, missing context, unclear prompts) that the orchestrator doesn't see. Without a capture mechanism, this friction is lost, and the system can't improve.

---

### Finding 2: Orchestrators have Session Reflection with three checkpoints

**Evidence:** The orchestrator skill includes "Session Reflection (Before Ending Orchestrator Session)" with three checkpoints:

1. **Friction Audit:** "What was harder than it should have been?"
2. **Gap Capture:** "What knowledge should have been surfaced but wasn't?"
3. **System Reaction Check:** "Does this session suggest system improvements?"

Gate: Run `orch learn`, `kn` command, or explicit skip.

**Source:** `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md:1121-1152`

**Significance:** The orchestrator reflection is well-designed but:
1. Not integrated with `orch session end` (which prompts for Knowledge/Next, not friction)
2. Doesn't receive friction from workers (orchestrators can only reflect on what they experienced)
3. The three checkpoints are good framing but lack structured output format

---

### Finding 3: `orch session end` captures D.E.K.N. but not friction

**Evidence:** The `orch session end` command:
1. Warns about in-progress agents
2. Prompts for Knowledge and Next sections (human-authored)
3. Auto-gathers Delta and Evidence from git stats
4. Saves to SESSION_HANDOFF.md

**What's missing:** No prompt for friction. The handoff document tells the next session what happened, but not what was hard.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/session.go:302-446`

**Significance:** The ceremony exists but doesn't ask the right questions. Adding friction prompts to `orch session end` would create a forcing function.

---

### Finding 4: SYNTHESIS.md has Unexplored Questions but no Friction section

**Evidence:** The SYNTHESIS.md template includes:
- Delta (What Changed)
- Evidence (What Was Observed)
- Knowledge (What Was Learned)
- Next (What Should Happen)
- **Unexplored Questions** (Questions that emerged)

But no section for friction. The template explicitly says "fill AS YOU WORK, not at the end" for progressive capture.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md:140-154`

**Significance:** Adding a Friction section to SYNTHESIS.md would:
1. Capture worker friction at the source
2. Flow upstream via `orch complete` (orchestrator reviews SYNTHESIS.md)
3. Integrate with existing progressive documentation pattern

---

### Finding 5: Friction and knowledge are distinct concerns

**Evidence:** Analyzing the purpose of each:

| Concern | Focus | Example | Who Can Act |
|---------|-------|---------|-------------|
| **Knowledge** | "What did I learn?" | "Redis needs idempotency keys" | Next agent |
| **Friction** | "What was hard?" | "Spawn prompt was missing X" | System maintainer |

Knowledge helps the next agent do similar work. Friction helps the system get better at supporting agents.

The "Pressure Over Compensation" principle (orchestrator skill L570-616) explicitly says: "When the system fails to surface knowledge, don't compensate by providing it manually. Let the failure surface."

**Source:** 
- Analysis of skill patterns
- `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md` (Pressure Over Compensation section)

**Significance:** Friction capture is a distinct concern that needs its own format and flow. It's not just another knowledge type - it's system feedback.

---

### Finding 6: kb learn infrastructure exists for pattern detection

**Evidence:** The `orch learn` command (now delegated to `kb learn`) was designed to:
- Surface recurring context gaps
- Show suggestions for improvement
- Track whether improvements helped

The infrastructure exists but isn't fed by structured friction capture.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/learn.go`

**Significance:** If friction is captured in a structured format, `kb learn` could aggregate patterns ("5 agents reported missing X context") and surface them for action.

---

## Synthesis

**Key Insights:**

1. **Friction flows upstream, knowledge flows forward** - Workers capture knowledge for the next agent doing similar work. Workers capture friction for the orchestrator/system maintainer to improve the system. These are different flows with different consumers.

2. **Capture at source, act at authority** - Workers are at the source of friction (they experience it). Orchestrators have authority to act (update skills, spawns, tools). The design must bridge these levels.

3. **Progressive capture beats recall** - Both SYNTHESIS.md guidance and Session Amnesia principle emphasize capturing during work, not reconstructing at end. Friction should be logged as it happens.

4. **Gating creates pressure** - Leave it Better is gated (can't complete without it). Session Reflection suggests gating. Without a gate, reflection becomes optional under time pressure.

**Answer to Investigation Question:**

### 1. What's the right format for friction capture?

**For workers (SYNTHESIS.md):**
Add a "Friction" section between Knowledge and Unexplored Questions:

```markdown
## Friction (What Was Hard)

**Tool/System Issues:**
- [Tool that failed or behaved unexpectedly]
- [Missing capability that caused workaround]

**Context Gaps:**
- [Knowledge that should have been in spawn context]
- [Prior finding that would have helped but wasn't surfaced]

**Process Friction:**
- [Step that felt harder than necessary]
- [Guidance that was unclear or contradictory]

*(If straightforward session, note: "No friction - smooth execution")*
```

This format:
- Distinguishes tool issues from context gaps from process friction
- Parallels Unexplored Questions structure
- Supports progressive capture (add items as friction occurs)

**For orchestrators (`orch session end`):**
Add friction prompts to the D.E.K.N. section:

```
Friction (What was harder than it should have been?):
  Tool/system issues, context gaps, process friction.
  > [user input]

System Reaction (Does this suggest improvements?):
  Skill update, CLAUDE.md update, new tooling?
  > [user input]
```

### 2. How should workers report friction upstream?

**Via SYNTHESIS.md → `orch complete`:**

1. Worker fills Friction section in SYNTHESIS.md
2. When orchestrator runs `orch complete`, the review step shows Friction section
3. Orchestrator decides: create beads issue, run `kn constrain`, update spawn template, etc.

**Optional enhancement:** Structured JSON in workspace for `kb learn` aggregation:
```json
// .orch/workspace/{name}/friction.json
{
  "tool_issues": ["orch status showed stale sessions"],
  "context_gaps": ["No prior art on X topic"],
  "process_friction": ["Unclear whether to spawn or investigate first"]
}
```

This enables `kb learn` to detect patterns across agents.

### 3. Should reflection be gated/required?

**For workers:** Soft gate. Friction section in SYNTHESIS.md is displayed in template but can be filled with "No friction - smooth execution". This is acceptable because:
- Not all sessions have friction
- Hard gate would add overhead for straightforward work
- Presence in template is a prompt/reminder

**For orchestrators:** Soft gate with explicit skip. The Session Reflection section says:
> Gate: Run at least one of `orch learn`, `kn` command, or explicit skip

This matches the `orch session end` flow - prompting is mandatory, but "skipped" is a valid answer.

### 4. How does this integrate with `orch session end` and `orch complete`?

**`orch session end` changes:**

Current prompts:
- Knowledge
- Next

New prompts:
- Knowledge (keep)
- Next (keep)
- **Friction** (new)
- **System Reaction** (new)

These get saved to SESSION_HANDOFF.md alongside the D.E.K.N. sections.

**`orch complete` changes:**

When reviewing worker SYNTHESIS.md, surface the Friction section:
```
Friction reported:
- Missing context on session lifecycle
- orch status showed 40+ stale agents
```

Orchestrator can then:
- Create beads issue: `bd create "Fix stale session display"`
- Record constraint: `kn constrain "..."`
- Update spawn template

---

## Structured Uncertainty

**What's tested:**

- ✅ Current SYNTHESIS.md template has Unexplored Questions section (verified: read template file)
- ✅ `orch session end` prompts for Knowledge and Next (verified: read session.go:367-395)
- ✅ Orchestrator Session Reflection has three checkpoints (verified: read SKILL.md:1127-1143)
- ✅ Workers have Leave it Better focused on knowledge externalization (verified: read multiple skill files)

**What's untested:**

- ⚠️ Whether workers will actually fill Friction section (compliance not tested)
- ⚠️ Whether `kb learn` can aggregate structured friction data (integration not tested)
- ⚠️ Whether orchestrators will act on surfaced friction (behavior not tested)
- ⚠️ Optimal prompting language for friction capture (phrasing not A/B tested)

**What would change this:**

- If workers consistently ignore Friction section, harder gating might be needed
- If friction patterns don't recur, aggregation may not be valuable
- If orchestrators don't act on friction, upstream flow needs different mechanism

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Structured friction sections + enhanced session-end prompts** - Add Friction section to SYNTHESIS.md template, add friction prompts to `orch session end`, update `orch complete` to surface worker friction.

**Why this approach:**
- Builds on existing infrastructure (SYNTHESIS.md, session commands)
- Captures at source (workers log friction as they experience it)
- Creates upstream flow (orchestrator sees worker friction in review)
- Supports pattern detection (structured format enables `kb learn` aggregation)

**Trade-offs accepted:**
- Soft gate may lead to empty Friction sections
- Orchestrator must remember to act on friction (no automation)
- Extra prompts in `orch session end` add ~30 seconds

**Implementation sequence:**
1. **SYNTHESIS.md template** - Add Friction section (lowest risk, immediate value)
2. **`orch session end`** - Add Friction and System Reaction prompts
3. **`orch complete`/review** - Surface Friction section in review display
4. **Orchestrator skill** - Update Session Reflection to reference new flow
5. **(Optional)** `friction.json` - Structured file for `kb learn` aggregation

### Alternative Approaches Considered

**Option B: Beads comment-based friction**
- **Pros:** Immediate visibility in issue timeline, no template changes
- **Cons:** Unstructured, hard to aggregate, mixed with progress comments
- **When to use instead:** If SYNTHESIS.md overhead is too high for quick spawns

**Option C: Dedicated `orch friction` command**
- **Pros:** Explicit action, could auto-create beads issues
- **Cons:** Extra step workers might skip, doesn't integrate with session-end
- **When to use instead:** If friction needs immediate escalation (blocking issues)

**Option D: Post-session survey**
- **Pros:** Could prompt for specific friction types, quantifiable
- **Cons:** Recall is worse than real-time capture, adds ceremony
- **When to use instead:** For periodic system health assessment, not session-level

**Rationale for recommendation:** Option A integrates with existing patterns (SYNTHESIS.md progressive capture, session-end ceremony) without adding new tools or ceremonies. It creates pressure through prompting rather than hard gates, matching the design philosophy of existing systems.

---

### Implementation Details

**What to implement first:**

1. **SYNTHESIS.md template update:**
   - Add Friction section between Knowledge and Unexplored Questions
   - Include sub-categories: Tool/System Issues, Context Gaps, Process Friction
   - Add "No friction" escape hatch for smooth sessions

**Proposed Friction section:**

```markdown
---

## Friction (What Was Hard)

**Tool/System Issues:**
- [Tool that failed or behaved unexpectedly]
- [Missing capability that caused workaround]

**Context Gaps:**
- [Knowledge that should have been in spawn context]
- [Prior finding that would have helped but wasn't surfaced]

**Process Friction:**
- [Step that felt harder than necessary]
- [Guidance that was unclear or contradictory]

*(If straightforward session, note: "No friction - smooth execution")*
```

2. **`orch session end` prompt additions:**

In `cmd/orch/session.go`, add after the Knowledge prompt:

```go
// Friction prompt
fmt.Println()
fmt.Println("Friction (What was harder than it should have been?):")
fmt.Println("  Tool issues, context gaps, process friction.")
fmt.Print("  > ")
friction, _ := reader.ReadString('\n')
friction = strings.TrimSpace(friction)
if friction == "" {
    fmt.Println("  (skipped)")
}
handoffData.DEKN.Friction = friction

// System Reaction prompt
fmt.Println()
fmt.Println("System Reaction (Does this suggest improvements?):")
fmt.Println("  Skill update, CLAUDE.md update, new tooling?")
fmt.Print("  > ")
reaction, _ := reader.ReadString('\n')
reaction = strings.TrimSpace(reaction)
if reaction == "" {
    fmt.Println("  (skipped)")
}
handoffData.DEKN.SystemReaction = reaction
```

Also add to `DEKNSummary` struct:
```go
Friction       string `json:"friction,omitempty"`       // What was hard
SystemReaction string `json:"system_reaction,omitempty"` // Improvements needed
```

3. **Orchestrator skill update:**

Update Session Reflection section to reference the new flow:

```markdown
**Orchestrator Session-End Flow:**
1. Run `orch session end`
2. Answer prompts: Knowledge, Next, Friction, System Reaction
3. If friction surfaced, decide: beads issue, kn entry, or skill/template update
4. Handoff saved to ~/.orch/session/{date}/SESSION_HANDOFF.md
```

**Things to watch out for:**

- ⚠️ Friction prompts should be optional (empty string valid)
- ⚠️ Handoff markdown template needs updating to include new sections
- ⚠️ Friction section might be unfilled initially (workers need training)
- ⚠️ Don't make prompts so long they create fatigue

**Areas needing further investigation:**

- How to surface worker friction in `orch complete` display
- Whether `friction.json` structured file is worth the complexity
- Integration with `kb learn` for pattern detection

**Success criteria:**

- ✅ Workers have Friction section in SYNTHESIS.md template
- ✅ `orch session end` prompts for Friction and System Reaction
- ✅ SESSION_HANDOFF.md includes Friction and System Reaction sections
- ✅ Orchestrator can see worker friction when reviewing via `orch complete`
- ✅ At least one instance of friction leading to system improvement (validates the flow)

---

## References

**Files Examined:**
- `/Users/dylanconlin/.claude/skills/meta/orchestrator/SKILL.md` - Session Reflection section
- `/Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md` - Leave it Better pattern
- `/Users/dylanconlin/.claude/skills/worker/feature-impl/SKILL.md` - Leave it Better pattern
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md` - Current template
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/session.go` - Session end implementation
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/handoff.go` - Handoff data structures

**Commands Run:**
```bash
# Check existing Session Reflection in skill
grep -n "Session Reflection" ~/.claude/skills/**/*.md

# Check SYNTHESIS.md template structure
cat .orch/templates/SYNTHESIS.md

# Check orch session end prompts
grep -A20 "Knowledge prompt" cmd/orch/session.go
```

**Related Artifacts:**
- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-26-inv-session-end-workflow-orchestrators.md` - Prior work on orchestrator session-end (this extends it)
- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-reflection-checkpoint-pattern-agent-sessions.md` - Unexplored Questions design
- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-21-inv-design-self-reflection-protocol-specification.md` - Self-reflection architecture

---

## Investigation History

**2026-01-01 09:00:** Investigation started
- Initial question: How to make friction capture a standard session-end ritual
- Context: Realization that orchestrators and workers both need reflection, but have different patterns

**2026-01-01 09:15:** Context gathering complete
- Found: Workers have Leave it Better (knowledge), not friction
- Found: Orchestrators have Session Reflection (checkpoints), not structured capture
- Found: `orch session end` prompts for Knowledge/Next, not friction
- Found: SYNTHESIS.md has Unexplored Questions, not Friction

**2026-01-01 09:30:** Design synthesized
- Key insight: Friction flows upstream, knowledge flows forward
- Proposed: SYNTHESIS.md Friction section + enhanced `orch session end` prompts
- Integration: Worker friction surfaces via `orch complete` review

**2026-01-01 09:45:** Investigation completed
- Status: Complete
- Key outcome: Structured friction capture design with implementation sequence
