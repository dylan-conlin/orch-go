# Session Synthesis

**Agent:** og-debug-detect-agents-exhausting-03jan
**Issue:** orch-go-71pa
**Duration:** 2026-01-03 → 2026-01-03
**Outcome:** success

---

## TLDR

Implemented context exhaustion detection for `orch status` that combines token usage thresholds (150K warning, 180K critical) with git uncommitted work detection to surface at-risk agents. When an agent has high token usage AND uncommitted changes, a RISK column appears with warnings like "⚠️ AT-RISK" or "🚨 CRITICAL".

---

## Delta (What Changed)

### Files Created
- `pkg/verify/context_risk.go` - Context exhaustion risk assessment logic with `AssessContextRisk()` and `HasUncommittedWork()` functions
- `pkg/verify/context_risk_test.go` - Comprehensive tests for risk assessment and uncommitted work detection

### Files Modified
- `cmd/orch/status_cmd.go` - Added `ContextRisk` and `ProjectDir` fields to `AgentInfo`, integrated risk assessment after token fetch, updated display to show RISK column when agents are at-risk

### Commits
- (pending) - Add context exhaustion detection to orch status

---

## Evidence (What Was Observed)

- Token stats already fetched in status_cmd.go:387-394 via `GetSessionTokens()`
- Git status pattern exists in handoff.go:545-556 that can be reused
- Project directory extraction available via `extractProjectDirFromWorkspace()`
- Claude's context limit is ~200K tokens, so 150K (75%) and 180K (90%) are reasonable thresholds

### Tests Run
```bash
# Context risk tests
go test ./pkg/verify/... -run TestContext -v
# PASS: TestContextExhaustionRisk_FormatMethods (0.00s)

# Full test suite
go test ./...
# PASS: all packages pass including new tests
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-03-inv-detect-agents-exhausting-context-uncommitted.md` - Design rationale and implementation details

### Decisions Made
- **Token thresholds:** 150K for warning, 180K for critical - based on ~200K context limit
- **Risk levels:** Warning only triggers with uncommitted work at <180K tokens (high usage alone isn't risky if no uncommitted work)
- **Display approach:** RISK column only appears when there are at-risk agents (no noise when everything is fine)

### Constraints Discovered
- Git status requires the actual project directory, not the orchestrator's directory (for cross-project agents)
- Token stats may be nil if session has no messages yet

### Externalized via `kn`
- N/A (constraints documented in investigation file)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Tests passing
- [x] Investigation file updated
- [x] SYNTHESIS.md created
- [ ] Ready for `orch complete orch-go-71pa`

### Future Enhancement (Not Blocking)
**Issue:** Add context exhaustion alerting to orch monitor
**Skill:** feature-impl
**Context:**
```
Use the ShouldAlert() method from ContextExhaustionRisk to trigger desktop notifications
via orch monitor when agents hit critical thresholds. This requires SSE integration.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should `orch monitor` also show real-time alerts for context exhaustion?
- Should there be automatic intervention (e.g., `orch send` to prompt agent to commit)?

**Areas worth exploring further:**
- Fine-tuning token thresholds based on real-world usage patterns
- Caching git status results if performance becomes an issue with many agents

**What remains unclear:**
- Optimal token thresholds may need adjustment based on actual agent behavior

---

## Session Metadata

**Skill:** systematic-debugging
**Model:** claude-opus
**Workspace:** `.orch/workspace/og-debug-detect-agents-exhausting-03jan/`
**Investigation:** `.kb/investigations/2026-01-03-inv-detect-agents-exhausting-context-uncommitted.md`
**Beads:** `bd show orch-go-71pa`
