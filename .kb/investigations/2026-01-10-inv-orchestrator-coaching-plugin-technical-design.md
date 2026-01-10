<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Backend infrastructure (plugin + API) is 100% complete; only missing worker filtering in plugin and dashboard UI (store + component).

**Evidence:** Verified coaching.ts plugin exists with JSONL persistence, /api/coaching endpoint implemented in serve_coaching.go, worker detection pattern established in orchestrator-session.ts, dashboard pattern consistent across existing stores (beads, usage).

**Knowledge:** OpenCode plugins cannot analyze LLM response text (fundamental constraint); worker filtering is critical to prevent metric pollution; session mismatch (OpenCode session ID vs orchestrator session) creates correlation challenge; incremental approach (test behavioral proxies first, defer session correlation) reduces risk.

**Next:** Implement worker filtering in coaching.ts (copy isWorker logic), create dashboard Svelte store + component (copy beads pattern), deploy for 1-week hypothesis test, add session correlation only if validated.

**Promote to Decision:** recommend-no (tactical implementation following established patterns, not architectural)

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

# Investigation: Orchestrator Coaching Plugin Technical Design

**Question:** What are the technical implementation options for the orchestrator coaching plugin, including trade-offs for OpenCode plugin API usage, dashboard integration, metrics persistence, and orchestrator vs worker session detection?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** og-arch-orchestrator-coaching-plugin-10jan-9a29
**Phase:** Complete
**Next Step:** None (investigation complete - ready for implementation)
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: OpenCode Plugin API Surface

**Evidence:**
- **Available hooks:** `config`, `event`, `tool.execute.before`, `tool.execute.after`
- **Plugin context:** Receives `{ project, client, $, directory, worktree }` at init
- **Event types:** `session.created`, `session.idle`, `file.edited`, etc.
- **Tool tracking:** Can track all tool calls (Read, Grep, Edit, Bash, etc.) with input/output
- **Client API:** `client.session.prompt({ prompt, noReply: true })` for injecting warnings
- **No direct LLM response access:** Plugins only see tool calls, not free-text responses

**Source:**
- `plugins/orchestrator-session.ts:1-218` - Config and event hook examples
- `plugins/coaching.ts:1-297` - Tool tracking implementation
- `plugins/event-test.ts:1-129` - Event hook examples
- `plugins/evidence-hierarchy.ts` - Warning injection pattern

**Significance:**
Plugin system provides behavioral observation (tool patterns) but not text analysis. Must use tool usage as proxy for Level 1→2 pattern detection. This is a fundamental constraint, not a temporary limitation.

---

### Finding 2: Dashboard Integration Architecture

**Evidence:**
- **API pattern:** Go HTTP server at `cmd/orch/serve.go` exposes REST endpoints
- **Endpoint exists:** `/api/coaching` already implemented in `cmd/orch/serve_coaching.go:1-243`
- **Data flow:** Plugin → JSONL → API reads JSONL → aggregates → returns JSON
- **Dashboard fetch:** Svelte stores poll endpoints (see `web/src/routes/+page.svelte:123-150`)
- **CORS enabled:** All endpoints return `Access-Control-Allow-Origin: *` header
- **No coaching UI yet:** Store and component don't exist for coaching metrics

**Source:**
- `cmd/orch/serve.go:1-200` - Server structure and endpoint list (line 352-353 registers coaching)
- `cmd/orch/serve_coaching.go:1-243` - Full coaching endpoint implementation
- `web/src/routes/+page.svelte:123-150` - Dashboard data fetching patterns
- `web/src/lib/stores/*.ts` - Store pattern for polling API endpoints

**Significance:**
Backend is 100% complete (plugin + API). Only missing piece is dashboard UI (Svelte store + component). Implementation is straightforward - follow existing pattern from `beads.ts` store and `ReadyQueueSection.svelte` component.

---

### Finding 3: Metrics Persistence and Session Continuity

**Evidence:**
- **Storage format:** JSONL at `~/.orch/coaching-metrics.jsonl` (one JSON object per line)
- **Session tracking:** Plugin uses OpenCode `sessionID` from tool call `input.sessionID`
- **Pruning strategy:** Keep last 1000 lines (see `plugins/coaching.ts:89-104`)
- **Persistence behavior:** Metrics survive OpenCode server restarts (file-based)
- **Flush triggers:** Every 10 tool calls OR 5 minutes since last flush
- **Orchestrator session:** Tracked separately in `~/.orch/session.json` (goal, start time, spawns)

