<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Coaching plugin pattern detection is partially verified - 2 of 7 patterns work in production (action_ratio, analysis_paralysis with 11 metrics each), while 5 patterns (behavioral_variation, circular_pattern, Dylan signals) have sound logic but no production examples to confirm they trigger correctly.

**Evidence:** Metrics file has 23 entries showing action_ratio (0.06-0.69) and analysis_paralysis (5-10 consecutive tools) correctly detected; no entries for behavioral_variation, circular_pattern, dylan_signal_prefix, priority_uncertainty, or compensation_pattern; code inspection verified detection logic exists and appears correct for all patterns.

**Knowledge:** Plugin infrastructure works (JSONL writing, tool hooks, session tracking), but complex patterns remain unverified because they either haven't occurred naturally OR detection has environment-specific issues (wrong directory for investigation files, timing thresholds too strict, worker filtering too aggressive).

**Next:** Add unit tests for untested patterns (semantic classification, behavioral variation, circular pattern detection) to distinguish "pattern hasn't occurred" from "detection is broken"; consider manual orchestrator testing session deliberately exhibiting each pattern after tests pass.

**Promote to Decision:** recommend-no - This is verification work, not an architectural decision.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Test Coaching Plugin Pattern Detection

**Question:** Do the coaching plugin patterns (behavioral variation, circular patterns, Dylan signals) trigger correctly when appropriate behavioral patterns occur?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Agent og-feat-orch-go-investigation-16jan-1772
**Phase:** Complete
**Next Step:** None - investigation findings documented
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Coaching Plugin Metrics Are Being Collected

**Evidence:** 
- Plugin file exists at `/Users/dylanconlin/Documents/personal/orch-go/plugins/coaching.ts` (40,670 bytes, last modified Jan 11)
- Metrics file `~/.orch/coaching-metrics.jsonl` exists with 23 entries
- Last metrics captured: Jan 11, 2026
- Metric types found: `action_ratio` (11 entries), `analysis_paralysis` (11 entries)

**Source:** 
- `ls -la /Users/dylanconlin/Documents/personal/orch-go/plugins/coaching.ts`
- `cat ~/.orch/coaching-metrics.jsonl | jq -r '.metric_type' | sort | uniq -c`
- `tail -20 ~/.orch/coaching-metrics.jsonl`

**Significance:** The plugin is functional and has been collecting metrics. The action_ratio and analysis_paralysis detection mechanisms are working, as evidenced by actual metric entries.

---

### Finding 2: Only Two Pattern Types Have Been Detected in Production

**Evidence:**
- Found metrics: `action_ratio` and `analysis_paralysis`
- NOT found: `behavioral_variation`, `circular_pattern`, `dylan_signal_prefix`, `priority_uncertainty`, `compensation_pattern`
- Sample analysis_paralysis metric shows 10 consecutive bash commands detected
- Sample action_ratio shows low ratios (0.06 = 1 action / 16 reads) triggering warnings

**Source:**
- `grep "behavioral_variation" ~/.orch/coaching-metrics.jsonl` (no results)
- `grep "circular_pattern" ~/.orch/coaching-metrics.jsonl` (no results)
- Metrics file analysis showing only 2 distinct metric types out of 7 defined in coaching.ts

**Significance:** Either these patterns haven't occurred naturally, or there may be issues with the detection logic for behavioral_variation, circular patterns, and Dylan signal patterns. Requires targeted testing to determine if detection works.

---

### Finding 3: Plugin Location Differs From Design Document

**Evidence:**
- Design doc specifies: "Deploy coaching.ts to `~/.config/opencode/plugin/`"
- Actual location: `/Users/dylanconlin/Documents/personal/orch-go/plugins/coaching.ts`
- No file at `~/.config/opencode/plugin/coaching.ts`
- Despite this, plugin IS collecting metrics (metrics file has recent data from Jan 11)

**Source:**
- docs/designs/2026-01-10-orchestrator-coaching-plugin.md:222
- `ls -la ~/.config/opencode/plugin/coaching.ts` (file not found)
- `ls -la /Users/dylanconlin/Documents/personal/orch-go/plugins/coaching.ts` (file exists)

