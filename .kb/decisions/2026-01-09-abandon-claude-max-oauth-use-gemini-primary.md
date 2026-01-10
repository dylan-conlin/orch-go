# Decision: Abandon Claude Max OAuth, Use Gemini Flash as Primary Model

**Date:** 2026-01-09
**Status:** Accepted
**Context:** Anthropic OAuth blocking, community workaround fragility
**Scope:** orch-go model selection strategy

---

## Decision

**Abandon Claude Max OAuth workarounds. Use Gemini Flash as primary model, Sonnet API key as fallback.**

## Context

### What Happened

On January 9, 2026, Anthropic blocked all third-party OAuth access to Claude Max subscriptions. The error:

> "This credential is only authorized for use with Claude Code and cannot be used for other API requests."

This affects OpenCode and all third-party tools using Claude Max OAuth.

### Investigation History

1. **Jan 8, 2026**: Attempted header spoofing to bypass Opus 4.5 auth gate
   - Result: Failed, caused Gemini Flash hangs and zombie agents
   - Investigation: `2026-01-08-inv-opus-auth-gate-fingerprinting.md`

2. **Jan 9, 2026**: Synthesized community workarounds
   - Community published 5 different bypass methods
   - All have critical flaws for orch-go
   - Investigation: `2026-01-09-inv-anthropic-oauth-community-workarounds.md`

### Community Workarounds Assessed

| Workaround | Reliability | Longevity | Maintenance | Integration | Verdict |
|------------|-------------|-----------|-------------|-------------|---------|
| Official plugin (0.0.7) | ⚠️ Worked initially | ❌ 6 hours | ✅ Easy | ✅ Good | **Fragile** |
| Tool renaming | ✅ Confirmed | ⚠️ Unknown | ❌ High | ❌ Poor | **Unsustainable** |
| Rotating suffix | ✅ Resilient | ⚠️ Days? | ❌ Very high | ❌ Poor | **Not worth it** |
| Kilo Code | ✅ Stable | ✅ Long-term | ✅ Low | ❌ VS Code only | **Wrong environment** |
| Alternative models | ✅ Perfect | ✅ Permanent | ✅ Zero | ✅ Perfect | **Winner** |

### Key Findings

1. **Workarounds are fragile**: Official plugin got re-blocked within 6 hours
2. **Anthropic wins cat-and-mouse**: They iterate faster than community can stabilize bypasses
3. **Source edits are risky**: Jan 8 investigation showed they cause hangs and zombie agents
4. **Maintenance burden too high**: Dylan's OpenCode fork conflicts with workaround patches
5. **Alternatives work perfectly**: Gemini Flash already default, Sonnet API available

## Rationale

### Why NOT Claude Max OAuth Workarounds

**Technical fragility:**
- Official plugin lasted 6 hours before re-blocking
- Tool renaming requires source edits on every OpenCode update
- Rotating suffix requires maintaining forked builds
- All approaches conflict with Dylan's OpenCode fork workflow

**Risk to orch-go stability:**
- Jan 8 investigation: Header spoofing caused Gemini Flash hangs
- Jan 8 investigation: Created zombie agents (5 cleanup required)
- Source modifications unpredictable in production
- Could break existing working workflows

**Maintenance burden:**
- Ongoing cat-and-mouse game with Anthropic
- Requires monitoring for re-blocking
- Requires updating bypasses when blocked
- Distracts from actual orch-go development

**Community consensus:**
- 119+ users upvoted canceling Claude Max subscriptions
- Quote: "I've cancelled my Anthropic subscription completely... Will explore other models."
- Migration to alternatives is dominant response

### Why Gemini Flash as Primary

**Already working:**
- Current default model in orch-go
- Zero configuration changes needed
- No migration required

**Reliability:**
- No OAuth blocking risk
- Not affected by Anthropic policy changes
- Google's API has been stable

**Cost:**
- Free tier via AI Studio
- Pay-as-you-go pricing predictable
- No subscription lock-in

**Quality:**
- Gemini 2.5 Flash competitive with Sonnet 3.5
- Sufficient for investigation, debugging, feature implementation
- Fast inference, good for spawned agents

**Integration:**
- Perfect fit for headless CLI spawns
- Works with orch-go's session management
- No environment constraints

### Why Sonnet API Key as Fallback

**Availability:**
- Can use Anthropic API directly (no OAuth)
- Pay-per-token pricing
- No workaround maintenance

**Use cases:**
- Claude-specific features if needed
- Tasks requiring Claude's specific strengths
- Compatibility testing

**Trade-offs:**
- Higher cost than subscription (~5-20x)
- But predictable and stable
- Use sparingly for critical tasks only

