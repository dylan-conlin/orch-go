# Investigation: Frustration Detection Triggers Session Boundary, Not Prompt

**Date:** 2026-03-27
**Skill:** architect
**Issue:** orch-go-u5x4g
**Model:** orchestrator-session-lifecycle, coaching-plugin
**Status:** Complete

---

## Design Question

When frustration is detected — frame-collapse signal, repeated frustration keywords, or explicit user signal — the system should trigger a session boundary, not a better prompt. How should this work within the existing orch-go infrastructure?

---

## Problem Framing

### The Root Cause

Evidence from a real session: orchestrator detected its own frustration ("You're right. Let me stop"), then produced more of the same analysis. Dylan exported the session, closed it, and started a new one — manually doing what the system should do automatically.

The orchestrator-session-lifecycle model explains why: **framing is stronger than instructions** (model.md line ~193). Once a conversation's attention patterns are established (the "frame"), in-conversation instructions compete against the frame's gravity. The frame defines what the model sees as relevant. "Let me stop" is an instruction competing against an established frame — the frame wins.

A new session = new frame = genuine cognitive reset. This is why Dylan's manual action (close session, start fresh) works but mid-conversation "let me stop" doesn't.

### Success Criteria

1. System detects when a conversation is fighting the user/agent (resistance signal)
2. System proposes or triggers a session boundary (not a prompt injection)
3. The QUESTION carries forward to the new session (not the conversation)
4. The old session's failure mode becomes context for the new one
5. Works within Claude Code session model constraints

### Constraints

- Claude Code CLI sessions cannot be programmatically restarted (no API for that)
- OpenCode headless sessions CAN be managed via API (create new, send to)
- Coaching plugin cannot see LLM response text or user text (fundamental invariant)
- UserPromptSubmit hooks CAN see user text (Claude Code only)
- Must connect to five-element product surface (resistance is the fifth element)
- "No local agent state" constraint — cannot build a frustration registry

### Scope

- IN: Detection signals, boundary mechanism, carryforward artifact design, per-backend implementation path
- OUT: Implementation code, coaching plugin modifications, dashboard UI

---

## Exploration

### Fork 1: Where Do Frustration Signals Come From?

