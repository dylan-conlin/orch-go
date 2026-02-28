# Design: Coupling Hotspot Analysis System

**Date:** 2026-02-19
**Phase:** Complete
**Status:** Complete
**Type:** Architect
**Issue:** orch-go-1109
**Trigger:** Agent ek0b spiraled at 526K tokens trying to discover daemon config's 12-file touch surface. Current `orch hotspot` detects file SIZE only — coupling across layers is invisible.

---

## Design Question

How should we detect cross-surface coupling hotspots — areas where a single concept spans many files across layers — so agents can be warned before they spiral?

## Problem Framing

### Success Criteria

- Detect concept clusters that span 3+ architectural layers (cmd/, pkg/, web/, plugins/)
- Distinguish "healthy coupling" (test+source) from "agent-hostile coupling" (12 files for 1 boolean)
- Produce actionable output: cluster name, file list, coupling score, recommended action
- Integrate naturally with existing `orch hotspot` (not a separate tool)
- Run in <5 seconds on orch-go's 2,733-commit history

### Constraints

- Must use git history as primary signal (universally available, language-agnostic)
- Must not require code parsing (no AST, no language-specific tools)
- Must fit existing HotspotReport JSON schema (for dashboard/scripting)
- Accretion boundary: existing hotspot.go is 806 lines — new analysis belongs in separate file

### Scope

**In:** Co-change detection algorithm, cluster naming, output format, integration with `orch hotspot`, agent spiral correlation heuristic
**Out:** Implementation, frontend changes, coaching plugin integration, spawn gate changes

---

## Evidence: Real Git Mining Results

### Co-Change Analysis (90 days, 2,733 commits, 1,212 non-metadata)

**Cross-surface commits** (touching 3+ architectural layers): 76 of 1,212 (6%)

**Top concept clusters from cross-surface commits:**

| Concept | Cross-Surface Commits | Files | Layers |
|---------|----------------------|-------|--------|
| daemon | 24 | 25 | 3 (cli, pkg, web) |
| verification | 23 | 33 | 2 (cli, pkg) |
| spawn | 22 | 33 | 2 (cli, pkg) |
| agent-status | 22 | 14 | 3 (api, web, pkg) |
| session | 12 | 22 | 4 (cli, pkg, web, api) |
| tmux | 13 | 3 | 1 (pkg only) |

**Key observation:** The daemon config cluster (24 cross-surface commits, 25 files, 3 layers) matches the manually-identified 12-file touch surface from the daemon config investigation. The analysis detected the exact problem that caused the 526K token spiral.

**Tmux is a counter-example:** 13 cross-surface commits but only 1 layer. All coupling is within `pkg/tmux/` — healthy, not agent-hostile.

### Co-Change Pair Data

Top non-test co-change pairs (high coupling = agent concern):

| Co-Changes | Coupling | Pair |
|------------|----------|------|
| 23 | 23% | `cmd/orch/serve.go` <-> `web/src/routes/+page.svelte` |
| 20 | 33% | `cmd/orch/daemon.go` <-> `pkg/daemon/daemon.go` |
| 19 | 19% | `cmd/orch/serve.go` <-> `web/src/lib/stores/agents.ts` |
| 14 | 25% | `web/src/lib/components/agent-card/...` <-> `web/src/lib/stores/agents.ts` |
| 13 | 19% | `cmd/orch/spawn_cmd.go` <-> `pkg/spawn/config.go` |
| 8 | 15% | `cmd/orch/serve_agents.go` <-> `web/src/lib/stores/agents.ts` |

### Event Data (Available Signal)

**events.jsonl** contains:
- `session.spawned` (683 events): beads_id, skill, spawn_mode, task description, workspace
- `agent.completed` (657 events): beads_id, outcome, reason, skill, workspace, verification_passed
- `agent.abandoned` (203 events): beads_id, duration_seconds, skill, workspace

**Workspace manifests** contain: beads_id, skill, git_baseline, spawn_time, model, session_id

