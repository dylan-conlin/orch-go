# Probe: Dirty Worktree Reconciliation — Closed Issues with Uncommitted Code

**Model:** completion-verification
**Date:** 2026-03-26
**Status:** Complete
**claim:** n/a (system-level probe, no single claim ID)
**verdict:** extends

---

## Question

The completion-verification model describes three code paths with divergent verification state and a "verification signal bypass from non-human paths." Does the dirty worktree contain concrete instances where completed/closed issues left behind uncommitted code that bypasses verification — and in at least one case, breaks the build?

---

## What I Tested

Full dirty-worktree audit against beads issue status. For each dirty tracked file (excluding .orch/workspace and experiments/), identified: what changed, which issue produced it, and whether that issue is open or closed.

```bash
# 1. Enumerate dirty tracked files (excluding workspace/experiments)
git diff --name-only HEAD -- ':(exclude).orch/workspace' ':(exclude)experiments/' | sort

# 2. For each dirty code file, read the diff
git diff HEAD -- <file>

# 3. Cross-reference against beads
bd show <issue-id>  # for each referenced issue

# 4. Verify build state of committed code (stash dirty, build, pop)
git stash && go build ./cmd/orch/ 2>&1; git stash pop

# 5. Verify working tree compiles and routing tests pass
go build ./cmd/orch/
go test ./pkg/daemon/ -run "TestRouteModel|TestSkillCapability|TestInferEffort|TestRouteIssue"
```

---

## What I Observed

### BUILD-BREAKING: Committed code does not compile

Commit `da9b666b4` (orch-go-r7avo, **closed**) modified `ooda.go` and `preview.go` to call:
- `RouteModel(skill, selected)` — function only exists in dirty `skill_inference.go`
- `d.RouteIssueForSpawn(issue, skill, model, reason)` (4 args) — committed signature only accepts 3 args
- `route.ModelRouteReason` — field only exists in dirty `coordination.go`

Build errors on committed code:
```
pkg/daemon/ooda.go:237:16: undefined: RouteModel
pkg/daemon/ooda.go:250:72: too many arguments in call to d.RouteIssueForSpawn
pkg/daemon/ooda.go:266:36: route.ModelRouteReason undefined
pkg/daemon/preview.go:162:16: undefined: RouteModel
```

**Root cause:** Agent orch-go-r7avo committed caller changes (ooda.go, preview.go) but NOT the implementation changes (skill_inference.go, coordination.go, allocation.go). Issue was marked closed. Build broke silently because no CI runs.

### Full Reconciliation Map

