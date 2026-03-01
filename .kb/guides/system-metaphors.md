# System Metaphors as Comprehension Aids

**Purpose:** Catalog the metaphor domains used in orch-go, explain what system concepts they map to, and provide guidance for extending the metaphor vocabulary consistently.

**Last verified:** Feb 28, 2026

**Why this matters:** Agents working in this codebase inherit these metaphors through CLAUDE.md, skills, and spawn context. When metaphors are consistent, agents reason correctly about system behavior by analogy. When metaphors conflict or are misapplied, agents make wrong assumptions about how components relate.

---

## How Metaphors Work Here

orch-go is an agent orchestration system. The domain is abstract — "coordinate AI agents to do software work" has no physical referent. Metaphors bridge this gap by mapping unfamiliar system concepts onto familiar domains.

**Good metaphors carry inference:** If an agent "spawns," you correctly infer it has a lifecycle, can become orphaned, and eventually dies. You don't need to be told these things — the metaphor carries them. This is the comprehension aid: the metaphor does work that documentation would otherwise need to do explicitly.

**Metaphor failure mode:** When two metaphor domains conflict, agents inherit contradictory intuitions. If we call agents both "workers" (industrial) and "organisms" (biological), which wins when we need a term for "agent that stopped responding"? Industrial says "broken machine." Biological says "dead." We chose biological ("stalled," "dead," "zombie") — and that choice must be consistent.

---

## The Core Metaphor Domains

### 1. Biological: Agents as Organisms

**The dominant metaphor.** Agents are born, live, and die. This is the most load-bearing metaphor in the system.

| Term | Maps To | Why It Works |
|------|---------|--------------|
| **spawn** | Create an agent session | Implies creation of an independent entity, not just starting a process |
| **lifecycle** | Agent state progression | Carries birth→maturity→death arc without explanation |
| **orphan** | Workspace without a session, or session without a workspace | Immediately communicates "parent is gone, this needs care" |
| **zombie** | Agent that finished but wasn't cleaned up | UNIX heritage — process that exited but wasn't reaped |
| **ghost** | State artifact from a terminated agent | Something that appears present but isn't real |
| **liveness** | Whether an agent is making progress | Borrowed from health checks — binary alive/dead |
| **heartbeat** | `bd comment "Phase: X"` updates | Regular signal proving the agent is still working |
| **health** | Whether infrastructure services are running | `orch doctor` checks system health |

**Extended into nervous system:** The coaching plugin is explicitly called a "digital nervous system." Agents "feel" friction through their "sensory stream" (tool-layer messages). This extension works because it preserves the organism frame — organisms have nervous systems that carry pain signals.

**Where it appears:** CLAUDE.md (spawn backends, architecture), `pkg/spawn/`, `pkg/daemon/`, agent-lifecycle.md, all skill files.

---

### 2. Military/Hierarchy: Command and Delegation

**The coordination metaphor.** Defines who decides what.

| Term | Maps To | Why It Works |
|------|---------|--------------|
| **orchestrator** | Strategic coordinator session | Implies conducting many parts toward a unified result |
| **worker** | Implementation agent | Clear subordinate role with defined scope |
| **authority** | Decision-making rights per tier | Carries chain-of-command intuition |
| **escalate** | Route a decision to a higher tier | Military "send up the chain" — implies urgency and hierarchy |
| **triage** | Issue classification for daemon | Medical/military: classify by urgency, not arrival order |
| **constitution** | Hard limits on agent behavior | Evokes foundational law that overrides all authority |
| **bypass** | Override a safety gate | Implies deliberate circumvention with accountability |

**Key insight:** The military metaphor defines the *authority* structure. The biological metaphor defines the *lifecycle* structure. They don't conflict because they address different concerns — an organism can exist within a hierarchy.

**Where it appears:** decision-authority.md, worker-base skill, CLAUDE.md (authority section), all skill files.

---

### 3. Gates and Phases: Process Control

**The safety metaphor.** Defines what must be true before proceeding.

| Term | Maps To | Why It Works |
|------|---------|--------------|
| **gate** | Mandatory check that blocks progression | Physical gate: you cannot pass until it opens |
| **preflight** | Pre-spawn checks before side effects | Aviation: verify before takeoff, not during flight |
| **phase** | Named stage of agent work | Discrete, ordered, reportable stages |
| **checkpoint** | Orchestrator-blocking beads issue | A point where forward progress pauses for verification |
| **rollback** | Cleanup after failed spawn | Transaction semantics: undo partial work |
| **hook** | Event callback at lifecycle points | Fishing: something that catches events as they pass |

