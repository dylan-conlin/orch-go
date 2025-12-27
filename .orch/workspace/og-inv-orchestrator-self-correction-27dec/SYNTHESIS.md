# Session Synthesis

**Agent:** og-inv-orchestrator-self-correction-27dec
**Issue:** orch-go-eq8k
**Duration:** 2025-12-27 16:00 → 2025-12-27 17:00
**Outcome:** success

---

## TLDR

Investigated why orchestrator AI doesn't self-correct when repeatedly making the same mistake. Found that existing mechanisms (`kn tried`, `orch learn`, Session Reflection) track knowledge gaps, not behavioral patterns. Tool failures are ephemeral and untracked, making self-correction impossible. Recommended implementing action outcome logging with pattern detection.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md` - Complete investigation with findings and recommendations

### Files Modified
- None (investigation-only session)

### Commits
- Pending (investigation file to be committed)

---

## Evidence (What Was Observed)

- `learning.go:26-55`: GapEvent struct tracks KB context gaps (knowledge), not action outcomes (behavior)
- `attempts.go:29-36`: IsRetryPattern() only detects spawn/abandon patterns at agent lifecycle level
- `post-tool-use.sh`: Hook only triggers on orch commands, doesn't log tool outcomes
- `check.go:488-503`: Tier system is correctly implemented in verification code
- Session Reflection (SKILL.md:1316-1346): Manual checkpoint at session end, too late and dependent on AI self-awareness

### Tests Run
```bash
# Searched for pattern detection mechanisms
grep "pattern.*detect|detect.*pattern" pkg/ --include=*.go
# Found: attempts.go retry patterns, but scoped to spawn lifecycle only

# Reviewed hooks
ls ~/.claude/hooks/
# Found: post-tool-use.sh, session-start.sh, but no action outcome logging

# Checked kn capabilities
kn --help
# Found: decide, tried, constrain, question - but requires explicit AI call
```

---

## Knowledge (What Was Learned)

### Key Insight: Knowledge vs Behavior Gap

The failure mode isn't a knowledge gap (orchestrator knows about tiers) - it's a behavioral pattern (orchestrator doesn't consistently apply knowledge). Current mechanisms assume "if we know, we'll do" but this isn't true.

### Decisions Made
- Action outcome logging is the right approach (matches existing events.jsonl pattern)
- PostToolUse hook is the right insertion point (already exists, can be extended)
- Pattern surfacing should be periodic, not real-time (avoid noise)

### Constraints Discovered
- Tool results are ephemeral - no persistence of action outcomes
- Claude Code hooks can inject context but can't modify AI behavior directly
- Pattern detection requires threshold (3+ occurrences) to avoid false positives

### Externalized via `kn`
- (To be done on completion - constrain about ephemeral tool results)

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement action outcome logging for behavioral pattern detection
**Skill:** feature-impl
**Context:**
```
Investigation found that orchestrator self-correction is impossible because tool failures are 
ephemeral. Implement:
1. ActionEvent struct + action-log.jsonl persistence
2. PostToolUse hook extension for logging
3. orch patterns command for surfacing repeated futile actions

See: .kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Can PostToolUse hooks access full tool output including errors? (would need API testing)
- Should action logging automatically trigger `kn tried` entries? (needs design session)
- How to distinguish "expected empty" from "unexpected empty" tool results?

**Areas worth exploring further:**
- LLM-based pattern analysis at session end (more sophisticated, higher cost)
- Real-time pattern detection with immediate feedback (complex, may be noisy)
- Integration with Claude Code's internal retry mechanisms (if any exist)

**What remains unclear:**
- Whether pattern surfacing alone is sufficient or if automatic correction is needed
- What the performance impact of comprehensive action logging would be
- Whether this pattern generalizes beyond orchestrator to worker agents

---

## Session Metadata

**Skill:** investigation
**Model:** claude-sonnet-4-20250514 (via opencode)
**Workspace:** `.orch/workspace/og-inv-orchestrator-self-correction-27dec/`
**Investigation:** `.kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md`
**Beads:** `bd show orch-go-eq8k`
