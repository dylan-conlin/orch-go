<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Playwright MCP + headed browser with `--extension` mode provides the best foundation for shared browser experience, enabling both agent automation and human visibility/intervention via a Chrome extension bridge.

**Evidence:** Reviewed Playwright MCP docs (--extension flag, --cdp-endpoint for connecting to existing browser), browser-use architecture (Python-based, higher overhead), and CDP protocol capabilities for real-time state synchronization.

**Knowledge:** True "shared control" requires either (1) CDP-based state sync with turn-taking protocol, (2) VNC/screen-share for visual sync, or (3) the Playwright MCP extension approach which connects to human's actual browser; accessibility tree snapshots are superior to screenshots for AI interaction.

**Next:** Recommend starting with Playwright MCP `--extension` mode for simplest integration path - human installs browser extension, agent connects to their session.

---

# Investigation: Shared Browser Experience Orch Ecosystem

**Question:** What would it take for orchestrator and human to share a browser session where both can navigate, see state, and interact? Consider real-time sync, control handoff, state visibility, and integration with existing orch patterns.

**Started:** 2025-12-26
**Updated:** 2025-12-26
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None - ready for orchestrator review
**Status:** Complete

---

## Findings

### Finding 1: Playwright MCP Offers Two Shared Browser Approaches

**Evidence:** The Playwright MCP server (github.com/microsoft/playwright-mcp, 24.8k stars) offers multiple connection modes:

1. **`--extension` mode**: Connect to a running browser via the "Playwright MCP Bridge" browser extension. The human has a Chrome/Edge browser open with the extension installed, and the AI agent connects to their active session.

2. **`--cdp-endpoint <endpoint>`**: Connect to an existing browser instance via Chrome DevTools Protocol. If human runs Chrome with `--remote-debugging-port=9222`, agent can connect via `ws://localhost:9222/devtools/browser/{id}`.

3. **`--user-data-dir <path>`**: Share persistent browser profile between sessions, so auth state, cookies, and history persist.

4. **`--shared-browser-context`**: Reuse the same browser context between all connected HTTP clients.

**Source:** https://github.com/microsoft/playwright-mcp README, configuration flags documentation

**Significance:** The `--extension` mode is specifically designed for "connect to existing browser tabs and leverage your logged-in sessions and browser state." This is the closest to true shared browser experience currently available - human has the browser visible on their screen, agent connects and can navigate programmatically.

---

### Finding 2: Playwright Uses Accessibility Tree, Not Screenshots

**Evidence:** From Playwright MCP documentation:
- "Uses Playwright's accessibility tree, not pixel-based input"
- "LLM-friendly. No vision models needed, operates purely on structured data"
- "Deterministic tool application. Avoids ambiguity common with screenshot-based approaches"

The `browser_snapshot` tool "captures accessibility snapshot of the current page, this is better than screenshot" - returns structured data about elements, roles, and states rather than visual pixels.

**Source:** Playwright MCP Key Features, browser_snapshot tool description

**Significance:** For agent-side interaction, accessibility tree is superior to screenshots:
- Faster (structured text vs image processing)
- More deterministic (exact element references vs coordinate guessing)
- Works headless or headed
- Human can observe the actual visual changes in the browser

This means if human and agent share a browser, the agent can describe exactly what it's doing ("clicking button ref='btn-submit'") while human sees the visual result.

---

### Finding 3: browser-use Takes a Different Approach (Higher-Level Agent Framework)

**Evidence:** browser-use (74.2k stars) is a Python-based framework that wraps browser automation into a higher-level "agent" abstraction:

```python
agent = Agent(
    task="Find the number of stars of the browser-use repo",
    llm=llm,
    browser=browser,
)
history = await agent.run()
```

Key differences from Playwright MCP:
- Python-only (vs Node.js for Playwright MCP)
- Integrated LLM calls within the framework
- Cloud service option (`use_cloud=True`) for remote stealth browsers
- Sandbox deployment model (`@sandbox()` decorator)

**Source:** https://github.com/browser-use/browser-use README, examples

