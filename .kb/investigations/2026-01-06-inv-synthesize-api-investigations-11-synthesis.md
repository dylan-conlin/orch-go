## Summary (D.E.K.N.)

**Delta:** Synthesized 11 API investigations into a comprehensive guide (api-development.md) covering patterns, performance, and best practices.

**Evidence:** Read all 11 investigations, identified 5 major patterns (handler structure, CORS, N+1 elimination, HTTP timeouts, SSE streaming), created `.kb/guides/api-development.md` with consolidated guidance.

**Knowledge:** The investigations revealed recurring themes: N+1 queries causing timeouts (agents 26s→0.35s, pending-reviews timeout→10ms), HTTP clients need explicit timeouts, SSE requires special handling, and domain-based file splits keep serve.go maintainable.

**Next:** Close - guide created and committed.

---

# Investigation: Synthesize API Investigations (11 Total)

**Question:** What patterns and best practices emerge from 11 API-related investigations, and how should they be consolidated into an authoritative guide?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Feature-impl agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: N+1 Query Pattern is the Primary Performance Issue

**Evidence:** Two investigations documented the same root cause:
- `/api/agents`: 26 seconds → 0.35 seconds after parallelization (75x improvement)
- `/api/pending-reviews`: Timeout → 10ms after batch fetching

Both issues stemmed from O(N) sequential API calls during iteration.

**Source:** 
- `2025-12-27-inv-api-agents-endpoint-takes-19s.md`
- `2026-01-05-inv-pending-reviews-api-times-out.md`

**Significance:** This is the #1 performance pattern to avoid. The fix is consistent: batch operations (List + filter) instead of individual calls, parallel fetching with semaphore (max 20 concurrent).

---

### Finding 2: HTTP Client Timeouts Prevent Hangs

**Evidence:** The `/api/agents` hang investigation found that `http.DefaultClient` has no timeout, causing indefinite hangs when OpenCode entered a redirect loop.

Fix: Configure HTTP clients with 10-second timeout and max 10 redirects. Exception: SSE connections need redirect limits but NO timeout (long-running streams).

**Source:** `2025-12-26-inv-api-endpoint-api-agents-hangs.md`

**Significance:** All external HTTP calls need explicit timeouts. This is a foundational pattern that prevents cascading failures.

---

### Finding 3: SSE Streaming Has Distinct Patterns

**Evidence:** The agentlog endpoint investigation established the SSE pattern: file polling at 500ms intervals, flusher interface check, proper headers (Content-Type: text/event-stream, Cache-Control: no-cache).

Trade-off: Up to 500ms latency for new events, but simple and reliable without file watch dependencies.

**Source:** `2025-12-20-inv-add-api-agentlog-endpoint-serve.md`

**Significance:** SSE endpoints are fundamentally different from JSON endpoints. They need special handling for long-running connections.

---

### Finding 4: Endpoint Addition Follows Consistent Pattern

**Evidence:** Multiple investigations (agentlog, usage, errors, changelog, reflect) all followed the same pattern:
1. Define JSON-tagged response struct
2. Implement handler with method check → query params → business logic → JSON encode
3. Register with corsHandler wrapper
4. Add tests (method validation, JSON format, error handling)

**Source:** All endpoint addition investigations

**Significance:** This is the "golden path" for adding new endpoints. Following it ensures consistency and testability.

---

### Finding 5: File Organization Prevents God Objects

**Evidence:** The serve.go mapping investigation found 2921 lines with 9 handler groups. Recommended split into 6-7 domain-based files (agents, beads, system, learn, errors, reviews).

Phase approach: 500-800 lines per extraction to avoid context exhaustion.

**Source:** `2026-01-03-inv-map-serve-go-api-handler.md`

**Significance:** Proactive file splitting prevents maintenance burden. Domain-based organization aligns with pkg/ dependency boundaries.

---

## Synthesis

**Key Insights:**

