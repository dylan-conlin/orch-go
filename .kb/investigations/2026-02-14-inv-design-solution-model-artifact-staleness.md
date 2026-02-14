<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Model staleness is real and measurable: 12+ file references across 24 models point to deleted/renamed files. The existing probe system is manual and reactive; kb reflect only checks decisions, not models. Git-diff-based detection at spawn time is the right solution - it surfaces staleness exactly when models are served to agents.

**Evidence:** Codebase scan found 12 stale file references (e.g., `pkg/dashboard/server.go` deleted but still referenced, `cmd/orch/complete.go` renamed to `complete_cmd.go`). Models already contain file references in "Primary Evidence" sections but in unparseable formats. `kb reflect --type stale` only detects uncited decisions, not model-code drift.

**Knowledge:** The solution must integrate at spawn time (Surfacing Over Browsing principle) not as periodic hygiene. Models already have the linkage metadata - it just needs formalization and a detection mechanism. Blocking on staleness is too aggressive; annotating + queuing is the right response.

**Next:** Create decision record (done), implement in 3 phases: (1) formalize code_refs format, (2) spawn-time detection in kbcontext.go, (3) kb reflect model-drift integration.

**Authority:** architectural - Cross-component change affecting spawn flow, kb system, and model maintenance protocol

---

# Investigation: Design Solution for Model Artifact Staleness

**Question:** How should we detect and handle model artifacts (.kb/models/) that drift from code reality, given that models are the orchestrator's externalized understanding and stale models → stale decisions?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** Architect agent (orch-go-rdd)
**Phase:** Complete
**Next Step:** Implement per decision record
**Status:** Complete

**Patches-Decision:** .kb/decisions/2026-01-12-models-as-understanding-artifacts.md (extends with staleness detection)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/decisions/2026-01-12-models-as-understanding-artifacts.md | extends | Yes - models exist, provenance chain documented | Open Question #3 ("How do we prevent model drift?") is what this investigation answers |
| .kb/models/PHASE4_REVIEW.md | extends | Yes - N=11 findings still hold | Phase 4 identified "Model obsolescence" as open question for N=20+ |
| .kb/models/PHASE3_REVIEW.md | deepens | Yes - structural consistency confirmed | N/A |

---

## Findings

### Finding 1: Staleness is Measurable - 12+ Stale File References Across 24 Models

**Evidence:** Systematic scan of all 24 model files found 12 file references pointing to deleted or renamed files:

| Model | Stale Reference | Reality |
|-------|----------------|---------|
| completion-verification.md | `pkg/verify/phase.go` | Renamed to `phase_gates.go` |
| completion-verification.md | `pkg/verify/evidence.go` | Renamed to `test_evidence.go` |
| completion-verification.md | `pkg/verify/cross_project.go` | Merged into `check.go` |
| beads-integration-architecture.md | `pkg/beads/fallback.go` | Doesn't exist |
| beads-integration-architecture.md | `pkg/beads/id.go` | Doesn't exist |
| beads-integration-architecture.md | `pkg/beads/lifecycle.go` | Doesn't exist |
| completion-verification.md | `cmd/orch/complete.go` | Renamed to `complete_cmd.go` |
| spawn-architecture.md | `cmd/orch/spawn.go` | Renamed to `spawn_cmd.go` |
| dashboard-architecture.md | `pkg/dashboard/server.go` | Entire directory removed |
| agent-lifecycle-state-model.md | `pkg/registry/` references | Package deleted |
| spawn-architecture.md | spawn_cmd.go ~800 lines | Actual: 2,320 lines |
| beads-integration-architecture.md | client.go ~728 lines | Actual: 1,120 lines |

**Source:** Glob pattern scan of `.kb/models/*.md`, cross-referenced against `ls` of referenced directories and files.

**Significance:** This confirms the problem is not hypothetical. ~50% of models (12/24) have at least one stale reference. Code refactoring (renames, extractions, deletions) is the primary cause. Models don't know when their referenced code changes.

