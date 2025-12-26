# Session Synthesis

**Agent:** og-arch-evaluate-whether-web-26dec
**Issue:** orch-go-untracked-1766785873
**Duration:** 2025-12-26 14:30 → 2025-12-26 15:30
**Outcome:** success

---

## TLDR

Evaluated when to use MCP servers vs CLI tools for agent capabilities. **Recommendation: Replace web-to-markdown MCP server with CLI.** MCP should only be used when agents need interactive browser control or stateful connections—not for one-shot transformations that work via Bash.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-inv-evaluate-whether-web-markdown-mcp.md` - Full investigation with MCP vs CLI decision framework

### Files Modified
- None

### Commits
- None yet (investigation file ready for commit)

---

## Evidence (What Was Observed)

- Built-in WebFetch tool already handles static HTML → markdown conversion (webfetch.ts:188 lines)
- CLI `url-to-markdown.sh` is 58 lines vs MCP server `index.js` at 694 lines (12x difference)
- Both use identical underlying tools (shot-scraper, markitdown) for JavaScript-rendered content
- MCP server adds 4 tool definitions (web_to_markdown, web_to_markdown_advanced, youtube_to_markdown, web_metadata)
- MCP requires per-project config in opencode.json; CLI requires only PATH inclusion

### Tests Run
```bash
# Code size comparison
wc -l index.js url-to-markdown.sh
# 694 index.js
# 57 url-to-markdown.sh

# Tool availability check
pip3 show shot-scraper
# Version: 1.8, installed

pip3 show markitdown
# Version: 0.0.1a1, installed
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-evaluate-whether-web-markdown-mcp.md` - MCP vs CLI decision framework

### Decisions Made
- **MCP for interactivity, CLI for transformation:** MCP's value is stateful connections and tool discovery. For one-shot transformations, CLI via Bash is simpler with identical results.
- **Retire web-to-markdown MCP server:** CLI provides same capability with 12x less code and zero config overhead.

### Constraints Discovered
- MCP tool definitions consume agent context tokens even when unused
- MCP requires config per project; CLI requires only PATH

### Externalized via `kn`
- None (pattern documented in investigation is sufficient)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for orchestrator review

### Follow-up Tasks (Optional)
1. **Remove web-to-markdown MCP configs** from any projects that have them
2. **Ensure url-to-markdown.sh is in PATH** for agent access
3. **Document CLI usage** in relevant CLAUDE.md files or skills
4. **Archive MCP server** (don't delete) in case edge cases emerge

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Are there other MCP servers in the ecosystem that could be simplified to CLIs?
- Should there be a formal "MCP necessity test" before creating new MCP servers?

**Areas worth exploring further:**
- Audit all MCP servers in ~/Documents/personal/mcp for CLI alternatives
- Pattern: "MCP readiness checklist" before wrapping tools

**What remains unclear:**
- Whether any agents currently depend on the MCP server (need usage audit)
- Whether the YouTube/metadata tools are used (can add to CLI if needed)

---

## Session Metadata

**Skill:** architect
**Model:** claude (via OpenCode)
**Workspace:** `.orch/workspace/og-arch-evaluate-whether-web-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-evaluate-whether-web-markdown-mcp.md`
**Beads:** Issue appears to have been cleaned up during session

---

## Generalized Pattern

**MCP vs CLI Decision Framework:**

| Use MCP When | Use CLI When |
|--------------|--------------|
| Interactive browser (click, scroll, forms) | One-shot content extraction |
| Multi-step flows requiring state | Simple transformations |
| Tool discovery/schema is the feature | You control the interface |
| Remote/cloud execution needed | Local execution is fine |

**Rule of thumb:** If you can write a working shell one-liner, don't wrap it in MCP.
