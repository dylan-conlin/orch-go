# Design: Project Group Model for orch-go

**Date:** 2026-02-25
**Phase:** Complete
**Status:** Complete
**Beads:** orch-go-1235
**Type:** Architect Design

## Design Question

How should orch-go model project groups to solve three converging needs: sibling kb context across work projects, parent-child project relationships, and daemon polling/account scoping?

## Problem Framing

### Success Criteria

1. Spawning into toolshed auto-surfaces price-watch kb artifacts (and vice versa for all SCS siblings)
2. scs-special-projects is recognized as the coordination parent for work projects
3. Daemon can scope polling to a project group and route accounts per group
4. Dashboard can filter by group
5. No breaking changes to existing behavior — ungrouped projects work as before

### Constraints

- **Session Amnesia:** Config must be discoverable at standard locations
- **No Local Agent State:** Groups are project infrastructure config, NOT agent state. Like accounts.yaml, not like a session registry.
- **Inline Lineage Metadata:** Doesn't apply — groups are about project relationships, not individual artifact provenance
- **Existing multi-project polling works:** Don't break `ListReadyIssuesMultiProject`

### Scope

**In scope:** Group model, config format, cascade to kb/daemon/dashboard/accounts
**Out of scope:** Cross-repo beads dependency edges, portfolio dashboard redesign, auto-completion of parent issues

## Exploration (Fork Navigation)

### Fork 1: Where does group config live?

**Options:**
- A: Global `~/.orch/groups.yaml` (single file, all groups)
- B: Per-project `.orch/config.yaml` with `group:` field
- C: Inferred from directory structure only (no config)
- D: Hybrid (path inference + explicit config)

**Substrate says:**
- Principle (Session Amnesia): Config must be discoverable. Single file at a standard location is more discoverable than scattered per-project configs.
- Decision (single-daemon-orchestration-home): Daemon runs from orch-go. Global config at `~/.orch/` aligns with daemon's home.
- Evidence: SCS parent-child is perfectly inferable from paths. Orch ecosystem is NOT inferable (orch-knowledge at ~/orch-knowledge breaks pattern; blog/snap in ~/Documents/personal/ are not orch ecosystem).

**Recommendation:** **Option D (Hybrid)** — Path inference handles SCS automatically; explicit config handles orch ecosystem. Global `~/.orch/groups.yaml` for explicit definitions + auto-inference for parent-project patterns.

**Trade-off accepted:** Two group resolution mechanisms (inference + explicit) add implementation complexity, but eliminate config maintenance for the most common pattern (parent-child directories).

### Fork 2: How are groups defined?

**Options:**
- A: Named group with explicit member list
- B: Parent-project groups (parent project + auto-discovered children)
- C: Both

**Substrate says:**
- Evidence from `kb projects list`: 19 projects, 6 are SCS (5 children + parent), 7 are orch ecosystem, 6 are ungrouped personal
- SCS parent-child is structural (directory nesting)
- Orch ecosystem is semantic (shared purpose, not shared directory)

**Recommendation:** **Option C (Both)** — Two group types:
1. **Parent groups** — Declared by naming a registered parent project. Children auto-discovered from paths.
2. **Explicit groups** — Declared by listing project names.

### Fork 3: How does kb context use groups?

**Options:**
- A: Replace `OrchEcosystemRepos` hardcode with group-based filter
- B: Search ALL groups the current project belongs to
- C: Search current group + always include orch ecosystem

**Substrate says:**
- Current behavior: local search first, expand to global with orch-ecosystem filter if sparse
- The current orch-ecosystem filter is the RIGHT pattern, just hardcoded to the wrong scope
- When spawning in toolshed, the agent needs SCS sibling context, not orch ecosystem context

**Recommendation:** **Option A** — Replace `OrchEcosystemRepos` with dynamic group membership. When expanding to global search, filter to the current project's group(s). If the project has no group, fall back to no filtering (return all global matches, same as current local-only behavior).

### Fork 4: How does the daemon use groups?

**Options:**
- A: `--group` flag to scope polling to one group
- B: Always poll all, groups only affect account routing
- C: Per-group daemon instances
- D: `--group` flag with multi-group support (`--group scs,orch`)

