<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** API cost tracking should use local token counting first since Anthropic billing API returns "Not Found" error.

**Evidence:** Existing usage infrastructure displays Max subscription data; billing API tests fail; OpenCode sessions include token metadata for cost calculation.

**Knowledge:** Local token counting provides immediate visibility without external dependencies, following dashboard patterns for color-coded budget thresholds.

**Next:** Implement cost tracking with `/api/usage/cost` endpoint and dashboard widget using Sonnet 4.5 pricing.

**Promote to Decision:** recommend-no (implementation detail, follows prior investigation recommendation)

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

# Investigation: Add Api Cost Tracking Widget

**Question:** How to implement API cost tracking widget for Sonnet API usage in dashboard?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** Implement local token counting cost tracking
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Existing usage infrastructure tracks Claude Max subscription but not API costs

**Evidence:** Current implementation in `pkg/usage/usage.go` fetches Max subscription usage via OAuth from `/api/oauth/usage` endpoint. Dashboard displays 5-hour and weekly usage percentages with reset times. API route `/api/usage` returns `UsageAPIResponse` with percentage fields.

**Source:** `pkg/usage/usage.go:1-381`, `cmd/orch/serve_system.go:30-80`, `web/src/lib/stores/usage.ts:1-63`, `web/src/routes/+layout.svelte:61-97`

**Significance:** Infrastructure exists for displaying usage data in dashboard header. Need to extend with API cost tracking using Anthropic's `/v1/billing/cost` endpoint.

---

### Finding 2: Anthropic provides billing/cost API endpoint for pay-per-token usage

**Evidence:** Investigation `2026-01-12-inv-sonnet-cost-tracking-requirements.md` documents `/v1/billing/cost` endpoint that returns daily costs and total amount. Requires API key authentication (not OAuth). Response format includes date, amount, currency arrays.

**Source:** `.kb/investigations/2026-01-12-inv-sonnet-cost-tracking-requirements.md:50-73`

**Significance:** Official API exists for tracking Sonnet API spend. Need to implement client for this endpoint and integrate with existing usage infrastructure.

---

### Finding 3: Dashboard usage display is in layout header with color-coded percentages

**Evidence:** Layout component imports usage store, displays 5-hour and weekly percentages with color coding (green <60%, yellow 60-80%, red >80%). Shows account name/email and reset times.

**Source:** `web/src/routes/+layout.svelte:5-97` - imports usage store, defines `getUsageColor()` and `formatPercent()` helpers, displays usage in compact header

**Significance:** API cost tracking should follow similar pattern - display in dashboard with clear visual indicators. Could add separate widget or extend existing usage display.

---

### Finding 4: Anthropic billing API endpoint returns "Not Found" error

**Evidence:** Testing `/v1/billing/cost` endpoint with API key returns `{"type":"error","error":{"type":"not_found_error","message":"Not Found"}}`. Tried with date parameters, version header, and organization header - same result.

**Source:** Test commands:
```bash
curl -s -H "x-api-key: $ANTHROPIC_API_KEY" "https://api.anthropic.com/v1/billing/cost?start_date=2026-01-01&end_date=2026-01-18"
curl -s -H "x-api-key: $ANTHROPIC_API_KEY" -H "anthropic-version: 2023-06-01" "https://api.anthropic.com/v1/billing/cost"
```

**Significance:** Billing API endpoint may not be publicly available, require different authentication, or have changed. Local token counting approach should be implemented first as recommended in investigation.

---

## Synthesis

**Key Insights:**

1. **Existing infrastructure can be extended** - Dashboard already displays Max subscription usage with color-coded percentages and reset times. Similar pattern can be used for API cost tracking.

2. **Billing API access uncertain** - Anthropic `/v1/billing/cost` endpoint returns "Not Found", suggesting it may not be publicly available or require different permissions.

3. **Local token counting is viable alternative** - OpenCode sessions include token usage metadata that can be used to calculate costs locally using Sonnet 4.5 pricing ($3/1M input, $15/1M output).

**Answer to Investigation Question:**

API cost tracking widget should be implemented using local token counting first, following the existing dashboard usage display pattern. Create new API endpoint `/api/usage/cost` that calculates costs from agent session metadata and returns daily/weekly totals. Extend dashboard to display cost widget alongside existing usage display.

---

## Structured Uncertainty

**What's tested:**

- ✅ Anthropic billing API endpoint returns "Not Found" (verified: curl commands with API key)
- ✅ Existing usage infrastructure displays Max subscription data (verified: code examination)
- ✅ Dashboard layout includes usage display with color coding (verified: Svelte component analysis)

**What's untested:**

- ⚠️ Local token counting accuracy vs actual billing (hypothesis: should be within 5% based on investigation)
- ⚠️ OpenCode session metadata includes token counts for all requests (hypothesis: message.usage field exists)
- ⚠️ Cost calculation formula using Sonnet 4.5 pricing (hypothesis: $3/1M input, $15/1M output)

**What would change this:**

