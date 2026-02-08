## Summary (D.E.K.N.)

**Delta:** Visual intent divergence happens because agents receive ambiguous descriptions ("more compact") and guess interpretations rather than forcing specificity upfront.

**Evidence:** Current clarifying-questions phase focuses on functional requirements (edge cases, error handling, integration); visual specifications not addressed. Directive-guidance pattern exists but isn't applied to visual work.

**Knowledge:** The fix isn't a new mechanism—it's extending spawn-time discipline to require verifiable visual specs for UI work. This is cheaper than dedicated agents and stronger than optional skill phases.

**Next:** Add UI visual spec requirement to spawn-time checks; create visual spec template for orchestrators to fill before spawning UI work.

---

# Investigation: UI Intent Clarification - Force Specificity Before Visual Changes

**Question:** Who should own pre-implementation visual clarification: worker skill phase, dedicated clarification agent, or spawn-time discipline?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Agent og-feat-ui-intent-clarification-26dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current clarifying-questions phase doesn't address visual specificity

**Evidence:** 
- Phase guidance (`~/.claude/skills/worker/feature-impl/reference/phase-clarifying-questions.md`) focuses on:
  - Edge cases (empty inputs, limits, concurrency)
  - Error handling (retry behavior, messages)
  - Integration points (API contracts, data flow)
  - Backward compatibility, Performance, Security
- No mention of visual specifications, layout constraints, or measurable UI criteria

**Source:** `~/.claude/skills/worker/feature-impl/reference/phase-clarifying-questions.md:40-49`

**Significance:** The existing mechanism could be extended but currently misses visual work entirely. This explains why agents guess at "more compact" interpretations.

---

### Finding 2: Directive-guidance pattern provides the right communication model

**Evidence:** 
- Pattern (`~/.orch/patterns/directive-guidance.md`) defines how to present recommendations with visible reasoning
- Already used for implementation decisions and architectural choices
- Works well for "confirm intent, don't quiz" approach needed for visual clarification

**Source:** `~/.orch/patterns/directive-guidance.md:1-100`

**Significance:** The communication pattern exists—we don't need to invent new ways for agents to ask about visual intent. We just need to apply directive-guidance to visual specifications.

---

### Finding 3: Spawn-time is the earliest and strongest enforcement point

**Evidence:**
- SPAWN_CONTEXT.md template (`pkg/spawn/context.go`) is already the control point for agent behavior
- Skills already have tier classification (`TierLight` vs `TierFull`) and server context inclusion
- Orchestrator already runs `kb context` before spawning (pre-spawn knowledge check)
- The "Surface Before Circumvent" pattern already gates workarounds

**Source:** `pkg/spawn/config.go:45-60`, `pkg/spawn/context.go:382-429`

**Significance:** Spawn-time enforcement ensures agents never start UI work without specs. This is stronger than relying on agents to self-enforce a clarifying phase.

---

## Synthesis

**Key Insights:**

1. **Visual intent divergence is a task specification problem, not an agent capability problem** - Agents can follow precise specs perfectly. The failure happens when specs like "more compact" have multiple valid interpretations. The root cause is accepting ambiguous task descriptions, not agents making wrong choices.

2. **Enforcement should happen at the earliest possible point** - Worker phases are optional and depend on agent self-discipline. Spawn-time enforcement guarantees specificity before any work begins. This follows the "gate early" principle.

3. **Visual specs need measurable criteria** - "More compact" fails because it's subjective. "Reduce card height by 30%" or "Remove whitespace between sections" succeeds because it's verifiable. The fix is a template that forces measurable visual requirements.

**Answer to Investigation Question:**

**Spawn-time discipline** should own visual clarification. Here's why each option ranks:

| Option | Ownership | Pros | Cons |
|--------|-----------|------|------|
| **Spawn-time discipline** ⭐ | Orchestrator | Guarantees specificity, no agent overhead, catches problem at source | Requires orchestrator training |
| Worker skill phase | Agent | Leverages existing phase structure | Optional, agent may skip, late in process |
| Dedicated clarification agent | System | Thorough analysis | Adds latency, cost, complexity |

Spawn-time wins because it prevents the problem rather than detecting it. If the task description lacks visual specificity, the orchestrator doesn't spawn—forcing clarification before agent context is consumed.

---

## Structured Uncertainty

**What's tested:**

- ✅ Clarifying-questions phase exists and works for functional requirements (verified: read phase-clarifying-questions.md)
- ✅ Directive-guidance pattern is documented and used (verified: read directive-guidance.md pattern)
- ✅ SPAWN_CONTEXT.md template is the spawn-time control point (verified: read pkg/spawn/context.go)

**What's untested:**

- ⚠️ Whether orchestrators will follow spawn-time visual spec requirements (not deployed)
- ⚠️ Whether visual spec template captures all common UI ambiguities (not validated against real failures)
- ⚠️ Whether spawn-time enforcement adds meaningful overhead for orchestrators (not measured)

**What would change this:**

