<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Agent Self-Health Context Injection operates through three layers: (1) coaching.ts metrics detection → (2) tiered signal transformation with pain thresholds → (3) tool-layer injection via OpenCode hooks for real-time feedback plus daemon-driven recovery protocol.

**Evidence:** coaching.ts already injects via `noReply: true` pattern (lines 697-710), stuck-agent-recovery investigation established advisory-first tiered recovery, daemon has poll-spawn-complete loop architecture that can accommodate health monitoring.

**Knowledge:** Pain-as-Signal works by treating behavioral friction (low action ratio, analysis paralysis, repeated tool failures) as first-class data that gets surfaced to agents in real-time rather than post-hoc analysis. The key insight is that agents can self-correct if given timely feedback about their own behavioral patterns.

**Next:** Implement in phases: (1) Extend coaching.ts with agent-side metric visibility, (2) Add health context section to SPAWN_CONTEXT.md template, (3) Add recovery loop to daemon with tiered escalation.

**Promote to Decision:** Superseded - coaching plugin disabled (2026-01-28)

---

# Investigation: Design Agent Self-Health Context Injection

**Question:** How should coaching metrics (from plugins/coaching.ts) be transformed into real-time agent signals, what is the 'Pain as Signal' mechanism for tool-layer injection, and what is the recovery protocol for stuck agents?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-arch-design-agent-self-17jan-daa6
**Phase:** Complete
**Next Step:** None - design ready for implementation
**Status:** Complete

<!-- Lineage -->
**Extracted-From:** N/A (original design)
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Coaching Metrics Are Already Captured and Classified

**Evidence:** `plugins/coaching.ts` implements comprehensive behavioral pattern detection:

| Metric Type | What It Detects | Current Action |
|-------------|-----------------|----------------|
| `action_ratio` | Low actions-to-reads ratio (<0.5 with 6+ reads) | Injects coaching message |
| `analysis_paralysis` | 3+ consecutive same-tool repetition | Injects coaching message |
| `behavioral_variation` | 3+ variations in same semantic group without pause | Writes to metrics file |
| `frame_collapse` | Orchestrator editing code files directly | Tiered injection (1st warning, 3+ strong warning) |
| `circular_pattern` | Decision contradicts prior investigation recommendation | Streams to coach session |
| `dylan_signal_prefix` | User uses explicit signal prefix (frame-collapse:, focus:, etc.) | Streams to coach session |
| `compensation_pattern` | Repeated keyword overlap (>30%) in user messages | Streams to coach session |
| `priority_uncertainty` | 2+ "what's next?" type questions | Streams to coach session |

**Source:** `plugins/coaching.ts:70-76` (CoachingMetric interface), lines 566-638 (flushMetrics), lines 645-715 (injectCoachingMessage)

**Significance:** The detection layer is mature. The gap is that agents don't have visibility into their own metrics - they receive coaching messages but can't proactively access their health state.

---

### Finding 2: Tool-Layer Injection Uses noReply Pattern

**Evidence:** OpenCode supports context injection via `client.session.prompt()` with `noReply: true`:

```typescript
await client.session.prompt({
  path: { id: sessionId },
  body: {
    noReply: true,
    parts: [{ type: "text", text: message }]
  }
})
```

This pattern is used in:
- `coaching.ts:697-710` - Frame collapse warnings
- `evidence-hierarchy.ts:284-294` - Evidence reminders
- Multiple existing plugins for gates and guidance

The injection appears as a user message but doesn't block the agent's workflow.

**Source:** `plugins/coaching.ts:697-710`, `.kb/investigations/2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md:135`

**Significance:** The injection mechanism is proven. The design decision is what to inject and when, not how to inject.

---

### Finding 3: Daemon Has Parallel Loop Architecture for Recovery

**Evidence:** The daemon operates with parallel loops:
1. **Spawn loop** (every 30s): Polls `bd ready`, spawns triage:ready issues
2. **Completion loop** (every 60s): Detects "Phase: Complete" comments, verifies, closes

From `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md`:
- Advisory-first principle established
- Tiered recovery: resume (non-destructive) → surface (visibility) → human decision
- Rate limiting: 1 resume/hour per agent to prevent infinite loops

**Source:** `.kb/guides/daemon.md:259-299`, `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md:46-58`

**Significance:** Recovery fits naturally as a third parallel loop. The design established by stuck-agent-recovery investigation should be followed.

---

