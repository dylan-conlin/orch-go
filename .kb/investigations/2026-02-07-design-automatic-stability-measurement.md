## Summary (D.E.K.N.)

**Delta:** Designed automatic stability measurement for Phase 3 reliability tracking. System uses a dedicated `~/.orch/stability.jsonl` log with periodic health snapshots (from doctor daemon) and automatic manual-recovery detection (service health transitions not caused by daemon).

**Evidence:** Existing infrastructure covers 80% of needs — events.jsonl tracks service crashes/restarts (with `auto_restart` flag), agent abandonments; doctor daemon already polls health every 30s with self-healing. Gap: no periodic snapshots, no manual-recovery detection, no streak computation.

**Knowledge:** The doctor daemon's health transition tracking + `auto_restart` flag on service events provides the foundation for distinguishing automatic vs manual recovery. Key insight: we don't need to detect manual shell commands directly — we detect their *effects* (service transitions not caused by daemon).

**Next:** Implement as specified. Files: `pkg/stability/stability.go`, `cmd/orch/stability_cmd.go`, modifications to `cmd/orch/doctor_watch.go`.

**Authority:** implementation — Extends existing doctor/events infrastructure within established patterns, no architectural changes.

---

# Investigation: Design Automatic Stability Measurement for Phase 3

**Question:** How should we automatically measure system stability for the Phase 3 "1 week without manual recovery" target?

**Started:** 2026-02-07
**Updated:** 2026-02-07
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** Implementation
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-02-07-inv-system-reliability-crisis-diagnosis-and-fix.md` | extends (Phase 3 measurement) | Yes — read Phase 3 requirements | None |

---

## Findings

### Finding 1: events.jsonl Already Tracks Most Intervention Signals

**Evidence:** The events logger (`pkg/events/logger.go`) already emits:
- `service.crashed` — service went down (line 29)
- `service.restarted` with `auto_restart` boolean (line 32, 328) — distinguishes daemon vs manual restarts
- `agent.abandoned` — agent had to be killed (line 38)
- `service.started` — service came up (line 34)

The `auto_restart` flag is the key differentiator. When the doctor daemon restarts a service, `auto_restart: true`. When someone restarts it manually, `auto_restart: false`.

**Source:** `pkg/events/logger.go:328`, `pkg/service/event_adapter.go:45`

**Significance:** We don't need to build detection from scratch. The infrastructure for distinguishing automatic vs manual recovery already exists in the event system.

### Finding 2: Doctor Daemon Has Health Transition Detection

**Evidence:** The doctor daemon (`cmd/orch/doctor_watch.go`) already:
- Polls health every 30 seconds (line 256)
- Tracks `previousHealth` map for state transitions (line 259, 317-319)
- Performs self-healing: kills orphaned vite (line 283), long-running bd (line 286), restarts crashed services (line 289)
- Logs interventions to `~/.orch/doctor.log` (line 216-237)

The daemon knows what it fixed each cycle. Comparing health transitions against daemon actions reveals manual recoveries.

**Source:** `cmd/orch/doctor_watch.go:278-329`

**Significance:** The daemon cycle is the natural place to record stability snapshots and detect manual recoveries, since it already has all the needed state.

### Finding 3: Stats Command Shows events.jsonl Scanning Pattern

**Evidence:** `cmd/orch/stats_cmd.go` scans events.jsonl line-by-line, filtering by timestamp. This is the established pattern for deriving metrics from the event log.

**Source:** `cmd/orch/stats_cmd.go:80-100`

**Significance:** The stability command should follow the same pattern. However, adding periodic snapshots (every 5min = 2016/week) to events.jsonl would pollute the general event log. A separate stability.jsonl is cleaner.

---

## Synthesis

**Key Insights:**

1. **Derive, don't duplicate** — Most intervention signals already exist in events.jsonl. The stability system should derive streak data from existing events plus minimal new data (snapshots).

2. **Detect effects, not actions** — We can't detect every manual shell command (kill, restart). But we can detect their effects: a service that was unhealthy is now healthy, and the daemon didn't fix it. This eliminates the need for manual recording.

3. **Separate log for snapshots** — Periodic health snapshots (every 5min) would add ~2000 entries/week. A dedicated `~/.orch/stability.jsonl` keeps stability data clean and fast to query without polluting the general event log.

**Answer to Investigation Question:**

Record health snapshots and interventions to `~/.orch/stability.jsonl`. Hook into the existing doctor daemon loop to record snapshots every 5 minutes and detect manual recoveries via health state transitions. A new `orch stability` command reads the file and computes the clean-session streak.

---

## Structured Uncertainty

**What's tested:**

- ✅ events.jsonl has `auto_restart` flag for service restarts (verified: read `pkg/events/logger.go:328`, `pkg/service/event_adapter.go:45`)
- ✅ Doctor daemon tracks health transitions via `previousHealth` map (verified: read `doctor_watch.go:259,317`)
- ✅ Doctor daemon knows what it fixed each cycle via return values from healing functions (verified: read `doctor_watch.go:283-289`)

**What's untested:**

- ⚠️ Manual recovery detection accuracy — need to verify during 1-week trial
- ⚠️ Snapshot frequency (5min) adequacy — may need adjustment
- ⚠️ Whether `orch doctor --fix` and `orch-dashboard restart` reliably emit events

**What would change this:**

- If doctor daemon doesn't run continuously (currently started via launchd), snapshots would have gaps
- If manual recoveries happen through paths the daemon can't observe (e.g., rebuilding SQLite DB), false negatives occur

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Implement stability measurement with separate JSONL + doctor daemon integration | implementation | Extends existing patterns, no new architectural decisions, reversible |

### Recommended Approach ⭐

**Stability JSONL + Doctor Daemon Integration** — Record health snapshots and manual recovery detections to `~/.orch/stability.jsonl` via the existing doctor daemon loop.

**Why this approach:**
- Reuses existing doctor daemon infrastructure (health checks, transition detection, self-healing)
- Fully automatic — no manual recording needed
- Amnesia-proof — next Claude runs `orch stability` and gets the full picture
- Minimal new code — ~200 lines in `pkg/stability/`, ~100 lines in CLI, ~30 lines doctor integration

**Trade-offs accepted:**
- Only detects manual recoveries the daemon can observe (service health transitions)
- Requires doctor daemon to be running continuously for complete data

**Implementation sequence:**

1. **`pkg/stability/stability.go`** — Core types and logic
   - `Recorder` for writing snapshots/interventions to stability.jsonl
   - `ComputeReport()` for streak calculation from stability.jsonl

2. **`cmd/orch/stability_cmd.go`** — CLI command
   - `orch stability` — human-readable report with streak, progress bar, recent interventions
   - `orch stability --json` — machine-readable output

3. **Modify `cmd/orch/doctor_watch.go`** — Hook into daemon loop
   - Record snapshot every 5 minutes (not every 30s poll)
   - Detect manual recoveries: unhealthy→healthy transitions not caused by daemon
   - Record intervention when manual recovery detected

### Alternative Approaches Considered

**Option B: Derive entirely from events.jsonl**
- **Pros:** No new file, single source of truth
- **Cons:** No periodic snapshots (can't compute health %), events.jsonl grows faster, slower scanning
- **When to use instead:** If storage is constrained

**Option C: Separate stability daemon process**
- **Pros:** Decoupled from doctor daemon, independent lifecycle
- **Cons:** Adds another process (we just eliminated one!), duplicates health checks
- **When to use instead:** Never (violates the "reduce processes" lesson from reliability crisis)

---

### Implementation Details

**Data format (`~/.orch/stability.jsonl`):**

```json
{"type":"snapshot","ts":1707300000,"healthy":true,"services":{"OpenCode":true,"orch serve":true,"Overmind":true}}
{"type":"intervention","ts":1707300600,"source":"manual_recovery","detail":"OpenCode recovered without daemon action","services":["OpenCode"]}
{"type":"intervention","ts":1707301200,"source":"agent_abandoned","detail":"orch-go-abc12 abandoned","beads_id":"orch-go-abc12"}
```

**Streak computation:**
- Scan stability.jsonl backward from now
- Find most recent intervention entry
- Streak = now - intervention.timestamp
- If no interventions found, streak = now - first snapshot timestamp

**Intervention sources (streak-breaking events):**

| Source | Trigger | Automatic Detection |
|--------|---------|-------------------|
| `manual_recovery` | Service health transition not caused by daemon | Doctor daemon health transition + daemon action comparison |
| `agent_abandoned` | `orch abandon` invoked | Hook in abandon command |
| `doctor_fix` | `orch doctor --fix` invoked | Hook in doctor --fix path |

**CLI output format:**
```
Stability Report (Phase 3 Reliability Tracking)
================================================

