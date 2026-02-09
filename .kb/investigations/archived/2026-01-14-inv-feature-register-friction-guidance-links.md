<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented load-bearing guidance registration in skillc (manifest parsing, validation logic, tests, CLI output) per the 2026-01-08 data model decision.

**Evidence:** All tests pass (TestParseManifest_LoadBearing, TestValidateLoadBearing, TestCheckResult_*LoadBearing); end-to-end test shows patterns missing with error severity block checks, warn severity patterns produce warnings; skillc check displays provenance and evidence for missing patterns.

**Knowledge:** Implementation follows established patterns (OutputConstraints, BudgetResult, ChecksumResult); case-insensitive substring matching as specified; severity defaults to error when omitted; CLI display logic was already present in committed code.

**Next:** Close - implemented under .kb/decisions/2026-01-08-load-bearing-guidance-data-model.md.

**Promote to Decision:** recommend-no - Implementation task, no new architectural decisions made.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.
Promote to Decision: recommend-no (tactical fix, not architectural)

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Promote to Decision: flag for orchestrator/human - recommend-yes if this establishes a pattern, constraint, or architectural choice worth preserving
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Feature Register Friction Guidance Links

**Question:** How do we implement friction→guidance link registration in skillc?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** og-feat-feature-register-friction-14jan-622d
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** .kb/decisions/2026-01-08-load-bearing-guidance-data-model.md

---

## Findings

### Finding 1: Data model already defined in decision document

**Evidence:** LoadBearingEntry struct specification with fields: pattern (string), provenance (string), evidence (string), severity (string). YAML structure shown with examples.

**Source:** .kb/decisions/2026-01-08-load-bearing-guidance-data-model.md:92-106

**Significance:** Implementation is straightforward - no design work needed, just translate the Go struct to manifest.go and add parsing/validation.

---

### Finding 2: skillc is a separate repository

**Evidence:** skillc source code found at ~/Documents/personal/skillc with pkg/compiler/manifest.go (156 lines) and pkg/checker/checker.go (237 lines). No manifest.go or checker.go in orch-go repo.

**Source:** File exploration via glob, found files at /Users/dylanconlin/Documents/personal/skillc/pkg/

**Significance:** Work needs to be done in skillc repo, not orch-go. This is a cross-repo feature implementation.

---

### Finding 3: Existing validation patterns in checker.go

**Evidence:** checker.go already validates checksums, token budgets, and markdown links. Has CheckResult struct that aggregates validation results with HasErrors() and HasWarnings() methods. Uses severity-based blocking (budget errors block, checksum/links warn).

**Source:** /Users/dylanconlin/Documents/personal/skillc/pkg/checker/checker.go:39-63

**Significance:** Can follow established patterns for load-bearing validation - add LoadBearingResult to CheckResult, implement ValidateLoadBearing(), integrate with Check().

---

## Synthesis

**Key Insights:**

1. **Cross-repo feature** - Work needs to be done in skillc repo (~/Documents/personal/skillc), not orch-go, despite being tracked as orch-go-lv3yx.4 issue. This is by design since skillc is the tool that processes skill.yaml files.

2. **Follow established patterns** - checker.go already has validation infrastructure with severity-based blocking (errors block, warnings advise). Load-bearing validation should integrate with existing CheckResult aggregation pattern.

3. **Minimal new code** - Data model is already defined in decision document. Implementation is straightforward struct addition + validation function following existing patterns.

**Answer to Investigation Question:**

Implementation requires: (1) Add LoadBearingEntry struct to manifest.go with YAML tags, (2) Add LoadBearing []LoadBearingEntry field to Manifest, (3) Add ValidateLoadBearing() to checker.go that searches compiled output for each pattern, (4) Add LoadBearingResult to CheckResult, (5) Integrate with Check() function. Testing patterns established in manifest_test.go and checker_test.go should be followed. Uncertainty remains about kb friction CLI command mentioned in spawn context vs skill.yaml approach in decision.

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