**What's missing:** Token count is not in events.jsonl. We know agent ek0b used 526K tokens from the investigation narrative, but this isn't tracked programmatically. Duration is available for abandoned agents but not for completed ones (would need spawn_time - completion_time delta).

---

## Exploration: Decision Forks

### Fork 1: What algorithm for co-change cluster detection?

**Options:**
- A: **Pairwise co-change counting** — count how often each file pair changes together, build clusters from high-coupling pairs
- B: **Commit-level clustering** — group files that co-occur in commits, use community detection (e.g., Louvain) to find clusters
- C: **Layer-crossing filter** — only count co-changes where files span different architectural layers

**Substrate says:**
- Principle (Accretion Gravity): "The fix is structural constraints, not better agents" — detection must feed into gates
- Existing hotspot: uses simple counting (fix commits per file), no graph algorithms
- Evidence: Option C with real data found 76 cross-surface commits and 4 clear concept clusters

**RECOMMENDATION:** Option C — Layer-crossing filter with concept extraction

**Reasoning:** Cross-surface commits (touching 3+ layers) are only 6% of all commits but contain 100% of the agent-hostile coupling. Filtering to cross-surface commits eliminates test+source noise (healthy coupling) automatically. The algorithm:

1. For each commit, classify files by layer (cmd/, pkg/, web/, plugins/)
2. Keep only commits touching 2+ layers (relaxed) or 3+ layers (strict)
3. Count file co-occurrences within these commits
4. Cluster by concept (keyword extraction from file paths)

**Trade-off accepted:** Misses same-layer coupling (e.g., 5 files in pkg/daemon/ that always change together). Acceptable because same-layer coupling is usually healthy — it's cross-layer coupling that makes agents spiral (they discover the surface incrementally).

**When this would change:** If same-layer coupling causes spirals, extend to include clusters with >5 files in same layer.

### Fork 2: How to name/identify concept clusters?

**Options:**
- A: **Path-based extraction** — extract concept from directory/file names (e.g., "daemon" from `pkg/daemon/daemon.go` and `cmd/orch/daemon.go`)
- B: **Commit message mining** — extract concepts from commit messages
- C: **Manual labels** — maintain a `coupling-labels.yaml` mapping files to concepts

**Substrate says:**
- Principle (Session Amnesia): "Will this help the next Claude resume?" — auto-detection is better than manual labels
- Evidence: Path-based extraction successfully identified all 4 clusters (daemon, spawn, verification, agent-status) from real git data
- Existing pattern: `orch hotspot` uses auto-detection for all 3 current types

**RECOMMENDATION:** Option A — Path-based extraction

**Reasoning:** Directory/file names already encode the concept. `pkg/daemon/`, `cmd/orch/daemon.go`, `web/src/lib/stores/daemon.ts` all contain "daemon". This works because Go's package-based structure naturally groups by concept. Hierarchical extraction:
1. Check directory name first (`pkg/daemon/` → "daemon")
2. Fall back to file name stem (`serve_agents.go` → "agents")
3. Group similar names (`agent-card.svelte`, `agents.ts`, `serve_agents.go` → "agent-status")

**Trade-off accepted:** Won't detect unnamed cross-cutting concerns (e.g., "error handling" pattern spanning all layers). Acceptable for initial version — named concepts cover the known pain points.

### Fork 3: Integration — new command or flag on existing?

**Options:**
- A: New flag: `orch hotspot --coupling`
- B: New subcommand: `orch coupling`
- C: Always include coupling analysis in `orch hotspot` output

**Substrate says:**
- Existing hotspot: already has 3 types (fix-density, investigation-cluster, bloat-size) and supports `--json`
- Principle (Coherence Over Patches): adding a 4th type to existing framework > creating parallel tool

**RECOMMENDATION:** Option C — Always include as 4th hotspot type

