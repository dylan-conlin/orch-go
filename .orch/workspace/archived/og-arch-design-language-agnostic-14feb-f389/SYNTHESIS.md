# Session Synthesis

**Agent:** og-arch-design-language-agnostic-14feb-f389
**Issue:** orch-go-foy
**Duration:** 2026-02-14 → 2026-02-14
**Outcome:** success

---

## TLDR

Designed language-agnostic accretion metrics for cross-project orchestration. Confirmed via spike that uniform 800/1500 line thresholds work correctly across Go and TypeScript — language-specific tuning is unnecessary. Recommended two-phase implementation: (1) enrich event schema with project_dir + add generated file exclusion + add 4 metric primitives, (2) add --all-projects flag to orch hotspot + dashboard health panel.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-14-design-language-agnostic-accretion-metrics.md` - Full design investigation with 5 navigated forks, implementation plan, acceptance criteria
- `.kb/models/completion-verification/probes/2026-02-14-language-agnostic-accretion-metrics.md` - Probe confirming uniform thresholds work cross-language

### Files Modified
- None (design-only session)

### Commits
- (to be committed)

---

## Evidence (What Was Observed)

- **orch-go (Go):** Largest authored files: spawn_cmd.go (2,173), doctor.go (1,909), complete_cmd.go (1,847) — all correctly flagged by 800/1500 thresholds
- **opencode (TypeScript):** Largest file types.gen.ts (5,065) — false positive, generated code. Authored TS files well below 800.
- **beads (Go):** server_issues_epics.go (2,020), queries.go (1,893) — correctly flagged
- **Portfolio composition:** 95%+ Go/TypeScript across 10+ .orch/-enabled projects
- **AccretionDeltaData schema:** Missing `ProjectDir` and `ProjectName` fields, blocking cross-project aggregation
- **Generated file gap:** `isSourceFile()` has no concept of generated code; only false positive source found

### Tests Run
```bash
# Cross-language threshold validation spike
find ~/Documents/personal/orch-go -name "*.go" | xargs wc -l | sort -rn | head -15
find ~/Documents/personal/opencode -name "*.ts" -not -path "*/node_modules/*" | xargs wc -l | sort -rn | head -15
find ~/Documents/personal/beads -name "*.go" | xargs wc -l | sort -rn | head -15
# Result: Uniform thresholds correctly flag structural issues in both Go and TS
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-14-design-language-agnostic-accretion-metrics.md` - Design for cross-project accretion metrics with 4 new metric primitives
- `.kb/models/completion-verification/probes/2026-02-14-language-agnostic-accretion-metrics.md` - Confirmed uniform thresholds are language-agnostic

### Decisions Made
- **Uniform thresholds over language-specific:** 800/1500 works for all languages in portfolio. Over-engineering to add per-language config.
- **Enrich events.jsonl over new metrics store:** Adding ProjectDir field enables cross-project aggregation with zero new infrastructure. Follows Local-First principle.
- **--all-projects flag over separate command:** Compose Over Monolith — extend orch hotspot, don't add commands.
- **Pattern-based generated file exclusion:** *.gen.*, gen/, dist/ as defaults with .orch/config.yaml overrides.

### Constraints Discovered
- AccretionDeltaData events have never been emitted in production (grep of events.jsonl returned empty)
- Generated files are the only cross-language false positive source (types.gen.ts at 5,065 lines)
- Cross-project scanning requires bloat-only analysis (fix-commit and investigation signals are CWD-specific)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + probe)
- [x] Investigation file has `**Phase:** Complete`
- [x] Probe file has `**Status:** Complete`
- [ ] Ready for `orch complete orch-go-foy`

### Implementation Follow-up

Two phases recommended:

**Phase 1 (implementation-level, ~2h):** Enrich AccretionDeltaData schema with ProjectDir/ProjectName, add generated file exclusion patterns to hotspot, add four metric primitives to HotspotReport.

**Phase 2 (architectural-level, ~4h):** Add --all-projects flag to orch hotspot, add /api/hotspot/all endpoint, add dashboard cross-project health panel.

---

## Verification Contract

**Probe:** `.kb/models/completion-verification/probes/2026-02-14-language-agnostic-accretion-metrics.md`
**Investigation:** `.kb/investigations/2026-02-14-design-language-agnostic-accretion-metrics.md`

Key outcomes verified:
- Uniform thresholds confirmed correct via cross-project spike (3 projects, 2 languages)
- Schema gap identified (missing ProjectDir in AccretionDeltaData)
- 5 decision forks navigated with substrate traces
- Implementation plan with file targets and acceptance criteria

---

## Unexplored Questions

- **Accretion velocity trending:** How to visualize accretion_velocity over time on the dashboard? Time-series charts require frontend work not scoped here.
- **Coaching plugin accretion detection:** The prior investigation designed this but it's not yet implemented. Would real-time warnings change agent behavior?
- **Cross-project extraction prioritization:** If multiple projects have high extraction_debt, which should be extracted first? Needs orchestrator judgment.

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-arch-design-language-agnostic-14feb-f389/`
**Investigation:** `.kb/investigations/2026-02-14-design-language-agnostic-accretion-metrics.md`
**Beads:** `bd show orch-go-foy`
