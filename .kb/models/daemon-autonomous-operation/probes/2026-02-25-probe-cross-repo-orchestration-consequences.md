# Probe: Cross-Repo Orchestration Consequences for orch-go

**Model:** daemon-autonomous-operation
**Date:** 2026-02-25
**Status:** Complete

---

## Question

The cross-repo orchestration decision (`scs-special-projects/.kb/decisions/2026-02-25-cross-repo-orchestration-from-parent.md`) says: strategic issues live in scs-special-projects parent repo beads, implementation issues stay in child repos, orchestrator runs from parent and spawns into child repos with `--workdir`. After fixes orch-go-1230/1231, what in orch-go already supports this pattern, what's broken/missing, and what gaps need filling?

---

## What I Tested

### 1. scs-special-projects Registry Status
```bash
kb projects list --json
# Result: 18 projects listed
# scs-special-projects is NOT registered
# Child repos ARE registered: price-watch, toolshed, specs-platform, sendassist
```

### 2. scs-special-projects Beads Config
```bash
cat ~/Documents/work/SendCutSend/scs-special-projects/.beads/config.yaml
# Result: issue-prefix is commented out (no explicit prefix)
# Default would be "scs-special-projects" (from directory name)
```

### 3. Daemon ProjectRegistry Source
```go
// pkg/daemon/project_resolution.go:37-67
// NewProjectRegistry builds from `kb projects list --json`
// Falls back to directory basename if .beads/config.yaml missing
```

### 4. ListReadyIssuesMultiProject Coverage
```go
// pkg/daemon/issue_adapter.go:408-446
// Iterates over registry.Projects(), queries each project's beads
// Only queries projects IN the registry
```

### 5. Completion Processing Scope
```go
// pkg/daemon/completion_processing.go:85-137
// ListCompletedAgentsDefault() calls verify.ListOpenIssues()
// This uses the CURRENT project's beads database only
// No cross-project completion scanning
```

### 6. Dashboard Beads API
```go
// cmd/orch/serve_beads.go:464-501
// handleBeads() accepts project_dir query param
// Can query any project's beads via CLI client
// Dashboard CAN show scs-special-projects if given the path
```

### 7. Work-Graph Cross-Project Support
```go
// cmd/orch/serve_beads.go:920-1036
// handleBeadsGraph() accepts project_dir query param
// Builds nodes/edges from a SINGLE project at a time
// No cross-project dependency edge rendering
```

### 8. Agent Discovery Cross-Project
```go
// cmd/orch/serve_agents_cache.go:396, 439
// listSessionsAcrossProjects() queries all kb projects
// getKBProjects() parses kb projects list
// scs-special-projects NOT in kb projects → invisible
```

---

## What I Observed

### WHAT WORKS TODAY (After orch-go-1230/1231)

1. **`orch spawn --workdir` works end-to-end** (`spawn_cmd.go:178, 497-503`)
   - Resolves project directory, loads skill, generates workspace
   - All beads calls respect `--workdir` after fix 1230

2. **`orch work --workdir` works end-to-end** (`spawn_cmd.go:394-407`)
   - Sets `beads.DefaultDir` from `--workdir` before any beads calls (fix 1230)
   - Skill inference, MCP inference, and issue lookup all use target project

3. **Daemon multi-project polling** (`daemon.go:256-264, issue_adapter.go:408-446`)
   - `ProjectRegistry` maps issue prefixes to project dirs
   - `ListReadyIssuesMultiProject()` queries all registered projects
   - Issues carry `ProjectDir` field for correct `--workdir` passing

4. **Daemon cross-project spawn** (`issue_adapter.go:352-367`)
   - `SpawnWork()` passes `--workdir` when `ProjectDir` is non-empty
   - Status updates use `UpdateBeadsStatusForProject()` with correct dir

5. **Dashboard beads API accepts project_dir** (`serve_beads.go:464-501, 546-593`)
   - `/api/beads?project_dir=X` queries target project stats
   - `/api/beads/ready?project_dir=X` shows target project ready queue
   - Cache is project-aware (per-project TTL entries)

6. **Dashboard graph API accepts project_dir** (`serve_beads.go:920-1036`)
   - `/api/beads/graph?project_dir=X` renders target project graph

7. **Agent discovery across projects** (`serve_agents_cache.go:396-430`)
   - `listSessionsAcrossProjects()` queries OpenCode across kb projects
   - Tracked agents include beads ID → graph enrichment works

### WHAT'S BROKEN/MISSING

1. **scs-special-projects is NOT in `kb projects list`**
   - The daemon's ProjectRegistry builds from `kb projects list`
   - scs-special-projects is not registered → daemon never polls it
   - **Fix:** `cd ~/Documents/work/SendCutSend/scs-special-projects && kb init`
   - This is an operational gap, not a code gap

2. **scs-special-projects has no explicit beads issue-prefix**
   - `config.yaml` has `issue-prefix` commented out
   - ProjectRegistry falls back to directory name: `scs-special-projects`
   - Prefix is very long → issue IDs like `scs-special-projects-123`
   - **Fix:** Set `issue-prefix: scs-sp` in `.beads/config.yaml`

3. **Completion processing is single-project only**
   - `ListCompletedAgentsDefault()` calls `verify.ListOpenIssues()` which uses current project beads
   - If daemon runs from orch-go, it can't detect completed agents working on scs-sp issues
   - The daemon can SPAWN cross-project but can't COMPLETE cross-project
   - **Impact:** Agents spawned into child repos from parent strategic issues won't be auto-completed

4. **Work-graph has no cross-project edge rendering**
   - Each call to `/api/beads/graph?project_dir=X` returns ONE project's graph
   - A parent issue `scs-sp-10` blocking `toolshed-200` can't be rendered as an edge
   - The graph shows each project in isolation
   - **Impact:** The "full picture" of parent strategic + child implementation is invisible

