# Design: Layered Constraint Enforcement Architecture for Skill Documents

**Question:** How should skill constraints be classified and enforced, given that behavioral constraints fail at 10+ competing constraints and the production orchestrator skill has 50+?

**Defect-Class:** architecture-design

**Started:** 2026-03-02
**Updated:** 2026-03-02
**Owner:** og-arch-design-layered-enforcement-02mar-3240
**Phase:** Complete
**Next Step:** Implementation — Phase 1 hooks, Phase 2 skill restructuring
**Status:** Complete

**Patches-Decision:** N/A (new design, not patching existing)
**Extracted-From:** .kb/investigations/2026-03-01-inv-test-constraint-dilution-threshold.md (orch-go-89at)

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-03-01-inv-test-constraint-dilution-threshold.md | Motivating evidence | yes — behavioral dilution at 10C, production at 50+ | - |
| .kb/decisions/2026-02-26-two-layer-action-compliance.md | Prior decision | yes — infrastructure + prompt pattern established | - |
| .kb/models/architectural-enforcement/model.md | Extends | yes — four enforcement layers confirmed | This design adds constraint taxonomy and prompt budgeting to the model |
| .kb/investigations/2026-02-24-design-orchestrator-skill-behavioral-compliance.md | Root cause | yes — identity vs action compliance distinction | - |

---

## Problem Framing

### Design Question

The orchestrator skill contains 151 constraints (87 behavioral, 64 knowledge) in a 2,368-line document. Investigation orch-go-89at proved that behavioral constraints regress to bare parity at 10 competing constraints. The production skill has 50+ constraints, so all behavioral constraints are non-functional in prompt. How do we enforce them?

### Success Criteria

1. **Taxonomy**: Clear classification of every constraint type with enforcement routing
2. **Coverage**: Every behavioral constraint has an enforcement mechanism (hard gate, coaching nudge, or budgeted prompt)
3. **Budget compliance**: Prompt sections contain ≤4 behavioral constraints (the empirically-tested safe limit)
4. **Integration**: Design uses existing hook infrastructure (no new mechanism types needed)
5. **Actionable**: Implementation agents can build from this design without architectural ambiguity

### Constraints

- Must use existing Claude Code hook API (PreToolUse, PostToolUse, Stop, SessionStart, SessionEnd)
- Must use CLAUDE_CONTEXT env var for session type detection (already implemented)
- Cannot modify Claude Code system prompt
- Must maintain escape hatches per "Escape Hatches" principle
- Must avoid gate calibration death spiral (lessons from hotspot gate history)
- Hook timeout limits: 10s for PreToolUse, 15s for Stop, 30s for SessionStart

### Scope

- **IN**: Orchestrator skill constraints (most urgent, 50+ constraints)
- **IN**: Generalized constraint taxonomy applicable to all skills
- **IN**: Integration with existing hook infrastructure
- **OUT**: Implementation of new hooks (design only)
- **OUT**: Worker skill enforcement (follow-up, simpler case)
- **OUT**: Modifying Claude Code internals

---

## Exploration

### Fork 1: What is the constraint taxonomy?

**Options:**
- A: Binary (behavioral vs knowledge)
- B: Ternary (hard behavioral, soft behavioral, knowledge)
- C: Four-type (hard behavioral, coaching behavioral, judgment behavioral, knowledge)

**Substrate says:**
- Investigation orch-go-89at: Behavioral constraints regress to bare at 10, knowledge survives at 10+
- Two-Layer Action Compliance decision: Infrastructure + prompt is the established pattern
- Gate Calibration Death Spiral: Gates that are too strict create bypass culture

**RECOMMENDATION:** Option C (four-type) — maps directly to enforcement mechanisms

The four types:

| Type | Definition | Dilution Sensitivity | Enforcement |
|------|-----------|---------------------|-------------|
| **Hard behavioral** | Actions enforceable by tool/command interception | High (bare at 10) | Infrastructure: deny gates |
| **Coaching behavioral** | Actions where blocking causes worse behavior | High (bare at 10) | Infrastructure: coaching nudges |
| **Judgment behavioral** | Actions requiring reasoning, not interceptable | High (bare at 10) | Prompt: budgeted ≤4/section |
| **Knowledge** | Information additive to model defaults | Low (functional at 10+) | Prompt: no strict budget |

