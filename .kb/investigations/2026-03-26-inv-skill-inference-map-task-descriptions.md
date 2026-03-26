<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Task descriptions only influence daemon skill routing after label and title checks miss, and the description parser maps text into three buckets: investigation, research, or detailed debugging.

**Evidence:** Verified in `pkg/daemon/skill_inference.go`, its focused tests in `pkg/daemon/skill_inference_test.go`, and passing runs of `go test ./pkg/daemon -run TestInferSkillFromDescription -v` and `go test ./pkg/daemon -run 'TestInferSkill|TestInferModelFromSkill|TestInferBrowserToolFromLabels|TestQuestionTypeRoutesToArchitect|TestOriginalBugReproduction'`.

**Knowledge:** The "NLP" layer is deterministic substring matching, not model inference, and vague descriptions intentionally fall through to coarse issue-type defaults such as `task -> feature-impl`.

**Next:** Close this investigation and use the documented routing chain when deciding whether an issue needs a `skill:*` label, a stronger title prefix, or a clearer description.

**Authority:** implementation - This is a code-path trace and documentation update inside existing daemon behavior, with no cross-system design change.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Authority: implementation - Tactical fix within existing patterns, no architectural impact

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Authority: Classify by who decides - implementation (worker within scope), architectural (orchestrator across boundaries), strategic (Dylan for irreversible/value choices)
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Skill Inference Map Task Descriptions

**Question:** How does `pkg/daemon/skill_inference.go` translate issue/task descriptions into skill types, and where does that description-based routing sit in the overall daemon inference pipeline?

**Started:** 2026-03-26
**Updated:** 2026-03-26
**Owner:** OpenCode
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Model:** daemon-autonomous-operation

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| `.kb/models/daemon-autonomous-operation/model.md` | extends | yes | - |
| `.kb/models/measurement-honesty/probes/2026-03-21-probe-issue-quality-baseline-inference-honesty.md` | confirms | yes | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: Description text is the third routing tier, not the first

**Evidence:** `InferSkillFromIssue()` checks sources in a fixed order: `skill:*` labels first, then title patterns, then description heuristics, and only then falls back to `InferSkill(issue.IssueType)`. For a plain `task`, that fallback is `feature-impl`; for `question`, it is `architect`.

**Source:** `pkg/daemon/skill_inference.go:21`; `pkg/daemon/skill_inference.go:214`; `pkg/daemon/daemon.go:376`; `pkg/daemon/preview.go:152`; `pkg/daemon/ooda.go:181`.

**Significance:** A task description only changes routing if no explicit skill label and no title cue already resolved the skill. This explains why some richly described issues still route by type when their titles already matched, or when their descriptions are too vague to trigger a heuristic.

---

### Finding 2: The description "NLP" is deterministic substring matching across three buckets

**Evidence:** `InferSkillFromDescription()` lowercases the description and scans it with `strings.Contains`. Investigation terms (`audit`, `analyze`, `investigate`, `understand`, `how does`, `why do`) map to `investigation`. Research terms (`compare`, `evaluate`, `research`, `best practice`, `what should we use`) map to `research`. Debugging terms (`fix`, `broken`, `error`, `crash`, `fails`, `failing`) only map to `systematic-debugging` when a second pass finds cause indicators like `error:`, `stack trace`, `returns`, `expected`, `actual`, `when i`, or `steps:`.

**Source:** `pkg/daemon/skill_inference.go:146`; `pkg/daemon/skill_inference_test.go:8`; `go test ./pkg/daemon -run TestInferSkillFromDescription -v`.

**Significance:** The code is not performing semantic classification; it is a rule-based keyword router. That makes the behavior legible and cheap, but also means phrasing matters a lot and descriptions that do not include these exact signals provide no routing lift.

---

### Finding 3: Vague descriptions intentionally drop to type fallback, and the daemon records which tier won

**Evidence:** When a description mentions a bug but not a concrete cause, `InferSkillFromDescription()` returns empty instead of guessing. The tests assert that `"Fix the authentication issue"`, `"The dashboard is broken"`, and `"There's an error in production"` produce no description match. `InferSkillFromIssue()` then logs booleans for label/title/description usage in `spawn.skill_inferred` events, while `inferSkillForScoring()` reuses the same chain without logging for queue scoring.

