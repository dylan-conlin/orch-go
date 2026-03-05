# Model: Skillc Testing

**Domain:** Skill Compilation / Behavioral Testing Infrastructure
**Created:** 2026-03-04
**Status:** Active
**Source:** Consolidated from scattered knowledge across MEMORY.md, SESSION_HANDOFF.md, kb quick constraints, behavioral-grammars probes, and skillc source code review.

---

## Summary (30 seconds)

Skillc test is a behavioral testing harness that measures whether a skill document changes an LLM's response distribution. It works by running the same scenario prompts with and without a skill (variant vs bare), scoring responses against detection patterns, and comparing pass rates. The harness uses Claude CLI's `--print` mode — single-turn, no tool execution, text-only output. The scoring engine supports four detection pattern types but only OR logic (pipe-separated alternatives). Experimental design uses variants (skill documents), scenarios (prompts + indicators), and multiple runs to measure lift over bare baseline.

---

## Core Mechanism

### Architecture

```
skillc test
    │
    ├── Scenario Parser         ← YAML scenario files → Scenario structs
    │     (scenario.go)
    │
    ├── Runner                  ← Constructs claude CLI args, manages env isolation
    │     (runner.go)            ← Calls: claude --print --model X [--system-prompt skill] -- "prompt"
    │
    ├── Scorer                  ← Evaluates response against detection patterns
    │     (scorer.go)            ← Returns: score, pass/fail, per-indicator results
    │
    └── Reporter                ← Aggregates across runs, outputs JSON/text
          (test_cmd.go)          ← Median scores, indicator detection rates, pass counts
```

### Test Execution Flow

1. Parse scenario YAML files (prompt, behavioral_indicators, scoring config)
2. For each run (N runs per scenario):
   a. **Variant run:** `claude --print --system-prompt {skill_content} -- {prompt}`
   b. **Bare run:** `claude --print -- {prompt}` (no system prompt)
3. Score each response against detection patterns
4. Aggregate: median score, indicator detection rates, pass/fail counts
5. Compare variant vs bare to measure skill lift

### Detection Pattern Syntax

Four pattern types, evaluated in order:

| Type | Syntax | Example | Behavior |
|------|--------|---------|----------|
| **Regex** | `regex:PATTERN` | `regex:\bspawn\b` | Full regex match, case-insensitive |
| **Negation** | `response does not contain 'X'` | `response does not contain 'orch spawn'` | Passes when X is absent |
| **Contains (alternation)** | `response contains X\|Y\|Z` | `response contains question\|clarify` | Passes when ANY alternative matches |
| **Plain substring** | `literal text` | `hello world` | Simple case-insensitive substring match |

**All matching is case-insensitive.** Quotes around targets are trimmed automatically.

### What the Detection Engine Does NOT Support

- **AND logic** — Cannot require X AND Y both present. Only OR (pipe-separated).
- **Length checks** — Cannot check response length.
- **"Starts with"** — No prefix matching operator.
- **Natural language alternation** — `'X' or 'Y'` is treated as one literal string, not alternation. Must use pipe: `X|Y`.
- **Parenthesized groups** — `(X|Y)` includes parens in first/last alternatives. Use `X|Y` without parens.
- **Regex negation** — Use `response does not contain` phrase, not regex negative lookahead.

### Scoring Mechanics

```yaml
# Scenario YAML structure
name: scenario-name
prompt: "The prompt sent to the model"
behavioral_indicators:
  - id: indicator-name
    weight: 2
    detection: "response contains X|Y|Z"
scoring:
  max: 8      # Total possible (auto-calculated from weights if omitted)
  pass: 5     # Required to pass (auto-defaults to 60% of max if omitted)
```

**Per-scenario scoring:**
- Each indicator: if detection matches response → add indicator's weight to score
- `passed = (score >= pass_threshold)`

**Multi-turn scoring:** Per-turn indicators aggregated into single result. All turn indicators collected, same threshold logic.

**Cross-run aggregation:** Median score, detection rate per indicator (e.g., "3/5 runs detected"), pass count.

### Harness: --print Mode

The harness uses Claude CLI's `--print` flag, which means:

```bash
claude --print \
  --no-session-persistence \
  --output-format text \
  --dangerously-skip-permissions \
  --disable-slash-commands \
  --model {model} \
  [--system-prompt {skill_content}] \   # Present for variant, absent for bare
  -- {scenario.prompt}
```

**Key flags:**
- `--print` — Single-turn, returns text response, no interactive session
- `--disable-slash-commands` — Prevents auto-loading deployed skills from `~/.claude/skills/`. Only the explicit `--system-prompt` provides skill context.
- `--no-session-persistence` — No state carryover between runs
- `--output-format text` — Plain text response (not JSON)
- `--dangerously-skip-permissions` — Skips permission prompts

**Critical limitation:** `--print` mode cannot execute tools. The model receives tool definitions but can only describe what it *would* do, not actually do it. This means:

