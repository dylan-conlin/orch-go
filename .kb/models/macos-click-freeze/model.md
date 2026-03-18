# Model: macOS Click Freeze

**Domain:** macOS input subsystem — trackpad click events stop registering while cursor movement continues
**Last Updated:** 2026-03-18
**Synthesized From:** Session 11 (systematic elimination), Session 14 (recurrence + 3 research probes + OpenCode fork resource audit), Session 15 (nuclear elimination — Karabiner uninstalled, 23+ services disabled), Session 16 (freeze recurrence — service state probe), Session 17 (stability observation + reactive capture tooling)

---

## Summary (30 seconds)

Trackpad clicks stop registering while cursor movement and keyboard continue working. `sudo killall -HUP WindowServer` fixes it every time (HUP = reconfigure, not restart). This points to WindowServer accumulating corrupted state in its click event pipeline. **Breakthrough in Session 15:** nuclear elimination of ~23 services stopped the freeze. **Freeze returned (2026-02-13)** after gradual service re-enablement — first recurrence in ~2 days. **However (2026-02-14):** the same service set that triggered the Feb 13 freeze ran stable for 5+ hours, suggesting the freeze is **intermittent/stochastic** rather than deterministic. Frequency has decreased from every ~15 min (Sessions 11-14) to rare occurrences. macOS updated to 15.7.4 (from 15.6.1) between sessions. **H6 (aggregate service contention) remains the leading hypothesis** but is weakened by the stability observation. **New approach:** reactive capture script (`scripts/click-freeze-capture.sh`) to snapshot full system state during freeze occurrences for correlation analysis, rather than disruptive elimination testing.

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
| yabai | Window management, event interception | ⚠️ Previously eliminated (Session 14) but freeze recurred with it running (Session 16). Re-testing needed. |
| skhd | Hotkey daemon, event interception | ⚠️ Disabled since Session 15 — not running, but see H7: architectural analysis shows tap corruption mechanism |
| borders | Window border drawing | ⚠️ Disabled in batch — not individually tested |
| sketchybar | Status bar | ⚠️ Disabled in batch — not individually tested |
| NI HardwareAgent | Native Instruments audio HID service (root) | ✅ **Fully uninstalled** — top suspect, IOKit HID layer, ran as root |
| Ollama | LLM inference server | ✅ **Fully uninstalled** — memory pressure contributor |
| ~20 LaunchAgents | Various background services | ⚠️ Disabled in batch — see Session 15 elimination record |

---

## Why This Fails

### Hypothesis 1: yabai event interception corrupts WindowServer state — WEAKENED (RE-TESTING NEEDED)

yabai uses the macOS Accessibility API to manage windows. Freeze recurred with yabai fully stopped in Session 14 (`yabai --stop-service`, confirmed no process via `pgrep`).

**Evidence against:** Freeze recurred within ~30 minutes with yabai completely stopped (Session 14). No yabai process running. Seemed eliminated.

**Evidence for re-testing (Session 16):** The Session 14 elimination was a single 30-min test with everything else still running. The environment is now different — yabai is running with a reduced service set (NI/Ollama gone, many agents disabled). yabai may not be the sole cause but could be a necessary component of H6 (aggregate contention). Worth re-testing by stopping yabai in current environment.

### Hypothesis 2: Karabiner DriverKit drops click events at kernel level — ELIMINATED

Karabiner was fully uninstalled (app removed, DriverKit extension gone, no IOKit registry entries, no LaunchAgents). Freeze persisted immediately after fresh reboot with no Karabiner components present.

**Evidence against:** Completely uninstalled — no process, no DriverKit extension (`systemextensionsctl list` clean), no IOKit entries (`ioreg` clean), no LaunchAgents. Freeze still occurred immediately post-reboot. Definitively eliminated.

### Hypothesis 3: WindowServer internal corruption (no external cause) — WEAKENED

macOS 15.6.1 (Sequoia) may have a bug where WindowServer's click event routing table gets corrupted over time. This would be independent of any third-party software.

**Evidence for:** Would explain why eliminating multiple apps in Session 11 didn't fix it.