| File | Change | Source Issue | Issue Status | Disposition |
|------|--------|-------------|--------------|-------------|
| **Cluster 1: Capability routing (BUILD-BREAKING)** | | | | |
| `pkg/daemon/skill_inference.go` | RouteModel, SkillCapability, ModelRoute impl | orch-go-r7avo / orch-go-19k8o | r7avo: closed, 19k8o: open | **COMMIT** — fixes build |
| `pkg/daemon/skill_inference_test.go` | Tests for capability routing (all pass) | orch-go-r7avo / orch-go-19k8o | same | **COMMIT** — fixes build |
| `pkg/daemon/coordination.go` | ModelRouteReason field, 4-arg RouteIssueForSpawn | orch-go-r7avo | closed | **COMMIT** — fixes build |
| `pkg/daemon/coordination_test.go` | Updated test calls for 4-arg signature | orch-go-r7avo | closed | **COMMIT** — fixes build |
| `pkg/daemon/allocation.go` | Uses RouteModel instead of InferModelFromSkill | orch-go-r7avo / orch-go-19k8o | same | **COMMIT** — fixes build |
| **Cluster 2: Backend verification removal** | | | | |
| `pkg/verify/backend.go` | DELETED (imports pkg/opencode) | orch-go-8l4h9 | closed | **COMMIT** — migration cleanup |
| `pkg/verify/check.go` | Remove serverURL from signatures | orch-go-8l4h9 | closed | **COMMIT** — migration cleanup |
| `pkg/verify/check_test.go` | Updated test calls | orch-go-8l4h9 | closed | **COMMIT** — migration cleanup |
| `cmd/orch/complete_verification.go` | Remove serverURL from calls | orch-go-8l4h9 | closed | **COMMIT** — migration cleanup |
| `pkg/daemon/compliance.go` | Remove serverURL from call | orch-go-8l4h9 | closed | **COMMIT** — migration cleanup |
| `cmd/orch/doctor_defect_scan.go` | Remove deleted funcs from allowlist | orch-go-8l4h9 | closed | **COMMIT** — migration cleanup |
| `cmd/orch/doctor_defect_scan_test.go` | Update ListSessions test signatures | orch-go-8l4h9 | closed | **COMMIT** — migration cleanup |
| **Cluster 3: Comprehension mark-as-read** | | | | |
| `cmd/orch/serve_briefs.go` | Add comprehension:processed label removal | orch-go-c29fl | open | **LEAVE DIRTY** — belongs to open issue |
| **Cluster 4: Knowledge artifacts** | | | | |
| `.kb/decisions/...plan-mode...` | Auto-linked investigation | orch-go-7jfi8 | closed | **COMMIT** — auto-link residue |
| `.kb/global/models/signal-to-design-loop.md` | Extended model (physical realizability gap) | cross-project | n/a | **COMMIT** — knowledge update |
| `.kb/models/cc-agent-config/model.md` | Auto-linked investigation | orch-go-1dhv8 | closed | **COMMIT** — auto-link residue |
| `.kb/quick/entries.jsonl` | New constraint (core/substrate ratio) | orch-go-wgkj4 | closed | **COMMIT** — knowledge residue |
| `.kb/threads/...plan-lifecycle...` | Auto-linked investigation | various | mixed | **COMMIT** — thread evolution |
| `.kb/threads/...openclaw...` | Added resolved_by field | various | mixed | **COMMIT** — thread evolution |
| `.kb/threads/...threads-primary...` | 2026-03-25/26 entries, title update | various | mixed | **COMMIT** — thread evolution |
| **Cluster 5: Skill templates** | | | | |
| `skills/src/meta/orchestrator/.skillc/SKILL.md.template` | Frustration protocol update | various | n/a | **COMMIT** — skill improvement |
| `skills/src/worker/architect/.skillc/SKILL.md.template` | Composition claims, consequence sensor | various | n/a | **COMMIT** — skill improvement |
| `skills/src/*/stats.json` (11 files) | Compilation metadata | auto-generated | n/a | **COMMIT** — compilation artifacts |
| **Cluster 6: Beads state** | | | | |
| `.beads/issues.jsonl` | Local beads state | continuous | n/a | **LEAVE DIRTY** — updated by bd CLI |

### Issue Status Corrections Needed

1. **orch-go-r7avo** (closed) — Prematurely closed. Committed partial work that broke the build. The dirty skill_inference.go and coordination.go are the missing pieces. Once committed, the closure is retroactively valid.

2. **orch-go-19k8o** (open/parked) — The capability routing in skill_inference.go IS the implementation this issue requested. After committing Cluster 1, this issue should be updated to reflect that core implementation landed (via r7avo spillover). Remaining work: integration testing of the GPT-5.4 routing path.

3. **orch-go-8l4h9** (closed) — Closed correctly for the main migration. Cluster 2 is follow-on cleanup (removing the now-unnecessary backend verification layer). Not a status error — just residue.

---

## Model Impact

- [x] **Extends** model with: A new failure mode — "split commit" — where an agent commits callers but not implementations. This is invisible without CI because `go build` is not gated. The completion-verification model's "verification signal bypass from non-human paths" probe identified that daemon-triggered completions skip human review; this is a concrete instance where a daemon-routed agent (orch-go-r7avo) closed with a broken build because no gate checked compilation of the commit.

**New model claim (proposed):** Completion verification should include a "committed code compiles" gate. The current gates verify SYNTHESIS.md, Phase: Complete, accretion, and decision lifecycle — but none verify that the committed code builds. This is the most basic form of verification signal that is currently absent.

---

## Notes

- The working tree is internally consistent (compiles, tests pass). The problem is the committed state, not the dirty state.
- Cluster 2 (serverURL removal) does NOT break the committed build — it's a separate, non-urgent cleanup.
- `serve_briefs.go` (Cluster 3) is correctly left dirty — it's an in-progress experiment tied to the open comprehension question orch-go-c29fl.
- 11 stats.json files are skillc compilation artifacts that accumulate naturally.
