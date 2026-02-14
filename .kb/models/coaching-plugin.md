# Model: Coaching Plugin

**Domain:** OpenCode Plugins / Behavioral Monitoring
**Last Updated:** 2026-02-14
**Synthesized From:** 15 investigations (Jan 10-18, 2026) exploring coaching plugin implementation, worker detection failures, and injection architecture

---

## Summary (30 seconds)

The Coaching Plugin is an OpenCode plugin that provides real-time behavioral feedback to orchestrators and workers through tool usage pattern detection. It implements the "Pain as Signal" architectural pattern: agents should feel friction in real-time rather than learning about it post-hoc.

The plugin hooks `tool.execute.after` to observe tool usage patterns (it cannot see LLM response text—fundamental constraint), detects 8 behavioral patterns using behavioral proxies (action ratio, analysis paralysis, frame collapse, etc.), and injects coaching messages via `client.session.prompt({ noReply: true })`. Metrics persist to `~/.orch/coaching-metrics.jsonl` and are exposed via `/api/coaching` for dashboard visualization.

**Current status (Jan 2026):** Orchestrator coaching works (50+ metrics collected), worker health tracking doesn't fire (0 metrics collected despite implemented code). Root cause: worker detection has failed through multiple implementations (caching bugs, invalid detection signals, unverified metadata-based detection).

---

## Core Mechanism

### Overall Architecture

```
OpenCode Plugin Layer
    ↓ tool.execute.after hook
Behavioral Detection (8 patterns)
    ↓ threshold crossing
Metrics Calculation & Persistence (JSONL)
    ↓ flushMetrics trigger
Pain Signal Transformation
    ↓ client.session.prompt({ noReply: true })
Agent Sensory Stream (tool-layer injection)
    ↓ optional
Dashboard Visualization (/api/coaching)
```

### Detection Patterns (8 Behavioral Proxies)

| Pattern | What It Detects | Trigger Condition | Action |
|---------|-----------------|-------------------|--------|
| `action_ratio` | Low actions vs reads (option theater) | ratio < 0.5, reads >= 6 | Inject coaching message |
| `analysis_paralysis` | Tool repetition sequences | 3+ same tool consecutive | Inject warning |
| `behavioral_variation` | Semantic group thrashing | 3+ variations without 30s pause | Write to JSONL |
| `frame_collapse` | Orchestrator editing code | edit/write on code file | Tiered injection (1st warning, 3+ strong) |
| `circular_pattern` | Contradicting prior investigations | Decision keywords vs investigation Next | Stream to coach session |
| `dylan_signal_prefix` | User explicit signals | `frame-collapse:`/`compensation:`/`focus:`/`step-back:` | Stream to coach |
| `compensation_pattern` | Repeated keyword overlap | >30% keyword overlap | Stream to coach |
| `premise_skipping` | "How to X" without "Should we X" | Strategic verb in how-to pattern | Inject coaching |

**Worker health metrics (implemented but not firing):**
- `tool_failure_rate` - consecutive tool failures (threshold: 3+)
- `context_usage` - estimated token consumption (threshold: 80%+)
- `time_in_phase` - minutes since phase change (threshold: 15+ min)
- `commit_gap` - time since last commit (threshold: 30+ min)

### Data Flow

**Metric Collection:**
1. Tool call happens (Read, Grep, Edit, Bash, etc.)
2. `tool.execute.after` hook fires with `{ tool, args, result, sessionId }`
3. Plugin updates session state (in-memory Map)
4. Detection logic runs against semantic groups and thresholds
5. Metrics written to `~/.orch/coaching-metrics.jsonl` (append-only, pruned to 1000 lines)
6. Flush every 10 tool calls OR 5 minutes since last flush

**Injection Mechanism:**
1. `flushMetrics()` calculates behavioral metrics
2. Threshold crossing triggers `injectCoachingMessage()`
3. `client.session.prompt({ path: { id: sessionId }, body: { noReply: true, parts: [...] } })`
4. Message appears in agent's context stream (no response expected)

**Dashboard Visualization:**
1. `orch serve` reads `~/.orch/coaching-metrics.jsonl`
2. Aggregates by session ID and metric type
3. `/api/coaching` endpoint returns JSON: `{ overall_status, status_message, session_duration, metrics, worker_health }`
4. Dashboard Svelte store polls endpoint every 30s
5. UI displays health indicator (🟢/🟡/🔴) + last coaching timestamp

### Worker Detection Approaches (Evolution)

**Approach 1: File-path heuristics (Jan 10-11)**
- Detection signal: Any `args.filePath` containing `.orch/workspace/`
- Status: Removed in commit b82715c1, was most reliable signal

**Approach 2: ORCH_WORKER env var + SPAWN_CONTEXT.md (Jan 10-17)**
- Detection signals:
  - `ORCH_WORKER=1` env var (set by `orch spawn`)
  - `SPAWN_CONTEXT.md` file existence
  - Bash `args.workdir` containing `.orch/workspace/`
