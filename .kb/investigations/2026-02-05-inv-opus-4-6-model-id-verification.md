# Investigation: Opus 4.6 Model ID Verification

**Date:** 2026-02-05
**Status:** Complete
**Beads ID:** orch-go-21321

## Question

What is the exact model ID for Claude Opus 4.6 to update the DefaultModel constant?

## Summary

**Finding:** Claude Opus 4.6 uses simplified model ID `claude-opus-4-6` (no date stamp)

**Evidence:** Anthropic documentation at https://docs.anthropic.com/en/docs/about-claude/models

**Significance:** Anthropic has changed naming convention - newer models use version-only format (`claude-opus-4-6`) instead of date-stamped format (`claude-opus-4-5-20251101`)

## Findings

### Finding 1: Opus 4.6 Model ID is `claude-opus-4-6`

**Evidence:** From Anthropic docs table:

```
| Feature | Claude Opus 4.6 | Claude Sonnet 4.5 | Claude Haiku 4.5 |
| Claude API ID | claude-opus-4-6 | claude-sonnet-4-5-20250929 | claude-haiku-4-5-20251001 |
| Claude API alias | claude-opus-4-6 | claude-sonnet-4-5 | claude-haiku-4-5 |
```

**Significance:** The model ID is `claude-opus-4-6` - no date stamp like previous Opus 4.5 (`claude-opus-4-5-20251101`). This represents a naming convention change by Anthropic.

**Tested:** No (documentation-based finding)

### Finding 2: Opus 4.5 is now in "Legacy Models" section

**Evidence:** Anthropic docs list `claude-opus-4-5-20251101` under "Legacy models" section with recommendation: "we recommend migrating to current models for improved performance"

**Significance:** Opus 4.5 is deprecated and should be replaced with Opus 4.6 as the default.

**Tested:** No (documentation-based finding)

### Finding 3: Opus 4.6 has improved capabilities

**Evidence:** From Anthropic docs:

- "Our most intelligent model for building agents and coding"
- Supports adaptive thinking (Opus 4.6 only, not available in Sonnet or Haiku)
- Knowledge cutoff: May 2025 (same as 4.5)
- Training data cutoff: Aug 2025 (same as 4.5)
- Same pricing as 4.5: $5 input, $25 output per MTok

**Significance:** Opus 4.6 is a direct replacement for 4.5 with no cost increase and additional capabilities.

**Tested:** No (documentation-based finding)

## Synthesis

Opus 4.6 represents a straightforward upgrade path:

1. Model ID: `claude-opus-4-6` (simplified naming)
2. Drop-in replacement for 4.5 (same pricing, improved capabilities)
3. No breaking changes in API format

Update targets:

- `pkg/model/model.go` - DefaultModel constant and aliases
- `.kb/guides/model-selection.md` - Documentation references
- `.kb/quick/entries.jsonl` - kb constraint
- Any hardcoded references to `claude-opus-4-5-20251101`

## Sources

- [Anthropic Models Documentation](https://docs.anthropic.com/en/docs/about-claude/models) - Accessed 2026-02-05
- Current codebase: `pkg/model/model.go:20-23`
