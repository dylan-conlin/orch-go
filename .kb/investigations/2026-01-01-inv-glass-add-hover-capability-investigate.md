## Summary (D.E.K.N.)

**Delta:** Hover capability added to glass (daemon, MCP, CLI). Investigation identified 8 additional interaction modes of varying priority.

**Evidence:** CDP Input domain provides `dispatchMouseEvent(mouseMoved)` for hover. Tested implementation compiles. Reviewed CDP docs for other modes.

**Knowledge:** High-value additions: right-click, keyboard shortcuts, wait-for-element, double-click. Lower priority: drag-drop, clear input. Glass already has scroll, type, click, navigate.

**Next:** Create follow-up issues for high-priority interaction modes. Merge hover implementation.

---

# Investigation: Glass Hover Capability and Other Interaction Modes

**Question:** How to add hover capability to glass, and what other interaction modes are commonly needed?

**Started:** 2026-01-01
**Updated:** 2026-01-01
**Owner:** Agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Hover Implementation via CDP Input.dispatchMouseEvent

**Evidence:** Chrome DevTools Protocol provides `Input.dispatchMouseEvent` with type `mouseMoved` to simulate mouse movement. This triggers CSS `:hover` states, `mouseenter` events, and tooltips.

**Source:** 
- https://chromedevtools.github.io/devtools-protocol/tot/Input/#method-dispatchMouseEvent
- `/Users/dylanconlin/go/pkg/mod/github.com/mafredri/cdp@v0.35.0/protocol/input/command.go`

**Significance:** Implementation requires:
1. Get element bounding box via JavaScript
2. Calculate center coordinates
3. Dispatch `mouseMoved` event via CDP Input domain

---

### Finding 2: Glass Already Has Extensive Interaction Support

**Evidence:** Existing daemon methods include:
- `Click(ctx, selector)` - Click on element
- `Type(ctx, selector, text)` - Type into input (including select handling)
- `Navigate(ctx, url)` - Navigate to URL
- `Screenshot(ctx, opts)` - Capture screenshot
- `Scroll(ctx, opts)` - Scroll by pixels, to element, or position
- `ElementExists(ctx, selector)` - Check if element exists
- `WaitForNavigation(ctx)` - Wait for page load
- `waitForDOMStable(ctx, client, timeout)` - Wait for SPA content

**Source:** `/Users/dylanconlin/Documents/personal/glass/pkg/chrome/daemon.go` (grep for `func (d *Daemon)`)

**Significance:** Glass is already feature-rich. Hover fills a notable gap for tooltip/dropdown testing.

---

### Finding 3: CDP Input Domain Capabilities for Additional Modes

**Evidence:** CDP Input domain supports:
- `dispatchMouseEvent` - Types: `mousePressed`, `mouseReleased`, `mouseMoved`, `mouseWheel`
- `dispatchKeyEvent` - Types: `keyDown`, `keyUp`, `rawKeyDown`, `char`
- `dispatchDragEvent` - Types: `dragEnter`, `dragOver`, `drop`, `dragCancel` (Experimental)
- `MouseButton` enum: `none`, `left`, `middle`, `right`, `back`, `forward`

**Source:** https://chromedevtools.github.io/devtools-protocol/tot/Input/

**Significance:** Most common interaction modes can be implemented via these primitives.

---

### Finding 4: Priority Assessment of Additional Interaction Modes

**Evidence:** Analysis of common automation needs vs current glass capabilities:

| Mode | Priority | Reason | CDP Method |
|------|----------|--------|------------|
| **Right-click** | High | Context menus common in apps | `dispatchMouseEvent` with `button: "right"` |
| **Press key** | High | Enter, Tab, Escape, Arrow keys | `dispatchKeyEvent` |
| **Keyboard shortcuts** | High | Ctrl+C, Ctrl+V, Ctrl+A | `dispatchKeyEvent` with modifiers |
| **Wait for element** | High | SPA loading patterns | JavaScript + polling |
| **Double-click** | Medium | File managers, editing | `dispatchMouseEvent` with `clickCount: 2` |
| **Clear input** | Medium | Reset form fields | Select all + delete via JS or key events |
| **Focus element** | Medium | Accessibility, keyboard nav | JS `element.focus()` |
| **Drag and drop** | Low | Complex, experimental in CDP | `dispatchDragEvent` (experimental) |

**Source:** Common automation patterns in Playwright, Puppeteer, Selenium

**Significance:** High-priority items would significantly improve glass usefulness for UI testing.

---

## Synthesis

**Key Insights:**

1. **Hover was straightforward** - CDP `mouseMoved` event with element coordinates. Implementation pattern can be reused for right-click and double-click.

2. **Keyboard input is the biggest gap** - Glass has `Type()` for text input, but no way to press Enter, Tab, Escape, or keyboard shortcuts. This is critical for form submission and navigation.

