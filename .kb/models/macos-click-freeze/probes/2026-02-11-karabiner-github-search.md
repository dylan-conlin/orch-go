# Probe: Karabiner-Elements GitHub Issues for Click Freeze Reports

**Model:** macos-click-freeze
**Date:** 2026-02-11
**Status:** Complete

---

## Question

Are there GitHub issue reports in pqrs-org/Karabiner-Elements for click freeze, trackpad clicks not registering, mouse click events dropped, or WindowServer issues that might be related to macOS 15.6.1 Sequoia with DriverKit virtual HID devices?

---

## What I Tested

**Commands:**
```bash
gh search issues --repo pqrs-org/Karabiner-Elements "click freeze" --limit 100 --json number,title,state,url
gh search issues --repo pqrs-org/Karabiner-Elements "trackpad clicks not registering" --limit 100 --json number,title,state,url
gh search issues --repo pqrs-org/Karabiner-Elements "mouse click dropped" --limit 100 --json number,title,state,url
gh search issues --repo pqrs-org/Karabiner-Elements "WindowServer" --limit 100 --json number,title,state,url
gh search issues --repo pqrs-org/Karabiner-Elements "DriverKit" --limit 100 --json number,title,state,url
gh search issues --repo pqrs-org/Karabiner-Elements "click" --limit 100 --json number,title,state,url
gh search issues --repo pqrs-org/Karabiner-Elements "mouse freeze" --limit 100 --json number,title,state,url
gh search issues --repo pqrs-org/Karabiner-Elements "input lag" --limit 100 --json number,title,state,url
gh search issues --repo pqrs-org/Karabiner-Elements "Sequoia" --limit 100 --json number,title,state,url
gh search issues --repo pqrs-org/Karabiner-Elements "M3" --limit 50 --json number,title,state,url
gh search issues --repo pqrs-org/Karabiner-Elements "15.6" --limit 50 --json number,title,state,url

# Detailed inspection of key issues
gh issue view 2017 --repo pqrs-org/Karabiner-Elements  # WindowServer crash
gh issue view 4211 --repo pqrs-org/Karabiner-Elements  # Trackpad click modifications
gh issue view 2755 --repo pqrs-org/Karabiner-Elements  # Repeated clicks
gh issue view 2566 --repo pqrs-org/Karabiner-Elements  # Mouse lag
gh issue view 2895 --repo pqrs-org/Karabiner-Elements  # DriverKit VirtualHIDDevice
```

**Environment:**
- Target repo: pqrs-org/Karabiner-Elements
- Search scope: Both open and closed issues
- Date: 2026-02-11

---

## What I Observed

**Output:**

### Click Freeze Searches (Direct)
- "click freeze": **1 result** - #2035 "Keyboard freezing forcing a restart" (open, not click-specific)
- "trackpad clicks not registering": **0 results**
- "mouse click dropped": **0 results**
- "mouse freeze": **2 results** - #2114 (caps lock related), #2035 (duplicate)

### Related Input Issues
- "click" (general): **100 results** including:
  - #4367 "Force Click as right click" (open)
  - #2753 "ctrl+left click to left click is not working 100% of the time" (open)
  - #4211 "Built in Trackpad click modification issues" (closed) - M1 Pro, Sequoia 15.5
  - #2157 "Remapping Left click does not work on the trackpad" (open)
  - #2566 "Having mouse enabled in Karabiner introduces heavy intermittent mouse lag" (open, 36 comments)
  
- "input lag": **5 results**
  - #2566 (most detailed) - intermittent heavy mouse lag when mouse device enabled in Karabiner
  - #234, #962 (both closed, older versions)

### WindowServer Issues
- **10 results** from "WindowServer" search:
  - #2017 "Karabiner-Elements crashes OSX WindowServer" (closed) - macOS Catalina 10.15, fixed
  - #2880 "Karabiner keeps M1 Mac from sleeping" (open)
  - #2788 "Major lagging whenever I press caps lock" (closed)
  - #3254 "scrolling delays in osx Ventura" (closed)