### Finding 4: SPAWN_CONTEXT.md Already Includes KB Context

**Evidence:** Spawn context generation includes knowledge base context via `kb context` query:
- `pkg/spawn/kbcontext.go` - Runs `kb context` with task keywords
- Formats constraints, decisions, models, guides, investigations
- Truncates to ~80k chars (MaxKBContextChars) to prevent token bloat

This shows the pattern for including contextual health information at spawn time.

**Source:** `pkg/spawn/kbcontext.go:96-142`, `pkg/spawn/context.go:467-479`

**Significance:** Health context can follow the same pattern - computed at spawn time and included in SPAWN_CONTEXT.md, with real-time updates via plugin injection.

---

### Finding 5: Session State Is Per-Session with Worker Detection

**Evidence:** coaching.ts maintains per-session state with worker detection:
- Worker sessions are detected by: workspace path containing `.orch/workspace/`, SPAWN_CONTEXT.md reads, file paths in workspace
- Worker sessions are EXCLUDED from coaching metrics (lines 988-1036)
- This is intentional - coaching is for orchestrators, not workers

```typescript
// Detection signal 1: bash tool with workdir in .orch/workspace/
if (tool === "bash" && args?.workdir) {
  if (args.workdir.includes(".orch/workspace/")) {
    isWorker = true
  }
}
```

**Source:** `plugins/coaching.ts:988-1036`

**Significance:** Self-health context injection for workers needs different signals than orchestrator coaching. Workers need: context token visibility, tool failure patterns, stuck detection. Orchestrators need: delegation metrics, frame collapse warnings.

---

## Synthesis

**Key Insights:**

1. **Pain is Already Being Captured, Just Not Surfaced to Agents** - coaching.ts detects 8+ behavioral patterns but only injects messages when thresholds are crossed. Agents should have continuous visibility into their own health metrics (Finding 1).

2. **Dual-Mode Self-Health: Static (Spawn) + Dynamic (Runtime)** - Health context should be injected both at spawn time (baseline metrics from prior sessions) and at runtime (real-time feedback as patterns emerge). This follows the KB context pattern (Finding 4).

3. **Worker vs Orchestrator Need Different Health Signals** - Workers need token visibility, tool success rates, time-in-phase tracking. Orchestrators need delegation metrics, frame collapse warnings. The existing worker detection (Finding 5) provides the routing mechanism.

**Answer to Investigation Question:**

The Agent Self-Health Context Injection system operates through three interconnected mechanisms:

### 1. Metrics → Signals Transformation

**Architecture:**

```
┌─────────────────────────────────────────────────────────────────┐
│                     coaching.ts (Detection Layer)               │
│                                                                 │
│  tool.execute.after ──► Session State ──► Metrics Calculation  │
│                              │                                  │
│                              ▼                                  │
│                    ┌─────────────────┐                          │
│                    │ Metric Types:   │                          │
│                    │ • action_ratio  │                          │
│                    │ • analysis_para │                          │
│                    │ • frame_collapse│                          │
│                    │ • tool_failures │ (NEW)                    │
│                    │ • context_usage │ (NEW)                    │
│                    └────────┬────────┘                          │
│                             │                                   │
│                             ▼                                   │
│    ┌────────────────────────────────────────────────────┐       │
│    │           Pain Thresholds (per signal)             │       │
│    │                                                    │       │
│    │  action_ratio < 0.5      → "Low action warning"    │       │
│    │  analysis_paralysis >= 3 → "Stuck pattern warning" │       │
│    │  frame_collapse >= 1     → "Frame collapse warning"│       │
│    │  tool_failures >= 3      → "Tool friction warning" │ (NEW) │
│    │  context_usage > 80%     → "Context budget warning"│ (NEW) │
│    └────────────────────────────────────────────────────┘       │
│                             │                                   │
│                             ▼                                   │
│                   ┌──────────────────┐                          │
│                   │ Signal Emission  │                          │
│                   │ • Inject message │                          │
│                   │ • Write metrics  │                          │
│                   │ • Stream to coach│                          │
│                   └──────────────────┘                          │
└─────────────────────────────────────────────────────────────────┘
```

**New Metrics to Add:**

| Metric | What It Measures | Pain Threshold | Agent Type |
|--------|-----------------|----------------|------------|
| `tool_failure_rate` | Consecutive tool failures | 3 failures → warning | Worker |
| `context_usage` | Estimated tokens used / limit | 80% → warning | Both |
| `time_in_phase` | Minutes since last phase change | 15 min → warning | Worker |
| `commit_gap` | Time since last commit with changes | 30 min → warning | Worker |

