# Model: macOS Click Freeze

**Domain:** macOS input subsystem — trackpad click events stop registering while cursor movement continues
**Last Updated:** 2026-02-11
**Synthesized From:** Session 11 (systematic elimination), Session 14 (recurrence + 3 research probes)

---

## Summary (30 seconds)

Trackpad clicks stop registering every ~15 minutes while cursor movement and keyboard continue working. `sudo killall -HUP WindowServer` fixes it every time (HUP = reconfigure, not restart). This points to WindowServer accumulating corrupted state in its click event pipeline. Eliminated: resource exhaustion, BetterTouchTool, Hammerspoon, Shortcat, middleClick, Raycast, Karabiner mouse rules, yabai focus_follows_mouse. Not yet eliminated: yabai itself, Karabiner (kernel-level DriverKit), skhd, borders, sketchybar.

---

## Core Mechanism

### What Happens

1. Trackpad **cursor moves** normally
2. Trackpad **clicks don't register** — no response from any app
3. Keyboard still works (can type, use shortcuts)
4. `sudo killall -HUP WindowServer` immediately restores clicks
5. Recurs within ~15 minutes

### What HUP Does

`SIGHUP` to WindowServer tells it to reconfigure — reload display config, reinitialize input routing. It does NOT restart WindowServer (that would log you out). The fact that HUP fixes it means:

- WindowServer's click routing state gets corrupted
- The corruption is in a soft state that reconfigure resets
- It's NOT a hardware issue (hardware would survive HUP)
- It's NOT a display issue (cursor still moves)

### Input Event Pipeline (macOS)

```
Hardware (trackpad) 
  → IOKit HID driver
    → Karabiner DriverKit (if installed — intercepts at kernel level)
      → WindowServer (routes events to apps)
        → App (receives click)
```

Click events and move events travel the same path but are **different event types**. Something is dropping/blocking click events specifically while passing move events through.

### Key Components

| Component | Role | Eliminated? |
|-----------|------|-------------|
| Trackpad hardware | Generates raw events | ✅ Yes — HUP fix proves it's not hardware |
| IOKit HID | Kernel input driver | ❌ Not tested — but unlikely (move events work) |
| Karabiner DriverKit | Kernel-level input interception | ❌ **Not yet tested** — strongest remaining suspect at kernel level |
| WindowServer | Routes events to apps | Partially — it's WHERE the problem manifests (HUP fixes it) |
| yabai | Window management, event interception | 🔄 **Currently testing** (stopped service) |
| skhd | Hotkey daemon, event interception | ❌ Not yet tested |
| borders | Window border drawing | ❌ Not yet tested (unlikely — display only) |
| sketchybar | Status bar | ❌ Not yet tested (unlikely — display only) |

---

## Why This Fails

### Hypothesis 1: yabai event interception corrupts WindowServer state (TESTING)

yabai uses the macOS Accessibility API to manage windows. It intercepts focus events and can manipulate window properties. If yabai's event handling creates a race condition or leaves WindowServer in an inconsistent state for click routing, clicks could be swallowed.

**Evidence for:** yabai is the most invasive remaining interceptor. It manipulates window focus, which is tightly coupled to click routing. yabai issue #2715 reports "window freezes during drag, mouse continues moving" — closest matching symptom found anywhere.

**Evidence against:** Disabling `focus_follows_mouse` in Session 11 didn't fix it (but that's just one feature — yabai does much more). No exact click freeze reports found in yabai GitHub (searched extensively).

### Hypothesis 2: Karabiner DriverKit drops click events at kernel level

Karabiner operates at the kernel level via DriverKit. It intercepts ALL input events before they reach WindowServer. If Karabiner's virtual HID device has a bug where click events get stuck in a buffer or filtered incorrectly, clicks would stop while moves continue.

**Evidence for:** Karabiner is the ONLY component operating at kernel level. It has separate handling for mouse/trackpad events vs keyboard events. A bug in click event passthrough would explain the selective failure. Karabiner issue #2566 (36 comments) documents "heavy intermittent mouse lag" when mouse device is enabled — different symptom but same subsystem.

**Evidence against:** Karabiner config only has keyboard rules (no mouse/trackpad rules). But DriverKit still processes all events even without rules. No exact click freeze reports found in Karabiner GitHub (searched extensively).

### Hypothesis 3: WindowServer internal corruption (no external cause) — WEAKENED

macOS 15.6.1 (Sequoia) may have a bug where WindowServer's click event routing table gets corrupted over time. This would be independent of any third-party software.

