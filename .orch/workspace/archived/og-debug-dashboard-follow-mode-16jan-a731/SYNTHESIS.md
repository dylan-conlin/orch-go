# Session Synthesis

**Agent:** og-debug-dashboard-follow-mode-16jan-a731
**Issue:** orch-go-mg741
**Duration:** 2026-01-16T18:06 → 2026-01-16T18:15
**Outcome:** success

---

## TLDR

Fixed dashboard follow mode not showing price-watch agents by adding "pw" to included projects mapping. The directory name ("price-watch") didn't match the beads ID prefix ("pw"), so agents were filtered out.

---

## Delta (What Changed)

### Files Created
- `pkg/tmux/follower_test.go` - Unit test for GetIncludedProjects function

### Files Modified
- `pkg/tmux/follower.go` - Added MultiProjectConfig for price-watch→pw mapping, also added price-watch/pw to orch-go's included projects

### Commits
- (pending) - fix: add price-watch/pw mapping to dashboard follow mode

---

## Evidence (What Was Observed)

- Context API (`/api/context`) returns `project: "price-watch"` from directory basename (serve_context.go:94)
- Agents have `Project: "pw"` from `extractProjectFromBeadsID("pw-xxxx")` (shared.go:130-142)
- Dashboard filter compared "pw" against ["price-watch"] → no match
- Existing pattern in DefaultMultiProjectConfigs handles this for orch-go with multiple repos

### Tests Run
```bash
# Build
make build
# PASS: Building orch... (no errors)

# Unit tests
go test -v -run TestGetIncludedProjects ./pkg/tmux/
# PASS: price-watch_includes_pw_alias
# PASS: orch-go_includes_ecosystem_repos_and_price-watch
# PASS: unknown_project_returns_just_itself
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-16-inv-dashboard-follow-mode-project-mismatch.md` - Root cause analysis

### Decisions Made
- Use existing MultiProjectConfig pattern: simple config change following proven pattern
- Add mapping both ways: price-watch includes pw, orch-go includes both

### Constraints Discovered
- When directory name differs from beads ID prefix, explicit mapping required in DefaultMultiProjectConfigs()

### Externalized via `kn`
- None needed (tactical config fix, documented in investigation)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-mg741`

---

## Unexplored Questions

Straightforward session, no unexplored territory.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** opus
**Workspace:** `.orch/workspace/og-debug-dashboard-follow-mode-16jan-a731/`
**Investigation:** `.kb/investigations/2026-01-16-inv-dashboard-follow-mode-project-mismatch.md`
**Beads:** `bd show orch-go-mg741`
