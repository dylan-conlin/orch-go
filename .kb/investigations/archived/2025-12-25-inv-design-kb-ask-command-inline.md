<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** `kb ask` should be a kb-cli command that synthesizes context into answers, with strict provenance enforcement (no context = no answer).

**Evidence:** Examined kb-cli context.go (650 lines), orch-go main.go (existing `orch ask` is alias for `send`), principles.md (Provenance, Session Amnesia, Gate Over Remind). kb-cli already has all retrieval infrastructure; missing only LLM synthesis layer.

**Knowledge:** The distinction between retrieval (`kb context`) and synthesis (`kb ask`) maps cleanly to existing architecture. `--force` flag should NOT exist - Provenance principle prohibits ungrounded answers. Tiered save (--kn quick / --save full) bridges ephemeral to durable knowledge.

**Next:** Implement in kb-cli as new command - feature item feat-013 created.

**Confidence:** High (85%) - Design is well-grounded in principles and existing architecture; main uncertainty is optimal LLM integration approach (OpenCode API vs direct SDK vs CLI).

---

# Investigation: Design Kb Ask Command for Inline Mini-Investigations

Question: Should `kb ask` be a kb-cli feature or orch feature? What's the interface? How does it differ from `kb context`?

## Problem Framing

- **Design Question:** How to design a `kb ask` command for inline mini-investigations that provides synchronous answers without spawn overhead.
- **Success Criteria:**
    - Provides fast, inline answers using existing knowledge
    - Composes existing `kb context` effectively (not duplicates it)
    - Offers optional mechanism to save answers as investigation or kn entry
    - Handles no-context case gracefully (with explicit guidance, not hallucination)
    - Adheres to Meta-Orchestration Principles (Provenance, Session Amnesia, Evidence Hierarchy)
- **Constraints:**
    - Must be ephemeral by default (speed over persistence)
    - Must leverage existing `kb context` retrieval
    - Needs tiered bridge to investigation workflow
    - Performance is critical (avoid spawn overhead)
    - **Provenance is non-negotiable** - no answer without external grounding
- **Scope:**
    - In: Design of `kb ask` command, interface, architecture decision
    - Out: Full implementation, detailed UX beyond CLI

## Findings

### Finding 1: `kb context` Already Has Complete Retrieval Infrastructure

