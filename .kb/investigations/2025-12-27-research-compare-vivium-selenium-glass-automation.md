<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Vibium (by Jason Huggins, Selenium/Appium creator) uses WebDriver BiDi protocol for AI-native browser automation; CDP-based tools (Playwright, Puppeteer) use Chrome DevTools Protocol. Vibium is designed for MCP integration with LLMs.

**Evidence:** Vibium GitHub (1.7k stars), V1/V2 roadmaps, architecture docs. "Glass" as described doesn't exist - useglass.ai is a discontinued React AI coding tool, not browser automation.

**Knowledge:** Vibium's key innovation is MCP-first design + WebDriver BiDi (cross-browser standard). CDP tools are more mature but Chrome-centric. For AI agent browser control, Vibium offers simpler integration via `claude mcp add vibium`.

**Next:** Close investigation. Recommend Vibium for new AI agent browser automation projects; CDP tools for mature test suites needing network interception/performance profiling.

---

# Research: Vibium (Selenium Creator) vs CDP Browser Automation Approaches

**Question:** What is Vibium's architecture and how does it differ from CDP-based browser automation? What are the tradeoffs (reliability, speed, capabilities, AI-native features)?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Research agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Confidence:** High (85%)

---

## Findings

### Finding 1: Vibium Architecture and Purpose

**Evidence:**
- **Creator:** Jason Huggins (Selenium and Appium creator, NOT Simon Stewart)
- **Purpose:** "Browser automation for AI agents and humans"
- **Protocol:** WebDriver BiDi (not CDP)
- **Core Component:** "Clicker" - single Go binary (~10MB) that handles:
  - Browser lifecycle management
  - WebDriver BiDi proxy
  - MCP server for LLM integration
- **Published:** npm package `vibium` (December 2025)
- **Stars:** 1.7k on GitHub
- **License:** Apache 2.0

**Architecture:**
```
LLM/Agent (Claude, Codex, Gemini)
       ↓ MCP Protocol (stdio)
    Vibium Clicker
       ↓ WebSocket BiDi
    Chrome Browser
```

**Source:** https://github.com/VibiumDev/vibium

**Significance:** Vibium is purpose-built for AI agents with MCP as a first-class interface. It's the first major browser automation tool designed for LLM integration from the ground up.

---

### Finding 2: WebDriver BiDi vs Chrome DevTools Protocol (CDP)

**Evidence:**

| Aspect | WebDriver BiDi (Vibium) | Chrome DevTools Protocol (Playwright, Puppeteer) |
|--------|------------------------|--------------------------------------------------|
| **Standard** | W3C standard (cross-browser) | Chrome-specific (adopted by Edge, partially Firefox) |
| **Browser Support** | Chrome, Firefox, Edge (planned) | Chrome, Chromium-based browsers primarily |
| **Connection** | Bidirectional WebSocket | Bidirectional WebSocket |
| **Event Model** | Event subscriptions | Domain-based events |
| **Maturity** | Emerging (2023-2025) | Mature (2017+) |
| **Selenium Relation** | Selenium 4+ includes BiDi support | Selenium 4 added CDP access |

**Key Difference:** BiDi is designed as a cross-browser standard that all browsers can implement. CDP is Chrome-native and requires each browser to adopt Chrome's protocol.

**Source:** 
- https://www.selenium.dev/documentation/webdriver/bidi/cdp/
- https://github.com/VibiumDev/vibium/blob/main/README.md

**Significance:** Vibium's choice of BiDi positions it for cross-browser support as the standard matures, while CDP-based tools are locked to Chrome/Chromium.

---

### Finding 3: CDP-Based Tools (Playwright, Puppeteer, Chrome DevTools MCP)

**Evidence:**

**Playwright (Microsoft):**
- Multi-browser support via CDP + proprietary Firefox/WebKit protocols
- Rich API: network interception, device emulation, tracing
- Primary use: E2E testing, web scraping
- MCP: Not native, requires separate MCP server implementation

**Puppeteer (Google):**
- Chrome/Chromium only via CDP
- Lighter weight than Playwright
- Foundation for many CDP-based tools
- MCP: Not native

