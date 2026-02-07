# Design: Orchestrator Coaching Plugin

**Date:** 2026-01-10
**Status:** Proposed
**Owner:** og-feat-orchestrator-coaching-plugin-10jan-2249

---

## Problem Statement

Orchestrators need quantified behavioral metrics to detect Level 1→2 patterns (option theater, missing strategic reasoning). Current system provides guidance but no feedback loop showing whether behaviors are improving.

**Hypothesis:** Do quantified metrics drive orchestrator behavior change?

---

## Goals

1. **Minimal prototype** - Simple keyword/pattern detection, not sophisticated NLP
2. **Dashboard metrics view** - Real-time visibility into orchestrator behavioral patterns  
3. **Coaching reports** - Actionable feedback on detected patterns
4. **Test hypothesis** - Measure if metrics drive behavior change

---

## Constraints

- OpenCode plugins only see tool calls, not free-text LLM responses
- Must work with existing plugin system (no OpenCode core changes)
- Dashboard must work at 666px width (half MacBook Pro screen)
- Should not block orchestrator workflow (warnings only, not gates)

---

## Design

### Architecture

```
┌─────────────────┐
│ OpenCode Plugin │ (TypeScript)
│ coaching.ts     │
└────────┬────────┘
         │ detects tool patterns
         ├─ context-gathering ratio
         ├─ action/read balance  
         ├─ tool sequence patterns
         │
         ▼
┌─────────────────────────┐
│ Metrics Storage         │
│ ~/.orch/coaching-       │
│         metrics.jsonl   │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│ API Endpoint            │ (Go)
│ /api/coaching           │
│ cmd/orch/serve_         │
│        coaching.go      │
└────────┬────────────────┘
         │ returns JSON
         ▼
┌─────────────────────────┐
│ Dashboard View          │ (Svelte)
│ Coaching Metrics Card   │
│ web/src/routes/         │
│        +page.svelte     │
└─────────────────────────┘
```

### Components

#### 1. OpenCode Plugin (`plugins/coaching.ts`)

**Purpose:** Detect behavioral patterns from tool usage

**Tracked Metrics:**
- `context_checks` - Count of `kb context` calls via Bash tool
- `spawn_count` - Count of spawns (detected via `orch spawn` in Bash)
- `action_count` - Count of Edit/Write/Bash tools (excluding reads)
- `read_count` - Count of Read/Grep/Glob tools
- `tool_sequences` - Sequences of same tool type (e.g., 5 Reads in a row)

**Detection Logic:**
```typescript
- Hook: tool.execute.after
- Track per session (sessionID):
  - Last 10 tool calls (sliding window)
  - Cumulative counters for metrics
  - Detect sequences: 3+ same tool type → analysis paralysis signal
- Periodic metrics flush: Every 10 tool calls, write to JSONL
```

**Output Format (JSONL):**
```json
{
  "timestamp": "2026-01-10T12:34:56Z",
  "session_id": "ses_abc",
  "metric_type": "context_ratio",
  "value": 0.75,
  "details": {
    "context_checks": 3,
    "spawns": 4
  }
}
```

#### 2. Metrics Storage (`~/.orch/coaching-metrics.jsonl`)

**Format:** JSONL (one JSON object per line)
**Rotation:** Keep last 1000 lines (prune older on plugin init)
**Schema:** See Output Format above

#### 3. API Endpoint (`cmd/orch/serve_coaching.go`)

**Route:** `GET /api/coaching`

**Response:**
```json
{
  "session": {
    "session_id": "ses_abc",
    "started": "2026-01-10T12:00:00Z",
    "duration_minutes": 45
  },
  "metrics": {
    "context_ratio": {
      "value": 0.75,
      "label": "Context checks per spawn",
      "status": "good" // good/warning/poor
    },
    "action_ratio": {
      "value": 0.6,
      "label": "Actions per reads",
      "status": "warning"
    },
    "analysis_paralysis": {
      "sequences": 2,
      "label": "Tool repetition sequences",
      "status": "warning"
    }
  },
  "coaching": [
    "✅ Good context-gathering ratio (0.75)",
    "⚠️ Low action ratio - consider more decisive action",
    "⚠️ 2 analysis paralysis sequences detected"
  ]
}
```

**Implementation:**
- Read last 100 lines from JSONL
- Aggregate by current session_id
- Calculate ratios and thresholds
- Generate coaching messages

