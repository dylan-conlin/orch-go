<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Bidirectional communication between orchestrators and workers currently exists in fragmentary form (5 existing mechanisms) with 5 key gaps: dead session detection, structured response visibility, question notification, mid-flight course correction, and bidirectional progress acknowledgment.

**Evidence:** Analyzed existing implementations: `orch send` (one-way async), `orch question` (polling-based), `bd comment` (one-way status updates), SSE events (OpenCode-level, not worker-level), and Phase reporting (worker→orchestrator only).

**Knowledge:** The gaps fall into two categories: (1) Session liveness/health detection and (2) Interactive dialogue patterns. The former needs infrastructure; the latter needs protocol definition. CLI is correct for orchestrators/scripts; MCP is not needed for this.

**Next:** Create epic with 6 children addressing each gap. Implement in priority order: 1) Session health detection, 2) Question notification, 3) Response visibility, 4) Course correction protocol.

**Confidence:** High (85%) - design grounded in concrete code analysis and existing constraint "MCP for agent-internal use, CLI for orchestrator/scripts/humans"

---

# Investigation: Orchestrator-Worker Bidirectional Communication Interaction Patterns

**Question:** What interaction patterns should exist between orchestrators and workers? What are the current gaps and how should they be addressed?

**Started:** 2026-01-05
**Updated:** 2026-01-05
**Owner:** Design session agent
**Phase:** Synthesizing
**Next Step:** Produce epic with child issues
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Problem Framing

The orchestrator-worker relationship is inherently bidirectional:
- **Orchestrator → Worker:** Send instructions, provide context, course-correct
- **Worker → Orchestrator:** Report progress, ask questions, request guidance

Current tooling supports these directions unevenly. The spawn context captures initial instructions well, but ongoing interaction during agent execution has gaps.

### Identified Gaps (from task description)

1. **Session health detection** - How to know if a session is dead/unresponsive
2. **Response visibility** - `orch send` works but no structured way to view responses
3. **Worker questions** - `orch question` polls but doesn't notify orchestrator
4. **Course correction** - No protocol for mid-flight redirection
5. **Progress acknowledgment** - `bd comment` is one-way (worker→orchestrator)

---

## Findings

### Finding 1: Five Existing Communication Mechanisms

**Evidence:** The codebase has five mechanisms for orchestrator-worker communication:

| Mechanism | Direction | Type | Implementation |
|-----------|-----------|------|----------------|
| `orch send <id> "msg"` | O→W | CLI | `cmd/orch/send_cmd.go` - sends via OpenCode API or tmux fallback |
| `orch question <id>` | W→O (poll) | CLI | `cmd/orch/question_cmd.go` - extracts AskUserQuestion from session |
| `bd comment <id> "msg"` | W→O | CLI | Beads comments - persistent, searchable |
| SSE events | W→O (stream) | HTTP | `serve_agents_events.go` - OpenCode event proxy |
| Phase reporting | W→O | Protocol | `Phase: X` in bd comments, parsed by `verify.ParsePhaseFromComments` |

**Source:** 
- `cmd/orch/send_cmd.go:51-74` - runSend with API/tmux dual-path
- `cmd/orch/question_cmd.go:35-119` - runQuestion with workspace→API→tmux search
- `pkg/verify/beads.go` - GetPhaseStatus, ParsePhaseFromComments

**Significance:** The building blocks exist but are disconnected. No unified "conversation view" or notification layer ties them together.

---

### Finding 2: Session Health Detection Gap

**Evidence:** Current status detection relies on:
1. OpenCode session "processing" flag (is API call in flight?)
2. Session update timestamp (idle > 30 min = "at-risk")
3. Phase: Complete comment (definitive completion signal)

**Missing:**
- No heartbeat mechanism for long-running tasks
- No detection of crashed/hung sessions (process alive but Claude unresponsive)
- No distinction between "thinking" (healthy idle) vs "stuck" (unhealthy idle)

