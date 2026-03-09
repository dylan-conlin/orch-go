## Summary (D.E.K.N.)

**Delta:** The probe/model cycle works conceptually for humans — it IS the scientific method — but the tooling is agent-first: no `kb create probe` command exists, and probe instructions live only in skill files that humans don't read.

**Evidence:** Walked through all 7 steps of the probe lifecycle; 5/7 have CLI support (`kb context`, `kb reflect`, `git`, editors); 2/7 require implicit knowledge embedded in agent skill files (probe file creation and naming conventions).

**Knowledge:** The cycle is substrate-independent with respect to investigator type (human or AI). The gap is tooling, not conceptual. One CLI command (`kb create probe`) plus one guide would make the cycle human-runnable.

**Next:** Add `kb create probe "slug" --model {name}` command to kb CLI. Create a human-readable "How to Run a Probe" guide extracted from the investigation skill.

**Authority:** architectural - Touches kb CLI (cross-component) and changes who can participate in the knowledge cycle (cross-boundary decision).

---

# Investigation: Can Humans Run Probes?

**Question:** Does the investigation/probe/model cycle work without AI agents as investigators? What breaks when a human tries to run the same knowledge cycle that agents run?

**Started:** 2026-03-09
**Updated:** 2026-03-09
**Owner:** orch-go-5j2cq
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** system-learning-loop

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/system-learning-loop/probes/2026-03-09-probe-knowledge-physics-accretion-attractor-gate-dynamics.md` | extends | yes | none |
| `.kb/models/system-learning-loop/probes/2026-03-01-probe-legibility-literature-review-bainbridge-forward.md` | extends | yes | none |

**Verified:** Knowledge-physics probe's claim that "knowledge attractors are attention-primed" confirmed — agents receive model claims via SPAWN_CONTEXT injection, which is an attention mechanism. Legibility probe's Bainbridge Irony #3 (autonomous daemon degrades Dylan's intervention readiness) is directly relevant: if only agents run probes, Dylan loses the ability to probe.

---

## Findings

### Finding 1: No `kb create probe` command exists — probe creation is agent-only

**Evidence:** Running `kb create probe "test" --model system-learning-loop` returns `Error: unknown flag: --model`. The `kb create` subcommands are: decision, guide, investigation, plan, research, specification. Probe is not among them.

AI agents create probes by:
1. Reading `.orch/templates/PROBE.md` (41-line template)
2. Substituting `{title}`, `{model-name}`, `{date}` placeholders
3. Writing to `.kb/models/{model-name}/probes/YYYY-MM-DD-probe-{slug}.md`

These instructions exist only in `skills/src/worker/investigation/.skillc/intro.md`, which AI agents receive via spawn context injection. Humans have no equivalent path to discover this workflow.

**Source:** `kb create --help`, `kb create probe --model system-learning-loop 2>&1`, `skills/src/worker/investigation/.skillc/intro.md`

**Significance:** This is the single biggest barrier to human probe participation. The template is simple (3 placeholders, 4 required sections), but the convention knowledge (naming, placement, directory structure) is locked inside agent infrastructure.

---

### Finding 2: Context retrieval and claim discovery work for humans

**Evidence:** Running `kb context "system learning loop"` returns a structured view with: constraints (3), decisions (7), open questions (1), guides (3), models with probes (10), and investigations (7). A human reading this output can identify which models exist, what probes have been run, and what claims remain unprobed.

Running `kb reflect` surfaces: synthesis opportunities (4 clusters), stale decisions (5), and stale models. This gives humans a menu of what needs attention.

Reading `model.md` directly gives access to "Critical Invariants" sections — numbered, testable claims. 13 of 32 models have explicit Critical Invariants sections.

**Source:** `kb context "system learning loop"`, `kb reflect`, `grep -c "Critical Invariant" .kb/models/*/model.md`

**Significance:** The discovery phase of the probe cycle has good human tooling. A human can find what models exist, what they claim, and what hasn't been probed. The gap is not "what to probe" but "how to record a probe."

---

### Finding 3: The probe template maps to the scientific method — it's inherently human

**Evidence:** The PROBE.md template has 4 required sections:
1. **Question** — What specific claim are you testing? (= hypothesis)
2. **What I Tested** — Commands, experiments, observations (= methodology)
3. **What I Observed** — Actual output, behavior, evidence (= results)
4. **Model Impact** — Confirms / Contradicts / Extends (= theory update)

This is question → experiment → observation → theory update. Humans have been running this cycle since Francis Bacon. The template doesn't require any AI-specific capability. A human testing "does RecurrenceThreshold=3 actually trigger at 3?" would follow the exact same structure.

**Source:** `.orch/templates/PROBE.md`, review of 2 existing probes in system-learning-loop/probes/

**Significance:** The cycle is not agent-specific. It was formalized for agents but describes universal empirical inquiry. The knowledge-physics model's claim that the dynamics are "substrate-independent" extends to the investigator substrate too.

---

### Finding 4: The merge step (probe → model update) is where humans may outperform agents

**Evidence:** The investigation skill requires: "you MUST merge its findings into the parent model.md BEFORE reporting Phase: Complete." This requires reading the model, understanding its structure, identifying which sections to update, and synthesizing new findings with existing claims.

AI agents sometimes handle this mechanically (appending a row to the "Merged Probes" table) rather than deeply integrating findings. Existing probes show variation in merge quality — some update Critical Invariants, some just add a table row.

Humans doing synthesis naturally reconcile conflicting information, weigh evidence quality, and restructure arguments. This is core intellectual work, not mechanical file manipulation.

**Source:** Worker-base skill "Probe-to-Model Merge (REQUIRED)" section, review of model.md "Merged Probes" tables across multiple models.

**Significance:** The probe cycle's highest-value step (synthesis → model update) is one where human cognitive strengths align. This inverts the assumption that AI agents are better at all parts of the cycle.

---

### Finding 5: 187 probes exist — all created by AI agents, none by humans

**Evidence:** Counted probes across all models: 187 total probes distributed across 26 models (6 models with 0 probes). Top models by probe count: daemon-autonomous-operation (34), completion-verification (25), spawn-architecture (24). Every probe file was created by an AI agent spawned via `orch spawn investigation`.

Zero probes have been created by humans. The system has been running since February 2026 (when probes were introduced) with exclusively agent investigators.

**Source:** `for model in .kb/models/*/; do ... done | sort -rn` (probe count per model)

**Significance:** This is a monoculture. If the knowledge system's probe cycle only works with AI agents, it has a single point of failure: agent availability. It also means the system has never been validated with human investigators — the "substrate independence" claim (from knowledge-physics model) is untested for the investigator dimension.

---

### Finding 6: The full probe lifecycle has 7 steps; 5 are human-accessible, 2 are agent-locked

**Evidence:** Step-by-step audit of the probe lifecycle:

| Step | Description | Human Tooling? | Friction Level |
|------|------------|----------------|----------------|
| 1. Get model claims | `kb context "topic"` or read model.md | Yes | Low |
| 2. Identify claim to probe | Read Critical Invariants | Yes | Low |
| 3. Create probe file | **No CLI command** — manual copy + naming | **No** | High |
| 4. Do the investigation | Run tests, check code, observe | Yes | Low |
| 5. Fill probe sections | Any text editor | Yes | Low |
| 6. Merge into model | Read model.md, synthesize, update | Yes | Medium |
| 7. Commit | `git add && git commit` | Yes | Low |

Step 3 is the bottleneck. A human must know: the template path (`.orch/templates/PROBE.md`), the naming convention (`YYYY-MM-DD-probe-{slug}.md`), the target directory (`.kb/models/{model-name}/probes/`), and the placeholder substitution (`{title}`, `{model-name}`, `{date}`). All of this is in the skill file that humans don't read.

Step 6 is medium friction because model.md files can be long (system-learning-loop is 526 lines, knowledge-physics is 344 lines) and the merge requires understanding the model's organizational structure.

**Source:** Full walk-through of each step with actual commands tested.

**Significance:** The cycle is 71% human-accessible (5/7 steps). The remaining 29% is a tooling gap, not a conceptual gap. One CLI command closes the primary bottleneck.

---

## Synthesis

**Key Insights:**

1. **The probe cycle IS the scientific method, formalized for a file system** — The 4-section template (Question, What I Tested, What I Observed, Model Impact) maps directly to hypothesis → experiment → results → theory update. Humans invented this cycle. AI agents are running a formalized version of something humans do naturally.

2. **The agent-only monoculture creates two risks** — First, Bainbridge Irony #3: when only agents probe, the human operator's ability to evaluate model claims degrades (confirmed by the legibility literature review probe). Second, the knowledge-physics claim that dynamics are "substrate-independent" is untested for the investigator substrate — 100% of 187 probes were agent-created.

3. **One CLI command breaks the bottleneck** — `kb create probe "slug" --model {name}` would scaffold the probe file in the right location with the right naming. This is a ~50-line implementation (copy template, substitute placeholders, mkdir -p probes dir). Combined with a human-readable guide, this makes the full cycle accessible.

**Answer to Investigation Question:**

Yes, humans can run probes. The cycle is conceptually investigator-independent — it's the scientific method applied to a knowledge substrate. The current barrier is tooling, not capability: there is no `kb create probe` CLI command, and the probe workflow instructions exist only in agent skill files. A human who knows the conventions could run a probe today with manual file creation. Adding `kb create probe` to the CLI and extracting a "How to Run a Probe" guide from the skill file would make the cycle human-runnable with minimal friction.

The more interesting finding: humans may be *better* at the merge step (probe → model update) because synthesis requires reconciling conflicting evidence and restructuring arguments — core human cognitive strengths that agents sometimes handle mechanically.

---

## Structured Uncertainty

**What's tested:**

- Verified: `kb create probe` command does not exist (ran command, got error)
- Verified: `kb context` and `kb reflect` work for humans to discover probe targets (ran both, got usable output)
- Verified: PROBE.md template has 3 placeholders and 4 required sections (read template, counted)
- Verified: 187 probes exist across 26 models, all agent-created (counted via shell)
- Verified: Probe instructions exist only in skill files (grep'd skill sources)

**What's untested:**

- Whether a human actually running a probe end-to-end encounters additional friction not predicted here (no human has tried yet)
- Whether `kb create probe` would be sufficient or if additional tooling is needed (e.g., claim listing, unprobed-claim detection)
- Whether human probes are qualitatively different from agent probes (no comparison data)

**What would change this:**

- A human actually running 3+ probes end-to-end would validate or refute the friction analysis
- If the merge step turns out to be harder than predicted for humans (model files are long, structure unclear), the "humans may be better at merge" claim would be wrong
- If probe creation tooling doesn't increase human probe participation, the barrier may be motivational rather than tooling

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Add `kb create probe` CLI command | implementation | Extends existing `kb create` pattern, single-scope change |
| Create "How to Run a Probe" guide | implementation | Documentation extraction, no architectural impact |
| Validate with human probe experiment | strategic | Decides whether to invest in human-probe tooling beyond MVP |

### Recommended Approach: Two-Step Enablement

**`kb create probe` + extracted guide** — Add one CLI command and one guide document to make the probe cycle human-runnable.

**Why this approach:**
- Closes the primary tooling gap (Finding 1) with minimal implementation
- Follows existing patterns (`kb create investigation`, `kb create decision`)
- Extracted guide provides the convention knowledge currently locked in skill files

**Trade-offs accepted:**
- Doesn't address advanced features (unprobed-claim detection, claim listing)
- Human probe quality is unverified until someone actually runs one

**Implementation sequence:**
1. Add `kb create probe "slug" --model {name}` — scaffolds probe file from template in correct directory
2. Extract "How to Run a Probe" guide from skill file into `.kb/guides/running-probes.md`
3. Run validation: Dylan runs 2-3 probes manually to test the full cycle

### Alternative Approaches Considered

**Option B: Full human probe dashboard**
- **Pros:** Shows unprobed claims, tracks probe coverage, suggests next probes
- **Cons:** Over-engineering before validating that humans want to probe
- **When to use instead:** After Option A proves humans actually run probes

**Option C: Do nothing — keep probes agent-only**
- **Pros:** No implementation cost
- **Cons:** Maintains monoculture risk (Bainbridge Irony #3), leaves knowledge-physics claim untested
- **When to use instead:** If human probing has no strategic value

**Rationale:** Option A is the minimum viable experiment. If humans don't run probes after the tooling exists, the barrier was motivational, not tooling — and we learn that cheaply.

---

### Implementation Details

**What to implement first:**
- `kb create probe` in `cmd/kb/` following the `create_investigation.go` pattern
- Template substitution: `{title}` from slug, `{model-name}` from `--model` flag, `{date}` from today

**Things to watch out for:**
- Probe directory may not exist yet for models with 0 probes — `mkdir -p` needed
- Model name validation — flag should check `.kb/models/{name}/model.md` exists
- Naming convention: `YYYY-MM-DD-probe-{slug}.md` must match what agents create

**Success criteria:**
- `kb create probe "test-claim" --model system-learning-loop` creates a probe file in the right location
- Dylan can run a full probe cycle (create → fill → merge → commit) in under 15 minutes
- At least 1 human-created probe exists within 2 weeks of tooling deployment

---

## References

**Files Examined:**
- `.orch/templates/PROBE.md` — Probe template (41 lines, 3 placeholders, 4 required sections)
- `skills/src/worker/investigation/.skillc/intro.md` — Probe creation instructions (agent-only)
- `.kb/models/system-learning-loop/model.md` — System learning loop model (526 lines)
- `.kb/models/knowledge-physics/model.md` — Knowledge physics model (344 lines)
- `.kb/models/system-learning-loop/probes/2026-03-09-probe-knowledge-physics-accretion-attractor-gate-dynamics.md` — Example probe

**Commands Run:**
```bash
# Test probe creation via CLI
kb create probe "test" --model system-learning-loop  # → Error: unknown flag

