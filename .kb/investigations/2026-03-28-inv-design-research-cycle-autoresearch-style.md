## Summary (D.E.K.N.)

**Delta:** The research cycle is not a tight autoresearch-style loop — it's a cross-session knowledge cycle that makes the claim→probe→merge pipeline visible and systematic. The minimum viable design is three components: `orch research` command (claim parsing + agent spawning), a research skill (structured probe protocol), and orient integration (claim status display).

**Evidence:** Codebase analysis confirms existing primitives: --loop (task iteration via eval→rework), --explore (parallel decomposition), probe artifacts (structured hypothesis testing), model claims tables (testable predictions with how-to-verify). The gap is NOT execution speed — the 373-paper bibliometrics probe took hours and worked fine manually. The gap is visibility (knowing what's untested), context setup (giving agents the right claim context), and result tracking (knowing which claims are confirmed).

**Knowledge:** The autoresearch lesson applies at the meta-level: constraint-first design. The tightest constraint for research is one claim, one method, one verdict, merge or archive. The "loop" operates across sessions (orient→research→probe→merge→orient), not within them. Automating the inner loop (experiment execution) is premature — agents already do this well. Automating the outer loop (what to test next, whether results merged) is the leverage point.

**Next:** Three implementation issues: (1) `orch research` command with claim parser, (2) research skill following probe protocol, (3) orient integration for claim status. Two blocking questions for orchestrator: claim table format standardization, daemon integration boundary.

**Authority:** architectural - Cross-component design (new command, new skill, orient integration, model format implications)

---

# Investigation: Design Research Cycle — Autoresearch-Style Loop for Hypothesis Testing Against Model Claims

**Question:** How should orch-go implement a systematic research cycle that takes model claims, generates testable experiments, spawns research agents, and tracks results — inspired by autoresearch's constraint-first loop pattern?

**Started:** 2026-03-28
**Updated:** 2026-03-28
**Owner:** architect agent (orch-go-47ppm)
**Phase:** Complete
**Next Step:** Create implementation issues for 3 components
**Status:** Complete
**Model:** named-incompleteness

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-03-22-inv-investigate-karpathy-autoresearch-48k-stars.md` | extends | yes | None — autoresearch's tight-loop pattern applies at knowledge level, not code level |
| `.kb/investigations/2026-03-25-inv-design-loop-spawn-flag-compose.md` | extends | yes | None — --loop is task iteration; research cycle is knowledge iteration. Orthogonal concerns |
| `.kb/models/named-incompleteness/model.md` | informs | yes | None — NI claims table IS the input for the research cycle |
| `.kb/models/compositional-accretion/model.md` | informs | yes | None — CA's design criterion (outward-pointing, opt-out, natural to creation) guides probe artifact design |

---

## Findings

### Finding 1: The gap is visibility and context, not execution speed

**Evidence:** The existing manual probe workflow works well for individual experiments. The 373-paper bibliometrics probe (2026-03-28) produced a significant quantitative result through a manual process: human read model → designed experiment → spawned agent → agent wrote probe → merged into model. The friction is not in any single step. It's in the spaces between:

1. **Visibility gap:** No command shows "which claims are untested." You must read each model.md, scan the claims table, count probes manually. Named-incompleteness has 6 claims — manageable. But compositional-accretion has 6, knowledge-accretion has 14+, defect-class-taxonomy has 7. Cross-model, there are ~33 testable claims with no systematic way to see their status.

2. **Context setup gap:** When spawning a probe agent, the orchestrator manually assembles: claim text, how-to-verify field, relevant model context, prior probe results. This is ~10 minutes of context gathering per spawn. The research cycle should do this automatically.

3. **Result tracking gap:** After a probe completes, the probe-to-model merge gate ensures the model gets updated. But there's no aggregate view: "NI-01 confirmed (3 probes), NI-02 untested, NI-06 untested." This cross-claim status lives nowhere.

**Source:** Manual inspection of `.kb/models/*/model.md` claims tables. Count of models with claims: named-incompleteness (6), compositional-accretion (6), knowledge-accretion (14+), defect-class-taxonomy (7), completion-verification (4+). Probe count by model: `ls .kb/models/*/probes/ | wc -l`.

**Significance:** The research cycle's value isn't in speeding up experiments — it's in making the claim-probe pipeline systematic. This is the same insight as autoresearch: the power is in problem formulation (which claims to test, with what method) not in execution machinery.

---

### Finding 2: The cycle operates across sessions, not within them

**Evidence:** autoresearch loops in 5-minute cycles because ML experiments take 5 minutes. Knowledge probes take 1-4 hours per experiment. The "loop" isn't a tight iteration — it's a cross-session knowledge cycle:

```
orient (see untested claims) → research (spawn probe agent) → probe completes → merge → orient (updated status)
```

Cycle time: hours to days, not seconds. This means:

- **No eval→rework loop needed.** The --loop flag iterates within a session until eval passes. Research probes don't retry — each probe is a single attempt with its own method. A "failed" probe (disconfirming result) is valuable data, not a reason to rework.
- **No blocking wait needed.** The orchestrator spawns a probe and moves on. When the probe completes, the merge gate fires. The next orient shows updated status.
- **The "loop" is the human/orchestrator returning.** Each orient session shows what's tested, what's not, and what to prioritize. The cycle is driven by attention, not automation.

**Source:** Comparison of autoresearch `program.md` loop (lines 94-106: edit→run→check→keep/discard, 5 min) vs. actual probe workflow (373-paper study: ~3 hours from spawn to completion). `pkg/orch/loop.go` loop controller structure (wait→eval→rework, blocks until completion).

**Significance:** Trying to build a tight autoresearch-style loop for knowledge research would be misguided. The cycle time is fundamentally different. The right abstraction is: make claims visible, make spawning easy, track results — and let the orchestrator drive the cycle through normal session flow.

---

### Finding 3: Existing primitives compose into the research workflow

**Evidence:** The orch-go codebase has all the building blocks:

| Primitive | What it does | Research cycle role |
|-----------|-------------|-------------------|
| `--explore` flag (`spawn_cmd.go:264-290`) | Parallel decomposition of one question into N subproblems | V2: test one claim with N methods in parallel |
| Model claims table (e.g., NI-01 through NI-06) | Structured testable predictions with "How to Verify" | Input: the claim + verification method IS the hypothesis |
| Probe artifacts (`.kb/models/*/probes/`) | Structured experiment results with verdict | Output: the research result |
| Probe-to-model merge gate (worker-base skill) | Ensures findings flow back to parent model | Result aggregation: already exists |
| `orch orient` (`cmd/orch/orient_cmd.go`) | Session-start context with thinking surface | Visibility: show claim status |
| `kb create investigation` CLI | Creates investigation file from template | Infrastructure: already handles file creation |

What's missing is the glue:
1. **Claim parser:** Read model.md, extract claims table, determine tested/untested status by cross-referencing probes
2. **Research context assembler:** Build SPAWN_CONTEXT with claim text, how-to-verify, prior probe results, model context
3. **Claim status aggregator:** Count probes per claim, extract verdicts, render status

**Source:** `cmd/orch/spawn_cmd.go:264-290` (explore flag), `pkg/orch/loop.go` (loop controller), `.kb/models/named-incompleteness/model.md:69-79` (claims table), `.kb/models/named-incompleteness/probes/` (4 probe files)

**Significance:** The research cycle doesn't need new subsystems. It needs a thin composition layer (~300-400 lines) that connects existing primitives with claim-aware context. This follows the --loop precedent: ~200 lines composing spawn+wait+rework.

---

### Finding 4: The claim table format is already the hypothesis specification

**Evidence:** Every model claims table already contains the hypothesis specification:

```markdown
| ID | Claim | How to Verify |
|----|-------|---------------|
| NI-01 | Named gaps compose... | Compare clustering effectiveness... Cross-subfield replication... |
| NI-06 | Named incompleteness has optimal specificity... | Measure clustering effectiveness as function of gap specificity... |
```

The "How to Verify" column IS the experiment design. The agent reads it and designs a specific method. No separate hypothesis bank is needed. This is the constraint-first insight: the model already carries its own research program.

However, the format varies across models:
- Named-incompleteness: Fully structured (ID, Claim, How to Verify, inline status notes)
- Compositional-accretion: Same structure (ID, Claim, How to Verify)
- Knowledge-accretion: Less structured (some claims have IDs, some don't)
- Defect-class-taxonomy: Different format (Class ID, but no explicit "How to Verify")

A claim parser would need to handle this variation or standardize the format.

**Source:** `.kb/models/named-incompleteness/model.md:69-79`, `.kb/models/compositional-accretion/model.md:186-194`, `.kb/models/knowledge-accretion/model.md` (various locations)

**Significance:** The claim table doubles as hypothesis bank, but format standardization is needed for automated parsing. This is a blocking question: standardize the format (breaking change to existing models) or build a fuzzy parser (complexity).

---

### Finding 5: Autoresearch's keep/discard maps to merge/archive, not confirm/disconfirm

**Evidence:** In autoresearch, keep/discard is based on val_bpb improvement. The equivalent for knowledge research is:

| autoresearch | research cycle | Meaning |
|---|---|---|
| val_bpb improved → keep commit | Probe has clear verdict → merge into model | Result is useful |
| val_bpb same/worse → discard commit | Probe is inconclusive → archive | Result is noise |

Critically, a **disconfirming** probe is a **keep**, not a discard. A probe that disconfirms NI-01 is extremely valuable — it updates the model by weakening or correcting the claim. The keep/discard decision is about evidence quality (clear verdict vs. inconclusive), not about confirmation bias (confirms vs. disconfirms).

This means the eval criterion is: "Does the probe have a clear verdict?" not "Does the probe confirm the claim?"

The existing probe format already handles this. Every probe has:
- Verdict: confirms / disconfirms / extends
- Model Impact: what changed in the model
- Structured uncertainty: what's tested vs. untested

The research cycle doesn't need to parse these — it just verifies they exist. The agent provides the judgment; the system provides the structure.

**Source:** autoresearch `program.md:94-106` (keep/discard logic), `.kb/models/named-incompleteness/probes/2026-03-28-probe-bibliometrics-full-study-373-papers.md` (example of clear verdict with quantitative evidence)

**Significance:** The eval criterion is simpler than it first appears. No statistical threshold parsing needed. Just: did the probe produce a clear verdict? This follows the --loop design: V1 uses the simplest possible eval (exit code = binary). V1 research uses the simplest possible eval (verdict field exists and is non-empty).

---

### Finding 6: Daemon auto-spawning research would produce compliance-driven probes (NI Failure Mode 4)

**Evidence:** The named-incompleteness model documents Failure Mode 4: "False Gaps — Named incompleteness that doesn't correspond to real possibility space." Instances include "compliance-driven probes: probes created to satisfy the probe-to-model merge gate but testing nothing genuinely uncertain."

If the daemon auto-spawns research agents for untested claims, two failure modes activate:

1. **Gap inflation (FM3):** All untested claims get queued, regardless of whether they're ripe for testing. Some claims (NI-05: "resolution is side effect, not goal") require careful philosophical analysis, not automated spawning. Automated scheduling treats all claims as equally actionable.

2. **False gaps (FM4):** Agents spawned by automation produce probes to satisfy the research cycle, not because they have genuine uncertainty. The probe format is correct but the content is vacuous. This is the knowledge equivalent of "agents filling template-mandated questions with generic text."

The compositional-accretion model's CA-06 confirms: "Only opt-out signals achieve >80% adoption; opt-in signals plateau at 15-25%." But adoption rate is the wrong metric for research quality. High-adoption research (every claim tested) could mean low-quality research (compliance-driven probes).

**Source:** `.kb/models/named-incompleteness/model.md:130-148` (Failure Modes 3 and 4), `.kb/models/compositional-accretion/model.md:96-107` (CA-06 adoption threshold)

**Significance:** Research scheduling requires judgment about which claims are ripe for testing, which methods would be informative, and how to interpret ambiguous results. This judgment is best exercised by the orchestrator reading orient output, not by the daemon applying a rule. The design should make research EASY to trigger, not AUTOMATIC.

---

## Synthesis

**Key Insights:**

1. **The research cycle is the orient→research→probe→merge→orient loop, operating across sessions.** It's not a tight autoresearch-style inner loop — it's a knowledge cycle where each session can advance one or more claims. The "automation" is in making the cycle visible and reducing context-setup friction, not in speeding up execution.

2. **Constraint-first design applies at the meta-level.** The tightest constraint for knowledge research: one claim, one method, one verdict, merge or archive. This maps to autoresearch's one file, one metric, keep/discard. The constraint IS the architecture — no loop controller, no statistical eval parser, no hypothesis bank needed.

3. **The claim table IS the research program.** Models already carry their own testable predictions with verification methods. The research cycle doesn't generate hypotheses — it makes existing ones actionable by providing visibility (which claims are untested), context (assembling claim + method for agents), and tracking (aggregating probe results).

4. **Disconfirmation is a keep, not a discard.** The eval criterion is evidence quality (clear verdict), not confirmation (agrees with claim). This prevents the system from optimizing for confirmation and preserves NI's core insight: the system should pressure its own models.

5. **Manual trigger with automated context is the right V1.** The orchestrator decides what to test (judgment); the system assembles the context (automation). This avoids compliance-driven probes while eliminating context-setup friction.

**Answer to Investigation Question:**

The research cycle should be implemented as three components that compose existing primitives:

1. **`orch research` command** — Parses model claims tables, shows claim status (tested/untested/N probes), spawns agents with pre-assembled research context. This is the trigger and context assembler.

2. **Research skill** — A structured probe protocol that constrains agents to: read model + claim → design one method → execute experiment → write probe artifact → merge into model. This is the experiment executor.

3. **Orient integration** — Adds "research surface" to orient output: per-model claim status showing what's been tested and what hasn't. This closes the loop by making untested claims visible at every session start.

The cycle operates across sessions: orient (visibility) → research (action) → probe (result) → merge (integration) → orient (updated visibility). The human/orchestrator drives the cycle by choosing which claims to test and when — this judgment cannot be automated without producing compliance-driven probes.

V2 composes `--explore` with the research skill for parallel multi-method probing of a single claim. V3 adds `--sweep` for batch-spawning probes for all untested claims. But V1 is manual trigger with automated context.

---

## Structured Uncertainty

**What's tested:**

- ✅ Existing primitives compose into research workflow (verified: --loop, --explore, probe artifacts, model claims tables all exist and are well-structured)
- ✅ Model claims tables contain hypothesis specifications (verified: 5 models examined, all have "How to Verify" or equivalent)
- ✅ Probe-to-model merge gate exists and works (verified: 4 NI probes all merged correctly)
- ✅ Orient already renders thinking-surface data and is extensible (verified: `cmd/orch/orient_cmd.go` collects 5 elements)
- ✅ --loop precedent confirms thin composition layers work (~200 lines, verified: `pkg/orch/loop.go`)

**What's untested:**

- ⚠️ Claim table format parsing across all models (only 5 models examined; format varies)
- ⚠️ Whether automated context assembly produces better probes than manual context gathering
- ⚠️ Whether orient-integrated claim status actually changes research behavior (would need 30-day measurement)
- ⚠️ Research skill effectiveness (no skill exists yet; assumes probe protocol can be structured into a skill)
- ⚠️ Whether cross-session cycle actually completes (risk: orient shows untested claims but orchestrator never triggers research)

**What would change this:**

- If claim table formats are too varied to parse reliably, standardization becomes a prerequisite (adds scope)
- If orchestrators ignore orient research surface consistently (>80% of untested claims remain untested after 30 days), daemon integration becomes necessary despite FM4 risk
- If research skill agents consistently produce low-quality probes, the skill needs tighter constraints or the research skill concept is wrong
- If a single tight-loop mode proves valuable (e.g., iterating on a bibliometrics study: run N=50 → adjust method → run N=300), the --loop flag could be composed with research

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Build `orch research` command | architectural | New command, cross-component (claims parsing, spawn pipeline, orient) |
| Create research skill | architectural | New skill following probe protocol, cross-component (skill system, probe format, model format) |
| Standardize claims table format | architectural | Breaking change to existing models, cross-component (all models, research command parser) |
| Wire claim status into orient | implementation | Extends existing orient data collection, single-scope |
| Daemon research scheduling (V3) | strategic | Changes what the system does autonomously, risk of compliance-driven probes |

### Recommended Approach: Three-component composition with manual trigger

**`orch research` command + research skill + orient claim status**

**Why this approach:**
- Follows constraint-first design (autoresearch lesson): the claim table IS the research program
- Composes existing primitives (spawn, probe format, model claims, orient) without new subsystems
- Preserves human judgment on what to test (avoids compliance-driven probes)
- Makes untested claims visible (orient) without making them automatic (daemon)
- Each component is independently valuable and testable

**Trade-offs accepted:**
- No automation of research scheduling (V1 is manual trigger only)
- Requires claims table format standardization or fuzzy parser (scope question)
- Cycle depends on orchestrator reading orient output and deciding to act
- No statistical eval of probe quality (binary: verdict exists or not)

**Implementation sequence:**

1. **Claims parser + `orch research` status mode** (~150 lines)
   - Parse model claims tables
   - Cross-reference with probe files to determine tested/untested
   - `orch research <model>` → show claim status
   - Why first: Provides visibility without any spawning. Immediately useful.

2. **Research skill** (~80 lines of SKILL.md)
   - Structured probe protocol: read claim → design method → execute → write probe → merge
   - Constrains agent to one claim, one method, one verdict
   - Why second: The skill defines what spawned agents do. Must exist before spawning.

3. **`orch research` spawn mode** (~200 lines)
   - `orch research <model> <claim-id>` → spawn agent with research skill + claim context
   - Auto-assembles SPAWN_CONTEXT with: claim text, how-to-verify, prior probe summaries, model context
   - Why third: Composes parser (step 1) with skill (step 2) via existing spawn infrastructure.

4. **Orient integration** (~50 lines)
   - Add "research surface" to orient: per-model claim status
   - `orch orient` → "named-incompleteness: 2/6 confirmed, 4 untested"
   - Why fourth: Closes the cycle. Now untested claims are visible at every session start.

### Alternative Approaches Considered

**Option B: Extend --loop for research iteration**
- **Pros:** Reuses existing loop controller, no new command needed
- **Cons:** --loop is task iteration (fix until passing); research doesn't iterate the same way. A disconfirming probe isn't a "failure" to retry. Conflates two orthogonal concerns.
- **When to use instead:** If a specific research pattern emerges that benefits from within-session iteration (e.g., "run pilot N=50, check power, increase to N=300"). This would be V2: `orch research --loop --loop-eval "check_power.sh"`.

**Option C: Daemon-scheduled research**
- **Pros:** Fully automated; no human attention needed to advance claims
- **Cons:** Produces compliance-driven probes (NI Failure Mode 4). The daemon can't judge which claims are ripe for testing or which methods would be informative. Research quality requires judgment.
- **When to use instead:** Only after V1 proves that the research cycle works with manual triggers and we have quality metrics for probes. If probe quality is consistently high (>80% producing clear verdicts with genuine evidence), daemon scheduling becomes safer.

**Option D: Research as --explore variant (parallel multi-method)**
- **Pros:** Tests one claim with N different methods simultaneously
- **Cons:** Premature for V1. Most claims have been tested with one method so far. Parallel multi-method is valuable for replication, but replication comes after initial confirmation.
- **When to use instead:** V2, once claims have at least one probe. `orch research --replicate NI-01` spawns N agents with different methods to cross-validate.

**Rationale for recommendation:** V1 solves the actual bottleneck (visibility + context setup) with minimal machinery. More automation can be earned once we measure whether the manual cycle produces good research.

---

### Implementation Details

**What to implement first:**
- Claims table parser that handles current format variations
- `orch research named-incompleteness` → displays claim status immediately
- This provides value before any spawning infrastructure exists

**Things to watch out for:**
- ⚠️ **Defect Class 0 (Scope Expansion):** Claims parser initially targets named-incompleteness format. Other models vary. Build allowlist of supported formats, fail gracefully on others.
- ⚠️ **Defect Class 3 (Stale Artifact Accumulation):** Probes that don't merge create phantom "tested" status. Cross-reference: only count probes that updated the model's Evidence/Probes section.
- ⚠️ **Defect Class 5 (Contradictory Authority Signals):** Claim status could diverge between what the model says (inline notes like "CONFIRMED") and what probe files show. Derive status from probes, don't store it separately in the model.
- ⚠️ **Claims table format variation:** Knowledge-accretion model has 14+ claims without consistent IDs. Standardization needed or fuzzy matching.

**Areas needing further investigation:**
- Claims table format standardization across all models (how many models, how much variation, migration cost)
- Research skill design details (probe protocol constraints, what guidance to include, how to prevent compliance-driven probes)
- Interaction between research cycle and comprehension queue (do probe briefs enter comprehension?)

**Success criteria:**
- ✅ `orch research <model>` shows accurate claim status (tested/untested/N probes) for at least 3 models
- ✅ `orch research <model> <claim-id>` spawns agent that produces valid probe artifact
- ✅ Orient shows research surface with claim status aggregated across models
- ✅ At least one research-spawned probe produces a clear verdict that merges into a model
- ✅ End-to-end cycle completes: orient (see untested) → research (spawn) → probe → merge → orient (updated)

---

### Composition Claims

| ID | Claim | Components Involved | How to Verify |
|----|-------|--------------------|----|
| CC-1 | Research spawns produce probe artifacts that merge into models | command + skill | Spawn research for a claim, verify probe exists in `.kb/models/*/probes/` and model.md updated |
| CC-2 | Orient displays claim status derived from actual probe files | command parser + orient | Run orient, verify counts match `ls .kb/models/*/probes/` per claim |
| CC-3 | Prior probe context prevents redundant research | command context assembler + skill | Spawn research for already-confirmed claim, verify agent acknowledges prior probes and extends rather than duplicates |

---

## References

**Files Examined:**
- `cmd/orch/spawn_cmd.go:264-290` — --explore flag precedent for spawn behavior transformation
- `cmd/orch/spawn_cmd.go:560-595` — --loop flag integration with spawn pipeline
- `pkg/orch/loop.go` — Loop controller as composition layer precedent (~250 lines)
- `cmd/orch/orient_cmd.go` — Orient data collection and rendering
- `pkg/orient/compose.go` — Compose summary formatting for orient
- `.kb/models/named-incompleteness/model.md:69-79` — Claims table format (reference model)
- `.kb/models/compositional-accretion/model.md:186-194` — Claims table format (second model)
- `.kb/models/named-incompleteness/probes/` — 4 probe files showing probe artifact structure
- `.kb/investigations/2026-03-22-inv-investigate-karpathy-autoresearch-48k-stars.md` — Autoresearch architecture analysis
- `.kb/investigations/2026-03-25-inv-design-loop-spawn-flag-compose.md` — --loop design precedent

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-03-22-inv-investigate-karpathy-autoresearch-48k-stars.md` — Autoresearch pattern: constraint-first, single metric, git as state
- **Investigation:** `.kb/investigations/2026-03-25-inv-design-loop-spawn-flag-compose.md` — --loop as thin composition layer precedent
- **Model:** `.kb/models/named-incompleteness/model.md` — Target model; claims table is the research cycle's input
- **Model:** `.kb/models/compositional-accretion/model.md` — Design criterion for probe artifact format (outward-pointing, opt-out signals)

---

## Investigation History

**2026-03-28:** Investigation started
- Initial question: How should orch-go implement an autoresearch-style loop for hypothesis testing against model claims?
- Context: Spawned by orchestrator to design the research cycle as an architect task

**2026-03-28:** Context gathering — read autoresearch investigation, --loop design, 5 model claims tables, probe artifacts, orient command, explore flag, loop controller
- Key insight: The gap is visibility and context setup, not execution speed. The cycle operates across sessions.

**2026-03-28:** 6 design forks identified and navigated
- Fork 0 (placement): `orch research` command — new command because research is distinct from spawn/explore/loop
- Fork 1 (trigger): Manual trigger — avoids compliance-driven probes
- Fork 2 (hypothesis): Claim table IS the hypothesis — no separate bank needed
- Fork 3 (execution): Research skill — structured probe protocol
- Fork 4 (eval): Verdict existence — simplest possible binary check
- Fork 5 (aggregation): Existing merge gate + orient integration
- Fork 6 (iteration): Single-shot V1 — earn the abstraction

**2026-03-28:** Investigation completed
- Status: Complete
- Key outcome: Three-component design (command + skill + orient) that composes existing primitives. The "loop" is cross-session, driven by orient visibility, not automation.
