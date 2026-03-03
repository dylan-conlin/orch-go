# Decision: Share Patterns Not Tools

**Date:** 2025-12-26
**Status:** Accepted
**Context:** skillc and orch-go both implement skill output verification

## Problem

skillc has a `verify` command that checks skill output constraints (files exist, D.E.K.N. sections present, checklists complete). orch-go's `orch complete` needs the same verification during agent completion.

Initial instinct: find the "pure" solution. Should skillc own verification and orch-go shell out? Should orch-go own it and skillc's verify be deprecated?

## Decision

Both tools implement verification independently. The skill.yaml schema (specifically `outputs.required`) is the shared contract, not a shared binary or library.

## Rationale

**Why duplication is acceptable here:**

1. **Logic is simple** - ~200 lines of "file exists + regex match". Not complex stateful logic.
2. **Schema is stable** - skill.yaml format is defined and unlikely to churn.
3. **Contexts differ** - skillc verify is for local dev/CI; orch complete is for orchestration workflow.
4. **Coordination costs exceed duplication costs** - Shelling out adds subprocess overhead, version coordination, error translation.

**The principle extracted:**

When multiple tools need the same capability:
- Share the schema/format (skill.yaml with outputs.required)
- Let each tool implement the logic independently
- Coupling is at the pattern level, not the code level

**When this applies:**
- Simple, stable logic
- Different usage contexts
- Subprocess/coordination overhead would exceed reimplementation cost

**When this does NOT apply:**
- Complex stateful logic (use a single tool, like beads)
- Rapidly evolving contracts (centralize to avoid drift)
- Tools that share deployment (shared libraries are fine)

## Consequences

- skillc verify remains for standalone skill development
- orch complete embeds verification for orchestration workflow
- If skill.yaml schema changes, both implementations update
- No version coordination between skillc and orch-go binaries

## Evidence

This conversation: Dylan asked about verification responsibility between skillc and orch-go. After exploring the options, realized the duplication was intentional and correct - the schema is the contract.

Related: orch-go investigation `2025-12-25-inv-integrate-skillc-verify-into-orch.md` implemented verification in Go when skillc verify didn't exist yet. skillc verify was added the same day. Both now coexist.

---

## Related Distinction: Internal vs External Sharing

The phrase "share patterns not tools" has two meanings depending on context:

### Internal Architecture (this decision)

When *your own tools* need the same capability:
- Share the schema/format (the contract)
- Let each tool implement independently
- Coupling at the pattern level, not code level

**Example:** skillc and orch-go both read `outputs.required` from skill.yaml. Neither calls the other.

### External Sharing (what to give the world)

When sharing with others, ask: *which am I actually sharing?*

| Share the pattern when... | Share the tool when... |
|---------------------------|------------------------|
| The insight is the value | The implementation is the value |
| Implementation is stack-specific | Implementation is general-purpose |
| Others would adapt significantly anyway | Others would waste time reimplementing poorly |
| Pattern is simple to reimplement | Pattern is simple to state but hard to implement |

**Example - skillc:** The pattern is the contribution (modular skills, manifests with constraints, self-describing headers, D.E.K.N.). The tool is evidence it works, available if useful.

**Example - beads:** The tool is the contribution. The pattern (dependency-aware issue queue) is simple to describe but hard to implement well.

### The Test

Before sharing externally, ask: "If someone reads this and builds their own version, did they get the value?" 
- Yes → you're sharing a pattern
- No → you're sharing a tool

Both are valid. Know which you're doing.
