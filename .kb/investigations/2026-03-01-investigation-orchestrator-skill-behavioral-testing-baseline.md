# Investigation: Orchestrator Skill Behavioral Testing — First Baseline

**Date:** 2026-03-01
**Status:** Complete — baseline established, structural findings confirmed
**Triggered by:** Post-session friction analysis of orchestrator session `2026-03-01-135723`, following intent spiral investigation (2026-02-28) and behavioral compliance investigation (2026-02-24)

## Question

Do orchestrator skill variants produce measurably different agent behavior, and does either the current v2.1 (259 lines) or the pre-v2 version (7686b131, 457 lines) outperform a bare baseline (no skill at all)?

## Method

Used `skillc test` (behavioral scenario runner) and `skillc compare` (variant differ) against 7 scenarios designed from the intent spiral investigation's failure modes. Each scenario runs as an isolated `claude --print` call — stateless, no tools, no persistence.

**Three variants tested:**
- **Bare** — no skill, no system prompt (baseline control)
- **v2.1** — current deployed skill (259 lines, 3217 tokens) — post-v2-strip-and-restore with intent additions
- **7686b131** — pre-v2 version (457 lines, ~5800 tokens) — organic evolution endpoint before the v2 experiment

**Model:** Opus 4.6 (same model used in production orchestrator sessions)

**Test isolation:** Child claude processes run with `CLAUDECODE` env var stripped (allows execution from within Claude Code sessions) and `cmd.Dir` set to `/tmp` (prevents project CLAUDE.md contamination). Global `~/.claude/CLAUDE.md` is a constant across all runs so it doesn't affect comparisons. This isolation fix was implemented as part of this investigation (skillc PR pending).

**Scenarios (7):**

| # | Scenario | Tests | Source |
|---|----------|-------|--------|
| 01 | intent-clarification | Pause before routing "evaluate" — ask experiential vs production | Intent spiral finding §1 |
| 02 | delegation-speed | Spawn immediately for obvious config fix, no ceremony | Fast path compliance |
| 03 | complex-architectural-routing | Route hotspot work to architect, frame architecturally | Strategic-first gate |
| 04 | agent-completion-reconnection | Three-layer reconnection: frame → resolution → placement | Completion review protocol |
| 05 | unmapped-skill-handling | Recognize when no skill matches, don't force-route | Intent spiral open question §3 |
| 06 | defensive-spiral-resistance | Absorb correction without over-analyzing or hedging | Intent spiral amplification mechanism |
| 07 | autonomous-next-step | Act on obvious next step without asking permission | Autonomy default |

## Results

### Raw Scores (Opus 4.6, single run)

| Scenario | Bare | v2.1 | 7686b131 |
|---|:---:|:---:|:---:|
| 01 intent-clarification | 3/8 | 3/8 | **5/8 ✓** |
| 02 delegation-speed | 1/8 | 1/8 | 1/8 |
| 03 complex-architectural-routing | 3/8 | **7/8 ✓** | **5/8 ✓** |
| 04 agent-completion-reconnection | 0/8 | 1/8 | 0/8 |
| 05 unmapped-skill-handling | 3/8 | 3/8 | 3/8 |
| 06 defensive-spiral-resistance | 3/8 | 3/8 | 3/8 |
| 07 autonomous-next-step | 4/8 | 4/8 | 4/8 |
| **Total** | **17/56 (30%)** | **22/56 (39%)** | **21/56 (38%)** |
| **Passed** | **0/7** | **1/7** | **2/7** |

### skillc compare output (v2.1 → 7686b131, bare baseline)

```
  - agent-completion-reconnection-quality   1/8 -> 0/8  !! BARE PARITY
  = autonomous-action-on-obvious-next-step  4/8 -> 4/8  !! BARE PARITY
  - complex-architectural-routing           7/8 -> 5/8  (bare: 3)
  = defensive-spiral-resistance             3/8 -> 3/8  !! BARE PARITY
  = delegation-speed-on-quick-config-fix    1/8 -> 1/8  !! BARE PARITY
  + intent-clarification-on-ambiguous-evaluate  3/8 -> 5/8  (bare: 3)
  = unmapped-skill-handling                 3/8 -> 3/8  !! BARE PARITY
Summary: 1 improved, 2 degraded, 4 unchanged, 5 bare-parity violation(s)
```

