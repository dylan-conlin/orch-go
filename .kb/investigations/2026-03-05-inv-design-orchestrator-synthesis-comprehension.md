## Summary (D.E.K.N.)

**Delta:** Orchestrator synthesis fails because it has no procedure, no infrastructure, and a template that elicits event logs. Triage works because it has all three (daemon, bd ready, label taxonomy). Teaching comprehension requires specific cognitive moves, a restructured debrief template, and advisory measurement.

**Evidence:** Read orchestrator skill (SYNTHESIZE is 1 line of description vs TRIAGE's 60+ lines of routing/infrastructure), debrief template (5 sections, all factual), 3 actual debriefs (pure event logs — "Spawned X, Completed Y" with zero interpretation), behavioral grammars model (Claim 3: situational pull overwhelms static reinforcement), ATC audit Finding 3.1 (post-flight debrief is biggest gap).

**Knowledge:** Comprehension requires three cognitive moves: thread identification (what question motivated the session), connection (what emerged from seeing agent findings together), and reframing (what Dylan now understands differently). These parallel Three-Layer Reconnection at session scope. The debrief template must demand interpretation, not permit summaries. The gate should be advisory and detect summary-absence-of-comprehension rather than attempting to judge comprehension quality.

**Next:** Implement in 4 phases: (1) Teach comprehension in skill — replace SYNTHESIZE definition with cognitive moves, (2) Restructure debrief template, (3) Enhance orch debrief with comprehension prompt and advisory quality check, (4) Close the loop via orch orient consuming prior session comprehension.

**Authority:** architectural — Changes span skill design, template structure, CLI infrastructure, and cross-session feedback loop. Multiple components affected with coordination needs.

---

# Investigation: Design Orchestrator Synthesis Comprehension

**Question:** How should orchestrators produce comprehension (rich understanding that connects work to WHY it matters) instead of summaries (lists of what happened), and what gates enforce the difference?

**Started:** 2026-03-05
**Updated:** 2026-03-05
**Owner:** architect (orch-go-4wr2w)
**Phase:** Complete
**Next Step:** None — proceed to implementation issues
**Status:** Complete

**Patches-Decision:** N/A (new design, may produce decision)
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/investigations/2026-02-28-inv-atc-lens-feature-audit.md` | extends | yes — Finding 3.1 confirmed: post-flight debrief is biggest ATC gap | none |
| `.kb/models/orchestrator-session-lifecycle/probes/2026-02-28-probe-session-debrief-artifact-design.md` | extends | yes — confirmed interactive sessions have no durable comprehension artifact | none |
| `~/.kb/models/behavioral-grammars/model.md` | grounds | yes — Claim 3 (situational pull overwhelms static reinforcement) explains why synthesis fails | none |
| `.kb/investigations/2026-02-28-investigation-orchestrator-intent-spiral.md` | confirms | yes — intent degrades across boundaries; synthesis is where meaning gets reconstituted | none |
| `.kb/models/orchestrator-session-lifecycle/probes/2026-02-27-probe-communication-breakdown-postmortem-3-sessions.md` | extends | yes — confirms orchestrators comply with identity but fail at action under pressure | none |

---

## Findings

### Finding 1: SYNTHESIZE has no procedure — TRIAGE has everything

**Evidence:** In the orchestrator skill template:
- TRIAGE: ~80 lines of infrastructure — Fast Path surface table (19 routing rules), Skill Selection decision tree, Label Taxonomy, Beads Tracking rules, Stall Triage classification. Plus external infrastructure: daemon auto-spawning, `bd ready`, `orch review`, `orch review triage`.
- SYNTHESIZE: 1 line of definition ("Combine findings from completed agents into coherent understanding") + 1 principle ("Synthesis is orchestrator work, not spawnable. Workers produce knowledge atoms; you compose them into models.") + no procedure, no examples, no cognitive moves, no infrastructure.

The Completion Lifecycle section has Three-Layer Reconnection and Follow-up Extraction, but these are per-agent completion steps, not session-level synthesis.

**Source:** `skills/src/meta/orchestrator/.skillc/SKILL.md.template` lines 9-13 (Role), 188-229 (Completion Lifecycle)

**Significance:** Behavioral grammars Claim 3: situational pull overwhelms static reinforcement. TRIAGE has gravitational infrastructure that pulls the orchestrator toward it. SYNTHESIZE has a norm and zero pull. The orchestrator will triage over synthesize every time — not because it's ignoring synthesis, but because triage has affordances and synthesis doesn't.

---

### Finding 2: The debrief template elicits event logs, not comprehension

**Evidence:** The template has 5 sections:
1. "What Happened" — "Summarize threads worked — completions, spawns, decisions made."
2. "What Changed" — "Durable changes — constraints, decisions, model updates."
3. "What's In Flight" — "Active agents, pending triage:review items, open questions."
4. "What's Next" — "1-3 proposed threads for next session."
5. "Session Health" — checkpoint discipline, frame collapse, discovered work.

Every section asks for facts. Zero sections ask for interpretation. The template's language actively models summary behavior: "One sentence per thread", "Summarize threads worked."

The actual 2026-03-04 debrief confirms this: 80 lines of "Spawned: X — task description" and "Fixed: Y — details". The "What Changed" section is raw completion reasons from events.jsonl. Zero sentences connecting work to meaning.

**Source:** `.kb/sessions/TEMPLATE.md`, `.kb/sessions/2026-03-04-debrief.md`, `.kb/sessions/2026-03-01-debrief.md`

**Significance:** The template is the strongest behavioral signal for debrief content. It directly shapes what the orchestrator produces. A template that asks for facts will get facts. A template that asks for interpretation will get interpretation — or at least reveal when comprehension is absent.

---

### Finding 3: `orch debrief` is an event formatter, not a synthesis tool

**Evidence:** `cmd/orch/debrief_cmd.go` and `pkg/debrief/debrief.go` show the pipeline:
1. Reads `events.jsonl` → formats as "Spawned: X / Completed: Y / Abandoned: Z"
2. Reads `bd list --status=in_progress` → formats as issue list
3. Reads `bd ready` → formats as next-work list
4. Merges `--changed` flag → user override for "What Changed"
5. Writes markdown from template

There is no LLM involvement. No comprehension prompt. No quality signal. The tool auto-populates factual sections and writes them to disk. The orchestrator's role is to optionally pass `--changed` with manual text.

**Source:** `cmd/orch/debrief_cmd.go:69-137`, `pkg/debrief/debrief.go:48-77`

**Significance:** This is the infrastructure gap. `orch debrief` could be the synthesis trigger — the moment where the orchestrator is forced to produce comprehension. Currently it's a passive formatter. The debrief command is the natural injection point because it already runs at session end (Session End Protocol step 1).

---

### Finding 4: Three-Layer Reconnection has a session-scope analog

**Evidence:** Three-Layer Reconnection (per-agent completion):
1. **Frame** — Start from Dylan's words
2. **Resolution** — What changed in terms of the frame
3. **Placement** — Connect to larger thread

This maps naturally to session scope:
1. **Thread** — What question/tension motivated this session? (Frame at session scope)
2. **Insight** — What emerged from seeing agent findings together? (Resolution at session scope)
3. **Position** — What does Dylan now understand differently, and where does the thread lead? (Placement at session scope)

The per-agent version works because it's procedural and triggered at completion. The session version fails because it's a norm in a template with no trigger.

**Source:** `SKILL.md.template` lines 188-196 (Three-Layer Reconnection), task description (examples of bad vs good synthesis)

**Significance:** The cognitive moves for session synthesis are already implicit in the system's design — they just need to be made explicit and procedural. This isn't new theory; it's extending an existing working pattern to the scope where it's missing.

---

### Finding 5: Comprehension absence is detectable; comprehension quality is not

**Evidence:** Analyzing the task description's examples:

Bad (summary): "Spawned two architects. First mapped 22 decidability boundaries. Second designed plan persistence. 10 issues created."
- All sentences start with action verbs (Spawned, Mapped, Designed, Created)
- No causal connectives (because, enables, means, implies, reveals)
- No reference to Dylan's cognitive state or understanding
- Structure: entity + action + count

Good (comprehension): "Started from metaphors and discovered that decidability graphs and behavioral grammars are two halves of the same enforcement problem..."
- Sentences connect concepts (decidability + behavioral grammars = same problem)
- Causal language (because, which means, where)
- References Dylan's cognitive state ("where Dylan carries cognitive load")
- Structure: thread + connection + implication

A regex can't judge comprehension quality, but it can detect summary patterns (sentences starting with "Spawned/Completed/Fixed/Added", lack of connective language, pure event listing). This is enough for an advisory gate — "this looks like a summary, not comprehension."

**Source:** Task description examples, behavioral grammars model (Claim 5: grammars can't detect their own failures — external measurement needed)

**Significance:** The gate design should follow the behavioral grammars prescription: measure before blocking. An advisory quality signal that detects summary-shaped output is achievable and useful. A blocking gate that requires "good comprehension" is not achievable and would be gamed.

---

## Synthesis

**Key Insights:**

1. **The asymmetry is infrastructure, not instruction.** TRIAGE works because it has a daemon, CLI commands, routing tables, and label taxonomies that pull the orchestrator toward it. SYNTHESIZE has a sentence. Adding more sentences about synthesis won't fix this — adding infrastructure will. This follows directly from behavioral grammars Claim 3 and the ATC audit's Finding 3.1.

2. **The cognitive moves for comprehension are: Thread → Insight → Position.** This is Three-Layer Reconnection at session scope. Thread = what question motivated the session. Insight = what emerged from seeing agent findings together that wasn't visible in isolation. Position = what Dylan now understands differently and where the thread leads. These are teachable, procedural, and verifiable-by-absence.

3. **The debrief template is the highest-leverage change.** The template shapes behavior more than the skill description (it's the most proximate behavioral signal). Restructuring the template to demand interpretation over factual listing will have more effect than any skill text change. The skill text teaches the cognitive moves; the template enforces them through structure.

4. **Advisory measurement before blocking gates.** Detecting summary absence (pure event listing, no connectives, no thread language) is achievable. Judging comprehension quality is not. The first gate should be advisory: "This debrief looks like a summary. The 'What We Learned' section should connect findings to meaning." Behavioral grammars says measure the failure mode before designing the enforcement.

**Answer to Investigation Question:**

Orchestrators should synthesize using three cognitive moves (Thread → Insight → Position) that parallel Three-Layer Reconnection at session scope. The skill needs to teach these moves explicitly, replacing the current 1-line SYNTHESIZE definition. The debrief template needs restructuring to elicit interpretation rather than permit summaries — specifically replacing "What Changed" (factual) with "What We Learned" (interpretive). The gate should be advisory, detecting summary-shaped output rather than judging comprehension quality. Infrastructure support comes through enhancing `orch debrief` to prompt the orchestrator for comprehension and providing quality feedback, and having `orch orient` consume the prior session's comprehension section to close the feedback loop.

---

## Structured Uncertainty

**What's tested:**

- ✅ Current skill SYNTHESIZE definition is hollow compared to TRIAGE (verified: counted lines, infrastructure, and affordances for each)
- ✅ Debrief template elicits summaries (verified: read template, then read 3 actual debriefs — all event logs)
- ✅ `orch debrief` is an event formatter with no LLM involvement (verified: read full implementation in debrief_cmd.go and debrief.go)
- ✅ Three-Layer Reconnection pattern maps to session scope (verified: structural mapping of Frame/Resolution/Placement to Thread/Insight/Position)
- ✅ Summary patterns are regex-detectable (verified: analyzed bad vs good examples for structural markers)

**What's untested:**

- ⚠️ Whether teaching Thread→Insight→Position in the skill actually changes orchestrator synthesis behavior (requires behavioral testing with skillc test)
- ⚠️ Whether the restructured template produces comprehension or just longer summaries (requires observation over 3+ sessions)
- ⚠️ Whether the advisory gate's heuristic (detecting summary absence) has acceptable false positive/negative rates (requires calibration)
- ⚠️ Whether `orch orient` consuming prior comprehension changes session-start quality (requires longitudinal observation)

**What would change this:**

- If behavioral testing shows the skill text change has zero effect on synthesis quality, the approach should shift entirely to infrastructure (template + tool) with no skill text
- If the restructured template produces "creative summaries" (longer event logs with filler connectives), the template approach is insufficient and a conversational gate (orchestrator self-reviews its debrief) may be needed
- If the advisory gate produces >30% false positives (flagging legitimate concise comprehension as summary), the heuristic needs refinement

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Skill text changes (teach synthesis cognitive moves) | architectural | Modifies the orchestrator policy document that shapes all orchestrator behavior |
| Debrief template restructure | architectural | Changes the structure of a cross-session knowledge artifact |
| `orch debrief` enhancement (comprehension prompt + quality signal) | implementation | Extends existing CLI command within established patterns |
| `orch orient` consuming prior comprehension | implementation | Adds data source to existing command within established patterns |
| Advisory gate on debrief quality | architectural | New enforcement mechanism affecting session-end protocol |

### Recommended Approach ⭐

**Layered comprehension infrastructure** — Teach the cognitive moves in the skill, restructure the template to demand them, enhance `orch debrief` to prompt for and measure them, close the feedback loop via `orch orient`.

**Why this approach:**
- Follows behavioral grammars prescription: infrastructure > instruction. All four layers provide pull toward synthesis.
- Follows the existing pattern: just as triage evolved from skill instruction → daemon + bd ready + label taxonomy, synthesis needs its own infrastructure stack.
- Minimal skill text change (replaces, doesn't add) respects constraint density.
- Advisory-first gating respects the "measure before blocking" principle.

**Trade-offs accepted:**
- Advisory gate can be ignored. Accepted because behavioral grammars says measuring the failure mode comes before blocking it — you need data on how often orchestrators produce summaries before designing a blocking mechanism.
- Template restructure requires coordination with `orch debrief` code. Accepted because both are in orch-go and can ship together.

**Implementation sequence:**

1. **Phase 1: Teach (skill + template)** — Skill edits to SYNTHESIZE definition and Completion Lifecycle. Template restructure. These are the highest-leverage changes.
2. **Phase 2: Prompt (orch debrief)** — Enhance debrief command to output comprehension prompts and a quality advisory.
3. **Phase 3: Close the loop (orch orient)** — Have orient read prior session's comprehension section.
4. **Phase 4: Measure (advisory gate)** — Add `orch debrief --quality` that runs heuristic summary detection.

### Alternative Approaches Considered

**Option B: Pure infrastructure (no skill text change)**
- **Pros:** Behavioral grammars says infrastructure > instruction. Skip the skill text entirely.
- **Cons:** The cognitive moves (Thread→Insight→Position) need to be taught somewhere. The template can demand "What We Learned" but can't teach HOW to produce it. The skill is the teaching layer; the template and tools are the enforcement layer.
- **When to use instead:** If skillc testing shows zero behavioral effect from skill text changes across 3+ variants.

**Option C: Conversational gate (orchestrator self-reviews debrief)**
- **Pros:** LLM can judge comprehension quality better than regex. The orchestrator re-reads its own debrief and asks "is this comprehension or summary?"
- **Cons:** Behavioral grammars Claim 5: agents can't detect their own grammar failures. An orchestrator that summarized instead of synthesizing is unlikely to recognize the failure. The convergence attractor (Claim 7) means the human's verification also degrades.
- **When to use instead:** If the advisory heuristic has unacceptable false positive rates (>30%) after calibration.

**Rationale for recommendation:** Option A combines teaching (skill) with structural enforcement (template) with tooling (orch debrief/orient) with measurement (advisory gate). This is the multi-layer approach that behavioral grammars prescribes for counter-instinctual constraints. Synthesis is counter-instinctual for LLMs — the prior is toward reporting what happened (summary) not interpreting what it means (comprehension).

---

### Implementation Details

**Specific skill edits (SKILL.md.template):**

Replace the SYNTHESIZE bullet in the Role section:

```markdown
## Role

**Three jobs:** COMPREHEND, TRIAGE, SYNTHESIZE. Implementation and investigation are worker roles.

- **COMPREHEND** — Build mental models by reading SYNTHESIS.md, kb context, prior investigations
- **TRIAGE** — Review issues, ensure correct typing, release to daemon via `triage:ready`
- **SYNTHESIZE** — Connect findings into understanding using Thread → Insight → Position:
  1. **Thread** — What question or tension motivated the work?
  2. **Insight** — What emerged from seeing findings together that wasn't visible in isolation?
  3. **Position** — What does Dylan now understand differently, and where does the thread lead?
```

Replace the "Synthesis is orchestrator work" line with:

```markdown
**Synthesis is comprehension, not reporting.** "We spawned two architects and got 10 issues" is a summary. "We discovered that X and Y are two halves of the same problem, and the gap between them is where cognitive load lives" is comprehension. Workers produce atoms; you compose meaning.
```

Add to Completion Lifecycle section, after Three-Layer Reconnection:

```markdown
### Session-Level Synthesis (Every Debrief)

Three-Layer Reconnection is per-agent. Session synthesis applies the same pattern to the whole session:

1. **Thread** — "This session was about [the question/tension that motivated work]"
2. **Insight** — "What emerged: [connection or discovery visible only from seeing multiple results together]"
3. **Position** — "Dylan now [what he sees/knows differently], which means [where the thread leads]"

The debrief's "What We Learned" section is where this lives. If it reads like an event log, it's a summary. If it reads like a paragraph you'd tell someone to explain why the session mattered, it's comprehension.
```

**Specific debrief template changes (.kb/sessions/TEMPLATE.md):**

```markdown
# Session Debrief: YYYY-MM-DD

**Date:** YYYY-MM-DD
**Duration:** ~Xh
**Focus:** [primary goal/thread for this session]

---

## What We Learned

<!-- This is the synthesis section. Connect the dots between what agents found.
     Not "we spawned X and completed Y" — that's an event log.
     Instead: "We discovered that A and B are related because C, which means D for Dylan."
     Use Thread → Insight → Position:
     Thread: What question motivated the session?
     Insight: What emerged from seeing results together?
     Position: What does Dylan understand now that he didn't before? -->

## What Happened

<!-- Brief factual record. One line per significant event. This is the log. -->

-

## What's In Flight

<!-- Active agents, pending triage:review items, open questions.
     What will keep running? What needs attention next session? -->

-

## What's Next

<!-- Where the thread leads — not a to-do list but strategic direction.
     "The thread leads toward X because today's work revealed Y." -->

1.

## Session Health

<!-- Scale to session weight. A 20-minute session gets a line. A 4-hour session gets the full picture. -->

- **Checkpoint discipline:** [ok / warning / exceeded]
- **Frame collapse:** [none / detected — describe]
- **Discovered work:** [issues created or "none"]
```

Key changes:
1. "What Changed" → **"What We Learned"** — moved to top, demands interpretation
2. "What Happened" demoted below synthesis — it's the log, not the point
3. "What's Next" reframed from "proposed threads" to "where the thread leads" — strategic, not tactical
4. HTML comments teach Thread→Insight→Position in the template itself

**orch debrief enhancement (Phase 2):**

After auto-populating facts, the command should:
1. Print a comprehension prompt: "What did this session's work mean? Use Thread → Insight → Position."
2. Accept `--learned "text"` flag for the "What We Learned" section.
3. With `--quality` flag, run heuristic on the "What We Learned" section and output advisory.

Quality heuristic (summary detection):
- Flag if section is empty or < 2 sentences
- Flag if all sentences start with action verbs (Spawned, Completed, Fixed, Added, Created, Updated)
- Flag if no connective language present (because, enables, means, implies, reveals, connects, shifts, discovered that, now understands)
- Output: "Advisory: This looks like a summary. Comprehension connects findings to meaning."

**orch orient enhancement (Phase 3):**

Add to orient output: "Last session's insight: [What We Learned section from most recent debrief]" — surfaces the comprehension thread across session boundaries.

**Things to watch out for:**

- ⚠️ Skill text change must replace, not add. Current orchestrator skill is at constraint density limits per behavioral grammars model. The SYNTHESIZE definition grows by ~3 lines but replaces the existing 1-line definition and the "Synthesis is orchestrator work" principle — net +2 lines. The session-level synthesis subsection adds ~7 lines to Completion Lifecycle. Total: +9 lines. Monitor skillc test scores.
- ⚠️ Debrief template change requires corresponding update to `pkg/debrief/debrief.go` `RenderDebrief()` function — the code generates the markdown, not just the template.
- ⚠️ The advisory gate must not block session end. Orchestrator sessions run under checkpoint pressure; a blocking synthesis gate creates competing pressures (checkpoint says "end soon" vs gate says "this isn't good enough").
- ⚠️ `orch debrief --learned` is a CLI flag designed for orchestrator programmatic use, not Dylan typing — matches the User Interaction Model constraint.

**Success criteria:**

- ✅ Debrief "What We Learned" sections contain connective language (because, means, reveals) at least 50% of the time across 5+ sessions
- ✅ Advisory gate correctly flags pure event logs as summaries and does not flag comprehension paragraphs
- ✅ `orch orient` surfaces prior session's insight
- ✅ skillc test scores for orchestrator skill do not degrade after edits (behavioral testing)

---

## References

**Files Examined:**
- `skills/src/meta/orchestrator/.skillc/SKILL.md.template` — Orchestrator skill source, SYNTHESIZE definition, Completion Lifecycle
- `.kb/sessions/TEMPLATE.md` — Debrief template structure
- `.kb/sessions/2026-03-04-debrief.md` — Actual debrief showing pure event logging
- `.kb/sessions/2026-03-01-debrief.md` — Actual debrief confirming summary pattern
- `~/.kb/models/behavioral-grammars/model.md` — Claims 3 (situational pull), 5 (self-detection failure), 7 (convergence attractor)
- `.kb/investigations/2026-02-28-inv-atc-lens-feature-audit.md` — Finding 3.1 (post-flight debrief is biggest gap)
- `.kb/models/orchestrator-session-lifecycle/model.md` — Session lifecycle patterns, strategic comprehender model
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-28-probe-session-debrief-artifact-design.md` — Confirmed interactive sessions lack durable comprehension
- `.kb/models/orchestrator-session-lifecycle/probes/2026-02-27-probe-communication-breakdown-postmortem-3-sessions.md` — Identity/action compliance gap under pressure
- `cmd/orch/debrief_cmd.go` — Debrief command implementation
- `pkg/debrief/debrief.go` — Debrief data model and rendering

**Related Artifacts:**
- **Model:** `~/.kb/models/behavioral-grammars/model.md` — Theoretical grounding for why infrastructure > instruction
- **Model:** `.kb/models/orchestrator-session-lifecycle/model.md` — Session lifecycle patterns this extends
- **Investigation:** `.kb/investigations/2026-02-28-inv-atc-lens-feature-audit.md` — Identified post-flight debrief as biggest gap

---

## Investigation History

**2026-03-05:** Investigation started
- Initial question: How should orchestrators produce comprehension instead of summaries, and what gates enforce it?
- Context: Dylan experiencing pure event logs where synthesis should be. Behavioral grammars predicts this (no infrastructure pull toward synthesis).

**2026-03-05:** Evidence gathering complete
- Read all 6 context sources. Confirmed: SYNTHESIZE is hollow, debrief template elicits summaries, orch debrief is event formatter, Three-Layer Reconnection maps to session scope, summary absence is detectable.

**2026-03-05:** Design complete with 5 forks navigated
- Fork 1: How to teach comprehension → Thread→Insight→Position cognitive moves
- Fork 2: Template change → "What Changed" becomes "What We Learned" (top position, interpretation demanded)
- Fork 3: Gate design → Advisory summary detection (regex for event-log patterns)
- Fork 4: Infrastructure → orch debrief enhancement + orch orient feedback loop
- Fork 5: Session-level analog → Three-Layer Reconnection at session scope

**2026-03-05:** Investigation completed
- Status: Complete
- Key outcome: Four-phase implementation plan — teach (skill+template), prompt (orch debrief), close loop (orch orient), measure (advisory gate)
