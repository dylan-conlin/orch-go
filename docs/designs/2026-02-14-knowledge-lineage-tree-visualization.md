# Design: Knowledge Lineage Tree Visualization

**Date:** 2026-02-14
**Status:** Proposed
**Owner:** Orchestrator System
**Context:** The knowledge base contains rich lineage relationships (investigations → decisions → issues, models → probes, guides ← investigations) but no way to visualize the natural parent-child structure. This design creates a tree-based visualization that serves as Dylan's primary interface for understanding project state.

## Problem Statement

The knowledge base has grown to 861 investigations, 54 decisions, 41 models, and many guides, with natural parent-child relationships embedded in:
- Prior-Work tables in investigations
- Evidence sections in decisions  
- Synthesized From headers in models
- Filesystem hierarchy for probes
- Beads issue references to decisions

**Current pain:** No way to see the knowledge landscape, spot investigation clusters without decisions, understand where new work fits, or orient new session Claudes.

**The resonant example:** Dylan saw this tree rendering and said "wow, this is a really cool view":

```
orch-go knowledge tree
│
├─◉ Entropy Spiral
│ ├─◉ post-mortems/2026-01-02-system-spiral-dec27-jan02.md
│ ├─◉ investigations/2026-02-12-inv-entropy-spiral-postmortem.md
│ ├─◉ investigations/2026-02-14-inv-entropy-spiral-deep-analysis.md
│ │ ├─★ decisions/2026-02-14-verifiability-first-hard-constraint.md
│ │ │ ├─● orch-go-5sc  Decision record                    CLOSED
│ │ │ ├─● orch-go-7jl  Daemon pause after N completions   CLOSED
│ │ │ └─◇ orch-go-6th  Skill system update                triage:review
│ │ └─● orch-go-agr  Trajectory audit                     IN PROGRESS
│ └─◉ handoffs/2026-02-13-entropy-spiral-recovery.md
```

## Design Questions

### 1. Where Does This Render?

**Answer:** Both dashboard (persistent browsing) and orchestrator session (contextual verification).

**Dashboard (Primary):**
- **Location:** New route `/knowledge-tree` in web dashboard
- **Interaction:** Full-featured tree browser with expand/collapse, filtering, search
- **Persistence:** Tree state saved to localStorage (expanded nodes, filters)
- **Use cases:** Session start orientation, triage, exploration

**Orchestrator Session (Secondary):**
- **Location:** Inline rendering during `orch complete` verification flow
- **Interaction:** Read-only ASCII tree showing where new deliverable fits
- **Context:** Shown when daemon pauses after N completions, highlights new nodes
- **Use cases:** Verification context, explain-back preparation

**CLI Command (Tertiary):**
- **Command:** `orch tree` or `kb tree` for quick lookups
- **Output:** ASCII tree like example above
- **Options:** `--cluster <name>`, `--depth <n>`, `--format json|text`
- **Use cases:** Quick orientation, scripting, CI checks

**Recommendation:** Start with dashboard (primary value), add CLI command second (cheap), defer orchestrator integration (complex, needs verification flow design).

### 2. Data Model: How to Extract the Tree?

**Answer:** Parse existing artifacts, no new metadata required. Relationships already exist.

**Extraction Sources:**

| Relationship | Source | Pattern | Example |
|--------------|--------|---------|---------|
| Investigation → Prior Work | `Prior-Work` table in investigation.md | Markdown table row | `\| .kb/investigations/...md \| extends \| yes \| None \|` |
| Decision → Evidence | `Evidence` section in decision.md | Inline references | Reference to investigation path |
| Model → Investigations | `Synthesized From:` header in model.md | Header value | `31 investigations + completion.md guide...` |
| Model → Probes | Filesystem hierarchy | `.kb/models/{name}/probes/*.md` | Directory structure |
| Decision → Issues | Context section in decision.md | Beads ID references | `Related Epic: orch-go-xxxxx` |
| Issue → Decision | Beads issue description | Description text | Reference to decision path |
| Guide → Investigations | Guide frontmatter/content | Inline references | References to investigation paths |

**Extraction Algorithm:**

