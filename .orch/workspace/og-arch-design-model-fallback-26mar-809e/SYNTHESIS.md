# Session Synthesis

**Agent:** og-arch-design-model-fallback-26mar-809e
**Issue:** orch-go-2mlhl
**Duration:** 2026-03-26 ~09:56 -> 2026-03-26 ~10:25
**Outcome:** success

---

## Plain-Language Summary

The question was what orch-go should do when a task normally wants Opus, but the Anthropic path is rate-limited or otherwise unavailable.

The answer is a staged fallback, not a single downgrade. First, try to stay on Opus by choosing the healthiest Anthropic account. If Opus itself is exhausted but the Claude path is still healthy, drop to Sonnet on the Claude backend. Only if the Anthropic path is actually unavailable should the system cross providers, and even then GPT-5.4 should be automatic for `feature-impl` only.

The important constraint is that reasoning-heavy skills (`architect`, `investigation`, `systematic-debugging`, `research`, `codebase-audit`) should not silently spill onto GPT-5.4 yet. The current evidence only validates GPT-5.4 as implementation overflow, and orch-go does not yet ingest the `seven_day_opus` signal needed to tell "Opus exhausted" apart from "Anthropic unavailable."

## Verification Contract

See `VERIFICATION_SPEC.yaml` for the artifact checklist and source reads.
Key outcome: investigation complete with a recommended routing order, explicit non-goals, and a follow-up implementation issue (`orch-go-4i9bs`).

---

## TLDR

Designed an Anthropic-first fallback cascade for Opus pressure: alternate Opus account -> Sonnet on Claude backend -> GPT-5.4 for `feature-impl` only -> stop/escalate for reasoning-heavy skills. The design also identified the missing implementation seam: orch-go sees generic Claude headroom but drops the Opus-specific `seven_day_opus` signal that should drive the Sonnet branch.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-26-inv-design-model-fallback-cascade-opus.md` - Design investigation with findings, routing recommendation, and implementation guidance
- `.orch/workspace/og-arch-design-model-fallback-26mar-809e/SYNTHESIS.md` - Session synthesis for orchestrator review
- `.orch/workspace/og-arch-design-model-fallback-26mar-809e/VERIFICATION_SPEC.yaml` - Verification contract for design deliverables
- `.orch/workspace/og-arch-design-model-fallback-26mar-809e/BRIEF.md` - Dylan-facing comprehension brief

### Files Modified
- None (design session, no product code changes)

### Commits
- Pending local commit for session artifacts

---

## Evidence (What Was Observed)

- `pkg/daemon/skill_inference.go:271` hardcodes `opus` only for reasoning-heavy skills; `feature-impl` is not pinned and falls through to default model selection
- `pkg/model/model.go:91` sets Sonnet as the default model, which means the non-pinned lane already has a lower-cost Claude fallback
- `pkg/spawn/resolve.go:537` already implements a same-provider account heuristic based on 5-hour and 7-day headroom
- `pkg/account/capacity.go:74` receives `seven_day_opus` from Anthropic, but `pkg/account/capacity.go:225` never stores it in `CapacityInfo`
- `cmd/orch/serve_system.go:29` declares `weekly_opus_percent` output fields, but the handler only fills generic weekly data
- `.kb/threads/2026-03-24-openclaw-migration-from-claude-code.md:78` promotes GPT-5.4 to `feature-impl` overflow only, and `.kb/threads/2026-03-24-openclaw-migration-from-claude-code.md:80` explicitly keeps reasoning-heavy work on Opus

### Tests Run
```bash
# No code changes - design session
# Verification was structural: source reads, model/thread reconciliation, and issue creation
```

---

## Architectural Choices

### Preserve Anthropic before crossing providers
- **What I chose:** `alternate Opus account -> Sonnet -> GPT-5.4 feature-impl only`
- **What I rejected:** Jumping directly from Opus pressure to a cross-provider fallback
- **Why:** Current evidence validates GPT-5.4 only for implementation overflow, while Anthropic routing and Sonnet behavior are already first-class paths
- **Risk accepted:** Some reasoning-heavy work will still stop instead of auto-fallbacking

### Treat Opus exhaustion as different from Anthropic failure
- **What I chose:** Require Opus-specific telemetry (`seven_day_opus`) before deciding the Sonnet branch
- **What I rejected:** Using generic weekly Claude headroom as the only signal
- **Why:** Generic Claude capacity cannot distinguish "Opus exhausted, Sonnet still okay" from "Anthropic path unhealthy"
- **Risk accepted:** Slightly more plumbing work across account/cache/UI surfaces

### Skill-scoped OpenAI fallback
- **What I chose:** Allow automatic GPT-5.4 fallback only for `feature-impl`
- **What I rejected:** Universal GPT-5.4 fallback for architect/investigation/debugging too
- **Why:** The evidence base in the Mar 26 thread explicitly stops short of blessing reasoning-heavy fallback
- **Risk accepted:** Operators may need a manual decision for blocked reasoning-heavy tasks

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-design-model-fallback-cascade-opus.md` - Durable design investigation for fallback routing

### Decisions Made
- Decision 1: Preserve provider/backend before degrading capability
- Decision 2: `seven_day_opus` is the missing signal for a correct Sonnet branch
- Decision 3: GPT-5.4 auto-fallback stays implementation-only until reasoning-heavy benchmarks exist

### Constraints Discovered
- Reasoning-heavy skills are still explicitly Opus-pinned in daemon routing
- Current Anthropic capacity plumbing discards Opus-specific weekly telemetry
- Operator-facing model-selection guidance still contains stale Gemini-secondary advice

### Externalized via `kb quick`
- `kb quick decide "When Opus is rate-limited, preserve Anthropic first..."` - Captured the routing recommendation in the knowledge system

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** `orch-go-4i9bs` - Implement Opus rate-limit fallback cascade
**Skill:** `feature-impl`
**Context:**
```text
Implement the policy from .kb/investigations/2026-03-26-inv-design-model-fallback-cascade-opus.md.
Plumb seven_day_opus through capacity/cache/UI surfaces, add one canonical fallback
decision function, and enforce: alternate Opus account -> Sonnet -> GPT-5.4 for
feature-impl only -> stop/escalate for reasoning-heavy skills.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How often does `seven_day_opus` differ materially from generic weekly Claude capacity in real usage?
- Should the alternate-account branch remain in the default policy now that the system is provisioned around a single Anthropic subscription path?

**Areas worth exploring further:**
- GPT-5.4 benchmark focused only on architect/investigation/systematic-debugging
- Better operator messaging for why a fallback branch fired

**What remains unclear:**
- Whether Sonnet should automatically absorb every Opus miss or only explicit Opus-capacity failures

---

## Friction

No friction - smooth session.

---

## Session Metadata

**Skill:** architect
**Model:** openai/gpt-5.4
**Workspace:** `.orch/workspace/og-arch-design-model-fallback-26mar-809e/`
**Investigation:** `.kb/investigations/2026-03-26-inv-design-model-fallback-cascade-opus.md`
**Beads:** `bd show orch-go-2mlhl`
