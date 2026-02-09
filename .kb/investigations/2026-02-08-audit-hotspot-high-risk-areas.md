<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Hotspot output is diluted by artifact/test-pattern noise while a smaller set of dual-signal files represent the true architectural risk.

**Evidence:** `orch hotspot --json` reported 66 hotspots, including 11 generated/artifact paths and two confirmed dual-signal production hotspots (`cmd/orch/spawn_cmd.go`, `web/src/lib/components/work-graph-tree/work-graph-tree.svelte`).

**Knowledge:** Exclusion behavior is structurally limited (exact/suffix only, no directory pattern support, defaults replaced when `--exclude` is passed), so operators cannot reliably suppress noisy classes of paths.

**Next:** Implement hotspot filtering fixes first, then run architect-led decomposition for the top dual-signal files.

**Authority:** architectural - Follow-ups cross multiple subsystems (hotspot engine + core orchestration + web graph surface) and require coordinated sequencing.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Hotspot High Risk Areas

**Question:** Which hotspot results are highest-risk and actionable right now, and what targeted follow-up issues should be filed?

**Started:** 2026-02-08
**Updated:** 2026-02-08
**Owner:** Claude (spawned worker)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** `.kb/decisions/2026-01-30-bloat-control-enforcement-patterns.md`
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/archived/2026-01-17-inv-design-800-line-bloat-gate.md` | extends | yes | none |
| `.kb/investigations/archived/2026-01-17-inv-implement-bloat-size-hotspot-type.md` | deepens | yes | none |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Artifact paths consume a meaningful portion of hotspot output

**Evidence:** Current run reports 66 hotspots total; 11 match generated/artifact roots (`web/.svelte-kit`, `web/playwright-report`, `build/`, `.kb/`) and therefore do not represent maintainability hotspots in hand-maintained source.

**Source:** Command: `orch hotspot --json`; command: `jq '{total_hotspots, generated_or_artifact_hotspots}' /tmp/hotspots.json`; output: `total_hotspots: 66`, `generated_or_artifact_hotspots: 11`.

**Significance:** ~17% of results are low-actionability noise, reducing signal quality and pulling attention from true architectural risk.

---

### Finding 2: Exclusion semantics cannot express common hotspot filtering intent

**Evidence:** `--exclude "web/.svelte-kit/*"` fails to remove `.svelte-kit` hotspots; matcher implementation only supports exact path or `*suffix` checks. Passing any `--exclude` value also replaces default exclusions, which changed output from 66 to 68 in testing.

**Source:** `cmd/orch/hotspot.go:229` (`matchesExclusionPattern`), `cmd/orch/hotspot.go:73` (`StringSliceVar` default wiring), command: `orch hotspot --json --exclude "web/.svelte-kit/*"`, command: `orch hotspot --json --exclude "web/.svelte-kit/output/server/entries/pages/_page.svelte.js"`.

**Significance:** Operators cannot reliably prune path classes (directory/prefix) and may accidentally drop defaults, creating unstable or misleading hotspot baselines.

---

### Finding 3: Dual-signal files identify true high-risk architectural debt

**Evidence:** Five files appear in both bloat-size and fix-density sets; top two are `cmd/orch/spawn_cmd.go` (1062 lines, 24 fix commits/28d) and `web/src/lib/components/work-graph-tree/work-graph-tree.svelte` (1364 lines, 17 fix commits/28d). `web/tests/work-graph.spec.ts` is also extreme (2017 lines, 8 fix commits).

**Source:** Command: `jq '{dual_signal_details}' /tmp/hotspots.json`; command: `git log --since='28 days ago' --pretty=format:'%s' -- "cmd/orch/spawn_cmd.go" | rg '^fix(\(.+\))?:' -c`; command: `wc -l "cmd/orch/spawn_cmd.go" "web/src/lib/components/work-graph-tree/work-graph-tree.svelte" "web/tests/work-graph.spec.ts"`.

**Significance:** Risk is concentrated, not diffuse; targeted decomposition work in these files is likely to produce the highest reliability payoff.

---

### Finding 4: Targeted follow-up issues were filed for immediate queueing

**Evidence:** Three follow-ups created: `orch-go-68ue7` (hotspot filtering bug), `orch-go-wkeag` (spawn command decomposition), `orch-go-um2xq` (work-graph component/spec split).

**Source:** Commands: `bd create ...`; output IDs returned by bd CLI.

**Significance:** The audit result is now actionable in backlog form, with clear separation between hygiene fix and architectural refactors.

---

## Synthesis

**Key Insights:**

1. **Signal quality is the first blocker** - Before using hotspot output for planning, filtering behavior must be corrected so generated artifacts and exclusion misconfiguration do not dominate results (Findings 1-2).

2. **Risk concentrates in dual-signal files** - Files with both churn and bloat provide stronger evidence of structural instability than single-signal files (Finding 3).

3. **Backlog now matches audit reality** - Filed follow-ups map directly to highest-risk observations instead of generic cleanup work (Finding 4).

**Answer to Investigation Question:**

The highest-value follow-up is to fix hotspot filtering semantics, then prioritize decomposition work for `cmd/orch/spawn_cmd.go` and the work-graph surface. Evidence shows current output includes 11/66 artifact hotspots and exclusion behavior is currently insufficient for directory-class filtering (Findings 1-2). After removing that noise, dual-signal files (`spawn_cmd.go`, `work-graph-tree.svelte`, plus oversized `work-graph.spec.ts`) remain the clearest reliability risks (Finding 3), and these were captured as targeted issues (Finding 4).

---

## Structured Uncertainty

**What's tested:**

- ✅ Hotspot totals and type distribution were measured from actual CLI output (`orch hotspot --json` + jq summaries).
- ✅ Directory wildcard exclusion failure was reproduced with an executed command (`--exclude "web/.svelte-kit/*"`).
- ✅ Top dual-signal file risk was validated with both line counts (`wc -l`) and fix-count history (`git log ... | rg ... -c`).

**What's untested:**

- ⚠️ Exact best default exclusion list for all projects (beyond current artifact classes) is not finalized.
- ⚠️ Expected reduction in future fix-density after decomposition is inferred, not yet measured.
- ⚠️ Whether `.spec.ts` should be globally excluded vs scored separately is not yet decided.

**What would change this:**

- If rerunning hotspot after exclusion fixes still surfaces mostly actionable source files with minimal artifact noise, Finding 1 impact drops.
- If decomposition of dual-signal files does not reduce fix churn over subsequent windows, Insight 2 should be revised.
- If test-file hotspot scoring proves predictive for regressions, excluding `.spec.ts` would be the wrong policy.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Fix hotspot filtering semantics and exclusion ergonomics (`orch-go-68ue7`) | implementation | Localized CLI behavior and tests; reversible and bounded.
| Decompose `cmd/orch/spawn_cmd.go` (`orch-go-wkeag`) | architectural | Affects core spawn flow boundaries and requires sequencing across extracted modules.
| Split work-graph component + large spec (`orch-go-um2xq`) | architectural | Crosses UI component structure and test strategy, with multiple valid designs.

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Two-Stage Hotspot Risk Burn-Down** - Fix hotspot signal integrity first, then execute targeted decompositions on dual-signal hotspots.

**Why this approach:**
- Removes noisy artifact paths so hotspot output can be trusted for triage.
- Focuses engineering effort where both churn and bloat are already proven.
- Directly maps to filed issues with clear ownership and acceptance criteria.

**Trade-offs accepted:**
- Broad, codebase-wide bloat cleanup is deferred.
- This is acceptable because concentrated hotspots provide higher ROI and faster risk reduction.

**Implementation sequence:**
1. Ship hotspot filtering/exclusion fixes (`orch-go-68ue7`) to stabilize measurement.
2. Run architect-guided decomposition for `cmd/orch/spawn_cmd.go` (`orch-go-wkeag`).
3. Refactor work-graph component and test split (`orch-go-um2xq`) and compare hotspot deltas.

### Alternative Approaches Considered

**Option B: Refactor high-risk files immediately without fixing hotspot filtering**
- **Pros:** Faster direct action on obvious hotspots.
- **Cons:** Keeps noisy measurement in place, making reprioritization and future audits unreliable.
- **When to use instead:** If immediate incident pressure requires tactical refactor before tooling correction.

**Option C: Restrict follow-up to one file (`spawn_cmd.go`)**
- **Pros:** Minimal coordination overhead.
- **Cons:** Leaves equally visible web graph risks and hotspot tooling defects unresolved.
- **When to use instead:** If only one team has available capacity this cycle.

**Rationale for recommendation:** Stage-gated approach protects decision quality (clean signals first) while still addressing the most severe dual-signal risks immediately afterward.

---

### Implementation Details

**What to implement first:**
- Hotspot exclusion matcher and default behavior adjustments.
- Regression tests for exclusion matching and default-preservation behavior.
- Decomposition design artifact for spawn and work-graph follow-up tasks.

**Things to watch out for:**
- ⚠️ `--exclude` UX changes can break existing scripts if semantics change without compatibility path.
- ⚠️ Work-graph spec splitting can mask shared setup assumptions if fixtures are not centralized.
- ⚠️ Spawn command extraction must preserve behavior across all backends and flags.

**Areas needing further investigation:**
- Evaluate separate hotspot class for large test files instead of pure exclusion.
- Add trend snapshots so hotspot deltas can be compared before/after refactors.
- Assess whether investigation-cluster topics should share exclusion controls.

**Success criteria:**
- ✅ Artifact-path hotspots are removed/reduced by default and directory exclusions behave as intended.
- ✅ Follow-up refactors reduce line counts in targeted files without regressions.
- ✅ Next hotspot run shows reduced dual-signal severity for targeted files.

---

## References

**Files Examined:**
- `cmd/orch/hotspot.go` - Verified exclusion matcher behavior, default exclude wiring, and bloat scan rules.
- `.kb/investigations/archived/2026-01-17-inv-design-800-line-bloat-gate.md` - Verified original intent for bloat hotspot behavior.
- `.kb/investigations/archived/2026-01-17-inv-implement-bloat-size-hotspot-type.md` - Checked implementation context and prior assumptions.

**Commands Run:**
```bash
# Generate current hotspot report
orch hotspot --json

