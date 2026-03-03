---
date: "2025-11-30"
updated: "2025-12-12"
status: "Resolved via Docker workaround"
github_issue: https://github.com/anthropics/claude-code/issues/12786
canonical_issue: https://github.com/anthropics/claude-code/issues/630
---

# Claude Code Cross-Account Rate Limit Bug

**TLDR:** Claude Code incorrectly applies rate limits from one Max account to another on the same device. Despite Account B showing 4% usage, Claude Code blocks it after Account A hit its limit. Server-side bug - Anthropic has known since **March 2025** (9 months) and hasn't fixed it.

## Question

Why does Claude Code show "weekly limit reached" for an account with only 4% usage, and how can it be fixed?

## Context

- Two Claude Max subscriptions:
  - `dylan.conlin@sendcutsend.com` - at weekly limit (80% on 2025-12-12)
  - `dylan.conlin@gmail.com` - fresh quota available
- After hitting limit on sendcutsend account, gmail account is also blocked in Claude Code
- Gmail account works fine in claude.ai web interface

## GitHub Issue Status (2025-12-12)

The issue has been bounced around as duplicates:
- #12786 (mine) → closed as dup of #12190
- #12190 → closed as dup of #3857
- #3857 → closed as dup of #5001
- #5001 → closed as dup of #3857 (circular!)
- **#630 is the canonical OPEN issue** (since March 2025)

Anthropic's only response (March 2025):
> "We'll keep this in mind for a future enhancements."

They treated it as a feature request rather than a bug. Issue #630 is about to auto-close in 30 days due to inactivity.

## What I tried

1. `claude logout` → `claude login` with gmail account - still blocked
2. Deleted `~/.claude/statsig/` directory (including stable_id) - still blocked after re-login
3. Revoked ALL authorization tokens at claude.ai/settings/claude-code - still blocked
4. Fresh login after clearing everything - still blocked
5. **(2025-12-12)** Deleted keychain entry (`security delete-generic-password -s "Claude Code-credentials"`) → re-logged in with gmail → **still blocked**

## What I observed

1. **Statsig stores device identifiers:**
   - `~/.claude/statsig/statsig.stable_id.*` contains a UUID device identifier
   - Deleting and re-logging generates a NEW UUID, but doesn't fix the issue

2. **Statsig cached evaluations contain multiple IDs:**
   ```
   userID:           7115c82fae2cbe6fc52db585ba14f8d9600c1f8e5bf5810417ee947f278bff79
   stableID:         9386f434-49f6-4342-8e51-ccffdb947f09
   accountUUID:      ece16e48-2ec7-46f9-b97b-cc4c110ae50d
   organizationUUID: cd188ad7-9a43-40bb-bb70-f8d9107fa0e0
   ```

3. **OpenCode works with the same gmail account** - This proves the account has available quota

4. **OpenCode and Claude Code use the same OAuth client ID:**
   ```javascript
   const CLIENT_ID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
   ```

5. **Key difference:** OpenCode doesn't use Statsig - it makes raw API calls with OAuth tokens. Claude Code sends additional telemetry that triggers different rate limit enforcement.

6. **(2025-12-12) Device fingerprinting investigation:**
   - Hardware UUID: `75216945-4E53-5558-BC49-393DF90AD35D`
   - The `userID`/`device_id` is NOT a direct hash of hardware UUID (tested SHA256 variants)
   - Likely derived from combination of identifiers or generated and stored
   - Keychain entry (`Claude Code-credentials`) stores credentials by local username, not email
   - `organizationUUID` (`cd188ad7...`) appears to be the sendcutsend org - possible source of contamination

7. **Statsig domain:** `api.statsigcdn.com` - could potentially be blocked via /etc/hosts but risky

## Test performed

**Test:** Use OpenCode with gmail account immediately after Claude Code rejects it

**Result:** OpenCode works perfectly. Same OAuth credentials, same client ID, same account - but OpenCode bypasses whatever is blocking Claude Code.