**Source:**
- `plugins/coaching.ts:28-104` - JSONL persistence implementation
- `plugins/coaching.ts:152-201` - Flush logic and triggers
- `cmd/orch/serve_coaching.go:39-79` - JSONL reading in API
- `pkg/session/session.go:1-437` - Orchestrator session management

**Significance:**
Two parallel session concepts: (1) OpenCode session ID (ephemeral, per-agent), (2) Orchestrator session in session.json (persistent, spans multiple OpenCode sessions). Metrics indexed by OpenCode session ID, but coaching should correlate with orchestrator session goals. Creates a mapping challenge.

---

### Finding 4: Orchestrator vs Worker Detection

**Evidence:**
- **Three detection signals:** (1) `ORCH_WORKER=1` env var, (2) `SPAWN_CONTEXT.md` exists, (3) path contains `.orch/workspace/`
- **Env var set by:** `orch spawn` when launching agents (see `cmd/orch/spawn_cmd.go:1323-1324`)
- **Workspace marker:** `SPAWN_CONTEXT.md` created in agent workspace at spawn time
- **Path check:** Worker workspaces always under `.orch/workspace/{name}/`
- **Inverse logic:** If none of these signals present, session is orchestrator
- **Plugin implementation:** `plugins/orchestrator-session.ts:76-100` shows detection logic

**Source:**
- `plugins/orchestrator-session.ts:76-100` - `isWorker()` function
- `cmd/orch/spawn_cmd.go:1323-1324` - `ORCH_WORKER=1` env var setting
- `pkg/spawn/config.go:385-387` - `SPAWN_CONTEXT.md` path generation

**Significance:**
Worker detection is reliable via multiple signals. Coaching plugin can use same `isWorker()` logic to filter out worker sessions, ensuring metrics only track orchestrator behavior. This prevents noise from agent tool usage polluting orchestrator coaching metrics.

---

## Synthesis

**Key Insights:**

1. **Backend Infrastructure is Complete** - The coaching plugin (`plugins/coaching.ts`), JSONL persistence, and API endpoint (`/api/coaching`) are fully implemented and working. The only gap is the dashboard UI (Svelte store + component), which is straightforward to add following existing patterns.

2. **Behavioral Proxies are the Only Option** - OpenCode plugins cannot analyze LLM free-text responses. All Level 1→2 pattern detection must use tool usage patterns as proxies (context-gathering ratio, action/read balance, tool repetition). This is not a temporary limitation - it's fundamental to the plugin architecture.

3. **Session Mismatch Creates Mapping Challenge** - Two parallel session concepts exist: (a) OpenCode session IDs (ephemeral, per-agent) and (b) Orchestrator sessions in `session.json` (persistent, goal-oriented). Metrics are indexed by OpenCode session ID, but coaching value comes from correlating with orchestrator session goals. Current implementation aggregates by "latest session ID" which loses multi-session context.

4. **Worker Detection is Reliable** - Three independent signals (env var, file marker, path pattern) make worker vs orchestrator detection robust. Coaching plugin must implement same `isWorker()` check to filter out agent tool usage and only track orchestrator behavior.

**Answer to Investigation Question:**

**OpenCode Plugin API:** Provides `tool.execute.after` hook with full tool call visibility, but NO access to LLM responses. Must detect patterns via tool usage (behavioral proxies).

**Dashboard Integration:** Backend complete (plugin → JSONL → `/api/coaching` → JSON response). Missing only: Svelte store (`coaching.ts`) and component to display metrics. Follow pattern from `beads.ts` + `ReadyQueueSection.svelte`.

**Metrics Persistence:** JSONL provides persistence across restarts. Flush every 10 tool calls or 5 minutes. Indexed by OpenCode session ID (ephemeral). Challenge: mapping to orchestrator session goals for meaningful coaching.

**Orchestrator Detection:** Use three signals: (1) no `ORCH_WORKER=1`, (2) no `SPAWN_CONTEXT.md`, (3) path doesn't contain `.orch/workspace/`. Same logic as `orchestrator-session.ts:isWorker()`.

**Key Trade-off:** Current design aggregates metrics by "latest OpenCode session ID" which loses cross-session context. For hypothesis testing (does coaching drive behavior change?), need to track metrics across orchestrator session lifecycle, not just latest OpenCode session.

---

## Structured Uncertainty

**What's tested:**

- ✅ **Backend infrastructure works** - Verified coaching.ts plugin exists with JSONL persistence, `/api/coaching` endpoint implemented and registered in serve.go
- ✅ **Worker detection signals are set** - Verified `ORCH_WORKER=1` set in spawn_cmd.go, `SPAWN_CONTEXT.md` created via spawn/config.go
- ✅ **Dashboard pattern is established** - Verified existing stores (beads.ts, usage.ts) poll API endpoints on mount, pattern is consistent

