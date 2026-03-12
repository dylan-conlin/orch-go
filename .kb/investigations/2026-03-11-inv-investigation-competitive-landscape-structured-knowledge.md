<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** No existing tool in the 2026 landscape treats knowledge as governance (enforced lifecycle with structural gates). All competitors treat it as storage (retrieve facts by similarity). The genuine unique advantage is "memory as governance" — the investigation/probe/model cycle with spawn gates, accretion boundaries, and daemon escalation.

**Evidence:** Surveyed 40+ tools across 8 categories (Cursor Rules, CLAUDE.md, mem0/Letta/Zep, Obsidian+AI, ADR tools, developer wikis, agent orchestration frameworks, context management). None implement re-investigation prevention, knowledge lifecycle enforcement, or cross-session structural coordination. Closest competitor is Acontext (learned skills from agent executions) but it filters to successes only and has no enforcement gates.

**Knowledge:** The market has three unsolved problems: (1) re-investigation prevention (agents redoing solved work), (2) failure knowledge capture (what didn't work and why), (3) knowledge-as-governance (decisions constraining future agent behavior mechanically, not advisorily). The ETH Zurich paper finding that AGENTS.md files can *reduce* task success validates that unstructured context injection is insufficient.

**Next:** Position kb-cli around "knowledge governance for AI agents" — the only tool that makes wrong paths mechanically harder. Immediate opportunity: publish the investigation/probe/model pattern as a framework-agnostic protocol that users of CrewAI, LangGraph, Claude Code, etc. can adopt. Strategic decision needed on target market (solo AI-heavy developers vs teams).

**Authority:** strategic - Market positioning and product direction are irreversible value judgments requiring Dylan's decision.

---

# Investigation: Competitive Landscape — Structured Knowledge for AI-Assisted Development (2026)

**TLDR:** No tool in the 2026 landscape combines accumulated knowledge, structural enforcement, and AI agent awareness. The market treats knowledge as retrieval; orch-go's system treats it as governance. The genuine positioning gap is "memory as governance" — making wrong paths mechanically harder rather than advisorily discouraged.

**Question:** What tools exist in the 'structured knowledge management for AI-assisted development' space in 2026? Where does kb-cli have a genuine unique advantage that isn't just 'it's different'?

**Started:** 2026-03-11
**Updated:** 2026-03-11
**Owner:** Investigation agent
**Phase:** Complete
**Next Step:** None — strategic positioning decision needed from Dylan
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A - novel investigation | - | - | - |

---

## Findings

### Finding 1: Cursor Rules and CLAUDE.md are instructions, not knowledge

**Evidence:** Cursor Rules (.cursor/rules/*.mdc) and CLAUDE.md are static text injected into context windows. They tell agents what to do but cannot capture what was learned, why decisions were made, or how past investigations resolved. Cursor's Memories feature (auto-extracted facts from conversations) was an attempt to bridge this gap but remains unstable as of March 2026 — documentation removed, UI bugs, feature status unclear.

Claude Code has a richer layered memory architecture (CLAUDE.md → .claude/rules/ → Auto Memory → Session Memory → Skills) but Auto Memory is limited to 200 lines in MEMORY.md and tends toward shallow notes. Session Memory injects past session summaries but they are background, not structured investigation state.

Key stat: Cursor has 1M+ daily active users, 360K paying customers, $1B ARR. Claude Code reached ~$2.5B run-rate with 29M daily installs. The massive user bases mean any genuine gap affects millions of developers.

**Source:** Web research across Cursor docs, Claude Code docs, community forums, awesome-cursorrules (38.4K GitHub stars), multiple blog posts and guides.

**Significance:** These tools dominate the market but leave a fundamental gap: they provide *instructions* (static, advisory) but not *knowledge* (accumulated, enforced, lifecycle-managed). Every session starts with the same static context. The ETH Zurich paper (Feb 2026) found that AGENTS.md files can actually *reduce* task success rates by 20%+, validating that unstructured context injection is insufficient.

---

### Finding 2: AI memory layers (mem0, Letta, Zep) solve storage, not governance

**Evidence:** Surveyed 6 AI memory tools:

| Tool | Approach | Adoption | Key Limitation |
|------|----------|----------|----------------|
| **mem0** | Automatic fact extraction → vector store. Hybrid semantic + optional graph memory. | YC-backed, $24M Series A, 50K+ devs | Flat facts, no enforcement, no lifecycle |
| **Letta (MemGPT)** | Agent self-manages 3-tier memory (core/recall/archival). LLM-as-OS paradigm. | Open source, strong community, NeurIPS paper | Depends on agent judgment, no structural guarantees |
| **Zep** | Temporal knowledge graph via Graphiti. Tracks fact evolution over time. | YC-backed, SaaS "far from polished" | Complex, enterprise-focused, still conversational memory |
| **Cognee** | Document → knowledge graph via ECL pipeline. | $7.5M seed, 70+ companies, 1M+ pipeline runs | Knowledge graph builder, not full agent memory |
| **LangMem** | Memory primitives for LangGraph agents. | Part of massive LangChain ecosystem | Ecosystem lock-in, no structure |
| **Claude-Mem** | Session capture → compression → local SQLite for Claude Code. | Community plugin, early growth | Single-agent, no enforcement |

**Critical distinction:** All these tools store *facts* and retrieve by *similarity*. None enforce investigation lifecycles, prevent re-investigation, or gate future work on existing knowledge. An O'Reilly article on multi-agent memory engineering found 36.9% of multi-agent failures stem from inter-agent misalignment, and multi-agent systems consume ~15x more tokens than single agents due to redundant work.

**Source:** mem0.ai, letta.com, getzep.com, cognee.ai, langchain docs, claude-mem docs, O'Reilly, ML Mastery, IBM, DEV Community.

**Significance:** The memory layer market is well-funded and growing but solves a different problem. These tools answer "what do we know similar to this query?" — they cannot answer "has this been investigated before and what was the conclusion?" Mem0's $24M Series A validates market demand for persistent AI memory, but the product category stops at retrieval.

---

### Finding 3: Agent orchestration frameworks have no knowledge governance

**Evidence:** Surveyed 6 major frameworks:

| Framework | Knowledge Mechanism | Re-investigation Prevention | Stars |
|-----------|--------------------|-----------------------------|-------|
| **CrewAI** | ChromaDB + SQLite3, Knowledge sources (RAG) | None | ~20K |
| **AutoGen** | External only (Mem0, ChromaDB, Zep) | None | ~50K (maintenance mode) |
| **LangGraph** | Checkpoints + Store (semantic/episodic/procedural) | None | ~25K |
| **Semantic Kernel** | Mem0Provider + WhiteboardProvider (experimental) | None | ~27K |
| **Claude Agent SDK** | File-based memory tool (/memories directory) | Prompt-level only | Growing |
| **OpenAI Agents SDK** | Sessions with multiple backends | None | Growing |

None implement: investigation lifecycle enforcement, probe/model cycles, accretion boundaries, or cross-session knowledge coordination with enforcement.

Semantic Kernel's WhiteboardProvider is the closest — it extracts "requirements, proposals, decisions, actions" from conversation — but these categories are not enforced and have no lifecycle.

Claude Agent SDK's memory tool instruction ("ASSUME INTERRUPTION: Your context window might be reset at any moment") is the closest to enforcement, but it is a prompt instruction, not a structural guarantee.

**Source:** Official docs for CrewAI, AutoGen, LangGraph, Semantic Kernel, Claude Agent SDK, OpenAI Agents SDK. Framework comparison articles from DEV Community, Shakudo, OpenAgents.

**Significance:** Every major framework treats memory as a storage problem. None treat it as a governance problem. The pattern of investigations → probes → models → guides → enforcement does not exist in any mainstream agent orchestration framework.

---

### Finding 4: ADR tools solve a narrow subset; no tool handles knowledge evolution

**Evidence:** Surveyed 10+ ADR tools. The space is mature but narrow:

- **adr-tools** (npryce): ~5K stars, the original. Pure file-creation utility. Mature but minimal development.
- **Log4brains**: ~1.4K stars, ADRs + static site. Maintenance status questionable (issue #29).
- **MADR**: Template standard, not a tool. Most widely adopted format.
- **Backstage ADR Plugin**: Production-ready within Backstage ecosystem. Read-only view.

Emerging: **Agent Decision Records (AgDR)** — a new standard specifically for AI agent decisions. Installable as Claude Code plugin, Cursor rules. First tool to explicitly target AI agent decision tracing.

Academic: **AgenticAKM** (Feb 2026 paper) validates that architectural knowledge management requires multiple specialized agents, not single-prompt LLMs.

No ADR tool handles: investigations (exploratory work preceding decisions), accumulated learning/models (patterns across decisions), guides (operational knowledge derived from decisions), cross-project knowledge, or knowledge lifecycle evolution.

**Source:** ADR GitHub organization, adr-tools, log4brains, MADR, Backstage, AgDR, AgenticAKM paper (arXiv).

**Significance:** ADR tools validate demand for structured decision records but represent only one artifact type in the knowledge lifecycle. The gap between "document a decision" and "accumulate and operationalize engineering knowledge" is exactly what the broader .kb/ system fills.

---

### Finding 5: Developer knowledge management tools are web-first and AI-unaware

**Evidence:** Surveyed team knowledge tools:

- **Confluence**: Enterprise wiki, Jira-integrated. Rovo AI added 20+ documentation agents. No CLI, no git-native, web-only.
- **Notion**: All-in-one workspace. AI bundled at Business tier ($20/user/mo). No CLI, no git-native.
- **GitBook**: Git-native documentation. $65/site. External-facing docs focus.
- **Mintlify**: Docs-as-code. Used by Anthropic. 5,000+ companies, 20M+ devs. External docs focus.
- **Backstage TechDocs**: Docs-as-code in developer portal. Heavy infrastructure.
- **Swimm**: Code-coupled docs that auto-update when code changes. Narrow scope.
- **Guru**: Card-based with **verification workflows** (prompts experts to re-verify at intervals). Addresses knowledge decay but is enterprise-focused, not developer-specific.

**Key finding:** CLI-first internal knowledge management does not exist as a product category. Every team wiki is web-first. The developer-specific tools (GitBook, Mintlify) are git-native but focused on external documentation.

**Critical research finding:** A February 2026 "Codified Context" paper proposes a three-tier architecture — hot memory (constitution, always loaded), domain specialists (agents, invoked per task), cold memory (knowledge base, retrieved on demand) — that maps directly to what orch-go already implements (CLAUDE.md → skills → .kb/) but has no product implementation.

**Source:** Docsie, SaaS CRM Review, FernDesk, Mintlify, Backstage, AGENTS.md standard, ETH Zurich paper (arXiv), Codified Context paper (arXiv), Martin Fowler (context engineering article), Augment Code blog.

**Significance:** The market has no CLI-first, git-native, AI-agent-aware internal knowledge management tool. This is an empty product category.

---

### Finding 6: Obsidian + AI is powerful for humans, not for agents

**Evidence:** Obsidian has 1.5M+ active users, 1,000+ plugins, and strong AI integration:
- **Smart Connections**: 786K+ downloads, local semantic embeddings, zero-config. Dominant AI plugin.
- **Copilot for Obsidian**: RAG across vault, supports local LLMs.
- **MCP Bridge** (2026): Multiple MCP servers expose vaults to Claude Code and other agents.
- **Obsidian CLI** (v1.12, 2026): Official CLI (`obsidian create`, `obsidian search`). Significant shift.
- **Headless Sync** (Feb 2026): Server-side vault access without GUI.

But: No daemon/loop concept, no structured issue tracking, no spawn gates, no accretion boundaries, no event logging, no agent lifecycle tracking. MCP integration bridges the gap but agent logic must live outside Obsidian.

**Source:** Smart Connections GitHub (786K downloads), Obsidian official docs (CLI, headless sync), obsidian-claude-code-mcp, Obsidian usage statistics (1.5M users, $2M ARR).

**Significance:** Obsidian optimizes for human knowledge exploration with visual graph views. CLI-first optimizes for machine-readable knowledge that feeds agent workflows. They are complementary, not competitive. Obsidian cannot enforce investigation cycles or gate agent work.

---

### Finding 7: Context management tools solve retrieval, not accumulation

**Evidence:** Surveyed 12+ context management tools:

| Tool | Approach | What It Can't Do |
|------|----------|-----------------|
| **Context7** | MCP server for library docs | Only third-party docs, not YOUR knowledge |
| **Repomix** | Repo → single file for LLM | Brute-force packing, no memory, no learning |
| **Aider repo map** | Tree-sitter AST → PageRank file relevance | No persistent memory, session-scoped only |
| **Cursor workspace index** | Semantic codebase index + dynamic context discovery | No persistent knowledge, session reset |
| **Augment Context Engine** | Semantic dependency graph, 400K+ files, cross-repo | Code topology, not "why we made this decision" |
| **Sourcegraph Cody** | Enterprise code search via RAG | Structural search, not accumulated knowledge |
| **Greptile** | Code review with multi-hop graph investigation | Learns from PR feedback, but review-scoped |
| **Windsurf Memories** | Auto-generated workspace-scoped memories | Not shareable, not validated, opaque |
| **Copilot Spaces** | Curated context collections, shareable | Manually curated, static, no learning |
| **Acontext** | Agent execution → reusable skills (Markdown) | Only validated successes, no failure knowledge |
| **MemU** | Persistent memory per subagent domain | Early-stage, unclear structural enforcement |

**Acontext** (memodb-io) is the closest competitor to the accumulated knowledge approach. It transforms agent executions into reusable skills with goal, reasoning chain, operational steps, context conditions, and evidence of correctness. Skills are stored as Markdown files. But it filters to validated successes only — discarding failure knowledge — and has no enforcement gates.

**Source:** Context7 docs, Repomix GitHub, Aider repo map blog, Cursor docs, Augment Context Engine, Cody docs, Greptile docs, Windsurf docs, Copilot docs, Acontext GitHub, MemU blog.

**Significance:** The entire context management category operates on "give the LLM better input." None operate on "prevent the LLM from doing redundant work" or "enforce that past knowledge constrains future behavior."

---

### Finding 8: The positioning gap — memory as governance

**Evidence:** Across all 40+ tools surveyed, the following capabilities exist NOWHERE:

1. **Re-investigation prevention**: Named artifacts + glob lookup that deterministically prevent duplicate work. Every tool relies on semantic similarity (probabilistic) or agent judgment (unreliable).

2. **Knowledge lifecycle enforcement**: No tool distinguishes "preliminary finding" from "validated knowledge" with structural gates. Investigations → findings → probes → models → guides → decisions is a lifecycle no product implements.

3. **Failure knowledge capture**: Acontext explicitly filters to successes. Mem0/Letta/Zep store facts with no quality tier. No tool systematically captures "what we tried that didn't work and why" as first-class knowledge.

4. **Decision enforcement**: A decision in .kb/decisions/ can be referenced by spawn gates, daemon routing, and governance checks. No other system connects past decisions to mechanical enforcement on future agent behavior.

5. **Accretion boundaries**: No tool monitors knowledge artifact growth and enforces extraction when files become unwieldy. The 85.5% orphan rate finding and >800-line hotspot gates are unique.

6. **Cross-project structural knowledge**: Most tools are project-scoped. The .kb/global/ symlink pattern for shared knowledge across independent projects exists nowhere else.

**Source:** Synthesis across all 8 research categories. Verified by checking official documentation for every major tool.

**Significance:** This is not "it's different" — it is a genuinely unoccupied market position. The key insight is the distinction between *advisory* knowledge (CLAUDE.md: "please don't do X") and *enforced* knowledge (spawn gates: "you cannot do X without architect review"). Advisory knowledge has an 85.5% orphan rate. Enforced knowledge has structural integrity.

---

## Synthesis

**Key Insights:**

1. **"Memory as retrieval" vs "memory as governance" is the fundamental split.** Every tool in the landscape — from Cursor Rules to mem0 to LangGraph — treats knowledge as something to retrieve. None treat it as something that constrains. The orch-go system is unique in making wrong paths mechanically harder (spawn gates, accretion boundaries, daemon escalation). This is not a feature difference; it is a category difference.

2. **The market validates demand but leaves the governance gap.** Mem0's $24M Series A, Cursor's $1B ARR, Claude Code's $2.5B run-rate — the market clearly values AI-assisted development tools. But the ETH Zurich finding that AGENTS.md files can reduce performance, plus the O'Reilly finding that 36.9% of multi-agent failures are from inter-agent misalignment, prove that "more context" is not the answer. Structured, enforced, lifecycle-managed knowledge is.

3. **The "Codified Context" three-tier architecture is the academic validation.** A February 2026 paper proposes: hot memory (always loaded), domain specialists (invoked per task), cold memory (retrieved on demand). This maps directly to CLAUDE.md → skills → .kb/. The academic community has identified the architecture orch-go already implements, but no product delivers it.

**Answer to Investigation Question:**

The competitive landscape in 2026 has ~40+ tools across 8 categories, none of which combine: accumulated knowledge, structural lifecycle enforcement, AI agent awareness, and cross-project knowledge federation. The genuine unique advantage is **knowledge governance** — the property that past investigations, decisions, and models mechanically constrain future agent behavior rather than merely being available for retrieval.

The positioning is not "another knowledge tool" but "the enforcement layer that makes existing AI coding tools (Cursor, Claude Code, CrewAI, LangGraph) produce coherent work across sessions."

---

## Structured Uncertainty

**What's tested:**

- ✅ Surveyed 40+ tools across 8 categories via web research (verified: multiple sources per tool, official documentation checked)
- ✅ No tool implements re-investigation prevention with structural enforcement (verified: checked docs for all major agent orchestration frameworks)
- ✅ ETH Zurich finding that AGENTS.md can reduce performance is published research (verified: arxiv paper 2602.11988)
- ✅ Acontext is the closest competitor for accumulated knowledge (verified: checked GitHub repo and docs)
- ✅ No CLI-first, git-native, AI-agent-aware internal knowledge management product category exists (verified: exhaustive search across developer tooling landscape)

**What's untested:**

- ⚠️ Whether the "memory as governance" positioning resonates with the target market (not user-tested)
- ⚠️ Whether solo developers would adopt a structured knowledge system with enforcement overhead (adoption risk)
- ⚠️ Whether the investigation/probe/model cycle can be simplified enough for users who aren't running multi-agent orchestration (complexity risk)
- ⚠️ Whether Cursor/Claude Code will build native knowledge governance features that close the gap (platform risk)
- ⚠️ Whether Acontext's skills-from-execution approach could evolve to include enforcement (competitive risk)

**What would change this:**

- If Cursor ships persistent cross-session knowledge with enforcement mechanisms, the positioning gap narrows significantly
- If mem0 or Letta adds lifecycle management and structural gates, they become direct competitors
- If the "Codified Context" paper spawns an open-source implementation, it validates the architecture but creates competition
- If user research shows solo developers reject enforcement overhead, the target market needs revision

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Position kb-cli as "knowledge governance for AI agents" | strategic | Market positioning is irreversible value judgment |
| Publish investigation/probe/model as framework-agnostic protocol | strategic | Community strategy requires resource commitment |
| Build Acontext-style auto-learning alongside enforcement | architectural | Cross-component integration, multiple valid approaches |
| Add MCP server for kb context | implementation | Technical integration within existing patterns |

### Recommended Approach ⭐

**"Knowledge Governance Protocol"** — Position the investigation/probe/model cycle as a framework-agnostic protocol that any AI coding workflow can adopt, with kb-cli as the reference implementation.

**Why this approach:**
- No existing tool occupies the "knowledge governance" position — first-mover advantage
- The protocol can be adopted by users of Cursor, Claude Code, CrewAI, LangGraph without switching tools
- Academic validation exists (Codified Context paper, AgenticAKM paper) but no product implementation
- The 85.5% orphan rate statistic is a compelling proof point

**Trade-offs accepted:**
- Requires Dylan to commit to a public-facing effort (documentation, community)
- Protocol standardization takes longer than just shipping features
- Risk of being ignored if the market doesn't care about governance (just wants "better memory")

**Implementation sequence:**
1. Extract the investigation/probe/model cycle into a standalone specification (not orch-go-specific)
2. Build an MCP server for `kb context` so Cursor/Claude Code users can query the knowledge base
3. Publish the 85.5% orphan rate finding and harness engineering concept as a blog post or talk
4. Build Acontext-style auto-learning (agent execution → knowledge artifacts) but with enforcement gates

### Alternative Approaches Considered

**Option B: Pure product play — ship kb-cli as a standalone CLI tool**
- **Pros:** Faster to market, concrete product, can charge for it
- **Cons:** Small addressable market (solo AI-heavy developers), no network effects, competes with free tools (CLAUDE.md, .cursorrules)
- **When to use instead:** If Dylan wants revenue sooner rather than community influence

**Option C: Integration play — build kb as a plugin/MCP for Cursor and Claude Code**
- **Pros:** Rides existing distribution (Cursor 1M DAU, Claude Code 29M installs), lower adoption friction
- **Cons:** Platform dependency, limited to what MCP can express, may not support enforcement gates
- **When to use instead:** If the protocol approach fails to gain traction

**Rationale for recommendation:** Option A (protocol) creates the largest possible addressable market and positions the system as infrastructure rather than a feature. The other options can be pursued in parallel as concrete implementations of the protocol.

---

### Implementation Details

**What to implement first:**
- MCP server for `kb context` — lowest-friction way for Cursor/Claude Code users to try the knowledge base
- Standalone specification document for the investigation/probe/model lifecycle
- One compelling demo: "here's what happens with and without knowledge governance over 10 agent sessions"

**Things to watch out for:**
- ⚠️ Acontext is early but directly addressing accumulated agent knowledge — monitor closely
- ⚠️ Cursor's Memories feature, if stabilized, could partially close the gap for their 1M DAU
- ⚠️ The enforcement overhead must be justified by measurable reduction in wasted work — need concrete metrics beyond the 85.5% orphan rate
- ⚠️ The "Codified Context" paper authors may build an implementation — check for follow-up work

**Areas needing further investigation:**
- User research with solo AI-heavy developers on willingness to adopt structural enforcement
- Quantitative measurement of token waste from re-investigation (the "15x more tokens" multi-agent stat from O'Reilly)
- Whether enforcement gates can work via MCP or require deeper integration

**Success criteria:**
- ✅ kb context MCP server usable from Cursor and Claude Code
- ✅ At least one external user adopts the investigation/probe/model cycle
- ✅ Measurable reduction in re-investigation rate (baseline: 85.5% orphan rate without enforcement)

---

## Competitive Landscape Map

```
                         STRUCTURAL ENFORCEMENT
                                ↑
                                |
                    [EMPTY — orch-go .kb/ is here alone]
                                |
                                |
     Obsidian+AI ←——————————————+——————————————→ Acontext
     (human knowledge           |                (learned skills
      exploration)              |                 from executions)
                                |
                    RETRIEVAL ←—+—→ GOVERNANCE
                                |
     Cursor Rules  ←————————————+——————————————→ mem0, Letta, Zep
     CLAUDE.md                  |                (AI memory layers)
     AGENTS.md                  |
     (static instructions)      |
                                |
     Context7, Repomix ←————————+——————————————→ CrewAI, LangGraph
     Aider, Augment             |                AutoGen, Semantic Kernel
     (context packing/          |                (agent orchestration)
      code search)              |
                                |
                                ↓
                         NO ENFORCEMENT
```

---

## References

**Tools Examined:**
- Cursor Rules (.cursor/rules/*.mdc), CLAUDE.md, AGENTS.md
- mem0, Letta/MemGPT, Zep, Cognee, LangMem, Claude-Mem
- Obsidian + Smart Connections, Copilot plugin
- adr-tools, Log4brains, MADR, ADR Manager, AgDR, AgenticAKM
- Confluence, Notion, GitBook, Mintlify, Backstage TechDocs, Swimm, Guru
- CrewAI, AutoGen, LangGraph, Semantic Kernel, Claude Agent SDK, OpenAI Agents SDK
- Context7, Repomix, Aider repo map, Augment Context Engine, Sourcegraph Cody, Greptile, Windsurf Memories, Copilot Spaces, Acontext, MemU

**Key Research Papers:**
- ETH Zurich: "Evaluating AGENTS.md" (arxiv 2602.11988, Feb 2026) — AGENTS.md can reduce task success rates
- "Codified Context: Infrastructure for AI Agents" (arxiv 2602.20478, Feb 2026) — three-tier architecture matching orch-go's pattern
- AgenticAKM (arxiv, Feb 2026) — multi-agent architecture knowledge management
- MemGPT (NeurIPS 2023) — foundational agent self-managing memory paper

**Industry Sources:**
- O'Reilly: "Why Multi-Agent Systems Need Memory Engineering" — 36.9% failure rate from inter-agent misalignment
- Martin Fowler: "Context Engineering for Coding Agents"
- Augment Code: "Your agent's context is a junk drawer"
- Thoughtworks Technology Radar: Context7

---

## Investigation History

**[2026-03-11]:** Investigation started
- Initial question: What tools exist in the 'structured knowledge management for AI-assisted development' space? Where does kb-cli have a genuine unique advantage?
- Context: 1,166+ investigations produced, 85.5% orphan rate discovered, harness engineering developed

**[2026-03-11]:** 8 parallel research agents launched covering all competitive categories

**[2026-03-11]:** All research complete, synthesis produced
- Key outcome: The market has no tool that treats knowledge as governance rather than retrieval. This is the genuine positioning gap.
