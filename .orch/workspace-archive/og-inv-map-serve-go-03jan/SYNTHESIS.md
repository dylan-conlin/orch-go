# Session Synthesis

**Agent:** og-inv-map-serve-go-03jan
**Issue:** (untracked)
**Duration:** 2026-01-03
**Outcome:** success

---

## TLDR

Investigated serve.go (2921 lines) to map API handler dependencies and recommend a split strategy. Identified 9 handler groupings with distinct pkg/ dependencies, recommended 4-phase split into 6-7 domain-based files (~500-800 lines each phase). Phase 1 starts with serve_agents.go (highest complexity).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-03-inv-map-serve-go-api-handler.md` - Full investigation with handler mappings and phase plan

### Files Modified
- None

### Commits
- Initial checkpoint with handler groupings
- Final commit with complete investigation

---

## Evidence (What Was Observed)

- serve.go is 2921 lines, not 4125 as initially stated (verified: `wc -l`)
- 9 natural handler groupings identified:
  1. **Agents/Sessions** (~440 lines): handleAgents, handleEvents, handleAgentlog*, workspace cache
  2. **Beads** (~230 lines): handleBeads, handleBeadsReady, handleIssues
  3. **Usage/Focus/Config** (~250 lines): handleUsage, handleFocus, handleConfig*
  4. **Servers/Daemon** (~180 lines): handleServers, handleDaemon
  5. **Gaps/Reflect** (~270 lines): handleGaps, handleReflect
  6. **Errors** (~170 lines): handleErrors, error pattern helpers
  7. **Pending Reviews** (~400 lines): handlePendingReviews, handleDismissReview
  8. **Changelog** (~40 lines): handleChangelog (already split)
  9. **Server Setup** (~560 lines): runServe, CORS, route registration

- External helpers used from shared.go, review.go, wait.go, status_cmd.go
- Test coverage in serve_test.go (1016 lines) well-organized by handler

### Tests Run
```bash
# Verified line counts
wc -l cmd/orch/serve.go  # 2921 lines

# Identified declarations
grep -n "^func\|^type" cmd/orch/serve.go | wc -l  # 73 declarations
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-03-inv-map-serve-go-api-handler.md` - Complete split strategy

### Decisions Made
- Domain-based file split (not monolithic, not pkg/ promotion) because it balances maintainability with minimal structural change
- Keep response types with handlers (not consolidated) for cohesion
- Keep workspace caching in serve_agents.go (not promoted to pkg/ yet)

### Constraints Discovered
- `beadsClient` global must remain in serve.go (initialized in runServe)
- `corsHandler` must remain in serve.go (used at route registration)
- Some response types reference each other (AgentAPIResponse → SynthesisResponse)

### Externalized via `kn`
- None needed (findings captured in investigation file)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Split serve.go Phase 1: Extract serve_agents.go

**Skill:** feature-impl

**Context:**
```
Extract handleAgents, handleEvents, handleAgentlog* and workspace cache (~600 lines) 
from serve.go to serve_agents.go. Move corresponding tests to serve_agents_test.go.
See .kb/investigations/2026-01-03-inv-map-serve-go-api-handler.md for full plan.
```

Additional follow-up phases:
- Phase 2: Extract serve_beads.go + serve_reviews.go (~700 lines)
- Phase 3: Extract serve_system.go (~400 lines)
- Phase 4: Extract serve_learn.go + serve_errors.go (~600 lines)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should workspace caching be promoted to pkg/workspace/ for reuse? (Currently only used by handleAgents)
- Should response types have JSON schema validation?
- Performance implications of multi-project workspace cache scanning

**Areas worth exploring further:**
- Import cycle testing after split (assumed clean, not verified)
- Test fixture sharing across handler test files

**What remains unclear:**
- Exact test file dependencies (may need shared test helpers file)

---

## Session Metadata

**Skill:** investigation
**Model:** Claude
**Workspace:** `.orch/workspace/og-inv-map-serve-go-03jan/`
**Investigation:** `.kb/investigations/2026-01-03-inv-map-serve-go-api-handler.md`
**Beads:** (untracked)
