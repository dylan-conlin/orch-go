# Probe: Project Group Model — Scoping Daemon, KB Context, and Account Routing

**Date:** 2026-02-25
**Model:** daemon-autonomous-operation
**Status:** Complete
**Beads:** orch-go-1235

## Question

The daemon-autonomous-operation model describes multi-project polling via `ProjectRegistry` and `ListReadyIssuesMultiProject`. Three converging needs require a project GROUP concept: (1) sibling kb context for SCS work projects, (2) parent-child project relationships, (3) daemon polling and account scoping. Does the current flat polling model support these needs, or does it require a grouping layer?

## What I Tested

### 1. KB Context Scoping (kbcontext.go)

**Tested:** How global search filters results when spawning in non-orch-ecosystem projects.

```go
// pkg/spawn/kbcontext.go:15-22
var OrchEcosystemRepos = map[string]bool{
    "orch-go": true, "orch-cli": true, "kb-cli": true,
    "orch-knowledge": true, "beads": true, "kn": true,
}
```

**Observed:** The `filterToOrchEcosystem()` function (line 207) uses a HARDCODED allowlist. When spawning in `toolshed` and local search returns <3 matches, global search expands but then post-filters to only orch ecosystem repos. **All SCS project matches (price-watch, specs-platform, sendassist, work-slack) are discarded.** This means:
- Spawning into toolshed CANNOT surface price-watch kb artifacts
- Spawning into price-watch CANNOT surface toolshed kb artifacts
- The sibling context need is completely blocked by hardcoded allowlist

### 2. Daemon Project Discovery (project_resolution.go)

**Tested:** What ProjectRegistry returns and how it's used.

```bash
kb projects list --json
# Returns 19 projects, ALL registered, including work-monorepo
```

**Observed:** ProjectRegistry calls `kb projects list --json` and builds a flat `prefixToDir` map. No grouping concept exists. The daemon iterates ALL 19 projects on every poll cycle (line 583 of daemon.go). There is no way to:
- Scope polling to a subset of projects
- Route different accounts to different projects
- Set different capacity limits per group

### 3. Parent-Child Inference from Paths

**Tested:** Whether registered project paths encode parent-child relationships.

**Observed:** YES — clear structural pattern in kb projects output:
```
work-monorepo → ~/Documents/work/WorkCorp/work-monorepo
price-watch          → ~/Documents/work/WorkCorp/work-monorepo/price-watch
toolshed             → ~/Documents/work/WorkCorp/work-monorepo/toolshed
specs-platform       → ~/Documents/work/WorkCorp/work-monorepo/specs-platform
sendassist           → ~/Documents/work/WorkCorp/work-monorepo/sendassist
work-slack            → ~/Documents/work/WorkCorp/work-monorepo/work-slack
```

Child project paths are all subdirectories of the parent project path. This relationship is unambiguous and stable.

For orch ecosystem, path inference DOES NOT work:
```
orch-go         → ~/Documents/personal/orch-go
orch-knowledge  → ~/orch-knowledge  (DIFFERENT parent!)
opencode        → ~/Documents/personal/opencode
blog            → ~/Documents/personal/blog  (NOT orch ecosystem)
```

### 4. Decision Precedent: Single Daemon Architecture

**Tested:** Whether existing decisions constrain the design.

**Observed:** Decision `2026-01-16-single-daemon-orchestration-home.md` chose Option A (single daemon, orch-go home, NO cross-project polling). BUT the code has ALREADY EVOLVED past this — `ListReadyIssuesMultiProject` polls all 19 projects. The decision is **stale**. The code reflects Option B (cross-project polling) which the decision explicitly rejected.

This means the decision should be superseded. The current reality (cross-project polling works) is the new baseline.

### 5. Account Routing Gap

**Tested:** Whether daemon can route accounts per project.

**Observed:** The daemon uses a single globally-active account for ALL spawns. The spawn flow at `daemon.go:583→SpawnWork()` passes `model` and `workdir` but has NO account parameter. When spawning SCS work (which should use "work" account) from orch-go (which uses "personal" account), there's no mechanism to switch.

## What I Observed (Summary)

| Need | Current State | Gap |
|------|--------------|-----|
| Sibling kb context | Hardcoded `OrchEcosystemRepos` blocks SCS | Need group-based filter |
| Parent workdir | Paths encode hierarchy but nothing reads it | Need parent-child resolution |
| Daemon scoping | Flat polling all 19 projects | Need group filter + account routing |

## Model Impact

**EXTENDS the daemon-autonomous-operation model:**

1. **Multi-project polling works** but lacks scoping — confirms ProjectRegistry implementation is functional, extends with the need for group-level filtering
2. **Account routing is missing** — new model claim: daemon needs per-group account selection to avoid using personal account for work projects
3. **KB context filtering is coupled to orch-ecosystem** — the `OrchEcosystemRepos` hardcode is a proxy for "project group" but only serves one group; needs generalization

**CONTRADICTS one decision:**
- Decision `2026-01-16-single-daemon-orchestration-home` chose no cross-project polling, but code already implements it. Decision is stale and should be superseded.

**NEW model claim to add:**
- Projects form groups through two mechanisms: structural (parent directory) and semantic (explicit config). Groups scope three concerns: kb context search, daemon polling, and account routing.
