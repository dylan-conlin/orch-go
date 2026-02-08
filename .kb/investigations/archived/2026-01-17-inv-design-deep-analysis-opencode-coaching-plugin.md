<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode coaching plugin operates through tool.execute.after hooks and experimental.chat.messages.transform hooks, with metrics persisted to JSONL and context injected via noReply:true pattern; primary reliability issue is detectWorkerSession caching bug that misclassifies workers as orchestrators.

**Evidence:** Read coaching.ts (1860 lines), 4 related investigations, 8 plugin files; verified zero worker-specific metrics in coaching-metrics.jsonl despite worker health tracking code being implemented; caching bug at line 1319-1360 returns cached false early, preventing detection on subsequent tool calls.

**Knowledge:** The "nervous system" architecture works through three layers: (1) behavioral detection in plugins, (2) pain threshold transformation, (3) tool-layer injection for real-time feedback. The fundamental constraint is that plugins cannot see LLM response text—only tool calls—making behavioral proxies the only detection mechanism.

**Next:** Verify the detectWorkerSession fix from commit b82715c1 follow-up is deployed; monitor coaching-metrics.jsonl for worker-specific metrics; consider implementing the daemon-based architecture to decouple injection from observation.

**Promote to Decision:** Superseded - coaching plugin disabled (2026-01-28-coaching-plugin-disabled.md)

---

# Investigation: Deep Analysis of OpenCode Coaching Plugin Architecture

**Question:** How does the OpenCode coaching plugin work (event hooks, injection mechanisms, detection logic), what are the current issues and limitations, and how does it relate to the "nervous system" concept?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-arch-deep-analysis-opencode-17jan-5651
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage -->
**Extracted-From:** N/A (original analysis)
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Plugin Architecture - Three Execution Patterns

**Evidence:** OpenCode plugins execute through three patterns, each with distinct purposes:

| Pattern | Hook | Can Block? | Use Case |
|---------|------|------------|----------|
| **Gates** | `tool.execute.before` | Yes (throw) | Block dangerous actions (bd-close-gate.ts) |
| **Context Injection** | `tool.execute.before`, `event` | No | Surface guidance (guarded-files.ts, coaching.ts) |
| **Observation** | `tool.execute.after`, `event` | No | Track behavior (coaching.ts, action-log) |

Plugin locations:
- Global: `~/.config/opencode/plugin/` (4 files: friction-capture.ts, session-compaction.ts, guarded-files.ts, session-resume.js)
- Project: `.opencode/plugin/` in project root (varies by project)
- orch-cli project: 5 plugins (session-context.ts, bd-close-gate.ts, usage-warning.ts, agentlog-inject.ts)

**Source:** `.kb/guides/opencode-plugins.md:30-60`, global plugin directory listing, orch-cli `.opencode/plugin/`

**Significance:** The coaching plugin uses all three patterns: observation (tool.execute.after for metrics), context injection (client.session.prompt for warnings), and experimental hooks (chat.messages.transform for Dylan pattern detection).

---

### Finding 2: Coaching Plugin Detection Logic - 8 Behavioral Patterns

**Evidence:** `plugins/coaching.ts` implements detection for 8 behavioral patterns:

| Metric Type | What It Detects | Trigger Condition | Action |
|-------------|-----------------|-------------------|--------|
| `action_ratio` | Low actions vs reads | ratio < 0.5, reads >= 6 | Inject coaching message |
| `analysis_paralysis` | Tool repetition | 3+ same tool consecutive | Inject warning |
| `behavioral_variation` | Semantic group thrashing | 3+ variations without 30s pause | Write to JSONL |
| `frame_collapse` | Orchestrator editing code | edit/write on code file | Tiered injection (1st warning, 3+ strong) |
| `circular_pattern` | Contradicting prior investigations | Decision keywords vs investigation Next | Stream to coach session |
| `dylan_signal_prefix` | User explicit signals | frame-collapse:/compensation:/focus:/step-back: | Stream to coach |
| `compensation_pattern` | Repeated keyword overlap | >30% keyword overlap | Stream to coach |
| `premise_skipping` | "How to X" without "Should we X" | Strategic verb in how-to pattern | Inject coaching |

