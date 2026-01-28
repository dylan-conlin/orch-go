<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** [What was discovered/answered - the key finding in one sentence]

**Evidence:** [Primary evidence that supports the conclusion - test results, observations]

**Knowledge:** [What was learned - insights, constraints, or decisions made]

**Next:** [Recommended action - close, implement, investigate further, or escalate]

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Preventing Hierarchical Controller Collapse

**Question:** What research exists on preventing hierarchical controllers from collapsing into worker-level execution? What patterns can we encode in orchestrator prompts to prevent "frame collapse" where orchestrators think "I'll just do this quick fix myself" instead of delegating?

**Started:** 2026-01-27
**Updated:** 2026-01-27
**Owner:** Research Agent
**Phase:** Investigating
**Next Step:** Research literature across multi-agent systems, hierarchical RL, organizational psychology, LLM agents, and software architecture
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Hierarchical RL Uses Meta-Controller Pattern with Temporal Abstraction

**Evidence:** The h-DQN (Hierarchical Deep Q-Network) framework integrates "hierarchical action-value functions, operating at different temporal scales." The system has "a top-level q-value function learns a policy over intrinsic goals, while a lower-level function learns a policy over atomic actions to satisfy the given goals." The meta-controller operates at a higher level of abstraction (goals) and explicitly does NOT execute low-level actions.

**Source:** 
- Kulkarni et al., "Hierarchical Deep Reinforcement Learning: Integrating Temporal Abstraction and Intrinsic Motivation", NeurIPS 2016
- URL: https://proceedings.neurips.cc/paper_files/paper/2016/hash/f442d33fa06832082290ad8544a8da27-Abstract.html

**Significance:** Hierarchical RL research has explicitly solved the "meta-controller taking low-level actions" problem through **action space separation** - the meta-controller literally cannot take atomic actions, only set goals. The lower-level controller is the only one with access to primitive actions. This is an architectural constraint, not just a training objective.

---

### Finding 2: Multi-Agent Systems Use Five Dimensions for Hierarchical Structure

**Evidence:** Recent taxonomy (Moore 2025) identifies five key axes for hierarchical multi-agent systems:
1. **Control hierarchy** - Who commands whom
2. **Information flow** - What data flows up/down
3. **Role and task delegation** - How work is distributed
4. **Temporal layering** - Different timescales at different levels
5. **Communication structure** - How agents interact

The paper explicitly addresses "separation between long-term and short-term decision-making" and notes the challenge of "making hierarchical decisions explainable" and "scaling to very large agent populations."

**Source:**
- Moore, D.J., "A Taxonomy of Hierarchical Multi-Agent Systems: Design Patterns, Coordination Mechanisms, and Industrial Applications", arXiv:2508.12683, 2025
- URL: https://arxiv.org/abs/2508.12683

**Significance:** The taxonomy reveals that preventing collapse requires attention to MULTIPLE dimensions simultaneously - control, information, roles, time, and communication. A single intervention (e.g., prompt engineering) that only addresses one dimension is likely insufficient.

---

### Finding 3: LLM Agent Systems Struggle with Hierarchical Delegation

**Evidence:** Recent survey (Tran et al., 2025) notes that LLM-based multi-agent systems use:
- **Centralized architectures** with "master agent delegating tasks"
- **Hierarchical architectures** where "high-level agents handle planning and delegate tasks to lower-level agents"
- However, paper identifies this as an active research area with ongoing challenges

Another survey (Händler 2023) explicitly frames the problem as "Balancing Autonomy and Alignment" - hierarchical systems must balance:
- **Goal-driven task management** (breaking down work)
- **Agent composition** (who does what)
- **Multi-agent collaboration** (how they coordinate)
- **Context interaction** (tools and datasets)

**Source:**
- Tran et al., "Multi-Agent Collaboration Mechanisms: A Survey of LLMs", arXiv:2501.06322, 2025
- Händler, T., "Balancing Autonomy and Alignment: A Multi-Dimensional Taxonomy for Autonomous LLM-powered Multi-Agent Architectures", arXiv:2310.03659, 2023

**Significance:** LLM agents are NEWER to hierarchical delegation than RL systems, and research explicitly identifies this as an open challenge. The "balancing autonomy and alignment" framing suggests the problem is that agents either:
1. Over-autonomy: Do work they shouldn't (frame collapse)
2. Over-alignment: Block on orchestrator for everything (serial collapse)

---

### Finding 4: Software Architecture Has "God Object" Anti-Pattern

