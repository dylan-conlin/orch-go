# Session Synthesis

**Agent:** og-inv-shared-browser-experience-26dec
**Issue:** orch-go-46ee
**Duration:** 2025-12-26 07:33 → 2025-12-26 08:10
**Outcome:** success

---

## TLDR

Investigated approaches for shared browser experience where orchestrator and human can both navigate, see state, and interact with the same browser. **Playwright MCP's `--extension` mode is the recommended approach** - human installs a Chrome extension, agent connects to their actual browser session, enabling real-time shared control without complex sync infrastructure.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-26-inv-shared-browser-experience-orch-ecosystem.md` - Comprehensive investigation with 6 findings covering Playwright MCP, browser-use, CDP, and control handoff patterns

### Files Modified
- None (investigation-only task)

### Commits
- To be committed after review

---

## Evidence (What Was Observed)

- Playwright MCP (24.8k GitHub stars) offers `--extension` mode specifically for "connect to existing browser tabs and leverage your logged-in sessions and browser state"
- Accessibility tree approach is superior to screenshots for AI interaction - "LLM-friendly, no vision models needed, operates purely on structured data"
- browser-use (74.2k stars) is Python-based with own agent loop, less suitable for orch Go ecosystem integration
- CDP (Chrome DevTools Protocol) supports multiple simultaneous clients since Chrome 63, enabling both human and agent to connect
- Control handoff is the unsolved problem - existing tools don't provide turn-taking protocols

### Tests Run
```bash
# Documentation and capability review
# Searched for existing browser patterns in orch-go
rg -l "playwright|browser" /Users/dylanconlin/Documents/personal/orch-go --type md
# (No hits - this is new territory for orch ecosystem)
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-26-inv-shared-browser-experience-orch-ecosystem.md` - Full investigation with implementation recommendations

### Decisions Made
- **Playwright MCP extension mode is recommended approach** because it requires minimal new infrastructure, human sees their actual browser (no sync needed), and leverages existing MCP integration

### Constraints Discovered
- Extension mode only works with Chrome/Edge (not Firefox/Safari)
- Same-machine sharing only in Phase 1 (VNC needed for remote)
- "Last action wins" control model initially - explicit lock protocol would be Phase 2

### Externalized via `kn`
- None yet - recommend `kn decide "Playwright MCP extension mode for shared browser" --reason "Simplest integration, human sees their actual browser, no sync infrastructure needed"`

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created with 6 findings + synthesis)
- [x] Tests passing (N/A - documentation investigation)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for `orch complete orch-go-46ee`

### Follow-up Work (if orchestrator decides to proceed)

**Issue:** Implement Playwright MCP extension mode for orch spawn
**Skill:** feature-impl
**Context:**
```
Based on investigation, add `--mcp playwright --extension` equivalent to orch spawn.
Human installs Playwright MCP Bridge extension → agent connects to their browser session.
See .kb/investigations/2025-12-26-inv-shared-browser-experience-orch-ecosystem.md
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How would control indicator UI work in browser extension? (Shows "agent controlling" vs "human controlling")
- Can accessibility tree snapshots be streamed to orch dashboard for real-time visibility without the full browser?
- What happens when agent and human issue commands simultaneously? (Race condition handling)

**Areas worth exploring further:**
- Cross-machine browser sharing (VNC layer or remote CDP)
- Integration with orch daemon for autonomous browser work
- Recording/replay of shared browser sessions for debugging

**What remains unclear:**
- Real-world latency and reliability of extension mode (not hands-on tested)
- Security model of extension bridging agent to browser

---

## Session Metadata

**Skill:** investigation
**Model:** Claude
**Workspace:** `.orch/workspace/og-inv-shared-browser-experience-26dec/`
**Investigation:** `.kb/investigations/2025-12-26-inv-shared-browser-experience-orch-ecosystem.md`
**Beads:** `bd show orch-go-46ee`
