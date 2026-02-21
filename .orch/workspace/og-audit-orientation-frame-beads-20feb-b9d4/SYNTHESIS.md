# Session Synthesis

**Agent:** og-audit-orientation-frame-beads-20feb-b9d4
**Issue:** orch-go-1165
**Duration:** 2026-02-20
**Outcome:** success

---

## Plain-Language Summary

The beads fork has diverged significantly from upstream with 43 local commits — all made after the Dec 2025 "clean slate" decision that explicitly said to drop the fork. That decision was correct for the old features (bd ai-help, bd health, bd tree) which genuinely weren't used, but in the 2 months since, entirely new features were developed in the fork that orch-go now actively depends on (question entity type, title-based dedup, Phase: Complete gate on close, investigation issue type, question dependency blocking). The knowledge base documents (model, decision) are significantly stale: the model references 5 files that don't exist, has the wrong socket path, and states a "never use exec.Command directly" constraint that's violated 11 times. The model's core architectural claim (RPC-first with CLI fallback at three lifecycle points) remains accurate.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for verification details.

Key outcomes:
- Probe file created documenting 4 contradictions, 3 confirmations, and 1 extension to the model
- Investigation file created with full fork inventory (43 commits categorized), usage status for each fork feature, and 7 prioritized recommendations
- No code changes — audit/research only

---

## Delta (What Changed)

### Files Created
- `.kb/models/beads-integration-architecture/probes/2026-02-20-beads-fork-integration-audit.md` — Probe documenting model claims vs reality
- `.kb/investigations/2026-02-20-inv-audit-beads-fork-integration.md` — Full investigation with fork inventory, staleness assessment, and recommendations
- `.orch/workspace/og-audit-orientation-frame-beads-20feb-b9d4/SYNTHESIS.md` — This file
- `.orch/workspace/og-audit-orientation-frame-beads-20feb-b9d4/VERIFICATION_SPEC.yaml` — Verification spec

### Files Modified
- None (audit only)

---

## Evidence (What Was Observed)

- **43 fork commits** beyond upstream, all post-Dec 21 clean-slate decision (verified via `git log origin/main..HEAD`)
- **6 fork features actively used** by orch-go code (question type, dedup, Phase gate, investigation type, question deps, close exit codes)
- **5 fork features implicitly depended on** (sandbox detection, JSONL-only, restart prevention, file locking, fingerprint validation)
- **5 of 6 model "Primary Evidence" files don't exist** (fallback.go, lifecycle.go, id.go, tracking.go, spawn.go)
- **Socket path wrong in model**: claims `~/.beads/daemon.sock`, actual is `.beads/bd.sock`
- **11 direct exec.Command("bd") calls** across 7 files outside pkg/beads, violating model constraint
- **RPC-first with CLI fallback pattern confirmed** as core architecture
- **Clean-slate decision reversed** within 9 days, never formally superseded

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-20-inv-audit-beads-fork-integration.md` — Complete audit
- `.kb/models/beads-integration-architecture/probes/2026-02-20-beads-fork-integration-audit.md` — Model probe

### Constraints Discovered
- The beads fork is now a critical dependency, not an optional enhancement
- exec.Command("bd") constraint is aspirational, not enforced

---

## Next (What Should Happen)

**Recommendation:** close

### Follow-up work identified:
1. **Supersede clean-slate decision** — New decision documenting active fork relationship
2. **Update model** — Fix file references, socket path, code examples
3. **Migrate exec.Command calls** — 5-7 calls could use BeadsClient interface
4. **Upstream contribution strategy** — PR generally-useful fixes to reduce fork burden

### If Close
- [x] All deliverables complete (probe + investigation)
- [x] Investigation file has complete findings
- [x] Ready for `orch complete orch-go-1165`

---

## Unexplored Questions

- **Upstream sync frequency**: How often is the fork rebased against upstream? No rebase evidence found.
- **Fork push status**: The `fork` remote exists but unclear if all 43 commits are pushed to dylan-conlin/beads on GitHub.
- **Beads daemon health**: How reliably does the RPC daemon stay running? Model mentions health checks but no data on actual uptime.

---

## Session Metadata

**Skill:** codebase-audit
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-audit-orientation-frame-beads-20feb-b9d4/`
**Investigation:** `.kb/investigations/2026-02-20-inv-audit-beads-fork-integration.md`
**Beads:** `bd show orch-go-1165`