- Status: Failed - caching bug cached false results prematurely; bash workdir arg doesn't exist

**Approach 3: session.metadata.role (Jan 17+, current)**
- Chain of trust:
  1. `orch spawn` sets `x-opencode-env-ORCH_WORKER=1` header
  2. OpenCode server sets `session.metadata.role = 'worker'`
  3. Plugin checks `session?.metadata?.role === 'worker'`
- Status: Unverified in production - zero worker metrics suggest it's not working

### Critical Invariants

1. **Plugins cannot see LLM response text** - Only tool calls visible, not free-text responses. Fundamental constraint, not fixable.
2. **Behavioral proxies are the only detection mechanism** - All pattern detection uses tool usage as proxy signals.
3. **Metrics persist, session state doesn't** - JSONL file survives restarts, in-memory Map is ephemeral.
4. **Observation coupled to intervention** - Injection only fires from `flushMetrics` within `tool.execute.after` hook.
5. **Worker detection caching is one-way** - Only cache `true` results (confirmed worker), never cache `false`.
6. **Two injection mechanisms serve different purposes** - `config.instructions` adds file references at config time (static context like skills), `client.session.prompt(noReply: true)` injects content at runtime (immediate coaching).

---

## Why This Fails

### Failure Mode 1: Worker Detection Caching Bug (Jan 10-17)

**Symptom:** Zero worker health metrics in production despite implemented code

**Root cause:** `detectWorkerSession()` cached both `true` AND `false` results, permanently misclassifying workers if ANY tool call happened before a worker-identifying signal

**Why it happens:**
1. Worker session starts, first tool = `glob` (no detection signal)
2. `isWorker = false` → cached → function returns `false` forever
3. Second tool = `read SPAWN_CONTEXT.md` → cached result returned, detection skipped
4. Worker treated as orchestrator for entire session

**Cascade:**
```
First non-matching tool call → cache false → subsequent detection signals ignored → worker health code never runs
```

**Fix (Jan 17):** Only cache `true` results (confirmed worker), never cache `false`. Allow re-evaluation on each tool call until worker confirmed.

**Pattern established:** "Never cache negative results in per-session detection"

---

### Failure Mode 2: Invalid Detection Signal (Bash workdir)

**Symptom:** Detection code exists but never fires

**Root cause:** Detection checked for `args.workdir` on bash tool, but bash tool has no `workdir` argument

**Why it happens:**
- Bash tool args are: `command`, `timeout`, `dangerouslyDisableSandbox`, `run_in_background`
- No `workdir` argument exists
- Detection signal `if (tool === "bash" && args?.workdir)` never matches

**Fix (Jan 17):** Removed broken detection signal, restored file-path detection for any `.orch/workspace/` path

---

### Failure Mode 3: Observation Coupled to Intervention (Restart Brittleness)

**Symptom:** Coaching messages stop after OpenCode server restart, even though metrics show problems

**Root cause:** Injection is implemented as side effect of metric collection, not as separate concern that can operate independently

**Why it happens:**
- Metrics are **persistent** (JSONL file, survive restart)
- Session state is **in-memory** (Map, lost on restart)
- Injection logic **coupled to metric collection** (only happens via `flushMetrics`)
- After restart: metrics file shows "poor" status, but no session state exists, so injection never fires

**Cascade:**
```
Server restart → session state lost → flushMetrics not called → injection doesn't fire → coaching stops
```

**Architectural fix (not yet implemented):** Separate injection into independent daemon that reads metrics from JSONL and injects via OpenCode API, completely decoupled from plugin's observation code path

**Key insight:** This is "Observer Effect" problem - observation *enables* intervention, but intervention should happen based on what was observed (persistent), not whether we're currently observing (ephemeral)

---

### Failure Mode 4: Removed Most Reliable Detection Signal

**Symptom:** Detection became worse after "fix"

**Root cause:** Commit b82715c1 ("fix: enable plugin loading and refine worker detection") removed file-path detection

**What was removed:**
```typescript
// Detection signal 3: any tool with filePath in .orch/workspace/
if (args?.filePath && typeof args.filePath === "string") {
  if (args.filePath.includes(".orch/workspace/")) {
    isWorker = true
  }
}
```

**Why it matters:** Workers frequently read/write files in their workspace. File-path detection was the most reliable signal since it triggers on ANY file operation in worker directory.

**Result:** Detection now relies entirely on `session.metadata.role`, which is unverified in production.

---

### Failure Mode 5: session.metadata.role Detection Unverified

**Symptom:** Zero worker metrics in production after metadata-based detection implemented

**Root cause:** Unverified assumption that OpenCode server sets `session.metadata.role` from `x-opencode-env-ORCH_WORKER` header

