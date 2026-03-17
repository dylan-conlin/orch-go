---
Status: Complete
Question: Is cmd/orch/spawn_cmd.go a hotspot requiring extraction?
Date: 2026-03-17
---

**TLDR:** spawn_cmd.go is healthy at 542 lines after a major extraction 30 days ago. The "+1043 lines/30d" metric reflects gross churn, not net growth. No extraction needed.

## D.E.K.N. Summary

- **Delta:** spawn_cmd.go was extracted from 1171→505 lines ~30 days ago. Since then it's grown modestly to 542 lines (+37 net). The "+1043 lines/30d" hotspot metric counts gross additions (including features added then removed), not net change.
- **Evidence:** `git log --numstat --since="30 days ago"` shows 373 lines added, 782 deleted = net -409 lines. The extraction commit (8c866e395) accounts for -684 deletions alone.
- **Knowledge:** Gross-addition hotspot metrics can be misleading for recently-extracted files. Post-extraction churn (new features, removals) inflates gross counts while net growth remains low. Growth rate is ~1.2 lines/day — 2+ years to reach 1500-line critical threshold.
- **Next:** No extraction needed. Re-evaluate if file reaches 800 lines.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
| --- | --- | --- | --- |
| N/A — novel investigation | - | - | - |

## Findings

### Finding 1: Prior extraction was effective

The file was extracted on commit `8c866e395` ("refactor: extract spawn_cmd.go (1171→505 lines) into 4 cohesive files"). This created:

- `cmd/orch/spawn_helpers.go` (209 lines) — config loading, spawn mode, scaffolding
- `cmd/orch/spawn_dryrun.go` (272 lines) — dry-run validation, formatting
- `pkg/orch/spawn_pipeline.go` (463 lines) — pipeline step functions
- `pkg/orch/spawn_preflight.go` (173 lines) — pre-flight gates
- `pkg/orch/spawn_types.go` (87 lines) — shared types
- Plus other `pkg/orch/spawn_*.go` files

Total spawn family: 1487 lines across 6 files in `cmd/orch/` (including tests).

### Finding 2: Post-extraction growth is modest

18 commits touched spawn_cmd.go since extraction. Net effect: +37 lines (505→542).

Notable additions:
- `--explore` flag and validation (+23 lines)
- `--architect-ref` design injection (+12 lines)
- Triage label removal race fix (+10 lines)

Notable removals:
- Ceremonial spawn gates (-22 lines, commit 185dd2f06)
- Advisory gates (-8 lines, commit 7d5531463)

### Finding 3: File structure is well-decomposed

The 542-line file breaks down as:
- Flag variables: ~42 lines (28-69)
- Cobra command definition: ~90 lines (74-164) — includes long help text
- Flag registration init(): ~40 lines (166-204)
- Main pipeline `runSpawnWithSkillInternal`: ~328 lines (214-542)

The main pipeline is already delegated through 11 `orch.*` helper calls. The cmd-layer code is primarily:
- Flag validation (~60 lines)
- Struct building for input/context/resolved-settings (~100 lines)
- Cross-repo/triage label handling (~30 lines)
- Hotspot adapter function (~25 lines)

### Finding 4: The hotspot metric is calibrated for gross, not net

The "+1043 lines/30d" counts total lines added across all commits. This is useful for detecting high-churn files but misleading for files that recently underwent extraction. The actual net change was -409 lines (373 added, 782 deleted).

## Test Performed

```bash
# Verified current file size
wc -l cmd/orch/spawn_cmd.go
# 542 lines

# Verified git stats over 30 days
git log --oneline --numstat --since="30 days ago" -- cmd/orch/spawn_cmd.go
# 18 commits, net -409 lines (extraction dominant)

# Verified spawn family total
wc -l cmd/orch/spawn_*.go
# 1487 total across 6 files

# Verified extraction commit
git log --oneline --grep="extract spawn_cmd" --since="30 days ago"
# 8c866e395 refactor: extract spawn_cmd.go (1171→505 lines) into 4 cohesive files
```

## Conclusion

**No extraction needed.** spawn_cmd.go is at a healthy 542 lines, well below the 800-line warning threshold and far from the 1500-line critical threshold. The prior extraction was effective, and post-extraction growth is modest (~1.2 lines/day). The "+1043 lines/30d" metric was a false positive triggered by gross churn, not actual file bloat.

**Recommendation:** Re-evaluate at 800 lines. If growth accelerates, the most extractable section would be flag validation (lines 216-274, ~60 lines) into a `validateSpawnFlags()` function in `spawn_helpers.go`.
