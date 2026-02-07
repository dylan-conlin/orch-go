<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode's native orchestration is minimal (Task tool + subagents), while third-party plugins (Oh My OpenCode, Open Orchestra) provide sophisticated multi-agent coordination - but orch provides unique value through beads integration, kb context injection, daemon automation, and triple spawn modes that neither OpenCode native nor plugins offer.

**Evidence:** OpenCode docs show Task tool spawns subagents but can't recurse and has REST API bugs; oh-my-opencode has 10 agents + 31 hooks but no issue tracking; orch has skill system, beads tracking, kb context, daemon, 3 backends, rate limit management.

**Knowledge:** The "orchestration" term means different things: OpenCode native = in-session subagent spawning; plugins = multi-model delegation; orch = full lifecycle management across sessions/projects. They're complementary layers, not replacements.

**Next:** No migration needed. Continue using orch for lifecycle orchestration. Monitor OpenCode plugin ecosystem for features worth adopting (hooks, multi-model routing). Consider extracting orch patterns as OpenCode plugin if community value is high.

**Promote to Decision:** recommend-no - This validates existing architecture rather than establishing new patterns.

---

# Investigation: OpenCode Orchestration Infrastructure Comparison

**Question:** What orchestration features does OpenCode now have, and how do they compare to orch? Should we adopt, ignore, or selectively integrate?

**Started:** 2026-01-23
**Updated:** 2026-01-23
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: OpenCode Native Orchestration is Minimal - Task Tool + Subagents

**Evidence:**
OpenCode's built-in orchestration consists of:

| Feature | Description |
|---------|-------------|
| **Primary agents** | Build (full access), Plan (read-only) - switch with Tab |
| **Subagents** | General (full access), Explore (read-only) |
| **Task tool** | Spawns subagents in separate sessions/context windows |
| **@ mentions** | Manual subagent invocation |
| **Permissions** | Glob patterns control which subagents can spawn (`permission.task`) |
| **Model override** | Different model per agent |
| **Hidden agents** | `hidden: true` for internal-only agents |

**Limitations identified:**
- Subagents cannot spawn subagents (recursion prevention flag in code)
- REST API has bug: sessions hang indefinitely when Task tool spawns subagents (works in TUI only)
- No cross-session lifecycle management
- No external issue tracking integration
- No daemon/background processing

**Source:**
- https://opencode.ai/docs/agents/
- https://github.com/anomalyco/opencode/issues/6573
- https://github.com/anomalyco/opencode/issues/9280

**Significance:** OpenCode's native orchestration is **intra-session** subagent spawning, not **cross-session** lifecycle management. It's a building block, not a complete orchestration system.

---

### Finding 2: Third-Party Plugins Provide Sophisticated Multi-Agent Coordination

**Evidence:**
Two major orchestration plugins exist:

**Oh My OpenCode (OMO):**
- 10 specialized agents across multiple models (Claude Opus, GPT-5.2, GLM-4.7, Gemini-3, Grok)
- Sisyphus/Atlas as primary orchestrators
- 31 lifecycle hooks
- 20+ tools (LSP, AST-Grep, delegation)
- `delegate_task` tool (1,038 lines) for parallel agent calls
- Background Agent Manager (1,335 lines) for task lifecycle
- Temperature enforcement (0.1-0.3 for code agents)

**Open Orchestra:**
- Hub-and-spoke architecture
- Central orchestrator + specialized workers
- Profile-based spawning with auto-model resolution
- Dynamic port allocation
- Session-based isolation

**Source:**
- https://github.com/code-yeongyu/oh-my-opencode/blob/dev/AGENTS.md
- Web search results on OpenCode orchestration plugins

**Significance:** Community has built sophisticated orchestration ON TOP of OpenCode. This validates the demand but also shows OpenCode's native features are insufficient for real multi-agent workflows.

---

### Finding 3: orch Provides Unique Value Not Found in OpenCode or Plugins

**Evidence:**