**Evidence against:** Three research probes (2026-02-11) searched GitHub, Reddit, Apple Discussions exhaustively — **zero matching reports** for this symptom pattern (clicks stop, cursor moves, HUP fixes). If this were a Sequoia bug, community reports would exist. This significantly weakens H3.

### Hypothesis 4: Memory pressure from OpenCode instance accumulation — EFFECTIVELY ELIMINATED

OpenCode accumulates instances (with LSP/MCP/file watchers) per unique project directory. Each instance costs 300-500MB for LSP alone.

**Evidence against (Session 15):**
- Freeze occurred **immediately after fresh reboot** — memory was abundant, OpenCode hadn't even started yet
- This strongly suggests memory pressure is NOT the primary cause
- After disabling ~23 services, OpenCode + orch + 3 concurrent agents ran fine with no freeze

**Evidence against (Session 16, 2026-02-13):**
- Freeze recurred with **78% memory free and zero swap** — memory is abundant
- This effectively eliminates memory pressure as a cause or significant contributor

**Status:** Effectively eliminated. Memory pressure is not a factor.

### Hypothesis 5: NI HardwareAgent (Native Instruments) corrupts IOKit HID state — WEAKENED (NOT SOLE CAUSE)

Native Instruments NIHardwareAgent ran as root via system LaunchDaemon. Audio hardware services enumerate HID devices (MIDI controllers, control surfaces) which registers them on the same IOKit HID bus as the trackpad. If NI's agent periodically re-enumerates or refreshes HID device state, it could corrupt WindowServer's click event routing.

**Evidence for:**
- Operated at IOKit HID layer — same bus as trackpad
- Ran as root (kernel-level access)
- Audio HID services register virtual devices that share the input pipeline
- Was killed as part of the nuclear batch that stopped the freeze
- Never previously tested in isolation

**Evidence against:**
- Killed as part of a batch (~23 services) — not individually isolated before removal
- No known reports of NI causing click freeze specifically
- Cannot re-test since fully uninstalled (would need to reinstall to confirm)
- **Freeze recurred (2026-02-13) with NI fully uninstalled** — NI cannot be the sole cause

**Status:** Weakened. NI may have been a contributor but is not the sole cause. Freeze recurred without it.

### Hypothesis 6: Aggregate service load / event contention — LEADING HYPOTHESIS

Not a single culprit but the combination of services creating enough IOKit/WindowServer event contention to corrupt click routing state. No single service triggers it alone, but a critical mass does.

**Evidence for:**
- Individual elimination of Karabiner, yabai, skhd (in earlier sessions) didn't fix it
- Only the nuclear "disable everything" approach worked
- Would explain why no single culprit was found in Sessions 11-14
- **Freeze returned (2026-02-13) as services were gradually re-enabled** — directly consistent with aggregate theory
- NI fully uninstalled yet freeze recurred — no single culprit
- Reduced service set (fewer than original) still triggered freeze — threshold is lower than "everything"

**Evidence against:**
- The system ran fine with this same service set for months/years before the freeze started
- Something specific likely changed (XProtect update Feb 10? macOS update? NI update?)
- **(2026-02-14)** Same suspect set from Feb 13 freeze ran stable for 5+ hours without a freeze. If aggregate contention were deterministic, this shouldn't happen. Suggests the threshold is probabilistic or requires an additional transient trigger.

**Current suspect set (running during 2026-02-13 freeze AND stable 2026-02-14):**
1. **yabai** (PID 9116) — re-enabled, Accessibility API + window event interception
2. **sketchybar** (PID 9743) — re-enabled since Feb 13 (was disabled in batch)
3. **colima + Docker** (manually started despite disabled LaunchAgents)
4. **emacs-plus@31** (manually started despite disabled LaunchAgent)
5. **Phase 1 services** — mysql, redis, disk-cleanup, disk-threshold, tmuxinator
6. **Karabiner** (15.9.0) — running, but already eliminated as sole cause

**Next approach (SUPERSEDED 2026-03-18):** ~~Reactive capture and binary search plans~~ — never executed. macOS upgraded to 26.3.1 (Tahoe), changing the investigation landscape. **Current recommendation:** Observe for freeze recurrence on macOS 26 before resuming any elimination testing. If no recurrence after 30+ days on macOS 26, consider archiving this model.

