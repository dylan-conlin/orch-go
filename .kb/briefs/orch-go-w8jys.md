# Brief: orch-go-w8jys

## Frame

Architect agents design multi-phase work — Phase 1, 2, 3 — and the handoff gate checks that implementation issues exist before the architect can complete. Twice now, the gate waved through a 3-phase design with only 1 issue. Phases 2 and 3 got lost. The architect did the thinking, but the system treated "one issue exists" as "all work is captured."

## Resolution

The bug was in the question the gate asked. It asked "does at least one issue exist?" — a yes/no question — when the right question was "do enough issues exist?" The fix reads the SYNTHESIS.md, looks for Phase/Layer/Step/Stage indicators (regex against numbered headings and bold markers), counts the distinct phases, and requires that many issues before the gate passes. The three existing signal paths (auto-created title pattern, manual handoff comment, explicit opt-out) all still work, but now the first two are count-checked instead of boolean-checked.

The interesting part was how cleanly this connected to the attractor-gate model. The model says gates alone fail — and this was a gate alone. It checked a boolean, not a structural property. After the fix, it's still a gate (not an attractor), but it's a structurally-informed gate: it knows something about the shape of the work it's guarding. This suggests a refinement to the intervention hierarchy: structurally-informed gates sit above boolean gates, which sit above advisory gates.

## Tension

The gate now catches the gap, but the auto-create mechanism (`maybeAutoCreateImplementationIssue`) still only creates one issue per architect design. The gate will correctly block completion, but the architect will have to create phases 2 and 3 manually. A follow-up to make auto-create phase-aware would close this properly — the question is whether that's worth the complexity or if manual creation is fine since the gate now prevents silent loss.
