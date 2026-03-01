# Agent Framework Behavioral Constraints: Practitioner Landscape

**Status:** Complete
**Date:** 2026-03-01
**Beads:** orch-go-pasm

## D.E.K.N. Summary

- **Delta:** Surveyed 8 agent frameworks for behavioral constraint patterns. Found a universal gap: all frameworks enforce at the tool/output layer, none enforce at the decision layer. The field converges on "interceptor" patterns but none solve the competing-instruction-hierarchy problem orch-go faces.
- **Evidence:** Framework documentation, API specs, and academic papers (AgentSpec ICSE '26, NeMo Guardrails EMNLP '23)
- **Knowledge:** Three enforcement tiers exist across the industry: prompt-level (weakest), output-validation (medium), action-interception (strongest). No framework enforces behavioral intent — only observable actions.
- **Next:** Consider AgentSpec-style reference monitor pattern for orch spawn vs Task tool enforcement. The hooks + permission rules approach in Claude Agent SDK is the closest production pattern to what orch-go needs.

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|---|---|---|---|
| .kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md | extends | yes | None — that investigation identified the problem; this one surveys how others solve it |
| .kb/models/orchestrator-session-lifecycle/probes/2026-02-24-probe-orchestrator-skill-behavioral-compliance.md | extends | yes | None — probe confirmed prompt-level constraints fail; this surveys alternatives |

## Question

How do production agent frameworks handle keeping agents within behavioral bounds? What patterns exist, what fails, and how does the industry approach map to orch-go's specific problem (agents ignoring skill-level action constraints when system-prompt-level affordances compete)?

## Findings

### Finding 1: The Three Tiers of Enforcement

Across all frameworks surveyed, behavioral constraints fall into three enforcement tiers:

| Tier | Mechanism | Enforcement Strength | Examples |
|---|---|---|---|
| **Prompt-level** | System prompt instructions, role descriptions | Weakest — advisory only, no runtime enforcement | AutoGPT constraints, CrewAI role definitions, orch-go skill instructions |
| **Output-validation** | Post-generation validators that check/reject/retry | Medium — catches violations after they happen | Guardrails AI RAIL specs, CrewAI task guardrails, OpenAI output guardrails |
| **Action-interception** | Pre-execution interceptors that block/modify actions | Strongest — prevents violations before they execute | NeMo Guardrails dialog rails, OpenAI tool guardrails, Claude Agent SDK permission hooks, AgentSpec reference monitors |

**Key insight:** No framework operates at the "decision" layer — they all intercept at observable boundaries (input, output, tool calls). The internal reasoning that leads an agent to choose Tool A over Tool B is not constrainable by any production framework.

### Finding 2: Framework-by-Framework Analysis

#### LangChain / LangGraph

**Approach:** Middleware-based guardrails + structural graph constraints

- **Guardrails middleware** intercepts execution at strategic points (before agent starts, after completion, around model/tool calls)
- **Two implementation strategies:** Rule-based (regex, keyword matching — fast, deterministic) and LLM-driven (semantic evaluation — flexible, expensive)
- **Loop prevention:** `max_iterations` parameter caps agent turns; hard stop prevents runaway loops
- **Human-in-the-loop:** LangGraph's `interrupt()` function pauses graph execution, persists state, waits for human decision (approve/edit/reject)
- **What works:** Graph structure naturally constrains agent flow — nodes define what's possible, edges define transitions. The agent can't take actions not represented in the graph.
- **What fails:** Guardrails are opt-in and post-hoc. Within a node, the agent has full freedom. No way to prevent an agent from choosing one tool over another at the reasoning level.

**Relevance to orch-go:** LangGraph's graph structure is the closest analog to orch-go's skill system — both define what an agent *should* do. But neither enforces it at the decision layer.

#### CrewAI

**Approach:** Role-based constraints + task guardrails

- **Role definitions** specify agent's backstory, goal, and expected behavior (prompt-level, Tier 1)
- **Task guardrails** validate outputs after generation: function-based (deterministic) or LLM-as-Judge (semantic)
- **Iteration limits** prevent infinite loops (`max_iter` on tasks)
- **Bounded delegation:** Agents can delegate to other agents, but only within defined crew structure
- **What works:** Task guardrails catch bad outputs reliably. Crew structure limits delegation scope.
- **What fails:** Role descriptions are advisory — an agent can hold the "researcher" identity while behaving like an "implementer." Same identity-vs-action gap orch-go's probe documented.

**Relevance to orch-go:** CrewAI's identity-vs-action gap mirrors orch-go's exactly. Their solution (output validation) is necessary but doesn't prevent the wrong tool from being called in the first place.

#### AutoGPT

**Approach:** Loop architecture + resource limits

- **Goal → Plan → Execute → Reflect → Iterate** loop with configurable limits
- **The "reflection" step** (agent critiques own output) is the primary self-correction mechanism
- **What works:** Hard resource limits (token budgets, iteration caps) prevent runaway costs
- **What fails:** The reflection step is notoriously unreliable — agents spiral into loops where reflection generates identical plans. The self-prompting mechanism can't distinguish between "making progress" and "repeating the same failed approach." This is the most well-documented failure mode in the agent ecosystem.

