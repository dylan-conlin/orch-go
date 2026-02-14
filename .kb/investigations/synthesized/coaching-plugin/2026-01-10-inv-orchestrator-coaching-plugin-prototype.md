<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** OpenCode plugins can only detect behavioral patterns via tool hooks, not analyze free-text responses, requiring proxy metrics for Level 1→2 pattern detection.

**Evidence:** Existing plugins (`evidence-hierarchy.ts`, `action-log.ts`) demonstrate tool-level hooks only; no message/response event hooks found; `action-log.jsonl` pattern shows JSONL persistence works.

**Knowledge:** Must use behavioral proxies (context-gathering ratio, action/read balance, tool sequences) instead of direct text analysis; can follow action-log pattern for metrics storage and `orch patterns` for surfacing.

**Next:** Implement coaching plugin with tool hooks → JSONL metrics → API endpoint → dashboard view.

**Promote to Decision:** recommend-no (tactical implementation following established patterns)

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

# Investigation: Orchestrator Coaching Plugin Prototype

**Question:** How should the orchestrator coaching plugin detect Level 1→2 patterns and display metrics?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** og-feat-orchestrator-coaching-plugin-10jan-2249
**Phase:** Complete
**Next Step:** None (investigation complete, moving to design phase)
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: OpenCode Plugin System Structure

**Evidence:** 
- Three existing plugins provide clear patterns: `evidence-hierarchy.ts`, `orchestrator-session.ts`, `action-log.ts`
- Plugins export a function receiving `{ project, client, $, directory, worktree }`
- Hook system supports: `config`, `tool.execute.before/after`, `event`
- `client.session.prompt({ prompt, noReply: true })` injects warnings without blocking
- Can track state across tool calls using Maps/Sets

**Source:** 
- `plugins/evidence-hierarchy.ts` (362 lines) - Complete warning injection example
- `plugins/orchestrator-session.ts` (218 lines) - Event handling and config hooks
- `.orch/workspace/og-feat-opencode-plugin-evidence-08jan-6ca9/SYNTHESIS.md` - Prior implementation notes

**Significance:** Provides proven patterns for detecting tool usage patterns and injecting coaching messages without disrupting agent flow

---

### Finding 2: Dashboard API Structure

**Evidence:**
- API server in `cmd/orch/serve.go` provides REST + SSE endpoints
- Existing endpoints: `/api/agents`, `/api/events`, `/api/beads`, `/api/usage`, `/api/focus`, etc.
- Server runs on port 3348 (DefaultServePort constant)
- Endpoints return JSON, dashboard fetches via HTTP
- Service state available via `serviceMonitor` global with thread-safe mutex

**Source:**
- `cmd/orch/serve.go:1-200` - Server initialization and endpoint list
- `web/src/routes/+page.svelte` - Main dashboard view

**Significance:** Can add `/api/coaching` endpoint following existing patterns to expose plugin metrics to dashboard

---

### Finding 3: Metrics Storage Approach Needed

**Evidence:**
- Plugins are stateless (reset on reload) - use Maps/Sets for session state
- No existing metrics persistence pattern in plugins
- Dashboard fetches from API, not directly from plugins
- Need bridge: plugin tracks → storage → API exposes → dashboard displays

**Source:**
- `plugins/evidence-hierarchy.ts:213-220` - Session-local Sets for tracking
- No file-based persistence in existing plugins

**Significance:** Need to decide: in-memory only (session metrics) vs persistent (file/JSON) for coaching metrics history

---

## Synthesis

**Key Insights:**

1. **Plugin System Provides Tool-Level Hooks Only** - OpenCode plugins see tool calls (Read, Grep, Edit, Bash) but not free-text responses. Pattern detection must work from behavioral signals (tool usage patterns) rather than text analysis of orchestrator messages.

2. **Existing Action-Log Infrastructure Can Be Extended** - The `action-log.ts` plugin demonstrates JSONL-based metrics persistence and the `orch patterns` command shows the pattern for surfacing behavioral analysis. Can follow same approach for coaching metrics.

3. **Proxy Metrics for Level 1→2 Detection** - Since direct text analysis isn't available, use behavioral proxies:
   - **Context-gathering ratio**: kb context checks before spawns (strategic) vs spawns without context (tactical)
   - **Action ratio**: autonomous actions (Edit/Write/Bash) vs passive reads (Read/Grep)
   - **Tool sequence patterns**: Multiple tools of same type in sequence (analysis paralysis indicator)

**Answer to Investigation Question:**

The orchestrator coaching plugin should:
1. Hook `tool.execute.before/after` to track tool usage patterns
2. Detect behavioral signals as proxies for Level 1→2 patterns (context-gathering ratio, action/read balance, tool sequences)
3. Store metrics in `~/.orch/coaching-metrics.jsonl` (following action-log pattern)
4. Expose via new `/api/coaching` endpoint
5. Display in dashboard with simple metrics card showing ratios and trends

Limitation: Cannot directly detect "option theater" text patterns without access to LLM responses, must infer from tool behaviors.

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

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

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
