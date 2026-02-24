# Design: Evaluate Whether Claude Code Plan Mode Should Integrate Into Feature-Impl

**TLDR:** Claude Code's native plan mode should NOT be integrated into feature-impl's Planning phase. Plan mode requires interactive human approval (breaks daemon-spawned headless agents), clears conversation context by default (destroys SPAWN_CONTEXT instructions), and produces artifacts outside the orchestration layer's visibility. The existing phase model is architecturally superior for orchestrated agents. Plan mode is appropriate only for interactive sessions.

**Question:** Should Claude Code's native plan mode replace or augment feature-impl's prompt-driven Planning phase?

**Started:** 2026-02-24
**Updated:** 2026-02-24
**Owner:** og-arch-evaluate-whether-claude-24feb-22f4
**Phase:** Complete
**Status:** Complete

---

## Problem Framing

### Design Question

Agent 1169 (architect skill) spontaneously used Claude Code's `EnterPlanMode` tool — wrote a plan to `~/.claude/plan/`, got human approval via the interactive prompt, then executed. This raised the question: should this pattern be formalized into feature-impl's Planning phase?

### Success Criteria

A good answer must address:
1. Alignment with the existing phase model (investigation → design → implementation → validation)
2. Daemon compatibility (can headless agents use plan mode?)
3. Quality improvement vs current self-reported phases
4. Interaction with skill phase reporting protocol (bd comment lifecycle)

### Constraints

- Daemon is the primary spawn path — any phase model change must work headlessly
- SPAWN_CONTEXT.md carries operational instructions that must persist throughout the agent lifecycle
- Phase reporting via bd comments is the orchestrator's visibility mechanism
- The "Gate Over Remind" principle favors enforceable gates over prompt reminders

### Scope

- IN: Evaluating plan mode integration into feature-impl
- IN: Recommending alternative approaches to planning enforcement
- OUT: Implementing changes to feature-impl skill
- OUT: Evaluating plan mode for non-worker skills (orchestrator, meta-orchestrator)

---

## Exploration

### Fork 1: Does Plan Mode Align with the Feature-Impl Phase Model?

**Options:**
- A: Full alignment — replace feature-impl Planning with plan mode
- B: Partial alignment — use plan mode for some phases, prompt-driven for others
- C: No alignment — plan mode and feature-impl phases serve different purposes

**Substrate says:**
- Principle (Gate Over Remind): Plan mode IS a gate (blocks writes during planning) — good
- Principle (Surfacing Over Browsing): Plan mode outputs to `~/.claude/plan/` — bad, orchestrator can't surface this
- Model (agent-lifecycle-state-model): Assumes continuous observability via phase reporting
- Model (daemon-autonomous-operation): Requires fully headless operation

**Evidence:**

| Aspect | Feature-Impl Phases | Claude Code Plan Mode |
|--------|--------------------|-----------------------|
| Enforcement | Prompt-driven | Tool-level (system prompt blocks writes) |
| Approval | Orchestrator via bd comment | Human via interactive prompt |
| Artifact location | Workspace + .kb/ | `~/.claude/plan/` (ephemeral) |
| Context | SPAWN_CONTEXT preserved | DEFAULT clears context on approval |
| Observability | bd comments → orchestrator | Invisible (bash blocked → no bd comments) |
| Headless | Fully compatible | Breaks (hangs at approval gate) |

**Recommendation:** Option C — no alignment. Plan mode and feature-impl phases are architecturally different mechanisms designed for different operational contexts.

---

### Fork 2: Daemon Compatibility

**Options:**
- A: Integrate plan mode, add bypass for daemon spawns
- B: Integrate plan mode only for interactive spawns (--tmux, --inline)
- C: Don't integrate — daemon incompatibility is disqualifying

**Substrate says:**
- Model (model-access-spawn-paths): Daemon is the primary spawn path for autonomous operation
- Model (daemon-autonomous-operation): Daemon spawns are headless, no human present
- Decision (dual spawn architecture): Primary path must work headlessly

**Evidence:**
- `SpawnWork()` (daemon/issue_adapter.go:352) runs `orch work <beadsID>` — fully headless
- Plan mode's `ExitPlanMode` fires an interactive approval prompt with no programmatic bypass
- Claude Code issue #16571 requested `--plan-only` and `--plan-file` flags but was closed as duplicate of #13395 — no resolution
- A daemon-spawned agent entering plan mode would hang indefinitely at the approval gate

**Recommendation:** Option C — daemon incompatibility is disqualifying for default integration. Making feature-impl's core flow depend on plan mode would break the primary spawn path. Option B is possible but creates skill configuration complexity (plan mode for interactive, prompt-driven for headless) that adds maintenance burden for marginal benefit.

---

### Fork 3: Quality Improvement Over Current Approach

**Options:**
- A: Plan mode significantly improves planning quality — worth integration cost
- B: Plan mode marginally improves planning quality — not worth cost
- C: Current approach is already equivalent or better

**Substrate says:**
- Principle (Provenance): Feature-impl produces durable artifacts with provenance (.kb/ files, design docs)
- Principle (Session Amnesia): Plan mode plan files are ephemeral; feature-impl artifacts survive sessions

**Evidence:**

Plan mode's quality benefits:
1. Tool-level enforcement prevents premature implementation (**real advantage**)
2. Human approval ensures alignment before execution (feature-impl already has this via Design Phase + orchestrator approval)

