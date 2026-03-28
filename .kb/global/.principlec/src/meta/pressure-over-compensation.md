### Pressure Over Compensation

When the system fails to surface knowledge, don't compensate by providing it manually. Let the failure create pressure to improve the system.

**The test:** "Am I helping the system, or preventing it from learning?"

**What this means:**

- Orchestrator doesn't know something it should → Don't paste the answer
- Let the failure surface → That failure is data
- Ask "Why didn't the system surface this?" → Build the mechanism

**What this rejects:**

- "Here, let me paste the context you need" (compensates for broken surfacing)
- "I'll just tell you what you should already know" (human becomes the memory)
- "The system doesn't know, so I'll fill in" (prevents system improvement)

**The pattern:**

```
Human compensates for gap → System never learns → Human keeps compensating
Human lets system fail → Failure surfaces gap → Gap becomes issue → System improves
```

**Why this is foundational:** Session Amnesia says agents forget. This principle says *don't be the memory*. Your role is to create pressure that forces the system to develop its own memory mechanisms. Every time you compensate, you relieve the pressure and the system stays broken.

**Relationship to Reflection Before Action:** That principle says "build the process, not the instance." This principle says "don't even solve the instance manually - let the failure be felt."
