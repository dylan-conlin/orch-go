## Summary (D.E.K.N.)

**Delta:** `orch dialogue` should use direct Anthropic Messages API for the toolless questioner and an OpenCode session for the knowledgeable expert, with the orch binary orchestrating a turn-based relay loop.

**Evidence:** Codebase analysis confirms: (1) `pkg/anthropic/` already has OAuth headers for API calls, (2) `SendMessageAsync` + SSE idle-polling exists for expert-side messaging, (3) `ExtractRecentText` extracts response text from sessions, (4) workspace/beads/events infrastructure handles artifact production.

**Knowledge:** The power of the original pattern comes from three enforced asymmetries: knowledge (context-loaded vs context-free), capability (tools vs no tools), and pressure direction (questions flow up, answers flow down). All three must be structurally enforced, not suggested.

**Next:** Implement in 4 phases: (1) `pkg/dialogue/` core loop with API client, (2) `cmd/orch/dialogue_cmd.go` CLI, (3) transcript/artifact production, (4) event tracking + dashboard visibility.

**Authority:** architectural - New command introducing cross-agent communication pattern, requires orchestrator-level decision on API client approach.

---

# Investigation: Design orch dialogue Command

**Question:** How should `orch dialogue` automate the asymmetric two-agent design conversation pattern, preserving the properties that made the manual experiment effective?

**Started:** 2026-02-08
**Updated:** 2026-02-08
**Owner:** architect agent (orch-go-21485)
**Phase:** Complete
**Next Step:** Create implementation issues for feature-impl agents
**Status:** Complete

**Patches-Decision:** N/A (new capability)
**Extracted-From:** Manual Dylan experiment (Claude web + orchestrator, ~30min → better design than 2hr tool-equipped agent)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Two-model experiments decision | extends | Yes - confirmed cross-model value for design critique | None |
| Spawn Architecture model | extends | Yes - workspace/beads patterns apply to dialogue | None |
| Agent Lifecycle State Model | extends | Yes - SSE idle-polling is the expert response detection mechanism | None |

---

## Findings

### Finding 1: The Questioner MUST Be Structurally Tool-Free

**Evidence:** The task description explicitly states the power comes from the toolless agent being "forced to think instead of grep." An OpenCode session with a "please don't use tools" instruction violates this — the agent CAN use tools, and under pressure WILL. OpenCode sessions always load CLAUDE.md and project context, contaminating "fresh eyes."

**Source:** `pkg/spawn/context.go:122-130` (WriteContext always injects CLAUDE.md), `pkg/opencode/client.go:40-79` (ClientInterface includes tool-use methods)

**Significance:** The questioner must use the Anthropic Messages API directly (no tool_use blocks in the API call) to get structural enforcement. This is not a policy decision — it's the only way to guarantee the asymmetry.

---

### Finding 2: Expert-Side Infrastructure Already Exists

**Evidence:** The full send-wait-extract cycle is already proven:
1. `SendMessageAsync(sessionID, content, model)` — inject text into a session (`pkg/opencode/message.go:231`)
2. SSE idle-polling detects when the expert finishes (`pkg/opencode/message.go:269-374`, `SendMessageWithStreaming` implements this exact pattern)
3. `GetMessages(sessionID)` + `ExtractRecentText(messages, lines)` extracts the response (`pkg/opencode/message.go:37-228`)

**Source:** `pkg/opencode/message.go`, `cmd/orch/send_cmd.go:77-113`

**Significance:** The expert side needs zero new infrastructure. We compose existing OpenCode primitives. The only new work is the relay loop and questioner API client.

---

### Finding 3: OAuth Tokens Likely Work for Messages API

**Evidence:** `pkg/anthropic/anthropic.go` already constructs `Bearer` + OAuth token requests with `anthropic-beta: oauth-2025-04-20` header. The OAuth beta header explicitly enables OAuth tokens for API endpoints. Account management (`pkg/account/`) provides token refresh.

**Source:** `pkg/anthropic/anthropic.go:23-46`, `pkg/account/oauth.go`

**Significance:** We can likely call `POST /v1/messages` with the existing OAuth token and beta headers. Sonnet is accessible via API (no fingerprinting block). Fallback: `ANTHROPIC_API_KEY` env var for users with separate API keys.

---

### Finding 4: Workspace and Event Infrastructure Applies Directly

**Evidence:** The dialogue needs: workspace directory, beads issue tracking, event logging, SYNTHESIS.md. All exist:
- `spawn.GenerateWorkspaceName()` — creates workspace names
- `spawn.WriteContext()` — creates workspace with `.beads_id`, manifest, etc.
- `events.NewLogger().Log()` — writes to `~/.orch/events.jsonl`
- SYNTHESIS.md template — exists at `.orch/templates/SYNTHESIS.md`