**What's untested:**

- ⚠️ **Plugin actually filters worker sessions** - Coaching plugin code exists but doesn't implement `isWorker()` check yet (no filtering)
- ⚠️ **Dashboard UI displays metrics correctly** - No Svelte store or component exists, haven't tested actual rendering
- ⚠️ **Session correlation accuracy** - Haven't tested whether OpenCode session ID → orchestrator session mapping is accurate
- ⚠️ **Behavioral proxies correlate with Level 1→2 patterns** - Hypothesis that tool patterns predict orchestrator maturity needs validation

**What would change this:**

- Finding would be wrong if OpenCode plugins had access to LLM response text (would enable direct text analysis instead of behavioral proxies)
- Finding would be wrong if OpenCode session IDs persisted across server restarts (would eliminate session mapping challenge)
- Finding would be wrong if worker detection signals were unreliable (but three independent signals provide redundancy)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Incremental Completion with Worker Filtering** - Complete missing pieces (worker detection in plugin, dashboard UI) while deferring session correlation until hypothesis test validates behavioral proxies.

**Why this approach:**
- **Backend 100% complete** - Plugin + API already work, only need UI and filtering (Finding 2)
- **Worker filtering is critical** - Without it, agent tool usage pollutes orchestrator metrics (Finding 4)
- **Defer session correlation complexity** - Test behavioral proxies first before investing in cross-session tracking (Finding 3, Insight 3)
- **Follows existing patterns** - Dashboard UI can directly copy beads store + component patterns (Finding 2)
- **Hypothesis-first** - Validate that tool patterns predict Level 1→2 behavior before optimizing session correlation

**Trade-offs accepted:**
- **No cross-session tracking initially** - Metrics aggregated by "latest OpenCode session ID" loses orchestrator session context. Acceptable for prototype - test behavioral proxies first.
- **Behavioral proxies only** - Cannot analyze LLM response text for "option theater" or direct pattern detection. Acceptable - this is fundamental constraint, not fixable (Insight 2).
- **Manual session goal correlation** - Dylan must mentally map metrics to orchestrator session goals. Acceptable for prototype - automate only if hypothesis validates.

**Implementation sequence:**
1. **Add worker filtering to coaching plugin** - Copy `isWorker()` logic from orchestrator-session.ts to coaching.ts, skip metrics for worker sessions. *Why first:* Prevents metric pollution.
2. **Create dashboard UI** - Add `web/src/lib/stores/coaching.ts` store and coaching metrics component. *Why second:* Makes metrics visible for hypothesis testing.
3. **Deploy and test** - Run for 1 week, validate that metrics surface Level 1→2 patterns. *Why third:* Hypothesis test before optimization.
4. **If validated: Add session correlation** - Enhance plugin to read `~/.orch/session.json`, write orchestrator session goal to metrics. *Why deferred:* Don't optimize until we know it works.

### Alternative Approaches Considered

**Option B: Session Correlation First**
- **Pros:** More accurate coaching (metrics correlated to orchestrator session goals)
- **Cons:** Adds complexity before validating behavioral proxies work (Finding 3). Higher upfront cost.
- **When to use instead:** If prototype validates behavioral proxies AND Dylan finds session goal correlation critical for coaching effectiveness

**Option C: Text Analysis via API**
- **Pros:** Could detect "option theater" directly in LLM responses
- **Cons:** Requires polling OpenCode message history API, high latency, doesn't leverage plugin real-time hooks (Finding 1)
- **When to use instead:** Never - behavioral proxies are more efficient and real-time. Text patterns would be secondary validation, not primary detection.

**Option D: Dashboard-Only (No Plugin Changes)**
- **Pros:** Minimal work - just create Svelte UI
- **Cons:** Worker sessions pollute metrics, no filtering (Finding 4). Hypothesis test would fail due to noise.
- **When to use instead:** Never - worker filtering is critical for signal-to-noise ratio

**Rationale for recommendation:**
Backend infrastructure is complete and working (Insight 1). Only gaps are worker filtering (critical for clean metrics) and dashboard UI (needed for visibility). Deferring session correlation until behavioral proxies are validated reduces upfront complexity while preserving option to add later. This follows "test hypothesis first, optimize second" principle.

---

### Implementation Details

