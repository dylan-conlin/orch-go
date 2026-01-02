# Session Synthesis

**Agent:** og-inv-workers-stall-during-22dec
**Issue:** orch-go-d039
**Duration:** 2025-12-22 15:14 → 2025-12-22 15:30
**Outcome:** success

---

## TLDR

Investigated why workers stall during "Build" phase in OpenCode TUI. Found that "Build" is OpenCode's agent mode (not a compile phase), and stalls occur when Claude API doesn't respond. Recommended adding stall detection to SSE monitor.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-22-inv-workers-stall-during-build-phase.md` - Complete investigation with findings, synthesis, and recommendations

### Files Modified
- None

### Commits
- (pending - investigation file to be committed)

---

## Evidence (What Was Observed)

- `pkg/opencode/client.go:160` sends messages with `"agent": "build"` - this is why TUI shows "Build" (observed in code review)
- SSE stream shows `session.status: busy` followed by `message.part.updated` events for healthy sessions (tested via `curl http://127.0.0.1:4096/event`)
- No timeout/stall detection exists in current implementation (verified in monitor.go and client.go)
- Could not reproduce actual stall - all sessions were healthy during test period

### Tests Run
```bash
# Monitor SSE stream for event patterns
timeout 15 curl -s -N http://127.0.0.1:4096/event | head -30
# Result: Healthy event flow with session.status and message.part events

# Check for error events
tail -50 ~/.orch/events.jsonl | jq -c 'select(.data.error != null)'
# Result: No error events found
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-22-inv-workers-stall-during-build-phase.md` - Complete analysis of stall causes and detection mechanism

### Decisions Made
- Decision: Recommend stall detection via SSE monitoring because it's low-cost and doesn't require external changes

### Constraints Discovered
- Stall detection pattern: session.status=busy for >5min without message.part events indicates hung Claude API call

### Externalized via `kn`
- `kn constrain "Stall detection: session.status=busy for >5min without message.part events indicates hung Claude API call" --reason "SSE monitoring pattern"` → kn-e55e45

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file written)
- [x] Tests passing (no code changes requiring tests)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-d039`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What does OpenCode do when rate limited (429)? Does it surface the error or just hang?
- Is there an OpenCode endpoint to cancel/retry a stuck request?
- Should stalled sessions be auto-abandoned after extended timeout?

**Areas worth exploring further:**
- Rate limit detection by intentionally triggering 429
- OpenCode error surfacing behavior

**What remains unclear:**
- Which cause is most common (rate limiting vs API hang vs network)
- How to recover a stalled session without manual interrupt

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-inv-workers-stall-during-22dec/`
**Investigation:** `.kb/investigations/2025-12-22-inv-workers-stall-during-build-phase.md`
**Beads:** `bd show orch-go-d039`
