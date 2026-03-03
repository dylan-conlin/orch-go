# Probe: Orchestrator Skill Behavioral Compliance — Identity vs Action Compliance Gap

**Status:** Complete
**Date:** 2026-02-24
**Model:** behavioral-grammars
**Triggered by:** orch-go-1178

## Question

The model claims "Framing cues override skill instructions" and "Frame collapse is prevented by restricting action space, not just guidelines." If this is true, why do orchestrators comply with identity declarations ("I'm your orchestrator") but fail to comply with action constraints ("use orch spawn, not Task tool")? Is the action space actually restricted, or merely described?

## What I Tested

### Test 1: Skill Structure Analysis — Where Are Action Constraints?

Read the deployed orchestrator SKILL.md (640 lines, ~8K tokens). Mapped the exact location of action constraints:

- **Line 68**: Pre-response check: "Spawn method gate: Am I about to use Task tool? → STOP"
- **Line 75**: One-line critical note: "The Task tool is NOT how orchestrators spawn workers"
- **Lines 582-609**: "ABSOLUTE DELEGATION RULE" and "Tool Action Space" table (Section 6 of 7)

Measured salience positioning:
- Action constraints first appear at line 68 (10% into document) — but inside a 9-item checklist
- The standalone critical note is at line 75 (12% into document) — a single line after a 40-row table
- The full Tool Action Space table is at line 594 (88% into document)

### Test 2: Claude Code System Prompt — Competing Instructions

Examined the system prompt that every Claude Code session receives. Found direct competition:

**System prompt says (high salience, persistent):**
- "Launch a new agent to handle complex, multi-step tasks autonomously" (Task tool description)
- "Use the Task tool with specialized agents when the task at hand matches the agent's description"
- "Launch multiple agents concurrently whenever possible"
- Available agent types listed: Bash, general-purpose, Explore, Plan, etc.

**Orchestrator skill says (low salience, one-shot):**
- "The Task tool is NOT how orchestrators spawn workers" (1 line at line 75)
- "Use `bd create -l triage:ready` (primary) or `orch spawn SKILL "task"` (exception)" (1 line)

**Signal ratio**: The system prompt has ~500 words promoting Task tool usage. The skill has ~30 words constraining it. That's roughly a 17:1 competing signal ratio.

### Test 3: Identity vs Action Instruction Types

Compared the structural characteristics of identity declarations vs action constraints:

**Identity declarations:**
- "You are a strategic comprehender" (line 33)
- "ORIENT → DELEGATE → RECONNECT (never implement)" (line 35)
- "Synthesis Is Orchestrator Work (Not Spawnable)" (lines 96-108)
- These are ADDITIVE — they don't conflict with anything in the system prompt
- Claude naturally cooperates with role framing

**Action constraints:**
- "Don't use Task tool" — CONFLICTS with system prompt promoting Task tool
- "Use orch spawn instead" — requires overriding a built-in affordance
- "Use orch complete, not bd close" — conflicts with `bd close` being documented in CLAUDE.md and beads guidance
- These are SUBTRACTIVE — they require suppressing default behaviors

### Test 4: Temporal Positioning Analysis

Examined when instructions are processed:

1. Skill content: injected at session start or in SPAWN_CONTEXT.md (early in context)
2. System prompt: injected and reinforced on every turn (persistent, recent)
3. User message "spawn a worker": triggers pattern-matching against BOTH skill and system prompt

The system prompt's Tool descriptions are more recent in context than the skill's constraints, giving them higher attention weight.

### Test 5: Prior Evidence Cross-Reference

- **Probe 2026-02-16 (Orientation Redesign)**: Confirmed skill organized around orchestrator identity, not Dylan's needs. Four critical moments scattered across sections.
- **Probe 2026-02-17 (Injection Path Trace)**: Found 5 injection paths with init-time caching bug. Stale versions being served.
- **Investigation 2026-01-06 (18% Completion Rate)**: Low completion is by design, but event correlation broken.
- **Constraint**: "LLM guidance compliance requires signal balance - overwhelming counter-patterns (56:13 ratio) drowns specific exceptions"

## What I Observed

### Finding 1: Action Constraints Are Described, Not Enforced

The model claims "Frame collapse is prevented by restricting action space." But examining the actual mechanism:

- The "Tool Action Space" is a **description in a markdown table**, not a tool-layer enforcement
- The Task tool remains fully available and functional for orchestrator sessions
- `bd close` remains fully available and functional
- No hook, plugin, or tool-layer gate prevents orchestrators from using prohibited tools
- The "restriction" is instructional, not infrastructural

**Verdict: The model's claim is aspirational, not factual.** The action space is *described as restricted* but not *actually restricted*.

### Finding 2: The Competing Instruction Hierarchy is Structurally Unwinnable

The Claude Code system prompt occupies a privileged position:
- It's injected by the platform (highest authority in Claude's instruction hierarchy)
- It describes available tools including Task tool with explicit encouragement
- It's reinforced on every turn
- It pattern-matches strongly against user requests like "spawn a worker"

The orchestrator skill occupies a subordinate position:
- It's injected as user-level content (CLAUDE.md or skill system)
- Its action constraints are 1-2 lines buried in 640 lines of content
- It's static (injected once, not reinforced)
- Its pattern doesn't match the user's request framing

Under Claude's instruction hierarchy (system > user > assistant), the system prompt's "use Task tool for subagents" has structural priority over the skill's "don't use Task tool."

### Finding 3: Identity Compliance Is Not Predictive of Action Compliance

This is the central finding. Identity and action compliance are mechanistically different:

| Dimension | Identity Declaration | Action Constraint |
|-----------|---------------------|-------------------|
| Relationship to defaults | Additive (no conflict) | Subtractive (conflicts with system prompt) |
| Framing match | Congruent with role-play | Incongruent with built-in affordances |
| Signal strength | Repeated throughout skill | 1-2 lines in 640 |
| Processing mode | Semantic (who am I?) | Procedural (what do I do?) |
| Verification method | Self-report ("I'm an orchestrator") | Behavioral (did it use orch spawn?) |
| Competing instructions | None | System prompt actively promotes alternatives |

The agent can hold identity ("I'm an orchestrator") while violating action constraints ("use Task tool instead of orch spawn") because these operate on different dimensions. Identity is belief. Action is affordance selection under competing pressures.

### Finding 4: Prompt Engineering Research Confirms This Pattern

Recent research (ICLR 2025, AgentSpec ICSE 2026) confirms:
- Current LLMs lack native ability to prioritize competing instruction sources
- "Plan-Then-Execute" pattern constrains agents to fixed action plans
- Policy compilers (PCAS) provide deterministic enforcement via reference monitors intercepting actions before execution
- Prompt-level constraints provide no enforcement guarantees

The field consensus: **prompts describe desired behavior; infrastructure enforces it**.

## Model Impact

### Confirms

1. **"Framing cues override skill instructions"** — Confirmed. The system prompt's framing of Task tool as "how you spawn subagents" overrides the skill's instruction to use orch spawn. The user saying "spawn a worker" matches the Task tool framing, not the orch spawn instruction.

2. **"NOT the fix: Adding more ABSOLUTE DELEGATION RULE warnings"** — Confirmed. The rule exists at multiple levels (pre-response checks, dedicated section, hard constraints). Adding more warnings won't help because the problem is structural, not informational.

### Extends

3. **The model says "Frame collapse is prevented by restricting action space, not just guidelines."** This claim is ASPIRATIONAL. The action space is described as restricted but the restriction is implemented as a guideline, not infrastructure. **Proposed model extension:** Distinguish between *described action space* (prompt-level, current state) and *enforced action space* (tool-layer, aspirational). The model should note that until infrastructure enforcement exists, prompt-level action space restrictions operate as guidelines with the same failure modes as any other prompt instruction.

4. **New failure mode not in model: Competing Instruction Hierarchy.** The model documents frame collapse (orchestrator drops to worker) and self-termination (orchestrator tries to /exit). It does not document the **competing instruction hierarchy** failure where the system prompt actively promotes behaviors the skill tries to suppress. This is a distinct failure mode from frame collapse — the orchestrator doesn't "drop into worker mode," it uses the wrong tool for orchestrator-level work while maintaining orchestrator identity.

### Contradicts

5. **Implicit model assumption: Skill content is the primary instruction source.** The model's "Why This Fails" section treats skill instructions as the primary behavioral influence. In reality, the Claude Code system prompt has structural priority AND higher salience. The skill is a secondary instruction source competing against a primary one. This reframes the entire problem from "how do we write better skill instructions" to "how do we work within (or modify) the instruction hierarchy."