**Evidence:** `kb-cli/cmd/kb/context.go` (652 lines) provides:
- Unified search across kn entries (constraints, decisions, attempts, questions) and kb artifacts (investigations, decisions, guides)
- Both local and global (`--global`) search
- JSON output format for machine consumption
- Stale detection for investigations linked to closed beads issues

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/context.go:128-183`

**Significance:** `kb ask` should NOT duplicate this retrieval logic. It should compose `kb context` and add LLM synthesis on top. The pattern is: `kb context` → retrieval, `kb ask` → retrieval + synthesis.

---

### Finding 2: orch-go's `orch ask` Is Already Taken (Alias for Send)

**Evidence:** In `cmd/orch/main.go:265-275`:
```go
var askCmd = &cobra.Command{
    Use:   "ask [identifier] [prompt]",
    Short: "Send a message to an existing session (alias for send)",
```

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:265`

**Significance:** The `ask` command semantics differ between tools:
- `orch ask <session-id> <message>` = send message to running agent (communication)
- `kb ask <question>` = synthesize answer from knowledge base (RAG)

This name collision is acceptable - different tools, different domains. Command naming follows tool context.

---

### Finding 3: Provenance Principle Prohibits `--force` Flag

**Evidence:** From `~/.kb/principles.md`:
> "Every conclusion must trace to something outside the conversation."
> "The failure mode: Closed loops feel like progress. Each step feels validated. But the chain references itself - nothing outside the conversation."

**Source:** `/Users/dylanconlin/.kb/principles.md:14-41`

**Significance:** A `--force` flag that allows LLM answers without kb context would violate the foundational Provenance principle. When no context is found, the system MUST:
1. Inform the user: "No relevant knowledge found"
2. Suggest alternatives: rephrase, spawn investigation
3. NOT generate an ungrounded answer

---

### Finding 4: Gate Over Remind Supports Tiered Save Options

**Evidence:** From `~/.kb/principles.md`:
> "Gates make capture unavoidable... Cannot `/exit` without kn check → capture happens"

**Source:** `/Users/dylanconlin/.kb/principles.md:145-159`

**Significance:** Save options should be explicit flags (gates), not post-hoc prompts (reminders):
- `--kn` → Create quick kn decide entry
- `--save` → Create full investigation artifact
- Neither → Ephemeral (default)

This respects the principle while allowing speed for ephemeral queries.

---

## Synthesis

**Key Insights:**

1. **Retrieval vs Synthesis Distinction** - `kb context` = retrieval (what do we know?), `kb ask` = synthesis (what does it mean?). The command should be in kb-cli because it's fundamentally about knowledge synthesis, not agent coordination.

2. **Provenance as Hard Gate** - The `--force` flag should not exist. Answers without grounding are closed loops. When no context, guide user to spawn investigation (the principled path to new knowledge).

3. **Tiered Persistence Model** - Three tiers: ephemeral (default, fast), kn entry (quick capture), investigation (full externalization). Each tier has appropriate overhead.

**Answer to Design Question:**

`kb ask` should be a **kb-cli command** that:
1. Calls `kb context` for retrieval
2. Sends context + question to LLM for synthesis
3. Streams answer to terminal
4. Optionally saves via `--kn` or `--save` flags

It should NOT be an orch command because:
- It's about knowledge synthesis, not agent coordination
- kb-cli already owns the retrieval infrastructure
- Keeps tools focused (Compose Over Monolith)

**Interface:**
```bash
kb ask "How does authentication work?"              # Ephemeral answer
kb ask "How does authentication work?" --kn         # Saves as kn decide
kb ask "How does authentication work?" --save       # Saves as investigation
kb ask "How does authentication work?" --global     # Search all projects
kb ask "How does authentication work?" --model flash  # Override model
```

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**
- Design is well-grounded in existing architecture and principles
- Clear separation from existing commands
- Main uncertainty is LLM integration approach

**What's certain:**
- ✅ Should be kb-cli command (kb context lives there)
- ✅ No `--force` flag (Provenance principle)
- ✅ Tiered save options (Gate Over Remind principle)
- ✅ Ephemeral by default (speed requirement)

**What's uncertain:**
- ⚠️ LLM integration approach (OpenCode API vs direct SDK vs Claude CLI)
- ⚠️ Optimal context window management (dynamic N)
- ⚠️ Cost implications of frequent LLM calls

**What would increase confidence to Very High (95%):**
- Prototype LLM integration in kb-cli
- Benchmark response latency with different approaches
- Test context window edge cases

---

## Implementation Recommendations

### Recommended Approach ⭐

**kb-cli Command with OpenCode Integration** - Implement `kb ask` as kb-cli command that uses `opencode` CLI for LLM synthesis (same approach orch-go uses).

**Why this approach:**
- Consistent with existing ecosystem (OpenCode manages LLM credentials)
- Avoids duplicating LLM configuration in kb-cli
- Can reuse model selection from OpenCode accounts
- Streams output naturally (OpenCode already handles streaming)

**Trade-offs accepted:**
- Dependency on OpenCode installation
- Slightly higher latency than direct SDK
- Acceptable because: OpenCode is already required for orchestration

**Implementation sequence:**
1. Add `ask` command to kb-cli with `kb context` integration
2. Pipe context + question to `opencode prompt` (non-interactive mode)
3. Add `--kn` and `--save` flags for persistence
4. Add `--model` flag for model override

### Alternative Approaches Considered

**Option B: Direct Anthropic SDK Integration**
- **Pros:** Lower latency, no OpenCode dependency
- **Cons:** Duplicate credential management, additional dependency
- **When to use instead:** If kb-cli needs to work without OpenCode

**Option C: orch-go Command**
- **Pros:** Uses existing OpenCode client code
- **Cons:** Wrong tool (orch = coordination, kb = knowledge)
- **When to use instead:** Never - violates Compose Over Monolith

**Rationale for recommendation:** OpenCode integration aligns with existing patterns and avoids introducing new credential/model management into kb-cli.

---

### Implementation Details

**What to implement first:**
- Basic `kb ask` command with `kb context` piped to OpenCode
- No-context gate (refuse to answer without context)
- Streaming output to terminal

**Things to watch out for:**
- ⚠️ Context window limits - need dynamic N based on token count
- ⚠️ OpenCode not running - graceful error with instructions
- ⚠️ Model availability - respect OpenCode account settings

**Areas needing further investigation:**
- How to compute optimal N for context window
- Whether to cache context between rapid-fire questions
- Integration testing with different OpenCode configurations

**Success criteria:**
- ✅ `kb ask "question"` returns synthesized answer from kb context
- ✅ Empty context → helpful error, not hallucination
- ✅ `--kn` creates kn entry with answer
- ✅ `--save` creates investigation with answer + sources

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/context.go` - Core retrieval implementation
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/search.go` - Search patterns for kb artifacts
- `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go` - Existing ask command (alias for send)
- `/Users/dylanconlin/.kb/principles.md` - Foundational principles (Provenance, Gate Over Remind)

**Commands Run:**
```bash
# Check for LLM integrations in kb-cli
grep -rn "LLM\|claude\|anthropic" kb-cli/cmd/kb/

# Check existing ask command in orch-go
grep -n "askCmd" orch-go/cmd/orch/main.go
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-kb-reflect-command-interface.md` - Similar kb-cli command design
- **Principles:** `~/.kb/principles.md` - Provenance, Gate Over Remind

---

## Investigation History

**2025-12-25 13:30:** Investigation started
- Initial question: Design kb ask command for inline mini-investigations
- Context: Orchestrator needs quick answers without spawn overhead

**2025-12-25 14:00:** Exploration complete
- Found kb-cli has full retrieval infrastructure
- Found orch ask is alias for send (different semantics)
- Established Provenance principle prohibits --force

**2025-12-25 14:30:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: kb ask should be kb-cli command with OpenCode integration

---

**Started:** 2025-12-25
**Updated:** 2025-12-25
**Owner:** Claude
**Phase:** Complete
**Status:** Complete
**Confidence:** High (85%)