**Why "gate" dominates over "check":** A "check" implies you can proceed regardless. A "gate" implies you cannot. The orch-go system has hard gates (spawn aborts) and soft gates (warnings). The metaphor communicates the blocking nature that "validation" or "check" would not.

**Where it appears:** spawn.md, completion.md, completion-gates.md, all skill files (phase reporting).

---

### 4. Spatial: Workspaces and Boundaries

**The isolation metaphor.** Defines where agents live and what they can touch.

| Term | Maps To | Why It Works |
|------|---------|--------------|
| **workspace** | `.orch/workspace/{name}/` directory | Physical space an agent inhabits — isolated, contained |
| **hotspot** | File exceeding 1,500 lines | Heat metaphor: concentrated activity/complexity |
| **boundary** | File size limit requiring extraction | Physical edge that shouldn't be crossed |
| **window** | Tmux pane per agent | Literal window into agent's work |
| **cross-project** | Work spanning multiple repositories | Spatial: crossing a border |
| **attractor** | Well-named package that draws future code | Physics: gravitational pull toward the right location |

**The accretion/gravity extension:** Files grow via "accretion" (geological: slow accumulation). The fix is "extraction" into "attractor packages" that exert "gravity" on future code. This physics extension works because it explains *why* extraction succeeds — it's not just moving code, it's creating a gravitational center that naturally routes future additions.

**Where it appears:** CLAUDE.md (accretion boundaries), code-extraction-patterns.md, workspace-lifecycle.md.

---

### 5. Navigation: Orient and Focus

**The wayfinding metaphor.** Defines how agents and orchestrators find direction.

| Term | Maps To | Why It Works |
|------|---------|--------------|
| **orient** | Build situational awareness at session start | Military OODA: Observe-Orient-Decide-Act |
| **north star** | Current primary focus (`~/.orch/focus.json`) | Navigation: fixed reference point for direction |
| **frame** | Compressed statement of why work matters | Photography/perception: what you see depends on your frame |
| **reconnect** | Resurface context during completion review | Restore a broken connection to original purpose |
| **drift** | Agent diverging from intended behavior | Navigation: moving off course without realizing |

**The ORIENT-DELEGATE-RECONNECT loop:** The orchestrator's core loop uses navigation metaphors end-to-end. Orient (find direction) → Delegate (send agents) → Reconnect (return to the human's frame). This metaphor carries the insight that orchestration is fundamentally about maintaining direction across distributed work.

**Where it appears:** orchestrator skill, spawned-orchestrator-pattern.md, CLAUDE.md.

---

### 6. Knowledge: Synthesis and Provenance

**The epistemic metaphor.** Defines how the system learns and remembers.

| Term | Maps To | Why It Works |
|------|---------|--------------|
| **synthesis** | `SYNTHESIS.md` — agent's structured output | Chemistry: combining elements into something new |
| **probe** | Targeted test of a model claim | Scientific: controlled test of a hypothesis |
| **investigation** | Open-ended exploration | Detective: gather evidence, form conclusions |
| **model** | `.kb/models/` — persistent understanding | Scientific: simplified representation of reality |
| **provenance** | Whether a claim traces to external evidence | Art/science: documented origin and chain of custody |
| **atom** | Individual finding before synthesis | Chemistry: smallest unit that retains identity |

**Session amnesia and handoff:** "Session amnesia" names the core problem — each new Claude session starts with no memory. "Handoff" names the solution — passing context forward like a relay baton. These metaphors make the abstract problem (LLM context windows are ephemeral) concrete and actionable.

**Where it appears:** understanding-artifact-lifecycle.md, all skill files (D.E.K.N. template), CLAUDE.md (knowledge placement table).

---

### 7. Ecological: Failure Modes

**The systems-thinking metaphor.** Describes how the system can degrade.

| Term | Maps To | Why It Works |
|------|---------|--------------|
| **entropy spiral** | Runaway agent activity making things worse | Thermodynamics: disorder increases without energy input |
| **death spiral** | Circular dependency blocking recovery | Aviation: control inputs worsen the situation |
| **escape hatch** | Independent path surviving primary failure | Architecture: emergency exit |
| **strangler fig** | Gradual replacement via extraction | Biology: new growth slowly replaces old structure |
| **circuit breaker** | Automatic fallback on failure | Electrical: protection against overload |

**Why ecological, not mechanical:** Mechanical failure is simple — a part breaks, you replace it. Ecological failure is emergent — the system is "working" at every individual level but degrading as a whole. The orch-go entropy spirals (Dec 27 – Jan 2, Jan 18 – Feb 12) were ecological: each agent completed its task, but the aggregate effect was destructive. "Entropy spiral" captures this better than "bug" or "failure."