**Trade-off accepted:** Four types is more complex than two, but the enforcement mapping is cleaner.
**When this would change:** If coaching nudges prove as effective as hard gates, types 1 and 2 could merge.

---

### Fork 2: Which behavioral constraints move to infrastructure?

**Options:**
- A: All 87 behavioral constraints to hooks (maximum enforcement)
- B: Only hard-enforceable ones (~31) to hooks, rest stays in prompt
- C: Hard + coaching-enforceable (~59) to hooks, judgment stays in prompt (~28)

**Substrate says:**
- Principle "Infrastructure Over Instruction": Prefer hooks
- Gate Calibration Death Spiral: Over-enforcement creates bypass culture
- Investigation orch-go-89at: Behavioral constraints need <5 competitors in prompt
- gate-orchestrator-code-access.py history: Blocking code reads caused fabrication

**RECOMMENDATION:** Option C — maximize infrastructure coverage while respecting that judgment constraints can't be intercepted

**Coverage breakdown:**

| Category | Count | Examples | Enforcement |
|----------|-------|---------|-------------|
| Hard behavioral | 31 | Tool restrictions, command blocking, spawn validation | Hooks: deny |
| Coaching behavioral | 28 | Code read patterns, investigation drift, context gaps | Hooks: allow + coaching |
| Judgment behavioral | 28 | Option filtering, question framing, autonomy decisions | Prompt: budgeted |
| Knowledge | 64 | Routing rules, architecture facts, responsibility boundaries | Prompt: standard |

**Trade-off accepted:** 28 judgment constraints remain prompt-only with a ≤4 budget per section.
**When this would change:** If output pattern validation becomes reliable, some judgment constraints could become coaching constraints.

---

### Fork 3: How to handle the prompt budget for remaining behavioral constraints?

**Options:**
- A: Keep only 4 most critical behavioral constraints in a single prompt section
- B: Split into multiple themed sections of ≤4 behavioral constraints each
- C: Remove all behavioral from prompt, rely entirely on infrastructure
- D: Reformulate some behavioral constraints as knowledge, reducing the behavioral count

**Substrate says:**
- Investigation orch-go-89at: Behavioral budget is ~2-4 competitors. Knowledge budget is 10+.
- Finding 5: Cross-constraint specificity holds (constraints don't interfere with each other's targets)
- UNTESTED: Whether document-level attention budget applies across isolated sections

**RECOMMENDATION:** Option D + A — reformulate what we can as knowledge, budget the rest

**The reformulation technique:**

Some constraints phrased as prohibitions ("don't do X") can be reformulated as norms ("the pattern is Y"):

| Behavioral (fights defaults) | → Knowledge (additive) |
|------------------------------|----------------------|
| "Don't ask 'want me to complete them?'" | "Orchestrators auto-complete agents at Phase: Complete" |
| "Don't filter options without recommending" | "The presentation pattern: recommend → alternatives → tradeoffs" |
| "State intent before multi-action operations" | "The Propose-and-Act pattern: describe intent, then execute" |
| "Don't investigate when asked to design" | "Orchestrators route: investigation→architect→implementation" |

**Why this works:** Knowledge constraints are additive (layer on model defaults), so they survive dilution better. A norm ("orchestrators auto-complete") adds to what the model knows. A prohibition ("don't ask permission") fights what the model wants to do.

**Estimated reformulation:** ~20 of 28 judgment constraints can be reformulated as knowledge, leaving ~8 true behavioral constraints for prompt. With 2 themed sections of ≤4 each, this fits the budget.

**Critical behavioral constraints to keep in prompt (2 sections, ≤4 each):**

