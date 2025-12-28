# Session Synthesis

**Agent:** og-arch-glass-vs-playwright-28dec
**Issue:** orch-go-ow3g
**Duration:** 2025-12-28 → 2025-12-28
**Outcome:** success

---

## TLDR

Glass and Playwright serve distinct, complementary roles: Glass for collaborative browser work (human + agent share same Chrome), Playwright for headless verification and E2E tests. Skills should NOT migrate - both tools are needed, just with clear guidance on when to use each.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-28-inv-glass-vs-playwright-role-clarity.md` - Design investigation documenting role boundaries

### Files Modified
- None - this was a design investigation, not implementation

### Commits
- (pending) architect: glass vs playwright role clarity

---

## Evidence (What Was Observed)

- Glass connects to Dylan's Chrome via CDP on port 9222 (shared browser session)
  - Source: ~/.config/opencode/opencode.jsonc:36-40
- Playwright spawns isolated browser instances for testing
  - Source: ~/.config/opencode/opencode.jsonc:43-47
- Visual verification patterns in pkg/verify/visual.go already detect BOTH tools (lines 93-98)
- Spawn command supports `--mcp playwright` and `--mcp glass` flags (cmd/orch/main.go:256-257)
- Existing E2E tests use Playwright (web/tests/*.spec.ts - 6 test files)
- Glass CLI provides assert command for orchestrator validation gates
- Prior constraint kn-3c7aaf requires Glass for dashboard interactions

### Tests Run
```bash
# Verified Glass binary works
glass --help
# Output: 9 commands available including snap, screenshot, assert, mcp

# Verified MCP configurations
# Both Glass and Playwright configured in opencode.jsonc
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-28-inv-glass-vs-playwright-role-clarity.md` - Design decision on tool boundaries

### Decisions Made
- **Keep both tools:** Glass and Playwright serve different browser models (shared vs isolated)
- **No skill migration:** Skills don't need to change tools - they need clarity on when to use each
- **Documentation is the fix:** The confusion was about boundaries, not architecture

### Constraints Discovered
- Glass requires Chrome launched with `--remote-debugging-port=9222`
- Playwright requires npx in PATH (or absolute path)
- Don't spawn with both --mcp flags - choose one per spawn based on use case

### Key Distinction

| Dimension | Glass | Playwright |
|-----------|-------|------------|
| **Browser** | Dylan's Chrome (shared) | Isolated instance |
| **Primary User** | Orchestrator | Worker agents |
| **Use Case** | Dashboard interaction | E2E tests, verification |
| **MCP Flag** | Default (always available) | `--mcp playwright` |
| **CLI** | glass assert | npx playwright test |

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete - investigation with clear recommendations
- [x] Tests passing - N/A (design investigation)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-ow3g`

### Follow-up Items (not blocking close)
1. Update global CLAUDE.md with Glass vs Playwright guidance
2. Consider adding brief guidance to feature-impl skill about when to use each

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should there be a way to switch from Glass to Playwright mid-session? (Currently fixed at spawn time)
- Would a unified browser tool abstraction reduce confusion? (Probably over-engineering)

**Areas worth exploring further:**
- Performance comparison between Glass and Playwright for same tasks
- Whether workers should ever use Glass (currently assumed no)

**What remains unclear:**
- How often workers actually use `--mcp playwright` flag vs default no-MCP

---

## Session Metadata

**Skill:** architect
**Model:** claude-sonnet-4-5-20250929
**Workspace:** `.orch/workspace/og-arch-glass-vs-playwright-28dec/`
**Investigation:** `.kb/investigations/2025-12-28-inv-glass-vs-playwright-role-clarity.md`
**Beads:** `bd show orch-go-ow3g`