**Source:** `pkg/spawn/config.go:397-443`, `pkg/spawn/context.go:118-196`, `pkg/events/logger.go`

**Significance:** Dialogue workspaces follow the same conventions as spawn workspaces. `orch status` and `orch complete` work without modification.

---

## Synthesis

**Key Insights:**

1. **Three enforced asymmetries create the value** — The original experiment worked because three asymmetries were real, not suggested: (a) knowledge asymmetry (context-loaded expert vs context-free questioner), (b) capability asymmetry (tools vs no tools), (c) pressure asymmetry (questions force articulation). All three must be structurally enforced via architecture, not prompts.

2. **The orch binary IS the orchestrator** — Unlike spawn (fire-and-forget), dialogue requires the orch binary to actively manage a turn-based loop. This is a new execution pattern: orch as a real-time message relay, not just a session launcher. The binary holds the questioner's conversation state and shuttles messages between API calls and OpenCode sessions.

3. **Compose, don't rebuild** — The expert side needs zero new infrastructure. `SendMessageAsync` → SSE poll → `GetMessages` → extract text is already a proven path. The only genuinely new code is: (a) a thin Anthropic Messages API client for the questioner, and (b) the relay loop that connects them.

**Answer to Investigation Question:**

`orch dialogue` should be a Go command that manages a turn-based relay between two asymmetric agents:
- **Questioner**: Direct Anthropic Messages API (Sonnet by default). No tools, no CLAUDE.md, no project context. Conversation state held in Go memory. Structurally cannot grep.
- **Expert**: Standard OpenCode session with full tools and project context. Messages injected via `SendMessageAsync`, responses extracted via `GetMessages` after SSE idle detection.
- **Relay Loop**: The orch binary manages turns. Questioner asks → orch relays to expert → expert responds → orch relays back → loop until questioner proposes.
- **Termination**: Questioner's system prompt instructs natural transition to `## PROPOSAL` when understanding is sufficient. Hard cap via `--max-turns` flag.
- **Artifacts**: Full transcript saved to workspace. Proposal extracted as design investigation. SYNTHESIS.md produced for `orch complete`.

---

## Structured Uncertainty

**What's tested:**

- ✅ `SendMessageAsync` + SSE idle poll + `GetMessages` extracts expert responses (verified: this is exactly what `SendMessageWithStreaming` does in `message.go:269-374`)
- ✅ Workspace/beads/events infrastructure works for new command types (verified: `spawn`, `clean`, `complete` all use these)
- ✅ `pkg/anthropic/` has the OAuth header setup for Anthropic API calls (verified: read `anthropic.go:23-46`)

**What's untested:**

- ⚠️ OAuth tokens from Max subscription work with `/v1/messages` endpoint (not verified — only used for `/api/oauth/usage` today)
- ⚠️ Sonnet API calls don't hit fingerprinting blocks (assumed based on Opus-only fingerprinting, not tested)
- ⚠️ Turn-based relay latency is acceptable for natural-feeling dialogue (depends on model response time)

**What would change this:**

- If OAuth tokens DON'T work with Messages API → need `ANTHROPIC_API_KEY` env var or a different auth approach
- If Sonnet quality is insufficient for design questioning → try Gemini Flash (cross-model perspective)
- If expert response extraction is unreliable → need structured response format instead of free-text extraction

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Direct API for questioner, OpenCode for expert | architectural | Introduces new API client pattern, cross-component communication |
| OAuth token auth with API key fallback | implementation | Uses existing infrastructure, reversible |
| Workspace conventions match spawn | implementation | Follows established patterns |
| New event types for dialogue tracking | implementation | Extends existing events system |

### Recommended Approach ⭐

**Asymmetric Relay Architecture** — Direct Anthropic API for structurally-toolless questioner, OpenCode session for tool-equipped expert, orch binary as real-time message relay.

**Why this approach:**
- Structural enforcement of asymmetries (API level, not prompt level)
- Composes existing infrastructure (zero new expert-side code)
- Independent of OpenCode for the questioner (escape hatch principle)
- Observable via dashboard (workspace + beads + events)

**Trade-offs accepted:**
- Requires API access (OAuth token or API key) — but the questioner uses Sonnet which is cheap (~$0.50-1.00 per dialogue)
- New `pkg/dialogue/` package — but dialogue is a genuinely new concept, not a variation of spawn
- Orch binary blocks during dialogue — but `--background` flag can run it in a tmux window

**Implementation sequence:**

1. **Phase 1: Core API client** (`pkg/dialogue/api.go`) — Anthropic Messages API client using OAuth tokens. Test with simple completion call. This validates the auth approach early.

