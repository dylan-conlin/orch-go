<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Antigravity auth plugin is NOT viable as a rate limit escape hatch due to active Google API revocation (Jan 15, 2026) and account ban risk.

**Evidence:** Issue #191 documents IAM_PERMISSION_DENIED errors starting Jan 15, 2026; README explicitly warns "small number of users have reported Google accounts being banned."

**Knowledge:** Google actively enforces against third-party Antigravity API access. The plugin's auth flow doesn't match official clients (missing onboarding email), triggering detection.

**Next:** Do not integrate. Continue with existing escape hatches (Docker fingerprint isolation, Claude CLI bypass). Monitor plugin repo for recovery but don't depend on it.

**Promote to Decision:** recommend-no (tactical evaluation, plugin is broken)

---

# Investigation: Evaluate Opencode Antigravity Auth Plugin

**Question:** Is opencode-antigravity-auth viable as an additional escape hatch for Claude Max rate limiting?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Dylan/Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Plugin Mechanism - OAuth Against Google Antigravity IDE

**Evidence:** The plugin enables OpenCode to authenticate with Google's Antigravity IDE via OAuth:
- Browser-based OAuth callback at `localhost:51121`
- Tokens stored locally after Google account grant
- Multi-account rotation strategies: "sticky" (1 account), "hybrid" (2-5 accounts), "round-robin" (5+)
- `pid_offset_enabled: true` distributes parallel sessions across accounts
- Exposes Claude models (Opus 4.5, Sonnet) through Google's quota instead of Anthropic's

**Source:** GitHub README analysis via WebFetch

**Significance:** The architecture is sound for bypassing Anthropic's direct rate limits by routing through Google's Antigravity quota. Multi-account rotation could spread load across accounts. This would be a 4th backend option (alongside claude, opencode, docker).

---

### Finding 2: Active API Revocation (Jan 15, 2026) - BLOCKING ISSUE

**Evidence:**
- Issue #191: "Google Antigravity API Permission Denied - IAM_PERMISSION_DENIED (Since 2026-01-15)"
- Error: `Permission 'cloudaicompanion.companions.generateChat' denied on resource '//cloudaicompanion.googleapis.com/projects/rising-fact-p41fc/locations/global'`
- Timeline: Morning of January 15, 2026
- Issue status: CLOSED (completed) but resolution unclear - no clear fix documented

**Source:** https://github.com/NoeFabris/opencode-antigravity-auth/issues/191

**Significance:** Google appears to have actively revoked API access for the Cloud AI Companion API that the plugin uses. This is a blocking issue - the plugin may not work at all currently. The "completed" closure status without clear resolution suggests workarounds or continued investigation, not a permanent fix.

---

### Finding 3: Account Ban Risk - Explicit Warning in Documentation

**Evidence:**
- README states: "Using this plugin may violate Google's Terms of Service"
- README warns: "A small number of users have reported their Google accounts being banned or shadow-banned"
- Fresh accounts and new accounts with paid subscriptions face higher risk
- Advisory: use established accounts, not new ones created for this plugin

**Source:** GitHub README analysis via WebFetch

**Significance:** Unlike our Docker fingerprint isolation (which only risks Anthropic rate throttling, not account bans), the Antigravity plugin risks Google account bans. Google accounts are often used for email, photos, docs - losing access has significant collateral damage.

---

### Finding 4: Auth Flow Detection Gap - Root Cause of Bans

**Evidence:**
- Issue #178 documents why accounts get banned
- Current auth flow doesn't trigger onboarding email ("Get started with Google Antigravity") that official clients do
- Accounts authenticated through non-official method receive "account ineligible for antigravity" error
- OAuth calls use "gemini cli node headers instead of antigravity ones"
- Test branch `feat/official-onboarding-flow` shows redirect triggers proper email, but implementation incomplete

**Source:** https://github.com/NoeFabris/opencode-antigravity-auth/issues/178

**Significance:** Google detects non-official usage because the auth flow fingerprint differs from legitimate Antigravity clients. This is analogous to Anthropic's TLS/HTTP2 fingerprinting - providers actively detect and block unofficial API access patterns.

---

### Finding 5: Comparison to Existing Escape Hatches

**Evidence:**

| Escape Hatch | Risk Level | What it Bypasses | Dependencies |
|--------------|------------|------------------|--------------|
| Docker fingerprint | Low (rate throttle only) | Device-level rate limits | Docker image, separate config dir |
| Claude CLI bypass | None (legitimate) | OpenCode server dependency | Max subscription, tmux |
| Antigravity OAuth | HIGH (account ban) | Anthropic quota (via Google) | Google account, plugin stability |

**Source:** Analysis comparing `.kb/models/model-access-spawn-paths.md` with plugin research

**Significance:** The Antigravity plugin has fundamentally different risk characteristics. Docker/Claude CLI risks are operational (rate throttling, tmux management). Antigravity risks are existential (Google account bans affecting email, photos, docs).

---

## Synthesis

**Key Insights:**

1. **Pattern Recognition: Same Cat-and-Mouse as Opus Workarounds** - Just as Anthropic blocks unofficial API access via TLS/HTTP2/header fingerprinting (documented in `2026-01-09-inv-anthropic-oauth-community-workarounds.md`), Google is doing the same for Antigravity. The plugin's value proposition depends on continued Google inaction, which Jan 15's API revocation proves is not guaranteed.

2. **Risk Asymmetry: Google Account Loss vs Rate Throttling** - Our existing escape hatches (Docker, Claude CLI) have bounded risk - at worst, rate throttling or subscription cost. Antigravity plugin risks losing a Google account that may have years of email, photos, and documents. The risk/reward is fundamentally different.