The `status-dashboard.md` guide explicitly notes: "Only Phase: Complete indicates actual completion. Idle status is ambiguous."

**Source:** 
- `cmd/orch/status_cmd.go:261-269` - isProcessing check via client.IsSessionProcessing
- `.kb/guides/status-dashboard.md:109-113` - "Agent shows as idle but is actually working"

**Significance:** Orchestrators cannot reliably distinguish between healthy-idle and dead sessions. This forces manual probing via `orch send "status?"`.

---

### Finding 3: Question Notification Gap

**Evidence:** The `orch question` command must be explicitly invoked - it polls on demand:

```go
// question_cmd.go:35-36
func runQuestion(beadsID string) error {
    client := opencode.NewClient(serverURL)
```

There is no:
- Push notification when agent asks a question
- Integration with SSE events for question detection
- Dashboard indicator for "agent has pending question"

The SSE event stream (`serve_agents_events.go:17-91`) proxies OpenCode events but doesn't parse for AskUserQuestion patterns.

**Source:** 
- `cmd/orch/question_cmd.go` - Pull-based, requires beadsID
- `cmd/orch/serve_agents_events.go` - Forwards raw events, no question parsing
- `pkg/question/question.go` - Extraction logic exists but not integrated into event stream

**Significance:** Questions go unanswered until orchestrator polls. Agents block waiting for responses. This creates hidden queues.

---

### Finding 4: Response Visibility Gap

**Evidence:** `orch send` has two modes:

1. `--async=true` (default): Fire-and-forget, no response visibility
   ```go
   // send_cmd.go:96-102
   if sendAsync {
       if err := client.SendMessageAsync(sessionID, message, ""); err != nil {
           return fmt.Errorf("failed to send message asynchronously: %w", err)
       }
       fmt.Printf("✓ Message sent to session %s (via API)\n", sessionID)
       return nil
   }
   ```

2. `--async=false`: Blocking stream to stdout
   ```go
   // send_cmd.go:105-107
   if err := client.SendMessageWithStreaming(sessionID, message, os.Stdout); err != nil {
       return fmt.Errorf("failed to send message: %w", err)
   }
   ```

**Missing:**
- Historical view of send/response pairs
- Dashboard integration for message history
- Correlation between orchestrator sends and agent reactions

**Source:** `cmd/orch/send_cmd.go:51-114`

**Significance:** Orchestrators send messages but have no persistent record of the conversation. Events are logged (`events.jsonl`) but not surfaced.

---

### Finding 5: Course Correction Protocol Gap

**Evidence:** No defined protocol for mid-flight course correction. Current options:

1. `orch send` - sends message but agent may not prioritize/acknowledge
2. `orch abandon` - kills the agent entirely (too heavy)
3. Manual tmux intervention - breaks headless workflow

**Missing:**
- Signal priority levels (normal message vs urgent intervention)
- Agent acknowledgment protocol for course corrections
- Structured "redirect" command distinct from "additional context"

**Source:** 
- `cmd/orch/send_cmd.go` - no priority parameter
- `cmd/orch/abandon_cmd.go` - binary kill, no soft redirect
- Prior kn decision: "orch send vs spawn: use task relatedness not session age"

**Significance:** Orchestrators can't effectively redirect agents mid-task. Either the message gets lost in context, or the agent is killed entirely.

---

### Finding 6: Progress Acknowledgment is One-Way

**Evidence:** `bd comment` is used by workers to report progress:
```
bd comment <id> "Phase: Planning - Starting analysis"
```

But orchestrators have no equivalent "I received your update" signal. Workers don't know if:
- Orchestrator saw the update
- Orchestrator approved/disapproved the direction
- Orchestrator wants them to continue or pause

**Source:** SPAWN_CONTEXT.md template includes `bd comment` for worker→orchestrator, but nothing for reverse.

