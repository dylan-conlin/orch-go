# Session Synthesis

**Agent:** og-arch-design-bead-prioritization-06feb-cd2d
**Issue:** orch-go-21413
**Duration:** 2026-02-06 22:44 → 2026-02-06 23:20
**Outcome:** success

---

## TLDR

Designed a bead prioritization sorting system with 5 named sort strategies (Unblock, Flow State, Firefight, Debt Reduction, Decision Harvest) implemented as Go functions in `pkg/daemon/sort/`, selected by daemon config, consuming cached frontier data. Key design decision: sort lives in orch daemon (not bd CLI) because strategies require cross-system data (beads + frontier + sessions).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-06-inv-design-bead-prioritization-sorting-system.md` - Complete design investigation with 6 navigated forks and recommendations

### Files Modified
- None (design-only session)

### Commits
- (pending commit)

---

## Evidence (What Was Observed)

- Daemon currently sorts by priority only (`daemon_queue.go:49`) — single dimension
- `bd ready --sort` supports hybrid/priority/oldest but lacks cross-system dimensions
- `ReadyArgs.SortPolicy` field exists in `pkg/beads/types.go:125` but is never populated by orch-go
- `frontier.CalculateFrontier()` already computes transitive leverage — reusable for Unblock Mode
- Area label coverage is ~33% (19/58 open issues) — sparse but sufficient for opt-in locality sorting
- Attention system has its own priority model but daemon is explicitly separate (decision 2026-02-03)
- 5 sort dimensions require 3+ data sources: beads (priority, type, labels), frontier (leverage), OpenCode (sessions/area)

### Tests Run
```bash
# Design investigation - no code changes to test
# Verified existing sort behavior via code inspection
# Verified bd ready --sort capabilities via bd ready --help
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-06-inv-design-bead-prioritization-sorting-system.md` - Complete design with 6 decision forks navigated

### Decisions Made
- Sort logic lives in orch daemon (not bd CLI) because strategies need cross-system data
- Named presets only (no composability) because strategies optimize for conflicting targets
- Go code for pipelines (not config DSL) because pipelines call multiple Go packages
- Skip missing dimensions with neutral score for graceful degradation
- Pre-compute frontier per poll cycle, cache for sort functions
- Require-going-forward for metadata (don't backfill 58 existing issues)

### Constraints Discovered
- `verification_cost` dimension has no data source — defer until estimation mechanism exists
- `touch_count` not tracked by beads — proxy via comment count or defer
- `daemon_crossproject.go:58` has its own sort that must also use strategy system

---

## Issues Created

No discovered work during this session. The investigation itself IS the deliverable.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with design)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-21413`

**Follow-up implementation:**
The investigation recommends a phased implementation:
1. Create `pkg/daemon/sort/` package with Strategy interface
2. Implement Priority (current behavior) and Unblock strategies
3. Add frontier cache to daemon poll loop
4. Wire into NextIssueExcluding()
5. Implement remaining 3 strategies

This would be a separate feature-impl issue, not part of this architect session.

---

## Unexplored Questions

**Questions that emerged during this session:**
- How should active area detection work? (Needs to query active sessions and extract area labels)
- Should sort strategy be observable in orch status output? (Probably yes — "sort: unblock")
- How does sort mode interact with the attention system's priority? (They're independent per 2026-02-03 decision, but should they inform each other?)

**What remains unclear:**
- Performance characteristics of frontier cache in practice
- Whether 5 strategies is the right number (might need more, might need fewer)

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-arch-design-bead-prioritization-06feb-cd2d/`
**Investigation:** `.kb/investigations/2026-02-06-inv-design-bead-prioritization-sorting-system.md`
**Beads:** `bd show orch-go-21413`
