<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The harness engineering blog post is the best wedge — not a tool. Content has zero adoption friction and naturally funnels readers to the tools (skillc, kb-cli). The prior probe that identified kb-cli as primary wedge was evaluating tools in isolation from their distribution mechanism.

**Evidence:** Examined 6 candidates across the ecosystem: skillc (standalone, on GitHub, niche audience), beads (not Dylan's tool, high friction), harness engineering (5,800-word draft exists, references OpenAI/Fowler/Anthropic/MAST paper, novel "compliance vs coordination" distinction), coordination demo (80-trial experiment with surprising results), CLAUDE.md patterns (universal Claude Code need), kb-cli (strong tool, high conceptual overhead). Blog drafts directory shows 15 posts in various stages.

**Knowledge:** The wedge hierarchy is content → templates → tools. Content has zero friction. Templates have low friction. Tools have high friction. Getting "one external user" requires the lowest-friction entry point. A blog post someone reads IS usage. A tool someone installs requires 10x more motivation.

**Next:** Publish the harness engineering blog post. It's the highest-leverage action with the lowest effort (draft already exists, needs light editing). Follow with CLAUDE.md template as the first "tool" wedge. kb-cli and skillc come later as readers deepen.

**Authority:** strategic - This is a distribution strategy decision involving irreversible public positioning and value judgment about what to lead with.

---

# Investigation: Wedge Candidate Inventory for the orch-go Ecosystem

**Question:** Beyond kb-cli, what other discrete, standalone tools or concepts exist in Dylan's orch-go ecosystem that could serve as a distributable wedge — something that solves a problem OTHER people have and could get one external user?

**Started:** 2026-03-11
**Updated:** 2026-03-11
**Owner:** Dylan Conlin
**Phase:** Complete
**Next Step:** None
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Prior kb-cli wedge probe (2026-03-09) | Extends | yes | Yes — kb-cli is a strong tool but not the best *first* wedge. Content is. |

---

## Findings

### Finding 1: The Harness Engineering Blog Post Is the Strongest Wedge

**Evidence:** A 5,800-word draft exists at `blog/src/content/posts/harness-engineering.md` (draft: true). It:
- References three authoritative sources: OpenAI's Codex harness engineering, Fowler & Böckeler's analysis, Anthropic's long-running agent harness guide
- Cites an academic paper (Cemri et al., MAST taxonomy, arxiv 2503.13657)
- Contains original data: 12 weeks, 50+ agents/day, 3 entropy spirals, 265 contrastive trials
- Introduces a novel distinction: compliance failure vs coordination failure — with opposite trajectories as models improve
- Provides a concrete "Getting Started" section (Day 0, Day 1, Week 1)
- Includes honest assessment of what isn't working yet

**Source:** `~/Documents/personal/blog/src/content/posts/harness-engineering.md` (274 lines), `~/Documents/personal/blog/drafts/2026-03-08-soft-harness-doesnt-work.md` (earlier draft exploring same ideas)

**Significance:** This is the highest-leverage candidate because:
1. **Zero adoption friction** — reading a blog post requires no installation, no account, no commitment
2. **Problem resonance** — anyone running >5 AI agents on a codebase will recognize accretion
3. **Credibility through specificity** — real data, real failures, real measurements, honest "what isn't working" section
4. **Natural funnel** — readers who resonate will want the tools (orch-go, skillc, kb-cli)
5. **Timely** — OpenAI published their harness engineering post, Fowler published his analysis, Anthropic published theirs. The conversation is active.

---

### Finding 2: skillc Is the Most Extractable Tool but Needs the Blog Post Context

**Evidence:** skillc is completely standalone:
- Separate repository (`~/Documents/personal/skillc`, `github.com/dylan-conlin/skillc`)
- Zero dependencies on orch-go (only fsnotify, uuid, yaml.v3)
- Published on GitHub already
- Has a blog draft (`drafts/2025-12-23-skillc.md` — "English is the New Programming Language. It Shipped Without a Type System.")
- Commands: init, build, check, deploy, stats, watch, lint, test, compare, verify

But the concept of "skill compilation" is meaningless without context. Why would someone compile skills? The harness engineering post provides that context: "behavioral constraints dilute at scale → move them to hard harness → skillc compiles modular skill sources into verifiable artifacts."

**Source:** Agent exploration of skillc repo structure, `~/Documents/personal/skillc/`, `skills/src/` in orch-go

**Significance:** skillc is a strong Tier 2 wedge — the first tool people try AFTER the blog post convinces them the problem is real. Lead with the problem (accretion, coordination failure), then offer the tool.

---

### Finding 3: CLAUDE.md Template Is the Lowest-Friction Tool Wedge

**Evidence:** Every Claude Code user needs a CLAUDE.md. Most have minimal ones. Dylan's is 427 lines with:
- Architecture overview with ASCII diagrams
- Spawn flow documentation
- Accretion boundaries section
- Tab-indented file editing workarounds (solves a real, documented Claude Code bug)
- Key packages reference
- Gotchas section (20+ specific issues)
- Command reference

A "CLAUDE.md best practices" guide + template would:
- Solve a universal problem (how to document a project for AI agents)
- Require zero tool installation
- Be immediately applicable
- Drive traffic to the blog and tools

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/CLAUDE.md`

**Significance:** This is potentially the highest-reach wedge because the audience is all Claude Code users (not just multi-agent operators). But it's also the shallowest — a template doesn't create ongoing engagement the way a tool does.

---

### Finding 4: Coordination Demo Has Novel Findings Worth Publishing

**Evidence:** 80-trial experiment (4 conditions × 2 tasks × 10 trials):
- **Placement-based coordination:** 100% clean merge rate (20/20 trials)
- **No coordination:** 0% clean merge rate (0/20)
- **Context sharing:** 0% clean merge rate (0/20) — surprising: giving agents each other's task description doesn't help
- **Messaging:** 0% clean merge rate (0/20) — agents created plans but still conflicted

The finding that "context sharing doesn't reduce conflicts" contradicts intuition. Most people assume that giving agents more information about each other's work will help. This data shows it doesn't — only explicit placement instructions work.

Self-contained bash framework, reproducible with just Claude CLI + Go + git.

**Source:** `experiments/coordination-demo/redesign/results/20260310-174045/`, `redesign/run.sh`

**Significance:** Strong blog post or technical report material. Could be a standalone post or a supporting section of the harness engineering post. The experimental framework itself is distributable but has a very narrow audience (researchers running multi-agent experiments).

---

### Finding 5: beads Is Not a Viable Wedge (Not Dylan's, High Friction)

**Evidence:** beads is Steve Yegge's project (`github.com/steveyegge/beads`). Dylan uses it but doesn't maintain it. Key characteristics:
- Git-backed JSONL issue tracking
- Merge-conflict-free via hash-based IDs
- Daemon auto-sync
- 1,368 issues in orch-go's production database

The merge-conflict-free property is genuinely novel but only matters at scale (50+ agents). For 1-5 agents, GitHub Issues works fine. The conceptual overhead (JSONL, hash IDs, SQLite cache layer, daemon sync) is high for a new user.

**Source:** Agent exploration of `~/Documents/personal/beads/`, `.beads/` in orch-go

**Significance:** Not a wedge candidate. It's infrastructure that supports the system but doesn't standalone for external users. The value proposition ("issue tracking that doesn't conflict when 50 agents work simultaneously") requires a scale most people don't have.

---

### Finding 6: kb-cli Is Strong but Requires Conceptual Buy-In

**Evidence:** kb-cli is fully standalone:
- Separate repo (`github.com/dylan-conlin/kb-cli`), single Go binary
- Minimal dependencies (snowball stemming, cobra, yaml)
- Core commands: init, create, search, context, reflect, quick
- The investigation → probe → model cycle is powerful but unfamiliar

The problem: explaining kb-cli requires explaining the cycle. "A CLI for managing a structured knowledge base" sounds like... a wiki? a note-taking app? The value only becomes clear when you understand session amnesia (AI agents forget between sessions) and knowledge compounding (investigations feed models feed probes feed investigations).

**Source:** Agent exploration of `~/Documents/personal/kb-cli/`, orch-go's `cmd/orch/kb*.go`

**Significance:** kb-cli remains a strong tool wedge but is better positioned as a Tier 2 offering — something people discover after the blog post explains *why* knowledge compounding matters for AI agent systems.

---

## Synthesis

**Key Insights:**

1. **The wedge hierarchy is content → templates → tools.** Each level has higher adoption friction. A blog post someone reads IS a "user" for zero friction. A template someone copies requires 5 minutes. A tool someone installs requires 30+ minutes and conceptual buy-in. The prior probe evaluated tools in isolation, missing that content is the zero-friction entry point.

2. **The harness engineering post is uniquely well-timed.** OpenAI, Fowler, and Anthropic all published harness-related content in late 2025 / early 2026. The conversation is active. Dylan's contribution is distinct: the compliance vs coordination failure distinction, real multi-agent data, and the honest "what isn't working" assessment. This positions him as a practitioner, not a vendor.

3. **Tools need problems before they need features.** kb-cli and skillc are strong tools, but they solve problems most people don't know they have yet. The blog post creates problem awareness ("your codebase will degrade under multi-agent pressure") that then creates demand for the tools.

**Answer to Investigation Question:**

The best wedge is NOT a tool — it's the harness engineering blog post. It solves the "getting one external user" problem by being zero-friction and timely. The ranked assessment:

| Rank | Candidate | Type | Audience Size | Time to Ship | Adoption Friction | Problem Clarity |
|------|-----------|------|--------------|--------------|-------------------|-----------------|
| 1 | Harness engineering blog post | Content | 10,000+ (AI-assisted devs) | Days (draft exists) | Zero (read a post) | High (everyone running agents recognizes accretion) |
| 2 | CLAUDE.md template/guide | Template | 5,000+ (all Claude Code users) | 1 week | Very low (copy template) | High (everyone needs a CLAUDE.md) |
| 3 | Coordination demo findings | Content | 2,000+ (AI engineering) | 1-2 weeks | Zero (blog post) | Medium (need to be running multi-agent) |
| 4 | skillc | Tool | 500+ (Claude Code power users) | Already shipped | Medium (install + learn concept) | Low without blog context |
| 5 | kb-cli | Tool | 1,000+ (AI agent developers) | Already shipped | Medium-high (conceptual overhead) | Low without blog context |
| 6 | beads | Tool | 100+ (heavy multi-agent operators) | N/A (not Dylan's) | High (complex system) | Very low |

The winning strategy: **Publish harness engineering post (Rank 1) → Link to CLAUDE.md template (Rank 2) → Reference skillc and kb-cli as tools for practitioners who want to go deeper (Ranks 4-5).**

---

## Structured Uncertainty

**What's tested:**

- ✅ All 6 candidates exist and were directly examined (code, docs, blog drafts)
- ✅ skillc is fully standalone with zero orch-go dependencies (verified go.mod)
- ✅ Harness engineering blog draft exists and is substantial (5,800 words, 274 lines)
- ✅ Coordination demo has 80 completed trials with clear results
- ✅ beads is maintained by Steve Yegge, not Dylan

**What's untested:**

- ⚠️ Whether the harness engineering post would actually attract readers (no distribution channel validated)
- ⚠️ Whether CLAUDE.md templates have demand (no search volume data)
- ⚠️ Whether the coordination demo findings would be surprising to the AI engineering community (no external feedback)
- ⚠️ Whether "zero readers" on the blog means zero distribution capability or just no content worth reading yet

**What would change this:**

- If Dylan has no distribution channel (no Twitter following, no HN karma, no community presence), even the best content won't reach anyone. The blog post is necessary but not sufficient — distribution matters.
- If Claude Code's user base is much smaller than estimated, the CLAUDE.md template audience shrinks proportionally.
- If someone else publishes a similar "multi-agent coordination failure" analysis first, the timing advantage disappears.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Publish harness engineering blog post as primary wedge | strategic | Irreversible public positioning, value judgment about what to lead with |
| Create CLAUDE.md template as first tool-adjacent wedge | implementation | Low-risk, reversible, within existing patterns |
| Write coordination demo findings as standalone post | strategic | Public research positioning, irreversible |

### Recommended Approach ⭐

**Content-First Wedge Strategy** — Publish the harness engineering blog post, use it to create problem awareness, then offer tools to readers who want to go deeper.

**Why this approach:**
- Zero friction for the first "user" (reader)
- Draft already exists (days of work, not weeks)
- Timely — references active conversation (OpenAI, Fowler, Anthropic)
- The post naturally mentions tools without being a sales pitch
- Positions Dylan as practitioner with data, not vendor with product

**Trade-offs accepted:**
- A blog post doesn't produce tool adoption directly
- "Readers" are a weaker form of "users" than tool installs
- Distribution is the unsolved problem (no audience yet)

**Implementation sequence:**
1. Edit and publish harness engineering blog post (1-3 days)
2. Create a "CLAUDE.md best practices" companion post or GitHub gist with template (1 week)
3. Write coordination demo findings as a follow-up post (1-2 weeks)
4. Add "Tools" section to blog linking to skillc and kb-cli repos

### Alternative Approaches Considered

**Option B: Lead with skillc as tool wedge**
- **Pros:** Already shipped, on GitHub, tangible product
- **Cons:** Concept is unfamiliar without context, narrow audience (Claude Code power users), README doesn't explain WHY you'd want skill compilation
- **When to use instead:** If blog gets traffic and readers ask "how do you manage skills?"

**Option C: Lead with kb-cli as tool wedge (prior probe recommendation)**
- **Pros:** Stronger standalone value proposition, broader audience than skillc
- **Cons:** High conceptual overhead (investigation/probe/model cycle), requires explaining session amnesia first
- **When to use instead:** If targeting developers who already understand the knowledge compounding problem

**Rationale for recommendation:** Content creates the demand that tools fulfill. Without problem awareness, tools are solutions looking for problems. The harness engineering post creates problem awareness for the exact audience who would then want skillc and kb-cli.

---

### Implementation Details

**What to implement first:**
- Light edit pass on harness engineering draft (it's already strong)
- Add links to skillc repo and kb-cli repo in the "Getting Started" section
- Publish (set draft: false)

**Things to watch out for:**
- ⚠️ Distribution is the real blocker — a post with no readers has no users. Consider: HN submission, relevant Discord/Slack communities, Twitter/X if Dylan has presence
- ⚠️ The post is currently 5,800 words — may benefit from cutting to 3,000-4,000 for broader reach
- ⚠️ The "Getting Started" section references `orch init` which requires orch-go — should generalize for readers without orch

**Areas needing further investigation:**
- Dylan's distribution channels (social media presence, community memberships, HN account)
- Whether the blog has analytics / any prior traffic at all
- Whether cross-posting to dev.to, Medium, or similar platforms would increase reach

**Success criteria:**
- ✅ Blog post published and accessible at public URL
- ✅ At least 1 external person reads and engages (comment, share, star)
- ✅ CLAUDE.md template available as GitHub gist or repo

---

## References

**Files Examined:**
- `~/Documents/personal/blog/src/content/posts/harness-engineering.md` — 5,800-word draft on harness engineering
- `~/Documents/personal/blog/src/content/posts/gate-over-remind.md` — Published post on gates vs reminders
- `~/Documents/personal/blog/drafts/2025-12-23-skillc.md` — skillc blog draft
- `~/Documents/personal/blog/drafts/` — 15 draft posts in various stages
- `~/Documents/personal/orch-go/CLAUDE.md` — 427-line project documentation
- `~/Documents/personal/orch-go/skills/src/` — Skill source directory
- `~/.claude/skills/` — Deployed skill destination
- `~/Documents/personal/skillc/` — Standalone skillc repository
- `~/Documents/personal/kb-cli/` — Standalone kb-cli repository
- `~/Documents/personal/beads/` — beads issue tracking (Steve Yegge's)
- `experiments/coordination-demo/redesign/` — Multi-agent coordination experiments

**Related Artifacts:**
- **Investigation:** Prior kb-cli wedge probe (2026-03-09)
- **Blog posts:** 8 published, 15 drafts in pipeline

---

## Investigation History

**2026-03-11 18:05:** Investigation started
- Initial question: What wedge candidates exist beyond kb-cli?
- Context: Prior probe identified kb-cli as primary wedge; testing if that's the best option

**2026-03-11 18:15:** Parallel exploration of all 6 candidates
- Explored: skillc, beads, coordination demo, harness engineering, CLAUDE.md patterns, kb-cli
- Key discovery: harness engineering blog draft already exists and is substantial

**2026-03-11 18:25:** Blog content analysis
- Found 8 published posts + 15 drafts
- Harness engineering draft is the most complete and timely
- Gate Over Remind post already published — establishes the concept

**2026-03-11 18:30:** Investigation completed
- Status: Complete
- Key outcome: Content (blog post) is the best wedge, not tools. The prior probe was right about kb-cli as a tool but wrong about tools being the right first move.
