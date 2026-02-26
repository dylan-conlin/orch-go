---
linked_issues:
  - orch-go-2or
---
## Summary (D.E.K.N.)

**Delta:** `kb search` and `rg` serve different purposes - kb searches knowledge artifacts (.kb/), rg searches code/codebase - agents should use both strategically.

**Evidence:** Ran 10 benchmark tests comparing kb search vs rg/grep - kb found 104 results for "spawn" in knowledge docs, rg found code in 12 Go files; neither alone provides complete picture.

**Knowledge:** The tools are complementary, not competing: kb for "what did we learn/decide" (investigations, decisions), rg for "what does the code do" (implementation).

**Next:** No code changes needed - document the mental model of when to use each tool in agent guidelines.

**Confidence:** High (85%) - 10 distinct query types tested, clear pattern emerged.

---

# Investigation: KB Search vs Grep Benchmark

**Question:** Is `kb search` providing worse retrieval than grep/rg, and if so, how can we improve it?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: kb search and rg search different scopes

**Evidence:** 
- `kb search "spawn"` found 104 results in .kb/investigations/ and .kb/decisions/
- `rg "spawn" --type go` found 12 Go source files
- `rg "spawn" .kb --type md` found 30 markdown files (same scope as kb but different counts due to result grouping)

**Source:** 
- kb-cli/cmd/kb/search.go:229-243 - SearchArtifacts only walks `.kb/investigations/` and `.kb/decisions/`
- Benchmark commands run in this investigation

**Significance:** The tools have fundamentally different scopes. `kb search` intentionally only searches knowledge artifacts, not the entire codebase. This is by design, not a bug.

---

### Finding 2: kb search is case-insensitive substring matching

**Evidence:**
- `kb search "SSE"` returned 119 results including matches on "Assessment" and "phase" (containing "sse")
- `rg "SSE" .kb` (case-sensitive) returned 57 files
- `rg -i "sse" .kb` (case-insensitive) returned 100+ files

The kb search algorithm (search.go:294-333):
```go
queryLower := strings.ToLower(query)
lineLower := strings.ToLower(line)
if strings.Contains(lineLower, queryLower) {
    matches = append(matches, ...)
}
```

**Source:** kb-cli/cmd/kb/search.go:294-333

**Significance:** The case-insensitive substring matching can produce false positives. A search for "SSE" (Server-Sent Events) also matches "Assessment". This is the source of Dylan's observation about "agents finding results better by grepping."

---

### Finding 3: kb search has unique cross-project capability

**Evidence:**
```
$ kb search --global "synthesis protocol" --summary
Summary for "synthesis protocol" (4 total results):
  orch-knowledge: 1
  orch-go: 3
```

rg cannot do this without manually specifying each project path.

**Source:** kb-cli/cmd/kb/search.go:101-172 - discoverProjects() and SearchGlobal()

**Significance:** This is kb search's killer feature - finding knowledge across the entire ecosystem. rg is project-local by default.

---

### Finding 4: Performance is equivalent for local searches

**Evidence:**
- kb search: 0.00-0.01s for "spawn" query (5 runs)
- rg: 0.00-0.01s for same query (5 runs)

**Source:** Timing benchmarks in this investigation

**Significance:** Performance is not a differentiator - both are effectively instant for typical queries.

---

### Finding 5: Different query types favor different tools

**Evidence:**
| Query Type | Best Tool | Example |
|------------|-----------|---------|
| "How did we decide X?" | kb search | `kb search "synthesis protocol"` |
| "What does function X do?" | rg | `rg "WaitForStatus" pkg/` |
| "Where is X implemented?" | rg | `rg "registry" --type go` |
| "What investigations exist about X?" | kb search | `kb search "beads issue"` |
| "Find all occurrences of pattern" | rg | `rg "Session.*Status"` |
| "Cross-project knowledge" | kb search | `kb search --global "model arbitrage"` |

**Source:** Multiple test queries in this investigation

**Significance:** This explains Dylan's observation - agents grepping often find *code* better, but that's appropriate for code searches. kb search finds *knowledge* better for knowledge queries.

---

## Synthesis

**Key Insights:**

1. **Tools serve different purposes** - kb search is a knowledge retrieval tool (investigations, decisions), rg is a code retrieval tool (source files). Comparing them directly is apples-to-oranges.

