# Session Synthesis

**Agent:** og-debug-ghost-agents-cross-28feb-f2b1
**Issue:** orch-go-xptz
**Outcome:** success

---

## Plain-Language Summary

Ghost agents from other projects (toolshed, scs-special-projects, orch-knowledge) appeared in `orch status` because they had stale `orch:agent` labels in their beads databases — left behind by agents that died without cleanup. The cross-project visibility system (`kb projects list` + `queryTrackedAgents`) correctly found these issues across 19 registered projects, but `orch abandon` couldn't manage them because it only queried local beads. Fixed by adding automatic cross-project resolution to `orch abandon` and a new `orch clean --ghosts` command that removes stale `orch:agent` labels across all registered projects.

## Verification Contract

See `VERIFICATION_SPEC.yaml` for reproducible verification steps.

Key outcomes:
- `orch status --all` no longer shows the 5 ghost agents (ok-wn64, scs-sp-45z, toolshed-3fn, toolshed-blx, toolshed-anr)
- `orch abandon <cross-project-id>` auto-resolves project dir without `--workdir`
- `orch clean --ghosts` finds and removes stale cross-project orch:agent labels
- All tests pass: `go test ./cmd/orch/ ... ok (3.059s)`, `go test ./pkg/beads/... ok (0.015s)`

---

## TLDR

Cross-project ghost agents in `orch status` were caused by stale `orch:agent` labels on beads issues in other projects. Added auto-resolution to `orch abandon` and `orch clean --ghosts` to clean them up.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/abandon_cmd.go` - Auto-resolve cross-project workdir when local beads lookup fails
- `cmd/orch/clean_cmd.go` - Added `--ghosts` flag and `cleanGhostAgents()` for cross-project ghost cleanup
- `cmd/orch/shared.go` - Added `resolveProjectDirForBeadsID()` helper
- `pkg/beads/client.go` - Added `FallbackRemoveLabelInDir()` for cross-project label operations
- `cmd/orch/clean_test.go` - Fixed import (pre-existing issue from parallel agent)

---

## Evidence (What Was Observed)

- `orch status --all` showed 8 agents, 5 from other projects (ok-wn64, scs-sp-45z, toolshed-3fn, toolshed-blx, toolshed-anr)
- All 5 ghosts had `orch:agent` label + `status: in_progress` in their respective project beads
- `orch abandon toolshed-3fn` failed: "The issue ID suggests it belongs to project 'toolshed', but you're in 'orch-go'"
- `kb projects list` returns 19 projects — the cross-project scan discovers issues across all of them
- Root cause: agents died without running lifecycle cleanup (orch:agent label never removed)

### Tests Run
```bash
go test ./cmd/orch/ -count=1 -timeout 60s
# ok  github.com/dylan-conlin/orch-go/cmd/orch  3.059s

go test ./pkg/beads/... -count=1
# ok  github.com/dylan-conlin/orch-go/pkg/beads  0.015s

go vet ./cmd/orch/
# (no output - clean)
```

### Smoke Tests
```bash
# Dry-run found all 5 ghosts
orch clean --ghosts --dry-run
# Ghost: ok-wn64 in orch-knowledge, scs-sp-45z in scs-special-projects, toolshed-3fn/blx/anr in toolshed

# Auto-resolve abandon worked
orch abandon toolshed-3fn
# Auto-resolved cross-project issue: toolshed-3fn in /Users/.../toolshed
# Removed orch:agent label, Cleared assignee, Killed tmux, Deleted session

# Ghost cleanup worked
orch clean --ghosts
# Cleaned ghost: ok-wn64, scs-sp-45z, toolshed-blx, toolshed-anr (4 remaining)

# Verification: no ghosts remain
orch status --all | grep -E "ok-wn64|scs-sp-45z|toolshed-3fn|toolshed-blx|toolshed-anr"
# No ghost agents found
```

---

## Architectural Choices

### Auto-resolve via iterating kb projects vs beads ID prefix matching
- **What I chose:** Iterate all kb projects, trying `FallbackShowWithDir` in each
- **What I rejected:** Parse beads ID prefix to infer project name
- **Why:** Beads ID prefixes don't always match directory basenames (e.g., `ok-wn64` → `orch-knowledge`, `scs-sp-45z` → `scs-special-projects`). Iterating is O(n) but n=19 and each call is fast.
- **Risk accepted:** Slightly slower for projects at the end of the list; acceptable given N is small

### Ghost cleanup as separate --ghosts flag vs folding into --orphans
- **What I chose:** Separate `--ghosts` flag on `orch clean`
- **What I rejected:** Extending `--orphans` to handle cross-project
- **Why:** Ghosts and orphans have different root causes and cleanup actions. Orphans are local agents with no execution; ghosts are cross-project issues with stale labels. Keeping them separate makes the semantics clear.
- **Risk accepted:** One more flag to remember; mitigated by `--all` including both

---

## Knowledge (What Was Learned)

### Constraints Discovered
- Cross-project agent discovery (via `queryTrackedAgents` + `getKBProjectsFn`) creates a visibility scope that exceeds the management scope of per-project commands like `orch abandon`
- Beads ID prefixes are NOT reliable for project name inference (ok ≠ opencode-knowledge, scs-sp ≠ scs-special-projects)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Bug reproduction verified fixed
- [x] Ready for `orch complete orch-go-xptz`

---

## Unexplored Questions

- **Why do agents die without cleanup?** The root cause of ghosts is agents that crash/exit before the lifecycle manager removes `orch:agent`. This fix treats the symptom (stale labels) not the cause (unclean shutdown). A daemon-level heartbeat monitor could detect dead agents and trigger cleanup automatically.
- **Should daemon auto-run ghost cleanup?** The daemon could periodically run `cleanGhostAgents` to prevent ghost accumulation. Currently requires manual `orch clean --ghosts`.

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-debug-ghost-agents-cross-28feb-f2b1/`
**Beads:** `bd show orch-go-xptz`
