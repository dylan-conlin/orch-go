<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Coaching plugin is 90% complete - orchestrator metrics work (50 metrics collected), worker health metrics do not work (0 collected despite implemented code), and the root cause is session.metadata.role detection not firing.

**Evidence:** Live metrics file shows 19 action_ratio, 19 analysis_paralysis, 12 compensation_pattern entries; zero worker metrics (tool_failure_rate, context_usage, time_in_phase, commit_gap); detection code updated to use session.metadata.role on Jan 17 (commit 37b9b0b0).

**Knowledge:** Worker detection depends on OpenCode server setting session.metadata.role='worker' when x-opencode-env-ORCH_WORKER=1 header is present; orch spawn sets this header but the plugin isn't detecting workers, suggesting either metadata not being set or cached false detection.

**Next:** Debug session.metadata.role by adding logging to detectWorkerSession(); verify OpenCode actually sets metadata.role from the header; consider falling back to file-path detection if metadata-based detection proves unreliable.

**Promote to Decision:** recommend-no - This is debugging/status work, not an architectural decision.

---

# Investigation: Understand Coaching Plugin Status and Current Implementation

**Question:** What is the current status of the coaching plugin, what's implemented in plugins/coaching.ts, and what work remains?

**Started:** 2026-01-18
**Updated:** 2026-01-18
**Owner:** Agent og-arch-understand-coaching-plugin-18jan-f9b8
**Phase:** Complete
**Next Step:** None - status documented, roadmap identified
**Status:** Complete

<!-- Lineage -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Plugin Implementation is Comprehensive (1831 lines)

**Evidence:** The coaching plugin at `plugins/coaching.ts` implements 8 behavioral detection patterns:

| Pattern | Purpose | Status | Metrics Found |
|---------|---------|--------|---------------|
| `action_ratio` | Low actions vs reads (option theater) | Working | 19 |
| `analysis_paralysis` | Tool repetition sequences | Working | 19 |
| `compensation_pattern` | Dylan providing repeated context | Working | 12 |
| `behavioral_variation` | Semantic group thrashing | Implemented | 0 |
| `circular_pattern` | Contradicting prior investigations | Implemented | 0 |
| `frame_collapse` | Orchestrator editing code files | Implemented | 0 |
| `dylan_signal_prefix` | Explicit signals (frame-collapse:, etc.) | Implemented | 0 |
| `premise_skipping` | "How to X" without "Should we X" | Implemented | 0 |

Worker health metrics (separate from orchestrator patterns):
- `tool_failure_rate` - consecutive tool failures (0 collected)
- `context_usage` - estimated token usage (0 collected)
- `time_in_phase` - minutes since phase change (0 collected)
- `commit_gap` - time since last commit (0 collected)

**Source:** `plugins/coaching.ts:1-1831`, `~/.orch/coaching-metrics.jsonl`

**Significance:** Core orchestrator coaching infrastructure works. Worker health tracking code exists (lines 1157-1289) but never fires due to worker detection failure.

---

### Finding 2: Worker Detection Updated to Use session.metadata.role (Jan 17)

**Evidence:** The `detectWorkerSession()` function was rewritten on Jan 17 (commit 37b9b0b0) to use session.metadata.role:

```typescript
function detectWorkerSession(sessionId: string, session?: { metadata?: { role?: string } }): boolean {
  // Check cache first
  const cached = workerSessions.get(sessionId)
  if (cached === true) return true

  // Use session metadata role (set by OpenCode from x-opencode-env-ORCH_WORKER header)
  if (session?.metadata?.role === 'worker') {
    workerSessions.set(sessionId, true)
    log(`Worker detected (session.metadata.role): session ${sessionId}`)
    return true
  }

  return false
}
```

The chain of trust:
1. `orch spawn` calls `pkg/opencode/client.go:561` which sets `req.Header.Set("x-opencode-env-ORCH_WORKER", "1")`
2. OpenCode server is supposed to set `session.metadata.role = 'worker'` from this header
3. Plugin reads `session.metadata.role` in tool hooks

**Source:** `plugins/coaching.ts:1317-1330`, `pkg/opencode/client.go:561`, `.kb/investigations/2026-01-17-inv-update-coaching-plugin-session-metadata.md`

