# Decision: kb reflect Command Interface

**Date:** 2025-12-21
**Status:** Accepted
**Author:** og-arch-design-kb-reflect-21dec

## Context

The system addresses session amnesia but lacks project-lifetime memory. Artifacts accumulate without awareness of their relationships, evolution, or potential for synthesis. Humans must notice patterns manually.

Five prior investigations (ws4z.7 citations, ws4z.8 temporal signals, ws4z.9 chronicle, ws4z.10 constraint validation, 4kwt.8 reflection checkpoints) discovered specific patterns that require human attention:
- Investigation clustering (3+ investigations on same topic)
- Repeated constraints (duplicates indicate discovery failure)
- Low citation decisions (potentially stale)
- Constraints contradicted by code (drift)

## Decision

Implement `kb reflect` as a single command with `--type` flag for four reflection modes:

```bash
kb reflect                     # Run all types, show summary
kb reflect --type synthesis    # Investigations needing consolidation
kb reflect --type stale        # Decisions with low/no citations
kb reflect --type drift        # Constraints contradicted by code
kb reflect --type promote      # kn entries ready for kb promotion
```

### Key Design Choices

1. **Single command with --type flag** (not separate commands)
   - Consistent with `kb context`, `kb search`, `kb chronicle` pattern
   - Most discoverable (one command to learn)
   - Extensible (add new types easily)

2. **Shell script MVP** (not Go implementation)
   - Matches existing `kb` command pattern
   - Faster iteration on detection heuristics
   - Zero dependencies (uses rg, jq, awk)

3. **Content parsing (grep) for detection** (not index/database)
   - Sufficient at current scale (172 investigations, 30 kn entries)
   - Zero maintenance overhead
   - Tested in ws4z.7 citation investigation (<100ms for 138 files)

4. **Density thresholds over time intervals**
   - "3+ investigations on topic" not "weekly review"
   - Matches actual patterns discovered in ws4z.8

## Alternatives Considered

### Separate Commands
```bash
kb synthesis-check
kb stale-check
kb drift-check
kb promote-check
```
**Rejected because:** Four commands to learn, no unified "show me everything" view, inconsistent with kb pattern.

### Query-Style Interface
```bash
kb query "synthesis candidates"
kb query "stale decisions"
```
**Rejected because:** Requires query parsing, less discoverable, harder to document.

### Go Implementation from Start
**Rejected because:** kb is currently shell-based, shell allows faster iteration on heuristics, can migrate to Go later if needed.

## Consequences

### Positive
- Single discoverable command for all reflection needs
- Low maintenance (no indexes or databases)
- Easy to tune detection thresholds
- Aligns with daemon integration path (daemon runs `kb reflect`, surfaces summary)

### Negative
- Shell has limits for complex parsing (may need Go migration)
- Drift detection has high false positive rate (requires human validation)
- Cross-platform concerns (macOS vs Linux sed/awk differences)

## Implementation Notes

- **MVP Scope:** synthesis and promote types first (simplest detection)
- **Output Format:** Human-readable with actionable suggestions; `--json` flag for machine-readable
- **Performance Target:** <5 seconds for all types on 200+ artifacts
- **Success Metric:** <20% false positive rate

## Related

- **Design Investigation:** `.kb/investigations/2025-12-21-design-kb-reflect-command-specification.md`
- **Parent Epic:** orch-go-ws4z "System Self-Reflection - Temporal Pattern Awareness"
- **Input Investigations:** ws4z.7, ws4z.8, ws4z.9, ws4z.10, 4kwt.8
- **Prior Decision:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` (5+3 artifact types)
