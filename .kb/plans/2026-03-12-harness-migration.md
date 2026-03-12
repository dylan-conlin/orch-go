## Summary (D.E.K.N.)

**Delta:** Standalone harness CLI becomes the single governance tool — absorbs all orch harness functionality, usable via `go install` with zero orch dependencies.

**Evidence:** Migration inventory (orch-go-q69uj): 70% already migrated, 5 commands missing. Architect design (orch-go-sb13k): separate repo, zero deps. Knowledge-feedback failure (orch-go-85e5c): confirmed tools must stay separate.

**Knowledge:** The bug that started this (relative-path hook breaking all projects) was caused by scope confusion between two tools doing the same thing. One tool eliminates the confusion.

**Next:** Fix the relative-path bug in standalone harness (Phase 1).

---

# Plan: Harness Migration

**Date:** 2026-03-12
**Status:** Active
**Owner:** Dylan

**Extracted-From:** orch-go-q69uj (migration inventory), orch-go-sb13k (architect design)

---

## Objective

Standalone `harness` at `~/Documents/personal/harness/` (`github.com/dylan-conlin/harness`) becomes the single governance CLI. `orch harness` is either removed or becomes a thin wrapper. Anyone can `go install github.com/dylan-conlin/harness` and get project governance without orch.

---

## Phases

### Phase 1: Fix standalone harness init bug

**Goal:** Fix the relative-path bug that started this whole thread
**Deliverables:** `harness init` writes to project-level settings only, never user-level with relative paths
**Exit criteria:** Run `harness init` in project A, verify no impact on project B
**Depends on:** Nothing
**Beads:** orch-go-ayh (harness repo), orch-go-1eo (harness repo, original bug)

### Phase 2: Add lock/unlock/status/verify commands

**Goal:** Port control plane immutability management from orch harness
**Deliverables:** `harness lock`, `harness unlock`, `harness status`, `harness verify` commands
**Exit criteria:** `harness lock` + `harness unlock` works on `~/.claude/settings.json` and hook scripts
**Depends on:** Phase 1
**Beads:** orch-go-716 (harness repo)

### Phase 3: Add snapshot command

**Goal:** Port accretion velocity snapshot from orch harness
**Deliverables:** `harness snapshot` captures directory-level line counts
**Exit criteria:** Snapshots comparable to `orch harness snapshot` output
**Depends on:** Phase 1
**Beads:** orch-go-92m (harness repo)

### Phase 4: Wire orch harness as wrapper

**Goal:** `orch harness` delegates to standalone `harness` binary
**Deliverables:** `orch harness init/check/lock/unlock/status/verify/snapshot` all call `harness` under the hood
**Exit criteria:** All `orch harness` commands produce identical output via delegation
**Depends on:** Phases 2, 3
**Beads:** orch-go-pe1fn

### Phase 5: Remove orch harness implementation

**Goal:** Delete harness implementation code from orch-go, keep only the wrapper (or remove entirely)
**Deliverables:** `cmd/orch/harness_*.go` reduced to wrapper or removed. `harness` is the canonical binary.
**Exit criteria:** `orch harness` either delegates cleanly or is gone. No duplicate code.
**Depends on:** Phase 4
**Beads:** orch-go-olr0x

---

## Structured Uncertainty

**What's tested:**
- The standalone repo structure works (already has init/check/report)
- Lock/unlock is OS-level chflags with no orch dependencies
- 70% of code already migrated per inventory

**What's untested:**
- Whether `harness report` can work without orch's event system (report/snapshot may need a standalone event format)
- Whether `orch harness` wrapper adds enough value to keep vs just removing it

**What would change this plan:**
- If `harness report` requires deep orch integration, it stays in orch and standalone harness stays at init/check/lock/unlock/verify/snapshot

---

## Success Criteria

- [ ] `harness init` safe to run in any project (no cross-project impact)
- [ ] `harness lock/unlock/status/verify` work standalone
- [ ] `go install github.com/dylan-conlin/harness` gives a working governance tool
- [ ] No duplicate harness implementation code in orch-go