**Section 1: Core Identity Actions (top of skill, 0% depth)**
1. "Orchestrators NEVER investigate or implement — comprehend, triage, synthesize only"
2. "Default: daemon path (bd create -l triage:ready). Direct spawn requires --bypass-triage"
3. "Act silently on obvious next steps. State intent for multi-actions. Wait after presenting options"
4. "Answer the question Dylan asked, not a related one"

**Section 2: Completion Protocol (near completion phase)**
1. "Auto-complete agents at Phase: Complete without asking"
2. "Classify spawn outcome before completing: decision / follow-up / archive"
3. "Surface 'what changes for you?' not 'does this look good?'"
4. "Three-Layer Reconnection for high-impact completions"

**Trade-off accepted:** Only 8 behavioral constraints survive in prompt. The other 20 are reformulated as knowledge (which survives dilution).
**When this would change:** If section isolation testing shows that sections don't share attention budget, more sections of ≤4 would expand the effective behavioral budget.

---

### Fork 4: What new hook mechanisms are needed?

**Options:**
- A: Extend existing hooks only (no new files)
- B: Create new hooks for each gap (~6 new hooks)
- C: Create a single "orchestrator enforcement" hook that handles all checks

**Substrate says:**
- Existing hooks follow single-responsibility pattern (one hook per behavior)
- Hook timeout is 10s for PreToolUse — complex hooks risk timeout
- Maintenance: fewer files = less to maintain, but complex files are harder to debug

**RECOMMENDATION:** Option B — new hooks for each gap, following existing pattern

**New hooks needed (prioritized):**

| Priority | Hook | Trigger | Enforcement | Constraints Covered |
|----------|------|---------|-------------|-------------------|
| P1 | `gate-orchestrator-bash-write.py` | PreToolUse (Bash) | Deny | Block filesystem-modifying commands (mkdir, rm, cp, mv, tee, redirect) |
| P1 | `gate-orchestrator-git-remote.py` | PreToolUse (Bash) | Deny | Block git push/merge without explicit approval |
| P2 | `nudge-orchestrator-spawn-context.py` | PreToolUse (Bash) | Coaching | Warn on `orch spawn` without prior `kb context` |
| P2 | `nudge-orchestrator-investigation-drift.py` | PreToolUse (Read) | Coaching | Warn on sequential code file reads (>3) suggesting delegation |
| P3 | `gate-orchestrator-session-triage.py` | SessionStart | Coaching | Surface stale triage:review issues at session start |
| P3 | `gate-completion-outcome-classification.py` | PreToolUse (Bash) | Coaching | Nudge outcome classification on `orch complete` |

**Why these 6 cover the gap:**

The existing 7 enforcement hooks cover: Task tool blocking, Edit/Write/NotebookEdit blocking, bd close gating, code file read coaching, phase complete enforcement, knowledge capture gating.

The 6 new hooks cover: filesystem write blocking, git remote blocking, spawn context validation, investigation drift detection, session triage surfacing, completion outcome classification.

Together (13 hooks), they enforce 59 of 87 behavioral constraints via infrastructure, leaving 28 for prompt (of which ~20 get reformulated as knowledge, leaving 8 in the behavioral budget).

**Trade-off accepted:** 6 new hook files add maintenance burden. Each is <100 lines following existing patterns.
**When this would change:** If we build a "policy engine" hook that reads constraint rules from a config file, all 6 could consolidate into one.

---

### Fork 5: How does this integrate with existing gate-orchestrator-code-access.py?

**Options:**
- A: Keep as-is (coaching nudge for individual code file reads)
- B: Extend to detect investigation drift (sequential reads → escalated coaching)
- C: Convert to hard gate (block code reads entirely)

**Substrate says:**
- History in hook docstring: Blocking caused fabrication and wrong diagnoses
- Two-layer decision: --disallowedTools handles Edit/Write already
- Orchestrators NEED Read for .kb/, CLAUDE.md, SYNTHESIS.md
- Investigation drift is the actual behavioral concern, not individual reads

**RECOMMENDATION:** Option A + separate hook for drift — keep existing hook's calibration, add new `nudge-orchestrator-investigation-drift.py` for the sequential-read pattern

