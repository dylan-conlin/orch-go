## Summary (D.E.K.N.)

**Delta:** `orch status` performance improved from 15s/437 lines to 1.6s/75 lines through compact mode and IsSessionProcessing optimization.

**Evidence:** Before: 15.27s, 437 lines. After: 1.6s, 75 lines. Verified via `time orch status`.

**Knowledge:** Main bottleneck was `IsSessionProcessing` calling GetMessages for ALL sessions (O(n) HTTP calls). Fixed by: (1) only checking recently updated sessions, (2) compact mode showing only running/needing-attention agents.

**Next:** Close issue - targets met (<2s execution, significant line reduction).

**Promote to Decision:** recommend-no - tactical optimization, not architectural change.

---

# Investigation: Fix Orch Status Performance Output

**Question:** How to fix `orch status` performance (15s execution) and output verbosity (437 lines)?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: IsSessionProcessing was the main bottleneck

**Evidence:** `IsSessionProcessing()` calls `GetMessages()` which makes an HTTP request per session. With 106 active agents, this was ~106 HTTP calls taking ~100ms each = ~10.6s overhead.

**Source:** `pkg/opencode/client.go:376-402`, `cmd/orch/status_cmd.go:297,310,395,498`

**Significance:** This was the primary cause of the 15s execution time.

---

### Finding 2: Token fetching added more overhead

**Evidence:** `GetSessionTokens()` was called for EVERY filtered agent, making another HTTP call per agent (~100ms each).

**Source:** `cmd/orch/status_cmd.go:574-582`

**Significance:** Secondary performance issue, especially when showing all agents.

---

### Finding 3: Default mode showed too many agents

**Evidence:** 106 active agents, 134 completed, many with unknown task/phase. All were shown by default.

**Source:** `orch status` output before fix

**Significance:** Most agents were idle and not actionable. User only cares about running agents and those needing attention.

---

## Synthesis

**Key Insights:**

1. **Session staleness as proxy for processing status** - Sessions not updated in 5+ minutes are definitely not processing. Skip the expensive HTTP check for these.

2. **Compact mode reduces both I/O and output** - Only fetching tokens/risk for running agents AND only showing actionable agents reduces both API calls and output lines.

3. **Phase-based filtering** - Running + Phase:Complete/BLOCKED/QUESTION captures all actionable agents without showing idle noise.

**Answer to Investigation Question:**

Implemented compact default mode with optimized `IsSessionProcessing`. Results:
- Execution: 15.27s -> 1.6s (89% improvement, meets <2s target)
- Output: 437 lines -> 75 lines (83% reduction)
- `--all` flag available for full output when needed

---

## Structured Uncertainty

**What's tested:**

- ✅ Compact mode execution time <2s (verified: `time orch status` = 1.6s)
- ✅ `--all` mode still works (verified: shows 296 lines with all agents)
- ✅ Running agents always shown (verified: filters check `IsProcessing`)
- ✅ Phase: Complete/BLOCKED/QUESTION agents shown (verified: string comparison in filter)

**What's untested:**

- ⚠️ Edge case: What if session was updated 5:01 minutes ago but still processing? (unlikely, processing sessions update frequently)
- ⚠️ High agent count (1000+) performance (only tested with ~100)

**What would change this:**

- Finding would be wrong if sessions stop updating while processing
- Line count target might need revisiting if many agents have Phase: Complete simultaneously

---

## Implementation Recommendations

### Recommended Approach ⭐

**Compact mode as default with optimized processing check** - Default shows only actionable agents, `--all` for full view.

**Why this approach:**
- Addresses both performance (API calls) and UX (output length)
- Preserves full functionality via `--all` flag
- No data loss - just presentation change

**Trade-offs accepted:**
- Users might not see idle agents without `--all`
- Accepted: status message shows idle count with hint

**Implementation sequence:**
1. Add `isSessionLikelyProcessing()` helper (skip HTTP for stale sessions)
2. Update filtering to compact mode (running + attention-needed only)
3. Skip token/risk fetch for non-running in compact mode
4. Add hint message about hidden agents

### Alternative Approaches Considered

**Option B: Parallel HTTP calls**
- **Pros:** Would speed up without changing what's shown
- **Cons:** Still making 100+ calls, just faster; doesn't fix output verbosity
- **When to use instead:** If compact mode filtering isn't acceptable

**Option C: Server-side batch endpoint**
- **Pros:** Single HTTP call for all processing status
- **Cons:** Requires OpenCode server changes; more work
- **When to use instead:** If orch status needs to support 1000+ agents

---

## References

**Files Examined:**
- `cmd/orch/status_cmd.go` - Main status command implementation
- `pkg/opencode/client.go` - IsSessionProcessing and GetMessages

**Commands Run:**
```bash
# Baseline
time orch status 2>&1 | wc -l  # 437 lines, 15.27s

# After fix
time /tmp/orch-test status 2>&1 | wc -l  # 75 lines, 1.6s
```

---

## Investigation History

**2026-01-16 12:00:** Investigation started
- Initial question: Fix 15s/437 lines performance/verbosity issue
- Context: Spawned from issue orch-go-21zst

**2026-01-16 12:30:** Root cause identified
- IsSessionProcessing making O(n) HTTP calls
- Default showing all agents regardless of state

**2026-01-16 13:00:** Implementation complete
- Compact mode with optimized processing check
- Results: 1.6s/75 lines (89%/83% improvement)
