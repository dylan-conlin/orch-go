# Model: macOS Click Freeze

**Domain:** macOS input subsystem — trackpad click events stop registering while cursor movement continues
**Last Updated:** 2026-02-11
**Synthesized From:** Session 11 (systematic elimination), Session 14 (recurrence + 3 research probes + OpenCode fork resource audit)

---

## Summary (30 seconds)

Trackpad clicks stop registering every ~15 minutes while cursor movement and keyboard continue working. `sudo killall -HUP WindowServer` fixes it every time (HUP = reconfigure, not restart). This points to WindowServer accumulating corrupted state in its click event pipeline. Eliminated: BetterTouchTool, Hammerspoon, Shortcat, middleClick, Raycast, Karabiner mouse rules, yabai focus_follows_mouse, **yabai entire daemon**. **Two active hypotheses:** H2 (Karabiner DriverKit — only kernel-level interceptor) and H4 (memory pressure from OpenCode instance accumulation — correlates with onset timing). Not yet eliminated: Karabiner DriverKit, skhd, memory pressure.

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

### Hypothesis 4: Memory pressure from OpenCode instance accumulation — TESTING

OpenCode accumulates instances (with LSP/MCP/file watchers) per unique project directory. Each instance costs 300-500MB for LSP alone. With MAX_INSTANCES=20 and orchestrator spawning agents to many worktree directories, memory fills up. OpenCode grew from 336MB to 3.5GB in 15 min, and was at 8.6GB before restart.

**Evidence for:**
- Click freeze started when concurrent agent spawning ramped up (yesterday)
- System had 607MB free of 36GB when freeze occurred — severe memory pressure
- OpenCode: 8.6GB RSS before restart, 336MB after, already 3.5GB after 15 min
- OpenCode fork audit confirmed: MAX_INSTANCES=20 × ~500MB LSP = up to 10GB for LSP alone
- No freeze (so far) since OpenCode restart freed 8.3GB — **active test running**
- The "89% free" snapshot from audit agent was a point-in-time measurement shortly after restart, NOT when freeze actually occurred

**Evidence against:**
- Session 11 found "CPU 74% idle, 12GB free" during one freeze — but that was a single check, memory is dynamic
- Audit agent claimed freeze recurred at 89% free, but timing is unclear — the agent may have conflated the restart with the freeze occurrence
- Could be correlation (heavy spawning = more tool activity = more Karabiner events)

**Status:** Test in progress. OpenCode restarted, memory freed. If freeze does NOT recur for 30+ min with low memory, H4 strengthened. If freeze recurs with 10GB+ free, H4 weakened.

**OpenCode tuning (worth doing regardless):** Reduce MAX_INSTANCES 20→8, IDLE_TTL 30min→5min for headless mode, add disposeAll to server.stop(), add periodic eviction timer. See `~/Documents/personal/opencode/.kb/investigations/2026-02-11-inv-opencode-fork-resource-audit-investigate.md`.

---

## Elimination Record

### Session 11 (2026-02-11, morning)

| Suspect | Action | Result | Conclusion |
|---------|--------|--------|------------|
| CPU/RAM exhaustion | Checked during freeze | CPU 74% idle, 12GB free | ⚠️ **Revisit** — single point-in-time check; system was at 607MB free later when freeze occurred |
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

### Session 14, continued (2026-02-11, afternoon)

| Suspect | Action | Result | Conclusion |
|---------|--------|--------|------------|
| Memory pressure (H4) | Restarted OpenCode (8.6GB → 336MB), freed 8.3GB RAM | ⏳ Testing... | 🔄 In progress — no freeze yet since restart |
| OpenCode fork leak | Resource audit investigation | Fork is BETTER than upstream (has LRU/TTL eviction). Not a leak, but params too high for orchestrator. | ✅ Not a bug — tuning issue |

### Remaining Test Plan

**Active test: H4 (memory pressure)**
- OpenCode restarted, memory freed from 607MB → 11GB free
- If no freeze for 30+ min: H4 strengthened → implement OpenCode tuning fixes
- If freeze recurs with abundant memory: H4 eliminated → test Karabiner

**Next: Quit Karabiner entirely** (H2 — if H4 eliminated)
1. Stop all Karabiner components (Karabiner-Elements, DriverKit VirtualHIDDevice)
2. Wait 20+ minutes — if freeze stops: **Karabiner confirmed**
3. If confirmed: investigate Karabiner DriverKit mouse/trackpad passthrough, check for updates, file issue

**Then: Stop skhd** (if Karabiner eliminated)
1. `skhd --stop-service`
2. If still recurs: stop borders + sketchybar (unlikely — display only)

**Finally: Nothing running** (if all eliminated)
1. If still recurs with NOTHING running + free RAM: macOS bug or hardware → Apple Support

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

**2026-02-11 (Session 14):** **yabai eliminated** — freeze recurred with yabai fully stopped (confirmed no process). Three research probes searched GitHub (yabai, Karabiner, broad), Reddit, Apple Discussions — zero matching reports found anywhere. Hypothesis 3 (macOS bug) significantly weakened. New H4: memory pressure — system at 35GB/36GB, 607MB free, OpenCode alone 8.4GB.

**2026-02-11 (Session 14, continued):** OpenCode fork resource audit found fork is better than upstream (LRU/TTL eviction added Feb 7). But params too high for orchestrator (MAX_INSTANCES=20, IDLE_TTL=30min). OpenCode grew from 336MB → 3.5GB in 15 min, was at 8.6GB before restart. Restarted OpenCode, freed 8.3GB. **H4 test in progress** — no freeze since restart. Audit agent prematurely eliminated H4 based on a point-in-time memory snapshot, not when freeze actually occurred. Both H2 (Karabiner) and H4 (memory pressure) remain active hypotheses.

---

## References

**Investigations:**
- Session 11 handoff in `.orch/HANDOFF.md` — detailed elimination record
- `~/Documents/personal/opencode/.kb/investigations/2026-02-11-inv-opencode-fork-resource-audit-investigate.md` — OpenCode fork resource audit (eliminated H4, found optimization opportunities)

**Probes:**
- `.kb/models/macos-click-freeze/probes/2026-02-11-github-apple-support-search.md` — Broad search: zero matching reports
- `.kb/models/macos-click-freeze/probes/2026-02-11-karabiner-github-search.md` — Karabiner: mouse lag (#2566) but no click freeze
- `.kb/models/macos-click-freeze/probes/2026-02-11-yabai-github-issues-search.md` — yabai: drag freeze (#2715) closest match, no click freeze

**Issues:**
- `orch-go-uvtgi` [P2] — Click freeze tracking issue

**Related models:**
- None (macOS system issue, not orch-go)

**Related issues (side-findings):**
- OpenCode fork optimizations: MAX_INSTANCES 20→8, IDLE_TTL 30→5min, disposeAll in server.stop(), periodic eviction timer (worth doing regardless — may also fix click freeze if H4 confirmed)
