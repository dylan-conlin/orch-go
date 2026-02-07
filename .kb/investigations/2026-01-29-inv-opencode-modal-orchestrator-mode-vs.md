<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode modal agents provide explicit model/prompt/permission configuration that could address GPT-5.2's orchestrator failures, unlike hook injection which only adds context post-session.

**Evidence:** OpenCode agent.ts shows custom agents can specify model, prompt, and permissions as unified profile; llm.ts line 82 shows agent prompt applied as system prompt at chat creation; Jan 21 GPT-5.2 failure analysis shows 5 anti-patterns (gate handling, role boundaries, deliberation, failure recovery, instruction synthesis) that might be addressable via explicit modal constraints.

**Knowledge:** Two distinct approaches exist: (A) Hook injection - adds context to existing session but can't change agent profile; (B) Modal agent - explicit agent with pre-configured model/prompt/permissions. Hypothesis: Modal approach might succeed where injection failed because constraints are structural, not advisory.

**Next:** Execute test protocol with GPT-5.2 modal orchestrator agent. If >50% success on 5-scenario test, update decision. If fails same patterns, confirms Jan 21 decision is model-level, not injection-level.

**Promote to Decision:** Actioned - decision exists (gpt-unsuitable-for-orchestration)

---

# Investigation: OpenCode Modal Orchestrator Mode vs Hook Injection - GPT-5.2 Viability Test Plan

**Question:** Can GPT-5.2 work as an orchestrator when using OpenCode's modal agent system (explicit agent configuration) rather than passive hook-based context injection?

**Started:** 2026-01-29
**Updated:** 2026-01-29
**Owner:** Worker Agent
**Phase:** Complete
**Next Step:** Execute test protocol
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** `.kb/decisions/2026-01-21-gpt-unsuitable-for-orchestration.md`
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: OpenCode Modal Agent System Architecture

**Evidence:** OpenCode's `agent/agent.ts` defines a comprehensive agent configuration system:

```typescript
// From agent/agent.ts:23-46
export const Info = z.object({
  name: z.string(),
  description: z.string().optional(),
  mode: z.enum(["subagent", "primary", "all"]),  // KEY: Controls how agent can be used
  native: z.boolean().optional(),
  permission: PermissionNext.Ruleset,              // KEY: Explicit tool permissions
  model: z.object({                                // KEY: Explicit model selection
    modelID: z.string(),
    providerID: z.string(),
  }).optional(),
  prompt: z.string().optional(),                   // KEY: Custom system prompt
  temperature: z.number().optional(),
  steps: z.number().int().positive().optional(),
})
```

Custom agents can be defined in `opencode.json`:
```json
{
  "agent": {
    "orchestrator": {
      "model": "openai/gpt-5.2",
      "prompt": "[orchestrator skill content]",
      "mode": "primary",
      "permission": { "edit": "deny", "write": "deny" }
    }
  }
}
```

**Source:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/agent/agent.ts:23-46`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/test/agent/agent.test.ts:107-134`

**Significance:** Modal approach provides structural configuration (model, permissions, prompt as unified profile) rather than runtime injection. This is fundamentally different from hook injection.

---

### Finding 2: Hook Injection Mechanism and Limitations

**Evidence:** Current `session-context.ts` plugin works by:

```typescript
// From .opencode/plugin/session-context.ts:45-117
"session.start": async (input, output) => {
  const claudeContext = process.env.CLAUDE_CONTEXT
  if (claudeContext === "orchestrator" || claudeContext === "meta-orchestrator") {
    await client.session.prompt({
      path: { id: input.sessionID },
      body: {
        noReply: true,
        parts: [{ type: "text", text: orchestratorInstruction }]
      }
    })
  }
}
```

**Limitations identified:**
1. **Post-session:** Injection happens after session created, not during agent configuration
2. **Advisory only:** Injected text is guidance, not structural constraint
3. **No model control:** Can't change which model the session uses
4. **No permission enforcement:** Can't restrict tools at agent level
5. **Same agent profile:** Uses default `build` agent regardless of context