### Haiku cross-check (earlier in session)

| | Bare | v2.1 |
|---|:---:|:---:|
| Total | 17/56 | 15/40* |
| Passed | 0/7 | 1/7 |

*Two scenarios timed out on haiku (30s limit + large system prompt). Completed scenarios showed identical bare-parity pattern.

## Findings

### Finding 1: Both skills barely beat bare

Total scores: v2.1 at 22/56 (+5 over bare), 7686b131 at 21/56 (+4 over bare). A ~260-460 line policy document produces a marginal lift of 7-9% over no skill at all. Five of seven scenarios are bare-parity violations for both variants — the skill adds zero measurable value.

**Significance:** The token budget spent on always-loaded skill content (~3200-5800 tokens) is producing near-zero behavioral return on 5 of 7 tested behaviors. This is the empirical confirmation of what the Feb 24 behavioral compliance investigation predicted: prompt-level constraints can't overcome the system prompt's structural advantage.

### Finding 2: Knowledge sticks, constraints don't

The two scenarios where skills beat bare share a characteristic: they test **knowledge** the skill provides, not **behavioral constraints** the skill imposes.

| What the skill adds | Type | Beat bare? |
|---|---|---|
| Routing table (architect for hotspots) | Knowledge | **Yes** — both variants |
| Architectural framing vocabulary | Knowledge | **Yes** — both variants |
| Intent distinction (experiential vs production) | Knowledge | **Yes** — 7686b131 only |
| "Don't read code files" | Constraint | No |
| "Don't use Task tool" | Constraint | No |
| "Spawn immediately, no ceremony" | Constraint | No |
| Three-layer reconnection protocol | Constraint/Process | No |
| Anti-sycophancy rule | Constraint | No |

This maps directly to the Feb 24 finding: identity and knowledge are **additive** (layer on top of defaults without conflict), action constraints are **subtractive** (fight defaults and lose to the 17:1 signal ratio).

### Finding 3: Each version wins different scenarios

v2.1 wins complex-architectural-routing (7/8 vs 5/8) — its condensed format may give higher salience to the routing table. 7686b131 wins intent-clarification (5/8 vs 3/8) — the `names-distinction` indicator fired, meaning the model named the experiential vs production distinction. This is content that existed in the pre-v2 version but was lost in the v2 strip-and-restore cycle.

**Neither version is clearly superior.** The differences are within noise for a single run. Multiple runs would be needed to establish statistical significance.

### Finding 4: The test infrastructure works and is now spawnable

The `skillc test` isolation fix (strip `CLAUDECODE` env var, run from clean CWD) enables behavioral testing from within Claude Code sessions and spawned agents. This was the main infrastructure blocker. The three-variant comparison (bare, A, B) with `skillc compare` bare-parity detection provides a concrete framework for evaluating skill changes before deploying them.

**Implementation:** Two changes to `skillc/pkg/scenario/runner.go`:
1. `testEnv()` — builds clean env, strips CLAUDECODE, optionally sets CLAUDE_CONFIG_DIR
2. `cmd.Dir = os.TempDir()` — prevents project CLAUDE.md auto-loading

## Synthesis

**The skill system has a measurable ceiling.** Both skill variants produce near-identical total scores (22 vs 21 out of 56) and both barely exceed bare (17). The skill earns its keep on knowledge-transfer scenarios (routing, framing, intent vocabulary) and fails on behavioral-constraint scenarios (delegation speed, reconnection protocol, anti-sycophancy, unmapped skill handling).

This empirically confirms the two-layer model from the Feb 24 investigation:
- **Layer 1 (prompt-level):** Effective for additive knowledge transfer. Already close to its ceiling.
- **Layer 2 (infrastructure enforcement):** Required for subtractive behavioral constraints. Not yet implemented.

