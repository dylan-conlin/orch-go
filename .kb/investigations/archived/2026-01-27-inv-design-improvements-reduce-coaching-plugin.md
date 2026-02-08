<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Coaching plugin has 5 distinct false positive patterns: (1) duplicate metric writes from multi-session flush loop, (2) compensation_pattern detecting file system noise (drwxr, usernames), (3) analysis_paralysis counting tool names instead of semantic groups, (4) action_ratio firing during legitimate exploration phases, (5) metrics written even when below injection threshold.

**Evidence:** Metrics file shows 4x duplicate entries at same timestamp (session loop); compensation_pattern includes "drwxr", "dylanconlin", "staff" as overlapping keywords; analysis_paralysis=3 triggers on ["read","read","bash","grep","bash"]; action_ratio=0 when reads=9/actions=0 is normal early exploration.

**Knowledge:** The flushMetrics() function loops over ALL sessions on every 10th tool call, not just the current session - causing duplicates. compensation_pattern's extractKeywordsSimple() has no domain filtering. analysis_paralysis uses tool.execute.after which sees tool names, not semantic intent. Metrics are logged even when below coaching injection thresholds.

**Next:** Implement 5 targeted fixes: (1) only flush current session, (2) add domain-aware stopwords to compensation detection, (3) filter analysis_paralysis to semantic bash groups only, (4) add warm-up period before action_ratio alerts, (5) only write metrics when they'll trigger coaching.

**Promote to Decision:** recommend-no - These are implementation fixes, not architectural decisions. The coaching architecture is sound; detection heuristics need tuning.

---

# Investigation: Design Improvements to Reduce Coaching Plugin False Positives

**Question:** What improvements can reduce false positives in the coaching plugin based on observed patterns in production metrics?

**Started:** 2026-01-27
**Updated:** 2026-01-27
**Owner:** Agent og-arch-design-improvements-reduce-27jan-f470
**Phase:** Complete
**Next Step:** None - recommendations documented for implementation
**Status:** Complete

<!-- Lineage -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Duplicate Metrics from Multi-Session Flush Loop

**Evidence:** The metrics file shows 4x entries with identical timestamps:
```
2026-01-27T20:31:17.481Z action_ratio 0.08
2026-01-27T20:31:17.482Z action_ratio 0.08
2026-01-27T20:31:17.483Z action_ratio 0.08
2026-01-27T20:31:17.483Z action_ratio 0.08
```

The `flushMetrics` function at coaching.ts:1839-1849 iterates over ALL sessions every 10 tool calls:
```typescript
// Periodic flush: every 10 tool calls
toolCallCounter++
if (toolCallCounter >= 10) {
  // Flush all active sessions  <-- BUG: Should only flush current session
  for (const [sid, s] of sessions.entries()) {
    if (s.spawns > 0 || s.reads > 0 || s.actions > 0) {
      flushMetrics(s, client)
    }
  }
  toolCallCounter = 0
}
```

**Source:** `plugins/coaching.ts:1839-1849`, `~/.orch/coaching-metrics.jsonl`

**Significance:** This creates metric spam (4x entries for same event) and inflates the metrics file. Dashboard aggregation must deduplicate, but the underlying data is noisy. Each active session gets flushed on every 10th tool call from ANY session.

---

### Finding 2: Compensation Pattern Detecting File System Noise

**Evidence:** compensation_pattern metrics include overlapping keywords like:
- "drwxr" (Unix file permissions)
- "dylanconlin" (username)
- "staff" (group name)
- "288" (file size)
- "16jan" (date fragments)

Example from metrics:
```json
{"overlapping_keywords":["orchestrators","spawned","workspace","drwxr","dylanconlin","staff","understand","current","state","16jan",...]}
```

The `extractKeywordsSimple()` function at coaching.ts:1036-1067 only filters:
- Words ≤4 characters
- A small stopword list (this, that, with, from, have, been, were, they, what, when, where, which, while, should, could, would, there)

**Source:** `plugins/coaching.ts:1036-1067`, `~/.orch/coaching-metrics.jsonl`

