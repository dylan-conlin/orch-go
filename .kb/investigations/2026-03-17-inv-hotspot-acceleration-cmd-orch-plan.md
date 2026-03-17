## Summary (D.E.K.N.)

**Delta:** `cmd/orch/plan_hydrate.go` hotspot (+320 lines/30d) is a false positive — 93% of the churn (298/320 insertions) is birth churn from initial file creation. File is 250 lines (well below 1,500 accretion boundary) and stabilizing.

**Evidence:** Git log shows exactly 2 commits: `fa7f8409f` (298 insertions, file creation) and `02bb68063` (22 ins, 70 del, refactor/cleanup). The "320 lines/30d" metric sums raw insertions across both commits. Net size is 250 lines after the refactor pass already reduced it by 48 lines.

**Knowledge:** This is the 5th consecutive false-positive hotspot from the same root cause: the churn metric counts file creation as growth. Files born within the 30-day window are indistinguishable from files that are genuinely accumulating. The file has clean separation (250 lines, single responsibility: plan hydration), existing tests (189 lines), and no architectural concerns.

**Next:** No extraction needed. Recommend improving hotspot detection to exclude birth churn (files created within the measurement window should have their initial commit excluded from the churn calculation).

**Authority:** operational - no architectural changes needed

---

# Investigation: plan_hydrate.go Hotspot Acceleration

**Question:** Is `cmd/orch/plan_hydrate.go` (+320 lines/30d, now 250 lines) genuinely accumulating and in need of extraction?

**Started:** 2026-03-17
**Updated:** 2026-03-17
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 876dcb1a8 - plist_test.go hotspot false positive (birth churn) | confirms | yes | - |
| cad9536aa - thread_test.go hotspot false positive (birth churn) | confirms | yes | - |
| ace78d1c1 - preview_test.go hotspot false positive (birth churn) | confirms | yes | - |
| 59dec8ef7 - control_cmd.go hotspot false positive (birth churn) | confirms | yes | - |

## Finding 1: Git History Analysis

**What I tested:** Examined full git history of `cmd/orch/plan_hydrate.go` to decompose the "320 lines/30d" churn metric.

**What I observed:**
- Commit `fa7f8409f` (initial): 298 insertions, 0 deletions — file creation
- Commit `02bb68063` (refactor): 22 insertions, 70 deletions — lifecycle enforcement cleanup
- Total insertions: 320 (matches the hotspot metric exactly)
- Net result: 250 lines (298 + 22 - 70)

**Conclusion:** 93% of measured churn is the initial file creation. The remaining 7% is a refactor that *reduced* the file by 48 net lines.

## Finding 2: File Structure Assessment

**What I tested:** Read the full file and checked the plan command area.

**What I observed:**
- `plan_hydrate.go`: 250 lines — single responsibility (hydrate plan phases into beads issues)
- `plan_cmd.go`: 368 lines — plan show/status/create commands
- `plan_hydrate_test.go`: 189 lines — tests for hydration
- `plan_cmd_test.go`: 387 lines — tests for plan commands
- Total plan area: 1,194 lines across 4 files

**Conclusion:** Well-structured. Each file has clear responsibility. No function exceeds ~40 lines. The plan subsystem is 1,194 lines across 4 files — healthy distribution with no single file near the 1,500 line boundary.

## Finding 3: Pattern — 5th Consecutive Birth Churn False Positive

This is the 5th hotspot investigation in a row that concluded "false positive — birth churn":

1. `plist_test.go` — birth churn
2. `pkg/thread/thread_test.go` — birth churn (570 lines from creation)
3. `preview_test.go` — birth churn (56% extraction birth)
4. `control_cmd.go` — birth churn (delete/recreate)
5. `plan_hydrate.go` — birth churn (93% from creation) ← this investigation

**Root cause:** The hotspot detection algorithm counts raw insertions within a 30-day window. Files created within that window have their entire initial content counted as "growth." This is a systematic measurement flaw, not an edge case.

## Test Performed

```bash
# Verified exact churn decomposition
git log --oneline --follow -- cmd/orch/plan_hydrate.go
# Output: 2 commits

git log --since="30 days ago" --format="%H" -- cmd/orch/plan_hydrate.go | \
  xargs -I {} git diff --stat {}^..{} -- cmd/orch/plan_hydrate.go
# Output:
#   fa7f8409f: 298 insertions
#   02bb68063: 22 insertions, 70 deletions

wc -l cmd/orch/plan_hydrate.go
# Output: 250 lines
```

## Conclusion

**False positive.** No extraction needed. The file is 250 lines, well-structured, has tests, and the measured churn is entirely from file creation + one cleanup pass. The file is *stabilizing*, not accumulating.

**Systemic recommendation:** The hotspot detection system should exclude birth churn — when a file was created within the measurement window, subtract the initial commit's insertions from the churn total. This would eliminate the most common class of false positives (5 of the last 5 hotspot investigations).
