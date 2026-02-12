# Design Investigation: Ecosystem Stability for Production Use

**Question:** How can the orch/bd/kb ecosystem evolve rapidly without destabilizing the platform Dylan depends on for paid SCS work?

**Started:** 2026-02-11
**Updated:** 2026-02-11
**Owner:** og-work-ecosystem-stability-production-11feb-6b0d
**Phase:** Complete
**Status:** Complete

**TLDR:** The ecosystem has no concept of "known good state." Every `make install` immediately replaces the binary that running services and all SCS projects depend on. The fix is three-layered: (1) staged binary deployment with smoke testing, (2) contract tests at integration boundaries, and (3) a stability channel for SCS projects. These can be implemented incrementally without slowing evolution pace.

---

## Problem Framing

### The Quantified Problem

**Last 7 days (Feb 4-11):**
- 399 total commits across orch-go
- 71 fix/revert commits (18%) — nearly 1 fix for every feature
- 64 feature commits (16%)
- 68 chore/refactor commits (17%)
- 98 bd sync commits (25%)
- Average 57 commits/day
- 55 agent branches, most unmerged

**Today (Feb 11) alone:**
- 53 commits, 25 agent branches
- 9 fix commits specifically addressing worktree cascade breakage
- `complete_pipeline.go`, `status_statedb.go`, `beads/client.go` each touched 3+ times
- Fix chain: worktree isolation → beads workdir resolution → phase reporting → complete gates

**File churn hotspots (7 days):**
- `complete_cmd.go`: 20 changes
- `spawn_cmd.go`: 17 changes
- `daemon.go`: 17 changes
- `beads/client.go`: 16 changes
- `verify/check.go`: 16 changes

### The Fix-Feature Cascade Pattern

The dominant failure mode is not "bugs" but **feature-fix cascades**:

```
New Feature (worktree isolation)
    ↓ Introduces new code path
Fix 1: beads workdir resolution doesn't know about worktrees
    ↓ Reveals another integration point
Fix 2: phase reporting doesn't set beads DefaultDir for worktrees
    ↓ Another integration point
Fix 3: complete gates don't check worktree path for artifacts
    ↓ Yet another
Fix 4: status display doesn't resolve worktree → source project
    ↓ Log spam from fallback code
Fix 5: bd_workdir_fallback suppression for expected worktree paths
```

Each fix is correct. The problem is that there are 5 integration points that a feature must update, and nothing tests whether they're all covered BEFORE the feature goes live.

### Why This Matters for SCS Work

**10+ projects** depend on beads (.beads directories found). **10+ projects** depend on orch (.orch directories found). When `make install` runs:

1. The `orch` binary at `~/bin/orch` is replaced immediately
2. `orch serve` (dashboard), `orch doctor`, and daemon all use this binary
3. All SCS projects that run `bd` commands are affected
4. There is no rollback mechanism

Running services: `orch serve` (2.8GB RSS), `orch doctor --daemon`, OpenCode server — all depend on the binary staying compatible with the current state DB schema, beads JSONL format, and worktree layout.

---

## Cascade Failure Taxonomy

### Type 1: Binary Cascade
**Mechanism:** `make install` replaces `~/bin/orch`. All running services and future commands use the new binary immediately.
**Blast radius:** Everything — dashboard, doctor, daemon, all spawns, all SCS projects.
**Frequency:** Multiple times per day during active development.
**Recovery:** Manual rollback (`git checkout <good-hash> && make install`). No automated detection of breakage.

### Type 2: Integration Boundary Cascade
**Mechanism:** A change to one component (spawn) doesn't propagate to all integration partners (complete, phase, status, beads).
**Blast radius:** Specific workflows fail. Today's worktree cascade is the canonical example.
**Frequency:** Every major feature introduction. Worktrees caused 5+ cascading fixes.
**Recovery:** Multiple sequential fix commits across multiple files.

### Type 3: Shared State Cascade
**Mechanism:** SQLite state DB, beads JSONL, and gate-skips.json are mutated by concurrent agents, the orch binary, and the daemon. A schema or behavioral change corrupts the view for running agents.
**Blast radius:** All active agents, dashboard display, completion verification.
**Frequency:** Less common but high severity (Feb 7 crisis, ghost completions).
**Recovery:** Often requires killing and restarting services.

### Type 4: Cross-Project Cascade
**Mechanism:** `bd` and `orch` binaries are global. An update for orch-go development affects SCS project workflows.
**Blast radius:** All projects using the tooling simultaneously.
**Frequency:** Every binary update.
**Recovery:** None — no per-project version pinning exists.

### Type 5: Agent Conflict Cascade
**Mechanism:** 25 agent branches touch overlapping files. Merge conflicts compound when integrating.
**Blast radius:** Master branch quality after batch merge.
**Frequency:** Daily under current volume (55 branches unmerged).
**Recovery:** Manual conflict resolution, often introducing new bugs.