### 2. Pain-as-Signal Tool-Layer Injection

**Mechanism:** Extend the existing `injectCoachingMessage()` function to support agent-specific health warnings.

**Injection Points:**

| Trigger | Signal Type | Message Template |
|---------|-------------|------------------|
| `tool.execute.after` with failure | `tool_friction` | "⚠️ {N} consecutive tool failures. Consider stepping back to reassess approach." |
| Periodic check (every 10 tool calls) | `context_budget` | "📊 Context usage: ~{X}% ({Y}k tokens estimated). {Z}k remaining." |
| Periodic check (every 5 min) | `time_check` | "⏱️ {N} minutes in {phase} phase. Consider checkpointing progress." |
| Periodic check (after 30 min) | `commit_reminder` | "💾 {N} minutes since last commit. Consider committing checkpoint." |

**Implementation Pattern:**

```typescript
// Add to coaching.ts tool.execute.after hook
async function checkAgentHealth(state: SessionState, tool: string, success: boolean): Promise<void> {
  // Track tool failures
  if (!success) {
    state.consecutiveFailures = (state.consecutiveFailures || 0) + 1
    if (state.consecutiveFailures >= 3) {
      await injectHealthSignal(client, state.sessionId, "tool_friction", {
        failures: state.consecutiveFailures,
        lastTool: tool
      })
    }
  } else {
    state.consecutiveFailures = 0
  }

  // Check context usage (estimate from tool calls)
  state.estimatedTokens = estimateTokenUsage(state)
  if (state.estimatedTokens > 80000) { // ~80% of typical 100k limit
    await injectHealthSignal(client, state.sessionId, "context_budget", {
      usage: Math.round(state.estimatedTokens / 1000),
      remaining: Math.round((100000 - state.estimatedTokens) / 1000)
    })
  }
}
```

### 3. Recovery Protocol for Stuck Agents

**Following the design from `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md`:**

```
┌─────────────────────────────────────────────────────────────┐
│                  Daemon Recovery Loop                        │
│                  (every 60s, parallel to completion loop)   │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                 STUCK DETECTION                      │   │
│  │                                                      │   │
│  │  For each active agent:                              │   │
│  │  1. Check last beads comment timestamp               │   │
│  │  2. Check session activity (SSE events)              │   │
│  │  3. If idle >10 min AND no "Phase: Complete"         │   │
│  │     → Mark as "potentially stuck"                    │   │
│  └────────────────────┬────────────────────────────────┘   │
│                       │                                     │
│                       ▼                                     │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                 TIER 1: AUTO-RESUME                  │   │
│  │                                                      │   │
│  │  • Rate-limited: 1 resume / hour / agent             │   │
│  │  • Non-destructive: just sends continuation prompt   │   │
│  │  • Prompt: "Check your SPAWN_CONTEXT.md and continue"│   │
│  │  • Track: resume_attempts++ for agent                │   │
│  └────────────────────┬────────────────────────────────┘   │
│                       │                                     │
│                       ▼ (if still stuck after 15 min)       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              TIER 2: SURFACE FOR ATTENTION           │   │
│  │                                                      │   │
│  │  • Add to "Needs Attention" in dashboard             │   │
│  │  • Send notification: "Agent {X} stuck for {Y} min"  │   │
│  │  • Log to events: stuck_agent_surfaced               │   │
│  └────────────────────┬────────────────────────────────┘   │
│                       │                                     │
│                       ▼ (human intervention)                │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              TIER 3: HUMAN DECISION                  │   │
│  │                                                      │   │
│  │  Options:                                            │   │
│  │  • orch resume <agent>   (try again)                 │   │
│  │  • orch abandon <agent>  (give up, preserve context) │   │
│  │  • orch respawn <agent>  (new session, fresh start)  │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

**Recovery Prompt Template:**

```markdown
## 🔄 Recovery Check

You may have become stuck or lost context. Please:

1. Re-read your SPAWN_CONTEXT.md: `cat {workspace}/SPAWN_CONTEXT.md`
2. Check your investigation file status
3. Report current phase: `bd comment {beads_id} "Phase: {phase} - {brief}"`
4. Continue work or report if blocked: `bd comment {beads_id} "BLOCKED: {reason}"`

