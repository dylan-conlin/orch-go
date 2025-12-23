# Design: Practitioner Research Infrastructure

**Phase:** Complete
**Question:** What abstraction should we use to give agents access to practitioner signal from HN, Reddit, and similar communities?
**Started:** 2025-12-23
**Updated:** 2025-12-23
**Status:** Complete

## Problem Framing

### Design Question
How should orch-go agents access practitioner signal (real-world experiences, opinions, sentiment) from developer communities like Hacker News, Reddit (r/LocalLLaMA, r/MachineLearning), and potentially others?

### Success Criteria
1. **Autonomous operation** - Agents can research without interactive auth flows
2. **Structured output** - Returns analyzable data, not just raw text
3. **Cross-project reuse** - Not tied to orch-go specifically
4. **Secure credential management** - OAuth tokens stored safely
5. **Rate limit aware** - Handles API limits gracefully
6. **Integration-friendly** - Works with existing spawn/skill patterns

### Constraints
- Must work in headless agent sessions
- OAuth credentials need secure storage (keychain or equivalent)
- Reddit API has 100 req/min rate limit
- HN has no auth/rate limits (Firebase API)
- Should leverage existing content-analyzer work, not duplicate
- Agent token budget matters (can't fetch unlimited content)

### Scope
**In scope:**
- Hacker News (Firebase API - no auth)
- Reddit (r/LocalLLaMA, r/MachineLearning, r/programming, etc.)
- Integration with orch spawn/skill patterns
- Structured extraction (not just raw HTML)

**Out of scope:**
- Twitter/X (API access increasingly difficult, 3rd party wrappers unreliable)
- Discord (no public API, would require bot tokens per server)
- Real-time streaming/monitoring (batch queries only)

---

## Exploration

### Approach 1: MCP Server for Practitioner Sources

**Mechanism:**
Build an MCP server (like the existing Playwright MCP) that exposes HN and Reddit as tools:
- `hn_search(query, timeframe)` - Search HN via Algolia API
- `hn_get_thread(story_id)` - Fetch story + comments
- `reddit_search(subreddit, query, sort)` - Search within subreddit
- `reddit_get_thread(url)` - Fetch post + comments
- `practitioner_sentiment(topic)` - Multi-source aggregation

**Pros:**
- Clean abstraction - agent calls tools like any other MCP
- Reusable across all projects (MCP servers are portable)
- Credential management handled once in server
- Can add new sources without changing agents
- Structured JSON responses (not raw HTML)
- Rate limiting handled server-side

**Cons:**
- New infrastructure to build and maintain
- Another server process to run
- MCP protocol overhead for simple queries
- Requires OpenCode MCP configuration

**Complexity:** Medium-high (new server, OAuth handling, MCP protocol)

**Evidence:**
- Existing `--mcp playwright` pattern in orch-go shows this works
- OpenCode already supports MCP servers
- content-analyzer PRAW code is proven and portable

---

### Approach 2: Research Skill Enhancement

**Mechanism:**
Extend the existing `research` skill with practitioner-source capabilities:
- Teach agents how to use HN Algolia API (curl-based, no auth)
- Teach agents how to use PRAW (Python script in spawn context)
- Inject API docs + example code into spawn context
- Agent executes directly, writes investigation file

**Pros:**
- Minimal infrastructure - just skill content
- Agents already understand research skill pattern
- Works immediately with existing spawn/skill patterns
- No new servers or processes
- Proven pattern (content-analyzer does this today)

**Cons:**
- Each agent needs credentials injected (env vars or .env file)
- Agent must install/import PRAW (Python dependency)
- No structured extraction - agent parses raw API responses
- Duplicates knowledge across spawned agents
- Rate limits handled per-agent (inconsistent)

**Complexity:** Low (skill file + templates)

**Evidence:**
- content-analyzer's analyze-reddit.sh and analyze-hn.sh work well
- Research skill already produces investigation files
- Agents successfully call APIs via curl/Python today

---

### Approach 3: Dedicated orch Command

**Mechanism:**
Build `orch research "question" --sources hn,reddit`:
- Fetches relevant threads from specified sources
- Summarizes via lightweight model (Haiku)
- Returns structured JSON or writes investigation file
- Agent consumes summary, not raw data

**Pros:**
- Single command, simple interface
- Handles all source-specific logic internally
- Token-efficient (summarizes before returning to agent)
- Built into orch binary (no extra processes)
- Central credential management

**Cons:**
- Tight coupling to orch-go
- Not reusable by non-orch projects
- Requires Go implementation of Reddit auth
- Summarization adds latency and potential information loss
- Less flexible than agent-driven research

**Complexity:** Medium (Go implementation, OAuth, summarization)

**Evidence:**
- Existing orch commands (spawn, send, etc.) work well
- Go has good Reddit API libraries (go-reddit)

---

### Approach 4: Hybrid - Thin MCP + Agent Intelligence

**Mechanism:**
Build minimal MCP server with just data fetching:
- `hn_thread(id)` → raw thread data (JSON)
- `reddit_thread(url)` → raw thread data (JSON)
- `reddit_search(sub, query)` → list of post IDs

Agent does all analysis/extraction using its intelligence.

**Pros:**
- MCP handles auth/rate-limiting (hard parts)
- Agent handles analysis (LLM strength)
- Simpler server (no analysis logic)
- Flexible - agent decides what to extract
- Reusable across projects

**Cons:**
- Token cost for raw data (can be large)
- Still need MCP server infrastructure
- Agent must understand data format

**Complexity:** Medium (simpler than full MCP, still needs server)

---

## Synthesis

### Recommendation

⭐ **RECOMMENDED: Approach 4 - Hybrid Thin MCP + Agent Intelligence**

**Why this approach:**

1. **Separation of concerns** - MCP handles the hard parts (OAuth, rate limits, credential storage) while agents handle the easy part for LLMs (analysis and extraction).

2. **Token efficiency is manageable** - A typical Reddit thread is 50-200 comments. With comment text only (no metadata), this fits in 10-20K tokens. Agents can sample (top 50 comments by score) to reduce further.

3. **Reusability** - MCP servers work with any OpenCode-based workflow, not just orch-go. Useful for content-analyzer and future projects.

4. **Leverage existing code** - content-analyzer's PRAW patterns are proven. Port to Go for the MCP server or run Python subprocess.

5. **Extensibility** - Adding new sources (eventually Twitter, Stack Overflow) means adding new MCP endpoints, not changing agent skills.

**Trade-off accepted:**
- MCP server is new infrastructure to maintain
- Acceptable because: OAuth/credential management is a one-time cost that pays off across all research tasks

**When this would change:**
- If only HN needed (no auth) → Approach 2 (skill enhancement) sufficient
- If research is rare (< 1/week) → Approach 2 simpler, don't justify MCP
- If Go Reddit libs prove difficult → Keep Python, call via subprocess

### Implementation Sketch

**MCP Server: `practitioner-research-mcp`**

```
Location: ~/.config/opencode/mcp/practitioner-research/
Language: Python (PRAW proven, faster iteration than Go)

Tools:
- hn_search(query: str, days: int = 30) -> list[{id, title, score, comments}]
- hn_thread(id: int, max_comments: int = 100) -> {story, comments[]}
- reddit_search(subreddit: str, query: str, limit: int = 25) -> list[{id, title, score}]  
- reddit_thread(url: str, max_comments: int = 100) -> {post, comments[]}

Config:
- REDDIT_CLIENT_ID, REDDIT_CLIENT_SECRET, REDDIT_USER_AGENT (from keychain or env)
- Rate limiting: Built-in delays for Reddit (100/min limit)
```

**Skill Enhancement: `research` skill addendum**

```markdown
## Practitioner Research Sources

When research question benefits from practitioner experience (not just docs):

### Using practitioner-research MCP

1. Search for relevant threads:
   - `hn_search("your topic", 30)` for Hacker News
   - `reddit_search("LocalLLaMA", "your topic")` for Reddit

2. Fetch promising threads (high score, recent):
   - `hn_thread(12345, max_comments=50)` 
   - `reddit_thread("https://reddit.com/r/...", max_comments=50)`

3. Analyze: Extract consensus, dissent, and notable insights
```

**orch spawn integration:**

```bash
# Enable MCP for research agents
orch spawn --mcp practitioner-research research "What models work best for coding agents?"
```

---

## Structured Uncertainty

**What's tested:**
- ✅ HN Firebase API works without auth (content-analyzer uses it)
- ✅ PRAW OAuth flow works in headless mode (content-analyzer uses it)
- ✅ MCP integration pattern works (`--mcp playwright` exists)
- ✅ Agent analysis of community threads produces valuable output (see reddit analysis example)

**What's untested:**
- ⚠️ Python MCP server in OpenCode (need to verify process management)
- ⚠️ Keychain credential access from MCP server subprocess
- ⚠️ Token budget for typical research tasks (need to measure)
- ⚠️ Rate limit behavior under concurrent agent spawns

**What would change this:**
- If Python MCP proves problematic → Go implementation or subprocess wrapper
- If token costs prohibitive → Add summarization layer in MCP (move toward Approach 3)
- If only HN needed initially → Start with skill enhancement, defer MCP

---

## 80/20 Analysis: Which Sources Matter?

**Tier 1 (Do first):**
- **Hacker News** - Best signal/noise ratio for tech, no auth, already have code
- **Reddit r/LocalLLaMA** - Primary source for LLM practitioner experience

**Tier 2 (Add when needed):**
- **Reddit r/MachineLearning** - More academic, less practitioner-focused
- **Reddit r/programming** - General, less LLM-specific

**Tier 3 (Defer/Skip):**
- **Twitter/X** - API access difficult, would need 3rd party scraping
- **Discord** - No public API, per-server bot tokens, high friction
- **Stack Overflow** - Good for Q&A but less "practitioner sentiment"

**Recommendation:** Start with HN + r/LocalLLaMA only. Covers 80%+ of the use cases mentioned ("What models are practitioners actually using?").

---

## Recommendations

⭐ **RECOMMENDED:** Hybrid MCP Server + Agent Intelligence

**Implementation plan:**

1. **Phase 1: Validate with skill-only approach** (1-2 hours)
   - Add practitioner research section to research skill
   - Include HN curl examples (no auth needed)
   - Test with `orch spawn research "What models do practitioners use for coding agents?"`
   - Validates demand before building infrastructure

2. **Phase 2: Build minimal MCP server** (4-6 hours)
   - Python MCP server with PRAW
   - HN + Reddit endpoints only
   - Keychain credential storage
   - Test in content-analyzer project

3. **Phase 3: Integrate with orch spawn** (1-2 hours)
   - Add to OpenCode MCP config
   - Update research skill with MCP usage patterns
   - Document in CLAUDE.md

**Alternative: Skill-only approach**
- **Pros:** No new infrastructure, works today
- **Cons:** Credential management messier, no rate limit coordination
- **When to choose:** If practitioner research is occasional (< 1/week)

---

## Feature List Updates

Based on this design, the following items should be added:

1. **practitioner-research-mcp** - Build Python MCP server with HN/Reddit endpoints
2. **research-skill-practitioner** - Add practitioner source guidance to research skill  
3. **orch-mcp-config** - Document MCP server setup in orch-go

---

## Self-Review

- [x] Problem framing complete with success criteria
- [x] 4 approaches explored with pros/cons
- [x] Clear recommendation with reasoning
- [x] Trade-offs acknowledged
- [x] Uncertainty assessed (tested/untested)
- [x] 80/20 analysis for source prioritization
- [x] Implementation phases defined

**Self-Review Status:** PASSED
