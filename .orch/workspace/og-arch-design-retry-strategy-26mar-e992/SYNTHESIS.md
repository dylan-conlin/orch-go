# Session Synthesis

**Agent:** og-arch-design-retry-strategy-26mar-e992
**Issue:** orch-go-7iw5a
**Duration:** 2026-03-26T09:55:22-07:00 -> 2026-03-26T10:26:00-07:00
**Outcome:** success

---

## Plain-Language Summary

I looked into whether orch should automatically retry GPT-5.4 sessions that "die silently" after accepting work but producing no tokens. The answer is yes, but only in a narrow OpenCode-specific case: orch should classify an empty execution using multiple signals, retry it once, and escalate if it happens again so we do not hide real failures or create duplicate work.

---

## TLDR

This session turned a vague retry question into an implementation-ready design. I wrote the investigation, created a phased plan, and opened four follow-up issues covering the classifier, retry handoff, observability, and end-to-end proof.

---

## Delta (What Changed)

### Files Created
- `.orch/workspace/og-arch-design-retry-strategy-26mar-e992/VERIFICATION_SPEC.yaml` - structural verification contract for this design session
- `.orch/workspace/og-arch-design-retry-strategy-26mar-e992/SYNTHESIS.md` - orchestration synthesis artifact
- `.orch/workspace/og-arch-design-retry-strategy-26mar-e992/BRIEF.md` - Dylan-facing comprehension brief
- `.kb/plans/2026-03-26-gpt54-empty-execution-retry.md` - phased implementation plan

### Files Modified
- `.kb/investigations/2026-03-26-inv-design-retry-strategy-gpt-silent.md` - filled with findings, recommendation, and implementation handoff

### Issue Decomposition
- `orch-go-o9k80` - classifier
- `orch-go-phbzy` - one-shot retry handoff
- `orch-go-bfzfc` - telemetry and status visibility
- `orch-go-6k6o8` - integration proof with dependencies on the three component issues

### Commits
- None yet in this workspace session

---

## Evidence (What Was Observed)

- The benchmark investigation captured a GPT-5.4 task that died with zero tokens and then succeeded unchanged on rerun, proving a recoverable transient exists.
- `pkg/orch/spawn_modes.go` and `pkg/opencode/client.go` show current retry/verification stops at prompt acceptance plus brief error listening, not empty execution detection.
- `pkg/agent/lifecycle_impl.go` uses `SessionExists`, while `pkg/opencode/client.go` documents that persisted idle sessions still count as existing, which explains why current orphan recovery misses this failure class.
- `cmd/orch/status_cmd.go` and `cmd/orch/status_test.go` show zero-token sessions are visually collapsed to `-`, which hides the evidence a human would need to trust automatic retry.

### Verification Contract

See `.orch/workspace/og-arch-design-retry-strategy-26mar-e992/VERIFICATION_SPEC.yaml`.
Key outcomes:
- Design recommendation is backend-scoped and one-shot, not generic retry-on-zero-tokens.
- Plan and implementation issues are created and linked.
- Manual verification remains structural because no production code changed.

---

## Architectural Choices

### Retry trigger design
- **What I chose:** compound empty-execution fingerprint
- **What I rejected:** retry on zero tokens alone
- **Why:** zero tokens by itself is too weak and invites duplicate work
- **Risk accepted:** implementation is slightly more complex because multiple signals must be gathered

### Retry budget design
- **What I chose:** one immediate retry then escalation
- **What I rejected:** reuse broad retry budgets or retry until success
- **Why:** benchmark evidence proves one rerun can recover, but not that repeated retries are safe
- **Risk accepted:** some second-transient recoveries will still need a human

### Scope boundary
- **What I chose:** OpenCode-only behavior
- **What I rejected:** cross-backend generic retry policy
- **Why:** Claude/tmux workers do not expose the same session-status and token APIs
- **Risk accepted:** multi-backend consistency is deferred in favor of correctness

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-design-retry-strategy-gpt-silent.md` - final design answer with recommendations
- `.kb/plans/2026-03-26-gpt54-empty-execution-retry.md` - phased implementation handoff

### Decisions Made
- Retry should be keyed off an OpenCode empty-execution fingerprint, not zero tokens alone.
- Automatic retry should happen once, then escalate on repeat failure.

### Constraints Discovered
- Persisted OpenCode sessions make `SessionExists` an unreliable proxy for liveness in this failure mode.
- Zero-token evidence is currently hidden in status output, so observability must be part of the design rather than a follow-up nice-to-have.

### Externalized via `kb quick`
- `kb quick constrain "Zero-token silent-death retry must be OpenCode-scoped and one-shot" --reason "Persisted idle sessions make SessionExists unreliable for liveness, and zero-token alone is too weak to justify repeated retries without duplicate-work risk."`

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** `orch-go-o9k80`, `orch-go-phbzy`, `orch-go-bfzfc`, `orch-go-6k6o8`
**Skill:** feature-impl
**Context:**
```text
The design work is done. Implement an OpenCode-only empty-execution classifier, wire a one-shot retry path, surface retry evidence in status/review, and then prove the behavior end to end using the linked plan and issue acceptance criteria.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Can OpenCode expose a stronger server-native terminal reason so orch can stop inferring empty execution from client-side evidence?
- Should empty-execution retries be labeled in beads for analytics, or kept in events/status only?

**Areas worth exploring further:**
- Larger GPT-5.4 reasoning benchmark after the retry path lands

**What remains unclear:**
- Whether partial work can ever coexist with zero aggregated tokens in OpenCode's message model

---

## Friction

No friction - smooth session.

---

## Session Metadata

**Skill:** architect
**Model:** openai/gpt-5.4
**Workspace:** `.orch/workspace/og-arch-design-retry-strategy-26mar-e992/`
**Investigation:** `.kb/investigations/2026-03-26-inv-design-retry-strategy-gpt-silent.md`
**Beads:** `bd show orch-go-7iw5a`
