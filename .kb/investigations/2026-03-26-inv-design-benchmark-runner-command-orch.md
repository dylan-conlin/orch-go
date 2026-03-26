<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The right shape is a top-level `orch benchmark` command that runs config-defined benchmark suites on top of existing spawn, wait, and rework primitives rather than extending `orch harness`.

**Evidence:** `cmd/orch/harness_cmd.go` shows harness is reserved for governance and telemetry workflows, `cmd/orch/spawn_cmd.go` plus `pkg/orch/loop.go` already provide the live-execution substrate, and targeted tests for loop plus reliability-testing spawn context both pass.

**Knowledge:** Reliability benchmarking is a cross-backend orchestration concern with defect exposure to Multi-Backend Blindness and Contradictory Authority Signals, so the design must keep one canonical runner and one canonical result collector.

**Next:** Implement the plan in four tracked issues: CLI surface, execution engine, result artifacts/reporting, and behavioral integration verification.

**Authority:** architectural - The command cuts across CLI shape, execution flow, artifact placement, and follow-up issue decomposition.

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

# Investigation: Design Benchmark Runner Command Orch

**Question:** What should `orch benchmark` look like if it needs to automate model reliability testing without duplicating `orch harness` analytics or inventing a second orchestration path?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** og-arch-design-benchmark-runner-26mar-1999
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: `orch harness` is the wrong home for live benchmark execution

**Evidence:** The harness command is explicitly scoped to control-plane immutability, governance helpers, and telemetry reporting; its subcommands either delegate to the standalone `harness` binary or summarize previously emitted events. Nothing in that command family manages worker runs.

**Source:** `cmd/orch/harness_cmd.go:13`, `cmd/orch/harness_cmd.go:16`, `cmd/orch/harness_cmd.go:40`, `cmd/orch/harness_report_cmd.go:19`, `cmd/orch/harness_audit_cmd.go:20`, `cmd/orch/harness_gate_effectiveness_cmd.go:21`

**Significance:** Putting benchmark execution under harness would conflate live experiment orchestration with after-the-fact governance analytics. By the principle of Evolve by Distinction, the benchmark runner should be separate from the harness reporting surface.

---

### Finding 2: The live-execution substrate already exists in spawn, wait, and rework primitives

**Evidence:** `spawn --loop` already wires tracked spawn output into `orch.RunLoop`, which waits for `Phase: Complete`, runs an eval command, and triggers rework through `runReworkWithParams` when evaluation fails. The rework path already knows how to reopen an issue with structured feedback.

**Source:** `cmd/orch/spawn_cmd.go:564`, `cmd/orch/spawn_cmd.go:575`, `pkg/orch/loop.go:16`, `pkg/orch/loop.go:53`, `pkg/orch/loop.go:82`, `pkg/orch/loop.go:148`, `cmd/orch/rework_cmd.go:30`, `cmd/orch/rework_cmd.go:91`

**Significance:** `orch benchmark` should be a thin batch runner over this substrate, not a second implementation of spawn lifecycle management. This reduces defect exposure to Duplicate Action and Premature Destruction.

---

### Finding 3: Reliability benchmarking is already a named need, but today it is manual and evidence-scattered

**Evidence:** The reliability-testing skill is explicitly about repeated real-world validation until a target threshold is met, while the recent GPT-5.4 benchmark investigation had to assemble evidence from workspace inspection, OpenCode DB analysis, beads comments, and git history to decide viability. The benchmark standard exists conceptually but not as a first-class command.

**Source:** `/Users/dylanconlin/.opencode/skill/reliability-testing/SKILL.md:19`, `/Users/dylanconlin/.opencode/skill/reliability-testing/SKILL.md:45`, `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md:3`, `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md:136`

**Significance:** The runner should standardize suite definition, evidence capture, and verdict calculation so future model comparisons do not require bespoke investigation work each time.

---

### Finding 4: Existing model metadata and protocol tests are sufficient to support a suite-driven first version

**Evidence:** Model aliases and context-window metadata already resolve OpenAI, Anthropic, Google, and DeepSeek variants, and spawn context tests already enforce that reliability-testing sessions carry investigation deliverables. Focused tests passed for the loop controller and for reliability-testing context generation.

**Source:** `pkg/model/model.go:91`, `pkg/model/model.go:98`, `pkg/model/model.go:104`, `pkg/spawn/context_test.go:1564`, `go test ./pkg/orch -run Loop -count=1`, `go test ./pkg/spawn -run 'TestGenerateContext/reliability-testing includes investigation deliverable' -count=1`

**Significance:** Version one can rely on declarative suite files plus existing model resolution instead of inventing a model registry or a new benchmark-specific worker protocol.

---

## Synthesis

**Key Insights:**