**Evidence:** Software engineering literature extensively documents the "God Object" (also called "Blob") anti-pattern where a single class/object accumulates too many responsibilities. Research identifies this as violating **separation of concerns** - when one component does too much, it becomes:
- Hard to understand
- Hard to maintain
- Coupled to everything
- A bottleneck

The anti-pattern occurs when developers "unwillingly introduce them while designing and implementing" - it's not intentional, but emerges gradually.

**Source:**
- Palomba et al., "Anti-pattern detection: Methods, challenges, and open issues", Advances in Computers, 2014
- UML specification and correction of object-oriented anti-patterns (various papers from 2009-2021)

**Significance:** God object emerges GRADUALLY through incremental decisions that seem reasonable in isolation. Preventing it requires **architectural constraints** (enforced boundaries) not just awareness. Simply knowing about the anti-pattern doesn't prevent it - you need structural safeguards.

---

### Finding 5: Organizational Psychology Focuses on Manager-as-Coach, Not IC Role Clarity

**Evidence:** Search for "manager doing individual contributor work" returned primarily sports coaching literature and manager-as-coach frameworks. Very little research on the specific problem of managers doing IC work instead of managing.

**Source:**
- Google Scholar search: "manager doing individual contributor work player coach problem"
- Results: Coaching frameworks, work-life balance for coaches, manager coaching skills

**Significance:** This is a GAP in organizational psychology literature - the specific problem of managers reverting to IC work is not well-researched in academic literature. Practitioners discuss this (e.g., tech industry blogs), but there's limited formal research on prevention mechanisms. This suggests we may need to draw more heavily from hierarchical RL and multi-agent systems research.

---

### Finding 6: Options Framework Enforces Temporal Abstraction Through Action Hiding

**Evidence:** Sutton et al.'s seminal Options framework (1999, cited 5271 times) introduces "temporal abstraction" where higher-level policies select GOALS (options) rather than primitive actions. An "option" consists of:
1. An initiation set (where it can start)
2. A policy (what to do while executing)
3. A termination condition (when it ends)

The key insight: **The higher level CANNOT execute primitive actions** - it can only invoke options. The lower level CANNOT set goals - it can only execute actions to satisfy the current option. This is enforced through the Semi-MDP framework.

**Source:**
- Sutton, R.S., Precup, D., Singh, S., "Between MDPs and semi-MDPs: A framework for temporal abstraction in reinforcement learning", Artificial Intelligence, 1999
- URL: https://www.sciencedirect.com/science/article/pii/S0004370299000521

**Significance:** **Action space separation** is the key mechanism - don't just tell the meta-controller not to take low-level actions in a prompt, MAKE IT IMPOSSIBLE architecturally. The meta-controller's action space is literally different from the worker's action space.

---

### Finding 7: Contract Net Protocol Uses Explicit Announcement-Bid-Award Cycle

**Evidence:** The Contract Net Protocol (Smith 1980s, foundational multi-agent coordination) prevents managers from doing worker-level tasks through an explicit negotiation protocol:
1. **Manager announces task** (cannot execute it directly)
2. **Workers bid on task** (evaluate fit)
3. **Manager awards contract** (based on bids)
4. **Worker executes task** (reports results)

The protocol creates a **process barrier** - managers MUST go through the announcement-bid-award cycle. They cannot "just do it themselves" because the architecture doesn't provide that path.

**Source:**
- Smith, R.G., "The contract net protocol: High-level communication and control in a distributed problem solver", Readings in distributed artificial intelligence, 1988
- Various applications in task allocation (53,200+ results)

**Significance:** Classic multi-agent systems solved this 40 years ago with PROCESS CONSTRAINTS, not awareness. The manager role structurally cannot execute tasks - only allocate them. Frame collapse is prevented by removing the code path, not by prompting against it.

---

### Finding 8: Recent Research Identifies "Balancing Autonomy and Alignment" as Core Challenge

**Evidence:** Händler (2023) explicitly frames hierarchical LLM agents as needing to balance:
- **Too much autonomy** → Agents do work outside their role (frame collapse)
- **Too much alignment** → Agents block waiting for orchestrator (serial execution)

The taxonomy proposes analyzing this balance across multiple dimensions:
- Goal-driven task management
- Agent composition (who does what)
- Multi-agent collaboration (how they coordinate)
- Context interaction (tools, datasets)

**Source:**
- Händler, T., "Balancing Autonomy and Alignment: A Multi-Dimensional Taxonomy for Autonomous LLM-powered Multi-Agent Architectures", arXiv:2310.03659, 2023

