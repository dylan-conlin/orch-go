# Decision: Beads OSS Relationship - Clean Slate

**Date:** 2025-12-21
**Status:** Accepted
**Enforcement:** context-only
**Deciders:** Dylan

## Context

Dylan maintains a local clone of `steveyegge/beads` with a `local-features` branch containing ~10 custom commits. A rebase conflict triggered this design session to determine the right OSS relationship strategy.

**Local features on `local-features` branch:**
- `bd ai-help` - Machine-readable CLI metadata for AI agents
- `bd health` - Combined backlog health summary
- `bd tree` - Convergence tracking via discovered-from trees
- `--discovered-from` flag - Foundation for tree command
- Short description warning - UX for issue quality
- 3-char hash fix - Bug fix for short hash ID imports

**Relationship to upstream:** External contributor (no maintainer access). PRs would go through standard review process.

## Decision

**Drop all local features and use upstream beads as-is.**

### Actions Required

1. Abort current rebase: `cd ~/Documents/personal/beads && git rebase --abort`
2. Reset main to upstream: `git checkout main && git reset --hard origin/main`
3. Delete local-features branch: `git branch -D local-features`
4. Reinstall from upstream: `go install ./cmd/bd`

### One Skill Update Needed

The orchestrator SKILL.md mentions `--discovered-from` once. Update to use standard `bd dep add` instead:

```diff
- **If fails:** `bd create "Integration issue: [problem]" --discovered-from <epic-id>` → don't close until resolved.
+ **If fails:** `bd create "Integration issue: [problem]"` then `bd dep add <new-id> <epic-id>` → don't close until resolved.
```

## Rationale

### Why not fork/contribute?

1. **Features not used:** Investigation found none of the local features are actually used by orch-go or skills:
   - `bd ai-help` - Never integrated
   - `bd health` - Never called
   - `bd tree` - Never called
   - `--discovered-from` - One mention in SKILL.md (easily replaced)

2. **Maintenance burden:** Active rebase conflict demonstrates ongoing friction with zero benefit.

3. **External contributor friction:** PRs require review cycles. Not worth it for unused features.

### Why not keep local patches?

Same features, same non-usage. Local patches just add rebase burden.

### What about the 3-char hash fix?

Not hitting this bug in practice. If it surfaces later, can PR upstream at that point.

## Consequences

**Positive:**
- Zero rebase maintenance going forward
- Always on latest upstream
- No fork to manage
- Cleaner mental model

**Negative:**
- Lose ability to use custom features immediately (but we weren't using them)
- If 3-char hash bug surfaces, need to either PR upstream or work around

**Neutral:**
- One SKILL.md update needed (trivial)

## Alternatives Considered

### Fork with Selective PRs
Maintain fork, PR generally-useful features upstream, keep orchestration-specific locally.
**Rejected:** Features aren't being used, so no value in maintaining them anywhere.

### Pure Local Patches
Keep local-features branch, rebase regularly.
**Rejected:** Current situation causing friction with no benefit.

### Pure Upstream Contribution
Submit all features as PRs, wait for merge.
**Rejected:** Features aren't needed, so no point PRing them.

## Follow-Up

- [ ] Execute cleanup commands (abort rebase, reset, delete branch)
- [ ] Update orchestrator SKILL.md to remove `--discovered-from` reference
- [ ] Reinstall bd from upstream

## References

- Investigation: `.kb/investigations/2025-12-21-inv-beads-oss-relationship-fork-vs.md`
- Beads repo: `~/Documents/personal/beads`
- Upstream: `https://github.com/steveyegge/beads`

## Auto-Linked Investigations

- .kb/investigations/archived/2025-12-21-inv-beads-kb-workspace-relationships-how.md
- .kb/investigations/archived/2025-12-21-inv-beads-oss-relationship-fork-vs.md
