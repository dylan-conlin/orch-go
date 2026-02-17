# Probe: Orchestrator Skill Orientation Redesign

**Model:** orchestrator-session-lifecycle
**Date:** 2026-02-16
**Status:** Complete

---

## Question

The orchestrator session lifecycle model claims:
1. Orchestrator sessions operate via COMPREHEND -> TRIAGE -> SYNTHESIZE (not implement)
2. Frame collapse occurs when orchestrators drop levels and do work below their station - detected externally, not self-diagnosed
3. Session boundaries have three distinct patterns (worker, orchestrator, cross-session) each optimized for its context
4. The "completed by level above" invariant governs lifecycle

Does reorganizing the orchestrator skill around "keep Dylan oriented at four moments" (spawn, during work, completion, session boundaries) conflict with, confirm, or extend these claims? Specifically: does the current COMPREHEND -> TRIAGE -> SYNTHESIZE frame serve Dylan's orientation needs, or does it serve the orchestrator's self-conception?

---

## What I Tested

Read and analyzed:
1. Current deployed orchestrator skill (640 lines)
2. SKILL.md.template (614 lines, identical content to deployed)
3. skill.yaml load-bearing patterns (5 patterns)
4. Orchestrator session lifecycle model (5 phases of evolution)
5. Completion verification model (three-gate system, tier-aware)
6. Verifiability-first decision (two-gate verification, explain-back)
7. Verification bottleneck investigation (462 lost commits)
8. Orchestrator value-add investigation (80% daemon / 20% judgment split)
9. Orchestrator skill drift audit (19 drift items)
10. 18% completion rate investigation (by design, not a bug)
11. Principles.md (27 principles)

Cross-referenced each section of current skill against: (a) which of the four orientation moments it serves, (b) whether it's organized for the orchestrator or for Dylan.

---

## What I Observed

### 1. The current skill is organized around orchestrator identity, not Dylan's needs

The top-level sections are:
- Identity: Strategic Comprehender (who the orchestrator IS)
- COMPREHEND / TRIAGE / SYNTHESIZE (what the orchestrator DOES)
- Skill Selection / Spawning (HOW the orchestrator does it)
- Orchestrator Autonomy (HOW to interact)
- Meta-Orchestrator Interface (WHO Dylan is to the orchestrator)

"Dylan's Reality" appears at line 464 — past the halfway mark. His orientation needs are treated as context for the orchestrator, not as the organizing principle.

### 2. The four moments are already present but scattered

| Moment | Currently Lives In | Lines | Notes |
|--------|-------------------|-------|-------|
| Spawn time | Spawning (349-412) + Skill Selection (306-345) + Triage (158-208) | 153 lines across 3 sections | Mechanics-heavy. No "why Dylan cares" capture |
| During work | Session Management (552-581) + monitoring references scattered | ~30 lines | Minimal — mostly "run bd ready and orch status" |
| Completion | SYNTHESIZE (212-302) | 90 lines | Best-developed section. But starts with "the what" not "why you cared" |
| Session boundaries | Session Mgmt (552-581) + Hygiene Checkpoint (200-206) + Meta-Orch Interface (488-494) | ~40 lines across 3 sections | Scattered. Session start protocol is buried in Meta-Orchestrator Interface |

### 3. The completion review section's explain-back gate is well-designed but starts in the wrong place

Current flow: Read SYNTHESIS.md -> Present "the what" -> Ask Dylan to explain "the so-what"

The gap: By completion time, the "why Dylan cares" frame has decayed. The orchestrator reconstructs from SYNTHESIS.md (agent-centric), not from the original motivation (Dylan-centric). The explain-back assumes Dylan still has the context he had at spawn time.

### 4. Frame collapse detection is self-referential

Current: "Warning signs: About to read code? Gathering context >5 min?"

These are observable from inside the frame. But the model says "Orchestrators can't self-diagnose frame collapse. The frame defines what's visible." The detection signals in the skill are instructions-level (read code = investigation), but the model says framing is stronger than instructions. This is a known tension, not a contradiction — the instructions are defense-in-depth, not the primary mechanism.

### 5. The COMPREHEND -> TRIAGE -> SYNTHESIZE frame is correct but incomplete

It describes what the orchestrator does at a high level. But it doesn't answer: "Is Dylan oriented right now?" An orchestrator following this frame perfectly could still leave Dylan disoriented:
- Spawn 3 agents without establishing why Dylan cares about each
- Complete them with technically correct explain-back but no frame reconnection
- End session with hygiene checkpoint that doesn't tell Dylan "here's where we are"

---

## Model Impact

- [x] **Confirms** invariant: "Orchestrators can't self-diagnose frame collapse" — skill attempts detection signals but model correctly notes these are defense-in-depth, not reliable self-diagnosis
- [x] **Confirms** invariant: "Session boundaries have three distinct patterns" — the redesign preserves these patterns (worker → Phase:Complete, orchestrator → SESSION_HANDOFF, cross-session → manual)
- [x] **Extends** model with: The COMPREHEND -> TRIAGE -> SYNTHESIZE frame is correct for orchestrator behavior but doesn't address a fourth dimension: Dylan's orientation state. The model's "three-tier hierarchy" describes information flow but not orientation preservation. A new claim is needed: "Orchestrator effectiveness is measured not by correctly executing COMPREHEND -> TRIAGE -> SYNTHESIZE, but by whether Dylan is oriented at each transition point."

---

## Notes

The redesign doesn't contradict the existing model — it extends it with a dimension the model doesn't currently address: the human's orientation state. The model describes HOW orchestrator sessions work (lifecycle, boundaries, frame collapse). The redesign asks FOR WHOM they work (Dylan's ability to remain oriented).

This is consistent with the model's own evolution: Phase 6 (Jan 7, 2026) shifted from "tactical execution" to "strategic comprehension." The proposed redesign is the next evolution: from "strategic comprehension" (orchestrator-centric) to "orientation preservation" (Dylan-centric).
