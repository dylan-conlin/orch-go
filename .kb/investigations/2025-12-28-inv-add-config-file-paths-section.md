<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Added Config File Locations section to orchestrator skill documenting paths to ~/.orch/config.yaml, opencode.json, and MCP server configs.

**Evidence:** Orchestrator skill updated in 3 locations (meta/orchestrator SKILL.md.template, meta/orchestrator .skillc/SKILL.md, policy/orchestrator SKILL.md). Skills redeployed to ~/.claude/skills/.

**Knowledge:** Config files are at: ~/.orch/config.yaml (orch-go), {project}/opencode.json (OpenCode), ~/Library/Application Support/Claude/claude_desktop_config.json (Claude Desktop MCP).

**Next:** Close issue - config paths are now documented in orchestrator skill.

---

# Investigation: Add Config File Paths Section

**Question:** Where are the key config files for the orchestration system located?

**Started:** 2025-12-28
**Updated:** 2025-12-28
**Owner:** Agent spawn
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: orch-go config location

**Evidence:** `~/.orch/config.yaml` exists with content:
```yaml
backend: opencode
auto_export_transcript: true
```

**Source:** `ls -la ~/.orch/config.yaml` and `cat ~/.orch/config.yaml`

**Significance:** Primary orch-go user config. Also `~/.orch/accounts.yaml` for Claude Max accounts.

---

### Finding 2: OpenCode uses project-level config only

**Evidence:** Found `opencode.json` files in multiple project directories:
- `/Users/dylanconlin/orch-knowledge/opencode.json`
- `/Users/dylanconlin/Documents/personal/orch-cli/opencode.json`
- `/Users/dylanconlin/Documents/personal/beads-ui-svelte/opencode.json`

No global OpenCode config at `~/.config/opencode/` or `~/.opencode/`.

**Source:** `glob **/opencode.json` and checking standard config locations

**Significance:** OpenCode config is per-project, not global. Each project can have its own MCP servers and instructions.

---

### Finding 3: Claude Desktop MCP server config location

**Evidence:** MCP servers configured in `~/Library/Application Support/Claude/claude_desktop_config.json` with structure:
```json
{
  "mcpServers": {
    "server-name": { "command": "...", "args": [...], "env": {...} }
  }
}
```

**Source:** `cat ~/Library/Application\ Support/Claude/claude_desktop_config.json`

**Significance:** This is where Claude Desktop's MCP servers are configured. OpenCode uses per-project opencode.json instead.

---

## Synthesis

**Key Insights:**

1. **Orchestration config is centralized** - ~/.orch/ contains orch-go config, accounts, registry, and daemon logs.

2. **OpenCode config is distributed** - Each project has its own opencode.json, no global config.

3. **MCP configs split by client** - Claude Desktop uses ~/Library/Application Support/Claude/, OpenCode uses project-level opencode.json.

**Answer to Investigation Question:**

Key config files are at:
- Orch: `~/.orch/config.yaml`, `~/.orch/accounts.yaml`
- OpenCode: `{project}/opencode.json`  
- Claude Desktop MCP: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Claude CLI: `~/.claude/CLAUDE.md` (global), `{project}/CLAUDE.md` (project)

---

## Structured Uncertainty

**What's tested:**

- ✅ ~/.orch/config.yaml exists and contains expected content (verified: cat command)
- ✅ opencode.json is per-project (verified: glob found multiple project copies, no global)
- ✅ Claude Desktop MCP config location (verified: cat command showed mcpServers structure)

**What's untested:**

- ⚠️ Other possible OpenCode config locations not checked exhaustively

**What would change this:**

- OpenCode adding a global config file in future versions

---

## Implementation Recommendations

**Purpose:** Document config paths in orchestrator skill so orchestrator doesn't waste time searching.

### Recommended Approach ⭐

**Add Config File Locations section to orchestrator skill**

**Why this approach:**
- Direct solution to the stated problem
- Section placed early in skill (before Tool Ecosystem)
- Covers all config types: orch, opencode, MCP, Claude CLI

**Implementation sequence:**
1. Update SKILL.md.template in meta/orchestrator
2. Update .skillc/SKILL.md in meta/orchestrator
3. Update policy/orchestrator SKILL.md
4. Deploy skills via `skills deploy`

---

## References

**Files Examined:**
- `~/.orch/config.yaml` - orch-go user config
- `~/Library/Application Support/Claude/claude_desktop_config.json` - Claude Desktop MCP config
- `/Users/dylanconlin/orch-knowledge/opencode.json` - example project OpenCode config

**Commands Run:**
```bash
# Check orch config
ls -la ~/.orch/config.yaml && cat ~/.orch/config.yaml

# Find opencode.json files
glob **/opencode.json

# Check Claude Desktop MCP config
cat ~/Library/Application\ Support/Claude/claude_desktop_config.json
```

**Files Modified:**
- `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template`
- `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md`
- `/Users/dylanconlin/orch-knowledge/skills/src/policy/orchestrator/SKILL.md`

---

## Investigation History

**2025-12-28 10:16:** Investigation started
- Initial question: Where are key config files located so orchestrator can reference them directly?
- Context: Constraint added requiring orchestrator skill to document config paths

**2025-12-28 10:25:** Investigation completed
- Status: Complete
- Key outcome: Added Config File Locations section to orchestrator skill with paths to all key config files
