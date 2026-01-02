# Session Synthesis

**Agent:** og-debug-opencode-api-returns-23dec
**Issue:** orch-go-16vb
**Duration:** 2025-12-23 16:30 → 2025-12-23 16:40
**Outcome:** success

---

## TLDR

Investigated "redirect loop" errors on OpenCode API endpoints. Found this is expected OpenCode behavior - the local server proxies unknown routes to desktop.opencode.ai (a web app). The orch-go client already uses correct paths (`/session/{id}/prompt_async`) and is unaffected.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-23-debug-opencode-api-redirect-loop.md` - Investigation documenting root cause

### Files Modified
- None - no code changes needed

### Commits
- None - investigation-only session

---

## Evidence (What Was Observed)

- `/session` and `/session/{id}` return 200 OK (handled locally)
- `/session/{id}/prompt_async` returns 204 No Content (works correctly)
- `/health` returns 500 with "response redirected too many times" (proxied upstream)
- `/prompt_async` at root returns 500 with "UnexpectedRedirect fetching https://desktop.opencode.ai/prompt_async"
- `curl https://desktop.opencode.ai/health` returns HTML web app, not API response
- `pkg/opencode/client.go:177` - orch-go uses correct path `/session/{id}/prompt_async`

### Tests Run
```bash
# Verified working endpoints
curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:4096/session"  # 200
curl -s -o /dev/null -w "%{http_code}" -X POST "http://127.0.0.1:4096/session/ses_xxx/prompt_async" ...  # 204

# Verified failing endpoints  
curl -s -o /dev/null -w "%{http_code}" "http://127.0.0.1:4096/health"  # 500
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-23-debug-opencode-api-redirect-loop.md` - Documents OpenCode routing behavior

### Decisions Made
- No code changes needed: orch-go already uses correct API paths
- Close as non-issue: the "bug" is expected OpenCode upstream proxy behavior

### Constraints Discovered
- OpenCode routes: `/session/*` handled locally, other routes proxied to desktop.opencode.ai
- desktop.opencode.ai is a web app, not an API - returns HTML causing redirect loops

### Externalized via `kn`
- Not applicable - straightforward investigation with no new reusable constraints

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Tests passing (no code changes, verified correct behavior)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-16vb`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Why does OpenCode proxy unknown routes upstream rather than returning 404?
- Is there a local /health endpoint planned for OpenCode?

**Areas worth exploring further:**
- None directly relevant to orch-go

**What remains unclear:**
- Whether OpenCode's upstream proxy behavior is documented

*(Note: This is expected upstream behavior, not actionable for orch-go)*

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude
**Workspace:** `.orch/workspace/og-debug-opencode-api-returns-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-debug-opencode-api-redirect-loop.md`
**Beads:** `bd show orch-go-16vb`
