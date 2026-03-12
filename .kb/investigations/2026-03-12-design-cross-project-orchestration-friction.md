# Design: Cross-Project Orchestration Friction Audit

**Date:** 2026-03-12
**Phase:** Complete
**Status:** Complete
**Type:** Architect Design
**Model:** daemon-autonomous-operation

## Design Question

The system (daemon, beads, kb context, spawn) assumes single-project operation. Every cross-project interaction requires manual stitching: project registration, prefix config, daemon restart, workdir flags. How do we eliminate this manual stitching to make cross-project spawning as frictionless as single-project?

## Problem Framing

### Success Criteria

1. Cross-project spawn (`orch spawn --workdir ~/other-project`) delivers correct KB context from target project
2. Daemon automatically switches accounts when spawning across project groups
3. Daemon can scope polling to specific project groups
4. Ad-hoc cross-project work doesn't require --no-track (invisible agents)
5. No breaking changes to single-project workflow

### Constraints

- **No Local Agent State:** Groups are project infrastructure config, not agent state (constraint from CLAUDE.md)
- **Session Amnesia:** Config must be discoverable at standard locations
- **Existing group package:** `pkg/group/` is fully implemented and tested — don't redesign, wire it
- **Existing ProjectRegistry:** `pkg/identity/` resolves prefix → directory — leverage, don't replace
- **Prior decision:** Project Group Model (Feb 25) accepted hybrid groups via `~/.orch/groups.yaml`

### Scope

**In scope:** Configuration creation, daemon account routing, daemon group scoping, --no-track replacement, FindSocketPath("") cleanup prioritization
**Out of scope:** Cross-repo beads dependency edges, portfolio dashboard redesign, parent issue auto-completion, cross-project work graph rendering

## Exploration (Fork Navigation)

### Fork 1: What unlocks the most value with the least effort?

**Substrate consulted:**
- Decision (2026-02-25): Project Group Model accepted, pkg/group/ implemented
- Probe (2026-02-25): groups.yaml doesn't exist, everything falls back to hardcode
- Probe (2026-02-27): Cross-repo spawn injects wrong-project KB context because local search uses spawner CWD
- Code: `kbcontext_filter.go:64-68` — group.Load() called, falls back on error

**Analysis:** The group package is fully implemented. The KB context filter code checks for groups.yaml and falls back when it doesn't exist. Creating the config file immediately enables:
- Group-based KB context filtering (sibling knowledge for SCS projects)
- Foundation for daemon account routing and --group flag

**This is a zero-code change** — just create `~/.kb/groups.yaml` (primary) or `~/.orch/groups.yaml` (fallback).

**Recommendation:** Phase 0 — Create config file first, validate behavior, then add code.

### Fork 2: Should the daemon auto-discover groups or use explicit config?

**Options:**
- A: Auto-discover from directory structure only (no config needed)
- B: Explicit config via groups.yaml
- C: Hybrid (auto-discover + explicit overrides)

**Substrate consulted:**
- Design investigation (2026-02-25): Chose Option D (hybrid) — path inference for parent-child, explicit for orch ecosystem
- Probe (2026-02-25): SCS parent-child inferable from paths, orch ecosystem is NOT inferable

**Recommendation:** **B (with inference) — Already decided.** The project group model decision accepted hybrid groups. Don't re-decide. The `pkg/group/` package implements both resolution mechanisms. Wire it.

### Fork 3: How should account routing work?

**Options:**
- A: Per-group account in groups.yaml (already designed)
- B: Per-spawn `--account` flag on `orch work`
- C: Environment-based (set CLAUDE_ACCOUNT before spawn)

**Substrate consulted:**
- Design investigation (2026-02-25): Recommended per-group account in groups.yaml
- Code: `spawnIssue()` in spawn_execution.go has no account parameter
- Code: `orch work` accepts `--workdir` but no `--account`

**Analysis:** Option A is already designed and the config format supports it. The daemon knows the target project's directory → can resolve group → can read account. The `orch account switch <name>` command already exists for runtime switching.

**Recommendation:** **A** — Per-group account in groups.yaml. Daemon reads group config before spawning, switches account if target project's group differs from current account. ~100 LOC in daemon spawn path.

### Fork 4: What replaces --no-track for cross-project ad-hoc work?

**Options:**
- A: Lightweight tracking — real beads issue with `no-verify` label (spawn-architecture model recommendation)
- B: Cross-project beads support — `bd create --project-dir ~/other-project`
- C: Keep --no-track but add tmux visibility
- D: Auto-track with agent's target project beads

**Substrate consulted:**
- Probe (2026-03-03): --no-track creates invisible agents outside both two-lane lanes; 5 isUntrackedBeadsID guards
- Prior decision: "Cross-project epics use --no-track" — but this was described as "only working pattern today"
- Constraint: BEADS_DIR env var injection already works for cross-repo spawns