```go
type KnowledgeNode struct {
    ID           string                 // Unique identifier
    Type         string                 // investigation, decision, model, probe, guide, issue
    Path         string                 // File path or beads ID
    Title        string                 // Display name
    Children     []*KnowledgeNode       // Child nodes
    Metadata     map[string]interface{} // Status, dates, etc.
}

func ExtractKnowledgeTree(rootDir string) (*KnowledgeNode, error) {
    root := &KnowledgeNode{Type: "root", Title: "orch-go knowledge"}
    
    // 1. Parse all .kb/ files, extract metadata
    investigations := parseInvestigations(rootDir + "/.kb/investigations")
    decisions := parseDecisions(rootDir + "/.kb/decisions")
    models := parseModels(rootDir + "/.kb/models")
    guides := parseGuides(rootDir + "/.kb/guides")
    
    // 2. Build relationship graph from Prior-Work tables, Evidence sections, etc.
    graph := buildRelationshipGraph(investigations, decisions, models, guides)
    
    // 3. Cluster by topic (inferred from area labels, filesystem paths, or explicit tags)
    clusters := inferTopLevelClusters(graph)
    
    // 4. For each cluster, build tree by following parent→child edges
    for _, cluster := range clusters {
        clusterNode := buildClusterTree(cluster, graph)
        root.Children = append(root.Children, clusterNode)
    }
    
    return root, nil
}
```

**Prior-Work Table Parser:**
```go
func parsePriorWorkTable(markdown string) []Relationship {
    // Find "## Prior Work" section
    // Extract markdown table rows
    // Parse: | path | relationship | verified | conflicts |
    // Return structured relationships
}
```

**No Manual Curation:** Tree is 100% derived from existing artifact content. Adding a Prior-Work reference = adding to tree.

### 3. Interaction: Expand/Collapse, Filtering, Verification State?

**Answer:** Rich interaction model optimized for triage and orientation.

**Core Interactions:**

| Feature | Implementation | Use Case |
|---------|----------------|----------|
| **Expand/Collapse** | Click node to toggle children | Navigate large trees |
| **Filter by Type** | Checkboxes: investigations, decisions, models, issues | Focus on specific artifact types |
| **Filter by Area** | Dropdown: spawn, completion, dashboard, etc. | Focus on specific domains |
| **Search** | Fuzzy search on titles | Find specific artifacts |
| **Highlight Verification State** | Color coding: ◉ (complete), ◇ (triage), ● (in progress) | See work status at a glance |
| **Show Health Smells** | Warnings icon on clusters with 15+ investigations but no decision | Spot synthesis needs |
| **Click to Navigate** | Click node → navigate to file or beads issue | Quick access to artifacts |

**Visual Design:**

```
◉ Investigation (complete)     - Green
◇ Investigation (triage)       - Yellow  
● Investigation (in progress)  - Blue
★ Decision                     - Gold
◆ Model                        - Purple
◈ Guide                        - Cyan
◐ Issue (open)                 - Orange
◓ Issue (closed)               - Gray
⚠ Cluster smell (15+ invs)    - Red badge
```

**Interaction Modes:**

1. **Browse Mode (default):** Full tree, all types visible
2. **Triage Mode:** Filter to show only clusters with health smells (many investigations, no decisions)
3. **Verification Mode:** Show only recently added nodes (last 7 days) with status highlighting

**Keyboard Navigation:**
- `j/k` - Navigate up/down  
- `Enter` - Expand/collapse
- `o` - Open artifact in editor
- `f` - Focus search
- `/` - Quick filter

### 4. Integration with Verify Flow

**Answer:** Show tree when daemon pauses, highlight new deliverable in context.

**Verification Context Protocol:**

```bash
# During verification (orch complete flow)
1. Agent completes work, reports Phase: Complete
2. Daemon detects completion, pauses before next spawn
3. Orchestrator runs: orch complete orch-go-xxxx
4. Before explain-back prompt, show:
   
   📍 Where this deliverable fits:
   
   orch-go knowledge tree (filtered to current cluster)
   │
   ├─◉ Entropy Spiral
   │ ├─◉ investigations/2026-02-14-inv-entropy-spiral-deep-analysis.md
   │ │ ├─★ decisions/2026-02-14-verifiability-first-hard-constraint.md  ← existing
   │ │ │ └─● orch-go-7jl  Daemon pause after N completions [NEW]  ← deliverable fits here
   
   This shows Dylan:
   - What cluster the work belongs to
   - What parent artifacts it builds on  
   - Where it fits in the lineage
   
5. Dylan proceeds with explain-back with full context
```

