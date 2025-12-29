## Summary (D.E.K.N.)

**Delta:** Added `--project` flag to `kb context` CLI in kb-cli, exposing the existing `ContextOptions.Project` field that was already used internally.

**Evidence:** Flag tested with `kb context "spawn" --global --project orch-go` - correctly filters to only orch-go results.

**Knowledge:** The kb-cli `ContextOptions` struct already had `Project` field and `GetContextGlobalWithProjects` already filtered by it - only the CLI exposure was missing.

**Next:** Close - the feature is implemented and tested. Optional follow-up: update orch-go's `runKBContextQuery()` to use `--project` flag for more efficient filtering at source.

---

# Investigation: Pass ContextOptions.Project Field to CLI

**Question:** How to expose the existing ContextOptions.Project field in kb context CLI so orch-go can filter global results to specific projects?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Feature-impl agent
**Phase:** Complete
**Next Step:** None - ready for closure
**Status:** Complete

**Supersedes:** None
**Extracted-From:** orch-go-gcf8 investigation recommendations

---

## Findings

### Finding 1: ContextOptions.Project field exists but wasn't CLI-exposed

**Evidence:** 
```go
// kb-cli/cmd/kb/context.go:59-65
type ContextOptions struct {
    Global  bool   // Search across all known projects
    Limit   int    // Maximum number of results per category
    Project string // Filter to specific project name (for global search)  <- EXISTS
    Stale   bool   // Enable stale detection
}
```

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/context.go:63`

**Significance:** The internal struct already supported project filtering - only CLI wiring was missing.

---

### Finding 2: GetContextGlobalWithProjects already uses opts.Project

**Evidence:**
```go
// kb-cli/cmd/kb/context.go:198-208
func GetContextGlobalWithProjects(projects []string, query string, opts ContextOptions) (ContextResult, error) {
    // ...
    for _, projectDir := range projects {
        projectName := filepath.Base(projectDir)
        
        // Skip if project filter is set and doesn't match
        if opts.Project != "" && projectName != opts.Project {
            continue
        }
```

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/context.go:206-208`

**Significance:** The filtering logic was already implemented and tested. Adding CLI flag was trivial.

---

### Finding 3: New flag works correctly

**Evidence:**
```bash
$ kb context "spawn" --global --project orch-go
# Returns only [orch-go] entries, no noise from other projects
```

**Source:** Manual test after implementation

**Significance:** The feature is working as expected. Orch-go can now use `--project` to filter at source.

---

## Synthesis

**Key Insights:**

1. **Work was minimal** - The existing architecture already supported project filtering; only CLI plumbing was needed.

2. **This completes Phase 3** - From the parent investigation (orch-go-gcf8), this was Phase 3 of the noise filtering plan.

3. **Phase 4 is optional** - Orch-go's current tiered approach (local first, then global with post-filter) works. Using `--project` at source is more efficient but not blocking.

**Answer to Investigation Question:**

The `ContextOptions.Project` field was exposed by adding a `-p, --project string` flag to the `kb context` CLI command. The implementation required 4 lines of code changes:
1. Declare `project` variable
2. Add flag definition with StringVarP
3. Pass to ContextOptions struct
4. Update help examples

---

## Structured Uncertainty

**What's tested:**

- ✅ Build succeeds (`go build ./...`)
- ✅ All context tests pass (`go test -v ./cmd/kb/... -run Context`)
- ✅ Flag appears in help (`kb context --help`)
- ✅ Filtering works (`kb context "spawn" --global --project orch-go` returns only orch-go entries)

**What's untested:**

- ⚠️ Integration with orch-go's runKBContextQuery() (not yet updated to use flag)
- ⚠️ Performance impact of --project vs post-filtering (likely marginal)

**What would change this:**

- If orch-go required multiple `--project` flags (would need multi-value flag)
- If filtering needed to be by path not project name (would need different API)

---

## Implementation Recommendations

**Implemented solution:** Added `--project` flag to kb context CLI.

**Changes made:**
1. `cmd/kb/context.go`: Added `project` variable declaration
2. `cmd/kb/context.go`: Added flag definition `-p, --project string "Filter to specific project (use with --global)"`
3. `cmd/kb/context.go`: Added `Project: project` to ContextOptions initialization
4. `cmd/kb/context.go`: Updated help examples to show `--project` usage

**Commit:** `feat(context): add --project flag for filtering global search results` in kb-cli

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/context.go` - CLI implementation
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/kbcontext.go` - Orch-go kb context integration

**Commands Run:**
```bash
# Build and test
go build ./...
go test -v ./cmd/kb/... -run Context

# Verify flag
kb context --help | grep project
kb context "spawn" --global --project orch-go
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-22-inv-pre-spawn-kb-context-noise-filtering.md` - Parent investigation
- **Issue:** orch-go-gcf8 - Follow-up from

---

## Investigation History

**2025-12-28 21:55:** Investigation started
- Initial question: How to expose ContextOptions.Project in CLI?
- Context: Follow-up from orch-go-gcf8 noise filtering investigation

**2025-12-28 22:05:** Found existing infrastructure
- ContextOptions.Project field already exists at line 63
- GetContextGlobalWithProjects already filters by opts.Project

**2025-12-28 22:10:** Implementation completed
- Added --project flag to CLI
- Tests pass, flag works
- Committed to kb-cli

**2025-12-28 22:15:** Investigation completed
- Status: Complete
- Key outcome: --project flag added to kb context CLI, exposing existing filtering capability