Current streak:     3d 14h 22m (target: 7d)
Phase 3 progress:   ████████████░░░░░░░░ 51%

Last 7 days: 2 interventions
  2026-02-07 14:22  manual_recovery   OpenCode recovered without daemon action
  2026-02-06 09:15  agent_abandoned   orch-go-abc12 abandoned

Health snapshots:   2016 recorded
  Healthy:          1998/2016 (99.1%)
```

**File targets:**
- Create: `pkg/stability/stability.go` (~200 lines)
- Create: `pkg/stability/stability_test.go` (~150 lines)
- Create: `cmd/orch/stability_cmd.go` (~120 lines)
- Modify: `cmd/orch/doctor_watch.go` — add stability recording to daemon cycle (~30 lines)
- Modify: `cmd/orch/abandon_cmd.go` — emit stability intervention on abandon (~5 lines)

**Acceptance criteria:**
- ✅ `orch stability` reports current streak with no manual setup
- ✅ Doctor daemon records snapshots every 5 minutes to stability.jsonl
- ✅ Manual recoveries (service health transitions not from daemon) are detected and recorded
- ✅ `orch abandon` records an intervention
- ✅ Streak resets when intervention is recorded
- ✅ `go build ./cmd/orch/` and `go vet ./cmd/orch/` pass

**Things to watch out for:**
- ⚠️ Doctor daemon must be running for snapshots — if it's not, `orch stability` should report "No data (doctor daemon not running?)"
- ⚠️ First 5 minutes after daemon start will have no snapshots — use daemon start time as streak start if no prior data
- ⚠️ stability.jsonl should be rotated eventually (not MVP) — will grow ~400KB/week

---

## References

**Files Examined:**
- `cmd/orch/doctor.go` — Doctor command structure, ServiceStatus type
- `cmd/orch/doctor_watch.go` — Daemon loop, health transitions, self-healing actions
- `cmd/orch/doctor_liveness.go` — Individual health checks for each service
- `pkg/events/logger.go` — Event types and logging patterns
- `pkg/service/event_adapter.go` — `auto_restart` flag on service events
- `cmd/orch/abandon_cmd.go` — Abandon workflow, event emission
- `cmd/orch/stats_cmd.go` — Pattern for scanning events.jsonl
- `.kb/investigations/2026-02-07-inv-system-reliability-crisis-diagnosis-and-fix.md` — Phase 3 requirements

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-02-07-inv-system-reliability-crisis-diagnosis-and-fix.md` — Defines Phase 3 success metric
