<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** CLAUDE.md contained references to deleted pkg/registry/, incorrect cmd/orch/ listing, and duplicated model section content.

**Evidence:** File system check showed no pkg/registry/ directory, cmd/orch/ has 100+ files not 4, model section lines 164-169 repeated identical content 3 times.

**Knowledge:** Documentation drift occurs when code structure changes but documentation isn't updated; systematic verification needed.

**Next:** Changes committed; close issue after verification.

**Authority:** implementation - Direct documentation fixes with no architectural impact.

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

# Investigation: Fix Claude Md Remove Deleted

**Question:** What documentation errors exist in CLAUDE.md regarding deleted code and duplicated content?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** orch-go-wdl agent
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

### Finding 1: pkg/registry/ reference exists but directory deleted

**Evidence:** CLAUDE.md lines 37-38 referenced pkg/registry/ with description "Agent state management", but `ls pkg/` showed no registry directory among 35 actual directories.

**Source:** CLAUDE.md:37-38, `ls -la pkg/` output

**Significance:** Documentation referenced deleted code, misleading readers about current architecture.

---

### Finding 2: cmd/orch/ listing showed 4 files, actually has 100+

**Evidence:** CLAUDE.md lines 22-26 listed only main.go, daemon.go, resume.go, wait.go, but `ls cmd/orch/` showed 100+ .go files including spawn_cmd.go, complete_cmd.go, serve*.go, session.go, etc.

**Source:** CLAUDE.md:22-26, `ls -la cmd/orch/` output showing 100 files

**Significance:** Oversimplified listing didn't reflect actual command structure, making it harder to understand codebase organization.

---

### Finding 3: Model section duplicated 3 times

**Evidence:** CLAUDE.md lines 164-169 showed identical content repeated:
- Lines 164-165: `Resolve(spec)` and aliases
- Lines 166-167: Exact duplicate
- Lines 168-169: Exact duplicate

**Source:** CLAUDE.md:164-169 (visual inspection in file read)

**Significance:** Copy-paste error created redundant documentation, cluttering the file.

---

## Synthesis

**Key Insights:**

1. **Documentation drift happens during refactoring** - When pkg/registry/ was removed, the CLAUDE.md reference wasn't cleaned up, creating misleading architecture documentation.

2. **Simplified listings become inaccurate** - The cmd/orch/ listing was likely accurate when first written (4 files), but became misleading as the codebase grew to 100+ files without the listing being updated.

3. **Copy-paste errors accumulate** - The duplicated model section suggests a copy-paste error during editing that went unnoticed.

**Answer to Investigation Question:**

CLAUDE.md had three documentation errors: (1) reference to deleted pkg/registry/ on lines 37-38, (2) oversimplified cmd/orch/ listing showing 4 files when 100+ exist, and (3) duplicated model section content on lines 166-169. All three have been fixed by removing the registry reference, updating the cmd/orch/ listing to reflect actual structure, and removing duplicate lines.

---

## Structured Uncertainty

**What's tested:**

- ✅ [Claim with evidence of actual test performed - e.g., "API returns 200 (verified: ran curl command)"]
- ✅ [Claim with evidence of actual test performed]
- ✅ [Claim with evidence of actual test performed]

**What's untested:**

- ⚠️ [Hypothesis without validation - e.g., "Performance should improve (not benchmarked)"]
- ⚠️ [Hypothesis without validation]
- ⚠️ [Hypothesis without validation]

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

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