2. **Phase 2: Relay loop** (`pkg/dialogue/relay.go`) — Turn-based loop: API call → extract questioner message → send to expert → wait for idle → extract expert response → loop. Termination detection via `## PROPOSAL` marker.

3. **Phase 3: CLI command** (`cmd/orch/dialogue_cmd.go`) — `orch dialogue "topic" [flags]`. Flags: `--max-turns`, `--questioner-model`, `--expert-model`, `--issue`, `--background`.

4. **Phase 4: Artifact production** (`pkg/dialogue/artifacts.go`) — Transcript to workspace, proposal extraction, SYNTHESIS.md generation, event logging.

### Alternative Approaches Considered

**Option B: Both sides via OpenCode sessions**
- **Pros:** No new API client needed, simpler auth
- **Cons:** Cannot structurally prevent tool use on questioner side; CLAUDE.md contaminates "fresh eyes"; violates key property
- **When to use instead:** If OAuth/API auth proves infeasible, this is the fallback — use tool permission denials and stripped context directory

**Option C: Claude Code CLI for questioner**
- **Pros:** Uses existing binary
- **Cons:** CLI is designed for interactive TUI use; programmatic message relay through CLI is fragile; still loads CLAUDE.md
- **When to use instead:** If a `claude --print --no-tools` mode exists or is added upstream

**Option D: File-based IPC between two agents**
- **Pros:** Both sides autonomous, agents read/write shared files
- **Cons:** Over-engineered; polling latency; no structural tool restriction; complex coordination
- **When to use instead:** Never — this is the least-good option for this use case

**Rationale for recommendation:** Option A is the only approach that structurally enforces all three asymmetries. The API client is thin (~100 lines), and the expert side requires zero new code. The alternatives all compromise on the key property that made the original experiment work.

---

### Implementation Details

**What to implement first:**
- `pkg/dialogue/api.go` — thin Anthropic Messages API client (conversation-stateful, no tools)
- Auth validation — confirm OAuth tokens work with `/v1/messages`
- If auth fails → add `ANTHROPIC_API_KEY` env var fallback immediately

**System prompts (critical to quality):**

Questioner system prompt:
```
You are a design consultant brought in with fresh eyes. You know NOTHING about the codebase, architecture, or implementation. Your job is to:

1. UNDERSTAND the problem by asking probing questions
2. BUILD a mental model of the system through the expert's answers
3. CHALLENGE assumptions you notice in the expert's explanations
4. PROPOSE a concrete design when you have sufficient understanding

Rules:
- Ask ONE question at a time (focused, specific)
- Build on previous answers (show you're learning)
- When something sounds overly complex, ask "why?"
- When you've built sufficient understanding, transition to a proposal

When ready to propose, start your message with "## PROPOSAL" followed by your design.
Your proposal should include: approach, tradeoffs, and implementation steps.
```

Expert system prompt (injected as first message):
```
You are the expert side of a design dialogue about: [TOPIC]

You have FULL access to the codebase via tools (Read, Grep, Glob, Bash).
The other participant has NO access — they can only think and ask questions.

Your job:
- Answer questions with SPECIFIC evidence (file paths, line numbers, function names)
- Be concise but precise
- When asked "why?", explain the reasoning and tradeoffs
- Correct misconceptions immediately
- Don't volunteer information beyond what's asked (let them drive)

Topic context: [TOPIC DESCRIPTION]
```

**Things to watch out for:**
- ⚠️ OAuth token expiry during long dialogues — need token refresh within the relay loop
- ⚠️ Expert session going idle for wrong reasons (tool permissions prompt, rate limit) — need to distinguish "idle because done responding" from "idle because waiting for user"
- ⚠️ Context window limits on questioner side — 20 turns of dialogue could hit 100k+ tokens. Use Sonnet's 200k window but monitor.
- ⚠️ Expert responses that include tool output (file contents, grep results) will be verbose — may need summarization before relaying to questioner

**Areas needing further investigation:**
- OAuth token compatibility with Messages API (`/v1/messages`) — needs empirical test
- Optimal `--max-turns` default — 15-20 seems right from the original experiment
- Whether Gemini Flash makes a better questioner (genuinely different training, different blind spots)

**Success criteria:**
- ✅ `orch dialogue "topic"` produces a design proposal within 15-25 turns
- ✅ Questioner never uses tools (structural enforcement via API)
- ✅ Expert responses cite specific code (files, line numbers)
- ✅ Transcript + proposal saved to workspace
- ✅ `orch complete` works on dialogue workspaces

---

## Architecture Diagram