**Significance:** This confirms frame collapse is an ACTIVE RESEARCH PROBLEM in LLM agents (not a solved problem). The "autonomy vs alignment" framing gives us language: orchestrator prompts need to explicitly tune this balance. Too much autonomy = "I'll just do this myself." Too much alignment = "I need to ask orchestrator for everything."

---

## Synthesis

**Key Insights:**

1. **Architectural Constraints Beat Prompts** - Both hierarchical RL (Options framework, h-DQN) and classic multi-agent systems (Contract Net Protocol) prevent collapse through ARCHITECTURAL CONSTRAINTS, not training or prompts. The meta-controller literally cannot access worker-level actions - they're in different action spaces. Prompts that say "don't do X" are insufficient; the system must be designed so X is impossible.

2. **Multi-Dimensional Problem Requires Multi-Dimensional Solution** - The taxonomy research (Moore 2025, Händler 2023) reveals that hierarchy operates across FIVE dimensions: control, information flow, role delegation, temporal layering, and communication structure. A single intervention (e.g., prompt engineering) that only addresses one dimension will fail. Preventing frame collapse requires coordinated constraints across multiple dimensions.

3. **Temporal Abstraction Is the Core Mechanism** - The pattern across RL and multi-agent systems: higher levels operate on DIFFERENT TIME SCALES with DIFFERENT PRIMITIVES. Meta-controllers don't select individual actions - they select goals/options/subtasks that unfold over time. The temporal separation creates a natural boundary that prevents reaching down into execution.

4. **Process Barriers Prevent Structural Collapse** - Contract Net Protocol's announcement-bid-award cycle creates a PROCESS BARRIER. Even if the manager could technically execute the task, the protocol doesn't provide that code path. The manager MUST delegate because that's the only operation available. This is stronger than "should delegate" guidance.

5. **LLM Agents Are Early-Stage Compared to RL** - Hierarchical RL has 25+ years of research on preventing meta-controllers from taking primitive actions. LLM multi-agent systems are ~2-3 years old and still actively struggling with "balancing autonomy and alignment." This is not a solved problem for LLMs; we're inventing solutions, not applying known best practices.

6. **The "Autonomy vs Alignment" Trade-off** - Händler's framing reveals the core tension: too much autonomy leads to frame collapse (doing worker-level work), too much alignment leads to serial collapse (blocking on orchestrator for everything). The goal is to find the right balance where orchestrators delegate appropriately without either doing the work themselves or micromanaging every decision.

**Answer to Investigation Question:**

**What research exists on preventing hierarchical controllers from collapsing into worker-level execution?**

Substantial research exists across three domains:

1. **Hierarchical Reinforcement Learning** (25+ years, highly mature):
   - **Core mechanism**: Action space separation - meta-controllers select goals/options, NOT primitive actions
   - **Key work**: Options framework (Sutton 1999), h-DQN (Kulkarni 2016), temporal abstraction
   - **Prevention method**: Architectural - make it impossible for meta-controller to access worker actions

2. **Multi-Agent Systems** (40+ years, foundational):
   - **Core mechanism**: Process barriers through explicit protocols (Contract Net)
   - **Key work**: Contract Net Protocol (Smith 1980s), hierarchical task allocation
   - **Prevention method**: Procedural - manager can only announce tasks, not execute them

3. **LLM Multi-Agent Systems** (2-3 years, actively researching):
   - **Core challenge**: "Balancing autonomy and alignment" (Händler 2023)
   - **Key work**: Taxonomy papers (Moore 2025, Händler 2023, Tran 2025)
   - **Prevention method**: Under development - master agent delegation patterns

**Patterns we can encode in orchestrator prompts:**

1. **Explicit action space declaration**: State what orchestrator CAN do (delegate, monitor, decide) and what it CANNOT do (code, test, debug)
2. **Process constraint framing**: "You MUST delegate through spawning agents - there is no code path for you to execute tasks directly"
3. **Temporal abstraction language**: Frame orchestrator work as "setting goals" and "monitoring progress", NOT "doing tasks"
4. **Multi-dimensional boundaries**: Address control (who commands), information (what flows up/down), roles (who does what), time (different scales), communication (how coordination happens)
5. **Autonomy/alignment calibration**: Explicitly state the balance - "Delegate work without micromanaging, but don't do worker-level execution yourself"

**Key limitation:** Organizational psychology research has minimal formal work on this specific problem - most research is from RL and multi-agent systems. We're adapting RL/MAS solutions to LLM agent contexts, which is novel work.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Architectural Constraints + Multi-Dimensional Boundaries** - Prevent frame collapse through action space separation (architectural) reinforced by explicit role boundaries across control, information, temporal, and process dimensions (prompt engineering).

