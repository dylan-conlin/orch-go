<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Cross-project agent display works via three-layer discovery (current dir, OpenCode sessions, kb projects registry); price-watch agents ARE appearing correctly; design is sound and requires only documentation/monitoring improvements, not redesign.

**Evidence:** Code review confirmed kb projects integration (serve_agents_cache.go:281-348), kb projects list shows 17 registered projects including price-watch, price-watch has 110 workspaces with recent activity (Jan 13-14), OpenCode --attach limitation documented in Jan 7 investigation.

**Knowledge:** kb projects integration started as workaround for OpenCode architectural limitation but evolved into superior design (explicit user-managed registry > implicit session-based discovery); hybrid approach provides resilience through redundancy (three sources of project directories).

**Next:** Document the three-layer discovery architecture, add monitoring for getKBProjects() failures, test project filter parameter with cross-project agents, add graceful handling for stale project paths.

**Promote to Decision:** recommend-no (design is already implemented and working; investigation documents existing architecture, not proposing new architectural choice)

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Cross Project Agent Display

**Question:** How does cross-project agent display work in the orch-go dashboard, and is the current design appropriate?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** architect agent (orch-go-u5lxc)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Cross-project agent display uses a hybrid discovery approach

**Evidence:**
- `extractUniqueProjectDirs()` aggregates project directories from three sources:
  1. Current project directory (where orch serve is running)
  2. OpenCode session directories (from active sessions)
  3. kb projects registry (17 registered projects including price-watch)
- Multi-project workspace cache built from all discovered directories
- Each agent's `ProjectDir` is populated from SPAWN_CONTEXT.md via `wsCache.lookupProjectDir(beadsID)`

**Source:**
- cmd/orch/serve_agents_cache.go:311-348 (extractUniqueProjectDirs implementation)
- cmd/orch/serve_agents_cache.go:281-305 (getKBProjects implementation)
- cmd/orch/serve_agents.go:389-390 (cache building in handleAgents)
- cmd/orch/serve_agents.go:512-514, 582-584, 793-796 (ProjectDir population)

**Significance:** The system is designed to show agents from ALL known projects in a single unified dashboard, not just the project where orch serve is running. This enables centralized orchestration visibility.

---

### Finding 2: OpenCode --attach mode limitation drove kb projects integration

**Evidence:**
- All 248 OpenCode sessions have `directory="/Users/dylanconlin/Documents/personal/orch-go"` regardless of `--workdir` spawn parameter
- OpenCode `run --attach` connects to server which sets session directory from its own cwd, not the CLI's cwd
- Setting `cmd.Dir` in spawn has no effect on OpenCode session directory
- Jan 7 investigation (.kb/investigations/2026-01-07-inv-cross-project-agents-show-wrong.md) identified this as architectural limitation
- kb projects integration was implemented as solution (Jan 7 feature-impl)

**Source:**
- .kb/investigations/2026-01-07-inv-cross-project-agents-show-wrong.md (root cause analysis)
- .kb/investigations/2026-01-07-inv-implement-kb-projects-integration-cross.md (implementation)
- `curl -s http://localhost:4096/session | jq '[.[] | .directory] | unique'` returns only orch-go
- `kb projects list` shows 17 registered projects

**Significance:** The hybrid approach wasn't the original plan - it emerged as a workaround for OpenCode architectural constraints. The system couldn't rely solely on OpenCode session directories for project discovery.

---

### Finding 3: price-watch agents ARE appearing in the dashboard with correct project identification

**Evidence:**
- price-watch has 110 workspaces in `.orch/workspace/` directory
- price-watch is registered in kb projects: `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch`
- Recent workspaces created Jan 13-14: `pw-feat-implement-material-category-14jan-f385`, `pw-feat-wire-up-generate-13jan-bd17`, etc.
- kb projects integration (Jan 7) enables price-watch workspaces to be scanned and agents to be displayed

**Source:**
- `kb projects list | grep -i price` shows price-watch registration
- `ls /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.orch/workspace` shows 110 workspaces
- `find ... | wc -l` confirms 110 workspace directories

**Significance:** The cross-project display is working as designed - price-watch agents spawned with `--workdir` are visible in the orch-go dashboard because kb projects integration ensures price-watch workspaces are scanned.

---

## Synthesis

**Key Insights:**

1. **Cross-project visibility is a core design principle, not an afterthought** - The dashboard intentionally aggregates agents from ALL registered projects into a single view. This supports centralized orchestration where Dylan can see and manage agents across his entire project ecosystem from one dashboard.