**Analysis:**
- Option A requires changes to spawn path to create issue with special label, but gives full lifecycle management
- Option B requires changes to `bd` CLI (different repo) — higher effort
- Option C doesn't solve the fundamental invisibility problem
- Option D is the cleanest: when spawning with `--workdir ~/other-project`, create the beads issue in the TARGET project's beads database, not the spawner's. The BEADS_DIR injection already handles this for in-agent `bd comment` calls.

**Recommendation:** **D** — Auto-track in target project. When `--workdir` is set, create beads issue in target project's beads. This means:
1. Remove the need for --no-track in cross-project spawns
2. Agent's beads issue lives where its code lives
3. `orch complete` can find the issue via ProjectRegistry
4. Dashboard visibility via cross-project agent discovery

**Trade-off:** The spawner (orchestrator in orch-go) can't use `bd dep add` to link cross-project issues. This is already a known limitation (cross-repo deps are text-only).

### Fork 5: How to prioritize remaining FindSocketPath("") fixes?

**Analysis of remaining calls:**

| Call Site | Cross-Project Risk | Fix Priority |
|-----------|-------------------|-------------|
| `active_count.go:111` (GetClosedIssuesBatch) | High — wrong closed detection | P1 |
| `active_count.go:258` (BeadsActiveCount) | Medium — undercount | P2 |
| `cleanup.go:173` | Low — cleanup misses agents | P3 |
| `discovery.go:150, 240` | Medium — invisible agents | P2 |
| `artifact_sync_default.go:81` | Low — sync misses artifacts | P3 |
| 7x in issue_adapter.go single-project variants | Low — CLI callers use CWD | P3 |

**Recommendation:** Fix P1 and P2 as part of cross-project hardening. P3 can wait — they affect cleanup and discovery which are non-critical paths.

## Synthesis

### The "80/20 Insight"

The system is **80% architecturally ready** for frictionless cross-project operation. The remaining 20% is:

1. **Config gap (0 code):** groups.yaml doesn't exist
2. **Wiring gap (~150 LOC):** Daemon doesn't consume groups for account routing
3. **Ceremony gap (~200 LOC):** --no-track is the only cross-project ad-hoc pattern
4. **Correctness gap (~70 LOC):** FindSocketPath("") in active_count and discovery

### Recommended Phasing

#### Phase 0: Configuration (Zero Code — Immediate)

Create `~/.kb/groups.yaml`:

```yaml
groups:
  orch:
    account: personal
    projects:
      - orch-go
      - orch-cli
      - kb-cli
      - beads
      - kn
      - opencode

  scs:
    account: work
    parent: scs-special-projects
    # Children auto-discovered: toolshed, price-watch, specs-platform, sendassist, scs-slack
```

**Validates immediately:**
- `orch spawn --workdir ~/toolshed investigation "X"` → KB context includes sibling knowledge from price-watch
- `orch spawn investigation "X"` → KB context filtered to orch ecosystem (same as before, but dynamic)

**Risk:** None. Falls back gracefully if config is malformed.

#### Phase 1: Daemon Account Routing (~100 LOC)

Add to daemon spawn path (`spawnIssue()` in `pkg/daemon/spawn_execution.go`):

```
1. Resolve target project's group via group.GroupsForProject()
2. Read group's account field
3. If different from current account, call account.Switch() before spawn
4. Restore original account after spawn (or let next spawn switch as needed)
```

**Files:** `pkg/daemon/spawn_execution.go`, `pkg/daemon/daemon.go` (load group config at startup)

#### Phase 2: Cross-Project Auto-Tracking (~200 LOC)

Replace --no-track cross-project pattern with auto-tracking in target project:

```
1. In spawn path, when --workdir is set:
   a. Create beads issue in TARGET project's beads (set BEADS_DIR before bd create)
   b. Issue ID uses target project's prefix (e.g., toolshed-xxx)
   c. Return target-project beads ID for lifecycle management
2. Remove --no-track recommendation from cross-project docs
3. Keep --no-track for genuinely untracked spawns (rare)
```

**Files:** `pkg/orch/spawn_beads.go`, `cmd/orch/spawn_cmd.go`

**Impact:** Cross-project ad-hoc spawn drops from 4 flags to 1:
- Before: `orch spawn --bypass-triage --no-track --workdir ~/other --reason "cross-project" investigation "task"`
- After: `orch spawn --bypass-triage --workdir ~/other investigation "task"`

#### Phase 3: Daemon Group Scoping (~50 LOC)

Add `--group` flag to `orch daemon run`:

```
1. Load groups.yaml
2. Resolve group members via ResolveGroupMembers()
3. Filter ProjectRegistry.Projects() to group members
4. Pass filtered list to ListReadyIssuesMultiProject()
```

**Files:** `cmd/orch/daemon_loop.go`, `pkg/daemon/daemon.go`

#### Phase 4: FindSocketPath("") Hardening (~70 LOC)

Fix P1/P2 calls:
- `active_count.go:111` — Accept projectDir parameter, pass to FindSocketPath
- `active_count.go:258` — Accept projectDir parameter
- `discovery.go:150, 240` — Accept projectDir parameter

### What Changes for Each Actor

**For the orchestrator (AI):**
- Cross-project spawn just works: `orch spawn --workdir ~/toolshed investigation "task"`
- KB context automatically includes sibling knowledge
- No --no-track needed for ad-hoc cross-project work
- Spawned agents visible in dashboard

**For the daemon:**
- `orch daemon run --group scs` polls only SCS projects with work account
- `orch daemon run --group orch --group scs` polls both groups
- Account automatically switches per group
- No `--group` = poll all (backward compatible)

**For Dylan:**
- Cross-project work appears in dashboard without special configuration
- Review queue includes cross-project agents naturally
- Account management is automatic

### Defect Class Exposure

| Phase | Defect Class | Mitigation |
|-------|-------------|------------|
| Phase 0 (config) | Class 5 (Contradictory Authority) — groups.yaml vs OrchEcosystemRepos | Graceful fallback: groups.yaml takes precedence, hardcode is fallback |
| Phase 1 (account routing) | Class 4 (Cross-Project Boundary Bleed) — wrong account for wrong project | Thread account from group config, not global state |
| Phase 2 (auto-tracking) | Class 4 (Cross-Project Boundary Bleed) — beads issue in wrong project | BEADS_DIR explicitly set to target project before bd create |
| Phase 3 (group scoping) | Class 0 (Scope Expansion) — new --group flag could accidentally exclude projects | No-flag default = poll all (backward compatible) |
| Phase 4 (FindSocketPath) | Class 4 (Cross-Project Boundary Bleed) — the bug being fixed | Thread projectDir explicitly |

## Recommendations

⭐ **RECOMMENDED:** Four-phase approach, starting with zero-code config creation.

- **Why:** The system is 80% ready. Phase 0 (config file) unlocks KB context group filtering immediately. Each subsequent phase adds one capability with clear scope.
- **Trade-off:** Four phases means 4 implementation issues, but each is independently valuable and shippable.
- **Expected outcome:** Cross-project spawn drops from 4 flags to 1. Daemon account routing eliminates manual switching. KB context automatically surfaces sibling knowledge.

**Critical insight:** The highest-leverage change is creating `~/.kb/groups.yaml` — a single YAML file that immediately enables group-based KB context filtering, because all the code to consume it already exists.

**Alternative: Big-bang implementation**
- **Pros:** Single implementation effort, all features at once
- **Cons:** Larger blast radius, harder to validate incrementally
- **When to choose:** If cross-project work is blocked entirely (it's not — --no-track works)

**Alternative: Do nothing (keep --no-track pattern)**
- **Pros:** Zero effort
- **Cons:** Invisible agents, 4-flag ceremony, no lifecycle management for cross-project work
- **When to choose:** If cross-project work is rare enough that the friction is acceptable

## Discovered Work

- **Model update needed:** daemon-autonomous-operation model claim "can SPAWN cross-project but cannot COMPLETE cross-project" is stale — multi-project completion is implemented
- **Config needed:** `~/.kb/groups.yaml` (Phase 0 — operational, not a code issue)

## References

- Decision: `.kb/decisions/2026-02-25-project-group-model.md` — Accepted hybrid group model
- Design: `.kb/investigations/2026-02-25-design-project-group-model.md` — Original group model design
- Probe: `.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-cross-repo-orchestration-consequences.md`
- Probe: `.kb/models/spawn-architecture/probes/2026-02-27-probe-cross-repo-spawn-context-quality-audit.md`
- Probe: `.kb/models/spawn-architecture/probes/2026-03-03-probe-no-track-invisible-agent-operational-cost.md`
- Probe: `.kb/models/daemon-autonomous-operation/probes/2026-02-25-probe-project-group-model-design.md`
- Code: `pkg/group/group.go` — Group resolution implementation
- Code: `pkg/identity/identity.go` — ProjectRegistry implementation
- Code: `pkg/spawn/kbcontext_filter.go` — Group-aware KB context filtering
- Code: `pkg/daemon/spawn_execution.go` — Daemon spawn path (no account routing)
- Code: `pkg/daemon/completion_processing.go` — Multi-project completion (implemented)