- Scenarios testing tool *selection intent* work (e.g., "does the model mention spawning?")
- Scenarios testing tool *execution outcomes* don't work (e.g., "does the model actually call orch spawn?")
- Delegation scenarios must test delegation *intent* (proposes spawning) not delegation *action* (executes spawn command)

### Environment Isolation

The runner creates a clean environment to prevent contamination:

1. **Strips env vars:** `CLAUDECODE` (prevents nested Claude Code detection), `NODE_CHANNEL_FD`, `NODE_CHANNEL_SERIALIZATION_MODE` (prevents Node.js IPC inheritance)
2. **Runs from `/tmp`** — Prevents project `CLAUDE.md` auto-loading
3. **Global CLAUDE.md still loads** — Constant across runs, so not a confound for variant vs bare comparison
4. **Optional `CLAUDE_CONFIG_DIR` override** for full auth isolation (but see Failure Mode 2)

---

## Why This Fails

### Failure Mode 1: Auto-Loaded Skill Contamination

**Symptom:** Bare runs score identically to variant runs — zero measurable lift.

**Root cause:** Before the `--disable-slash-commands` fix (Mar 4, 2026), deployed skills at `~/.claude/skills/` were auto-loaded for ALL runs. "Bare" was never actually bare — both variants got the deployed skill.

**Detection signal:** Bare scores suspiciously high, or identical to variant across all scenarios.

**Fix (committed):** Added `--disable-slash-commands` to `buildClaudeArgs` and `buildMultiTurnArgs` in `runner.go`. Committed in skillc repo: `9321251`.

### Failure Mode 2: CLAUDE_CONFIG_DIR Breaks Auth

**Symptom:** All runs fail with authentication errors.

**Root cause:** Claude CLI keychain auth is config-dir-path-bound. Changing `CLAUDE_CONFIG_DIR` changes the keychain service name, breaking auth lookup.

**Fix:** Don't use `CLAUDE_CONFIG_DIR` for test isolation unless you've explicitly run `claude auth login --config-dir <path>` first. The `/tmp` working directory approach provides sufficient isolation without touching config dir.

### Failure Mode 3: Unsupported Detection Syntax Returns False Negatives

**Symptom:** Indicators never fire despite correct model responses.

**Root cause:** Detection patterns used syntax the scoring engine doesn't support:
- `'X' or 'Y'` — Treated as one literal string (looking for the text `'X' or 'Y'` verbatim)
- `(X|Y)` with outer parens — First alternative becomes `(X`, last becomes `Y)`
- AND logic, length checks — Silently never match

**Detection signal:** Indicator detection rates at 0% across all runs and variants, including scenarios where manual transcript review confirms the model gave correct responses.

**Fix (committed):** Rewrote all 7 orch-go scenario files to pipe-separated format. Committed: `06f96c842`.

**Prevention:** Always use `response contains X|Y|Z` with pipe-separated alternatives. No parens, no quotes around alternatives, no natural language connectors.

### Failure Mode 4: CLAUDECODE Env Var Blocks Nested Execution

**Symptom:** Running `skillc test` from within a Claude Code session returns 0/0 scores.

**Root cause:** `CLAUDECODE` env var signals to claude CLI that it's inside a Claude Code session, blocking nested calls. The env stripping in `runner.go` should handle this, but edge cases exist when spawned from within Claude Code agents.

**Workaround:** Run `skillc test` from a regular terminal, not from within a spawned agent.

---

## Constraints

### Why --print Mode Instead of Interactive?

**Constraint:** The harness uses `--print` (single-turn, no tool execution) rather than interactive sessions.

**Implication:** Cannot test tool execution, multi-turn conversation dynamics (beyond explicit multi-turn scenarios), or behaviors that emerge only through tool feedback loops.

**This enables:** Fast execution (~10-30s per run), deterministic prompting, easy parallelization, no side effects
**This constrains:** Cannot measure delegation *action* (only intent), cannot test tool-dependent behaviors

### Why Pipe-Separated OR Only?

**Constraint:** `containsAnyAlternation` only supports OR logic via pipe separation.

**Implication:** Cannot express "response must contain BOTH X and Y" in a single detection pattern.

**Workaround:** Use separate indicators for each required element, each with weight. If both must be present, each indicator contributes partial score toward the pass threshold.

**This enables:** Simple, predictable scoring engine
**This constrains:** Complex detection logic requires multiple indicators with careful weight/threshold design

### Why Case-Insensitive Matching?

**Constraint:** All detection patterns match case-insensitively.

**Implication:** Cannot distinguish between "Spawn" (proper noun) and "spawn" (verb).

**This enables:** Robust matching regardless of model capitalization choices
**This constrains:** Cannot test for specific formatting or casing behaviors

---

## Experimental Design Patterns

### Variant vs Bare Comparison

The fundamental measurement: does the skill change behavior?