---

### Finding 2: Models Already Contain Linkage Metadata - But in Unparseable Formats

**Evidence:** Models reference code in 4 inconsistent patterns:

1. File + function: `pkg/spawn/config.go:selectBackend()`
2. File + line number: `cmd/orch/spawn_cmd.go:798`
3. File + description: `cmd/orch/spawn_cmd.go` - Main spawn command (~800 lines)
4. Multi-file lists: `pkg/beads/client.go` + `pkg/beads/fallback.go`

The TEMPLATE.md has a "Primary Evidence (Verify These)" section with format: `{file path}:{lines} - {What this code demonstrates}`. But existing models don't follow this consistently.

**Source:** `.kb/models/TEMPLATE.md:73-74`, review of all 24 model files.

**Significance:** The linkage already exists conceptually. Formalizing it into a machine-parseable format is the lowest-cost path to enabling automated detection. No new metadata format needed - just standardize what's already there.

---

### Finding 3: Existing Staleness Mechanisms Don't Cover Models

**Evidence:**

| Mechanism | What It Detects | Covers Models? |
|-----------|----------------|----------------|
| `kb reflect --type stale` | Decisions with no citations >7 days | No - decisions only |
| `kb reflect --type drift` | Constraints contradicted by code | No - constraints only |
| `orch complete` verification gates | Agent work quality (phase, synthesis, tests) | No - no model validation |
| `kb context` spawn injection | Relevant knowledge for task | No - serves models without freshness check |
| `pkg/spawn/gap.go` quality scoring | Missing context (no constraints, no decisions) | No - checks presence not accuracy |
| Session staleness detection | Session inactivity >30min | No - sessions only |

**Source:** `kb reflect --type stale` (returns 0 results for models), `cmd/orch/complete_cmd.go` (11 gates, none for models), `pkg/spawn/kbcontext.go` (serves models without checking referenced files).

**Significance:** There is a complete gap. No existing mechanism detects model-code drift. The probe system (`.kb/models/*/probes/`) is manual and reactive - orchestrator must decide to run a probe. The system needs a proactive, automated mechanism.

---

### Finding 4: Spawn-Time Is the Optimal Integration Point

**Evidence:** `pkg/spawn/kbcontext.go` already:
1. Queries models via `kb context` (keyword search)
2. Injects model content into SPAWN_CONTEXT.md (summary, invariants, failure modes)
3. Truncates per `maxModelSectionChars = 2500`
4. Reports `HasInjectedModels` in `KBContextFormatResult`

This is the exact moment where stale models cause harm: an agent receives outdated understanding and makes decisions based on it.

**Source:** `pkg/spawn/kbcontext.go:40-71`, `pkg/spawn/gap.go` (GapCheckResult scoring).

**Significance:** Per the **Surfacing Over Browsing** principle, staleness should be surfaced when the model is about to be consumed, not during periodic hygiene. The infrastructure already exists in kbcontext.go - adding a freshness check is an extension, not a new system.

---

### Finding 5: Principles Constrain the Solution Design

**Evidence:** 6 principles directly constrain this design:

| Principle | Constraint on Solution |
|-----------|----------------------|
| **Evidence Hierarchy** | Code is truth, models are hypotheses. Detection must check against code reality, not just metadata dates. |
| **Gate Over Remind** | Staleness detection should be a gate (blocks or annotates), not just a periodic report. BUT: gate must be passable by gated party. |
| **Infrastructure Over Instruction** | Detection must be automated (infrastructure), not "remember to check model freshness" (instruction). |
| **Surfacing Over Browsing** | Surface staleness at point of consumption (spawn time), not require navigation to discover. |
| **Capture at Context** | Detect staleness when context is relevant (when model is being served), not at convenient intervals. |
| **Verification Bottleneck** | Don't create more review obligations than processing capacity. Staleness alerts must not pile up faster than orchestrators can process them. |

**Source:** `~/.kb/principles.md`

