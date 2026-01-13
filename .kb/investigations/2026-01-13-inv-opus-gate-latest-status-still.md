<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Opus 4.5 auth gate is still active and was reinforced on Jan 9, 2026; community workaround (opencode-anthropic-auth) no longer functional.

**Evidence:** GitHub issue #12 (30+ reactions) shows error "This credential is only authorized for use with Claude Code"; official docs show no announcements about lifting restrictions; two paths remain: direct API ($5/$25/MTok) or Claude Code CLI (Max subscription).

**Knowledge:** Anthropic is actively enforcing OAuth restrictions to official tools only; workaround attempts are not sustainable; dual-path architecture (opencode+sonnet default, claude+opus escape hatch) is correct approach.

**Next:** Update `.kb/` constraints to reflect Jan 2026 status; remove any opencode-anthropic-auth references from orch-go config; maintain dual spawn mode architecture.

**Promote to Decision:** recommend-no - This updates existing constraint knowledge; reinforces prior decision (dual spawn mode) rather than establishing new architectural choice.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Opus Gate Latest Status Still

**Question:** Is the Opus 4.5 auth gate still active as of January 2026? Are there any announcements about lifting it, known workarounds, or community discussion?

**Started:** 2026-01-13
**Updated:** 2026-01-13
**Owner:** Research Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Official Opus 4.5 API Access is Unrestricted

**Evidence:** 
- Anthropic announced Claude Opus 4.5 on November 24, 2025
- Model is available via API using `claude-opus-4-5-20251101`
- Pricing: $5 input / $25 output per million tokens
- API documentation shows standard authentication via API key (x-api-key header)
- For Max and Team Premium users, Opus-specific caps have been removed
- No mention of OAuth restrictions or auth gates in official documentation

**Source:** 
- https://www.anthropic.com/news/claude-opus-4-5 (announcement page)
- https://platform.claude.com/docs/en/api/overview (API documentation)

**Significance:** The official API access through Anthropic's direct API appears to have no restrictions for Opus 4.5. Users with API keys can access the model directly. This suggests any "auth gate" issues are specific to third-party integrations or OAuth flows, not the core API.

---

### Finding 2: Community Workaround (opencode-anthropic-auth) No Longer Working as of Jan 9, 2026

**Evidence:**
- GitHub repository `anomalyco/opencode-anthropic-auth` exists with 151 stars, last updated 4 days ago (Jan 9, 2026)
- Issue #12 opened Jan 9, 2026: "The auth no longer works"
- Error message reported: "This credential is only authorized for use with Claude Code and cannot be used for other API requests."
- 30+ people reacted to the issue, indicating widespread impact
- User reported: "It worked today but just stopped working :/"

**Source:**
- https://github.com/anomalyco/opencode-anthropic-auth/issues/12

**Significance:** Anthropic has updated their enforcement mechanisms to specifically restrict Claude Max OAuth credentials to only work with the official Claude Code tool. The community workaround that was previously mentioned in prior knowledge (opencode-anthropic-auth@0.0.7) appears to have been blocked by Anthropic's server-side changes. This is consistent with the "active fingerprinting and updating enforcement" mentioned in prior knowledge constraints. The timing (Jan 9, 2026) is very recent, suggesting this is an actively evolving situation.

---

### Finding 3: Current Access Methods and Pricing

**Evidence:**
- **Direct API Access:** Opus 4.5 available at $5 input/$25 output per million tokens via Anthropic API with API key
- **Claude Code official tool:** Requires Claude Pro ($17/month), Max ($100/month), Teams, or Enterprise subscription
- **Claude Console:** API access with pre-paid credits, can be used with Claude Code CLI
- **No official announcements** about lifting OAuth restrictions or changing auth gates
- Pricing page shows Claude Max now offers "Choose 5x or 20x more usage than Pro" with "Higher output limits for all tasks"