**Chrome DevTools MCP (Google):**
- Official MCP server from Chrome team (Sep 2025)
- Uses CDP under the hood via Puppeteer
- Focus: AI-assisted debugging (breakpoints, performance profiling)
- Not general-purpose browser automation

**Source:**
- https://developer.chrome.com/blog/chrome-devtools-mcp
- https://playwright.dev/
- https://pptr.dev/

**Significance:** CDP tools excel at deep browser debugging but lack native MCP integration. Vibium is simpler for AI agent use but less capable for advanced debugging.

---

### Finding 4: Glass Browser Automation (Clarification)

**Evidence:**
- **useglass.ai** - "Glass Devtools, Inc." - Y Combinator company
- **GlassJS** - Discontinued AI coding tool for React/Next.js
- **NOT** a CDP-based browser automation tool
- Current focus: Void Editor (open source code editor with LLM support)

**Source:** https://useglass.ai/

**Significance:** "Glass" as referenced in the original task doesn't exist as a browser automation tool. The comparison should be Vibium vs CDP-based approaches (Playwright, Puppeteer, Chrome DevTools MCP).

---

### Finding 5: Vibium V2 Roadmap (Future Features)

**Evidence:**
Planned but not yet implemented:
- **Cortex:** SQLite-backed "app map" with navigation memory
- **Retina:** Chrome extension for recording sessions
- **AI-powered locators:** `vibe.do("click the login button")`
- **Video recording:** Built-in screen recording
- **Python/Java clients**

**Source:** https://github.com/VibiumDev/vibium/blob/main/V2-ROADMAP.md

**Significance:** Vibium's roadmap shows ambition beyond basic automation into AI-native features that CDP tools don't have.

---

## Options Evaluated

### Option 1: Vibium (WebDriver BiDi + MCP)

**Overview:** AI-native browser automation with single-command MCP setup.

**Pros:**
- Zero-setup MCP integration: `claude mcp add vibium -- npx -y vibium`
- Single binary handles browser + protocol + MCP server
- WebDriver BiDi is a W3C standard (cross-browser future)
- Sync and async JS APIs
- Auto-wait/actionability checks (Playwright-inspired)
- AI-focused design from day one

**Cons:**
- Very new (December 2025)
- Limited feature set (no network interception, no video recording yet)
- Chrome-only for now (Firefox/Edge planned)
- No Python/Java clients yet
- Community and ecosystem immature

**Evidence:**
- V1 shipped Dec 22, 2025
- 1.7k GitHub stars (rapid growth)
- 105 commits, active development

---

### Option 2: CDP-Based Tools (Playwright, Puppeteer)

**Overview:** Mature browser automation via Chrome DevTools Protocol.