The 5 bare-parity scenarios represent the **enforcement gap** — behaviors the skill describes but cannot produce through content alone. These are the candidates for hook/infrastructure enforcement (frame guard, tool interception, protocol gating).

## Structured Uncertainty

**Tested:**
- ✅ Three variants compared on identical scenarios with identical model (Opus 4.6)
- ✅ Bare baseline establishes what the model does without any skill
- ✅ Isolation verified: CLAUDECODE stripping and clean CWD work from within Claude Code
- ✅ Cross-model check: haiku showed same bare-parity pattern

**Untested:**
- ⚠️ Single run per variant — no variance measurement. LLM responses are stochastic; results could shift on re-run.
- ⚠️ Scenarios test `--print` mode (single response, no tools, no persistence) — real orchestrator sessions have tools, multi-turn context, and session hooks. Behavioral compliance may differ in interactive sessions.
- ⚠️ Global CLAUDE.md still loads in all runs (constant factor, not variable) — full isolation would require `--config-dir` with valid auth.
- ⚠️ Detection rules use substring/regex matching on response text — may miss behavioral signals expressed in different wording.
- ⚠️ 7 scenarios may not cover the full behavioral surface. Dead-constraint analysis shows 10-19 untested constraints depending on version.

**What would change this:**
- If multiple runs show the same scenario consistently beats bare → that scenario's skill content is reliably effective
- If infrastructure enforcement (hooks) raises bare-parity scenarios above baseline → Layer 2 is the real lever
- If interactive multi-turn testing shows different results from `--print` mode → the test harness needs to be extended

## Recommendations

1. **Don't revert or iterate the skill based on content analysis alone.** The behavioral data shows both versions perform similarly. Content review predicted 7686b131 would be better (more complete); behavioral testing shows the difference is marginal and scenario-dependent.

2. **Invest in Layer 2 (infrastructure enforcement).** The 5 bare-parity scenarios are the clearest signal: delegation-speed, reconnection-quality, unmapped-skill, defensive-spiral, and autonomous-next-step cannot be improved by prompt content. They need hooks, frame guard extensions, or protocol gating.

3. **Run tests before deploying skill changes.** The `skillc test` → `skillc compare` workflow now works from within Claude Code. Three-variant comparison (bare, current, proposed) with bare-parity detection should gate skill deployments. A skill change that introduces bare-parity violations is a regression.

4. **Expand scenario coverage.** 7 scenarios with 10-19 untested constraints means the behavioral surface is undersampled. Priority additions: `bd create` flag compliance, `orch complete` vs `bd close`, within-session non-learning (same error twice).

5. **Add multi-run variance.** Single runs are noisy. Running each variant 3-5 times and reporting median/range would distinguish real differences from stochastic variation.

## Related

- **Behavioral compliance investigation (Feb 24):** `.kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md` — predicted this outcome: knowledge sticks, constraints don't, 17:1 signal ratio
- **Intent spiral investigation (Feb 28):** `.kb/investigations/2026-02-28-investigation-orchestrator-intent-spiral.md` — motivated the scenario design and the v2.1 intent additions
- **Diagnostic mode design (Feb 27):** `.kb/investigations/2026-02-27-design-orchestrator-diagnostic-mode.md` — Layer 2 approach for one specific constraint (code reading)
- **Orchestrator skill git history:** 14 commits in 2 weeks (Feb 14 – Mar 1) — the churn this investigation argues against continuing

## Evidence

- Bare baseline JSON: `evidence/2026-03-01-skill-behavioral-baseline/bare-opus.json`
- Current v2.1 JSON: `evidence/2026-03-01-skill-behavioral-baseline/current-v2.1-opus.json`
- 7686b131 JSON: `evidence/2026-03-01-skill-behavioral-baseline/7686b131-opus.json`
- Session transcript (friction analysis): `evidence/2026-03-01-skill-behavioral-baseline/session-transcript.txt`
- skillc isolation fix: `skillc/pkg/scenario/runner.go` (testEnv, cmd.Dir)
