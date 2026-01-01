## Summary (D.E.K.N.)

**Delta:** The 10 API investigations reveal a maturing API layer for orch-go's dashboard, covering endpoints (/api/agents, /api/agentlog, /api/usage, /api/errors, /api/reflect, /api/pending-reviews), performance optimizations (19s→0.35s parallelization), and reliability patterns (HTTP timeouts, closed-issue filtering).

**Evidence:** All 10 investigations are Complete with high confidence (85-98%); 8 resulted in implemented features; 2 were investigative (ToS/proxy evaluation, local PDF processing).

**Knowledge:** The API layer should follow these patterns: (1) parallel batch operations over sequential calls, (2) HTTP timeouts for all external calls, (3) consistent filtering between CLI and API endpoints, (4) events.jsonl as the event source for dashboard widgets.

**Next:** Close - this synthesis consolidates 10 investigations. No supersession needed as investigations document independent features/fixes.

---

# Investigation: Synthesis of 10 API-Related Investigations (Dec 2025)

**Question:** What patterns, lessons, and potential consolidations emerge from the 10 API-related investigations in orch-go?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Synthesis Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Extracted-From:** N/A
**Supersedes:** N/A (synthesis document, not replacement)
**Superseded-By:** N/A

---

## Investigations Analyzed

| # | Date | Investigation | Category | Status | Key Outcome |
|---|------|---------------|----------|--------|-------------|
| 1 | 2025-12-20 | add-api-agentlog-endpoint | Endpoint | Complete | /api/agentlog with SSE + JSON modes |
| 2 | 2025-12-20 | poc-port-python-standalone-api | Infrastructure | Complete | Standalone spawning + TUI detection ported |
| 3 | 2025-12-24 | add-api-usage-endpoint | Endpoint | Complete | /api/usage exposing Claude Max limits |
| 4 | 2025-12-26 | add-api-errors-endpoint | Endpoint | Complete | /api/errors with pattern analysis |
| 5 | 2025-12-26 | api-endpoint-api-agents-hangs | Bug Fix | Complete | HTTP timeouts + redirect limits |
| 6 | 2025-12-26 | evaluate-building-api-proxy-layer | Policy | Complete | ToS violation - proxy rejected |
| 7 | 2025-12-26 | add-api-reflect-endpoint | Endpoint | Complete | /api/reflect for synthesis suggestions |
| 8 | 2025-12-27 | api-agents-endpoint-takes-19s | Performance | Complete | Parallelization: 19s→0.35s |
| 9 | 2025-12-30 | pending-reviews-api-count-doesn | Bug Fix | Complete | Filter closed issues in API |
| 10 | 2025-12-30 | pre-process-documents-locally-vs-api | Analysis | Complete | Local-first PDF triage pattern |

---

## Findings

### Finding 1: API Endpoints Follow Consistent Patterns

**Evidence:** Five investigations added new endpoints (/api/agentlog, /api/usage, /api/errors, /api/reflect, /api/pending-reviews). All follow the same pattern:
- CORS handler wrapping
- HTTP method validation
- JSON encoding response
- Error handling with appropriate status codes
- Events/data sourced from standard locations (events.jsonl, reflect-suggestions.json, beads)

**Source:** 
- 2025-12-20-inv-add-api-agentlog-endpoint-serve.md
- 2025-12-24-inv-add-api-usage-endpoint-serve.md
- 2025-12-26-inv-add-api-errors-endpoint-error.md
- 2025-12-26-add-api-reflect-endpoint-expose.md

**Significance:** The API layer has established conventions. New endpoints should follow these patterns. The dashboard has a growing set of data sources for widgets.

---

### Finding 2: Performance Requires Parallel Batch Operations

**Evidence:** Two investigations addressed performance:
1. `/api/agents` hung due to http.DefaultClient having no timeout (fixed with 10s default)
2. `/api/agents` took 19-26s due to O(N) sequential calls for 564 agents (fixed with parallelization → 0.35s)

Key patterns established:
- Use `sync.WaitGroup` + semaphore for bounded concurrency (max 20)
- Prefer List+filter over N individual Show calls
- HTTP clients MUST have timeouts (10s default)
- SSE streaming gets redirect limits but no timeout

**Source:**
- 2025-12-26-inv-api-endpoint-api-agents-hangs.md
- 2025-12-27-inv-api-agents-endpoint-takes-19s.md

**Significance:** As agent count grows (564 in production), O(N) operations become unacceptable. All new API code should default to parallel batch patterns.

---

### Finding 3: CLI-API Consistency Requires Explicit Attention

**Evidence:** The pending-reviews investigation (2025-12-30) found that `handlePendingReviews` API did not filter closed issues, while `getCompletionsForReview` CLI did. This caused stale entries in dashboard counts.

Pattern: Both paths need same filtering logic (filterClosedIssues).

**Source:** 2025-12-30-inv-pending-reviews-api-count-doesn.md

**Significance:** When CLI and API expose similar data, filtering logic MUST be shared or explicitly duplicated. Dashboard bugs often stem from this consistency gap.

---

### Finding 4: Events.jsonl is the Central Event Source

