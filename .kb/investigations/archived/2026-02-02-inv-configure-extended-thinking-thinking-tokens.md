<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Extended thinking (reasoning tokens) is supported and instrumented but disabled by default - requires explicit variant selection ("high" or "max") which orch-go spawns don't currently set.

**Evidence:** (1) Found 1 session with 7553 reasoning_tokens out of 17 samples via `orch tokens --all`, (2) OpenCode variant code shows default is empty object (no thinking config), (3) orch-go CreateSession doesn't pass variant parameter.

**Knowledge:** Extended thinking is opt-in via OpenCode's variant system - "high" (16k tokens, ~$0.048/session) and "max" (32k tokens, ~$0.096/session) variants available for Anthropic models but not enabled without explicit selection.

**Next:** **STRATEGIC DECISION REQUIRED:** Choose between Status Quo (no thinking, current behavior), Selective Thinking (workers only with "high" variant, ~15% cost increase), or Full Thinking (all agents with "high" variant, ~25% cost increase).

**Authority:** strategic - Cost/value tradeoff affects all agent operations, requires orchestrator/Dylan input on whether quality improvements justify overhead.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Configure Extended Thinking Thinking Tokens

**Question:** How are extended thinking / thinking tokens configured in Claude Code and OpenCode? What are the current settings, what options exist, and what should the settings be for orchestrator vs worker agents?

**Started:** 2026-02-02
**Updated:** 2026-02-02
**Owner:** og-inv-configure-extended-thinking-02feb-764c
**Phase:** Complete
**Next Step:** None (awaiting strategic decision from orchestrator/Dylan)
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Beta header for interleaved-thinking is already enabled in orch-go

**Evidence:** Both `pkg/usage/usage.go:32` and `pkg/account/account.go:429` include `"interleaved-thinking-2025-05-14"` in the `AnthropicBetaHeaders` sent with every API request to Anthropic.

**Source:** 
- `pkg/usage/usage.go:29-34`
- `pkg/account/account.go:428-429`

**Significance:** Extended thinking is already enabled at the API level in orch-go's direct API usage (for usage tracking). This means Claude models accessing the API should already be using interleaved thinking mode by default.

---

### Finding 2: OpenCode has thinking budget configuration via variants

**Evidence:** OpenCode's `transform.ts` shows Anthropic models support thinking configuration with `{ type: "enabled", budgetTokens: X }`. Two variants are defined: "high" (16000 tokens) and "max" (31999 tokens).

**Source:**
- `~/Documents/personal/opencode/packages/opencode/src/provider/transform.ts:427-440`
- Variants: `high: { thinking: { type: "enabled", budgetTokens: 16000 } }` and `max: { thinking: { type: "enabled", budgetTokens: 31999 } }`

**Significance:** OpenCode provides explicit control over thinking token budgets through variants, but there's no "none" or "disabled" variant visible, and no evidence of a default variant when none is specified.

---

### Finding 3: orch-go CreateSession does not specify thinking parameters

**Evidence:** The `CreateSession` function in `pkg/opencode/client.go` only passes `title`, `directory`, and `model` parameters when creating an OpenCode session. No thinking configuration or variant is specified.

**Source:**
- `pkg/opencode/client.go:399-427`
- `CreateSessionRequest` struct at line 387

**Significance:** When orch-go spawns OpenCode agents, it doesn't explicitly configure thinking behavior. This means agents use OpenCode's default behavior for the selected model.

---

### Finding 4: Reasoning tokens ARE being tracked and used in production

**Evidence:** Running `orch tokens --all --json` shows actual reasoning token usage. Session `ses_3e65254b5ffeDv4X6m8UOEL5TU` (orch-go-21130) shows: `input_tokens: 154976`, `output_tokens: 10113`, `reasoning_tokens: 7553`. This is the only session in the sample with non-zero reasoning tokens.

**Source:**
- Command: `orch tokens --all --json`
- Found reasoning_tokens: 7553 for one session out of 17 sampled sessions

