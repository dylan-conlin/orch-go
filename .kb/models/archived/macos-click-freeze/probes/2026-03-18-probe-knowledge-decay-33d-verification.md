# Probe: Knowledge Decay Verification — macOS Click Freeze (33d)

**Date:** 2026-03-18
**Model:** macos-click-freeze
**Type:** knowledge-decay
**Trigger:** 33 days since last probe (model last updated 2026-03-06, probes from 2026-02-11–2026-02-13)

---

## Verification Method

Checked live system state against model claims. No user interview conducted (cannot determine freeze frequency subjectively).

---

## Findings

### 1. MAJOR: macOS version is dramatically different

**Model claims:** macOS 15.7.4 (Sequoia), updated from 15.6.1
**Actual:** macOS 26.3.1 (Tahoe) — build 25D2128

This is a full major version upgrade (15 → 26). The entire WindowServer subsystem, IOKit HID stack, and CGEventTap infrastructure may have changed. This invalidates or at minimum requires re-evaluation of:
- H3 (WindowServer internal corruption) — different WindowServer version
- H7 (skhd CGEventTap pipeline corruption) — CGEventTap API behavior may differ
- Apple bug FB12113281 reference — may be fixed in macOS 26
- All architectural reasoning about the input event pipeline

**Verdict:** Model environment section is **STALE**. Core mechanism description may still be directionally correct but is unverified on macOS 26.

### 2. CONFIRMED: Service elimination states are accurate

| Service | Model State | Actual State | Match |
|---------|-------------|--------------|-------|
| yabai | ENABLED, running | Running (PID 93141) | ✅ |
| skhd | DISABLED, not running | Not running, service not loaded | ✅ |
| Karabiner | 15.9.0, running | Running, DriverKit active (ioreg confirmed) | ✅ |
| sketchybar | Re-enabled (Session 17) | Running (PID 2251) | ✅ |
| borders | "Disabled in batch" | Running (PID 2254) | ❌ Model stale — borders re-enabled |
| NI HardwareAgent | Fully uninstalled | No services found | ✅ |
| Ollama | Fully uninstalled | Not running | ✅ |
| colima/Docker | "Manually started" | Not running | ⚠️ Model describes Session 16 state, not current |

### 3. CONFIRMED: Reactive capture script exists but was never used

**Model claims:** `scripts/click-freeze-capture.sh` built, outputs to `~/.orch/click-freeze-captures/`
**Actual:** Script exists at `orch-go/scripts/click-freeze-capture.sh` (6.4KB, last modified 2026-02-18). No capture directory exists — **zero captures were ever collected**.

The "next approach" (bind to Karabiner hotkey, collect correlation data) was never executed. The binary search fallback plan was also never executed.

### 4. STALE: System daemon references

Docker/xquartz/Zoom plists still exist in `/Library/LaunchDaemons/` but model marks them as "disabled." Their state on macOS 26 is unknown (launchctl semantics may differ).

### 5. UNKNOWN: Current freeze frequency

Model's last observation (Session 17, 2026-02-14): "frequency decreased from every ~15 min to rare/intermittent." 33 days have passed with a major OS upgrade. Freeze may be:
- Fixed by macOS 26 (WindowServer rewrite/fixes)
- Still present but rare
- No longer occurring due to changed service landscape

**Cannot determine without user input.**

---

## Overall Verdict

**Model is PARTIALLY STALE.**

| Aspect | Status |
|--------|--------|
| Core mechanism (HUP fix, click-specific) | Directionally correct but unverified on macOS 26 |
| Hypothesis rankings (H6 leading) | Unknown — major OS change invalidates environment assumptions |
| Service states | Mostly accurate, borders re-enabled (minor) |
| Environment section | **STALE** — wrong macOS version (15.7.4 → 26.3.1) |
| Next steps / test plan | **STALE** — never executed, OS upgrade may make them moot |
| Capture tooling | Exists but unused — no data collected |

---

## Recommended Model Updates

1. **Environment:** Update macOS version to 26.3.1 (Tahoe), note the major version jump
2. **borders:** Update from "disabled in batch" to "re-enabled, running"
3. **colima/Docker:** Note not currently running
4. **Freeze status:** Add note that freeze status is unknown since macOS 26 upgrade
5. **H7 (skhd):** Note that CGEventTap behavior may differ on macOS 26; Apple bug FB12113281 status unknown
6. **Capture script:** Note that it was never used and no data was collected
7. **Next steps:** Mark binary search plan and capture-hotkey plan as superseded by OS upgrade; recommend fresh observation period on macOS 26 before resuming investigation

---

## Open Questions for User

1. Has the click freeze occurred at all since upgrading to macOS 26?
2. If yes, does `sudo killall -HUP WindowServer` still fix it?
3. Should this model be archived if the freeze hasn't recurred on macOS 26?
