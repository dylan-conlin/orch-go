# Probe: Context Mode — Compression Architecture and Agent Session Extension

**Model:** orchestrator-session-lifecycle
**Date:** 2026-02-28
**Status:** Complete

---

## Question

Can external context compression (via PreToolUse hooks) meaningfully extend agent working sessions without losing actionable information? The orchestrator-session-lifecycle model assumes agents operate within fixed context windows with SPAWN_CONTEXT + skill + kb context consuming significant budget upfront. Context Mode (mksglu/claude-context-mode) claims to compress tool outputs (e.g., Playwright snapshots 56KB → 299B). What is the actual compression mechanism, and does it confirm, contradict, or extend our model's assumptions about context management?

---

## What I Tested

Cloned mksglu/claude-context-mode (v0.7.3) and read the complete source code:

```bash
git clone --depth 1 https://github.com/mksglu/claude-context-mode.git /tmp/claude-context-mode
# Read: src/server.ts (1192 lines), src/executor.ts (330 lines), src/store.ts (807 lines)
# Read: src/runtime.ts, src/cli.ts, hooks/pretooluse.sh, skills/context-mode/SKILL.md
# Read: BENCHMARK.md, README.md, tests/ecosystem-benchmark.ts
# Read: tests/fixtures/playwright-snapshot.txt (57KB real Playwright accessibility tree)
# Checked: all 7 open GitHub issues, plugin.json, marketplace.json
```

Specific areas examined:
1. The `execute` tool's subprocess isolation mechanism (executor.ts)
2. The `ContentStore` FTS5 schema and chunking algorithm (store.ts)
3. The `pretooluse.sh` hook routing logic and failure modes
4. The benchmark test code to understand how "56KB → 299B" is actually measured
5. The intent-driven search pipeline (intentSearch in server.ts)

---

## What I Observed

### 1. Architecture: NOT LLM Compression — Subprocess Isolation + Agent-Authored Summarization

**Critical finding:** Context Mode does NOT use LLM-powered summarization. The blog post's framing is misleading. Here is what actually happens:

**The `execute` tool** spawns a child process (`child_process.spawn`) in a temp directory. The agent writes code that processes data and outputs a summary via stdout. Only stdout enters the conversation context. The raw data never leaves the subprocess.

Example from the benchmark — "Playwright 56KB → 299B":
```javascript
// The AGENT writes this code, not the tool:
const links = [...FILE_CONTENT.matchAll(/- link "([^"]+)"/g)].map(m => m[1]);
const buttons = [...FILE_CONTENT.matchAll(/- button "([^"]+)"/g)].map(m => m[1]);
console.log("Links:", links.length, "| Buttons:", buttons.length);
console.log("Top 5:", stories.slice(0, 5).join(", "));
```

So the "compression" is: the agent writes analysis code → code runs in subprocess → only the printed summary enters context. The compression quality depends entirely on how good the agent's analysis code is.

**The `index` + `search` tools** provide an alternative path: raw content is chunked into a SQLite FTS5 database, and the agent queries it on-demand with BM25 search. This returns exact text chunks (not summaries), achieving 44-93% savings depending on query selectivity.

### 2. Compression Quality: What Is Lost

**For `execute`/`execute_file` (agent-authored summaries):**
- The 56KB Playwright snapshot → 299B means the agent gets: `"Links: 87 | Buttons: 3 | Text nodes: 142 | Stories found: 30 | Top 5: [titles]"`
- What is lost: ALL element refs ([ref=e37]), ALL URLs, ALL nesting structure, ALL accessibility attributes, ALL form field details
- An agent CANNOT make meaningful UX judgments from 299B. It can count elements and read titles. It cannot, for example, determine if a form has proper label associations, whether navigation is accessible, or click interactive elements by ref.

**For `index` + `search` (FTS5 retrieval):**
- The same Playwright snapshot indexed and searched returns exact chunks (~2-3KB) containing the actual accessibility tree fragments matching the query
- This preserves actionable detail but requires the agent to know what to search for
- Savings are lower (44-85%) but information quality is higher

