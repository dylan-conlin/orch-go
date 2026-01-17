# Verification Steps for Session Handoff Bug Fix

## Bug Description
Worker sessions were receiving orchestrator session handoff content when they should only receive SPAWN_CONTEXT.md.

## Fix Applied
Modified `~/.config/opencode/plugin/session-resume.js` to check for SPAWN_CONTEXT.md presence instead of process.env.ORCH_WORKER.

## Verification Steps

### Step 1: Restart OpenCode Server
```bash
# Kill existing server
pkill -f "opencode serve"

# Start new server (loads updated plugin)
opencode serve --port 4096 &

# Wait for startup
sleep 3
```

### Step 2: Test Worker Spawn (Should NOT Receive Handoff)
```bash
# Enable plugin debug logging
export ORCH_PLUGIN_DEBUG=1

# Spawn a test worker
cd /Users/dylanconlin/Documents/personal/orch-go
orch spawn feature-impl "verify handoff fix" --no-track

# Check the OpenCode session startup
# EXPECTED: No "📋 Session Resumed" message
# EXPECTED: Only SPAWN_CONTEXT.md content is shown
# EXPECTED: Plugin log shows: "Skipping injection for worker session (SPAWN_CONTEXT.md found)"
```

### Step 3: Test Orchestrator Session (Should Receive Handoff)
```bash
# Start interactive orchestrator session in project with session history
cd /Users/dylanconlin/Documents/personal/orch-go
oc

# EXPECTED: "📋 Session Resumed" message appears with handoff content
# EXPECTED: Plugin log shows: "No SPAWN_CONTEXT.md found, proceeding with handoff injection"
```

### Step 4: Report Verification
If both tests pass:
```bash
bd comment orch-go-9hasd "Reproduction verified: Worker spawns no longer receive handoff (SPAWN_CONTEXT.md check works), orchestrators still receive handoff correctly. Plugin logs confirm correct behavior."
```

## Success Criteria
- ✅ Worker sessions show NO handoff content
- ✅ Worker sessions only display SPAWN_CONTEXT.md content  
- ✅ Orchestrator sessions show handoff content
- ✅ Plugin debug logs show correct skip/proceed decisions

## If Verification Fails
1. Check OpenCode server logs for errors
2. Verify plugin file changes were saved: `cat ~/.config/opencode/plugin/session-resume.js | grep -A5 "SPAWN_CONTEXT.md"`
3. Confirm server picked up changes (server must be restarted)
4. Report failure via: `bd comment orch-go-9hasd "Verification failed: [describe what happened]"`