**Significance:** browser-use is more opinionated - it's an "AI browser agent" framework, not just a browser automation tool. For orch ecosystem, this adds friction:
- Different language (Python vs Go for orch tooling)
- Own agent loop (conflicts with orch's existing agent patterns)
- Cloud dependency for some features
- Less flexibility for custom control handoff protocols

---

### Finding 4: Chrome DevTools Protocol Enables Real-Time State Sync

**Evidence:** CDP (Chrome DevTools Protocol) provides:
- **Page domain**: Navigation, screenshots, accessibility snapshots
- **DOM domain**: Live DOM tree inspection, mutations
- **Input domain**: Synthetic input events (clicks, typing)
- **Target domain**: Tab/page management
- **Network domain**: Request/response monitoring

Multiple clients can connect simultaneously (Chrome 63+). When a client disconnects, it receives a `detached` event with reason.

The protocol is WebSocket-based: `ws://localhost:9222/devtools/page/{targetId}`

**Source:** https://chromedevtools.github.io/devtools-protocol/

**Significance:** CDP is the foundational layer that enables shared browser experiences:
- Human opens DevTools = CDP client
- Agent connects via Playwright/Puppeteer = CDP client
- Both see same browser state
- Events can be streamed to both in real-time

The challenge is coordinating who controls input when - CDP doesn't inherently solve the "turn-taking" problem.

---

### Finding 5: Control Handoff Patterns for Shared Sessions

**Evidence:** Analyzed several approaches to "who has control when":

1. **Sequential handoff**: Agent operates, pauses with `page.pause()`, human takes over, resumes. Playwright Inspector supports this with step-through debugging.

2. **Observation mode**: One party observes while other controls. Human watches agent work, or agent monitors human's actions via DOM mutation observers.

3. **Explicit lock**: Software mutex - agent acquires control, releases when done. Human has "take control" button that interrupts agent.

4. **Region-based**: Agent controls certain elements, human controls others (complex, rarely practical).

5. **Event replay**: Agent records intended actions, human reviews and approves before execution (asynchronous).

**Source:** Playwright debug documentation, general collaborative editing patterns

**Significance:** For orch ecosystem, the simplest patterns are:
- **Agent-first + human takeover**: Agent automates, human interrupts when needed
- **Human-first + agent assist**: Human drives, calls agent for specific subtasks
- Both require a protocol for signaling control transfer

---

### Finding 6: Visual Sync Options for Human Observation

**Evidence:** For human to see what agent is doing:

1. **Headed browser on shared display**: Agent runs browser in headed mode on same machine human is viewing (e.g., via tmux + Ghostty on Mac). Human sees the actual browser window.

2. **VNC/screen sharing**: Remote display protocol. High latency for real-time interaction but works cross-machine.

3. **Video recording**: Playwright's `--save-video=800x600` records session. Human reviews afterward (not real-time).

4. **Trace viewer**: Playwright Trace Viewer captures full session with DOM snapshots, network, console. Interactive replay but not real-time.

5. **Live DOM streaming**: Agent streams DOM state via WebSocket to human's dashboard. Lower bandwidth than video, more structured.

6. **Extension approach**: Human's browser is the browser - they see everything natively.

**Source:** Playwright MCP `--save-video`, `--save-trace` flags; Trace Viewer documentation

**Significance:** The `--extension` approach is unique: human doesn't need a separate viewing mechanism because they're looking at their own browser. Agent's actions appear as if a remote user is controlling their mouse/keyboard.

---

## Synthesis

**Key Insights:**

1. **Extension Mode is the Simplest Shared Experience** - Playwright MCP's `--extension` mode lets an agent connect to the human's actual browser via a bridge extension. The human sees everything natively, the agent operates programmatically. This avoids complex sync protocols entirely - there's only one browser, two controllers.

2. **Accessibility Tree > Screenshots for Agent Interaction** - When agents use accessibility snapshots instead of screenshots, they can describe their actions precisely ("clicking 'Submit' button") while humans see the visual result. This creates a natural shared mental model.

3. **Control Handoff Needs Explicit Protocol in Orch** - Current tools don't solve the turn-taking problem. For orch ecosystem integration, we'd need:
   - A way for orchestrator to signal "agent has control" / "human has control"
   - Potentially a browser extension that shows control state indicator
   - Integration with beads/orch commands for control transfer

4. **CDP is the Foundation Layer** - Whether using Playwright MCP, browser-use, or custom tooling, Chrome DevTools Protocol is the substrate. Understanding CDP capabilities helps design shared experiences.

**Answer to Investigation Question:**

To achieve shared browser experience in orch ecosystem, the most practical path is:

**Phase 1 (MVP)**: Use Playwright MCP with `--extension` mode
- Human installs "Playwright MCP Bridge" extension in Chrome/Edge
- Agent spawns connect via `orch spawn --mcp playwright` (already supported)
- Agent connects to human's browser session
- Human watches/takes over directly in their browser
- Control is "last action wins" - simple but functional

**Phase 2 (Enhanced)**: Add explicit control protocol
- New `orch browser` subcommands: `orch browser lock`, `orch browser release`
- Browser extension shows control indicator (green = agent, blue = human)
- beads integration for tracking browser work sessions

**Phase 3 (Advanced)**: State visibility dashboard
- SSE stream of accessibility tree changes to orch dashboard
- Human can observe agent's "view" of the page
- Replay and analysis of browser sessions

---

## Structured Uncertainty

**What's tested:**

- ✅ Playwright MCP supports `--extension` flag for connecting to existing browser (verified: documentation review)
- ✅ CDP supports multiple simultaneous clients (verified: protocol documentation states Chrome 63+ support)
- ✅ Accessibility tree is used instead of screenshots (verified: Playwright MCP feature description)

**What's untested:**

- ⚠️ Actual performance of extension mode for real-time shared interaction (not tested hands-on)
- ⚠️ Whether both human and agent can issue commands simultaneously without race conditions
- ⚠️ Integration complexity with existing orch spawn patterns
- ⚠️ Browser extension installation UX for Dylan

**What would change this:**

- If extension mode proves laggy or unreliable in practice, CDP direct connection might be needed
- If turn-taking becomes problematic, explicit lock protocol would need priority
- If cross-machine sharing is required, VNC layer would need to be added

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Playwright MCP Extension Mode + Headed Browser** - Start with the simplest integration: human runs Chrome with the bridge extension, agent connects via Playwright MCP `--mcp playwright --extension` equivalent.

**Why this approach:**
- Minimal new infrastructure - uses existing Playwright MCP integration
- Human sees their actual browser - no sync mechanism needed
- Leverages logged-in sessions and auth state
- Agent uses superior accessibility tree approach

**Trade-offs accepted:**
- Human must have browser on their screen (can't be headless)
- Same-machine only in Phase 1 (no remote sharing)
- "Last action wins" control model initially

**Implementation sequence:**
1. Human installs Playwright MCP Bridge extension in Chrome
2. Configure orch spawn to use `--extension` mode for browser tasks
3. Document workflow: human opens browser → agent connects → shared session begins
4. Add to orchestrator skill as spawn option pattern

### Alternative Approaches Considered

**Option B: VNC + Headless Browser**
- **Pros:** Works cross-machine, agent can run browser independently
- **Cons:** Higher latency, additional VNC infrastructure, less natural UX
- **When to use instead:** If human needs to observe from different machine than agent runs on

**Option C: browser-use Cloud**
- **Pros:** Managed infrastructure, stealth browsers, sandboxed
- **Cons:** Python dependency, external service, different agent loop model
- **When to use instead:** If needing cloud-based automation at scale without local browser

**Option D: Custom CDP Client**
- **Pros:** Maximum flexibility, custom control protocol
- **Cons:** High implementation effort, reinventing Playwright's work
- **When to use instead:** If existing tools don't meet specialized requirements

**Rationale for recommendation:** Extension mode provides immediate value with minimal implementation - Dylan installs an extension, agent connects. No new servers, no sync protocols, no custom tooling needed for Phase 1.

---

### Implementation Details

**What to implement first:**
- Document extension installation and configuration steps
- Test that `--extension` mode works with existing `--mcp playwright` spawn pattern
- Create simple usage example in orchestrator docs

**Things to watch out for:**
- ⚠️ Extension requires Chrome/Edge (not Firefox/Safari support)
- ⚠️ Multiple tabs scenario - agent needs to target correct tab
- ⚠️ Race conditions if both human and agent act simultaneously
- ⚠️ Security: extension bridges agent to browser with full page access

**Areas needing further investigation:**
- How to display "agent is controlling" indicator to human
- Integration with orch daemon for autonomous browser tasks
- Recording/replay of shared sessions for debugging
- Cross-machine sharing if needed in future

**Success criteria:**
- ✅ Human can watch agent navigate a site in real-time
- ✅ Human can interrupt and take control when needed
- ✅ Agent actions appear in browser immediately (no lag > 1s)
- ✅ Auth state (logged-in sessions) is preserved and usable

---

## References

**Files Examined:**
- Playwright MCP GitHub README and configuration docs
- browser-use GitHub README and examples
- Chrome DevTools Protocol documentation

**Commands Run:**
```bash
# Searched for existing browser/playwright references in orch-go
rg -l "playwright|browser" /Users/dylanconlin/Documents/personal/orch-go --type md
```

**External Documentation:**
- https://github.com/microsoft/playwright-mcp - Playwright MCP server (24.8k stars)
- https://github.com/browser-use/browser-use - browser-use framework (74.2k stars)
- https://chromedevtools.github.io/devtools-protocol/ - CDP reference
- https://playwright.dev/docs/debug - Playwright debugging including Inspector

**Related Artifacts:**
- No prior investigations on this topic found in `kb context "shared"` results

---

## Investigation History

**2025-12-26 07:33:** Investigation started
- Initial question: What would it take for orchestrator and human to share a browser session?
- Context: Exploring collaborative browser automation for orch ecosystem

**2025-12-26 07:45:** Reviewed Playwright MCP documentation
- Discovered `--extension` mode for connecting to existing browser
- Noted accessibility tree approach vs screenshots

**2025-12-26 07:50:** Reviewed browser-use framework
- Higher-level Python framework with different architecture
- Less suitable for orch integration due to language mismatch

**2025-12-26 07:55:** Reviewed CDP protocol
- Foundation layer for all browser automation
- Supports multiple simultaneous clients

**2025-12-26 08:05:** Investigation completed
- Status: Complete
- Key outcome: Playwright MCP extension mode provides simplest path to shared browser experience

---

## Self-Review

- [x] Real test performed (reviewed actual documentation and capabilities)
- [x] Conclusion from evidence (based on documented features)
- [x] Question answered (clear recommendation for shared browser approach)
- [x] File complete (all sections filled)

**Self-Review Status:** PASSED

Note: This is a documentation-based investigation. The recommendations should be validated with hands-on testing of the extension mode before committing to implementation.
