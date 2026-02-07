<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Extended coaching.ts plugin with behavioral variation detection that tracks 3+ consecutive bash commands in the same semantic group without a 30-second strategic pause.

**Evidence:** All 28 semantic grouping tests pass; 6/6 variation logic tests pass; sess-4432 real-world pattern detected at variation 3.

**Knowledge:** Semantic grouping (overmind/tmux/launchctl → "process_mgmt") is more reliable than tool name matching; pattern order matters (test before build); 30s pause heuristic resets variation counter effectively.

**Next:** Close - all deliverables complete, tests passing, ready for orchestrator verification.

**Promote to Decision:** recommend-no (tactical implementation, not architectural)

---

# Investigation: Phase Behavioral Variation Detection Extend

**Question:** Can we extend the coaching.ts plugin with behavioral variation detection (3+ debugging attempts without strategic pause) using semantic tool grouping?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** og-feat-phase-behavioral-variation-10jan-1eed
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Semantic Grouping via Regex Patterns Works Reliably

**Evidence:** Created 8 semantic groups (process_mgmt, git, build, test, knowledge, orch, file_ops, network) with regex patterns. All 28 test cases passed:
- `overmind start/status/restart` → process_mgmt
- `tmux new-session` → process_mgmt  
- `launchctl kickstart/list` → process_mgmt
- `ps aux | grep overmind` → process_mgmt
- `git status/diff/commit` → git
- `npm test` vs `npm install` → test vs build (order matters)

**Source:** `~/.config/opencode/plugin/coaching.ts:87-132` (SEMANTIC_PATTERNS array)

**Significance:** Regex patterns are simple, fast, and extensible. Pattern order matters - more specific patterns (npm test) must come before general patterns (npm).

---

### Finding 2: Variation Counter Logic Correctly Detects Patterns

**Evidence:** Implemented variation tracking with these behaviors:
- Increments counter on consecutive same-group commands
- Resets counter on group switch
- Resets counter on 30s+ pause
- Emits `behavioral_variation` metric at threshold (3+)

6/6 variation logic tests passed including:
- 3 consecutive → DETECTED at variation 3
- Group switch → NO DETECTION (correct)
- Strategic pause → NO DETECTION (correct)
- Sess-4432 pattern → DETECTED at variation 3

**Source:** `~/.config/opencode/plugin/coaching.ts:385-454` (variation detection logic)

**Significance:** The implementation correctly implements the detection rules from Probe 2 (`.kb/investigations/2026-01-10-inv-probe-pattern-retrospective-validate-detection.md`).

---

### Finding 3: Strategic Pause Heuristic (30s) Effectively Resets State

**Evidence:** When `timeSinceLastTool >= 30000ms`:
- Logs strategic pause detection
- Resets `currentGroup` to null
- Resets `variationCount` to 0

Test verified: Commands separated by 35s delay do NOT trigger detection even with 3+ commands in same group.

**Source:** `~/.config/opencode/plugin/coaching.ts:389-401` (pause detection)

**Significance:** 30s threshold aligns with Probe 2 recommendation ("strategic pause" = stepping back to think). May need calibration based on real-world data.

---

## Synthesis

**Key Insights:**

1. **Semantic Grouping is More Reliable than Tool Names** - Grouping commands by domain (process_mgmt, git, build) captures the intent better than literal tool matching. `overmind status` and `launchctl list` are both "process management debugging" even though they're different tools.

2. **Pattern Order is Critical** - More specific patterns must come first. `npm test` must match "test" before the general `npm` pattern matches "build". This is a constraint for future pattern additions.

3. **Behavioral Proxy Beats Time-Based Threshold** - The original Probe 2 finding was validated: "3+ variations without pause" catches patterns faster than "15 minutes of debugging". In sess-4432, detection would occur at 3rd variation (~line 167) instead of waiting 15 minutes.

**Answer to Investigation Question:**

**YES** - The coaching.ts plugin was successfully extended with behavioral variation detection. Implementation includes:
- Semantic tool grouping (8 categories)
- Variation counter (increments on same-group, resets on switch/pause)
- Strategic pause heuristic (30s)
- Metric emission (`behavioral_variation`) at threshold