The existing hook's coaching nudge is well-calibrated:
- Individual code reads are legitimate (checking a file before spawning)
- Sequential code reads indicate investigation drift (should delegate)
- Blocking all reads caused worse behavior (fabrication)

The new drift hook would:
- Track code file read count per session (via env var or temp file)
- After 3+ code file reads, escalate from individual coaching to pattern detection:
  "You've read 4 code files this session. This looks like investigation work — consider spawning an investigation agent instead."
- Reset count when `orch spawn` is detected (legitimate context gathering before spawn)

**Trade-off accepted:** Two hooks for related behavior (read coaching + drift detection). Could be one hook, but separation follows single-responsibility pattern.

---

## Synthesis

### The Architecture: Four Enforcement Layers for Skill Constraints

```
┌─────────────────────────────────────────────────────────┐
│                  SKILL CONSTRAINT                        │
│               (151 total in orchestrator)                │
└─────────────┬───────────────────────────────┬───────────┘
              │                               │
    ┌─────────▼──────────┐          ┌─────────▼──────────┐
    │   87 BEHAVIORAL    │          │   64 KNOWLEDGE     │
    │ (fights defaults)  │          │ (additive to       │
    │                    │          │  defaults)          │
    └───┬────┬────┬──────┘          └────────────────────┘
        │    │    │                   Stays in prompt
        │    │    │                   (survives at 10+)
        │    │    │
   ┌────▼┐ ┌▼────▼┐ ┌──────────┐
   │ 31  │ │ 28   │ │ 28       │
   │HARD │ │COACH │ │JUDGMENT  │
   └──┬──┘ └──┬───┘ └────┬─────┘
      │       │          │
      ▼       ▼          ▼
  HOOKS:    HOOKS:     PROMPT:
  DENY     COACHING    BUDGETED
  (block)  (nudge)    (≤4/section)
                         │
                    ┌────┴────┐
                    │~20 can  │
                    │reformulate│
                    │as KNOW- │
                    │LEDGE    │
                    └─────────┘
```

### Layer 1: Hard Infrastructure Gates (31 behavioral constraints)

**Mechanism:** PreToolUse deny, --disallowedTools, Stop block

**Already implemented (7 constraints):**

| Constraint | Enforcement | Hook/Mechanism |
|-----------|-------------|----------------|
| No Task tool | deny | `--disallowedTools` + `gate-orchestrator-task-tool.py` |
| No Edit tool | deny | `--disallowedTools` |
| No Write tool | deny | `--disallowedTools` |
| No NotebookEdit tool | deny | `--disallowedTools` |
| No `bd close` on agent work | deny | `gate-bd-close.py` |
| Phase: Complete before exit | block | `enforce-phase-complete.py` |
| Knowledge capture before close | deny | `pre-commit-knowledge-gate.py` + `orchestrator-session-kn-gate.py` |

**New hooks needed (24 constraints):**

| Constraint Group | Hook | Constraints Covered |
|-----------------|------|-------------------|
| No filesystem writes via Bash | `gate-orchestrator-bash-write.py` | mkdir, rm, cp, mv, touch, chmod, ln, sed -i, tee, redirect (>/>>) |
| No git push/merge without approval | `gate-orchestrator-git-remote.py` | git push, git merge, git rebase (remote-affecting) |
| Spawn validation | `nudge-orchestrator-spawn-context.py` | kb context check, orientation frame check |
| Completion outcome classification | `gate-completion-outcome-classification.py` | Must classify before completing |

**Calibration principle:** Deny messages include redirect (tell agent what to do instead). Example:
```
"⚠️ ORCHESTRATOR WRITE GUARD: Filesystem modification blocked.
Orchestrators cannot modify files directly. Instead:
- To create/edit code: orch spawn feature-impl 'task description'
- To create/edit .kb/: Use approved write paths (SYNTHESIS.md, probes)
Command blocked: mkdir -p /path/to/dir"
```

### Layer 2: Coaching Infrastructure (28 behavioral constraints)

