## Summary (D.E.K.N.)

**Delta:** A Ralph loop mode for orch-go should be a spawn flag (`orch spawn --loop`) that wraps the existing rework primitive in an automated iteration cycle, with pluggable completion criteria per domain.

**Evidence:** orch-go already has rework semantics (ReworkNumber, PriorSynthesis, ReworkFeedback in spawn.Config), inline spawn blocking via WaitForSessionIdle, exploration iteration events (exploration.iterated), and the `orch wait` command. These compose into a loop controller without new primitives. The iterate-design skill and autoresearch both confirm the pattern: fresh context + prior results + completion check.

**Knowledge:** The loop pattern decomposes into three independent concerns: (1) completion criteria (scalar metrics, test pass rates, checklist convergence), (2) cross-iteration knowledge transfer (prior SYNTHESIS.md → next SPAWN_CONTEXT.md, already supported by rework fields), and (3) iteration lifecycle (spawn, wait, evaluate, decide). orch-go's unique value-add over raw `while` loops is in #2 — the SPAWN_CONTEXT system provides structured knowledge transfer that a naked `cat PROMPT.md | claude` loop cannot.

**Next:** Architect session to design `--loop` flag implementation. Key decisions: where the loop controller lives (shell script vs Go binary), how completion criteria are specified (--metric, --test-cmd, --max-iterations), and whether the controller is a new command or spawn flag.

**Authority:** architectural - Cross-component design (spawn pipeline, rework, wait, events), requires orchestrator synthesis

---

# Investigation: Operationalizing Ralph Loop as orch-go Spawn Mode

**Question:** How should orch-go operationalize the Ralph Wiggum loop (`while :; do cat PROMPT.md | claude-code; done`) as a first-class pattern? What completion criteria work, what value does orch-go add over raw loops, and should this be a spawn mode, skill type, or daemon behavior?

**Started:** 2026-03-25
**Updated:** 2026-03-25
**Owner:** investigation agent (orch-go-pk7ds)
**Phase:** Complete
**Next Step:** Architect session to design --loop flag
**Status:** Complete

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-22-inv-investigate-karpathy-autoresearch-48k-stars.md | extends | yes | - |
| .kb/threads/2026-03-22-constraint-first-design-orchestration-wrong.md | extends | yes | - |

---

## Findings

### Finding 1: orch-go already has the primitives for a loop — they just aren't composed

**Evidence:** The codebase contains all the building blocks needed for iterative spawning:

1. **Rework context fields** (`pkg/spawn/config.go`): `ReworkFeedback string`, `ReworkNumber int`, `PriorSynthesis string` — these propagate cross-iteration knowledge into SPAWN_CONTEXT.md
2. **Inline blocking** (`pkg/orch/spawn_modes.go:66-109`): `runSpawnInline()` calls `WaitForSessionIdle()` which polls until busy→idle transition — this is the "wait for iteration to complete" primitive
3. **Wait command** (`cmd/orch/wait.go`): `orch wait <beads-id> --phase Complete --timeout 30m` blocks until a phase transition — this is the "wait for agent to signal done" primitive
4. **Exploration iteration events** (`pkg/events/logger.go:899-928`): `ExplorationIteratedData` tracks iteration number, gaps addressed, new workers spawned — event tracking for multi-iteration patterns already exists
5. **Rework command** (`cmd/orch/rework_cmd.go`): `orch rework <id> "feedback"` reopens an issue and spawns a fresh workspace with prior context — this IS a manual single-step loop

What's missing is a **controller** that composes: spawn → wait → evaluate completion criteria → decide (keep/discard/continue) → re-spawn with prior context.

**Source:** `pkg/spawn/config.go:ReworkFeedback/ReworkNumber/PriorSynthesis`, `pkg/orch/spawn_modes.go:66-154`, `cmd/orch/wait.go`, `pkg/events/logger.go:899-928`, `cmd/orch/rework_cmd.go`

**Significance:** The loop mode isn't a new subsystem — it's a thin composition layer over existing primitives. This dramatically reduces implementation risk and aligns with the constraint-first design principle from the autoresearch investigation.