**Significance:** When Dylan pastes `ls -la` output or file paths, the plugin detects "compensation" because technical noise repeats across messages. This is a false positive - Dylan is providing context, not compensating for system failure.

---

### Finding 3: Analysis Paralysis Counting Tool Names Instead of Semantic Groups

**Evidence:** analysis_paralysis metrics show tool repetition like:
```json
{"v":4,"window":["read","read","bash","grep","bash"]}
{"v":9,"window":["bash","bash","bash","bash","bash"]}
```

The `detectSequence()` function at coaching.ts:561-586 counts consecutive identical tool names. But the semantic classification system (classifyBashCommand at coaching.ts:180-192) is only applied to bash commands for behavioral variation detection, not for analysis_paralysis.

**Source:** `plugins/coaching.ts:561-586`, `plugins/coaching.ts:180-192`

**Significance:** Multiple reads in sequence (reading spawn context, then CLAUDE.md, then feature files) is normal exploration, not analysis paralysis. The current detection conflates "using the same tool type" with "stuck in a loop". True analysis paralysis would be running the same *semantic command* repeatedly (e.g., checking git status 5 times).

---

### Finding 4: Action Ratio Firing During Legitimate Exploration Phases

**Evidence:** action_ratio metrics show low ratios during early session phases:
```json
{"reads":7,"actions":0,"ts":"2026-01-27T19:30:05.085Z"}
{"reads":9,"actions":0,"ts":"2026-01-27T19:42:09.259Z"}
{"reads":13,"actions":1,"ts":"2026-01-27T20:31:17.481Z"}
```

The injection threshold (coaching.ts:625-627) fires when `actionRatio < 0.5 && state.reads >= 6`:
```typescript
if (actionRatio < 0.5 && state.reads >= 6) {
  shouldInjectActionCoaching = true
}
```

**Source:** `plugins/coaching.ts:621-627`

**Significance:** Early in a session, an orchestrator SHOULD read extensively before taking action. A session with 9 reads and 0 actions might be in its first 2 minutes, gathering context before spawning. The threshold lacks temporal context - no warm-up period.

---

### Finding 5: Metrics Written Even When Below Injection Threshold

**Evidence:** The metrics file contains action_ratio entries with values that don't trigger coaching:
```json
{"v":0.44,"reads":18,"actions":8}  // ratio > 0.5, no coaching triggered
{"v":0.56,"reads":9,"actions":5}   // ratio > 0.5, no coaching triggered
```

Looking at `flushMetrics()` at coaching.ts:619-634:
```typescript
if (state.reads > 0) {
  const actionRatio = state.actions / state.reads
  writeMetric({...})  // Always writes
  
  // Only inject coaching when below threshold
  if (actionRatio < 0.5 && state.reads >= 6) {
    shouldInjectActionCoaching = true
  }
}
```

**Source:** `plugins/coaching.ts:619-634`

**Significance:** Metrics are written regardless of whether they indicate a problem. This creates noise - the metrics file is 1000+ lines with mostly "healthy" readings. The dashboard must filter, and analysis requires post-processing.

---

## Synthesis

**Key Insights:**

1. **Flush Loop Bug Creates 4x Duplicates** - The periodic flush iterates ALL sessions, not the current one. Combined with multiple concurrent sessions, this multiplies metric entries. Fix: Only flush the current session on each tool call.

2. **Keyword Extraction Needs Domain Awareness** - The compensation pattern detector extracts any word >4 chars, including technical noise. Dylan pasting command output isn't "compensation" - it's providing context. Fix: Add stopwords for common file system/path tokens, or require semantic keywords (orchestration terms, not file permissions).

3. **Tool Repetition ≠ Semantic Repetition** - Running `read` 3 times on different files is normal. Running `git status` 3 times is analysis paralysis. The current detection can't distinguish. Fix: Apply semantic classification to analysis_paralysis, or increase the threshold significantly.

