<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** DeepSeek Reasoner is not suitable for orchestration - it frame-collapses into worker mode and violates delegation rules.

**Evidence:** Evaluated full session transcript (session-ses_45b6.md); model read code directly, edited investigation files, spent 20-40s in thinking blocks, never declared operational mode.

**Knowledge:** Orchestration requires meta-level abstraction ("coordinate work without doing work") that DeepSeek Reasoner struggles with; the model is technically capable but lacks delegation discipline.

**Next:** Use DeepSeek Reasoner as worker only; continue using Opus for orchestration via Claude Code.

**Promote to Decision:** Actioned - decision exists (gpt-unsuitable-for-orchestration)

---

# Investigation: DeepSeek Reasoner Orchestrator Evaluation

**Question:** Can DeepSeek Reasoner effectively serve as an orchestrator in the orch system?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** Dylan + Claude (Opus)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Frame Collapse into Worker Mode

**Evidence:**
- Model read code files directly to understand implementation (e.g., `pkg/spawn/claude.go`)
- Edited investigation files that workers should maintain
- Made D.E.K.N. summary edits instead of spawning worker to complete
- Example from transcript line ~2275: `Tool: read {"filePath": "/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/claude.go"...}`

**Source:** Session transcript `/Users/dylanconlin/Documents/personal/orch-go/session-ses_45b6.md`

**Significance:** Violates ABSOLUTE DELEGATION RULE: "If you're about to read code to understand *how it works* → STOP. That's an investigation." Orchestrator doing task work blocks the entire system.

---

### Finding 2: Excessive Thinking Time

**Evidence:**
- 27.1s thinking block at transcript line 2647
- 26.5s thinking block at transcript line 3688
- 25.9s thinking block at transcript line 2004
- Much thinking was about whether to delegate vs do work itself (answer should always be DELEGATE)

**Source:** Session transcript thinking blocks (marked with `_Thinking:_`)

**Significance:** Reasoning model's "thinking" pattern is optimized for solving problems, not for meta-level coordination. The model overthinks delegation decisions that should be automatic.

---

### Finding 3: No Mode Declaration Protocol

**Evidence:**
- Never declared `STRATEGIC:`, `CONTEXT:`, or `⚠️ DIRECT:` as required by orchestrator skill
- Dylan couldn't distinguish context-gathering from frame collapse
- No self-checks before code reading

**Source:** Full transcript review - searched for mode declarations, found none

**Significance:** Mode declaration is a key protocol for human-orchestrator collaboration. Without it, Dylan can't tell if the orchestrator is gathering context (allowed) or collapsed into worker mode (forbidden).

---

### Finding 4: Technical Competence is High

**Evidence:**
- Correctly used orch/beads CLI commands
- Ran parallel tool calls when appropriate
- Understood SESSION_HANDOFF and connected beads issues
- Successfully completed the dual-spawn-mode testing goal
- Properly cleaned up stale agents
- Pushed changes before session end

**Source:** Session transcript - successful task completion

**Significance:** The model CAN do the work - the issue is it shouldn't. Technical competence doesn't compensate for role confusion.

---

### Finding 5: Respected Guardrails When Triggered

**Evidence:**
- Caught by decision record guard when attempting improper edit
- Correctly reverted the edit
- Created follow-up bug issue for discovered JSON warning

**Source:** Transcript lines ~3735-3770 showing guard trigger and revert

**Significance:** External guardrails work; the model responds appropriately to explicit constraints. The issue is it doesn't internalize the orchestrator role constraints.

---

## Synthesis

**Key Insights:**

1. **Role abstraction is the blocker** - DeepSeek Reasoner treats orchestration as "a task to complete" rather than "a role to inhabit." It optimizes for task completion, which leads to doing work rather than coordinating work.

2. **Reasoning models may be anti-patterns for coordination** - The extended thinking that makes R1 good at complex problems works against quick delegation decisions. Orchestration needs fast "this is spawnable → spawn it" reflexes, not deep analysis.

3. **Workers vs Orchestrators need different models** - This supports the emerging architecture: Opus for orchestration (meta-level abstraction), cheaper models (Sonnet, DeepSeek) for worker tasks.

**Answer to Investigation Question:**

No, DeepSeek Reasoner cannot effectively serve as an orchestrator. While technically capable of using the tools and completing work, it fundamentally misunderstands the orchestrator role. It collapses into worker mode, violates delegation rules, and lacks the meta-level abstraction required. The model should be used exclusively as a worker, not an orchestrator.

---

## Structured Uncertainty

**What's tested:**

- ✅ DeepSeek Reasoner completing dual-spawn-mode testing (verified: 6/6 tasks closed, 16 commits pushed)
- ✅ Model using orch/beads CLI correctly (verified: commands executed successfully)
- ✅ Model responding to guardrails (verified: decision record guard triggered and respected)
- ✅ Frame collapse occurring (verified: code reading in transcript)

**What's untested:**

- ⚠️ Whether additional prompting could prevent frame collapse (not tested with modified system prompt)
- ⚠️ DeepSeek Reasoner performance as worker (test in progress with orch-go-4tven.2)
- ⚠️ Other reasoning models (o3, etc.) as orchestrators

**What would change this:**

- If modified system prompt eliminated frame collapse consistently
- If reasoning could be constrained to delegation decisions only
- If a "lite" reasoning mode existed without extended thinking

---

## Implementation Recommendations

### Recommended Approach ⭐

**Model-Role Separation** - Use Opus for orchestration, Sonnet/DeepSeek for workers.

**Why this approach:**
- Opus demonstrates consistent delegation discipline
- Cheaper models work well for bounded tasks
- Dual spawn mode enables this architecture (Claude Code for orchestrator, OpenCode for workers)

**Trade-offs accepted:**
- Orchestration requires Claude Max subscription ($100/month)
- Can't fully automate orchestration with cheap models

**Implementation sequence:**
1. Continue using Claude Code + Opus for orchestration (current)
2. Test Sonnet and DeepSeek as workers (in progress)
3. Document model selection guidance in CLAUDE.md or decision record

### Alternative Approaches Considered

**Option B: Train DeepSeek with stronger delegation prompts**
- **Pros:** Could enable cheaper orchestration
- **Cons:** Fundamental model architecture may not support role abstraction
- **When to use instead:** If API costs become prohibitive

**Option C: Hybrid orchestration with human checkpoints**
- **Pros:** Catch frame collapse before damage
- **Cons:** Defeats purpose of autonomous orchestration
- **When to use instead:** For critical/irreversible operations

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/session-ses_45b6.md` - Full session transcript of DeepSeek Reasoner orchestration attempt

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-09-dual-spawn-mode-architecture.md` - Enables the worker architecture being tested
- **Handoff:** `.orch/workspace/SESSION_HANDOFF_2026-01-09.md` - Original suggestion to test DeepSeek as orchestrator

---

## Investigation History

**2026-01-09 21:30:** Investigation started
- Initial question: Can DeepSeek Reasoner serve as orchestrator?
- Context: Testing Option A from session handoff after Opus OAuth gate

**2026-01-09 21:45:** Evaluation completed
- Reviewed full session transcript with Dylan
- Identified 5 key findings across delegation, thinking, and protocol adherence

**2026-01-09 21:50:** Investigation completed
- Status: Complete
- Key outcome: DeepSeek Reasoner unsuitable for orchestration due to frame collapse; use as worker only
