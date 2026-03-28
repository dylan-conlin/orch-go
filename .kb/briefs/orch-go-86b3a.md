# Brief: orch-go-86b3a

## Frame

When an architect agent designs a three-phase implementation, the auto-create mechanism fires once and produces one issue — for Phase 1. Phases 2 and 3 become orphaned design work that nobody picks up. The handoff gate (from orch-go-w8jys) now catches this gap and blocks completion, but the auto-create itself never fills it. The architect hits a wall: "the gate says I need 3 issues but the system only made 1."

## Resolution

The fix teaches `maybeAutoCreateImplementationIssue` to read the same phase structure the gate already detects. When SYNTHESIS.md contains Phase/Layer/Step/Stage indicators with 2+ distinct numbers, the auto-create switches from single-issue mode to multi-phase mode: one issue per phase, each with a phase-specific title ("Phase 1: Add parser (from architect orch-go-xyz)"), its own description, skill inference, and KB context. The idempotency check compares existing issue count against phase count rather than a boolean exists-or-not.

The interesting wrinkle was naming collisions — `buildPhaseTitle` and `regexPhaseHeading` already existed in `plan_hydrate.go` and `plan_hydration.go` with completely different semantics. Plan hydration creates issues from explicit plan files; architect auto-create infers phases from SYNTHESIS.md prose. Same concept (phases → issues), different entry points.

## Tension

The idempotency model is coarse: if auto-create is interrupted after creating 2 of 3 phase issues, re-running won't fill the gap (it sees 2 < 3 but has no way to know *which* phases are missing without per-title matching). The gate will still catch it, but the human has to create the missing phase manually. This feels acceptable for now — the interrupted-mid-creation scenario requires the `bd create` CLI to fail partway through a loop — but it's worth knowing about if multi-phase designs become common.
