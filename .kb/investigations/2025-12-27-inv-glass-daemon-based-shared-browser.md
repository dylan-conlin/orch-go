<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Glass MCP tools had double-prefix naming issue (`glass_glass_*`) preventing tools from appearing correctly in agents; fixed by removing prefix from tool registration since opencode adds MCP client name as prefix automatically.

**Evidence:** MCP tools/list returned `glass_tabs` when registered as `glass_tabs`, but opencode prefixes with client name → `glass_glass_tabs`. After fix, tools are registered as `tabs`, `page_state` etc. and become `glass_tabs`, `glass_page_state` via opencode.

**Knowledge:** MCP tool naming convention: tools should NOT include client name prefix - opencode adds `{client}_{toolname}` automatically. Binary corruption in ~/bin/ causes exit 137; requires recopy from source.

**Next:** Glass is ready for use. Consider disabling playwright MCP (currently failing - npx not in PATH) since glass provides equivalent functionality and connects to Dylan's actual browser tabs.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Glass Daemon Based Shared Browser

**Question:** Why do glass MCP tools not appear in orchestrator sessions despite glass being connected, and should we disable playwright MCP in favor of glass?

**Started:** 2025-12-27
**Updated:** 2025-12-27
**Owner:** Investigation Agent
**Phase:** Complete
**Next Step:** None - ready for orchestrator to decide on playwright MCP
**Status:** Complete

---

## Findings

### Finding 1: Double-Prefix Naming Issue

**Evidence:** 
- Glass MCP server registered tools as `glass_tabs`, `glass_page_state`, etc.
- OpenCode adds MCP client name as prefix automatically: `{client}_{toolname}`
- Result: tools became `glass_glass_tabs`, `glass_glass_page_state` - not matching expected names
- Prior investigation (orch-go-ixxg) noted tools weren't appearing but didn't identify the naming cause

