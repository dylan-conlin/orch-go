### Surfacing Over Browsing

Bring relevant state to the agent. Don't require navigation.

**Why:** Agents lack persistent memory and spatial intuition. Every file read costs context. Browsing that's cheap for humans is expensive for agents.

**Pattern:** Commands answer "what's relevant now?" not "here's all the data."

**Examples:**

- `bd ready` surfaces unblocked work (vs `bd list`)
- SessionStart hook injects context (vs manual file reads)
- SPAWN_CONTEXT.md contains everything needed (vs agent searching)

**The test:** Does this tool/command require the agent to navigate, or does it surface what matters?