# Summarize hotspot composition and artifact noise
orch hotspot --json > /tmp/hotspots.json && jq '{total_hotspots: (.hotspots|length), by_type: (.hotspots|group_by(.type)|map({type: .[0].type, count:length})), generated_or_artifact_hotspots: (.hotspots|map(select((.path|test("^web/\\.svelte-kit/|^web/playwright-report/|^build/|^\\.kb/"))))|length)}' /tmp/hotspots.json

# Reproduce exclusion matcher limitation
orch hotspot --json --exclude "web/.svelte-kit/*" > /tmp/hotspots-exclude-dirpattern.json

# Validate dual-signal hotspots
jq '{dual_signal_details: [((.hotspots|map(select(.type=="bloat-size")|{(.path): .score})|add) as $b | (.hotspots|map(select(.type=="fix-density")|{(.path): .score})|add) as $f | ($b|keys[]) as $k | select($f[$k] != null) | {path:$k, bloat_lines:$b[$k], fix_commits:$f[$k]})] | sort_by(-.fix_commits)}' /tmp/hotspots.json

# Validate churn and bloat for top files
git log --since='28 days ago' --pretty=format:'%s' -- "cmd/orch/spawn_cmd.go" | rg '^fix(\(.+\))?:' -c
git log --since='28 days ago' --pretty=format:'%s' -- "web/src/lib/components/work-graph-tree/work-graph-tree.svelte" | rg '^fix(\(.+\))?:' -c
wc -l "cmd/orch/spawn_cmd.go" "web/src/lib/components/work-graph-tree/work-graph-tree.svelte" "web/tests/work-graph.spec.ts"