### Hypothesis 7: skhd CGEventTap pipeline corruption — ACTIVE (architectural analysis, not reproduction)

skhd registers an **active CGEventTap at HEAD position** (`kCGHeadInsertEventTap`, `kCGEventTapOptionDefault`) in the session event pipeline. This means all events flow past this tap point, even though skhd only registers a keyboard-only mask (`kCGEventKeyDown | NX_SYSDEFINED`).

**The corruption mechanism:**
1. macOS can disable any active tap that exceeds its callback timeout
2. skhd handles `kCGEventTapDisabledByTimeout` by immediately re-enabling via `CGEventTapEnable` — a rapid disable/re-enable at HEAD position
3. During the disable→re-enable cycle, WindowServer reconfigures event routing twice in quick succession
4. This pipeline reconfiguration can leave routing state for **non-masked event types** (mouse clicks) in an inconsistent state
5. The corruption persists after skhd is killed — only HUP (which triggers full WindowServer reconfiguration) clears it

**Why click events specifically, not move events:**
Move events (`kCGEventMouseMoved`) are high-frequency and processed on a fast path. Click events (`kCGEventLeftMouseDown/Up`) have additional window-targeting logic in WindowServer's routing table — the part most susceptible to inconsistent state.

**Amplifying factor:** 78 skhd bindings invoke `yabai -m window --focus`, and yabai has `focus_follows_mouse autofocus`. This creates a chain: skhd callback → fork → yabai focus change → WindowServer focus move → autofocus reacts → more events. This amplifies pipeline load during skhd-triggered actions, increasing tap callback timeout frequency.

**Apple bug FB12113281** (macOS 13.4+, unresolved): CGEvent taps can stop receiving events permanently under heavy app activity. skhd's HEAD position + active tap type + timeout cycling could trigger the same underlying WindowServer state corruption.