---

## Decision Forks

### Fork 1: Staged Binary Deployment

**Question:** How to deploy orch changes without immediately breaking running services?

**Options:**

**A: Staged build with smoke test (Recommended)**
- `make build` outputs to `build/orch` (already does this)
- `make smoke` runs a fast smoke suite against the NEW binary
- `make install` only promotes if smoke passes
- Add `make install-safe` that runs `smoke` automatically before promoting

**B: Multiple binary versions**
- `orch-dev` for development, `orch` for production
- Services pin to `orch`, development uses `orch-dev`
- More complex but stronger isolation

**C: Systemd-style rolling restart**
- New binary coexists with running services
- Graceful restart triggered explicitly
- Too complex for current needs

**SUBSTRATE:**
- Principle: "Escape hatches — critical paths need independent secondary paths"
- Model: System Reliability Feb 2026 — "unbounded resource pattern" applies here too (unbounded blast radius)
- Evidence: Every fix today went immediately live via `make install`

**RECOMMENDATION: Option A now, evolve to Option B if needed.**

Staged deployment with smoke testing is achievable in hours, not days. The smoke suite catches Type 1 (binary cascade) and Type 2 (integration boundary cascade) before they reach production.

**Implementation:**

```makefile
# New Makefile targets
smoke: build
    @echo "Running smoke tests..."
    ./build/orch version
    ./build/orch status --json >/dev/null
    ./build/orch doctor --check >/dev/null
    go test -run 'TestSmoke' ./cmd/orch/ ./pkg/...
    @echo "Smoke tests passed."

install-safe: smoke
    @echo "Promoting build/orch to ~/bin/orch..."
    # ... existing install logic
```

---

### Fork 2: Integration Boundary Testing

**Question:** How to catch cascading breakage at integration boundaries before deploy?

**Options:**

**A: Contract tests at each boundary (Recommended)**
- Test each integration: orch↔beads, orch↔worktree, orch↔opencode, orch↔statedb
- When a feature adds a new code path, contract tests reveal untested boundaries
- Fast (Go unit tests with mocked externals), run in `make test`

**B: E2E workflow tests**
- Real daemon run with verification
- More realistic but slower and flakier
- Better for nightly/weekly validation

**C: Manual integration checklist**
- Add to PR template: "Did you update all integration points?"
- Relies on human memory — already proven unreliable

**SUBSTRATE:**
- Principle: "Gate over remind" — reminders don't work, gates do
- Evidence: Today's 5-fix cascade — each fix was correct, but nothing caught missing integration points beforehand
- Model: Simplification review identified 14 gates but 0 integration tests

**RECOMMENDATION: Option A (contract tests) as primary, Option B (E2E) for weekly validation.**

**Specific contract tests needed:**

| Boundary | What to Test | Failure This Would Catch |
|----------|-------------|--------------------------|
| orch↔beads | `resolveBDWorkDir()` with worktree paths, source paths, missing paths | Today's bd_workdir_fallback cascade |
| orch↔worktree | Spawn creates worktree, complete finds artifacts in worktree, clean removes worktree | Today's artifact-not-found-in-worktree |
| orch↔statedb | Schema migration, concurrent read/write, cross-project resolution | State corruption from schema changes |
| orch↔opencode | Session lifecycle, SSE events, dead session detection | Feb 7 jetsam crash cascade |
| beads↔worktree | bd commands from worktree cwd, beads project resolution | Phase reporting failure in worktrees |

---

### Fork 3: Stability Channel for SCS Projects

**Question:** How to isolate SCS work from orch development churn?

**Options:**

**A: "Last known good" binary snapshot (Recommended)**
- After smoke+contract tests pass, tag the binary as "stable"
- `make release-stable` copies `build/orch` to `~/.orch/bin/orch-stable`
- SCS projects can use `orch-stable` alias for critical operations
- Orch evolution continues on `orch` binary

**B: Git tag-based release channel**
- Proper semver tags, release branches
- More ceremony than needed for single-developer workflow
- Good for future when multiple consumers exist

**C: Feature flags for new behavior**
- Gate new code paths behind `ORCH_FEATURE_*` env vars
- SCS projects run without experimental features
- Adds complexity to every feature

**SUBSTRATE:**
- Principle: "Pain as signal" — Dylan is feeling the pain of no stability boundary
- Evidence: 65 commits today, mostly fixing cascade breakage from worktree rollout
- Decision: Worktree isolation decision says "Roll out behind a feature flag" — this hasn't happened

**RECOMMENDATION: Option A immediately, evolve toward B as ecosystem matures.**

Option A is achievable in 30 minutes:

```bash
# In Makefile
release-stable: smoke
    @echo "Releasing stable binary..."
    @mkdir -p ~/.orch/bin
    @cp build/orch ~/.orch/bin/orch-stable
    @echo "Stable binary at ~/.orch/bin/orch-stable"
    @echo "Version: $(VERSION)"
```