**Pros:**
- Battle-tested (Puppeteer since 2017, Playwright since 2020)
- Rich features: network interception, device emulation, tracing
- Strong community and ecosystem
- Multiple language bindings (JS, Python, Java, C#)
- Playwright supports Firefox/WebKit via custom protocols

**Cons:**
- No native MCP support (requires custom integration)
- More complex setup for AI agent use
- CDP is Chrome-centric (not a true cross-browser standard)
- Designed for testing, not AI agents

**Evidence:**
- Playwright: 72k GitHub stars
- Puppeteer: 89k GitHub stars

---

### Option 3: Chrome DevTools MCP (Google)

**Overview:** Official Google MCP server for browser debugging.

**Pros:**
- Official Google support
- Deep debugging: breakpoints, performance profiling, network inspection
- Uses battle-tested Puppeteer under the hood

**Cons:**
- Focused on debugging, not general automation
- More complex than Vibium for simple browser control
- Chrome-only

**Evidence:**
- Released Sep 2025
- Part of official Chrome DevTools initiative

---

## Recommendation

**I recommend Vibium for new AI agent browser automation projects** because it provides the simplest path to MCP integration with zero configuration. Key factors:

1. **MCP-first design:** Single command setup vs custom integration for CDP tools
2. **AI-native features:** Actionability checks, auto-wait, planned AI locators
3. **Cross-browser future:** WebDriver BiDi positions for Firefox/Edge support
4. **Active development:** Rapid iteration with clear V2 roadmap

**Trade-offs I'm accepting:**
- Fewer features than Playwright/Puppeteer (no network interception)
- Immature ecosystem (new, limited community resources)
- Chrome-only for now

**When this recommendation changes:**
- If you need network interception/mocking → Use Playwright
- If you need performance profiling/debugging → Use Chrome DevTools MCP
- If you need Python/Java → Wait for Vibium V2 or use Playwright
- If you need production-hardened test automation → Use Playwright/Puppeteer

---

## Confidence Assessment

**Current Confidence:** High (85%)

**What's certain:**
- Vibium exists and uses WebDriver BiDi (verified via GitHub repo)
- Vibium has native MCP server (verified via README and package.json)
- CDP tools don't have native MCP (verified via documentation)
- "Glass" as a CDP browser automation tool doesn't exist (verified)

**What's uncertain:**
- Vibium's reliability in production (too new to have track record)
- WebDriver BiDi performance vs CDP (not benchmarked)
- Vibium V2 features timeline (roadmap only, no dates)

**What would increase confidence to 95%+:**
- Run Vibium in production AI agent workflow for 1 week
- Benchmark Vibium vs Playwright for common automation tasks
- Test cross-browser support when Firefox/Edge ships

---

## Implementation Recommendations

### For AI Agent Browser Control (New Projects)

**Recommended Approach:** Start with Vibium

**Why:**
- Simplest MCP integration
- Designed for LLM use from day one
- Active development addressing AI-specific needs

**Implementation sequence:**
1. `npm install vibium` or `claude mcp add vibium -- npx -y vibium`
2. Use MCP tools: `browser_launch`, `browser_navigate`, `browser_click`, etc.
3. For programmatic use: `import { browser } from 'vibium'`

### For Test Automation / Complex Scraping

**Recommended Approach:** Use Playwright

**Why:**
- Mature, battle-tested
- Rich feature set for network interception, multi-browser
- Strong ecosystem for test frameworks

### For Debugging / Performance Analysis

**Recommended Approach:** Use Chrome DevTools MCP

**Why:**
- Official Google support
- Deep debugging capabilities
- Designed for AI-assisted code debugging

---

## Self-Review

- [x] Each option has evidence with sources
- [x] Clear recommendation (not "it depends")
- [x] Confidence assessed honestly
- [x] Research file complete and committed

**Self-Review Status:** PASSED

---

## References

**Primary Sources:**
- https://github.com/VibiumDev/vibium - Vibium GitHub repository
- https://github.com/VibiumDev/vibium/blob/main/V1-ROADMAP.md - V1 Roadmap
- https://github.com/VibiumDev/vibium/blob/main/V2-ROADMAP.md - V2 Roadmap
- https://vibium.com - Official website
- https://developer.chrome.com/blog/chrome-devtools-mcp - Chrome DevTools MCP

**Secondary Sources:**
- https://skakarh.medium.com/vibium-ai-the-future-of-test-automation-65cdb4b90360 - Medium article
- https://useglass.ai/ - Glass Devtools (clarification that Glass is not browser automation)
- https://www.selenium.dev/documentation/webdriver/bidi/cdp/ - Selenium BiDi/CDP docs

---

## Investigation History

**2025-12-27 Initial:** Investigation started
- Original question: Compare "Vivium" vs "Glass" browser automation
- Context: Research request for AI agent browser automation

**2025-12-27 Correction:** Name clarified
- "Vivium" → "Vibium" (correct spelling)
- Jason Huggins creator (not Simon Stewart)
- Simon Stewart = WebDriver creator, still at Selenium
- Jason Huggins = Selenium/Appium creator, now building Vibium

**2025-12-27 Glass clarification:**
- useglass.ai = discontinued AI coding tool for React/Next.js
- Not a CDP-based browser automation tool
- Pivoted comparison to Vibium vs CDP approaches

**2025-12-27 Complete:**
- Status: Complete
- Key outcome: Vibium recommended for new AI agent projects; CDP tools for complex test automation
