## Summary (D.E.K.N.)

**Delta:** The orch-ecosystem is an amnesia-resilient AI orchestration framework that emerged from practical work at SendCutSend, now stabilizing into a coherent philosophy with clear identity.

**Evidence:** 8 repos, 1195 beads issues, 1179 workspaces, 265+ investigations, 6 foundational principles documented. Architecture follows Session Amnesia as foundational constraint.

**Knowledge:** This is not a productivity tool or project management system - it's infrastructure for AI agents working across sessions. The core insight is that LLM memory loss is a design constraint requiring specific patterns (externalization, surfacing, gating).

**Next:** Create decision document capturing identity and audience. Recommend: personal system first, selective sharing of patterns (not tools).

**Confidence:** High (85%) - clear patterns emerged, but audience/future questions require Dylan's input.

---

# Investigation: What is orch-ecosystem?

**Question:** What is the orch-ecosystem? What patterns/philosophy emerged? What is its identity (orchestration framework? productivity system? new way of working with AI?)? What audience does it serve (just Dylan? solo devs? teams? OSS?)?

**Started:** 2025-12-24
**Updated:** 2025-12-24
**Owner:** Design-session agent
**Phase:** Complete
**Next Step:** Review with Dylan, produce decision if direction is clear
**Status:** Complete
**Confidence:** High (85%)

---

## Context Gathered

### Scale of the System

The ecosystem has significant operational history:

| Metric | Count | Source |
|--------|-------|--------|
| Total beads issues (orch-go) | 1,195 | `bd stats` |
| Workspaces (orch-go) | 391 | `.orch/workspace/` |
| Workspaces (orch-knowledge) | 788 | `.orch/workspace/` |
| Investigations (orch-go) | 265 | `.kb/investigations/` |
| kn entries (orch-go) | 188 | `.kn/entries.jsonl` |
| Registered kb projects | 17 | `kb projects list` |
| Skills (worker category) | 10 | `~/.claude/skills/worker/` |
| Decisions tracked in orch-knowledge | 68 | README claims |
| Investigations in orch-knowledge | 299 | README claims |

**Total artifacts:** 390+ in orch-knowledge, 265+ investigations in orch-go, ~1180 workspaces combined.

### The 8 Repos

