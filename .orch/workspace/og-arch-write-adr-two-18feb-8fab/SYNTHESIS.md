# Session Synthesis

**Agent:** og-arch-write-adr-two-18feb-8fab
**Issue:** orch-go-1081
**Duration:** 2026-02-18 23:21 → 2026-02-18 23:35
**Outcome:** success

---

## Plain-Language Summary

This ADR documents the architectural decision to split agent observability into two lanes: **tracked work** (agents with beads issues, visible in `orch status`) and **untracked sessions** (orchestrators, ad-hoc agents, `--no-track` spawns, visible in `orch sessions`). 

The decision resolves 6 weeks of ghost agent bugs by:
1. **Locking domain boundaries** — beads owns work lifecycle, OpenCode owns liveness only, workspace manifests own binding
2. **Requiring atomic spawn** — all 3 writes (beads + manifest + session) succeed or spawn fails entirely
3. **Adding reason codes** — silent empty metadata becomes visible failure with explicit causes
4. **Forbidding persistent lifecycle caches** — CI lint rule blocks new `pkg/state/`, `pkg/registry/`, `pkg/cache/` packages

This breaks the 5-iteration cache/remove cycle by making cache accretion structurally impossible.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for acceptance test matrix derived from the ADR.

**Key outcomes:**
- ADR captures all 6 design points from the beads issue specification
- Acceptance test matrix covers all 12 scenarios from the issue
- Regression guardrails include both contract tests and architecture lint rules
- Complementary work (OpenCode fork metadata API) is documented as future simplification

---

## Delta (What Changed)

### Files Created
- `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — The ADR itself

### Files Modified
- None

### Commits
- To be committed after Phase: Complete reported

---

## Evidence (What Was Observed)

- Read beads issue orch-go-1081 with full design spec (6 design points, 12-scenario acceptance matrix)
- Read prior investigation: `.kb/investigations/2026-02-18-design-agent-observability-rethink.md` — confirmed work-centric discovery model
- Read prior decisions: `2026-02-14-lifecycle-ownership-own-accept-build.md`, `2026-01-12-registry-is-spawn-cache.md` — confirmed this ADR supersedes registry decisions
- Read historical context: `2025-12-21-synthesis-registry-evolution-and-orch-identity.md` — confirmed 5-iteration cycle narrative

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/decisions/2026-02-18-two-lane-agent-discovery.md` — Comprehensive ADR with design decisions and acceptance test matrix

### Decisions Made
- Two-lane split (tracked vs untracked) resolves semantic mismatch that killed Jan 5 beads-first attempt
- Atomic spawn with rollback on failure — no partial state allowed
- Reason codes on all queries — silent failures become visible
- CI lint gate on lifecycle state packages — structural prevention of drift

### Constraints Discovered
- Beads availability becomes hard dependency for spawn (intentional tradeoff)
- CLI performance target is 2s without persistent cache (acceptable)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (ADR written with all 6 design points + acceptance matrix)
- [x] Investigation file has `**Status:** Accepted`
- [x] Ready for `orch complete orch-go-1081`

**Follow-up work (tracked separately):**
- orch-go-1080: Implement OpenCode session metadata integration
- Implementation of atomic spawn, reason codes, and lint rules (spawn from this ADR)

---

## Unexplored Questions

**Straightforward session, no unexplored territory.**

The design spec was complete in the beads issue. ADR documents it faithfully.

---

## Session Metadata

**Skill:** architect
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-write-adr-two-18feb-8fab/`
**Decision:** `.kb/decisions/2026-02-18-two-lane-agent-discovery.md`
**Beads:** `bd show orch-go-1081`
