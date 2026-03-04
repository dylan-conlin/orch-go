## Summary (D.E.K.N.)

**Delta:** Claude CLI's --append-system-prompt enables skill injection at system prompt level. The ~4 behavioral constraint budget was measured in isolation (--system-prompt replaces base prompt), NOT at user level — making the production budget unknown. Additionally, skillc and orch spawn have duplicated agent-running machinery that should converge: skillc owns scenarios/scoring, orch spawn owns agent execution.

**Evidence:** skillc test uses --system-prompt (full replacement); orch spawn pipes SPAWN_CONTEXT.md via stdin (user level); Claude CLI supports both flags in all modes; current spawns double-load skills; Feb 24 probe proved intent ≠ action compliance (--print measures wrong thing).

**Knowledge:** Three injection levels form a hierarchy (isolated system > appended system > user level), budgets unmeasured. Two measurement modes needed: print (intent) and interactive (action). The interactive tier should delegate to orch spawn (eliminates duplicated machinery, matches production by construction).

**Next:** Phase 1: Implement --append-system-prompt in orch spawn (serves both production and testing). Phase 1b: Add --injection-mode to skillc quick tier. Run Tier 1 experiment (36 runs, ~$18). If signal found, run Tier 2 interactive experiment via orch spawn (18 runs, ~$27-54).

**Authority:** architectural - Cross-component change (spawn system + skillc + behavioral model) with duplicated machinery to converge and unknown empirical outcomes.

---

# Investigation: Design System-Prompt Skill Injection for orch spawn

**Question:** Should orch spawn inject skill content at system prompt level (via --append-system-prompt) instead of user level (piped stdin), and does this expand the effective behavioral constraint budget?

**Started:** 2026-03-04
**Updated:** 2026-03-04
**Owner:** og-arch-design-system-prompt-04mar-f095
**Phase:** Complete
**Next Step:** Implement experiment in skillc, then implement spawn changes
**Status:** Complete

**Patches-Decision:** N/A (new capability)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| Constraint dilution threshold probe (2026-03-01) | extends | Yes - re-read methodology, confirmed --system-prompt usage | Corrects: issue states "measured at user level" but was actually measured in isolation via --system-prompt |
| Layered constraint enforcement design probe (2026-03-02) | extends | Yes - 87 behavioral / 64 knowledge taxonomy confirmed | None |
| Emphasis language probe (2026-03-02) | deepens | Yes - emphasis provides partial lift at high density | None |
| Orchestrator skill behavioral compliance probe (2026-02-24) | confirms | Yes - 17:1 signal ratio, system prompt structural priority | None |
| Agent framework behavioral constraints landscape probe (2026-03-01) | confirms | Yes - industry consensus on infrastructure over instruction | None |

---

## Findings

### Finding 1: The ~4 budget was measured in isolation, not at user level

**Evidence:** The dilution threshold probe (2026-03-01) used `skillc test --variant variants/<name>.md --model sonnet --runs 3`. The skillc test harness passes variant content via `--system-prompt`, which REPLACES the entire Claude Code base system prompt. This means:
- The ~4 behavioral constraint budget is the **isolated ceiling** — skill content as the ONLY system prompt, with no Claude Code base instructions competing
- In production, orch spawn pipes SPAWN_CONTEXT.md (containing embedded skill content) via stdin as the first user message, with Claude Code's full base system prompt active at system level
- The actual production budget at user level is **unmeasured** and likely lower than 4

**Source:**
- skillc test harness: `skillc/pkg/scenario/runner.go` uses `args = append(args, "--system-prompt", string(content))`
- Dilution probe: `skillc test --scenarios scenarios/ --variant variants/<name>.md --model sonnet --runs 3`
- orch spawn: `pkg/spawn/claude.go:131` — `cat %q | claude --dangerously-skip-permissions`

**Significance:** The experiment must establish budgets at THREE levels, not just compare two. The isolated measurement is our known ceiling. The production (user-level) budget and the proposed (appended system-level) budget are both unknown.

---

### Finding 2: Current orch spawn has a double-loading problem

**Evidence:** When orch spawn creates an agent:
1. Skill content is loaded via `pkg/skills/loader.go` and embedded in SPAWN_CONTEXT.md (user level, piped via stdin)
2. Claude Code auto-discovers skills from `~/.claude/skills/` (also user level, loaded as available skills)

