# Decision: Extract Automatable Self Review Checks To Orch Com

**Date:** 2026-03-10
**Status:** Accepted

## Context

This decision was initially captured as a quick kn entry and has now been promoted to a full decision record for better documentation.

## Decision

**Chosen:** Extract automatable self-review checks to orch complete gate

**Rationale:** Reduces cross-cutting behavioral weight by 357 lines across 5 skills. Debug statements, commit format, placeholder data, and orphaned Go files now checked at completion time via self_review V1 gate.

## Consequences

- Positive: [Expand on positive outcomes]
- Risks: [Consider potential risks]

## Source

**Promoted from:** quick entry kb-ae2e70 (decision)
**Original date:** 2026-03-06

