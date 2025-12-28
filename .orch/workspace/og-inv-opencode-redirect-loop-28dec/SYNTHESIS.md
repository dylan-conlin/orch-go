# Session Synthesis

**Agent:** og-inv-opencode-redirect-loop-28dec
**Issue:** orch-go-yhwn
**Duration:** 2025-12-28 15:30 → 2025-12-28 15:55
**Outcome:** success

---

## TLDR

OpenCode's "redirected too many times" error is NOT a bug - it's intentional behavior where unknown routes are proxied to `desktop.opencode.ai` (a web app). This has been investigated 4 times now; the real issue is knowledge surfacing, not the behavior itself.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-28-inv-opencode-redirect-loop-health-sessions.md` - Comprehensive investigation consolidating 3+ prior findings

### Files Modified
- None

### Commits
- (pending) Investigation file documenting root cause and consolidating prior findings

---

## Evidence (What Was Observed)

- `/session` endpoint returns 200 OK with session list (verified: `curl http://127.0.0.1:4096/session`)
- `/health` endpoint returns 500 with "redirected too many times" (verified: `curl http://127.0.0.1:4096/health`)
- OpenCode `server.ts:2475-2482` has catch-all proxy: `.all("/*")` → `proxy(desktop.opencode.ai)`
- orch-go client only uses valid endpoints (`/session`, `/session/{id}/*`, `/event`)
- 3 prior investigations exist on this exact topic, all reaching same conclusion

### Tests Run
```bash
# Test valid endpoint
curl -s -w "\nStatus: %{http_code}\n" http://127.0.0.1:4096/session
# Status: 200

# Test invalid endpoint
curl -s -w "\nStatus: %{http_code}\n" http://127.0.0.1:4096/health
# {"name":"UnknownError","data":{"message":"Error: The response redirected too many times..."}}
# Status: 500
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-28-inv-opencode-redirect-loop-health-sessions.md` - Consolidates 4 investigations on this topic

### Decisions Made
- No code changes needed - orch-go uses correct endpoints
- This is expected OpenCode behavior, not a bug to fix

### Constraints Discovered
- OpenCode has NO `/health` endpoint - use `/session` for health checks
- The catch-all proxy to `desktop.opencode.ai` is intentional design
- Error message "redirected too many times" is misleading (not auth-related)

### Externalized via `kn`
- N/A - constraint already exists as `kn-2a4e34`: "OpenCode has no /health endpoint - use /session to check server status"

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file)
- [x] Tests passing (verified endpoints work correctly)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-yhwn`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why does this issue get re-investigated every few weeks? (Possible: knowledge not surfaced in spawn context, misleading error message triggers new investigations)
- Should OpenCode add a local `/health` endpoint? (Would prevent confusion)

**Areas worth exploring further:**
- Better knowledge surfacing to prevent duplicate investigations
- Possible orch lint rule to detect when spawn tasks duplicate prior investigations

**What remains unclear:**
- Whether OpenCode team would accept a PR to add local `/health` endpoint

---

## Session Metadata

**Skill:** investigation
**Model:** Opus
**Workspace:** `.orch/workspace/og-inv-opencode-redirect-loop-28dec/`
**Investigation:** `.kb/investigations/2025-12-28-inv-opencode-redirect-loop-health-sessions.md`
**Beads:** `bd show orch-go-yhwn`