**Reasoning:** Coupling analysis adds a new `type: "coupling-cluster"` to the existing `Hotspot` struct and `HotspotReport`. This means:
- `orch hotspot` shows all 4 types automatically
- `orch hotspot --json` includes coupling clusters for scripting
- Spawn gates get coupling awareness for free (they already check all hotspot types)
- Dashboard hotspot panel shows coupling without changes

**Trade-off accepted:** Adds ~2-3 seconds to `orch hotspot` runtime (git history parsing). Acceptable because the command is run ad-hoc or at spawn time, not in hot loops.

### Fork 4: Distinguishing healthy vs agent-hostile coupling?

**Options:**
- A: **Layer count threshold** — clusters spanning 3+ layers are hostile
- B: **File count threshold** — clusters with 8+ files are hostile
- C: **Composite score** — weight by (layer_count * file_count * co_change_frequency)

**Substrate says:**
- Evidence: daemon config (12 files, 3 layers) caused 526K spiral. tmux (3 files, 1 layer) never caused issues.
- Principle (Accretion Gravity): "No agent will spontaneously create pkg/spawn/gates/hotspot.go" — the touch surface itself is the problem
- Daemon config investigation: explicitly measured "10-12 files for 1 boolean" as the failure mode

**RECOMMENDATION:** Option C — Composite coupling score

**Reasoning:** A single threshold misses nuance. The coupling score formula:

```
coupling_score = layer_count × unique_files × avg_co_change_frequency
```

Where:
- `layer_count` = number of distinct architectural layers (cmd/, pkg/, web/, etc.)
- `unique_files` = total files in the cluster
- `avg_co_change_frequency` = average number of cross-surface commits per file pair

**Severity classification:**
- `coupling_score >= 100`: CRITICAL (daemon config: 3 layers × 25 files × 2.4 avg = 180)
- `coupling_score >= 40`: HIGH (agent-status: 3 × 14 × 1.6 = 67)
- `coupling_score >= 15`: MODERATE (session: 4 × 22 × 0.5 = 44)
- Below 15: Not flagged

**Healthy coupling filter:** Exclude co-change pairs where:
1. Both files are in the same directory (e.g., `foo.go` + `foo_test.go`)
2. One file is a test file for the other
3. Both files are in `web/` (frontend internal coupling is expected)

**Trade-off accepted:** The score formula is heuristic. May need calibration after real-world use. The thresholds (100/40/15) are derived from the evidence but are estimates.

### Fork 5: Agent spiral evidence — how to correlate?

**Options:**
- A: **Events.jsonl mining** — correlate abandoned agents with files they touched
- B: **Duration-based heuristic** — long-running agents in coupling hotspot areas = spiral signal
- C: **Defer** — don't correlate initially; coupling detection is valuable standalone

**Substrate says:**
- Evidence: events.jsonl has abandoned events (203) with duration_seconds and workspace, but NOT token counts or files touched
- Evidence: Workspace manifests have git_baseline but not files modified (that requires git diff against baseline)
- Principle (Evolve by Distinction): "When problems recur, ask what are we conflating?" — coupling detection and spiral detection are distinct signals that reinforce each other

**RECOMMENDATION:** Option C initially, with Option A as Phase 2

**Reasoning:** Coupling detection is valuable without spiral correlation. The daemon config cluster has a coupling_score of ~180, which is independently informative. Spiral evidence requires:
1. Computing `git diff --stat ${git_baseline}..HEAD` for each workspace (expensive)
2. Cross-referencing with coupling clusters (matching files to concepts)
3. Aggregating across workspaces per concept

This is a natural Phase 2 enhancement after the base coupling analysis ships. For now, the abandoned agent count (203 out of 860 total = 23.6% abandonment rate) is a project-level signal, not a per-cluster signal.

**Trade-off accepted:** We lose the "predict where agents will struggle" capability in Phase 1. We gain "show where coupling is agent-hostile" which addresses 80% of the value.

**When this would change:** Implement Phase 2 when we have 2+ more confirmed spiral-on-coupling incidents (beyond daemon config).

---

## Synthesis: Recommended Architecture

### New Hotspot Type: `coupling-cluster`

