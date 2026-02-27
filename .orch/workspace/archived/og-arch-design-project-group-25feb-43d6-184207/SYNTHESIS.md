# Session Synthesis

**Agent:** og-arch-design-project-group-25feb-43d6
**Issue:** orch-go-1235
**Outcome:** success

---

## Plain-Language Summary

Designed a "project group" model that solves three problems at once: (1) when an agent spawns into a work project like toolshed, it can't see kb artifacts from sibling projects like price-watch because the global kb search is hardcoded to only include orch-ecosystem repos; (2) scs-special-projects is the coordination parent for SCS work but orch-go has no concept of parent-child project relationships; (3) the daemon polls all 19 registered projects flat with no way to scope by group or route different accounts. The design introduces a `~/.orch/groups.yaml` config with two group types: explicit groups (list project names, like the orch ecosystem) and parent-inferred groups (name a parent project, children auto-discovered from directory paths). Groups cascade to kb context filtering (replacing the hardcoded `OrchEcosystemRepos` allowlist), daemon polling (new `--group` flag), account routing (per-group account field), and dashboard filtering (new group query parameter).

## Verification Contract

See `VERIFICATION_SPEC.yaml` — architect design with 5 acceptance criteria covering all three needs, config format, cascade documentation, and implementation readiness.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-02-25-design-project-group-model.md` — Full architect investigation with 5 navigated forks, recommendations, and implementation-ready output
- `.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-project-group-model-design.md` — Probe confirming/extending daemon model with group scoping findings

### Files Modified
- None (architect design, no code changes)

### Commits
- Pending

---

## Evidence (What Was Observed)

- `OrchEcosystemRepos` in `pkg/spawn/kbcontext.go:15-22` is a hardcoded 6-project allowlist that blocks all SCS cross-project context
- `ProjectRegistry` in `pkg/daemon/project_resolution.go` builds flat `prefixToDir` map from `kb projects list` — no grouping
- `kb projects list --json` returns 19 projects; SCS projects share parent path `scs-special-projects/`, enabling path-based inference
- Decision `2026-01-16-single-daemon-orchestration-home` is STALE — code already implements cross-project polling via `ListReadyIssuesMultiProject`
- Daemon has NO account routing — single global account for all spawns

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-02-25-design-project-group-model.md` — Project group model design
- `.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-project-group-model-design.md` — Probe extending daemon model

### Decisions Made
- Hybrid group resolution (path inference + explicit config) recommended over pure-explicit or pure-convention
- `~/.orch/groups.yaml` as single config location (not per-project)
- Per-group account routing (not per-project)

### Constraints Discovered
- `OrchEcosystemRepos` is a proxy for "project group" but only serves one group — generalization needed
- Orch ecosystem can't use path inference (orch-knowledge at ~/orch-knowledge, blog in same parent as orch-go)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + probe)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-1235`

### Implementation Follow-up
When promoted to decision, implementation should be split into:
1. `pkg/group/group.go` — Group model, config loading, resolution
2. Replace `OrchEcosystemRepos` in `kbcontext.go` with group-based filter
3. Add `--group` flag to daemon
4. Account routing in daemon spawn
5. Dashboard group filter

### Discovered Work
- Stale decision `2026-01-16-single-daemon-orchestration-home` should be superseded (code already implements cross-project polling)

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5
**Workspace:** `.orch/workspace/og-arch-design-project-group-25feb-43d6/`
**Investigation:** `.kb/investigations/2026-02-25-design-project-group-model.md`
**Beads:** `bd show orch-go-1235`