---

### Finding 2: Completion criteria form a taxonomy from fully automated to human-judgment

**Evidence:** Analyzing existing domains that use iterative improvement:

| Domain | Criterion | Type | Automatable? |
|--------|-----------|------|-------------|
| ML research (autoresearch) | val_bpb ↓ | Scalar metric | Fully — grep log, compare numbers |
| OpenSCAD (iterate-design) | Facet count, gate pass, CGAL validation | Multi-gate | Fully — exit codes + JSON parsing |
| Test coverage improvement | Coverage % ↑ | Scalar metric | Fully — `go test -cover` output |
| Performance optimization | Latency/throughput ↑↓ | Scalar metric | Fully — benchmark output |
| Prompt engineering | Eval score ↑ | Scalar metric | Fully — eval harness output |
| Code quality / refactoring | Lint count ↓, accretion lines ↓ | Scalar metric | Fully — `wc -l`, linter output |
| Feature implementation | Tests pass + PRD checklist | Boolean + checklist | Partially — tests yes, PRD needs judgment |
| Investigation | Knowledge completeness | Judgment | Not automatable — needs human eval |

The fully-automatable criteria share a pattern: **run a command, parse output, compare to threshold/prior value**. This suggests the completion criterion interface should be: `evaluate(iteration_output) → {improve, regress, stall, complete}`.

**Source:** autoresearch `program.md:94-106` (val_bpb), `.harness/openscad/skills/iterate-design/SKILL.md` (facets + gates), `skills/src/worker/feature-impl/reference/phase-implementation-verification-first.md:263` (feature-impl completion)

**Significance:** The completion criterion is the key design decision. orch-go should support pluggable criteria with a standard interface, starting with the simplest: "run this command and tell me if the number went up or down."

---

### Finding 3: orch-go's value-add over raw while loops is cross-iteration knowledge transfer

**Evidence:** A raw Ralph loop (`while :; do cat PROMPT.md | claude-code; done`) gives each iteration:
- The PROMPT.md content (static)
- Whatever the agent reads from the filesystem (git state, files)
- Nothing about what previous iterations attempted or learned

