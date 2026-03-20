# Design: Tension-Cluster Backlog Generation

**Status:** Design Complete
**Date:** 2026-03-19
**Issue:** orch-go-euabw

## Problem

Models accumulate claim tensions (cross-model edges) that individually surface in orient but never trigger implementation work. Currently:

- Orient shows up to 2 tension edges per session (informational only)
- Daemon generates probes for stale/unconfirmed individual claims
- Neither mechanism detects **clusters** of related tensions that indicate a structural problem needing architect attention

The gap: when 3+ tensions converge on the same domain area, that's a signal for an architect session to design a resolution — not another investigation.

## Current State (88 claims, 8 models)

Tension inventory (from claims.yaml audit):
- **41 total tensions** across all models
- **0 contradicts** — models are internally consistent
- **~10 extends** — one model deepens/qualifies another
- **~14 confirms** — independent discovery of same principle
- **~17 relates** — topical connection without directional relationship

**Hottest tension targets** (most referenced claims):
- **AE-09** (displacement from deny hooks): 4 tensions from MH, KA models
- **AE-01** (infrastructure > instruction): 4 tensions from ATE, CI, KA models
- **MH-01** (false confidence metrics): 3 tensions from ATE, KA models
- **MH-04** (precision before operational): 3 tensions from SCT, CI, KA models

## Design

### 1. Tension Clustering

**Approach: Domain-tag overlap + shared tension target.**

Two claims are in the same cluster if:
1. They share a **tension target** (both reference the same claim via tensions), OR
2. They share **2+ domain_tags** AND are within the same tension subgraph (connected by extends/contradicts edges)

**Threshold:** A cluster triggers action when it has **3+ claims from 2+ models**.

**Why these numbers:**
- 3+ claims: Matches the synthesis threshold (3+ investigations → synthesis). Below 3, individual probes suffice.
- 2+ models: Single-model tensions are internal consistency issues (handled by model maintenance). Cross-model tensions indicate structural design questions.

**Implementation: `pkg/claims/cluster.go`**

```go
// TensionCluster represents a group of related tensions converging on a domain area.
type TensionCluster struct {
    ID          string   // e.g., "tc-displacement-governance"
    DomainTags  []string // union of domain_tags from clustered claims
    Claims      []ClusterMember
    TargetClaim string   // the claim most tensions point to (hub)
    Models      []string // distinct models involved
    Score       float64  // urgency score
}

type ClusterMember struct {
    ClaimID   string
    ModelName string
    Text      string
    TensionType string // extends, contradicts, confirms
    Note      string
}

// FindClusters scans all claims files and returns tension clusters
// that meet the threshold (3+ claims from 2+ models).
func FindClusters(files map[string]*File, threshold int) []TensionCluster
```

**Clustering algorithm:**
1. Build adjacency graph: for each tension edge, connect source claim → target claim
2. Group by target claim (hub detection): claims that tension-reference the same target
3. For each hub with 3+ inbound tensions from 2+ models → emit cluster
4. Merge overlapping clusters (>50% claim overlap) to avoid duplicates
5. Score by: `contradicts * 3 + extends * 2 + confirms * 1 + (distinct_models - 1) * 2`

**Why hub-based over tag-based:** Domain tags are noisy (many claims share "gates" or "enforcement"). Hub detection finds genuine convergence — multiple models independently discovering the same thing points to a structural design question.

### 2. Architect Context (What the Spawned Architect Receives)

When a tension cluster triggers an architect spawn, the architect's SPAWN_CONTEXT includes:

```
## Tension Cluster: {cluster.ID}

### Convergence Point
{target claim text} ({target claim ID} in {target model})

### Tensions ({len(claims)} claims from {len(models)} models)
{for each member:}
- **{claim.ClaimID}** ({claim.ModelName}): {claim.Text}
  Tension: {claim.TensionType} — {claim.Note}

### Domain Tags
{union of all domain_tags}

### Related Model Sections
{for each unique model_md_ref in cluster:}
- {model_name}/model.md: {model_md_ref}

### Current Focus
{focus goal if set, else "none"}

### Question for Architect
These {len(claims)} claims from {len(models)} models converge on {target claim text}.
Design: what implementation work resolves, strengthens, or restructures this tension area?
```

**What's NOT included:** Full model.md content (too large). The architect reads model files as needed — the context provides pointers (model_md_ref) for targeted reads.

### 3. Output Contract (What the Architect Produces)

The architect produces **typed implementation issues** with claim provenance:

```yaml
# ARCHITECT_OUTPUT.yaml (written to workspace)
cluster_id: tc-displacement-governance
resolution_type: restructure | strengthen | accept | defer
summary: "1-2 sentence resolution strategy"

issues:
  - title: "Add redirect hints to deny hooks for governance-protected files"
    skill: feature-impl
    priority: 2
    claim_provenance:
      - AE-09  # displacement from deny hooks
      - KA-10  # anti-accretion mechanisms creating second-order problems
    depends_on: []  # other issue indices (0-based) in this list
    description: |
      Deny hooks currently block edits without suggesting where code should go.
      Add redirect hints (suggested file/package) to hook error messages.

  - title: "Implement displacement tracking metric for deny hook effectiveness"
    skill: feature-impl
    priority: 3
    claim_provenance:
      - MH-05  # absent-signal trap
      - MH-07  # two-gap independence
    depends_on: [0]  # must know redirect destinations first
    description: |
      Track whether denied edits land in architecturally correct locations.
      Compare agent code placement before/after redirect hints.
```