**Evidence for:** Would explain why eliminating multiple apps in Session 11 didn't fix it.

**Evidence against:** Three research probes (2026-02-11) searched GitHub, Reddit, Apple Discussions exhaustively — **zero matching reports** for this symptom pattern (clicks stop, cursor moves, HUP fixes). If this were a Sequoia bug, community reports would exist. This significantly weakens H3.

---

## Elimination Record

### Session 11 (2026-02-11, morning)

| Suspect | Action | Result | Conclusion |
|---------|--------|--------|------------|
| CPU/RAM exhaustion | Checked during freeze | CPU 74% idle, 12GB free | ✅ Eliminated |
| BetterTouchTool | Uninstalled entirely | Freeze recurred | ✅ Eliminated |
| Hammerspoon | Uninstalled | Freeze recurred | ✅ Eliminated |
| Shortcat | Uninstalled | Freeze recurred | ✅ Eliminated |
| middleClick | Uninstalled | Freeze recurred | ✅ Eliminated |
| Raycast | Uninstalled | Freeze recurred | ✅ Eliminated |
| Karabiner mouse rules | Checked config | Only keyboard rules present | ✅ Eliminated (rules, not daemon) |
| yabai focus_follows_mouse | Disabled setting | Freeze recurred | ✅ Eliminated (setting, not yabai) |

### Session 14 (2026-02-11, afternoon)

| Suspect | Action | Result | Conclusion |
|---------|--------|--------|------------|
| yabai (entire daemon) | `yabai --stop-service` | ⏳ Testing... | 🔄 In progress |

### Remaining Test Plan

If yabai elimination FAILS (freeze recurs without yabai):
1. Stop skhd: `skhd --stop-service`
2. If still recurs: quit Karabiner entirely (kernel-level test)
3. If still recurs: stop borders + sketchybar (unlikely but complete)
4. If still recurs with NOTHING running: macOS bug or hardware → Apple Support

If yabai elimination SUCCEEDS (no freeze for 20+ minutes):
1. Restart yabai: `yabai --start-service`
2. If freeze returns: yabai confirmed as cause
3. Investigate yabai version, check GitHub issues for similar reports
4. Options: update yabai, configure differently, or replace

---

## Constraints

### Why can't we just remove everything?

**Constraint:** yabai + skhd + Karabiner are Dylan's core window management and keyboard customization stack. Removing them degrades daily workflow significantly.

**Implication:** Need to identify the specific culprit, not blanket-remove. If Karabiner is the cause, need to find a config fix or update rather than removing it.

### Why HUP and not restart?

**Constraint:** Restarting WindowServer logs you out of macOS. HUP just reconfigures.

**Implication:** HUP is a viable workaround but not a fix. Automating `sudo killall -HUP WindowServer` every 10 minutes would mask the problem.

---

## Environment

- **macOS:** 15.6.1 (Sequoia)
- **Hardware:** Mac15,7 (M3 Pro)
- **Karabiner:** DriverKit-based (org.pqrs.Karabiner-DriverKit-VirtualHIDDevice)
- **yabai:** /opt/homebrew/bin/yabai
- **skhd:** /opt/homebrew/bin/skhd

---

## Evolution

**2026-02-11 (Session 11):** First systematic investigation. 4 freezes in ~1 hour. Eliminated 5 apps (BTT, Hammerspoon, Shortcat, middleClick, Raycast). BTT was a red herring (correlated but not causal).

**2026-02-11 (Session 14):** Recurrence confirmed pattern persists. Started elimination of remaining suspects with yabai disabled. Three research probes searched GitHub (yabai, Karabiner, broad), Reddit, Apple Discussions — zero matching reports found anywhere. Hypothesis 3 (macOS bug) significantly weakened. Closest match: yabai #2715 (drag freeze). Karabiner #2566 (mouse lag, 36 comments) is same subsystem but different symptom.

---

## References

**Investigations:**
- Session 11 handoff in `.orch/HANDOFF.md` — detailed elimination record

**Probes:**
- `.kb/models/macos-click-freeze/probes/2026-02-11-github-apple-support-search.md` — Broad search: zero matching reports
- `.kb/models/macos-click-freeze/probes/2026-02-11-karabiner-github-search.md` — Karabiner: mouse lag (#2566) but no click freeze
- `.kb/models/macos-click-freeze/probes/2026-02-11-yabai-github-issues-search.md` — yabai: drag freeze (#2715) closest match, no click freeze

**Issues:**
- `orch-go-uvtgi` [P2] — Click freeze tracking issue

**Related models:**
- None (macOS system issue, not orch-go)
