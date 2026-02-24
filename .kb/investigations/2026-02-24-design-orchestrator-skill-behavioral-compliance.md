## Summary (D.E.K.N.)

**Delta:** Orchestrator agents comply with identity declarations but not action constraints because identity is additive (no competing instructions) while actions are subtractive (directly conflicts with Claude Code system prompt that promotes Task tool and bd close).

**Evidence:** Signal ratio analysis shows 17:1 competing instruction disadvantage — system prompt has ~500 words promoting Task tool, skill has ~30 words constraining it. Action constraints appear at 88% depth in 640-line skill. System prompt has structural priority in Claude's instruction hierarchy (system > user content).

**Knowledge:** The problem is not skill content quality but instruction hierarchy position. Prompt-level action restrictions operate as guidelines, not enforcement. Recent research (AgentSpec ICSE 2026, PCAS) confirms: prompts describe desired behavior, infrastructure enforces it. A two-layer fix is needed: restructure skill for salience AND add tool-layer enforcement.

**Next:** Implement the recommended two-layer approach: (1) restructure skill with action-identity fusion at top, (2) add Claude Code hook or plugin that intercepts prohibited tool usage for orchestrator sessions.

**Authority:** architectural - Crosses skill system, Claude Code hooks, and plugin boundaries. Orchestrator decides.

---

# Investigation: Orchestrator Skill Behavioral Compliance — Why Agents Load the Skill but Revert to Claude Code Defaults

**Question:** Why do orchestrator agents comply with identity declarations ("I'm your orchestrator, I don't write code") but fail to comply with action constraints ("use orch spawn, not Task tool") at action time? What structural changes would make action constraints stick?

**Defect-Class:** configuration-drift

**Started:** 2026-02-24
**Updated:** 2026-02-24
**Owner:** architect (orch-go-1178)
**Phase:** Complete
**Next Step:** None — ready for implementation follow-up
**Status:** Complete

**Patches-Decision:** N/A (new investigation)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| 2026-01-06-inv-diagnose-orchestrator-skill-18-completion.md | extends | Yes | None — that inv found low completion is by design; this finds action non-compliance is structural |
| Probe 2026-02-16 orchestrator-skill-orientation-redesign | extends | Yes | None — that probe found skill organized around identity, not Dylan's needs |
| Probe 2026-02-17 orchestrator-skill-injection-path-trace | extends | Yes | None — that probe found stale versions and 5 injection paths |
| Probe 2026-02-18 orchestrator-skill-cli-staleness-audit | confirms | Yes | None — 13 stale references confirm skill quality issues |

---

## Findings

### Finding 1: Identity Compliance and Action Compliance Are Mechanistically Different

**Evidence:** Structural analysis of the deployed orchestrator SKILL.md (640 lines) reveals two distinct instruction types:

| Dimension | Identity Declaration | Action Constraint |
|-----------|---------------------|-------------------|
| Relationship to defaults | Additive (no conflict) | Subtractive (conflicts with system prompt) |
| Framing match | Congruent with role-play | Incongruent with built-in affordances |
| Signal strength | Repeated throughout skill (lines 33, 35, 37-39, 96-108) | 1-2 lines in 640 (lines 68, 75) |
| Processing mode | Semantic ("who am I?") | Procedural ("what do I do?") |
| Competing instructions | None — system prompt doesn't say "you are NOT an orchestrator" | Direct — system prompt says "use Task tool for subagents" |

Identity declarations like "You are a strategic comprehender" (line 33) and "ORIENT → DELEGATE → RECONNECT (never implement)" (line 35) are additive — they layer on top of Claude's base behavior without conflicting with anything. Claude is designed to cooperate with role framing.

Action constraints like "Don't use Task tool" and "Use orch spawn instead" are subtractive — they require suppressing a default behavior that's actively promoted by the system prompt. This is fighting the instruction hierarchy, not working with it.

**Source:** `~/.claude/skills/meta/orchestrator/SKILL.md` (lines 33-108 for identity, lines 68-75 and 582-609 for action constraints)

**Significance:** This explains the observed behavior perfectly. The agent can hold identity ("I'm an orchestrator") while violating action constraints ("use Task tool") because these operate on different cognitive dimensions. Identity is belief; action is affordance selection under competing pressures.

---

### Finding 2: The Claude Code System Prompt Creates a 17:1 Signal Disadvantage

**Evidence:** The Claude Code system prompt (visible in every session) contains extensive promotion of the Task tool:

> "Launch a new agent to handle complex, multi-step tasks autonomously."
> "Use the Task tool with specialized agents when the task at hand matches the agent's description."
> "Launch multiple agents concurrently whenever possible, to maximize performance."

The system prompt describes 7+ agent types (Bash, general-purpose, Explore, Plan, etc.) and provides detailed usage patterns. This occupies approximately 500 words of high-salience, system-level instruction.

The orchestrator skill counters this with:
- Line 68: "Spawn method gate: Am I about to use Task tool? → STOP" (inside a 9-item checklist)
- Line 75: "The Task tool is NOT how orchestrators spawn workers" (one line after a 40-row table)
- Lines 594-604: Tool Action Space table (at 88% depth in the document)

Total counter-signal: approximately 30 words of user-level content.

**Signal ratio: ~17:1 in favor of Task tool usage.**

Additionally, the system prompt occupies a privileged position in Claude's instruction hierarchy: system instructions > user instructions > assistant context. The skill content is injected as user-level content (via CLAUDE.md, hooks, or SPAWN_CONTEXT.md), structurally subordinate to the system prompt.

**Source:** Claude Code system prompt (visible at top of any Claude Code session); `~/.claude/skills/meta/orchestrator/SKILL.md` lines 68, 75, 594-604

**Significance:** Even a perfectly written action constraint at the user level faces an impossible signal ratio disadvantage against system-level tool promotion. This isn't a skill quality problem — it's a structural positioning problem.

---

### Finding 3: The Action Space Is Described, Not Enforced

**Evidence:** The orchestrator skill's section 6 includes a "Tool Action Space (Architectural Constraint)" table (lines 594-604):

```
| You CAN (meta-actions)              | You CANNOT (primitive actions)     |
|--------------------------------------|------------------------------------|
| orch spawn/complete/status/review    | Edit/Write tools (code editing)    |
| bd create/show/ready/close           | Read code files (.go, .ts, etc.)   |
| kb context, kb quick decide/...      | Most bash commands                 |
| git status (read-only)               | Direct file operations             |
```

The skill text says: "Frame collapse is prevented by restricting action space, not just guidelines."

However, examination shows the "restriction" is entirely instructional — a markdown table in a prompt. The actual tool set available to orchestrator sessions is identical to any other Claude Code session. The Task tool, Edit tool, Write tool, and all Bash commands remain fully functional and available. No hook, plugin, or tool-layer gate intercepts or prevents prohibited tool usage.

**Source:** `~/.claude/skills/meta/orchestrator/SKILL.md` lines 594-604; Claude Code tool availability (all tools available to all sessions)

**Significance:** The model's claim that "frame collapse is prevented by restricting action space" is aspirational, not factual. The action space is described as restricted but implemented as a guideline. This is exactly the pattern that the "Infrastructure Over Instruction" principle warns against: "Relying on an agent's reasoning to maintain system discipline under pressure."

---

### Finding 4: `bd close` vs `orch complete` Has the Same Competing Signal Problem

**Evidence:** The second observed failure — orchestrator using `bd close` instead of `orch complete` — follows the identical pattern:

- `bd close` is documented in the beads guidance injected at every session start (via `bd prime` hook)
- `bd close` appears in CLAUDE.md, beads session context, and skill guidance for workers
- `orch complete` appears only in the orchestrator skill and orch-go CLAUDE.md
- The beads hook runs on every session, injecting `bd close` as a first-class command
- When an agent thinks "close this issue," `bd close` is the most salient path

Furthermore, `bd close` is shorter, simpler, and appears more frequently in the agent's context than `orch complete`. The orchestrator skill says to use `orch complete` for its verification gates, but this instruction competes with the pervasive `bd close` signal.

**Source:** SessionStart hooks (`bd prime` output), CLAUDE.md beads section, orchestrator skill lines 504-511

**Significance:** This is the same structural problem as Task tool vs orch spawn. A well-documented default command (`bd close`) competes with a role-specific override (`orch complete`) that appears less frequently and with lower salience.

---

### Finding 5: Temporal Dynamics Favor Defaults Over Skill Constraints

**Evidence:** Instruction processing timing analysis:

1. **Skill content**: Injected at session start or in SPAWN_CONTEXT.md — early in context, attention decays over time
2. **System prompt tool descriptions**: Present on every turn — persistent, reinforced
3. **User request ("spawn a worker")**: Triggers pattern-matching at action time