**Implementation:**

```go
func ShowVerificationContext(beadsID string) error {
    // 1. Extract deliverable type and lineage from SYNTHESIS.md or beads description
    deliverable := extractDeliverable(beadsID)
    
    // 2. Find parent artifacts (what investigation/decision spawned this?)
    parents := findParentArtifacts(deliverable)
    
    // 3. Render tree filtered to relevant cluster, highlight new node
    tree := extractKnowledgeTree(".")
    cluster := findCluster(tree, parents)
    
    // 4. Render ASCII tree with [NEW] marker
    renderTree(cluster, highlightNode(beadsID))
    
    return nil
}
```

**Benefits:**
- Dylan sees context before explain-back
- Reduces "where does this fit?" questions
- Makes verification more informed
- Natural checkpoint for understanding before approval

### 5. CLI vs Web: Should 'orch tree' or 'kb tree' Exist?

**Answer:** Both. Different use cases.

**CLI Command Design:**

```bash
# Quick lookup - show full tree
orch tree

# Filter to cluster
orch tree --cluster "entropy-spiral"

# Show only specific depth
orch tree --depth 2

# JSON output for scripting
orch tree --format json

# Show only artifacts with health smells
orch tree --smells

# Show what's new in last N days
orch tree --since 7d
```

**Ownership:**
- `orch tree` - Lives in orch-go, focuses on orchestration artifacts (workspaces, synthesis, handoffs)
- `kb tree` - Lives in kb-cli, focuses on knowledge artifacts (investigations, decisions, models)
- Overlap is fine - both can render full tree from different perspectives

**Web Dashboard:**
- Full-featured browser with interaction, state persistence
- Primary interface for exploration and triage
- Supports all use cases CLI doesn't (expand/collapse, filtering, visual design)

**Decision Matrix:**

| Use Case | Tool |
|----------|------|
| Quick orientation at session start | CLI: `orch tree` |
| Deep triage for synthesis needs | Web: `/knowledge-tree` |
| Scripting/automation | CLI: `orch tree --format json` |
| Verification context | CLI: inline during `orch complete` |
| Browsing knowledge landscape | Web: `/knowledge-tree` |

**Recommendation:** Implement CLI first (simpler, faster MVP), add web dashboard second (richer interaction).

### 6. Grouping: How to Determine Top-Level Clusters?

**Answer:** Multi-signal clustering with fallback hierarchy.

**Clustering Strategy:**

```
Priority 1: Explicit area: labels (if present in beads integration)
Priority 2: Filesystem paths (investigations/synthesized/{cluster}/)  
Priority 3: Shared Prior-Work references (investigations citing same root)
Priority 4: Lexical similarity (embedding-based clustering as fallback)
```

**Examples:**

| Cluster Name | Signal | Evidence |
|--------------|--------|----------|
| Entropy Spiral | Filesystem | `.kb/investigations/synthesized/entropy-spiral/*.md` |
| Serve Performance | Filesystem | `.kb/investigations/synthesized/serve-performance/*.md` |
| Coaching Plugin | Filesystem | `.kb/investigations/synthesized/coaching-plugin/*.md` |
| Completion Verification | Model | `.kb/models/completion-verification/` + related investigations |
| Daemon Autonomous Operation | Model | `.kb/models/daemon-autonomous-operation/` + related investigations |

**Cluster Inference Algorithm:**

