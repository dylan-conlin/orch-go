## Summary (D.E.K.N.)

**Delta:** daemon-status.json + capacity-cache.json already provide all data needed for a sketchybar widget — no new Go code or commands required. The existing orch.lua widget is broken (references removed agent-registry.json) and needs a rewrite.

**Evidence:** daemon-status.json contains capacity, ready_count, last_poll, verification, phase_timeout, question_detection. capacity-cache.json has per-account 5h/7d usage. Existing orch_status.sh reads from agent-registry.json which no longer exists (removed per No Local Agent State constraint).

**Knowledge:** The optimal architecture is a hybrid: fast file reads (daemon-status.json) for bar display updated every 5s, expensive `orch status --json` call only on popup click for agent detail. health_signals.go already computes traffic-light thresholds for 6 daemon health dimensions.

**Next:** Implement the rewrite — new orch_status.sh reading daemon files, updated orch.lua with health-colored display and multi-section popup.

**Authority:** implementation - Uses existing data files and sketchybar patterns, no architectural changes to orch-go

---

# Investigation: Design Sketchybar Widget for Live Daemon Observability

**Question:** How should a sketchybar widget surface live daemon metrics (active agents, ready queue, comprehension pending, account usage) as a cross-project global status indicator?

**Started:** 2026-03-24
**Updated:** 2026-03-24
**Owner:** architect agent
**Phase:** Complete
**Next Step:** Implementation (see decomposition)
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Existing orch.lua widget | extends | Yes — read current code | Current widget references removed agent-registry.json |

---

## Findings

### Finding 1: daemon-status.json is the ideal primary data source

**Evidence:** The daemon writes `~/.orch/daemon-status.json` atomically on every poll cycle (~30s). Current contents:

```json
{
  "pid": 24339,
  "capacity": { "max": 5, "active": 2, "available": 3 },
  "last_poll": "2026-03-24T11:40:05Z",
  "last_spawn": "2026-03-24T11:38:26Z",
  "last_completion": "2026-03-24T11:03:16Z",
  "ready_count": 4,
  "status": "running",
  "verification": { "is_paused": false, "completions_since_verification": 0, "threshold": 3, "remaining_before_pause": 3 },
  "phase_timeout": { "unresponsive_count": N },
  "question_detection": { "question_count": N },
  "beads_health": { ... }
}
```

Reading a JSON file takes <1ms vs `orch status --json` taking 2-5s (makes HTTP calls to OpenCode, shells to bd CLI).

**Source:** `pkg/daemon/status.go:15-65` (DaemonStatus struct), `pkg/daemon/status.go:108-138` (WriteStatusFile), `~/.orch/daemon-status.json` (live data)

**Significance:** The bar display (updated every 5s) should read this file, NOT call `orch status`. This is 1000x faster and the daemon already computes all health metrics.

---

### Finding 2: Health signal thresholds are already codified

**Evidence:** `pkg/daemon/health_signals.go` defines `ComputeDaemonHealth()` with 6 signals, each with green/yellow/red thresholds:

| Signal | Green | Yellow | Red |
|--------|-------|--------|-----|
| Daemon Liveness | <2min since poll | 2-10min | >10min |
| Capacity | <80% slots used | >=80% | Saturated + queued work |
| Queue Depth | <20 ready | 20-50 | >50 |
| Evidence Check | >2 remaining | 1-2 remaining | Paused |
| Unresponsive | 0 agents | 1 agent | >1 agents |
| Questions | 0 waiting | 1-2 waiting | >2 waiting |

These thresholds can be directly reused in the event provider script (computed in bash from the JSON).

**Source:** `pkg/daemon/health_signals.go:40-176`

**Significance:** Widget color should be the worst (highest-severity) signal across all 6 dimensions — matching exactly what the daemon's health system already computes.

---

### Finding 3: Existing orch.lua widget is broken but architecturally sound

**Evidence:** The existing widget (`~/.config/sketchybar/items/widgets/orch.lua`) has correct architecture:
- Event provider script fires custom events on an interval
- Lua widget subscribes to events and updates display
- Click toggles popup with detail view

