# Session Synthesis

**Agent:** og-arch-design-benchmark-runner-26mar-1999
**Issue:** orch-go-8mpry
**Duration:** 2026-03-26 10:06 -> 2026-03-26 10:24
**Outcome:** success

---

## TLDR

I designed `orch benchmark` as a top-level, suite-driven command for automating model reliability tests. The key decision was to keep live benchmarking separate from `orch harness`, and to reuse existing spawn, wait, and rework primitives instead of creating a second execution stack.

---

## Plain-Language Summary

This session answered where a benchmark runner belongs and what it should actually do. The recommendation is to add a dedicated `orch benchmark` command that can run a repeatable set of model tests, capture the important evidence in one place, and produce a verdict the orchestrator can trust without manually inspecting workspaces.

The main reason this matters is that recent model benchmarks worked, but only by stitching together comments, workspace files, and git history after the fact. Making that workflow first class turns benchmarking from a one-off investigation into a repeatable capability.

---

## Delta (What Changed)

### Files Created
- `.orch/workspace/og-arch-design-benchmark-runner-26mar-1999/VERIFICATION_SPEC.yaml` - Verification contract for this design session
- `.orch/workspace/og-arch-design-benchmark-runner-26mar-1999/SYNTHESIS.md` - Session synthesis for orchestrator review
- `.orch/workspace/og-arch-design-benchmark-runner-26mar-1999/BRIEF.md` - Dylan-facing comprehension brief

### Files Modified
- `.kb/investigations/2026-03-26-inv-design-benchmark-runner-command-orch.md` - Filled with findings, synthesis, and architectural recommendation
- `.kb/plans/2026-03-26-benchmark-runner.md` - Turned template into phased implementation plan

### Commits
- Pending session commit with design artifacts only

---

## Evidence (What Was Observed)

- `cmd/orch/harness_cmd.go` scopes harness to governance and telemetry workflows, so it is the wrong home for live benchmark execution.
- `cmd/orch/spawn_cmd.go` and `pkg/orch/loop.go` already provide tracked spawn plus wait/eval/rework behavior suitable for reuse.
- The recent GPT-5.4 benchmark investigation shows the current manual workflow is evidence-scattered and hard to repeat.
- Follow-up work was decomposed into four issues plus an implementation plan so the design can land without ambiguity.

### Tests Run
```bash
# Verified loop primitives are already working
go test ./pkg/orch -run Loop -count=1
# PASS

# Verified reliability-testing spawns already require investigation deliverables
go test ./pkg/spawn -run 'TestGenerateContext/reliability-testing includes investigation deliverable' -count=1
# PASS
```

---

## Architectural Choices

### Command placement
- **What I chose:** Top-level `orch benchmark`
- **What I rejected:** Extending `orch harness`
- **Why:** Harness is analytics and governance oriented, while benchmarking is active orchestration.
- **Risk accepted:** Another root namespace increases CLI surface area.

### Execution strategy
- **What I chose:** Thin benchmark runner over existing spawn/wait/rework primitives
- **What I rejected:** Building a benchmark-specific lifecycle engine
- **Why:** Reuse keeps behavior aligned with existing orchestration and lowers regression risk.
- **Risk accepted:** Some adapter seams may feel awkward during implementation.

### Artifact strategy
- **What I chose:** Canonical benchmark run artifacts plus summary reporting
- **What I rejected:** Ad hoc investigations and comments only
- **Why:** Centralized evidence is the main value of making benchmarks first class.
- **Risk accepted:** Version one may need to refine artifact shape after implementation feedback.

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/investigations/2026-03-26-inv-design-benchmark-runner-command-orch.md` - Architectural recommendation for benchmark runner placement and shape
- `.kb/plans/2026-03-26-benchmark-runner.md` - Phase-by-phase implementation plan

### Decisions Made
- Benchmark execution belongs in a top-level `orch benchmark` namespace, not under harness.
- The first version should be suite-driven and serial by default.
- The runner should reuse spawn lifecycle primitives and focus new code on suite expansion plus result collection.

### Constraints Discovered
- Harness semantics are already occupied by governance and telemetry analysis.
- Benchmarking is exposed to Multi-Backend Blindness and Contradictory Authority Signals if result collection is not canonical.

### Externalized via `kb quick`
- `kb quick decide "Benchmark runner should live at top-level orch benchmark, not under orch harness" --reason "Harness commands analyze governance telemetry or delegate to the external harness binary, while benchmark needs to orchestrate live model runs using existing spawn/loop/rework primitives."`

---

## Verification Contract

See `.orch/workspace/og-arch-design-benchmark-runner-26mar-1999/VERIFICATION_SPEC.yaml`.

Key outcomes:
- Targeted substrate tests passed for loop reuse and reliability-testing spawn requirements.
- The design was externalized into an investigation, a phased plan, and four follow-up issues with integration dependencies.

---

## Next (What Should Happen)

**Recommendation:** spawn-follow-up

### If Spawn Follow-up
**Issue:** Implement benchmark runner plan
**Skill:** feature-impl
**Context:**
```text
Start with orch-go-xjdx9 to define the CLI surface and suite schema. The benchmark command should stay top-level, reuse existing spawn lifecycle primitives, and produce canonical benchmark artifacts so verdicts do not depend on manual workspace inspection.
```

---

## Unexplored Questions

**Questions that emerged during this session that weren't directly in scope:**
- Should benchmark run artifacts live under `.orch/benchmarks/` permanently, or should mature runs be promotable into a knowledge surface?
- Should `orch benchmark report` ship in the first cut or after artifact structure stabilizes?

**Areas worth exploring further:**
- Sample suite design for common model-routing decisions
- Resume behavior for interrupted benchmark runs

**What remains unclear:**
- The final artifact layout for repeated benchmarks across projects

---

## Friction

**System friction experienced during this session:**
- tooling: Shell command substitution fired when backticks appeared inside `bd update` and `bd comments add` bash strings, causing harmless `unknown command "benchmark"` noise before the intended beads operations completed

---

## Session Metadata

**Skill:** architect
**Model:** openai/gpt-5.4
**Workspace:** `.orch/workspace/og-arch-design-benchmark-runner-26mar-1999/`
**Investigation:** `.kb/investigations/2026-03-26-inv-design-benchmark-runner-command-orch.md`
**Beads:** `bd show orch-go-8mpry`