# File follow-up issues
bd create "Hotspot audit: filter generated artifacts and fix --exclude semantics" --type bug --priority P1 --labels triage:ready ...
bd create "Architectural follow-up: decompose cmd/orch/spawn_cmd.go hotspot" --type task --priority P1 --labels triage:review ...
bd create "Work-graph hotspot follow-up: split tree component and oversized spec" --type task --priority P1 --labels triage:review ...
```

**External Documentation:**
- `.kb/guides/code-extraction-patterns.md` - Extraction guidance referenced by existing bloat recommendations.

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-30-bloat-control-enforcement-patterns.md` - Defines hotspot-based bloat detection intent.
- **Investigation:** `.kb/investigations/archived/2026-01-17-inv-design-800-line-bloat-gate.md` - Design baseline for bloat hotspot behavior.
- **Investigation:** `.kb/investigations/archived/2026-01-17-inv-implement-bloat-size-hotspot-type.md` - Implementation baseline for current behavior.
- **Workspace:** `.orch/workspace/og-feat-polish-run-codebase-08feb-86bd/SPAWN_CONTEXT.md` - Task scope and deliverable contract.

---

## Investigation History

**[2026-02-08 19:02]:** Investigation started
- Initial question: Which hotspot results are highest-risk and actionable now?
- Context: Spawn task requested focused hotspot audit and follow-up issue creation.

**[2026-02-08 19:08]:** Baseline hotspot and noise profile captured
- Verified 66 hotspots total and 11 generated/artifact-path hotspots.

**[2026-02-08 19:13]:** Exclusion behavior tested
- Confirmed directory wildcard exclusion mismatch and default-exclusion replacement behavior.

**[2026-02-08 19:18]:** High-risk concentration confirmed
- Validated dual-signal set and top offenders (`spawn_cmd.go`, `work-graph-tree.svelte`, `work-graph.spec.ts`).

**[2026-02-08 19:26]:** Follow-up issues created
- Created `orch-go-68ue7`, `orch-go-wkeag`, `orch-go-um2xq`.

**[2026-02-08 19:29]:** Investigation completed
- Status: Complete
- Key outcome: Converted hotspot audit into three targeted follow-ups with evidence-backed prioritization.
