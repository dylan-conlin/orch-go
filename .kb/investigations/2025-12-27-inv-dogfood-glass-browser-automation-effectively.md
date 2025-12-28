<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Glass dogfooding is blocked by Chrome profile isolation. Glass works correctly (MCP config exists, CLI tools work), but debug Chrome uses separate profile from Dylan's browser, breaking the "see what agent sees" UX.

**Evidence:** Chrome debug instance runs with `--user-data-dir=/tmp/chrome-debug`; Glass sees tabs (localhost:5188 dashboard, etc.) but on separate Chrome profile; MCP config exists at `~/.config/opencode/opencode.jsonc` with Glass enabled.

**Knowledge:** The friction is NOT a Glass bug but a Chrome launch configuration issue. Fix requires enabling remote debugging on Dylan's primary Chrome profile (not launching a separate instance).

**Next:** Create documentation for launching Chrome with remote debugging on primary profile, then test Glass dogfooding workflow with real dashboard interaction.

---

# Investigation: Dogfood Glass Browser Automation Effectively

**Question:** How to dogfood Glass browser automation effectively? What's the minimal path to regular Glass usage, and what needs fixing first?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** Implement fix in beads issue orch-go-cg9s
**Status:** Complete

---

## Findings

### Finding 1: Glass MCP is Already Configured in OpenCode

**Evidence:** 
```json
// From ~/.config/opencode/opencode.jsonc
"mcp": {
  "glass": {
    "type": "local",
    "command": ["/Users/dylanconlin/bin/glass", "mcp"],
    "enabled": true
  },
  "playwright": {
    "type": "local",
    "command": ["npx", "@playwright/mcp@latest", "--viewport-size=1440x900"],
    "enabled": true
  }
}
```

**Source:** `/Users/dylanconlin/Documents/dotfiles/.config/opencode/opencode.jsonc:35-47`

**Significance:** Glass is ready for agent use. The `--mcp glass` spawn flag is not required - Glass MCP is always available to agents. The spawn flag pattern (`--mcp playwright`) is for Playwright (which is heavier and optional).

---

### Finding 2: Chrome Debug Instance Uses Separate Profile (Root Cause)

**Evidence:** 
From `launch-chrome.sh`:
```bash
"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" \
  --remote-debugging-port=9222 \
  --user-data-dir=/tmp/chrome-debug \  # <-- Separate profile!
```

Running Chrome processes confirm:
```
/Users/dylanconlin/Library/Caches/com.spotify.client  # Spotify (Chromium-based)
/tmp/chrome-debug  # Debug Chrome (separate profile)
```

**Source:** `/Users/dylanconlin/Documents/personal/glass/launch-chrome.sh:9-12`

**Significance:** This is why Glass "opens new Chrome window" - it's actually connecting to a separate Chrome instance with an isolated profile. Dylan's extensions, bookmarks, and session state don't exist in this profile.

---

### Finding 3: Glass CLI Tools Work Correctly

**Evidence:**
```bash
$ glass tabs
[0] localhost - http://localhost:3000/quotes/comparison
[1] Comparison View | Price Watch - http://localhost:5173/comparison
...
[5] Swarm Dashboard - http://localhost:5188/

$ glass assert url-contains:localhost  
# Works but detected wrong "focused" tab (chrome-error page)
```

Chrome debug port is accessible:
```json
$ curl http://localhost:9222/json/version
{
  "Browser": "Chrome/143.0.7499.170",
  "webSocketDebuggerUrl": "ws://localhost:9222/devtools/browser/..."
}
```

**Source:** Manual tests during investigation

**Significance:** Glass infrastructure is working. The bug (orch-go-cg9s) is about Chrome profile isolation, not Glass functionality. Glass connects to whatever Chrome has remote debugging enabled.

---

### Finding 4: Focused Tab Detection Has Edge Cases

**Evidence:**
```bash
$ glass url
chrome-error://chromewebdata/

$ glass tabs
[0] localhost - http://localhost:3000/quotes/comparison
...
[5] Swarm Dashboard - http://localhost:5188/
```