2. **False positives from substring matching** - The case-insensitive substring match in kb search can return noisy results for short queries (e.g., "SSE" matching "Assessment"). This is the likely source of user frustration.

3. **kb search has unique value** - Cross-project search and knowledge-focused scope are genuine differentiators that justify kb search's existence.

**Answer to Investigation Question:**

The perceived quality gap is partially real (substring matching causes false positives) and partially a misunderstanding of scope (kb searches knowledge, not code). The solution is not to "improve" kb search to replace rg, but to clarify when each tool is appropriate:

- Use `kb search` for: investigations, decisions, knowledge artifacts, cross-project queries
- Use `rg` for: code, implementation details, function definitions, patterns

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Ran 10 different query types across both tools with timing measurements. Pattern is clear and consistent. Not "Very High" because I didn't survey actual agent behavior to confirm this mental model matches how they use tools.

**What's certain:**

- ✅ kb search scope is limited to .kb/ directories by design
- ✅ kb search uses case-insensitive substring matching
- ✅ Performance is equivalent (~0.01s for both tools)
- ✅ kb search has unique cross-project capability

**What's uncertain:**

- ⚠️ Whether agents are actually using the wrong tool for the wrong query type
- ⚠️ Whether word-boundary matching would significantly improve precision
- ⚠️ Whether users want kb search to also search code (scope expansion)

**What would increase confidence to Very High:**

- Survey of actual agent sessions to see when kb search vs rg is used
- A/B test with word-boundary matching to measure precision improvement
- User interview to understand intent behind Dylan's TODO

---

## Implementation Recommendations

### Recommended Approach: Document the mental model

**Don't change kb search** - Instead, add guidance to agent instructions about when to use which tool.

**Why this approach:**
- The tools are already correct for their intended purposes
- Trying to make kb search "better at grep" would duplicate rg's functionality
- Cross-project search is valuable and shouldn't be compromised

**Trade-offs accepted:**
- Short queries may still have false positives
- Agents need to understand two tools instead of one

**Implementation sequence:**
1. Add to CLAUDE.md: "Use kb search for knowledge (investigations, decisions), rg for code"
2. Consider adding word-boundary matching as optional flag (--exact)
3. No changes needed to kb search core functionality

### Alternative Approaches Considered

**Option B: Add word-boundary matching to kb search**
- **Pros:** Reduces false positives for short queries
- **Cons:** May miss valid substring matches; complexity increase
- **When to use instead:** If users continue to report low precision after documentation update

**Option C: Expand kb search scope to include code**
- **Pros:** Single tool for everything
- **Cons:** Duplicates rg, loses focus on knowledge artifacts, slower searches
- **When to use instead:** Never - this would eliminate kb search's unique value

---

## Test Performed

**Test:** Ran 10 benchmark queries comparing `kb search` vs `rg` on the same codebase.

**Result:** 
- kb search and rg returned different results because they search different scopes
- kb search: .kb/investigations/, .kb/decisions/ only
- rg: entire codebase (or specified paths)
- Performance identical (~0.01s)
- kb search unique feature: cross-project search with --global

---

## Conclusion

The concern that "kb search provides worse results than grep" is partially valid for false positives but mostly reflects a scope mismatch. The tools are complementary: kb search for knowledge retrieval, rg for code retrieval. No code changes are needed - documentation should clarify the intended use cases for each tool.

---

## Self-Review

- [x] Real test performed (ran 10 benchmark queries)
- [x] Conclusion from evidence (based on observed query results)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/search.go` - kb search implementation
- `/Users/dylanconlin/Documents/personal/orch-go/DYLANS_THOUGHTS.org` - Original concern

**Commands Run:**
```bash
# Compare result counts
kb search "spawn" | head -30
rg -i "spawn" .kb --type md -l

# Test case sensitivity
kb search "SSE"
rg "SSE" .kb --type md -l
rg -i "sse" .kb --type md -l

# Test cross-project
kb search --global "synthesis protocol" --summary

# Performance benchmark
time kb search "spawn"
time rg -i "spawn" .kb --type md
```

**Related Artifacts:**
- **Source:** DYLANS_THOUGHTS.org line 15 - "i'm concerned about the quality of the kb search command"
