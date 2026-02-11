#!/bin/bash
# Install the orch reap launchd agent for automatic zombie process cleanup.
# Runs `orch reap` every 5 minutes to prevent bun process accumulation.

set -e

PLIST_SRC="$(dirname "$0")/com.orch.reap.plist"
PLIST_DST="$HOME/Library/LaunchAgents/com.orch.reap.plist"
LOG_DIR="$HOME/.orch/logs"

# Ensure log directory exists
mkdir -p "$LOG_DIR"

# Unload existing if present
if launchctl list | grep -q com.orch.reap; then
	echo "Unloading existing com.orch.reap..."
	launchctl unload "$PLIST_DST" 2>/dev/null || true
fi

# Copy plist
cp "$PLIST_SRC" "$PLIST_DST"
echo "Installed plist: $PLIST_DST"

# Load
launchctl load "$PLIST_DST"
echo "Loaded com.orch.reap"

# Verify
if launchctl list | grep -q com.orch.reap; then
	echo "✅ orch reap agent is running (every 5 minutes)"
	echo "   Logs: $LOG_DIR/reap.log"
else
	echo "❌ Failed to start orch reap agent"
	exit 1
fi