**Contract rules:**
- Each issue references 1+ claim IDs from the cluster (provenance)
- `depends_on` forms a DAG (no cycles) for daemon sequencing
- `skill` must be a valid skill name (feature-impl, investigation, systematic-debugging)
- `resolution_type` tells the daemon how to interpret the output:
  - `restructure`: issues redesign the tension area (highest value)
  - `strengthen`: issues add measurement/enforcement to confirm claims
  - `accept`: tension is acknowledged as inherent, issues add documentation
  - `defer`: not actionable now, architect explains why

### 4. Daemon Integration

**New periodic task: `TaskTensionClusterScan`**

```go
// In pkg/daemon/scheduler.go
TaskTensionClusterScan = "tension_cluster_scan"

// In pkg/daemonconfig/config.go
TensionClusterScanEnabled  bool
TensionClusterScanInterval time.Duration // default: 24h
TensionClusterThreshold    int           // default: 3
```

**Lifecycle:**
1. Daemon periodic scan calls `claims.FindClusters()` (same as probe generation reads claims)
2. For each cluster meeting threshold:
   a. Dedup: check if open architect issue already exists for this cluster ID
   b. Create beads issue: `bd create "{cluster summary}" --type task -l triage:ready -l skill:architect -l daemon:tension-cluster`
   c. Issue description includes the tension cluster context (section 2 above)
3. Max 1 cluster issue per cycle (same as probe generation)
4. Emit event: `daemon.tension_cluster` with cluster metadata

**Routing:** The `skill:architect` label ensures daemon skill inference routes to architect (Priority 1 label override, per DAO-04). The architect skill is appropriate because this is "should we / how should we design X" — not implementation.

**After architect completes:** The ARCHITECT_OUTPUT.yaml issues are created as beads issues by the completion pipeline (or orchestrator), each with `triage:ready` and inferred skill labels. Dependencies use beads `depends_on` field for daemon sequencing (per DAO-15 sibling sequencing pattern).

### 5. Worked Example: AE-09 Displacement Cluster

**Current tensions pointing at AE-09:**
- MH-02 (measurement-honesty): "Deny count without displacement tracking is the absent-signal trap"
- MH-05 (measurement-honesty): "Measuring action without consequence"
- MH-07 (measurement-honesty): "Displacement is the concrete consequence deny counts miss"
- KA-10 (knowledge-accretion): "Deny hooks without redirect create displacement"

**Cluster:** 4 claims from 2 models (MH + KA) → threshold met (3+ claims, 2+ models)

**Score:** 0 contradicts + 3 extends + 1 confirms + (2-1)*2 = 0 + 6 + 1 + 2 = 9

**Architect would receive:** The 4 tension edges, AE-09's text and falsifies_if, pointers to model.md sections, and the design question.

**Expected architect output:** 2-3 implementation issues around redirect hints in deny hooks and displacement tracking metrics — exactly the work identified but never generated.

### 6. What This Does NOT Do

- **Does not replace probes.** Probes validate individual claims. Tension clusters generate implementation work from claim relationships.
- **Does not auto-resolve tensions.** The architect decides the resolution type. The system only detects clusters and routes them.
- **Does not scan `confirms` tensions by default.** Confirms edges indicate healthy convergence, not design questions. Only `extends` and `contradicts` create actionable tension. (Configurable: a `--include-confirms` flag for completeness.)
- **Does not require new claim fields.** Uses existing tensions, domain_tags, and model_md_ref fields in claims.yaml.

## Implementation Scope

| Component | File | Effort |
|-----------|------|--------|
| Clustering logic | `pkg/claims/cluster.go` | New file, ~150 lines |
| Clustering tests | `pkg/claims/cluster_test.go` | New file, ~200 lines |
| Daemon periodic task | `pkg/daemon/tension_cluster.go` | New file, ~120 lines (follows claim_probe_generation.go pattern) |
| Daemon config | `pkg/daemonconfig/config.go` | Add 3 fields |
| Scheduler registration | `pkg/daemon/scheduler.go` | Add 1 task |
| Periodic handler | `cmd/orch/daemon_periodic.go` | Add handler (~30 lines, follows existing pattern) |
| Architect output parser | `pkg/claims/architect_output.go` | New file, ~80 lines (YAML parse of ARCHITECT_OUTPUT.yaml) |
| Orient surfacing | `pkg/claims/claims.go` | Extend `CollectEdges` to include cluster warnings |

**Total:** ~580 lines of new code across 4 new files + 4 existing file modifications.

## Dependencies

- `pkg/claims` (existing) — claim types, ScanAll, domain_tags
- `pkg/daemon` (existing) — scheduler, periodic task pattern
- `pkg/beads` (existing) — issue creation
- Architect skill (existing) — receives tension cluster context via SPAWN_CONTEXT