**Chain of trust:**
1. `orch spawn` sets header: `req.Header.Set("x-opencode-env-ORCH_WORKER", "1")` ✅ verified in code
2. OpenCode server sets metadata: `session.metadata.role = 'worker'` ❓ unverified
3. Plugin reads metadata: `session?.metadata?.role === 'worker'` ✅ implemented

**Missing verification:** Need runtime logging to confirm OpenCode actually sets the metadata

**Recommended debug approach:**
1. Add `console.log` in `detectWorkerSession()` showing: sessionId, session?.metadata, return value
2. Restart OpenCode server
3. Spawn a worker, check logs for detection output
4. If metadata undefined → problem is OpenCode not setting it
5. If metadata.role exists but not 'worker' → problem is header not being read
6. If metadata.role === 'worker' but function returns false → problem in plugin logic

---

## Constraints

### Why Can't Plugins Analyze LLM Response Text?

**Constraint:** OpenCode plugins only see tool calls (via `tool.execute.before`/`after`), not free-text responses from the LLM

**Implication:** All pattern detection must use tool usage as behavioral proxies. Cannot detect "option theater" in agent's reasoning text directly.

**Workaround:** Use behavioral signals (low action ratio, tool repetition, semantic grouping) as proxies for underlying patterns

**This enables:** Real-time observation of agent behavior without polling
**This constrains:** Cannot analyze actual reasoning quality, only infer from actions

**Why this matters:** Agents experiencing "option theater" (analyzing without acting) can be detected by action_ratio < 0.5, but the specific reasoning can't be analyzed.

---

### Why Behavioral Proxies Instead of Direct Detection?

**Constraint:** Since plugins can't see LLM text, pattern detection requires inferring mental states from actions

**Implication:** Metrics like "action_ratio" and "analysis_paralysis" are proxies, not direct measurements

**Trade-offs:**
- **Pro:** Real-time detection without text parsing
- **Pro:** Lower latency (triggers on tool calls, not response parsing)
- **Con:** False positives (low action ratio might be legitimate investigation)
- **Con:** Cannot detect patterns that don't manifest in tool usage

**This enables:** Immediate feedback on behavioral patterns
**This constrains:** Cannot detect internal reasoning patterns (e.g., hallucination, confusion) unless they manifest in tool sequences

---

### Why Observation Coupled to Intervention?

**Constraint:** Injection logic is triggered from `flushMetrics` within `tool.execute.after` hook

**Implication:** Coaching can only happen when plugin is actively observing tool calls

**Current design:**
```
tool call → hook fires → update state → check thresholds → inject if needed
```

**Proposed design (not implemented):**
```
Observation: tool call → hook fires → update state → write metrics
Intervention: daemon polls metrics file → threshold check → inject via API
```

**This enables:** Simple implementation, low latency
**This constrains:** Restart brittleness, can't inject independently of observation

---

### Why In-Memory Session State?

**Constraint:** Session state (current tool sequence, timing, etc.) is stored in Map, not persisted

**Implication:** Server restart loses all session context

**Why in-memory:**
- Fast access (no file I/O on every tool call)
- Ephemeral by nature (sessions are temporary)
- Simplifies implementation

**Why this matters:** After restart, plugin can't resume coaching for existing sessions. Must re-detect patterns from scratch.

**This enables:** High-performance observation
**This constrains:** Restart loses context, coaching stops until new patterns emerge

---

### Why JSONL Format for Metrics?

**Constraint:** Metrics file is append-only JSONL, pruned to last 1000 lines

**Implication:** Metrics persistence is durable but not queryable

**Trade-offs:**
- **Pro:** Simple (no database, just append)
- **Pro:** Durable (survives crashes, restarts)
- **Pro:** Human-readable (can grep/tail for debugging)
- **Con:** No efficient time-range queries
- **Con:** Pruning loses historical data
- **Con:** Dashboard must read entire file to aggregate

**This enables:** Simple persistence without infrastructure
**This constrains:** Limited historical analysis, inefficient aggregation

---

## Evolution

**Jan 10, 2026: Initial Design and Prototype**
- Technical design investigation (orch-go-9a29)
- Explored OpenCode plugin API surface
- Identified behavioral proxies as only detection mechanism
- Backend infrastructure (plugin + JSONL + API) 100% complete

**Jan 10, 2026: Worker Filtering Implementation**
- Added `isWorker()` logic (3 detection signals)
- Implemented worker health metrics code
- **Caching bug introduced:** Cached both true and false results

**Jan 11, 2026: Pivot to AI Injection + Simplified Dashboard**
- Shifted from passive dashboard metrics to active AI injection
- Implemented `injectCoachingMessage()` using `client.session.prompt(noReply: true)`
- Simplified dashboard from 3-column metrics grid to single health indicator
- Discovered injection/observation coupling issue