## Conclusion

The rate limit is enforced **server-side** based on something Claude Code sends that OpenCode doesn't (likely Statsig telemetry or device fingerprinting). The server incorrectly associates the device with the exhausted sendcutsend account's rate limit state, even when authenticated with a different account.

This is an Anthropic server-side bug - no local fix is possible.

---

## Workarounds

### 1. OpenCode (confirmed working)

Use OpenCode instead of Claude Code. OpenCode uses the same OAuth flow but doesn't trigger the erroneous rate limiting (no Statsig telemetry).

### 2. Docker Container (in progress - 2025-12-12)

Run Claude Code in a Docker container to get a fresh device fingerprint:

```dockerfile
FROM node:22-slim
RUN npm install -g @anthropic-ai/claude-code
WORKDIR /workspace
ENTRYPOINT ["claude"]
```

```bash
# Build
docker build -t claude-clean .

# Run (mounts project, persistent ~/.claude in volume)
docker run -it --rm \
  -v $(pwd):/workspace \
  -v claude-config:/root/.claude \
  -v ~/.gitconfig:/root/.gitconfig:ro \
  -v ~/.ssh:/root/.ssh:ro \
  claude-clean
```

**Why this should work:**
- Container has different (virtualized) hardware identifiers
- Fresh `~/.claude/` directory
- New stableID generated on first run
- Server sees it as a completely different device

**Status:** ✅ CONFIRMED WORKING (2025-12-12)

**Why not used in practice:** While technically working, the Docker workaround was abandoned because:
1. **tmux-in-tmux problem** - Dylan's workflow uses tmux on the host; running Claude Code inside Docker creates nested tmux sessions (confusing keybindings, session management issues)
2. **Environment isolation** - Would require rebuilding entire development environment inside the container (tools, configs, credentials) rather than using existing host setup
3. **Workflow friction** - Easier to switch to OpenCode or Gemini than to maintain a parallel Docker-based workflow

### 3. Block Statsig (risky, untested)

Add to `/etc/hosts`:
```
127.0.0.1 api.statsigcdn.com
```

Risk: May break Claude Code entirely.

### 4. Wait for weekly reset

Just wait until the sendcutsend account resets. Sucks but guaranteed to work.

## Actions Taken

- Filed GitHub issue: https://github.com/anthropics/claude-code/issues/12786 (closed as dup)
- Canonical issue: https://github.com/anthropics/claude-code/issues/630 (open since March 2025)
- Email to support@anthropic.com (pending)
- Recorded kn decision: `kn-f31d6d` - account switching requires keychain deletion + re-login (doesn't fix rate limit bug though)

## Technical Details

### OpenCode's OAuth Implementation

OpenCode uses the `opencode-anthropic-auth` plugin which:
1. Uses same client ID as Claude Code
2. Performs PKCE OAuth flow to `claude.ai/oauth/authorize`
3. Exchanges code at `console.anthropic.com/v1/oauth/token`
4. Injects Bearer token with `anthropic-beta: oauth-2025-04-20` header
5. Does NOT use Statsig or send device telemetry

### Claude Code's Additional Tracking

Claude Code includes Statsig integration that tracks:
- Device stable ID (persists across sessions)
- Session ID
- Account UUID
- Organization UUID
- Feature flags and experiments

This additional tracking appears to be where the cross-account rate limit contamination occurs.

## Next Steps

1. [ ] Build and test Docker container workaround
2. [ ] If Docker works, create a proper setup with all needed tools (git, node, python, etc.)
3. [ ] Consider commenting on #630 to keep it alive and clarify it's a bug, not feature request
4. [ ] Consider direct email to Anthropic support with technical details

## Related

- `.orch/investigations/simple/2025-11-26-consumer-oauth-api-pattern.md` - How OpenCode uses Max subscription via OAuth
- `kn-f31d6d` - Account switching requires keychain deletion (discovered 2025-12-12)
- `kn-9f9558` - Claude Max usage can be checked via undocumented API endpoint