**Mechanism:** PreToolUse allow + additionalContext, SessionStart context injection

**Already implemented (1 constraint):**

| Constraint | Enforcement | Hook |
|-----------|-------------|------|
| Delegation consideration on code reads | coaching | `gate-orchestrator-code-access.py` |

**New hooks needed (27 constraints):**

| Constraint Group | Hook | Trigger | Message Pattern |
|-----------------|------|---------|----------------|
| Investigation drift | `nudge-orchestrator-investigation-drift.py` | PreToolUse (Read), >3 code files | "You've read N code files. Consider spawning investigation agent." |
| Stale triage items | `gate-orchestrator-session-triage.py` | SessionStart | "N triage:review issues pending. Consider triaging before new work." |
| Spawn context depth | `nudge-orchestrator-spawn-context.py` | PreToolUse (Bash), orch spawn without kb context | "No kb context found this session. Consider running kb context before spawning." |
| Completion quality | `gate-completion-outcome-classification.py` | PreToolUse (Bash), orch complete | "Remember to classify outcome: decision / follow-up / archive" |

**Calibration principle:** Coaching nudges ALLOW the action but add context. Never block for coaching-level constraints. The model decides whether to heed the nudge.

### Layer 3: Prompt Layer (Budgeted Behavioral + Full Knowledge)

**Behavioral budget:** ≤4 per section, 2 sections maximum = 8 behavioral constraints in prompt

**Section 1: Core Identity Actions (position: top of skill, 0% depth)**

These 4 constraints are the most critical for orchestrator role identity. They're positioned at the top of the skill where attention is highest.

1. **"Orchestrators NEVER investigate or implement — comprehend, triage, synthesize only"**
   - WHY in prompt: Identity-action fusion. This is the #1 behavioral constraint and gets infrastructure backup from tool restrictions + coaching nudges.

2. **"Default: daemon path (bd create -l triage:ready). Direct spawn requires --bypass-triage"**
   - WHY in prompt: Spawn routing is the primary action. Infrastructure can validate but can't decide WHEN to spawn.

3. **"Act silently on obvious next steps. State intent for multi-actions. Wait after presenting options"**
   - WHY in prompt: Autonomy model can't be gated — it's a judgment call per interaction.

4. **"Answer the question Dylan asked, not a related one"**
   - WHY in prompt: Response targeting is pure judgment. No infrastructure can detect "answering the wrong question."

**Section 2: Completion Protocol (position: near completion phase, ~80% depth)**

These 4 constraints govern the completion handoff. Positioned near where they're needed.

1. **"Auto-complete agents at Phase: Complete without asking"**
   - WHY in prompt: Judgment about when to auto-complete vs. wait.

2. **"Classify spawn outcome: decision / follow-up / archive"**
   - WHY in prompt: Gets coaching backup from infrastructure, but classification is judgment.

3. **"Surface 'what changes for you?' not 'does this look good?'"**
   - WHY in prompt: Framing constraint that can't be detected by infrastructure.

4. **"Three-Layer Reconnection for high-impact completions"**
   - WHY in prompt: Complex output pattern that requires understanding of "high-impact."

**Knowledge constraints (64):** Stay in prompt, organized by topic. No strict budget — they survive dilution at 10+ competitors per the evidence.

### Layer 4: Behavioral → Knowledge Reformulation (~20 constraints)

These constraints were behavioral (fighting defaults) but can be reformulated as knowledge (additive norms):

