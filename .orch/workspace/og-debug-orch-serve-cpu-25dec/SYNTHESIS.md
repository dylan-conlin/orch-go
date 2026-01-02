# Session Synthesis

**Agent:** og-debug-orch-serve-cpu-25dec
**Issue:** orch-go-7iis
**Duration:** 2025-12-25 22:00 → 2025-12-25 22:45
**Outcome:** success

---

## TLDR

Fixed recurring CPU runaway in `orch serve` caused by O(n*m) file operations per /api/agents request. Implemented workspace caching to reduce to O(m) operations once per request.

---

## Delta (What Changed)

### Files Modified
- `cmd/orch/serve.go` - Added workspace cache, replaced directory scanning with O(1) lookups

### Commits
- `e0303569` - fix: orch serve CPU runaway - cache workspace scanning to reduce I/O

---

## Evidence (What Was Observed)

- Previous fix (ed772bac) removed IsProcessing HTTP calls and added 500ms debounce, but CPU still hit 124%
- Root cause: `handleAgents()` calls `findWorkspaceByBeadsID()` for each agent (O(n))
- `findWorkspaceByBeadsID()` scans all ~466 workspace directories (O(m))
- With SSE-triggered refetches every 500ms, this creates O(n*m*2) file operations per second
- 5-minute stress test showed CPU at 4-7% during polling (down from 124%)
- CPU returns to 0% when polling stops (confirms no goroutine leaks)

### Tests Run
```bash
# Build verification
go build ./cmd/orch/...  # PASS

# Unit tests
go test ./cmd/orch/... -v -count=1  # PASS
go test ./pkg/... -v -count=1  # PASS

# Smoke test: 5 minutes continuous polling
# Minute 1: CPU=5.2%
# Minute 2: CPU=7.1%
# Minute 3: CPU=4.8%
# Minute 4: CPU=7.2%
# Minute 5: CPU=4.7%
# Final after settling: 0.0%
```

---

## Knowledge (What Was Learned)

### Decisions Made
- Cache workspace metadata once per request rather than per-agent lookup
- Added pprof at /debug/pprof/ for future diagnostics (no runtime cost when unused)

### Constraints Discovered
- Dashboard polling at 500ms with frequent SSE events creates high request volume
- ~466 workspace directories exist - any per-workspace operation needs caching

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] 5-minute smoke test confirms fix
- [x] Ready for `orch complete orch-go-7iis`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should workspace cache be persistent across requests? (Currently rebuilt each request)
- Could workspace enumeration be lazy-loaded or paginated for very large counts?

**Areas worth exploring further:**
- Profile memory usage during high polling scenarios
- Consider debouncing /api/agents at server level, not just client

**What remains unclear:**
- Original 124% CPU may have had multiple contributing factors; this fix addresses the O(n*m) issue

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus-4
**Workspace:** `.orch/workspace/og-debug-orch-serve-cpu-25dec/`
**Investigation:** `.kb/investigations/2025-12-25-inv-orch-serve-cpu-runaway-recurring.md`
**Beads:** `bd show orch-go-7iis`
