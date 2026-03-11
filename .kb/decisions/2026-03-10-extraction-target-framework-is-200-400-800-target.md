# Decision: Extraction Target Framework Is 200 400 800 Target

**Date:** 2026-03-10
**Status:** Accepted

## Context

This decision was initially captured as a quick kn entry and has now been promoted to a full decision record for better documentation.

## Decision

**Chosen:** Extraction target framework is 200/400/800 (target/warning/intervention), not just under-800

**Rationale:** Architect analysis (orch-go-yeidp) showed satellites 100-300 lines have 0 post-extraction commits, while residuals >600 re-cross 800 within weeks. Under-800 is necessary but insufficient — durable extraction requires 200-400 line targets.

## Consequences

- Positive: [Expand on positive outcomes]
- Risks: [Consider potential risks]

## Source

**Promoted from:** quick entry kb-0a934e (decision)
**Original date:** 2026-03-10

