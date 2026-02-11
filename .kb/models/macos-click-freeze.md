# Model: macOS Click Freeze

**Domain:** macOS input subsystem — trackpad click events stop registering while cursor movement continues
**Last Updated:** 2026-02-11
**Synthesized From:** Session 11 (systematic elimination), Session 14 (recurrence + 3 research probes)

---

## Summary (30 seconds)

Trackpad clicks stop registering every ~15 minutes while cursor movement and keyboard continue working. `sudo killall -HUP WindowServer` fixes it every time (HUP = reconfigure, not restart). This points to WindowServer accumulating corrupted state in its click event pipeline. Eliminated: resource exhaustion (single check), BetterTouchTool, Hammerspoon, Shortcat, middleClick, Raycast, Karabiner mouse rules, yabai focus_follows_mouse, **yabai entire daemon**. **New leading hypothesis:** system-wide memory pressure from concurrent agent spawning (35GB/36GB used, 607MB free, OpenCode 8.4GB). Not yet eliminated: Karabiner DriverKit, skhd, memory pressure.

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
| Karabiner DriverKit | Kernel-level input interception | ❌ **Primary suspect** — only kernel-level interceptor remaining |
| WindowServer | Routes events to apps | Partially — it's WHERE the problem manifests (HUP fixes it) |
| yabai | Window management, event interception | ✅ **Eliminated** — freeze recurred with yabai fully stopped |
| skhd | Hotkey daemon, event interception | ❌ Not yet tested |
| borders | Window border drawing | ❌ Not yet tested (unlikely — display only) |
| sketchybar | Status bar | ❌ Not yet tested (unlikely — display only) |

---

## Why This Fails

### Hypothesis 1: yabai event interception corrupts WindowServer state — ELIMINATED

yabai uses the macOS Accessibility API to manage windows. Freeze recurred with yabai fully stopped (`yabai --stop-service`, confirmed no process via `pgrep`).

**Evidence against:** Freeze recurred within ~30 minutes with yabai completely stopped. No yabai process running. Definitively eliminated.

### Hypothesis 2: Karabiner DriverKit drops click events at kernel level — PRIMARY SUSPECT

Karabiner operates at the kernel level via DriverKit. It intercepts ALL input events before they reach WindowServer. If Karabiner's virtual HID device has a bug where click events get stuck in a buffer or filtered incorrectly, clicks would stop while moves continue.

**Evidence for:** Karabiner is the ONLY component operating at kernel level. It has separate handling for mouse/trackpad events vs keyboard events. A bug in click event passthrough would explain the selective failure. Karabiner issue #2566 (36 comments) documents "heavy intermittent mouse lag" when mouse device is enabled — different symptom but same subsystem. **Now the primary suspect** after yabai elimination. XProtect update on Feb 10 may have changed DriverKit security policies.

**Evidence against:** Karabiner config only has keyboard rules (no mouse/trackpad rules). But DriverKit still processes all events even without rules. No exact click freeze reports found in Karabiner GitHub (searched extensively).

**Next test:** Quit Karabiner entirely and wait 20+ minutes.

### Hypothesis 3: WindowServer internal corruption (no external cause) — WEAKENED

macOS 15.6.1 (Sequoia) may have a bug where WindowServer's click event routing table gets corrupted over time. This would be independent of any third-party software.

**Evidence for:** Would explain why eliminating multiple apps in Session 11 didn't fix it.

**Evidence against:** Three research probes (2026-02-11) searched GitHub, Reddit, Apple Discussions exhaustively — **zero matching reports** for this symptom pattern (clicks stop, cursor moves, HUP fixes). If this were a Sequoia bug, community reports would exist. This significantly weakens H3.

### Hypothesis 4: Memory pressure from concurrent agent spawning — NEW, STRONG

System running at 35GB/36GB RAM with only 607MB free. 1.9GB in compressor. Load average 16.22 (extremely high for M3 Pro). This coincides with ramping up concurrent agent spawning — each headless OpenCode session + bun worker + tsserver stack consumes significant RAM. OpenCode alone is 8.4GB.

**Evidence for:**
- Click freeze started when concurrent agent spawning ramped up (yesterday)
- System has 607MB free of 36GB — severe memory pressure
- 699 processes, 4140 threads — extreme process count
- WindowServer event buffers under memory pressure could drop click events (smaller, lower-priority) while move events (continuous stream, higher priority) still get through
- HUP fix is consistent — reconfiguring WindowServer reallocates fresh event buffers
- Explains why eliminating individual apps didn't help — the problem is cumulative memory pressure, not any single app
- Explains why no community reports — unique to heavy concurrent AI agent workload
- Explains "started yesterday" timing — that's when the orch reliability chain spawned 4-7 concurrent agents

**Evidence against:** Session 11 checked CPU/RAM during a freeze and found "CPU 74% idle, 12GB free" — but that was a single point-in-time check. Memory pressure is dynamic; the system may have been at 607MB free when the freeze occurred, then freed memory by the time we checked. Need to monitor memory at the exact moment of freeze.

**Test:** Kill idle OpenCode sessions to free RAM. Monitor memory at next freeze. If freeze stops with fewer concurrent agents, memory pressure confirmed.

**Key insight:** This isn't about any single tool — it's about the system-wide memory footprint of running many headless AI agents simultaneously.

---

## Elimination Record

### Session 11 (2026-02-11, morning)

| Suspect | Action | Result | Conclusion |
|---------|--------|--------|------------|
| CPU/RAM exhaustion | Checked during freeze | CPU 74% idle, 12GB free | ⚠️ **Revisit** — single point-in-time check, memory now at 607MB free |
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
| yabai (entire daemon) | `yabai --stop-service` + confirmed no process | Freeze recurred ~30 min later | ✅ **Eliminated** |

### Remaining Test Plan

**Priority 1: Test memory pressure hypothesis (H4)**
1. Kill idle OpenCode sessions to free RAM — target getting back to 10GB+ free
2. Monitor memory at exact moment of next freeze (`memory_pressure` command)
3. If freeze stops with fewer agents: **memory pressure confirmed**
4. If freeze continues with free RAM: move to Karabiner test

**Priority 2: Quit Karabiner entirely** (if H4 disproven)
1. Stop all Karabiner components
2. If still recurs: stop skhd (`skhd --stop-service`)
3. If still recurs: stop borders + sketchybar (unlikely — display only)
4. If still recurs with NOTHING running + free RAM: macOS bug or hardware → Apple Support

**If memory pressure confirmed:**
1. Set `--max-agents` lower (3 instead of 5)
2. Aggressively clean up idle sessions after completion
3. Monitor OpenCode memory footprint per session
4. Consider lighter-weight agent backend

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

**2026-02-11 (Session 14):** **yabai eliminated** — freeze recurred with yabai fully stopped (confirmed no process). Three research probes searched GitHub (yabai, Karabiner, broad), Reddit, Apple Discussions — zero matching reports found anywhere. Hypothesis 3 (macOS bug) significantly weakened. **New H4: memory pressure** — system at 35GB/36GB, 607MB free, OpenCode alone 8.4GB. Concurrent agent spawning ramped up yesterday (when freeze started). This is now the leading hypothesis — explains timing, explains why individual app elimination failed, explains unique-to-Dylan symptom.

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
