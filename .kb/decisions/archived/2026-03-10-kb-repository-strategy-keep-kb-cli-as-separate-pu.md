# Decision: Kb Repository Strategy Keep Kb Cli As Separate Pu

**Date:** 2026-03-10
**Status:** Accepted

## Context

This decision was initially captured as a quick kn entry and has now been promoted to a full decision record for better documentation.

## Decision

**Chosen:** KB repository strategy: keep kb-cli as separate public repo, not monorepo under orch-go

**Rationale:** First-user-profile probe showed external user does NOT need orch/beads/skillc/daemon. Separate repo is cleanest signal — discoverable, no orchestration baggage. kb-cli already exists as separate repo.

## Consequences

- Positive: [Expand on positive outcomes]
- Risks: [Consider potential risks]

## Source

**Promoted from:** quick entry kb-69dedf (decision)
**Original date:** 2026-03-09

