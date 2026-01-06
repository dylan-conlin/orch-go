# Session Synthesis

**Agent:** og-inv-benchmark-kb-search-20dec
**Issue:** orch-go-2or
**Duration:** 2025-12-20 ~19:17 → ~19:35
**Outcome:** success

---

## TLDR

Investigated whether `kb search` provides worse retrieval than grep - conclusion is the tools serve different purposes (kb for knowledge artifacts, rg for code) and are complementary rather than competing. No code changes needed.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-20-inv-benchmark-kb-search-vs-grep.md` - Complete investigation with 5 findings

### Files Modified
- None - this was an investigation, not an implementation

### Commits
- Investigation file to be committed

---

## Evidence (What Was Observed)

- `kb search "spawn"` returned 104 results from .kb/ investigations and decisions
- `rg "spawn" --type go` returned 12 Go source files - completely different scope
- kb search uses case-insensitive substring matching (search.go:294-333)
- Performance is identical: both ~0.01s for typical queries
- kb search has unique `--global` flag for cross-project search

### Tests Run
```bash
# Benchmark 10 different query types
kb search "spawn" vs rg -i "spawn" .kb --type md
kb search "SSE" vs rg "SSE" .kb vs rg -i "sse" .kb
kb search "beads issue" vs rg "beads.*issue" --type go
kb search --global "synthesis protocol" --summary
time kb search "spawn" # 0.00-0.01s
time rg -i "spawn" .kb # 0.00-0.01s
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-20-inv-benchmark-kb-search-vs-grep.md` - Full benchmark analysis

### Decisions Made
- Decision: kb search and rg are complementary tools, not competing - because kb searches knowledge (.kb/), rg searches code

### Constraints Discovered
- kb search intentionally limits scope to .kb/investigations/ and .kb/decisions/
- Case-insensitive substring matching can cause false positives for short queries

### Externalized via `kn`
- `kn decide "kb search and rg are complementary tools" --reason "kb searches knowledge artifacts (.kb/), rg searches code - agents should use both strategically based on query type"` - kn-a92bcb

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file with DEKN summary)
- [x] Tests passing (10 benchmark queries executed)
- [x] Investigation file has Status: Complete
- [x] Ready for `orch complete orch-go-2or`

### Optional Follow-up
Consider adding guidance to agent instructions about when to use each tool:
- `kb search` for: investigations, decisions, knowledge, cross-project queries
- `rg` for: code, implementation details, function definitions, patterns

---

## Session Metadata

**Skill:** investigation
**Model:** claude
**Workspace:** `.orch/workspace/og-inv-benchmark-kb-search-20dec/`
**Investigation:** `.kb/investigations/2025-12-20-inv-benchmark-kb-search-vs-grep.md`
**Beads:** `bd show orch-go-2or`