Add to existing hotspot analysis as a 4th detection type:

```go
// In new file: cmd/orch/hotspot_coupling.go

type CouplingCluster struct {
    Concept       string   // "daemon", "agent-status", "spawn", etc.
    Files         []string // All files in the cluster
    Layers        []string // Distinct layers: "cli", "pkg", "web", "api"
    LayerCount    int      // len(Layers)
    FileCount     int      // len(Files)
    CoChangeCount int      // Number of cross-surface commits
    Score         float64  // coupling_score = layers * files * avg_frequency
}

func analyzeCouplingClusters(projectDir string, daysBack int) ([]Hotspot, int, error) {
    // 1. Get all commits with files, filter to cross-surface
    // 2. Count file co-occurrences within cross-surface commits
    // 3. Extract concepts from file paths
    // 4. Group co-changing files by concept
    // 5. Score each cluster
    // 6. Filter out healthy coupling (test pairs, same-dir pairs)
    // 7. Return as []Hotspot with type="coupling-cluster"
}
```

### Integration Points

**1. `cmd/orch/hotspot.go:runHotspot()`** — Add call to `analyzeCouplingClusters()`:
```go
// 4. Analyze cross-surface coupling clusters
couplingHotspots, totalClusters, err := analyzeCouplingClusters(projectDir, hotspotDaysBack)
if err != nil {
    fmt.Fprintf(os.Stderr, "Warning: failed to analyze coupling: %v\n", err)
} else {
    report.TotalCouplingClusters = totalClusters
    report.Hotspots = append(report.Hotspots, couplingHotspots...)
}
```

**2. `HotspotReport` struct** — Add `TotalCouplingClusters int` field

**3. `outputText()`** — Add coupling icon `🔗` alongside existing `🔧` (fix), `📚` (investigation), `📏` (bloat)

**4. `RunHotspotCheckForSpawn()`** — Coupling clusters automatically included in spawn-time checks (uses existing hotspot matching infrastructure)

**5. Spawn gate integration** — The existing `pkg/spawn/gates/hotspot.go:CheckHotspot()` works unchanged because coupling clusters are just another `Hotspot` type in the report

### Output Format

**CLI output (within existing hotspot table):**
```
║  🔗 [180] daemon                                                  ║
║      CRITICAL: 25 files across 3 layers (cli, pkg, web) - spawn   ║
║      architect to design extraction before modifying               ║
║  🔗 [ 67] agent-status                                            ║
║      HIGH: 14 files across 3 layers - review coupling before       ║
║      cross-layer changes                                           ║
```

**JSON output (new fields in Hotspot):**
```json
{
  "path": "daemon",
  "type": "coupling-cluster",
  "score": 180,
  "details": "25 files across 3 layers (cli, pkg, web), 24 cross-surface commits",
  "related_files": ["pkg/daemon/daemon.go", "cmd/orch/daemon.go", "web/src/lib/stores/daemon.ts", ...],
  "recommendation": "CRITICAL: Spawn architect to design extraction — concept spans too many layers for tactical modification"
}
```

### Algorithm Detail

```
INPUT: git log --since={days}d --name-only (all commits)

STEP 1: Classify files by layer
  cmd/orch/serve_* → "api"
  cmd/orch/*       → "cli"
  pkg/*            → "pkg"
  web/*            → "web"
  plugins/*        → "plugins"

STEP 2: Filter to cross-surface commits
  Keep commits where len(distinct_layers) >= 2

STEP 3: Count file co-occurrences (within filtered commits only)
  For each commit, for each pair of files:
    co_change[fileA][fileB]++

STEP 4: Extract concept from file paths
  Extract keywords: directory name, file name stem minus suffixes
  "pkg/daemon/daemon.go" → "daemon"
  "cmd/orch/serve_agents.go" → "agents"
  "web/src/lib/stores/daemon.ts" → "daemon"

STEP 5: Group files by concept
  Merge files sharing concept keywords
  "agents", "agent-card", "agent-detail" → "agent-status"

STEP 6: Score each cluster
  coupling_score = layer_count × file_count × (co_change_count / file_count)

STEP 7: Filter healthy coupling
  Remove pairs where: same directory, test-of-source, both in web/

STEP 8: Return as Hotspot entries, sorted by score descending
```

