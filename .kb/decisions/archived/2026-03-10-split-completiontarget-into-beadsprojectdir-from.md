# Decision: Split Completiontarget Into Beadsprojectdir From

**Date:** 2026-03-10
**Status:** Accepted

## Context

This decision was initially captured as a quick kn entry and has now been promoted to a full decision record for better documentation.

## Decision

**Chosen:** Split CompletionTarget into BeadsProjectDir (from beads ID prefix) + WorkProjectDir (from manifest/workdir) for cross-project orch complete

**Rationale:** Single BeadsProjectDir conflated beads operations (issue lookup, close) with code operations (build, verification). When issue and workspace live in different repos, one directory can't serve both purposes. Beads ID prefix is the canonical indicator of which project owns the issue.

## Consequences

- Positive: [Expand on positive outcomes]
- Risks: [Consider potential risks]

## Source

**Promoted from:** quick entry kb-293459 (decision)
**Original date:** 2026-03-09

