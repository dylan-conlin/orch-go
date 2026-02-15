#!/bin/bash
# Click Freeze Reactive Capture Script
#
# Purpose: Capture full system state for click freeze diagnosis
# Trigger: Manual (later via Karabiner hotkey)
# Output: Timestamped snapshot in ~/.orch/click-freeze-captures/
#
# Context: macOS click freeze investigation (H6 - aggregate service contention)
# Model: .kb/models/macos-click-freeze.md

set -euo pipefail

# Configuration
CAPTURE_DIR="$HOME/.orch/click-freeze-captures"
TIMESTAMP=$(date +%Y%m%d-%H%M%S)
CAPTURE_FILE="$CAPTURE_DIR/capture-$TIMESTAMP.log"

# Ensure capture directory exists
mkdir -p "$CAPTURE_DIR"

# Header
{
	echo "======================================================================"
	echo "CLICK FREEZE CAPTURE - $TIMESTAMP"
	echo "======================================================================"
	echo ""
	echo "Triggered at: $(date '+%Y-%m-%d %H:%M:%S %Z')"
	echo "Hostname: $(hostname)"
	echo "macOS Version: $(sw_vers -productVersion)"
	echo "Hardware: $(sysctl -n hw.model)"
	echo ""
	echo "======================================================================"
	echo "PROCESS STATE"
	echo "======================================================================"
	echo ""

	echo "--- Suspect Service Processes (pgrep) ---"
	echo ""
	# Capture suspect services (non-fatal if not found)
	pgrep -fl 'skhd|yabai|sketchybar|borders|colima|docker|ollama|karabiner|emacs|mysql|redis|NIHardware' || echo "No matching processes found"
	echo ""

	echo "--- Full Process List (ps aux) ---"
	echo ""
	ps aux
	echo ""

	echo "======================================================================"
	echo "LAUNCHCTL SERVICE STATE"
	echo "======================================================================"
	echo ""

	echo "--- User LaunchAgents (disabled/enabled state) ---"
	echo ""
	launchctl print-disabled user/$(id -u)
	echo ""

	echo "--- System LaunchDaemons (disabled/enabled state) ---"
	echo ""
	sudo launchctl print-disabled system/ 2>/dev/null || echo "Unable to query system daemons (requires sudo)"
	echo ""

	echo "--- Loaded LaunchAgents (filtered for suspect services) ---"
	echo ""
	launchctl list | head -1 # header
	launchctl list | grep -E 'skhd|yabai|sketchybar|borders|colima|docker|ollama|karabiner|emacs|mysql|redis|orch|claude|NI' || echo "No matching services loaded"
	echo ""

	echo "--- LaunchAgent Plist Files Present ---"
	echo ""
	ls -la "$HOME/Library/LaunchAgents/" | grep -E 'skhd|yabai|sketchybar|borders|colima|docker|ollama|emacs|mysql|redis|orch|claude' || echo "No matching plist files"
	echo ""

	echo "======================================================================"
	echo "MEMORY STATE"
	echo "======================================================================"
	echo ""

	echo "--- Memory Pressure ---"
	echo ""
	memory_pressure
	echo ""

	echo "--- VM Statistics ---"
	echo ""
	vm_stat
	echo ""

	echo "--- Memory Info (top snapshot) ---"
	echo ""
	top -l 1 -n 0 -s 0 | head -20
	echo ""

	echo "======================================================================"
	echo "IOKIT HID STATE"
	echo "======================================================================"
	echo ""

	echo "--- HID Device Registry (IOKit) ---"
	echo ""
	# Capture HID subsystem state (trackpad, keyboard, Karabiner DriverKit)
	ioreg -c IOHIDDevice -r -l
	echo ""

	echo "--- System Extensions (DriverKit) ---"
	echo ""
	systemextensionsctl list
	echo ""

	echo "======================================================================"
	echo "WINDOWSERVER STATE"
	echo "======================================================================"
	echo ""

	echo "--- WindowServer Process ---"
	echo ""
	ps aux | grep -i windowserver | grep -v grep
	echo ""

	echo "--- WindowServer Connections (lsof) ---"
	echo ""
	# Show what's connected to WindowServer
	sudo lsof -c WindowServer 2>/dev/null | head -50 || echo "Unable to query WindowServer connections (requires sudo)"
	echo ""

	echo "--- CGEventTap Active Listeners (estimated via lsof) ---"
	echo ""
	# Services with event taps will have connections to WindowServer
	sudo lsof -c yabai -c skhd -c karabiner 2>/dev/null | grep -i "windowserver\|event" || echo "No event tap connections found"
	echo ""

	echo "======================================================================"
	echo "ACCESSIBILITY API STATE"
	echo "======================================================================"
	echo ""

	echo "--- Accessibility Clients (via sqlite) ---"
	echo ""
	# Query TCC database for Accessibility permissions
	# This shows which apps have Accessibility API access (yabai, skhd, etc.)
	sqlite3 "$HOME/Library/Application Support/com.apple.TCC/TCC.db" \
		"SELECT service, client, auth_value, last_modified FROM access WHERE service='kTCCServiceAccessibility'" \
		2>/dev/null || echo "Unable to query TCC database"
	echo ""

	echo "======================================================================"
	echo "DOCKER/COLIMA STATE (if running)"
	echo "======================================================================"
	echo ""

	if command -v docker &>/dev/null; then
		echo "--- Docker Version ---"
		docker version 2>/dev/null || echo "Docker not responding"
		echo ""

		echo "--- Docker Containers ---"
		docker ps -a 2>/dev/null || echo "Docker not responding"
		echo ""
	else
		echo "Docker not installed or not in PATH"
		echo ""
	fi

	if command -v colima &>/dev/null; then
		echo "--- Colima Status ---"
		colima status 2>/dev/null || echo "Colima not responding"
		echo ""
	else
		echo "Colima not installed or not in PATH"
		echo ""
	fi

	echo "======================================================================"
	echo "SYSTEM UPTIME & LOAD"
	echo "======================================================================"
	echo ""

	echo "--- Uptime ---"
	uptime
	echo ""

	echo "--- Last Reboot ---"
	who -b
	echo ""

	echo "======================================================================"
	echo "CAPTURE COMPLETE"
	echo "======================================================================"
	echo ""
	echo "Captured at: $(date '+%Y-%m-%d %H:%M:%S %Z')"
	echo "Output file: $CAPTURE_FILE"
	echo ""

} >"$CAPTURE_FILE" 2>&1

# Success notification
echo "✓ Click freeze capture complete"
echo "  Output: $CAPTURE_FILE"
echo "  Size: $(du -h "$CAPTURE_FILE" | cut -f1)"

# Optionally display notification (requires macOS)
if command -v osascript &>/dev/null; then
	osascript -e "display notification \"Capture saved to: $CAPTURE_FILE\" with title \"Click Freeze Capture\" sound name \"Purr\""
fi
