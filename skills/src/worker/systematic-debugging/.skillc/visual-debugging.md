## Visual Debugging Tools

### snap - Screenshot CLI (Recommended)

```bash
snap                          # Capture screen, returns file path
snap list --json              # Find window IDs
snap window "Firefox"         # Capture by app name
snap --json                   # JSON output: {"path": "/path/to/screenshot.png"}
```

**Use for:** Verifying UI state, documenting visual bugs, smoke-testing UI changes.

**Advantage:** Zero context cost (returns file path, not image data).

### Browser Automation

**USE:** playwright-cli - standalone CLI for browser automation

**AVOID:** browser-use MCP - causes context explosion (screenshots, full DOM)

**Decision flow:**
1. Need visual verification? → `snap` (zero context cost)
2. Need browser automation (clicking, typing, DOM inspection)? → `playwright-cli`
3. Need headless/CI testing? → `playwright-cli`

**playwright-cli advantages:**
- Standalone CLI (no MCP server needed)
- 3x fewer tokens than MCP equivalent
- Full browser automation (open, click, snapshot, screenshot, eval)
- Works in tmux-spawned agents