**Source:** `pkg/daemon/skill_inference.go:178`; `pkg/daemon/skill_inference.go:252`; `pkg/daemon/allocation.go:177`; `pkg/events/logger.go:711`; `cmd/orch/stats_inference.go:14`; `pkg/daemon/skill_inference_test.go:86`.

**Significance:** The system prefers predictable fallback over overconfident description parsing. Operationally, that means a vague task description usually ends up as `feature-impl` if the issue type is `task`, and later reporting can tell whether the daemon routed by label, title, description, or type.

---

## Synthesis

**Key Insights:**

1. **Descriptions are a rescue path for mislabeled tasks** - The daemon only consults description text after explicit `skill:*` labels and title cues fail, so descriptions mostly rescue generic `task` issues that were not enriched earlier.

2. **Routing uses lexical cues, not semantic understanding** - The parser is a lowercase-plus-substring matcher, which makes the inference chain auditable but sensitive to wording and blind to paraphrases outside the curated keyword lists.

3. **The pipeline is reused in both execution and analysis paths** - Actual spawn decisions in `daemon.go`, `preview.go`, and `ooda.go` use the logged inference path, while scoring in `allocation.go` mirrors the same chain without producing event noise.

**Answer to Investigation Question:**

`pkg/daemon/skill_inference.go` maps task descriptions to skills with a deterministic fallback heuristic: first it looks for investigation phrases, then research phrases, then detailed debugging phrases, and if none match it returns empty so the daemon falls back to issue type. That description stage is only tier 3 in the full routing pipeline (`skill:*` label -> title pattern -> description heuristic -> issue type), so description wording matters most for generic `task` issues that do not already advertise a skill in labels or titles.

---

## Structured Uncertainty

**What's tested:**

- ✅ Description keywords route to `investigation`, `research`, or `systematic-debugging` exactly as documented (verified with `go test ./pkg/daemon -run TestInferSkillFromDescription -v`).
- ✅ Type fallback still resolves `task -> feature-impl`, `bug -> systematic-debugging`, `investigation -> investigation`, `experiment -> investigation`, and `question -> architect` (verified with focused `go test ./pkg/daemon` runs).
- ✅ The daemon's runtime entrypoints call `InferSkillFromIssue()` before spawning and the analytics pipeline records which inference tier won (verified by code trace through daemon, preview, OODA, allocation, events, and stats packages).

**What's untested:**

- ⚠️ Corpus-level precision of the keyword lists against real issue data was not re-measured in this session.
- ⚠️ No live daemon spawn was executed, so the event payload was validated by code path and tests rather than by inspecting a fresh `events.jsonl` record.
- ⚠️ The gap between correct routing and successful completion remains out of scope for this trace.

**What would change this:**

- If another inference layer exists ahead of `InferSkillFromIssue()` for daemon spawns, this routing map would be incomplete.
- If tests or runtime evidence showed descriptions bypassing title precedence, the documented priority order would be wrong.
- If the keyword lists change, the exact description-to-skill map in this investigation will need refresh.

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Prefer `skill:*` labels or title prefixes when the desired skill is known, because description heuristics are later and narrower than they appear. | implementation | This is a usage recommendation inside the current routing design, not a system redesign. |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Issue-authoring guidance, not code changes** - Use explicit labels or stronger title wording when routing accuracy matters, and treat description text as a fallback hint rather than the primary control surface.

**Why this approach:**
- It matches the actual precedence in `InferSkillFromIssue()`.
- It avoids relying on brittle substring matches in the description parser.
- It directly reduces unwanted `task -> feature-impl` fallthrough.

**Trade-offs accepted:**
- Issue creators must do slightly more enrichment work up front.
- Some flexible natural-language descriptions will still route coarsely unless they use the recognized phrases.

**Implementation sequence:**
1. Add `skill:*` when the skill is already known.
2. If no label is appropriate, use a title prefix or first-word cue like `Design`, `Investigate`, or `Fix`.
3. Use description keywords to add context, but do not rely on them as the only routing signal.

### Alternative Approaches Considered

**Option B: Description-only routing**
- **Pros:** Less ceremony for issue authors.
- **Cons:** Loses precedence to labels and titles, and falls through on vague wording.
- **When to use instead:** When the issue is exploratory and no stronger routing signal is available yet.