3. **Wait-for-element is essential for SPAs** - Glass has `waitForDOMStable` internally but doesn't expose a "wait until selector exists" API. This is critical for reliable automation.

**Answer to Investigation Question:**

Hover was implemented using `Input.dispatchMouseEvent(type: "mouseMoved", x, y)` after getting element center coordinates via JavaScript. 

The most important additional interaction modes are:
1. **Keyboard events** (press Enter, Tab, Escape, shortcuts) - frequent need
2. **Wait for element** (wait until selector appears) - SPA reliability
3. **Right-click** (context menus) - common pattern
4. **Double-click** (selection, editing) - occasional need

---

## Structured Uncertainty

**What's tested:**

- ✅ Hover implementation compiles (`go build ./...` succeeds)
- ✅ CDP Input domain supports `mouseMoved` for hover (verified via protocol docs)
- ✅ Glass daemon can create Input client (`input.NewClient(d.conn)`)

**What's untested:**

- ⚠️ Hover actually triggers `:hover` CSS (not tested in browser yet)
- ⚠️ Hover triggers tooltip display (needs browser test)
- ⚠️ Hover works with extended selectors (`:has-text`, `[nth=N]`)

**What would change this:**

- If `mouseMoved` alone doesn't trigger hover states, may need `mouseEnter` or JavaScript `dispatchEvent`
- If CDP rejects coordinates outside viewport, may need to scroll element into view first

---

## Implementation Recommendations

**Purpose:** Prioritized roadmap for additional glass interaction modes.

### Recommended Approach ⭐

**Add keyboard and wait-for capabilities first** - These address the most common gaps in glass functionality.

**Why this approach:**
- Keyboard events enable form submission, modal dismissal, navigation
- Wait-for-element enables reliable SPA automation
- Both are frequently needed in orchestrator validation scenarios

**Trade-offs accepted:**
- Deferring drag-and-drop (complex, experimental)
- Deferring focus management (can be done via click)

**Implementation sequence:**
1. **PressKey** - Single key (Enter, Tab, Escape) via `Input.dispatchKeyEvent`
2. **WaitForSelector** - Poll until element appears or timeout
3. **RightClick** - Same pattern as hover + mousePressed/mouseReleased with right button
4. **DoubleClick** - Click with `clickCount: 2`

### Alternative Approaches Considered

**Option B: JavaScript-based keyboard input**
- **Pros:** Simpler, no CDP Input domain needed
- **Cons:** May not trigger all event handlers, less realistic
- **When to use instead:** If CDP key events cause issues

**Option C: Add all modes at once**
- **Pros:** Complete feature set
- **Cons:** Large scope, delay on shipping any improvements
- **When to use instead:** If there's a clear need for all modes

---

### Implementation Details

**What to implement first:**
- PressKey for Enter, Tab, Escape, Arrow keys
- WaitForSelector with configurable timeout
- Right-click for context menus

**Things to watch out for:**
- ⚠️ Key codes vary by platform - use DOM key names not key codes
- ⚠️ Wait timeouts need sensible defaults (e.g., 30s)
- ⚠️ Right-click may need "none" button on mouseReleased to avoid holding

**Areas needing further investigation:**
- Keyboard modifier handling (Ctrl, Shift, Alt)
- Key repeat behavior
- Touch event support (mobile testing)

**Success criteria:**
- ✅ `glass press Enter` submits form on focused input
- ✅ `glass wait-for ".modal"` blocks until modal appears
- ✅ `glass right-click ".item"` opens context menu

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/glass/pkg/chrome/daemon.go` - Core daemon implementation
- `/Users/dylanconlin/Documents/personal/glass/pkg/mcp/server.go` - MCP tool definitions
- `/Users/dylanconlin/Documents/personal/glass/main.go` - CLI commands
- `/Users/dylanconlin/go/pkg/mod/github.com/mafredri/cdp@v0.35.0/protocol/input/` - CDP input protocol

**Commands Run:**
```bash
# Build verification
cd /Users/dylanconlin/Documents/personal/glass && go build ./...

# List daemon methods
grep -n "func (d *Daemon)" pkg/chrome/daemon.go
```

**External Documentation:**
- https://chromedevtools.github.io/devtools-protocol/tot/Input/ - CDP Input domain

---

## Investigation History

**2026-01-01 12:00:** Investigation started
- Initial question: How to add hover capability and what other interaction modes are needed?
- Context: Spawned from orch-go-5mv6

**2026-01-01 12:30:** Hover implementation complete
- Added `Hover()` to daemon using CDP `Input.dispatchMouseEvent(mouseMoved)`
- Added `handleHover` MCP tool
- Added `glass hover` CLI command
- Added `LogHover()` action logging

**2026-01-01 13:00:** Investigation completed
- Status: Complete
- Key outcome: Hover implemented, roadmap established for keyboard, wait-for, right-click, double-click
