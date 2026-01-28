---
status: active
---

# Decision: Decision Gate for Architect Decisions

**Date:** 2026-01-28
**Status:** Active
**Decision:** Implement decision gate - a soft gate that blocks spawns when they conflict with existing architect decisions

## Context

The coaching plugin failure revealed that architect decisions had no teeth:
- Architect produced recommendations
- Orchestrators/agents spawned tactical work anyway
- 18 investigations, contradictory conclusions, no enforcement
- Decisions existed but didn't constrain future work

Fresh sessions would spawn investigation #19 without knowing investigations #1-18 existed.

## Solution: Decision Gate

Pre-spawn check that blocks spawns when they match a decision's `blocks` keywords:

```
$ orch spawn investigation "debug coaching plugin"

⚠️ ⚠️ ⚠️  DECISION CONFLICT  ⚠️ ⚠️ ⚠️

Decision: Coaching Plugin Disabled
Matched on: coaching plugin, worker detection

To proceed, acknowledge this decision:
  orch spawn --acknowledge-decision 2026-01-28-coaching-plugin-disabled ...
```

## How It Works

1. Decisions declare what they block via YAML frontmatter:
   ```yaml
   ---
   status: active
   blocks:
     - keywords:
         - coaching plugin
         - worker detection
       patterns:
         - "**/coaching*"
   ---
   ```

2. `orch spawn` checks kb decisions before spawning
3. If task matches blocked keywords → spawn blocked
4. Override requires explicit `--acknowledge-decision <id>` flag
5. Overrides logged to `~/.orch/decision-overrides.jsonl`

## Design Principles

- **Soft gate, not hard block** - Override is possible but visible and intentional
- **Teeth come from visibility** - Can't accidentally ignore a decision
- **Decisions compound** - Past judgment constrains future work
- **Override patterns are signal** - Frequent overrides suggest decision needs updating

## Trade-offs

- Requires decisions to declare `blocks` frontmatter (opt-in)
- Keyword matching is fuzzy (may have false positives)
- Can be overridden (intentionally - hard blocks would be brittle)

## References

- `orch-go-21002` - Implementation issue
- `.kb/decisions/2026-01-28-coaching-plugin-disabled.md` - First decision to use the gate
- `cmd/orch/spawn_validation.go` - Implementation