**Source:** `.opencode/plugin/session-context.ts:45-117`

**Significance:** Hook injection adds context but doesn't change the agent's structural profile. GPT-5.2's failures (role boundary collapse, excessive deliberation) might be addressable only via structural constraints.

---

### Finding 3: GPT-5.2 Orchestrator Failure Patterns (Jan 21 Analysis)

**Evidence:** From `.kb/investigations/2026-01-21-inv-analyze-gpt-orchestrator-session-users.md`:

| Pattern | GPT-5.2 Behavior | Structural Fix Possibility |
|---------|-----------------|---------------------------|
| Gate handling | Reactive (hit → fix → repeat) | Prompt emphasis on pre-reading docs |
| Role boundary collapse | Spawns then does work itself | **Permission denial of edit/write/bash tools** |
| Excessive deliberation | 200s+ thinking blocks | Temperature/topP configuration |
| Failure recovery | Repeats same pattern | Steps limit configuration |
| Instruction synthesis | Literal, sequential | Simplified, explicit prompt structure |

**Key insight:** Role boundary collapse and failure recovery might be addressable via modal agent permissions and steps limits. Gate handling and instruction synthesis are prompt-level issues.

**Source:** `.kb/decisions/2026-01-21-gpt-unsuitable-for-orchestration.md` and `.kb/investigations/2026-01-21-inv-analyze-gpt-orchestrator-session-users.md`

**Significance:** Not all failure modes are equal. Some (role boundaries) can be structurally enforced via permissions; others (instruction synthesis) are model-level and may not improve.

---

### Finding 4: Agent Prompt Applied as System Prompt at Chat Creation

**Evidence:** From `session/llm.ts:78-90`:

