# Session Synthesis

**Agent:** og-inv-explore-options-centralized-23dec
**Issue:** orch-go-NEW
**Duration:** 2025-12-23 (1 session)
**Outcome:** success

---

## TLDR

Investigated options for centralized server management across 20+ polyrepo projects. Recommended tmux-centric CLI commands (`orch servers`) that wrap existing port registry and tmuxinator infrastructure, delivering immediate value with ~200 lines of Go.

---

## Delta (What Changed)

### Files Created
- `.kb/investigations/2025-12-23-inv-explore-options-centralized-server-management.md` - Comprehensive investigation with findings, options matrix, and implementation recommendations

### Files Modified
None (investigation only)

### Commits
None yet (will commit investigation file)

---

## Evidence (What Was Observed)

- Port registry contains 24 projects (21 real, 3 test projects) with allocated web/api ports
- 24 tmuxinator worker configs exist at ~/.tmuxinator/workers-{project}.yml
- Currently 3 workers sessions running: orch-go (8 windows), price-watch (4 windows), another-test-project (3 windows)
- Researched industry tools: Foreman (single-project Procfile), Overmind (single-project + tmux), Nx (monorepo orchestration)
- None fit polyrepo use case - all target different architectures
- Existing infrastructure (pkg/port, pkg/tmux, tmuxinator configs) provides solid foundation
- Gap is discoverability/convenience, not process management

### Tests Run
```bash
# Listed all allocated ports
orch port list
# Result: 24 projects with web (5173-5199) and api (3333-3399) ports

# Counted tmuxinator configs
ls ~/.tmuxinator/workers-*.yml | wc -l
# Result: 24 configs

# Listed running workers sessions
tmux list-sessions | grep workers
# Result: 3 active sessions (orch-go, price-watch, another-test-project)

# Analyzed workers session structure
tmux list-sessions -F "#{session_name}" | grep workers | while read sess; do tmux list-windows -t "$sess" -F "#{window_name}"; done
# Result: Each session has servers window + multiple agent windows
```

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2025-12-23-inv-explore-options-centralized-server-management.md` - Full investigation with options analysis

### Decisions Made
- Decision 1: Tmux-centric approach over process manager because tmux already handles process management
- Decision 2: CLI-first over web UI because terminal workflow is primary and delivers immediate value
- Decision 3: Reuse existing infrastructure (port registry + tmuxinator) rather than new abstraction layer

### Constraints Discovered
- Constraint 1: Must handle missing tmuxinator configs gracefully (not all port allocations have configs)
- Constraint 2: Can't detect actual port usage (only allocation) - port registry shows allocation, not binding
- Constraint 3: Must avoid killing agent windows when stopping servers (workers sessions have mixed content)

### Externalized via `kn`
Not yet - will externalize key insights after implementation validates assumptions

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (investigation file created)
- [x] Investigation has clear recommendation with rationale
- [x] Investigation file has `**Phase:** Complete`
- [x] Ready for implementation in separate feature task

### Implementation Roadmap
**Next Task:** Create beads issue for implementation
**Skill:** feature-impl
**Estimated Scope:** ~200 lines of Go, 4-6 hours
**Sequence:**
1. Add `orch servers list` (foundation command)
2. Add `orch servers start <project>` 
3. Add `orch servers stop <project>`
4. Add `orch servers attach <project>`
5. Add `orch servers status` (summary view)
6. Add `orch servers open <project>` (browser launch)

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should servers auto-start when spawning agents for a project? (convenience vs explicitness tradeoff)
- Should there be health checks for server panes? (detect crashed servers vs just allocation)
- What's the right behavior for startup dependencies (redis, postgres)? (orchestration vs manual management)
- Should web UI be added for dashboard view? (visual monitoring vs CLI simplicity)

**Areas worth exploring further:**
- Auto-discovery: Show only allocated projects or also tmuxinator configs without allocations?
- Browser open behavior: All ports (web + api) or just web port?
- Filtering options: --running, --stopped, --by-stack (rails vs svelte vs go)
- Integration with orch spawn: Should spawn also start servers window?

**What remains unclear:**
- Usage patterns after implementation - will CLI be sufficient or will web UI become needed?
- Scale testing - how does this perform with 50+ projects?
- Cross-team usage - will this be adopted by other developers or just personal workflow?

---

## Session Metadata

**Skill:** investigation
**Model:** claude-sonnet-4-20250514
**Workspace:** `.orch/workspace/og-inv-explore-options-centralized-23dec/`
**Investigation:** `.kb/investigations/2025-12-23-inv-explore-options-centralized-server-management.md`
**Beads:** Issue not found (orch-go-NEW doesn't exist)
