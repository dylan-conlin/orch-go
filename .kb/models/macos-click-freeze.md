# Model: macOS Click Freeze

**Domain:** macOS input subsystem — trackpad click events stop registering while cursor movement continues
**Last Updated:** 2026-02-11
**Synthesized From:** Session 11 (systematic elimination), Session 14 (recurrence + 3 research probes + OpenCode fork resource audit), Session 15 (nuclear elimination — Karabiner uninstalled, 23+ services disabled)

---

## Summary (30 seconds)

Trackpad clicks stop registering every ~15 minutes while cursor movement and keyboard continue working. `sudo killall -HUP WindowServer` fixes it every time (HUP = reconfigure, not restart). This points to WindowServer accumulating corrupted state in its click event pipeline. **Breakthrough in Session 15:** nuclear elimination of ~23 services stopped the freeze. Karabiner reinstalled (upgraded 14.13→15.9.0) — **no freeze with Karabiner running**, further confirming H2 elimination. NI HardwareAgent and Ollama fully uninstalled (not just disabled). System stable under full agent workload (3 concurrent spawns + Karabiner + OpenCode + orch). **Still narrowing:** ~18 user LaunchAgents + 3 system LaunchDaemons remain disabled. Next: re-enable in phases to isolate culprit.

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
| Karabiner DriverKit | Kernel-level input interception | ✅ **Eliminated** — uninstalled (freeze persisted), reinstalled 15.9.0 (no freeze) |
| WindowServer | Routes events to apps | Partially — it's WHERE the problem manifests (HUP fixes it) |
| yabai | Window management, event interception | ✅ **Eliminated** — freeze recurred with yabai fully stopped |
| skhd | Hotkey daemon, event interception | 🔄 Re-enabled — testing individually (Session 15 Phase 2) |
| borders | Window border drawing | ⚠️ Disabled in batch — not individually tested |
| sketchybar | Status bar | ⚠️ Disabled in batch — not individually tested |
| NI HardwareAgent | Native Instruments audio HID service (root) | ✅ **Fully uninstalled** — top suspect, IOKit HID layer, ran as root |
| Ollama | LLM inference server | ✅ **Fully uninstalled** — memory pressure contributor |
| ~20 LaunchAgents | Various background services | ⚠️ Disabled in batch — see Session 15 elimination record |

---

## Why This Fails

### Hypothesis 1: yabai event interception corrupts WindowServer state — ELIMINATED

yabai uses the macOS Accessibility API to manage windows. Freeze recurred with yabai fully stopped (`yabai --stop-service`, confirmed no process via `pgrep`).

**Evidence against:** Freeze recurred within ~30 minutes with yabai completely stopped. No yabai process running. Definitively eliminated.

### Hypothesis 2: Karabiner DriverKit drops click events at kernel level — ELIMINATED

Karabiner was fully uninstalled (app removed, DriverKit extension gone, no IOKit registry entries, no LaunchAgents). Freeze persisted immediately after fresh reboot with no Karabiner components present.

**Evidence against:** Completely uninstalled — no process, no DriverKit extension (`systemextensionsctl list` clean), no IOKit entries (`ioreg` clean), no LaunchAgents. Freeze still occurred immediately post-reboot. Definitively eliminated.

### Hypothesis 3: WindowServer internal corruption (no external cause) — WEAKENED

macOS 15.6.1 (Sequoia) may have a bug where WindowServer's click event routing table gets corrupted over time. This would be independent of any third-party software.

**Evidence for:** Would explain why eliminating multiple apps in Session 11 didn't fix it.

**Evidence against:** Three research probes (2026-02-11) searched GitHub, Reddit, Apple Discussions exhaustively — **zero matching reports** for this symptom pattern (clicks stop, cursor moves, HUP fixes). If this were a Sequoia bug, community reports would exist. This significantly weakens H3.

### Hypothesis 4: Memory pressure from OpenCode instance accumulation — WEAKENED

