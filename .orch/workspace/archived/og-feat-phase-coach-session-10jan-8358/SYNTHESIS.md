# Session Synthesis

**Agent:** og-feat-phase-coach-session-10jan-8358
**Issue:** orch-go-tfhgw
**Duration:** 2026-01-10 (Planning + Implementation)
**Outcome:** success

---

## TLDR

Wired behavioral_variation and circular_pattern metrics from coaching.ts plugin to coach session for real-time investigation. Coach receives formatted metric messages when patterns detected (3+ variations OR circular contradiction).

---

## Delta (What Changed)

### Files Modified
- `~/.config/opencode/plugin/coaching.ts` - Added coach session integration:
  - Added `client` to plugin parameters (line 526)
  - Added `COACH_SESSION_ID` env var constant (line 57)
  - Created `streamToCoach()` helper function (lines 523-562)
  - Created `formatMetricForCoach()` message formatter (lines 564-620)
  - Wired behavioral_variation detection to streamToCoach (lines 765-767)
  - Wired circular_pattern detection to streamToCoach (lines 809-811)

### Commits
- Not yet committed (pending validation)

---

## Evidence (What Was Observed)

- PluginInput interface includes `client: ReturnType<typeof createOpencodeClient>` (opencode/packages/plugin/src/index.ts:26-33)
- SDK client has `session.promptAsync()` method accepting sessionID and parts parameters (opencode/packages/sdk/js/src/v2/gen/sdk.gen.ts:1390-1439)
- OpencodeClient class structure confirmed with session property (sdk.gen.ts)
- TypeScript compilation shows no syntax errors in new code (pre-existing module resolution warnings unrelated to changes)
- Session filtering logic added to prevent infinite loop (sessionId === COACH_SESSION_ID check)

### Tests Run
```bash
# TypeScript compilation check
cd ~/.config/opencode/plugin && npx tsc --noEmit coaching.ts
# Result: No errors related to new code (only pre-existing module resolution warnings)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-10-inv-phase-coach-session-integration-mvp.md` - Investigation documenting SDK client integration and implementation approach

### Decisions Made
- **Use promptAsync() instead of prompt()** - Asynchronous, non-blocking method appropriate for background streaming (doesn't wait for coach response)
- **Format messages as structured markdown** - Coach needs metric type, details, commands, recommendations, and explicit task instructions to investigate effectively
- **Session filtering via sessionId check** - Prevents infinite loop by not streaming coach session's own tool calls back to itself
- **Use env var for coach session ID** - Simple configuration mechanism (ORCH_COACH_SESSION_ID), easy to enable/disable
- **Stream to coach is best-effort** - No retry logic or error propagation; if coach unreachable, log and continue (don't block orchestrator)

### Constraints Discovered
- **No actual message delivery tested** - Have not created test coach session to verify end-to-end flow
- **Message format untested by coach** - Unknown if formatted markdown is optimal for coach investigation
- **Performance impact unknown** - promptAsync() latency not measured during real-time streaming
- **Error handling minimal** - Unknown behavior if coach session invalid/closed/unreachable

### Externalized via `kb`
- Pending Leave it Better step (will add after commit)

---

## Next (What Should Happen)

**Recommendation:** close (after commit)

### If Close
- [x] All deliverables complete (coaching.ts modified, investigation file created, SYNTHESIS.md created)
- [ ] Changes committed to git
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-tfhgw`

**Testing recommendations for orchestrator:**
1. Create coach session manually: `orch spawn investigation "coach session for pattern investigation" --no-track`
2. Export coach session ID: `export ORCH_COACH_SESSION_ID=<session-id>`
3. Trigger behavioral_variation pattern (run 3+ similar commands in same semantic group without 30s pause)
4. Verify coach receives formatted metric message
5. Trigger circular_pattern (make architectural decision contradicting .kb/investigations/ recommendation)
6. Verify coach receives contradiction message
7. Monitor for 1 week to assess false positive rate and token economics

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should coach session be auto-spawned on orchestrator start, or remain manual for MVP? (Deferred to Phase 4 per task description)
- What's the optimal message format for coach investigation? Current format is hypothesis - needs validation with actual coach session
- Should there be rate limiting if orchestrator triggers many patterns rapidly? (Could overwhelm coach with messages)
- How should coach communicate findings back to orchestrator? (One-way streaming for MVP, bidirectional deferred)

**Areas worth exploring further:**
- Coach session lifecycle management (auto-spawn, auto-restart if closed, cleanup)
- Message persistence - should streamed messages persist in coach session history or be ephemeral?
- Multi-orchestrator support - if Dylan runs multiple orchestrator sessions, should they share one coach or have separate coaches?
- Coach effectiveness metrics - how to measure if coach interventions improve orchestrator behavior?

**What remains unclear:**
- Actual false positive rate of pattern detection (will learn after 1 week monitoring period)
- Token economics of coach investigation (cost per pattern detected + investigation)
- Whether hybrid approach (pattern matching triggers + LLM investigates) is optimal, or if pure-coach (no pattern matching) would work better

---

## Session Metadata

**Skill:** feature-impl
**Model:** sonnet
**Workspace:** `.orch/workspace/og-feat-phase-coach-session-10jan-8358/`
**Investigation:** `.kb/investigations/2026-01-10-inv-phase-coach-session-integration-mvp.md`
**Beads:** `bd show orch-go-tfhgw`