### DriverKit Issues
- **100 results** from "DriverKit" search (many):
  - #2895 "DriverKit-VirtualHIDDevice does not completely hide the true underlying keyboard devices" (closed) - Discord hotkey issues
  - #3998 "Karabiner-DriverKit-VirtualHIDDevice active completes but no device show" (open)
  - #3022 "Allow Karabiner VirtualHIDDevice Manager alert is shown every time" (open)
  - #4048 "grabber_client connect_failed: Connection refused on Sequoia" (open)
  - #3941 "Unable to allow Karabiner on Driver Extensions - Sequoia 15.0" (open)
  - #4314 "Karabiner driver not loading after macOS Tahoe 26.1 update" (open)

### Sequoia-Specific Issues
- **100 results** from "Sequoia" search:
  - #4048 "grabber_client connect_failed: Connection refused on Sequoia" (open)
  - #3941 "Unable to allow Karabiner on Driver Extensions - Sequoia 15.0" (open)
  - #4032 "ISO keyboard keys swapped" (open)
  - #3945 "Karabiner starts sending streams of unwanted keystrokes" (closed)
  - #4314 "Karabiner driver not loading after macOS Tahoe 26.1 update" (open)

### M3-Specific Issues
- **43 results** from "M3" search:
  - Mostly keyboard layout issues (ISO/ANSI key swapping)
  - #4043 "ticked 'Modify events' on mouse is causing huge mouse lag with any cpu spike" (open)
  - #3908 "Internal keyboard not disabled on M3 Macbook Pro with macOS Sequoia Beta" (open)
  - No specific click freeze reports for M3

### macOS 15.6 Specific
- **4 results** from "15.6" search:
  - #4316 "Moving cursor closes Alt+Tab switching apps when using Karabiner" (open)
  - #4326 "Key remapping doesn't work since update to 15.7.0" (open)
  - #4314 "Karabiner driver not loading after macOS Tahoe 26.1 update" (open)

**Key observations:**
1. **No direct "click freeze" reports** matching the symptom description
2. **Mouse lag is documented** (#2566, #4043) but described as "lag/buffering" rather than "freeze"
3. **One WindowServer crash** (#2017) but from Catalina era, closed as fixed
4. **DriverKit issues are common** but mostly about driver loading/activation, not click events
5. **Sequoia has driver extension permission issues** (#3941, #4048, #4314)
6. **Trackpad click modification issues** (#4211) show that enabling click modifications can break trackpad functionality
7. **Most mouse issues** (#2566, #2755) describe repeated clicks or lag when mouse is enabled in Karabiner, not freeze

---

## Model Impact

**Verdict:** extends — confirms known mouse/input lag issues with Karabiner but no specific click freeze reports

**Details:**
The search found **extensive documentation of input lag and mouse issues** with Karabiner-Elements, but **no reports specifically matching "click freeze"** symptoms. The closest matches are:
1. **Mouse lag/buffering** (#2566) - intermittent heavy lag when mouse device is enabled in Karabiner, with 36 comments indicating this is a known issue
2. **Repeated clicks** (#2755) - clicks registering multiple times, but not failing to register
3. **Trackpad functionality loss** (#4211) - enabling click modifications breaks native trackpad features

The model assumption that "Karabiner might be involved" is partially supported by evidence of **mouse input issues when devices are enabled in Karabiner**, but the specific "freeze" pattern is not documented in the issue tracker.

**DriverKit-specific findings:**
- No reports of DriverKit causing click freeze
- DriverKit issues are primarily about driver activation/loading (#3941, #4048, #4314)
- One issue (#2895) about DriverKit not properly hiding underlying devices, but affects keyboard hotkeys, not mouse clicks

**Sequoia/M3 findings:**
- Sequoia issues are mostly driver permission/loading problems
- M3 issues are mostly keyboard layout problems
- No click freeze reports specific to Sequoia 15.6.1 or M3 hardware

**Confidence:** High — Searched 11 different query terms, reviewed 100+ issues, inspected detailed comments on 5 key issues. The absence of click freeze reports in a repo with 4000+ issues and active community is significant.