**Option C: Type-only routing**
- **Pros:** Fully deterministic and cheap.
- **Cons:** Collapses many `task` issues into `feature-impl`, even when the work is really investigation or architecture.
- **When to use instead:** When triage intentionally wants the coarse default.

**Rationale for recommendation:** The precedence chain already rewards explicit signals, so the most reliable operational move is to use those signals instead of expecting the description parser to infer intent from free text.

---

### Implementation Details

**What to implement first:**
- No code implementation is recommended from this trace.
- If follow-up work is needed, start with issue-enrichment guidance rather than parser expansion.
- Any parser changes should be justified by misrouting evidence, not by intuition.

**Things to watch out for:**
- ⚠️ `strings.Contains` can over-match common phrases and under-match synonyms not in the list.
- ⚠️ Expanding the description keywords without evidence could increase false positives.
- ⚠️ Stats on completion rate still do not prove inference correctness.

**Areas needing further investigation:**
- How often real daemon-routed tasks are corrected by title or description rather than by type fallback.
- Which description phrases in production issues are near misses for the current keyword lists.
- Whether question-type routing in current daemon behavior fully matches the model documentation.

**Success criteria:**
- ✅ Future readers can explain the precedence chain without re-reading the code.
- ✅ Investigation artifacts capture the exact keyword buckets and fallback behavior.
- ✅ Verification evidence shows the documented routing rules match current tests.

---

## References

**Files Examined:**
- `pkg/daemon/skill_inference.go` - Primary inference chain, description heuristics, and model mapping.
- `pkg/daemon/skill_inference_test.go` - Behavioral expectations for description, title, and type routing.
- `pkg/daemon/daemon.go` - Main spawn path that calls `InferSkillFromIssue()`.
- `pkg/daemon/preview.go` - Preview path that reports the inferred skill/model pair.
- `pkg/daemon/ooda.go` - Decide-phase path using the same inference before route extraction.
- `pkg/daemon/allocation.go` - Scoring path that mirrors the inference chain without logging.
- `pkg/events/logger.go` - `spawn.skill_inferred` event payload fields.
- `cmd/orch/stats_inference.go` - Downstream interpretation of logged method flags.
- `.kb/models/daemon-autonomous-operation/model.md` - Prior synthesized understanding of daemon routing.
- `.kb/models/measurement-honesty/probes/2026-03-21-probe-issue-quality-baseline-inference-honesty.md` - Prior evidence about fallback dominance and measurement limits.

**Commands Run:**
```bash
# Verify repo location
pwd

# Run focused daemon inference tests
go test ./pkg/daemon -run 'TestInferSkill|TestInferModelFromSkill|TestInferBrowserToolFromLabels|TestQuestionTypeRoutesToArchitect|TestOriginalBugReproduction'

# Run description heuristic tests with named subtests
go test ./pkg/daemon -run TestInferSkillFromDescription -v

# Record the verified inference-chain takeaway in KB quick form
kb quick decide "Daemon skill inference resolves skill in a fixed order: skill label, title pattern, description heuristic, then issue type fallback." --reason "Verified in pkg/daemon/skill_inference.go and pkg/daemon tests during orch-go-hv9lc on 2026-03-26."
```

**External Documentation:**
- None.

**Related Artifacts:**
- **Model:** `.kb/models/daemon-autonomous-operation/model.md` - Existing daemon model that this investigation validates and sharpens.
- **Probe:** `.kb/models/measurement-honesty/probes/2026-03-21-probe-issue-quality-baseline-inference-honesty.md` - Prior measurement-focused evidence about the same inference chain.
- **Workspace:** `.orch/workspace/og-inv-skill-inference-map-26mar-7445/` - Workspace containing synthesis, brief, and verification artifacts for this session.

---

## Investigation History

**[2026-03-26 09:18]:** Investigation started
- Initial question: How do daemon task descriptions map to skill inference in `pkg/daemon/skill_inference.go`?
- Context: Needed a precise trace of the description-to-skill path and its place in the overall daemon routing chain.

**[2026-03-26 09:28]:** Core code path and tests verified
- Confirmed the four-tier routing order, exact description keyword buckets, and the type fallback behavior through source reads and focused `go test` runs.

**[2026-03-26 09:34]:** Investigation completed
- Status: Complete
- Key outcome: Description-based routing is a narrow tier-3 heuristic built from explicit substring lists, with vague descriptions intentionally dropping to issue-type defaults.
