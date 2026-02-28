# Session Synthesis

**Agent:** og-arch-design-centralized-agent-27feb-bda3
**Issue:** orch-go-lckl
**Duration:** 2026-02-27
**Outcome:** success

---

## Plain-Language Summary

Four lifecycle bugs were discovered in one session (ghost agents after abandon, clean killing Claude agents, stale daemon status, orphaned in_progress issues), all caused by the same root problem: no single package owns the complete set of side effects for agent state transitions. Each command (spawn, complete, abandon, clean) independently implements a subset of the required cleanup, and each misses something different.

The design creates a `pkg/agent/lifecycle.go` package with a `LifecycleManager` struct that exposes one method per transition (Complete, Abandon, ForceComplete, ForceAbandon, DetectOrphans). Each method runs ALL required side effects in order — beads operations, workspace archival, OpenCode session cleanup, tmux window closure, event logging. This makes incomplete transitions structurally impossible because callers no longer assemble side effects themselves. The manager does NOT store agent state (respecting the "No Local Agent State" constraint) — it's a coordinator that reads from and writes to the four authoritative sources.

## TLDR

Designed a centralized agent lifecycle state machine with 7 states, 5 transitions, and explicit side-effect ordering. Created 6 implementation issues for phased migration (types → abandon → complete → GC → daemon → spawn). The abandon-first approach delivers an immediate P1 bug fix (ghost agents) while validating the pattern.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-27-inv-design-centralized-agent-lifecycle-state.md` - Full design with state machine, interface definitions, migration plan

### Commits
- (design-only — no code changes)

---

## Evidence (What Was Observed)

- Mapped all side effects across spawn (8), complete (14), abandon (8), clean (varies) — totaling 30+ individual operations across 5 subsystems
- Confirmed abandon_cmd.go lines 283-291 are missing `RemoveOrchAgentLabel` and `UpdateAssignee` calls (ghost agent bug)
- Confirmed clean_cmd.go line 478-480 checks OpenCode session existence only, missing phase-based liveness for Claude-mode agents
- Confirmed daemon-status.json is read without PID validation (serve_system.go line 417)
- Confirmed daemon sets in_progress but has no recovery for agents that die silently
- Verified atomic spawn pattern (pkg/spawn/atomic.go) proves centralized transitions work

---

## Architectural Choices

### Coordinator pattern over event sourcing
- **What I chose:** Synchronous side effects with ordered execution and explicit rollback
- **What I rejected:** Event-sourced transitions (emit events, materialize side effects asynchronously)
- **Why:** Event sourcing adds complexity (event store, materializers, eventual consistency) for a problem that's fundamentally about ordering — not about audit trails. The atomic spawn pattern already proves synchronous transitions work.
- **Risk accepted:** Long-running transitions block the caller. Acceptable because transitions are infrequent (seconds apart) and side effects are fast (RPC calls <100ms each).

### Verification stays in pkg/verify/
- **What I chose:** LifecycleManager calls into verify for preconditions, then runs cleanup
- **What I rejected:** Absorbing verification gates into the lifecycle package
- **Why:** "Evolve by distinction" principle — verification (should we close?) is a different concern than lifecycle (how do we close?). Merging them would create a God package.
- **Risk accepted:** Two packages must coordinate. The interface is clear: verify first, then lifecycle.

### Concrete struct over interface
- **What I chose:** `LifecycleManager` as concrete struct with injected client interfaces
- **What I rejected:** `LifecycleManager` as an interface
- **Why:** Go convention — define interfaces at the consumer. The struct is testable via its client interfaces (BeadsClient, OpenCodeClient, etc.).
- **Risk accepted:** If multiple implementations are ever needed, interface extraction is straightforward.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-27-inv-design-centralized-agent-lifecycle-state.md` - Comprehensive design

### Decisions Made
- Lifecycle manager is a coordinator (transition logic), not a store (state cache) — compatible with "No Local Agent State"
- Abandon-first migration because it's simplest + has clearest bug fix
- Clean becomes pure detection + delegation (detect orphans → call lifecycle transitions)

### Constraints Discovered
- Complete pipeline phases 1-3 (identification, discovery, knowledge) contain interactive prompts and must stay in complete_cmd.go — only phases 4-7 (cleanup) migrate to lifecycle manager
- Architecture lint tests in `architecture_lint_test.go` may flag `pkg/agent/` — need to verify they only block state *storage* packages, not transition *logic* packages

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` in workspace root.

---

## Next (What Should Happen)

**Recommendation:** close

### Implementation Issues Created

| Issue | Phase | Priority | Depends On |
|-------|-------|----------|------------|
| orch-go-bho1 | 1: Define types | P2 | — |
| orch-go-ohde | 2: Abandon transition | P1 | bho1 |
| orch-go-hbtr | 3: Complete transition | P2 | bho1, ohde |
| orch-go-vp6u | 4: Orphan detection + GC | P2 | bho1, ohde |
| orch-go-vem4 | 5: Daemon recovery | P2 | vp6u |
| orch-go-mgbk | 6: Spawn integration | P3 | bho1 |

All issues have `triage:ready` + `skill:feature-impl` labels and correct dependency chains.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-centralized-agent-27feb-bda3/`
**Investigation:** `.kb/investigations/2026-02-27-inv-design-centralized-agent-lifecycle-state.md`
**Beads:** `bd show orch-go-lckl`
