# Decision: Git-Diff Staleness Detection for Model Artifacts

**Date:** 2026-02-14
**Status:** Proposed
**Enforcement:** context-only
**Context:** Models in `.kb/models/` describe system behavior but drift from code reality as refactoring occurs. 12+ stale file references found across 24 models. No existing mechanism detects model-code drift.
**Extends:** `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` (answers Open Question #3: "How do we prevent model drift?")

## Decision

Implement **Detect-Annotate-Queue** staleness detection for model artifacts:

1. **Detect:** At spawn time, check if model-referenced code files have changed since the model's `Last Updated` date using git history
2. **Annotate:** Prepend staleness warnings to served model sections so agents know which claims to verify
3. **Queue:** Surface stale models in `kb reflect --type model-drift` for periodic orchestrator review

## Problem

Models are the orchestrator's externalized understanding. When models reference code files that have been renamed, deleted, or significantly changed, they become stale. Stale models served via `kb context` at spawn time cause agents to act on outdated understanding.

**Evidence of the problem:**

| Stale Reference | What Happened |
|----------------|---------------|
| `pkg/dashboard/server.go` | Entire directory removed |
| `pkg/registry/` | Package deleted |
| `cmd/orch/complete.go` | Renamed to `complete_cmd.go` |
| `cmd/orch/spawn.go` | Renamed to `spawn_cmd.go` |
| `pkg/verify/phase.go` | Renamed to `phase_gates.go` |
| `pkg/verify/evidence.go` | Renamed to `test_evidence.go` |
| `pkg/beads/fallback.go`, `id.go`, `lifecycle.go` | Don't exist |

**Why existing mechanisms don't solve this:**
- `kb reflect --type stale` only checks decisions, not models
- Probes are manual and reactive (orchestrator must decide to run one)
- `orch complete` has no model validation gate
- `kb context` serves models without freshness check

## Design: Three Components

### Component 1: Structured Code References (code_refs)

Add machine-parseable file reference blocks to model files:

```markdown
**Primary Evidence (Verify These):**
<!-- code_refs: machine-parseable file references for staleness detection -->
- `cmd/orch/complete_cmd.go` - Completion orchestration pipeline
- `cmd/orch/complete_pipeline.go` - Phase functions with typed I/O
- `pkg/verify/check.go` - Verification gate implementation
<!-- /code_refs -->
```

**Format:**
- `<!-- code_refs: -->` / `<!-- /code_refs -->` HTML comment markers (invisible in rendered markdown)
- One backtick-quoted file path per line, relative to project root
- Description after the path (for humans) is ignored by parser
- File paths extracted via regex: `` `([^`]+\.\w+)` `` within the block

**Why this format:**
- Extends existing "Primary Evidence" section (no structural change)
- Self-describing (markers explain their purpose)
- Machine-parseable without changing visible markdown
- Backward-compatible (models without markers are simply skipped)

### Component 2: Spawn-Time Staleness Detection

When `kb context` serves a model to an agent, check freshness:

```
For each served model:
  1. Parse code_refs block → list of file paths
  2. Parse "Last Updated:" field → date
  3. For each referenced file:
     a. Check if file exists (deleted = definitely stale)
     b. Check git log --since={Last Updated} -- {file} (changed = potentially stale)
  4. If any files changed/deleted:
     Prepend staleness annotation to served model section
```

**Annotation format:**

```markdown
> **STALENESS WARNING:** This model was last updated 2026-01-12.
> Changed files: `cmd/orch/complete_cmd.go` (3 commits), `pkg/verify/check.go` (1 commit).
> Deleted files: `pkg/dashboard/server.go`.
> Verify model claims about these files against current code.
```

**Integration point:** `pkg/spawn/kbcontext.go` where models are formatted for SPAWN_CONTEXT.md.

**Performance:** Batch git queries. Check file existence first (stat, no git needed). Only run `git log` for files that exist. Expected <100ms for typical model with 5-10 refs.

### Component 3: Periodic Detection via kb reflect

Add `--type model-drift` to `kb reflect`:

```bash
kb reflect --type model-drift
```

Output:
```
MODEL DRIFT DETECTED
====================

1. completion-verification.md (Last Updated: 2026-01-15)
   Changed: cmd/orch/complete_cmd.go (7 commits since)
   Deleted: pkg/verify/cross_project.go
   Action: Review and update model

2. beads-integration-architecture.md (Last Updated: 2026-01-12)
   Deleted: pkg/beads/fallback.go, pkg/beads/id.go, pkg/beads/lifecycle.go
   Action: Major update needed - referenced files removed