The same skill content appears twice in the agent's context — once in the piped SPAWN_CONTEXT.md and once from auto-discovery. This wastes attention budget and contributes to dilution.

**Source:**
- `pkg/spawn/context.go` template: `{{.SkillContent}}` embedded in SPAWN_CONTEXT.md
- `pkg/skills/loader.go`: loads from `~/.claude/skills/{category}/{skill}/SKILL.md`
- Claude Code auto-discovers deployed skills (same path)
- skillc test uses `--disable-slash-commands` to prevent this, but orch spawn does not

**Significance:** Any injection architecture change must eliminate double-loading. The `--disable-slash-commands` flag is available and already proven in skillc.

---

### Finding 3: --append-system-prompt preserves base behavior while --system-prompt destroys it

**Evidence:** Claude CLI help shows:
- `--system-prompt <prompt>` — "System prompt to use for the session" (REPLACES default)
- `--append-system-prompt <prompt>` — "Append a system prompt to the default system prompt" (ADDS to default)

Neither flag is restricted to `--print` mode. Both work in interactive mode (which orch spawn uses).

Using --system-prompt would lose all Claude Code base capabilities (tool usage instructions, safety guidelines, file operations, etc.). Using --append-system-prompt preserves base behavior while adding skill constraints at system level.

**Source:** `claude --help` output — both flags are general options, not --print restricted

**Significance:** --append-system-prompt is the only viable flag for production spawns. --system-prompt is appropriate for isolated testing (where we want to measure skill content without noise).

---

### Finding 4: Shell argument size is the primary implementation risk

**Evidence:** The --append-system-prompt flag takes content as a CLI string argument. Compiled skill sizes:
- Simple skills (feature-impl): ~5-10KB
- Complex skills (architect + worker-base): ~15-25KB
- Full orchestrator skill: ~50-80KB (2,368 lines)

macOS ARG_MAX is ~1MB, so even the largest skills fit. However, orch spawn uses tmux send-keys to execute the launch command, and tmux has its own buffer handling. The approach `--append-system-prompt "$(cat /path/to/file)"` uses shell command substitution, which is bounded by ARG_MAX but not by tmux buffer limits (the command substitution happens after tmux passes the command to the shell).

**Source:**
- `pkg/spawn/claude.go:165` — `tmux.SendKeys(windowTarget, launchCmd)`
- macOS: `getconf ARG_MAX` = 1048576 (1MB)
- Typical compiled skill sizes from `skillc deploy` output

**Significance:** The implementation can use file-based command substitution (`$(cat file)`) safely for all current skill sizes. If skills grow beyond ~500KB, a different approach would be needed (but this is far beyond current sizes).

---

### Finding 5: Three-level attention hierarchy maps to specific injection mechanisms

**Evidence:** Based on the behavioral grammars model and Claude Code architecture:

| Level | Mechanism | Content | Priority |
|-------|-----------|---------|----------|
| Base system prompt | Claude Code hardcoded | Tool usage, safety, file ops | Highest |
| Appended system prompt | --append-system-prompt | Skill behavioral constraints | High (structural priority, same level as base) |
| User-level context | Piped stdin / auto-discovery | Task description, KB context, skill content (current) | Lower (after system prompt in attention hierarchy) |

The Feb 24 probe confirmed "system prompt has structural priority over user-level content" with a 17:1 signal ratio. If --append-system-prompt content inherits this structural priority, skill constraints would compete with base instructions (high attention) rather than with all user-level content (diluted attention).

**Source:**
- Behavioral grammars model, Claim 6: "System prompt has structural priority over skill content"
- Feb 24 probe: 17:1 signal ratio for system prompt → Task tool vs skill → orch spawn
- Claude Code architecture: system prompt processed before user messages

**Significance:** This is the core hypothesis. If appended content gets structural priority, the effective behavioral constraint budget in production should be higher than at user level. The experiment tests this.

### Finding 6: --print mode measures intent, not behavior — the identity-action gap

**Evidence:** The Feb 24 behavioral compliance probe established a critical gap: agents comply with identity constraints ("I am an orchestrator, I delegate") while failing at action constraints (never actually calls orch spawn, uses Task tool instead). All existing dilution measurements use skillc's --print mode, which only captures text output — what the agent SAYS it would do, not what it actually DOES with tools.