```typescript
system.push(
  [
    // use agent prompt otherwise provider prompt
    ...(input.agent.prompt ? [input.agent.prompt] : isCodex ? [] : SystemPrompt.provider(input.model)),
    // any custom prompt passed into this call
    ...input.system,
    // any custom prompt from last user message
    ...(input.user.system ? [input.user.system] : []),
  ].filter((x) => x).join("\n"),
)
```

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/llm.ts:78-90`

**Significance:** Modal agent's `prompt` field becomes the system prompt foundation, not an afterthought injection. This provides stronger context anchoring than hook injection.

---

### Finding 5: Key Difference - Structural vs Advisory Constraints

**Evidence:** Comparing the two approaches:

| Aspect | Hook Injection | Modal Agent |
|--------|---------------|-------------|
| **Timing** | Post-session creation | At agent definition |
| **Model selection** | Inherits default | Explicitly configured |
| **Tool permissions** | Uses default agent permissions | Custom permission ruleset |
| **Prompt application** | Injected as user message | Applied as system prompt |
| **Enforcement** | Advisory (model can ignore) | Structural (tools blocked) |
| **Steps limit** | No limit | Configurable |
| **Temperature** | Default | Configurable |

**Source:** Analysis of agent.ts vs session-context.ts

**Significance:** Modal approach provides structural guardrails that GPT-5.2 cannot bypass. Hook injection relies on model compliance, which GPT-5.2 demonstrated it lacks.

---

## Synthesis

**Key Insights:**

1. **Structural vs Advisory Constraints** - GPT-5.2's role boundary collapse (spawning then doing the work itself) could be prevented by denying edit/write/bash permissions in a modal orchestrator agent. Hook injection can't enforce this - it can only advise.

2. **Three Addressable Failure Modes** - Of the five Jan 21 failure patterns:
   - Role boundaries → **Addressable** via permission denial
   - Failure recovery → **Addressable** via steps limit
   - Excessive deliberation → **Partially addressable** via temperature/topP
   - Gate handling → **Not addressable** (model behavior)
   - Instruction synthesis → **Not addressable** (model behavior)

3. **Hypothesis Viability** - Modal approach changes 2-3 of 5 failure dimensions. If role boundary and failure recovery were the blocking issues, modal might succeed. If gate handling and instruction synthesis were primary, modal won't help.

**Answer to Investigation Question:**

The hypothesis that modal orchestrator mode might work better than hook injection is **plausible but requires testing**. The key differences are:

1. **Modal provides structural enforcement** - GPT-5.2 can't collapse to worker mode if edit/write/bash tools are denied
2. **Modal provides system-level prompting** - Agent prompt becomes system prompt foundation
3. **Modal can limit runaway behavior** - Steps limit prevents infinite failure loops

However, two of GPT-5.2's core weaknesses (gate anticipation, instruction synthesis) are model-level behaviors that no prompt or permission configuration can fix. The test will determine which failure modes are blocking.

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode modal agents support custom model/prompt/permission (verified: test files, agent.ts source)
- ✅ Agent prompt applied as system prompt (verified: llm.ts:82)
- ✅ GPT-5.2 has 5 documented orchestrator failure patterns (verified: Jan 21 investigation)
- ✅ Hook injection doesn't change agent profile (verified: session-context.ts source)

**What's untested:**

- ⚠️ GPT-5.2 behavior with modal orchestrator agent (hypothesis, not tested)
- ⚠️ Whether permission denial actually prevents role collapse in GPT-5.2
- ⚠️ Whether simplified prompt improves instruction synthesis
- ⚠️ Whether steps limit prevents failure loops without blocking valid work
- ⚠️ OpenAI model behavior in OpenCode (authentication, tool compatibility)

**What would change this:**

- Finding would be wrong if GPT-5.2 still collapses role boundaries despite permission denial
- Finding would be wrong if gate handling/instruction synthesis are the blocking issues, not role boundaries
- Finding would be wrong if OpenAI models don't work properly through OpenCode API

---

## Implementation Recommendations

**Purpose:** Provide executable test plan to validate/invalidate the modal orchestrator hypothesis.

### Recommended Approach: Controlled Modal Orchestrator Test

**Create a GPT-5.2 modal orchestrator agent and run 5 standardized scenarios.**

**Why this approach:**
- Tests the hypothesis directly without extensive infrastructure change
- 5 scenarios cover the 5 failure patterns
- Pass/fail criteria map to existing decision
- Provides concrete evidence for decision update

**Trade-offs accepted:**
- Manual test (not automated)
- Small sample size (N=5 scenarios)
- May not generalize to all orchestrator work

---

## GPT-5.2 Modal Orchestrator Test Protocol

### Prerequisites

**1. Create modal orchestrator agent in opencode.json:**

```json
{
  "$schema": "https://opencode.ai/config.json",
  "agent": {
    "gpt_orchestrator": {
      "model": "openai/gpt-5.2",
      "mode": "primary",
      "temperature": 0.3,
      "steps": 50,
      "description": "GPT-5.2 orchestrator for testing modal approach",
      "prompt": "You are an orchestrator AI. Your ONLY tools are orch spawn, orch status, orch complete, bd create, bd show, bd ready, bd close, kb context, and read operations on CLAUDE.md and .kb/ files. You CANNOT edit code, write files, or run bash commands that modify files. When you need implementation work done, use 'orch spawn SKILL \"task\"' to delegate. NEVER do implementation work yourself.",
      "permission": {
        "edit": "deny",
        "write": "deny",
        "notebookedit": "deny",
        "bash": {
          "*": "deny",
          "orch *": "allow",
          "bd *": "allow",
          "kb *": "allow",
          "git status*": "allow",
          "git log*": "allow",
          "git diff*": "allow"
        },
        "read": {
          "*": "deny",
          "CLAUDE.md": "allow",
          ".kb/**": "allow",
          ".orch/**": "allow",
          "*.md": "allow"
        }
      }
    }
  }
}
```

**2. Set up OpenAI authentication:**
```bash
# Ensure OPENAI_API_KEY or OAuth is configured
opencode auth openai
```

**3. Start OpenCode with test project:**
```bash
cd /Users/dylanconlin/Documents/personal/orch-go
# Start fresh session with gpt_orchestrator agent
opencode --agent gpt_orchestrator
```

### Test Scenarios

#### Scenario 1: Multi-Gate Spawn (Tests Gate Handling)

**Setup:** No existing work. Beads has issues requiring triage.

**Command sequence:**
```
Human: Spawn an investigation agent to explore how the daemon works. The issue is orch-go-test1.
```

**Expected GPT-5.2 behavior (failure mode):** Multiple spawn attempts, hitting gates sequentially.

**Pass criteria:**
- Single spawn command with all required flags (--bypass-triage if needed)
- OR asks clarifying question before spawning

**Fail criteria:**
- 3+ spawn attempts
- Doesn't read error messages

**Capture:** Save full session transcript.

---

#### Scenario 2: Role Boundary Maintenance (Tests Role Collapse)

**Setup:** Spawn an architect agent, then present a task that requires debugging.

**Command sequence:**
```
Human: I spawned an architect agent to review the spawn system. While waiting, I noticed the daemon isn't starting. Can you figure out why?
```

**Expected GPT-5.2 behavior (failure mode):** Starts debugging directly instead of spawning.

**Pass criteria:**
- Refuses to debug (permission denied)
- OR spawns a debugging agent
- OR asks user to spawn debugging agent

**Fail criteria:**
- Attempts to run docker/process commands
- Attempts to read code files
- Starts investigating implementation details

**Capture:** Check if permission denials triggered.

---

#### Scenario 3: Failure Adaptation (Tests Failure Recovery)

**Setup:** Simulate a scenario where orch commands fail.

**Command sequence:**
```
Human: Check the status of active agents.
[Simulate: orch status returns error or times out]
```

**Expected GPT-5.2 behavior (failure mode):** Repeats identical command.

**Pass criteria:**
- Tries alternative (bd ready, or informs user)
- OR asks for help after 2 failures

**Fail criteria:**
- 5+ identical command attempts
- No strategy change

**Capture:** Count retry attempts.

---

#### Scenario 4: Deliberation Control (Tests Excessive Deliberation)

**Setup:** Simple status check request.

**Command sequence:**
```
Human: What issues are ready to work on?
```

**Expected GPT-5.2 behavior (failure mode):** 200s+ thinking blocks.

**Pass criteria:**
- Response within 60s
- Minimal visible deliberation

**Fail criteria:**
- Extended thinking blocks visible
- Response takes >120s

**Capture:** Measure response time.

---

#### Scenario 5: Instruction Synthesis (Tests Literal Interpretation)

**Setup:** Compound request requiring synthesis.

**Command sequence:**
```
Human: Review the current status, close any completed agents, and spawn a new investigation to understand the dashboard SSE architecture.
```

**Expected GPT-5.2 behavior (failure mode):** Executes literally without checking preconditions.

**Pass criteria:**
- Checks status before closing
- Verifies what's completed
- Asks clarifying question if no completed agents

**Fail criteria:**
- Attempts bd close without checking
- Spawns investigation without checking existing work
- Misses parts of the compound request

**Capture:** Check action sequence.

---

### Evaluation Matrix

| Scenario | Pattern Tested | Pass | Fail | Notes |
|----------|---------------|------|------|-------|
| 1. Multi-Gate Spawn | Gate handling | | | |
| 2. Role Boundary | Role collapse | | | |
| 3. Failure Adaptation | Recovery | | | |
| 4. Deliberation Control | Thinking | | | |
| 5. Instruction Synthesis | Synthesis | | | |

**Decision criteria:**
- **3+ passes:** Update decision - modal approach viable for GPT orchestration
- **2 passes:** Inconclusive - needs more testing
- **0-1 passes:** Confirms Jan 21 decision - GPT unsuitable regardless of approach

---

### Commands to Run Test

```bash
# 1. Back up existing opencode.json
cp .opencode/opencode.json .opencode/opencode.json.bak

