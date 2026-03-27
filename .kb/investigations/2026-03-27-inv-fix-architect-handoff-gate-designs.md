## Summary (D.E.K.N.)

**Delta:** Fixed the architect_handoff gate to fail when SYNTHESIS.md is missing, closing the gap where architects at V1 verification level could complete without implementation issues.

**Evidence:** 13 unit tests pass, including root-cause reproduction test. `go test ./pkg/verify/` and `go build ./...` clean.

**Knowledge:** The bug was a level mismatch: architect_handoff gate (V1) deferred to synthesis gate (V2+) for SYNTHESIS.md validation, but architects default to V1 where the synthesis gate never runs. Also added comment-based opt-out and evidence signals for the new Phase 6: Handoff pattern.

**Next:** Close — code change is complete, tests pass.

**Authority:** implementation — Tactical fix within existing gate infrastructure, no architectural changes.

---

# Investigation: Fix Architect Handoff Gate Designs

**Question:** Why do architect agents complete without creating implementation issues, and how to enforce the handoff gate structurally?

**Started:** 2026-03-27
**Updated:** 2026-03-27
**Owner:** worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** completion-verification

---

## Findings

### Finding 1: Verification level mismatch — architect_handoff gate deferred to V2+ synthesis gate

**Evidence:** `VerifyArchitectHandoff` (architect_handoff.go:67-70) returned `Passed: true` when SYNTHESIS.md was missing, with comment "synthesis gate handles that separately." But architect defaults to V1 (`SkillVerifyLevelDefaults["architect"] = VerifyV1` in verify_level.go:29), and the synthesis gate is V2+ (`gatesByLevel[VerifyV2]` includes `GateSynthesis` in level.go:22).

**Source:** `pkg/verify/architect_handoff.go:67-70`, `pkg/verify/level.go:8-34`, `pkg/spawn/verify_level.go:29`

**Significance:** This is the exact root cause. An architect at V1 with no SYNTHESIS.md passes both the architect_handoff gate (early return) AND has no synthesis gate running. Result: clean completion with no implementation issues and no SYNTHESIS.md.

---

### Finding 2: Auto-create mechanism depends on SYNTHESIS.md parsing

**Evidence:** `maybeAutoCreateImplementationIssue` (complete_architect.go:38-41) returns empty string if SYNTHESIS.md parsing fails. No error logged, no fallback. The auto-create silently does nothing when SYNTHESIS.md is missing.

**Source:** `cmd/orch/complete_architect.go:38-41`

**Significance:** Both the gate AND the auto-create depend on SYNTHESIS.md. Without it, both silently pass/skip. The gate should be the safety net — it needs to fail when the auto-create can't run.

---

### Finding 3: Phase 6: Handoff skill text introduces comment-based signals

**Evidence:** The deployed architect skill (SKILL.md.template:240-283) defines Phase 6: Handoff (MANDATORY) with two patterns:
- `"Phase: Handoff - Created implementation issues: <ids>"` (evidence of manual creation)
- `"Phase: Handoff - No implementation issues: [reason]"` (explicit opt-out)

**Source:** `skills/src/worker/architect/.skillc/SKILL.md.template:240-283`

**Significance:** The code gate should recognize both comment patterns as valid handoff signals, not just the title-pattern match from auto-create. This makes the gate compatible with manual issue creation and advisory-only designs.

---

## Synthesis

**Key Insights:**

1. **Level mismatch was invisible** — The architect_handoff gate's comment said "synthesis gate handles that" but never verified this was true for the architect's verification level. The assumption was wrong.

2. **Three handoff signals needed** — Auto-create title pattern, manual creation comment, and opt-out comment. The original gate only checked one signal.

3. **Defect class: Filter Amnesia (Class 1)** — The synthesis check existed in the V2 path but was missing from the V1 architect_handoff path. Classic "filter exists in path A, missing in path B."

---

## Structured Uncertainty

**What's tested:**

- ✅ Missing SYNTHESIS.md now fails architect_handoff gate (TestVerifyArchitectHandoff_MissingSynthesis, TestVerifyArchitectHandoff_MissingSynthesis_RootCause)
- ✅ Comment opt-out detected (TestVerifyArchitectHandoff_CommentOptOut)
- ✅ Comment evidence detected (TestVerifyArchitectHandoff_CommentEvidence)
- ✅ Non-architect skills still pass (TestVerifyArchitectHandoff_NonArchitectSkill)
- ✅ All existing test scenarios still pass (13 tests total)
- ✅ Full `go build ./...` and `go test ./pkg/verify/` clean

**What's untested:**

- ⚠️ Integration with live beads — HasImplementationFollowUp with real beads data
- ⚠️ Daemon auto-complete path with real architect workspace

---

## References

**Files Modified:**
- `pkg/verify/architect_handoff.go` — Gate logic: fail on missing SYNTHESIS.md, add comment-based signals
- `pkg/verify/architect_handoff_test.go` — Updated tests, added 4 new test cases
- `pkg/verify/check.go` — Pass comments to VerifyArchitectHandoff call