Glass reports chrome-error page as focused even though valid tabs exist.

**Source:** `/Users/dylanconlin/Documents/personal/glass/pkg/chrome/daemon.go:128-141`

**Significance:** The "focused tab" detection (using `document.hasFocus()`) may not work reliably for error pages. This is a minor issue - can be worked around with explicit tab selection or fixed later.

---

### Finding 5: Beads Issue orch-go-cg9s Captures the Friction

**Evidence:**
```
orch-go-cg9s: Glass opens new Chrome window instead of using user's existing browser

**Observed:** Glass actions happen in a separate Chrome instance
**Expected:** Glass should attach to or be visible in user's existing browser session

**Context:** Discovered during price-watch orchestration session.
```

**Source:** `bd show orch-go-cg9s`

**Significance:** The friction is already tracked. The root cause (Chrome profile isolation via `--user-data-dir`) was unknown until this investigation.

---

## Synthesis

**Key Insights:**

1. **Glass infrastructure is complete** - MCP config exists, CLI tools work, Chrome connects. The perceived "integration" gap is actually a Chrome configuration issue.

2. **The fix is in Chrome, not Glass** - To use Dylan's actual browser, Chrome must be launched with `--remote-debugging-port=9222` WITHOUT the `--user-data-dir` flag. This attaches debugging to the primary profile.

3. **Two valid dogfooding modes** - 
   - **Primary profile mode (desired):** Dylan's actual Chrome with extensions, logged-in sessions, etc.
   - **Isolated mode (current):** Fresh profile for reproducible testing (still useful for CI/automation)

**Answer to Investigation Question:**

**Minimal dogfooding path:**

1. **Quick fix (immediate):** Quit existing Chrome, relaunch with:
   ```bash
   /Applications/Google\ Chrome.app/Contents/MacOS/Google\ Chrome --remote-debugging-port=9222
   ```
   (No `--user-data-dir` flag = uses primary profile)

2. **Better fix (persistent):** Create AppleScript or Automator app to launch Chrome with debugging enabled, or configure Chrome to always enable debugging.

3. **What needs fixing first:** Update `launch-chrome.sh` to support both modes:
   - `./launch-chrome.sh` - Primary profile (for dogfooding)
   - `./launch-chrome.sh --isolated` - Fresh profile (for testing)

---

## Structured Uncertainty

**What's tested:**

- ✅ Glass CLI connects to Chrome (verified: `glass tabs` shows 9 tabs)
- ✅ Glass MCP config exists (verified: `~/.config/opencode/opencode.jsonc` has glass entry)
- ✅ Chrome debug port responds (verified: `curl localhost:9222/json/version`)
- ✅ Glass assert command works (verified: ran assertions, got expected failures)

**What's untested:**