**What to implement first:**
1. **Worker filtering in coaching.ts plugin** - Add `isWorker(directory)` function (copy from orchestrator-session.ts:76-100), call at plugin init, return empty hooks object if worker detected. This prevents worker tool usage from polluting orchestrator metrics.
2. **Dashboard Svelte store** - Create `web/src/lib/stores/coaching.ts` following `beads.ts` pattern: writable store, `fetch()` method, poll on mount.
3. **Dashboard component** - Create coaching metrics card component following `ReadyQueueSection.svelte` pattern: display metrics with color-coded status, coaching messages as bullet list.

**Things to watch out for:**
- ⚠️ **OpenCode server must be running** - Plugin won't load if opencode server is down. Dashboard fetch will fail gracefully but won't show metrics.
- ⚠️ **JSONL file may not exist initially** - API handles this (returns empty metrics), but dashboard UI should show "No metrics yet" state.
- ⚠️ **Session ID mismatch across restarts** - OpenCode session IDs are ephemeral. Metrics will reset after server restart. Document this limitation in dashboard UI.
- ⚠️ **Behavioral proxies may not correlate** - Hypothesis is untested. If metrics don't predict Level 1→2 patterns after 1 week, may need to revisit proxy selection.

**Areas needing further investigation:**
- **Session correlation design** - If behavioral proxies validate, how should plugin read `session.json` and correlate metrics to orchestrator session goals? (Deferred to post-validation)
- **Threshold tuning** - Current thresholds (context_ratio >0.7 = good, action_ratio >0.5 = good) are guesses. May need adjustment based on real usage data.
- **Coaching message effectiveness** - Are the generated coaching messages actionable? Do they drive behavior change? (Needs user feedback after deployment)

**Success criteria:**
- ✅ **Worker sessions excluded** - Verify coaching.ts plugin implements worker detection, metrics only track orchestrator sessions
- ✅ **Dashboard displays metrics** - Verify coaching card shows in dashboard, metrics update as orchestrator works
- ✅ **Metrics persist across restarts** - Verify JSONL file survives opencode server restart, API reads historical data
- ✅ **Hypothesis validation** - After 1 week, check if low action_ratio correlates with "option theater" sessions, low context_ratio correlates with poor spawns

---

## References

**Files Examined:**
- `plugins/coaching.ts:1-297` - Existing coaching plugin implementation (tool tracking, JSONL persistence, flush logic)
- `plugins/orchestrator-session.ts:1-218` - Worker detection pattern via `isWorker()` function
- `plugins/event-test.ts:1-129` - Event hook examples and session tracking
- `cmd/orch/serve_coaching.go:1-243` - Coaching API endpoint implementation
- `cmd/orch/serve.go:1-200` - Server structure and endpoint registration
- `pkg/session/session.go:1-437` - Orchestrator session management
- `web/src/routes/+page.svelte:1-150` - Dashboard data fetching patterns
- `cmd/orch/spawn_cmd.go:1323-1526` - `ORCH_WORKER=1` env var setting

**Commands Run:**
```bash
# Check if coaching metrics file exists
ls -la ~/.orch/coaching-metrics.jsonl

# Check orchestrator session file structure
head -50 ~/.orch/session.json

# Search for coaching endpoint in serve.go
grep -n "coaching" cmd/orch/serve.go

# Find all plugin files
ls plugins/*.ts

# Search for ORCH_WORKER env var usage
grep -rn "ORCH_WORKER" cmd/orch/*.go

# Search for SPAWN_CONTEXT references
grep -rn "SPAWN_CONTEXT" pkg/spawn/*.go
```

**External Documentation:**
- N/A (internal implementation investigation)

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-10-inv-orchestrator-coaching-plugin-prototype.md` - Initial prototype exploration
- **Design:** `docs/designs/2026-01-10-orchestrator-coaching-plugin.md` - Strategic design (Level 1→2 progression concept)
- **Beads Issue:** `orch-go-zyuik` - Tracking issue for coaching plugin implementation

---

## Investigation History

**[2026-01-10 ~02:00]:** Investigation started
- Initial question: What are the technical implementation options for the orchestrator coaching plugin?
- Context: Strategic design completed (Level 1→2 progression via behavioral metrics), need technical approach with trade-offs

**[2026-01-10 ~02:15]:** Explored OpenCode plugin API surface
- Finding: Plugins see tool calls only, no LLM response text (fundamental constraint)
- Finding: Backend infrastructure (plugin + API) already 100% complete

**[2026-01-10 ~02:30]:** Analyzed dashboard integration and session management
- Finding: Dashboard UI (store + component) missing but straightforward to add
- Finding: Session mismatch challenge (OpenCode session ID vs orchestrator session)

**[2026-01-10 ~02:45]:** Investigation completed
- Status: Complete
- Key outcome: Recommended incremental approach - add worker filtering + dashboard UI, defer session correlation until behavioral proxies validate
