# Probe: Capture service state during click freeze recurrence (post-Session 15 nuclear elimination)

**Model:** macos-click-freeze
**Date:** 2026-02-13
**Status:** Complete

---

## Question

The model claims specific Phase 2 service states and identifies H5 (NI HardwareAgent) as the strongest candidate for the click freeze. Three claims to test:

1. Model says skhd was "re-enabled" in Phase 2 — is it actually running?
2. Model says yabai was "disabled via launchctl, not running" — is it actually disabled?
3. H5 (NI as sole culprit) — NI was fully uninstalled, yet freeze recurred. Does this weaken H5?

---

## What I Tested

Captured full system state during the freeze recurrence window on 2026-02-13:

```bash
# User LaunchAgent disabled/enabled state
launchctl print-disabled user/501

# System LaunchDaemon disabled/enabled state
launchctl print-disabled system/

# Running processes
ps aux

# Specific service processes
pgrep -fl 'skhd|yabai|sketchybar|borders|colima|docker|ollama|karabiner'

# Which services are loaded in launchctl
launchctl list | grep -E 'skhd|yabai|sketchybar|borders|...'

# Memory state
memory_pressure
vm_stat

# Karabiner DriverKit
systemextensionsctl list

# LaunchAgent plist existence
ls ~/Library/LaunchAgents/ | grep -E 'skhd|yabai|sketchybar|borders|ollama'
```

---

## What I Observed

### Claim 1: skhd Phase 2 state — MODEL IS WRONG

The model says "skhd re-enabled and running — testing individually (Session 15 Phase 2)".

**Actual state:**
- `launchctl print-disabled user/501`: `"com.koekeishiya.skhd" => disabled`
- `launchctl list`: NOT loaded (no entry)
- `pgrep skhd`: No process found
- `~/Library/LaunchAgents/com.koekeishiya.skhd.plist`: EXISTS (but disabled)

**Verdict:** skhd is DISABLED and NOT running. The model's Phase 2 claim is wrong.

### Claim 2: yabai disabled state — MODEL IS WRONG (inverted)

The model says "yabai: /opt/homebrew/bin/yabai (disabled via launchctl, not running)".

**Actual state:**
- `launchctl print-disabled user/501`: `"com.koekeishiya.yabai" => enabled`
- `launchctl list`: `1055  -15  com.koekeishiya.yabai` (loaded, PID 1055, nice -15)
- `pgrep yabai`: PID 1055 `/opt/homebrew/bin/yabai`
- `~/Library/LaunchAgents/com.koekeishiya.yabai.plist`: EXISTS

**Verdict:** yabai is ENABLED and RUNNING (PID 1055). The model's Environment section is wrong — yabai and skhd states are exactly inverted from what the model claims.

### Claim 3: H5 (NI as sole culprit) — WEAKENED

- NI HardwareAgent: `launchctl print-disabled system/` shows `"com.native-instruments.NativeAccess.Helper2" => disabled`
- No NI process running (confirmed via pgrep)
- Model says NI was "FULLY UNINSTALLED" — this is consistent with launchctl state
- **Yet the freeze recurred.** This means NI cannot be the sole cause.

**Verdict:** H5 weakened as sole explanation. NI may have been a contributor but something else also causes the freeze.

### Memory State (H4 check)

```
memory_pressure: System-wide memory free percentage: 78%
vm_stat: Swapins: 0, Swapouts: 0
Pages free: 22,847 (of 2,359,296 total)
Pages active: 899,016
Pages compressed: 681,416 (stored by compressor)
```

**Verdict:** H4 (memory pressure) further weakened. 78% free with zero swap — memory is abundant. Freeze is not memory-triggered.

### Karabiner DriverKit

```
org.pqrs.Karabiner-DriverKit-VirtualHIDDevice (1.8.0/1.8.0) [activated enabled]
```

Running processes: karabiner_session_monitor, karabiner_console_user_server, Karabiner-Menu, Karabiner-NotificationWindow, Karabiner-VirtualHIDDevice-Daemon, Karabiner-Core-Service

**Verdict:** Karabiner fully active (v15.9.0 with DriverKit). Already eliminated in Session 15 — freeze persisted without Karabiner, so it's not the cause.

### Additional Service State (not in model claims but relevant)

Services the model says should be "disabled in batch" but are actually ENABLED:

