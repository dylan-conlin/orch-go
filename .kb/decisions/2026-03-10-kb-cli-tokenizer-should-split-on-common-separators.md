# Decision: Kb Cli Tokenizer Should Split On Common Separators

**Date:** 2026-03-10
**Status:** Accepted

## Context

This decision was initially captured as a quick kn entry and has now been promoted to a full decision record for better documentation.

## Decision

**Chosen:** kb-cli tokenizer should split on common separators (dots, hyphens, slashes, underscores) not just whitespace

**Rationale:** Compound tokens like sync.Pool were treated as single tokens, preventing multi-word queries from matching. Coverage score was halved (1/2 keywords matched) dropping models below MinStemmedScore filter.

## Consequences

- Positive: [Expand on positive outcomes]
- Risks: [Consider potential risks]

## Source

**Promoted from:** quick entry kb-bc993c (decision)
**Original date:** 2026-03-09