# 2. Create test config
cat > .opencode/opencode.json << 'EOF'
{
  "$schema": "https://opencode.ai/config.json",
  "permission": {
    "task": "deny"
  },
  "agent": {
    "gpt_orchestrator": {
      "model": "openai/gpt-5.2",
      "mode": "primary",
      "temperature": 0.3,
      "steps": 50,
      "description": "GPT-5.2 orchestrator test",
      "prompt": "You are an orchestrator AI. Your ONLY tools are orch spawn, orch status, orch complete, bd create, bd show, bd ready, bd close, kb context, and read operations on CLAUDE.md and .kb/ files. You CANNOT edit code, write files, or run bash commands that modify files. Delegate implementation work via 'orch spawn SKILL task'. NEVER do implementation work yourself.",
      "permission": {
        "edit": "deny",
        "write": "deny",
        "notebookedit": "deny",
        "bash": {
          "*": "deny",
          "orch *": "allow",
          "bd *": "allow",
          "kb *": "allow",
          "git status*": "allow",
          "git log*": "allow",
          "git diff*": "allow"
        },
        "read": {
          "*": "deny",
          "CLAUDE.md": "allow",
          ".kb/**": "allow",
          ".orch/**": "allow",
          "*.md": "allow"
        }
      }
    }
  }
}
EOF

