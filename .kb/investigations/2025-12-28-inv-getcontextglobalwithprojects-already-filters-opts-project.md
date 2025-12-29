## Summary (D.E.K.N.)

**Delta:** kb-cli's `GetContextGlobalWithProjects` function already supports filtering by `opts.Project`, but this capability is NOT exposed in the `kb context` CLI command (unlike `kb search --project`).

**Evidence:** Reviewed `kb-cli/cmd/kb/context.go:199-207` - the function skips projects when `opts.Project` is set. `kb context --help` shows no `--project` flag. `kb search --help` shows `--project` flag exists at line 95 of search.go.

**Knowledge:** orch-go's post-filtering via `filterToOrchEcosystem()` is currently the only way to filter kb context results to specific projects. This is less efficient than server-side filtering but works.

**Next:** Create kb-cli issue to add `--project` flag to `kb context` command (parity with `kb search`).

---

# Investigation: GetContextGlobalWithProjects Already Filters By opts.Project

**Question:** What does "GetContextGlobalWithProjects already filters by opts.Project" mean and what action is needed?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None - findings documented, follow-up issue needed in kb-cli
**Status:** Complete

**Supersedes:** None (this clarifies a detail from the prior noise filtering investigation)
**Related:** `.kb/investigations/2025-12-22-inv-pre-spawn-kb-context-noise-filtering.md` (Finding 3)

---

## Findings

### Finding 1: GetContextGlobalWithProjects supports project filtering internally

**Evidence:** From `kb-cli/cmd/kb/context.go:199-207`:

```go
func GetContextGlobalWithProjects(projects []string, query string, opts ContextOptions) (ContextResult, error) {
    var combined ContextResult

    for _, projectDir := range projects {
        projectName := filepath.Base(projectDir)

        // Skip if project filter is set and doesn't match
        if opts.Project != "" && projectName != opts.Project {
            continue
        }
        // ...
    }
}
```

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/context.go:199-207`

**Significance:** The filtering capability exists in kb-cli's API but is not exposed to CLI users. This was noted in the prior investigation (Finding 3) but not acted upon.

---

### Finding 2: kb context CLI does not expose --project flag

**Evidence:** Output of `kb context --help`:

```
Flags:
  -f, --format string   Output format (text, json) (default "text")
  -g, --global          Search across all known projects
  -h, --help            help for context
  -l, --limit int       Maximum results per category (0 = no limit)
      --stale           Enable stale detection for investigations (slower)
```

No `--project` flag is available.

**Source:** `kb context --help` command output

**Significance:** Users (including orch-go) cannot filter `kb context --global` results to specific projects via CLI. Must use post-processing.

---

### Finding 3: kb search DOES expose --project flag

**Evidence:** From `kb-cli/cmd/kb/search.go:95`:

```go
searchCmd.Flags().StringVarP(&projectFilter, "project", "p", "", "Filter to specific project (use with -g)")
```

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/search.go:95`

**Significance:** This is a parity gap. `kb search` has the flag but `kb context` doesn't. The underlying `ContextOptions.Project` field exists (line 63 of context.go), it's just not wired to a CLI flag.

---

### Finding 4: orch-go uses post-filtering as workaround

**Evidence:** From `orch-go/pkg/spawn/kbcontext.go:130`:

```go
// Post-filter to orch ecosystem repos
globalResult.Matches = filterToOrchEcosystem(globalResult.Matches)
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/kbcontext.go:130`

**Significance:** orch-go compensates for kb context's missing `--project` flag by post-filtering results. This works but is less efficient than server-side filtering - all projects are searched and then results are discarded.

---

## Synthesis

**Key Insights:**

1. **API-CLI gap** - kb-cli's Go API (`GetContextGlobalWithProjects`) supports project filtering, but the CLI command doesn't expose this capability.

2. **Parity issue** - `kb search --project` exists but `kb context --project` doesn't, despite both using similar filtering patterns internally.

3. **orch-go workaround is correct** - Until kb-cli adds the flag, post-filtering with `filterToOrchEcosystem()` is the right approach.

**Answer to Investigation Question:**

The issue title "GetContextGlobalWithProjects already filters by opts.Project" is a note that:
1. The underlying capability EXISTS in kb-cli's Go API
2. But it's NOT exposed in the CLI
3. Therefore, orch-go's post-filtering approach is necessary

The follow-up action is to create an issue in kb-cli to add `--project` flag to `kb context` command.

---

## Structured Uncertainty

**What's tested:**

- ✅ kb context CLI has no --project flag (verified: `kb context --help`)
- ✅ kb search CLI has --project flag (verified: code at search.go:95)
- ✅ GetContextGlobalWithProjects filters by opts.Project (verified: code at context.go:205-207)
- ✅ orch-go post-filters results (verified: code at kbcontext.go:130)

**What's untested:**

- ⚠️ Whether adding --project flag to kb context would require schema changes
- ⚠️ Performance improvement from server-side vs client-side filtering

**What would change this:**

- If kb-cli adds `--project` flag, orch-go could simplify to use it directly
- If orch-go needs to support projects outside ecosystem, post-filtering would still be needed

---

## Implementation Recommendations

### Recommended Approach ⭐

**Create kb-cli issue to add --project flag to kb context**

**Why this approach:**
- The underlying API already supports it (zero backend changes)
- Parity with `kb search --project`
- Would allow orch-go to simplify filtering

**Trade-offs accepted:**
- Requires kb-cli change (separate repo)
- Current orch-go post-filtering works fine

**Implementation sequence:**
1. Create beads issue in kb-cli: "Add --project flag to kb context command"
2. Implementation: Add flag wiring (~10 lines of code in context.go)
3. Update orch-go to use --project flag instead of post-filtering (optional, current approach works)

### Alternative Approaches Considered

**Option B: Keep orch-go post-filtering only**
- **Pros:** No cross-repo coordination needed
- **Cons:** Less efficient, processes all projects then discards results
- **When to use instead:** If kb-cli change is blocked or deferred

**Rationale for recommendation:** The API capability exists - exposing it via CLI is minimal work and enables better efficiency.

---

## References

**Files Examined:**
- `kb-cli/cmd/kb/context.go` - GetContextGlobalWithProjects function and CLI flags
- `kb-cli/cmd/kb/search.go` - Reference for --project flag implementation
- `orch-go/pkg/spawn/kbcontext.go` - Post-filtering implementation
- `.kb/investigations/2025-12-22-inv-pre-spawn-kb-context-noise-filtering.md` - Prior investigation noting this gap

**Commands Run:**
```bash
# Check kb context help
kb context --help

# Check kb search help (for comparison)
kb search --help

# Check GetContextGlobalWithProjects implementation
grep -n "opts.Project" kb-cli/cmd/kb/context.go
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-22-inv-pre-spawn-kb-context-noise-filtering.md` - Finding 3 noted this exact gap

---

## Investigation History

**2025-12-28 21:40:** Investigation started
- Initial question: What does this issue title mean and what action is needed?
- Context: Follow-up from orch-go-gcf8

**2025-12-28 21:50:** Root cause identified
- Found API-CLI gap in kb-cli
- Confirmed orch-go workaround is correct

**2025-12-28 21:55:** Investigation completed
- Status: Complete
- Key outcome: kb-cli needs --project flag added to kb context command
