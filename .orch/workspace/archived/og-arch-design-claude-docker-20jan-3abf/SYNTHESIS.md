# Session Synthesis

**Agent:** og-arch-design-claude-docker-20jan-3abf
**Issue:** orch-go-2ge8e
**Duration:** 2026-01-20 10:45 → 2026-01-20 12:00
**Outcome:** success

---

## TLDR

Designed Docker backend integration as a third escape hatch for orch spawn. Recommendation: minimal implementation using host tmux + Docker container pattern (like claude backend), with fresh Statsig fingerprint per spawn and no dashboard visibility (tmux provides visibility for escape hatches).

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2026-01-20-inv-design-claude-docker-backend-integration.md` - Complete design investigation with 7 findings, synthesis, recommendations, and implementation code sketch

### Files Modified
- None (design investigation only)

### Commits
- (pending commit)

---

## Evidence (What Was Observed)

- **Existing Docker workaround works** - `~/.claude/docker-workaround/` with Dockerfile, run.sh has been confirmed working since Dec 2025 for Statsig fingerprint isolation
- **Claude backend pattern is reusable** - `pkg/spawn/claude.go` shows tmux window creation + send keys pattern that Docker can follow
- **Dashboard SSE requires OpenCode server** - `cmd/orch/serve_agents.go` queries OpenCode sessions; Docker runs isolated Claude CLI without OpenCode
- **Backend selection is advisory-only** - `cmd/orch/backend.go:85-99` infrastructure detection warns but never overrides, Docker should follow same pattern
- **Escape hatch visibility comes from tmux, not dashboard** - `.kb/models/escape-hatch-visibility-architecture.md` explicitly documents this architectural decision

### Key Design Decisions Navigated

| Fork | Decision | Rationale |
|------|----------|-----------|
| Session management | Use host tmux + registry with mode: "docker" | Matches claude backend pattern |
| Dashboard integration | None (tmux for visibility) | Escape hatch trades dashboard for independence |
| Credential handling | Mount ~/.claude-docker/ as ~/.claude | Maintains fresh fingerprint with cached OAuth |
| Concurrency model | One container per spawn | Simplicity > optimization for rare usage |
| tmux integration | Host tmux creates window, runs Docker inside | Avoids nested tmux complexity |
| Backend priority | Explicit --backend docker (no auto-select) | Follows advisory-only pattern |

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-01-20-inv-design-claude-docker-backend-integration.md` - Complete design with implementation sketch

### Decisions Made
- **Host tmux over nested tmux**: Run Docker container inside host tmux window rather than tmux inside Docker - simpler, matches existing pattern
- **No dashboard visibility**: Accept this trade-off; escape hatches provide independence at cost of convenience
- **Fresh fingerprint per spawn**: One container per spawn (no persistent pool) - simplicity appropriate for rare usage

### Constraints Discovered
- **Docker backend cannot have SSE events**: Claude CLI has no HTTP interface; dashboard integration would require OpenCode inside container (defeats independence criterion)
- **Rate limit is per-device fingerprint**: Docker provides fresh "device" to Anthropic's rate limit system; this is the core value proposition

### Design Principle Applied
- **Escape Hatch Philosophy**: Critical paths need mechanisms that don't depend on what might fail. Docker doesn't depend on OpenCode server (unlike opencode backend). Trade-off: no dashboard, but survives OpenCode crashes and rate limits.

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation with recommendations)
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for implementation via feature-impl spawn
- [ ] Ready for `orch complete orch-go-2ge8e`

### Implementation Follow-up Suggested

**Issue:** Implement Docker backend for orch spawn
**Skill:** feature-impl
**Context:**
```
Design complete at .kb/investigations/2026-01-20-inv-design-claude-docker-backend-integration.md
Implementation sequence: 1) pkg/spawn/docker.go, 2) backend.go add "docker", 3) wire --backend docker flag
Code sketch included in investigation document.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- MCP server functionality inside container - documented as working but not recently verified
- Git credential passthrough for Docker agents that need to commit
- Performance benchmarking for Docker startup overhead (estimated 2-5s)

**Areas worth exploring further:**
- Auto-detecting rate limit and suggesting `--backend docker` (advisory warning)
- Option to wipe fingerprint (`--fresh-fingerprint`) for true fresh start

**What remains unclear:**
- Whether Anthropic might detect Docker containers and change fingerprinting
- Exact OAuth token persistence behavior across container restarts

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-claude-docker-20jan-3abf/`
**Investigation:** `.kb/investigations/2026-01-20-inv-design-claude-docker-backend-integration.md`
**Beads:** `bd show orch-go-2ge8e`
