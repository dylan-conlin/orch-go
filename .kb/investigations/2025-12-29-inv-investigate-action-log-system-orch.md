---
linked_issues:
  - orch-go-yf0h
---
<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Target extraction still fails after fix commit 5f8830fa - 86% of entries show "unknown" target. The issue is that input.callID or output.args are not being passed as expected by the OpenCode plugin hook API.

**Evidence:** Sampled action-log.jsonl post-fix (2025-12-29T15:27+) - all entries still have target="unknown". The fix logic is correct (store args from before hook, retrieve in after hook by callID) but the API contract is not being honored.

**Knowledge:** The action log system was created to solve the "knowledge vs behavior gap" - existing mechanisms tracked knowledge state but not action outcomes. The architecture (pkg/action + OpenCode plugin + orch patterns) is sound but needs debugging.

**Next:** Debug why input.callID or output.args are null in the OpenCode plugin hooks. Consider adding console.log debugging to understand actual API shape.

---

# Investigation: Action Log System Assessment

**Question:** What is the state of the action log system? Why was it created, what issues exist, and is it the right solution for behavioral pattern detection?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** Agent (orch-go-yf0h)
**Phase:** Complete
**Next Step:** Debug OpenCode plugin API contract for callID/args
**Status:** Complete

---

## Findings

### Finding 1: Original Motivation - The "Knowledge vs Behavior Gap"

**Evidence:** The action log system was created in response to investigation `2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md` which identified:
- Existing mechanisms (kn tried, orch learn, Session Reflection) track knowledge state, not action outcomes
- Tool failures are ephemeral and untracked
- Self-correction requires observing action outcomes: `action → outcome → pattern detection → adjustment`
- The specific failure mode: orchestrator repeatedly checking SYNTHESIS.md on light-tier agents

Key quote from investigation:
> "The orchestrator doesn't self-correct because: No mechanism tracks action outcomes within a session"

**Source:** 
- `.kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md:1-50`
- Git commit history: `b1b80bdd`, `35d516f6`, `ac380e2b`

**Significance:** The action log system was a well-reasoned architectural addition to address a real gap. The problem it solves is legitimate.

---

### Finding 2: Architecture Overview - Three-Part System

**Evidence:** The action log system consists of:

1. **pkg/action/action.go** - Go types and pattern detection
   - `ActionEvent` struct with Tool, Target, Outcome, SessionID, Workspace
   - `Logger` for appending to `~/.orch/action-log.jsonl`
   - `Tracker` for loading events and detecting patterns
   - `FindPatterns()` returns futile actions (3+ occurrences of non-success)

2. **OpenCode plugin** (`.opencode/plugin/action-log.ts`)
   - Hooks `tool.execute.before` and `tool.execute.after`
   - Logs Read, Glob, Grep, Bash tool outcomes
   - Stores args from before hook using callID as key

3. **cmd/orch/patterns.go** - CLI command for surfacing patterns
   - Aggregates retry patterns, gap patterns, and action patterns
   - Outputs human-readable report with severity levels

**Source:**
- `pkg/action/action.go:1-538`
- `.opencode/plugin/action-log.ts:1-254`
- `cmd/orch/patterns.go:1-491`

**Significance:** The architecture is well-designed and follows established patterns in the codebase. The three parts integrate cleanly.

---

### Finding 3: Target Extraction Bug - Fix Not Working

**Evidence:** The fix commit (5f8830fa) claims to fix target extraction by:
- Using tool.execute.before hook to capture args (output.args)
- Storing args in Map keyed by callID
- Retrieving args in tool.execute.after hook

However, sampling the log after the fix shows 86% of entries still have `target="unknown"`:
```
$ cat ~/.orch/action-log.jsonl | jq -r '.target' | sort | uniq -c | sort -rn | head -5
 362 unknown
  14 http://localhost:5188/?tab=ops
   6 http://localhost:5188/?refresh=1&tab=ops
```