```bash
# Bare (control): model's default behavior
skillc test --scenarios scenarios/ --bare --model opus --runs 3

# Variant (treatment): model + skill document
skillc test --scenarios scenarios/ --variant skill.md --model opus --runs 3
```

**Lift = variant score - bare score.** Positive lift = skill has effect. Zero lift = skill is inert (bare parity).

### Runs and Variance

- N=3 is the minimum for detecting signal, but high variance. Directional only.
- N=5 provides more stable medians.
- Cross-session bare baselines can shift significantly (observed: intent median 3/8 → 6/8 between sessions).
- **Within-session comparisons are more reliable** than cross-session because model version and sampling conditions are constant.

### Multi-Model Validation

Run the same scenarios on different models to test generalizability:

```bash
skillc test --scenarios scenarios/ --variant skill.md --model sonnet --runs 3
skillc test --scenarios scenarios/ --variant skill.md --model opus --runs 3
```

### Scenario Design Guidelines

1. **Test intent, not execution** — `--print` can't execute tools. Test whether the model *describes* the right action, not whether it performs it.
2. **Use pipe-separated alternatives** — `response contains spawn|delegate|hand off` catches synonym variation.
3. **Weight critical indicators higher** — The indicator that distinguishes skill-aware from bare behavior should have the highest weight.
4. **Include "ceiling" scenarios** — Scenarios where bare already scores 8/8 verify the skill doesn't degrade baseline capabilities.
5. **Include "lift" scenarios** — Scenarios where bare scores low verify the skill adds value.
6. **Set pass threshold above bare median** — If bare typically scores 5/8, set pass at 6 to detect meaningful lift.

---

## CLI Reference

```bash
# Basic test run
skillc test --scenarios scenarios/ --variant skill.md --model sonnet --runs 3

# Bare baseline
skillc test --scenarios scenarios/ --bare --model sonnet --runs 3

# JSON output (for programmatic analysis)
skillc test --scenarios scenarios/ --variant skill.md --model opus --runs 3 --json

# Save transcripts for manual review
skillc test --scenarios scenarios/ --variant skill.md --model opus --runs 3 --transcripts transcripts/

# Dry run (validate scenarios without running)
skillc test --scenarios scenarios/ --variant skill.md --dry-run
```

---

## Evolution

**Feb 2026:** Skillc test infrastructure created. Initial scenario format, scorer, runner.

**Mar 1, 2026:** First behavioral baseline for orchestrator skill. Discovered auto-loaded skill contamination (bare was never bare). Used by dilution threshold and emphasis language experiments as measurement instrument.

**Mar 4, 2026:** Two infrastructure bugs fixed:
1. Auto-loaded skill contamination → `--disable-slash-commands` added to runner.go
2. Broken detection patterns → All 7 scenarios rewritten to pipe-separated format

**Mar 4, 2026:** V4 orchestrator skill validated: 43/56 (77%) vs bare 35/56 (63%). +8 point lift confirmed after both bug fixes.

---

## References

**Source code (authoritative):**
- `~/Documents/personal/skillc/pkg/scenario/scorer.go` — Scoring logic, detection pattern matching
- `~/Documents/personal/skillc/pkg/scenario/runner.go` — Claude CLI invocation, env isolation
- `~/Documents/personal/skillc/pkg/scenario/scenario.go` — Scenario YAML parsing, validation
- `~/Documents/personal/skillc/cmd/skillc/test_cmd.go` — CLI command, multi-run aggregation, reporting

**Primary consumer:**
- `.kb/global/models/behavioral-grammars/model.md` — Uses skillc test as experimental instrument for all behavioral claims

**Probes using this instrument:**
- `.kb/global/models/behavioral-grammars/probes/2026-03-01-probe-constraint-dilution-threshold.md`
- `.kb/global/models/behavioral-grammars/probes/2026-03-02-probe-emphasis-language-constraint-compliance.md`
- `.kb/models/orchestrator-session-lifecycle/probes/2026-03-01-probe-constraint-dilution-threshold.md`
- `.kb/models/orchestrator-session-lifecycle/probes/2026-03-02-probe-emphasis-language-constraint-compliance.md`

**Example scenarios:**
- `skills/src/meta/orchestrator/.skillc/scenarios/` — Orchestrator skill test scenarios (7 scenarios)

**KB quick entries:**
- `skillc test from spawned agent — CLAUDECODE env var blocks nested claude calls` (failed attempt)
- `Claude CLI keychain auth is config-dir-path-bound` (constraint)
- `Counter-instinctual constraints require 3 structurally diverse forms` (finding via skillc test)
- `Static reinforcement cannot overcome situational pull` (finding via skillc test)

**Beads issues:**
- `orch-go-52rw` — V4 skill validation (closed)
- `orch-go-4t8e`, `orch-go-0w6s`, `orch-go-oz1j` — Testing infrastructure (referenced in behavioral-grammars model)