### 3. Knowledge Base: SQLite FTS5 Implementation

Schema (from store.ts:188-217):
```sql
-- Porter stemming for English word matching
CREATE VIRTUAL TABLE chunks USING fts5(
  title, content, source_id UNINDEXED, content_type UNINDEXED,
  tokenize='porter unicode61'
);
-- Trigram for substring/partial matching
CREATE VIRTUAL TABLE chunks_trigram USING fts5(
  title, content, source_id UNINDEXED, content_type UNINDEXED,
  tokenize='trigram'
);
-- Vocabulary for fuzzy correction
CREATE TABLE vocabulary (word TEXT PRIMARY KEY);
```

**Chunking strategy:** Markdown is split by headings (H1-H4), keeping code blocks intact. Plain text is split by blank lines or into 20-line chunks with 2-line overlap.

**Search fallback cascade:** Porter → Trigram → Levenshtein fuzzy correction. Three layers ensure typos and partial terms still match.

**Comparison to our `kb context` system:** Context Mode's FTS5 is session-ephemeral (temp DB deleted on exit), designed for within-session retrieval. Our `kb context` is persistent across sessions, designed for cross-session knowledge accumulation. They solve different problems — Context Mode is tactical (reduce this tool output), kb context is strategic (what do we know about this domain).

### 4. PreToolUse Hook Routing: Mechanism and Failure Modes

The `pretooluse.sh` hook intercepts 5 tool types:

| Tool | Action | Mechanism |
|------|--------|-----------|
| Bash (curl/wget) | **Redirect** | Replaces command with echo telling agent to use `execute` instead |
| Bash (inline HTTP) | **Redirect** | Same — replaces `node -e "fetch(...)"` with echo |
| Bash (other) | **Passthrough** | git, mkdir, etc. pass through unmodified |
| WebFetch | **Deny** | `permissionDecision: deny` with redirect instructions |
| Read | **Nudge** | `additionalContext` suggesting execute_file for large files |
| Grep | **Nudge** | `additionalContext` suggesting execute for large results |
| Task (subagent) | **Inject** | Appends CONTEXT_WINDOW_PROTECTION routing block to prompt |

**Failure modes identified:**

1. **False positive on Bash commands:** Any Bash command containing "curl" or "wget" as a substring gets redirected, even in comments or variable names. The regex `grep -qiE '(^|\s|&&|\||\;)(curl|wget)\s'` is word-boundary-aware but not perfect.

2. **Read tool only nudges, doesn't block:** The hook adds `additionalContext` saying "prefer execute_file for large files" but the Read still executes. The agent may ignore the nudge. This is by design (you need Read for files you want to edit).

3. **Subagent prompt injection is append-only:** The routing block is appended to the end of the subagent prompt. If the original prompt already has conflicting instructions (like "use Bash for everything"), the agent must resolve the conflict.

4. **No Edit/Write interception:** The hook doesn't touch Edit or Write tools, which is correct — those are file mutations, not data reads.

5. **Bash subagent upgrade to general-purpose:** When a Task has `subagent_type: "Bash"`, the hook upgrades it to `general-purpose` so the subagent can access MCP tools. This is clever but changes the subagent's entire tool surface area.

### 5. Applicability to Our Stack

**Our spawn context budget:** SPAWN_CONTEXT + skill + kb context can consume 5-20KB at spawn time. Agent working sessions use context for tool outputs (Read, Grep, Bash results). Context exhaustion is a real constraint — agents slow down and lose coherence after ~45 min of heavy tool use.

**What Context Mode could help with:**
- **Playwright snapshot compression in UX audits:** Our recent probe (2026-02-28-probe-playwright-cli-vs-mcp-ux-audit) involves accessibility tree snapshots. These are the exact use case Context Mode targets.
- **Large log/output analysis:** When agents run `go test ./...` or read large files, the output floods context. The `execute` + intent-search pattern could keep raw output in sandbox.
- **Subagent context protection:** Our Agent tool spawns subagents that can return large results. Context Mode's Task hook injection could limit subagent output bloat.

