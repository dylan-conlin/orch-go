# Probe: Cross-Model Blind Spot Analysis

**Model:** completion-verification
**Date:** 2026-03-01
**Status:** Complete

---

## Question

The model states (line 56): "Agent reviewing agent code is a closed loop — same model family, same blind spots, no provenance chain." The decision (2026-02-25-no-code-review-gate) notes a revisit condition: "If a fundamentally different model family (different blind spots) becomes available for review."

Is the "closed loop" claim true for same-model review but false for cross-model review? Does the 6-model logout benchmark and recent agent completion data support or refute the idea that cross-model review would add value that same-model review cannot?

---

## What I Tested

### Test 1: 6-Model Logout Benchmark Analysis

Read the benchmark data from git history (commit 5faa937a2, `.kb/benchmarks/2026-01-28-logout-fix-6-model-comparison.md`). This benchmark ran the same debugging task (admin logout not working) across 6 models with the same context.

```bash
git show 5faa937a2:.kb/benchmarks/2026-01-28-logout-fix-6-model-comparison.md
```

### Test 2: 5 Recent Agent Completions — Pattern Analysis

Read SYNTHESIS.md and git diffs for 5 recent completions, all by Claude (Opus) models:

| Agent | Beads ID | Skill | Model | Commit |
|---|---|---|---|---|
| og-debug-fix-duration-bug-01mar-1d48 | orch-go-4yyr | systematic-debugging | claude-opus-4-6 | 1b2ce020f |
| og-debug-bug-orch-complete-01mar-243c | orch-go-v454 | systematic-debugging | (unspecified) | f7b0868bb |
| og-debug-rework-cmd-go-01mar-1e14 | orch-go-82ge | systematic-debugging | claude-opus-4-5 | 18553654c |
| og-debug-ghost-agents-cross-28feb-f2b1 | orch-go-xptz | systematic-debugging | claude-opus-4-5 | 8de20511b |
| og-inv-hotspot-system-measures-01mar-60e8 | orch-go-fln2 | investigation | claude-opus-4-5 | 5bf4e047d |

```bash
# Read each SYNTHESIS.md
# Read git diffs for each commit
git diff <commit>^..<commit>
```

### Test 3: Cross-Reference — Did Agents Miss Root Causes?

For each completion, asked: "What would a DIFFERENT model family (one with the Codex/DeepSeek diagnostic pattern from the benchmark) likely flag?"

---

## What I Observed

### Benchmark Findings: Cross-Model Divergence Is Real

The 6-model benchmark provides the strongest evidence:

| Model Family | Result | Approach |
|---|---|---|
| Codex (GPT-5.2-codex) | **PASS** | Backend cookie `path="/"` fix |
| DeepSeek | **PASS** | Backend cookie + `prompt=login` |
| Gemini Pro 2.5 | FAIL | Frontend-only (LoginPage.tsx) |
| GPT-5.2 | FAIL | Frontend-only (AdminLogin.tsx) |
| Claude Sonnet 4.5 | FAIL | Frontend-only (Login.tsx) |
| Claude Opus 4.5 | FAIL | URL routing confusion (different wrong answer) |

**Key observations:**
1. **Opus and Sonnet share a blind spot** — both missed backend root cause, both gravitated to frontend
2. **Within the same family, variant matters** — Codex (GPT-5.2-codex) passed while GPT-5.2 failed
3. **DeepSeek used 9x more tokens** (83K vs 9.4K) and found the root cause — suggests thoroughness-of-exploration is a differentiator
4. **Opus failed TWICE** on retests (Jan 28 and Jan 29) with different wrong approaches — the blind spot is consistent, not stochastic

### Agent Completion Findings: Three Shared Blind Spot Patterns

**Pattern 1: Proximal Fix Over Root Cause**

All 5 completions were by Claude Opus. Three show a pattern of fixing the proximal symptom rather than questioning the root cause:

- **Duration bug (orch-go-4yyr):** Capped display at 24h instead of fixing why sessions never expire. The agent's own "Unexplored Questions" acknowledges this: "Should `orch session` have a separate max-age expiration independent of spawn activity?" — correctly identifies the root cause but dismisses it as out of scope.

- **Ghost agents (orch-go-xptz):** Added `orch clean --ghosts` command and auto-resolve abandon — treats the symptom (stale labels) not the cause (unclean shutdown). The agent explicitly acknowledges this: "This fix treats the symptom (stale labels) not the cause (unclean shutdown)."

- **DefaultDir (orch-go-82ge):** Applied defer-restore pattern to 2 more files. This is the THIRD time the same bug class was fixed in different files (`spawn_cmd.go`, `complete_cmd.go`, now `rework_cmd.go` + `abandon_cmd.go`). No agent questioned: why is `beads.DefaultDir` a mutable global at all?

**This mirrors the logout benchmark exactly:** Opus fixes the frontend symptom while the backend root cause persists.

**Pattern 2: Same-Family Pattern Propagation Without Questioning**

The DefaultDir fix illustrates this perfectly. Three separate Opus agents, across multiple sessions, each applied `prevDefaultDir/defer` to another file. Each correctly identified the pattern. None questioned the underlying design:
- `spawn_cmd.go:419-421` (original)
- `complete_cmd.go:236-237` (orch-go-obdv)
- `rework_cmd.go` + `abandon_cmd.go` (orch-go-82ge)

