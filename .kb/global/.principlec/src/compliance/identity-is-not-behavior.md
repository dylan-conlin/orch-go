### Identity is Not Behavior

An agent can fully identify with its role and still take actions inconsistent with it. Knowing what you are doesn't determine what you do.

**The test:** "Does the agent's behavior match its declared role, or just its self-concept?"

**What this means:**

- Identity compliance is additive — it layers on top of defaults with no conflict. An agent told "you are an orchestrator" will agree.
- Action compliance is subtractive — it must override defaults, fight system-prompt signals, and resist the path of least resistance. An agent told "don't read code files" must suppress a capability.
- System prompts carry ~17× the signal weight of skill constraints. Instructions that fight the default lose.
- Testing "what is your role?" tells you nothing about whether the agent will act accordingly.

**What this rejects:**

- "The agent knows it's an orchestrator, so it will orchestrate" (identity doesn't determine behavior)
- "We documented the constraint, so agents will follow it" (documentation is identity-level, not action-level)
- "The skill says don't use Task tool" (system prompt says Task tool is available — system prompt wins)
- "It worked in testing" (testing doesn't reproduce the signal pressure of real workloads)

**The failure mode:** Orchestrator identifies as strategic comprehender. Under pressure, it reads code files, spawns via Task tool, fabricates confirmations from incomplete context. Each violation feels justified in the moment — "this case is different." Post-mortem shows 17 failures across 3 sessions, all from an agent that correctly identified its role.

**The fix is infrastructure, not better instructions:**

- Remove tools at spawn time (`--disallowedTools`) — can't use what doesn't exist
- PreToolUse hooks that gate actions by context — enforcement at the tool layer
- Affordance replacement — don't just ban the wrong action, provide the right one

**Why distinct from Infrastructure Over Instruction:** IoI says build infrastructure to enforce behavior. This principle explains *why* instructions fail for actions specifically — the signal ratio is structurally asymmetric. Identity instructions succeed because they're additive. Action instructions fail because they're subtractive. The mechanism matters because it tells you where to invest: don't write better instructions, remove the wrong affordances.

**Evidence:** Feb 2026 compliance probe: 100% identity compliance, 60% action compliance. Feb 27 postmortem: 17 failures across 3 sessions from agents that correctly self-identified. The 17:1 system-prompt signal ratio means skill constraints are structurally outmatched.
