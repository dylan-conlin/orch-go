# Session Synthesis

**Agent:** og-inv-sketchybar-comprehension-widget-27mar-392a
**Issue:** orch-go-qloo8
**Outcome:** success

---

## Plain-Language Summary

The sketchybar widget showed "comprehension: 0" even when 54 items were waiting for review, because the daemon had been killed (SIGKILL) and the status file it writes froze at its last value. The fix adds a process liveness check (`kill -0 PID`) to the event provider — when the daemon is dead, the provider queries beads directly for the live comprehension count instead of trusting the stale file. The widget now shows "dead C:54" in red, giving Dylan accurate ambient visibility even when the daemon crashes.

## Verification Contract

See `VERIFICATION_SPEC.yaml` — key outcomes: dead PID detected correctly, live PID does not trigger fallback, bd fallback returns live count (54 vs stale 0), bash syntax valid.

---

## TLDR

Widget showed stale comprehension count 0 because daemon-status.json froze after SIGKILL. Added PID liveness check to provider — falls back to live bd query when daemon is dead.

---

## Delta (What Changed)

### Files Modified
- `~/.config/sketchybar/helpers/event_providers/orch_status/orch_status.sh` — Added PID liveness check (kill -0), STATUS=dead override, bd fallback for comprehension count when daemon dead or file stale
- `~/.config/sketchybar/items/widgets/orch.lua` — Added "dead" status handling with red label showing live comprehension count

### Files Created
- `.kb/investigations/2026-03-27-inv-sketchybar-comprehension-stale-fallback.md` — Investigation with D.E.K.N.

---

## Evidence (What Was Observed)

- daemon-status.json with dead PID 99999 and comprehension count 0: provider detected dead PID via kill -0
- bd search -l comprehension:unread returned 54 items (live count)
- Live daemon PID 97725: fallback correctly NOT triggered (zero overhead in normal operation)
- Full e2e test: STATUS=dead, LIVENESS=red, COMPREHENSION=54 — all expected

### Tests Run
```bash
# Dead PID detection
bash /tmp/test_stale_daemon.sh
# PASS: Detected dead daemon PID 99999, live count 54

# Live daemon no-fallback
bash /tmp/test_alive_daemon.sh
# PASS: Daemon alive, no fallback needed

# End-to-end provider logic
bash /tmp/test_e2e_stale.sh
# ALL TESTS PASSED

# Syntax validation
bash -n ~/.config/sketchybar/helpers/event_providers/orch_status/orch_status.sh
# PASS: syntax OK
```

---

## Architectural Choices

### PID liveness check vs file mtime check
- **What I chose:** `kill -0 $PID` from the daemon-status.json pid field
- **What I rejected:** Checking file mtime with `stat` against 2*INTERVAL threshold
- **Why:** PID check is instant (catches SIGKILL on next poll), mtime has the same 2-minute blind spot as last_poll. PID check is also more robust — it detects the specific failure mode (process dead but file fresh).
- **Risk accepted:** If PID field is 0 or missing, the check is skipped; liveness_level still catches it after 2 minutes.

---

## Knowledge (What Was Learned)

### Constraints Discovered
- The existing last_poll liveness detection has a 2-minute blind spot after SIGKILL — adequate for gradual degradation, inadequate for immediate crash detection

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
- [x] Investigation file has Phase: Complete
- [x] Ready for `orch complete orch-go-qloo8`

---

## Unexplored Questions

- **Why was the daemon SIGKILL'd?** — OOM? launchd watchdog? This is option 2 from the issue. The current fix handles the symptom; root cause investigation would prevent recurrence.
- **Should the popup also query bd when daemon is dead?** — Currently the popup reads daemon-status.json directly (Lua), not via the event provider. When daemon is dead, popup would show stale data too.

---

## Friction

No friction — smooth session

---

## Session Metadata

**Skill:** investigation
**Workspace:** `.orch/workspace/og-inv-sketchybar-comprehension-widget-27mar-392a/`
**Investigation:** `.kb/investigations/2026-03-27-inv-sketchybar-comprehension-stale-fallback.md`
**Beads:** `bd show orch-go-qloo8`