#### 4. Dashboard View (Svelte Component)

**Location:** Add to `web/src/routes/+page.svelte`

**UI Design:**
```
┌─────────────────────────────────┐
│ 📊 Orchestrator Coaching        │
├─────────────────────────────────┤
│ Session: 45 min                 │
│                                 │
│ Context Ratio:  ✅ 0.75        │
│ Action Ratio:   ⚠️  0.60       │
│ Analysis Loops: ⚠️  2          │
│                                 │
│ Coaching:                       │
│ • Good context gathering        │
│ • Consider more decisive action │
│ • 2 analysis paralysis detected │
└─────────────────────────────────┘
```

**Implementation:**
- Fetch `/api/coaching` on component mount
- Poll every 30 seconds for updates
- Show metrics with color-coded status
- Display coaching messages as bullet list

---

## Behavioral Proxies for Level 1→2 Detection

Since we can't analyze free-text responses, we use tool patterns as proxies:

| Level 1→2 Pattern | Tool-Based Proxy |
|-------------------|------------------|
| **Option theater** | Low action ratio (reads >> actions) |
| **Missing strategic reasoning** | Low context-gathering ratio (spawns without kb context) |
| **Analysis paralysis** | Tool repetition sequences (5+ reads without action) |

---

## Testing Strategy

### Unit Tests
- Plugin metrics calculation (context ratio, action ratio)
- JSONL read/write operations
- API response format validation

### Integration Tests
- End-to-end: tool call → JSONL → API → dashboard
- Dashboard display with mock data

### Manual Validation
- Run orchestrator session with plugin enabled
- Verify metrics appear in dashboard
- Check coaching messages are actionable

---

## Rollout Plan

1. **Phase 1:** Plugin + metrics storage (no dashboard yet)
   - Deploy `coaching.ts` to `~/.config/opencode/plugin/`
   - Verify JSONL logging works
   - Validate metrics calculation

2. **Phase 2:** API endpoint
   - Add `serve_coaching.go`
   - Test endpoint returns correct data
   - Manual curl testing

3. **Phase 3:** Dashboard view
   - Add coaching card to dashboard
   - Verify 666px width constraint
   - Test with real orchestrator session

4. **Phase 4:** Hypothesis testing
   - Run for 1 week with metrics visible
   - Track if behavior changes (pre/post comparison)
   - Document findings

---

## Trade-offs

### Chosen: Behavioral Proxies
**Pro:** Works within OpenCode plugin constraints
**Con:** Indirect measurement (tool patterns vs actual text analysis)
**Mitigation:** Start with simple metrics, iterate based on feedback

### Chosen: JSONL Storage
**Pro:** Simple, follows action-log pattern, easy to inspect
**Con:** No structured query capabilities
**Mitigation:** Keep last 1000 lines, sufficient for prototype

### Chosen: Warnings Only (Not Gates)
**Pro:** Non-blocking, won't disrupt workflow
**Con:** Can be ignored
**Mitigation:** Prominent dashboard visibility creates social pressure

---

## Open Questions

1. **Thresholds for status (good/warning/poor)?**
   - Proposed: context_ratio >0.7 = good, 0.4-0.7 = warning, <0.4 = poor
   - Proposed: action_ratio >0.5 = good, 0.3-0.5 = warning, <0.3 = poor
   - Proposed: analysis_paralysis 0 = good, 1-2 = warning, 3+ = poor

2. **How often to flush metrics to JSONL?**
   - Proposed: Every 10 tool calls OR every 5 minutes

3. **Should coaching messages be injected into session?**
   - Proposed: No for prototype, only dashboard display
   - Future: Could use `client.session.prompt` like evidence-hierarchy

---

## Success Criteria

- [ ] Plugin tracks metrics and writes to JSONL
- [ ] API endpoint returns aggregated metrics
- [ ] Dashboard displays coaching card at 666px width
- [ ] Coaching messages are actionable (not just numbers)
- [ ] Hypothesis test plan documented

---

## References

- Investigation: `.kb/investigations/2026-01-10-inv-orchestrator-coaching-plugin-prototype.md`
- Beads issue: `orch-go-zyuik`
- Related plugins: `plugins/evidence-hierarchy.ts`, `plugins/action-log.ts`
- Constraint: Dashboard width (666px) from kb context