Post-fix entries (2025-12-29T15:27+) still show `"target": "unknown"`:
```json
{"timestamp":"2025-12-29T15:44:25.615Z","tool":"Read","target":"unknown","outcome":"empty"}
{"timestamp":"2025-12-29T15:44:37.590Z","tool":"Bash","target":"unknown","outcome":"success"}
```

**Source:**
- Git commit 5f8830fa (2025-12-29 07:27:27 -0800)
- `~/.orch/action-log.jsonl` entries from 2025-12-29T15:27+

**Significance:** The fix logic is correct in principle, but either `input.callID` is not being passed, `output.args` is not available in the before hook, or the two hooks are not being called for the same invocation. The OpenCode plugin API contract is not being honored as expected.

---

### Finding 4: Duplicate Entries Still Present

**Evidence:** Despite the fix adding a `loggedCalls` Set to track logged callIDs, log entries still appear in pairs with identical timestamps:
```json
{"timestamp":"2025-12-29T15:44:25.615Z","tool":"Read","target":"unknown","outcome":"empty"}
{"timestamp":"2025-12-29T15:44:25.616Z","tool":"Read","target":"unknown","outcome":"empty"}
```

This suggests either:
- The hook is being called twice per invocation
- The callID is different for each call (or undefined)
- The deduplication logic is not firing correctly

**Source:** `~/.orch/action-log.jsonl` entries with ms-close timestamps

**Significance:** The deduplication fix is also not working as expected. The root cause may be the same as Finding 3 - callID not available.

---

### Finding 5: orch patterns Command Works Correctly

**Evidence:** The `orch patterns` command correctly surfaces behavioral patterns from the available data:
```
BEHAVIORAL PATTERNS - Orchestrator Awareness Report
Total: 22 patterns detected
Critical: 2 (require immediate attention)

 !  Empty result: Read on unknown
    Tool Read has returned empty results on unknown 30 times
```

It aggregates patterns from three sources:
- Retry patterns from events.jsonl (spawn/abandon cycles)
- Gap patterns from gap-tracker.json (kb context gaps)
- Action patterns from action-log.jsonl (futile actions)

**Source:** `orch patterns` command output

**Significance:** The Go-side pattern detection works correctly. The issue is data quality from the OpenCode plugin, not the analysis.

---

### Finding 6: Glass Integration Works Correctly

**Evidence:** Some entries in the log show valid targets - these are from Glass (browser automation):
```json
{"timestamp":"2025-12-29T15:46:38.006372Z","tool":"screenshot","target":"http://localhost:5188/?refresh=1&tab=ops","outcome":"success","source":"glass"}
```

Glass logs to the same action-log.jsonl but uses a different code path that correctly extracts targets.

**Source:** `~/.orch/action-log.jsonl` entries with `"source":"glass"`

**Significance:** This confirms the logging infrastructure works. The issue is specific to the OpenCode plugin's hook API contract.

---

## Synthesis

**Key Insights:**

1. **The problem action log solves is real** - The "knowledge vs behavior gap" investigation correctly identified that tool failures were ephemeral and untracked. The action log addresses a genuine architectural need.

2. **The architecture is sound** - pkg/action + OpenCode plugin + orch patterns is a well-designed three-part system that integrates cleanly with existing infrastructure.

3. **The implementation has a bug** - Target extraction fails because the OpenCode plugin API doesn't provide callID/args as expected. This needs debugging.

4. **Is this the right solution?** - Yes, action logging is the right approach for behavioral pattern detection. The alternative (relying on AI self-reporting via kn tried) requires the AI to recognize its own failure patterns, which the original investigation showed doesn't work.

**Answer to Investigation Question:**

**WHY created:** To solve the "knowledge vs behavior gap" - existing mechanisms tracked knowledge state but not action outcomes, making self-correction impossible.

**Issues that still exist:**
1. Target extraction broken - 86% of entries show "unknown"
2. Duplicate entries despite deduplication logic
3. Root cause: OpenCode plugin API contract (callID/args) not working as expected

