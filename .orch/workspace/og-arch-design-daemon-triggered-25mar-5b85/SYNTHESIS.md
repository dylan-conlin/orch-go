# Session Synthesis

**Agent:** og-arch-design-daemon-triggered-25mar-5b85
**Issue:** orch-go-exgxp
**Duration:** 2026-03-25
**Outcome:** success

---

## TLDR

Designed a daemon-triggered headless orchestrator for automatic completion review. The core insight: the headless orchestrator's job is specifically Phase 3 of the existing completion pipeline (advisories — discovered work, explain-back, brief writing), externalized to run without stdin/stdout. Recommends on-demand session resume over persistent pipe, a two-state comprehension label model, and the same orchestrator skill with Norm 2 ("translate, don't forward") as the quality bar.

---

## Plain-Language Summary

When agents finish work at 2am, their completions sit in a queue until Dylan starts a conversation the next morning. This design removes that bottleneck. The daemon detects completion, wakes a headless orchestrator via Claude Code's `--resume` flag, and the orchestrator reads the agent's synthesis, verifies it against actual source code, writes a brief for Dylan's reading queue, and files follow-up work. Dylan still reads the brief and confirms comprehension — the system shifts from synchronous (conversation-gated) to asynchronous (reading-gated) review. The key constraint: the headless orchestrator runs the same skill as the live orchestrator. If it can't produce good synthesis alone, that's a skill quality problem, not a mode to work around.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-25-inv-design-daemon-triggered-headless-orchestrator.md` — Full architect investigation with 6 findings, 3 blocking questions, implementation recommendations

### Beads Created
- `orch-go-e9zpl` — Question: Session lifecycle fork (pipe vs resume)
- `orch-go-fc1xv` — Question: Explain-back gate fork (bypass vs produce)
- `orch-go-y9ey6` — Question: Comprehension interaction fork (one-state vs two-state)

---

## Evidence (What Was Observed)

- Current completion pipeline in `complete_pipeline.go` cleanly separates mechanical (Phases 1,2,4) from judgmental (Phase 3 advisories)
- Daemon's `ProcessCompletion()` already handles Phases 1,2,4 — headless orchestrator only needs Phase 3
- Stream-JSON pipe has catastrophic failure mode (pipe break loses all accumulated context) vs resume (each wakeup independent)
- DFM engine session (orch-go-hjllu) proves fresh context can produce better comprehension than accumulated context — supports stateless resume model
- Explain-back gate (`pkg/orch/completion.go`) currently verifies HUMAN comprehension — headless changes its meaning
- Comprehension queue has single state (`pending`) that conflates "processed" with "unread by human"
- Two briefs already produced and quality confirmed per comprehension artifacts thread (2026-03-24)

---

## Architectural Choices

### On-demand resume over persistent pipe
- **What I chose:** Stateless per-wakeup via `claude -p --resume <id>`
- **What I rejected:** Persistent stream-JSON stdin pipe (warm context)
- **Why:** Fire-and-forget matches daemon model; pipe break is catastrophic; DFM evidence shows fresh perspective beats accumulated context; natural parallelism for multiple completions
- **Risk accepted:** Cold-start overhead per completion; no cross-completion context accumulation

### Two-state comprehension labels over single state
- **What I chose:** `comprehension:processed` + `comprehension:unread` replacing single `comprehension:pending`
- **What I rejected:** Using existing `comprehension:pending` as "headless reviewed"
- **Why:** Conflating "AI reviewed" with "human comprehended" creates Defect Class 5 (Contradictory Authority) — two sources of truth about comprehension status
- **Risk accepted:** More label management complexity; daemon throttle logic needs update

### Same orchestrator skill, not dedicated headless skill
- **What I chose:** Headless orchestrator uses identical skill as live orchestrator
- **What I rejected:** Purpose-built synthesis agent or stripped-down headless skill
- **Why:** Spawn context constraint; skill quality should be mode-independent; maintaining two skills doubles drift risk
- **Risk accepted:** Orchestrator skill may be too heavy for headless context window; quality may differ without human feedback

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` — key outcomes:
- Investigation complete with 6 findings and synthesis
- 3 blocking questions created as beads entities
- Implementation sequence defined (3 phases)
- Defect class exposure analyzed

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Claude Code plan mode decision (2026-02-26) is direct precedent: interactive gates are fundamentally incompatible with headless operation
- The completion pipeline's four-phase architecture makes headless possible by isolating all judgment in Phase 3
- Context window is a real risk: orchestrator skill (~1251 lines) + completion context + source files may exceed limits

### Decisions Made
- Recommend on-demand resume (not persistent pipe) for session lifecycle
- Recommend two-state comprehension labels (not single state)
- Recommend same skill constraint (not dedicated headless skill)

---

## Next (What Should Happen)

**Recommendation:** escalate (3 blocking questions need Dylan's judgment)

### If Escalate
**Questions:**
1. orch-go-e9zpl: Session lifecycle — pipe vs resume? (Recommendation: resume)
2. orch-go-fc1xv: Explain-back gate — bypass vs produce? (Recommendation: produce with source verification)
3. orch-go-y9ey6: Comprehension interaction — single vs two-state? (Recommendation: two-state)

After Dylan answers, implementation can proceed in 3 phases:
1. `orch complete --headless` mode
2. Daemon integration (resume wakeup)
3. Two-state comprehension labels + brief quality feedback

---

## Unexplored Questions

- What's the actual context window cost of orchestrator skill + completion context + source files?
- Can session resume load the orchestrator skill reliably? (Needs practical test)
- What's the right quality threshold for brief self-check before publishing?
- Should the headless orchestrator attempt thread connection, or leave that for live conversation?
- How does this interact with cross-project completions (different project dirs)?

---

## Friction

Friction: none — smooth research session with comprehensive prior art available

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-design-daemon-triggered-25mar-5b85/`
**Investigation:** `.kb/investigations/2026-03-25-inv-design-daemon-triggered-headless-orchestrator.md`
**Beads:** `bd show orch-go-exgxp`