| Feature | orch | OpenCode Native | Oh My OpenCode |
|---------|------|-----------------|----------------|
| **Issue tracking integration** | beads (bd CLI) | ❌ | ❌ |
| **Knowledge context injection** | kb context | ❌ | ❌ |
| **Skill system** | ~/.claude/skills/ | Agent config files | 10 specialized agents |
| **Daemon automation** | orch daemon run | ❌ | Background Agent Manager |
| **Cross-project support** | --workdir, cross-project daemon | ❌ | ❌ |
| **Triple spawn modes** | Claude CLI, OpenCode API, Docker | Single mode | Single mode |
| **Rate limit management** | Auto-switch, block at 95% | ❌ | ❌ |
| **Workspace artifacts** | SPAWN_CONTEXT.md, SYNTHESIS.md | ❌ | ❌ |
| **Completion verification** | orch complete | Session end | Task completion |
| **Duplicate prevention** | Active agent detection | ❌ | ❌ |

**Unique orch capabilities not replicated elsewhere:**
1. **beads integration** - Issues tracked externally, survive session crashes, enable triage workflow
2. **kb context** - Prior decisions, constraints, investigations injected into spawn context
3. **Triple spawn modes** - Redundancy for infrastructure work (Claude CLI survives OpenCode crashes)
4. **Tiered spawns** - Light/full with SYNTHESIS.md requirements
5. **Cross-project daemon** - Polls multiple project directories

**Source:**
- `.kb/guides/spawn.md`
- `.kb/guides/daemon.md`
- `.kb/guides/opencode.md`

**Significance:** orch solves problems that neither OpenCode native nor plugins address: external issue tracking, knowledge injection, session crash resilience, and cross-project coordination. These are **lifecycle** concerns, not **agent** concerns.

---

### Finding 4: Recent OpenCode Updates Focus on Integration, Not Orchestration

**Evidence:**
Recent changelog entries (2026):
- v1.1.32 (Jan 22): "Mark subagent sessions as agent-initiated to exclude from quota limits"
- v1.1.31 (Jan 22): "Add chat.headers hook and update codex and copilot plugins"
- Jan 16: GitHub Copilot integration (authenticate via Copilot credentials)

**What's NOT in recent updates:**
- No daemon/background processing
- No external issue tracking
- No cross-session lifecycle management
- No knowledge base integration

**Source:** https://opencode.ai/changelog

**Significance:** OpenCode is focusing on **integrations** (Copilot, codex plugins) rather than **orchestration infrastructure**. The subagent quota handling (v1.1.32) is incremental improvement, not strategic orchestration.

---

### Finding 5: OpenCode's Task Tool Has Critical Bugs for API-Driven Orchestration

**Evidence:**
GitHub issue #6573 documents:
- Sessions hang indefinitely when Task tool spawns subagents via REST API
- Works in TUI but fails via `opencode serve`
- Root cause: Directory context mismatch in event subscription
- Subagent messages fetched with parent session's directory
- Both sessions stuck in "busy" state with no response

**Impact on orch:**
- orch uses OpenCode HTTP API for headless spawns
- If orch tried to use Task tool via API, it would hit this bug
- Current orch approach (spawn separate sessions via API) works because it doesn't use Task tool

**Source:** https://github.com/anomalyco/opencode/issues/6573

**Significance:** OpenCode's API-based orchestration is **not production-ready** for Task tool usage. orch's approach of managing sessions externally (not via Task tool) is architecturally safer.

---

## Synthesis

**Key Insights:**

1. **"Orchestration" Means Different Things at Different Layers** - OpenCode native = in-session subagent spawning (Task tool). Plugins = multi-model agent delegation. orch = cross-session lifecycle management with external tracking. These are complementary layers, not competing solutions.

2. **orch Solves Lifecycle Problems, Not Agent Problems** - The unique value of orch isn't spawning agents (OpenCode does that) - it's tracking them across sessions, injecting context, managing rate limits, and verifying completion. No OpenCode native feature or plugin provides this.

3. **OpenCode's REST API Orchestration Has Critical Bugs** - The Task tool hanging issue (#6573) means API-driven subagent spawning is unreliable. orch's approach of managing sessions externally avoids this entirely.

4. **Community Plugins Show Demand But Different Focus** - oh-my-opencode (10 agents, 31 hooks) focuses on multi-model routing and in-session coordination. Open Orchestra focuses on worker specialization. Neither addresses beads-style issue tracking or kb-style knowledge injection.