| Original (Behavioral) | Reformulated (Knowledge) | Why It Works |
|-----------------------|--------------------------|-------------|
| "Don't ask 'want me to complete them?'" | "Orchestrators auto-complete agents at Phase: Complete — this is the standard workflow" | Describes the norm, not the prohibition |
| "Don't filter options without recommending" | "The presentation pattern: lead with recommendation, then alternatives with tradeoffs" | Teaches the pattern, doesn't fight defaults |
| "State intent before multi-action operations" | "The Propose-and-Act protocol: 1) describe intent, 2) execute" | Names the protocol as knowledge |
| "Don't investigate when asked to design" | "Investigation → Architect → Implementation is the enforced sequence" | States the rule as system knowledge |
| "Don't dump all options — surface only viable ones" | "Option curation: orchestrators filter to 2-3 viable options before presenting" | Teaches the curation step |
| "Don't cascade questions — ask once then act" | "The one-question pattern: gather context, ask one targeted question, act on answer" | Names the anti-pattern as protocol |
| "Don't rescue workers mid-flow" | "Worker lifecycle: spawn → monitor → complete. Mid-session rescue = new spawn, not injection" | Describes the lifecycle rule |
| "Don't use push/deploy without approval" | "Git operations: all remote-affecting operations (push, merge) require user approval" | States the policy |
| "Complete agents without asking permission" | "Agent completion is autonomous: Phase: Complete → orch complete → synthesize" | Describes the workflow |
| "Don't spawn when context is missing" | "Pre-spawn protocol: kb context → check existing issues → spawn with ORIENTATION_FRAME" | Names the protocol |

**Why reformulation works:** Knowledge constraints are additive — they add information the model didn't have. Behavioral constraints are subtractive — they fight what the model wants to do by default. The same rule, phrased as "how things work here" instead of "what you can't do," switches from subtractive to additive compliance.

**Caveat:** Reformulation reduces enforcement strength. A knowledge constraint like "orchestrators auto-complete agents" is weaker than a behavioral prohibition "NEVER ask permission to complete." But at 50+ constraints, the prohibition has zero effect anyway (bare parity). The knowledge formulation at least survives dilution.

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Constraint taxonomy (4 types) | architectural | Affects all skill design, not just orchestrator |
| New hooks (P1: bash-write, git-remote) | implementation | Extends existing pattern, no new architecture |
| New hooks (P2-P3: coaching hooks) | implementation | Extends existing pattern, no new architecture |
| Prompt restructuring (budgeted sections) | architectural | Restructures orchestrator skill based on dilution evidence |
| Behavioral → knowledge reformulation | architectural | New technique for all skills, not just orchestrator |

### ⭐ RECOMMENDED: Phased Implementation

**Phase 1: Hard gates (1-2 days, 2 new hooks)**
- Create `gate-orchestrator-bash-write.py` (~80 lines)
- Create `gate-orchestrator-git-remote.py` (~60 lines)
- Register in ~/.claude/settings.json
- Test with manual orchestrator session

**Phase 2: Orchestrator skill restructuring (2-3 days)**
- Audit all 87 behavioral constraints against taxonomy
- Move 31 hard-behavioral to "Enforcement Reference" section (documented, not counted in prompt budget)
- Reformulate ~20 behavioral as knowledge
- Budget 8 critical behavioral constraints across 2 sections
- Reduce skill from 2,368 lines to <1,200 lines
- Re-run skillc test to validate compliance at new density

**Phase 3: Coaching hooks (1-2 days, 4 new hooks)**
- Create P2 and P3 coaching hooks
- Register in settings
- Monitor for false positive rate

**Phase 4: Validation and iteration (1 day)**
- Re-run constraint dilution test with restructured skill
- Measure behavioral compliance at new density
- Adjust prompt budget based on results

### Alternative: Constraint Policy Engine

Instead of individual hooks, build a single policy engine that reads constraint rules from a YAML config:

```yaml
# ~/.orch/policies/orchestrator.yaml
hard_gates:
  - name: no-filesystem-writes
    trigger: PreToolUse.Bash
    deny_patterns: ["mkdir", "rm ", "cp ", "mv ", "touch ", "chmod"]
    message: "Filesystem modification blocked. Delegate to worker."
  - name: no-git-remote
    trigger: PreToolUse.Bash
    deny_patterns: ["git push", "git merge"]
    message: "Remote git operations require user approval."

coaching:
  - name: investigation-drift
    trigger: PreToolUse.Read
    condition: code_file_count > 3
    message: "Multiple code reads detected. Consider investigation agent."
```