```go
func inferTopLevelClusters(graph *KnowledgeGraph) []Cluster {
    clusters := []Cluster{}
    
    // 1. Filesystem-based clusters (investigations/synthesized/{name}/)
    fsClusters := extractFilesystemClusters(".kb/investigations/synthesized")
    clusters = append(clusters, fsClusters...)
    
    // 2. Model-based clusters (.kb/models/{name}/ + referencing investigations)
    modelClusters := extractModelClusters(".kb/models", graph)
    clusters = append(clusters, modelClusters...)
    
    // 3. Decision-based clusters (decisions + citing investigations + spawned issues)
    decisionClusters := extractDecisionClusters(".kb/decisions", graph)
    clusters = append(clusters, decisionClusters...)
    
    // 4. Orphan investigations (no clear cluster) → "Uncategorized"
    orphans := findOrphans(graph, clusters)
    if len(orphans) > 0 {
        clusters = append(clusters, Cluster{Name: "Uncategorized", Nodes: orphans})
    }
    
    return clusters
}
```

**Cluster Display:**

```
orch-go knowledge tree
│
├─◉ Entropy Spiral (4 investigations, 2 decisions, 5 issues)  ⚠ health smell
├─◉ Serve Performance (7 investigations, 1 decision, 3 issues)
├─◆ Completion Verification (31 investigations, 1 model, 3 probes)
├─◆ Daemon Autonomous Operation (12 investigations, 1 model, 2 probes)
├─◉ Coaching Plugin (2 investigations, 1 decision)
└─◉ Uncategorized (23 investigations)
```

**Health Smells:**
- ⚠ 15+ investigations without a decision or model = needs synthesis
- ⚠ Decision without any spawned issues = not acted on
- ⚠ Model without probes = untested model claims

## Use Case Walkthrough

### Use Case 1: Session Start Orientation

**Scenario:** Dylan starts a new orchestrator session, needs to see the knowledge landscape.

**Flow:**
1. Dylan runs: `orch tree --smells`
2. CLI renders tree highlighting clusters with health smells
3. Dylan sees: "Entropy Spiral has 4 investigations and 2 decisions with 5 spawned issues"
4. Dylan decides: "This cluster is active, I should review before starting new work"
5. Dylan opens dashboard: `/knowledge-tree?cluster=entropy-spiral`
6. Dashboard shows full lineage, Dylan understands current state

**Value:** Immediate orientation without reading hundreds of files.

### Use Case 2: Triage for Synthesis Needs

**Scenario:** Orchestrator needs to find investigation clusters that need synthesis into decisions or models.

**Flow:**
1. Orchestrator reviews: `/knowledge-tree` in Triage Mode (filter: health smells)
2. Dashboard highlights: "23 investigations in Uncategorized, no decision/model"
3. Orchestrator spawns: `orch spawn investigation "synthesize uncategorized investigations"`
4. Investigation reviews cluster, proposes decision or model
5. After synthesis, health smell disappears from tree

**Value:** Systematic triage without manual file scanning.

### Use Case 3: Verification Context

**Scenario:** Agent completes work, Dylan needs to verify before next spawn.

**Flow:**
1. Agent reports: `Phase: Complete` for orch-go-7jl
2. Daemon pauses, Dylan runs: `orch complete orch-go-7jl`
3. Before explain-back, CLI shows tree snippet:
   ```
   📍 Where this deliverable fits:
   ├─◉ Entropy Spiral
   │ ├─★ decisions/2026-02-14-verifiability-first-hard-constraint.md
   │ │ └─● orch-go-7jl  Daemon pause after N completions [NEW]
   ```
4. Dylan sees this implements the decision about verification constraints
5. Dylan proceeds with informed explain-back

**Value:** Context before verification reduces cognitive load.

### Use Case 4: New Session Claude Orientation

**Scenario:** New Claude spawned, needs to understand project structure quickly.

**Flow:**
1. SPAWN_CONTEXT includes: "Run `orch tree --depth 2` for project orientation"
2. Agent runs command, sees top-level clusters and key artifacts
3. Agent understands: "Main work areas are Entropy Spiral, Serve Performance, Completion Verification"
4. Agent proceeds with context-aware work

**Value:** Faster onboarding for spawned agents.

## Implementation Phases

### Phase 1: CLI MVP (Week 1)
- [ ] Implement extraction algorithm for Prior-Work tables
- [ ] Implement filesystem-based cluster detection
- [ ] Implement ASCII tree renderer (Unicode box-drawing)
- [ ] Add `orch tree` command with basic options
- [ ] Test with existing .kb/ structure
- **Deliverable:** Working CLI command