5. **No Migration Path Needed - They're Different Tools** - OpenCode orchestration is a building block orch could use, not a replacement for orch. The comparison is like asking if a car engine replaces the steering wheel.

**Answer to Investigation Question:**

**What orchestration features does OpenCode have?**
- Task tool for spawning subagents
- Primary/subagent distinction with model overrides
- Permission controls via glob patterns
- Third-party plugins (oh-my-opencode, Open Orchestra) for sophisticated multi-agent

**How does it compare to orch?**
- OpenCode: in-session agent spawning
- orch: cross-session lifecycle management + external tracking + knowledge injection
- They're complementary layers, not competitors

**Integration implications:**
- **Adopt:** Nothing needed - orch already uses OpenCode as execution layer
- **Monitor:** oh-my-opencode hooks system could inform future orch plugin architecture
- **Ignore:** OpenCode native orchestration (Task tool) - too limited, has API bugs

**Migration path:** Not applicable - different problem domains.

---

## Feature Comparison Matrix

| Category | Feature | orch | OpenCode Native | oh-my-opencode | Open Orchestra |
|----------|---------|------|-----------------|----------------|----------------|
| **Spawning** | Agent creation | ✅ orch spawn | ✅ Task tool | ✅ delegate_task | ✅ Profile-based |
| | Multi-model support | ✅ --model flag | ✅ Per-agent config | ✅ 10 agents/5 models | ✅ Auto-resolution |
| | Cross-project | ✅ --workdir | ❌ | ❌ | ❌ |
| | Daemon automation | ✅ orch daemon | ❌ | ✅ Background Manager | ❌ |
| | Triple backend modes | ✅ Claude/OpenCode/Docker | ❌ | ❌ | ❌ |
| **Tracking** | External issue tracking | ✅ beads | ❌ | ❌ | ❌ |
| | Session persistence | ✅ Workspace files | ❌ | ❌ | ✅ Session isolation |
| | Phase reporting | ✅ bd comment | ❌ | ❌ | ❌ |
| | Completion verification | ✅ orch complete | ❌ | ✅ Task lifecycle | ❌ |
| **Context** | Knowledge injection | ✅ kb context | ❌ | ❌ | ❌ |
| | Skill system | ✅ SKILL.md files | ✅ Agent configs | ✅ 10 specialized | ✅ Worker profiles |
| | SPAWN_CONTEXT generation | ✅ | ❌ | ❌ | ❌ |
| | SYNTHESIS.md requirement | ✅ (full tier) | ❌ | ❌ | ❌ |
| **Safety** | Rate limit monitoring | ✅ Auto-switch at 95% | ❌ | ❌ | ❌ |
| | Duplicate prevention | ✅ Active agent check | ❌ | ❌ | ❌ |
| | Concurrency limits | ✅ --max-agents | ❌ | ✅ Concurrency control | ✅ Port allocation |
| **Resilience** | Crash recovery | ✅ Triple spawn modes | ❌ | ❌ | ❌ |
| | Escape hatches | ✅ Docker fingerprint | ❌ | ❌ | ❌ |
| **Hooks** | Lifecycle hooks | ❌ (uses OpenCode) | ✅ Plugin hooks | ✅ 31 hooks | ❌ |
| | Temperature control | ❌ | ✅ Per-agent | ✅ Enforced 0.1-0.3 | ❌ |

---

## Structured Uncertainty

**What's tested:**

