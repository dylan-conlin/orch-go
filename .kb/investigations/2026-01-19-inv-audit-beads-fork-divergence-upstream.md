<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Fork is 16 commits ahead (custom orch features) and 976 commits behind upstream (v0.41.0 vs v0.48.0), with 47+ merge conflicts making sync impractical.

**Evidence:** `git rev-list --left-right --count main...origin/main` → 16 976; attempted merge showed conflicts in cmd/bd/close.go, create.go, internal/storage/sqlite/*, internal/types/types.go, and 40+ other files.

**Knowledge:** Fork maintenance is unsustainable at current divergence rate (~140 commits/week). Our 16 custom features are orch-specific (question gates, phase verification, WAL freshness) that likely won't be upstreamed.

**Next:** Decision required - Option A: Continue fork isolation (accept missing upstream features) or Option B: Abandon fork, propose features upstream.

**Promote to Decision:** Actioned - fork strategy documented

---

# Investigation: Audit Beads Fork Divergence Upstream

**Question:** What is the divergence state between our beads fork and upstream steveyegge/beads, and what's the maintenance burden?

**Started:** 2026-01-19
**Updated:** 2026-01-19
**Owner:** Dylan (orchestrator spawned)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Fork is 16 commits ahead, 976 behind

**Evidence:**
```
$ git rev-list --left-right --count main...origin/main
16	976
```

Fork branched from v0.41.0 on December 30, 2025 (commit 7f5378ba). Upstream is now at v0.48.0.

**Source:** `~/Documents/personal/beads` - `git merge-base`, `git describe --tags`

**Significance:** 976 commits in ~3 weeks = ~140 commits/week. Upstream is extremely active. Gap will only widen.

---

### Finding 2: Our 16 fork commits are orch-specific features

**Evidence:**
| Commit | Description |
|--------|-------------|
| 744af9cf | feat(questions): implement question gates via dependency blocking |
| d14cf911 | feat(questions): wire question lifecycle status validation |
| 2dc8f7dc | feat(types): add question entity type with investigating/answered statuses |
| 12031e17 | feat: skip bd prime output in spawned contexts |
| 211be5e3 | feat: require user confirmation before push in session close protocol |
| 7a3ccce5 | feat: add self-healing binary auto-rebuild pattern |
| 5047a390 | fix(create): allow --reason with --no-understanding for epics |
| 1139ef93 | docs: add SYNTHESIS.md for epic readiness gate implementation |
| eb81ac53 | feat: add epic readiness gate requiring understanding section |
| f88e94df | feat: add could-not-reproduce as distinct close outcome for bugs |
| e93637cc | feat: add --repro flag to bd create for bug type |
| 60d791e2 | chore: add build/ to gitignore |
| 1a8a3dc4 | feat: switch install target to symlink pattern |
| be871d0c | feat: gate bd close on Phase: Complete verification |
| f8aa3ac0 | beads: sync issues from session |
| 2e0ce160 | fix: add WAL freshness checking to comment retrieval for daemon mode |

**Source:** `git log --oneline origin/main..main`

**Significance:** Features are orch orchestration-specific (phase verification, question lifecycle, epic gates). These are unlikely to be accepted upstream as they impose workflow constraints.

---

### Finding 3: 47+ files have merge conflicts

**Evidence:**
Attempted merge revealed conflicts in critical files:
- `cmd/bd/close.go` - our phase verification vs upstream changes
- `cmd/bd/create.go` - our --repro flag vs upstream changes
- `internal/types/types.go` - our question entity type vs upstream type changes
- `internal/storage/sqlite/*.go` - 20+ conflicts in storage layer
- `internal/validation/*.go` - validation rule conflicts

Full conflict list from `git merge --no-commit --no-ff origin/main`:
- cmd/bd/: close.go, create.go, duplicates.go, epic.go, gate.go, ready.go, show.go, version.go + tests
- internal/storage/: sqlite layer extensively modified upstream
- internal/types/types.go: core type definitions diverged

**Source:** Test merge attempt with `git stash && git merge --no-commit --no-ff origin/main`

**Significance:** Conflicts span core types, storage, and commands. Not a simple rebase - would require substantial conflict resolution across the codebase.

---

### Finding 4: Upstream has significant features we're missing

**Evidence:**
Key upstream features since fork:
1. **Dolt backend** - Version-controlled issue storage (v0.42+)
2. **Interactive sync** - Manual conflict resolution (v0.47+)
3. **Sync modes** - Configurable sync behavior
4. **VersionedStorage interface** - History/diff/branch operations
5. **--children flag** - `bd show --children` for hierarchy
6. **"enhancement" type alias** - `feature` type flexibility
7. Various bug fixes (sparse checkout, WAL, config validation)

**Source:** `git log --oneline main..origin/main | grep "^[a-f0-9]+ feat"` - 20+ feature commits

**Significance:** Dolt backend could be valuable for orch's multi-repo scenarios. Sync improvements could reduce friction. But adopting requires resolving the divergence first.

---

## Synthesis

**Key Insights:**

1. **Fork is not maintainable long-term** - At 140 commits/week upstream velocity, the fork will become completely divorced from upstream within months. Manual merge would take days now.

2. **Our features serve orch specifically** - Question lifecycle, phase verification, epic gates are orch workflow enforcement. These aren't general-purpose beads features and may not be accepted upstream.

3. **Clean slate may be necessary** - Prior decision (`.kb/decisions/`) already noted "Beads OSS: Clean Slate over Fork" - this audit confirms that assessment is still valid.

**Answer to Investigation Question:**

The fork has diverged significantly: 16 commits ahead with orch-specific features, 976 commits behind with 47+ conflicting files. Merge is impractical. The maintenance burden is HIGH - continuing the fork means missing Dolt backend, sync improvements, and accumulating technical debt. However, our features are workflow-specific and unlikely to be upstreamed, so we're stuck choosing between feature isolation or feature loss.

---

## Structured Uncertainty

**What's tested:**

- ✅ Commit counts verified via git rev-list (16 ahead, 976 behind)
- ✅ Merge conflicts verified via actual merge attempt (47+ files)
- ✅ Fork date confirmed via merge-base (Dec 30, 2025, commit 7f5378ba)
- ✅ Version confirmed (fork at v0.41.0-63, upstream at v0.48.0)

**What's untested:**

- ⚠️ Whether upstream would accept our features as optional/configurable additions
- ⚠️ Actual time to resolve conflicts if we attempted merge
- ⚠️ Whether Dolt backend would help orch use cases

**What would change this:**

- If upstream added optional workflow hooks, our features could be implemented as plugins
- If beads development slowed, the gap would stabilize
- If we dropped some features, fewer conflicts to resolve

---

## Implementation Recommendations

### Recommended Approach ⭐

**Accept Fork Isolation** - Continue with isolated fork, accept missing upstream features, plan for eventual migration when beads supports plugins/hooks.

**Why this approach:**
- Resolving 47+ conflicts is high-risk (could break things)
- Our features are in active use by orch daemon
- Upstream features (Dolt, sync modes) aren't critical for current use

**Trade-offs accepted:**
- Missing Dolt backend (version-controlled issues)
- Missing sync improvements (manual conflict resolution)
- Increasing divergence from upstream community

**Implementation sequence:**
1. Document fork as intentionally isolated (update README or CLAUDE.md)
2. Pin expectations - this fork is orch-specific, not tracking upstream
3. Re-evaluate quarterly or when specific upstream feature is needed

### Alternative Approaches Considered

**Option B: Cherry-pick critical fixes only**
- **Pros:** Get bug fixes without full merge
- **Cons:** Still diverges, cherry-picks get harder over time
- **When to use instead:** If specific bug fix is needed urgently

**Option C: Propose features upstream**
- **Pros:** Would eliminate fork need
- **Cons:** Features are orch-specific workflow enforcement, unlikely accepted
- **When to use instead:** If beads adds plugin/hook system

**Option D: Abandon fork, accept feature loss**
- **Pros:** Get all upstream improvements
- **Cons:** Lose question gates, phase verification, epic readiness - breaks orch workflows
- **When to use instead:** If orch workflows change or beads adds equivalent features

**Rationale for recommendation:** Fork serves orch-specific needs. Merge cost exceeds benefit. Isolation is honest about the situation.

---

## References

**Files Examined:**
- `~/Documents/personal/beads` - Fork repository root
- `cmd/bd/version.go` - Version string (v0.41.0)

**Commands Run:**
```bash
# Check remotes
git remote -v

# Count divergence
git rev-list --left-right --count main...origin/main

# List fork commits
git log --oneline origin/main..main

# Find merge base
git merge-base main origin/main

# Test merge for conflicts
git stash && git merge --no-commit --no-ff origin/main

# Check version
git describe --tags
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/` - "Beads OSS: Clean Slate over Fork" (prior decision)
- **Guide:** `.kb/guides/beads-integration.md` - Current integration patterns

---

## Investigation History

**2026-01-19 15:30:** Investigation started
- Initial question: How divergent is our beads fork from upstream?
- Context: Spawned by orchestrator to audit fork maintenance burden

**2026-01-19 15:35:** Core findings discovered
- Found 16/976 commit divergence
- Identified 47+ merge conflicts
- Catalogued our 16 fork commits

**2026-01-19 15:40:** Investigation completed
- Status: Complete
- Key outcome: Fork is isolated and should stay that way; merge is impractical
