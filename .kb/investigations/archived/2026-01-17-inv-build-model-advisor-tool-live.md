<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenRouter API provides comprehensive live model data (pricing, capabilities, context) for 300+ models; we can build model-advisor by combining this with our existing spawn tracking infrastructure.

**Evidence:** OpenRouter /api/v1/models endpoint tested successfully, returns detailed model metadata including pricing; existing events.jsonl tracks spawn lifecycle; model resolution infrastructure already exists in pkg/model.

**Knowledge:** Model selection currently relies on static pricing tables that go stale; combining live API data with local performance tracking enables dynamic "which model for this task at this budget?" recommendations; implementation can be incremental (start with pricing, add quality metrics later).

**Next:** Implement model-advisor tool with: 1) OpenRouter API client, 2) Extended ModelSpec with pricing, 3) CLI interface (`orch model recommend`), 4) Caching layer, 5) Enhanced spawn event tracking.

**Promote to Decision:** recommend-no (tactical feature implementation, not architectural decision requiring preservation)

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

# Investigation: Build Model Advisor Tool Live

**Question:** How can we build a tool that answers "which model for this task at this budget?" using live API data and our own spawn performance metrics?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** feature-impl agent
**Phase:** Complete
**Next Step:** Move to implementation phase
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: OpenRouter API provides comprehensive, live model data

**Evidence:** OpenRouter's `/api/v1/models` endpoint returns 300+ models with:
- Pricing data (prompt/completion costs per token)
- Context length and max completion tokens
- Architecture details (modality, parameters)
- Supported parameters
- Provider information

Example fields: `id`, `name`, `pricing`, `context_length`, `architecture`, `supported_parameters`, `top_provider`

**Source:** 
- `curl https://openrouter.ai/api/v1/models` (tested, returned 300+ models)
- OpenRouter homepage shows 300+ models across 60+ providers

**Significance:** This is our primary data source for live model comparison data. No API key required for model metadata, provides exactly the pricing/capability data we need.

---

### Finding 2: Existing events system tracks spawn lifecycles but not model performance

**Evidence:** Current events.jsonl tracks:
- `session.spawned` with prompt/title
- `session.completed` with timestamp
- `agent.completed` with workspace, skill, verification status
- Duration calculation in stats command (line 469-473 in stats_cmd.go)

Missing: Model used, quality assessment, task-specific metrics, latency per request

**Source:** 
- `pkg/events/logger.go` (lines 14-37: event types)
- `cmd/orch/stats_cmd.go` (lines 301-473: duration tracking)
- No model or quality tracking in Event struct

**Significance:** We have the infrastructure to track performance but need to extend Event struct to include model metadata and outcome quality.

---

### Finding 3: Static model selection guide exists but lacks dynamic pricing

**Evidence:** `.kb/guides/model-selection.md` contains:
- Model aliases (opus, sonnet, haiku, flash, etc.)
- Static pricing table (last updated: Jan 6, 2026)
- Usage recommendations by skill type
- Cost considerations section with hardcoded prices

Examples from guide:
- "Opus 4.5: $5.00/MTok input, $25.00/MTok output"
- "Sonnet 4.5: $3.00/MTok input (≤200K), $6.00/MTok (>200K)"

**Source:**
- `.kb/guides/model-selection.md` (lines 177-207: pricing tables)
- `pkg/model/model.go` (lines 24-60: static aliases)

**Significance:** Current guidance is static and goes stale. Users manually research pricing when limits/costs change. Model-advisor tool would replace manual updates with live data.

---

### Finding 4: Model resolution exists but is decoupled from live data

**Evidence:** `pkg/model/model.go` provides:
- `ModelSpec` struct with Provider and ModelID
- `Resolve()` function for alias → full model mapping
- Static `Aliases` map with 20+ models
- No connection to external APIs or pricing data

**Source:**
- `pkg/model/model.go` (lines 7-105)
- Resolution is purely local mapping

**Significance:** Model resolution is the right place to inject live data. Can extend ModelSpec to include pricing/latency/quality and populate from OpenRouter API.

---

## Synthesis

**Key Insights:**

1. **OpenRouter API is the right data source** - Provides live pricing, capabilities, and metadata for 300+ models without requiring authentication for model data. Eliminates need to maintain static pricing tables.

2. **Extend existing infrastructure, don't rebuild** - We already have model resolution (pkg/model), events logging (pkg/events), and duration tracking. Need to bridge these with live API data and add quality metrics.

3. **Two-layer architecture: Live API + Local tracking** - OpenRouter provides market data (pricing, capabilities). Our events.jsonl provides ground truth performance (actual latency, success rate, task fit). Combining both answers "which model for this task at this budget?"

**Answer to Investigation Question:**

Build a `model-advisor` tool that:
1. Fetches live model data from OpenRouter API (pricing, context, capabilities)
2. Extends spawn events to track model performance (latency, outcome quality, task type)
3. Provides CLI query: `orch model recommend --task <type> --budget <max-cost>` 
4. Returns ranked recommendations based on: API pricing + our historical performance data

Limitations: Initial version won't have quality scores (requires spawn outcome tracking). Can start with pricing + capabilities, add quality tracking incrementally.

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenRouter API is accessible and returns model data (verified: curl command returned 300+ models with pricing)
- ✅ Model data includes pricing, context length, architecture (verified: inspected JSON response structure)
- ✅ Events system tracks spawn/completion lifecycle (verified: read logger.go implementation)
- ✅ Duration tracking exists in stats command (verified: found calculation at stats_cmd.go:469-473)