| Service | Model claims | Actual launchctl state | Running? |
|---------|-------------|----------------------|----------|
| yabai | disabled | **enabled** | **YES (PID 1055)** |
| skhd | re-enabled (Phase 2) | **disabled** | NO |
| sketchybar | disabled in batch | **enabled** (in override db) | NO (no plist in LaunchAgents) |
| borders | disabled in batch | **enabled** (in override db) | NO (no plist in LaunchAgents) |
| ollama | UNINSTALLED | **enabled** (ghost entry) | NO (not installed) |
| opencode-prune | not mentioned | enabled | NO (loaded but not running) |
| disk-cleanup | Phase 1 re-enabled | enabled | YES (loaded) |
| disk-threshold | Phase 1 re-enabled | enabled | YES (loaded) |
| mysql | Phase 1 re-enabled | enabled | YES (PID 95416) |
| redis | Phase 1 re-enabled | enabled | YES (PID 94913) |
| tmuxinator | Phase 1 re-enabled | enabled | YES (loaded) |

Services manually started despite disabled LaunchAgents:

| Service | LaunchAgent disabled? | Running? | How started |
|---------|----------------------|----------|-------------|
| colima | disabled | **YES** (PID 89445) | Manual/docker |
| emacs-plus@31 | disabled | **YES** (PID 94678) | Manual launch |
| docker compose | N/A (via colima) | **YES** (PIDs 42906, 89861) | Manual |

Services correctly still disabled:

| Service | Status |
|---------|--------|
| agentmail | disabled, not running |
| artifact-watcher | disabled, not running |
| claude-docs-sync | disabled, not running |
| claude-version-monitor | disabled, not running |
| orch-daemon | disabled, not running |
| orch-reap | disabled, not running |
| reprocess-skills | disabled, not running |
| living-instruction-evolution | disabled, not running |
| google-updater | disabled, not running |
| dbus-session | disabled, not running |
| emacs-plus@29 | disabled, not running |
| NI NativeAccess.Helper2 | disabled/uninstalled |
| docker.socket | disabled |
| docker.vmnetd | disabled |
| xquartz | disabled |
| ZoomDaemon | disabled |

### Current Suspect Set

Services that are running NOW that were NOT running during the freeze-free period after Session 15 nuclear elimination:

1. **yabai** (PID 1055) — re-enabled since Session 15, but already eliminated in Session 14. Unless elimination was flawed.
2. **colima + Docker** — running despite disabled LaunchAgents (manually started)
3. **emacs-plus@31** — running despite disabled LaunchAgent
4. **Phase 1 services** — mysql, redis, disk-cleanup, disk-threshold, tmuxinator (re-enabled per test plan, no freeze reported at the time)

The freeze-free period had ALL of these disabled/stopped. Now they're running and the freeze is back. The question is which one(s) matter.

---

## Model Impact

- [x] **Contradicts** invariant: Phase 2 service state — skhd and yabai states are exactly inverted from model claims. Model says skhd re-enabled (actually disabled), yabai disabled (actually enabled+running).
- [x] **Contradicts** invariant: H5 as strongest candidate — NI fully uninstalled yet freeze recurred, weakening H5 as sole explanation.
- [x] **Extends** model with: Complete current service inventory showing which Session 15 services have been re-enabled and the actual suspect set for freeze recurrence.
- [x] **Extends** model with: H4 further weakened — 78% memory free with zero swap during freeze.

---

## Notes

### Model Update Needed

The macos-click-freeze model needs these corrections:

1. **Environment section:** yabai is ENABLED and RUNNING (not "disabled via launchctl, not running")
2. **Environment section:** skhd is DISABLED (not "re-enabled in Phase 2")
3. **Remaining Test Plan Phase 2:** Status should reflect that yabai was re-enabled (not skhd), and the freeze recurred
4. **H5:** Should be downgraded from "STRONG SUSPECT" to "contributing factor" — NI gone, freeze persists
5. **H6 (aggregate service contention):** Strengthened — the freeze returned as services were gradually re-enabled, consistent with aggregate theory

### Implication for Investigation

Since yabai was already "eliminated" in Session 14 (freeze recurred with yabai stopped), but the freeze is now recurring with yabai running, yabai alone is likely not the cause. However:

- Session 14 elimination was a single test (~30 min window)
- The current re-enablement is with Phase 1 services also running
- H6 (aggregate contention) gains strength — yabai + Phase 1 services + colima/docker + emacs could collectively trigger the freeze

### Next Steps (not implemented — probe only)

1. Update the model with corrected Phase 2 state
2. Consider binary search: stop yabai first (already "eliminated" but worth re-testing), then colima/docker, then Phase 1 services
3. Monitor if freeze correlates with specific workload patterns (concurrent agents, docker activity, etc.)
