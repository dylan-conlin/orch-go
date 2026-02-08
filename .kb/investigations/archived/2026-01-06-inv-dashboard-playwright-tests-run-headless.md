<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Playwright MCP server config was missing `--headless` flag, causing visible browser windows when spawning agents with `--mcp playwright`.

**Evidence:** Playwright MCP help shows `--headless  run browser in headless mode, headed by default` - global opencode.jsonc had no `--headless` flag.

**Knowledge:** MCP servers have different defaults than native Playwright tests. Native tests default to headless; MCP server defaults to headed mode for interactive debugging.

**Next:** Close - fix applied to `~/.config/opencode/opencode.jsonc` (dotfiles commit a4882eb).

---

# Investigation: Dashboard Playwright Tests Run Headless

**Question:** Why does `--mcp playwright` spawn visible browser windows instead of running headless?

**Started:** 2026-01-06
**Updated:** 2026-01-06
**Owner:** Agent (og-debug-dashboard-playwright-tests-06jan-f25f)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Two distinct Playwright usages in the ecosystem

**Evidence:** 
- `web/playwright.config.ts` - Standard Playwright E2E tests (default headless)
- `--mcp playwright` flag - Playwright MCP server for agent browser automation (default headed)

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/web/playwright.config.ts`
- `npx @playwright/mcp --help` shows: `--headless  run browser in headless mode, headed by default`

**Significance:** These are completely different tools with opposite defaults. The spawn context refers to MCP, not E2E tests.

---

### Finding 2: Global OpenCode config controls Playwright MCP behavior

**Evidence:** 
- Global config at `~/.config/opencode/opencode.jsonc` defines MCP servers
- Playwright MCP was configured as:
  ```json
  "command": ["npx", "@playwright/mcp@latest", "--viewport-size=1440x900"]
  ```
- Missing `--headless` flag caused headed (visible) browser mode

**Source:** `/Users/dylanconlin/Documents/dotfiles/.config/opencode/opencode.jsonc:44-49`

**Significance:** The fix requires adding `--headless` to the command array in the global config.

---

### Finding 3: orch spawn --mcp playwright doesn't dynamically configure MCP

**Evidence:**
- `cfg.MCP` value is only used for logging and event data
- No code translates `--mcp playwright` into MCP server configuration
- MCP enablement relies on the pre-configured server in opencode.jsonc

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go` (grep for `cfg.MCP` shows only logging uses)

**Significance:** The `--mcp` flag controls which pre-configured MCP server to enable, not how to configure it.

---

## Synthesis

**Key Insights:**

1. **Default behavior mismatch** - Playwright MCP defaults to headed mode for interactive debugging, but agent automation needs headless mode.

2. **Config-driven MCP** - MCP server behavior is controlled by opencode.jsonc, not dynamically by orch spawn flags.

3. **Simple fix** - Adding `--headless` to the Playwright MCP command in global config solves the issue.

**Answer to Investigation Question:**

The `--mcp playwright` flag spawns visible browsers because the Playwright MCP server defaults to headed mode, and the global OpenCode config was missing the `--headless` flag. The fix is to add `--headless` to the command array in `~/.config/opencode/opencode.jsonc`.

---

## Structured Uncertainty

**What's tested:**

- ✅ Playwright MCP help shows `--headless` option exists (verified: `npx @playwright/mcp --help`)
- ✅ Config change applied to correct file (verified: `git diff` shows change)
- ✅ Commit created in dotfiles repo (verified: commit a4882eb)

**What's untested:**

- ⚠️ Actual browser runs headless after config change (requires restarting OpenCode server)
- ⚠️ No regression in cases where headed mode is desired

**What would change this:**

- Finding would be wrong if Playwright MCP ignores `--headless` flag
- Finding would be wrong if there's a per-project override that sets headed mode

---

## Implementation Recommendations

**Purpose:** The fix is already implemented. This section documents the change.

### Recommended Approach ⭐

**Add --headless flag to Playwright MCP config** - Single line change in global opencode.jsonc

**Why this approach:**
- Addresses root cause directly
- No code changes required in orch-go
- Consistent with comment "Playwright is for isolated headless verification"

**Trade-offs accepted:**
- Users who want headed mode must explicitly remove `--headless`
- This is acceptable since agent automation should be invisible

**Implementation sequence:**
1. ✅ Edit `~/.config/opencode/opencode.jsonc` to add `--headless` flag
2. ✅ Commit change to dotfiles repo
3. Restart OpenCode server to pick up config change

### Alternative Approaches Considered

**Option B: Dynamic MCP configuration in orch spawn**
- **Pros:** More flexibility per spawn
- **Cons:** Requires significant code changes; MCP config is already well-structured
- **When to use instead:** If different spawns need different Playwright configs

**Rationale for recommendation:** The config-based approach is simpler, maintains existing patterns, and addresses the issue with minimal change.

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/dotfiles/.config/opencode/opencode.jsonc` - Global OpenCode config with MCP definitions
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/spawn_cmd.go` - How --mcp flag is used
- `/Users/dylanconlin/Documents/personal/playwright-mcp/config.d.ts` - Playwright MCP config types

**Commands Run:**
```bash
# Verify Playwright MCP --headless option exists
npx @playwright/mcp --help

# Verify config change
cd ~/Documents/dotfiles && git diff .config/opencode/opencode.jsonc

# Commit fix
git add .config/opencode/opencode.jsonc && git commit -m "fix: add --headless flag..."
```

**External Documentation:**
- Playwright MCP CLI help output - confirms `--headless` flag and default headed behavior

**Related Artifacts:**
- **Commit:** dotfiles:a4882eb - Fix applied
