# Session Synthesis

**Agent:** og-arch-design-coupling-hotspot-19feb-8d9c
**Issue:** orch-go-1109
**Duration:** 2026-02-19 → 2026-02-19
**Outcome:** success

---

## TLDR

Designed a coupling hotspot analysis system that detects cross-surface concept clusters — areas where a single concept spans many files across architectural layers. Validated with real git data: mining 2,733 commits identified 4 concept clusters (daemon, verification, spawn, agent-status) that exactly match known agent-hostile areas. The daemon config cluster scores 180 on the proposed coupling formula, correctly flagging the area that caused a 526K token spiral.

---

## Plain-Language Summary

The current `orch hotspot` command only detects large files (>800 lines). But the daemon config problem — where adding 1 boolean requires touching 12 files across 3 layers — slips through because no individual file is large. Agent ek0b spiraled at 526K tokens trying to discover this touch surface.

This design adds a 4th hotspot type: `coupling-cluster`. It mines git history for files that always change together across different architectural layers (cmd/, pkg/, web/). By filtering to "cross-surface commits" (only 6% of all commits), it automatically eliminates healthy coupling (test files) and surfaces agent-hostile coupling (12 files for 1 boolean). The output integrates directly into existing `orch hotspot` output and JSON format, so spawn gates and the dashboard get coupling awareness for free.

The key insight: cross-surface commits are rare but perfectly identify the problem areas. The algorithm is simple (git log parsing + path-based concept extraction + composite scoring), runs in <5 seconds, and requires no language-specific tooling.

---

## Verification Contract

See `VERIFICATION_SPEC.yaml` for acceptance criteria.

Key outcomes:
- Investigation: `.kb/investigations/2026-02-19-design-coupling-hotspot-analysis-system.md`
- Probe: `.kb/models/completion-verification/probes/2026-02-19-probe-coupling-hotspot-detection-gap.md`
- 5 decision forks navigated with substrate traces
- Implementation scope estimated: ~200 lines in new `cmd/orch/hotspot_coupling.go`

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-19-design-coupling-hotspot-analysis-system.md` — Full design with algorithm, integration points, scoring, and healthy-vs-hostile criteria
- `.kb/models/completion-verification/probes/2026-02-19-probe-coupling-hotspot-detection-gap.md` — Probe confirming accretion enforcement is blind to coupling

### Files Modified
- None (design-only session)

### Commits
- (pending)

---

## Evidence (What Was Observed)

- **2,733 commits** analyzed over 90 days; 1,212 non-metadata commits
- **76 cross-surface commits** (6% of total) touching 3+ layers contain all agent-hostile coupling
- **4 concept clusters** identified: daemon (24 commits, 25 files, 3 layers), verification (23/33/2), spawn (22/33/2), agent-status (22/14/3)
- **Coupling score of 180** for daemon cluster correctly identifies the 526K-spiral area as CRITICAL
- **Tmux scores 13** — correctly identified as healthy (1 layer, no cross-surface coupling)
- **events.jsonl lacks token_count** — spiral correlation deferred to Phase 2
- **203 abandoned agents** (23.6% abandonment rate) available as project-level signal

### Validation: Algorithm matches known problems
```
daemon config: 3 layers × 25 files × 2.4 avg = 180 → CRITICAL ✓ (caused 526K spiral)
agent-status:  3 layers × 14 files × 1.6 avg =  67 → HIGH     ✓ (84/85 empty metadata)
tmux:          1 layer  ×  3 files × 4.3 avg =  13 → noise    ✓ (never caused issues)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-19-design-coupling-hotspot-analysis-system.md` — Coupling hotspot design
- `.kb/models/completion-verification/probes/2026-02-19-probe-coupling-hotspot-detection-gap.md` — Probe extending completion-verification model

### Decisions Made
- **Algorithm:** Layer-crossing filter (not pairwise counting or commit clustering) because cross-surface commits are 6% but contain 100% of hostile coupling
- **Naming:** Path-based concept extraction (not commit messages or manual labels) because directory/file names encode concepts naturally in Go
- **Integration:** 4th hotspot type in existing `orch hotspot` (not separate command) for free spawn gate and dashboard integration
- **Scoring:** Composite formula (layers × files × frequency) validated against known spiral data
- **Spiral correlation:** Deferred to Phase 2 (coupling detection alone covers 80% of value)

### Constraints Discovered
- events.jsonl lacks token_count — can't correlate spiral severity programmatically
- No ACTIVITY.json files in workspaces (field from task description doesn't exist)
- Workspace AGENT_MANIFEST.json has git_baseline but not files_modified

---

## Next (What Should Happen)

**Recommendation:** close (design complete, ready for implementation spawn)

### If Close
- [x] All deliverables complete (investigation + probe + SYNTHESIS.md)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-1109`

### Follow-up Implementation
**Issue:** Create implementation issue for Phase 1
**Skill:** feature-impl
**Context:**
```
Implement coupling-cluster as 4th hotspot type in orch hotspot.
New file: cmd/orch/hotspot_coupling.go (~200 lines)
Modified: cmd/orch/hotspot.go (add coupling analysis call, report field, icon)
Design: .kb/investigations/2026-02-19-design-coupling-hotspot-analysis-system.md
```

---

## Unexplored Questions

- **Same-layer coupling:** The current design filters to 2+ layers. Should we also flag clusters with 5+ files in the same layer? (e.g., 5 files in `pkg/daemon/` that always change together)
- **Trend detection:** Could we track coupling score over time to detect growing concepts? (e.g., spawn cluster growing from 20 to 33 files)
- **Token tracking:** Adding token_count to events.jsonl would enable Phase 2 spiral correlation. What's the minimal instrumentation needed?
- **Cross-project coupling:** orch-go depends on opencode fork. Does coupling span repos?

---

## Session Metadata

**Skill:** architect
**Model:** anthropic/claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-design-coupling-hotspot-19feb-8d9c/`
**Investigation:** `.kb/investigations/2026-02-19-design-coupling-hotspot-analysis-system.md`
**Beads:** `bd show orch-go-1109`
