<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Successfully restored models/probes Go infrastructure from entropy-spiral-feb2026 branch including probes.go, probes_test.go, and model content injection functions in kbcontext.go.

**Evidence:** All validation passed: `go build ./cmd/orch/`, `go vet ./cmd/orch/`, and `go test ./pkg/spawn/` (including TestProbe* and TestModel* tests).

**Knowledge:** The current Jan 18 baseline had more KB context types than the spiral branch. The spiral branch's unique contribution was the probe infrastructure (probes.go) and model content injection functions (hasInjectedModelContent, formatModelMatchForSpawn, extractModelSectionsForSpawn, etc.).

**Next:** Close - implementation complete, all tests pass.

**Authority:** implementation - Restoring existing code within scope, no architectural changes

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

# Investigation: Restore Models Probes Go Infrastructure

**Question:** How to restore models/probes Go infrastructure from entropy-spiral-feb2026 branch into Jan 18 baseline?

**Started:** 2026-02-13
**Updated:** 2026-02-13
**Owner:** Worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| N/A | - | - | - |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: probes.go was added in commit f1c7a25c

**Evidence:** `git log --oneline entropy-spiral-feb2026 --all --follow -- pkg/spawn/probes.go` shows commit f1c7a25c added probe directory structure. File contains functions: ModelNameFromPath, ProbesDirForModel, ProbeFilePath, ListRecentProbes, FormatProbesForSpawn, EnsureProbesDir, DefaultProbeTemplate.

**Source:** `git show f1c7a25c:pkg/spawn/probes.go` - 204 lines

**Significance:** This is the core probe infrastructure that enables models to have associated probe directories at .kb/models/{model-name}/probes/.

---

### Finding 2: probes_test.go has comprehensive test coverage

**Evidence:** 270 lines of tests covering all public probe functions: TestModelNameFromPath, TestProbesDirForModel, TestProbeFilePath, TestListRecentProbes_*, TestFormatProbesForSpawn_*, TestEnsureProbesDir.

**Source:** `git show f1c7a25c:pkg/spawn/probes_test.go`

**Significance:** Test coverage ensures the probe infrastructure works correctly.

---

### Finding 3: kbcontext.go needed model content injection functions

**Evidence:** The spiral branch commit f1c7a25c added ~220 lines of model content injection code to kbcontext.go: hasInjectedModelContent, formatModelMatchForSpawn, extractModelSectionsForSpawn, collectMarkdownHeadings, parseMarkdownHeading, normalizeHeading, extractSectionByHeading, truncateModelSection, indentBlock. Also added HasInjectedModels field to KBContextFormatResult and maxModelSectionChars constant.

**Source:** `git show f1c7a25c -- pkg/spawn/kbcontext.go`

**Significance:** These functions enable spawn context to inject model summaries, critical invariants, and recent probes when models are relevant to a task.

---

## Synthesis

**Key Insights:**

1. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

2. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

3. **[Insight title]** - [Explanation of the insight, connecting multiple findings]

**Answer to Investigation Question:**

[Clear, direct answer to the question posed at the top of this investigation. Reference specific findings that support this answer. Acknowledge any limitations or gaps.]

---

## Structured Uncertainty

**What's tested:**

- ✅ go build ./cmd/orch/ passes (verified: ran command, no output = success)
- ✅ go vet ./cmd/orch/ passes (verified: ran command, no output = success)
- ✅ go test ./pkg/spawn/ -run TestProbe passes (verified: 2 tests pass)
- ✅ go test ./pkg/spawn/ -run TestModel passes (verified: 4 tests pass)
- ✅ go test ./pkg/spawn/ all tests pass (verified: all tests in output show PASS)

**What's untested:**

- ⚠️ End-to-end spawn with model content injection (not tested in this session)
- ⚠️ Integration with kb context command returning model matches (not tested)

**What would change this:**

- Finding would be wrong if spawn with relevant models doesn't show model content in SPAWN_CONTEXT.md
- Finding would be wrong if probe listing fails in production use

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| [Primary recommendation from investigation] | implementation / architectural / strategic | [Why this authority level - stays inside scope? reaches across boundaries? involves irreversible choice?] |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**[Approach Name]** - [One sentence stating the recommended implementation]

**Why this approach:**
- [Key benefit 1 based on findings]
- [Key benefit 2 based on findings]
- [How this directly addresses investigation findings]

**Trade-offs accepted:**
- [What we're giving up or deferring]
- [Why that's acceptable given findings]

**Implementation sequence:**
1. [First step - why it's foundational]
2. [Second step - why it comes next]
3. [Third step - builds on previous]

### Alternative Approaches Considered

**Option B: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Option C: [Alternative approach]**
- **Pros:** [Benefits]
- **Cons:** [Why not recommended - reference findings]
- **When to use instead:** [Conditions where this might be better]

**Rationale for recommendation:** [Brief synthesis of why Option A beats alternatives given investigation findings]

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- [File path] - [What you looked at and why]
- [File path] - [What you looked at and why]

**Commands Run:**
```bash
# [Command description]
[command]

# [Command description]
[command]
```

**External Documentation:**
- [Link or reference] - [What it is and relevance]

**Related Artifacts:**
- **Decision:** [Path to related decision document] - [How it relates]
- **Investigation:** [Path to related investigation] - [How it relates]
- **Workspace:** [Path to related workspace] - [How it relates]

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