1. **Placement is the first architectural fork** - Finding 1 shows harness is for governance and measurement of existing events, while Findings 2 and 3 show benchmarking is an active orchestration workflow; that makes top-level `orch benchmark` the coherent placement.

2. **Reuse beats reinvention here** - Findings 2 and 4 together show the command does not need new lifecycle primitives; it needs a suite expander, a canonical result collector, and a report layer over spawn/wait/rework.

3. **The main design risk is not execution, it is evidence drift** - Findings 3 and 4 show the system already can run tests, but benchmark conclusions currently depend on scattered artifacts; the runner must centralize verdict inputs to avoid Contradictory Authority Signals.

**Answer to Investigation Question:**

`orch benchmark` should be a top-level CLI namespace with an initial `run` command and a dedicated package (for example `pkg/benchmark`) that expands a benchmark suite into tracked cases, executes them serially by default using existing spawn/wait/rework primitives, and writes one canonical benchmark run artifact plus a summarized verdict report. The command should not live under `orch harness`, because harness is scoped to governance and telemetry reporting rather than live worker orchestration (Finding 1). It should not invent a new execution stack, because spawn loop and rework already cover the needed lifecycle (Finding 2). Its real value is standardizing reliability-testing evidence so repeated model comparisons become routine instead of bespoke investigations (Findings 3 and 4).

---

## Structured Uncertainty

**What's tested:**

- ✅ Loop controller behavior is already covered by passing package tests (verified: `go test ./pkg/orch -run Loop -count=1`)
- ✅ Reliability-testing spawn contexts include investigation deliverables (verified: `go test ./pkg/spawn -run 'TestGenerateContext/reliability-testing includes investigation deliverable' -count=1`)
- ✅ Current command boundaries place harness in governance/telemetry space, not live benchmarking (verified by reading `cmd/orch/harness_cmd.go` and its native subcommands)

**What's untested:**

- ⚠️ No end-to-end benchmark runner exists yet, so suite ergonomics and artifact shape are design recommendations rather than exercised behavior
- ⚠️ The right long-term home for benchmark artifacts (`.orch/benchmarks/` vs another persistent surface) has not been validated in implementation
- ⚠️ Parallel benchmark execution is intentionally deferred; serial execution is recommended first to avoid measurement contamination

**What would change this:**

- If harness is intentionally expanded to own live worker experiments, command placement should be revisited
- If existing spawn/wait/rework primitives prove too leaky for batch use, the recommendation to reuse them should narrow to partial reuse
- If benchmark artifact review shows the proposed canonical collector misses key signals, the reporting component should be redesigned before rollout

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Create a top-level `orch benchmark` command backed by a suite-driven runner and canonical result artifacts | architectural | This spans command placement, package boundaries, artifact lifecycle, and integration with existing orchestration primitives |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Top-level suite runner over existing orchestration primitives** - Add `orch benchmark run --suite <file>` as the first user-facing entry point, backed by a dedicated benchmark package that expands suites, runs tracked cases, and writes one canonical run summary.

**Why this approach:**
- Keeps command placement coherent: benchmarking is live orchestration, not harness telemetry
- Reuses proven lifecycle components instead of creating a second spawn stack
- Centralizes verdict inputs so model comparisons stop depending on manual workspace archaeology

**Trade-offs accepted:**
- Defers parallel execution until serial evidence collection is trustworthy
- Accepts a new artifact surface for benchmark runs because stuffing these results into generic investigations would make comparison harder

**Implementation sequence:**
1. Build the CLI surface plus suite-file loader so benchmark cases have one declarative source of truth
2. Implement the runner on top of spawn, wait, and optional rework hooks so case execution stays aligned with existing lifecycle behavior
3. Add canonical run artifacts and summarized verdict reporting so repeated benchmarks become comparable and reviewable

### Alternative Approaches Considered

**Option B: Extend `orch harness` with benchmark execution**
- **Pros:** Reuses an existing namespace associated with measurement
- **Cons:** Conflicts with harness's current governance/analytics identity and blurs the line between producing telemetry and analyzing it
- **When to use instead:** Only if the project explicitly redefines harness as the home for all measurement-related workflows

**Option C: Keep benchmarks as ad hoc scripts and investigations**
- **Pros:** Lowest upfront implementation cost
- **Cons:** Preserves scattered evidence, inconsistent metrics, and manual comparison work
- **When to use instead:** Only for one-off experiments that are not intended to influence routing or policy

**Rationale for recommendation:** The recommended approach is the only one that preserves existing command boundaries, reuses working orchestration primitives, and directly addresses the evidence-scatter problem surfaced by recent model benchmarks.

---

### Implementation Details