**Significance:** Extended thinking/reasoning tokens ARE actually being generated and tracked for Claude models. However, only 1 out of 17 sessions showed reasoning tokens, suggesting that either: (a) reasoning is enabled by default but rarely triggered, or (b) a specific configuration or model was used for that session that enabled extended thinking.

---

### Finding 5: Token tracking infrastructure supports reasoning tokens

**Evidence:** The codebase has full support for reasoning tokens: `pkg/opencode/types.go` defines `Reasoning int`, `pkg/cost/cost.go` includes `ReasoningPerMillion: 3.00` pricing at $3/million (same as input tokens), and token aggregation in `pkg/opencode/client.go` sums reasoning tokens into total.

**Source:**
- `pkg/opencode/types.go:45` - Token struct with Reasoning field
- `pkg/cost/cost.go:19-21` - Pricing structure
- `pkg/opencode/client.go:601-609` - Token aggregation

**Significance:** The system is fully instrumented to track, cost, and display reasoning tokens. This isn't aspirational code - it's production-ready infrastructure that's actively capturing reasoning token usage when it occurs.

---

### Finding 6: Default behavior is NO extended thinking - variants must be explicitly selected

**Evidence:** OpenCode's LLM invocation code shows: `const variant = !input.small && input.model.variants && input.user.variant ? input.model.variants[input.user.variant] : {}`. This means if no variant is specified by the user, an empty object is used (no thinking config).

**Source:**
- `~/Documents/personal/opencode/packages/opencode/src/session/llm.ts` (line with variant assignment)

**Significance:** Extended thinking is NOT enabled by default - it requires explicit variant selection. The session with 7553 reasoning tokens (Finding 4) must have had a variant explicitly selected (either "high" or "max"). This explains why 16 out of 17 sampled sessions had zero reasoning tokens.

---

### Finding 3: [Brief, descriptive title]

**Evidence:** [Concrete observations, data, examples]

**Source:** [File paths with line numbers, commands run, specific artifacts examined]

**Significance:** [Why this matters, what it tells us, implications for the investigation question]

---

## Synthesis

**Key Insights:**

1. **Infrastructure is ready, but not enabled** - The full stack for extended thinking is in place (beta headers, token tracking, cost calculation, variant system) but the default behavior is to NOT enable thinking. This is an opt-in feature that requires explicit configuration.

2. **Variant selection is the control mechanism** - OpenCode uses "reasoning variants" to enable extended thinking. For Anthropic models, two variants exist: "high" (16k thinking tokens) and "max" (32k thinking tokens). When no variant is specified, thinking is disabled.

3. **Current orch-go spawns use default (no thinking)** - The CreateSession API in orch-go doesn't specify variants, meaning all spawned agents currently run WITHOUT extended thinking. This explains why only 1 out of 17 sessions showed reasoning tokens (that session likely had manual variant selection in the UI).

**Answer to Investigation Question:**

**Current settings:**
- **Claude Code:** Uses interleaved-thinking beta header, but actual behavior depends on client implementation (not directly configurable in our code)
- **OpenCode:** Default is NO extended thinking. Variants must be explicitly selected via UI (Ctrl+T to cycle) or API parameter

**Configuration options:**
- OpenCode supports variants: "high" (16k thinking budget) and "max" (32k thinking budget) for Anthropic models
- These are set per-session via `input.user.variant` field
- orch-go's CreateSession API doesn't currently support setting variants

**Orchestrator vs Worker recommendation:**
This is a STRATEGIC decision requiring orchestrator/Dylan input. Key tradeoffs:
- **Cost:** Reasoning tokens cost $3/million (same as input tokens), adding ~15-25% overhead for thinking-enabled sessions
- **Speed:** Extended thinking adds latency (model must think before responding)
- **Quality:** Extended thinking may improve complex reasoning tasks but adds no value for simple tasks
- **Current behavior:** No agents currently use extended thinking by default - represents significant change

---

## Structured Uncertainty

**What's tested:**

