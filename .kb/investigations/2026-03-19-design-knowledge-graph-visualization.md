# Design: Knowledge Graph Visualization for Dashboard

**Status:** Complete
**Date:** 2026-03-19
**Issue:** orch-go-ziu4q

## Problem

The knowledge base has 88+ claims across 8 models with 41+ tensions, but there's no visual way to see the claim graph — which models are tightly coupled, where contradictions cluster, which claims are stale. The `orch kb clusters` CLI command surfaces clusters as text, but the dashboard has no equivalent.

## Design: Three Views in a "Claims" Tab

Add a new `/claims` route to the dashboard with three sub-views, selectable via toggle buttons (same pattern as knowledge-tree's Knowledge/Timeline toggle).

### Data Source

All data derives from `.kb/models/*/claims.yaml` files, already parsed by `pkg/claims`. The backend needs one new endpoint.

---

## Backend: `GET /api/claims/graph`

**File:** `cmd/orch/serve_claims.go` (new file, follows `serve_hotspot.go` pattern)

**Returns a single JSON payload containing all data needed for all three views:**

```go
type ClaimsGraphResponse struct {
    Models    []ClaimsModelSummary  `json:"models"`
    Claims    []ClaimNode           `json:"claims"`
    Tensions  []TensionEdge         `json:"tensions"`
    Clusters  []ClusterResponse     `json:"clusters"`
    Stats     ClaimsStats           `json:"stats"`
}

type ClaimsModelSummary struct {
    Name       string `json:"name"`
    ClaimCount int    `json:"claim_count"`
    LastAudit  string `json:"last_audit"`
    Confirmed  int    `json:"confirmed"`
    Stale      int    `json:"stale"`
    Unconfirmed int   `json:"unconfirmed"`
    Contested  int    `json:"contested"`
}

type ClaimNode struct {
    ID            string   `json:"id"`
    Model         string   `json:"model"`
    Text          string   `json:"text"`
    Type          string   `json:"type"`       // observation/mechanism/generalization/invariant
    Confidence    string   `json:"confidence"` // confirmed/stale/unconfirmed/contested
    Priority      string   `json:"priority"`   // core/supporting/peripheral
    DomainTags    []string `json:"domain_tags"`
    LastValidated string   `json:"last_validated"`
    FalsifiesIf   string   `json:"falsifies_if,omitempty"`
    TensionCount  int      `json:"tension_count"`
}

type TensionEdge struct {
    SourceClaim string `json:"source_claim"` // e.g. "HE-01"
    SourceModel string `json:"source_model"` // e.g. "harness-engineering"
    TargetClaim string `json:"target_claim"` // e.g. "AE-01"
    TargetModel string `json:"target_model"` // e.g. "architectural-enforcement"
    Type        string `json:"type"`         // confirms/extends/contradicts
    Note        string `json:"note"`
}

type ClusterResponse struct {
    ID          string   `json:"id"`
    TargetClaim string   `json:"target_claim"`
    TargetModel string   `json:"target_model"`
    Score       float64  `json:"score"`
    Models      []string `json:"models"`
    DomainTags  []string `json:"domain_tags"`
    MemberCount int      `json:"member_count"`
    Members     []ClusterMemberResponse `json:"members"`
}

type ClusterMemberResponse struct {
    ClaimID     string `json:"claim_id"`
    ModelName   string `json:"model_name"`
    Text        string `json:"text"`
    TensionType string `json:"tension_type"`
    Note        string `json:"note"`
}

type ClaimsStats struct {
    TotalClaims     int `json:"total_claims"`
    TotalTensions   int `json:"total_tensions"`
    TotalModels     int `json:"total_models"`
    TotalClusters   int `json:"total_clusters"`
    StaleCount      int `json:"stale_count"`
    UnconfirmedCount int `json:"unconfirmed_count"`
    ContestedCount  int `json:"contested_count"`
}
```

**Implementation logic:**
1. Call `claims.ScanAll(".kb/models")` — already exists
2. Flatten claims into `ClaimNode` array, building a `map[claimID+model]` index
3. Walk all tensions to build `TensionEdge` array (each `Tension` on each `Claim` produces one edge)
4. Call `claims.FindClusters(files, 3)` for cluster data
5. Compute stats summary
6. Return single JSON response

**Registration in `serve.go`:**
```go
mux.HandleFunc("/api/claims/graph", corsHandler(handleClaimsGraph))
```

---

## Frontend: `/claims` Route

### File Structure

```
web/src/
├── routes/claims/+page.svelte          # Page component (follows knowledge-tree pattern)
├── lib/stores/claims.ts                # Svelte store for fetch + state
├── lib/components/claims/
│   ├── index.ts                        # Barrel export
│   ├── TensionGraph.svelte             # View 1: Force-directed graph
│   ├── ClaimHealthGrid.svelte          # View 2: Health heatmap
│   └── ModelRelationshipMap.svelte     # View 3: Model relationship
```

### Navigation

Add to `+layout.svelte` nav bar (between "Knowledge Tree" and "Harness"):
```svelte
<a href="/claims" ...>
  <span class="sm:hidden">Claims</span>
  <span class="hidden sm:inline">Claims</span>
</a>
```

### Store: `claims.ts`

```typescript
interface ClaimsGraphData {
    models: ModelSummary[];
    claims: ClaimNode[];
    tensions: TensionEdge[];
    clusters: ClusterResponse[];
    stats: ClaimsStats;
}

// Simple fetch store (no SSE needed — claims change infrequently)
// Writable store with fetch() method, re-fetches on demand
```

### View 1: Tension Graph (Force-Directed)

**What it shows:** Claims as nodes, tensions as edges. Visual clustering emerges from the force simulation.

**Rendering approach:** SVG with D3 force simulation (d3-force). This is the one place where D3 is justified — force-directed layout is non-trivial to implement from scratch.

**Node encoding:**
- **Color by model** — each model gets a distinct hue (8 models = 8 colors). Use a categorical palette.
- **Size by tension count** — claims with more tensions are larger (hub claims are visually prominent)
- **Border by confidence:** solid = confirmed, dashed = unconfirmed, double = contested, faded = stale
- **Shape:** circles for all (simplest, D3 force works best with circles)

**Edge encoding:**
- **Color by type:** green = confirms, blue = extends, red = contradicts
- **Thickness:** uniform (1px) — type distinction is sufficient
- **Arrowhead:** directed (source → target)

**Interactions:**
- **Hover node:** tooltip with claim text, model, confidence, last_validated
- **Click node:** side panel or bottom drawer showing full claim details + all its tensions + falsifies_if
- **Click cluster:** highlight all nodes in that cluster, dim others. Show cluster score, member list, domain tags.
- **Zoom + pan:** standard D3 zoom behavior
- **Filter:** dropdown to filter by model (show/hide specific models)
- **Legend:** model colors + edge type colors

**Cluster visualization:**
- Clusters from the API are shown as convex hulls (semi-transparent background behind the cluster's nodes) when a cluster filter is active
- A "Clusters" toggle button shows/hides cluster hulls
- Cluster hulls are colored by score (higher score = warmer color)

**Layout tuning:**
- `forceLink` with distance proportional to edge count (tighter for highly connected pairs)
- `forceManyBody` with strength -100 (moderate repulsion)
- `forceCenter` to center the graph
- `forceCollide` with radius = node size + padding

**D3 dependency:** Add `d3-force`, `d3-selection`, `d3-zoom`, `d3-scale` (minimal D3 modules, not the full bundle). These are small (~30KB total gzipped).

### View 2: Claim Health Heatmap

**What it shows:** Per-model grid showing claim health distribution. Quick scan for "which models need attention?"

**Layout:** Grid/table — rows = models, columns = confidence categories.

| Model | Confirmed | Stale | Unconfirmed | Contested | Total |
|-------|-----------|-------|-------------|-----------|-------|
| harness-engineering | 🟢 8 | 🟡 2 | ⬜ 1 | 🔴 0 | 11 |
| architectural-enforcement | 🟢 5 | 🟡 3 | ⬜ 2 | 🔴 1 | 11 |

**Cell encoding:**
- Background color intensity proportional to count (deeper = more)
- Green = confirmed, yellow = stale, gray = unconfirmed, red = contested
- Click cell → expand to show individual claims in that bucket

**Additional row: aggregate "All Models"** at top showing totals.

**Rendering:** Pure HTML/CSS (Tailwind grid). No library needed. This is a table with colored cells.

**Interactions:**
- Click cell → list claims matching that model+confidence
- Sort rows by: name, total claims, stale count, health score (confirmed / total)
- Last audit date shown per row with staleness indicator

### View 3: Model Relationship Map

**What it shows:** Models as nodes, edges weighted by number of cross-model tensions. Shows which models are most interconnected.

**Layout:** Force-directed (same D3 modules as View 1, simpler graph).

**Node encoding:**
- **Size by total claims** — models with more claims are larger
- **Color:** same model colors as View 1
- **Label:** model name displayed inside/next to node

**Edge encoding:**
- **Thickness proportional to tension count** between the two models
- **Color by dominant tension type** between the pair:
  - If majority are contradicts → red
  - If majority are extends → blue
  - If majority are confirms → green
  - Mixed → gray

**Derived data (computed client-side from tensions array):**
```typescript
// Group tensions by (sourceModel, targetModel) pair
// For each pair: count total, count by type
// Create edge with weight = total, dominant type = most common type
```

**Interactions:**
- Hover edge → tooltip: "harness-engineering ↔ architectural-enforcement: 4 confirms, 1 extends, 0 contradicts"
- Click model node → show model summary (claim count, health breakdown, last audit) + list of tensions to/from that model
- Click edge → show individual tensions between the two models

---

## Daemon Spawn Integration

**How to show "what daemon would spawn":**

Each cluster in the API response includes its `score`. The daemon spawns architect issues for clusters above threshold. In the Tension Graph view:

- Clusters above the daemon threshold get a small icon overlay (e.g., robot/bolt icon) indicating "daemon would spawn from this"
- Tooltip on the icon: "Daemon would create architect issue for this cluster (score: N, threshold: 3)"
- This is informational only — the daemon itself handles actual spawning

The threshold value can be included in the API response as `daemon_threshold` (from daemon config, default 3).

---

## Implementation Plan

### Phase 1: Backend (1 file)
1. Create `cmd/orch/serve_claims.go` with `handleClaimsGraph`
2. Register route in `serve.go`
3. Test with `curl localhost:3348/api/claims/graph | jq`

### Phase 2: Frontend Store + Page Shell
1. Create `web/src/lib/stores/claims.ts`
2. Create `web/src/routes/claims/+page.svelte` with view toggle
3. Add nav link in `+layout.svelte`

### Phase 3: Health Heatmap (simplest view)
1. Create `ClaimHealthGrid.svelte`
2. Pure HTML/CSS grid, no external dependencies

### Phase 4: Model Relationship Map
1. Add `d3-force`, `d3-selection`, `d3-zoom`, `d3-scale` to package.json
2. Create `ModelRelationshipMap.svelte`
3. Simpler graph (8 nodes, ~20 edges)

### Phase 5: Tension Graph (most complex)
1. Create `TensionGraph.svelte`
2. Full force-directed with 88+ nodes, 41+ edges
3. Cluster hulls, filtering, click-to-drill

---

## Design Decisions

**Why one endpoint, not three?**
The data is small (88 claims, 41 tensions, 8 models). A single endpoint returns <10KB JSON. Splitting into three endpoints adds complexity for no performance benefit. The client can derive all three views from the same data.

**Why D3 force for Views 1 and 3?**
Force-directed layout is the natural choice for graph visualization. The alternative (manual layout) would require spatial algorithms that D3 already implements. The D3 modules needed are small (~30KB). No other visualization library (ECharts, Plotly) is needed.

**Why not Canvas instead of SVG?**
88 nodes + 41 edges is well within SVG's performance envelope. SVG enables click handlers on individual elements without hit-testing. Canvas would be premature optimization for this scale.

**Why not real-time SSE?**
Claims change via git commits (rare — maybe 2-3x per day). A manual refresh button or periodic poll (every 60s) is sufficient. SSE would add complexity for negligible benefit.

**Why a separate `/claims` route instead of a tab on Knowledge Tree?**
The knowledge tree page already has Knowledge + Timeline views. Adding a third graph view with D3 force simulation would bloat that page. A separate route keeps each page focused and loads D3 only when needed.