```

**Cadence:** Runs when orchestrator invokes `kb reflect`. Not automated (respects Verification Bottleneck).

## Why Not Other Approaches

### Why Not Block on Staleness?
Models are approximately correct even with stale file references. A rename (`complete.go` → `complete_cmd.go`) doesn't invalidate the model's mechanism understanding. Blocking spawn on staleness would be too aggressive - the model still provides valuable context. Annotation lets the agent make informed judgments.

### Why Not Periodic-Only (No Spawn-Time Detection)?
Violates **Surfacing Over Browsing** principle. If staleness is only detected during periodic `kb reflect`, agents receive stale models without warning between sweeps. The harm happens at spawn time, so detection should happen there.

### Why Not Auto-Fix (Spawn Agent to Update Model)?
Violates **Understanding Through Engagement** principle and **Verification Bottleneck**. Models require orchestrator synthesis (cross-agent context). Auto-spawning update agents would create review obligations faster than orchestrators can process them. The right cadence: detect at spawn, review periodically, update during synthesis sessions.

### Why Not Git Hooks on Every Commit?
Most commits don't affect models. Running staleness checks on every commit creates noise and violates **Verification Bottleneck** (creates review obligations faster than they can be processed). Spawn-time detection is more targeted: only checks models that are actually being served.

### Why Not Bidirectional Linkage (Code → Model)?
Violates **Evidence Hierarchy** (code is primary, shouldn't carry metadata about secondary artifacts). Code comments referencing models would break on every refactor, creating a maintenance burden proportional to code change velocity.

## Implementation Plan

### Phase 1: Formalize code_refs Format (Convention)

**Scope:** Template update + convention documentation

1. Update `.kb/models/TEMPLATE.md` - add `<!-- code_refs: -->` markers to Primary Evidence section
2. Update `.kb/models/README.md` - add "Maintaining Code References" section
3. Backfill existing models with `code_refs` blocks (worker task, ~1 hour for 24 models)

**Acceptance criteria:** All 24 models have `code_refs` blocks with accurate file paths.

### Phase 2: Spawn-Time Detection (Code)

**Scope:** `pkg/spawn/kbcontext.go` modification + new helper

1. Add `extractCodeRefs(modelContent string) []string` - parse code_refs block
2. Add `extractLastUpdated(modelContent string) string` - parse Last Updated field
3. Add `checkModelStaleness(modelPath, projectDir string) (*StalenessResult, error)` - git log check
4. Integrate into model formatting path in `FormatKBContext()` or `kb context` injection
5. Add `HasStaleModels` field to `KBContextFormatResult`

**Acceptance criteria:**
- Models with changed referenced files get staleness annotation
- Models without code_refs blocks are served unchanged (graceful degradation)
- Spawn time doesn't increase by more than 200ms

### Phase 3: kb reflect Integration (kb-cli)

**Scope:** `kb reflect` command extension

1. Add `--type model-drift` detection type
2. Scan all `.kb/models/*.md` for code_refs blocks
3. Check each referenced file against git history since Last Updated
4. Report stale models with specific changed/deleted files
5. Optional: `--create-issue` flag to auto-create beads review issues

**Acceptance criteria:**
- `kb reflect --type model-drift` reports models with changed referenced files
- Output includes actionable information (which files changed, how many commits)
- Cross-repo: CROSS_REPO_ISSUE for kb-cli

## Trade-offs

| Trade-off | Accepted Because |
|-----------|-----------------|
| False positives (file changed but model still accurate) | Annotation is informational, not blocking. Over-surfacing > missing drift. |
| Backfill cost (24 models need code_refs) | One-time cost. Can be parallelized across worker agents. |
| File-level detection (not semantic) | Semantic detection is expensive and fragile. File-level is reliable and fast. |
| No auto-fix | Models require orchestrator synthesis. Auto-fix would violate Understanding Through Engagement. |
| Git dependency | Already have git. No additional infrastructure. |

## Principles Applied

| Principle | How Applied |
|-----------|------------|
| **Evidence Hierarchy** | Checks code reality (git history), not just metadata (Last Updated date) |
| **Gate Over Remind** | Staleness annotation is a soft gate - agent sees warning in context, not just periodic report |
| **Infrastructure Over Instruction** | Automated detection (infrastructure) replaces "remember to check models" (instruction) |
| **Surfacing Over Browsing** | Staleness surfaced at consumption time (spawn), not requiring navigation |
| **Capture at Context** | Detection happens when model is being served (maximum context relevance) |
| **Verification Bottleneck** | Annotate + queue (bounded obligations), not auto-fix (unbounded obligations) |
| **Self-Describing Artifacts** | code_refs markers explain their purpose within the model file |
| **Provenance** | Strengthens model → code provenance chain with machine-parseable linkage |

## Success Criteria

**Short-term (Phase 1-2, within 2 weeks):**
- [ ] All 24 models have code_refs blocks
- [ ] Spawn-time detection annotates stale models
- [ ] Zero regression in spawn time (< 200ms additional)

**Medium-term (Phase 3, within 1 month):**
- [ ] `kb reflect --type model-drift` operational
- [ ] Stale model count decreasing (review queue being processed)
- [ ] Agents reference staleness warnings in their work

**Long-term (3 months):**
- [ ] Model staleness rate < 20% (currently ~50%)
- [ ] Recurring staleness pattern eliminated (renames caught at spawn time)
- [ ] Model accuracy improves (agents verify stale claims, surface corrections)

## Decision Gate Guidance (if promoting)

**Add blocks: frontmatter when:**
- Future agents might modify code files without updating referencing models
- Future agents might create models without code_refs blocks

**Suggested blocks keywords:**
- "model staleness", "model drift", "code_refs", "kb models"

## Open Questions

1. **Where should staleness check live?** In orch-go's `kbcontext.go` (when formatting context) or in kb-cli's `kb context` command (upstream)? Recommendation: orch-go first (faster iteration), migrate to kb-cli later.

2. **Should orch complete validate model updates?** When an agent modifies code referenced by a model, should `orch complete` warn that a model may need updating? Deferred - requires tracking which files were changed in the agent's commits.

3. **Cross-repo models?** Some models reference files in other repos (e.g., opencode-fork.md). How should cross-repo staleness work? Deferred - current models are primarily intra-repo.

## Related Decisions

- `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` - Foundational decision this extends
- `.kb/decisions/2026-01-07-synthesis-is-strategic-orchestrator-work.md` - Why auto-fix is wrong (synthesis requires orchestrator)

## Investigation

- `.kb/investigations/2026-02-14-inv-design-solution-model-artifact-staleness.md` - Full analysis with evidence