**Substrate says:**
- Decision (single-daemon-orchestration-home): Single daemon, unified coordination. Supports Option A or D.
- Current: daemon polls ALL 19 projects. Polling .doom.d and dotfiles for `triage:ready` issues is wasteful.
- Capacity: 3 concurrent agents shared across all projects. No per-group capacity.

**Recommendation:** **Option D** — `--group` flag with multi-group support. `orch daemon run --group scs --group orch` polls both groups. No `--group` = poll all (backward compatible). Groups affect both polling scope AND account routing.

### Fork 5: Account routing

**Options:**
- A: Per-group account in groups.yaml
- B: Per-project account in per-project .orch/config.yaml
- C: Both (per-project overrides group)

**Substrate says:**
- Current: single global account, no routing
- SCS projects ALL use "work" account. Per-project config would be redundant.
- Orch projects ALL use "personal" account. Same.
- Per-project only matters if one SCS project used a different account (unlikely)

**Recommendation:** **Option A** — Per-group account in groups.yaml. Simplest, covers all known cases. Can add per-project override later if needed.

## Synthesis

### The Model: Project Groups

A **project group** is a named collection of projects that share:
1. **KB context scope** — Sibling projects' kb artifacts are surfaced during spawn
2. **Daemon polling scope** — Daemon can filter to specific groups
3. **Account routing** — Group members use the same account

### Config: `~/.orch/groups.yaml`

```yaml
groups:
  orch:
    account: personal
    projects:
      - orch-go
      - orch-cli
      - kb-cli
      - orch-knowledge
      - beads
      - kn
      - opencode

  scs:
    account: work
    parent: scs-special-projects
    # Children auto-discovered from kb projects list:
    # toolshed, price-watch, specs-platform, sendassist, scs-slack
```

### Resolution Algorithm

```
GroupsForProject(projectName) → []Group:
  1. Check explicit groups: is projectName in any group's "projects" list?
  2. Check parent groups: does any group have a "parent" field?
     If so, query kb projects list. Any project whose path is a subdirectory
     of the parent project's path is a member.
  3. Parent project itself is always a member of its own group.
  4. Return all matching groups (empty if ungrouped).

SiblingsOf(projectName) → []string:
  groups := GroupsForProject(projectName)
  siblings := all projects in those groups, excluding projectName
  return siblings
```

### How Groups Cascade

| Consumer | Change | File |
|----------|--------|------|
| **KB context** | Replace `OrchEcosystemRepos` with `GroupsForProject(currentProject)` → filter global results to group members | `pkg/spawn/kbcontext.go` |
| **Daemon polling** | Add `--group` flag to `orch daemon run`. When set, `ListReadyIssuesMultiProject` filters to group projects only | `cmd/orch/daemon.go`, `pkg/daemon/project_resolution.go` |
| **Account routing** | Before spawning, look up target project's group account. Switch if different from current. | `pkg/daemon/daemon.go` |
| **Dashboard** | Add `?group=scs` filter parameter. Expose groups via `GET /api/groups` endpoint | `cmd/orch/serve_filter.go` |
| **Spawn context** | No change needed. Spawn already uses `--workdir` for cross-project. KB context change handles sibling surfacing. | — |

### New Package: `pkg/group/`

```go
package group

type Group struct {
    Name     string   `yaml:"name"`
    Account  string   `yaml:"account,omitempty"`
    Parent   string   `yaml:"parent,omitempty"`    // registered parent project name
    Projects []string `yaml:"projects,omitempty"`  // explicit members
}

type Config struct {
    Groups map[string]Group `yaml:"groups"`
}

// Load reads ~/.orch/groups.yaml
func Load() (*Config, error)

// GroupsForProject returns all groups a project belongs to.
// Checks explicit membership first, then parent-child inference.
func (c *Config) GroupsForProject(projectName string) []Group

// SiblingsOf returns all projects in the same group(s) as the given project.
func (c *Config) SiblingsOf(projectName string) []string

// AllProjectsInGroups returns all projects across specified groups.
func AllProjectsInGroups(groups []Group) []string
```

### Migration: OrchEcosystemRepos → Groups

1. Ship groups.yaml support alongside `OrchEcosystemRepos` (backward compat)
2. If `~/.orch/groups.yaml` exists, use group-based filtering
3. If not, fall back to `OrchEcosystemRepos` hardcode (current behavior)
4. After creating `~/.orch/groups.yaml`, remove hardcode