- ✅ Beta header is present in orch-go code (verified: read pkg/usage/usage.go:32 and pkg/account/account.go:429)
- ✅ Reasoning tokens are tracked in production (verified: `orch tokens --all --json` showed session with 7553 reasoning_tokens)
- ✅ OpenCode defines variants for Anthropic (verified: read transform.ts, found "high" and "max" with budgetTokens)
- ✅ Default behavior uses empty variant object (verified: read session/llm.ts variant assignment logic)
- ✅ orch-go CreateSession doesn't pass variants (verified: read pkg/opencode/client.go CreateSessionRequest struct)

**What's untested:**

- ⚠️ Whether extended thinking improves worker quality (not benchmarked - only 1 session with reasoning tokens found)
- ⚠️ Whether extended thinking slows orchestrator delegation (not measured - hypothesis based on model behavior)
- ⚠️ Exact cost overhead percentage (estimated ~15-25% based on typical reasoning token ratios, not measured in production)
- ⚠️ Whether "max" variant (32k tokens) is ever needed (no sessions found using it)

**What would change this:**

- Finding would be wrong if recent sessions show high reasoning token usage (would indicate default behavior changed or manual variant selection is common)
- Finding would be wrong if CreateSession API actually passes variants via a different mechanism (would mean agents DO have extended thinking enabled)
- Cost estimates would be wrong if reasoning token ratios differ significantly from the 7553/172642 ratio observed (4.4% of total tokens)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Whether to enable extended thinking for orchestrator/worker agents | strategic | Cost/value tradeoff, affects all agent operations, unclear if benefits justify overhead |
| How to implement variant passing in spawn code | implementation | Technical implementation within established patterns, reversible change |

### Three Options for Extended Thinking Configuration

**Option A: Status Quo (No Extended Thinking) ⭐**

Current behavior - no variants specified, extended thinking disabled by default.

**Why this approach:**
- Zero cost increase - reasoning tokens cost $3/million same as input
- No latency overhead - models respond immediately without thinking phase
- Works well for current operations - 16 out of 17 sessions had no reasoning needs
- Simple - no code changes required

**Trade-offs accepted:**
- May miss quality improvements on complex tasks (investigations, architecture, debugging)
- Can't evaluate if extended thinking would help orchestrator decisions
- One session DID use 7553 reasoning tokens - suggests some tasks benefit

**When to choose:** If current quality is acceptable and cost/speed are priorities.

---

**Option B: Selective Extended Thinking (Workers Only - "high" variant)**

Enable extended thinking for worker agents (investigations, features, debugging) with 16k token budget.

**Why this approach:**
- Workers handle complex reasoning tasks most likely to benefit
- Orchestrators stay fast for delegation decisions
- 16k budget balances quality vs cost (~15% overhead)
- Can measure impact before expanding

**Trade-offs accepted:**
- Adds ~$0.045 per 15k reasoning tokens (~$0.03 input equivalent)
- Workers may be slower to respond (thinking phase)
- Requires code change to pass variant parameter

**When to choose:** If worker quality/thoroughness is worth modest cost increase.

**Implementation:**
1. Add `variant` parameter to `CreateSessionRequest` struct
2. Pass `variant: "high"` when spawning investigation/debugging agents
3. Leave orchestrators and simple tasks with no variant
4. Monitor token usage and quality changes

---

**Option C: Full Extended Thinking (All Agents - "high" variant)**

Enable extended thinking for both orchestrators and workers with 16k token budget.

**Why this approach:**
- Orchestrators may benefit for strategic/architectural decisions
- Consistent behavior across all agents
- Maximum potential quality improvement
- Simple rule: always use thinking

**Trade-offs accepted:**
- Highest cost increase (~15-25% overhead on all sessions)
- Orchestrators may be too slow for rapid delegation
- May not need thinking for simple "spawn this task" decisions
- Over-engineering for tasks that don't need deep reasoning

**When to choose:** If orchestrator quality/strategic decisions are critical and cost is secondary.

**Implementation:**
1. Add `variant` parameter to `CreateSessionRequest` struct
2. Pass `variant: "high"` for ALL OpenCode spawns
3. Monitor for orchestrator slowdown in delegation loops
4. Consider "max" variant for long-running strategic work

---