If you're unable to proceed, describe what's blocking you.
```

**State Tracking (add to daemon):**

```go
type AgentHealthState struct {
    SessionID       string
    BeadsID         string
    LastActivity    time.Time
    LastPhaseChange time.Time
    ResumeAttempts  int
    LastResumeAt    time.Time
    Status          string // "active", "potentially_stuck", "stuck", "surfaced"
}
```

---

## Structured Uncertainty

**What's tested:**

- ✅ `injectCoachingMessage()` works with noReply:true pattern (verified: existing coaching.ts:697-710 uses this successfully)
- ✅ Worker detection via workspace path works (verified: coaching.ts:1006-1009 uses `.orch/workspace/` detection)
- ✅ Daemon has parallel loop architecture (verified: read daemon.go, completion.go show separate loops)
- ✅ Resume command exists and sends continuation prompt (verified: read resume.go:90-100)

**What's untested:**

- ⚠️ Token estimation accuracy (proposed estimateTokenUsage() function - algorithm not validated)
- ⚠️ 10-minute threshold for stuck detection (may need tuning based on real agent behavior)
- ⚠️ Context budget warning at 80% (threshold may be too early or too late)
- ⚠️ Tool failure rate of 3 as threshold (may need adjustment)
- ⚠️ Resume success rate for different failure modes (rate limits vs context exhaustion vs infinite loops)

**What would change this:**

- If token estimation is wildly inaccurate → need to integrate with OpenCode's actual token counting
- If 80% context warning is too late → agents run out before they can wrap up
- If 80% context warning is too early → agents ignore it as "not urgent"
- If stuck agents are rare (<5% of spawns) → recovery automation may not be worth the complexity
- If resume causes more problems than it solves (e.g., perpetuates bad state) → remove auto-resume tier

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Phased Self-Health Implementation** - Build incrementally: plugin metrics extension → spawn-time context → runtime injection → daemon recovery loop.

**Why this approach:**
- Leverages existing coaching.ts infrastructure (Finding 1) - minimal new code
- Uses proven noReply injection pattern (Finding 2) - no architectural risk
- Follows parallel loop pattern from daemon (Finding 3) - fits existing architecture
- Mirrors KB context pattern for spawn-time inclusion (Finding 4) - consistent approach
- Respects worker/orchestrator distinction (Finding 5) - appropriate signals per agent type

**Trade-offs accepted:**
- Token estimation will be approximate (no direct OpenCode integration initially)
- Some stuck agents won't auto-recover (context exhaustion, infinite loops)
- 10-15 minute detection latency before surfacing

**Implementation sequence:**
1. **Phase 1: Extend coaching.ts with worker health metrics** - Add tool_failure_rate, context_usage tracking for workers (currently skipped)
2. **Phase 2: Add health context to SPAWN_CONTEXT.md template** - Include baseline health visibility section
3. **Phase 3: Add real-time health injection** - Extend injectCoachingMessage with worker-specific signals
4. **Phase 4: Add daemon recovery loop** - Stuck detection + tiered recovery (resume → surface → human)

### Alternative Approaches Considered

**Option B: Centralized Health Service**
- **Pros:** Single source of truth, consistent across all agents
- **Cons:** New infrastructure, single point of failure, adds latency
- **When to use instead:** If plugin-based approach becomes unmaintainable at scale (>100 concurrent agents)

**Option C: Agent Self-Monitoring Only (no daemon recovery)**
- **Pros:** Simpler, no daemon changes, agents own their health
- **Cons:** Stuck agents can't self-recover (the problem this solves)
- **When to use instead:** If daemon recovery causes more problems than it solves

**Rationale for recommendation:** Option A builds on proven patterns (coaching.ts, daemon loops) and adds value incrementally. Each phase delivers standalone value and can be validated before proceeding.

---

### Implementation Details

**What to implement first:**

**Phase 1: Worker Health Metrics (highest priority)**
```typescript
// Add to SessionState in coaching.ts
interface WorkerHealthState {
  consecutiveToolFailures: number
  estimatedTokensUsed: number
  sessionStartTime: number
  lastPhaseUpdate: number
  lastCommitTime: number
}

