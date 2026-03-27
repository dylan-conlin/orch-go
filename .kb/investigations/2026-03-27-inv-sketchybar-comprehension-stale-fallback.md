## Summary (D.E.K.N.)

**Delta:** Sketchybar widget now falls back to live `bd search -l comprehension:unread` when daemon PID is dead, eliminating stale comprehension count display after SIGKILL crashes.

**Evidence:** Simulated dead daemon (PID 99999) with stale comprehension count 0 in daemon-status.json; provider correctly detected dead PID via `kill -0`, set STATUS=dead, and queried bd to get live count 54. Verified live daemon path does not trigger fallback (no unnecessary 500ms bd calls).

**Knowledge:** The existing liveness detection (last_poll age) has a 2-minute blind spot after SIGKILL — PID liveness check (`kill -0`) catches it immediately. The bd fallback adds ~500ms per poll only when daemon is confirmed dead, which is an acceptable tradeoff for correct data.

**Next:** Close. Fix is implemented and tested.

**Authority:** implementation — Widget provider is personal config outside the repo, fix is a surgical fallback that follows established health signal patterns.

---

# Investigation: Sketchybar Comprehension Stale Fallback

**Question:** Why does the sketchybar widget show comprehension count 0 when daemon is down but items are accumulating?

**Started:** 2026-03-27
**Updated:** 2026-03-27
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-26-inv-design-sketchybar-widget-integration-display.md | extends | Yes — read widget code, provider, daemon status struct | None — prior investigation identified comprehension field but didn't address staleness |

---

## Findings

### Finding 1: Provider has no daemon process liveness check

**Evidence:** The provider (`orch_status.sh`) reads `daemon-status.json` and trusts the `status` field ("running") even when the daemon process is dead. It checks `last_poll` age for liveness detection (lines 92-106), but this has a 2-minute blind spot — SIGKILL leaves the file with a recent `last_poll` that only becomes stale over time.

**Source:** `~/.config/sketchybar/helpers/event_providers/orch_status/orch_status.sh:64-106`

**Significance:** The root cause. When daemon is SIGKILL'd, the status file says "running" with a comprehension count of 0 (from before items accumulated). The provider trusts this without verifying the process is alive.

---

### Finding 2: PID liveness check via kill -0 catches SIGKILL immediately

**Evidence:** The daemon-status.json includes a `pid` field. Adding `kill -0 $PID 2>/dev/null` to the provider detects dead processes instantly — no 2-minute blind spot. Test: PID 99999 (dead) correctly detected as not alive; real daemon PID 97725 correctly detected as alive.

**Source:** Test scripts at `/tmp/test_stale_daemon.sh` and `/tmp/test_alive_daemon.sh`

**Significance:** This is the key fix mechanism. Combined with setting STATUS="dead", it enables both the provider's health computation and the Lua widget to react correctly.

---

### Finding 3: bd search -l comprehension:unread provides live fallback

**Evidence:** When daemon is detected as dead, querying `bd search -l comprehension:unread | wc -l` returns the live count (54 items) while the stale file showed 0. This call takes ~500ms, which is acceptable as a fallback-only cost (not fired when daemon is healthy).

**Source:** Test script at `/tmp/test_e2e_stale.sh` — full provider logic extraction with synthetic dead daemon

**Significance:** The fallback is only triggered when DAEMON_ALIVE=false or LIVENESS_LEVEL != green, so normal operation has zero overhead.

---

## Synthesis

**Key Insights:**

1. **PID liveness eliminates the blind spot** — The existing last_poll detection takes 2+ minutes to trigger. PID check is instant, catching SIGKILL crashes on the very next poll cycle.

2. **Fallback is conditional, not always-on** — The 500ms bd CLI cost only applies when daemon is confirmed dead. Normal operation is unchanged.

3. **STATUS="dead" gives the widget a clear signal** — The Lua widget now shows "dead C:54" in red instead of the misleading "2/5 C:0" that the stale file would produce.

**Answer to Investigation Question:**

The widget showed 0 because `orch_status.sh` read the stale daemon-status.json without verifying that the daemon process was alive. The fix adds a `kill -0 $PID` check to detect dead daemons immediately, then falls back to `bd search -l comprehension:unread` for a live count. The Lua widget handles the new "dead" status with a dedicated red display showing the live comprehension count.

---

## Structured Uncertainty

**What's tested:**

- ✅ Dead PID (99999) detected correctly via kill -0 (verified: test script)
- ✅ Live PID (97725) does not trigger fallback (verified: test script)
- ✅ bd fallback returns live count 54 vs stale count 0 (verified: e2e test)
- ✅ Bash syntax validation passes (verified: bash -n)
- ✅ Full provider logic extraction matches expected outputs (verified: e2e test)

**What's untested:**

- ⚠️ Actual sketchybar rendering of "dead C:54" label (visual only, not testable headlessly)
- ⚠️ Widget popup behavior when STATUS=dead (popup reads daemon-status.json directly, not event vars)
- ⚠️ Whether daemon SIGKILL root cause is OOM or launchd watchdog (option 2 from issue)

**What would change this:**

- If bd CLI becomes unavailable when daemon is dead, fallback would silently fail to 0
- If daemon PID field is 0 or missing, kill -0 check is skipped (fallback via liveness_level still works)

---

## References

**Files Examined:**
- `~/.config/sketchybar/helpers/event_providers/orch_status/orch_status.sh` — Event provider (modified)
- `~/.config/sketchybar/items/widgets/orch.lua` — Widget (modified)
- `pkg/daemon/status.go` — DaemonStatus struct (already has comprehension field)
- `pkg/daemon/health_signals.go` — Health signal computation (reference for parity)
- `.kb/investigations/2026-03-26-inv-design-sketchybar-widget-integration-display.md` — Prior investigation

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-26-inv-design-sketchybar-widget-integration-display.md` — Widget design decisions
- **Investigation:** `.kb/investigations/2026-03-24-inv-design-sketchybar-widget-live-daemon.md` — Initial widget design

---

## Investigation History

**2026-03-27:** Investigation started
- Initial question: Why does sketchybar show comprehension 0 when daemon is SIGKILL'd?
- Context: Dylan had no ambient visibility into queue backpressure

**2026-03-27:** Root cause identified + fix implemented
- Provider trusts stale file without PID liveness check
- Fix: kill -0 PID check + bd fallback + STATUS=dead for widget
- Verified with synthetic dead daemon tests (all passed)

**2026-03-27:** Investigation completed
- Status: Complete
- Key outcome: Two-file fix (orch_status.sh + orch.lua) eliminates stale comprehension display
