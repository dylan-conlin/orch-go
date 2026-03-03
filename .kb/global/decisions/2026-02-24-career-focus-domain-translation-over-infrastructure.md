# Decision: Focus on Domain Translation, Not AI Infrastructure

**Date:** 2026-02-24
**Status:** accepted
**Deciders:** Dylan
**Context:** Strategic career assessment after 4+ months building orch-go and observing the orchestration landscape

## Decision

Focus career energy on **domain-specific AI application** (knowing where to point AI) rather than **AI infrastructure** (building better orchestrators, tools, or platforms). Continue using orch-go as a personal system but stop treating it as a potential product.

## Context

### The Orchestration Landscape (Feb 2026)

Three major approaches to agent orchestration are emerging, each optimizing for different constraints:

| System | Optimizes for | Primary constraint |
|--------|---------------|--------------------|
| Dylan's orch | Understanding surviving across sessions | Session Amnesia |
| Yegge's Gas Town | Throughput — keep the engine fed | Velocity / design bottleneck |
| Anthropic's Agent Teams | Coordination — peers talking to peers | Communication topology |

Meanwhile, Reddit/HN is flooded with "checkout my orchestration system" posts — most of which are thin task boards with agent spawning, solving the visible problem (run multiple agents) without addressing the hard problems that surface after weeks of sustained use.

### Who Wins the Orchestration War

**Layer 1: Basic multi-agent coordination.** Platform owners (Anthropic, OpenAI, Google) absorb this. Agent Teams already exists. The Reddit orchestrators that are just "spawn + task board" are dead on arrival — you can't beat the platform owner at basic coordination of their own agents.

**Layer 2: Workflow and work decomposition.** Less clear. Yegge's MEOW stack is the most developed attempt. A portable agent work format (the "Dockerfile for agent work") is appealing but probably wrong — agent work is too nondeterministic and context-dependent for a single format. What standardizes is more likely the communication protocol between agents than the work description itself.

**Layer 3: Knowledge infrastructure and organizational memory.** This is where orch-go lives. No platform owner is playing here yet. But it's also the most context-dependent layer — what knowledge matters, how verification works, what coherence looks like varies by team and project. Unlikely to have a single winner.

**Assessment:** "General-purpose agent orchestration" may not be a coherent category, similar to how "general-purpose project management" is technically possible but practically nobody's real answer. What survives will be opinionated systems for specific workflows, not general-purpose orchestration.

### What Orch-Go Actually Is

Orch-go is exploratory building. The principles it produced (session amnesia, verification bottleneck, accretion gravity, etc.) are real and hard-won. The system works for Dylan's specific context. But as a product competing for users, it faces structural forces that can't be matched — the platform owners will absorb basic coordination, and the deeper problems are too context-dependent for a general tool.

**Value of orch-go:** The understanding it produced, not the software itself. Those principles apply regardless of what orchestration tool is used.

### The "AI Engineer" Role

Dylan fills a common role of this era — the person who integrates AI into a business. At SendCutSend, this produced four production systems: Price Watch (competitive intelligence), Specs Platform (material data), Toolshed (internal AI ops), SendAssist (customer-facing AI search).

**The compression risk:** As models improve and orchestration becomes built-in, the mechanical part of this role (call API, build wrapper, deploy) becomes trivial. Someone without Dylan's background will be able to stand up something like SendAssist in an afternoon.

**What doesn't get compressed:** Domain translation — understanding a business deeply enough to know where AI creates real leverage versus where it's theater. Specifically:
- Knowing that Kenneth spends 40 hours/week on manual pricing collection and that this has a specific automatable shape
- Understanding that engineering is skeptical and positioning AI as "automation" leading with business value
- Recognizing that Drew's curated data should take priority over API data (a judgment call requiring business understanding)
- Building four interconnected systems as an ecosystem, not disconnected tools
- The organizational work of adoption in skeptical environments

### Where Innovation Lives for Individuals

| Layer | Who wins | Individual opportunity |
|-------|----------|----------------------|
| Infrastructure (models, orchestration, dev tools) | Big AI companies (Anthropic, OpenAI, Google) | Window closing fast |
| Application (AI applied to specific domains) | Domain experts who can build | Wide open, will stay open |
| Organizational adoption | People embedded in businesses | Gets more valuable as tech gets cheaper |

**The frontier:** Every industry has its SendCutSend — businesses with specific, messy, domain-rich problems that general-purpose AI tools don't solve. Manufacturing, logistics, healthcare, legal, agriculture, construction. The closer to physical-world operations with weird data and institutional knowledge, the further from what big companies commoditize.

### Market Context

SaaS is showing stress as orchestration begins to roll out. If agents can orchestrate workflows directly, the "glue" SaaS layer gets compressed. But this cuts both ways — companies being disrupted need people who understand their domain AND can build AI-native replacements. That's not a general-purpose AI engineer; that's someone embedded in the business.

## Consequences

### What This Means in Practice

1. **Orch-go continues as personal tooling.** Use it, evolve it, switch when something better appears. Stop treating it as something others should adopt.

2. **SCS work is the more valuable investment.** Four production systems serving real users with measurable impact. Domain knowledge of manufacturing, competitive intelligence, material specifications — this is the hard-to-replicate part.

3. **Career moat is domain translation, not technical capability.** Technical skills are necessary but insufficient. The differentiator is knowing where AI creates real leverage in specific business contexts.

4. **Watch for the adoption layer.** The hardest problem in AI right now isn't technical — it's getting real organizations to actually use it. Navigating skeptical engineering teams, positioning correctly, building trust through delivered value. This skill gets more valuable as technology gets cheaper.

### What to Monitor

- When does orchestration become "good enough" built-in that orch-go provides no advantage? (Switch point)
- How fast does the "AI engineer" role commoditize at the mechanical level?
- Which domains are furthest from platform commoditization? (Where to deepen)
- Does the SaaS compression create new opportunities for domain-embedded builders?

## Provenance

This decision emerged from a conversation analyzing three orchestration approaches (orch-go, Gas Town, Anthropic Agent Teams), the Reddit/HN orchestration proliferation, SaaS market signals, and reflection on what the SCS portfolio actually demonstrates about where value lives.