### Phase 2: Health Smells & Filtering (Week 2)
- [ ] Implement health smell detection (15+ investigations without decision)
- [ ] Add filtering options (--cluster, --depth, --smells)
- [ ] Add JSON output format for scripting
- [ ] Add color coding for node types
- **Deliverable:** Full-featured CLI tool

### Phase 3: Dashboard Integration (Week 3-4)
- [ ] Create `/knowledge-tree` route in web dashboard
- [ ] Implement Svelte tree component with expand/collapse
- [ ] Add filtering UI (type, area, status)
- [ ] Add search functionality
- [ ] Save tree state to localStorage
- [ ] Add click-to-navigate to artifacts
- **Deliverable:** Full-featured web browser

### Phase 4: Verification Integration (Week 5)
- [ ] Integrate tree rendering into `orch complete` flow
- [ ] Highlight new deliverable in context
- [ ] Add "where this fits" section before explain-back
- [ ] Test with daemon pause flow
- **Deliverable:** Verification context display

## Technical Decisions

### Parser Implementation

**Option A: Regex-based markdown parsing**
- Pros: Simple, no dependencies, fast
- Cons: Brittle, doesn't handle nested structures well

**Option B: Markdown AST parser (goldmark)**
- Pros: Robust, handles all markdown features, extensible
- Cons: Heavier dependency, more complex

**Recommendation:** Start with regex for Prior-Work tables (simple, well-structured), use goldmark if complexity grows.

### Tree Rendering

**Option A: Custom tree renderer**
- Pros: Full control, custom features
- Cons: Reinventing the wheel

**Option B: Existing library (go-tree-printer)**
- Pros: Battle-tested, saves time
- Cons: May not support all features we need