```
orch dialogue "design topic"
    │
    ├── Create workspace: .orch/workspace/og-dialogue-topic-08feb-XXXX/
    ├── Create beads issue (or --issue)
    │
    ├── QUESTIONER (Direct API)          EXPERT (OpenCode Session)
    │   ├── Model: Sonnet                ├── Model: Opus (or configured)
    │   ├── Auth: OAuth / API key        ├── Full tools (Read, Grep, etc.)
    │   ├── No tools (API has none)      ├── CLAUDE.md loaded
    │   ├── No project context           ├── Project context
    │   └── State: in-memory []Message   └── State: OpenCode session
    │
    └── RELAY LOOP (orch binary):
        1. Questioner asks (API call) ──────────────────┐
        2. Relay question to expert (SendMessageAsync)  │
        3. Wait for expert idle (SSE polling)            │ repeat
        4. Extract expert response (GetMessages)        │ until
        5. Relay response to questioner (add to history)│ PROPOSAL
        6. Get questioner's next message (API call) ────┘
        7. Detect ## PROPOSAL → enter proposal phase
        8. Send proposal to expert for review
        9. Save transcript + proposal + SYNTHESIS.md
```

## CLI Design

```
orch dialogue "design auth system for multi-tenant" [flags]

Flags:
  --max-turns int          Maximum conversation turns (default 20)
  --questioner-model str   Model for fresh-eyes questioner (default "sonnet")
  --expert-model str       Model for knowledgeable expert (default from config)
  --issue str              Beads issue ID to attach to
  --background             Run in background (tmux window)
  --topic-context str      Additional context for the questioner (optional)
  --transcript-only        Save transcript without creating investigation
  --verbose                Print turns to stdout as they happen

Output:
  workspace/DIALOGUE_TRANSCRIPT.md   Full conversation
  workspace/SYNTHESIS.md             Summary for orch complete
  .kb/investigations/...             Design investigation (from proposal)
```

---

## File Targets

| File | Purpose |
|------|---------|
| `pkg/dialogue/api.go` | Anthropic Messages API client (conversation-stateful) |
| `pkg/dialogue/relay.go` | Turn-based relay loop |
| `pkg/dialogue/transcript.go` | Transcript formatting and artifact production |
| `pkg/dialogue/prompts.go` | System prompts for questioner and expert |
| `cmd/orch/dialogue_cmd.go` | CLI command with flags |

Estimated: ~500-700 lines of new Go code across 5 files.

---

## References

**Files Examined:**
- `pkg/opencode/message.go` — SendMessageAsync, SSE streaming, ExtractRecentText
- `pkg/opencode/client.go` — Client interface, session creation
- `pkg/opencode/types.go` — Message/Session types
- `pkg/anthropic/anthropic.go` — OAuth header setup for API calls
- `pkg/spawn/config.go` — SpawnConfig struct, workspace naming
- `pkg/spawn/context.go` — WriteContext, workspace creation
- `cmd/orch/send_cmd.go` — Existing send command (message relay pattern)
- `cmd/orch/wait.go` — Phase polling, beads ID resolution
- `cmd/orch/spawn_cmd.go` — Spawn flow, backend selection

**Related Artifacts:**
- **Decision:** "Two-model experiments worth running selectively" — validates cross-model design critique
- **Decision:** "Dual-mode architecture (tmux for visual, HTTP for programmatic)" — dialogue uses both modes
- **Model:** Spawn Architecture — workspace/beads/events patterns apply directly
- **Model:** Agent Lifecycle State Model — SSE idle-polling is proven for response detection
- **Guide:** Spawned Orchestrator Pattern — hierarchical orchestration reference
- **Principles:** Compose Over Monolith, Escape Hatches, Authority is Scoping, Perspective is Structural

---

## Investigation History

**2026-02-08 ~09:00:** Investigation started
- Initial question: How to automate the asymmetric two-agent design dialogue pattern?
- Context: Dylan manually relayed messages between Claude web (toolless) and orchestrator (tooled). Produced better design in 30min than tool-equipped agents in 2hr.

**2026-02-08 ~09:30:** Codebase exploration complete
- Found: Expert-side infrastructure fully exists (SendMessageAsync + SSE poll + GetMessages)
- Found: pkg/anthropic/ has OAuth headers for API calls
- Found: Workspace/beads/events infrastructure applies directly

**2026-02-08 ~10:00:** 5 forks identified and navigated
- Fork 1: Questioner implementation → Direct API (structural tool restriction)
- Fork 2: Expert implementation → OpenCode session (existing infrastructure)
- Fork 3: Orchestration → orch binary relay loop (simplest, most reliable)
- Fork 4: Termination → PROPOSAL marker + max-turns safety valve
- Fork 5: Artifacts → Transcript + investigation + SYNTHESIS.md

**2026-02-08 ~10:30:** Investigation completed
- Status: Complete
- Key outcome: Asymmetric relay architecture recommended — direct API for questioner, OpenCode for expert, orch binary as relay.
