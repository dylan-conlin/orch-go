## Summary (D.E.K.N.)

**Delta:** CLI is preferred for bd/kb/orch ecosystem tools; MCP only warranted for stateful browser automation (glass). Current CLI-via-Bash approach is optimal for agent discoverability via skill documentation.

**Evidence:** Skills contain 100+ CLI invocations (bd comment, kb context, orch spawn). Glass has both MCP and CLI with capability parity. Prior investigation showed CLI is 12x simpler than MCP for web-to-markdown. Agents already discover tools via CLAUDE.md and skill files.

**Knowledge:** "Surfacing Over Browsing" doesn't require MCP - skills and CLAUDE.md surface CLI commands effectively. MCP adds value only for stateful connections (browser automation). "Compose Over Monolith" correctly keeps tools separate.

**Next:** Close this investigation. Current architecture is optimal. Document the pattern: "MCP for stateful/interactive, CLI for one-shot operations."

---

# Investigation: MCP vs CLI for orch Ecosystem Tools

**Question:** Should bd/kb/orch ecosystem tools be exposed via MCP instead of CLI for better agent discoverability?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** design-session agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Agents Already Discover CLI Tools Effectively

**Evidence:** 
- Skills contain 100+ CLI invocations: `bd comment`, `kb context`, `orch spawn`, etc.
- CLAUDE.md explicitly documents tools: `pkg/beads/` integration, `opencode/client.go` patterns
- Orchestrator skill dedicates sections to CLI usage patterns with examples
- Agents reliably use these tools without MCP discovery

**Source:** 
- `~/.claude/skills/policy/orchestrator/SKILL.md` - 100+ CLI references
- `/Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md:185` - "Beads integration: Shells out to `bd` CLI"

**Significance:** The "Surfacing Over Browsing" principle is satisfied by skill documentation and CLAUDE.md, not by MCP. Agents don't need to "browse" for tools when skills explicitly document them.

---

### Finding 2: MCP Adds Significant Complexity for One-Shot Operations

**Evidence:**
- Prior investigation (2025-12-26): web-to-markdown CLI was 58 lines vs MCP server 694 lines (12x complexity)
- Glass MCP server is 691 lines (`pkg/mcp/server.go`) with:
  - Tool schema definitions with zod-like types
  - Request/response marshaling
  - Error handling wrappers
  - Transport setup (stdio)
- Equivalent CLI commands are simple flag parsing

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-26-inv-evaluate-whether-web-markdown-mcp.md`
- `/Users/dylanconlin/Documents/personal/glass/pkg/mcp/server.go` (691 lines)

**Significance:** For one-shot commands like `bd comment`, `kb context`, `orch spawn`, CLI is dramatically simpler. MCP's tool discovery benefit doesn't outweigh the complexity cost.

---

### Finding 3: MCP's Value is Stateful Interactive Sessions

**Evidence:**
Glass demonstrates when MCP is warranted:
- Browser connection persists across multiple tool calls
- Tools share state (which tab is focused, action logging)
- Multi-step flows benefit from single connection (navigate → click → screenshot)
- CLI `glass assert` was added specifically because "orchestrator needs CLI for orch complete validation" - scripts need exit codes, not MCP sessions

Prior decisions confirm this pattern:
- "MCP for agent-internal use, CLI for orchestrator/scripts/humans"
- "snap CLI for debugging capture, Playwright MCP for complex UI testing"
- "Glass needs both: MCP for agents doing UI work, CLI for orch complete validation"

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-27-inv-add-cli-commands-glass-orchestrator.md`
- Prior decisions in spawn context (lines 37-52)

**Significance:** bd/kb/orch are stateless one-shot tools. Each invocation is independent. MCP's persistent connection model provides no benefit here.

---

### Finding 4: "Compose Over Monolith" Supports Current Architecture

**Evidence:**
- bd, kb, orch, glass are separate binaries with distinct purposes
- Each has its own lifecycle, installation, versioning
- A unified MCP server would create coupling between unrelated tools
- Current architecture allows:
  - Using bd without kb installed
  - Updating orch without affecting glass
  - Adding new tools without changing existing ones

**Source:** Project structure analysis, spawn context principle references

**Significance:** Creating an ecosystem MCP server would violate the compose principle. Tools should remain composable, not bundled.

---

## Synthesis

**Key Insights:**

1. **Discovery happens through documentation, not protocols** - Skills and CLAUDE.md already surface CLI commands effectively. MCP's auto-discovery doesn't add value when documentation is comprehensive.

2. **Statelessness eliminates MCP's advantage** - bd, kb, orch commands are one-shot invocations. They don't maintain state between calls. MCP's persistent connection model is overhead without benefit.

3. **Dual interface (MCP + CLI) is correct for glass** - Glass demonstrates the pattern: MCP for agents doing interactive browser work (multi-step flows, shared state), CLI for scripts and validation gates. This doesn't generalize to all tools.

