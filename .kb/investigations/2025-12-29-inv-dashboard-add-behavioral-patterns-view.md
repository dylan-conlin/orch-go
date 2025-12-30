## Summary (D.E.K.N.)

**Delta:** Dashboard now displays behavioral patterns (repeated failures from action-log.json) in the Needs Attention section.

**Evidence:** Visual verification shows BEHAVIORAL section with purple color scheme, pattern counts, severity icons, and suppress action buttons.

**Knowledge:** The patterns API reads from pkg/patterns analyzer; only critical/warning patterns are shown in dashboard to avoid noise.

**Next:** Close - implementation complete and visually verified.

---

# Investigation: Dashboard Add Behavioral Patterns View

**Question:** How to add behavioral patterns from action-log.json to the dashboard?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: pkg/patterns analyzer already exists

**Evidence:** The patterns package in pkg/patterns/analyzer.go provides LoadLog(), DetectPatterns(), and Pattern struct.

**Source:** pkg/patterns/analyzer.go

**Significance:** Only needed to expose existing functionality via API endpoint.

---

### Finding 2: Dashboard follows established pattern for new data sources

**Evidence:** Other data sources (gaps, errors, beads) use pattern: API endpoint in serve.go -> Svelte store -> component integration.

**Source:** cmd/orch/serve.go, web/src/lib/stores/gaps.ts

**Significance:** Followed same pattern for patterns endpoint and store.

---

### Finding 3: NeedsAttention already has similar section for gaps

**Evidence:** The PATTERN section for recurring gaps uses blue color scheme with count badges and action buttons.

**Source:** web/src/lib/components/needs-attention/needs-attention.svelte

**Significance:** Could follow same visual pattern for BEHAVIORAL section with purple color scheme.

---

## Structured Uncertainty

**What's tested:**

- ✅ API endpoint returns patterns (verified: curl http://localhost:3348/api/patterns returns JSON)
- ✅ Dashboard renders BEHAVIORAL section (verified: screenshot with mock data)
- ✅ Go build succeeds (verified: go build ./cmd/orch/)

**What's untested:**

- ⚠️ Production action log data (tested with mock data only)
- ⚠️ Suppress button actually suppresses (button copies command, doesn't call API)

---

## References

**Files Modified:**
- cmd/orch/serve.go - Added /api/patterns endpoint and handler
- web/src/lib/stores/patterns.ts - New store for patterns data
- web/src/lib/components/needs-attention/needs-attention.svelte - Added BEHAVIORAL section
- web/src/routes/+page.svelte - Added patterns.fetch() to mount and refresh

**Commands Run:**
```bash
# Test API endpoint
curl -s http://localhost:3348/api/patterns

# Build verification
go build ./cmd/orch/
```

---

## Investigation History

**2025-12-29:** Investigation completed
- Status: Complete
- Key outcome: Behavioral patterns view implemented and visually verified
