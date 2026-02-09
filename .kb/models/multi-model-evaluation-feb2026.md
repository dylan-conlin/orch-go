# Multi-Model Evaluation Model (Feb 2026)

**Summary:** GPT-5.3-Codex is now the default model for all agent spawns. After an initial rough showing on Feb 6 (one extraction task produced non-compiling code), it has been the primary workhorse since Feb 7 — completing 18+ agents across feature-impl, systematic-debugging, and investigation tasks. The main failure mode is not code quality but **completion protocol compliance** (agents finish work then forget to run `orch phase Complete`). Model routing via beads labels is the validated integration path for cases where specific models are needed.

**Synthesized from:** 6 investigations, 1 decision (daemon resilience), production usage data (Feb 7-8), direct orchestrator engagement, 1 probe (Feb 8, 2026).

**Recent Probes:**
- `probes/2026-02-08-daemon-model-routing-not-yet-wired.md` — **Confirms** label-based per-issue model routing remains unimplemented in daemon spawn path. `orch work` infers skill and MCP only; no `InferModelFromLabels` function exists. Focus-level model preference also absent. Confidence: High.

**Feb 8-9 Update:** Codex performance evaluation across 5 verification-spec implementation issues: 5/5 closed in 88 min total, clean Go code, proper tests (424 lines, 10 cases), faithful to decision record spec. Token efficiency: 1.48M total (highest single issue 638K on batch CLI integration). Comparable speed to overnight Opus baseline on implementation tasks. Protocol compliance improved (all 5 reported proper phase progression). Still needs evaluation on investigation/debugging/architect skills.

---

## Current State: GPT-5.3-Codex is Default

**Decision:** "Use gpt-5.3-codex for all spawns" — primarily to avoid Claude Max session throttling during orchestration.

**Feb 7 production evidence:** 18+ agents spawned on GPT-5.3-Codex in a single day:
- Feature implementation (stability measurement, operator health, constraint enforcement, spawn templates, linter rules, cache constructors, resource ceilings, work-graph updates)
- Systematic debugging (work-graph endless polling, beads concurrency, process census false positives, crash-free streak)
- Investigations (model-specific completion failures, architecture audits, dashboard health)

Most completed successfully. This is not a "secondary model" — it's the primary.

---

## The Real Failure Mode: Completion Protocol, Not Code Quality

The Feb 6 extraction comparison (GPT duplicated 17 test functions) was one bad data point. The actual pattern across many agents is different:

**GPT-5.3-Codex's weakness:** After successfully implementing and testing code, the agent ends with conversational text instead of running the completion protocol (`orch phase Complete` + `/exit`). In 3 of 4 analyzed stalled sessions, the agent finished code + validation and then stopped without declaring completion.

**This is a protocol compliance issue, not a code quality issue.** The work itself is good — the agent just forgets the last step.

**Mitigations in progress:**
- Stronger completion language in spawn context templates
- Phase-aware daemon nudge behavior (instead of generic "continue work" prompts)
- Model-profile tuning for completion-protocol emphasis

---

## Initial Evaluation (Feb 6) — Contextualized

### Side-by-Side Code Extraction

| Dimension | GPT-5.3-Codex | Claude Opus |
|-----------|--------------|-------------|
| **Compiles** | ❌ No (17 duplicate test funcs) | ✅ Yes |
| File granularity | 11 files (finer split) | 4 files |
| Under 500-line target | 10/11 pass | 2/4 fail |
| Testability patterns | Globals | Function var injection |

**What this actually shows:** GPT has better structural instincts (finer splits, closer to size targets) but was sloppy about test file deduplication in this specific task. This was the first GPT extraction task — subsequent tasks with better prompt guidance have not reproduced this failure class.

### Session Deaths (Feb 6)

Three GPT sessions died simultaneously. Investigation confirmed: single OpenCode server restart (jetsam kill), not GPT-specific. Control-plane reliability issue, not model quality.

---

## Token Economics

### Tokens Are Apples-to-Apples (Mostly)

| Provider | Reasoning tokens | How counted |
|----------|-----------------|-------------|
| OpenAI | Separate field (`reasoningTokens`) | Counted separately from output |
| Anthropic | Baked into `output_tokens` | No separate field; thinking included in output |

Both represent total token usage. orch-go computes `total = input + output + reasoning` — correct for both.

**Multi-step overwrite bug:** OpenCode overwrites previous step's token counts instead of accumulating. Affects both providers.

### ChatGPT Pro Economics

- GPT-5.3-Codex: Codex-only model, **no API access** (ChatGPT subscription auth only)
- Limits are **message-count based**, not token-based
- Pro ($200/mo): 300-1500 local messages/5h, 50-400 cloud tasks/5h
- High-token tasks proportionally MORE economical (1 task = 1 message regardless of tokens)

---

## Model Routing (Decided, Partially Implemented)

The daemon resilience decision includes model routing via beads labels:

```bash
bd label <id> model:sonnet    # Routes to Claude Sonnet
bd label <id> model:opus      # Routes to Claude Opus (forces Claude backend)
bd label <id> model:pro       # Routes to GPT via OpenCode
```

**Current config default:** `default_model: openai/gpt-5.3-codex` in `~/.orch/config.yaml`.

**Status:** Default model routing works via config. Label-based per-issue routing (`InferModelFromLabels`) not yet implemented. Issue: `orch-go-21350`.

---

## When to Use Which Model

Based on production evidence (not just the single extraction test):

| Task Type | Default (GPT-5.3) | When to use Claude |
|-----------|-------------------|-------------------|
| Feature implementation | ✅ Working well | Complex multi-system architectural changes |
| Debugging | ✅ Working well | Subtle root cause analysis requiring deep reasoning |
| Investigation/research | ✅ Working well | — |
| Code extraction | ✅ Working (with better prompts) | — |
| Architecture/design | ✅ Adequate | When tradeoff analysis depth matters most |

**The real decision axis is not "which model" but "do I need Claude Max credits for orchestrator sessions?"** GPT-5.3-Codex frees up Claude Max quota for orchestrator work.

---

## Open Questions

1. **Completion protocol compliance:** Can spawn template improvements and daemon nudges close the gap, or is this a fundamental model behavior that needs a structural workaround?

2. **Token overwrite bug:** Still present in OpenCode fork. Affects visibility into agent costs.

3. **Quality at scale:** With 18+ agents in one day, are there patterns in which tasks GPT handles well vs. struggles with? Need more production data.

---

## References

### Investigations (Provenance Chain)
- `2026-02-06-inv-side-by-side-code-quality-gpt53-vs-claude-opus.md` — Initial extraction comparison (one data point)
- `2026-02-07-inv-design-orch-handles-model-specific.md` — Completion protocol failure analysis (the real issue)
- `2026-02-06-inv-research-chatgpt-pro-gpt-5.3-codex-quota.md` — Economics and access paths
- `2026-02-06-inv-token-counting-discrepancy-gpt-vs-claude.md` — Token accounting differences
- `2026-02-06-inv-test-gpt-codex-non-mechanical.md` — Session death investigation (server restart, not GPT)
- `2026-02-06-inv-design-two-prompt-variants-gpt.md` — Prompt variant design

### Decisions
- `2026-02-06-daemon-resilience-retry-staging-model-routing.md` — Decision 5: model routing via labels
- kn: "Use gpt-5.3-codex for all spawns"
