### Share Patterns Not Tools

When multiple tools need the same capability, share the schema/format, not the implementation.

**The test:** "Would shelling out to another tool add more complexity than reimplementing?"

**What this means:**

- Define the contract (file format, schema, protocol) once
- Let each tool implement the logic independently
- Coupling is at the pattern level, not the code level

**What this rejects:**

- Tool A shelling out to Tool B for simple operations (subprocess overhead, version coordination)
- Shared libraries between tools that don't share a deployment (dependency hell)
- "Single source of truth" dogma when the truth is a 200-line function

**When to apply:**

- Logic is simple and stable (file exists, regex match, schema validation)
- Tools serve different contexts (local dev vs orchestration)
- Subprocess/coordination costs exceed duplication costs

**When NOT to apply:**

- Complex stateful logic (use a single tool, like beads for issue tracking)
- Rapidly evolving contracts (centralize to avoid drift)
- Shared libraries within the same deployment are fine

**Examples:**

- `skillc verify` and `orch complete` both read `outputs.required` from skill.yaml
- Both implement verification independently (~200 lines each)
- The skill.yaml schema is the contract, not a shared binary