**Source:** 
- `/Users/dylanconlin/Documents/personal/glass/pkg/mcp/server.go:114-206` - tool registration
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/mcp/index.ts:407-410` - prefix logic
- MCP tools/list output: `{"name":"glass_tabs"}` → becomes `glass_glass_tabs` via opencode

**Significance:** This explains why glass tools weren't usable in agents. The naming convention requires tools to NOT include the client name prefix.

---

### Finding 2: Binary Corruption Still Occurring

**Evidence:**
- Newly installed binary at `~/bin/glass` returned exit code 137 (SIGKILL)
- Project binary at `~/Documents/personal/glass/glass` worked correctly
- Binaries had identical checksums
- Manual recopy with `cp` fixed the issue

**Source:**
- Command: `/Users/dylanconlin/bin/glass tabs` → exit 137
- Command: `/Users/dylanconlin/Documents/personal/glass/glass tabs` → success
- `shasum` showed identical content

**Significance:** The `make install` process may have a timing issue or the destination file is somehow corrupted during copy. Root cause unclear but workaround is manual copy.

---

### Finding 3: Playwright MCP Failing Due to PATH

**Evidence:**
- `opencode mcp list` shows: `✗ playwright [failed] Executable not found in $PATH: "npx"`
- Playwright config uses: `["npx", "@playwright/mcp@latest", "--viewport-size=1440x900"]`
- npx not in PATH for spawned agent environments

**Source:**
- Command: `opencode mcp list`
- Config: `~/.config/opencode/opencode.jsonc:43-47`

**Significance:** Glass uses absolute path (`/Users/dylanconlin/bin/glass`) which is more reliable. Playwright could be fixed by using absolute path to npx.

---

## Synthesis

**Key Insights:**

1. **MCP tool naming convention is counterintuitive** - Tools should NOT include the client name prefix because the MCP client (opencode) adds it automatically. This is an easy mistake to make when building custom MCP servers.

2. **Binary installation reliability** - Even with `make install`, binaries can get corrupted. The safest approach is to verify the binary works after installation (`glass tabs`).

3. **Glass vs Playwright positioning** - Glass is purpose-built for Dylan's workflow (connects to actual browser tabs), while Playwright is more general-purpose (headless, spawns new instances). For orchestrator dashboard validation, Glass is the better fit.

**Answer to Investigation Question:**

Glass MCP tools weren't appearing because of a double-prefix naming issue - tools registered as `glass_*` became `glass_glass_*` when exposed via opencode. This has been fixed by removing the prefix from tool registration.

Regarding playwright vs glass:
- **Recommend disabling playwright MCP** for orchestrator sessions
- **Reason:** Glass is working, connects to Dylan's actual browser tabs, and is designed for this workflow
- **Playwright is failing anyway** (npx not in PATH)
- **If keeping both:** Fix playwright by using absolute path to npx

---

## Structured Uncertainty

**What's tested:**

- ✅ MCP tools are now named without prefix (verified: tools/list returns `tabs`, `page_state`, etc.)
- ✅ Glass binary works after manual recopy (verified: `/Users/dylanconlin/bin/glass tabs` succeeds)
- ✅ Playwright MCP is failing (verified: `opencode mcp list` shows failed status)
- ✅ Visual verification in orch-go detects glass tools (verified: regex patterns in visual.go)

**What's untested:**

- ⚠️ Glass MCP tools actually appear in new agent sessions (requires session restart)
- ⚠️ Whether disabling playwright MCP has any side effects for workers spawned with `--mcp playwright`
- ⚠️ Root cause of binary corruption during `make install`

**What would change this:**

- If glass tools still don't appear after session restart → further opencode investigation needed
- If playwright is needed for some workflows → keep both, fix npx path
- If binary corruption recurs → need to investigate macOS security or file system issues

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Disable playwright MCP, use glass exclusively for orchestrator** - Glass is working and purpose-built for Dylan's workflow.

**Why this approach:**
- Glass connects to Dylan's actual Chrome tabs (not headless)
- Glass is currently working; playwright is failing (npx PATH issue)
- Existing kn constraint: "orchestrator uses Glass for all browser interactions"
- Reduces complexity (one MCP server instead of two)

**Trade-offs accepted:**
- Workers spawned with `--mcp playwright` won't have playwright available
- Need to update spawn documentation if playwright was intended for workers
- If playwright is needed later, can re-enable

**Implementation sequence:**
1. **Verify glass tools work in fresh session** - Start new orchestrator session, check glass_* tools available
2. **Disable playwright in opencode.jsonc** - Set `enabled: false` for playwright
3. **Update documentation** - Note that glass is the browser automation tool

### Alternative Approaches Considered

**Option B: Keep both, fix playwright npx issue**
- **Pros:** Maximum flexibility, both tools available
- **Cons:** Complexity, playwright is more heavyweight than needed
- **When to use instead:** If workers need headless browser automation for CI/testing

**Option C: Fix npx path for playwright, disable glass**
- **Pros:** Playwright is more mature/documented
- **Cons:** Playwright doesn't connect to Dylan's actual tabs, glass is purpose-built
- **When to use instead:** Never - glass is the right tool for this use case

**Rationale for recommendation:** Glass was built specifically for Dylan's workflow (CDP to existing tabs). Playwright is overkill and currently broken. Simplify by using only what works and fits the use case.

---

### Implementation Details

**What to implement first:**
- Verify glass works in fresh session (restart opencode server or start new session)
- Test glass_tabs, glass_page_state, glass_click tools
- Disable playwright in config if glass works

**Things to watch out for:**
- ⚠️ Binary at ~/bin/glass may get corrupted - always verify after install
- ⚠️ Chrome must be running with `--remote-debugging-port=9222`
- ⚠️ opencode may cache MCP status - may need full restart

**Areas needing further investigation:**
- Why does make install sometimes corrupt the binary?
- Should orch spawn have a `--mcp glass` shorthand?
- Auto-launch Chrome with debugging port when needed?

**Success criteria:**
- ✅ `opencode mcp list` shows glass=connected
- ✅ glass_* tools appear in agent tool list
- ✅ Can use glass_tabs, glass_click, etc. in agent sessions
- ✅ Orchestrator can verify dashboard UI via glass tools

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/glass/pkg/mcp/server.go` - MCP tool registration
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/mcp/index.ts` - MCP prefix logic
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/verify/visual.go` - Glass detection patterns
- `~/.config/opencode/opencode.jsonc` - MCP server configuration

**Commands Run:**
```bash
# Verify glass binary
/Users/dylanconlin/bin/glass tabs
# → Exit 137 (corrupted), then success after recopy

# Check MCP status
opencode mcp list
# → glass: connected, playwright: failed

# Test MCP tools list
echo '{"jsonrpc":"2.0","id":1,"method":"initialize"...}' | /Users/dylanconlin/bin/glass mcp
# → tools: click, elements, focus, navigate, page_state, recent_actions, tabs, type, enable_user_tracking
```

**External Documentation:**
- OpenCode MCP docs - How MCP clients prefix tool names

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-27-inv-orchestrator-see-playwright-browser-tools.md` - Prior investigation on glass binary corruption
- **Investigation:** `.kb/investigations/2025-12-27-inv-glass-integration-status-orch-ecosystem.md` - Glass integration status
- **kn:** kn-3c7aaf - Constraint: orchestrator uses Glass for browser interactions

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
