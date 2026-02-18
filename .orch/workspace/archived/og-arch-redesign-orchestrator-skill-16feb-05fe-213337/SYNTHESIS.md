# Session Synthesis

**Agent:** og-arch-redesign-orchestrator-skill-16feb-05fe
**Issue:** orch-go-1003
**Duration:** 2026-02-16
**Outcome:** success

---

## Plain-Language Summary

The orchestrator skill tells the AI orchestrator *what to do* (comprehend, triage, synthesize) but doesn't organize around *why it exists*: keeping Dylan oriented. When an agent finishes work, the orchestrator reconstructs "what happened" from the agent's SYNTHESIS.md — but by then, Dylan may have forgotten why he cared about this work in the first place. The redesign proposes: (1) capture "why Dylan cares" at spawn time as a FRAME annotation, (2) reorganize the skill's sections around four orientation moments (spawn, during work, completion, session boundaries) instead of orchestrator actions, and (3) restructure completion review as a three-layer reconnection (frame reconnection → resolution → contextual placement) before the behavioral verification gate. All five load-bearing hard constraints are preserved. The change is structural (reorder sections, add frame capture), not a rewrite.

## Verification Contract

See: `.kb/investigations/2026-02-16-design-orchestrator-skill-orientation-redesign.md`

Key outcomes:
- Proposed 7-section structure with constraint mapping table
- Three-layer completion review language (ready to implement)
- Frame capture mechanism (SPAWN_CONTEXT.md + beads comment)
- 2 blocking questions for Dylan (identity statement, implementation friction)

---

## TLDR

Designed a reorganization of the orchestrator skill from action-based (COMPREHEND/TRIAGE/SYNTHESIZE) to orientation-based (spawn/during/completion/boundaries). The core addition is capturing "why Dylan cares" at spawn time and using it to reconnect him at completion. All hard constraints preserved. Decision document ready for review.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-16-design-orchestrator-skill-orientation-redesign.md` - Full design investigation with proposed structure, constraint mapping, completion review rewrite, and frame capture mechanism
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-16-orchestrator-skill-orientation-redesign.md` - Probe confirming model claims and extending with orientation state dimension
- `.orch/workspace/og-arch-redesign-orchestrator-skill-16feb-05fe/SYNTHESIS.md` - This file

### Files Modified
- None (design investigation, no code changes)

---

## Evidence (What Was Observed)

- Current skill puts "Dylan's Reality" at line 464 — past the halfway mark. His orientation needs are treated as context for the orchestrator, not as the organizing principle.
- The four orientation moments are already present but scattered: spawn (153 lines across 3 sections), during (~30 lines), completion (90 lines, best-developed), boundaries (~40 lines across 3 sections).
- All five load-bearing patterns from skill.yaml map cleanly into the four-moment structure without loss.
- The completion review explain-back gate is well-designed but starts from the agent's output (SYNTHESIS.md) rather than from Dylan's original motivation — the frame decay problem.
- The COMPREHEND -> TRIAGE -> SYNTHESIZE frame describes orchestrator behavior correctly but doesn't measure whether Dylan is oriented. An orchestrator can perfectly execute C->T->S while leaving Dylan disoriented.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-16-design-orchestrator-skill-orientation-redesign.md` - Design investigation with full proposed structure

### Decisions Made
- Hybrid reorganization (Option C): four moments as top-level sections, action content preserved within each moment
- Frame capture via both SPAWN_CONTEXT.md and beads comments (Option D): dual-write for agent visibility and orchestrator access
- Three-layer completion augments Gate 1 (Option B): frame reconnection → resolution → contextual placement, then existing behavioral gate preserved as Gate 2

### Constraints Discovered
- The three-layer completion model must still respect the pacing test (light/medium/heavy) — abbreviated for simple completions, expanded for architectural changes
- Frame capture at spawn time adds workflow friction; may need mechanical enforcement if consistently skipped
- Token budget of ~8000 means the reorganization can't add significant new content — it's a restructure, not an expansion

---

## Next (What Should Happen)

**Recommendation:** close (design complete, ready for Dylan review and acceptance)

### If Close
- [x] All deliverables complete (investigation, probe, synthesis)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for orchestrator review

### Implementation Follow-ups (after acceptance)

1. **Phase 1: Skill template rewrite** — Edit `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template` following proposed structure
   - Skill: feature-impl
   - Scope: Restructure sections, add three-layer completion language, add frame capture checklist

2. **Phase 2: SPAWN_CONTEXT template** — Add ORIENTATION_FRAME section to `pkg/spawn/context.go`
   - Skill: feature-impl
   - Scope: Add template section, wire orchestrator's frame annotation into generated context

3. **Phase 3: CLI support** — Add `orch spawn --frame "..."` flag
   - Skill: feature-impl
   - Scope: New flag that writes FRAME to beads comment and SPAWN_CONTEXT

---

## Unexplored Questions

**Questions that emerged during this session:**
- Should the identity statement change from "Strategic Comprehender" to "Orientation Keeper"? (surfaced as blocking question Q1 in investigation)
- How much implementation friction is acceptable for frame capture? (surfaced as blocking question Q2)
- Could the three-layer model be tested informally (orchestrator does it conversationally) before committing to the full skill restructure?

**What remains unclear:**
- Whether the four-moment frame will feel natural to orchestrator agents or if the action-based frame is more directly navigable
- Whether frame capture will be consistently used or become another skipped checklist item

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-redesign-orchestrator-skill-16feb-05fe/`
**Investigation:** `.kb/investigations/2026-02-16-design-orchestrator-skill-orientation-redesign.md`
**Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-02-16-orchestrator-skill-orientation-redesign.md`
**Beads:** `bd show orch-go-1003`
