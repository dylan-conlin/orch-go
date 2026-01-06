# Session Synthesis

**Agent:** og-inv-can-we-auto-26dec
**Issue:** orch-go-zz42
**Duration:** 2025-12-26 18:34 → 2025-12-26 19:15
**Outcome:** success

---

## TLDR

Investigated whether orch should implement OAuth token auto-refresh. Conclusion: No - OpenCode already handles this automatically via the anthropic-auth plugin, which refreshes tokens transparently at fetch time when they expire.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-inv-can-we-auto-refresh-opencode.md` - Complete investigation documenting OpenCode's auto-refresh mechanism

### Files Modified
- None

### Commits
- Investigation file creation (to be committed)

---

## Evidence (What Was Observed)

- **OpenCode anthropic-auth plugin (v0.0.5)** contains auto-refresh logic: checks `auth.expires < Date.now()` before each API call and refreshes if needed (source: extracted index.mjs)
- **Refresh endpoint works**: `https://console.anthropic.com/v1/oauth/token` responds correctly with `grant_type: refresh_token`
- **Token storage is correct**: `~/.local/share/opencode/auth.json` contains `refresh`, `access`, and `expires` fields as expected
- **Prior investigation (orch-go-16vb)** confirmed: "redirected too many times" error is NOT auth-related - it's caused by hitting incorrect API endpoints that get proxied to desktop.opencode.ai

### Tests Run
```bash
# Verified Anthropic refresh endpoint exists and responds correctly
curl -s -X POST "https://console.anthropic.com/v1/oauth/token" \
  -H "Content-Type: application/json" \
  -d '{"grant_type": "refresh_token", "refresh_token": "invalid", "client_id": "..."}'
# Result: {"error": "invalid_grant", "error_description": "Refresh token not found or invalid"}
# Confirms endpoint is functional
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-can-we-auto-refresh-opencode.md` - Documents OpenCode's auto-refresh mechanism and answers the original question

### Decisions Made
- No orch implementation needed: OpenCode already handles token refresh automatically via plugin

### Constraints Discovered
- OpenCode auth plugins use a custom fetch wrapper pattern - they intercept all API requests and handle auth transparently
- The `expires` field in auth.json is milliseconds since epoch (not seconds)

### Externalized via `kn`
- `kn decide "OpenCode handles OAuth auto-refresh via anthropic-auth plugin" --reason "Code inspection shows token check at fetch time - no orch implementation needed"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete - Investigation documented
- [x] Tests passing - API endpoint tested
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-zz42`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What happens when the refresh token itself expires or is revoked? (OpenCode plugin throws error, but unclear how orch should handle)
- Does the refresh token have a limited lifetime? (Anthropic documentation not found)

**Areas worth exploring further:**
- Adding error detection in orch daemon for auth failures (currently no easy way to distinguish auth vs other errors)
- Whether OpenCode could expose auth status via API

**What remains unclear:**
- Exact error message/behavior when refresh token is invalid (we tested with invalid token, but not with expired refresh token)
- Whether there's a proactive way to check auth health without making an API call

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-20250514
**Workspace:** `.orch/workspace/og-inv-can-we-auto-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-can-we-auto-refresh-opencode.md`
**Beads:** `bd show orch-go-zz42`