**Rationale for recommendation:** Option A (Status Quo) is recommended unless there's evidence that current quality is insufficient. The single session with reasoning tokens suggests some tasks DO benefit, but 16/17 sessions working fine without it indicates extended thinking isn't necessary for most work. Option B is the natural next step if we want to experiment - target workers where complex reasoning is most valuable.

---

### Implementation Details

**What to implement first:**
- Add `variant` field to `CreateSessionRequest` struct in `pkg/opencode/client.go`
- Add `variant` parameter to `CreateSession` function signature
- Update spawn callers to pass variant (or empty string for default)

**Things to watch out for:**
- ⚠️ OpenCode API may reject unknown variant names - verify "high" and "max" are the only valid values
- ⚠️ Variant selection is per-session, not per-message - can't toggle mid-session
- ⚠️ Cost increase may be higher if agents use thinking frequently (the 4.4% ratio from one session may not be representative)
- ⚠️ Reasoning tokens count toward context limits (thinking budget + output must fit within model's max tokens)

**Areas needing further investigation:**
- How does thinking token usage correlate with task complexity? (need more samples with variants enabled)
- What's the latency impact of thinking phase? (not measured)
- Can orchestrators benefit from thinking for strategic decisions? (unclear - fast delegation may be more valuable)
- Is there a way to enable thinking conditionally based on task type? (would require orchestrator intelligence)

**Success criteria:**
- ✅ Variant parameter successfully passed to OpenCode API
- ✅ Sessions created with variant show reasoning_tokens in `orch tokens` output
- ✅ Cost increase matches expectations (~15-25% for thinking-enabled sessions)
- ✅ Worker quality improves or stays same (subjective - need qualitative eval)

---

## References

**Files Examined:**
- `pkg/usage/usage.go` - Checked for Anthropic beta headers (interleaved-thinking)
- `pkg/account/account.go` - Verified beta headers in account switching code
- `pkg/opencode/client.go` - Analyzed CreateSession API and token tracking
- `pkg/opencode/types.go` - Found Reasoning token field definition
- `pkg/cost/cost.go` - Checked reasoning token pricing ($3/million)
- `~/Documents/personal/opencode/packages/opencode/src/provider/transform.ts` - Found variant definitions for Anthropic
- `~/Documents/personal/opencode/packages/opencode/src/session/llm.ts` - Discovered default variant behavior

**Commands Run:**
```bash
# Check for thinking-related code
rg "thinking|reasoning" pkg --type go

# Get actual token usage from production sessions
orch tokens --all --json

# Examine session with reasoning tokens
orch tokens ses_3e65254b5ffeDv4X6m8UOEL5TU --json

# Search OpenCode source for variant logic
rg "variant|thinking" ~/Documents/personal/opencode/packages/opencode/src --type ts
```

**External Documentation:**
- Anthropic API beta headers - interleaved-thinking feature flag
- OpenCode variant system - reasoning configuration mechanism

**Related Artifacts:**
- **Related issues:** Session with reasoning tokens was orch-go-21130 (Evidence Hierarchy warning task)

---

## Investigation History

**[2026-02-02 14:30]:** Investigation started
- Initial question: How to configure extended thinking / thinking tokens in Claude Code and OpenCode?
- Context: Need to understand current settings and determine appropriate configuration for orchestrator vs worker agents

**[2026-02-02 14:45]:** Found interleaved-thinking beta header
- Discovered beta header is already enabled in orch-go API code
- Not clear if this actually enables thinking by default

**[2026-02-02 15:10]:** Found evidence of reasoning tokens in production
- Ran `orch tokens --all` and found session with 7553 reasoning tokens
- Proved that extended thinking IS being tracked and used (at least sometimes)

**[2026-02-02 15:30]:** Discovered variant system and default behavior
- Found OpenCode variant definitions: "high" (16k) and "max" (32k) for Anthropic
- Discovered default behavior is NO extended thinking (empty variant object)
- Explains why only 1 out of 17 sessions had reasoning tokens

**[2026-02-02 16:00]:** Investigation completed
- Status: Complete
- Key outcome: Extended thinking is opt-in via variants, currently NOT enabled by default for orch-go spawns
