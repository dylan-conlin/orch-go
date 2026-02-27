# Probe: KB Context Query Derivation and Assembly Pipeline

**Model:** spawn-architecture
**Date:** 2026-02-27
**Status:** Complete

---

## Question

Model claims "KB context uses --global flag" (Invariant 3) and that SPAWN_CONTEXT.md "embeds skill content + task description + kb context". How is the kb context query derived from the task title, and does the matching algorithm produce relevant results for cross-domain tasks? Specifically: when an architect spawn targets "toolshed pricing KPI redesign", does the pipeline deliver pricing/KPI-relevant context or irrelevant infrastructure knowledge?

---

## What I Tested

### 1. Traced query derivation pipeline

The full pipeline from task title to kb context query:

```
orch spawn → runSpawnWithSkillInternal (spawn_cmd.go:484)
  → task = issue.Title (spawn_cmd.go:449)
  → GatherSpawnContext(task, ...) (extraction.go:807)
    → runPreSpawnKBCheckFull(task, ...) (extraction.go:1253)
      → ExtractKeywords(task, 3) (kbcontext.go:97-122)
        → stopword filter + first 3 words > 2 chars
      → RunKBContextCheckForDir(keywords, ...) (kbcontext.go:176)
        → runKBContextQuery(query, false)  [local first]
        → runKBContextQuery(query, true)   [global fallback if <3 matches]
      → FormatContextForSpawnWithLimitAndMeta(...) (kbcontext.go:683)
```

### 2. Tested keyword extraction for cross-domain tasks

```bash
# Go program testing spawn.ExtractKeywords()
# Task: "Redesign toolshed pricing KPI dashboard"
# Keywords(3): "redesign toolshed pricing"
# Keywords(1): "redesign"

# Task: "Architect pricing KPI redesign for toolshed"
# Keywords(3): "architect pricing kpi"
# Keywords(1): "architect"

# Task: "Investigate spawn context assembly"
# Keywords(3): "investigate spawn context"
# Keywords(1): "investigate"
```

### 3. Tested kb context matching with extracted keywords

```bash
kb context "redesign toolshed pricing"
# Result: 1 match → "Orchestrator Skill Orientation Redesign" (matches on "redesign")

kb context "architect pricing kpi"
# Result: constraints matching on "architect", decisions matching on "architect"

kb context "pricing kpi"
# Result: 1 match → "[global] Focus on Domain Translation, Not AI Infrastructure"
```

### 4. Traced matching algorithm in kb-cli

File: `kb-cli/internal/search/matcher.go` — Uses Snowball Porter2 stemming.
Matching is **per-keyword OR** (any single stemmed keyword matching ANY content token = match).

```go
// MatchWithStemming: returns true if ANY stemmed query token appears in stemmed content
for _, stemmedToken := range stemmedQuery {
    if stemmedContent[stemmedToken] {
        return true  // ANY match = included
    }
}
```

Scoring: title match = +10, filename match = +3, content match = +1/(n+1).

### 5. Tested how FRAME (OrientationFrame) is delivered

```
spawn_cmd.go:449-450:
  task := issue.Title                    # Short title → drives workspace name + kb query
  spawnOrientationFrame = issue.Description  # Full description → ORIENTATION_FRAME section

context.go:91-93 (template):
  TASK: {{.Task}}
  {{if .OrientationFrame}}
  ORIENTATION_FRAME: {{.OrientationFrame}}
  {{end}}
```

### 6. Checked if beads issue comments are included in spawn context

Default path (no `requires` in skill): **Comments are NOT included**.

The `gatherBeadsIssueContext` function (skill_requires.go:224) *does* include comments (last 5), but it's only called via the `requires.BeadsIssue` path — which requires the skill to declare `<!-- requires: beads-issue -->`. Standard skills don't.

FRAME is recorded as a beads comment at spawn time (extraction.go:1117), but it's read from the beads issue *description* field at spawn time, not from comments. The FRAME comment is only consumed during `orch complete` (completion.go:201) for the explain-back gate prompt.

---

## What I Observed

### Root Cause of the 66KB Irrelevant Context Bug

The pipeline has **three compounding failures** for cross-domain tasks (e.g., pricing KPI work spawned from orch-go):

1. **Query derived from task TITLE only** — `ExtractKeywords(task, 3)` uses the issue title, which for `orch work` is the beads issue title. The OrientationFrame (issue description with rich domain context) is never used for kb query construction.

2. **OR-based stemmed matching with no relevance gating** — `MatchWithStemming` returns true if ANY single keyword stem matches ANY token in the file. For a query like "architect pricing kpi", every file containing "architect" anywhere matches. The orch-go knowledge base has ~280 investigations, many mentioning "architect", so the query floods with orch-go infrastructure results.

3. **Local-first search in current project** — `RunKBContextCheckForDir` searches the current project first (line 178). If ≥3 matches found locally, it never searches globally. Since orch-go has extensive documentation matching on "architect"/"redesign"/"spawn" etc., the 3-match threshold is trivially met with irrelevant local results, and the cross-domain project (toolshed) is never even searched.