**Source:**
- https://code.claude.com/docs/en/quickstart
- https://claude.com/pricing

**Significance:** Anthropic has not announced any plans to lift the OAuth restrictions. The official paths are: (1) Pay-per-token via direct API ($5/$25 per MTok for Opus 4.5), or (2) Subscription-based via Claude Max ($100-200/month) using the official Claude Code CLI tool only. There are no announced workarounds or alternative access methods.

---

## Synthesis

**Key Insights:**

1. **The "Opus gate" is still active and was recently reinforced** - As of Jan 9, 2026, Anthropic strengthened enforcement to explicitly restrict Claude Max OAuth credentials to official Claude Code only. The community workaround (opencode-anthropic-auth) that was working stopped functioning with error message: "This credential is only authorized for use with Claude Code and cannot be used for other API requests."

2. **Two viable paths remain: Direct API or Official CLI** - Users can access Opus 4.5 either through (1) pay-per-token API with API key ($5/$25 per MTok), or (2) Claude Max subscription ($100-200/month) using only the official Claude Code CLI. Third-party integrations via OAuth are actively blocked.

3. **No announcements about lifting restrictions** - Anthropic's official documentation and news releases contain no mention of plans to lift OAuth restrictions or provide broader API access for Claude Max subscriptions. The enforcement appears to be intentional and ongoing.

**Answer to Investigation Question:**

**Is the Opus 4.5 auth gate still active?** YES - The auth gate is not only still active, but was recently reinforced on January 9, 2026.

**Any announcements about lifting it?** NO - There are no official announcements about lifting the OAuth restrictions. Anthropic's documentation continues to show the same two access paths (API key or official Claude Code CLI).

**Known workarounds?** NO - The community workaround (opencode-anthropic-auth plugin) that was previously functional stopped working on Jan 9, 2026 when Anthropic updated their enforcement. As documented in prior knowledge, Anthropic is "actively fingerprinting and updating enforcement" - this is a cat-and-mouse game not worth chasing.

**Community discussion?** YES - The GitHub issue #12 on the opencode-anthropic-auth repository has 30+ reactions, indicating widespread impact. The community appears to have accepted that the workaround path is no longer viable.

**Limitations:** This research is based on publicly available information as of Jan 13, 2026. Anthropic may have internal plans not yet announced publicly.

---

## Structured Uncertainty

**What's tested:**