Feature-impl's existing quality mechanisms:
1. Step 0: Scope Enumeration — forces explicit requirement listing
2. Investigation Phase — durable exploration artifacts in .kb/
3. Design Phase — documented approach, orchestrator approval gate
4. Harm Assessment — pre-implementation ethics checkpoint
5. Self-Review Phase — mandatory quality gate before completion

**What plan mode adds that feature-impl lacks:** Mechanical enforcement of read-only during planning. Feature-impl relies on prompt instructions ("explore before coding"), which agents CAN ignore.

**What feature-impl has that plan mode lacks:** Observable progress (bd comments), durable artifacts (.kb/), context preservation, headless operation, orchestrator integration.

**Recommendation:** Option C — current approach is already better for orchestrated agents. The one genuine advantage (tool-level read-only enforcement) doesn't justify the operational costs.

---

### Fork 4: Skill Phase Reporting Interaction

**Options:**
- A: Plan mode can integrate with phase reporting via hooks
- B: Plan mode creates an unacceptable observability gap

**Substrate says:**
- Model (agent-lifecycle-state-model): Phase transitions must be observable via bd comments
- Principle (Gate Over Remind): A gate that makes the agent invisible is worse than a reminder

**Evidence:**
- During plan mode, bash is blocked → `bd comment` cannot run → no phase reporting
- Plan mode creates a "dark period" where the agent is active but invisible to orchestration
- The orchestrator monitors agent health via bd comments — silence triggers "unresponsive" detection
- Plan file location (`~/.claude/plan/`) is outside workspace → not captured in SYNTHESIS.md

**Recommendation:** Option B — the observability gap is unacceptable. An agent in plan mode would appear stuck/unresponsive to the daemon's monitoring, potentially triggering false abandon signals.

---

## Synthesis

### ⭐ RECOMMENDED: Do NOT Integrate Plan Mode Into Feature-Impl

**Why:**
1. **Daemon incompatibility is disqualifying** — plan mode requires interactive approval that no daemon-spawned agent can provide. Since daemon is the primary spawn path, this blocks the most common use case.
2. **Context clearing destroys operational instructions** — the DEFAULT plan approval option clears conversation context, losing SPAWN_CONTEXT, skill guidance, and phase reporting instructions.
3. **Observability gap breaks orchestration** — during plan mode, agents cannot report phases via bd comment, making them invisible to the orchestrator.
4. **Current approach is architecturally superior** for orchestrated agents — Investigation + Design phases produce durable, observable, context-preserving artifacts.

**Trade-off accepted:** We sacrifice plan mode's tool-level enforcement of read-only during planning. Agents CAN ignore prompt-level planning instructions and jump to implementation.

**When this would change:**
- If Claude Code adds `--plan-only` / `--plan-file` flags for programmatic plan mode (no interactive approval)
- If plan mode preserves conversation context by default (not clearing SPAWN_CONTEXT)
- If plan mode allows bd comment execution (bash unblocked for specific commands)
- All three would need to be true simultaneously

### Alternative Approach: Strengthen Prompt-Level Planning Enforcement

Instead of plan mode, strengthen the existing feature-impl skill:

1. **Explicit anti-premature-implementation instruction** in Step 0: "Do NOT create or modify source files until Phase: Implementation begins. File reads and exploration only during Planning/Investigation."
2. **Planning checkpoint gate** via bd comment: Agent MUST report `bd comment "Phase: Planning - Scope: ..."` with enumerated requirements before `orch complete` will accept the work
3. **Coaching plugin detection** (existing infrastructure in pkg/verify/): Detect premature source file writes during Investigation/Design phases and inject friction via coaching plugin messages

This approach preserves all orchestration properties while closing the "agents skip planning" gap.

### Where Plan Mode IS Appropriate

Plan mode remains appropriate for:
- **Interactive `orch spawn architect -i` sessions** — human is present in tmux, can approve plans
- **Direct Claude Code usage** — not through the orch spawn system
- **Ad-hoc exploration** — not tracked via beads, no phase reporting needed

Agent 1169's spontaneous use of plan mode was correct for its context (interactive architect session) but should not be generalized to all feature-impl spawns.

---

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when:**
- This recommendation resolves a recurring question about plan mode integration
- Future agents may attempt to integrate plan mode

**Suggested blocks keywords:**
- "plan mode"
- "claude code plan"
- "feature-impl planning"
- "planning enforcement"

---

## Recommendations

⭐ **RECOMMENDED:** Do not integrate Claude Code plan mode into feature-impl
- **Why:** Daemon incompatibility, context clearing, observability gap
- **Trade-off:** Lose tool-level read-only enforcement during planning
- **Expected outcome:** Feature-impl continues working reliably for both headless and interactive spawns

**Alternative: Strengthen prompt-level planning enforcement**
- **Pros:** Closes "agents skip planning" gap without operational costs
- **Cons:** Still prompt-level (can be ignored by agents, though less likely with explicit language)
- **When to choose:** If premature implementation becomes a recurring quality issue

**Alternative: Plan mode for interactive-only spawns**
- **Pros:** Gets tool-level enforcement for sessions with human present
- **Cons:** Creates two code paths in feature-impl (plan mode vs prompt-driven based on spawn mode), increases maintenance complexity
- **When to choose:** Only if the prompt-level approach proves insufficient AND interactive spawns are a significant percentage of feature-impl usage