**Substrate consulted:**
- Coaching plugin model: "Plugins cannot see LLM response text — only tool calls visible" (invariant #1)
- Orchestrator session lifecycle model: "Frame collapse detection is external, not self-diagnosed" (failure mode #1)
- Five-element product surface: Resistance = "this conversation is fighting you"

**Two detection domains exist with different infrastructure:**

| Domain | Infrastructure | Can See | Best For |
|--------|---------------|---------|----------|
| Agent behavior | Coaching plugin (`tool.execute.after`) | Tool calls, timing, phase | Worker sessions, spawned orchestrators |
| User text | UserPromptSubmit hooks | What user types | Interactive Claude Code sessions |

**Agent behavioral signals (extend coaching plugin):**

| Signal | What It Means | Detection | Threshold |
|--------|--------------|-----------|-----------|
| `stall_with_activity` | Tokens increasing but phase stuck | High token delta + same phase >15 min | Token velocity >5K + phase unchanged 15 min |
| `approach_oscillation` | Trying, undoing, trying different approach | Edit/write patterns that reverse previous edits | 3+ reversals on same file region |
| `escalating_tool_failure` | Each attempt fails worse | Consecutive tool errors with increasing scope | 5+ consecutive failures with broadening patterns |
| `circular_questioning` | Same question asked in different forms | Read/Grep patterns revisiting same files | 3+ reads of same file without intervening edits |

**User text signals (Claude Code hooks):**

| Signal | Pattern | Examples |
|--------|---------|---------|
| `explicit_frustration` | Frustration keywords in user text | "this isn't working", "let's try something completely different", "we keep going in circles" |
| `repeated_correction` | User corrects same behavior 3+ times | "no, not that", "I already said...", "again, the problem is..." |
| `session_abandon_signal` | User signals intent to restart | "let me just start over", "forget all of that", "new approach" |

**Recommendation:** Implement BOTH domains. Agent behavioral signals work for headless workers. User text signals work for interactive sessions where Dylan is present. Together they cover the full session type matrix.

### Fork 2: What Happens When Frustration Is Detected?

**Three options evaluated:**

**Option A: Injection (prompt-level)**
Inject "stop and reflect" message into the session.
- Pro: Simple, uses existing `client.session.prompt({ noReply: true })` infrastructure
- Con: **This is exactly what the task says doesn't work.** The attention patterns are baked in. Injection is another prompt competing against the frame. The evidence shows "You're right. Let me stop" followed by the same behavior.
- Verdict: **Rejected.** Violates the core insight of this design.

**Option B: Hard boundary (automatic restart)**
Automatically save state, kill session, respawn with question-only context.
- Pro: Actually resets the cognitive frame. Addresses root cause.
- Con: Disruptive. May kill productive sessions that just hit a rough patch. No human judgment involved.
- Verdict: **Partially accepted.** Appropriate for headless workers where daemon manages lifecycle. Not appropriate for interactive sessions where Dylan should decide.

**Option C: Boundary proposal (signal + suggest)**
Detect frustration, surface the signal, propose a boundary.
- For interactive sessions: inject a boundary proposal that includes what to carry forward, user decides
- For headless workers: report `Phase: Boundary` via beads, daemon decides whether to respawn
- Pro: Respects agency. Different automation levels for different session types.
- Con: Proposal may be ignored (like checkpoint warnings). But: the five-element product surface ALREADY handles this — resistance is surfaced, not enforced.
- Verdict: **Accepted.** Matches the product surface philosophy (surface signals, don't enforce).

**Recommendation:** Option C for interactive sessions. Option B for headless workers (daemon-managed). This mirrors how checkpoint discipline works: warnings for interactive, automation for managed.

**Principle cited:** "Perspective is structural" (principles.md) — the frustration signal needs to reach the right vantage point. For workers, that's the daemon. For interactive sessions, that's Dylan.

### Fork 3: How to Carry Forward Question Without Conversation?

**Substrate consulted:**
- DEKN structure (Delta/Evidence/Knowledge/Next) — designed for success-path handoffs
- SESSION_HANDOFF.md — carries everything, no filtering
- SPAWN_CONTEXT.md — already demonstrates "carry context into new session" pattern

**The distinction:**

SESSION_HANDOFF.md answers: "What did I accomplish? What's next?"
FRUSTRATION_BOUNDARY answers: "What was I trying to answer? Why couldn't I?"

These are structurally different. A frustration boundary is a **failure-path handoff** — it carries the question and the diagnosis, not the work and the progress.

**Proposed artifact: FRUSTRATION_BOUNDARY.md**

```markdown
# Frustration Boundary

**Trigger:** [signal that triggered boundary]
**Session:** [session identifier]
**Duration before boundary:** [time]

## The Question
[What were we actually trying to answer? Extracted from spawn context or conversation.]

## What Was Tried
[Approaches attempted, in order. Brief — one line each.]

## Why It Didn't Work
[Diagnosis: circular reasoning? Wrong premise? Frame collapse? Scope too large?]

## Suggested Fresh Angle
[What should the next attempt try differently?]

## Do Not Repeat
[Specific approaches or framings that led to the failure mode.]
```

**Key design decision:** This artifact is SMALL. The whole point is to NOT carry forward the conversation. The question is typically 1-3 sentences. The approaches list is 3-5 bullet points. The diagnosis is 1-2 sentences. Total: <200 words.

**Recommendation:** New artifact type alongside SESSION_HANDOFF.md and SYNTHESIS.md. Discovered by session resume protocol via the same `.orch/session/{window}/latest/` path.

---

## Question Generation

### Q1: Should frustration boundary be a Phase value or a separate mechanism?

**Authority:** architectural
**Subtype:** judgment

The daemon polls for `Phase: Complete`. Adding `Phase: Boundary` would let daemon detect frustration boundaries through the existing polling mechanism. But Phase is a progress indicator, not a quality signal — mixing them may cause confusion.

**Alternative:** Use a label (`frustration:boundary`) instead of a Phase value. Daemon already handles labels for comprehension routing.

**What changes based on answer:** If Phase: the daemon completion pipeline parses a new Phase value. If label: the daemon needs a new label-based polling path.

**Recommendation:** Use `Phase: Boundary - frustration` as a Phase value. The daemon already has Phase parsing infrastructure. The summary after the dash provides context. This is consistent with `Phase: BLOCKED` and `Phase: QUESTION` which already extend Phase beyond simple progress states.

### Q2: Who writes FRUSTRATION_BOUNDARY.md for headless workers?

**Authority:** implementation
**Subtype:** factual

Workers report frustration signals via coaching metrics and phase comments. But writing the boundary artifact requires understanding the question and diagnosis — work that the frustrated agent may not do well (since it's in a degraded cognitive state).

**Options:**
- A: Frustrated agent writes it before boundary (may be low quality since agent is in degraded state)
- B: Daemon writes it from external signals (Phase comments, coaching metrics, spawn context)
- C: New session's agent reads the old session's coaching metrics and spawn context to reconstruct

**Recommendation:** Option C. The NEW session should be the one that synthesizes what went wrong. The old session is in a degraded frame — asking it to diagnose itself violates the model's claim that "frame collapse is detected externally, not self-diagnosed." The daemon's job is to detect the boundary and respawn. The new session's job is to understand what happened and chart a different course.

### Q3: How aggressive should automatic boundary detection be?

**Authority:** strategic
**Subtype:** judgment

False positive boundaries (killing a productive session that's just working through difficulty) are worse than false negatives (letting a frustrated session continue). The checkpoint discipline model explicitly chose warnings over hard blocks to respect agent/user judgment.

**What changes based on answer:** Threshold tuning for all detection signals. Conservative = fewer boundaries, more false negatives. Aggressive = more boundaries, more false positives.

**Recommendation:** Conservative for interactive sessions (surface signal, don't enforce). Moderate for headless workers (daemon can respawn cheaply; the cost of a false positive is one respawn, not disruption to Dylan's flow).

---

## Synthesis

### Design: Two-Track Frustration Boundary System

The design splits into two tracks because interactive and headless sessions have fundamentally different control planes:

#### Track 1: Interactive Sessions (Claude Code + Dylan)

**Detection:** UserPromptSubmit hook analyzes user text for frustration signals.

```
User types message
  → UserPromptSubmit hook fires
  → Hook analyzes text for frustration patterns
  → If signal detected:
    → Inject boundary proposal via additionalContext
    → Proposal includes: extracted question, suggested fresh start
  → Dylan decides: continue or restart
  → If restart: hook writes FRUSTRATION_BOUNDARY.md to .orch/session/{window}/
  → On new session start: session resume discovers and injects boundary artifact
```

**Implementation: frustration-boundary-hook.sh**

New UserPromptSubmit hook (alongside comprehension-queue-count.sh) that:
1. Receives user message text from stdin
2. Pattern-matches for frustration signals (keyword list + structural patterns)
3. Tracks signal count in a simple counter file (`.orch/session/{window}/frustration_count`)
4. At threshold (3+ signals in session): returns boundary proposal in additionalContext
5. Proposal includes: "This conversation may be fighting you. The question appears to be: [extracted from recent context]. Want to save this and start fresh?"

**Boundary execution:** When Dylan says "yes, start fresh" or similar:
1. Current session writes FRUSTRATION_BOUNDARY.md (question + what was tried)
2. Dylan exits session (`/exit` or closes)
3. New session starts, session resume protocol discovers FRUSTRATION_BOUNDARY.md
4. New session gets only the question + failure context, not the old conversation

**Key constraint:** The hook cannot force a session restart. It can only surface the signal and propose. This matches the product surface philosophy: resistance is surfaced, not enforced.

#### Track 2: Headless Workers (Daemon-Managed)

**Detection:** Extended coaching plugin behavioral signals + daemon Phase parsing.

```
Worker session runs
  → Coaching plugin detects behavioral frustration signals
  → Plugin writes frustration metrics to coaching-metrics.jsonl
  → At threshold: plugin injects Phase: Boundary via beads comment
  → Daemon polls, detects Phase: Boundary
  → Daemon extracts original question from SPAWN_CONTEXT.md
  → Daemon respawns with frustration-aware SPAWN_CONTEXT:
    - Original question
    - What the previous agent tried (from Phase comments)
    - "Do not repeat" list (from boundary comment)
```

**New coaching plugin pattern: `frustration_compound`**

Instead of adding a single new detection pattern, create a compound signal that fires when multiple existing signals co-occur:

```
frustration_compound triggers when 2+ of:
  - behavioral_variation threshold crossed (thrashing)
  - time_in_phase > 15 min (stuck)
  - circular_pattern detected (contradicting prior work)
  - tool_failure_rate > 3 consecutive (failing)
  - frame_collapse detected (wrong level)
```

Single signals can be false positives. Co-occurrence is a strong indicator. This avoids the noise problem that caused action_ratio and analysis_paralysis to be removed (72% of events were noise from single-signal detection).

**Daemon respawn logic:**

When daemon detects `Phase: Boundary - frustration`:
1. Read original SPAWN_CONTEXT.md from workspace
2. Extract the TASK section (the question)
3. Read Phase comments (what was tried)
4. Create new spawn with augmented context:

```markdown
## PRIOR ATTEMPT (Failed — frustration boundary)

**Original question:** [extracted from previous SPAWN_CONTEXT.md TASK section]

**What was tried:**
- [approach 1 from Phase comments]
- [approach 2]

**Why it failed:** [from Phase: Boundary comment summary]

**Do not repeat:** [specific approaches to avoid]

---

[Normal SPAWN_CONTEXT.md content follows]
```

### Connection to Five-Element Product Surface

| Element | How Frustration Boundary Connects |
|---------|-----------------------------------|
| Threads | Frustration boundary may reveal a thread worth tracking ("we keep circling back to X") |
| Briefs | Failed session produces a brief: what was tried, what didn't work |
| Tensions | The unresolvable question from a frustrated session IS a tension |
| Shape | Frustration boundary changes the work shape: "this isn't execution, it's search" |
| **Resistance** | **Frustration boundary IS the resistance signal made actionable** |

The five-element product surface identifies resistance as a first-class element but currently has no mechanism to detect or surface it. This design provides that mechanism: resistance is detected (via behavioral signals or user text), surfaced (via boundary proposal or Phase: Boundary), and acted upon (via session restart with question carryforward).

### Defect Class Exposure

| Defect Class | Exposure | Mitigation |
|--------------|----------|------------|
| Class 3 (Stale Artifact Accumulation) | FRUSTRATION_BOUNDARY.md files could accumulate | Session resume protocol already manages artifact lifecycle via `latest` symlink |
| Class 5 (Contradictory Authority Signals) | Frustration hook vs checkpoint discipline could give conflicting advice | Frustration boundary subsumes checkpoint — if frustration detected before 2h, boundary takes precedence |
| Class 6 (Duplicate Action) | Daemon could respawn the same frustrated agent repeatedly | Dedup via beads label: `boundary:processed` after first respawn |

---

## Recommendations

### Recommendation 1: Implement Interactive Track (UserPromptSubmit Hook)

**Priority:** High — this is where Dylan directly feels the pain.

**Deliverables:**
- New hook: `.claude/hooks/frustration-boundary.sh`
- Frustration signal pattern list (configurable keyword list + structural patterns)
- FRUSTRATION_BOUNDARY.md template in `.orch/templates/`
- Session resume protocol extension to discover frustration boundaries

**Effort:** ~2-4 hours implementation

**Acceptance criteria:**
- Hook detects frustration keywords in user text
- At threshold (3+ signals), proposes boundary via additionalContext
- FRUSTRATION_BOUNDARY.md written to `.orch/session/{window}/latest/`
- Session resume discovers and injects boundary on new session start

### Recommendation 2: Implement Headless Track (Coaching Plugin + Daemon Extension)

**Priority:** Medium — worker frustration is less visible but still wastes cycles.

**Deliverables:**
- New coaching pattern: `frustration_compound` (multi-signal co-occurrence)
- Daemon extension: detect `Phase: Boundary` and respawn with augmented context
- SPAWN_CONTEXT.md augmentation for frustration-aware respawns

**Effort:** ~4-6 hours implementation

**Acceptance criteria:**
- Compound signal fires when 2+ existing signals co-occur
- Phase: Boundary comment includes summary of what failed
- Daemon detects boundary and respawns with prior-attempt context
- New spawn context includes "do not repeat" section

### Recommendation 3: Create FRUSTRATION_BOUNDARY.md Template

**Priority:** High — needed by both tracks.

**Deliverables:**
- Template at `.orch/templates/FRUSTRATION_BOUNDARY.md`
- Documentation in session-resume-protocol guide

**Effort:** ~30 minutes

### Implementation Sequence

1. Template (Rec 3) — foundational, blocks both tracks
2. Interactive hook (Rec 1) — highest user-facing impact
3. Headless daemon extension (Rec 2) — lower priority, can follow

---

## Evidence Quality

| Claim | Confidence | Source |
|-------|-----------|--------|
| Framing is stronger than instructions | High | Orchestrator session lifecycle model (40 investigations, 18 probes) |
| Mid-session reframing doesn't work | High | Direct evidence (user exported session, restarted manually) |
| Coaching plugin can't see text | High | Fundamental OpenCode plugin constraint (invariant #1, verified) |
| UserPromptSubmit hooks can see user text | High | Verified in comprehension-queue-count.sh implementation |
| Session resume protocol can discover artifacts | High | Implemented and working (session_resume.go) |
| Compound signals reduce false positives | Medium | Inference from action_ratio/analysis_paralysis noise removal (72% noise rate for single signals) |
| Conservative thresholds preferred | Medium | Inference from checkpoint discipline design philosophy |

---

## References

- Orchestrator session lifecycle model: `.kb/models/orchestrator-session-lifecycle/model.md`
- Coaching plugin model: `.kb/models/coaching-plugin/model.md`
- Five-element product surface: `.kb/threads/2026-03-27-product-surface-five-elements-not.md`
- Session resume protocol: `.kb/guides/session-resume-protocol.md`
- Comprehension queue hook: `.claude/hooks/comprehension-queue-count.sh`
- Coaching plugin source: `plugins/coaching.ts`, `plugins/coaching-types.ts`
- Session resume code: `cmd/orch/session_resume.go`
- Send command: `cmd/orch/send_cmd.go`
- Daemon completion: `pkg/daemon/completion_processing.go`