3. **Active Instability: Not a Mature Solution** - The plugin has an open issue (#178) for auth flow fixes to prevent bans, and a recently-closed-but-unclear (#191) for API permission denial. This is an actively-broken tool, not a stable escape hatch.

**Answer to Investigation Question:**

**No, opencode-antigravity-auth is NOT viable as an escape hatch.** Three factors make this unsuitable:

1. **Currently broken** - IAM_PERMISSION_DENIED errors since Jan 15, 2026
2. **High risk** - Account bans reported, Google accounts have significant collateral value
3. **Unstable long-term** - Google actively detecting and blocking unofficial access

Our existing escape hatches (Docker fingerprint isolation, Claude CLI bypass) remain the recommended approach. They have lower risk and don't depend on a third party's continued inaction.

---

## Structured Uncertainty

**What's tested:**

- ✅ Plugin exists and has documented functionality (verified: read README via WebFetch)
- ✅ API permission errors documented since Jan 15, 2026 (verified: Issue #191)
- ✅ Account ban warnings exist in documentation (verified: README explicitly warns)
- ✅ Auth flow detection documented as root cause (verified: Issue #178 analysis)

**What's untested:**

- ⚠️ Whether plugin currently works (didn't install/test - risk too high)
- ⚠️ How many accounts actually banned vs "small number" (no data)
- ⚠️ Whether Issue #191 resolution actually works (status unclear)
- ⚠️ Integration complexity with orch spawn (didn't prototype)

**What would change this:**

- If Google reverses API revocation AND auth flow fix lands (#178)
- If ban rate is negligible AND Dylan has throwaway Google account
- If Anthropic makes Max subscription unavailable/expensive enough to justify risk

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Do Not Integrate** - Continue with existing escape hatches; do not add Antigravity backend to orch spawn.

**Why this approach:**
- Plugin is currently broken (Jan 15 API revocation)
- Risk profile incompatible with Dylan's use case (Google account loss too costly)
- Existing escape hatches sufficient (Docker for rate limits, Claude CLI for independence)

**Trade-offs accepted:**
- Lose potential additional quota source
- Stay dependent on Anthropic's Max subscription economics

**Implementation sequence:**
1. None - this is a "do nothing" recommendation
2. Optionally: Add GitHub watch on repo to monitor for recovery/stabilization
3. Re-evaluate in 6 months if situation changes

### Alternative Approaches Considered

**Option B: Install with throwaway Google account**
- **Pros:** Low-risk testing, understand real behavior
- **Cons:** Plugin currently broken per Issue #191, even testing would fail
- **When to use instead:** If API revocation is resolved AND you want to validate claims

**Option C: Wait for official auth flow fix (Issue #178)**
- **Pros:** Would reduce ban risk if implemented
- **Cons:** Issue open since creation, no clear timeline, doesn't address Jan 15 API revocation
- **When to use instead:** If #178 and #191 both resolved, might reconsider

**Rationale for recommendation:** The plugin's value proposition (bypass Anthropic rate limits via Google quota) is undermined by (a) current breakage and (b) risk of losing valuable Google account. Our existing Docker escape hatch provides rate limit isolation without these risks.

---

### Implementation Details

**What to implement first:**
- Nothing - recommendation is to not integrate

**Things to watch out for:**
- ⚠️ If Anthropic significantly changes Max subscription, may need to revisit third-party options
- ⚠️ If plugin stabilizes AND auth flow fix lands, worth re-evaluating
- ⚠️ Community may fork/create alternative implementations

**Areas needing further investigation:**
- None currently - blocked by plugin instability

**Success criteria:**
- ✅ Decision documented (this investigation)
- ✅ Existing escape hatches continue working
- ✅ No time invested in broken/risky integration

---

## References

**Files Examined:**
- `.kb/models/model-access-spawn-paths.md` - Current escape hatch architecture
- `.kb/guides/resilient-infrastructure-patterns.md` - Escape hatch design principles
- `.kb/guides/opencode-plugins.md` - OpenCode plugin integration patterns

**Commands Run:**
```bash
# Create investigation file
kb create investigation evaluate-opencode-antigravity-auth-plugin
```

**External Documentation:**
- https://github.com/NoeFabris/opencode-antigravity-auth - Plugin repository
- https://github.com/NoeFabris/opencode-antigravity-auth/issues/191 - API Permission Denied since Jan 15
- https://github.com/NoeFabris/opencode-antigravity-auth/issues/178 - Official auth flow proposal

**Related Artifacts:**
- **Model:** `.kb/models/model-access-spawn-paths.md` - Documents existing escape hatch architecture
- **Decision:** `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` - Current Max subscription strategy
- **Investigation:** `2026-01-09-inv-anthropic-oauth-community-workarounds.md` - Similar cat-and-mouse pattern with Anthropic

---

## Investigation History

**2026-01-23 02:30:** Investigation started
- Initial question: Is opencode-antigravity-auth viable as rate limit escape hatch?
- Context: Dylan dealing with 'opus gate' rate limiting issues

**2026-01-23 02:45:** Research completed
- Found active API revocation (Issue #191) since Jan 15
- Found explicit account ban warnings in README
- Found auth flow detection as root cause (Issue #178)
- Compared to existing escape hatch risk profiles

**2026-01-23 03:00:** Investigation completed
- Status: Complete
- Key outcome: NOT viable - currently broken, high risk, unstable long-term
