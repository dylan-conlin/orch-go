# Decision: No Code Review Gate — Expand Execution-Based Verification Instead

**Date:** 2026-02-25
**Status:** Proposed
**Deciders:** Dylan
**Blocks:** code review, completion pipeline, diff review, agent review, go vet, build gate, review gate

## Context

The orchestrator's completion review is a knowledge management ceremony — nobody reads the diff. The only checks are: tests pass (pattern-matched claim), build succeeds (the one execution-based gate), accretion check, and SYNTHESIS.md review. This is analogous to approving a PR by reading the description and seeing CI green.

The question: should a code review agent be added to the completion pipeline to close this gap?

## Decision

**Do NOT add a code review gate.** Instead:

1. **Expand the build gate to include `go vet`** — execution-based, zero-cost, deterministic, catches real bugs without the closed-loop problem of agent review.

2. **Add hotspot-triggered advisory warnings** — when `orch complete` detects changes in high-risk areas, emit a warning suggesting the orchestrator spawn a review investigation. Advisory, not blocking.

## Rationale

### Why not code review?

Five substrate signals converge against agent code review at completion:

1. **Provenance (principle):** Agent reviewing agent code is a closed loop. Code review findings are agent opinions (hypotheses), not executable evidence. "An agent reviewed it" has no provenance chain — the orchestrator can't independently verify the review was thorough.

2. **Phased adversarial verification (decision):** Existing decision says post-completion review doesn't prevent flawed conclusions, only detects them. Code review at completion IS post-completion review.

3. **V0-V3 gate proliferation (decision):** Gates proliferate → flag combos needed → `--force` becomes default → theater. Adding another gate pushes toward this known failure mode.

4. **Evidence Hierarchy (principle):** Code is truth. Code review produces more artifacts (hypotheses about code), not more truth. Only execution produces truth.

5. **Verification Bottleneck (principle):** Human verification bandwidth is the rate limiter. Agent code review is NOT human verification. It doesn't increase Dylan's verification bandwidth — it adds an agent judgment layer that nobody can independently evaluate.

### Why `go vet` instead?

`go vet` is the build gate's natural expansion because it shares the same properties that make `go build` the "only unfakeable gate":

| Property | `go build` | `go vet` | Agent review |
|---|---|---|---|
| Execution-based | ✓ | ✓ | ✗ |
| Deterministic | ✓ | ✓ | ✗ |
| Zero token cost | ✓ | ✓ | ✗ (opus) |
| Verifiable output | ✓ | ✓ | ✗ |
| Same code = same result | ✓ | ✓ | ✗ |
| Catches real bugs | compilation | unused vars, format strings, unreachable code, suspicious constructs | opinions |

### Three-type gate taxonomy

This decision reveals a useful taxonomy:

- **Execution-based gates** (build, vet, tests): Produce provenance. Binary pass/fail. Machine-verifiable.
- **Evidence-based gates** (test evidence, git diff, SYNTHESIS): Pattern match against claims. Anti-theater detection.
- **Judgment-based gates** (explain-back, behavioral): Human comprehension. Only valid when human performs them.

Agent code review would be a fourth type — **agent judgment** — that produces opinion without provenance. The pipeline should not contain this type.

### Why hotspot advisory instead of hotspot blocking?

The V0-V3 decision established "spawn-only immutability" — verification level set at spawn, not changed at completion. Blocking completion based on completion-time hotspot discovery would violate this. Advisory warnings preserve orchestrator judgment (human in the loop) while surfacing the signal.

## Consequences

**Positive:**
- `go vet` catches 10-30% of "compiles but has issues" bugs at zero cost
- Hotspot warnings provide actionable signal for high-risk completions
- No gate proliferation — vet extends existing V2 build gate, warnings are advisory
- No closed loops — all new verification is execution-based
- No additional completion time in common case (~2s for vet)

**Negative:**
- Logic errors, security vulnerabilities, and edge cases that pass `go build` AND `go vet` will NOT be caught automatically
- Hotspot warnings are advisory — orchestrator may ignore them
- No fresh perspective on the code (same gap, just with deterministic checks covering more surface area)

**Risks:**
- `go vet` may produce false positives in some code patterns (mitigate: start as warnings, promote to errors after confidence period)
- Hotspot warnings may become noise if too frequent (mitigate: only fire for HIGH/CRITICAL scores)

## What This Accepts

**Accepted gap:** The system has no mechanism where a fresh perspective (human or AI) reviews every code change. This is accepted because:
1. Agent review is a closed loop (same model, same blind spots)
2. Human review is bottlenecked by Dylan's bandwidth
3. The explain-back gate provides human comprehension of WHAT was built
4. The behavioral gate provides human observation of WHETHER it works
5. Targeted review is available as escape hatch (`orch spawn investigation "review orch-go-XXXX"`)

**When this decision should be revisited:**
- If a new class of bugs emerges that `go vet` + `go build` + tests consistently miss
- If a fundamentally different model family (different blind spots) becomes available for review
- If the overhead of selective review investigations becomes too high (suggesting the need for automation)

## Implementation

### Phase 1: `go vet` in build gate
- Add `go vet ./...` to `pkg/verify/build_verification.go`
- Report as warnings initially, promote to errors after <5% false positive rate

### Phase 2: Hotspot advisory warnings
- Add hotspot check to `cmd/orch/complete_cmd.go` (after verification passes)
- Create `pkg/verify/hotspot_advisory.go` for matching diff against hotspot data
- Advisory output with suggested follow-up command

## References

- Investigation: `.kb/investigations/2026-02-25-design-code-review-gate-for-completion-pipeline.md`
- Probe: `.kb/models/completion-verification/probes/2026-02-25-probe-code-review-gate-design.md`
- Model: `.kb/models/completion-verification.md`
- Prior decisions: verifiability-first, V0-V3 levels, phased adversarial verification
