# Decision: Health Score Floor Gate Downgraded From Blocking T

**Date:** 2026-03-10
**Status:** Accepted
**Enforcement:** convention

## Context

This decision was initially captured as a quick kn entry and has now been promoted to a full decision record for better documentation.

## Decision

**Chosen:** Health score floor gate downgraded from blocking to advisory

**Rationale:** Phase 4 probe showed 37→73 improvement is 89% calibration artifact — same baseline scores 69 under new formula with zero extractions. Gate at 65 will never fire. Pre-commit accretion gate and hotspot blocking are the real hard enforcement. Score remains useful for orientation via orch health / kb health.

## Consequences

- Positive: [Expand on positive outcomes]
- Risks: [Consider potential risks]

## Source

**Promoted from:** quick entry kb-3e651d (decision)
**Original date:** 2026-03-10


## Auto-Linked Investigations

- .kb/investigations/2026-02-28-design-daemon-agreements-integration.md
- .kb/investigations/2026-03-01-design-control-plane-data-plane-separation.md
- .kb/investigations/2026-03-05-inv-design-domain-aware-spawn-routing.md
- .kb/investigations/2026-03-10-design-kb-health-portable-codebase-health-score.md