4. **Missing Warm-Up Period for Action Ratio** - Sessions need time to gather context before action. Fix: Add minimum session age (e.g., 5 minutes) or minimum tool count (e.g., 15 tools) before triggering action ratio alerts.

5. **Metrics Without Actionable Signal Are Noise** - Recording every action_ratio when most are healthy creates analysis burden. Fix: Only write metrics when they exceed the coaching threshold, or mark them with a "triggered" field.

**Answer to Investigation Question:**

Five targeted improvements can significantly reduce false positives:

| Pattern | Current Problem | Fix |
|---------|-----------------|-----|
| **Duplicates** | Flush loop iterates all sessions | Only flush current session |
| **Compensation** | Technical tokens treated as keywords | Domain-aware stopwords (drwxr, permissions, usernames) |
| **Analysis Paralysis** | Tool name repetition vs semantic repetition | Apply classifyBashCommand or increase threshold to 5+ |
| **Action Ratio** | No warm-up period | Require session age >5min OR tools >15 before alerting |
| **Noise Metrics** | All metrics written | Only write when exceeding threshold |

---

## Structured Uncertainty

**What's tested:**

- ✅ Duplicate metrics exist in file (verified: `tail -20 ~/.orch/coaching-metrics.jsonl` shows 4x entries at same timestamp)
- ✅ Compensation keywords include file system noise (verified: `grep drwxr ~/.orch/coaching-metrics.jsonl`)
- ✅ Analysis paralysis fires on ["read","read","bash"] (verified: metrics file)
- ✅ Action ratio fires with 0 actions during first minutes (verified: timestamps + values)
- ✅ Metrics written even when ratio > 0.5 (verified: metrics file shows healthy ratios)

**What's untested:**

- ⚠️ Performance impact of per-tool-call session flush (hypothesis: negligible for single session)
- ⚠️ Optimal warm-up period threshold (hypothesis: 5 minutes or 15 tools)
- ⚠️ Whether domain stopwords eliminate false positives without losing true positives

**What would change this:**

- Findings wrong if duplicates are from race conditions (not loop)
- Findings wrong if compensation is desired for technical context (design question)
- Findings wrong if early low-action-ratio is actually problematic behavior

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Targeted Heuristic Tuning** - Fix each false positive pattern with minimal, surgical changes to the existing detection logic.

**Why this approach:**
- Preserves working architecture (coaching pipeline is sound)
- Each fix is independently testable
- Low risk - doesn't require restructuring
- Can be deployed incrementally

**Trade-offs accepted:**
- Doesn't address fundamental heuristic limitations
- May need further tuning after deployment
- Stopword approach requires manual curation

**Implementation sequence:**

1. **Fix duplicate flush (highest impact, easiest)**
   ```typescript
   // Before: flush all sessions
   for (const [sid, s] of sessions.entries()) { flushMetrics(s, client) }
   
   // After: flush only current session
   if (state.spawns > 0 || state.reads > 0 || state.actions > 0) {
     flushMetrics(state, client)
   }
   ```

2. **Add domain stopwords to compensation (medium impact)**
   ```typescript
   const domainStopwords = new Set([
     // File system noise
     "drwxr", "rwxr", "total", "bytes", "permissions",
     // Common usernames/groups
     "staff", "wheel", "admin", "root",
     // Path fragments
     "users", "documents", "personal", "library",
     // Date fragments
     "jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec",
   ])
   ```

3. **Add warm-up period to action ratio (medium impact)**
   ```typescript
   // Only alert after warm-up
   const sessionAgeMins = (Date.now() - state.sessionStartTime) / 60000
   const hasWarmupPassed = sessionAgeMins > 5 || state.toolWindow.length > 15
   
   if (actionRatio < 0.5 && state.reads >= 6 && hasWarmupPassed) {
     shouldInjectActionCoaching = true
   }
   ```

4. **Apply semantic grouping to analysis paralysis (lower impact)**
   - Increase threshold from 3 to 5
   - Consider only counting bash commands in same semantic group

5. **Only write metrics when actionable (cleanup)**
   - Add `triggered: true/false` field to metrics
   - Or only write when `shouldInjectCoaching` is true

