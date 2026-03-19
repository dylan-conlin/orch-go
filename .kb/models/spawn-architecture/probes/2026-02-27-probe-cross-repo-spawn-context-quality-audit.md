# Probe: Cross-Repo Spawn Context Quality Audit (toolshed, price-watch)

**Model:** spawn-architecture
**Date:** 2026-02-27
**Status:** Complete

---

## Question

When the daemon spawns agents for cross-repo projects (toolshed, price-watch), does the injected kb context contain knowledge from the TARGET project or from the SPAWNER's project (orch-go)? The Feb 25 probe fixed group resolution for global search filtering, but does the local search (step 1) still use the spawner's CWD?

---

## What I Tested

### Test 1: Compare `kb context` output from different CWDs

```bash
# From orch-go CWD (daemon's perspective)
$ kb context "pricing strategy"
# Returns: orch-go guides (Model Selection, Two-Tier Sensing, Web Layout Breakpoints)
# Returns: orch-go decisions (global only)
# Returns: ZERO toolshed knowledge

# From toolshed CWD (correct target)
$ cd ~/Documents/work/WorkCorp/work-monorepo/toolshed
$ kb context "pricing strategy"
# Returns: toolshed decisions (SSE streaming for AI pricing panel)
# Returns: toolshed models (Toolshed ↔ Price Watch Integration, Toolshed Architecture)
# Returns: relevant global decisions
```

### Test 2: Read real SPAWN_CONTEXT.md from cross-repo toolshed spawns

Examined 3 recent toolshed spawns from daemon (all Feb 27):
- `to-feat-ai-pricing-strategy-27feb-c64d` (toolshed-74)
- `to-feat-fix-playwright-mcp-27feb-ee5b` (toolshed-jcx)
- `to-feat-add-claude-anthropic-27feb-682c` (toolshed-98)

### Test 3: Trace `runKBContextQuery` code path

```go
// kbcontext.go:250 - NO cmd.Dir set, uses process CWD
func runKBContextQuery(query string, global bool) (*KBContextResult, error) {
    var cmd *exec.Cmd
    if global {
        cmd = exec.CommandContext(ctx, "kb", "context", "--global", query)
    } else {
        cmd = exec.CommandContext(ctx, "kb", "context", query)
    }
    // NOTE: cmd.Dir is never set — uses process CWD
    output, err := cmd.Output()
}
```

### Test 4: Trace the step 1 → step 2 skip logic

```go
// kbcontext.go:200-230
func RunKBContextCheckForDir(query string, projectDir string) {
    // Step 1: LOCAL search (no --global) — runs from CWD, NOT projectDir
    result, err := runKBContextQuery(query, false)

    // Step 2: ONLY expands to global if local returned < 3 matches
    if result == nil || len(result.Matches) < MinMatchesForLocalSearch {
        globalResult, err := runKBContextQuery(query, true)
        // Group filter applied here — uses projectDir ✅ (Feb 25 fix)
        allowlist := resolveProjectAllowlistForDir(projectDir)
    }
}
```

### Test 5: Verify gap analysis scores from events.jsonl

```
toolshed-74: gap_context_quality: 90, gap_has_gaps: false
toolshed-jcx: gap_context_quality: 75, gap_has_gaps: true, gap_types: ["no_constraints"]
toolshed-98: gap_context_quality: 95, gap_has_gaps: false
```

### Test 6: Verify verification gate cross-repo issue

From events.jsonl, toolshed-128 (cross-repo fix in price-watch):
```json
{"type":"verification.failed","data":{
  "errors":["SYNTHESIS.md claims 2 local file(s) not in git diff:","  - price-watch/backend/app/controllers/..."],
  "gates_failed":["architectural_choices","git_diff"]
}}
```
Verification git_diff gate checked toolshed repo but changes were in price-watch.

---

## What I Observed

### Finding 1: Local KB search uses spawner CWD — WRONG project knowledge injected

The `runKBContextQuery` function (kbcontext.go:250) never sets `cmd.Dir`. When daemon runs from orch-go and spawns toolshed agents:

1. **Step 1 (local search)** runs `kb context "pricing strategy"` from orch-go CWD
2. Returns orch-go knowledge (Model Selection Guide, Dashboard Architecture, etc.)
3. If ≥3 matches found (common), **Step 2 is never reached**
4. The Feb 25 group-filtering fix in Step 2 is bypassed entirely

### Finding 2: Real toolshed SPAWN_CONTEXT.md contains orch-go knowledge

`to-feat-ai-pricing-strategy-27feb-c64d/SPAWN_CONTEXT.md` for toolshed-74:
- **Lines 53-59**: Orch-go constraints and decisions ("Dashboard event panels max-h-64", "Slide-out panel for agent card detail view")
- **Lines 69-131**: Orch-go probes (spawn-architecture, orchestration-cost-economics)
- **Lines 113-178**: Full orch-go Dashboard Architecture model with invariants and failure modes
- **Zero toolshed knowledge** (no Toolshed ↔ Price Watch Integration model, no toolshed architecture, no pricing decisions)

An agent building a pricing strategy panel for toolshed received knowledge about orch-go's dashboard connection pool and progressive disclosure instead.

### Finding 3: Gap analysis scores are misleading

Gap analysis checks that CATEGORIES are populated (has constraints? has models? has decisions?) — not whether the content is from the correct project. toolshed-74 scored 90% quality despite 100% wrong-project knowledge injection.

### Finding 4: Verification git_diff gate doesn't handle cross-repo changes

toolshed-128 made changes in the price-watch repo to fix a bug, but verification only checked git diff in toolshed. Required manual bypass.

---

## Model Impact

- [x] **Extends** model with: `runKBContextQuery` has a critical CWD bug for cross-repo spawns. The Feb 25 fix (probe: cross-project-kb-context-group-resolution) correctly threaded `projectDir` through the group resolution chain for GLOBAL search filtering (Step 2), but `runKBContextQuery` itself never received `projectDir` to set `cmd.Dir`. When the local search (Step 1) returns ≥3 results from the WRONG project, Step 2 is never reached and the group fix is bypassed entirely. Real production spawns confirm: 3 toolshed agents spawned Feb 27 received orch-go Dashboard Architecture knowledge instead of Toolshed Architecture knowledge.
- [x] **Extends** model with: Gap analysis (context quality scoring) measures category population, not content relevance. Spawns can score 90-95% quality while containing 100% wrong-project knowledge.

---

## Recommended Fix

### Fix 1: Thread `projectDir` to `runKBContextQuery` (Critical)

```go
func runKBContextQuery(query string, global bool, projectDir string) (*KBContextResult, error) {
    var cmd *exec.Cmd
    if global {
        cmd = exec.CommandContext(ctx, "kb", "context", "--global", query)
    } else {
        cmd = exec.CommandContext(ctx, "kb", "context", query)
    }
    if projectDir != "" {
        cmd.Dir = projectDir  // Search target project's .kb/, not spawner's
    }
    output, err := cmd.Output()
}
```

Update `RunKBContextCheckForDir` to pass `projectDir` to both Step 1 and Step 2:
```go
result, err := runKBContextQuery(query, false, projectDir)  // Step 1
globalResult, err := runKBContextQuery(query, true, projectDir)  // Step 2
```

### Fix 2: Add project-source tagging to gap analysis (Nice-to-have)

Gap analysis should verify that injected knowledge matches the target project, not just that categories are populated.

### Fix 3: Cross-repo verification awareness (Nice-to-have)

Verification git_diff gate should accept `cross_repo_dirs` in workspace manifest to check additional project directories.

---

## Notes

- The Feb 25 probe identified the correct bug class but fixed only one layer (group filtering). The CWD dependency in `runKBContextQuery` is the root cause.
- Priority: Critical for cross-repo spawn quality. Every cross-repo toolshed/price-watch spawn is currently receiving wrong knowledge.
- The `scs` group in groups.yaml uses `parent: work-monorepo` which correctly maps toolshed and price-watch as siblings. Once `cmd.Dir` is set, the local search will find toolshed's .kb/ knowledge.
- The gap analysis false-positive (high quality score on wrong-project knowledge) means operators have no signal that spawns are degraded.