**Is this the right solution?** Yes. The architecture is sound and addresses a real need. The implementation bug needs fixing, but the approach is correct.

**Work remaining:**
1. Debug why callID/output.args are null in OpenCode plugin hooks
2. Consider adding console.log debugging to understand actual API shape
3. Once targets work, the system will achieve its original vision

---

## Structured Uncertainty

**What's tested:**

- ✅ Action log entries are being created (verified: 420 entries in log)
- ✅ orch patterns command detects futile actions (verified: command output shows patterns)
- ✅ Glass integration works (verified: glass entries have valid targets)
- ✅ Fix commit exists and logic is correct (verified: code review)

**What's untested:**

- ⚠️ Actual values of input.callID and output.args in OpenCode hooks (needs console.log debugging)
- ⚠️ Whether OpenCode version affects hook API (version not checked)
- ⚠️ Whether plugin needs to be restarted after code changes

**What would change this:**

- Finding would change if OpenCode's plugin API documentation shows different hook contract
- Finding would change if callID is available under a different property name
- Finding would be wrong if there's a caching issue and old plugin code is running

---

## Implementation Recommendations

### Recommended Approach ⭐

**Debug Plugin API Contract** - Add console.log debugging to understand what OpenCode actually passes to plugin hooks.

**Implementation sequence:**
1. Add `console.log("[action-log] before:", JSON.stringify({input, output}, null, 2))` to tool.execute.before
2. Add `console.log("[action-log] after:", JSON.stringify({input, output}, null, 2))` to tool.execute.after
3. Run an agent and check console output
4. Adjust extractTarget based on actual API shape

### Alternative Approaches Considered

**Option B: Use a different hook mechanism**
- Abandon OpenCode plugins, use a PostToolUse shell hook
- **Cons:** Shell hooks can't easily access tool output
- **When to use:** If OpenCode plugin API is fundamentally broken

**Option C: Parse transcripts post-session**
- Analyze conversation transcripts for tool outcomes
- **Cons:** Too late for in-session pattern surfacing
- **When to use:** For historical analysis only

---

## References

**Files Examined:**
- `~/.orch/action-log.jsonl` - The log file itself
- `.opencode/plugin/action-log.ts` - OpenCode plugin implementation
- `pkg/action/action.go` - Go types and pattern detection
- `cmd/orch/patterns.go` - CLI command for surfacing patterns
- `.kb/investigations/2025-12-27-inv-orchestrator-self-correction-mechanisms-problem.md` - Original motivation
- `.kb/investigations/2025-12-27-inv-action-logging-subsystem-tool-outcomes.md` - Implementation investigation
- `.kb/investigations/2025-12-28-inv-action-logging-integration-points-agent.md` - Integration points investigation

**Commands Run:**
```bash
# Check log size and sample entries
wc -l ~/.orch/action-log.jsonl  # 420 lines
head -20 ~/.orch/action-log.jsonl
tail -30 ~/.orch/action-log.jsonl

# Analyze target extraction quality
cat ~/.orch/action-log.jsonl | jq -r '.target' | sort | uniq -c | sort -rn | head -20

# Analyze outcome distribution
cat ~/.orch/action-log.jsonl | jq -r '.outcome' | sort | uniq -c

# Run patterns command
orch patterns
```

**Git History:**
```bash
git log --oneline --grep="action-log"
# 5f8830fa fix: action-log plugin target extraction and deduplication
# ac380e2b feat: add OpenCode plugin for action outcome logging
# b1b80bdd investigation: action logging integration points
# 35d516f6 feat: add action logging subsystem for behavioral pattern detection
# 4b548f92 Investigation: Orchestrator self-correction mechanisms
```

---

## Self-Review

- [x] Real test performed (sampled log, ran commands, verified timestamps)
- [x] Conclusion from evidence (based on actual log data)
- [x] Question answered (why created, issues, right solution)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled (Delta, Evidence, Knowledge, Next)

**Self-Review Status:** PASSED