**Source:** `plugins/coaching.ts:70-76` (CoachingMetric interface), lines 92-172 (semantic groups), lines 568-658 (flushMetrics)

**Significance:** Detection is comprehensive for orchestrator behavior but fundamentally limited—plugins cannot analyze LLM response text, only tool calls. This makes behavioral proxies the only detection mechanism.

---

### Finding 3: Worker-Specific Health Metrics - Implemented but Not Firing

**Evidence:** Worker health tracking is implemented in coaching.ts (lines 1157-1289) but produces zero metrics:

| Worker Metric | Purpose | Threshold |
|---------------|---------|-----------|
| `tool_failure_rate` | Consecutive tool failures | 3+ failures → warning |
| `context_usage` | Estimated token consumption | 80%+ → warning |
| `time_in_phase` | Minutes since phase change | 15+ min → warning |
| `commit_gap` | Time since last commit | 30+ min → warning |

Verification:
```bash
grep -E "tool_failure_rate|context_usage|time_in_phase|commit_gap" ~/.orch/coaching-metrics.jsonl
# Returns empty - zero worker metrics despite active workers
```

**Source:** `plugins/coaching.ts:1157-1289` (trackWorkerHealth function), `~/.orch/coaching-metrics.jsonl` (actual output)

**Significance:** Worker health tracking code exists but is never reached due to the detectWorkerSession bug (Finding 4).

---

### Finding 4: detectWorkerSession Caching Bug - Root Cause Identified and Fixed

**Evidence:** The detectWorkerSession function (lines 1319-1360) had a critical bug: it cached both true AND false results, permanently misclassifying workers as orchestrators if ANY tool call happened before a worker-identifying signal.

**Bug behavior:**
1. Worker session starts, first tool = `glob` (no detection signal)
2. `isWorker = false` → cached → function returns `false` forever
3. Second tool = `read SPAWN_CONTEXT.md` → cached result returned, detection skipped
4. Worker treated as orchestrator for entire session

**Fix documented in** `.kb/investigations/2026-01-17-inv-fix-detectworkersession-caching-bug-coaching.md`:
- Only cache `true` results (confirmed worker)
- Never cache `false` (allow re-evaluation)
- Restore filePath-based detection for any `.orch/workspace/` path
- Add file_path variant (snake_case support)

**Current code (fixed version):**
```typescript
function detectWorkerSession(sessionId: string, tool: string, args: any): boolean {
  const cached = workerSessions.get(sessionId)
  if (cached === true) {  // Only return early for confirmed workers
    return true
  }
  // ... detection logic ...
  if (isWorker) {
    workerSessions.set(sessionId, true)  // Only cache positive results
  }
  return isWorker
}
```

**Source:** `plugins/coaching.ts:1319-1360`, `.kb/investigations/2026-01-17-inv-design-review-coaching-plugin-failures.md`

**Significance:** This was the root cause of worker metrics not appearing. The fix requires server restart to take effect.

---

### Finding 5: Context Injection Mechanism - noReply Pattern

**Evidence:** Coaching messages are injected using OpenCode's `client.session.prompt()` with `noReply: true`:

```typescript
await client.session.prompt({
  path: { id: sessionId },
  body: {
    noReply: true,
    parts: [{ type: "text", text: message }]
  }
})
```

This pattern:
- Injects content as a user message in the session
- Doesn't block agent's workflow (no response expected)
- Used by: coaching.ts, guarded-files.ts, friction-capture.ts, session-resume.js, usage-warning.ts

The alternative pattern is `config.instructions.push(path)` which adds file references at config time (used by session-context.ts for orchestrator skill).

**Source:** `plugins/coaching.ts:747-757` (injectCoachingMessage), `.kb/guides/opencode-plugins.md:130-168`

**Significance:** Two distinct injection mechanisms: instructions (file references, config-time) vs prompt (content, runtime). Coaching uses runtime injection for immediate feedback.

---

### Finding 6: Nervous System Architecture - "Pain as Signal"

