# Probe: Playwright CLI vs MCP for Browser Automation UX Audits

**Model:** orchestrator-session-lifecycle
**Date:** 2026-02-28
**Status:** Active

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

[To be filled as tests run]

---

## Model Impact

- [ ] **Confirms** invariant: Backend independence — CLI tools provide MCP-independent browser automation
- [ ] **Contradicts** invariant: [TBD]
- [ ] **Extends** model with: Comparative analysis of CLI vs MCP for autonomous agent browser tasks

---

## Notes

This is a head-to-head comparison. An MCP-based agent is performing the same UX audit simultaneously. Key comparison metrics: token count, tool calls, time taken, and quality of findings.