**Significance:** This is the intended worker detection mechanism, replacing older file-path heuristics. Zero worker metrics in production suggest either: (a) OpenCode isn't setting the metadata, or (b) the session object isn't being passed correctly to detection.

---

### Finding 3: Prior detectWorkerSession Bugs Documented

**Evidence:** Two recent investigations identified detection bugs:

1. **Caching bug** (`.kb/investigations/2026-01-17-inv-design-review-coaching-plugin-failures.md`):
   - Original code cached BOTH true and false results
   - First non-matching tool call would permanently mark session as non-worker
   - Fix: Only cache true, never cache false

2. **Broken bash workdir check** (same investigation):
   - Detection signal checking `args?.workdir` on bash tools
   - Bash tool has no `workdir` argument - signal never fires
   - Was removed in favor of metadata-based detection

3. **Deep analysis** (`.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md`):
   - Confirmed plugin infrastructure works (orchestrator metrics appearing)
   - Worker detection is the blocking issue
   - Recommended verifying the fix is deployed and monitoring for worker metrics

**Source:** `.kb/investigations/2026-01-17-inv-*-coaching*.md` (3 investigations)

**Significance:** Multiple fix attempts have been made. Current metadata-based approach should work IF OpenCode sets the metadata and IF the session object is passed to detection.

---

### Finding 4: API and Dashboard Integration Complete

**Evidence:** The coaching system has full stack integration:

- **Plugin** (`plugins/coaching.ts`): Writes to `~/.orch/coaching-metrics.jsonl`
- **API** (`cmd/orch/serve_coaching.go`, 321 lines): Reads JSONL, aggregates by session, returns JSON
- **Dashboard** (`web/src/lib/stores/coaching.ts`): Svelte store for UI consumption

API endpoint `/api/coaching` returns:
- Overall status (good/warning/poor)
- Status message ("Orchestrator delegating well", etc.)
- Session duration and metrics
- Worker health map (currently empty due to detection bug)

**Source:** `cmd/orch/serve_coaching.go:1-321`, design doc `docs/designs/2026-01-10-orchestrator-coaching-plugin.md`

**Significance:** The pipeline from detection to display is complete. Only the worker detection issue blocks full functionality.

---

### Finding 5: Tier 2 Streaming (Context Injection) Implemented

**Evidence:** The plugin implements real-time coaching injection using OpenCode's `client.session.prompt()` with `noReply: true`:

```typescript
await client.session.prompt({
  sessionID: sessionId,
  prompt: message,
  noReply: true,
})
```

Injection triggers:
- `action_ratio < 0.5` with `reads >= 6` → coaching message
- `analysis_paralysis >= 3` → warning about tool repetition
- `frame_collapse` (code file edit) → tiered warnings (1st gentle, 3rd+ strong)
- `premise_skipping` → graduated reminders

**Source:** `plugins/coaching.ts:667-761` (injectCoachingMessage), `plugins/coaching.ts:545-658` (flushMetrics triggers)

**Significance:** This is the "Pain as Signal" pattern - agents feel friction in real-time rather than learning about it post-hoc. Worker health injection (`injectHealthSignal`) is implemented but doesn't fire because workers aren't detected.

---

### Finding 6: Design Document Rollout Status

**Evidence:** The design doc (`docs/designs/2026-01-10-orchestrator-coaching-plugin.md`) defined 4 phases:

| Phase | Description | Status |
|-------|-------------|--------|
| Phase 1 | Plugin + metrics storage | Complete |
| Phase 2 | API endpoint | Complete |
| Phase 3 | Dashboard view | Complete |
| Phase 4 | Hypothesis testing | Blocked |

Success criteria from design:
- [x] Plugin tracks metrics and writes to JSONL
- [x] API endpoint returns aggregated metrics
- [x] Dashboard displays coaching card
- [ ] Coaching messages are actionable (worker signals not reaching agents)
- [ ] Hypothesis test plan documented

**Source:** `docs/designs/2026-01-10-orchestrator-coaching-plugin.md:219-286`

**Significance:** Infrastructure complete, effectiveness validation blocked by worker detection issue.

---