**Evidence:** The "nervous system" concept is documented in CLAUDE.md and `.kb/investigations/2026-01-17-inv-design-agent-self-health-context.md`:

**Architecture:**
```
Detection Layer (coaching.ts)
    ↓ tool.execute.after
Metrics Calculation (behavioral proxies)
    ↓ threshold crossing
Pain Signal Transformation
    ↓ client.session.prompt(noReply: true)
Agent Sensory Stream (tool-layer injection)
```

**Key principle:** Agents should "feel" friction in real-time, not learn about it post-hoc. Behavioral patterns (low action ratio, analysis paralysis, frame collapse) should be surfaced immediately when detected.

**Three-layer system:**
1. **Detection:** coaching.ts observes tool calls
2. **Transformation:** Pain thresholds convert metrics to signals
3. **Injection:** Tool-layer messages provide real-time feedback

**Source:** `CLAUDE.md` ("Pain as Signal" principle), `.kb/investigations/2026-01-17-inv-design-agent-self-health-context.md`

**Significance:** This is the architectural pattern—friction is first-class data that gets surfaced to agents, not just logged for human review.

---

### Finding 7: Metrics Persistence and API

**Evidence:** Coaching metrics are persisted to JSONL and exposed via HTTP API:

- **Storage:** `~/.orch/coaching-metrics.jsonl` (one JSON object per line)
- **Pruning:** Keeps last 1000 lines (MAX_LINES constant)
- **Flush triggers:** Every 10 tool calls OR 5 minutes since last flush
- **API endpoint:** `/api/coaching` in orch serve (cmd/orch/serve_coaching.go)
- **Dashboard:** Svelte store + component for visualization

**Source:** `plugins/coaching.ts:53-54, 391-419` (writeMetric, pruneMetrics), `cmd/orch/serve_coaching.go`

**Significance:** Metrics are durable across restarts. The gap is in-memory session state (lost on restart) and the coupling between observation and injection.

---

### Finding 8: Known Architectural Limitation - Injection Coupled to Observation

**Evidence:** The Jan 11 investigation (`.kb/investigations/2026-01-11-inv-review-design-coaching-plugin-injection.md`) identified a fundamental issue:

- Injection is triggered from `tool.execute.after` hook
- If server restarts, all session state is lost
- Coaching messages can't be injected independently of tool observation
- Detection and intervention are tightly coupled

**Recommendation not yet implemented:** Separate injection into an independent daemon that reads metrics from JSONL and injects coaching messages via API.

**Source:** `.kb/investigations/2026-01-11-inv-review-design-coaching-plugin-injection.md`

**Significance:** Current architecture has restart brittleness. Long-running sessions lose coaching capability after server restart.

---

## Synthesis

**Key Insights:**

1. **Plugins Can Only See Tool Calls, Not LLM Text** - This is a fundamental constraint, not a bug. All behavioral pattern detection must use tool usage as proxy signals (action ratio, repetition, semantic grouping). Direct analysis of agent reasoning or "option theater" in responses is impossible at the plugin level.

2. **Detection Is Comprehensive But Delivery Is Broken** - The coaching plugin implements 8 behavioral patterns with appropriate thresholds. The problem is delivery: worker sessions were being misclassified as orchestrators, so worker health metrics never fired. The fix exists but requires verification.

3. **"Pain as Signal" Is the Architectural Pattern** - The nervous system concept isn't about monitoring—it's about real-time feedback. Agents should feel friction immediately (tool-layer injection) rather than having it logged for post-hoc analysis. This is the "pressure over compensation" principle from CLAUDE.md.

4. **Two Injection Mechanisms Serve Different Purposes** - `config.instructions` adds file references at config time (good for large static context like skills). `client.session.prompt(noReply: true)` injects content at runtime (good for immediate coaching). The orchestrator skill uses the former; coaching warnings use the latter.

5. **Worker vs Orchestrator Need Different Health Signals** - Workers need: tool failure tracking, context budget warnings, time-in-phase reminders, commit gap nudges. Orchestrators need: frame collapse warnings, action ratio coaching, premise skipping detection. The existing detection correctly routes by session type when detection actually works.

**Answer to Investigation Question:**

