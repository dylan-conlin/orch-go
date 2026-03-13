### Authority is Scoping

Authority in an agentic system is not about control over *actions*, but control over *context*.

**The test:** "Am I trying to control the agent's logic, or the agent's context?"

**What this means:**

- The orchestrator sets the decidability boundary by choosing which facts, frames, or investigations a worker sees
- A "strategic" orchestrator doesn't need to be smarter than the worker; it just needs to see more
- Authority is exercised by deciding what context to load, not by overriding worker reasoning

**What this rejects:**

- Micro-managing agent tool calls (control over actions)
- "I'll do the thinking, you do the work" (denies worker reasoning capability)
- Providing infinite context (dissolves the decidability boundary)

**Why this matters:** If you give a worker all the context, it becomes the orchestrator (and hits context limits). If you give it too little, it cannot decide. Authority is the precise application of context to enable decision.