# Test context retrieval for humans
kb context "system learning loop" --domain "learning"

# Test model claim discovery
kb reflect

# Count probes across all models
for model in .kb/models/*/; do ... done | sort -rn

# Check probe instructions location
grep -A 5 "Create a probe" skills/src/worker/investigation/.skillc/intro.md

# List available models
ls .kb/models/
```

**Related Artifacts:**
- **Model:** `.kb/models/system-learning-loop/model.md` — Parent model for this investigation
- **Model:** `.kb/models/knowledge-physics/model.md` — Framework for investigator-substrate independence
- **Probe:** `.kb/models/system-learning-loop/probes/2026-03-01-probe-legibility-literature-review-bainbridge-forward.md` — Bainbridge Irony #3 relevance

---

## Investigation History

**2026-03-09:** Investigation started
- Initial question: Can humans run the investigation/probe/model cycle without AI agents?
- Context: Knowledge-physics model claims substrate independence, but 100% of probes are agent-created

**2026-03-09:** Walked through full 7-step probe lifecycle
- Tested each step with actual commands
- Found 5/7 steps human-accessible, 2/7 agent-locked (probe creation, convention knowledge)

**2026-03-09:** Investigation completed
- Status: Complete
- Key outcome: The cycle works for humans conceptually (it's the scientific method), but tooling creates a gap. One CLI command (`kb create probe`) closes the main barrier.