**How the coaching plugin works:**
- **Hooks:** Primarily `tool.execute.after` for observation, `experimental.chat.messages.transform` for Dylan pattern detection
- **Detection:** 8 behavioral patterns using tool usage as proxies (can't see LLM text)
- **Injection:** `client.session.prompt(noReply: true)` for immediate feedback
- **Persistence:** JSONL file at ~/.orch/coaching-metrics.jsonl, exposed via /api/coaching

**Current issues:**
1. **Worker detection caching bug** - Fixed in code, needs server restart verification
2. **Injection coupled to observation** - Server restart breaks coaching
3. **No LLM text analysis** - Fundamental constraint, not fixable at plugin level

**Nervous system relationship:**
The coaching plugin IS the nervous system—it implements "Pain as Signal" by detecting behavioral friction and injecting it into agent context in real-time. The architecture is: detection → threshold transformation → tool-layer injection.

---

## Structured Uncertainty

**What's tested:**

- ✅ Coaching metrics file exists with recent entries (verified: tail -30 ~/.orch/coaching-metrics.jsonl shows data)
- ✅ Zero worker-specific metrics exist (verified: grep for tool_failure_rate/context_usage returns empty)
- ✅ Orchestrator metrics ARE working (verified: recent entries show action_ratio, analysis_paralysis, compensation_pattern)
- ✅ detectWorkerSession bug documented and fix implemented (verified: read investigation and code at lines 1319-1360)
- ✅ noReply injection pattern used in multiple plugins (verified: grep across plugin files)

**What's untested:**

- ⚠️ Whether the detectWorkerSession fix is currently deployed (requires server restart)
- ⚠️ Token estimation accuracy for context_usage metric (rough approximation)
- ⚠️ Actual effectiveness of coaching messages (do agents change behavior?)
- ⚠️ Performance impact of not caching false results (each tool call rechecks detection)

**What would change this:**

- Finding would be wrong if worker metrics start appearing (would mean fix is deployed and working)
- Finding would be wrong if plugins gain access to LLM response text (would enable direct text analysis)
- Nervous system interpretation would change if injection was decoupled from observation (would need updated architecture diagram)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation.

### Recommended Approach ⭐

**Verify and Monitor** - Restart OpenCode server to pick up detectWorkerSession fix, then monitor coaching-metrics.jsonl for worker-specific metrics to confirm the fix is working.

**Why this approach:**
- Detection code is mature and comprehensive (Finding 2)
- Fix for worker detection exists (Finding 4)
- Metrics persistence works (Finding 7)
- Just needs deployment verification

**Trade-offs accepted:**
- Doesn't address injection/observation coupling (requires larger refactor)
- Token estimation remains approximate
- Post-hoc verification, not prevention

**Implementation sequence:**
1. Restart OpenCode server (`orch-dashboard restart`)
2. Spawn a worker agent with `orch spawn feature-impl "test task" --issue TEST`
3. After worker completes, verify: `grep "tool_failure_rate\|context_usage" ~/.orch/coaching-metrics.jsonl`
4. If metrics appear → fix is working
5. If metrics still missing → debug detectWorkerSession further

### Alternative Approaches Considered

**Option B: Implement Daemon-Based Injection Architecture**
- **Pros:** Decouples observation from injection, survives restarts, more resilient
- **Cons:** Higher implementation cost, new infrastructure, scope expansion
- **When to use instead:** If restart brittleness becomes a significant operational problem

**Option C: Add Debug Logging for Worker Detection**
- **Pros:** Immediate visibility into detection behavior
- **Cons:** Requires ORCH_PLUGIN_DEBUG=1, adds noise
- **When to use instead:** If verification shows fix still doesn't work

**Rationale for recommendation:** Option A is the simplest path to validate existing fix. Options B and C are escalation paths if A fails.

---

### Implementation Details

**What to implement first:**
- Verify detectWorkerSession fix is deployed
- Monitor coaching-metrics.jsonl for worker metrics

**Things to watch out for:**
- ⚠️ Server restart required for plugin changes to take effect
- ⚠️ Worker detection depends on tool call order (first signals matter)
- ⚠️ In-memory session state lost on restart (coaching warnings stop)
- ⚠️ Orchestrators reading workspace files might be misclassified (edge case)

**Areas needing further investigation:**
- Daemon-based architecture to decouple injection from observation
- OpenCode token counting API for accurate context_usage
- Effectiveness measurement: do agents actually change behavior?
- Session correlation: map OpenCode session IDs to orchestrator sessions

**Success criteria:**
- ✅ Worker sessions emit tool_failure_rate metrics when tools fail
- ✅ Worker sessions emit context_usage metrics every 50 tool calls
- ✅ Orchestrator metrics continue to work (action_ratio, frame_collapse)
- ✅ Zero regression in existing coaching functionality

---

## References

**Files Examined:**
- `plugins/coaching.ts:1-1861` - Main coaching plugin implementation
- `~/.config/opencode/plugin/*.ts,js` - 4 global plugins
- `~/Documents/personal/orch-cli/.opencode/plugin/*.ts` - 5 project plugins
- `.kb/guides/opencode-plugins.md` - Plugin system guide
- `.kb/investigations/2026-01-10-inv-orchestrator-coaching-plugin-technical-design.md` - Original technical design
- `.kb/investigations/2026-01-16-inv-audit-opencode-session-start-injection.md` - OpenCode vs Claude Code audit
- `.kb/investigations/2026-01-17-inv-fix-detectworkersession-caching-bug-coaching.md` - Caching bug fix
- `.kb/investigations/2026-01-17-inv-design-review-coaching-plugin-failures.md` - Failure root cause analysis
- `.kb/investigations/2026-01-17-inv-investigate-missing-coaching-metrics-frame.md` - Missing metrics investigation
- `.kb/investigations/2026-01-17-inv-design-agent-self-health-context.md` - Agent self-health design
- `.kb/models/context-injection.md` - Context injection architecture model
- `.kb/guides/resilient-infrastructure-patterns.md` - Pain as Signal principle
- `CLAUDE.md` - Pain as Signal architectural principle

**Commands Run:**
```bash
# Check recent coaching metrics
tail -30 ~/.orch/coaching-metrics.jsonl

# Check for worker-specific metrics
grep -E "tool_failure_rate|context_usage|time_in_phase|commit_gap" ~/.orch/coaching-metrics.jsonl

# List global plugins
ls -la ~/.config/opencode/plugin/

# Search for coaching-related files
grep -r "coaching" /Users/dylanconlin/Documents/personal
```

**External Documentation:**
- OpenCode Plugin API - noReply pattern for context injection

**Related Artifacts:**
- **Guide:** `.kb/guides/opencode-plugins.md` - Plugin system reference
- **Model:** `.kb/models/context-injection.md` - Context injection architecture
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-agent-self-health-context.md` - Pain as Signal design
- **Investigation:** `.kb/investigations/2026-01-17-inv-fix-detectworkersession-caching-bug-coaching.md` - Caching bug fix

---

## Investigation History

**2026-01-17 21:23:** Investigation started
- Initial question: Deep analysis of OpenCode coaching plugin architecture, issues, and nervous system concept
- Context: Spawned as architect session to produce comprehensive analysis

**2026-01-17 21:28:** Plugin architecture cataloged
- Found 4 global plugins, 5 orch-cli project plugins
- Identified 3 plugin execution patterns (gates, injection, observation)
- Read coaching.ts (1861 lines) and related investigations

**2026-01-17 21:35:** Root cause identified
- detectWorkerSession caching bug prevents worker health metrics
- Fix documented but needs deployment verification
- Metrics file shows orchestrator metrics but zero worker metrics

**2026-01-17 21:45:** Nervous system concept synthesized
- "Pain as Signal" = real-time friction surfacing, not post-hoc logging
- Three-layer architecture: detection → transformation → injection
- Fundamental constraint: plugins can't see LLM text, only tool calls

**2026-01-17 21:55:** Investigation completed
- Status: Complete
- Key outcome: Comprehensive architecture documented; primary issue is detectWorkerSession caching bug; nervous system = Pain as Signal pattern