**Recommendation:** Custom renderer (it's ~100 lines for ASCII tree, gives us exact format from example).

### Cluster Detection

**Option A: Filesystem-only**
- Pros: Simple, explicit, no magic
- Cons: Requires manual directory organization

**Option B: Embedding-based clustering**
- Pros: Automatic, discovers hidden patterns
- Cons: Opaque, requires ML dependencies

**Recommendation:** Filesystem-first (explicit clusters exist), add embedding-based for orphans later if needed.

## Constraints & Validation

### Constraint 1: Work with Existing .kb/ Structure ✓
- **Validation:** No new metadata required, all relationships extracted from existing Prior-Work tables, Evidence sections, filesystem hierarchy
- **Test:** Run extraction on current .kb/ directory, verify all relationships found

### Constraint 2: Extract Relationships, Don't Require Manual Curation ✓
- **Validation:** 100% automated extraction, no manual tagging needed
- **Test:** Add new investigation with Prior-Work reference, verify it appears in tree without config changes

### Constraint 3: Tree IS Dylan's Primary Interface ✓
- **Validation:** Both CLI and web dashboard designed for browsing, not just visualization
- **Design principle:** Tree is not a "nice to have" debugging tool, it's the primary way Dylan understands project state
- **Test:** Dylan reviews tree instead of manually scanning .kb/ directory

## Open Questions

1. **Should tree show temporal ordering?** (newest first vs alphabetical)
   - Recommendation: Newest first for investigations, alphabetical for decisions/models
   
2. **How deep should default expansion go?** (1 level vs 2 vs fully expanded)
   - Recommendation: 2 levels (shows clusters and top-level artifacts)
   
3. **Should we cache the extracted tree?** (recompute on every render vs cache with invalidation)
   - Recommendation: Cache with invalidation on .kb/ file changes (performance optimization)
   
4. **Should beads issues be included in tree or separate view?** (integrated vs split)
   - Recommendation: Integrated (issues are outcomes of decisions, natural tree extension)

## Success Criteria

- [ ] Dylan can run `orch tree` and see knowledge landscape in <2 seconds
- [ ] Dashboard `/knowledge-tree` loads and renders full tree interactively
- [ ] Health smells correctly identify clusters needing synthesis
- [ ] New session Claudes can orient via tree before starting work
- [ ] Verification context shows where deliverable fits before explain-back
- [ ] Tree updates automatically when new artifacts added (no manual sync)

## References

- Example tree rendering from task description (resonated with Dylan)
- `.kb/investigations/` directory structure (existing clusters)
- `.kb/models/` directory structure (model + probes hierarchy)
- `pkg/verify/` code (patterns for artifact parsing)
- Beads integration guide (`.kb/guides/beads-integration.md`)

## Design Addendum: Two-View Model (from Dylan review, Feb 14)

### The Issue Role Question

Dylan asked: "Are issues secondary or on the same level as knowledge artifacts?"

**Answer:** Issues are the **active manifestation** of knowledge artifacts — the live edge of work. They're not leaf nodes hanging off decisions. They're where the energy is.

This leads to a **two-view model** — same data, same tree, different root:

### Knowledge View (`orch tree`)

Knowledge artifacts are primary. Issues hang off as "what this knowledge produced."
Question it answers: **"What do we understand?"**

```
orch-go knowledge tree
│
├─◉ Entropy Spiral
│ ├─◉ post-mortems/2026-01-02-system-spiral-dec27-jan02.md
│ ├─◉ investigations/2026-02-14-inv-entropy-spiral-deep-analysis.md
│ │ ├─★ decisions/2026-02-14-verifiability-first-hard-constraint.md
│ │ │ ├─● orch-go-5sc  Decision record                    CLOSED
│ │ │ ├─● orch-go-7jl  Daemon pause after N completions   CLOSED
│ │ │ └─◇ orch-go-6th  Skill system update                triage:review
│ │ └─● orch-go-agr  Trajectory audit                     IN PROGRESS
│ └─◉ handoffs/2026-02-13-entropy-spiral-recovery.md
```

### Work View (`orch tree --work`)

Issues are primary. Knowledge artifacts hang off as "where this work came from."
Question it answers: **"What are we doing and why?"**

This is likely the view Dylan opens most often — when the daemon has paused and he sits down to review.

```
orch-go work tree
│
├─⚡ NEEDS VERIFICATION (3)
│ ├─● orch-go-7jl  Daemon pause after N completions
│ │ └─ from ★ verifiability-first-hard-constraint
│ │    └─ from ◉ entropy-spiral-deep-analysis
│ ├─● orch-go-tyi  Explain-back verification gate
│ │ └─ from ★ verifiability-first-hard-constraint
│ └─● orch-go-8f7  Knowledge tree visualization
│   └─ from design session (Feb 14)
│
├─◇ TRIAGE:REVIEW (2)
│ ├─◇ orch-go-6th  Skill system update
│ │ └─ from ★ verifiability-first-hard-constraint
│ └─◇ orch-go-cem  Click freeze reactive capture
│
├─● IN PROGRESS (1)
│ └─● orch-go-agr  Trajectory audit
│   └─ from ◉ entropy-spiral-deep-analysis
│
└─░ QUEUED (7)
  ├─● orch-go-2qj  Model staleness detection
  ├─● orch-go-4tz  Completion accretion gate
  └─ ... 5 more
```

### Why Two Views

| View | When | Question | Primary nodes | Secondary nodes |
|------|------|----------|---------------|-----------------|
| Knowledge | Orientation, triage, synthesis | "What do we understand?" | Investigations, decisions, models | Issues as outcomes |
| Work | Review session, daemon pause, daily standup | "What are we doing and why?" | Issues grouped by state | Knowledge artifacts as provenance |

Same underlying data graph. Same extraction algorithm. Different traversal root and grouping logic.

### Impact on Implementation Phases

Phase 1 (CLI MVP) should include **both** views:
- `orch tree` — knowledge view (default)
- `orch tree --work` — work view (likely used more often)

The work view is cheap once the knowledge view exists — it's the same tree traversed from the other end.

### Dylan's Note

"This UI design could entirely transform how I work if we do this right."

The key is: **Dylan drives what 'right' means through usage.** CLI MVP ships, Dylan uses it, we iterate. This is the verifiability-first loop applied to its own tooling.

---

**Next Steps:**
1. ✅ Design reviewed with orchestrator — phased approach approved
2. ✅ Two-view model added (knowledge view + work view)
3. Start Phase 1: CLI MVP implementation (both views)
4. Test with current .kb/ structure
5. Iterate based on Dylan's feedback on resonance
