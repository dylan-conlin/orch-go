## Summary (D.E.K.N.)

**Delta:** Snap CLI is well-suited for native macOS window/screen capture but cannot replace Playwright MCP for web UI verification because it lacks browser control (navigation, interaction, waiting for load).

**Evidence:** Tested snap commands successfully for screen capture; reviewed feature-impl skill requiring browser_navigate, browser_click before screenshot; snap only captures what's visible without controlling browser state.

**Knowledge:** Visual verification for web/ changes requires 1) navigate to URL, 2) wait for load, 3) optionally interact, 4) capture. Snap provides only step 4. Playwright MCP provides all 4.

**Next:** Do NOT integrate snap into feature-impl validation for web/ changes. Consider snap for native macOS UI verification if that use case emerges.

**Confidence:** High (85%) - Clear architectural distinction between capture (snap) vs browser automation (Playwright).

---

# Investigation: Evaluate snap CLI Integration for Visual Verification

**Question:** Can snap CLI integrate with feature-impl skill for visual verification, and if so, how does it compare to the current Playwright MCP approach?

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Snap CLI is a working native screenshot utility

**Evidence:** 
```bash
$ snap --help
snap is a screenshot CLI utility designed for AI coding agents.
Available Commands:
  capture     Take a screenshot
  list        List visible windows
  window      Capture a specific window

$ snap --output /tmp/test.png
/tmp/snap-test.png  # 264KB file created

$ snap list --json
[{"app": "Firefox", "id": 58214, "title": ""}, ...]
```

Snap wraps macOS `screencapture` and provides AI-first patterns (--json, --ai-help).

**Source:** Direct CLI testing at `/Users/dylanconlin/Documents/personal/snap/snap`

**Significance:** Snap is functional for capturing whatever is currently visible on screen. It's designed for AI agents and outputs paths that agents can use with the Read tool.

---

### Finding 2: Feature-impl validation requires browser control, not just capture

**Evidence:** Current Playwright MCP workflow for web/ validation (from phase-validation.md):

```bash
# 1. Navigate to the page
mcp__playwright__browser_navigate url="http://localhost:3000/feature"

# 2. Wait for load / get page state  
mcp__playwright__browser_snapshot

# 3. Optionally interact (click, fill forms)
mcp__playwright__browser_click element="Submit button" ref="e12"

# 4. Capture screenshot
mcp__playwright__browser_take_screenshot filename="smoke-test-feature.png"
```

This is a 4-step workflow where screenshot is the LAST step. Before capturing, the agent must:
1. Start the browser and navigate to the correct URL
2. Wait for the page to fully load
3. Potentially interact with UI elements

**Source:** `/Users/dylanconlin/.claude/skills/worker/feature-impl/reference/phase-validation.md` (lines 147-167)

**Significance:** Visual verification for web changes is NOT just "capture what's visible" - it's "navigate, wait, interact, then capture." Snap only provides the capture step.

---

### Finding 3: Snap captures what's visible; Playwright controls what's visible

**Evidence:** Comparison of capabilities:

| Capability | Snap CLI | Playwright MCP |
|------------|----------|----------------|
| Capture screen | ✅ | ✅ |
| Capture specific window | ✅ (by ID/name) | ✅ (controlled browser) |
| Navigate to URL | ❌ | ✅ browser_navigate |
| Wait for page load | ❌ | ✅ browser_snapshot |
| Click elements | ❌ | ✅ browser_click |
| Fill forms | ❌ | ✅ browser_type |
| Control browser state | ❌ | ✅ Full control |
| Works with any app | ✅ | ❌ Browser only |
| Requires browser running | ❌ | ✅ |
| MCP server overhead | None | Running server |

**Source:** snap --help, Playwright MCP tool list, feature-impl skill documentation

**Significance:** They serve different purposes. Snap is for "capture the current state" while Playwright is for "control the browser then capture the state you want."

---

### Finding 4: Snap window capture requires exact window ID discovery

**Evidence:** Testing showed that window capture by app name requires the window to be in the list with a matching ID:

```bash
$ snap list --json | jq '.[] | select(.app == "Firefox")'
{"app": "Firefox", "id": 58214, "title": ""}
{"app": "Firefox", "id": 42271, "title": ""}

$ snap window --id 58214 --output /tmp/firefox.png
Error: failed to capture window: screencapture failed: exit status 1
```

Window capture by ID fails even with valid ID (may be permissions or headless context issue). Full screen capture works. This is a separate bug, but illustrates that snap requires more setup for targeted capture.

**Source:** Direct testing

**Significance:** Even if snap were integrated, targeted window capture has reliability concerns that would need investigation.

---

### Finding 5: Current feature-impl skill already mentions snap for systematic-debugging

**Evidence:** In systematic-debugging skill:
```
1. Need visual verification? → `snap`
```

This suggests snap is already positioned as a tool for general visual verification (capturing current screen state for debugging), not as a replacement for Playwright's browser control workflow.

**Source:** `/Users/dylanconlin/.claude/skills/worker/systematic-debugging/SKILL.md` (line 198)