5. **Dashboard has no "parent repo" context concept**
   - Dashboard shows stats/agents for whatever project it's configured for
   - No mechanism to show "scs-special-projects as coordination hub" with child repos underneath
   - No aggregate view of strategic issues + their child implementation issues

6. **Cross-repo dependency tracking is text-only**
   - Decision says: "Parent issues reference child issues by repo prefix: `toolshed-74`, `pw-9059`"
   - `bd dep add` only works within a single repo's beads database
   - Dependencies across repos are conventions in description text, not enforced edges
   - **Impact:** Daemon can't honor cross-repo dependency ordering

7. **`orch complete` is single-project scoped**
   - The `orch complete` command runs verification against the current project's beads
   - Completing a strategic parent issue (`scs-sp-10`) that has spawned work in child repos requires:
     - Checking each child repo's implementation issues are closed
     - Aggregating verification across repos
   - None of this exists

8. **Daemon where-to-run ambiguity**
   - Decision says "orchestrator runs from parent"
   - But daemon currently runs from orch-go (where it's built)
   - Running daemon from scs-special-projects would make parent polling automatic
   - But then orch-go issues wouldn't be polled (unless also in registry)
   - **The daemon needs to run from SOMEWHERE and poll EVERYWHERE**

### NEW QUESTIONS

1. **Where does the daemon physically run?**
   - If from orch-go: parent issues invisible unless scs-sp registered in kb
   - If from scs-sp: orch-go issues invisible unless registered in kb
   - Answer: daemon should run from wherever, but poll ALL kb-registered projects
   - This already works IF scs-sp is registered (`kb init`)

2. **How does a strategic parent issue get completed?**
   - Parent: "AI Pricing Panel" (scs-sp-10)
   - Child: "Build API endpoints" (pw-200), "Build UI" (toolshed-300)
   - Current: No mechanism to know when all children are done
   - Need: Something that watches child repo closures and signals parent readiness

3. **What context does cc personal from scs-special-projects get?**
   - Claude Code loads CLAUDE.md from current directory
   - scs-special-projects CLAUDE.md describes the portfolio structure
   - Orchestrator skill loaded via `~/.claude/skills/meta/orchestrator/`
   - `orch spawn --workdir ~/Documents/work/SendCutSend/toolshed investigation "X"` would work
   - The orchestrator has correct strategic context from parent CLAUDE.md

4. **How does the work-graph render cross-repo relationships?**
   - Currently: it doesn't — each project is a separate graph
   - Need: Either a merged graph endpoint that queries multiple projects
   - Or: a "portfolio graph" that shows strategic nodes + child stubs

---

## Model Impact

- [x] **Extends** model with: New operational gap — "Cross-repo parent polling requires kb registration"
- [x] **Extends** model with: New architectural gap — "Completion processing is single-project, can't detect cross-project agent completion"

### Extended Model Claims

**New Invariant:** The daemon's cross-project support is asymmetric: it can POLL and SPAWN cross-project, but it cannot COMPLETE cross-project. Completion processing (`ListCompletedAgentsDefault`) only checks the current project's beads, while polling (`ListReadyIssuesMultiProject`) checks all registered projects.

**New Failure Mode: Parent Issue Orphaning**
- Daemon spawns work from scs-sp strategic issue into toolshed
- Agent in toolshed reports Phase: Complete
- Daemon's completion loop (running from orch-go) checks orch-go beads only
- Toolshed agent completion never detected by daemon
- Strategic parent issue in scs-sp remains in_progress indefinitely
- **Severity:** This blocks the core workflow the decision describes

**Operational Prerequisites (must be done to make decision work):**
1. `cd ~/Documents/work/SendCutSend/scs-special-projects && kb init` — register parent repo
2. Set `issue-prefix: scs-sp` in `scs-special-projects/.beads/config.yaml`
3. Daemon must run with ProjectRegistry that includes scs-sp

---

## Gap Prioritization (Blocks Workflow vs Nice-to-Have)

### P0 — Blocks the workflow entirely

| Gap | Why it blocks | Fix complexity |
|-----|---------------|----------------|
| scs-sp not in kb projects | Daemon never polls parent issues | 1 command: `kb init` |
| No explicit issue-prefix | Long IDs, ProjectRegistry confusion | 1 line config change |

### P1 — Workflow works manually but auto-processing breaks

| Gap | Why it matters | Fix complexity |
|-----|----------------|----------------|
| Completion processing single-project | Daemon can't auto-complete cross-project agents | Medium: extend `ListCompletedAgentsDefault` to query all registry projects |
| `orch complete` single-project | Manual completion of parent issues requires per-repo work | Medium: add `--project-dir` to complete cmd or auto-detect from beads ID |

### P2 — Workflow works but visibility is poor

| Gap | Why it matters | Fix complexity |
|-----|----------------|----------------|
| No cross-project graph edges | Can't see parent→child dependency edges | Hard: need cross-project edge discovery |
| No portfolio dashboard view | Can't see strategic + implementation together | Medium: new "portfolio" API endpoint |
| No cross-repo dependency enforcement | Daemon can't honor parent→child ordering | Hard: need cross-repo dep resolution |

### P3 — Nice-to-have for full maturity

| Gap | Why it matters | Fix complexity |
|-----|----------------|----------------|
| Parent issue auto-completion | Close parent when all children done | Hard: cross-repo dependency traversal |
| Cross-project content dedup | Prevent duplicate spawns across repos | Medium: extend FindInProgressByTitle |
| Aggregate verification | Verify parent by aggregating child results | Hard: new verification mode |