But the implementation is broken:
1. Event provider reads `~/.orch/agent-registry.json` (removed per No Local Agent State constraint)
2. Lua file references old orch path: `/Users/dylanconlin/Documents/personal/aider/aider-env/bin/orch`
3. Popup calls `orch status --global --format json` (flag doesn't exist — actual flag is `--json`)
4. No daemon health, capacity, comprehension, or account usage displayed

**Source:** `~/.config/sketchybar/items/widgets/orch.lua`, `~/.config/sketchybar/helpers/event_providers/orch_status/orch_status.sh`

**Significance:** The rewrite can follow the same architecture pattern but with correct data sources and richer display.

---

### Finding 4: capacity-cache.json provides account usage data

**Evidence:** The daemon polls account capacity every 5 minutes and caches to `~/.orch/capacity-cache.json`:

```json
{
  "fetched_at": "2026-03-24T11:35:56Z",
  "accounts": [{
    "name": "work",
    "email": "dylan.conlin@sendcutsend.com",
    "capacity": {
      "FiveHourUsed": 62, "FiveHourRemaining": 38,
      "SevenDayUsed": 43, "SevenDayRemaining": 57,
      "Email": "..."
    }
  }]
}
```

**Source:** `pkg/account/capacity_file_cache.go:25-31` (path), `~/.orch/capacity-cache.json` (live data)

**Significance:** Account usage is available without any API call. The widget can show account capacity in the popup by reading this file.

---

### Finding 5: orch status --json provides comprehensive agent detail for popup

**Evidence:** `orch status --json` (status_cmd.go) returns a `StatusOutput` struct with:
- `swarm`: active, processing, idle, phantom, queued, completed counts
- `accounts[]`: name, email, used_percent, reset_time, is_active
- `agents[]`: session_id, beads_id, model, skill, phase, project, status flags, tokens, context_risk
- `review_queue`: ready count
- `session_metrics`: time_in_session, spawn_count, goal
- `infrastructure`: service health

This is expensive (2-5s) but provides everything needed for the click-to-expand popup view.

**Source:** `cmd/orch/status_cmd.go:146-154` (StatusOutput), `cmd/orch/status_cmd.go:432-438` (JSON output)

**Significance:** Use for popup only (on-demand when clicked), NOT for bar display polling.

---

## Synthesis

**Key Insights:**

1. **Hybrid data source is optimal** — daemon-status.json for the bar (0ms, every 5s), orch status --json for popup (2-5s, on-demand). This gives responsive bar updates without burning CPU polling expensive commands.

2. **No new Go code needed** — The daemon already writes all required metrics. The health signal thresholds in health_signals.go can be replicated in the bash event provider script. This is purely a sketchybar widget rewrite.

3. **Worst-signal-wins coloring** — The bar icon color should reflect the worst health signal across all 6 dimensions. This gives an instant "is my daemon healthy?" answer without reading any text.

**Answer to Investigation Question:**

The widget should use a two-tier data architecture:
- **Bar display (fast path):** Read `~/.orch/daemon-status.json` every 5s. Show `active/max` agents, colored by worst health signal (green/yellow/red). This is effectively free.
- **Popup detail (on-demand):** On click, run `orch status --json` once and render sections: health signals, account usage, review queue, agent list. This is expensive but acceptable for user-initiated interaction.

The existing `orch.lua` widget architecture (event provider + lua subscriber) is sound — the event provider script just needs to read the right files and pass richer environment variables.

---

## Structured Uncertainty

**What's tested:**

- daemon-status.json exists and contains capacity, ready_count, verification, phase_timeout, question_detection (verified: read live file)
- capacity-cache.json exists and contains per-account 5h/7d usage (verified: read live file)
- orch status --json works and returns full StatusOutput (verified: read source code, saw --json flag in cobra definition)
- Existing orch.lua uses event provider pattern that works with sketchybar (verified: read working render.lua widget)

**What's untested:**

- Actual latency of orch status --json in popup context (estimated 2-5s from code analysis)
- Whether 5s poll interval feels responsive enough for bar updates
- Whether sketchybar popup can handle 10+ items without visual overflow

**What would change this:**

- If daemon-status.json stops being updated (daemon not running), widget needs a fallback (show "stopped" state)
- If orch status --json takes >5s, popup may feel sluggish — could need a loading indicator or pre-cached approach

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Rewrite event provider to read daemon-status.json | implementation | Drop-in replacement using existing data, no new APIs |
| Rewrite orch.lua with health-colored display | implementation | UI change within existing widget architecture |
| Add comprehension:pending to daemon-status.json | architectural | Modifies daemon's status file schema (cross-component) |

### Recommended Approach: Hybrid File-Read + On-Demand CLI

**Rewrite the orch.lua widget and orch_status.sh event provider to use daemon-status.json as primary data source, with orch status --json for popup detail.**

**Why this approach:**
- 0ms file reads for bar display vs 2-5s CLI calls
- No new Go code — daemon already writes everything
- health_signals.go thresholds are directly reusable
- Matches existing sketchybar plugin pattern (render.lua uses same architecture)

**Trade-offs accepted:**
- Comprehension:pending count is NOT in daemon-status.json — would need a schema addition or a separate bd CLI call (~500ms)
- Bar display shows daemon's view of active agents (from pool tracker), not orch status's cross-source discovery
- Popup data is up to 5s stale due to orch status --json latency

**Implementation sequence:**

1. **Rewrite `orch_status.sh`** — Read daemon-status.json + capacity-cache.json with jq, compute worst health signal, fire sketchybar event with rich env vars (ACTIVE, MAX, AVAILABLE, READY, STATUS, HEALTH_COLOR, ACCOUNT_USAGE, REVIEW_READY, UNRESPONSIVE, QUESTIONS, VERIFICATION_REMAINING)

2. **Rewrite `orch.lua` bar display** — Show `active/max` with worst-signal color. Format: `[icon] 2/5` in green, yellow, or red. Add secondary indicators for attention states (review ready, unresponsive, questions).

3. **Rewrite `orch.lua` popup** — Multi-section popup on click:
   - Section 1: Daemon Health (6 traffic-light signals)
   - Section 2: Account Usage (per-account 5h% / 7d%)
   - Section 3: Active Agents (from orch status --json, clickable to tmux)

### Alternative Approaches Considered

**Option B: Poll `orch status --json` for everything**
- **Pros:** Single data source, most complete data
- **Cons:** 2-5s per poll, wastes CPU/network, defeats purpose of status bar (should be instant)
- **When to use instead:** If daemon is not running and status file is stale

**Option C: Add daemon HTTP endpoint (socket/port)**
- **Pros:** Real-time push updates possible (SSE)
- **Cons:** Over-engineering — daemon already writes a file, no need for IPC
- **When to use instead:** If multiple consumers need real-time daemon state (unlikely)

**Rationale for recommendation:** File reads are 1000x faster than CLI calls. The daemon already writes comprehensive status. The popup's on-demand CLI call is acceptable because it's user-initiated and infrequent.

---

### Implementation Details

**What to implement first:**
- `orch_status.sh` rewrite (data provider is the foundation)
- Then `orch.lua` bar display (core visibility)
- Then `orch.lua` popup (detail view)

**Things to watch out for:**
- When daemon is not running, daemon-status.json may be stale or missing — event provider must detect this (check PID liveness or file age) and show "stopped" state
- jq must be available on PATH (standard on macOS with Homebrew, but verify)
- Sketchybar popup items are dynamically added/removed — use consistent naming pattern to avoid leaks

**Areas needing further investigation:**
- Whether to add `comprehension_pending` field to DaemonStatus (requires Go code change in daemon periodic tasks)
- Whether to show recent daemon decisions in popup (would require tailing daemon.log, adding complexity)

**Success criteria:**
- Bar shows accurate active/max count with health-appropriate color
- Bar turns yellow/red when daemon is stalled, verification paused, or agents unresponsive
- Click popup shows daemon health, account usage, and agent list
- Widget responds within <100ms for bar updates
- Widget gracefully handles daemon-not-running state

---

## Appendix: Proposed Widget Layout

### Bar Display (always visible)

```
[robot-icon] 2/5          -- green: all healthy
[robot-icon] 4/5          -- yellow: capacity >80% or 1 unresponsive
[robot-icon] 5/5 Q:12     -- red: saturated + queued, show queue depth
[robot-icon] --           -- grey: daemon not running
```

Color encoding (worst-signal-wins):
- Green (0xff50fa7b): all 6 health signals green
- Yellow (0xfff1fa8c): any signal yellow, none red
- Orange (0xffffb86c): verification paused (action needed)
- Red (0xffff5555): any signal red (daemon stalled, >1 unresponsive, etc.)
- Grey (0xff6272a4): daemon not running

### Popup (on click)

```
+----------------------------+
| DAEMON HEALTH              |
|  Liveness     polling ok   |  (green dot)
|  Capacity     2/5 slots    |  (green dot)
|  Queue        4 ready      |  (green dot)
|  Evidence     3 remaining  |  (green dot)
|  Unresponsive 0            |  (green dot)
|  Questions    0            |  (green dot)
+----------------------------+
| ACCOUNTS                   |
|  work   5h:62% 7d:43%     |
|  personal  5h:8% 7d:15%   |
+----------------------------+
| AGENTS (2 active)          |
|  [impl] orch-go-7c4s1     |  (blue - working)
|  [arch] orch-go-abc12      |  (blue - working)
|  [done] orch-go-xyz99      |  (green - complete)
+----------------------------+
```

---

## References

**Files Examined:**
- `cmd/orch/status_cmd.go` — Status command with --json flag, StatusOutput struct, SwarmStatus, AgentInfo, AccountUsage
- `cmd/orch/status_format.go` — Terminal formatting, printSwarmStatus, color logic
- `cmd/orch/status_infra.go` — Infrastructure health checking, DaemonStatus (CLI-side)
- `pkg/daemon/status.go` — DaemonStatus struct, WriteStatusFile, ReadStatusFile, StatusFilePath
- `pkg/daemon/health_signals.go` — ComputeDaemonHealth, 6 health signals with traffic-light thresholds
- `pkg/daemon/comprehension_queue.go` — ComprehensionQuerier, CheckComprehensionThrottle
- `pkg/daemon/log.go` — DaemonLogger, DaemonLogPath (~/.orch/daemon.log)
- `pkg/account/capacity_file_cache.go` — CapacityFileCache, DefaultCapacityFileCachePath
- `~/.config/sketchybar/items/widgets/orch.lua` — Existing (broken) orch widget
- `~/.config/sketchybar/helpers/event_providers/orch_status/orch_status.sh` — Existing event provider
- `~/.config/sketchybar/items/widgets/render.lua` — Reference widget for pattern
- `~/.config/sketchybar/colors.lua` — Dracula color palette

**Commands Run:**
```bash
# Read live daemon status
cat ~/.orch/daemon-status.json

# Read live capacity cache
cat ~/.orch/capacity-cache.json
```

---

## Investigation History

**2026-03-24 11:40:** Investigation started
- Initial question: How to design a sketchybar widget for live daemon observability
- Context: Dylan wants cross-project agent status at a glance in the menu bar

**2026-03-24 11:45:** Found existing orch.lua widget — broken but architecturally sound
- References removed agent-registry.json, old orch path
- Architecture pattern (event provider + lua subscriber) is correct

**2026-03-24 11:50:** Discovered daemon-status.json as ideal data source
- Written atomically every poll cycle with capacity, health metrics, verification state
- 0ms file read vs 2-5s CLI call

**2026-03-24 12:00:** Investigation completed
- Status: Complete
- Key outcome: Hybrid architecture — daemon-status.json for bar, orch status --json for popup. No new Go code needed.