## Synthesis

**Key Insights:**

1. **Orchestrator Coaching Works, Worker Coaching Doesn't** - The plugin successfully detects and tracks orchestrator behavioral patterns (50+ metrics). Worker health tracking has comprehensive code but produces zero metrics because workers aren't being detected. This is a single-point-of-failure in `detectWorkerSession()`.

2. **Detection Mechanism Changed Three Times** - Worker detection evolved from: (a) ORCH_WORKER env var → (b) file-path heuristics → (c) session.metadata.role. Each change was documented in investigations but the current approach remains unverified in production.

3. **"Pain as Signal" Architecture is Sound** - The three-layer system (detection → threshold transformation → tool-layer injection) is implemented correctly. Real-time coaching works for orchestrators. The issue is purely in worker identification, not the coaching mechanism itself.

4. **Missing Link: OpenCode Metadata Propagation** - The coaching plugin expects `session.metadata.role === 'worker'`, and orch spawn sets `x-opencode-env-ORCH_WORKER=1` header. The untested assumption is that OpenCode server translates the header into session metadata.

**Answer to Investigation Question:**

The coaching plugin is **90% complete**:
- **Implemented and working:** Orchestrator metrics (action_ratio, analysis_paralysis, compensation_pattern), API endpoint, dashboard integration, real-time coaching injection
- **Implemented but broken:** Worker health metrics (tool_failure_rate, context_usage, time_in_phase, commit_gap) - code exists but detection fails
- **Implemented but unverified:** behavioral_variation, circular_pattern, dylan_signal_prefix, frame_collapse (may work but no production examples)

**Remaining work:**
1. **Critical:** Debug session.metadata.role detection - why aren't workers being detected?
2. **Medium:** Verify untested patterns trigger correctly
3. **Optional:** Consider daemon-based architecture to decouple injection from observation (architectural improvement)

---

## Structured Uncertainty

**What's tested:**

- ✅ Orchestrator metrics appear in JSONL file (verified: tail -50 showing 50 entries dated today)
- ✅ action_ratio calculation works (verified: 19 entries with values 0-0.88)
- ✅ analysis_paralysis detection works (verified: 19 entries showing 3-10 consecutive tools)
- ✅ compensation_pattern detection works (verified: 12 entries from today)
- ✅ API endpoint exists and is wired up (verified: read serve_coaching.go)
- ✅ orch spawn sets x-opencode-env-ORCH_WORKER header (verified: pkg/opencode/client.go:561)
- ✅ detectWorkerSession caches only true results (verified: current code at lines 1317-1330)

**What's untested:**

