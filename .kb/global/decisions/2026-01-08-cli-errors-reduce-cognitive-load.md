# Decision: CLI Errors Should Reduce Cognitive Load

**Date:** 2026-01-08
**Status:** Accepted
**Scope:** All CLIs in orch ecosystem (bd, kb, orch, skillc)

## Decision

CLI errors should reduce cognitive load, not add to it. When a command fails:
1. If a sensible default exists → use it (or prompt to confirm)
2. If the fix is known → show the corrected command
3. If context is missing → suggest how to get it

## Context

Recurring friction: orchestrators (both human and AI) hit terse CLI errors that require parsing, understanding, and figuring out the fix. Example: `skillc deploy meta/orchestrator` fails with "Error: --target is required" - the CLI knows what's wrong but doesn't help.

AI agents hit this harder than humans - no muscle memory, no implicit context from prior sessions.

## The Principle

**Errors are teaching moments.** Every error should either:
- Fix itself (sensible default)
- Teach the fix (show corrected command)
- Guide discovery (suggest diagnostic steps)

## Examples

| Current | Improved |
|---------|----------|
| `Error: --target is required` | `Error: --target is required. Example: skillc deploy --target ~/.claude/skills/ meta/orchestrator` |
| `Error: no issue found matching "ok-9ph0"` | `Error: Issue ok-9ph0 not found in current repo. Try: cd ~/orch-knowledge && bd show ok-9ph0` |
| `Error: spawn blocked` | Already good - shows wall of guidance |

## Application

When building or modifying CLI commands:
1. **Default check:** Is there a sensible default? Use it or suggest it.
2. **Context check:** Does the error have enough context to suggest the fix? Include it.
3. **Agent check:** Would an agent with no memory of prior commands understand how to proceed?

## Provenance

- Trigger: `skillc deploy` friction (kn-252dd3, issue ok-9ph0)
- Pattern recognized: Same friction across multiple CLIs
- Session: 2026-01-08 orchestrator session with Dylan
