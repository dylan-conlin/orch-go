---
status: accepted
applies-to:
  - interactive orchestrator sessions (human-in-loop)
blocks:
  - keywords:
      - autonomous gpt orchestrator
      - daemon gpt orchestration
      - gpt orchestrator spawn
---

# Decision: Allow GPT-5.2 for Interactive Orchestrator-Assist (Human-in-Loop)

**Date:** 2026-01-30
**Status:** Accepted
**Decider(s):** Dylan (via orch-go-21055)
**Related:** Narrows/refines 2026-01-21-gpt-unsuitable-for-orchestration.md

## Context

The Jan 21, 2026 decision established that GPT-5.2 is unsuitable for autonomous orchestration due to behavioral anti-patterns:
- Reactive gate handling (learn by hitting, not anticipating)
- Role boundary collapse (doing worker tasks instead of delegating)
- Failure adaptation deficits (repeating identical failures)
- Excessive deliberation (200+ second thinking blocks)

However, a new use case has emerged: **interactive orchestrator-assist** where a human orchestrator uses GPT-5.2 as an assistant while maintaining active supervision and control. This is distinct from autonomous orchestration where the agent operates unattended.

## Decision

**Allow GPT-5.2 for interactive orchestrator-assist mode under these conditions:**

1. **Human-in-loop required** - A human orchestrator must be actively supervising and able to intervene
2. **Strict tool gating** - All high-risk operations (spawn, close, push, etc.) require human approval
3. **No autonomous/daemon use** - GPT-5.2 remains blocked for unattended orchestration
4. **Preserved block keywords** - Keep "autonomous gpt orchestrator", "daemon gpt orchestration", "gpt orchestrator spawn" to prevent accidental autonomous use

**Continue to block GPT-5.2 for:**
- Default orchestration (unattended operation)
- Daemon orchestration (background services)
- Autonomous orchestrator spawns

## Alternatives Considered

### Option 1: Keep GPT-5.2 completely blocked for all orchestration
- **Pros:** Simple, no policy ambiguity, zero risk of anti-patterns
- **Cons:** Wastes potential value in supervised contexts, ignores human mitigation capability
- **Why rejected:** Too restrictive - human supervision addresses the core anti-patterns

### Option 2: Allow GPT-5.2 for all orchestration (autonomous + interactive)
- **Pros:** Simple policy, no special cases
- **Cons:** Reintroduces all anti-patterns from Jan 21 investigation, high risk of failures
- **Why rejected:** Evidence from ses_4207 shows GPT-5.2 fundamentally unsuitable for autonomous orchestration

### Option 3: Create graduated permission system with multiple tiers
- **Pros:** Maximum flexibility, granular control
- **Cons:** Complex to maintain, high cognitive overhead, unclear boundaries
- **Why rejected:** Over-engineered - binary distinction (human-in-loop vs autonomous) is sufficient

## Reasoning

The original anti-patterns that make GPT-5.2 unsuitable for autonomous orchestration are **mitigated by human supervision**:

1. **Reactive gate handling** → Human can anticipate and provide flags upfront
2. **Role boundary collapse** → Human redirects back to delegation when agent starts doing work
3. **Failure adaptation deficits** → Human recognizes repeated failures and suggests strategy changes
4. **Excessive deliberation** → Human can interrupt and provide direction

With tool gating as an additional safety layer, the risk profile becomes acceptable for interactive use cases where cost ($200/mo ChatGPT Pro vs $200/mo Claude Max) or multi-model comparison is valuable.

**Key insight:** The problem with GPT-5.2 isn't the model itself, it's the behavioral patterns *when running unattended*. Human oversight changes the risk equation.

## Consequences

### Positive
- Enables cost-effective human-assisted orchestration for certain workflows
- Leverages GPT-5.2's strengths (different reasoning style) under supervision
- Preserves optionality for users with ChatGPT Pro subscriptions
- Allows multi-model orchestration experiments with human safety net

### Negative/Trade-offs
- Requires maintaining two orchestration policies (autonomous vs interactive)
- Cognitive overhead for users to understand the distinction
- Documentation must clearly communicate the boundaries
- Risk of policy erosion if boundaries aren't enforced via tooling

### Neutral
- The Jan 21 decision remains valid for autonomous use cases
- Block keywords shift from preventing "all GPT orchestration" to "autonomous GPT orchestration"
- Tool gating requirements may slow down interactive workflows slightly

## Implementation Notes

1. **Update .kb/decisions/2026-01-21-gpt-unsuitable-for-orchestration.md**
   - Add note narrowing scope to autonomous orchestration
   - Reference this decision for interactive use cases
   - Preserve block keywords (they now target autonomous use)

2. **Update .kb/guides/model-selection.md**
   - Change "Worker tasks only" to "Worker tasks + interactive orchestrator-assist"
   - Add section explaining human-in-loop distinction
   - Maintain warnings about autonomous orchestration

3. **Block keywords enforcement**
   - Keep existing blocks for "gpt orchestrator", "gpt-5 orchestration", "openai orchestrator" in Jan 21 decision (targets autonomous use)
   - Add new blocks for "autonomous gpt orchestrator", "daemon gpt orchestration", "gpt orchestrator spawn" in this decision (explicit autonomous targeting)

4. **Tool gating requirements (future work)**
   - Consider implementing explicit "interactive mode" flag for orchestrator spawns
   - Tool gating for high-risk operations (spawn, close, push) should be configurable
   - Could gate via `--interactive` flag or environment variable

## Revisit Triggers

Reconsider this decision if:
- GPT-5.2 anti-patterns appear in interactive sessions despite human supervision
- Tool gating proves insufficient to prevent problematic behaviors
- Future GPT versions show different behavioral patterns
- The cost/benefit ratio of maintaining dual policies becomes negative
- Evidence emerges that humans can't effectively mitigate the anti-patterns

## Related

- **Supersedes/narrows:** `.kb/decisions/2026-01-21-gpt-unsuitable-for-orchestration.md` (now applies to autonomous only)
- **Investigation:** `.kb/investigations/2026-01-21-inv-analyze-gpt-orchestrator-session-users.md` (original anti-pattern analysis)
- **Guide:** `.kb/guides/model-selection.md` (implementation of this policy)

---

## AI Reasoning for Recording This Decision

**Criteria met:**
- **Architectural impact** - Changes system model selection policy and orchestration boundaries
- **Cross-project relevance** - Policy applies to all orch-go orchestration use cases
- **Non-obvious trade-offs** - Balances GPT-5.2 limitations vs human supervision capability
- **Security implications** - Tool gating and block keywords prevent unsafe autonomous use

**Justification:** This decision establishes a nuanced policy that requires clear documentation. The distinction between autonomous and interactive orchestration isn't obvious - future developers need to understand when GPT-5.2 is acceptable vs prohibited, and why. The trade-offs (cognitive overhead vs optionality) require explicit reasoning.

**Risk of not documenting:** Without this decision record, the Jan 21 "GPT unsuitable" policy would be interpreted as absolute, blocking potentially valuable interactive use cases. Conversely, loosening the policy without documentation risks reintroducing the anti-patterns. The dual-policy approach (autonomous vs interactive) requires explicit documentation to prevent confusion and policy erosion.