When the user says "spawn a worker to investigate X":
- **Pattern match to Task tool**: Strong. The system prompt literally says "Launch a new agent to handle complex, multi-step tasks autonomously." The word "spawn" maps directly to "launch an agent."
- **Pattern match to orch spawn**: Weak. Requires remembering a specific CLI command from skill content injected earlier in the session.

The more time passes between skill injection and the action moment, the weaker the skill's constraints become. System prompt instructions don't decay because they're architecturally persistent.

**Source:** Analysis of Claude Code instruction architecture and attention dynamics

**Significance:** Even if skill content were perfectly structured, temporal decay means system prompt instructions become relatively more salient over time. This is a structural disadvantage that cannot be overcome by skill content alone.

---

### Finding 6: Research Confirms Prompt-Level Constraints Are Insufficient for Action Enforcement

**Evidence:** Recent research on LLM agent safety and reliability:

- **ICLR 2025 (Instruction Hierarchy)**: Proposes Instructional Segment Embedding to differentiate instruction types at the token level. Confirms current LLMs lack native ability to prioritize competing instruction sources.
- **AgentSpec (ICSE 2026)**: Runtime enforcement for safe LLM agents via customizable guardrails that check actions before execution.
- **PCAS (Policy Compiler for Agentic Systems)**: Deterministic policy enforcement via reference monitors intercepting actions. Models agent state as dependency graph with policies as declarative rules.
- **Plan-Then-Execute pattern**: Constraining agents to fixed action plans prevents deviation from tool calls returning untrusted data.

**Field consensus: Prompts describe desired behavior; infrastructure enforces it.**

This directly validates the orch-go principle "Infrastructure Over Instruction" and suggests the fix is not better prompts but tool-layer enforcement.