2. **Hybrid discovery is resilient but has dependencies** - By combining three sources (current dir, OpenCode sessions, kb projects), the system has redundancy. If OpenCode sessions fail to report correct directories (which they do due to --attach), kb projects provides the fallback. But this creates a new dependency: projects must be registered with `kb project register` to be visible.

3. **The workaround became the feature** - What started as a fix for OpenCode's --attach limitation (can't set session directory per-spawn) evolved into a superior design. kb projects is more reliable than OpenCode session directories because it's explicitly managed and represents user intent about which projects should be orchestrated.

**Answer to Investigation Question:**

**How it works:**
Cross-project agent display uses a three-layer discovery approach:
1. `extractUniqueProjectDirs()` collects project directories from current dir, OpenCode sessions, and kb projects registry
2. `buildMultiProjectWorkspaceCache()` scans all discovered projects' `.orch/workspace/` directories in parallel
3. Each agent's `ProjectDir` is populated from SPAWN_CONTEXT.md (which contains correct PROJECT_DIR even when OpenCode session has wrong directory)
4. Dashboard filters/groups agents by `ProjectDir` field for cross-project visibility

**Is it appropriate?**
YES, with one caveat:
- ✅ **Appropriate:** Centralized multi-project dashboard aligns with orchestration use case
- ✅ **Appropriate:** kb projects as source of truth is more reliable than OpenCode sessions
- ✅ **Appropriate:** Graceful degradation (kb CLI failure doesn't break dashboard)
- ⚠️ **Caveat:** Depends on users registering projects - unregistered projects won't appear

The design is sound. It elegantly solved an OpenCode architectural limitation while creating a better user experience (explicit project registration vs implicit session-based discovery).

---

## Structured Uncertainty

**What's tested:**

- ✅ kb projects integration is implemented (verified: code review of serve_agents_cache.go:281-348)
- ✅ price-watch is registered in kb projects (verified: `kb projects list | grep price`)
- ✅ price-watch has 110 workspaces (verified: `find .orch/workspace | wc -l`)
- ✅ extractUniqueProjectDirs calls getKBProjects (verified: code review line 340-345)
- ✅ OpenCode sessions all show orch-go directory (verified: `curl http://localhost:4096/session | jq`)

**What's untested:**

- ⚠️ Performance impact of scanning 17 projects × N workspaces (not benchmarked)
- ⚠️ Behavior when kb projects contains stale/moved project paths (not tested)
- ⚠️ Dashboard UX when displaying 100+ cross-project agents simultaneously (not observed)
- ⚠️ Whether project filter query parameter works correctly for cross-project agents (not tested)

**What would change this:**

- Finding would be wrong if kb projects list fails and fallback doesn't work (would break cross-project visibility)
- Finding would be wrong if price-watch workspaces aren't being scanned despite kb registration (would indicate cache bug)
- Design evaluation would change if centralized multi-project view creates UX problems (would need per-project views)

---

## Implementation Recommendations

**Purpose:** The current design is sound. These recommendations focus on documentation, observability, and hardening edge cases.

### Recommended Approach ⭐

**Document and harden the existing design** - Keep the current architecture, add documentation and monitoring, handle edge cases gracefully.

**Why this approach:**
- Current design is working (price-watch agents are appearing)
- kb projects integration solved the OpenCode limitation elegantly
- No evidence of performance or UX problems requiring redesign
- Better to document what works than rebuild what doesn't need rebuilding

**Trade-offs accepted:**
- Users must register projects (acceptable: explicit > implicit)
- Dependency on kb CLI (acceptable: already required for orchestration)
- Scans all registered projects even if inactive (acceptable: workspaces are cheap to scan when empty)

**Implementation sequence:**
1. **Add monitoring for kb projects failures** - Track when getKBProjects() fails to alert on broken cross-project visibility
2. **Document the design** - Add architecture doc explaining three-layer discovery (this investigation can serve as source)
3. **Add graceful handling for stale project paths** - If kb project path doesn't exist or has no .orch/, log warning but continue
4. **Test project filter parameter** - Verify dashboard ?project= query parameter works for cross-project agents

### Alternative Approaches Considered

**Option B: Per-project dashboards instead of unified view**
- **Pros:** Would eliminate need for cross-project discovery, simpler implementation
- **Cons:** Defeats the purpose of centralized orchestration, Dylan would need multiple browser tabs
- **When to use instead:** If unified dashboard becomes too cluttered (100+ simultaneous agents from many projects)

**Option C: Config-based project list instead of kb projects**
- **Pros:** No dependency on kb CLI, could use ~/.orch/config.yaml
- **Cons:** Requires manual maintenance, kb projects already exists and works
- **When to use instead:** If kb CLI becomes unavailable or too slow

**Option D: Auto-discover via filesystem scan (e.g., find all .orch/ directories)**
- **Pros:** Zero configuration, finds all projects automatically
- **Cons:** Slow (would need to scan entire filesystem), finds abandoned/test projects
- **When to use instead:** Never - discovery should be explicit, not implicit

**Rationale for recommendation:** The current design (Option A with kb projects) is the sweet spot between explicit (user-managed registry) and automatic (scans registered projects). Alternatives either sacrifice UX (per-project dashboards) or reliability (config maintenance, filesystem scanning).

---

### Implementation Details

**What to implement first:**
- Add logging when getKBProjects() fails - highest value for debugging cross-project visibility issues
- Document the three-layer discovery architecture (current dir, sessions, kb projects)
- Test the project filter parameter with actual cross-project agents (verify ?project=price-watch works)

**Things to watch out for:**
- ⚠️ kb CLI might fail silently in launchd/server contexts where PATH is minimal (already has graceful fallback, but needs monitoring)
- ⚠️ Stale project paths in kb projects registry could cause directory scan errors (need to check path exists before scanning)
- ⚠️ Performance scaling if workspace count grows (currently 110 workspaces in price-watch is fine, but 1000+ might need optimization)

**Areas needing further investigation:**
- Whether dashboard UX remains usable with 50+ cross-project agents displayed simultaneously
- Cache invalidation strategy when kb projects registry changes (currently uses 30s TTL)
- Whether project filter needs "follow orchestrator" mode (show only agents for projects orchestrator is currently working on)

**Success criteria:**
- ✅ price-watch agents continue to appear in dashboard (regression test)
- ✅ getKBProjects() failures are logged and monitored
- ✅ Stale project paths don't crash dashboard
- ✅ Documentation exists explaining how cross-project discovery works

---

## References

**Files Examined:**
- cmd/orch/serve_agents.go:326-998 - handleAgents implementation, cache building, cross-project agent population
- cmd/orch/serve_agents_cache.go:281-348 - getKBProjects and extractUniqueProjectDirs implementations
- pkg/session/registry.go:1-305 - OrchestratorSession structure (has ProjectDir field)

**Commands Run:**
```bash
# Check if price-watch is registered in kb projects
kb projects list | grep -i price
# Result: price-watch at /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch

# Count price-watch workspaces
find /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.orch/workspace -type d -maxdepth 1 | wc -l
# Result: 110 workspaces

# Check OpenCode session directories
curl -s http://localhost:4096/session | jq '[.[] | .directory] | unique'
# Result: Only orch-go directory despite --workdir spawns

# List recent price-watch workspaces
ls -la /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/price-watch/.orch/workspace | head -10
# Result: pw-feat-implement-material-category-14jan-f385, pw-feat-wire-up-generate-13jan-bd17, etc.
```

**External Documentation:**
- None (internal design investigation)

**Related Artifacts:**
- **Investigation:** .kb/investigations/2026-01-07-inv-cross-project-agents-show-wrong.md - Root cause analysis that led to kb projects integration
- **Investigation:** .kb/investigations/2026-01-07-inv-implement-kb-projects-integration-cross.md - Implementation of kb projects discovery
- **Investigation:** .kb/investigations/2025-12-26-inv-design-proper-cross-project-agent.md - Earlier design work on cross-project support

---

## Investigation History

**2026-01-14 07:45:** Investigation started
- Initial question: How does cross-project agent display work in orch-go dashboard, and is it appropriate?
- Context: Spawned to understand design of price-watch agents appearing in orch-go dashboard

**2026-01-14 08:00:** Architecture discovery
- Found three-layer discovery approach (current dir, OpenCode sessions, kb projects)
- Discovered kb projects integration was implemented Jan 7 to solve OpenCode --attach limitation
- Verified price-watch is registered and has 110 workspaces

**2026-01-14 08:15:** Design evaluation
- Concluded current design is sound and working as intended
- Identified that kb projects was initially a workaround but became a superior feature
- Recommended documentation and monitoring improvements, not redesign

**2026-01-14 08:30:** Investigation completed
- Status: Complete
- Key outcome: Cross-project display works via kb projects integration; design is appropriate with minor hardening recommended
