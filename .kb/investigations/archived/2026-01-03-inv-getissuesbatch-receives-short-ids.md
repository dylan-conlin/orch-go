<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** GetIssuesBatch fails to find issues because List(IDs) requires full IDs like 'orch-go-51jz' but receives short IDs like '51jz' from session titles.

**Evidence:** Tested `bd list --id 9hld.1` returns `[]`, but `bd show 9hld.1` returns the issue. Show resolves short IDs, List does not.

**Knowledge:** beads CLI has two ID resolution behaviors: Show resolves short IDs via ResolveID(), List expects exact full IDs.

**Next:** Replace List(IDs) with parallel Show() calls in GetIssuesBatch since Show handles short ID resolution.

---

# Investigation: GetIssuesBatch Receives Short IDs

**Question:** Why does GetIssuesBatch return empty results for valid beads IDs?

**Started:** 2026-01-03
**Updated:** 2026-01-03
**Owner:** debugging agent
**Phase:** Complete
**Next Step:** None - implementing fix
**Status:** Complete

---

## Findings

### Finding 1: List(IDs) requires full IDs

**Evidence:** 
- `bd list --json --all --id 9hld.1` returns `[]`
- `bd list --json --all --id orch-go-9hld.1` returns the issue

**Source:** pkg/beads/client.go:730-756 FallbackListByIDs, tested via CLI

**Significance:** List with IDs filter does exact string matching against full issue IDs, not partial/short ID resolution.

---

### Finding 2: Show() resolves short IDs via ResolveID

**Evidence:**
- `bd show 9hld.1 --json` returns the issue with full ID `orch-go-9hld.1`
- Client.Show() calls the daemon which internally calls ResolveID

**Source:** pkg/beads/client.go:357-381 Show(), client.go:626-641 ResolveID()

**Significance:** Show has the capability to resolve short IDs that GetIssuesBatch needs.

---

### Finding 3: Session titles contain short IDs

**Evidence:**
- extractBeadsIDFromTitle() extracts `[beads-id]` from session titles
- Beads IDs in titles are short format like `51jz`, `q03k`
- GetIssuesBatch receives these short IDs from status_cmd.go:238

**Source:** cmd/orch/shared.go:27-35, cmd/orch/status_cmd.go:238

**Significance:** The IDs passed to GetIssuesBatch are inherently short IDs from session context.

---

## Synthesis

**Key Insights:**

1. **API asymmetry** - List and Show have different ID resolution behaviors. This is undocumented but intentional - List is for bulk filtering, Show is for single-issue lookup with convenience features.

2. **Short IDs are the norm** - All agent tracking uses short IDs in session titles and window names for brevity.

3. **Simple fix available** - Since Show handles short IDs and GetIssuesBatch already has fallback logic, switching to parallel Show calls is straightforward.

**Answer to Investigation Question:**

GetIssuesBatch returns empty results because it uses `client.List(&beads.ListArgs{IDs: beadsIDs})` which requires full IDs, but it receives short IDs from session titles. The fix is to use individual `client.Show()` calls instead, which handle short ID resolution via the daemon's ResolveID functionality.

---

## Structured Uncertainty

**What's tested:**

- ✅ `bd list --id` returns empty for short IDs (verified: ran `bd list --json --all --id 9hld.1`)
- ✅ `bd list --id` returns issue for full IDs (verified: ran `bd list --json --all --id orch-go-9hld.1`)
- ✅ `bd show` resolves short IDs (verified: ran `bd show 9hld.1 --json`)

**What's untested:**

- ⚠️ Parallel Show() calls won't overwhelm RPC daemon (reasonable assumption given existing GetCommentsBatch pattern)
- ⚠️ Performance impact of N Show calls vs 1 List call (likely negligible for typical 5-20 agents)

**What would change this:**

- Finding would be wrong if List(IDs) had an undocumented short ID resolution mode
- Solution would need adjustment if Show() became a bottleneck

---

## Implementation Recommendations

### Recommended Approach: Replace List with parallel Show calls

**Why this approach:**
- Show() already handles short ID resolution
- Pattern matches existing GetCommentsBatch which uses parallel RPC calls
- Minimal code change, maintains interface compatibility

**Trade-offs accepted:**
- N RPC calls instead of 1 (acceptable for typical agent counts)
- Slightly more complex error handling

**Implementation sequence:**
1. Replace List() call with loop of Show() calls
2. Use goroutines with semaphore for parallel execution (like GetCommentsBatchWithProjectDirs)
3. Maintain existing return semantics (map[string]*Issue)

### Alternative Approaches Considered

**Option B: Resolve IDs first with ResolveID()**
- **Pros:** Keeps batch List approach
- **Cons:** Adds N ResolveID calls + 1 List call = N+1 total calls
- **When to use instead:** If Show() turns out to be slower than ResolveID()

**Rationale for recommendation:** Using Show() is simpler (fewer calls) and matches how the codebase already handles single-issue lookups.

---

## References

**Files Examined:**
- pkg/verify/check.go:700-757 - GetIssuesBatch implementation
- pkg/beads/client.go:357-381 - Show() method
- pkg/beads/client.go:384-400 - List() method
- pkg/beads/client.go:626-641 - ResolveID() method
- cmd/orch/status_cmd.go:238 - GetIssuesBatch caller
- cmd/orch/shared.go:27-35 - extractBeadsIDFromTitle

**Commands Run:**
```bash
# Test List with short ID
bd list --json --all --id 9hld.1
# Result: []

# Test List with full ID
bd list --json --all --id orch-go-9hld.1
# Result: [issue object]

# Test Show with short ID
bd show 9hld.1 --json
# Result: [issue object with full ID orch-go-9hld.1]
```

---

## Investigation History

**2026-01-03:** Investigation started
- Initial question: Why does GetIssuesBatch fail to find issues with valid beads IDs?
- Context: orch status showing incorrect closed status for agents

**2026-01-03:** Root cause identified
- Confirmed List(IDs) requires full IDs via CLI testing
- Confirmed Show() resolves short IDs

**2026-01-03:** Investigation completed
- Status: Complete
- Key outcome: Replace List(IDs) with parallel Show() calls