**Source:** [ICLR 2025](https://proceedings.iclr.cc/paper_files/paper/2025/file/ea13534ee239bb3977795b8cc855bacc-Paper-Conference.pdf), [AgentSpec ICSE 2026](https://cposkitt.github.io/files/publications/agentspec_llm_enforcement_icse26.pdf), [PCAS](https://arxiv.org/html/2602.16708v1)

**Significance:** The problem is not orch-go specific. It's a known limitation of prompt-based behavioral constraints. The research community's solution direction — tool-layer enforcement — aligns with what orch-go needs.

---

## Synthesis

**Key Insights:**

1. **Identity compliance ≠ Action compliance.** These are mechanistically different. Identity is additive (layers on defaults). Action constraints are subtractive (fight defaults). An agent can believe it's an orchestrator while using worker tools. Testing "what is your role?" tells you nothing about action compliance.

2. **The problem is instruction hierarchy, not skill quality.** The Claude Code system prompt has structural priority (system > user), higher signal volume (17:1 ratio), persistent reinforcement (every turn), and stronger pattern matching for user requests. No amount of skill content improvement can overcome this structural disadvantage alone.

3. **"Restricting action space via guidelines" is an oxymoron.** The model claims action space restriction prevents frame collapse. But the restriction is implemented as a markdown table — a guideline, not infrastructure. This violates the system's own "Infrastructure Over Instruction" principle. Action space restriction must be tool-layer enforcement, not prompt-layer description.

4. **Two distinct fix layers are needed.** Layer 1 (prompt-level): Restructure skill to maximize the effectiveness of what prompts CAN do — salience, positioning, framing, repetition. Layer 2 (infrastructure-level): Add tool-layer enforcement that actually restricts the action space. Neither layer alone is sufficient.

**Answer to Investigation Question:**

Agents comply with identity declarations but not action constraints because identity is additive (doesn't conflict with anything) while action constraints are subtractive (directly conflicts with the system prompt's active promotion of the Task tool). The system prompt occupies a structurally superior instruction hierarchy position, has a 17:1 signal advantage, and is temporally persistent while skill content decays. The action "restrictions" are described in a markdown table rather than enforced by infrastructure.

The fix requires two layers: (1) restructure the skill to maximize prompt-level salience using action-identity fusion, affordance replacement, and strategic repetition; (2) add tool-layer enforcement via Claude Code hooks or plugins that intercept prohibited tool usage in orchestrator sessions.

---

## Structured Uncertainty

**What's tested:**

- ✅ Signal ratio analysis: counted words promoting Task tool in system prompt vs words constraining it in skill (verified: 17:1 ratio)
- ✅ Positioning analysis: action constraints at lines 68, 75, 594-604 of 640-line skill (verified: first substantive mention at 10%, full detail at 88%)
- ✅ Tool availability: confirmed Task tool, Edit, Write, all Bash commands remain available to orchestrator sessions (no enforcement mechanism exists)
- ✅ Prior probe cross-reference: orientation redesign (Feb 16), injection path trace (Feb 17), CLI staleness audit (Feb 18) all confirm structural issues

**What's untested:**

- ⚠️ Whether the recommended skill restructuring actually improves compliance (needs A/B testing with orchestrator sessions)
- ⚠️ Whether tool-layer enforcement via Claude Code hooks is technically feasible without modifying Claude Code internals
- ⚠️ Whether the signal ratio hypothesis holds for other LLMs (tested reasoning against Claude architecture only)
- ⚠️ Exact attention decay rate for skill content vs system prompt content over session duration

**What would change this:**

- If Claude's instruction hierarchy were modified to give CLAUDE.md/skill content equal or higher priority than system prompt for action constraints
- If evidence showed orchestrators correctly using orch spawn despite the competing signal (would need to investigate what made those sessions different)
- If a simpler prompt-only fix reliably achieves >90% action compliance without infrastructure enforcement

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Restructure skill content (Layer 1) | architectural | Crosses skill system and orch-knowledge repo |
| Add tool-layer enforcement (Layer 2) | architectural | Crosses Claude Code hooks, plugin system, and orchestrator detection |
| Investigate ORCH_ORCHESTRATOR env var for session detection | implementation | Tactical detection mechanism within existing patterns |

### Recommended Approach ⭐: Two-Layer Action Compliance

**Layer 1 + Layer 2 together** — Restructure skill for maximum prompt-level salience AND add tool-layer enforcement for prohibited actions.

**Why this approach:**
- Neither layer alone is sufficient (Finding 2 + Finding 3)
- Aligns with system principle "Infrastructure Over Instruction"
- Aligns with field research consensus (Finding 6)
- Layer 1 is quick to implement; Layer 2 is more durable but requires more work
- Together they provide defense in depth

**Trade-offs accepted:**
- Layer 2 (tool enforcement) adds complexity to the hook/plugin system
- Risk of false positives if orchestrator detection is wrong
- Maintenance burden for keeping enforcement rules in sync with skill

**Implementation sequence:**

#### Layer 1: Skill Restructuring (Prompt-Level — Do First)

**1a. Action-Identity Fusion at Top of Skill**

Move action constraints from Section 6 (88% depth) to the very top, fused with identity. Replace the current opening:

```markdown
## 1. Identity: Strategic Comprehender Who Keeps Dylan Oriented

You are a **strategic comprehender** who keeps Dylan oriented.
```

With an action-identity fusion block:

```markdown
## 1. Identity and Action Space

You are a **strategic comprehender** who keeps Dylan oriented.

**Your tools (exhaustive list):**
- `orch spawn` / `orch complete` / `orch review` — agent lifecycle
- `bd create` / `bd show` / `bd ready` — work tracking
- `kb context` / `kb quick` — knowledge
- Read: CLAUDE.md, .kb/*.md, SYNTHESIS.md — comprehension

**NOT your tools (these belong to workers):**
- ❌ Task tool — orchestrators use `orch spawn`, never Task tool
- ❌ Edit/Write — orchestrators don't write code
- ❌ Read code files — that's investigation work, delegate it

**When user says "spawn a worker":** `orch spawn SKILL "task"` (NOT Task tool)
**When closing completed work:** `orch complete <id>` (NOT `bd close`)
```

**Why this works:**
- Fuses action constraints WITH identity (leverages identity compliance)
- Places constraints at 0% depth (maximum salience)
- Provides explicit affordance replacement ("instead of X, use Y")
- Uses the "NOT your tools" framing to create identity-incongruent inhibition
- Addresses both failure modes (Task tool AND bd close)

**1b. Affordance Replacement at Decision Points**

At every point in the skill where spawning is discussed, include the replacement:

```markdown
**Spawn via:** `bd create -l triage:ready` (primary) or `orch spawn` (exception)
**NOT:** Task tool (that's Claude Code's default, not yours)
```

This creates repetition at the exact moments the agent is making spawn decisions.

**1c. Reduce Skill Length**

The current skill is 640 lines (~8K tokens). Every line that isn't action-relevant dilutes the action constraint signal. Target: 400 lines by removing or condensing:
- The 40-row fast path table (consolidate to 10-15 most critical rows)
- Duplicate content between sections (triage criteria appear twice)
- Detailed examples that could be in reference docs
- Cross-reference sections that could be links

This improves the signal-to-noise ratio from the current level.

**1d. Strategic Repetition of Critical Constraints**

Repeat the 3 most violated constraints at exactly 3 locations:
1. Top of skill (action-identity fusion, 1a above)
2. At spawn-time section (where the agent is deciding HOW to spawn)
3. At completion section (where the agent is deciding HOW to close)

Three repetitions at decision points is more effective than one declaration in a constraints section.

#### Layer 2: Tool-Layer Enforcement (Infrastructure-Level — Do Second)

**2a. Claude Code Hook for Orchestrator Sessions**

Create a Claude Code hook that detects orchestrator sessions and intercepts prohibited tool usage:

```
# .claude/hooks/orchestrator-guard.sh
# Triggered on: tool calls
# Detects: ORCH_ORCHESTRATOR=1 environment variable

if [[ "$ORCH_ORCHESTRATOR" == "1" ]]; then
  case "$TOOL_NAME" in
    "Task")
      echo "⚠️ ORCHESTRATOR ACTION VIOLATION: You used the Task tool. Orchestrators spawn workers via 'orch spawn SKILL task' or 'bd create -l triage:ready'. The Task tool is for Claude Code default agents, not orchestrator delegation."
      ;;
    "Edit"|"Write")
      echo "⚠️ ORCHESTRATOR ACTION VIOLATION: You used $TOOL_NAME. Orchestrators don't edit code. Delegate via 'orch spawn' or 'bd create'."
      ;;
  esac
fi
```

**Note:** The exact hook mechanism depends on Claude Code's hook API capabilities. This may need to be a `PreToolCall` hook or a `PostToolCall` feedback injection. The key requirement is: the violation message must appear in the agent's context IMMEDIATELY when the prohibited tool is used, creating real-time "pain as signal."

**2b. Orchestrator Session Detection**

Set `ORCH_ORCHESTRATOR=1` in the environment for orchestrator sessions. This could be:
- Set by the `orchestrator-session.ts` OpenCode plugin
- Set by the `load-orchestration-context.py` Claude Code hook
- Set by `orch spawn orchestrator` when spawning orchestator sessions
- Detected by checking if the orchestrator skill is loaded (presence of specific CLAUDE.md content)

**2c. Graduated Response**

First violation: Warning message injected into context (pain as signal)
Second violation: Stronger warning with explicit correction
Third violation: Block the tool call entirely (if hook API supports it)

This graduated approach avoids false-positive friction while providing escalating enforcement.

### Alternative Approaches Considered

**Option B: Prompt-only fix (skill restructuring without tool enforcement)**
- **Pros:** Simpler to implement, no infrastructure changes needed
- **Cons:** Research and evidence show prompt-level constraints are insufficient against system-level competing instructions. The 17:1 signal ratio remains. Will likely achieve ~60-70% compliance but not 90%+.
- **When to use instead:** As an interim fix while Layer 2 is being built

**Option C: Infrastructure-only fix (tool enforcement without skill restructuring)**
- **Pros:** Deterministic enforcement, doesn't depend on prompt salience
- **Cons:** Enforcement without understanding leads to frustration. Agent won't understand WHY it can't use Task tool. Error messages without context are noise. Also, hook/plugin API limitations may prevent full enforcement.
- **When to use instead:** Never in isolation — always pair with clear skill instructions explaining the rationale

**Option D: Modify Claude Code system prompt for orchestrator sessions**
- **Pros:** Would solve the instruction hierarchy problem at its root
- **Cons:** Claude Code system prompt is not user-modifiable. Would require modifying Claude Code source code, which is out of scope.
- **When to use instead:** If Anthropic adds support for user-customizable system prompt sections

**Rationale for recommendation:** Option A (two-layer) addresses both the salience problem (Layer 1) and the enforcement gap (Layer 2), matching the dual nature of the problem. Neither prompt improvements nor infrastructure alone addresses both failure mechanisms.

---

### Implementation Details

**What to implement first:**
- Layer 1a (action-identity fusion at top of skill) — highest impact, lowest effort
- Layer 1d (strategic repetition at spawn and completion sections) — reinforces 1a
- Layer 1c (reduce skill length) — improves signal-to-noise

**Things to watch out for:**
- ⚠️ Skill is auto-generated by skillc — edits must go to `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template`, not compiled output
- ⚠️ 7 stale copies of orchestrator skill exist (per probe 2026-02-17) — deployment must clean old locations
- ⚠️ Plugin init-time caching means skill updates require OpenCode server restart
- ⚠️ Tool-layer enforcement depends on Claude Code hook API capabilities — needs spike to confirm feasibility

**Areas needing further investigation:**
- Claude Code hook API: Can hooks intercept tool calls and inject feedback? Need to read hook documentation.
- Graduated enforcement: What's the right escalation curve? (warning → stronger warning → block)
- False positive rate: How to reliably detect orchestrator sessions vs worker sessions to avoid enforcing constraints on workers
- A/B testing: How to measure compliance improvement after skill restructuring

**Success criteria:**
- ✅ Orchestrator sessions use `orch spawn` (not Task tool) when asked to spawn workers — target >90% compliance
- ✅ Orchestrator sessions use `orch complete` (not `bd close`) when closing completed work — target >90% compliance
- ✅ When identity is queried AND action is observed, both are consistent (no more "knows it's orchestrator but uses Task tool")
- ✅ Skill length reduced from 640 to <450 lines while retaining all essential guidance

---

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when promoting** because:
- This addresses a recurring problem (3+ prior investigations on orchestrator compliance)
- Establishes constraints that future skill rewrites might violate
- Future orchestrator skill edits need to respect the action-identity fusion pattern

**Suggested blocks keywords:**
- orchestrator skill
- action constraints
- behavioral compliance
- Task tool
- orch spawn

---

## References

**Files Examined:**
- `~/.claude/skills/meta/orchestrator/SKILL.md` (640 lines) — deployed orchestrator skill, action constraints at lines 68, 75, 594-604
- `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/SKILL.md.template` (690 lines) — source template
- `~/orch-knowledge/skills/src/meta/orchestrator/.skillc/skill.yaml` — skill metadata
- `~/.kb/principles.md` — system principles, especially "Infrastructure Over Instruction"
- Claude Code system prompt — Task tool promotion, ~500 words

**External Documentation:**
- [ICLR 2025: Improving LLM Safety with Instruction Hierarchy](https://proceedings.iclr.cc/paper_files/paper/2025/file/ea13534ee239bb3977795b8cc855bacc-Paper-Conference.pdf) — Instruction Segment Embedding for prioritizing competing instructions
- [AgentSpec (ICSE 2026): Customizable Runtime Enforcement](https://cposkitt.github.io/files/publications/agentspec_llm_enforcement_icse26.pdf) — Runtime guardrails for LLM agent actions
- [PCAS: Policy Compiler for Agentic Systems](https://arxiv.org/html/2602.16708v1) — Deterministic enforcement via reference monitors

**Related Artifacts:**
- **Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-02-24-probe-orchestrator-skill-behavioral-compliance.md` — Companion probe for this investigation
- **Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-02-16-orchestrator-skill-orientation-redesign.md` — Skill organized around identity, not needs
- **Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-02-17-orchestrator-skill-injection-path-trace.md` — 5 injection paths, caching bugs, stale versions
- **Probe:** `.kb/models/orchestrator-session-lifecycle/probes/2026-02-18-orchestrator-skill-cli-staleness-audit.md` — 13 stale references in skill
- **Investigation:** `.kb/investigations/2026-01-06-inv-diagnose-orchestrator-skill-18-completion.md` — 18% completion rate is by design

---

## Investigation History

**2026-02-24 08:00:** Investigation started
- Initial question: Why do orchestrator agents comply with identity but not action constraints?
- Context: Spawned from orch-go-1178 after Toolshed orchestrator used Task tool and bd close despite correctly identifying as orchestrator

**2026-02-24 08:30:** Evidence gathering complete
- Read deployed skill, Claude Code system prompt, 3 prior probes, principles.md
- Identified 6 competing forces causing action non-compliance
- Calculated 17:1 signal ratio disadvantage for action constraints

**2026-02-24 09:00:** Synthesis and recommendations complete
- Diagnosed root cause as instruction hierarchy + signal ratio + prompt-only "enforcement"
- Proposed two-layer fix: skill restructuring + tool-layer enforcement
- Aligned with "Infrastructure Over Instruction" principle and field research

**2026-02-24 09:30:** Investigation completed
- Status: Complete
- Key outcome: Action non-compliance is structural (instruction hierarchy + signal ratio), not informational. Fix requires both prompt restructuring for salience AND infrastructure enforcement for durability.