// Extend tool.execute.after for workers (currently skipped)
if (isWorkerSession) {
  trackWorkerHealth(state, tool, success)
}
```

**Phase 2: Spawn-Time Health Context**
```go
// Add to pkg/spawn/context.go
func GenerateHealthContext(beadsID string) string {
    // Get prior session metrics from ~/.orch/coaching-metrics.jsonl
    // Return formatted health context section
}
```

**Phase 3: Runtime Health Injection**
```typescript
// New function in coaching.ts
async function injectHealthSignal(
  client: any,
  sessionId: string,
  signalType: "tool_friction" | "context_budget" | "time_check" | "commit_reminder",
  details: any
): Promise<void>
```

**Phase 4: Daemon Recovery Loop**
```go
// Add to pkg/daemon/health.go
func (d *Daemon) RunHealthLoop(ctx context.Context) {
    ticker := time.NewTicker(60 * time.Second)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            d.checkAgentHealth()
        }
    }
}
```

**Things to watch out for:**
- ⚠️ Token estimation is inherently approximate - don't present as exact
- ⚠️ Worker detection may have edge cases (test with various spawn modes)
- ⚠️ Resume can perpetuate bad state - rate limit strictly (1/hour/agent)
- ⚠️ Server restart invalidates sessions temporarily - delay recovery checks after restart
- ⚠️ Context injection frequency - too many injections = noise, too few = missed signals

**Areas needing further investigation:**
- OpenCode token counting API - could provide accurate context usage
- Phase change detection - currently requires parsing beads comments
- Commit detection for "commit gap" warning - need git status monitoring
- Dashboard "Needs Attention" integration - UI component needed

**Success criteria:**
- ✅ Agents receive context budget warnings before running out (>50% of near-exhaustion cases warned)
- ✅ Stuck agents surface in dashboard within 25 minutes of becoming stuck
- ✅ Auto-resume has >50% success rate for rate-limit-induced stalls
- ✅ Tool friction warnings reduce repeated failure loops (measurable in metrics)
- ✅ No increase in noise complaints from agents (coaching messages aren't overwhelming)

---

## References

**Files Examined:**
- `plugins/coaching.ts` - Main coaching plugin with behavioral pattern detection (lines 1-1459)
- `cmd/orch/serve_coaching.go` - HTTP endpoint for coaching metrics API
- `pkg/spawn/context.go` - SPAWN_CONTEXT.md template generation
- `pkg/spawn/kbcontext.go` - KB context query and formatting
- `pkg/daemon/daemon.go` - Daemon main loop architecture
- `pkg/daemon/completion.go` - Completion detection loop
- `web/src/lib/stores/coaching.ts` - Dashboard coaching data store
- `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md` - Prior recovery design

**Commands Run:**
```bash
# Find coaching plugin files
glob "**/coaching*.ts"

# Find related go files
grep "stuck|stalled|recovery|health" **/*.go

# Examine daemon structure
glob "**/daemon/*.go"
```

**External Documentation:**
- OpenCode Plugin API - noReply pattern for context injection

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-15-inv-design-stuck-agent-recovery-mechanism.md` - Established tiered recovery with advisory-first principle
- **Investigation:** `.kb/investigations/2026-01-08-inv-investigate-opencode-plugin-capabilities-ecosystem.md` - Documented noReply injection pattern
- **Model:** `.kb/models/completion-verification.md` - Agent verification architecture
- **Model:** `.kb/models/agent-lifecycle-state-model.md` - Agent state model
- **Guide:** `.kb/guides/daemon.md` - Daemon operation guide

---

## Investigation History

**2026-01-17 10:00:** Investigation started
- Initial question: Design Agent Self-Health Context Injection system - how coaching metrics transform into real-time agent signals, Pain-as-Signal mechanism, and recovery protocol
- Context: Spawned as architect session from orchestrator to design the self-health system

**2026-01-17 10:15:** Context gathering complete
- Read coaching.ts: found 8 metric types already implemented
- Read serve_coaching.go: found HTTP endpoint for metrics
- Read stuck-agent-recovery investigation: found tiered recovery design
- Read spawn context files: found KB context injection pattern

**2026-01-17 10:45:** Design synthesis complete
- Designed 3-layer architecture: metrics → signals → injection + recovery
- Identified 4 new worker health metrics to add
- Designed Pain-as-Signal tool-layer injection mechanism
- Integrated recovery protocol from prior investigation

**2026-01-17 11:00:** Investigation completed
- Status: Complete
- Key outcome: Three-layer self-health architecture with phased implementation: extend coaching.ts → spawn-time context → runtime injection → daemon recovery loop