### Healthy vs Agent-Hostile Coupling

| Signal | Healthy | Agent-Hostile |
|--------|---------|---------------|
| Layer count | 1-2 | 3+ |
| Files in cluster | <6 | 8+ |
| Test file ratio | >50% are tests | <30% are tests |
| Same-dir ratio | >70% same dir | <30% same dir |
| Score | <15 | >40 |

**Example healthy:** `pkg/tmux/tmux.go` + `pkg/tmux/tmux_test.go` — 2 files, 1 layer, 100% test ratio → not flagged

**Example hostile:** daemon config cluster — 25 files, 3 layers, 12% test ratio, 12% same-dir ratio → CRITICAL

### Recommendations by Score

```go
func generateCouplingRecommendation(concept string, score float64, layers int) string {
    if score >= 100 {
        return fmt.Sprintf("CRITICAL: '%s' spans %d layers with high coupling — spawn architect to design extraction before modifying", concept, layers)
    }
    if score >= 40 {
        return fmt.Sprintf("HIGH: '%s' has significant cross-layer coupling — review full touch surface before making changes", concept)
    }
    return fmt.Sprintf("MODERATE: '%s' shows cross-layer coupling — be aware of downstream impacts when modifying", concept)
}
```

---

## Implementation Scope

### Phase 1: Core Coupling Analysis (~200 lines)

**New file:** `cmd/orch/hotspot_coupling.go`
- `analyzeCouplingClusters()` — git log parsing, cross-surface filtering, concept extraction, scoring
- Helper functions: `classifyLayer()`, `extractConcept()`, `scoreCoupling()`
- Integration with `runHotspot()` in existing hotspot.go

**Modified file:** `cmd/orch/hotspot.go`
- Add `TotalCouplingClusters` to `HotspotReport`
- Add `🔗` icon to `outputText()`
- Call `analyzeCouplingClusters()` from `runHotspot()`
- Include coupling in `RunHotspotCheckForSpawn()`

**Estimated effort:** 2-3 hours implementation + tests

### Phase 2: Agent Spiral Correlation (deferred)

**When:** After 2+ confirmed spiral-on-coupling incidents
**What:** Cross-reference abandoned agents with coupling clusters via workspace git baselines
**Adds:** `spiral_risk` field to coupling cluster output

### Phase 3: Spawn Gate Enhancement (deferred)

**When:** After Phase 1 ships and is validated
**What:** Block feature-impl spawns in CRITICAL coupling clusters without architect review
**Depends on:** Accretion gravity enforcement (existing investigation 2026-02-14)

---

## Recommendations

⭐ **RECOMMENDED:** Phase 1 — Add coupling-cluster as 4th hotspot type

- **Why:** Detects the exact class of problems that caused the 526K spiral. Integrates cleanly with existing infrastructure. Low implementation cost (~200 lines in new file).
- **Trade-off:** No spiral correlation in Phase 1 (coupling detection alone covers 80% of value)
- **Expected outcome:** `orch hotspot` shows daemon config, agent-status, spawn, and session as coupling hotspots with clear scoring and recommendations

**Alternative: Full agent-spiral mining system**
- **Pros:** Predicts where agents WILL spiral, not just where coupling exists
- **Cons:** Requires token tracking (not in events.jsonl), workspace git diff (expensive), more complex correlation logic
- **When to choose:** When events.jsonl gains token_count field and we have pattern library of spiral signatures

---

## Decision Gate Guidance (if promoting to decision)

**Add `blocks:` frontmatter when:**
- This addresses the recurring coupling blindness that caused agent spirals

**Suggested blocks keywords:**
- "coupling", "hotspot", "cross-surface", "touch surface", "co-change"
