# Session Synthesis

**Agent:** og-arch-orchestrator-coaching-plugin-28jan-ef7f
**Issue:** orch-go-20983
**Duration:** 2026-01-28 ~18:35 → 2026-01-28 ~19:05
**Outcome:** success

---

## TLDR

Analyzed the coaching plugin worker detection problem and confirmed: title-based detection IS working (verified this session has zero coaching metrics); the "keeps failing" perception comes from edge cases, not a broken core mechanism; the proper fix is upstream (OpenCode exposing session.metadata.role), not more heuristics.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-28-inv-orchestrator-coaching-plugin-cannot-reliably.md` - Comprehensive design investigation documenting the architectural constraint and recommended approach

### Files Modified
- None - this was an investigation/design session, not implementation

### Commits
- None yet - investigation file ready to commit

---

## Evidence (What Was Observed)

- This session (`ses_3fa15c255ffe48kGcy54lo8N56`) has zero entries in coaching-metrics.jsonl despite 30+ tool calls (verified: `grep` returned empty)
- Orchestrator sessions have coaching metrics (verified: `ses_3fa1f49daffeAUjMfvo6o0k9sm` has `action_ratio`, `analysis_paralysis`)
- session.created directory is always project root, not workspace (verified: event-test.jsonl examination)
- orch-go sends x-opencode-env-ORCH_WORKER header (verified: `pkg/opencode/client.go:559-561`)
- Prior investigation documented 13+ commits to coaching.ts since Jan 10, each addressing different edge cases
- Title-based detection (`hasBeadsId && !isOrchestratorTitle`) correctly identified this session as worker

### Tests Run
```bash
# Verify worker detection for this session
grep "ses_3fa15c255ffe48kGcy54lo8N56" ~/.orch/coaching-metrics.jsonl
# Result: no output (worker correctly excluded from coaching)

# Verify orchestrator sessions have metrics  
grep "ses_3fa1f49daffeAUjMfvo6o0k9sm" ~/.orch/coaching-metrics.jsonl
# Result: multiple entries (action_ratio, analysis_paralysis)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-28-inv-orchestrator-coaching-plugin-cannot-reliably.md` - Comprehensive design investigation with architectural analysis and recommendations

### Decisions Made
- **Accept title-based detection as "good enough"** because it IS working (verified with this session), and adding more heuristics violates Coherence Over Patches principle
- **Fix upstream, not in plugin** because orch-go already sends the header, OpenCode just doesn't expose it to plugins
- **Document rather than patch** because each new heuristic introduces new edge cases while the root cause is architectural

### Constraints Discovered
- **Plugin runs in server process** - Plugins cannot see agent environment variables because they run in OpenCode server, not spawned agent processes
- **session.created directory is project root** - Can never be workspace directory, invalidating directory-based detection
- **session.metadata.role unreliable** - OpenCode doesn't consistently expose metadata from custom headers

### Externalized via `kn`
- Not applicable - findings documented in investigation file for orchestrator review

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file written)
- [x] Tests passing (verification via grep showed detection working)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-20983`

**Recommended follow-up actions (for orchestrator, not this session):**
1. Create OpenCode issue for exposing session.metadata.role from custom headers
2. Add documentation block to coaching.ts explaining the architectural constraint
3. Update `.kb/guides/opencode-plugins.md` with worker detection section

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- What's involved in contributing to OpenCode? (timeline, process)
- What percentage of spawns are actually detected correctly? (measurement, not just verification)
- Should we add telemetry to track detection accuracy?

**Areas worth exploring further:**
- Whether coaching alerts actually improve orchestrator behavior (effectiveness measurement)
- OpenCode's design decisions around metadata exposure (why isn't it reliable?)

**What remains unclear:**
- Whether OpenCode intentionally excludes metadata or has a bug
- Edge case frequency (ad-hoc spawns, manual sessions)

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-5-20251101
**Workspace:** `.orch/workspace/og-arch-orchestrator-coaching-plugin-28jan-ef7f/`
**Investigation:** `.kb/investigations/2026-01-28-inv-orchestrator-coaching-plugin-cannot-reliably.md`
**Beads:** `bd show orch-go-20983`