- If orchestrators consistently ignore spawn-time requirements → move to mandatory worker phase with blocking
- If visual spec template is too rigid and blocks valid work → allow "subjective override" with explicit acknowledgment
- If spawn-time overhead is unacceptable → fall back to dedicated clarification agent

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Spawn-Time Visual Spec Enforcement** - Add mandatory visual specification template for UI work, enforced at spawn time by orchestrator.

**Why this approach:**
- Prevents ambiguity before agent context is consumed (Finding 3)
- Leverages existing spawn-time enforcement patterns (Surface Before Circumvent)
- Forces orchestrator to clarify with user BEFORE spawning, not during or after

**Trade-offs accepted:**
- Requires orchestrator discipline (not agent-enforced)
- May add friction to quick UI tweaks
- Acceptable because: orchestrator is already the quality gate for spawn context

**Implementation sequence:**
1. **Create Visual Spec Template** - Markdown checklist forcing measurable visual requirements
2. **Update Orchestrator Skill** - Add "UI Work Detection" to pre-spawn checklist
3. **Optional: Add spawn-time validation** - Detect UI task descriptions without spec

### Alternative Approaches Considered

**Option B: Extended clarifying-questions phase**
- **Pros:** Uses existing skill infrastructure, agents already know this phase
- **Cons:** Optional, late in process (agent already spawned), agents may rationalize skipping
- **When to use instead:** If spawn-time overhead proves unacceptable

**Option C: Dedicated visual clarification agent**
- **Pros:** Thorough, can analyze screenshots, produce detailed specs
- **Cons:** Adds latency, cost (~$0.50+ per clarification), complexity
- **When to use instead:** For complex multi-component redesigns where spec generation is substantial work

**Rationale for recommendation:** Visual divergence is fundamentally a specification problem at input time. Fixing it at spawn-time (Option A) addresses the root cause. Options B/C treat the symptom (helping agents cope with ambiguity) rather than the cause (accepting ambiguity in the first place).

---

### Implementation Details

**What to implement first:**

1. **Visual Spec Template** (add to orchestrator skill or `.orch/templates/`):
```markdown
## Visual Specification

### Change Description
[What visual change is being requested?]

### Measurable Criteria
- [ ] Layout: [specific dimension changes, e.g., "reduce height by 30%"]
- [ ] Spacing: [specific padding/margin, e.g., "8px gap between items"]
- [ ] Typography: [specific font changes, e.g., "use 14px for labels"]
- [ ] Colors: [specific color values if applicable]

### Reference
- [ ] Screenshot of current state attached: [path or description]
- [ ] Mockup/wireframe of desired state: [path or description]
- [ ] OR: "Match existing pattern in [component name]"

### Acceptance Criteria
How will we verify this is correct?
- [ ] Visual comparison to mockup
- [ ] Playwright screenshot diff
- [ ] Manual orchestrator review
```

2. **Orchestrator Pre-Spawn Check** (add to "Pre-Spawn Checklist" section):
```markdown
**UI Work Detection:**
- Does task description mention visual changes (layout, spacing, styling, appearance)?
- If YES → Visual spec required before spawn
- If task has ambiguous visual terms ("more compact", "cleaner", "better") → STOP and clarify
```

**Things to watch out for:**
- ⚠️ Don't over-apply: pure logic/backend work shouldn't require visual specs
- ⚠️ Allow escape hatch for truly subjective exploratory UI work (but require explicit acknowledgment)
- ⚠️ Template should be fast to fill (~2 min), not a burden

**Areas needing further investigation:**
- How to detect "visual ambiguity" programmatically (vs relying on orchestrator judgment)
- Whether Playwright screenshots should be mandatory pre-spawn (current: optional)
- Integration with dashboard UI for visual spec display

**Success criteria:**
- ✅ Zero UI work spawns without measurable visual criteria
- ✅ Agents report Phase: Complete without "actually, that's not what I meant" rejections
- ✅ Orchestrator can fill visual spec in <2 minutes for typical UI tasks

---

## References

**Files Examined:**
- `~/.claude/skills/worker/feature-impl/reference/phase-clarifying-questions.md` - Current clarifying phase guidance
- `~/.orch/patterns/directive-guidance.md` - Communication pattern for recommendations
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template generation
- `pkg/spawn/config.go` - Spawn configuration and tier defaults

**Commands Run:**
```bash
# Check for prior knowledge
kb context "visual verification UI clarification intent"

# Check directive-guidance pattern
kb context "directive guidance"
```

**External Documentation:**
- None required (internal system analysis)

**Related Artifacts:**
- **Pattern:** `~/.orch/patterns/directive-guidance.md` - Communication pattern to apply
- **Decision:** Prior constraint "Ask 'should we' before 'how do we'" - Validates premise-first approach

---

## Investigation History

**2025-12-26 22:05:** Investigation started
- Initial question: Who should own visual intent clarification: worker phase, dedicated agent, or spawn-time?
- Context: Visual intent diverges because descriptions like "more compact" have multiple valid interpretations

**2025-12-26 22:15:** Key finding - existing phase doesn't address visual specs
- Clarifying-questions phase focuses on functional, not visual requirements

**2025-12-26 22:25:** Investigation completed
- Status: Complete
- Key outcome: Spawn-time discipline recommended - add visual spec template + orchestrator pre-spawn check
