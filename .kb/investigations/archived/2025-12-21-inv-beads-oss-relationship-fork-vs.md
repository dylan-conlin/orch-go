## Summary (D.E.K.N.)

**Delta:** Local beads features (ai-help, health, tree, --discovered-from) are not used by orch ecosystem. Clean slate is optimal.

**Evidence:** Searched orch-go codebase and skills - zero usage of local features. Only standard bd commands used: comment, create, close, show, list, ready.

**Knowledge:** Aspirational features without integration create maintenance burden. Drop unused code rather than maintaining it.

**Next:** Execute cleanup (abort rebase, reset to upstream, delete local-features branch, reinstall bd).

**Confidence:** High (90%) - Comprehensive search of orch-go and skills found no usage.

---

# Investigation: Beads OSS Relationship - Fork vs Contribute vs Local Patches

**Question:** What's the right OSS relationship strategy for beads customizations? Fork, contribute upstream, or local patches?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** worker
**Phase:** Complete
**Next Step:** None - decision made
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Active rebase conflict with unused local features

**Evidence:** `git status` in beads repo shows rebase in progress with conflict in `create.go`. Conflict between local "short description warning" feature and upstream template changes.

**Source:** `/Users/dylanconlin/Documents/personal/beads` - `git status` output

**Significance:** Demonstrates ongoing maintenance friction. Triggered this design session.

---

### Finding 2: Local-features branch contains 7 custom commits not in upstream

**Evidence:** 
```
6c14b4f6 feat: add bd ai-help command for machine-readable CLI metadata
d80137f9 feat: add bd health command for backlog health summary
3ff8a3d1 chore: sync workspace for beads audit investigation
d121f67e docs: add investigation - audit beads src changes
7d4c8f3e fix(import): support 3-char base36 hash IDs in prefix extraction
17b1bfc9 feat(cli): add bd tree command for convergence tracking
1eb091ba feat: add --discovered-from flag to bd create and lineage display in bd show
```

**Source:** `git log local-features --oneline --not origin/main`

**Significance:** These are the features we need to evaluate for fork/contribute/drop decision.

---

### Finding 3: None of the local features are used by orch-go

**Evidence:** 
- `rg "bd ai-help"` in orch-go: no results
- `rg "bd health"` in orch-go: no results  
- `rg "bd tree"` in orch-go: no results
- `rg "discovered-from"` in orch-go: no results

BD commands actually used (via exec.Command):
- `bd comments <id> --json`
- `bd comment <id> <msg>`
- `bd list --status open --json`
- `bd show <id> --json`
- `bd create <title>`
- `bd ready`

**Source:** Grep of `/Users/dylanconlin/Documents/personal/orch-go`

**Significance:** Core finding - local features have zero integration. Maintenance burden with no benefit.

---

### Finding 4: Only one skill reference to --discovered-from

**Evidence:** Single mention in orchestrator SKILL.md:
```
**If fails:** `bd create "Integration issue: [problem]" --discovered-from <epic-id>`
```

**Source:** `~/.claude/skills/policy/orchestrator/SKILL.md`

**Significance:** Easily replaced with standard `bd dep add` command. Not a blocker for clean slate.

---

### Finding 5: No JSON parsing bug in beads - issues were in orch-go

**Evidence:** 
- `orch-go-jz5`: Comment ID type mismatch - fixed in orch-go's Comment struct
- `orch-go-c4r`: Parsing "open" instead of ID - fixed in orch-go's parsing

Beads output format is correct (comments have numeric IDs as expected).

**Source:** `bd show orch-go-jz5`, `bd show orch-go-c4r`, `bd comments --json` output

**Significance:** No upstream bug to fix. Eliminates one reason for local patches.

---

## Synthesis

**Key Insights:**

1. **Aspirational vs Integrated** - The local features were built speculatively but never wired into the orch ecosystem. Building features without integration creates technical debt.

2. **Maintenance cost vs value** - Rebase conflicts are a recurring cost. With zero feature usage, the ROI is negative.

3. **External contributor friction** - As non-maintainer, PRs require review cycles. Not worth the overhead for unused features.

**Answer to Investigation Question:**

**Clean Slate** is the right strategy. Drop all local features, reset to upstream, delete local-features branch. The features aren't used, so there's nothing to fork or contribute. Future customizations should be integrated before maintaining.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**

Comprehensive search of both orch-go codebase and skills directory found zero usage of local features. The only reference (`--discovered-from` in SKILL.md) is easily replaced.

**What's certain:**

- Local features not called from orch-go code
- Local features not referenced in skills (except one easily-replaced mention)
- Rebase conflict is real and recurring friction
- Dylan confirmed no relationship to upstream maintainer

**What's uncertain:**

- Whether 3-char hash bug will surface in practice (hasn't yet)
- Whether future workflows might want these features

**What would increase confidence to Very High:**

- Longer observation period confirming no hash bug
- Explicit "we won't need these" statement for each feature

---

## Implementation Recommendations

### Recommended Approach: Clean Slate

**Why this approach:**
- Zero maintenance burden going forward
- Always on latest upstream automatically
- Matches actual usage (none of local features used)

**Trade-offs accepted:**
- Lose 3-char hash fix (can PR upstream if needed later)
- Lose `--discovered-from` syntax (use `bd dep add` instead)

**Implementation sequence:**
1. Abort rebase: `cd ~/Documents/personal/beads && git rebase --abort`
2. Reset to upstream: `git checkout main && git reset --hard origin/main`
3. Delete local branch: `git branch -D local-features`
4. Reinstall: `go install ./cmd/bd`
5. Update SKILL.md to remove `--discovered-from` reference

### Alternative: Minimal Fork (Bug Fix Only)

**When to use instead:** Only if 3-char hash import bug is actively blocking work.

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/beads/` - git status, log, remotes
- `/Users/dylanconlin/Documents/personal/orch-go/` - grep for bd command usage
- `~/.claude/skills/policy/orchestrator/SKILL.md` - bd references

**Commands Run:**
```bash
# Check rebase state
cd ~/Documents/personal/beads && git status

# Find local-only commits
git log local-features --oneline --not origin/main

# Search for feature usage
rg "bd ai-help|bd health|bd tree|discovered-from" ~/Documents/personal/orch-go
rg "exec\.Command.*bd" --type go ~/Documents/personal/orch-go
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-beads-oss-relationship-clean-slate.md`

---

## Investigation History

**2025-12-21 20:00:** Investigation started
- Initial question: Fork vs contribute vs local patches for beads customizations
- Context: Mid-rebase conflict, need decision framework

**2025-12-21 20:15:** Context gathered
- Found 7 local commits not in upstream
- Identified active rebase conflict

**2025-12-21 20:25:** Feature usage investigated
- Found zero usage of local features in orch-go
- Found one SKILL.md reference to --discovered-from

**2025-12-21 20:30:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Clean slate - drop all local features, reset to upstream