The dilution threshold probe's key indicator "proposes-delegation" is text-based: it checks whether the agent's text output mentions delegating to another agent. But in production, what matters is whether the agent actually calls the right tool (orch spawn via Bash) vs the wrong tool (Task/Agent). The 17:1 signal ratio from the base system prompt means the agent may DESCRIBE delegation while EXECUTING Task tool calls.

This means the ~4 constraint budget might be measuring TEXT compliance, not TOOL compliance. The actual behavioral budget (measured by tool call patterns in interactive mode) could be materially different — likely lower, since action constraints face the additional barrier of the base system prompt's tool promotion.

**Source:**
- Feb 24 probe: "Action constraints fail because they require suppressing default behaviors" — identity compliance ≠ action compliance
- skillc test harness: `--print` mode, text output only, no tool execution
- Dilution threshold probe: all indicators are text-pattern-based

**Significance:** This opens a second experimental dimension: not just injection LEVEL (system vs user) but measurement MODE (print/intent vs interactive/action). A complete experiment needs both dimensions to understand whether system-level injection improves actual tool-call behavior, not just text compliance.

---

### Finding 7: skillc and orch spawn have duplicated agent-running machinery — converge on orch spawn

**Evidence:** skillc's test runner (`skillc/pkg/scenario/runner.go`) independently implements agent-running machinery: `buildClaudeArgs`, env isolation, CLI invocation, output capture. This is parallel to what orch spawn already does (`pkg/spawn/claude.go`: `BuildClaudeLaunchCommand`, env vars, tmux integration, workspace setup). They diverged because skillc tests text output (--print mode) while orch runs interactive agents.

Now that we need to test actual behavior (tool calls, not just text), the goals converge. skillc needs to run agents the same way production does — which is exactly what orch spawn already does.

**Proposed separation of concerns:**

| Responsibility | Owner | Examples |
|---------------|-------|---------|
| WHAT to test | skillc | Scenarios, indicator definitions, detection patterns |
| HOW to score | skillc | Indicator scoring, threshold comparison, variance reporting |
| HOW to run agents | orch spawn | CLI args, env isolation, skill injection, workspace setup |
| WHERE to observe | skillc reads orch spawn output | Tool calls from session stream, workspace artifacts |

**Two-tier test architecture:**