- ⚠️ OpenCode actually sets session.metadata.role from the header (not verified)
- ⚠️ session object is passed to detectWorkerSession in tool hooks (assumed from code structure)
- ⚠️ Worker health injection triggers and reaches agents (0 worker metrics = can't test)
- ⚠️ behavioral_variation, circular_pattern patterns work in production
- ⚠️ Coaching messages actually influence agent behavior (hypothesis testing not done)

**What would change this:**

- Finding wrong if OpenCode logs show session.metadata.role being set correctly
- Finding wrong if worker metrics suddenly appear (would mean detection works)
- Finding wrong if adding debug logging shows detection succeeding but metrics not writing

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable debugging.

### Recommended Approach ⭐

**Debug Worker Detection** - Add logging to understand why workers aren't detected.

**Why this approach:**
- Worker detection is the single blocking issue (Finding 2)
- Code looks correct, need runtime data to diagnose
- Minimal invasive change (logging only)

**Trade-offs accepted:**
- Requires server restart to pick up logging changes
- Log noise until diagnosis complete

**Implementation sequence:**
1. Add `console.log` in `detectWorkerSession()` showing: sessionId, session?.metadata, return value
2. Restart OpenCode server (`orch-dashboard restart`)
3. Spawn a worker: `orch spawn investigation "test" --issue TEST-1`
4. Check OpenCode logs for detection output
5. If metadata undefined → problem is OpenCode not setting it
6. If metadata.role exists but not 'worker' → problem is header not being read
7. If metadata.role === 'worker' but function returns false → problem in plugin logic

### Alternative Approaches Considered

**Option B: Revert to File-Path Heuristics**
- **Pros:** Proven to work (was detecting before metadata approach)
- **Cons:** Less reliable (orchestrators reading workspace files could be misclassified)
- **When to use instead:** If metadata-based detection proves fundamentally broken

**Option C: Add Explicit Worker Flag in Spawn Context**
- **Pros:** Definitive identification, no heuristics
- **Cons:** Requires template changes, doesn't help existing sessions
- **When to use instead:** If all other detection mechanisms fail

**Rationale for recommendation:** Option A is diagnostic, Options B and C are fallbacks. Debug first before changing approach.

---

### Implementation Details

**What to implement first:**
- Debug logging in detectWorkerSession() (highest diagnostic value)
- Check OpenCode server logs for header processing

**Things to watch out for:**
- ⚠️ Plugin runs in OpenCode server process, not agent process
- ⚠️ Logging must use console.log (not plugin's log() which requires DEBUG=1)
- ⚠️ Server restart required for plugin changes

**Areas needing further investigation:**
- OpenCode's session metadata handling from x-opencode-* headers
- Whether session object is correctly passed in tool.execute.after hook
- Performance impact of checking metadata on every tool call

**Success criteria:**
- ✅ Worker sessions emit tool_failure_rate metrics when tools fail
- ✅ Worker sessions emit context_usage metrics every 50 tool calls
- ✅ Dashboard shows worker health for active agents
- ✅ Zero regression in orchestrator metrics

---

## References

**Files Examined:**
- `plugins/coaching.ts:1-1831` - Full plugin implementation
- `cmd/orch/serve_coaching.go:1-321` - API endpoint
- `docs/designs/2026-01-10-orchestrator-coaching-plugin.md` - Original design
- `pkg/opencode/client.go:561` - ORCH_WORKER header setting
- `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md` - Prior analysis
- `.kb/investigations/2026-01-17-inv-design-review-coaching-plugin-failures.md` - Detection bugs
- `.kb/investigations/2026-01-16-inv-orch-go-investigation-test-coaching.md` - Pattern testing

**Commands Run:**
```bash
# Check recent metrics by type
tail -50 ~/.orch/coaching-metrics.jsonl | jq -r '.metric_type' | sort | uniq -c
# Result: 19 action_ratio, 19 analysis_paralysis, 12 compensation_pattern

# Check for worker metrics
grep -E "tool_failure_rate|context_usage" ~/.orch/coaching-metrics.jsonl | wc -l
# Result: 0

# Check deployed plugin
ls -la .opencode/plugin/coaching.ts
# Result: 59KB, dated Jan 17

# Check header setting
grep "x-opencode-env-ORCH_WORKER" pkg/opencode/client.go
# Result: Line 561 sets header
```

**External Documentation:**
- OpenCode Plugin API - tool.execute.after hook, session.prompt() with noReply
- JSONL format for metrics storage

**Related Artifacts:**
- **Design:** `docs/designs/2026-01-10-orchestrator-coaching-plugin.md` - Original design
- **Investigation:** `.kb/investigations/2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md` - Comprehensive architecture analysis
- **Investigation:** `.kb/investigations/2026-01-17-inv-update-coaching-plugin-session-metadata.md` - Metadata detection implementation
- **Beads:** `orch-go-v3v8z` (closed) - Metadata detection update task

---

## Investigation History

**2026-01-18 09:38:** Investigation started
- Initial question: Understand coaching plugin status, current implementation, and pending work
- Context: Spawned to document comprehensive understanding for orchestrator

**2026-01-18 09:45:** Evidence gathering phase
- Read design doc (294 lines), implementation (1831 lines), 3 related investigations
- Found 50+ orchestrator metrics in production, zero worker metrics
- Identified session.metadata.role as current detection mechanism

**2026-01-18 09:55:** Synthesis phase
- Mapped 8 behavioral patterns to implementation status
- Documented the detection evolution (env var → file heuristics → metadata)
- Identified worker detection as single blocking issue

**2026-01-18 10:05:** Investigation completed
- Status: Complete
- Key outcome: Coaching plugin is 90% complete; worker detection failure is the remaining blocker; recommended debug logging approach