## Consequences

### Positive

✅ **Stability**: No risk of authentication breaking mid-session
✅ **Maintenance**: Zero ongoing workaround updates
✅ **Reliability**: Gemini Flash works consistently
✅ **Cost**: Free tier sufficient for most use
✅ **Focus**: Engineering time on orch-go features, not bypasses

### Negative

❌ **No Claude Max**: Can't leverage $200/month subscription
❌ **Cost for Claude**: Sonnet API fallback is expensive if used heavily
❌ **Different models**: Gemini Flash has different strengths/weaknesses than Claude

### Neutral

⚪ **Model diversity**: Already using multiple models, this formalizes it
⚪ **Future flexibility**: Can re-evaluate if OpenCode ships official fix

## Implementation

### 1. Document Default Model

Update CLAUDE.md to state:
- **Default model**: Gemini Flash (`google/gemini-2.5-flash-preview`)
- **Fallback**: Sonnet API key (`anthropic/claude-sonnet-4-5`)
- **Deprecated**: Claude Max OAuth (blocked by Anthropic as of Jan 9, 2026)

### 2. Remove Claude Max OAuth References

Remove or deprecate in documentation:
- Account switching for Claude Max
- OAuth token refresh workflows
- Any references to "use Claude Max subscription"

### 3. Monitor OpenCode Upstream

Watch for:
- Official Anthropic OAuth support in OpenCode
- Stable plugin releases (not cat-and-mouse games)
- Community consensus on viable solutions

Re-evaluate this decision if:
- OpenCode ships official, stable Anthropic OAuth
- Anthropic changes policy to allow third-party tools
- Community stabilizes a workaround with >30 day lifespan

### 4. Cleanup Deprecated Code

Optional (low priority):
- Remove `pkg/account/` OAuth token refresh logic
- Simplify model selection (no OAuth path)
- Clean up account switching UI

Do NOT remove immediately - keep for potential future re-enablement.

## Alternatives Considered

### Alternative 1: Adopt Official Plugin (0.0.7)

**Why rejected:**
- Got re-blocked within 6 hours
- No evidence of long-term viability
- Requires ongoing monitoring for re-blocking

### Alternative 2: Tool Renaming (Source Edits)

**Why rejected:**
- High maintenance burden (edit on every OpenCode update)
- Conflicts with Dylan's OpenCode fork workflow
- Risk of hangs/zombie agents (Jan 8 precedent)

### Alternative 3: Rotating Suffix (fivetaku)

**Why rejected:**
- Requires maintaining forked OpenCode builds
- Highest maintenance burden of all options
- Drift from upstream accumulates over time
- Still a cat-and-mouse game

### Alternative 4: Kilo Code

**Why rejected:**
- VS Code extension, not headless CLI
- Can't spawn background agents
- Wrong environment for orch-go workflow
- Stable but incompatible

### Alternative 5: Wait for Official Fix

**Why rejected:**
- No timeline for official fix
- Blocking current work while waiting
- Gemini Flash already works now
- Can re-evaluate later if fix ships

## Monitoring & Review

**Immediate (next 7 days):**
- Verify Gemini Flash quality for typical orch-go tasks
- Document any Claude-specific use cases that need Sonnet API fallback
- Track cost if using Sonnet API fallback

**Monthly (next 3 months):**
- Check OpenCode upstream for Anthropic OAuth developments
- Review GitHub #7410 for community consensus on stable solutions
- Assess Gemini Flash quality vs cost trade-offs

**Quarterly:**
- Re-evaluate model strategy if:
  - OpenCode ships official Anthropic OAuth support
  - Community achieves >30 day workaround stability
  - Gemini Flash quality degrades or pricing changes
  - Better alternatives emerge (DeepSeek, etc.)

## References

- Investigation (Jan 8): `2026-01-08-inv-opus-auth-gate-fingerprinting.md`
- Investigation (Jan 9): `2026-01-09-inv-anthropic-oauth-community-workarounds.md`
- GitHub Issue #7410: https://github.com/anomalyco/opencode/issues/7410 (474+ comments)
- Hacker News: https://news.ycombinator.com/item?id=46549823
- opencode-anthropic-auth PR #10: https://github.com/anomalyco/opencode-anthropic-auth/pull/10
- fivetaku/opencode-oauth-fix: https://github.com/fivetaku/opencode-oauth-fix

## Decision History

- **2026-01-09**: Initial decision - abandon Claude Max OAuth, use Gemini Flash primary
- Future updates will be appended here

---

**Signed off by:** Dylan (via orchestrator synthesis)
**Next review:** 2026-02-09 (monthly checkpoint)