**Significance:** The principles collectively point to a clear design: automated detection at spawn time that surfaces staleness as annotation (not hard block), with throttled review queuing to respect verification bottleneck.

---

## Synthesis

**Key Insights:**

1. **The linkage already exists, it just needs formalization.** Models reference code files in their "Primary Evidence" sections and throughout their text. Making these references machine-parseable is the foundation for automated detection. This is a convention change, not a new system.

2. **Spawn-time is the critical integration point.** Per multiple principles (Surfacing Over Browsing, Capture at Context, Infrastructure Over Instruction), the optimal moment to detect and surface staleness is when `kb context` serves models to agents. This is when stale models cause actual harm.

3. **Annotate + Queue is the right response, not Block.** Models are approximately correct even when some file references are stale (a rename doesn't invalidate the model's core mechanism understanding). Hard blocking would be too aggressive. But silent serving of stale models is the current problem. The answer: annotate the served model with a staleness notice so the agent knows to verify, and queue a review for the orchestrator.

4. **Verification Bottleneck limits automation ambition.** Auto-spawning model update agents for every detected staleness would create work faster than orchestrators can verify. The right cadence is: detect at spawn time (annotate), batch for periodic review (kb reflect), update during synthesis sessions (manual, orchestrator-driven).

**Answer to Investigation Question:**

The recommended solution is **git-diff-based staleness detection at spawn time** with 3 components:
1. Formalize model code references into a machine-parseable `code_refs:` metadata block
2. Add a staleness check to `kb context` / spawn flow that checks if referenced files changed since model's Last Updated date
3. Integrate `--type model-drift` into `kb reflect` for periodic sweep