OpenCode accumulates instances (with LSP/MCP/file watchers) per unique project directory. Each instance costs 300-500MB for LSP alone.

**Evidence against (Session 15):**
- Freeze occurred **immediately after fresh reboot** — memory was abundant, OpenCode hadn't even started yet
- This strongly suggests memory pressure is NOT the primary cause
- After disabling ~23 services, OpenCode + orch + 3 concurrent agents ran fine with no freeze

**Evidence for (still relevant):**
- Memory pressure may be a contributing factor that lowers the threshold for the real culprit
- System was at 607MB free when freeze occurred in Session 14

**Status:** Weakened as primary cause. May be contributing factor. OpenCode tuning still worth doing regardless.

**OpenCode tuning (worth doing regardless):** Reduce MAX_INSTANCES 20→8, IDLE_TTL 30min→5min for headless mode, add disposeAll to server.stop(), add periodic eviction timer. See `~/Documents/personal/opencode/.kb/investigations/2026-02-11-inv-opencode-fork-resource-audit-investigate.md`.

### Hypothesis 5: NI HardwareAgent (Native Instruments) corrupts IOKit HID state — STRONG SUSPECT (UNINSTALLED)

Native Instruments NIHardwareAgent ran as root via system LaunchDaemon. Audio hardware services enumerate HID devices (MIDI controllers, control surfaces) which registers them on the same IOKit HID bus as the trackpad. If NI's agent periodically re-enumerates or refreshes HID device state, it could corrupt WindowServer's click event routing.

**Evidence for:**
- Operated at IOKit HID layer — same bus as trackpad
- Ran as root (kernel-level access)
- Audio HID services register virtual devices that share the input pipeline
- Was killed as part of the nuclear batch that stopped the freeze
- Never previously tested in isolation
- **Fully uninstalled** — system remains stable with Karabiner reinstalled + agents running

**Evidence against:**
- Killed as part of a batch (~23 services) — not individually isolated before removal
- No known reports of NI causing click freeze specifically
- Cannot re-test since fully uninstalled (would need to reinstall to confirm)

**Status:** Fully uninstalled. If freeze never returns, H5 remains the strongest candidate by elimination. Cannot be definitively confirmed without reinstall test (not worth it).

### Hypothesis 6: Aggregate service load / event contention — POSSIBLE

Not a single culprit but the combination of many services (skhd, sketchybar, Ollama, NI, orch daemon, various LaunchAgents) creating enough IOKit/WindowServer event contention to corrupt click routing state. No single service triggers it, but the aggregate does.

**Evidence for:**
- Individual elimination of Karabiner, yabai, skhd (in earlier sessions) didn't fix it
- Only the nuclear "disable everything" approach worked
- Would explain why no single culprit was found in Sessions 11-14

**Evidence against:**
- The system ran fine with this same service set for months/years before the freeze started
- Something specific likely changed (XProtect update Feb 10? macOS update? NI update?)

**Next test:** Binary search through disabled services to narrow down.

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

**Current state:** Nuclear elimination worked. ~23 services disabled, NI HardwareAgent killed, Ollama killed. No freeze under full agent workload (3 concurrent spawns + OpenCode + orch).

**Phase 1: Re-enable safe services (low risk)**
Re-enable batch: disk-cleanup, disk-threshold, opencode-prune, aider-cleanup, redis, mysql, dnsmasq, nginx, php, tmuxinator, emacs
- If freeze returns → one of these is involved (unlikely)
- If clean → proceed to Phase 2

