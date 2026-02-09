## Summary (D.E.K.N.)

**Delta:** Branch-per-agent alone is insufficient; the safe default for 10-20+ concurrent agents is isolated git worktree per agent plus branch-per-agent inside that worktree.

**Evidence:** Code search confirms no spawn/complete git branch isolation today, and command experiments reproduced shared-tree checkout collisions plus uncommitted-change bleed while showing worktree isolation between concurrent agents.

**Knowledge:** The overnight ghost-completion failure is fundamentally a workspace isolation failure, so branch policy without filesystem isolation cannot satisfy daemon auto-complete correctness.

**Next:** Adopt the hybrid design in a phased rollout (metadata first, spawn isolation second, complete/merge third, cleanup and observability fourth) and implement the accepted decision record.

**Authority:** architectural - This changes spawn, completion verification, daemon behavior, and state model boundaries.

---

# Investigation: Git Isolation Strategy Multi Agent

**Question:** What git isolation strategy should orch-go adopt for multi-agent concurrency (worktrees vs branches vs worktrees+branches) so daemon auto-complete, Claude/GPT backends, and beads tracking remain correct?

**Started:** 2026-02-09
**Updated:** 2026-02-09
**Owner:** og-arch-git-isolation-strategy-09feb-d946
**Phase:** Complete
**Next Step:** Implement accepted phased plan from decision record
**Status:** Complete

**Patches-Decision:** `.kb/decisions/2026-02-09-git-isolation-worktree-plus-branch.md`
**Extracted-From:** N/A

## Prior Work

| Investigation                                                                              | Relationship | Verified | Conflicts                                              |
| ------------------------------------------------------------------------------------------ | ------------ | -------- | ------------------------------------------------------ |
| `.kb/investigations/archived/2025-12-23-inv-investigate-git-branching-strategies-swarm.md` | extends      | yes      | Prior recommendation missed shared working-tree hazard |
| `.kb/investigations/2026-02-09-inv-post-mortem-daemon-overnight-ghost-completions.md`      | confirms     | yes      | No conflict; root cause strengthened                   |

---

## Findings

### Finding 1: Current orchestration has no branch/worktree isolation in spawn/complete lifecycle

**Evidence:** Search found no `git checkout/switch/branch/merge/worktree` in `cmd/orch` spawn/complete pipeline; only `pkg/verify/build_blame.go` creates temporary worktrees for retrospective build attribution.

**Source:** `cmd/orch/spawn_cmd.go`, `cmd/orch/spawn_pipeline.go`, `cmd/orch/complete_pipeline.go`, `cmd/orch/complete_gates.go`, `pkg/verify/build_blame.go:147`

**Significance:** There is currently no first-class git isolation model for active agents, so concurrent workers inherently contend in one tree.

---

### Finding 2: Branch-only isolation fails under concurrent uncommitted work in one filesystem

**Evidence:** Reproduced checkout block in temp repo:
`error: Your local changes to the following files would be overwritten by checkout: file.txt`.
Also observed shared-tree visibility of uncommitted state (`git status --short` showed `M file.txt`).

**Source:** Command experiment (2026-02-09): `git checkout agent/b` from dirty `agent/a` worktree and `git status --short`.

