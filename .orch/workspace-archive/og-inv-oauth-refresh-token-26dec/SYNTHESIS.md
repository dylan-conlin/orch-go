# Session Synthesis

**Agent:** og-inv-oauth-refresh-token-26dec
**Issue:** (ad-hoc spawn, no beads tracking)
**Duration:** ~30 minutes
**Outcome:** success

---

## TLDR

Investigated why OAuth refresh tokens become invalid. Root cause is architectural: Anthropic rotates tokens on EVERY refresh (security by design), and orch-go + OpenCode maintain independent token storage, creating split-brain state. No code fix needed - the GetAccountCapacity bug was already fixed (commit 2f77d738). Provided operational guidance for users.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-inv-oauth-refresh-token-invalidation-root.md` - Full investigation with D.E.K.N. summary

### Files Modified
- None (investigation only)

### Commits
- (pending commit of investigation file)

---

## Evidence (What Was Observed)

1. **Token rotation is intentional** - opencode-anthropic-auth plugin shows `json.refresh_token` replaces old token on every refresh (index.mjs:102-108)

2. **Split-brain architecture confirmed** - Two independent storage locations:
   - OpenCode: `~/.local/share/opencode/auth.json`
   - orch-go: `~/.orch/accounts.yaml`

3. **Prior fix exists** - GetAccountCapacity (pkg/account/account.go:544-577) now updates OpenCode auth.json when querying the active account

4. **OpenCode auto-refresh is robust** - Plugin checks expiry before every API call, refreshes and persists atomically

5. **Git history shows iterative fixes** - 6 OAuth-related commits in December 2025

### Tests Run
```bash
# Verified current token storage state
cat ~/.local/share/opencode/auth.json  # OpenCode has tokens
cat ~/.orch/accounts.yaml              # orch-go has matching tokens

# Verified git history for prior fixes
git log --oneline --since="2025-12-01" -- "*.go" | grep -i token|oauth|auth
# Shows 6 relevant commits including GetAccountCapacity fix

# Extracted and analyzed opencode-anthropic-auth plugin
npm pack opencode-anthropic-auth@0.0.5
# Confirms token rotation behavior in plugin code
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-oauth-refresh-token-invalidation-root.md` - Comprehensive root cause analysis

### Decisions Made
- No code fix needed - issue is architectural, not a bug
- Operational guidance is the appropriate solution

### Constraints Discovered
- Token rotation is OAuth 2.0 security best practice (RFC 6749, 6819)
- Any process that refreshes must immediately persist new tokens
- Split-brain state is inherent to current orch-go/OpenCode architecture

### Externalized via `kn`
- N/A - Knowledge captured in investigation file

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Tests passing (N/A - investigation only)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete`

### Operational Guidance (for documentation)

1. **After orch operations that refresh tokens**: Consider running `orch account switch <current>` to resync OpenCode auth.json

2. **Don't run capacity checks while agents are running**: `GetAccountCapacity` and `ListAccountsWithCapacity` refresh tokens

3. **When re-auth required**: `orch account remove <name> && orch account add <name>`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**

- What is Anthropic's refresh token lifetime? (Unknown, not publicly documented)
- Are there other orch-go operations that refresh tokens? (SwitchAccount and GetAccountCapacity are the main ones)
- Should orch-go defer all token operations to OpenCode? (Architectural decision needed)

**Areas worth exploring further:**

- Production monitoring of token invalidation frequency
- Anthropic documentation on OAuth token policies

**What remains unclear:**

- Exact triggers for Anthropic token revocation
- Whether any long-term token lifetime exists

---

## Session Metadata

**Skill:** investigation
**Model:** claude-opus-4-20250514
**Workspace:** `.orch/workspace/og-inv-oauth-refresh-token-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-oauth-refresh-token-invalidation-root.md`
**Beads:** (ad-hoc spawn, no tracking)
