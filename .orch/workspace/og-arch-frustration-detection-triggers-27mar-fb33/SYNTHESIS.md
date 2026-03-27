# Session Synthesis

**Agent:** og-arch-frustration-detection-triggers-27mar-fb33
**Issue:** orch-go-u5x4g
**Duration:** 2026-03-27 15:09 → 2026-03-27 ~15:50
**Outcome:** success

---

## Plain-Language Summary

When a conversation goes bad — circling, stuck, or just fighting you — saying "let me stop" doesn't actually stop it. The model's attention patterns are baked into the conversation history, so a mid-conversation correction competes against the frame's gravity and loses. Dylan naturally does the right thing: close the session, start a new one. This design makes the system do that automatically.

The design has two tracks. For interactive sessions (Dylan + Claude Code), a UserPromptSubmit hook watches for frustration signals in what Dylan types and proposes a boundary: "This conversation is fighting you. Want to save the question and start fresh?" For headless workers, the coaching plugin detects when multiple behavioral signals co-occur (thrashing + stuck + contradicting prior work) and reports `Phase: Boundary`, which the daemon picks up and respawns with only the original question plus a "what didn't work" diagnosis. In both cases, the key principle is: carry forward the QUESTION, not the CONVERSATION.

## TLDR

Designed a two-track frustration boundary system that detects when conversations are fighting the user/agent and triggers session restarts that carry forward the question (not the conversation). Extended the orchestrator-session-lifecycle model with a 4th session boundary type and failure mode #14 (Cognitive Mode Lock-In).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-27-design-frustration-detection-session-boundary.md` - Full architect investigation with 3 forks, 3 questions, two-track design, 3 recommendations
- `.kb/models/orchestrator-session-lifecycle/probes/2026-03-27-probe-frustration-detection-session-boundary-design.md` - Probe testing 3 model claims

### Files Modified
- `.kb/models/orchestrator-session-lifecycle/model.md` - Added 4th session boundary type, failure mode #14, interaction patterns, probe reference

---

## Evidence (What Was Observed)

- Coaching plugin's fundamental constraint (can't see LLM text) means behavioral proxies are the only agent-side detection mechanism — confirmed by model invariant #1
- UserPromptSubmit hooks CAN see user text — verified in comprehension-queue-count.sh implementation
- Session resume protocol already handles artifact discovery via `.orch/session/{window}/latest/` — frustration boundary artifacts slot into this path naturally
- Single coaching signals have a 72% noise rate (action_ratio/analysis_paralysis evidence) — compound signals (2+ co-occurring) needed for reliable frustration detection
- The "framing is stronger than instructions" claim from the model directly explains WHY session boundaries work: new session = new frame = genuine cognitive reset

---

## Architectural Choices

### Two-track design (interactive vs headless) instead of unified approach
- **What I chose:** Different mechanisms for interactive (hook-based, user decides) vs headless (coaching plugin + daemon, automatic)
- **What I rejected:** Single unified mechanism for all session types
- **Why:** Interactive sessions need user consent (Dylan decides when to restart). Headless workers are daemon-managed — the daemon can respawn without asking. These have fundamentally different control planes.
- **Risk accepted:** Two codepaths to maintain instead of one

### Compound signal detection instead of new single signals
- **What I chose:** Trigger frustration boundary when 2+ existing coaching signals co-occur
- **What I rejected:** Adding new dedicated frustration detection patterns
- **Why:** Single signals have a 72% noise rate (evidence from action_ratio/analysis_paralysis removal). Co-occurrence dramatically reduces false positives without new detection infrastructure.
- **Risk accepted:** May miss frustration cases that manifest as only one signal

### FRUSTRATION_BOUNDARY.md as new artifact type instead of reusing SESSION_HANDOFF.md
- **What I chose:** New artifact type purpose-built for failure-path handoffs
- **What I rejected:** Extending SESSION_HANDOFF.md with a "frustration" mode
- **Why:** SESSION_HANDOFF.md is success-path: "what I accomplished, what's next." Frustration boundary is failure-path: "what I was trying to answer, what didn't work." These serve different cognitive needs for the receiving session.
- **Risk accepted:** Another artifact template to maintain

---

## Knowledge (What Was Learned)

### Model Updates
- orchestrator-session-lifecycle model extended: 4th session boundary type, failure mode #14, 3 interaction patterns

### Constraints Discovered
- Mid-session cognitive mode cannot be reset by prompts — this is structural (frame gravity), not a skill/instruction problem
- Frustration detection has two completely separate domains (agent behavior vs user text) with no overlap in infrastructure
- The new session's agent should diagnose the old session's failure (not the old session itself) — follows the model's claim that frame collapse can't be self-diagnosed

---

## Next (What Should Happen)

**Recommendation:** close (design complete, implementation is separate)

### Implementation Issues Needed
1. **Interactive hook** (`.claude/hooks/frustration-boundary.sh`) — highest priority, ~2-4h
2. **Headless track** (coaching plugin compound signal + daemon Phase: Boundary) — medium priority, ~4-6h
3. **Template** (`FRUSTRATION_BOUNDARY.md`) — blocks both tracks, ~30 min

### If Spawn Follow-up
**Issue:** Implement frustration boundary interactive hook
**Skill:** feature-impl
**Context:**
```
Design at .kb/investigations/2026-03-27-design-frustration-detection-session-boundary.md
Track 1 (interactive): UserPromptSubmit hook analyzing user text for frustration signals.
Template at .orch/templates/FRUSTRATION_BOUNDARY.md needed first.
```

---

## Unexplored Questions

- Could frustration boundaries feed back into the five-element product surface's resistance metric? (e.g., count of frustration boundaries per week as a system health signal)
- Should frustration boundary events be written to coaching-metrics.jsonl for dashboard visualization?
- The design assumes frustration is always bad — but some productive sessions feel frustrating (just-hard vs bolted-on resistance distinction from the resistance thread). Can the detection distinguish?

---

## Friction

Friction: none

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for deliverables and manual verification steps.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-frustration-detection-triggers-27mar-fb33/`
**Investigation:** `.kb/investigations/2026-03-27-design-frustration-detection-session-boundary.md`
**Beads:** `bd show orch-go-u5x4g`
