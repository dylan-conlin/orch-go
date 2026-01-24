## Summary (D.E.K.N.)

**Delta:** Implemented Synthesis Card display in Swarm Dashboard showing D.E.K.N. summary for completed agents.

**Evidence:** API returns synthesis data, frontend renders it, tests pass (go test + svelte-check).

**Knowledge:** verify.ParseSynthesis already parses SYNTHESIS.md; reused in serve.go for API response.

**Next:** Close - feature complete, all deliverables committed.

**Confidence:** High (90%) - tested with real completed agents that have SYNTHESIS.md files.

---

# Investigation: Implement Synthesis Card Display in Swarm Dashboard

**Question:** How to display condensed D.E.K.N. synthesis info for completed agents in the Swarm Dashboard?

**Started:** 2025-12-20
**Updated:** 2025-12-20
**Owner:** og-feat-implement-synthesis-card-20dec
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (90%)

---

## Findings

### Finding 1: Existing synthesis parsing in verify package

**Evidence:** `pkg/verify/check.go` already has `Synthesis` struct and `ParseSynthesis()` function that parses SYNTHESIS.md files.

**Source:** `pkg/verify/check.go:116-178`

**Significance:** No need to duplicate parsing logic - can reuse in serve.go for API response.

---

### Finding 2: Frontend already has agent data structure

**Evidence:** `web/src/lib/stores/agents.ts` has Agent interface that matches registry.Agent from Go. Just needed to add synthesis field.

**Source:** `web/src/lib/stores/agents.ts:6-21`

**Significance:** Minimal changes needed - add Synthesis type and field to existing Agent interface.

---

### Finding 3: Many completed agents already have SYNTHESIS.md

**Evidence:** 24 workspaces in `.orch/workspace/*/SYNTHESIS.md` have synthesis files ready to display.

**Source:** `ls .orch/workspace/*/SYNTHESIS.md | wc -l` returned 24

**Significance:** Feature immediately useful with existing data.

---

## Synthesis

**Key Insights:**

1. **Reuse existing parsing** - verify.ParseSynthesis handles all SYNTHESIS.md formats
2. **API-first approach** - serve.go parses synthesis and includes in JSON response
3. **Condensed display** - SynthesisCard shows TLDR, outcome, delta summary, and top 2 next actions

**Answer to Investigation Question:**

Extended /api/agents endpoint to include parsed synthesis data for completed agents. Created SynthesisCard Svelte component that renders condensed D.E.K.N. summary. Component shows TLDR, outcome badge, recommendation, delta summary, and first 2 next actions.

---

## Confidence Assessment

**Current Confidence:** High (90%)

**Why this level?**
- Tested with real completed agents via curl to /api/agents
- Frontend type-checks with svelte-check
- Go tests pass including new JSON serialization test

**What's certain:**
- API returns synthesis data correctly
- Frontend compiles without errors
- Existing SYNTHESIS.md files parse correctly

**What's uncertain:**
- Visual appearance may need polish (not tested in browser)
- Edge cases with malformed SYNTHESIS.md not tested

---

## References

**Files Modified:**
- `cmd/orch/serve.go` - Added AgentWithSynthesis type and synthesis parsing in handleAgents
- `cmd/orch/serve_test.go` - Added test for JSON serialization
- `web/src/lib/stores/agents.ts` - Added Synthesis interface
- `web/src/routes/+page.svelte` - Integrated SynthesisCard
- `web/src/lib/components/synthesis-card/` - New component

**Commands Run:**
```bash
# Verify Go build
go build ./cmd/orch/...

# Run tests
go test ./cmd/orch/...

# Verify frontend
bun run check

# Test API
curl -s http://127.0.0.1:3334/api/agents | jq 'map(select(.synthesis != null)) | .[0:2]'
```

---

## Investigation History

**2025-12-20:** Investigation completed
- Final confidence: High (90%)
- Status: Complete
- Key outcome: Synthesis Card display implemented with D.E.K.N. summary for completed agents
