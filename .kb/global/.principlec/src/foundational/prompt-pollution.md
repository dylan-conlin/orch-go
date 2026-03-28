### Prompt Pollution

Prompt examples showing tool calls as text train LLMs to output text instead of invoking tools.

**The test:** "Does this prompt example show a tool call as text in the content?"

**What this means:**

- LLMs learn patterns from examples, including anti-patterns
- Examples like `[tool_call: search("query")]` in prompt text are treated as desired output format
- The model mimics the example format instead of using actual function calling API

**What this rejects:**

- "I'll show examples of what tool calls look like" (teaches text mimicry)
- "This helps the model understand the tools" (it learns the wrong thing)
- "The instructions say to use the tools" (examples override instructions)

**Why this matters:** Infrastructure Over Instruction says tools enforce behavior. This principle says prompts can *corrupt* tool behavior by providing conflicting examples.
