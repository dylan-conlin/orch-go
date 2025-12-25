# Session Synthesis

**Agent:** og-inv-addendum-ecosystem-audit-24dec
**Issue:** orch-go-tyvz
**Duration:** 2025-12-24 ~15 min
**Outcome:** success

---

## TLDR

Analyzed OpenCode's role in the orch ecosystem as addendum to the original 8-repo audit. Concluded OpenCode is "runtime infrastructure" (like tmux) not "external tool" (like beads) - the deep integration (3617 LoC, 12+ API endpoints, auth management) is intentional and appropriate.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-24-inv-addendum-ecosystem-audit-opencode.md` - Full investigation with findings

### Files Modified
- None - investigation only, no code changes required

### Commits
- (pending) - Investigation file to be committed

---

## Evidence (What Was Observed)

- `pkg/opencode/` has 3617 lines across 8 files (vs beads: ~20 inline exec.Command calls)
- OpenCode HTTP API endpoints used: `/session`, `/session/:id`, `/session/:id/message`, `/session/:id/prompt_async`, `/event` (SSE)
- orch-go WRITES to OpenCode's `~/.local/share/opencode/auth.json` for account switching
- OpenCode is MIT licensed, maintained by SST (github.com/sst/opencode, 42k stars, 449 contributors)
- OpenAPI 3.1 spec published at `/doc` endpoint (stability signal)
- SSE format has already changed - orch-go handles via dual-format parsing (`pkg/opencode/sse.go:88-108`)

### Tests Run
```bash
# Package size comparison
wc -l pkg/opencode/*.go
# 3617 total lines

# Integration method analysis  
rg "exec.Command.*opencode" --type go
# 5 CLI calls (mostly for tmux attach mode)

rg "http\.(Get|Post|NewRequest)" pkg/opencode/*.go
# 11 HTTP API calls
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-24-inv-addendum-ecosystem-audit-opencode.md` - OpenCode ecosystem characterization

### Decisions Made
- OpenCode should be categorized as "Runtime Infrastructure" not "External Tool"
- No additional abstraction layer needed - `pkg/opencode/` is appropriate
- Current dual-format SSE parsing is the right approach for handling API changes

### Constraints Discovered
- orch-go cannot function without OpenCode (every spawn, session, event flows through it)
- OpenCode auth.json format must be compatible with orch-go's account management

### Externalized via `kn`
- Findings captured in investigation file (no separate kn entries needed)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Tests passing (investigation skill - no code changes)
- [x] Investigation file has `**Status:** Complete`
- [x] Ready for `orch complete orch-go-tyvz`

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Could orch-go detect OpenCode API version and adjust behavior? (version compatibility layer)
- Should there be integration tests that run against live OpenCode server?

**Areas worth exploring further:**
- OpenCode's plugin system for orch-specific extensions
- SST's long-term roadmap for OpenCode API stability

**What remains unclear:**
- SST's formal deprecation policy for API changes
- Whether auth.json format is considered stable API

---

## Session Metadata

**Skill:** investigation
**Model:** anthropic/claude-sonnet-4
**Workspace:** `.orch/workspace/og-inv-addendum-ecosystem-audit-24dec/`
**Investigation:** `.kb/investigations/2025-12-24-inv-addendum-ecosystem-audit-opencode.md`
**Beads:** `bd show orch-go-tyvz`
