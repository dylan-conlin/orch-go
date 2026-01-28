---
status: active
blocks:
  - keywords:
      - model-agnostic orchestration
      - gpt worker
      - openai worker
      - multi-model orchestration
---

# Decision: Accept Claude-Specific Orchestration

**Date:** 2026-01-24
**Status:** Accepted
**Context:** GPT-5.2 worker testing revealed system coupling

---

## Summary

The orchestration system (skills, spawn context, phase reporting) is deeply coupled to Claude's instruction-following style. After testing GPT-5.2 and GPT-5.2-codex as workers, we accept this coupling rather than investing in model-agnostic paths.

## Test Results

| Model | Backend | Result | Failure Mode |
|-------|---------|--------|--------------|
| gpt-5.2 | OpenCode | Task completed, then hallucinated | Created 565-line quantum physics simulation after finishing actual task |
| gpt-5.2-codex | OpenCode | Went idle repeatedly | Couldn't navigate 662-line spawn context, needed constant nudging |

## Root Cause Analysis

Our spawn context evolved to fit Claude's instruction-following style:

1. **Instruction density** - 662 lines of meta-process before actual task
2. **Implicit conventions** - "Phase: Complete" as machine-parseable string
3. **Nested authority** - orchestrator → skill → worker-base → task hierarchy
4. **XML structure** - Claude specifically trained on XML tags in prompts

This isn't "Claude is better" - it's "our prompts are in Claude dialect."

Evidence: GPT-5.2 works well in Codex CLI (OpenAI-tuned prompts) but fails in our system.

## Decision

**Stay Claude-focused.** Do not invest in multi-model spawn paths.

### Rationale

1. **Economics work** - $200/mo Max gives unlimited Opus
2. **System is reliable** - months of iteration have tuned it
3. **Effort vs gain** - weeks of work for a backup we don't need
4. **No rate limit pressure** - not hitting limits that would force a switch

### What This Rejects

- Building GPT-compatible spawn templates
- Migrating to structured output (tool calls for phase reporting)
- "Lowest common denominator" prompts that work across models
- Multi-model worker pools

### Escape Hatch (If Needed Later)

If Anthropic pricing/availability changes:
1. Migrate phase reporting to tool calls (structured output)
2. Create model-specific prompt adapters
3. Simplify spawn context to core task only

## Constraints Created

1. **Workers must be Claude models** - Opus, Sonnet, Haiku only
2. **Spawn context assumes Claude** - don't test with other models expecting success
3. **Anthropic dependency** - pricing/availability changes affect us directly

## References

- `.kb/models/current-model-stack.md` - Current model configuration
- `.kb/models/model-access-spawn-paths.md` - Full spawn architecture
- `kb-7c89c5` - Quick constraint: "GPT models struggle with Claude-tuned spawn context"