**Relevance to orch-go:** orch-go already has this pattern via coaching plugins that inject "loop detected" messages. The field consensus is that self-reflection alone is insufficient — external enforcement is required.

#### NVIDIA NeMo Guardrails

**Approach:** Event-driven dialog flow enforcement via Colang DSL

- **Colang** is a domain-specific language for defining conversational flows and safety constraints
- **Architecture:** Acts as a proxy between user and LLM, intercepting both input and output
- **Dialog rails** determine whether an action should execute, whether the LLM should be invoked, or whether a predefined response should be used instead
- **Event-driven:** All interactions are events (user input, LLM response, action trigger, guardrail trigger) flowing through an enforcement pipeline
- **What works:** Colang flows define *permitted conversation paths* — anything not in a flow is blocked. This is genuine action-space restriction, not just guidelines.
- **What fails:** Designed for conversational AI, not agent tool-use. Flows are dialogue patterns, not tool-selection patterns. Adapting to agent orchestration requires significant conceptual mapping.

**Relevance to orch-go:** NeMo's "anything not in a flow is blocked" pattern is the inverse of orch-go's current approach ("everything is allowed unless a guideline says otherwise"). This is the most architecturally different approach.

#### OpenAI Agents SDK

**Approach:** Tripwire guardrails + tool guardrails

- **Input guardrails** validate user input before agent processes it; can run in parallel with agent execution or blocking
- **Output guardrails** validate final agent response after generation
- **Tool guardrails** wrap individual tools with input/output validation — run before and after each tool call
- **Tripwire mechanism:** When violation detected, raises `GuardrailTripwireTriggered` exception, immediately halts agent execution
- **Parallel vs blocking modes:** Parallel = lower latency but agent may consume tokens before cancellation; Blocking = higher latency but agent never starts if input fails
- **What works:** Tool guardrails are the most granular — you can validate each tool call individually. Tripwire-halt is fast and deterministic.
- **What fails:** Guardrails only apply to function tools, not hosted or built-in execution tools. Can't prevent the agent from *choosing* the wrong tool — only validate *after* the choice is made.

**Relevance to orch-go:** The tripwire pattern is directly applicable. If orch-go could intercept when an orchestrator agent attempts to use Task tool, it could halt and redirect to orch spawn. But this requires tool-layer infrastructure that Claude Code doesn't expose to skill authors.

#### Claude Agent SDK (Anthropic)

**Approach:** Layered permission evaluation with hooks

- **4-step permission evaluation:** Hooks → Permission Rules (deny/allow/ask) → Permission Mode → canUseTool callback
- **Permission modes:** `default` (no auto-approvals), `acceptEdits` (auto-approve file operations), `bypassPermissions` (all approved), `plan` (no tool execution)
- **Declarative rules:** `settings.json` defines allow/deny rules evaluated before permission mode
- **Hooks:** Custom code that runs at key lifecycle points — can allow, deny, or modify tool requests
- **Dynamic modes:** Permission mode can change mid-session (e.g., start restrictive, loosen after trust builds)
- **What works:** The layered evaluation (hooks → rules → mode → callback) provides defense-in-depth. Hooks run custom code, not just declarations — genuine enforcement.
- **What fails:** Only controls tool-use permissions, not tool *selection*. Can prevent an agent from using a tool but can't force it to prefer one tool over another. Also, `bypassPermissions` propagates to all subagents and can't be overridden.

**Relevance to orch-go:** This is the closest production pattern to what orch-go needs. The hooks mechanism could theoretically intercept Task tool usage by orchestrators and redirect to orch spawn. But orch-go operates within Claude Code (not the Agent SDK), and Claude Code's hook system is more limited.

#### AgentSpec (ICSE 2026)

**Approach:** Reference monitor with declarative constraint language

- **DSL for constraints:** Rules specify triggers, predicates, and enforcement actions
- **Reference monitor pattern:** Intercepts proposed actions *before execution*, evaluates against rules
- **Enforcement mechanisms:** Action termination, user inspection, corrective invocation, self-reflection
- **Results:** >90% prevention of unsafe code executions, 100% compliance in embodied agent tasks
- **Implementation:** Modular framework integrating with LangChain, intercepting key execution stages
- **What works:** This is the strongest enforcement pattern in the literature. By intercepting *before* execution and using a reference monitor (not the agent itself), enforcement is independent of the agent's compliance.
- **What fails:** Computational overhead (though measured in milliseconds). Requires integration points in the agent framework. Can only constrain observable actions, not internal reasoning.

**Relevance to orch-go:** AgentSpec's reference monitor pattern is the theoretical ideal for orch-go's problem. A monitor that intercepts "orchestrator about to use Task tool" and redirects to "orch spawn" would solve the behavioral compliance gap. The challenge is implementing this within Claude Code's architecture.

#### Microsoft Agent Framework (AutoGen successor)

**Approach:** Termination conditions + task adherence detection