- ⚠️ Chrome primary profile with remote debugging (haven't launched Dylan's main Chrome with `--remote-debugging-port`)
- ⚠️ Agent using Glass MCP in real spawn (haven't spawned agent with `--mcp glass`)
- ⚠️ Focused tab detection on primary Chrome profile

**What would change this:**

- If Chrome refuses remote debugging on primary profile (security restrictions)
- If some Chrome extensions interfere with CDP
- If macOS code signing blocks modified Chrome launch

---

## Implementation Recommendations

**Purpose:** Enable Glass dogfooding this week with minimal changes.

### Recommended Approach ⭐

**Fix Chrome launch script to support primary profile**

**Why this approach:**
- Root cause is Chrome profile isolation, not Glass
- Minimal change (script update only)
- Enables immediate dogfooding
- Preserves isolated mode for testing

**Trade-offs accepted:**
- Dylan must manually launch Chrome with debugging (no auto-attach)
- If Chrome crashes, must relaunch with debugging enabled

**Implementation sequence:**
1. Update `launch-chrome.sh` to detect and warn if Chrome already running
2. Add `--primary` or `--isolated` flags to choose profile mode
3. Document the dogfooding workflow in Glass CLAUDE.md

### Alternative Approaches Considered

**Option B: Chrome Extension for debugging**
- **Pros:** No manual relaunch, always available
- **Cons:** Extension development overhead, may not work for CDP
- **When to use instead:** If manual relaunch is too disruptive

**Option C: Separate Chrome channel (Canary)**
- **Pros:** Always has debugging, won't affect main browsing
- **Cons:** Different profile, different extensions
- **When to use instead:** If can't enable debugging on main Chrome

**Rationale for recommendation:** Option A (fix launch script) is minimal, directly addresses root cause, and gives Dylan both modes.

---

### Implementation Details

**What to implement first:**
- Update `launch-chrome.sh` to support primary profile mode
- Test Glass with Dylan's actual Chrome
- Document in Glass CLAUDE.md

**Things to watch out for:**
- ⚠️ Chrome must be fully quit before relaunching with debugging
- ⚠️ `--remote-debugging-port` may conflict if port already in use
- ⚠️ Some extensions may behave differently with CDP attached

**Areas needing further investigation:**
- Can Chrome be launched with debugging AND attach to existing running instance?
- macOS-specific: Can we use `open -a` with custom flags?
- Is there a way to enable debugging via Chrome flags/config?

**Success criteria:**
- ✅ Dylan can launch Chrome, Glass sees his actual tabs
- ✅ Agent can interact with dashboard via Glass MCP
- ✅ Dylan and agent see the same browser state

---

## Dogfooding Workflow (Proposed)

**Daily workflow:**

1. **Morning launch:**
   ```bash
   # Quit Chrome if running
   osascript -e 'quit app "Google Chrome"'
   
   # Launch with debugging enabled
   open -a "Google Chrome" --args --remote-debugging-port=9222
   ```

2. **Spawn with Glass:**
   - Glass MCP already enabled in OpenCode config
   - No special `--mcp` flag needed
   - Agent can use `glass_*` tools immediately

3. **Validation in orch complete:**
   ```bash
   glass assert --url http://localhost:5188 title-contains:Dashboard
   ```

**Friction points to monitor:**
- How often does Chrome crash/restart? (requires relaunch with debugging)
- Does debugging affect Chrome performance?
- Are there tab focus issues?

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/glass/launch-chrome.sh` - Chrome launch script
- `/Users/dylanconlin/Documents/personal/glass/pkg/chrome/daemon.go` - Glass Chrome connection
- `/Users/dylanconlin/Documents/personal/glass/CLAUDE.md` - Glass documentation
- `/Users/dylanconlin/Documents/dotfiles/.config/opencode/opencode.jsonc` - OpenCode MCP config
- `.kb/investigations/2025-12-27-inv-glass-integration-status-orch-ecosystem.md` - Prior Glass investigation

**Commands Run:**
```bash
# Check Chrome debug port
curl http://localhost:9222/json/version

# List Glass tabs
glass tabs

# Test Glass assertion
glass assert url-contains:localhost

# Check beads issue
bd show orch-go-cg9s
```

**Related Artifacts:**
- **Beads Issue:** orch-go-cg9s - Glass opens new Chrome window (root cause identified)
- **Investigation:** 2025-12-27-inv-glass-integration-status-orch-ecosystem.md - Glass MCP status
- **Investigation:** 2025-12-27-inv-add-cli-commands-glass-orchestrator.md - Glass assert command

---

## Self-Review

- [x] Real test performed (ran Glass commands, verified Chrome connection)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered (dogfooding path and blockers identified)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled
- [x] NOT DONE claims verified (checked actual Chrome launch script)

**Self-Review Status:** PASSED

---

## Investigation History

**2025-12-27 13:35:** Investigation started
- Initial question: How to dogfood Glass effectively?
- Context: Dylan wants regular Glass usage but facing friction

**2025-12-27 13:40:** Root cause identified
- Found `--user-data-dir=/tmp/chrome-debug` in launch script
- This explains "opens new window" behavior in orch-go-cg9s

**2025-12-27 13:50:** Investigation completed
- Status: Complete
- Key outcome: Fix Chrome launch, not Glass. MCP already configured.
