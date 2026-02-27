# Session Synthesis

**Agent:** og-arch-analyze-consequences-cross-25feb-9ec2
**Issue:** untracked
**Outcome:** success

---

## Plain-Language Summary

The cross-repo orchestration decision (strategic issues in scs-special-projects, implementation in child repos) is **mostly supported by existing orch-go infrastructure** — the daemon can already poll multiple projects and spawn with `--workdir`, and the dashboard API already accepts `project_dir` query params. However, there are two **immediate operational fixes** (register scs-sp in kb, set issue prefix) and one **significant architectural gap** (completion processing is single-project only, so the daemon can spawn cross-project agents but can't detect when they complete).

The full gap analysis produced 4 tiers of issues. P0 gaps (2 items) are pure configuration — fixable in minutes. P1 gaps (2 items) are the real engineering work: extending completion processing to scan all registered projects, and making `orch complete` cross-project aware. P2/P3 gaps (5 items) are visibility and maturity improvements that can wait.

---

## Verification Contract

Probe file: `.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-cross-repo-orchestration-consequences.md`

Key findings:
- 7 components that already work for cross-repo orchestration
- 8 gaps identified, prioritized P0-P3
- Critical asymmetry: daemon can POLL+SPAWN cross-project but not COMPLETE cross-project

---

## Delta (What Changed)

### Files Created
- `.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-cross-repo-orchestration-consequences.md` — Full analysis probe with code evidence

### Files Modified
- None (analysis only)

---

## Evidence (What Was Observed)

1. **scs-special-projects is NOT in `kb projects list`** — 18 projects registered, parent repo missing. Child repos (price-watch, toolshed, specs-platform, sendassist) are registered.

2. **scs-special-projects has no explicit issue-prefix** — `.beads/config.yaml` has `issue-prefix` commented out. Would default to `scs-special-projects` (29 chars).

3. **Daemon already supports multi-project polling** — `project_resolution.go` builds `ProjectRegistry` from `kb projects list`, `issue_adapter.go` has `ListReadyIssuesMultiProject()` that queries all registered projects with dedup.

4. **Completion processing is single-project** — `completion_processing.go:85-137` calls `verify.ListOpenIssues()` which only checks the current project's beads database. Cross-project agents would never be detected as completed.

5. **Dashboard beads API is already project-aware** — All beads endpoints (`/api/beads`, `/api/beads/ready`, `/api/beads/graph`) accept `project_dir` query parameter and use per-project cache entries.

6. **Work-graph has no cross-project edge support** — Each graph query returns one project's nodes/edges. A strategic parent issue in scs-sp with implementation children in toolshed/pw can't be rendered as connected edges.

7. **`orch work --workdir` fix from orch-go-1230** — Sets `beads.DefaultDir` before any beads calls, verified in `spawn_cmd.go:394-407`. Cross-project beads lookups work correctly.

---

## Knowledge (What Was Learned)

### Constraints Discovered

1. **Cross-project asymmetry:** The daemon's cross-project support is half-built. Polling and spawning work across projects, but completion processing only looks at the current project. This means the daemon can START work everywhere but can only FINISH work in one place.

2. **Registration is the gate:** Everything in orch-go that works cross-project uses `kb projects list` as the source of truth. If scs-special-projects isn't registered, it doesn't exist to the system.

3. **Text-only cross-repo deps:** `bd dep add` only works within a single repo's beads. Cross-repo dependencies (parent scs-sp issue → child toolshed task) are conventions in description text, not enforceable edges. The daemon respects within-repo deps but can't honor cross-repo ones.

---

## Next (What Should Happen)

**Recommendation:** Create beads issues for the P0 and P1 gaps

### Immediate (P0 — operational, no code changes):
1. Register scs-special-projects in kb: `cd ~/Documents/work/SendCutSend/scs-special-projects && kb init`
2. Set issue prefix: Add `issue-prefix: scs-sp` to `scs-special-projects/.beads/config.yaml`

### Follow-up (P1 — code changes needed):

CROSS_REPO_ISSUE:
  repo: ~/Documents/personal/orch-go
  title: "Extend completion processing to scan all registered projects"
  type: feature
  priority: 2
  description: "ListCompletedAgentsDefault() only checks the current project's beads for Phase: Complete agents. When daemon spawns cross-project (e.g., from scs-special-projects into toolshed), completion is never detected. Need to iterate over ProjectRegistry and check each project's open/in_progress issues for Phase: Complete comments. This blocks the cross-repo orchestration workflow (decision: 2026-02-25-cross-repo-orchestration-from-parent.md)."

CROSS_REPO_ISSUE:
  repo: ~/Documents/personal/orch-go
  title: "Make orch complete cross-project aware via beads ID prefix resolution"
  type: feature
  priority: 2
  description: "orch complete currently runs verification against the current project's beads only. When completing a strategic parent issue in scs-special-projects, it should resolve the beads ID prefix to the correct project directory (using ProjectRegistry) and run verification there. This is the manual counterpart to the daemon completion gap."

---

## Unexplored Questions

- **Portfolio-level graph:** How should the dashboard render a combined view of strategic (parent) + implementation (child) issues across repos? Merged graph? Separate views with cross-links?
- **Cross-repo dep resolution:** Should `bd dep add` be extended to support cross-repo references (e.g., `toolshed-200 depends-on scs-sp-10`)? Or is text-based linking sufficient?
- **Parent auto-completion:** When all child implementation issues in toolshed/pw close, should the parent strategic issue in scs-sp be automatically marked ready-for-review?

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-analyze-consequences-cross-25feb-9ec2/`
**Probe:** `.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-cross-repo-orchestration-consequences.md`
