# Probe: Playwright CLI vs MCP for Browser Automation UX Audits

**Model:** orchestrator-session-lifecycle
**Date:** 2026-02-28
**Status:** Complete
**Updated:** 2026-03-02

---

## Question

Can Playwright CLI (`npx playwright screenshot`, scripts via `npx playwright test`) replace Playwright MCP for browser automation tasks like UX audits? What are the trade-offs in capability, friction, and token cost?

This tests the model's claims about tool selection — specifically whether backend independence matters for browser automation (the architectural principle that "critical paths need independent secondary mechanisms").

---

## What I Tested

### Test 1: Basic screenshot capture via Playwright CLI

```bash
npx playwright screenshot http://localhost:5188 screenshot.png --viewport-size=1280,800
```

### Test 2: Multiple viewport screenshots for responsiveness

```bash
npx playwright screenshot http://localhost:5188 --viewport-size=375,812 mobile.png
npx playwright screenshot http://localhost:5188 --viewport-size=768,1024 tablet.png
npx playwright screenshot http://localhost:5188 --viewport-size=1920,1080 desktop.png
```

### Test 3: Scripted interaction via Playwright test runner

```javascript
// playwright-audit.spec.js - hover, click, inspect elements
```

### Test 4: Element inspection and data extraction

```javascript
// Extract DOM state, text content, element counts
```

---

## What I Observed

### Test 1 & 2: Screenshot CLI — Works, Fast, Simple

- `npx playwright screenshot <url> <file> --viewport-size=W,H` — ~1 second per screenshot
- Full-page (`--full-page`), PDF (`npx playwright pdf`), device emulation (`--device "iPhone 11"`) all work
- Built-in `--wait-for-selector` and `--wait-for-timeout` options
- **Gotcha:** `--wait-for-timeout` is an explicit *delay* (wait N ms), NOT a timeout cap. Use `--timeout` for caps.
- **Gotcha:** `--wait-for-selector` with non-matching selector hangs indefinitely (no default timeout). Always pair with `--timeout`.

### Test 3 & 4: Scripted Playwright — Full DOM Access

- Node.js scripts with `require('playwright')` give full page interaction: click, type, extract text, evaluate JS
- Extracted nav items, page title, full body text in 2.7 seconds
- **Friction:** Requires `npm install playwright` locally (14MB node_modules) — the `npx` CLI reuses global install but scripts need the module
- Browser binaries shared (520MB in `~/Library/Caches/ms-playwright/`) — same cache for both CLI and scripts

### SSE Page Behavior

- Dashboard at localhost:5188 uses SSE for real-time updates
- `--wait-for-timeout 3000` works as a simple delay to let SSE data populate
- Confirmed prior constraint: SSE pages must NOT use networkidle (would hang forever)

### Comparison: CLI vs MCP

| Dimension | Playwright CLI | Playwright MCP |
|-----------|---------------|----------------|
| **Setup** | `npx playwright install chromium` (one-time) | MCP server config + launch per session |
| **Screenshots** | `npx playwright screenshot` (~1s) | `browser_take_screenshot` tool call |
| **DOM extraction** | Requires writing a .js script | `browser_evaluate` tool call |
| **Click/interact** | Requires writing a .js script | `browser_click` tool call |
| **Token cost** | Very low — just bash command + image | Higher — MCP protocol overhead per action |
| **Agent friction** | Agent must compose bash commands or write scripts | Agent uses structured tool calls |
| **Multi-step flows** | Script all steps in one .js file, run once | One tool call per step (many round-trips) |
| **Error visibility** | Stderr output in terminal | MCP error responses |
| **Independence** | No server dependency | Requires MCP server running |
| **Capabilities** | Full Playwright API via scripts | Subset exposed via MCP tools |

### Key Economic Insight

For **screenshot-only tasks** (visual verification, responsive checks), Playwright CLI is strictly better:
- 1 bash tool call vs 3+ MCP tool calls (navigate, wait, screenshot)
- ~0 tokens for the command vs MCP protocol overhead
- ~1 second vs MCP server startup + navigation + capture

For **interactive tasks** (click buttons, fill forms, extract specific DOM data), there's a trade-off:
- CLI requires writing a script file first (Write tool + Bash tool = 2 calls minimum)
- MCP gives per-action granularity (useful when agent needs to react to page state)
- But multi-step CLI scripts are more token-efficient (all steps in one execution)

### Verdict

**Playwright CLI should be the default for visual verification.** MCP adds value only when agents need to interactively explore a page (state-dependent navigation, unknown page structure). For known-structure verification (dashboard screenshots, responsive checks), CLI is faster, cheaper, and more reliable.

---

## Model Impact

- [x] **Confirms** invariant: Backend independence — CLI tools provide MCP-independent browser automation
- [ ] **Contradicts** invariant: (none)
- [x] **Extends** model with: For visual verification, CLI is strictly better than MCP. MCP's value is limited to interactive exploration of unknown page structures.

---

## Recommendations

1. **Default to CLI** for `feature-impl` visual verification step: `npx playwright screenshot <url> <file> --viewport-size=W,H --wait-for-timeout 2000 --timeout 10000`
2. **Keep MCP available** for UX audit tasks that require page exploration
3. **Always pair** `--wait-for-selector` with `--timeout` to avoid hangs on SSE pages
4. **For DOM extraction**, write a one-shot .js script rather than using multiple MCP calls
