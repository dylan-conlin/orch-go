### Session Amnesia

Every pattern in this system compensates for Claude having no memory between sessions.

**The test:** "Will this help the next Claude resume without memory?"

**What this means:**

- State must externalize to files (workspaces, artifacts, decisions)
- Context must be discoverable (standard locations, descriptive naming)
- Resumption must be explicit (Phase, Status, Next Step)

**What this rejects:**

- "It's in the conversation history" (next Claude won't see it)
- "The code is self-documenting" (next Claude won't remember reading it)
- "I'll document it later" (context is lost, later never comes)

This is THE constraint. When principles conflict, session amnesia wins.
