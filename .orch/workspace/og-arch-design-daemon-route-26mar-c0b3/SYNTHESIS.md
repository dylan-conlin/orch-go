# Session Synthesis

**Agent:** og-arch-design-daemon-route-26mar-c0b3
**Issue:** orch-go-3k1yo
**Duration:** 2026-03-26 14:31 → 2026-03-26 15:24
**Outcome:** success

---

## TLDR

I designed a capability-aware daemon routing policy for GPT-5.4 vs Opus. The recommendation keeps Opus as the default lane, allows GPT-5.4 only for bounded `feature-impl` overflow, and requires route observability plus one-step promotion back to Opus on classified GPT failure.

---

## Delta (What Changed)

### Files Created
- `.kb/plans/2026-03-26-daemon-gpt54-routing.md` - Implementation plan for the routing design.
- `.orch/workspace/og-arch-design-daemon-route-26mar-c0b3/SYNTHESIS.md` - Session synthesis for orchestrator review.
- `.orch/workspace/og-arch-design-daemon-route-26mar-c0b3/BRIEF.md` - Dylan-facing comprehension brief.
- `.orch/workspace/og-arch-design-daemon-route-26mar-c0b3/VERIFICATION_SPEC.yaml` - Verification evidence for the design session.

### Files Modified
- `.kb/investigations/2026-03-26-inv-design-daemon-route-tasks-gpt.md` - Recorded findings, synthesis, and implementation recommendations.

### Commits
- Pending local commit for design artifacts.

---

## Evidence (What Was Observed)

- Current daemon routing only hard-pins Opus for a sparse reasoning-skill map and otherwise leaves `feature-impl` to downstream defaults, so it cannot express complexity-aware GPT routing (`pkg/daemon/skill_inference.go:259`, `pkg/daemon/issue_adapter.go:426`).
- GPT-5.4 benchmark evidence is good enough for bounded `feature-impl` overflow but not strong enough for reasoning-default routing (`.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md:136`, `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md:220`).
- Existing daemon architecture already has the right seams for this work: route rewriting in coordination, combo-first policy resolution in daemon config, and existing `effort:*` labels as a first complexity signal (`pkg/daemon/coordination.go:37`, `pkg/daemonconfig/compliance.go:53`).
- Historical GPT failures and the retry-design work show routing and recovery must ship together; otherwise GPT retries risk duplicate work and hidden failure loops (`.kb/models/daemon-autonomous-operation/probes/2026-02-19-probe-daemon-spawned-agents-stall-gpt52-codex.md:91`, `.kb/plans/2026-03-26-gpt54-empty-execution-retry.md:48`).

### Tests Run
```bash
# Verify project location
pwd
# /Users/dylanconlin/Documents/personal/orch-go

# Create investigation artifact
kb create investigation design-daemon-route-tasks-gpt --orphan
# Created .kb/investigations/2026-03-26-inv-design-daemon-route-tasks-gpt.md

# Pull routing context
kb context "daemon model routing"
# Returned existing constraints, decisions, guides, and model links

# Create implementation plan artifact
orch plan create daemon-gpt54-routing
# Created .kb/plans/2026-03-26-daemon-gpt54-routing.md
```

---

## Architectural Choices

### Capability-aware routing instead of provider-wide routing
- **What I chose:** A two-lane router: Opus default lane plus GPT-5.4 bounded overflow lane.
- **What I rejected:** Routing all implementation work to GPT-5.4 or leaving GPT manual-only forever.
- **Why:** The benchmark clears GPT-5.4 for overflow but not for broad default routing.
- **Risk accepted:** Some safe implementation work will remain on Opus until labels or evidence get richer.

### Reuse existing daemon policy structures
- **What I chose:** Put the new routing logic in daemon coordination/config rather than patching spawn execution with GPT special cases.
- **What I rejected:** Hidden model choice logic in `SpawnWork()`.
- **Why:** One canonical derivation avoids contradictory routing signals and keeps observability straightforward.
- **Risk accepted:** Slightly larger refactor up front.

### Couple routing with bounded promotion
- **What I chose:** Promote classified GPT failures to Opus once.
- **What I rejected:** Infinite GPT retries or no automatic recovery.
- **Why:** Historical GPT failure modes make repeated blind retries unsafe, but the new benchmark justifies one bounded recovery step.
- **Risk accepted:** Promotion criteria may need tuning after first implementation.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-design-daemon-route-tasks-gpt.md` - Design investigation for daemon GPT-5.4 routing.
- `.kb/plans/2026-03-26-daemon-gpt54-routing.md` - Multi-phase implementation plan.

### Decisions Made
- Keep Opus as the daemon default for reasoning-heavy and high-complexity work.
- Route GPT-5.4 only for bounded `feature-impl` overflow in v1.
- Treat GPT routing and Opus promotion as one system, not separate follow-up concerns.

### Constraints Discovered
- Current daemon model inference is too coarse for complexity-aware routing.
- `effort:*` labels are the best existing complexity signal but will produce conservative false negatives when absent.

### Externalized via `kb quick`
- `kb quick decide "Daemon should keep Opus as the default route and use GPT-5.4 only for bounded feature-impl overflow until reasoning-skill benchmarks improve" --reason "Current daemon code only supports skill-level overrides, Mar 26 benchmark validated GPT-5.4 for feature-impl overflow, and historical GPT failure modes require bounded promotion back to Opus"` - Recorded the routing policy as a durable decision.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** `orch-go-ckddz`, `orch-go-r7avo`, `orch-go-kdyh6`, `orch-go-xi8tc`
**Skill:** `feature-impl`
**Context:**
```text
Implement the two-lane daemon router in phases: add the route object and eligibility logic first, then config + observability, then bounded promotion to Opus, and finally behavioral verification across both model lanes.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should Sonnet become a same-backend middle lane before GPT expands beyond overflow?
- Should future routing consume empirical success-rate data rather than static skill/effort rules?

**Areas worth exploring further:**
- GPT-5.4 N>=10 investigation/debugging benchmark.
- Label coverage quality for `effort:*` on daemon-ready issues.

**What remains unclear:**
- Exact promotion trigger set for GPT failures beyond `empty_execution`.

---

## Friction

**System friction experienced during this session:**
- No friction — smooth session.

---

## Session Metadata

**Skill:** architect
**Model:** openai/gpt-5.4
**Workspace:** `.orch/workspace/og-arch-design-daemon-route-26mar-c0b3/`
**Investigation:** `.kb/investigations/2026-03-26-inv-design-daemon-route-tasks-gpt.md`
**Beads:** `bd show orch-go-3k1yo`