**What Context Mode would NOT help with:**
- **Our primary context consumers:** SPAWN_CONTEXT, skill content, and kb context are injected at session start, not via tool calls. Context Mode only intercepts tool outputs.
- **Claude CLI agents (our default backend):** Context Mode is a Claude Code MCP plugin. Our agents run via `claude` CLI in tmux. MCP hooks work in Claude Code, not in the raw CLI.
- **Structured agent protocols:** Our worker-base patterns (bd comment, Phase reporting, SYNTHESIS.md) generate small, structured outputs that don't benefit from compression.

### 6. Limitations and Risks

**Known issues (from GitHub):**
- **OOM vulnerability (#5):** Before the fix, a command like `yes` or `cat /dev/urandom` could accumulate gigabytes in memory before timeout killed it. The fix adds a stream-level 100MB hard cap. (PR open, not merged as of repo state)
- **Shell $-expansion in paths (#7):** `executeFile` for shell language used double quotes, causing `$HOME` expansion in paths. Fix: switch to single quotes.
- **searchWithFallback not wired (#4):** The three-layer search cascade was implemented but never called from server.ts. All search went through bare `store.search()` (Porter only). Fix wires it in.

**Architectural risks:**
- **Agent must write good analysis code:** The quality of "compression" depends entirely on the agent writing good summarization scripts. A naive agent that does `console.log(JSON.stringify(data))` gets zero compression.
- **Two-hop indirection:** Instead of one tool call (Read file), the agent needs two: execute_file + potentially search. This doubles API round-trips.
- **Session-ephemeral DB:** The FTS5 database is tied to the MCP server process (PID-named temp file). If the server crashes, all indexed content is lost.
- **No sandboxing beyond process isolation:** The subprocess inherits PATH, HOME, and auth credentials. It's not a container or VM — it's `child_process.spawn` with a restricted env. A malicious agent could still access the filesystem.

---

## Model Impact

- [x] **Extends** model with: Context window management via tool-output interception is a viable strategy for extending agent session duration, but requires MCP integration (Claude Code plugin, not raw CLI). The mechanism is subprocess isolation + agent-authored summarization, NOT LLM compression. The approach is complementary to our upfront context budget (SPAWN_CONTEXT) — it targets tool-output bloat, not initial context injection. Our model should account for two distinct context pressure sources: (1) spawn-time injection (our current focus) and (2) runtime tool-output accumulation (what Context Mode addresses). For our Claude CLI agents, a similar approach would require building equivalent functionality into our spawn infrastructure or adopting Claude Code as the agent runtime.

---

## Notes

**Key insight for our stack:** The valuable idea from Context Mode is not the specific tool — it's the pattern of **subprocess-isolated execution with intent-driven retrieval**. This pattern could be implemented in our agent infrastructure without adopting the full Context Mode package:

1. **For Playwright UX audits:** Save snapshots to file, have agents write extraction scripts instead of loading full accessibility trees into context
2. **For large test output:** Pipe `go test` output through a summarization script before it enters agent context
3. **For kb context queries:** Our `kb context` already does intent-driven retrieval. Context Mode's FTS5 approach validates this pattern.

**The PreToolUse hook pattern is powerful** but only works in Claude Code. Our tmux-based Claude CLI agents don't have this hook mechanism. If we wanted similar routing, we'd need to build it into our spawn infrastructure (e.g., wrapping Bash commands in our own interception layer).

**Recommendation:** Don't adopt Context Mode directly (wrong runtime — we use Claude CLI, not Claude Code). Instead, apply the subprocess-isolation pattern selectively to our highest-context-cost operations (Playwright snapshots, large test output). This could be implemented as spawn-context guidance ("save snapshots to file, extract via script") rather than infrastructure.