- If Anthropic billing API becomes accessible, implement hybrid approach (API + local counting)
- If token metadata missing from some sessions, need fallback estimation
- If pricing changes, need update mechanism for cost calculation

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Local Token Counting First** - Implement cost tracking using OpenCode session token metadata with Sonnet 4.5 pricing, following existing dashboard usage display patterns.

**Why this approach:**
- Provides immediate cost visibility without external API dependencies
- Follows established dashboard patterns (color coding, percentages, reset times)
- Enables per-spawn cost tracking for optimization insights
- Can be validated later if billing API becomes accessible

**Trade-offs accepted:**
- No ground truth validation against actual bill (deferred)
- Pricing updates require code changes (acceptable for now)
- Doesn't account for failed/retried requests (minor impact)

**Implementation sequence:**
1. **Cost calculation backend** - Add cost tracking to agent completion workflow, store in registry
2. **API endpoint** - Create `/api/usage/cost` endpoint returning daily/weekly totals
3. **Dashboard widget** - Extend layout to display API costs alongside Max usage

### Alternative Approaches Considered

**Option B: Wait for billing API access**
- **Pros:** Official data, no pricing maintenance
- **Cons:** Blocked indefinitely, no visibility now, daily granularity only
- **When to use instead:** If billing API becomes publicly accessible with same authentication

**Option C: External service integration**
- **Pros:** Could use third-party cost tracking services
- **Cons:** Additional dependencies, complexity, potential cost
- **When to use instead:** If local counting proves inaccurate and billing API unavailable

**Rationale for recommendation:** Local token counting provides immediate operational value with minimal dependencies, following the hybrid approach recommended in prior investigation. Can be enhanced later if billing API access becomes available.

---

### Implementation Details

**What to implement first:**
- **Cost calculation in completion workflow** - Modify `pkg/complete/verify.go` to calculate cost from token usage
- **Cost storage** - Add cost field to agent registry metadata
- **API endpoint** - Create `/api/usage/cost` in `cmd/orch/serve_system.go`
- **Dashboard store** - Add cost store in `web/src/lib/stores/cost.ts`
- **UI component** - Extend layout to display cost widget

**Things to watch out for:**
- ⚠️ Token metadata format - verify OpenCode message.usage field structure
- ⚠️ Cache token pricing - different rates for cache creation/read ($3.75/$0.30 per 1M)
- ⚠️ Floating point precision - use decimal types for currency calculations
- ⚠️ Date ranges - aggregate costs by day/week/month for dashboard display

**Areas needing further investigation:**
- OpenCode message.usage field availability across all message types
- Historical cost aggregation from existing session data
- Budget alert mechanism (warning at 80%, critical at 95%)
- Cost comparison with Max subscription ($200/mo threshold)

**Success criteria:**
- ✅ Dashboard displays current month Sonnet API spend
- ✅ Per-spawn cost visible in agent detail view  
- ✅ Costs update in real-time as agents complete
- ✅ Color coding for budget thresholds (green <$100, yellow $100-$180, red >$180)

---

## References

**Files Examined:**
- `pkg/usage/usage.go` - Existing Max subscription usage implementation
- `cmd/orch/serve_system.go` - API endpoint handlers including `/api/usage`
- `web/src/lib/stores/usage.ts` - Frontend usage store
- `web/src/routes/+layout.svelte` - Dashboard usage display component
- `.kb/investigations/2026-01-12-inv-sonnet-cost-tracking-requirements.md` - Prior investigation with requirements and API details

**Commands Run:**
```bash
# Test Anthropic billing API endpoint
curl -s -H "x-api-key: $ANTHROPIC_API_KEY" "https://api.anthropic.com/v1/billing/cost?start_date=2026-01-01&end_date=2026-01-18"

# Test with version header
curl -s -H "x-api-key: $ANTHROPIC_API_KEY" -H "anthropic-version: 2023-06-01" "https://api.anthropic.com/v1/billing/cost"
```

**External Documentation:**
- Anthropic API Documentation - `/v1/billing/cost` endpoint reference (from prior investigation)
- Anthropic Pricing - Sonnet 4.5 rates: $3/1M input, $15/1M output, $3.75/1M cache write, $0.30/1M cache read

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-12-inv-sonnet-cost-tracking-requirements.md` - Provides requirements and hybrid approach recommendation
- **Workspace:** Current feature implementation workspace

---

## Investigation History

**2026-01-18 12:00:** Investigation started
- Initial question: How to implement API cost tracking widget for Sonnet API usage in dashboard?
- Context: Task to add API cost tracking widget referencing prior investigation

**2026-01-18 12:15:** Key findings documented
- Examined existing usage infrastructure and dashboard display patterns
- Tested Anthropic billing API endpoint (returns "Not Found")
- Identified local token counting as viable first step

**2026-01-18 12:30:** Investigation completed
- Status: Complete
- Key outcome: Recommended local token counting implementation following existing dashboard patterns, with API endpoint `/api/usage/cost` and extended UI
