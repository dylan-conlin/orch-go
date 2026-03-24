# Session Synthesis

**Agent:** og-debug-daemon-routes-architect-23mar-4d20
**Issue:** orch-go-xhgvm
**Outcome:** success (already fixed)

---

## Plain-Language Summary

The daemon was correctly reading `skill:architect` from issue labels and inferring "architect" as the skill, but then dropping it at the SpawnWork boundary — `orch work` never received the skill, so it re-inferred from the issue type (`task → feature-impl`). This was already fixed today in commit 8248e8ed4 which added a `--skill` parameter to the SpawnWork interface. Code analysis confirms no secondary routing failure exists — the fix covers the only path where the skill was lost.

## Verification Contract

See `VERIFICATION_SPEC.yaml`. Key outcomes:
- All 7 observations were pre-fix (before 8248e8ed4)
- `TestOODA_SkillLabelOverridesTypeInference` covers the exact scenario
- No remaining code path drops the skill label

---

## TLDR

Bug already fixed by commit 8248e8ed4 (today). Verified through code path analysis and test execution. No secondary routing failure found — the SpawnWork skill passthrough was the sole failure point.

---

## Delta (What Changed)

### Files Created
- None (no code changes needed)

### Files Modified
- None (fix already landed)

### Commits
- `8248e8ed4` (prior agent orch-go-z5uck) — Added `--skill` parameter to SpawnWork interface

---

## Evidence (What Was Observed)

- `pkg/daemon/skill_inference.go:232-235`: `InferSkillFromIssue` correctly prioritizes `skill:*` labels over type-based inference
- `pkg/daemon/issue_adapter.go:433-454`: `SpawnWork` now passes `--skill` flag to `orch work` when skill is non-empty
- `cmd/orch/work_cmd.go:178-206`: `orch work` uses daemon-provided `--skill` flag (line 180-181) before any re-inference
- `pkg/daemon/ooda.go:165-191`: OODA Decide phase infers skill → passes through RouteIssueForSpawn → Act → spawnIssue → SpawnWork
- `pkg/daemon/coordination.go:40-101`: RouteIssueForSpawn can escalate TO architect but never downgrades architect to something else

### Routing Chain (verified correct post-fix)
1. Daemon `Decide()` → `InferSkillFromIssue(issue)` → label "skill:architect" → "architect"
2. `RouteIssueForSpawn(issue, "architect", "opus")` → preserves "architect"
3. `Act(decision)` → `spawnIssue(issue, "architect", "opus")`
4. `SpawnWork(id, "architect", "opus", workdir, account)` → `orch work --skill architect <id>`
5. `orch work` receives `--skill architect` → skips re-inference → spawns with architect

### Tests Run
```bash
go test ./pkg/daemon/ -run "TestOODA_SkillLabelOverridesTypeInference" -v
# PASS (0.16s)

go test ./pkg/daemon/ -run "TestInferSkillFromIssue|TestInferSkillFromLabels" -v
# PASS (0.45s) — all subtests pass

go test ./cmd/orch/ -run "TestInferSkillFromIssueType" -v
# PASS (0.40s)
```

---

## Architectural Choices

No architectural choices — investigation-only session. Fix was already implemented.

---

## Knowledge (What Was Learned)

### Two Daemon Spawn Paths
- **OODA path** (production): `Once()` → `OnceExcluding()` → `Sense/Orient/Decide/Act` — goes through `RouteIssueForSpawn`
- **OnceWithSlot** (tests only): bypasses `RouteIssueForSpawn` — used only in `capacity_test.go`

### `orch rework` Has Similar Type-Based Fallback
- `cmd/orch/rework_cmd.go:114-120`: Falls back to `InferSkillFromIssueType(issue.IssueType)` when no `--skill` flag and no workspace manifest skill
- This would also route `task + skill:architect` to `feature-impl`, but only in the edge case where manifest is empty
- Not the reported bug (daemon path), noted as discovered work

---

## Next (What Should Happen)

**Recommendation:** close

- [x] All observations are pre-fix
- [x] Tests passing
- [x] No secondary routing failure
- [x] Fix at 8248e8ed4 is complete and tested

---

## Unexplored Questions

- `orch rework` has similar label-unaware fallback, but mitigated by workspace manifest priority. Low risk.

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-debug-daemon-routes-architect-23mar-4d20/`
**Beads:** `bd show orch-go-xhgvm`