orch-go's SPAWN_CONTEXT system adds:
- **PriorSynthesis**: TLDR + Delta from the previous iteration's SYNTHESIS.md (what was tried, what was learned)
- **ReworkFeedback**: Structured feedback about what to change (the "keep/discard" signal with reasoning)
- **ReworkNumber**: Which iteration this is (so the agent knows it's in a loop and can adapt strategy)
- **KBContext**: Relevant knowledge base entries that may have been updated by prior iterations
- **HotspotFiles**: Which files are accumulating complexity (prevents the loop from making accretion worse)

autoresearch compensates for this gap with `results.tsv` (a running log the agent reads each iteration) and git history (the agent can see prior commits). But those are unstructured — the agent must discover and interpret them. orch-go's approach is structured injection.

**Source:** `pkg/spawn/context.go:131-165` (contextData struct with rework fields), autoresearch `program.md:64-72` (results.tsv as cross-iteration memory)

**Significance:** This is the core competitive differentiator. Raw while loops have no knowledge transfer mechanism beyond filesystem state. orch-go's SPAWN_CONTEXT provides structured, curated cross-iteration context. This matters most in domains where "what was tried" is as important as "what to try next" — like prompt engineering, investigation, and refactoring.

---

### Finding 4: A spawn flag (--loop) beats a new skill type or daemon behavior

**Evidence:** Three options evaluated:

**Option A: `orch spawn --loop` flag**
- Adds iteration to any skill (investigation, feature-impl, architect)
- Uses existing spawn pipeline — just wraps it in a loop controller
- Completion criteria specified via flags: `--loop-metric "go test -cover"`, `--loop-max 10`, `--loop-threshold 80`
- Fresh context per iteration via the existing rework flow
- Precedent: `--explore` flag already transforms spawn behavior (swaps skill to exploration-orchestrator)

**Option B: New "iterative" skill type**
- Would require a new skill definition format with iteration semantics
- Overlaps with existing skills — every skill could benefit from iteration
- Creates skill explosion (feature-impl-iterative, investigation-iterative, etc.)
- Doesn't leverage the existing spawn/rework pipeline

**Option C: Daemon behavior**
- Daemon already has triage → spawn → complete → triage cycle
- But daemon operates at task level, not iteration level within a task
- Adding iteration logic to daemon conflates two concerns (work routing vs work improvement)
- `.kb/decisions/2026-02-26-plan-mode-incompatible-with-daemon-spawned-agents.md` established that daemon should route, not execute complex patterns

**Source:** `cmd/orch/spawn_cmd.go:64-67` (explore flags as precedent), `cmd/orch/spawn_cmd.go:246-267` (explore flag swaps skill name), `.kb/decisions/2026-02-26-plan-mode-incompatible-with-daemon-spawned-agents.md`

**Significance:** `--loop` as a spawn flag is the natural fit. It follows the `--explore` precedent, composes with any skill, and doesn't require new primitives.

---

### Finding 5: Fresh context per iteration is essential — and SPAWN_CONTEXT already supports it

**Evidence:** Both autoresearch and iterate-design use fresh context per iteration:
- autoresearch: Each iteration gets a fresh Claude context. The agent re-reads program.md, results.tsv, and the codebase from scratch. This is by design — Karpathy notes that "the context window is your RAM" and you want to flush it between experiments.
- iterate-design: Each parameter sweep is independent. The skill template says "NEVER modify the source .scad file" — each iteration works from the same pristine source with different parameters.

orch-go's `orch rework` already creates a **fresh workspace** with a **new SPAWN_CONTEXT.md** that includes `PriorSynthesis` from the previous workspace. This is exactly the "fresh context + prior knowledge" pattern that makes iterative loops effective.

The key difference from a raw while loop: in `while :; do cat PROMPT.md | claude-code; done`, the PROMPT.md is static. In orch-go's loop, each iteration's SPAWN_CONTEXT.md is dynamically generated with updated knowledge from the prior iteration.

**Source:** autoresearch `program.md:112` ("NEVER STOP"), `pkg/spawn/context.go:85-165` (GenerateContext with rework fields), `cmd/orch/rework_cmd.go:62-80` (rework creates fresh workspace)

**Significance:** The SPAWN_CONTEXT system turns the Ralph loop from a stateless retry into a stateful improvement loop. Each iteration starts fresh (context window flushed) but informed (prior results injected).

---

### Finding 6: Applicable domains beyond ML and OpenSCAD

**Evidence:** Domains with clear iterative improvement patterns:

**Strong fit (scalar metric, fully automatable):**
1. **Test coverage improvement**: `go test -cover` → parse % → iterate until threshold. Each iteration reads prior coverage gaps.
2. **Performance optimization**: Benchmark output → parse latency/throughput → iterate until target. Agent tries different optimizations each iteration.
3. **Prompt engineering**: Eval harness score → iterate prompts until eval score threshold. Prior results guide next prompt variation.
4. **Lint/quality improvement**: Linter count → iterate fixes until zero. Straightforward automation.
5. **Bundle size reduction**: Webpack/vite output → parse bundle size → iterate until target.

**Medium fit (structured criteria, partially automatable):**
6. **Accessibility compliance**: Axe/Lighthouse scores → iterate until WCAG conformance. Automated scoring with some judgment calls.
7. **Security hardening**: SAST tool findings → iterate fixes until zero critical findings.

**Weak fit (requires human judgment):**
8. **Documentation quality**: No scalar metric. Could use word count or coverage heuristics but quality is subjective.
9. **UX improvement**: Lighthouse scores are partial proxies. Real UX evaluation needs humans.

**Source:** Analysis of existing skill completion criteria across codebase skills

**Significance:** The strongest domains for loop mode are those with scalar metrics — exactly the autoresearch pattern. orch-go should start with test coverage (already has `go test` in the workflow) and expand to performance/prompt engineering.

---

## Synthesis

**Key Insights:**

1. **The loop is a composition, not a primitive.** orch-go already has spawn, wait, evaluate (via beads comments), and re-spawn with context (via rework). The loop controller is a thin orchestrator that composes these. This is analogous to how `--explore` composes spawn + judge + synthesize without new primitives.

2. **Cross-iteration knowledge transfer is the killer feature.** Raw while loops flush context completely between iterations. autoresearch compensates with results.tsv. orch-go's SPAWN_CONTEXT system provides structured, curated knowledge injection — `PriorSynthesis`, `ReworkFeedback`, `ReworkNumber` — that makes each iteration smarter than the last.

3. **Completion criteria determine the domain fit.** Domains with scalar metrics (coverage %, latency, eval score) are natural loop candidates. Domains requiring judgment (documentation quality, UX) are poor fits. The interface should be: "give me a command that outputs a number, and tell me if higher or lower is better."

4. **Fresh context per iteration is a feature, not a limitation.** Both autoresearch and the iterate-design skill confirm: flushing the context window between iterations prevents the agent from getting stuck in local optima. orch-go's rework flow already creates fresh workspaces with prior context injection — this is the right architecture.

**Answer to Investigation Question:**

orch-go should operationalize the Ralph loop as **`orch spawn --loop`**, a spawn flag that wraps the existing rework primitive in an automated iteration cycle. The implementation:

1. **Flag set**: `--loop`, `--loop-cmd "go test -cover"` (evaluation command), `--loop-target 80` (threshold), `--loop-max 10` (max iterations), `--loop-direction up|down` (whether higher or lower is better)
2. **Controller**: After each iteration completes (detected via `orch wait`), run the eval command, compare to prior/target, and either stop (target met or regression) or re-spawn with updated `PriorSynthesis` containing the eval results
3. **Knowledge transfer**: Each iteration's SPAWN_CONTEXT.md includes prior eval results, iteration number, and what was tried — using the existing `ReworkFeedback`/`PriorSynthesis`/`ReworkNumber` fields
4. **Observability**: Use `exploration.iterated` events to track loop progress, expose via `orch status`

The primary value-add over raw while loops: **structured cross-iteration knowledge, domain portability (any skill can loop), observability (events + beads tracking), and guardrails (max iterations, regression detection).**

---

## Structured Uncertainty

**What's tested:**

- Existing rework primitives support cross-iteration context (verified: read ReworkFeedback/PriorSynthesis/ReworkNumber in spawn.Config)
- WaitForSessionIdle provides completion detection (verified: read client.go:631-659)
- Exploration events track iteration metadata (verified: read logger.go:899-928)
- Both autoresearch and iterate-design use fresh-context-per-iteration pattern (verified: read source files)

**What's untested:**

- Whether the rework flow is fast enough for tight loops (rework creates workspace, generates context — overhead unknown)
- Whether PriorSynthesis is rich enough for effective cross-iteration knowledge transfer (may need additional fields)
- Whether scalar metrics actually improve with iterative agent work (no empirical evidence from orch-go agents)
- What happens when the loop controller and daemon interact (daemon may try to triage loop-spawned issues)

**What would change this:**

- If rework overhead is >30s per iteration, the loop controller might need a lighter-weight re-spawn path
- If agents don't effectively use PriorSynthesis to avoid repeating failed approaches, the knowledge transfer hypothesis fails
- If the daemon claims loop-spawned issues, a `loop:managed` label or flag would be needed to exempt them

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add `--loop` flag to spawn command | architectural | Cross-component (spawn, rework, wait, events), requires design |
| Completion criteria interface | architectural | New abstraction that affects multiple packages |
| Start with test coverage domain | implementation | Tactical first use case within existing patterns |

### Recommended Approach: `--loop` as spawn flag with pluggable eval command

**Why this approach:**
- Follows `--explore` precedent (spawn flag that transforms behavior)
- Composes existing primitives (spawn + wait + rework) without new subsystems
- Domain-portable via pluggable eval command
- Fresh context per iteration via rework flow

**Trade-offs accepted:**
- Loop controller adds ~200-400 lines to spawn pipeline (new file, not sprawl in existing files)
- Rework overhead per iteration (workspace creation, context generation) may be 10-30s
- First version only supports scalar metrics (covers ML, testing, performance — not investigation/design)

**Implementation sequence:**
1. **Architect session**: Design loop controller placement (new `pkg/orch/loop.go` or `cmd/orch/loop_cmd.go`), flag set, eval interface
2. **Implement**: Loop controller with spawn → wait → eval → re-spawn cycle
3. **First domain**: Test coverage improvement (`--loop-cmd "go test -cover ./..." --loop-target 80 --loop-direction up`)
4. **Extend**: Add `--loop-cmd` presets for common domains (`--loop-preset coverage`, `--loop-preset benchmark`)

### Alternative Approaches Considered

**Option B: New skill type with iteration semantics**
- **Pros:** Clean separation, iteration is explicit in skill definition
- **Cons:** Skill explosion, duplicates existing skills, doesn't leverage rework pipeline
- **When to use instead:** If iteration semantics need to be radically different per domain (unlikely)

**Option C: Daemon-driven loop behavior**
- **Pros:** Autonomous, no human trigger needed
- **Cons:** Conflates routing and execution, daemon should route not iterate, decision against plan-mode in daemon already exists
- **When to use instead:** Never for tight loops. Possibly for long-horizon improvement goals (improve coverage over days, not minutes)

**Rationale for recommendation:** `--loop` as spawn flag is the simplest composition of existing primitives that delivers the core value (automated iteration with knowledge transfer). It follows established precedent (`--explore`), avoids new subsystems, and supports the domains with the clearest ROI first.

---

## References

**Files Examined:**
- `pkg/spawn/config.go` — Spawn config with rework fields (ReworkFeedback, PriorSynthesis, ReworkNumber)
- `pkg/orch/spawn_modes.go` — Dispatch logic, inline blocking, tmux, headless, claude modes
- `pkg/orch/spawn_types.go` — SpawnInput/SpawnContext/Explore fields
- `pkg/spawn/context.go` — SPAWN_CONTEXT.md generation with dynamic rework context
- `cmd/orch/spawn_cmd.go` — Spawn command flags including --explore precedent
- `cmd/orch/rework_cmd.go` — Rework command (manual single-step loop)
- `cmd/orch/wait.go` — Wait command (blocking until phase transition)
- `pkg/opencode/client.go:631-659` — WaitForSessionIdle polling implementation
- `pkg/events/logger.go:899-928` — ExplorationIteratedData event type
- `.harness/openscad/skills/iterate-design/SKILL.md` — Structured iteration skill
- `.kb/investigations/2026-03-22-inv-investigate-karpathy-autoresearch-48k-stars.md` — autoresearch analysis

**External Documentation:**
- [karpathy/autoresearch](https://github.com/karpathy/autoresearch) — Viral Ralph loop for ML research

**Related Artifacts:**
- **Thread:** `.kb/threads/2026-03-22-constraint-first-design-orchestration-wrong.md` — Constraint-first design principle
- **Investigation:** `.kb/investigations/2026-03-22-inv-investigate-karpathy-autoresearch-48k-stars.md` — autoresearch analysis

---

## Investigation History

**2026-03-25:** Investigation started
- Initial question: How should orch-go operationalize the Ralph loop as a first-class pattern?
- Context: Spawned to investigate after autoresearch analysis and iterate-design skill development

**2026-03-25:** Analyzed existing spawn infrastructure
- Found rework primitives (ReworkFeedback, PriorSynthesis, ReworkNumber) in spawn.Config
- Found inline blocking via WaitForSessionIdle
- Found exploration iteration events already tracking multi-iteration patterns
- Found --explore flag as precedent for spawn-flag-driven behavior transformation

**2026-03-25:** Mapped completion criteria taxonomy
- Identified scalar metrics as the automatable sweet spot
- Classified domains by criterion type: fully automated → partially automated → human judgment

**2026-03-25:** Investigation completed
- Status: Complete
- Key outcome: Loop mode should be `--loop` spawn flag composing existing rework/wait/event primitives, with pluggable eval command for domain portability