**What's untested:**

- ⚠️ OpenRouter API rate limits (unknown, might hit limits with frequent queries)
- ⚠️ Model data format stability (schema could change without notice)
- ⚠️ Cache TTL of 24 hours is appropriate (might need more/less frequent updates)
- ⚠️ Recommendation algorithm effectiveness (haven't validated ranking logic against real decisions)
- ⚠️ Spawn outcome tracking is feasible (might require significant event schema changes)

**What would change this:**

- Finding would be wrong if OpenRouter API requires authentication for model metadata (tested: no auth needed)
- Finding would be wrong if model IDs in OpenRouter don't map to our aliases (partial: needs testing)
- Recommendation would fail if pricing alone isn't sufficient for model selection (quality scores needed)
- Implementation would fail if API response time >5s (not benchmarked, need to test)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Layered Model Advisor with OpenRouter API + Local Tracking** - Build a tool that combines live market data (OpenRouter) with our own performance metrics (events.jsonl) to recommend models for specific tasks and budgets.

**Why this approach:**
- OpenRouter provides comprehensive, live model data without authentication (Finding 1)
- Extends existing infrastructure (model resolution, events logging) rather than building from scratch (Finding 2)
- Combines market data (pricing, capabilities) with ground truth (our actual performance) for better recommendations (Finding 3, 4)
- Incremental implementation: Start with pricing/capabilities, add quality tracking later

**Trade-offs accepted:**
- Initial version won't have quality scores (requires spawn outcome tracking infrastructure)
- Relies on OpenRouter API availability (mitigate with caching)
- Doesn't cover all model providers (only those in OpenRouter's catalog)
- Why acceptable: Covers 300+ models across major providers, cacheable for offline use, quality tracking can be added incrementally

**Implementation sequence:**
1. **Add OpenRouter API client** (`pkg/advisor/openrouter.go`) - Foundational: provides live model data
2. **Extend model resolution** (`pkg/model/model.go`) - Add pricing/capabilities to ModelSpec, populate from API
3. **Build advisor CLI** (`cmd/orch/model.go`) - User-facing: `orch model recommend --task X --budget Y`
4. **Add spawn event tracking** (extend `pkg/events/logger.go`) - Track model used, task type for future quality analysis
5. **Caching layer** (`~/.orch/model-cache.json`) - Avoid API calls on every query, refresh daily

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- OpenRouter API client with model fetching and caching (foundational data layer)
- Extended ModelSpec struct with pricing, context, capabilities
- Basic CLI: `orch model list` to show models with live pricing
- Cache mechanism to avoid hitting API on every query

**Things to watch out for:**
- ⚠️ OpenRouter API rate limits (unknown, need to test) - mitigate with aggressive caching
- ⚠️ Model ID format differences between OpenRouter and our aliases (e.g., "anthropic/claude-opus-4-5-20251101" vs "opus")
- ⚠️ Pricing in different units (OpenRouter uses per-token strings like "0.00000175", we need float64)
- ⚠️ API response size (474KB for full model list) - cache aggressively, only fetch on first use or after TTL
- ⚠️ Offline mode: Tool should work without network (use cached data, degrade gracefully)

**Areas needing further investigation:**
- Artificial Analysis API availability (404 on /api endpoint, may require paid account)
- Quality metrics definition: What makes a model "good" for a task? (success rate, user satisfaction, completion time)
- Spawn outcome tracking: How to capture "this spawn succeeded/failed" in events (may need manual tagging)
- Tool-use accuracy measurement: How to programmatically assess if model used tools correctly

**Success criteria:**
- ✅ `orch model list` shows 50+ models with live pricing data
- ✅ `orch model recommend --task coding --budget 0.001` returns ranked recommendations
- ✅ Model data cached locally, tool works offline after first fetch
- ✅ Cache refreshes automatically after 24 hours (or on demand via `--refresh`)
- ✅ Integration: `orch spawn` logs model used in events.jsonl for future analysis

---

## References

**Files Examined:**
- `pkg/model/model.go` (lines 1-116) - Existing model resolution and alias system
- `pkg/events/logger.go` (lines 1-346) - Events logging infrastructure
- `cmd/orch/stats_cmd.go` (lines 301-473) - Duration tracking implementation
- `.kb/guides/model-selection.md` (lines 1-326) - Static model selection guide

**Commands Run:**
```bash
# Fetch OpenRouter model data
curl https://openrouter.ai/api/v1/models | head -100 | jq '.'

# Extract available fields from API response
jq '.data[0] | keys' /path/to/openrouter-response.json

# Search for existing performance tracking
grep -r "duration\|elapsed\|time_ms" --include="*.go"
```

**External Documentation:**
- OpenRouter API (https://openrouter.ai/api/v1/models) - Primary data source for live model metadata
- OpenRouter Homepage (https://openrouter.ai) - 300+ models, 60+ providers

**Related Artifacts:**
- **Guide:** `.kb/guides/model-selection.md` - Static guide to be complemented by live tool
- **Model:** `pkg/model/model.go` - Core model resolution to be extended with live data

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
