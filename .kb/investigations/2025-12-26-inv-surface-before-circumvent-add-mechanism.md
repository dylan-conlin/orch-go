<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added "Surface Before Circumvent" guidance to SPAWN_CONTEXT.md template in the AUTHORITY section.

**Evidence:** Tests pass for both tracked (bd comment) and untracked (investigation file fallback) scenarios.

**Knowledge:** The mechanism integrates naturally as an escalation trigger in the existing AUTHORITY section, with conditional formatting based on NoTrack flag.

**Next:** Close - implementation complete with tests.

---

# Investigation: Surface Before Circumvent Add Mechanism

**Question:** Where and how should the "Surface Before Circumvent" principle be implemented as a mechanism agents encounter?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: AUTHORITY section is the natural home for constraint surfacing

**Evidence:** The SPAWN_CONTEXT.md template already has an AUTHORITY section with:
- "You have authority to decide" - things agents can do autonomously
- "You must escalate" - things requiring orchestrator approval
- "When uncertain" - fallback guidance

**Source:** `pkg/spawn/context.go:83-99`

**Significance:** Working around a constraint is an escalation trigger. Adding "Surface Before Circumvent" to AUTHORITY section creates a consistent pattern: agents are trained to surface constraints just like they surface ambiguous requirements.

---

### Finding 2: Tracked vs untracked spawns need different instructions

**Evidence:** The template already uses conditional logic `{{if .NoTrack}}` to handle ad-hoc spawns. Tracked spawns use `bd comment` for progress tracking, while untracked spawns rely on investigation files and SYNTHESIS.md.

**Source:** `pkg/spawn/context.go:33-72`

**Significance:** The Surface Before Circumvent guidance must provide two paths:
1. Tracked: `bd comment <beads-id> "CONSTRAINT: [what] - [why]"` + wait for acknowledgment
2. Untracked: Document in investigation file + include in SYNTHESIS.md

---

### Finding 3: The principle extends existing patterns

**Evidence:** From `kb context "surface before circumvent"`:
- Decision: "Surface Before Circumvent: Before working around a constraint, surface it to the people involved"
- Reason: "The accountability is a feature, not a cost. Extends Pressure Over Compensation from systems to relationships."

**Source:** `~/.kb/principles.md:280-309` (Pressure Over Compensation), `kn decide` output

**Significance:** The implementation should:
1. Make the constraint surfacing action explicit (command or documentation)
2. Require acknowledgment before proceeding (for tracked spawns)
3. Explain WHY (prevents system learning, bypasses stakeholders, creates hidden debt)

---

## Synthesis

**Key Insights:**

1. **Escalation pattern** - Surface Before Circumvent fits naturally as an escalation trigger, alongside scope ambiguity and architectural decisions.

2. **Two-path handling** - The existing NoTrack conditional pattern provides the template for handling both tracked (bd comment) and untracked (investigation file) scenarios.

3. **Why matters** - Including the explanation of consequences (system learning, stakeholder bypass, hidden debt) helps agents internalize the principle rather than just following a rule.

**Answer to Investigation Question:**

The mechanism is implemented by adding a "Surface Before Circumvent" subsection to the AUTHORITY block in the SPAWN_CONTEXT.md template. For tracked spawns, agents use `bd comment <beads-id> "CONSTRAINT: [what] - [why]"` and wait for orchestrator acknowledgment. For untracked spawns, agents document constraints in their investigation file and SYNTHESIS.md.

---

## Structured Uncertainty

**What's tested:**

- ✅ Template compiles correctly with Go template conditionals (verified: all spawn tests pass)
- ✅ Tracked spawns include `bd comment` CONSTRAINT instruction (verified: TestGenerateContext_SurfaceBeforeCircumvent)
- ✅ Untracked spawns use investigation file fallback (verified: TestGenerateContext_SurfaceBeforeCircumvent)

**What's untested:**

- ⚠️ Agent compliance with the new guidance (not tested: would require spawning an agent and observing behavior)
- ⚠️ Orchestrator workflow for handling CONSTRAINT comments (not tested: orchestrator-side not modified)

**What would change this:**

- If agents ignore the guidance → would need stronger gating (hooks instead of documentation)
- If CONSTRAINT comments create noise → might need filtering or categorization

---

## Implementation Details

**What was implemented:**

1. Added "Surface Before Circumvent" section to AUTHORITY block in `pkg/spawn/context.go:101-117`
2. Used Go template conditionals to handle tracked vs untracked spawns
3. Added tests in `pkg/spawn/context_test.go` for both scenarios

**Files Modified:**
- `pkg/spawn/context.go` - Added Surface Before Circumvent section to template
- `pkg/spawn/context_test.go` - Added TestGenerateContext_SurfaceBeforeCircumvent tests

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - Main template and generation logic
- `pkg/spawn/context_test.go` - Existing test patterns
- `~/.kb/principles.md` - Pressure Over Compensation principle

**Commands Run:**
```bash
# Test template generation
go test ./pkg/spawn/... -v -run TestGenerateContext

# Verify output for tracked spawn
go run /tmp/test_template.go | grep -A 30 "Surface Before Circumvent"

# Verify output for untracked spawn
go run /tmp/test_template_notrack.go | grep -A 20 "Surface Before Circumvent"
```

**Related Artifacts:**
- **Decision:** kn entry "Surface Before Circumvent" - The principle being implemented
- **Investigation:** N/A - This is an implementation task, not an investigation