**Significance:** Branch-per-agent without per-agent worktree cannot support true parallelism and recreates ghost-completion conditions (agents validating each other's uncommitted state).

---

### Finding 3: Worktree isolation cleanly separates concurrent uncommitted changes

**Evidence:** Two worktrees (`agent/wta`, `agent/wtb`) each showed only their own untracked files (`?? a.txt` vs `?? b.txt`) with no cross-contamination.

**Source:** Command experiment (2026-02-09): `git worktree add ...`, `git -C <wt> status --short`.

**Significance:** Worktrees are the required primitive for per-agent filesystem isolation while still sharing object storage and repository history.

---

### Finding 4: Worktree-only (detached HEAD) is operationally weak for automation

**Evidence:** Detached worktree commits have no branch name (`git branch --show-current` empty), and root refs did not contain the detached commit.

**Source:** Command experiment (2026-02-09): `git worktree add --detach`, `git branch --show-current`, `git branch --contains <commit>`.

**Significance:** Worktree-only mode increases recovery and merge ambiguity; automation should attach each worktree to an explicit branch.

---

### Finding 5: Existing verification and project-resolution paths assume single project_dir and need explicit source-vs-worktree split

**Evidence:** `resolveProjectDir()` reads `PROJECT_DIR` from workspace context for beads commands, while verify gates execute git commands against `target.BeadsProjectDir`.

**Source:** `cmd/orch/shared.go:402`, `cmd/orch/complete_pipeline.go:156`, `cmd/orch/complete_gates.go:116`, `pkg/verify/git_diff.go:218`

**Significance:** Hybrid isolation needs two explicit paths in metadata: canonical source repo (beads/db/project identity) and per-agent git worktree root (verification and merge scope).

---

## Synthesis

**Key Insights:**

1. **Root problem is filesystem isolation, not naming isolation** - Branches name histories, worktrees isolate uncommitted state.
2. **Ghost-completion postmortem and git mechanics align** - Shared working tree explains both verification bleed and branch checkout contention.
3. **Correct design is compositional** - Use worktree for concurrency safety plus branch for merge identity and lifecycle operations.

**Answer to Investigation Question:**

Adopt **worktree-per-agent + branch-per-agent** as the default isolation model, with optional branch-per-epic target in the same worktree mechanism. Reject branch-only (unsafe under concurrent dirty state) and worktree-only detached mode (operationally ambiguous). This extends the Dec 23 branching recommendation by adding the missing filesystem isolation layer required by the Feb 9 ghost-completion evidence.

---

## Structured Uncertainty

**What's tested:**

- ✅ Branch-only checkout conflict was reproduced with real git commands (dirty tree blocks switching to another branch).
- ✅ Shared tree exposed uncommitted state globally (`git status --short` showed dirty state visible at repo root).
- ✅ Separate worktrees isolated uncommitted state per agent (`wt-a` and `wt-b` had independent status outputs).

**What's untested:**

- ⚠️ End-to-end daemon auto-complete with real isolated worktrees was not prototyped in orch-go code.
- ⚠️ Epic branch fan-in behavior under 20+ simultaneous completions was not load tested.
- ⚠️ Disk growth and prune policies for long-running worktree churn need empirical thresholds.

**What would change this:**

- If production prototype shows verify gates can reliably scope changes per agent without per-agent worktree, this recommendation weakens.
- If worktree lifecycle overhead causes unacceptable failure rates in spawn/cleanup, architecture may need a queue or pool.
- If backend constraints (Claude/GPT launch paths) cannot operate from worktree directories, implementation sequencing must adjust.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation                                                             | Authority      | Rationale                                                         |
| -------------------------------------------------------------------------- | -------------- | ----------------------------------------------------------------- |
| Adopt worktree+branch hybrid isolation for all tracked agents              | architectural  | Changes core lifecycle across spawn, verify, daemon, and complete |
| Keep branch naming semantics (`agent/<beads-id>`, optional `epic/<id>`)    | implementation | Tactical convention within chosen architecture                    |
| Add merge queue/concurrency throttling only if metrics show merge pressure | architectural  | Cross-cutting operational behavior                                |

### Recommended Approach

**Hybrid Git Isolation (worktree-per-agent + branch-per-agent)** - Each agent runs in its own worktree rooted under `.orch/worktrees/<workspace>`, checked out to a dedicated branch.

**Why this approach:**

- Eliminates shared-tree uncommitted bleed that caused ghost completions.
- Preserves tractable integration via explicit branches.
- Works with both Claude and GPT backends because launch cwd is already configurable per spawn.

**Trade-offs accepted:**

- Added lifecycle complexity (create/remove worktree, branch ownership metadata).
- Need explicit cleanup and orphan recovery for abandoned sessions.

**Implementation sequence:**

1. **Metadata split and invariants** - Add `source_project_dir` and `git_worktree_dir` to manifest/state so verification and beads resolution stop conflating concerns.
2. **Spawn isolation** - Create worktree + branch before session launch; run agent in worktree cwd.
3. **Completion integration** - Verify and merge from agent branch, then prune worktree/branch on success or abandonment.
4. **Operational hardening** - Add stale worktree janitor, metrics, and guardrails for merge contention.

### Alternative Approaches Considered

**Option B: Branch-per-agent in shared working tree**

- **Pros:** Lower initial implementation cost.
- **Cons:** Fails concurrency semantics and allows verification contamination from uncommitted peers.
- **When to use instead:** Single-agent or strictly serialized workflows only.

**Option C: Worktree-only (detached HEAD)**

- **Pros:** Strong isolation with minimal branch bookkeeping.
- **Cons:** Weak merge ownership and ambiguous lifecycle for detached commits.
- **When to use instead:** Disposable investigation sandboxes where no integration is expected.

**Rationale for recommendation:** Hybrid is the only option that satisfies both correctness (isolation) and operability (merge identity and cleanup).

---

### Implementation Details

**What to implement first:**

- Add dual-path agent metadata (`source_project_dir`, `git_worktree_dir`, `git_branch`).
- Teach spawn pipeline to create worktree/branch and set session cwd to worktree.
- Teach verification gates (`git_diff`, `build_blame`, commit evidence gate) to run against worktree dir.

**Things to watch out for:**

- ⚠️ `resolveProjectDir` currently drives beads context from `PROJECT_DIR`; preserve canonical beads dir as source repo, not worktree.
- ⚠️ `extractProjectDirFromWorkspace` and dashboard/status views must not regress cross-project detection.
- ⚠️ Abandon path must remove worktree safely even when session/process cleanup partially fails.

**Areas needing further investigation:**

- Merge queue policy under high completion concurrency.
- Epic branch model details (direct-to-main vs epic integration branch).
- Worktree storage and prune policy tuned by observed disk usage.

**Success criteria:**

- ✅ Two concurrent agents modifying same file no longer see each other's uncommitted changes during verification.
- ✅ Daemon does not auto-complete agents based on unrelated working-tree diffs.
- ✅ `orch complete` can merge and clean up isolated branches/worktrees deterministically.

---

## References

**Files Examined:**

- `cmd/orch/spawn_pipeline.go` - spawn lifecycle and cwd selection.
- `cmd/orch/complete_pipeline.go` - target resolution and completion flow.
- `cmd/orch/shared.go:402` - project-dir extraction from workspace context.
- `pkg/verify/git_diff.go:218` - diff source currently includes uncommitted baseline path.
- `pkg/verify/build_blame.go:147` - existing temporary worktree usage pattern.
- `.kb/investigations/2026-02-09-inv-post-mortem-daemon-overnight-ghost-completions.md` - incident evidence.

**Commands Run:**

```bash
# Search git isolation commands in active lifecycle code
rg -n "git (checkout|switch|branch|merge|worktree)" cmd/orch pkg

# Reproduce branch-only collision and worktree isolation behavior
git init -b main ...
git checkout agent/b   # from dirty agent/a tree -> checkout blocked
git worktree add ...   # separate worktrees show isolated status outputs

# Demonstrate detached worktree ambiguity
git worktree add --detach ...
git branch --show-current
```

**External Documentation:**

- https://git-scm.com/docs/git-worktree - Worktree behavior and detached/branch semantics.

**Related Artifacts:**

- **Decision:** `.kb/decisions/2026-02-09-git-isolation-worktree-plus-branch.md` - accepted architecture and phased plan.
- **Investigation:** `.kb/investigations/archived/2025-12-23-inv-investigate-git-branching-strategies-swarm.md` - prior branch-only recommendation.
- **Investigation:** `.kb/investigations/2026-02-09-inv-post-mortem-daemon-overnight-ghost-completions.md` - failure evidence motivating isolation.

---

## Investigation History

**[2026-02-09 10:20]:** Investigation started

- Initial question: choose between branches, worktrees, and hybrid for agent concurrency.
- Context: ghost completions proved shared tree failure at 22 agents.

**[2026-02-09 10:45]:** Code and lifecycle analysis completed

- Verified no branch/worktree isolation exists in spawn/complete paths.
- Identified project-dir coupling risk for beads vs git operations.

**[2026-02-09 11:05]:** Git behavior experiments completed

- Reproduced branch-only collision and shared-tree bleed.
- Validated isolation properties of separate worktrees.

**[2026-02-09 11:20]:** Investigation completed

- Status: Complete.
- Key outcome: recommend worktree-plus-branch hybrid with phased rollout.
