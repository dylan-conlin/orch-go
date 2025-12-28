# Session Synthesis

**Agent:** og-arch-design-daemon-managed-27dec
**Issue:** orch-go-h674
**Duration:** 2025-12-27 16:03 → 2025-12-27 16:50
**Outcome:** success

---

## TLDR

Designed daemon-managed development servers infrastructure. Recommended on-demand health checks with servers.yaml declarations and SessionStart integration over persistent daemon approaches - simpler, more transparent, leverages existing tmuxinator infrastructure.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-27-inv-design-daemon-managed-development-servers.md` - Full design investigation with 3 approaches analyzed and recommendation

### Files Modified
- `.orch/features.json` - Added 5 new feature items (feat-024 through feat-028) for phased implementation

### Commits
- N/A (investigation artifacts only - commit pending)

---

## Evidence (What Was Observed)

- Existing infrastructure is extensive: port registry (`~/.orch/ports.yaml`), tmuxinator configs (`~/.tmuxinator/workers-*.yml`), `orch servers` CLI, `/api/servers` endpoint (serve.go:223)
- orch daemon already writes status files to `~/.orch/daemon-status.json` with atomic writes - same pattern can apply to server health
- tmuxinator configs define server commands but lack health checks: price-watch has Docker (`make up`), orch-go has vite (`bun run dev`)
- Session Amnesia principle from `~/.kb/principles.md` requires externalized state for servers
- Glass (orch-go-tiav) validates persistent daemon pattern for complex resources, but dev servers are simpler

### Tests Run
```bash
# Examined existing infrastructure
bd show orch-go-h674  # Got issue details
bd show orch-go-tiav  # Got related Glass issue
ls ~/.tmuxinator/workers-*.yml  # Found 34 project configs
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-27-inv-design-daemon-managed-development-servers.md` - Complete design with servers.yaml schema, 3 approaches analyzed, recommendation with phased implementation

### Decisions Made
- **On-demand health checks over persistent daemon:** Servers are dependencies to verify, not resources to manage. Health checks at session boundaries are sufficient.
- **servers.yaml per project:** Following Local-First principle - declarations live with project, not in central registry
- **Extend existing CLI over new daemon:** Compose Over Monolith - add `check` and `up` subcommands to existing `orch servers`
- **SessionStart integration for gating:** Natural extension of environment verification already done by hook

### Constraints Discovered
- Docker Compose health checks can be slow (5-10s) - need generous timeouts
- tmuxinator start is not idempotent if session exists - must check first
- Port registry and servers.yaml must stay consistent

### Externalized via `kn`
- N/A (findings captured in investigation artifact)

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation + feature list update)
- [x] Tests passing (N/A for design work)
- [x] Investigation file has `**Phase:** Complete`
- [ ] Ready for `orch complete orch-go-h674`

### Implementation Phases (for orchestrator reference)

**Phase 1 - Foundation:**
- feat-024: servers.yaml schema and config parser
- feat-025: Health check command (`orch servers check`)
- feat-026: Dashboard visibility (`/api/servers` enhancement)

**Phase 2 - Gating:**
- feat-027: SessionStart hook integration

**Phase 3 - Automation:**
- feat-028: Auto-start capability (`orch servers up`)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- How to handle servers that need environment variables (e.g., DATABASE_URL for Rails)
- Whether to support health check dependencies (frontend depends on backend being up first)
- Integration specifics for price-watch Docker workflow (multi-container with fixed ports)

**Areas worth exploring further:**
- Continuous monitoring if health check only at session boundaries proves insufficient
- Integration with `orch doctor` for infrastructure-level server health

**What remains unclear:**
- Exact error messaging when SessionStart health check fails
- Whether to auto-start servers or just warn (probably configurable per-server via `critical` flag)

---

## Session Metadata

**Skill:** architect
**Model:** opus
**Workspace:** `.orch/workspace/og-arch-design-daemon-managed-27dec/`
**Investigation:** `.kb/investigations/2025-12-27-inv-design-daemon-managed-development-servers.md`
**Beads:** `bd show orch-go-h674`