**Quick tier (--print, skillc's existing runner):**
- Single-turn text output, skillc's own runner
- ~10-20 seconds per run, ~$0.10-0.50/run
- Measures: intent compliance (what agent says it would do)
- Use case: rapid iteration, constraint wording, CI

**Full tier (interactive, delegates to orch spawn):**
- Multi-turn with real tool execution, via `orch spawn` machinery
- ~60-300 seconds per run, ~$0.50-3.00/run
- Measures: action compliance (which tools agent actually calls)
- Uses production-identical injection (--append-system-prompt, --disable-slash-commands)
- Scoring observes tool calls from session/workspace output, not just text
- Use case: production-realistic validation, pre-deployment gates

**Why this eliminates the duplication problem:**
- orch spawn implements --append-system-prompt once → both production and testing use it
- Test conditions match production by construction (not by reimplementation)
- skillc focuses on its actual value: scenario design and behavioral scoring
- No risk of test/production injection divergence (Defect Class 5: Contradictory Authority Signals)

**Sandboxing for interactive tier:**
Since orch spawn already creates isolated workspaces, the sandbox is built-in. Additional safety:
- `--max-turns` limits execution time/cost
- `--allowed-tools` restricts to safe tools for test scenarios
- `--output-format stream-json` (if available in interactive mode) or transcript capture for tool call observation

**Scoring on tool calls:** Claude CLI supports `--output-format stream-json` which emits structured JSON including tool calls. This would enable indicators like:
- `calls-orch-spawn`: Agent actually invokes `orch spawn` via Bash tool
- `avoids-task-tool`: Agent never calls Task/Agent tool
- `uses-read-before-edit`: Agent reads files before editing (behavioral constraint)

**Source:**
- skillc test harness: `skillc/pkg/scenario/runner.go` — parallel CLI construction
- orch spawn: `pkg/spawn/claude.go` — production CLI construction
- Defect Class 5 (Contradictory Authority Signals): duplicated sources of truth for injection

**Significance:** This architectural insight reshapes the implementation plan. Instead of adding --injection-mode to skillc's own runner for the interactive tier, skillc delegates to orch spawn. The --append-system-prompt implementation in orch spawn serves both production and testing. The quick tier (--print) keeps skillc's own runner for speed. The injection level experiment's interactive tier runs through orch spawn, guaranteeing production-identical conditions.

---

## Synthesis

**Key Insights:**

1. **The baseline is wrong** — The ~4 constraint budget was measured in isolation (no Claude Code base prompt), so it's the ceiling, not the user-level budget. The actual production budget at user level is unknown and likely lower. This makes the experiment MORE valuable, not less — we're measuring something entirely new.

2. **Double-loading is pure waste** — Current spawns present skill content twice (SPAWN_CONTEXT.md + auto-discovery). Any architecture change must fix this by using --disable-slash-commands and removing embedded skill content from SPAWN_CONTEXT.md when system-level injection is active.

3. **The injection level experiment has three conditions, not two** — We need: isolated system (--system-prompt, existing baseline), appended system (--append-system-prompt, proposed), and user level (piped stdin, current production). The relationship between these three establishes the actual constraint budget at each production-relevant level.

4. **Intent ≠ action, and we've only measured intent** — All existing dilution measurements use --print mode (text output), but the Feb 24 probe proved identity compliance diverges from action compliance. The experiment needs an interactive tier that measures tool call patterns, not just text claims. This is a 2×3 matrix: {print, interactive} × {isolated, appended, user}.

5. **skillc and orch spawn have duplicated machinery — converge on orch spawn** — skillc rebuilt agent-running (CLI args, env isolation, invocation) that orch spawn already owns. Now that we need interactive behavioral testing, the goals converge. skillc should own WHAT to test and HOW to score; orch spawn should own HOW to run agents. The interactive test tier delegates to orch spawn, ensuring test conditions match production by construction. The quick tier (--print) keeps skillc's own runner for speed.

5. **Implementation is straightforward** — Write skill content to a file in workspace, use `--append-system-prompt "$(cat file)"`, add `--disable-slash-commands`, remove skill embedding from SPAWN_CONTEXT.md. The main risk is shell escaping, mitigated by file-based command substitution.

**Answer to Investigation Question:**

Yes, orch spawn should inject skill content at system prompt level via --append-system-prompt. The architectural rationale is sound based on the behavioral grammars model's confirmed finding that system-level content has structural priority. However, the actual budget expansion is **empirically unknown** — the ~4 budget was measured in isolation, and the production budget at both system and user levels needs measurement.

The design should proceed in two phases: (1) implement the experiment in skillc to measure the actual budget difference, (2) implement the orch spawn changes based on experimental results. If the experiment shows no meaningful difference, the architecture change still eliminates double-loading and aligns test conditions with production conditions.

---

## Structured Uncertainty

**What's tested:**

- ✅ Both --system-prompt and --append-system-prompt work in interactive mode (verified: claude --help shows no --print restriction)
- ✅ Double-loading occurs in current spawns (verified: SPAWN_CONTEXT.md embeds skill + auto-discovery loads from ~/.claude/skills/)
- ✅ The ~4 budget was measured via --system-prompt (verified: skillc test harness source code, runner.go)
- ✅ Shell command substitution handles skill-sized content (verified: ARG_MAX = 1MB, largest skill ~80KB)

**What's untested:**

- ⚠️ Whether --append-system-prompt content inherits structural priority from base system prompt (hypothesis from model, not empirically tested)
- ⚠️ Whether the effective constraint budget at system level (appended) differs from user level (production unknown)
- ⚠️ Whether Claude Code's base system prompt content interferes with appended behavioral constraints
- ⚠️ Whether --disable-slash-commands fully prevents auto-discovery in all edge cases
- ⚠️ tmux send-keys behavior with very large command strings containing command substitution
- ⚠️ Whether intent compliance (--print mode) and action compliance (interactive mode) respond differently to injection level changes
- ⚠️ Whether `--output-format stream-json` in interactive mode captures tool calls with enough fidelity to score behavioral indicators
- ⚠️ Whether sandbox isolation (temp dir + --dangerously-skip-permissions) produces realistic agent behavior

**What would change this:**

- If --append-system-prompt content has NO structural priority advantage over user-level content → architecture change still valuable for eliminating double-loading but not for budget expansion
- If Claude Code base system prompt actively interferes with appended constraints → --system-prompt (full replacement) would be needed, requiring reimplementation of base capabilities
- If experiment shows user-level budget is HIGHER than appended system level → the attention hierarchy model would need revision

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Run 3-level dilution experiment | architectural | Cross-component (skillc + behavioral model), empirical validation of unmeasured claim |
| Implement --append-system-prompt in orch spawn | architectural | Cross-component (spawn + skill loader + context generator), changes agent attention architecture |
| Add --disable-slash-commands to spawns | implementation | Simple flag addition, no cross-boundary impact |
| Remove skill embedding from SPAWN_CONTEXT.md | implementation | Template change within spawn system, follows from system-level injection |

### Recommended Approach ⭐

**Experiment-First System-Level Injection** — Run the dilution experiment first to quantify the budget at each injection level, then implement the spawn architecture change informed by results.

**Why this approach:**
- Avoids implementing based on an untested hypothesis (structural priority of appended content)
- Establishes the actual production constraint budget at user level (currently unknown)
- Creates reusable testing infrastructure for future constraint experiments
- Even if budget expansion is small, eliminates double-loading (pure improvement)

**Trade-offs accepted:**
- Delays the spawn architecture change by ~1 day (experiment first)
- Experiment costs ~$18-108 depending on model (36 runs)
- If experiment shows no difference, the architecture change is still valuable for double-loading elimination

**Implementation sequence:**

#### Phase 1: orch spawn --append-system-prompt (foundation for both production and testing)

**File: pkg/spawn/claude.go**
- Add `systemPromptFile` parameter to `BuildClaudeLaunchCommand`
- When set, add: `--append-system-prompt "$(cat /path/to/SKILL_PROMPT.md)" --disable-slash-commands`
- This serves BOTH production spawns AND the interactive test tier

**File: pkg/spawn/config.go**
- Add `SystemPromptFile string` to Config struct

**File: pkg/spawn/context.go**
- New function: `WriteSkillPromptFile(cfg *Config) error` — writes compiled skill content to `SKILL_PROMPT.md` in workspace
- When `cfg.SystemPromptFile` is set, omit `{{.SkillContent}}` from SPAWN_CONTEXT.md template

#### Phase 1b: skillc --injection-mode for quick tier (print mode only)

**File: skillc/pkg/scenario/runner.go**
- Add `InjectionMode` field to run configuration: "system" (default, existing), "append" (new), "user" (new)
- "system" mode: existing `--system-prompt <content>` (replaces base prompt)
- "append" mode: `--append-system-prompt <content>` (adds to base prompt)
- "user" mode: pipes skill content via stdin as first user message (no system prompt flag)
- All modes use `--disable-slash-commands` to prevent auto-discovery
- This only applies to the quick tier (--print mode) — skillc's own runner

**File: skillc/cmd/skillc/test_cmd.go**
- Add `--injection-mode` CLI flag: "system" | "append" | "user"
- Default: "system" (preserves existing behavior)

**Experiment execution — Two-tier approach:**

**Tier 1: Intent measurement (--print mode, existing infrastructure)**
```bash
# Condition A: Isolated system (existing baseline, ~4 budget ceiling)
skillc test --scenarios scenarios/ --variant variants/5C.md --model sonnet --runs 3 --injection-mode system

# Condition B: Appended system (proposed production)
skillc test --scenarios scenarios/ --variant variants/5C.md --model sonnet --runs 3 --injection-mode append

# Condition C: User level (current production approximation)
skillc test --scenarios scenarios/ --variant variants/5C.md --model sonnet --runs 3 --injection-mode user
```

Run all three conditions at 4 density levels (1C, 2C, 5C, 10C) × 3 runs = 36 total runs.
Estimated cost: ~$18 (Sonnet). Timeline: ~2-3 hours.

**Tier 2: Action measurement (interactive mode, new infrastructure)**
```bash
# Same three conditions but with --test-mode interactive
skillc test --scenarios scenarios/ --variant variants/5C.md --model sonnet --runs 3 \
  --injection-mode append --test-mode interactive --output-format stream-json
```

Interactive tier uses:
- `--output-format stream-json` to capture tool calls as structured data
- Sandbox directory (temp git repo) with `--dangerously-skip-permissions`
- Tool-call-based indicators: `calls-delegation-tool`, `avoids-restricted-tool`
- Multi-turn execution (agent runs until completion or max-turns limit)

Run focused subset: 2 density levels (5C, 10C) × 3 injection modes × 3 runs = 18 runs.
Estimated cost: ~$27-54 (Sonnet). Timeline: ~3-5 hours.

**Phasing recommendation:** Run Tier 1 first. If results show clear signal at injection level, proceed to Tier 2 to validate intent vs action alignment. If Tier 1 shows no difference, Tier 2 may still show divergence (system-level injection might affect tool selection more than text output).

**Expected outcome matrix (Tier 1 — Intent):**

| Density | Isolated System (A) | Appended System (B) | User Level (C) |
|---------|--------------------|--------------------|----------------|
| 1C | 8/8 (known) | ? | ? |
| 2C | 8/8 (known) | ? | ? |
| 5C | 8/8 w/variance (known) | ? | ? |
| 10C | 5/8 bare parity (known) | ? | ? |

**Expected outcome matrix (Tier 2 — Action, focused):**

| Density | Appended System (B) | User Level (C) | Delta |
|---------|--------------------|--------------------|-------|
| 5C | ? (tool calls) | ? (tool calls) | ? |
| 10C | ? (tool calls) | ? (tool calls) | ? |

**Key comparisons:**
- If B > C at 5C and 10C in BOTH tiers → system-level injection expands production budget for both intent and action
- If B > C in Tier 1 but B ≈ C in Tier 2 → system-level injection only improves text compliance, not actual behavior (the intent-action gap persists)
- If B ≈ C in Tier 1 but B > C in Tier 2 → system-level injection primarily affects tool selection, not text reasoning
- If A > B > C across both tiers → each level degrades from isolated ceiling, with measurable step-down

#### Phase 2: orch spawn Architecture Change

**File: pkg/spawn/config.go**
```go
type Config struct {
    // ... existing fields ...

    // SystemPromptFile is the path to the compiled skill content file
    // for injection via --append-system-prompt. When set, skill content
    // is NOT embedded in SPAWN_CONTEXT.md (prevents double-loading).
    SystemPromptFile string
}
```

**File: pkg/spawn/context.go**
- New function: `WriteSkillPromptFile(cfg *Config) error` — writes compiled skill content to `SKILL_PROMPT.md` in workspace
- Modify `GenerateSpawnContext()`: when `cfg.SystemPromptFile` is set, omit `{{.SkillContent}}` from template
- Add a note in SPAWN_CONTEXT.md: `NOTE: Skill guidance is loaded at system level. Do NOT invoke it via Skill tool.`

**File: pkg/spawn/claude.go**
- Add `systemPromptFile` parameter to `BuildClaudeLaunchCommand`
- When set, add: `--append-system-prompt "$(cat /path/to/SKILL_PROMPT.md)" --disable-slash-commands`
- The `--disable-slash-commands` prevents auto-discovery (eliminates triple-loading)

**File: cmd/orch/spawn_cmd.go** (or wherever spawn orchestration lives)
- After loading skill content, write it to SKILL_PROMPT.md and set SystemPromptFile
- Add `--user-level-skill` flag as escape hatch to revert to current behavior

#### Phase 3: Interactive Test Tier (skillc → orch spawn delegation)

**Architecture:** skillc defines scenarios and scoring. orch spawn runs agents. skillc observes results.

**File: skillc/cmd/skillc/test_cmd.go**
- Add `--test-mode` flag: "quick" (default, --print, skillc runner) | "full" (interactive, orch spawn)
- For "full" mode: skillc calls `orch spawn` with the test scenario as task, compiled skill as system prompt
- skillc captures session output (transcript or stream-json) and scores against indicators

**File: skillc/pkg/scenario/scorer.go (new)**
- Tool-call-based indicator scoring: parse structured output for tool call patterns
- Intent-based scoring (existing): text pattern matching on output
- Combined scoring: report both intent and action compliance per indicator

**Integration contract:**
- skillc provides: skill content path, scenario prompt, max-turns, model
- orch spawn provides: agent execution with production-identical injection
- skillc reads: workspace artifacts, session transcript, tool call log
- Scoring happens post-hoc: skillc reads what the agent DID, not just what it SAID

**Why this matters for the experiment:**
- Tier 1 (quick, --print): skillc's own runner with --injection-mode
- Tier 2 (full, interactive): orch spawn with --append-system-prompt (production path)
- Test conditions match production BY CONSTRUCTION, not by reimplementation
- No risk of skillc and orch spawn diverging in how they inject skills

### Alternative Approaches Considered

**Option B: --system-prompt (full replacement)**
- **Pros:** Maximum isolation, no base prompt noise, already proven in skillc
- **Cons:** Destroys all Claude Code base capabilities (tool instructions, safety); would need to reimplement base behavior in every skill; production agents would lose standard Claude Code behaviors
- **When to use instead:** Only for isolated testing scenarios (which skillc already does)

**Option C: Skill content at user level + --disable-slash-commands (fix double-loading only)**
- **Pros:** Simplest change; eliminates double-loading; no new injection mechanism
- **Cons:** Doesn't test the structural priority hypothesis; leaves constraint budget at user level; misses potential improvement opportunity
- **When to use instead:** If experiment shows no budget difference between appended system and user level

**Option D: Write custom system prompt combining base + skill (--system-prompt with base reimplemented)**
- **Pros:** Full control over attention allocation; could optimize base prompt to reduce noise
- **Cons:** Fragile — must track Claude Code base prompt changes across versions; enormous maintenance burden; breaks when Claude Code updates
- **When to use instead:** Never for production; possibly for research into attention allocation

**Rationale for recommendation:** Option A (experiment-first with --append-system-prompt) is the only approach that both tests the hypothesis AND provides production value (double-loading fix). The experiment infrastructure (skillc --injection-mode) has long-term value for all future constraint experiments.

---

### Implementation Details

**What to implement first:**
- skillc --injection-mode flag (enables experiment without touching orch spawn)
- Run the 36-condition experiment
- Based on results, implement orch spawn changes

**Things to watch out for:**
- ⚠️ Shell escaping in `--append-system-prompt "$(cat file)"` — skill content with backticks, dollar signs, or double quotes could break. Use single quotes around the command substitution or write a helper that properly escapes.
- ⚠️ tmux send-keys with very long commands — the combined command (env vars + cat pipe + --append-system-prompt + other flags) could be very long. Test with the largest skill (orchestrator ~80KB).
- ⚠️ --disable-slash-commands might affect more than just skill auto-discovery — verify it doesn't disable user-invoked slash commands needed during agent operation.
- ⚠️ The "user" injection mode for skillc needs careful design — piping skill content as stdin while also providing a scenario prompt requires either concatenation or a two-message approach.
- ⚠️ Defect Class 2 (Multi-Backend Blindness): This design only applies to Claude CLI backend. OpenCode backend has a different injection path (HTTP API). Ensure the Config changes don't break OpenCode spawns.

**Areas needing further investigation:**
- How does Claude Code's base system prompt content interact with appended behavioral constraints? (answered by experiment)
- Does --disable-slash-commands disable the `/exit` command that workers use for session completion? (quick test needed)
- Can --append-system-prompt accept content from a file directly (avoiding $(cat) command substitution)? Not currently, but could be added to Claude CLI fork.

**Success criteria:**
- ✅ Experiment produces clear signal: budget comparison across 3 injection levels × 4 density levels
- ✅ Double-loading eliminated: skill content appears exactly once in agent context
- ✅ Production spawns use --append-system-prompt without breaking existing agent behavior
- ✅ skillc test harness supports all three injection modes for ongoing experimentation
- ✅ No regression in agent completion rates after architecture change

---

## Addressing Issue Questions Directly

### Q1: --append-system-prompt vs --system-prompt?

**Answer: --append-system-prompt.** --system-prompt replaces Claude Code's base instructions, destroying tool usage guidance, safety guidelines, and standard behaviors. --append-system-prompt adds to the base, preserving all standard capabilities while giving skill content structural priority.

### Q2: Should SPAWN_CONTEXT.md remain user-level while skill content moves to system level? Or should both move?

**Answer: Split.** Skill behavioral content → system level (--append-system-prompt). Task description, KB context, beads tracking, deliverables → remain user level (piped SPAWN_CONTEXT.md). The behavioral grammars model shows behavioral constraints (≤4 budget) need priority placement while knowledge content (≤50 budget) survives at user level.

### Q3: Does --disable-slash-commands become the default for spawns?

**Answer: Yes, when system-level injection is active.** This prevents triple-loading (system prompt + auto-discovery + embedded in context). Skill content should appear exactly once — in --append-system-prompt. Without --disable-slash-commands, auto-discovery would still load deployed skills at user level.

### Q4: What happens to the skillc test harness?

**Answer: Add --injection-mode flag with three modes.**
- "system" (default, existing): Isolated testing, replaces base prompt. Best for measuring constraint budget ceiling.
- "append" (new): Matches proposed production. Best for measuring production-realistic constraint budget.
- "user" (new): Current production approximation. Best for measuring current baseline.

This makes test conditions configurable rather than forcing a single mode.

### Q5: Does this change the dilution curve?

**Answer: Unknown — this is the key experiment.** The ~4 budget was measured in isolation (skillc --system-prompt, no Claude Code base prompt). The production budget at user level and at appended system level are both unmeasured. The experiment compares all three levels to quantify the actual difference.

---

## References

**Files Examined:**
- `pkg/spawn/claude.go` — Claude CLI launch command construction (lines 53-132, BuildClaudeLaunchCommand)
- `pkg/spawn/config.go` — Config struct with all spawn parameters
- `pkg/spawn/context.go` — SPAWN_CONTEXT.md template generation with embedded skill content
- `pkg/skills/loader.go` — Skill discovery and loading from ~/.claude/skills/
- `~/.kb/models/behavioral-grammars/model.md` — Behavioral grammars model with 7 claims
- `~/.kb/models/behavioral-grammars/probes/2026-03-01-probe-constraint-dilution-threshold.md` — Dilution budget measurement
- `~/.kb/models/behavioral-grammars/probes/2026-03-02-probe-layered-constraint-enforcement-design.md` — 151 constraint taxonomy
- `~/.kb/models/behavioral-grammars/probes/2026-03-02-probe-emphasis-language-constraint-compliance.md` — Emphasis lift measurement
- `~/.kb/models/behavioral-grammars/probes/2026-02-24-probe-orchestrator-skill-behavioral-compliance.md` — 17:1 signal ratio, structural priority

**Commands Run:**
```bash
# Verify Claude CLI flags
claude --help 2>&1 | grep -A2 "system-prompt\|append-system"

# Verify ARG_MAX on macOS
getconf ARG_MAX  # 1048576
```

**Related Artifacts:**
- **Model:** `~/.kb/models/behavioral-grammars/model.md` — Core behavioral grammars model with dilution claims
- **Probe:** `~/.kb/models/behavioral-grammars/probes/2026-03-01-probe-constraint-dilution-threshold.md` — ~4 budget measurement
- **Issue:** `orch-go-u2y1` — Parent issue for this design

---

## Investigation History

**2026-03-04 14:00:** Investigation started
- Initial question: Should orch spawn inject skills at system prompt level, and does this expand the constraint budget?
- Context: Discovery that Claude CLI supports --append-system-prompt; behavioral grammars model suggests system-level structural priority

**2026-03-04 14:30:** Critical finding — baseline measurement correction
- The ~4 budget was measured via --system-prompt (isolated, replacing base prompt), NOT at user level
- This changes the experiment design: need three levels instead of two
- The production budget at user level is entirely unknown

**2026-03-04 15:00:** Architecture design complete
- 5 forks navigated: flag choice, content splitting, double-loading, skillc alignment, experiment design
- All forks resolved with substrate traces
- Investigation completed with implementation recommendations

**2026-03-04 15:30:** Investigation completed
- Status: Complete
- Key outcome: Experiment-first approach recommended — measure budget at 3 injection levels before implementing spawn changes. Architecture design ready for implementation after experiment validates hypothesis.