1. **Performance is about avoiding sequential calls** - Both major performance fixes (agents, pending-reviews) came from eliminating N+1 patterns. The pattern is: collect IDs first, batch fetch in parallel, then process.

2. **Defensive HTTP clients prevent cascading failures** - Timeouts and redirect limits are non-negotiable for external calls. The exception (SSE) proves the rule.

3. **Consistency accelerates development** - All 11 investigations followed similar patterns for endpoint implementation. The guide codifies these patterns so future work is faster.

4. **One ToS investigation revealed a non-technical constraint** - The proxy investigation (2025-12-26) found that account sharing violates Anthropic ToS. Not all API questions are technical.

**Answer to Investigation Question:**

The 11 investigations revealed 5 major patterns for API development:

1. **Handler structure** - Method check → params → logic → JSON response
2. **CORS middleware** - All handlers wrapped with corsHandler
3. **N+1 elimination** - Batch → parallelize → cache
4. **HTTP timeouts** - 10s default, redirect limits, except SSE
5. **Domain-based files** - Split by domain at 500-800 lines per phase

These patterns are now consolidated in `.kb/guides/api-development.md` as the authoritative reference for orch-go API work.

---

## Structured Uncertainty

**What's tested:**

- ✅ All 11 investigations read and analyzed
- ✅ Guide created at `.kb/guides/api-development.md`
- ✅ Patterns extracted match documented fixes in investigations

**What's untested:**

- ⚠️ Guide accuracy for edge cases not covered in original investigations
- ⚠️ Whether new developers can follow guide without friction
- ⚠️ Completeness of pattern coverage (other patterns may exist)

**What would change this:**

- New performance issues with different root causes would need guide updates
- Significant API architecture changes would obsolete some patterns
- User feedback on guide usability

---

## Implementation Recommendations

**Purpose:** The deliverable IS the guide. No further implementation needed.

### Recommended Approach ⭐

**Use `.kb/guides/api-development.md` as authoritative reference** - Read before adding endpoints, follow established patterns.

**Why this approach:**
- Consolidates 11 investigations into single reference
- Prevents rediscovery of known patterns
- Provides copy-paste templates for common operations

**Trade-offs accepted:**
- Guide may need updates as patterns evolve
- Not all edge cases covered

---

## References

**Files Examined:**

- `2025-12-20-inv-add-api-agentlog-endpoint-serve.md` - SSE file polling
- `2025-12-20-inv-poc-port-python-standalone-api.md` - TUI ready detection
- `2025-12-24-inv-add-api-usage-endpoint-serve.md` - Reusing pkg/ functions
- `2025-12-26-inv-add-api-errors-endpoint-error.md` - Error pattern analysis
- `2025-12-26-inv-api-endpoint-api-agents-hangs.md` - HTTP timeouts
- `2025-12-26-inv-evaluate-building-api-proxy-layer.md` - ToS constraints
- `2025-12-27-inv-api-agents-endpoint-takes-19s.md` - N+1 elimination
- `2026-01-03-inv-add-api-changelog-endpoint-orch.md` - Extracting shared logic
- `2026-01-03-inv-map-serve-go-api-handler.md` - File split strategy
- `2026-01-05-inv-pending-reviews-api-times-out.md` - Batch fetching
- `simple/2025-12-26-add-api-reflect-endpoint-expose.md` - Simple JSON endpoints

**Related Artifacts:**
- **Guide:** `.kb/guides/api-development.md` - Created by this synthesis

---

## Investigation History

**2026-01-06 09:00:** Investigation started
- Initial question: Synthesize 11 API investigations into guide
- Context: kb reflect identified 10+ investigations on API topic

**2026-01-06 09:15:** All investigations read and analyzed
- Identified 5 major patterns across investigations
- Found consistent themes (N+1, timeouts, SSE, structure)

**2026-01-06 09:30:** Investigation completed
- Status: Complete
- Key outcome: Created `.kb/guides/api-development.md` with consolidated patterns