**Significance:** Workers operate in the dark about orchestrator engagement. This creates uncertainty during long tasks.

---

## Synthesis

### Key Insights

1. **Two distinct gap categories:**
   - **Infrastructure gaps** (session health, question notification) - Need new detection/notification systems
   - **Protocol gaps** (course correction, acknowledgment) - Need defined interaction patterns

2. **CLI is the correct interface** - Prior decision: "MCP for agent-internal use, CLI for orchestrator/scripts/humans". Orchestrators use CLI. New mechanisms should be CLI commands.

3. **SSE/events layer exists but is underutilized** - The `serve_agents_events.go` already proxies OpenCode SSE. Question detection could be integrated here.

4. **Beads comments are the persistent record** - Phase reporting works via beads. Bidirectional acknowledgment could use the same channel.

5. **Dashboard is secondary to CLI** - Per constraint "Dylan doesn't interact with dashboard directly - orchestrator uses Glass for all browser interactions". Dashboard visibility is for orchestrators using Glass, not humans.

### Proposed Interaction Patterns

| Pattern | Direction | Mechanism | When |
|---------|-----------|-----------|------|
| **Progress Report** | W→O | `bd comment <id> "Phase: X"` | Phase transitions |
| **Question** | W→O | AskUserQuestion + notification | Agent needs input |
| **Answer** | O→W | `orch send <id> "answer"` | Responding to question |
| **Course Correct** | O→W | `orch redirect <id> "new direction"` | Mid-flight redirection |
| **Acknowledge** | O→W | `orch ack <id>` or via bd comment | Confirming receipt |
| **Health Check** | O↔W | `orch probe <id>` | Detecting dead sessions |

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch send` works for API and tmux modes (verified: code review of dual-path logic)
- ✅ `orch question` extracts AskUserQuestion pattern (verified: pkg/question tests exist)
- ✅ Phase reporting via bd comment works (verified: used in production)
- ✅ SSE event proxy functions (verified: serve_agents_events.go implementation)

**What's untested:**

- ⚠️ Question notification via SSE (proposed, not implemented)
- ⚠️ Session health heartbeat mechanism (proposed, not implemented)
- ⚠️ `orch redirect` command semantics (proposed, needs design)
- ⚠️ Bidirectional acknowledgment protocol (proposed, needs design)

**What would change this:**

- If agents can't reliably receive/process `orch send` messages → need different delivery mechanism
- If SSE events don't include enough info for question detection → need OpenCode API changes
- If orchestrators don't actually need notification (polling is fine) → deprioritize notification system

---

## Implementation Recommendations

### Recommended Approach ⭐

**Incremental Enhancement** - Add capabilities to existing mechanisms rather than building new systems.

**Why this approach:**
- Leverages existing CLI (`orch send`, `orch question`)
- Extends SSE events already proxied via `serve_agents_events.go`
- Uses beads as existing persistent conversation record
- Minimal new code, maximum reuse

**Trade-offs accepted:**
- Not building a full "conversation thread" view (defer to dashboard v2)
- Not implementing real-time push notifications to orchestrator terminal (SSE + dashboard is enough)

**Implementation sequence:**

1. **Session Health Detection** (highest priority)
   - Add `orch probe <id>` - sends health check, waits for response
   - Add session "last activity" with source (tool call vs thinking)
   - Dashboard: show "unresponsive" after probe timeout

2. **Question Notification** (high priority)
   - Parse incoming SSE events for AskUserQuestion in `serve_agents_events.go`
   - Emit `event: question` SSE event with parsed question
   - Dashboard: show "pending question" badge on agent card

3. **Response Visibility** (medium priority)
   - Log send/response pairs to events.jsonl (already done partially)
   - Add `orch history <id>` to view conversation history
   - Dashboard: show message history in agent detail view

4. **Course Correction** (medium priority)
   - Add `orch redirect <id> "instruction"` as alias/mode of send with priority flag
   - Agent SPAWN_CONTEXT.md: "If you receive a REDIRECT message, acknowledge and adjust"
   - Consider: interrupt current tool call vs wait for idle

5. **Acknowledgment** (lower priority)
   - Add `orch ack <id>` to confirm orchestrator saw phase report
   - Worker can check for ack before proceeding (optional protocol)
   - Alternative: bd comment from orchestrator is sufficient ack

### Alternative Approaches Considered

**Option B: Build MCP server for communication**
- **Pros:** Structured tool calls, schema validation
- **Cons:** Violates "CLI for orchestrator" decision; adds complexity
- **When to use instead:** If agents need to call orchestrator (they don't - workers shouldn't escalate via MCP)

**Option C: Build custom notification service**
- **Pros:** Real-time push to orchestrator terminal
- **Cons:** New infrastructure, daemon complexity
- **When to use instead:** If dashboard isn't sufficient and CLI-based polling is too slow

**Rationale for recommendation:** CLI + SSE + dashboard covers all use cases. Push notifications are a "nice to have" that can be deferred. The existing architecture is sound; it needs integration, not replacement.

---

### Implementation Details

**What to implement first:**
1. `orch probe <id>` - Critical for detecting dead sessions
2. Question detection in SSE - Unlocks notification without new infrastructure
3. `orch history <id>` - Visibility into what's been communicated

**Things to watch out for:**
- ⚠️ Rate limiting on OpenCode API - probes should be infrequent
- ⚠️ Event log size - `events.jsonl` may grow large; needs rotation
- ⚠️ Agent context pollution - redirect messages add to context; keep brief
- ⚠️ tmux fallback complexity - some mechanisms need API; tmux agents may have gaps

**Areas needing further investigation:**
- How do other orchestration tools (Temporal, Airflow) handle worker health?
- Should acknowledgment be explicit or implicit (no ack = continue)?
- How to handle multiple orchestrators talking to same agent?

**Success criteria:**
- ✅ Orchestrator can detect unresponsive agents within 5 minutes
- ✅ Pending questions are visible without explicit polling
- ✅ Course corrections are acknowledged by agents
- ✅ No regression in existing `orch send` / `orch question` functionality

---

## References

**Files Examined:**
- `cmd/orch/send_cmd.go` - Send command implementation
- `cmd/orch/question_cmd.go` - Question extraction command
- `cmd/orch/status_cmd.go` - Status and health detection
- `cmd/orch/serve_agents_events.go` - SSE event proxy
- `pkg/verify/check.go` - Completion verification
- `pkg/question/question.go` - Question extraction logic
- `.kb/guides/status-dashboard.md` - Status determination logic

**Commands Run:**
```bash
# Context gathering
kb context "orch send"
kb context "orch question"
kb context "bidirectional"

# Code exploration
glob **/cmd/orch/*.go
```

**External Documentation:**
- Prior decision: "MCP for agent-internal use, CLI for orchestrator/scripts/humans"
- Prior investigation: `.kb/investigations/2025-12-24-inv-explore-orch-send-vs-spawn.md`

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-25-design-orchestrator-completion-lifecycle-two.md` - Completion lifecycle design
- **Guide:** `.kb/guides/status-dashboard.md` - Status determination patterns

---

## Investigation History

**[2026-01-05 14:00]:** Investigation started
- Initial question: What interaction patterns should exist for orchestrator-worker communication?
- Context: Gaps identified in spawn context - 5 areas needing design

**[2026-01-05 14:30]:** Context gathering complete
- Analyzed 5 existing mechanisms (send, question, bd comment, SSE, Phase)
- Identified 5 gaps (health, notification, visibility, correction, acknowledgment)
- Decision: CLI-based approach (per existing constraint)

**[2026-01-05 15:00]:** Investigation complete
- Status: Complete
- Key outcome: Incremental enhancement approach recommended; epic with 6 children proposed