### OrientationFrame Not Used for Query

The OrientationFrame flows through the pipeline to SPAWN_CONTEXT.md (template line 93), but it is **never passed to ExtractKeywords or kb context**. The richer domain terms ("pricing", "KPI", "toolshed") in the description are invisible to the knowledge retrieval system.

### Beads Comments Absent from Default Path

Beads issue comments (including prior FRAME annotations, orchestrator notes, and Phase comments from previous rework attempts) are not included in SPAWN_CONTEXT.md unless the skill explicitly declares `<!-- requires: beads-issue -->`. The `gatherBeadsIssueContext` function exists and correctly fetches last 5 comments, but is gated behind skill requirements.

### Pipeline Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│ orch work / orch spawn                                              │
│   task = issue.Title        ← SHORT, used for kb query + workspace  │
│   orientationFrame = issue.Description  ← RICH, NOT used for query  │
└────────────────┬────────────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│ ExtractKeywords(task, 3)                                            │
│   Input:  "Architect pricing KPI redesign for toolshed"             │
│   Output: "architect pricing kpi"                                   │
│   Problem: "architect" matches 100+ orch-go artifacts               │
└────────────────┬────────────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│ RunKBContextCheckForDir(keywords, projectDir)                       │
│   Step 1: kb context "architect pricing kpi" (local, no --global)   │
│           → Matches on "architect" → gets orch-go infrastructure    │
│           → ≥3 matches → SKIPS global search entirely               │
│   Step 2: (skipped - local had ≥3 matches)                          │
│   Step 3: applyPerCategoryLimits (20 per category)                  │
└────────────────┬────────────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│ FormatContextForSpawnWithLimitAndMeta(result, 80000, ...)           │
│   Injects model summaries, probes, staleness warnings               │
│   Each model section up to 2,500 chars                              │
│   Total budget: 80k chars (~20k tokens)                             │
│   → 66KB of orch-go infrastructure knowledge                        │
└────────────────┬────────────────────────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│ SPAWN_CONTEXT.md Template Assembly                                  │
│   TASK: Architect pricing KPI redesign for toolshed                 │
│   ORIENTATION_FRAME: [description from beads issue]                 │
│   SPAWN TIER: full                                                  │
│   ## CONFIG RESOLUTION                                              │
│   ## PRIOR KNOWLEDGE (from kb context) ← 66KB irrelevant results    │
│   ## SKILL GUIDANCE                                                 │
│   [beads comments: NOT INCLUDED in default path]                    │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Model Impact

- [x] **Confirms** invariant 3: "KB context uses --global flag" — Confirmed, but only as fallback when local search returns <3 matches. The local-first strategy means global search is rarely triggered for projects with rich knowledge bases.
- [x] **Confirms** invariant 6: "Token estimation at 4 chars/token" — MaxKBContextChars = 80,000 chars ÷ 4 = 20k tokens, matches the model.
- [x] **Extends** model with: **Query is derived exclusively from task title** (Config.Task, which is beads issue title for `orch work`). OrientationFrame, beads comments, and issue description are NOT used for kb context query construction. This is a significant gap because:
  - Issue titles are concise (for workspace naming) and often contain generic action verbs ("architect", "redesign", "investigate") that match infrastructure-heavy keywords
  - Issue descriptions contain domain-specific terms that would produce better matches
  - The query does not consider the target project (--workdir) for cross-repo spawns
- [x] **Extends** model with: **Beads issue comments are only available via skill-declared `requires: beads-issue`** — The default spawn path does not include beads comments. The `gatherBeadsIssueContext` function exists but is only triggered by `ParseSkillRequires`.
- [x] **Extends** model with: **OR-based matching with no relevance scoring threshold** — kb context uses stemmed OR matching (any single keyword match = included). There's scoring but no minimum score threshold for inclusion, so a single-keyword match on "architect" in a 280-investigation knowledge base produces massive result sets.

---

## Notes

### Recommended Intervention Points (for architect follow-up)

1. **Query enrichment from OrientationFrame** (extraction.go:1258) — Use both task title AND orientation frame for keyword extraction. The frame contains domain-specific terms that are more discriminating.

2. **Mandatory conjunctive matching for multi-keyword queries** (kb-cli/internal/search/matcher.go:49) — Change from "ANY keyword matches" to "majority of keywords match" or add a minimum score threshold. Currently `MatchWithStemming` returns true on a single stemmed keyword hit.

3. **Cross-project query awareness** (kbcontext.go:176) — When `--workdir` is set, also run kb context in the target project directory to get domain-relevant knowledge, not just the orchestrator project's knowledge.

4. **Default beads comment inclusion** — Either add `requires: beads-issue` to standard skills (architect, investigation, feature-impl) or make beads comment inclusion the default when a beads ID is present.

5. **MinMatchesForLocalSearch threshold** (kbcontext.go:32, set to 3) — This threshold is too low. 3 single-keyword matches on "architect" in orch-go is trivially met, preventing global search expansion.
