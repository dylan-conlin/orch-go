# Session Synthesis

**Agent:** og-feat-synthesize-opencode-investigations-06jan-0739
**Issue:** orch-go-ucswq
**Duration:** 2026-01-06
**Outcome:** success

---

## TLDR

Synthesized 16 OpenCode investigations (spanning 2025-12-19 to 2025-12-26) into a comprehensive authoritative guide at `.kb/guides/opencode.md`. The guide consolidates API patterns, common problems and fixes, architecture decisions, and configuration details into a single reference document.

---

## Delta (What Changed)

### Files Created
- `.kb/guides/opencode.md` - Comprehensive OpenCode integration guide (400+ lines)
- `.kb/investigations/2026-01-06-inv-synthesize-opencode-investigations-16-synthesis.md` - Investigation artifact
- `.orch/workspace/og-feat-synthesize-opencode-investigations-06jan-0739/SYNTHESIS.md` - This synthesis

### Files Modified
- None (synthesis work, not code changes)

### Commits
- Will commit after marking complete

---

## Evidence (What Was Observed)

### Pattern Analysis from 16 Investigations

**Core Integration Patterns:**
- pkg/opencode/ has 3,617 lines across 8 files - substantial investment
- Uses 12+ HTTP API endpoints with structured types
- SSE monitoring via `/event` for real-time completion detection
- OAuth token management writes directly to `~/.local/share/opencode/auth.json`

**Common Problems Documented:**
- `/sessions` (plural) → redirect error (use `/session` singular)
- `/health` endpoint → proxied upstream (not available locally)
- Session accumulation → memory leak in monitor.go (session map cleanup)
- Plugin crashes → missing `@opencode-ai/plugin` dependency
- "No user message" → OpenCode bug (missing await in server.ts)

**Key Architectural Decisions:**
- OpenCode is runtime infrastructure, not external tool (unlike beads)
- Standalone + API Discovery is recommended spawn approach
- Sessions created via TUI ARE visible via API
- `x-opencode-directory` header controls disk vs memory session listing
- OAuth auto-refresh handled by OpenCode's anthropic-auth plugin

### Investigation Timeline

| Date Range | Focus |
|------------|-------|
| 2025-12-19 | POC, client package, session management |
| 2025-12-20 | Integration tradeoffs, spawn modes, context loading |
| 2025-12-21 | Disk session cleanup |
| 2025-12-23 | API redirect issues, plugin crashes |
| 2025-12-24 | Ecosystem audit, cleanup fixes |
| 2025-12-25-26 | Crashes, token refresh, session leaks, themes |

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/guides/opencode.md` - Single authoritative reference replacing 16 scattered investigations

### Key Sections in Guide

1. **Architecture** - Visual diagram showing orch-go ↔ OpenCode ↔ Anthropic flow
2. **Session Lifecycle** - Memory vs disk sessions, x-opencode-directory header
3. **Spawn Modes** - Headless (default), Tmux, Inline with use cases
4. **HTTP API Reference** - Working endpoints vs proxied endpoints
5. **Common Problems** - 5 documented issues with causes and fixes
6. **Key Decisions** - 6 settled decisions that shouldn't be re-investigated
7. **What Lives Where** - File locations for client, SSE, monitor, auth
8. **Configuration** - Instructions array, plugin system, theme system
9. **Debugging Checklist** - 6-step verification before spawning investigation

### Constraints Discovered
- OpenCode's local server proxies unknown routes to desktop.opencode.ai
- Plugin dependency auto-install skipped in local development mode
- Monitor session map requires explicit cleanup after completion

### Externalized via `kn`
- No new kn entries needed - consolidated existing decisions into guide

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete
  - [x] Guide created at `.kb/guides/opencode.md`
  - [x] Investigation artifact created
  - [x] SYNTHESIS.md created
- [x] No tests needed (documentation synthesis)
- [x] Investigation file has correct status
- [x] Ready for `orch complete orch-go-ucswq`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should we create similar synthesis guides for other high-investigation topics (daemon - 31 investigations, beads - 17)?
- Could we automate guide generation from kb chronicle output?

**What remains unclear:**
- Long-term stability of OpenCode API (SST maintains it with 449 contributors)
- Whether refresh token has limited lifetime (not documented)

---

## Session Metadata

**Skill:** feature-impl
**Model:** opus (default)
**Workspace:** `.orch/workspace/og-feat-synthesize-opencode-investigations-06jan-0739/`
**Investigation:** `.kb/investigations/2026-01-06-inv-synthesize-opencode-investigations-16-synthesis.md`
**Beads:** `bd show orch-go-ucswq`
