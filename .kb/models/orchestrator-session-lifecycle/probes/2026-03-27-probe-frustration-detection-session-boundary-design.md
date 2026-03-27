# Probe: Frustration Detection as Session Boundary — Design Validation

**Model:** orchestrator-session-lifecycle
**Date:** 2026-03-27
**Status:** Complete
**claim:** Session boundaries are the structural fix for cognitive mode lock-in; prompts cannot reset attention patterns
**verdict:** extends

---

## Question

The model documents 13 failure modes but treats session boundaries as static (protocol-driven for workers, state-driven for orchestrators, manual for cross-session). Can frustration/resistance signals trigger dynamic session boundaries? Does the existing infrastructure support carrying forward the QUESTION without the CONVERSATION?

Specifically testing:
1. Model claim that "frame collapse is detected externally, not self-diagnosed" — does this extend to ALL cognitive mode failures?
2. Model claim that "framing is stronger than instructions" — does this imply mid-session reframing is impossible?
3. Model's checkpoint discipline (2h/3h/4h) — is this the right proxy, or should quality degradation signals supplement duration?

---

## What I Tested

### Test 1: Can existing coaching plugin detect frustration-equivalent signals?

Reviewed coaching plugin detection patterns against the five-element product surface's "resistance" concept:

```
Existing patterns that map to frustration:
- frame_collapse: orchestrator editing code (detected via edit/write on code files)
- circular_pattern: contradicting prior investigations (detected via keyword matching)
- behavioral_variation: semantic group thrashing (detected via command classification)
- time_in_phase: stale phase without progress (detected via phase timestamp)

Missing patterns:
- No "repeated correction without resolution" detection
- No user text analysis (plugin can't see LLM/user text — invariant #1)
- No quality degradation trending (each metric evaluated independently)
- No "stall with activity" (high token velocity + stuck phase)
```

### Test 2: Can session resume protocol carry forward question-only context?

Examined session resume path at `.orch/session/{window}/latest/SESSION_HANDOFF.md`:

```
Current handoff carries EVERYTHING: spawns, evidence, knowledge, friction, focus progress, next steps
No mechanism to carry ONLY the question + failure context
Session resume injects full handoff — no filtering/distillation
```

### Test 3: Can hooks detect frustration in user text?

Examined UserPromptSubmit hook infrastructure:

```
Claude Code UserPromptSubmit hooks receive user message text
Current hook (comprehension-queue-count.sh) only counts queue items
Hook CAN analyze user text for frustration signals — mechanism exists but unused
Key: this is the ONE path where user text is visible (coaching plugin can't see it)
```

### Test 4: What mechanisms exist for dynamic session restart?

```
Claude Code: No programmatic restart API. Can suggest /exit + restart.
OpenCode headless: orch send, create new session via API, kill old session
Tmux: kill-pane + new spawn
Daemon: Phase: Complete triggers respawn logic — could extend to Phase: FRUSTRATED
```

---

## What I Observed

### Finding 1: The model's "framing is stronger than instructions" claim directly supports session boundaries

From model.md line ~193: "Framing shapes perception and available actions. Instructions can be overridden by situational reasoning, but framing defines what's visible."

This means: once a conversation's attention patterns are established (the "frame"), in-conversation instructions ("let me stop") compete against the frame's gravity. The frame literally defines what the model sees as relevant. A new session = new frame = genuine cognitive reset.

This is the theoretical foundation for "carry forward the question, not the conversation."

### Finding 2: Two detection domains exist with different infrastructure

**Agent-internal detection** (coaching plugin): Can detect behavioral patterns (tool usage, timing, phase progression) but cannot see text. Best for worker sessions where behavioral proxies are the only signal.

**User-text detection** (hooks): Can see what the user types in Claude Code sessions. Best for interactive sessions where Dylan's words are the frustration signal. This path is currently unused for frustration detection.

**Neither path currently supports:** Quality degradation trending, cross-session pattern memory, or "resistance" as a first-class signal.

### Finding 3: SESSION_HANDOFF.md is the wrong artifact for frustration boundaries

SESSION_HANDOFF.md is designed for success-path handoffs: "here's what I accomplished, here's what's next." A frustration boundary needs the opposite: "here's the question I couldn't crack, here's what I tried that didn't work, here's why."

The DEKN structure (Delta/Evidence/Knowledge/Next) maps to accomplished work. A frustration boundary artifact needs:
- **Question**: What were we actually trying to answer?
- **Approaches tried**: What didn't work?
- **Diagnosis**: Why did this session fail? (Circular reasoning? Frame collapse? Wrong premise?)
- **Fresh angle**: What should the next attempt try differently?

### Finding 4: Phase reporting can be extended for frustration boundaries

The daemon polls for `Phase: Complete` to detect agent completion. The same mechanism could detect `Phase: Boundary - frustration` to trigger respawn with frustration context. Worker sessions already report phase transitions — adding a frustration-specific phase is a natural extension.

### Finding 5: Checkpoint discipline (2h/3h/4h) is a duration proxy when quality signals exist

The model uses session duration as a proxy for context exhaustion. But the coaching plugin already tracks signals that are BETTER proxies for quality degradation:
- behavioral_variation (thrashing)
- circular_pattern (contradicting prior work)
- time_in_phase (stuck)
- frame_collapse (wrong level of work)

Duration is a trailing indicator. These behavioral signals are leading indicators. A frustration boundary could trigger BEFORE the 2h checkpoint if multiple signals co-occur.

---

## Model Impact

- [ ] **Confirms** invariant: Frame collapse detection is external, not self-diagnosed. Extends to ALL cognitive mode failures including mid-session frustration lock-in.
- [ ] **Confirms** invariant: Framing is stronger than instructions. This is WHY session boundaries work and prompts don't — new session = new frame.
- [x] **Extends** model with: Session boundaries should have a FOURTH type: frustration boundary (alongside worker protocol-driven, orchestrator state-driven, and cross-session manual). This type is signal-driven: detected via behavioral signals (agents) or user text analysis (interactive sessions), triggers dynamic session restart with question-only carryforward.

**Specific extensions to model:**
1. Failure Mode #14 candidate: "Cognitive Mode Lock-In" — mid-session attention patterns resist reframing regardless of prompt-level intervention
2. Session Types table needs 4th row: Frustration boundary | Signal-driven | FRUSTRATION_BOUNDARY.md | Question + failure context
3. Checkpoint discipline section should note that behavioral signals are leading indicators that can trigger boundaries before duration thresholds

---

## Notes

This probe is part of an architect session designing the frustration detection → session boundary mechanism. The investigation artifact will contain the full design with implementation recommendations.