| Repo | Purpose | Primary CLI |
|------|---------|-------------|
| **orch-go** | Agent orchestration (spawn/complete lifecycle) | `orch` |
| **kb-cli** | Knowledge base management | `kb` |
| **beads** | Issue tracking (Yegge's OSS) | `bd` |
| **beads-ui-svelte** | Web UI for beads | - |
| **skillc** | Skill compiler | `skillc` |
| **agentlog** | Agent event logging | `agentlog` |
| **kn** | Quick knowledge capture | `kn` |
| **orch-cli** | Legacy Python orchestration | `orch-py` |

---

## Findings

### Finding 1: Session Amnesia is the Foundational Constraint

**Evidence:** From `~/.kb/principles.md`:
> "Every pattern in this system compensates for Claude having no memory between sessions... This is THE constraint. When principles conflict, session amnesia wins."

The entire architecture - SPAWN_CONTEXT.md, SYNTHESIS.md, investigation artifacts, beads comments for phase tracking - exists to enable resumption by a Claude instance with no memory of previous sessions.

**Source:** `~/.kb/principles.md:13-32`

**Significance:** This is the core insight. The ecosystem isn't about productivity or project management - it's about compensating for LLM memory loss. Every design decision flows from this constraint.

---

### Finding 2: Philosophy Crystallized into 6 Principles

**Evidence:** The principles document contains two categories:

**LLM-First Principles (universal to LLM work):**
1. Session Amnesia (foundational)
2. Self-Describing Artifacts
3. Progressive Disclosure
4. Surfacing Over Browsing
5. Evidence Hierarchy
6. Gate Over Remind

**System Design Principles (this system's choices):**
- Local-First
- Compose Over Monolith
- Graceful Degradation

**Meta Principles (how to evolve):**
- Evolve by Distinction
- Reflection Before Action

**Source:** `~/.kb/principles.md`

**Significance:** These principles aren't theoretical - they emerged from practice ("Discovered Nov 2025 after recognizing 'habit formation' was really amnesia compensation"). This is empirical philosophy, not ideological.

---

### Finding 3: Three-Tier Temporal Model for Artifacts

**Evidence:** From the Minimal Artifact Taxonomy decision:

| Tier | Location | Examples |
|------|----------|----------|
| Ephemeral (session-bound) | `.orch/workspace/` | SPAWN_CONTEXT.md, SYNTHESIS.md |
| Persistent (project-lifetime) | `.kb/` | Investigations, decisions |
| Operational (work-in-progress) | `.beads/` | Issues, comments |

**Source:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md`

**Significance:** The system distinguishes between session state (ephemeral), accumulated knowledge (persistent), and work tracking (operational). This separation enables both immediate agent work and long-term knowledge accumulation.

---

### Finding 4: Cross-Repo Architecture is Decentralized by Design

**Evidence:** From the Ideal Cross-Repo Setup investigation:
> "The existing architecture is mostly correct - kb already supports cross-repo search, beads per-repo isolation matches Yegge's design philosophy."

Key architectural choices:
- Each repo has own `.beads/` (per-repo isolation)
- `kb context --global` searches across 17 registered projects
- No centralized meta-repo needed
- `~/.orch/ECOSYSTEM.md` documents relationships

**Source:** `.kb/investigations/2025-12-22-inv-design-ideal-cross-repo-setup.md`

**Significance:** The decentralization is intentional, not an oversight. Tools coordinate without centralization. This matches Unix philosophy of composable tools.

---

### Finding 5: The Tool Ecosystem Emerged from Composition

**Evidence:** Current CLI ecosystem:
```
orch        → "agent coordination"     (spawn, monitor, complete, daemon)
beads (bd)  → "what work needs doing"  (issues, dependencies, tracking)
kb          → "deep documentation"     (investigations, decision records)
kn          → "what we've learned"     (quick decisions, constraints, failures)
agentlog    → "what happened"          (error/event logging for agents)
skillc      → "skill compilation"      (source → deployed skills)
```

Each tool does one thing. They compose via file system and CLI.

**Source:** Orchestrator skill, ECOSYSTEM.md

**Significance:** This isn't a monolithic "AI productivity suite." It's a Unix-philosophy composition of focused tools. The orchestrator coordinates, but doesn't subsume.

---

### Finding 6: Origins in Practical Work, Not Theory

**Evidence:** From orch-knowledge README:
> "My personal orchestration knowledge base... This is the 'learning archive' - 390+ artifacts documenting patterns discovered through real orchestration work."

From principles lineage:
> "Session Amnesia: Discovered Nov 2025 after recognizing 'habit formation' was really amnesia compensation"
> "Gate Over Remind: Named Dec 2025 after observing LLMs consistently fail to externalize knowledge despite reminders"

**Source:** `~/orch-knowledge/README.md`, `~/.kb/principles.md:250-256`

**Significance:** Everything here was discovered through practice, not designed from theory. The system is an empirical accumulation of what works when coordinating AI agents across sessions.

---

## Synthesis

### Key Insights

1. **The Core Insight: Amnesia as Design Constraint**

   Unlike traditional productivity tools that assume human continuity, this system is built around the constraint that every session starts fresh. The entire architecture - from SPAWN_CONTEXT.md embedding full skill context to beads comments tracking phase - exists because the next Claude won't remember this session.

2. **Identity: Infrastructure for AI Agent Sessions**

   This is not a productivity tool (for humans), not project management (like Jira), not a framework (like Django). It's **infrastructure for AI agents working across sessions**. The closest analogy might be:
   - How dotfiles configure shells → this configures AI sessions
   - How CI/CD orchestrates builds → this orchestrates agent work
   - How observability platforms track services → this tracks agent sessions

3. **Philosophy: Empirical, Not Ideological**

   The principles weren't designed - they were discovered. Each emerged from a failure mode:
   - "Session amnesia" named after realizing "habit formation" metaphor was wrong
   - "Gate over remind" named after reminders failed under cognitive load
   - "Evolve by distinction" named after Phase/Status conflation caused problems

4. **Architecture: Composition Over Monolith**

   8 repos, each doing one thing well, coordinated via file system and CLI. This enables:
   - Using Yegge's beads as-is (OSS relationship)
   - Evolving any tool independently
   - Mixing personal and upstream components

### Answer to Investigation Question

**What is orch-ecosystem?**

An amnesia-resilient AI orchestration framework that provides infrastructure for AI agents to:
1. Resume work without memory (via externalized state)
2. Coordinate across sessions (via spawn/complete lifecycle)
3. Accumulate knowledge (via kb/kn/investigations)
4. Track work (via beads issues)

**What patterns/philosophy emerged?**

Session Amnesia as foundational constraint → 6 principles emerged empirically:
- Self-describing artifacts, progressive disclosure, surfacing over browsing
- Evidence hierarchy, gate over remind, evolve by distinction

**What is its identity?**

Not a productivity tool, not project management, not a "new way of working with AI."

It's **infrastructure for AI agent sessions** - the equivalent of dotfiles, CI/CD, and observability for AI orchestration.

**What audience does it serve?**

Currently: Dylan's personal system, evolved through his work at SendCutSend.

Potentially: Solo developers doing significant AI-assisted work who need session continuity. But the tools are deeply personal (390+ artifacts reflecting Dylan's decisions, workflows, preferences).

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Clear patterns emerged from extensive evidence. The identity question has a clear answer based on architecture. The audience question is less certain because it depends on Dylan's intentions.

**What's certain:**

- ✅ Session amnesia is the foundational constraint (explicitly stated in principles)
- ✅ The architecture follows LLM-first principles (observable in every design choice)
- ✅ This is infrastructure, not a productivity app (no human-centric UX)
- ✅ Principles emerged from practice (documented lineage)

**What's uncertain:**

- ⚠️ Should this be shared? (Dylan's decision, not discoverable from artifacts)
- ⚠️ What form would sharing take? (orch-cli OSS? patterns docs? neither?)
- ⚠️ Is there demand beyond Dylan? (no user research, just assumption)
- ⚠️ What's the relationship to SendCutSend work? (originated there, now personal)

**What would increase confidence to Very High:**

- Dylan confirming the identity framing resonates
- Decision on sharing/audience
- Validation from other users (if shared)

---

## Design Questions for Dylan

This investigation surfaced the "what" clearly but leaves the "what next" for Dylan to decide:

### Question 1: Sharing Philosophy

**Options:**
- A) **Patterns, not tools** - Share principles/docs (like 12-factor app manifesto), keep tools personal
- B) **Tools as OSS** - Open source orch-go, kb-cli, etc. (significant maintenance burden)
- C) **Personal only** - Keep everything private, focus on personal productivity
- D) **Selective extraction** - Extract reusable patterns into orch-cli (public), keep knowledge archive private

Current evidence suggests **Option D** is already happening (orch-knowledge README describes this split).

### Question 2: Identity Framing

Is "amnesia-resilient AI orchestration framework" the right identity? Alternatives:
- "AI agent session infrastructure"
- "LLM-first development environment"
- "Personal AI orchestration system"

### Question 3: Stabilization vs Evolution

The system has been rapidly evolving (orch-go rewrite just happened). Is it time to:
- **Stabilize** - Lock patterns, document, make accessible
- **Keep evolving** - Continue discovering through practice
- **Both** - Stabilize core, evolve edges

---

## Recommendation

Based on this investigation, I recommend **producing a Decision artifact** rather than an Epic.

**Why:**
- The identity question is answered (infrastructure for AI agent sessions)
- The "what to build" is not the question - it's already built
- The open questions are strategic (share? audience? future?) not tactical

**Proposed decision structure:**
1. Accept identity framing: "Amnesia-resilient AI orchestration infrastructure"
2. Confirm current approach: Personal system, selective pattern sharing
3. Define stabilization scope: What's stable vs still evolving
4. Defer audience expansion: Not ready for broader sharing

---

## References

**Files Examined:**
- `~/.kb/principles.md` - Foundational principles
- `~/.orch/ECOSYSTEM.md` - Ecosystem documentation
- `~/.claude/CLAUDE.md` - Global knowledge placement
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Artifact architecture
- `.kb/investigations/2025-12-22-inv-design-ideal-cross-repo-setup.md` - Cross-repo patterns
- `.kb/investigations/2025-12-21-design-deep-pattern-analysis-orchestration-artifacts.md` - Artifact analysis
- `~/orch-knowledge/README.md` - Knowledge archive purpose

**Commands Run:**
```bash
# Scale measurement
bd stats  # 1195 issues, 200 open, 977 closed
ls .orch/workspace/ | wc -l  # 391 (orch-go), 788 (orch-knowledge)
wc -l .kn/entries.jsonl  # 188 entries
kb projects list  # 17 registered

# Structure exploration
ls ~/.claude/skills/
ls ~/orch-knowledge/skills/src/worker/
```

---

## Investigation History

**2025-12-24 17:50:** Investigation started
- Initial question: What is orch-ecosystem? What patterns emerged? What is its identity?
- Context: Reflection on what's been built, moving from building to understanding

**2025-12-24 18:00:** Context gathering complete
- Examined 8 repos, principles, artifact taxonomy, ECOSYSTEM.md
- Scale discovered: 1195 issues, 1179 workspaces, 265+ investigations

**2025-12-24 18:20:** Synthesis complete
- Key insight: This is infrastructure for AI agent sessions, not a productivity tool
- Identity: Amnesia-resilient AI orchestration framework
- Confidence: High (85%) - patterns clear, audience questions for Dylan

**2025-12-24 18:30:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Identity clarified; recommend Decision artifact for strategic questions