SCS projects that need stability can symlink or alias to `orch-stable`. Development continues on `~/bin/orch`.

---

### Fork 4: Merge Queue for Agent Branches

**Question:** How to manage 25+ agent branches without cascade conflicts?

**Options:**

**A: Serial merge with test gate**
- Merge one branch at a time
- Run `make test && make smoke` between each merge
- Reject if tests fail, fix forward
- Slow but safe

**B: Squash merge only (Recommended)**
- Each agent produces a single squash commit
- Reduces conflict surface (no intermediate commits)
- Easier to revert individual agent contributions
- Combined with smoke test gate

**C: Batch merge with rollback**
- Merge batch, test, revert entire batch on failure
- Fast but recovery is all-or-nothing

**SUBSTRATE:**
- Evidence: 55 unmerged branches, many touching overlapping files
- Evidence: `complete_pipeline.go` changed 13 times in 7 days from different agents
- Principle: "Verification bottleneck" — tight verification at merge is cheaper than wide supervision

**RECOMMENDATION: Option B (squash merge) with smoke gate.**

The `orch complete` flow should squash-merge agent branches and run `make smoke` before promoting. If smoke fails, the merge is rejected and the agent is flagged for review.

---

## Synthesis: The Three-Layer Architecture

```
Layer 3: Stability Channel
    SCS projects use orch-stable binary
    Updated explicitly after validation
    Decoupled from development velocity

Layer 2: Integration Testing
    Contract tests at each boundary
    Smoke tests exercise critical workflows
    Run before every binary promotion

Layer 1: Staged Deployment
    make build → make smoke → make install
    No direct path from code change to production binary
    Explicit promotion gate
```

**What this preserves:**
- 57 commits/day pace — development on master is unrestricted
- Agent spawning — agents work on feature branches, evolve freely
- Rapid iteration — `make build && make smoke` takes <30 seconds

**What this prevents:**
- Binary cascade — smoke test catches compile errors and basic workflow breakage
- Integration cascade — contract tests catch missing integration points
- Cross-project cascade — SCS projects use `orch-stable`, updated on Dylan's schedule
- Agent conflict cascade — squash merge reduces conflict surface

**What this does NOT solve (out of scope):**
- OpenCode crash cascade — addressed by Feb 7 reliability model
- State DB schema migration — needs separate migration tooling (future work)
- Worktree lifecycle overhead — needs production metrics before optimizing

---

## Implementation Priority

| Priority | Deliverable | Effort | Impact | Catches |
|----------|------------|--------|--------|---------|
| **P0** | `make smoke` target | 1h | High | Type 1 (binary), Type 2 (integration) |
| **P0** | `make install-safe` (smoke before install) | 15min | High | Type 1 (binary) |
| **P1** | Contract tests for orch↔beads boundary | 2h | High | Type 2 (today's cascade) |
| **P1** | Contract tests for orch↔worktree boundary | 2h | High | Type 2 (today's cascade) |
| **P1** | `make release-stable` + orch-stable binary | 30min | Medium | Type 4 (cross-project) |
| **P2** | Contract tests for orch↔statedb boundary | 2h | Medium | Type 3 (shared state) |
| **P2** | Squash merge in `orch complete` | 3h | Medium | Type 5 (agent conflicts) |
| **P3** | Weekly E2E validation suite | 4h | Low | All types, regression detection |

---

## Structured Uncertainty

**What's tested (evidence-based):**
- Churn metrics are directly from git history
- Cascade patterns observed from today's commit chain
- Integration points identified from code search
- Dependency graph verified from filesystem scan

**What's untested:**
- Whether smoke tests actually catch the failure modes in practice
- Whether contract tests are maintainable at current development velocity
- Whether `orch-stable` creates confusion or version skew problems
- Whether squash merge loses valuable commit context from agents

**What would change this:**
- If churn drops significantly (current worktree rollout is a spike), Layer 2 may be overkill
- If SCS projects rarely use orch commands, Layer 3 is unnecessary
- If agent-to-master merge conflicts are rare, Fork 4 is premature

---

## References

**Investigations consulted:**
- `.kb/models/system-reliability-feb2026.md` — Prior crisis (same pattern at resource level)
- `.kb/investigations/2026-02-10-design-simplification-architecture-review.md` — Accidental complexity audit
- `.kb/decisions/2026-02-09-git-isolation-worktree-plus-branch.md` — Worktree decision (source of today's cascades)
- `.kb/investigations/2026-02-09-inv-git-isolation-strategy-multi-agent.md` — Isolation strategy analysis

**Data sources:**
- `git log --since="2026-02-04"` — 399 commits over 7 days
- `git branch -a | grep agent/` — 55 agent branches
- `find ~/Documents/personal/ -maxdepth 2 -name ".beads"` — 10+ dependent projects
- `ps aux | grep orch` — Running service inventory