- ✅ Opus 4.5 is available via direct API (verified: official docs show $5/$25 pricing and model ID `claude-opus-4-5-20251101`)
- ✅ opencode-anthropic-auth workaround stopped working Jan 9, 2026 (verified: GitHub issue #12 with error message and 30+ reactions)
- ✅ Claude Code requires subscription or Console account (verified: official quickstart documentation)
- ✅ No announcements about lifting restrictions (verified: checked Anthropic news page, API docs, pricing page - no mention)

**What's untested:**

- ⚠️ Whether Anthropic is working on OAuth API access behind the scenes (no public information, but possible)
- ⚠️ Whether future workarounds could emerge (community may continue trying, but prior knowledge suggests it's not sustainable)
- ⚠️ Whether the enforcement will remain this strict long-term (could change, but no signals suggesting it will)
- ⚠️ Exact technical mechanism of the new enforcement (appears to be server-side credential restriction, but not verified)

**What would change this:**

- Anthropic announcement of broader OAuth API access or removal of "Claude Code only" restriction
- New official API access method for Claude Max subscribers
- Discovery of stable workaround that Anthropic doesn't block (unlikely based on "active fingerprinting" pattern)
- Changes to Anthropic's business model around API access

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Accept the dual-path reality and abandon workaround attempts** - Use direct API for production systems and official Claude Code CLI for development workflows requiring Opus 4.5.

**Why this approach:**
- Anthropic is actively enforcing restrictions (Jan 9, 2026 update proves this)
- Workarounds create technical debt and unreliable systems
- Two stable paths exist that Anthropic officially supports
- Aligns with prior knowledge constraint: "cat and mouse game not worth chasing"

**Trade-offs accepted:**
- Higher costs compared to hypothetical workaround (but workarounds are now non-functional)
- Cannot use OpenCode/third-party tools with Opus via OAuth (must use official Claude Code CLI)
- Need to choose between pay-per-token API or subscription model

**Implementation sequence:**
1. **Audit current spawn configurations** - Check orch-go config for any opencode-anthropic-auth references, remove them
2. **Update constraints documentation** - Ensure `.kb/` constraints reflect Jan 2026 status (workaround no longer works)
3. **Implement dual spawn mode** - Use opencode+sonnet as default, claude+opus as escape hatch (already documented in prior decisions)

### Alternative Approaches Considered

**Option B: Wait for Anthropic to lift restrictions**
- **Pros:** Would restore opencode+opus capability if it happens
- **Cons:** No signals this will happen; indefinite wait; prior knowledge shows active enforcement trend
- **When to use instead:** Never - this is wishful thinking without evidence

**Option C: Pursue new workarounds**
- **Pros:** Could temporarily restore opencode+opus access
- **Cons:** Anthropic actively updating enforcement (Jan 9 proves this); high maintenance burden; unreliable
- **When to use instead:** Never - prior knowledge explicitly says "not worth chasing"

**Option D: Switch entirely to pay-per-token API**
- **Pros:** No auth gates, predictable access, works with any tool
- **Cons:** Higher costs at scale ($5/$25 per MTok vs flat subscription)
- **When to use instead:** If usage volume makes subscription uneconomical, or if third-party tool support is critical

**Rationale for recommendation:** The dual-path approach (opencode+sonnet default, claude+opus escape hatch) balances cost, reliability, and access to best reasoning quality when needed. Workaround attempts are no longer viable given active enforcement.

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- SPAWN_CONTEXT.md (lines 10-54) - Prior knowledge about Opus constraints and decisions

**Commands Run:**
```bash
# Searched for Reddit community discussions
curl -s "https://www.reddit.com/r/ClaudeAI/search.json?q=opus+oauth&sort=new&limit=5&t=month"

# No direct commands to test (research-only investigation)
```

**External Documentation:**
- https://www.anthropic.com/news/claude-opus-4-5 - Official Opus 4.5 announcement (Nov 24, 2025)
- https://platform.claude.com/docs/en/api/overview - API documentation showing authentication methods
- https://github.com/anomalyco/opencode-anthropic-auth/issues/12 - Community workaround failure report (Jan 9, 2026)
- https://code.claude.com/docs/en/quickstart - Official Claude Code documentation
- https://claude.com/pricing - Current pricing for subscriptions and API

**Related Artifacts:**
- **Decision:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-09-dual-spawn-mode-architecture.md` - Dual spawn mode (opencode+sonnet vs claude+opus)
- **Decision:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/decisions/2026-01-13-cancel-second-claude-max-subscription.md` - Recent decision about Max subscriptions
- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-08-inv-opus-auth-gate-fingerprinting.md` - Prior investigation of auth gate mechanisms

---

## Investigation History

**2026-01-13 ~15:30:** Investigation started
- Initial question: Is the Opus 4.5 auth gate still active, any announcements about lifting, workarounds, community discussion?
- Context: Spawned by orchestrator to check latest status given prior knowledge about restrictions and community workarounds

**2026-01-13 ~15:45:** Found critical evidence
- Discovered GitHub issue #12 showing opencode-anthropic-auth workaround stopped working Jan 9, 2026
- Error message: "This credential is only authorized for use with Claude Code"
- 30+ community reactions indicating widespread impact

**2026-01-13 ~16:00:** Investigation completed
- Status: Complete
- Key outcome: Auth gate is still active and was recently reinforced; no workarounds currently functional; two official paths remain (API or Claude Code CLI)