**Significance:** Snap has a role in the ecosystem - it's for quick visual capture of current state during debugging. Playwright MCP is for controlled web UI verification.

---

## Synthesis

**Key Insights:**

1. **Different problems, different tools** - Snap solves "capture what's on screen now" while Playwright solves "control a browser, put it in the right state, then verify visually." These are complementary, not substitutes.

2. **Web verification needs browser control** - An agent implementing a feature can't just screenshot Firefox; it needs to navigate to the right URL, wait for the page to load, maybe interact with UI, then capture. Snap skips all the control steps.

3. **Snap's value is native/non-browser capture** - For macOS UI verification (desktop apps, system dialogs, Finder states), snap is the right tool. Playwright can't help there.

**Answer to Investigation Question:**

Snap CLI **should NOT** replace or integrate with Playwright MCP for web/ file validation in feature-impl because:

1. Web validation requires browser control (navigate, wait, interact) before capture
2. Playwright provides all 4 steps as a unified workflow
3. Snap only provides the capture step

Snap **should** remain as a complementary tool for:
- Quick screen capture during debugging (as noted in systematic-debugging skill)
- Native macOS app verification (if that use case emerges)
- General "what does the screen look like right now" queries

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

The architectural distinction is clear and well-evidenced. Web verification requires browser control. Snap is a capture-only tool. The remaining uncertainty is whether there are edge cases where snap could augment (not replace) Playwright.

**What's certain:**

- ✅ Feature-impl web validation requires navigate → wait → interact → capture
- ✅ Snap only provides capture functionality
- ✅ Playwright MCP provides the full workflow
- ✅ Snap is already positioned for debugging-time capture

**What's uncertain:**

- ⚠️ Whether snap's window capture bug affects real use cases
- ⚠️ Whether there's value in snap as backup if Playwright MCP unavailable
- ⚠️ Whether native macOS UI verification is a real use case

**What would increase confidence to Very High (95%):**

- Confirm snap window capture works reliably
- Validate there's no scenario where snap could replace browser_take_screenshot
- Get feedback on whether native macOS UI verification is needed

---

## Implementation Recommendations

### Recommended Approach ⭐

**No integration changes** - Keep feature-impl using Playwright MCP for web/ validation; keep snap available for systematic-debugging as-is.

**Why this approach:**
- Clear separation of concerns (browser control vs native capture)
- Avoids adding complexity to validation workflow
- Snap is already documented for debugging use cases

**Trade-offs accepted:**
- If Playwright MCP is unavailable, web validation can't proceed (acceptable - validation is mandatory)
- Snap's full capabilities aren't leveraged in feature-impl (acceptable - not needed there)

**Implementation sequence:**
1. No changes needed to feature-impl skill
2. Consider documenting snap more prominently in debugging workflows
3. If native macOS UI verification becomes a requirement, add snap integration then

### Alternative Approaches Considered

**Option B: Add snap as fallback for Playwright**
- **Pros:** Resilience if MCP server fails
- **Cons:** Can't navigate/wait/interact; would produce useless screenshots
- **When to use instead:** Never for web/ validation

**Option C: Replace Playwright with snap**
- **Pros:** Simpler (no MCP server), faster capture
- **Cons:** Loses all browser control; fundamentally different tool
- **When to use instead:** Only if capturing visible state is sufficient (not for validation)

**Rationale for recommendation:** The problem space (web UI verification) requires browser control. Snap doesn't provide it. No integration makes sense.

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/snap/CLAUDE.md` - Project purpose
- `/Users/dylanconlin/Documents/personal/snap/.kb/investigations/2025-12-12-inv-design-snap-cli-screenshot-utility.md` - Original design
- `/Users/dylanconlin/.claude/skills/worker/feature-impl/reference/phase-validation.md` - Current validation workflow
- `/Users/dylanconlin/.claude/skills/worker/feature-impl/reference/validation-examples.md` - Playwright usage examples
- `/Users/dylanconlin/.claude/skills/worker/systematic-debugging/SKILL.md` - Existing snap mention

**Commands Run:**
```bash
# snap capability testing
/Users/dylanconlin/Documents/personal/snap/snap --help
/Users/dylanconlin/Documents/personal/snap/snap --output /tmp/test.png
/Users/dylanconlin/Documents/personal/snap/snap list --json

# Feature-impl skill analysis
grep -r "playwright\|browser_take_screenshot\|visual.*verif" ~/.claude/skills/
```

**Related Artifacts:**
- **Investigation:** `/Users/dylanconlin/Documents/personal/snap/.kb/investigations/2025-12-26-inv-evaluate-snap-ecosystem-integration.md` - Earlier ecosystem audit

---

## Investigation History

**2025-12-26 16:10:** Investigation started
- Initial question: Evaluate snap CLI for feature-impl visual verification
- Context: Spawned to assess integration possibilities

**2025-12-26 16:20:** Core testing complete
- Snap CLI works for screen capture
- Window capture has issues (separate bug)
- Playwright MCP workflow reviewed

**2025-12-26 16:30:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Snap is NOT suitable for web validation; keep current Playwright approach