- **Termination conditions** like `StopAfterNMessages(3)` provide hard stops
- **Task adherence:** Detects when an agent's next proposed action drifts off-task (announced at Ignite 2025)
- **Guardrails at agent level** (not just model level) — applied to the agent instance itself
- **What works:** Task adherence detection is the only framework-level attempt at constraining agent *intent*, not just actions
- **What fails:** Too new to have production failure data. AutoGen → Agent Framework migration is ongoing; stability unclear.

**Relevance to orch-go:** Task adherence detection is conceptually closest to what orch-go needs — detecting when an orchestrator is about to deviate from its role. But it's a proprietary Microsoft feature, not an open pattern.

### Finding 3: Universal Patterns and Gaps

**Patterns that work across frameworks:**

1. **Hard iteration/token limits** — Every framework uses these. Simple, deterministic, prevents runaway costs. orch-go has this via spawn tier configurations.
2. **Output validation with retry** — Guardrails AI, CrewAI, OpenAI all validate outputs and can trigger retries. Works for output quality, not for tool selection.
3. **Human-in-the-loop checkpoints** — LangGraph's interrupt, OpenAI's blocking mode, Claude SDK's canUseTool callback. The universal fallback when automated constraints are insufficient.
4. **Graph/flow structural constraints** — LangGraph graphs, NeMo Colang flows. Strongest pattern because it restricts the *possible* action space, not just the *desired* one.

**Universal gap: No framework constrains tool selection at the decision layer.**

Every framework can:
- Validate what the agent *produced* (output guardrails)
- Block specific *actions* from executing (permission rules, reference monitors)
- Limit *how many* actions the agent takes (iteration caps)

No framework can:
- Force the agent to *prefer* one tool over another
- Prevent the agent from *attempting* a prohibited action (only intercept it)
- Resolve competing instructions from different hierarchy levels (system prompt vs skill)

### Finding 4: The Competing Instruction Hierarchy is an Unsolved Problem

The orch-go probe (2026-02-24) identified that Claude's system prompt promotes Task tool usage with a 17:1 signal ratio over the skill's constraint against it. This is a specific instance of a general unsolved problem:

**When framework-level affordances (built-in tools, system prompts) conflict with application-level constraints (skills, guardrails), the framework wins.**

- In LangChain: Built-in tools are always available regardless of guardrails
- In OpenAI SDK: Guardrails only apply to function tools, not hosted/built-in tools
- In Claude Code: System prompt promoting Task tool overrides skill instructions discouraging it
- In NeMo: Closest to solving this by making "anything not in a flow" blocked — but requires rearchitecting around the constraint

**The field consensus (2026):** Prompts describe desired behavior; infrastructure enforces it. The gap between description and enforcement is where behavioral violations occur.

### Finding 5: Enforcement Strategies Mapped to orch-go

| Strategy | Framework Source | orch-go Applicability | Implementation Difficulty |
|---|---|---|---|
| Reference monitor (intercept before execution) | AgentSpec | High — would solve Task tool problem | High — requires Claude Code integration point |
| Hooks + permission rules | Claude Agent SDK | High — layered evaluation is proven | Medium — Claude Code already has hooks |
| Graph structural constraints | LangGraph | Medium — would require modeling orchestrator as graph | High — architectural change |
| Output validation + retry | Guardrails AI, CrewAI | Low — doesn't prevent wrong tool, only validates result | Low — already have completion verification |
| Colang flow enforcement | NeMo Guardrails | Medium — "deny by default" is powerful | High — requires complete approach change |
| Task adherence detection | Microsoft Agent Framework | High — detects intent drift | High — proprietary, no open implementation |

## Conclusion

The agent framework landscape (2026) has converged on a three-tier enforcement model: prompt-level (advisory), output-validation (post-hoc), and action-interception (pre-execution). **All production frameworks operate at the action boundary, not the decision boundary.** No framework can make an agent *want* to use the right tool — they can only prevent the wrong tool from executing.

For orch-go's specific problem (orchestrators using Task tool instead of orch spawn due to competing system-prompt instructions):

1. **What won't work:** More prompt-level instructions (already proven by 17:1 signal ratio analysis)
2. **What partially works:** Output validation / completion verification (catches the error after the fact)
3. **What would work:** A Claude Code hook that intercepts Task tool invocations by orchestrator sessions and either blocks them with a redirect message or rewrites them to orch spawn invocations. This maps to the AgentSpec reference monitor pattern and the Claude Agent SDK's hooks mechanism.

The closest existing production pattern is the Claude Agent SDK's 4-step permission evaluation (hooks → rules → mode → callback). Claude Code's hook system is a subset of this — the infrastructure exists, but the specific hook (intercept Task tool in orchestrator context) doesn't.

## Test Performed

Web research across 8 frameworks: LangChain/LangGraph, CrewAI, AutoGPT, NeMo Guardrails, OpenAI Agents SDK, Claude Agent SDK, AgentSpec, Microsoft Agent Framework. Examined official documentation, API specifications, academic papers (ICSE '26, EMNLP '23), and practitioner guides. Cross-referenced findings against orch-go's documented behavioral compliance gap (probe 2026-02-24).
