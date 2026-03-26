## Summary (D.E.K.N.)

**Delta:** The three pending widget questions (display format, comprehension data, ownership) are resolved by one framing: the widget is a passive status reflector, not a system component — this resolves all three forks toward minimal coupling and maximum glanceability.

**Evidence:** Widget already implements active/max with worst-signal color (orch.lua). Daemon already computes comprehension count for spawn throttling (compliance.go:56-57) but discards it — adding one field to daemon-status.json costs zero at the widget layer. Integration test (sketchybar_integration_test.go) already ensures bash/Go health parity without requiring repo ownership of widget files.

**Knowledge:** The unifying insight is ownership classification: the widget is a personal display layer (like terminal theme or font choice), not orchestration infrastructure. This means: optimize for glanceability (active/max with color wins), add signal only when the daemon already computes it (comprehension count: yes), and don't import personal config into the project repo (stay in ~/.config/sketchybar/).

**Next:** One implementation issue to add ComprehensionPending field to DaemonStatus struct and populate it during status file write.

**Authority:** architectural — Adding a field to daemon-status.json is a cross-component schema change, but narrow and additive. Widget ownership and display format are implementation (already decided by current code).

---

# Investigation: Design Sketchybar Widget Integration and Display Semantics

**Question:** How should the sketchybar widget's ownership model, default display semantics, and popup information density be designed so the widget reads as a trustworthy glance-layer for orchestration state?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** architect agent
**Phase:** Complete
**Next Step:** Implementation issue for comprehension_pending field
**Status:** Complete
**Model:** dashboard-architecture

**Extracted-From:** Supersedes orch-go-sagnf, orch-go-iliu3, orch-go-9a0v6

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-03-24-inv-design-sketchybar-widget-live-daemon | extends | Yes — read current widget code, daemon-status.json, integration test | None — prior investigation left these 3 questions explicitly unresolved |

---

## Findings

### Finding 1: The widget already implements the correct bar display format

**Evidence:** The current `orch.lua` widget (in `~/.config/sketchybar/items/widgets/orch.lua`) already shows `active/max` format (e.g., "2/5") with worst-signal color encoding. When at capacity with queued work, it extends to "2/5 Q:10". The event provider (`orch_status.sh`) computes worst-signal-wins health level across 6 dimensions matching `health_signals.go` thresholds.

The alternative — active-only with color — would conflate two channels (utilization and health) into one signal. A user seeing "3" in yellow has to ask: "yellow because 3 is high utilization, or yellow because something is unhealthy?" With "3/5" in yellow, the fraction answers utilization and the color answers health independently.

**Source:** `~/.config/sketchybar/items/widgets/orch.lua` (label logic), `~/.config/sketchybar/helpers/event_providers/orch_status/orch_status.sh` (health computation), `pkg/daemon/health_signals.go:40-176` (canonical thresholds)

**Significance:** Fork 1 is already resolved by the current implementation. No change needed. The active/max format gives two independent information channels (numeric fraction + chromatic health) which is strictly better for glanceability than collapsing both into color-only.

---

### Finding 2: Comprehension count is computed but discarded — adding it to daemon-status.json is zero-cost at the widget layer

**Evidence:** The daemon calls `CheckComprehensionThrottle()` during every spawn gate check (`pkg/daemon/compliance.go:56-57`), which calls `ComprehensionQuerier.CountPending()`. This count is used to decide whether to block spawning, then discarded. It is NOT written to `daemon-status.json`.

The current `DaemonStatus` struct (`pkg/daemon/status.go:15-65`) has no comprehension field. The status file write in `cmd/orch/daemon_loop.go:718-741` builds the status snapshot from pool state, periodic task results, and verification state — but not comprehension.

Adding a `ComprehensionPending *ComprehensionSnapshot` field to DaemonStatus would:
- Require ~10 lines of Go code (struct definition + population in daemon_loop.go)
- Cost zero at the widget layer (file read, no bd CLI call)
- Eliminate the 500ms-per-poll bd CLI call that would otherwise be needed in orch_status.sh
- Give the event provider bash script access to comprehension count via jq, same as all other metrics

