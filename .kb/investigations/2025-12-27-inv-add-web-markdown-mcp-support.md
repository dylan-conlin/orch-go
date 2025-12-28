# Investigation: Add web-to-markdown MCP Support for Research Skill

**Status:** Complete
**Started:** 2025-12-27
**Updated:** 2025-12-27

## TLDR

The `--mcp` flag in orch-go currently only logs/displays MCP server name but doesn't actually enable it. OpenCode MCP servers are configured via `opencode.json`, not CLI flags. Three options: 1) Document `web_to_markdown` tool already available via default MCP config, 2) Add `--mcp web-to-markdown` to dynamically create project opencode.json, 3) Update research skill to use existing WebFetch tool.

## Question

When spawning research agents that need to fetch external content (Substack, Reddit, HN, YouTube, etc), how should we enable the web-to-markdown MCP server?

## What I Tried

### 1. Analyzed Current MCP Flag Implementation

Searched for `--mcp` and `cfg.MCP` usage in orch-go codebase:

**Finding:** The `--mcp` flag exists but is only used for:
- Logging in event data (lines 1493-1494, 1580-1581, 1751-1752)
- Display in spawn summary (lines 1602-1603, 1785-1786)

**Evidence:** `BuildSpawnCommand` in `pkg/opencode/client.go:187` only accepts `prompt, title, model` - no MCP parameter.

### 2. Checked OpenCode CLI Options

Ran `opencode run --help`:

**Finding:** OpenCode CLI has no `--mcp` or `--mcp-server` flag. MCP servers are NOT configured via CLI arguments.

### 3. Investigated OpenCode MCP Configuration

Fetched OpenCode docs at `https://opencode.ai/docs/mcp-servers/`:

**Finding:** MCP servers are configured in `opencode.json` at project level:
```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "web_to_markdown": {
      "type": "local",
      "command": ["node", "/Users/dylanconlin/Documents/personal/mcp/web-to-markdown/index.js"],
      "enabled": true
    }
  }
}
```

### 4. Checked web-to-markdown MCP Structure

Located at `~/Documents/personal/mcp/web-to-markdown/`:
- Node.js MCP server
- Uses `@modelcontextprotocol/sdk`
- Provides tool(s) for converting URLs to markdown using Playwright

## What I Observed

1. **Gap Between Flag and Implementation:** The `--mcp` flag is decorative - it tracks which MCP was requested but doesn't actually enable it in OpenCode.

2. **OpenCode Configuration Architecture:** MCP servers must be configured in `opencode.json` before the session starts. They cannot be enabled mid-session or via CLI flags.

3. **Prior Decision Discovered:** kb context showed "Document existing capabilities before building new infrastructure" - WebFetch tool already exists in OpenCode.

## Options Evaluated

### Option 1: Document Existing WebFetch Tool (Minimal)

**Pros:**
- Zero infrastructure changes
- WebFetch already works for most JS-heavy sites
- Available in all OpenCode sessions by default

**Cons:**
- May not handle all edge cases (heavy SPA, auth-required sites)
- Less control over rendering than Playwright-based MCP

**Implementation:** Update research skill to mention using `WebFetch` tool for external content.

### Option 2: Pre-configure web-to-markdown MCP Globally

**Pros:**
- One-time setup
- Available in all projects

**Cons:**
- Adds to context for all sessions (even non-research)
- Requires manual setup per machine

**Implementation:** Add to `~/.opencode/opencode.json` (global config):
```json
{
  "mcp": {
    "web_to_markdown": {
      "type": "local", 
      "command": ["node", "~/Documents/personal/mcp/web-to-markdown/index.js"]
    }
  },
  "tools": {
    "web_to_markdown*": false  // Disabled globally
  }
}
```

Then enable per-agent via AGENTS.md or agent config.

### Option 3: Dynamic MCP Injection via orch spawn (Complex)

**Pros:**
- True on-demand MCP enablement
- `--mcp web-to-markdown` would actually work

**Cons:**
- Requires creating/modifying project `opencode.json`
- Complexity: What if file exists? Merge logic needed
- May require OpenCode restart after config change

**Implementation:**
1. When `--mcp` flag is set, check if MCP config exists in opencode.json
2. If not, create/update opencode.json with MCP definition
3. Proceed with spawn

### Option 4: Update Research Skill Only (Recommended)

**Pros:**
- No orch-go changes needed
- Agents can use WebFetch tool already available
- web-to-markdown MCP can be manually enabled when needed

**Cons:**
- Doesn't make `--mcp` flag functional

**Implementation:** Document in research skill that:
- Use `WebFetch` tool for external URLs
- For complex JS-heavy sites, web-to-markdown MCP available if pre-configured

## Recommendation

**Option 4: Update Research Skill Only** with documentation about WebFetch tool.

**Reasoning:**
1. Prior decision in kb context: "Document existing capabilities before building new infrastructure - WebFetch investigation showed tool already exists"
2. WebFetch handles most cases (Substack, Reddit, HN articles)
3. The `--mcp` flag can remain as metadata/documentation of intended MCP usage without requiring complex implementation

**Trade-offs accepted:**
- `--mcp` flag remains decorative (tracking only)
- Very complex SPAs may need manual web-to-markdown MCP setup

**What would change this recommendation:**
- If WebFetch consistently fails on common research targets
- If orchestrator needs programmatic MCP enablement for automation

## Conclusion

The web-to-markdown MCP support request reveals a gap between `--mcp` flag intention and implementation. Rather than building complex dynamic MCP injection:

1. **Current state:** `--mcp` flag is metadata-only
2. **Practical solution:** Research skill should use WebFetch tool (already available)
3. **Future option:** If needed, users can pre-configure web-to-markdown in global opencode.json

No code changes required for orch-go. Documentation update recommended for research skill.

## Self-Review

- [x] Evidence sourced with specific locations
- [x] Multiple options compared with pros/cons
- [x] Clear recommendation with reasoning
- [x] Trade-offs acknowledged

**Self-Review Status:** PASSED