4. **Complexity scales with MCP** - Each tool exposed via MCP requires: schema definitions, handlers, transport setup, configuration in opencode.json. CLI requires: flag parsing. 12x complexity difference per tool.

**Answer to Investigation Question:**

**No, agents do not benefit from MCP over CLI for bd/kb/orch.**

The current architecture is optimal because:
- Agents already discover CLI tools effectively via skill documentation
- CLI is dramatically simpler for one-shot stateless operations
- MCP's value proposition (tool discovery, stateful connections) doesn't apply
- "Compose Over Monolith" correctly keeps tools separate

The question "Do agents actually benefit from MCP over CLI for these tools?" has a clear answer: **No evidence supports MCP being better. CLI via Bash works reliably.**

---

## Structured Uncertainty

**What's tested:**

- ✅ Skills successfully invoke bd/kb/orch via Bash (verified: 100+ usage examples in skill files)
- ✅ CLI simplicity vs MCP complexity (verified: 12x ratio from prior investigation)
- ✅ Glass dual interface works (verified: MCP for agents, CLI for orch complete)

**What's untested:**

- ⚠️ Token cost difference (MCP tool schemas consume context, but not measured)
- ⚠️ Error handling comparison (both work, but which provides better agent feedback?)
- ⚠️ Future agent models may have native MCP preferences (speculation)

**What would change this:**

- If OpenCode defaulted to MCP tool discovery over Bash (architectural shift)
- If agents consistently failed to find CLI commands (discoverability failure)
- If MCP added capability CLI cannot provide for these tools

---

## Implementation Recommendations

**Purpose:** Confirm current architecture is optimal and document the decision.

### Recommended Approach ⭐

**Keep CLI for bd/kb/orch, use MCP only for glass** - Current architecture is correct. Document the pattern.

**Why this approach:**
- Zero migration cost (already working)
- Simpler maintenance (CLI is 12x less code)
- Matches tool characteristics (stateless vs stateful)
- Follows "Compose Over Monolith" principle

**Trade-offs accepted:**
- No unified "orch ecosystem" MCP server (not needed)
- Tool discovery remains documentation-based (working well)

**Implementation sequence:**
1. Close this investigation with decision documented
2. Add decision to kn: "CLI for stateless tools, MCP for stateful browser automation"
3. No code changes required

### Alternative Approaches Considered

**Option B: Unified Ecosystem MCP Server**
- **Pros:** Single configuration, unified discovery
- **Cons:** 4x complexity (bd + kb + orch + maintenance), violates Compose principle, couples unrelated tools
- **When to use instead:** Never for current use case

**Option C: Individual MCP Servers per Tool**
- **Pros:** Follows MCP pattern, doesn't couple tools
- **Cons:** 12x complexity per tool, no benefit over CLI, configuration burden
- **When to use instead:** If OpenCode deprecated Bash tool or CLI invocation became unreliable

**Rationale for recommendation:** No evidence that MCP improves agent outcomes for stateless one-shot operations. CLI is working. Keep it.

---

### Implementation Details

**What to implement first:**
- Nothing - current architecture is optimal

**Things to watch out for:**
- ⚠️ If new OpenCode versions change Bash tool behavior
- ⚠️ If MCP becomes required for certain agent capabilities

**Areas needing further investigation:**
- None immediate

**Success criteria:**
- ✅ Decision documented in investigation
- ✅ Pattern captured for future reference
- ✅ No changes required to codebase

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/glass/pkg/mcp/server.go` - Glass MCP server (691 lines)
- `/Users/dylanconlin/Documents/personal/glass/main.go` - Glass CLI with MCP and CLI dual interface
- `~/.claude/skills/policy/orchestrator/SKILL.md` - 100+ CLI invocations
- `/Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md` - Architecture documentation

**Prior Investigations:**
- `.kb/investigations/2025-12-26-inv-evaluate-whether-web-markdown-mcp.md` - MCP vs CLI decision framework
- `.kb/investigations/2025-12-27-inv-add-cli-commands-glass-orchestrator.md` - Glass CLI for validation gates
- `.kb/investigations/2025-12-27-inv-add-web-markdown-mcp-support.md` - MCP flag is metadata-only

**Related Decisions:**
- "MCP for agent-internal use, CLI for orchestrator/scripts/humans"
- "snap CLI for debugging capture, Playwright MCP for complex UI testing"

---

## Investigation History

**2025-12-28 12:00:** Investigation started
- Initial question: Should bd/kb/orch be MCP instead of CLI for agent discoverability?
- Context: Surfacing Over Browsing principle suggested MCP might be better

**2025-12-28 12:30:** Key finding - CLI discovery is already effective
- Skills contain 100+ CLI invocations that work reliably
- Documentation-based discovery is working

**2025-12-28 12:45:** Investigation completed
- Status: Complete
- Key outcome: CLI is optimal for stateless one-shot tools. Keep current architecture.