All success criteria met: detects 3+ variations in same semantic group without pause, including the sess-4432 real-world scenario.

---

## Structured Uncertainty

**What's tested:**

- ✅ Semantic grouping correctly classifies process_mgmt commands (verified: 28/28 tests pass)
- ✅ Variation counter triggers at 3+ consecutive same-group commands (verified: test2a, test3)
- ✅ Group switch resets counter (verified: test2c)
- ✅ Strategic pause resets counter (verified: test2e)
- ✅ Sess-4432 pattern detected at variation 3 (verified: test3)

**What's untested:**

- ⚠️ Real-world performance impact (not benchmarked)
- ⚠️ 30s pause threshold optimal for all scenarios (heuristic from Probe 2)
- ⚠️ Integration with dashboard display (Phase 3 work)
- ⚠️ Non-bash tool variation (only bash commands tracked)

**What would change this:**

- Finding would be WRONG if real orchestrator sessions have different semantic patterns than tests
- Finding would need REVISION if 30s pause is too long/short in production
- Finding would be INCOMPLETE if non-bash tools need variation tracking

---

## Implementation Recommendations

**Purpose:** Document what was implemented for orchestrator verification.

### Implemented Approach

**Extend Coaching Plugin with Behavioral Variation Detection** - Added semantic grouping, variation counter, and strategic pause detection to existing coaching.ts plugin.

**Why this approach:**
- Leverages existing plugin infrastructure (no new files)
- Follows Probe 2 recommendations for detection rules
- Testable independently from plugin runtime

**Trade-offs accepted:**
- Only tracks bash commands (not Read, Edit, etc.)
- Fixed 30s pause threshold (not configurable)
- Emits metric on every threshold crossing (may be noisy)

**Implementation sequence:**
1. ✅ Added `SemanticGroup` type and `SEMANTIC_PATTERNS` array
2. ✅ Added `classifyBashCommand()` function  
3. ✅ Extended `SessionState` with `VariationState`
4. ✅ Added variation detection in `tool.execute.after` hook
5. ✅ Added metric emission for `behavioral_variation`

---

### Implementation Details

**What was implemented:**
- Semantic grouping: 8 categories with regex patterns
- Variation tracking: counter + group + history
- Pause detection: 30s threshold
- Metric emission: `behavioral_variation` with group, commands, threshold

**Things to watch out for:**
- ⚠️ Pattern order matters - test before build
- ⚠️ Orch pattern uses `^orch\b` to avoid path matching
- ⚠️ "other" group commands don't affect variation count

**Areas needing further investigation:**
- Optimal pause threshold calibration
- Non-bash tool variation tracking
- Dashboard visualization of metrics

**Success criteria:**
- ✅ Detects 3+ variations in same semantic group
- ✅ Resets on group switch
- ✅ Resets on 30s pause  
- ✅ Detects sess-4432 pattern at variation 3

---

## References

**Files Examined:**
- `.kb/investigations/2026-01-10-inv-probe-1b-session-session-streaming.md` - SDK client.sendAsync() pattern
- `.kb/investigations/2026-01-10-inv-probe-pattern-retrospective-validate-detection.md` - Detection rules
- `~/.config/opencode/plugin/coaching.ts` - Existing plugin

**Commands Run:**
```bash
# Run unit tests
/opt/homebrew/bin/bun run test-variation-detection.ts
# Result: 28/28 grouping, 6/6 variation, sess-4432 detected
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-10-inv-probe-pattern-retrospective-validate-detection.md` - Source of detection rules
- **Workspace:** `.orch/workspace/og-feat-phase-behavioral-variation-10jan-1eed/` - Test files and synthesis

---

## Investigation History

**2026-01-10:** Investigation started
- Initial question: Can we extend coaching.ts with behavioral variation detection?
- Context: Phase 1 of Orchestrator Coaching Plugin epic

**2026-01-10:** Implementation complete
- Status: Complete
- Key outcome: Successfully extended coaching.ts with semantic grouping, variation counter, and pause detection. All tests pass.