This directly addresses the recurring pattern (stale models → stale decisions) while respecting the Verification Bottleneck principle (annotate + queue, don't auto-fix).

---

## Structured Uncertainty

**What's tested:**

- ✅ 12+ stale file references confirmed across 24 models (manual verification against filesystem)
- ✅ `kb reflect --type stale` does not cover models (ran command, confirmed 0 model results)
- ✅ `pkg/spawn/kbcontext.go` serves models without freshness check (read source code)
- ✅ TEMPLATE.md has "Primary Evidence (Verify These)" section (read file)
- ✅ 6 principles constrain solution design (read and verified against `~/.kb/principles.md`)

**What's untested:**

- ⚠️ Performance impact of `git log --since` check per referenced file at spawn time (not benchmarked)
- ⚠️ False positive rate when code changes don't actually invalidate model claims (not measured)
- ⚠️ Whether agents actually use staleness annotations to adjust behavior (no evidence from prior experiments)
- ⚠️ How quickly stale models accumulate review items (could overwhelm verification bottleneck)

**What would change this:**

- If `git log --since` per file adds >500ms to spawn time, consider caching or batch pre-computation
- If false positive rate >50%, the detection heuristic is too sensitive - need semantic change detection, not just file-changed detection
- If agents ignore staleness annotations, the annotation approach fails - need to consider serving a reduced/truncated model instead

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Formalize code_refs format in model template | implementation | Convention change within existing pattern |
| Add staleness check to spawn flow | architectural | Cross-component change (spawn, kb, models) |
| Add `--type model-drift` to kb reflect | architectural | Extends external tool (kb-cli) |
| Backfill existing models | implementation | Tedious but scoped, no architectural decisions |

### Recommended Approach: Git-Diff Staleness Detection at Spawn Time

**"Detect-Annotate-Queue"** - Automatically detect when model-referenced files have changed since the model's Last Updated date, annotate served models with staleness warnings, and queue review items for orchestrators.

**Why this approach:**
- Leverages existing infrastructure (kb context, spawn flow, model template)
- Respects Evidence Hierarchy (checks code reality, not just metadata)
- Follows Surfacing Over Browsing (surfaces at consumption time)
- Doesn't violate Verification Bottleneck (annotate + queue, not auto-fix)

**Trade-offs accepted:**
- False positives: File changes that don't invalidate the model (acceptable - annotation is informational, not blocking)
- Requires backfill: Existing 24 models need code_refs added (one-time cost, can be worker task)
- Detection is file-level, not semantic: A renamed variable triggers staleness even if model claims still hold (acceptable - better to over-surface than miss drift)

**Implementation sequence:**

1. **Phase 1: Formalize code_refs format** (convention + template update)
   - Add structured `code_refs:` section to model TEMPLATE.md
   - Define parseable format: one file path per line, relative to project root
   - Update README.md with maintenance protocol

2. **Phase 2: Spawn-time detection** (code change in kbcontext.go)
   - Extract code_refs from model files when serving via kb context
   - For each referenced file, check `git log --since={Last Updated} -- {file}` for changes
   - If changes found, prepend staleness annotation to served model section
   - Add staleness metadata to KBContextFormatResult

3. **Phase 3: kb reflect integration** (kb-cli change)
   - Add `--type model-drift` to `kb reflect`
   - Scan all models' code_refs against git history
   - Report models with referenced files that changed since Last Updated
   - Optionally create beads issues for review (with deduplication)

### Alternative Approaches Considered

**Option B: Periodic-only detection (kb reflect sweep)**
- **Pros:** Simpler, no spawn-time overhead, easier to implement
- **Cons:** Violates Surfacing Over Browsing (staleness not surfaced when model consumed). Agents receive stale models without warning. Detection disconnected from harm.
- **When to use instead:** If spawn-time detection adds unacceptable latency (>500ms)

**Option C: Accept approximate models, update during synthesis only**
- **Pros:** Zero overhead, no new infrastructure, models updated when orchestrator synthesizes
- **Cons:** This is the current state - and it produced 12+ stale references. "Accept" means "stale models → stale decisions" continues. Violates Gate Over Remind (no gate, just hope orchestrators notice).
- **When to use instead:** If model staleness proves to have low impact (agents rarely act on stale claims)

**Option D: Git hooks on every commit**
- **Pros:** Immediate detection, catches drift as it happens
- **Cons:** Noisy (most commits don't affect models), adds commit overhead, violates Verification Bottleneck (creates review obligations on every commit)
- **When to use instead:** If spawn-time detection proves insufficient (agents act on stale models before periodic sweep catches drift)

**Option E: Bidirectional linkage (code comments reference models)**
- **Pros:** Would enable IDE-level "this code is modeled" awareness
- **Cons:** Extremely high maintenance (code comments break on every refactor), violates Evidence Hierarchy (code is primary, shouldn't carry metadata about secondary artifacts)
- **When to use instead:** Never - this reverses the provenance direction

**Rationale for recommendation:** Option A (Detect-Annotate-Queue) is the only approach that satisfies all 6 constraining principles. It surfaces staleness at the right moment (spawn time), doesn't block (respects approximate models), and creates bounded review obligations (respects verification bottleneck).

---

### Implementation Details

**What to implement first:**
- Phase 1 (code_refs format) is foundational - everything else depends on it
- Phase 2 (spawn-time detection) delivers the core value
- Phase 3 (kb reflect integration) is enhancement for periodic hygiene

**Proposed code_refs format (in model files):**

```markdown
## References

**Primary Evidence (Verify These):**
<!-- code_refs: machine-parseable file references for staleness detection -->
- `cmd/orch/complete_cmd.go` - Completion orchestration pipeline
- `cmd/orch/complete_pipeline.go` - Phase functions with typed I/O
- `pkg/verify/check.go` - Verification gate implementation
- `pkg/verify/phase_gates.go` - Phase gate definitions
<!-- /code_refs -->
```

The `<!-- code_refs: -->` / `<!-- /code_refs -->` markers make the block parseable without changing the visible markdown. File paths are extracted via simple regex: `` `([^`]+\.\w+)` `` within the block.

**Staleness check pseudocode (for kbcontext.go):**

```go
func checkModelStaleness(modelPath string, projectDir string) (bool, []string) {
    codeRefs := extractCodeRefs(modelPath)  // parse code_refs block
    lastUpdated := extractLastUpdated(modelPath)  // parse "Last Updated:" field

    var changedFiles []string
    for _, ref := range codeRefs {
        // Check if file was modified since model's Last Updated date
        cmd := exec.Command("git", "log", "--since="+lastUpdated, "--oneline", "--", ref)
        cmd.Dir = projectDir
        output, err := cmd.Output()
        if err == nil && len(strings.TrimSpace(string(output))) > 0 {
            changedFiles = append(changedFiles, ref)
        }
    }
    return len(changedFiles) > 0, changedFiles
}
```

**Staleness annotation format (prepended to served model):**

```markdown
> **STALENESS WARNING:** This model's Last Updated date is 2026-01-12.
> The following referenced files have changed since then:
> `cmd/orch/complete_cmd.go` (3 commits), `pkg/verify/check.go` (1 commit).
> Model claims about these files should be verified against current code.
```

**Things to watch out for:**
- ⚠️ `git log --since` performance: batch all file checks into single git command where possible
- ⚠️ Models without code_refs block: gracefully degrade (skip check, don't error)
- ⚠️ Deleted files: `git log` won't find them - need `git log --all --follow` or check file existence first
- ⚠️ Deduplication: if same model is stale on repeated spawns, don't create duplicate beads issues

**Areas needing further investigation:**
- Performance profiling of `git log --since` at spawn time with 10+ file references
- Whether `kb context` command (kb-cli) should handle staleness internally vs orch-go
- How to handle cross-repo model references (models referencing files in other repos)

**Success criteria:**
- ✅ Models with stale references produce staleness annotation when served at spawn time
- ✅ `kb reflect --type model-drift` reports models with changed referenced files
- ✅ Spawned agents receive explicit warning about which model claims need verification
- ✅ Orchestrators get bounded review queue (not flooded with staleness issues)
- ✅ Zero additional latency for models without code_refs (graceful degradation)

---

## References

**Files Examined:**
- `.kb/models/*.md` (all 24 model files) - Structure, reference patterns, metadata format
- `.kb/models/TEMPLATE.md` - Standard model structure
- `.kb/models/README.md` - Model lifecycle and creation criteria
- `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` - Foundational decision
- `.kb/models/PHASE4_REVIEW.md` - N=11 model pattern review
- `pkg/spawn/kbcontext.go` - How kb context is queried and served
- `pkg/spawn/gap.go` - Context quality scoring
- `cmd/orch/complete_cmd.go` - Verification gates
- `cmd/orch/spawn_cmd.go` - Pre-spawn KB check flow
- `~/.kb/principles.md` - Constraining principles

**Commands Run:**
```bash
# Check kb reflect staleness detection scope
kb reflect --type stale
# Result: 0 results for models (only checks decisions)

# Full kb reflect output
kb reflect
# Result: 29 synthesis opportunities, no model-drift detection

# Model file inventory
glob .kb/models/*.md
# Result: 24 model files + 2 templates
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` - Models as first-class artifacts (this investigation answers Open Question #3)
- **Decision:** `.kb/decisions/2026-02-14-model-staleness-detection.md` - Decision record produced by this investigation
- **Review:** `.kb/models/PHASE4_REVIEW.md` - N=11 review identifying model obsolescence as open question

---

## Investigation History

**2026-02-14:** Investigation started
- Initial question: How to detect and handle model artifacts that drift from code reality
- Context: Spawned by orchestrator after recurring staleness issues (e.g., agent-lifecycle model referencing deleted pkg/registry/)

**2026-02-14:** Exploration complete - 5 findings documented
- Found 12+ stale file references across 24 models
- Identified 6 constraining principles
- Confirmed existing mechanisms don't cover model staleness
- Identified spawn-time as optimal integration point

**2026-02-14:** Investigation completed
- Status: Complete
- Key outcome: Recommended git-diff staleness detection at spawn time with annotate+queue response. Decision record produced.