The alternative — running `bd list --label comprehension:unread` in orch_status.sh — would add ~500ms per poll cycle (10s interval). This is 5% of each cycle spent on one metric, and the metric is already computed in Go.

**Source:** `pkg/daemon/compliance.go:55-62` (gate check calls CountPending), `pkg/daemon/status.go:15-65` (DaemonStatus struct, no comprehension field), `cmd/orch/daemon_loop.go:718-741` (status snapshot construction)

**Significance:** This is the one Go code change worth making. It moves comprehension signal from "expensive CLI call at widget layer" to "free file read already in the data pipeline." The daemon already pays the bd CLI cost for spawn gating — the widget should piggyback on that result.

---

### Finding 3: Integration test already provides the coupling guarantee that repo ownership would

**Evidence:** `pkg/daemon/sketchybar_integration_test.go` verifies parity between Go health computation (`ComputeDaemonHealth()`) and the bash health computation extracted from the actual `orch_status.sh` at its installed location (`~/.config/sketchybar/helpers/event_providers/orch_status/orch_status.sh`).

The test covers 8 scenarios (all_green, verification_yellow, verification_paused_red, capacity_saturated_red, capacity_yellow_80pct, unresponsive_red, questions_yellow, queue_depth_red). It writes synthetic daemon-status.json files, runs both Go and bash health computation, and asserts the HEALTH_LEVEL matches.

This means: if someone changes health thresholds in `health_signals.go` without updating `orch_status.sh` (or vice versa), `go test ./pkg/daemon/...` fails. This is exactly the coupling guarantee that "put the script in the repo" would provide, but without requiring the widget's personal config (colors, fonts, layout, position) to be in a project repo.

The widget is inherently machine-specific — it references Dracula colors, specific icon fonts, sketchybar popup positioning, and macOS-specific `date -j` commands. Pulling it into orch-go would create a maintenance burden (changes to personal aesthetics require project commits) without improving correctness (the test already catches drift).

**Source:** `pkg/daemon/sketchybar_integration_test.go` (8 test scenarios, Go/bash parity verification), `~/.config/sketchybar/items/widgets/orch.lua` (machine-specific config: colors, fonts, layout)

**Significance:** Fork 3 is resolved: the script stays in `~/.config/sketchybar/`. The integration test is the correct coupling mechanism — it protects correctness while respecting the widget's nature as personal config.

---

## Synthesis

**Key Insights:**

1. **The widget is a passive display layer, not a system component** — This is the unifying framing that resolves all three forks. Like a terminal theme or shell prompt, the widget reads system state but doesn't participate in it. Decisions should optimize for the display (glanceability) and minimize coupling to the system (no repo ownership, no expensive calls).

2. **Two independent information channels beat one** — Active/max + color gives the user a utilization fraction AND a health indicator simultaneously. Collapsing to active-only-with-color forces the user to interpret a single overloaded signal. This is why the current implementation is correct.

3. **Piggyback on existing computation, don't duplicate it** — The daemon already calls `CountPending()` for spawn throttling. Writing the result to daemon-status.json turns a 500ms widget-layer cost into a 0ms file read. This is the pattern: the event provider should never shell out to bd or orch when the daemon has already computed the answer.

**Answer to Investigation Question:**

The widget's ownership model, display semantics, and popup density should be:

- **Ownership:** Widget stays in `~/.config/sketchybar/`, coupled to orch-go via the existing integration test. Not a project artifact.
- **Bar display:** `active/max` with worst-signal-wins color (current implementation, confirmed correct). Two channels — fraction for utilization, color for health.
- **Popup density:** Include comprehension:pending count, but source it from daemon-status.json (requires adding the field to DaemonStatus), not a live bd CLI call. Show it as a 7th health signal row in the popup alongside the existing 6. Popup continues to use on-demand `orch status --json` for agent detail.