**Evidence:** Multiple endpoints read from events.jsonl:
- /api/agentlog - lifecycle events with SSE streaming
- /api/errors - error pattern analysis from session.error and agent.abandoned events

Event types used: session.started, session.error, agent.completed, agent.abandoned

**Source:**
- 2025-12-20-inv-add-api-agentlog-endpoint-serve.md
- 2025-12-26-inv-add-api-errors-endpoint-error.md

**Significance:** events.jsonl is the source of truth for agent lifecycle. New dashboard features should query events first before adding custom logging.

---

### Finding 5: ToS Constraints Block Certain API Patterns

**Evidence:** Investigation into API proxy layer for Claude Max account sharing found explicit ToS violation:
> "You may not share your Account login information, Anthropic API key, or Account credentials with anyone else."

This blocked a technically feasible feature on policy grounds.

**Source:** 2025-12-26-inv-evaluate-building-api-proxy-layer.md

**Significance:** ToS review should precede technical design for features involving third-party service sharing.

---

### Finding 6: Local-First Processing Reduces API Load

**Evidence:** PDF processing investigation found:
- Local pdftotext: 16ms for 4-page document
- API PDF processing: seconds + 1,500-3,000 tokens/page
- Token savings: 6-12x with local text extraction

Recommended triage pattern: pdfinfo → pdftotext → pdfseparate → send only relevant pages to API

**Source:** 2025-12-30-inv-pre-process-documents-locally-vs-api.md

**Significance:** For document-heavy workflows, local preprocessing reduces API costs and latency. This pattern applies beyond PDFs to any pre-processable content.

---

## Synthesis

**Key Insights:**

1. **API maturity trajectory** - In 10 days (Dec 20-30), orch-go gained 5 new dashboard endpoints, 2 critical performance fixes, and established patterns for endpoint development. The API layer is production-ready.

2. **Performance is a first-class concern** - Two investigations (25% of total) addressed performance. The 19s→0.35s improvement (50x) came from parallelization. All new API code should assume parallel-first design.

3. **Consistency between CLI and API is a trap** - The pending-reviews bug shows that similar functionality in CLI and API can diverge silently. Either share code or add tests that verify consistency.

4. **Events.jsonl as canonical store** - Multiple features derive from the same event stream. This is good architecture - new dashboard widgets should query events rather than add separate state.

**Answer to Investigation Question:**

The 10 API investigations consolidate into four categories:

| Category | Count | Key Pattern |
|----------|-------|-------------|
| New Endpoints | 5 | CORS + JSON + events.jsonl |
| Performance | 2 | Parallel + timeouts |
| Bug Fixes | 2 | CLI-API consistency |
| Analysis/Policy | 2 | ToS review, local-first |

**No supersession needed** - these investigations document independent features. The patterns extracted here should inform future API development:
1. All HTTP clients need timeouts
2. Batch operations should be parallel
3. CLI and API need consistent filtering
4. Events.jsonl is the canonical event source

---

## Structured Uncertainty

**What's tested:**

- All 10 investigations are marked Complete with evidence
- Performance claims verified (19s→0.35s in investigation #8)
- Endpoint patterns consistent (verified across 5 endpoint investigations)

**What's untested:**

- Whether all API endpoints have adequate test coverage
- Whether events.jsonl can scale to 10K+ agents
- Whether parallel batch patterns are applied consistently across all endpoints

**What would change this:**

- If new endpoints deviate from established patterns
- If performance issues recur in other endpoints (suggesting pattern not fully applied)
- If CLI-API consistency bugs recur (suggesting need for shared code)

---

## Implementation Recommendations

### Recommended Approach: Document Patterns in CLAUDE.md

**Pattern Library** - Add API development patterns to project CLAUDE.md for agent guidance.

**Why this approach:**
- Patterns are proven (10 investigations with working code)
- Agents need guidance for API development
- Prevents reinventing established patterns

**Trade-offs accepted:**
- CLAUDE.md grows larger
- Patterns may evolve (need maintenance)

**Implementation sequence:**
1. Add "API Development Patterns" section to CLAUDE.md
2. Document: CORS wrapping, JSON encoding, events.jsonl usage, parallel batch, HTTP timeouts
3. Reference specific files as examples (serve.go, check.go)

### Alternative: Create API Development Skill

**Option B: Dedicated skill for API development**
- **Pros:** More detailed guidance, checkpoints
- **Cons:** Overhead for simple endpoint additions
- **When to use:** If API development becomes frequent worker task

---

## References

**Files Examined:**
- 10 investigation files in .kb/investigations/
- cmd/orch/serve.go (API endpoint implementations)
- pkg/opencode/client.go (HTTP client patterns)
- pkg/verify/check.go (batch operations)

**Related Artifacts:**
- All 10 investigations listed in "Investigations Analyzed" table above

---

## Investigation History

**2026-01-01:** Investigation started
- Initial question: Synthesize patterns from 10 API investigations
- Context: kb topic accumulation triggered synthesis spawn

**2026-01-01:** Analysis completed
- Read all 10 investigations
- Identified 4 categories and 6 key findings
- Extracted patterns for future API development

**2026-01-01:** Investigation completed
- Status: Complete
- Key outcome: Patterns documented; no supersession needed; recommend adding patterns to CLAUDE.md