The hotspot investigation (orch-go-fln2, also Opus but with investigation framing) DID identify this as a design smell: "`beads.DefaultDir` global state pattern is fundamentally unsafe for cross-project operations without defer-restore." But this was an investigation, not a debugging task — the skill frame changed the analysis depth.

**Pattern 3: Skill Frame Matters as Much as Model Family**

The hotspot investigation (orch-go-fln2) was also Opus but found 5 bugs that implementation agents missed:
1. `serve_agents_discovery.go:305` filters only `"open" || "in_progress"` — missing `"blocked"` (dashboard invisible blocked agents)
2. `synthesis_parser.go:14` regex `\w+` cannot match "spawn-follow-up" (hyphenated values silently dropped)
3. `beads.DefaultDir` set without defer in `complete_pipeline.go:103`
4. 769 bare string occurrences of skill names across 7 packages with zero shared constants
5. Private key (`pkg/certs/key.pem`) tracked in git

This suggests the "closed loop" isn't purely about model family — it's about the combination of model + task framing. Investigation-framed Opus finds things that debugging-framed Opus misses.

### What Cross-Model Review Would Likely Catch

Based on the benchmark's demonstrated divergence, a model with DeepSeek/Codex's diagnostic pattern would likely flag:

1. **Duration fix:** "Why cap the display instead of fixing session expiration? The 24h cap treats the symptom."
2. **Ghost agents:** "Adding cleanup commands is a workaround. The fix should be in the shutdown lifecycle."
3. **DefaultDir:** "This is the third time the same bug class was fixed. The mutable global is the root cause."
4. **Tmux cleanup:** "Moving defer to success path is correct, but who OWNS cleanup? LifecycleManager or runComplete?"

These are the kinds of "backend root cause" observations that Codex/DeepSeek demonstrated in the benchmark.

---

## Model Impact

- [ ] **Confirms** invariant: "Agent reviewing agent code is a closed loop — same model family, same blind spots"
- [x] **Extends** model with: Cross-model review demonstrably escapes the closed loop, AND skill-frame divergence within the same model family also produces different blind spots

### Extension Details

The model's claim is **correct but incomplete**. The evidence supports two extensions:

**Extension 1: Cross-model review has demonstrated value.** The 6-model benchmark provides concrete evidence that different model families find different root causes. Codex found the backend cookie fix; Opus/Sonnet/GPT-5.2/Gemini did not. This is not theoretical — it's measured. The decision's revisit condition ("if a fundamentally different model family becomes available for review") appears to be met, at least partially. However, the decision's reasoning about provenance remains valid: cross-model review still produces opinions, not executable evidence. The extension is: cross-model review would add **signal** (where to look), even if it can't add **provenance** (proof something is wrong).

**Extension 2: The "closed loop" has a skill-frame dimension.** The same model (Opus) with investigation framing found 5 bugs that debugging-framed Opus missed. The loop isn't just "same model family" — it's "same model family + same task frame." The current model treats this as purely a model-family property, but the evidence shows task framing is a significant factor. This suggests the existing investigation-as-review pathway (`orch spawn investigation "review orch-go-XXXX"`) may capture some cross-frame value even without cross-model deployment.

**What this does NOT change:** The decision to exclude agent-judgment gates from the completion pipeline remains sound. Cross-model review would still produce opinions without provenance. The correct application is:
1. Cross-model review as **advisory** (hotspot-like signal), not blocking
2. Investigation-framed review as an existing mechanism for cross-frame analysis
3. Expanding execution-based gates (go vet, staticcheck) remains the highest-provenance path

### Specific Model Updates Recommended

1. **Line 56 of model.md** should extend the statement: "Agent reviewing agent code is a closed loop — same model family, same blind spots, no provenance chain. Cross-model review escapes the blind spot loop but not the provenance gap."

2. **Constraints section "Why No Agent-Judgment Gates?"** should note: "Cross-model review has demonstrated value (6-model benchmark: Codex/DeepSeek found backend root causes that Opus/Sonnet/GPT-5.2/Gemini missed), but the provenance objection still applies — cross-model opinions are still opinions, not executable evidence."

3. **The decision's revisit condition** (line 95: "If a fundamentally different model family becomes available for review") should be noted as partially met — the evidence exists, but the provenance constraint hasn't changed.

---

## Notes

### Methodological Limitations

1. The 6-model benchmark is a single task (N=1 for the task, N=7 for the model runs). The pattern (Codex/DeepSeek find backend, others don't) might not generalize to all task types.

2. The 5 agent completions are all from the orch-go project with similar domain context. The blind spot patterns might be domain-specific rather than model-general.

3. Opus was retested twice on the logout benchmark and failed both times with different wrong approaches. This strengthens the "consistent blind spot" claim but is still N=2.

### Connection to Non-Anthropic Model Constraint

The prior knowledge includes a constraint: "Non-Anthropic models (GPT-4o, GPT-5.2-codex) should not be used for protocol-heavy skills (architect, investigation)." This creates a tension: the models that might provide the most valuable cross-model review perspective (Codex, DeepSeek) are explicitly constrained from protocol-heavy work due to high stall rates. The blind spot escape comes at a reliability cost.

### The Proximal-Fix Pattern as Testable Hypothesis

The three proximal-fix instances in the agent completions suggest a testable hypothesis: "Opus preferentially fixes the proximal symptom rather than the structural root cause in debugging tasks." This could be tested with a structured benchmark: present the same bug to multiple models, measure how many fix the symptom vs. the root cause.