The one code change: add `ComprehensionPending` to `DaemonStatus` struct and populate it during the daemon's status file write. Everything else is widget-layer work in `~/.config/sketchybar/`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Widget already displays active/max with worst-signal color (verified: read orch.lua source)
- ✅ Daemon computes comprehension count during gate check (verified: read compliance.go:55-62)
- ✅ Integration test verifies Go/bash health parity for 8 scenarios (verified: read sketchybar_integration_test.go)
- ✅ DaemonStatus struct has no comprehension field (verified: read status.go:15-65)

**What's untested:**

- ⚠️ Whether comprehension count stale by one poll cycle (30s) is acceptable for display (likely fine — comprehension changes are infrequent)
- ⚠️ Whether adding a 7th health signal row to the popup causes visual overflow (sketchybar popup capacity unverified for >10 rows)
- ⚠️ Whether the daemon gate check result can be cleanly plumbed to the status writer (might need minor refactoring in daemon_loop.go)

**What would change this:**

- If comprehension count changes sub-second (never happens — requires agent completion), the 30s staleness would matter
- If sketchybar popup max items is <15, the layout may need compression
- If daemon architecture moves to not running CheckPreSpawnGates every cycle, comprehension would need a separate periodic query

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Keep bar display as active/max with color | implementation | Already implemented, confirmed correct, no change needed |
| Add ComprehensionPending to DaemonStatus | architectural | Cross-component schema change (daemon → status file → widget) |
| Keep widget in ~/.config/sketchybar/ | implementation | Already implemented, integration test provides coupling |
| Add comprehension row to widget popup | implementation | Widget-layer display change in personal config |

### Recommended Approach: Piggyback Comprehension on Daemon Status File

**Add ComprehensionPending snapshot to daemon-status.json** — the daemon already computes this count for spawn throttling; writing it to the status file makes it available to the widget at zero marginal cost.

**Why this approach:**
- Daemon already pays the bd CLI cost for `CountPending()` in every spawn gate check
- Widget event provider reads daemon-status.json every 10s — adding one jq field is free
- Eliminates the 500ms-per-poll alternative of running bd CLI in orch_status.sh
- Follows the established pattern: every widget metric comes from a file the daemon already writes

**Trade-offs accepted:**
- Comprehension count is stale by up to one daemon poll cycle (~30s). Acceptable because comprehension state changes infrequently (only on agent completion).
- Requires a Go code change to daemon_loop.go + status.go. This is the minimum viable Go change — one struct and one field assignment.

**Implementation sequence:**
1. **Add ComprehensionSnapshot to DaemonStatus** (status.go) — struct with `UnreadCount`, `Threshold`, `IsThrottled` fields
2. **Populate during status file write** (daemon_loop.go) — plumb the count from gate check result or re-query via ComprehensionQuerier
3. **Update orch_status.sh** to read comprehension fields from daemon-status.json and pass as env vars
4. **Update orch.lua popup** to show comprehension row (7th health signal)
5. **Extend integration test** to cover comprehension signal parity

### Alternative Approaches Considered

**Option B: Run bd CLI in orch_status.sh event provider**
- **Pros:** No Go code change, all changes in widget layer
- **Cons:** 500ms per poll cycle, duplicates work daemon already does, makes event provider slower and more fragile (bd CLI availability dependency)
- **When to use instead:** If daemon architecture changes and comprehension is no longer checked during gate cycle

**Option C: Show comprehension only in popup via orch status --json**
- **Pros:** No Go code change, no event provider change. `orch status --json` already includes review_queue
- **Cons:** Comprehension not visible in bar display at all. User must click popup to see "you have 7 items to review." Defeats the glance-layer purpose.
- **When to use instead:** If comprehension count is low-value signal (it's not — it's the primary "you have work to review" indicator)

**Rationale for recommendation:** Option A follows the principle "piggyback on existing computation." The daemon already pays the cost; the widget should read the receipt, not place a second order.

---

### Implementation Details

**What to implement first:**
- ComprehensionSnapshot struct + DaemonStatus field (Go, ~15 lines)
- daemon_loop.go status snapshot population (Go, ~10 lines)
- These gate all downstream widget changes