**What to implement first:**
- A `cmd/orch/benchmark_cmd.go` entry point with `run` subcommand and suite-file parsing
- A benchmark case schema that names model, backend, skill, task source, repeat count, and pass criteria
- A result collector that normalizes pass/fail, duration, retries, commit evidence, and verification artifact presence

**Things to watch out for:**
- ⚠️ Defect class exposure: Class 2 (Multi-Backend Blindness) if results assume Anthropic-only behavior or metadata
- ⚠️ Defect class exposure: Class 5 (Contradictory Authority Signals) if the report mixes beads comments, workspace files, and git state without a canonical precedence rule
- ⚠️ Benchmark runs can interfere with each other under concurrency, so serial-by-default is the safer starting point

**Areas needing further investigation:**
- Whether benchmark artifacts should eventually graduate from `.orch/benchmarks/` into a dedicated persistent knowledge surface
- Whether a `report` subcommand should ship in the first cut or follow once run artifacts stabilize
- Whether case tasks should be embedded inline, referenced by file, or generated from existing beads issues

**Success criteria:**
- ✅ A suite file can declare multiple model or backend cases and produce one benchmark run directory with per-case evidence
- ✅ Re-running the same suite yields stable pass/fail accounting and comparable summary metrics
- ✅ The output is sufficient for an orchestrator to decide whether a model path is viable without manually inspecting raw workspaces

---

## References

**Files Examined:**
- `cmd/orch/harness_cmd.go` - Checked current harness scope and command boundaries
- `cmd/orch/harness_report_cmd.go` - Verified harness report is telemetry reporting, not live execution
- `cmd/orch/harness_audit_cmd.go` - Verified harness audit consumes historical events
- `cmd/orch/harness_gate_effectiveness_cmd.go` - Verified harness focuses on post-hoc gate analysis
- `cmd/orch/spawn_cmd.go` - Checked where loop mode is wired into tracked spawns
- `pkg/orch/loop.go` - Examined reusable wait/eval/rework lifecycle
- `cmd/orch/rework_cmd.go` - Verified rework can be invoked programmatically from loop-like flows
- `pkg/model/model.go` - Verified model aliases and context metadata already exist
- `pkg/spawn/context_test.go` - Verified reliability-testing spawn expectations
- `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` - Used the recent manual benchmark as evidence for current pain and thresholds

**Commands Run:**
```bash
# Verify workspace location
pwd

# Inspect existing knowledge context
kb context "orchestrator"
kb context "benchmark runner"

# Validate reusable execution primitives
go test ./pkg/orch -run Loop -count=1

# Validate reliability-testing spawn contract
go test ./pkg/spawn -run 'TestGenerateContext/reliability-testing includes investigation deliverable' -count=1

# Create implementation plan and follow-up issues
orch plan create benchmark-runner
bd create "Implement orch benchmark CLI surface and suite config loading" --type feature -l triage:ready -l area:cli -l effort:medium
bd create "Build benchmark execution engine on spawn/wait/rework primitives" --type feature -l triage:ready -l area:opencode -l effort:large
bd create "Add benchmark result artifacts and summary reporting" --type feature -l triage:ready -l area:kb -l effort:medium
bd create "Integration: orch benchmark produces repeatable model reliability verdicts" --type task -l triage:ready -l area:cli -l effort:medium
```

**External Documentation:**
<!-- All URLs must use markdown hyperlinks: [Display Name](https://url) — never bare URLs or plain text -->
- None

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-02-28-atc-not-conductor-orchestrator-reframe.md` - Reinforces that orchestrator-facing features should optimize for structured orchestration, not end-user CLI ergonomics
- **Investigation:** `.kb/investigations/2026-03-26-inv-benchmark-worker-reliability-across-claude.md` - Shows the manual benchmark workflow that `orch benchmark` should replace
- **Plan:** `.kb/plans/2026-03-26-benchmark-runner.md` - Captures the phased implementation path produced by this design session
- **Workspace:** `.orch/workspace/og-arch-design-benchmark-runner-26mar-1999/` - Session artifacts and verification contract for this design work

---

## Investigation History

**[2026-03-26 10:06]:** Investigation started
- Initial question: What should an `orch benchmark` command look like if its job is automating model reliability testing?
- Context: Recent GPT-5.4 benchmarking proved the value of the question but required a bespoke investigation to gather evidence.

**[2026-03-26 10:14]:** Command-boundary fork resolved
- Reading `harness` and `spawn` surfaces made it clear the primary distinction is telemetry analysis versus live orchestration.

**[2026-03-26 10:22]:** Implementation path externalized
- Created a benchmark-runner plan plus four follow-up issues covering CLI, execution engine, reporting, and integration verification.

**[2026-03-26 10:24]:** Investigation completed
- Status: Complete
- Key outcome: Recommend a top-level, suite-driven `orch benchmark` command built over existing spawn lifecycle primitives.