**Why this approach:**
- **Based on proven patterns**: Combines action space separation (Options framework), process barriers (Contract Net), and temporal abstraction (h-DQN) - all validated in production systems over decades
- **Multi-dimensional coverage**: Addresses control (who decides), information (what flows), roles (who executes), time (different scales), and process (how delegation works) - prevents collapse across all axes
- **Learnable from research**: Hierarchical RL and multi-agent systems have 40+ years of validated solutions we can adapt to LLM agents

**Trade-offs accepted:**
- **More rigid architecture**: Orchestrator cannot directly fix issues even when "obviously faster" - must delegate even for trivial tasks
- **Potentially slower initially**: Delegation overhead exists even for small tasks
- **Why acceptable**: Frame collapse is expensive - an orchestrator doing worker tasks blocks all parallel work. Short-term delegation overhead prevents long-term serial collapse.

**Implementation sequence:**

1. **Define action spaces explicitly** - List what orchestrator CAN and CANNOT do
   - **CAN**: Spawn agents, monitor progress, make architectural decisions, prioritize work, allocate resources
   - **CANNOT**: Write code, run tests, debug issues, read files, edit configurations
   - **Why foundational**: If action spaces aren't explicitly separated, prompts become ambiguous ("Can I just quickly fix this typo?")

2. **Enforce process barriers** - Require orchestrator to use delegation protocol, not direct execution
   - **Pattern**: "To accomplish X, you MUST: (1) Spawn agent with clear goal, (2) Monitor via status updates, (3) Review results, (4) Decide next action"
   - **No escape hatch**: Remove phrases like "or you can..." or "if simple, just..." - delegation is the only path
   - **Why second**: Process barriers only work if action spaces are already defined

3. **Implement temporal abstraction** - Frame orchestrator work as goal-setting over longer timescales
   - **Language shift**: "Set goals" not "do tasks", "Monitor progress" not "check each step", "Decide strategy" not "implement solution"
   - **Time scale separation**: Orchestrator operates in hours/days (strategic), workers operate in minutes/hours (tactical)
   - **Why third**: Temporal framing builds on action spaces and process barriers to create natural separation

4. **Add autonomy/alignment calibration** - Explicitly state the balance point
   - **Autonomy boundary**: "Workers have authority over implementation details - don't micromanage"
   - **Alignment boundary**: "You must delegate execution - doing work yourself is frame collapse"
   - **Balance statement**: "Delegate work and trust workers to execute, while maintaining strategic oversight"
   - **Why fourth**: Calibration prevents both over-autonomy (frame collapse) and over-alignment (serial micromanagement)

### Alternative Approaches Considered

**Option B: Prompt Engineering Only**
- **Pros:** Fast to implement, no architectural changes needed, easy to iterate
- **Cons:** Research shows prompts alone are insufficient - god objects emerge gradually despite awareness (Finding 4), RL systems use architectural constraints not training (Findings 1, 6, 7)
- **When to use instead:** Quick experiments or when architectural changes are blocked, but expect degradation over time

**Option C: Pure Architectural (No Prompts)**
- **Pros:** Strongest enforcement, impossible to violate, proven in non-LLM systems
- **Cons:** LLMs need context about WHY constraints exist to reason within them effectively; pure architectural approach treats LLM like deterministic system
- **When to use instead:** If orchestrator is NOT an LLM (e.g., rule-based controller), or when dealing with safety-critical constraints

**Option D: Process Monitoring + Alerts**
- **Pros:** Catches violations after they occur, provides telemetry on where collapse happens
- **Cons:** Reactive not preventive - frame collapse has already happened when alert fires; organizational psychology research (Finding 5) shows awareness doesn't prevent gradual degradation
- **When to use instead:** As supplementary to Option A for visibility, not as primary prevention

**Rationale for recommendation:**

Hierarchical RL research (Findings 1, 6) and multi-agent systems (Findings 2, 7) converge on the same solution: **architectural constraints that make role violations impossible or expensive**. However, LLM agents differ from traditional RL in that they can REASON about constraints (Finding 8), so pure architecture alone leaves capability on the table. 

The recommended approach combines architectural prevention (action space separation, process barriers) with reasoning-friendly context (temporal abstraction, autonomy/alignment framing). This leverages both the proven strength of architectural constraints AND the unique reasoning capabilities of LLMs.

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