**Things to watch out for:**
- ⚠️ The ComprehensionQuerier field on Daemon is optional (nil = fail-open). The status file should write `null` when querier is nil, and orch_status.sh should handle missing field gracefully.
- ⚠️ The daemon calls CheckPreSpawnGates only when attempting to spawn. If no work is ready, the count won't refresh. Consider querying ComprehensionQuerier.CountPending() directly in the periodic status write path instead.
- ⚠️ Integration test parity: when adding comprehension to orch_status.sh, extend sketchybar_integration_test.go to cover comprehension signal.

**Areas needing further investigation:**
- Whether to show comprehension count in the bar label itself (e.g., "2/5 C:7") or only in popup. Current recommendation: popup only, to avoid bar label clutter.
- Whether to add comprehension:processed (Dylan hasn't read brief) as a separate signal. Deferred — unread is the actionable signal.

**Success criteria:**
- ✅ daemon-status.json includes comprehension_pending with unread_count, threshold, and is_throttled
- ✅ Widget popup shows comprehension row with count/threshold and appropriate color
- ✅ No bd CLI calls in orch_status.sh event provider loop
- ✅ Integration test covers comprehension signal parity
- ✅ Widget handles missing comprehension field (daemon querier nil) gracefully

---

## Defect Class Exposure

| Class | Exposure | Mitigation |
|-------|----------|------------|
| Class 5: Contradictory Authority Signals | Comprehension count from two sources (daemon status file vs live bd CLI) could disagree | Single source: daemon-status.json only. Widget never calls bd CLI. |
| Class 3: Stale Artifact Accumulation | daemon-status.json persists after daemon dies, showing stale comprehension count | Existing mitigation: event provider checks daemon PID liveness, shows "stopped" state |

---

## References

**Files Examined:**
- `pkg/daemon/status.go:15-65` — DaemonStatus struct (no comprehension field)
- `pkg/daemon/compliance.go:55-62` — CheckPreSpawnGates calls CheckComprehensionThrottle
- `pkg/daemon/comprehension_queue.go:158-173` — CheckComprehensionThrottle returns (allowed, count, threshold)
- `pkg/daemon/health_signals.go:40-176` — ComputeDaemonHealth with 6 signals
- `cmd/orch/daemon_loop.go:718-741` — Status file write (no comprehension)
- `pkg/daemon/sketchybar_integration_test.go` — Go/bash health parity test (8 scenarios)
- `~/.config/sketchybar/items/widgets/orch.lua` — Current widget implementation
- `~/.config/sketchybar/helpers/event_providers/orch_status/orch_status.sh` — Event provider script
- `.kb/investigations/2026-03-24-inv-design-sketchybar-widget-live-daemon.md` — Prior investigation

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-24-inv-design-sketchybar-widget-live-daemon.md` — Prior widget design investigation, left these 3 questions open
- **Model:** `.kb/models/dashboard-architecture/model.md` — Dashboard architecture model (substrate consulted)
- **Decision:** Usage display color thresholds: green <60%, yellow 60-80%, red >80% (established prior decision, consistent with health signal approach)

---

## Investigation History

**2026-03-26 16:00:** Investigation started
- Initial question: Three pending sketchybar widget judgment calls, consolidated from orch-go-sagnf/iliu3/9a0v6
- Context: Widget exists and mostly works, but three design questions were deferred

**2026-03-26 16:15:** Identified unifying framing
- The three questions share one axis: "what is the widget's relationship to the system?"
- Answer: passive display layer, not system component. This resolves all three.

**2026-03-26 16:25:** Found key technical insight
- Daemon already computes comprehension count (compliance.go:56-57) but discards it
- Adding to daemon-status.json eliminates 500ms/poll bd CLI call at widget layer

**2026-03-26 16:30:** Investigation completed
- Status: Complete
- Key outcome: active/max with color (keep), comprehension via daemon-status.json (one Go change), widget stays in ~/.config (keep)