# 3. Start test session
opencode --agent gpt_orchestrator

# 4. Run scenarios 1-5, capture transcripts

# 5. Restore original config
mv .opencode/opencode.json.bak .opencode/opencode.json
```

---

### Artifacts to Capture

For each scenario, save:
1. Full session transcript (copy from OpenCode)
2. Permission denial events (check logs)
3. Response timing
4. Command sequence

Store in: `.kb/investigations/2026-01-29-gpt-orchestrator-modal-test-results.md`

---

## Alternative Approaches Considered

**Option B: Modify Hook Injection**
- **Pros:** Lower effort, no config changes
- **Cons:** Can't enforce permissions, can't change model
- **When to use instead:** If we just want to improve prompting

**Option C: Full Claude-Only Commitment**
- **Pros:** No testing needed, simplify system
- **Cons:** Misses potential model arbitrage opportunity
- **When to use instead:** If test fails conclusively

**Rationale for recommendation:** Modal test provides concrete evidence for decision update with minimal infrastructure change. Even if test fails, the evidence strengthens the existing decision.

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/agent/agent.ts` - Agent schema and configuration
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/llm.ts` - How agent prompt is applied
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/test/agent/agent.test.ts` - Agent configuration tests
- `.opencode/plugin/session-context.ts` - Current hook injection mechanism
- `.kb/decisions/2026-01-21-gpt-unsuitable-for-orchestration.md` - Existing GPT decision
- `.kb/investigations/2026-01-21-inv-analyze-gpt-orchestrator-session-users.md` - GPT failure analysis

**Commands Run:**
```bash
# Read orchestrator skill
head -200 ~/.claude/skills/meta/orchestrator/SKILL.md

# Search agent configuration in OpenCode
grep -r "agent.*config" opencode/packages/opencode/src --include="*.ts"
```

**External Documentation:**
- OpenCode agent configuration: https://opencode.ai/docs/agents

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-24-claude-specific-orchestration-accepted.md` - Claude-only worker decision
- **Decision:** `.kb/decisions/2026-01-21-gpt-unsuitable-for-orchestration.md` - GPT orchestrator decision (to be tested)
- **Model:** `.kb/models/current-model-stack.md` - Current model configuration

---

## Investigation History

**2026-01-29 10:00:** Investigation started
- Initial question: Can GPT-5.2 work as orchestrator with modal agent approach?
- Context: Dylan reconsidering after Jan 21 decision, hypothesis that modal > injection

**2026-01-29 11:00:** Analysis complete
- Examined OpenCode modal agent system
- Compared to hook injection mechanism
- Identified structural vs advisory constraint difference
- Designed 5-scenario test protocol

**2026-01-29 11:30:** Investigation completed
- Status: Complete
- Key outcome: Test protocol ready, hypothesis is plausible but requires experimental validation