**Jan 11, 2026: Architectural Review - "Coherence Over Patches"**
- Investigation identified 8 bugs in coaching area
- Root cause: injection coupled to observation creates restart brittleness
- Recommended architectural separation (not yet implemented)
- Established "separate observation from intervention" pattern

**Jan 16, 2026: Testing and Validation**
- Tested coaching pattern triggers
- Confirmed orchestrator metrics working in production

**Jan 17, 2026: Multiple Detection Failures**
- Deep analysis investigation (orch-go-5651) - comprehensive architecture documentation
- Design review investigation (orch-go-792c) - identified caching bug and invalid bash workdir signal
- Fix detectWorkerSession caching bug (orch-go-hflo3) - only cache true results
- Updated to session.metadata.role detection (orch-go-v3v8z)
- **Worker metrics still not appearing**

**Jan 17, 2026: Aggregator Command**
- Created `orch coaching` CLI command to read metrics file
- Aggregates by session and displays coaching summary

**Jan 18, 2026: Status Review**
- Comprehensive status investigation (orch-go-f9b8)
- Confirmed: orchestrator coaching works (50+ metrics), worker coaching broken (0 metrics)
- Recommended debug logging to verify session.metadata.role

---

## References

**Synthesized From (15 Investigations):**

**Initial Design & Prototype:**
- `2026-01-10-inv-orchestrator-coaching-plugin-technical-design.md` - Backend infrastructure complete, worker filtering needed
- `2026-01-10-inv-orchestrator-coaching-plugin-prototype.md` - Plugin system structure, behavioral proxies pattern
- `2026-01-10-inv-add-worker-filtering-coaching-ts.md` - Copy isWorker() logic from orchestrator-session.ts
- `2026-01-10-inv-debug-worker-filtering-coaching-ts.md` - Worker detection implementation
- `2026-01-10-inv-trigger-coaching-patterns-test.md` - Pattern trigger testing

**Architectural Pivots:**
- `2026-01-11-inv-pivot-coaching-plugin-two-frame.md` - Shift to AI injection + simplified dashboard
- `2026-01-11-inv-review-design-coaching-plugin-injection.md` - Injection/observation coupling issue, "Coherence Over Patches"

**Detection Failures:**
- `2026-01-17-inv-design-review-coaching-plugin-failures.md` - Caching bug, invalid bash workdir signal
- `2026-01-17-inv-fix-detectworkersession-caching-bug-coaching.md` - Fix: only cache true results
- `2026-01-17-inv-investigate-missing-coaching-metrics-frame.md` - Missing metrics analysis

**Architecture Analysis:**
- `2026-01-17-inv-design-deep-analysis-opencode-coaching-plugin.md` - Comprehensive architecture documentation, "Pain as Signal" pattern
- `2026-01-16-inv-orch-go-investigation-test-coaching.md` - Pattern testing

**Metadata Detection:**
- `2026-01-17-inv-update-coaching-plugin-session-metadata.md` - Switch to session.metadata.role detection

**Status & Tooling:**
- `2026-01-17-inv-update-coaching-aggregator-cmd-orch.md` - CLI command for metrics aggregation
- `2026-01-18-inv-understand-coaching-plugin-status-current.md` - 90% complete status review

**Related Decisions:**
- `.kb/decisions/2026-01-10-orchestrator-coaching-plugin.md` (if exists) - Level 1→2 progression design
- "Pain as Signal" principle (CLAUDE.md) - Real-time friction surfacing

**Related Models:**
- `.kb/models/PHASE4_REVIEW.md` - Model pattern at N=11, cognitive investment priorities
- `.kb/models/context-injection.md` (if exists) - Context injection architecture

**Related Guides:**
- `.kb/guides/opencode-plugins.md` - Plugin system reference
- `.kb/guides/resilient-infrastructure-patterns.md` (if exists) - Pain as Signal principle

**Primary Evidence (Verify These):**
- `plugins/coaching.ts` - Main plugin implementation (1831 lines as of Jan 18)
- `cmd/orch/serve_coaching.go` - API endpoint (321 lines)
- `pkg/opencode/client.go:561` - ORCH_WORKER header setting
- `~/.orch/coaching-metrics.jsonl` - Metrics persistence file
- `web/src/lib/stores/coaching.ts` - Dashboard Svelte store
- `web/src/routes/+page.svelte` - Dashboard UI integration

**Design Documents:**
- `docs/designs/2026-01-10-orchestrator-coaching-plugin.md` - Original strategic design

**Beads Issues:**
- `orch-go-zyuik` - Initial coaching plugin implementation
- `orch-go-hflo3` - Fix detectWorkerSession caching bug
- `orch-go-v3v8z` - Update to session.metadata.role detection
- `orch-go-rcah9` - Abandoned debugging attempts (2x)