**Pros:** Single hook file, declarative rules, easier to maintain
**Cons:** More complex initial implementation, harder to debug, single-point-of-failure
**When to choose:** If hook count exceeds ~15, a policy engine reduces maintenance burden.
**Recommendation:** Start with individual hooks (Phase 1-3), migrate to policy engine if hook count grows.

---

## Decision Gate Guidance (if promoting to decision)

**Add blocks: frontmatter when promoting:**

This design establishes constraints future agents will encounter when:
- Building or modifying skill documents
- Creating new hooks for behavioral enforcement
- Restructuring the orchestrator skill
- Designing constraint systems for new agent roles

**Suggested blocks keywords:**
- constraint enforcement
- skill constraints
- behavioral constraints
- hook design
- orchestrator skill
- constraint dilution
- prompt budgeting

---

## Structured Uncertainty

**What's tested:**
- ✅ 151 constraints audited and classified (87 behavioral, 64 knowledge)
- ✅ 15 existing hooks inventoried and mapped to constraints
- ✅ Dilution evidence from orch-go-89at (36 test runs, 6 variants)
- ✅ Existing hook API capabilities verified (deny/allow/coaching/block)
- ✅ gate-orchestrator-code-access.py history confirming coaching > blocking for read constraints

**What's untested:**
- ⚠️ Section isolation: Whether separate prompt sections share attention budget (affects behavioral constraint capacity)
- ⚠️ Reformulation effectiveness: Whether knowledge-reformulated constraints actually achieve better compliance than behavioral formulation at high density
- ⚠️ Coaching hook effectiveness: Whether nudge messages change agent behavior (coaching is weaker than blocking)
- ⚠️ Hook performance: Whether 13 total PreToolUse hooks stay within 10s timeout budget
- ⚠️ False positive rate: New hooks may trigger on legitimate orchestrator actions

**What would change this:**
- If section isolation works → behavioral budget expands (3+ sections of ≤4 = 12+ behavioral in prompt)
- If reformulation doesn't improve compliance → more constraints need infrastructure enforcement
- If coaching hooks have no effect → convert to hard gates (with fabrication risk from code-access history)
- If hook count causes timeout issues → consolidate into policy engine
- If output pattern validation becomes reliable → judgment constraints could become coaching constraints

---

## References

**Files Examined:**
- `~/.claude/skills/meta/orchestrator/SKILL.md` — Orchestrator skill (2,368 lines, 151 constraints)
- `~/.orch/hooks/gate-orchestrator-code-access.py` — Code read coaching hook
- `~/.orch/hooks/gate-bd-close.py` — bd close enforcement hook
- `~/.orch/hooks/gate-orchestrator-task-tool.py` — Task tool blocking hook
- `~/.orch/hooks/enforce-phase-complete.py` — Phase complete enforcement hook
- `~/.orch/hooks/check-workspace-complete.py` — Workspace completion detection
- `~/.orch/hooks/pre-commit-knowledge-gate.py` — Knowledge capture gate
- `~/.orch/hooks/orchestrator-session-kn-gate.py` — Session-end knowledge gate
- `~/.claude/settings.json` (hooks section) — Hook registration inventory
- `pkg/spawn/claude.go` — --disallowedTools implementation

**Evidence:**
- `.kb/investigations/2026-03-01-inv-test-constraint-dilution-threshold.md` — 36 test runs proving dilution threshold
- `.kb/decisions/2026-02-26-two-layer-action-compliance.md` — Established infrastructure + prompt pattern
- `.kb/models/architectural-enforcement/model.md` — Multi-layer enforcement model

**Related Artifacts:**
- Probe: `.kb/models/architectural-enforcement/probes/2026-03-02-probe-layered-constraint-enforcement-design.md`

---

## Investigation History

**2026-03-02 (current):** Design started
- Audited all 151 constraints in orchestrator skill
- Inventoried 15 existing hooks
- Identified 5 decision forks, navigated all with substrate reasoning
- Produced 4-type constraint taxonomy and 4-phase implementation plan
- Identified behavioral → knowledge reformulation technique as key insight