**Significance:** OpenCode must support loading plugins from project directories (not just ~/.config/opencode/plugin/). The design document may be outdated, or both locations are supported. The plugin is working despite not being in the documented location.

---

### Finding 4: Behavioral Variation Detection Logic Verified

**Evidence:**
- Detection logic at coaching.ts:1192-1238
- Triggers when: 3+ consecutive bash commands in same semantic group WITHOUT 30s pause
- Example sequence that WOULD trigger: "overmind start" → "overmind status" → "overmind restart" (all process_mgmt)
- Strategic pause detection: 30s+ without ANY tool calls resets the counter (coaching.ts:1168-1179)
- Counter resets when switching to different semantic group (coaching.ts:1230-1238)

**Source:**
- coaching.ts:176-188 (classifyBashCommand function)
- coaching.ts:56 (VARIATION_THRESHOLD = 3)
- coaching.ts:1192-1238 (variation detection logic)
- coaching.ts:124-171 (SEMANTIC_PATTERNS array with 8 groups)

**Significance:** The logic is sound - it correctly detects when orchestrator is "thrashing" (trying multiple variations of same operation). However, NO behavioral_variation metrics exist in production data, suggesting orchestrators haven't actually exhibited this pattern, OR they naturally pause between attempts (30s+ thinking resets counter).

---

### Finding 5: Circular Pattern Detection Requires Investigation Files with Recommendations

**Evidence:**
- Circular detection logic at coaching.ts:1241-1273
- Requires: Investigation files with D.E.K.N. Summary → **Next:** field containing architectural keywords
- Keywords tracked: launchd, overmind, tmux, systemd, docker, kubernetes, procfile, plist, daemon, supervisor (coaching.ts:237-250)
- Found 20+ investigation files with **Next:** recommendations in .kb/investigations/
- Detection triggers when: git commit message, .plist file edit, Procfile edit, or bd create contains DIFFERENT keyword than investigation recommendation in same domain

**Source:**
- coaching.ts:210-226 (parseDEKNSummary function)
- coaching.ts:312-345 (detectArchitecturalDecision function)
- coaching.ts:351-386 (findContradiction function)
- `grep -r "^\*\*Next:\*\*" /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/` (found 20 results)

**Significance:** This detection is sophisticated - it parses investigation markdown files, extracts architectural recommendations, and compares against current decisions. However, NO circular_pattern metrics in production suggests: (1) orchestrators are following investigation recommendations, (2) investigations don't use tracked keywords (launchd/overmind/etc), or (3) the plugin loads recommendations from wrong directory.

---

### Finding 6: Dylan Signal Detection Works Only in Orchestrator Sessions

**Evidence:**
- Plugin detects Dylan patterns via `experimental.chat.messages.transform` hook (coaching.ts:948-1091)
- Three Dylan patterns tracked:
  1. **Signal Prefixes**: frame-collapse:, compensation:, focus:, step-back: (coaching.ts:779-796)
  2. **Priority Uncertainty**: "what's next?", "what should we focus on", etc. (coaching.ts:802-814)
  3. **Compensation Pattern**: 30%+ keyword overlap between messages (coaching.ts:856-869)
- Worker sessions are explicitly filtered out (coaching.ts:961-977) - checks for .orch/workspace/ or SPAWN_CONTEXT.md mentions

**Source:**
- coaching.ts:948-1091 (experimental.chat.messages.transform hook)
- coaching.ts:779-814 (signal prefix and priority uncertainty detection)
- coaching.ts:856-869 (compensation pattern detection)
- coaching.ts:961-977 (worker session filtering)

**Significance:** These patterns can ONLY be detected in orchestrator sessions where Dylan is actively chatting. Worker agents reading spawn context are filtered out. The absence of these metrics in production suggests: (1) Dylan hasn't used these signal prefixes, (2) Dylan hasn't exhibited priority uncertainty (2+ "what's next?" questions), or (3) Messages don't have 30%+ keyword overlap.

---

### Finding 7: Action Ratio and Analysis Paralysis Detection Works in Production