**Where it appears:** resilient-infrastructure-patterns.md, CLAUDE.md (pain as signal principle).

---

## Metaphor Interactions

The metaphor domains are not independent. They compose:

```
Biological (what agents ARE)
  + Military (how agents RELATE)
  + Gates (what agents MUST DO)
  + Spatial (where agents LIVE)
  + Navigation (how agents ORIENT)
  + Knowledge (what agents PRODUCE)
  + Ecological (how agents FAIL)
```

**Example composition:** An orchestrator (military) *spawns* (biological) a worker into a *workspace* (spatial). The worker passes *preflight gates* (process), *orients* (navigation) via SPAWN_CONTEXT, produces a *synthesis* (knowledge) artifact, and reports *Phase: Complete* (process). If the worker becomes a *zombie* (biological), the system avoids an *entropy spiral* (ecological) via *triage* (military/medical).

This sentence is comprehensible because the metaphors are consistent within their domains and compose without contradiction.

---

## Guidelines for Extending Metaphors

### Do: Extend Within Established Domains

When naming a new concept, first check if an existing metaphor domain covers it.

| New concept | Wrong choice | Right choice | Why |
|------------|-------------|-------------|-----|
| Agent that stopped responding | "broken agent" (mechanical) | "stalled agent" (biological) | Agents are organisms, not machines |
| New mandatory check before spawn | "spawn validator" (generic) | "spawn gate" (process) | Gates are the established blocking-check metaphor |
| Agent output document | "report" (generic) | "synthesis" (knowledge) | Knowledge domain owns agent outputs |
| Priority work item | "priority queue" (CS) | "north star" (navigation) | Navigation domain owns direction-finding |

### Don't: Mix Metaphor Domains for the Same Concept

**Anti-pattern:** Calling agents "workers" (industrial) in one context and "organisms" (biological) in another, then needing to describe a failed agent. Is it "broken" or "dead"? The mixed metaphor forces readers to context-switch.

**Resolution:** When domains conflict, the biological metaphor wins for agent state. The military metaphor wins for agent relationships. The process metaphor wins for workflow mechanics.

### Don't: Introduce New Domains Without Justification

A new metaphor domain is expensive — every agent that reads the documentation must learn it. Only introduce a new domain when existing domains genuinely cannot express the concept.

**Test:** "Can I express this using spawn/lifecycle/gate/workspace/orient vocabulary?" If yes, use existing vocabulary. If no, the new domain is justified — but document it in this guide.

### Do: Preserve Inference Chains

The value of a metaphor is the inferences it carries for free. When extending a metaphor, check that the inferences still hold.

**Good extension:** "Agent heartbeat" → agents have heartbeats → if heartbeat stops, agent might be dead → check liveness. Every inference is correct.

**Bad extension:** "Agent metabolism" → agents metabolize... tokens? → higher metabolism means... faster? more expensive? The inferences don't map cleanly. This metaphor would confuse more than it clarifies.

---

## Quick Reference: Which Domain Owns What

| Concept | Domain | Key Terms |
|---------|--------|-----------|
| Agent creation and termination | Biological | spawn, lifecycle, orphan, zombie, ghost |
| Agent behavior monitoring | Biological (nervous) | pain as signal, sensory stream, liveness, heartbeat |
| Who decides what | Military | orchestrator, worker, authority, escalate, constitution |
| Issue classification | Military/Medical | triage, diagnosis, symptom |
| Mandatory checks | Process | gate, preflight, checkpoint, phase |
| Workflow progression | Process | phase, transition, hook, rollback |
| Agent isolation | Spatial | workspace, window, boundary, hotspot |
| Code growth patterns | Spatial/Physics | accretion, attractor, strangler fig |
| Direction and purpose | Navigation | orient, north star, frame, reconnect, drift |
| Learning and output | Knowledge | synthesis, probe, investigation, model, provenance |
| System degradation | Ecological | entropy spiral, death spiral, escape hatch |

---

## References

- **CLAUDE.md:** Spawn backends, pain as signal, accretion boundaries
- **agent-lifecycle.md:** Biological metaphor in full detail
- **decision-authority.md:** Military/hierarchy metaphor
- **completion-gates.md:** Gate metaphor
- **resilient-infrastructure-patterns.md:** Ecological metaphor (escape hatch, death spiral)
- **spawn.md:** Biological + process metaphors composed
- **understanding-artifact-lifecycle.md:** Knowledge metaphor