- ✅ OpenCode Task tool exists and spawns subagents (verified: official docs)
- ✅ Task tool has REST API bug causing sessions to hang (verified: GitHub issue #6573)
- ✅ oh-my-opencode has 10 agents and 31 hooks (verified: AGENTS.md)
- ✅ orch has beads integration, kb context, triple spawn modes (verified: spawn.md guide)
- ✅ OpenCode recent changelog focuses on integrations, not orchestration (verified: changelog)

**What's untested:**

- ⚠️ Whether oh-my-opencode hooks could be adopted by orch (architectural compatibility unknown)
- ⚠️ Whether OpenCode will fix Task tool API bug (active issue, no ETA)
- ⚠️ Whether Open Orchestra's session isolation would work with orch's cross-project needs
- ⚠️ Whether OpenCode will add daemon/background processing natively

**What would change this:**

- If OpenCode adds native daemon automation → orch daemon might become redundant
- If OpenCode adds external issue tracking → beads integration might be replaceable
- If Task tool API bug is fixed → could use Task tool for in-session delegation
- If oh-my-opencode adds beads-style tracking → would become direct competitor

---

## Implementation Recommendations

### Recommended Approach ⭐

**Status Quo: Continue using orch as lifecycle layer, OpenCode as execution layer**

**Why this approach:**
- orch solves lifecycle problems OpenCode doesn't address (Finding 3)
- OpenCode's API orchestration has critical bugs (Finding 5)
- No feature overlap requiring consolidation (Finding 1, 3)
- Third-party plugins focus on different problems (Finding 2)

**Trade-offs accepted:**
- Maintaining custom infrastructure vs. using community plugins
- Not benefiting from oh-my-opencode's 31 hooks system
- Potential future work if OpenCode adds competing features

**Implementation sequence:**
1. **No changes needed** - current architecture is validated
2. **Monitor oh-my-opencode** - hooks system could inform future plugin architecture
3. **Track OpenCode changelog** - watch for daemon/background processing additions

### Alternative Approaches Considered

**Option B: Replace orch with oh-my-opencode**
- **Pros:** 31 hooks, multi-model routing, community maintained
- **Cons:** No beads tracking, no kb context injection, no triple spawn modes, no cross-project daemon
- **When to use:** Never - different problem domain (agent delegation vs lifecycle management)

**Option C: Contribute orch patterns to OpenCode core**
- **Pros:** Community benefit, reduced maintenance burden, potential standardization
- **Cons:** Would need significant refactoring for plugin architecture, beads/kb dependencies would need abstraction
- **When to use:** If orch patterns prove broadly useful and Dylan wants to share

**Option D: Hybrid - adopt oh-my-opencode hooks for orch**
- **Pros:** Would get 31 lifecycle hooks for event handling
- **Cons:** Different hook model, integration complexity, may not fit orch's architecture
- **When to use:** If orch needs more sophisticated in-session event handling

---

## References

**Files Examined:**
- `.kb/guides/spawn.md` - orch spawn implementation details
- `.kb/guides/opencode.md` - OpenCode integration architecture
- `.kb/investigations/2026-01-17-inv-identify-orchestrator-value-add-vs.md` - Prior analysis of orch value

**External Documentation:**
- [OpenCode Agents docs](https://opencode.ai/docs/agents/) - Native agent features
- [OpenCode Changelog](https://opencode.ai/changelog) - Recent updates
- [GitHub #6573](https://github.com/anomalyco/opencode/issues/6573) - Task tool API bug
- [GitHub #9280](https://github.com/anomalyco/opencode/issues/9280) - Subagent recursion request
- [oh-my-opencode AGENTS.md](https://github.com/code-yeongyu/oh-my-opencode/blob/dev/AGENTS.md) - Plugin agent architecture

**Related Artifacts:**
- `.kb/investigations/2026-01-13-research-opencode-zen-black-architecture-economics.md` - OpenCode business model research
- `.kb/guides/daemon.md` - orch daemon documentation

---

## Investigation History

**[2026-01-23 10:00]:** Investigation started
- Initial question: What orchestration features does OpenCode have and how do they compare to orch?
- Context: Reports of OpenCode releasing orchestration infrastructure in recent updates

**[2026-01-23 10:15]:** Found OpenCode native orchestration is minimal
- Task tool + subagents only
- No daemon, no external tracking, no cross-session lifecycle

**[2026-01-23 10:25]:** Found third-party plugins (oh-my-opencode, Open Orchestra)
- Sophisticated multi-agent coordination
- But different focus: agent delegation vs lifecycle management

**[2026-01-23 10:35]:** Identified unique orch value
- beads integration, kb context, triple spawn modes, daemon
- Not replicated in any OpenCode native or plugin feature

**[2026-01-23 10:45]:** Found critical Task tool API bug
- REST API sessions hang when spawning subagents
- orch's external session management approach is safer

**[2026-01-23 11:00]:** Investigation completed
- Status: Complete
- Key outcome: orch and OpenCode orchestration are complementary layers, not competitors
- Recommendation: Status quo - continue using orch for lifecycle, OpenCode for execution