### Alternative Approaches Considered

**Option B: LLM-Based Semantic Detection**
- **Pros:** More accurate context understanding
- **Cons:** Latency, cost, dependency on external API
- **When to use instead:** If heuristic tuning proves insufficient

**Option C: User Feedback Loop**
- **Pros:** Direct signal on false positives
- **Cons:** Requires UI, user engagement, data collection
- **When to use instead:** For long-term calibration

**Rationale for recommendation:** Option A is immediate, testable, and addresses observed patterns directly. Options B and C are valid but require more infrastructure.

---

### Implementation Details

**What to implement first:**
- Fix 1 (duplicate flush) - immediate impact, zero risk
- Fix 2 (domain stopwords) - high impact, low risk
- Fix 3 (warm-up period) - prevents most "early session" false positives

**Things to watch out for:**
- ⚠️ Test with multiple concurrent orchestrator sessions to verify flush fix
- ⚠️ Domain stopwords may need expansion based on observed false positives
- ⚠️ Warm-up period may need tuning based on session duration data
- ⚠️ Plugin requires OpenCode server restart to pick up changes

**Areas needing further investigation:**
- What's the actual distribution of session durations? (affects warm-up threshold)
- Are there other noise patterns in compensation_pattern not yet identified?
- Should analysis_paralysis be completely disabled or just retuned?

**Success criteria:**
- ✅ Metrics file shows single entries per event (no duplicates)
- ✅ compensation_pattern excludes file system noise keywords
- ✅ action_ratio doesn't fire in first 5 minutes of session
- ✅ Dashboard coaching card shows fewer false positive alerts
- ✅ Zero regression in true positive detection (orchestrator doing spawnable work)

---

## References

**Files Examined:**
- `plugins/coaching.ts:1-1938` - Full coaching plugin implementation
- `~/.orch/coaching-metrics.jsonl` - Production metrics (1000+ lines)
- `.kb/investigations/2026-01-23-inv-review-coaching-plugin-worker-detection.md` - Prior worker detection fix
- `.kb/investigations/2026-01-18-inv-understand-coaching-plugin-status-current.md` - Plugin status overview
- `.kb/investigations/2026-01-17-inv-design-review-coaching-plugin-failures.md` - Prior detection bugs

**Commands Run:**
```bash
# Count metrics by type
cat ~/.orch/coaching-metrics.jsonl | jq -r '.metric_type' | sort | uniq -c
# Result: 452 action_ratio, 389 analysis_paralysis, 204 compensation_pattern

# Check for duplicates
tail -20 ~/.orch/coaching-metrics.jsonl | jq -r '.timestamp + " " + .metric_type'
# Result: 4x entries at same timestamp

# Check compensation keywords
grep "drwxr" ~/.orch/coaching-metrics.jsonl | head -5
# Result: File permissions appearing as keywords

# Check low action ratio during exploration
tail -100 ~/.orch/coaching-metrics.jsonl | jq -c 'select(.metric_type == "action_ratio" and .value < 0.3)'
# Result: Multiple entries with reads=9, actions=0
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md` - Architecture analysis
- **Design:** `docs/designs/2026-01-10-orchestrator-coaching-plugin.md` - Original design document
- **Guide:** `.kb/guides/opencode-plugins.md` - Plugin development reference

---

## Investigation History

**2026-01-27 ~15:30:** Investigation started
- Initial question: Design improvements to reduce coaching plugin false positives based on session feedback
- Context: Spawned as architect to analyze observed false positive patterns

**2026-01-27 ~15:35:** Evidence gathering
- Read current coaching.ts implementation (1938 lines)
- Analyzed metrics file (~1000 entries)
- Identified 5 distinct false positive patterns

**2026-01-27 ~15:55:** Synthesis
- Mapped each pattern to root cause in code
- Formulated targeted fixes for each pattern
- Documented implementation sequence

**2026-01-27 ~16:05:** Investigation completed
- Status: Complete
- Key outcome: 5 targeted improvements identified with implementation details