**Phase 2: Re-enable workflow services**
Re-enable batch: skhd, sketchybar, yabai (Dylan's core workflow)
- If freeze returns → one of these (test individually)
- If clean → proceed to Phase 3

**Phase 3: Re-enable orch/claude services**
Re-enable batch: orch daemon, orch reap, claude-docs-sync, claude-version-monitor, reprocess-skills, artifact-watcher
- If freeze returns → one of these
- If clean → proceed to Phase 4

**Phase 4: The remaining suspects**
Re-enable ONE AT A TIME, wait 20+ min each:
1. ~~NI HardwareAgent~~ — FULLY UNINSTALLED
2. ~~Ollama~~ — FULLY UNINSTALLED
3. Docker/Colima
4. Zoom daemon
5. Google updaters

**Current approach:** NI and Ollama permanently removed. If freeze stays gone after re-enabling phases 1-3, the culprit was NI or Ollama (or their combination). If freeze returns during phases 1-3, it's something else.

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
- **Karabiner:** 15.9.0 (reinstalled, upgraded from 14.13.0 — running, no freeze)
- **yabai:** /opt/homebrew/bin/yabai (disabled via launchctl, not running)
- **skhd:** /opt/homebrew/bin/skhd (disabled via launchctl, not running)
- **NI HardwareAgent:** FULLY UNINSTALLED (was com.native-instruments.NativeAccess.Helper2)
- **Ollama:** FULLY UNINSTALLED (was /Applications/Ollama.app)

---

## Evolution

**2026-02-11 (Session 11):** First systematic investigation. 4 freezes in ~1 hour. Eliminated 5 apps (BTT, Hammerspoon, Shortcat, middleClick, Raycast). BTT was a red herring (correlated but not causal).

**2026-02-11 (Session 14):** **yabai eliminated** — freeze recurred with yabai fully stopped (confirmed no process). Three research probes searched GitHub (yabai, Karabiner, broad), Reddit, Apple Discussions — zero matching reports found anywhere. Hypothesis 3 (macOS bug) significantly weakened. New H4: memory pressure — system at 35GB/36GB, 607MB free, OpenCode alone 8.4GB.

**2026-02-11 (Session 14, continued):** OpenCode fork resource audit found fork is better than upstream (LRU/TTL eviction added Feb 7). But params too high for orchestrator (MAX_INSTANCES=20, IDLE_TTL=30min). OpenCode grew from 336MB → 3.5GB in 15 min, was at 8.6GB before restart. Restarted OpenCode, freed 8.3GB. **H4 test in progress** — no freeze since restart. Audit agent prematurely eliminated H4 based on a point-in-time memory snapshot, not when freeze actually occurred. Both H2 (Karabiner) and H4 (memory pressure) remain active hypotheses.

**2026-02-11 (Session 15, evening):** **Major breakthrough.** Karabiner fully uninstalled — freeze persisted immediately after reboot (H2 eliminated). H4 weakened — freeze on fresh reboot with abundant RAM. Nuclear elimination: disabled ~23 LaunchAgents/Daemons via `launchctl disable`, killed NI HardwareAgent (root process) and Ollama. **Freeze stopped.** System ran clean under full agent workload (3 concurrent spawns + OpenCode + orch dashboard). New hypotheses: H5 (NI HardwareAgent — IOKit HID layer, top suspect) and H6 (aggregate service contention). Next: binary search through disabled services to isolate culprit.

Services disabled in Session 15:
- **User LaunchAgents (18):** agentmail, artifact-watcher, colima, claude-docs-sync, living-instruction-evolution, google-updater (3), orch-daemon, orch-reap, claude-version-monitor, reprocess-skills, tmuxinator, emacs-plus@29, emacs-plus@31, mysql, redis, dbus-session
- **User LaunchAgents (3, already):** skhd, yabai, sketchybar
- **System LaunchDaemons (5):** NI NativeAccess.Helper2, docker.socket, docker.vmnetd, xquartz, ZoomDaemon
- **Killed processes:** NIHardwareAgent, Ollama
- **Permanently uninstalled:** NI HardwareAgent (all files removed), Ollama (app + ~/.ollama removed)
- **Re-enabled:** Karabiner-Elements 15.9.0 (upgraded from 14.13.0) — running with DriverKit active, no freeze
- **Phase 1 re-enabled:** disk-cleanup, disk-threshold, mysql, redis, tmuxinator — no freeze
- **Phase 2 in progress:** skhd re-enabled and running — testing individually before yabai and sketchybar

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
