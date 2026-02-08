# Multi-Model Evaluation Model (Feb 2026)

**Summary:** A cluster of 5 investigations evaluated GPT-5.3-Codex as a potential second model for the orch swarm. The verdict: GPT-5.3-Codex produces non-compiling code on extraction tasks (17 duplicate test functions), has structural token counting differences, and is only accessible via Codex CLI (no API). It is NOT ready to replace Claude Opus for code work, but may be useful for investigation/research tasks where compilation isn't the deliverable. Model routing via beads labels is the validated integration path.

**Synthesized from:** 5 investigations, 1 decision (daemon resilience), direct orchestrator engagement.

---

## What We Tested

### Side-by-Side Code Extraction (The Critical Experiment)

Two agents ran the same task (code extraction, ~2000 line file → multiple modules) in parallel:

| Dimension | GPT-5.3-Codex | Claude Opus | Winner |
|-----------|--------------|-------------|--------|
| **Compiles** | ❌ No (17 duplicate test funcs) | ✅ Yes | **Claude** |
| File granularity | 11 files (finer split) | 4 files | GPT (better split) |
| Under 500-line target | 10/11 pass | 2/4 fail | GPT (closer to target) |
| Test duplication | Critical: 17 funcs duplicated | None | **Claude** |
| Testability patterns | Globals everywhere | Function var injection | **Claude** |
| Error handling | Inconsistent | Consistent graceful degradation | **Claude** |

**Bottom line:** GPT-5.3-Codex has better structural instincts (finer file splits, closer to line targets) but produces non-compiling output. Claude Opus produces shippable code. For code work, Claude wins on the only metric that matters: does it compile and pass tests.

### Session Death Investigation

Three GPT-5.3-Codex sessions died simultaneously. Investigation found this was a single OpenCode server restart event (jetsam kill), not GPT-specific instability. The sessions were only 7-8 minutes old. This is a control-plane reliability issue, not a model quality issue.

---

## Token Economics

### Key Finding: Tokens Are Apples-to-Apples (Mostly)

| Provider | Reasoning tokens | How counted |
|----------|-----------------|-------------|
| OpenAI | Separate field (`reasoningTokens`) | Counted separately from output |
| Anthropic | Baked into `output_tokens` | No separate field; thinking included in output |

Both represent total token usage, just categorized differently. orch-go computes `total = input + output + reasoning`, which produces correct totals for both providers.

**Multi-step overwrite bug found:** OpenCode overwrites previous step's token counts instead of accumulating. This affects both providers but is more visible with OpenAI's separate reasoning field.

### ChatGPT Pro Economics

- GPT-5.3-Codex: Codex-only model, **no API access** (ChatGPT subscription auth only)
- Limits are **message-count based**, not token-based
- Pro ($200/mo): 300-1500 local messages/5h, 50-400 cloud tasks/5h
- A 150K token task and a 5K token task both count as 1 message
- This makes high-token tasks proportionally MORE economical

### Access Path Constraint

GPT-5.3-Codex is analogous to Claude Opus's restriction: only accessible through the vendor's CLI tool, not standard API. This means:
- OpenCode can route to it via provider config
- Cannot be used via raw API calls
- Subscription-based, not per-token billing

---

## Model Routing Decision (Already Decided)

The daemon resilience decision (2026-02-06) includes model routing via beads labels:

```bash
bd label <id> model:sonnet    # Routes to Claude Sonnet
bd label <id> model:opus      # Routes to Claude Opus (forces Claude backend)
bd label <id> model:pro       # Routes to GPT via OpenCode
```

**Implementation:** `InferModelFromLabels()` in `pkg/daemon/skill_inference.go` (not yet built). Follows existing `skill:*` label pattern.

**Status:** Decided but not implemented. Existing issue: `orch-go-21350`.

---

## Recommended Model Assignment

Based on the evidence:

| Task Type | Recommended Model | Rationale |
|-----------|------------------|-----------|
| Code implementation | Claude Opus | Compiles, passes tests, shippable |
| Code extraction/refactoring | Claude Opus | GPT duplicated test functions |
| Investigation/research | Either (GPT viable) | No compilation requirement |
| Architecture/design | Claude Opus | Reasoning quality matters most |
| Debugging | Claude Opus | Root cause analysis needs depth |
| Documentation/writing | Either | Both adequate |
| Trivial fixes (<5 files) | Claude Sonnet or GPT | Cost optimization |

**When to try GPT-5.3-Codex:**
- Investigations where the artifact is text, not code
- Tasks with strong verification gates (tests will catch issues)
- Cost optimization experiments (message-based billing favors large tasks)

**When NOT to use GPT-5.3-Codex:**
- Any task where "compiles" is a requirement without a human review step
- Multi-file refactoring (test duplication risk)
- Tasks using globals or complex test patterns

---

## Open Questions

1. **Does GPT-5.3-Codex improve with more explicit test dedup instructions?** The extraction experiment used identical prompts for both models. GPT might do better with "DO NOT duplicate test functions across files" explicit guidance.

2. **Is the token overwrite bug in OpenCode fixed?** Multi-step token counting affects both providers. This is an OpenCode fork issue, not model-specific.

3. **How do investigation-quality tasks compare?** We tested code extraction (GPT's weakness) but not investigation/research (potentially GPT's strength). A controlled test would be valuable.

---

## References

### Investigations (Provenance Chain)
- `2026-02-06-inv-side-by-side-code-quality-gpt53-vs-claude-opus.md` — Critical: compilation comparison
- `2026-02-06-inv-research-chatgpt-pro-gpt-5.3-codex-quota.md` — Economics and access paths
- `2026-02-06-inv-token-counting-discrepancy-gpt-vs-claude.md` — Token accounting differences
- `2026-02-06-inv-test-gpt-codex-non-mechanical.md` — Session death investigation (server restart, not GPT issue)
- `2026-02-06-inv-design-two-prompt-variants-gpt.md` — Prompt variant design for extraction

### Decisions
- `2026-02-06-daemon-resilience-retry-staging-model-routing.md` — Decision 5: model routing via labels