### Example Scenarios

**Spawning in toolshed:**
```
1. Agent spawned with task "fix pricing logic"
2. Local kb search: toolshed/.kb/ → 1 match (sparse)
3. Expand to global with group filter:
   - GroupsForProject("toolshed") → [scs]
   - Group "scs" members: [scs-special-projects, toolshed, price-watch, specs-platform, sendassist, scs-slack]
   - Filter global results to these projects
4. price-watch kb artifact about "pricing API changes" → INCLUDED ✓
5. orch-go kb artifact about "daemon config" → EXCLUDED (not in group)
```

**Daemon with group scoping:**
```
orch daemon run --group scs --group orch
1. ProjectRegistry loads all 19 projects
2. Group filter applied: only SCS (6) + orch (7) = 13 projects polled
3. .doom.d, blog, snap, dotfiles, agentlog, beads-ui-svelte → SKIPPED
4. When spawning SCS issue: switch to "work" account
5. When spawning orch issue: switch to "personal" account
```

**Dashboard group filter:**
```
GET /api/agents?group=scs
→ Returns only agents from toolshed, price-watch, etc.
```

## Implementation-Ready Checklist

- [x] Problem statement — sibling context, parent-child, daemon scoping
- [x] Approach — hybrid groups (explicit + parent inference)
- [x] File targets — new `pkg/group/group.go`, modify `kbcontext.go`, `project_resolution.go`, `daemon.go`, `serve_filter.go`
- [x] Acceptance criteria — see Success Criteria above (5 items)
- [x] Out of scope — cross-repo beads edges, portfolio dashboard, parent auto-completion

## Recommendations

⭐ **RECOMMENDED:** Hybrid project groups (explicit + parent inference)

- **Why:** Solves all three needs with minimal config. SCS grouping is free (path inference). Orch grouping is a 7-line YAML file. Both cascade to kb context, daemon, and dashboard.
- **Trade-off:** Two resolution mechanisms (inference + explicit) vs. one. Justified because pure-explicit requires listing 6 SCS projects that the filesystem already encodes, and pure-inference can't handle orch ecosystem.
- **Expected outcome:** Spawning in SCS projects surfaces sibling kb artifacts. Daemon can scope to work-only or orch-only polling with account routing. Dashboard can group agents.

**Alternative: Pure explicit config (no parent inference)**
- **Pros:** Single resolution mechanism, no ambiguity
- **Cons:** Must explicitly list all 6 SCS projects (redundant with filesystem). Must maintain list when adding new SCS repos.
- **When to choose:** If parent-child inference proves fragile or confusing in practice.

**Alternative: Per-project config only (no global groups.yaml)**
- **Pros:** Config lives with the project, discoverable per-project
- **Cons:** No single view of all groups. Daemon must scan all projects to discover groups. Account routing scattered across projects.
- **When to choose:** If projects are managed by different teams and global config isn't practical.

## Decision Gate Guidance (if promoting to decision)

**Add `blocks:` frontmatter when:**
- This decision establishes the grouping model that kb context, daemon, and dashboard all depend on

**Suggested blocks keywords:**
- "project group", "cross-project context", "daemon scoping", "account routing", "sibling kb"

## Discovered Work

- **Stale decision:** `2026-01-16-single-daemon-orchestration-home` chose no cross-project polling, but code already implements it via `ListReadyIssuesMultiProject`. Should be superseded with updated decision reflecting current reality.
- **scs-special-projects registration:** Already registered per `kb projects list` output (P0 from cross-repo consequences probe appears resolved).

## References

- Probe: `.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-project-group-model-design.md`
- Decision (stale): `.kb/decisions/2026-01-16-single-daemon-orchestration-home.md`
- Decision (SCS): `/Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/.kb/decisions/2026-02-25-cross-repo-orchestration-from-parent.md`
- Probe (consequences): `.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-cross-repo-orchestration-consequences.md`
- Investigation (prior): `.kb/investigations/2025-12-26-design-multi-project-orchestration-architecture.md`
- Code: `pkg/spawn/kbcontext.go:15-22` (OrchEcosystemRepos), `pkg/daemon/project_resolution.go` (ProjectRegistry)
