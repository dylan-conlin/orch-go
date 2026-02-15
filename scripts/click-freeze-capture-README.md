# Click Freeze Reactive Capture Script

**Purpose:** Capture full system state snapshots for diagnosing the macOS click freeze issue.

**Context:** See `.kb/models/macos-click-freeze.md` for the complete investigation. H6 (aggregate service contention) is the leading hypothesis — we need correlation data from multiple freeze occurrences to identify which services trigger the problem.

## Usage

### Manual Trigger (Current)

```bash
~/Documents/personal/orch-go/scripts/click-freeze-capture.sh
```

**When to trigger:** As soon as you notice clicks stop registering (before running `sudo killall -HUP WindowServer`).

### Output

Captures are stored in timestamped files:
```
~/.orch/click-freeze-captures/capture-YYYYMMDD-HHMMSS.log
```

Each capture contains:
- Process state (suspect services, full process list)
- LaunchAgent/Daemon disabled/enabled state
- Memory metrics (memory_pressure, vm_stat)
- IOKit HID device registry
- WindowServer connections and stats
- Accessibility API clients
- Docker/Colima state (if running)
- System uptime and load

### Captured Data Sections

1. **PROCESS STATE**
   - Suspect service processes (yabai, skhd, Karabiner, Docker, etc.)
   - Full process list (`ps aux`)

2. **LAUNCHCTL SERVICE STATE**
   - User LaunchAgents (disabled/enabled state)
   - System LaunchDaemons (disabled/enabled state)
   - Currently loaded services
   - LaunchAgent plist files present

3. **MEMORY STATE**
   - Memory pressure percentage
   - VM statistics (swapins, swapouts, page stats)
   - Top snapshot (memory usage)

4. **IOKIT HID STATE**
   - HID device registry (trackpad, keyboard, virtual devices)
   - System Extensions (DriverKit, e.g., Karabiner)

5. **WINDOWSERVER STATE**
   - WindowServer process info
   - WindowServer connections (via lsof)
   - CGEventTap listeners (services with event taps)

6. **ACCESSIBILITY API STATE**
   - Apps with Accessibility permissions (yabai, skhd, etc.)

7. **DOCKER/COLIMA STATE**
   - Docker version and containers (if running)
   - Colima status (if running)

8. **SYSTEM UPTIME & LOAD**
   - Uptime and load averages
   - Last reboot time

## Karabiner Hotkey Binding (Future)

**NOT YET CONFIGURED** — After manual testing validates the script works reliably, add this to `~/Documents/dotfiles/.config/karabiner/karabiner.json`:

```json
{
  "description": "Capture click freeze diagnostics (Cmd+Shift+F9)",
  "manipulators": [
    {
      "from": {
        "key_code": "f9",
        "modifiers": {
          "mandatory": ["command", "shift"]
        }
      },
      "to": [
        {
          "shell_command": "/Users/dylanconlin/Documents/personal/orch-go/scripts/click-freeze-capture.sh"
        }
      ],
      "type": "basic"
    }
  ]
}
```

Add this rule to the `rules` array in the Karabiner configuration.

**Recommended hotkey:** `Cmd+Shift+F9` (or any convenient key combo not already bound)

**Why Karabiner for trigger?** It's always running and can execute shell commands even when the click freeze is happening (keyboard still works during freeze).

## Workflow During Freeze

1. Notice clicks stop registering
2. Trigger capture:
   - **Manual:** Open Ghostty, run `~/Documents/personal/orch-go/scripts/click-freeze-capture.sh`
   - **Karabiner (future):** Press `Cmd+Shift+F9`
3. Wait for completion notification (macOS notification + terminal output)
4. Fix the freeze: `sudo killall -HUP WindowServer`
5. Review capture file for correlation patterns

## Analysis Strategy

After collecting **3-5 captures** from different freeze occurrences:

1. **Diff the PROCESS STATE sections** — which services are consistently present vs. variable?
2. **Check MEMORY STATE** — is memory pressure correlated with freezes?
3. **Compare LAUNCHCTL SERVICE STATE** — which services were enabled during each freeze?
4. **Review WINDOWSERVER connections** — which services had active connections?

Look for **common denominators** across all captures. H6 hypothesis predicts a specific set of services will be present in every freeze.

## Files

- **Script:** `scripts/click-freeze-capture.sh`
- **Captures:** `~/.orch/click-freeze-captures/capture-*.log`
- **Model:** `.kb/models/macos-click-freeze.md`
- **Recent probe:** `.kb/models/macos-click-freeze/probes/2026-02-13-service-state-freeze-recurrence.md`

## Maintenance

**When to update the script:**
- New suspect services identified (add to pgrep patterns)
- Additional diagnostic commands discovered
- macOS version changes break existing commands

**Capture retention:** Keep captures until freeze is solved. Each capture is ~2-3MB. After solution, archive to investigation file or delete.
