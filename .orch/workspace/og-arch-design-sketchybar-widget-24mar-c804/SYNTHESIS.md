# Session Synthesis

**Agent:** og-arch-design-sketchybar-widget-24mar-c804
**Issue:** orch-go-7c4s1
**Duration:** 2026-03-24 ~11:40 → ~12:10
**Outcome:** success

---

## Plain-Language Summary

Designed a sketchybar widget for live daemon observability. The key insight: `daemon-status.json` (written by the daemon every poll cycle, ~0ms to read) already contains all metrics needed for the bar display — active/max agents, capacity, queue depth, verification state, unresponsive count, and questions waiting. No new Go code is needed. The existing `orch.lua` widget is broken (references a removed `agent-registry.json` file) but its architecture (shell event provider + lua subscriber) is correct and should be reused. The design uses a hybrid approach: fast file reads for the bar (every 5s), expensive `orch status --json` only on popup click for agent detail. The bar encodes daemon health as color (green/yellow/red using thresholds already defined in `health_signals.go`).

---

## TLDR

Designed sketchybar widget that reads `~/.orch/daemon-status.json` (0ms) for bar display and `orch status --json` (2-5s) on popup click. Existing widget is broken and needs rewrite. Created 4 implementation issues (3 components + 1 integration).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-03-24-inv-design-sketchybar-widget-live-daemon.md` - Full design investigation with recommendations
- `.orch/workspace/og-arch-design-sketchybar-widget-24mar-c804/SYNTHESIS.md` - This file
- `.orch/workspace/og-arch-design-sketchybar-widget-24mar-c804/VERIFICATION_SPEC.yaml` - Verification contract

### Files Modified
- None (design only — no code changes)

---

## Evidence (What Was Observed)

- `daemon-status.json` contains: capacity (max/active/available), ready_count, status, verification, phase_timeout, question_detection, beads_health — updated every daemon poll cycle
- `capacity-cache.json` contains: per-account 5h/7d usage percentages — updated every 5 minutes by daemon
- Existing `orch.lua` references `$HOME/.orch/agent-registry.json` which no longer exists (removed per No Local Agent State constraint)
- Existing `orch_status.sh` also references old registry and wrong orch path (`aider-env/bin/orch`)
- `health_signals.go` defines 6 health signals with green/yellow/red thresholds — directly reusable for widget coloring
- `orch status --json` returns comprehensive StatusOutput (swarm, accounts, agents, review_queue, session_metrics)

---

## Architectural Choices

### Hybrid data source (file reads for bar, CLI for popup)
- **What I chose:** Read `daemon-status.json` every 5s for bar display, call `orch status --json` only on popup click
- **What I rejected:** Polling `orch status --json` every 10s (current broken approach), adding daemon HTTP/socket endpoint
- **Why:** File reads are 1000x faster (0ms vs 2-5s). The daemon already writes comprehensive metrics. No new Go code needed.
- **Risk accepted:** Bar shows daemon's pool tracker view of active agents, which may differ slightly from orch status's cross-source discovery

### Worst-signal-wins color encoding
- **What I chose:** Bar color = worst health signal across all 6 dimensions (liveness, capacity, queue, verification, unresponsive, questions)
- **What I rejected:** Per-metric separate indicators, multiple colored dots
- **Why:** A single color gives instant "is my daemon healthy?" answer. Detailed breakdown is available in the popup.
- **Risk accepted:** Can't distinguish which signal is driving the color at a glance — must click popup to know

---

## Verification Contract

Link: `VERIFICATION_SPEC.yaml` in workspace root

Key outcomes:
- Investigation file documents all 5 design forks with substrate-backed recommendations
- 4 implementation issues created with clear scope and acceptance criteria
- No Go code changes needed — design uses existing daemon file outputs

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-24-inv-design-sketchybar-widget-live-daemon.md` - Full design with fork analysis, implementation plan, and widget layout mockups

### Decisions Made
- Hybrid data source (daemon-status.json + on-demand orch status --json) because file reads are 1000x faster
- Worst-signal-wins coloring because it gives instant health overview in minimal bar space
- Reuse existing sketchybar architecture (event provider + lua subscriber) because it's proven by render.lua

### Constraints Discovered
- Comprehension:pending count is NOT in daemon-status.json — would need Go code change to add it (flagged as question orch-go-iliu3)
- Sketchybar uses Lua (not shell scripts) for widget logic — the event provider is shell, but the display logic is Lua

---

## Next (What Should Happen)

**Recommendation:** close (spawn follow-ups from created issues)

### Implementation Issues Created
1. `orch-go-biu97` - Rewrite orch_status.sh event provider
2. `orch-go-ggdyj` - Rewrite orch.lua bar display
3. `orch-go-fqpw1` - Rewrite orch.lua popup
4. `orch-go-vryep` - Integration verification (depends on above 3)

### Blocking Questions Created
1. `orch-go-sagnf` - Bar display format: active/max vs active-only?
2. `orch-go-iliu3` - Comprehension:pending in bar vs popup-only?
3. `orch-go-9a0v6` - Event provider file placement (orch-go repo vs sketchybar config)?

---

## Unexplored Questions

- Whether to add `comprehension_pending` field to DaemonStatus struct (requires Go code change in daemon periodic tasks)
- Whether to show recent daemon decisions in popup (would require tailing daemon.log)
- Whether 5s poll interval feels responsive enough vs 2-3s

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** architect
**Model:** claude-opus-4-6
**Workspace:** `.orch/workspace/og-arch-design-sketchybar-widget-24mar-c804/`
**Investigation:** `.kb/investigations/2026-03-24-inv-design-sketchybar-widget-live-daemon.md`
**Beads:** `bd show orch-go-7c4s1`
