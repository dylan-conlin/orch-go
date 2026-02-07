## Summary (D.E.K.N.)

**Delta:** Pre-spawn kb context check was missing `--global` flag, causing it to only search local project knowledge instead of cross-repo decisions and constraints.

**Evidence:** Git diff shows original commit had `kb context query` but local file had `kb context --global query`. Running `kb context "spawn"` without --global returns only orch-go entries; with --global returns cross-repo entries with [project] prefixes.

**Knowledge:** The --global flag enables cross-repo knowledge search, returning entries from all known projects with project prefixes like `[orch-knowledge]`. This is critical for spawned agents to receive cross-project constraints and decisions.

**Next:** Commit fix with test coverage for global output format parsing.

**Confidence:** High (95%) - Fix verified through direct testing of kb context output with and without --global flag.

---

# Investigation: Fix Pre-Spawn KB Context Check

**Question:** Why isn't the pre-spawn kb context check including cross-repo knowledge?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Agent og-feat-fix-pre-spawn-22dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (95%)

---

## Findings

### Finding 1: Original commit missing --global flag

**Evidence:** Git show of original commit `000e1b7`:
```go
cmd := exec.Command("kb", "context", query)
```

Current local file (uncommitted):
```go
cmd := exec.Command("kb", "context", "--global", query)
```

**Source:** `git show 000e1b7:pkg/spawn/kbcontext.go | grep -A3 "exec.Command"` and `git diff pkg/spawn/kbcontext.go`

**Significance:** The --global flag was added locally but never committed, explaining why spawned agents only receive local knowledge.

---

### Finding 2: Global flag enables cross-repo search

**Evidence:** Testing `kb context "spawn"` from orch-go directory:
- Without --global: Returns only orch-go entries (no project prefix)
- With --global: Returns entries from orch-knowledge, orch-cli, and orch-go with `[project]` prefixes

Example global output:
```
- [orch-knowledge] Orchestrators NEVER do spawnable work
  Reason: Orchestrator doing task work blocks the entire system
- [orch-cli] Worker agents must NEVER spawn other agents
  Reason: Recursive spawn testing incident
- [orch-go] Agents must not spawn more than 3 iterations
  Reason: Prevents runaway iteration loops
```

**Source:** `kb context --global "spawn"` vs `kb context "spawn"` command output

**Significance:** Cross-repo knowledge is essential for spawned agents to respect system-wide constraints like "Worker agents must NEVER spawn other agents" from orch-cli.

---

### Finding 3: Parser already handles project prefixes

**Evidence:** Existing parseKBContextOutput function correctly parses entries with `[project]` prefixes. Test added and passes:
```go
{
    name: "parses global output with project prefixes",
    output: `Context for "spawn":

## CONSTRAINTS (from kn)

- [orch-knowledge] Orchestrators NEVER do spawnable work
  Reason: Orchestrator doing task work blocks the entire system
...`,
    wantCount:   4,
    wantTypes:   []string{"constraint", "constraint", "constraint", "decision"},
    wantSources: []string{"kn", "kn", "kn", "kn"},
}
```

**Source:** `go test ./pkg/spawn -v -run TestParseKBContextOutput` - all tests pass

**Significance:** No parsing changes needed - just need to commit the --global flag addition.

---

## Synthesis

**Key Insights:**

1. **Local-only bug** - The original implementation searched only the current project's .kn/ and .kb/ directories, missing critical cross-repo constraints

2. **Simple fix** - The --global flag was already added locally but never committed, making this a single-line fix plus test coverage

3. **Parser robust** - The existing parser handles both local and global output formats without modification

**Answer to Investigation Question:**

The pre-spawn kb context check wasn't including cross-repo knowledge because the `--global` flag was missing from the `kb context` command. The flag was added locally but never committed. The fix is to commit the existing local change along with test coverage for the global output format.

---

## Confidence Assessment

**Current Confidence:** High (95%)

**Why this level?**
Direct verification through git history and command testing confirms the root cause and fix.

**What's certain:**

- ✅ Original commit lacked --global flag (verified via git show)
- ✅ Local change adds --global flag (verified via git diff)
- ✅ Tests pass with global output format (verified via go test)

**What's uncertain:**

- ⚠️ Whether other callers might need similar fixes (none found)

---

## Implementation Recommendations

### Recommended Approach ⭐

**Commit the existing fix with test coverage**

**Why this approach:**
- Fix already exists locally, just needs to be committed
- Test coverage ensures the global output format parsing works
- No architectural changes needed

**Implementation sequence:**
1. Stage kbcontext.go (contains --global flag fix)
2. Stage kbcontext_test.go (contains test for global output format)
3. Commit with descriptive message

---

## References

**Files Examined:**
- `pkg/spawn/kbcontext.go` - Contains kb context command execution
- `pkg/spawn/kbcontext_test.go` - Contains parsing tests

**Commands Run:**
```bash
# Check original commit
git show 000e1b7:pkg/spawn/kbcontext.go | grep -A3 "exec.Command"

# Check current diff
git diff pkg/spawn/kbcontext.go

# Test local vs global output
kb context "spawn"
kb context --global "spawn"

# Run tests
go test ./pkg/spawn -v
```

---

## Investigation History

**2025-12-22 15:45:** Investigation started
- Initial question: Why isn't pre-spawn kb context check including cross-repo knowledge?
- Context: SPAWN_CONTEXT for this session only had orch-go entries, no cross-repo constraints

**2025-12-22 15:50:** Root cause identified
- Git history shows original commit lacked --global flag
- Local change exists but was never committed

**2025-12-22 16:00:** Investigation completed
- Final confidence: High (95%)
- Status: Complete
- Key outcome: Single-line fix plus test coverage needed