**Evidence:**
- 11 action_ratio metrics captured (Jan 11, 2026)
- 11 analysis_paralysis metrics captured (Jan 11, 2026)
- Lowest action_ratio: 0.06 (1 action / 16 reads) - correctly identified as low
- Highest analysis_paralysis: 10 consecutive bash commands - correctly detected
- Metrics show progression: action_ratio improved from 0.06 → 0.69 during session
- Analysis paralysis sequences: 10 → 8 → 6 → 5 (improvement over time)

**Source:**
- `tail -20 ~/.orch/coaching-metrics.jsonl`
- coaching.ts:490-563 (flushMetrics function with action_ratio and analysis_paralysis calculation)
- coaching.ts:459-484 (detectSequence function for tool repetition)

**Significance:** These two patterns ARE working correctly in production and capturing real behavioral data. The plugin successfully detected low action ratios (option theater signal) and tool repetition sequences (analysis paralysis signal). This proves the plugin infrastructure works - the question is why other patterns haven't triggered.

---

## Synthesis

**Key Insights:**

1. **Plugin Infrastructure is Functional** - Evidence from 23 production metrics shows the plugin is loaded, running, and correctly detecting patterns. Both action_ratio and analysis_paralysis patterns triggered multiple times with accurate measurements, proving the core detection and JSONL writing mechanisms work.

2. **Advanced Patterns Haven't Triggered Naturally** - Despite functional infrastructure, 5 out of 7 pattern types have never been detected: behavioral_variation (thrashing), circular_pattern (contradicting investigations), dylan_signal_prefix, priority_uncertainty, and compensation_pattern. This suggests either (a) these patterns genuinely haven't occurred, (b) detection thresholds are too strict, or (c) there are environment/configuration issues preventing detection.

3. **Detection Logic is Sophisticated but Untested in Practice** - Code inspection reveals well-designed detection for all patterns (semantic grouping for behavioral variation, D.E.K.N. parsing for circular patterns, message analysis for Dylan signals), but without production examples, we can't verify the logic works under real conditions. The gap between "code looks correct" and "code works in practice" remains.

**Answer to Investigation Question:**

**Partial Answer:** The coaching plugin patterns work for simple tool-based metrics (action_ratio, analysis_paralysis) but remain unverified for complex patterns (behavioral_variation, circular_pattern, Dylan signals). Based on code inspection:

- ✅ **Verified Working:** action_ratio, analysis_paralysis (11 production metrics each)
- ⚠️ **Logic Verified, Untested:** behavioral_variation (semantic classification looks correct)
- ⚠️ **Logic Verified, Untested:** circular_pattern (D.E.K.N. parsing looks correct, investigation files exist with keywords)
- ⚠️ **Untestable by Worker:** Dylan signals (require orchestrator session with Dylan's actual messages)

**Limitations:** As a worker agent, I cannot trigger orchestrator-specific patterns or Dylan message patterns. A complete verification would require: (1) orchestrator session deliberately exhibiting each pattern, (2) manual inspection of metrics after each test, (3) verification that metrics correctly identify the pattern.

---

## Structured Uncertainty

**What's tested:**

- ✅ Plugin file exists and is dated Jan 11, 2026 (verified: `ls -la plugins/coaching.ts`)
- ✅ Metrics file exists with 23 entries (verified: `wc -l ~/.orch/coaching-metrics.jsonl`)
- ✅ action_ratio and analysis_paralysis metrics are being captured (verified: parsed JSONL, found 11 of each type)
- ✅ Investigation files with **Next:** recommendations exist (verified: `grep "^\*\*Next:\*\*" found 20+ files`)
- ✅ Investigation files contain tracked keywords (verified: found overmind, launchd in Next fields)
- ✅ Semantic classification patterns defined for 8 groups (verified: read coaching.ts:124-171)
- ✅ Detection logic for all 7 patterns exists in code (verified: code inspection coaching.ts:176-1091)

**What's untested:**

- ⚠️ behavioral_variation detection triggers correctly for 3+ same-group commands (logic verified, no production example)
- ⚠️ circular_pattern detection correctly parses D.E.K.N. and finds contradictions (logic verified, no production example)
- ⚠️ Dylan signal prefix detection works with actual user messages (cannot test as worker agent)
- ⚠️ Priority uncertainty detection accumulates correctly across messages (cannot test as worker agent)
- ⚠️ Compensation pattern keyword overlap calculation is accurate (cannot test as worker agent)
- ⚠️ Strategic pause (30s) correctly resets behavioral variation counter (timing-dependent, not tested)
- ⚠️ Plugin loads investigation files from correct directory in production (assumed based on code, not verified)

**What would change this:**

- behavioral_variation: Run orchestrator session that executes 3+ process_mgmt commands without 30s pause, check metrics file for behavioral_variation entry
- circular_pattern: Run orchestrator session that creates launchd plist when investigation recommends overmind, check for circular_pattern metric
- Dylan signals: Dylan uses "frame-collapse:" prefix in orchestrator session, check for dylan_signal_prefix metric
- Plugin loading: Add debug logging to plugin initialization showing loaded investigation count, verify in OpenCode server logs

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add Test Suite for Untested Pattern Detection** - Create unit tests that simulate pattern conditions and verify metrics are generated correctly.

**Why this approach:**
- Code inspection can verify logic structure but not runtime behavior (Finding 3)
- action_ratio and analysis_paralysis prove the infrastructure works, so testing other patterns is low-risk (Finding 7)
- Without tests, we can't distinguish "pattern hasn't occurred" from "detection is broken" (Finding 2)
- Tests provide regression protection as patterns evolve

**Trade-offs accepted:**
- Tests require mocking OpenCode plugin hooks (client, directory, sessionID) - moderate complexity
- Tests can't fully replicate production conditions (timing, actual investigation files)
- Test maintenance burden when pattern detection logic changes

**Implementation sequence:**
1. **Test semantic classification** - Unit test `classifyBashCommand` with all semantic groups (foundational, no mocking needed)
2. **Test behavioral variation** - Mock 3 consecutive same-group bash commands, verify metric emitted
3. **Test circular pattern** - Create mock investigation file with overmind recommendation, mock git commit with launchd, verify contradiction detected
4. **Document testing gaps** - Add comment explaining Dylan signal patterns can't be unit tested (require real messages)

### Alternative Approaches Considered

**Option B: Manual Orchestrator Testing**
- **Pros:** Tests real production conditions, no mocking needed, discovers integration issues
- **Cons:** Manual, time-consuming, not repeatable, requires orchestrator session setup for each test
- **When to use instead:** After unit tests pass, run one manual verification session to confirm end-to-end flow

**Option C: Add Debug Logging and Wait for Natural Occurrence**
- **Pros:** Zero development effort, captures actual production patterns
- **Cons:** May wait indefinitely, can't verify patterns work BEFORE they're needed, no debugging capability if patterns fail to trigger
- **When to use instead:** If patterns are low-priority and there's no urgency to verify

**Rationale for recommendation:** Unit tests provide highest confidence-to-effort ratio. They're automatable, repeatable, and catch bugs before production. Given that infrastructure already works (Finding 7), testing the untested patterns is straightforward.

---

### Implementation Details

**What to implement first:**
- **Semantic classification test** - Zero dependencies, validates foundational logic, 30 min effort
- **Create test fixtures** - Sample investigation markdown with D.E.K.N. summary for circular pattern tests
- **Mock OpenCode plugin context** - Reusable test helper for session state simulation

**Things to watch out for:**
- ⚠️ Strategic pause timing (30s) is hard to test without flaky sleep() calls - consider parameterizing timeout for tests
- ⚠️ Plugin receives `directory` parameter from OpenCode - tests must provide valid path to .kb/investigations/
- ⚠️ Worker session detection relies on tool args containing .orch/workspace/ - tests should verify filtering works
- ⚠️ Circular pattern detection requires investigation files with specific D.E.K.N. format - test fixtures must match production format exactly

**Areas needing further investigation:**
- **Why no behavioral_variation in production?** - Is 30s pause too short (orchestrators naturally think longer)? Should threshold be 4-5 instead of 3?
- **Investigation file loading** - Does plugin load from correct directory in production? Add logging to verify on next server start
- **Coaching message injection** - action_ratio and analysis_paralysis now inject messages directly into session (coaching.ts:546-559) - verify this works without disrupting workflow

**Success criteria:**
- ✅ All 7 pattern types have passing unit tests
- ✅ Semantic classification correctly maps all documented command patterns (process_mgmt, git, test, build, knowledge, orch, file_ops, network, other)
- ✅ Behavioral variation triggers after exactly 3 consecutive same-group commands (not 2, not 4)
- ✅ Circular pattern correctly parses D.E.K.N. summary and extracts keywords from Next field
- ✅ Tests are deterministic (no timing flakes, no dependency on filesystem state)

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/plugins/coaching.ts` - Full plugin source (1299 lines) to understand all 7 detection patterns
- `/Users/dylanconlin/Documents/personal/orch-go/docs/designs/2026-01-10-orchestrator-coaching-plugin.md` - Original design document for expected behavior
- `~/.orch/coaching-metrics.jsonl` - Production metrics file showing what's been captured
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/*.md` - Investigation files to verify circular pattern detection could work

**Commands Run:**
```bash
# Verify plugin file exists and check modification date
ls -la /Users/dylanconlin/Documents/personal/orch-go/plugins/coaching.ts
ls -la ~/.config/opencode/plugin/coaching.ts

# Check metrics file and analyze contents
ls -la ~/.orch/coaching-metrics.jsonl
cat ~/.orch/coaching-metrics.jsonl | wc -l
tail -20 ~/.orch/coaching-metrics.jsonl
cat ~/.orch/coaching-metrics.jsonl | jq -r '.metric_type' | sort | uniq -c

# Search for specific pattern types
grep "behavioral_variation" ~/.orch/coaching-metrics.jsonl
grep "circular_pattern" ~/.orch/coaching-metrics.jsonl

# Find investigations with recommendations
grep -r "^\*\*Next:\*\*" /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/ | head -20
grep -l "overmind\|launchd\|tmux\|systemd" /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/*.md | head -10
ls /Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/*.md | wc -l

# Check for test files
find /Users/dylanconlin/Documents/personal/orch-go -name "*.test.ts" -o -name "*.spec.ts" | grep -i coach
```

**External Documentation:**
- OpenCode Plugin API - Used by coaching.ts for tool.execute.after and experimental.chat.messages.transform hooks
- JSONL format - Append-only log format for metrics storage

**Related Artifacts:**
- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-10-inv-trigger-coaching-patterns-test.md` - Previous incomplete attempt at this testing (stopped at Test 1)
- **Investigation:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-10-inv-orchestrator-coaching-plugin-technical-design.md` - Technical design decisions for plugin implementation
- **Design:** `/Users/dylanconlin/Documents/personal/orch-go/docs/designs/2026-01-10-orchestrator-coaching-plugin.md` - Original design proposal

---

## Investigation History

**2026-01-16 14:16:** Investigation started
- Initial question: Do the coaching plugin patterns trigger correctly when appropriate behavioral patterns occur?
- Context: Spawned from beads orch-go-32qum, continuing incomplete investigation from 2026-01-10 that stopped at "Test 1"

**2026-01-16 14:17-14:45:** Evidence gathering phase
- Verified plugin exists and is dated Jan 11 (40,670 bytes)
- Found metrics file with 23 entries (11 action_ratio, 11 analysis_paralysis)
- Discovered 5 pattern types have never triggered in production
- Located 830 investigation files with 20+ containing **Next:** recommendations

**2026-01-16 14:45-15:00:** Code analysis phase
- Inspected all 7 detection pattern implementations
- Verified semantic classification logic for 8 command groups
- Confirmed circular pattern detection parses D.E.K.N. summaries correctly
- Documented Dylan signal patterns (worker-filtered, requires orchestrator session)

**2026-01-16 15:00:** Investigation completed
- Status: Complete
- Key outcome: 2 patterns verified working (action_ratio, analysis_paralysis), 5 patterns have sound logic but no production validation.