**Evidence for:**
- Architectural analysis of skhd source (`src/event_tap.c`, `src/skhd.c`) confirms HEAD-position active tap
- Explains all symptoms: click-specific freeze, cursor still works, HUP fixes it, killing skhd alone doesn't fix it (pipeline state already corrupted), skhd disabled since Session 15 and freeze has been less frequent/intermittent
- No community reports of click freeze — consistent with a configuration-specific trigger (requires skhd's specific tap + timeout cycling + heavy workload)

**Evidence against / limitations:**
- Not reproduced — architectural reasoning only
- skhd disabled since Session 15 but freeze still recurred (2026-02-13) — skhd alone is not sufficient; aggregate factors (yabai, colima, Phase 1 services) were running during that recurrence
- Definitive confirmation requires: (a) sustained freeze-free period with skhd disabled and same other services running, or (b) reproducing freeze by running `skhd -V` and correlating "restarting event-tap" messages with freeze onset

**Suggested next step:** Run `skhd -V` (verbose mode) in a test session and grep for "restarting event-tap" — frequency of timeout/re-enable cycles would confirm the corruption trigger rate.

**Interaction with H6:** H7 and H6 are compatible. skhd's tap cycling could be the specific mechanism through which aggregate service load crosses the contention threshold — heavy I/O from yabai/colima/emacs increases skhd callback latency, triggering timeout cycles, which corrupt the pipeline.

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

**Current state (2026-02-13):** Freeze recurred after gradual service re-enablement. NI and Ollama remain fully uninstalled. The original phased plan is superseded — freeze returned before Phase 2 was complete.

**What's currently running (verified via launchctl + ps, 2026-02-13):**
- ✅ Karabiner 15.9.0 (eliminated as cause)
- ✅ yabai (enabled, running PID 1055)
- ✅ Phase 1 services: disk-cleanup, disk-threshold, mysql, redis, tmuxinator
- ✅ Manually started: colima + Docker, emacs-plus@31

**What's still disabled:**
- skhd, sketchybar, borders (LaunchAgent disabled, no plist or not loaded)
- agentmail, artifact-watcher, claude-docs-sync, claude-version-monitor, orch-daemon, orch-reap, reprocess-skills, living-instruction-evolution, google-updater, dbus-session, emacs-plus@29
- System: NI (uninstalled), docker.socket, docker.vmnetd, xquartz, ZoomDaemon

**New plan: Binary search through running suspect set**

**Step 1: Stop yabai** — Quick re-test in reduced environment (was "eliminated" in Session 14 but environment is different now). Wait 30+ min.
- If freeze stops → yabai is a necessary component (even if not sole cause)
- If freeze persists → yabai still eliminated, proceed to Step 2

**Step 2: Stop colima/Docker** — Manually started, runs container networking + vmnet.
- If freeze stops → colima/Docker interaction
- If freeze persists → proceed to Step 3

**Step 3: Stop Phase 1 services** — mysql, redis, disk-cleanup, disk-threshold, tmuxinator.
- If freeze stops → one of these (test individually)
- If freeze persists → emacs or something not yet identified

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

- **macOS:** 26.3.1 (Tahoe, build 25D2128) — major upgrade from 15.7.4 Sequoia (between Session 17 and 2026-03-18 probe)
- **Hardware:** Mac15,7 (M3 Pro)
- **Karabiner:** Running, DriverKit active (ioreg confirmed PID 793)
- **yabai:** /opt/homebrew/bin/yabai (ENABLED, running)
- **skhd:** /opt/homebrew/bin/skhd (DISABLED, not running, service not loaded)
- **sketchybar:** Running (re-enabled since Session 17)
- **borders:** Running (re-enabled — was "disabled in batch, not individually tested")
- **colima/Docker:** Not running (was manually started during Session 16 freeze period)
- **NI HardwareAgent:** FULLY UNINSTALLED (was com.native-instruments.NativeAccess.Helper2)
- **Ollama:** FULLY UNINSTALLED (was /Applications/Ollama.app)

### macOS 26 Impact Note (2026-03-18)

The upgrade from macOS 15 (Sequoia) to macOS 26 (Tahoe) is a major version jump that may change the entire WindowServer and input event pipeline. All hypotheses (H3, H6, H7) and architectural reasoning about CGEventTap behavior, IOKit HID, and WindowServer internals are **unverified on macOS 26**. The freeze status since the OS upgrade is unknown — it may be resolved by OS changes. Apple bug FB12113281 (CGEvent tap event loss, macOS 13.4+) status on macOS 26 is unknown.

**Recommended:** Observe whether freeze recurs on macOS 26 before resuming investigation. If no recurrence in 30+ days, consider archiving this model.

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
- **Phase 1 re-enabled:** disk-cleanup, disk-threshold, mysql, redis, tmuxinator — no freeze at time of re-enablement

**2026-02-13 (Session 16):** **Freeze recurred** — first time since Session 15 nuclear elimination (~2 days freeze-free). Service state probe revealed model had Phase 2 states inverted: skhd was DISABLED (not re-enabled), yabai was ENABLED+running (not disabled). Additionally colima/Docker and emacs-plus@31 were manually started despite disabled LaunchAgents. H5 (NI) weakened — fully uninstalled yet freeze returned. H4 (memory) effectively eliminated — 78% free, zero swap. **H6 (aggregate contention) is now the leading hypothesis.** New binary search plan through current suspect set.

**2026-02-14 (Session 17):** **Stability observation.** Same suspect set from Feb 13 freeze (yabai, colima/Docker, emacs, Phase 1 services) ran stable for 5+ hours without a freeze — *plus* sketchybar (newly re-enabled). macOS updated to 15.7.4 (from 15.6.1). This weakens deterministic H6 — if aggregate contention were sufficient, the same stack should freeze reliably. Freeze frequency has decreased from every ~15 min (Sessions 11-14) to rare/intermittent. **Strategy shift:** built reactive capture script (`scripts/click-freeze-capture.sh`) to snapshot full system state during actual freeze occurrences for correlation analysis. Captures 10 sections: processes, launchctl state, memory, IOKit HID, WindowServer, Accessibility API, Docker/colima, uptime. Will bind to Karabiner hotkey after manual validation. This allows continued app usage while collecting diagnostic data.

**2026-03-18 (Knowledge decay probe):** **33-day verification.** Major finding: macOS upgraded from 15.7.4 (Sequoia) to 26.3.1 (Tahoe) — entire OS generation changed. Service states mostly match model (yabai running, skhd disabled, Karabiner active, NI/Ollama gone). borders re-enabled (was "disabled in batch"). colima/Docker not running. **Reactive capture script was never used** — zero captures collected, Karabiner hotkey binding never configured. Binary search plan from Session 16 was never executed. All hypotheses and architectural analysis are unverified on macOS 26. **Freeze recurrence status unknown** — requires user input. Investigation is effectively paused pending macOS 26 observation.

---

## References

**Investigations:**
- Session 11 handoff in `.orch/HANDOFF.md` — detailed elimination record
- `~/Documents/personal/opencode/.kb/investigations/2026-02-11-inv-opencode-fork-resource-audit-investigate.md` — OpenCode fork resource audit (eliminated H4, found optimization opportunities)

**Probes:**
- `.kb/models/macos-click-freeze/probes/2026-03-18-probe-knowledge-decay-33d-verification.md` — UPDATES: macOS 15.7.4→26.3.1 (major OS change); borders re-enabled; capture script never used; all hypotheses unverified on macOS 26; investigation effectively paused

**Probes (merged 2026-03-06):**
- `.kb/models/macos-click-freeze/probes/2026-02-13-service-state-freeze-recurrence.md` — CONTRADICTS: corrected Phase 2 service states (skhd/yabai inverted); WEAKENS H5 (NI uninstalled, freeze recurred); ELIMINATES H4 (78% memory free, zero swap during freeze); STRENGTHENS H6 (aggregate theory)
- `.kb/models/macos-click-freeze/probes/2026-02-12-skhd-event-tap-source-analysis.md` — EXTENDS: adds H7 (skhd HEAD-position active CGEventTap with timeout/re-enable cycling corrupts WindowServer routing state for non-masked event types including mouse clicks); architectural source analysis, not yet reproduced
- `.kb/models/macos-click-freeze/probes/2026-02-11-github-apple-support-search.md` — CONFIRMS: zero matching reports across GitHub and Reddit; strengthens case that H3 (macOS bug) is unlikely and issue is configuration-specific
- `.kb/models/macos-click-freeze/probes/2026-02-11-karabiner-github-search.md` — CONFIRMS: no click freeze reports in Karabiner repo (4000+ issues, 11 query terms); documents mouse lag (#2566, #4043) as distinct symptom; no DriverKit click corruption reports
- `.kb/models/macos-click-freeze/probes/2026-02-11-yabai-github-issues-search.md` — CONFIRMS: no click freeze reports in yabai repo; EXTENDS with WindowServer crash via scripting additions + Sidecar (#2573) and window drag freeze (#2715) as distinct related symptoms

**Tooling:**
- `scripts/click-freeze-capture.sh` — Reactive capture script (run during freeze, before HUP). Outputs to `~/.orch/click-freeze-captures/capture-YYYYMMDD-HHMMSS.log`. **Never used** — zero captures collected, Karabiner hotkey binding never configured (as of 2026-03-18).

**Issues:**
- `orch-go-uvtgi` [P2] — Click freeze tracking issue
- `orch-go-cem` — Reactive capture script (completed 2026-02-14)

**Related models:**
- None (macOS system issue, not orch-go)

**Related issues (side-findings):**
- OpenCode fork optimizations: MAX_INSTANCES 20→8, IDLE_TTL 30→5min, disposeAll in server.stop(), periodic eviction timer (worth doing regardless — may also fix click freeze if H4 confirmed)

**Primary Evidence (Verify These):**
- `~/Library/LaunchAgents/` - User launch agents (skhd disabled, yabai enabled; others per session state)
- `/Library/LaunchDaemons/` - System daemons (NI, Docker components uninstalled/disabled)
- WindowServer process - macOS input event routing (HUP signal fixes)
- `launchctl list | grep` output - Service state verification
- `ioreg` output - IOKit HID device registration (Karabiner DriverKit presence)
- Activity Monitor - Memory pressure metrics during freeze
