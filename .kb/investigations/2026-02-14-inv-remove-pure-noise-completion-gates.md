<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The three pure-noise gates (agent_running, model_connection, commit_evidence) were already removed from the codebase - they only exist in historical events logs and knowledge base documentation.

**Evidence:** Exhaustive grep search across entire repo shows zero references in Go code (pkg/verify/, cmd/orch/), only in .kb/ investigation files and archived workspaces.

**Knowledge:** These gates were identified by the probe as pure noise (∞:1, 71:1, and 11.8:1 bypass:fail ratios) but had already been removed from the code in a prior cleanup. The task is to document their removal in the completion-verification model, not remove code.

**Next:** Update .kb/models/completion-verification.md to document that these gates were removed due to noise patterns, verify build/tests pass.

**Authority:** implementation - Documentation update within existing patterns, no code changes needed

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

# Investigation: Remove Pure Noise Completion Gates

**Question:** Where are the three pure-noise gates (agent_running, model_connection, commit_evidence) defined in the codebase, and what needs to be removed?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** og-feat-remove-pure-noise-14feb-bd3d
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

### Finding 1: Gates Not Present in Current Codebase

**Evidence:** Grep search across entire repo for `agent_running|model_connection|commit_evidence` in *.go files returned zero results. Checked pkg/verify/check.go constants (lines 13-26) - only 11 gates defined: phase_complete, synthesis, session_handoff, handoff_content, constraint, phase_gate, skill_output, visual_verification, test_evidence, git_diff, build, decision_patch_limit.

**Source:** `grep -r "agent_running|model_connection|commit_evidence" /Users/dylanconlin/Documents/personal/orch-go --include="*.go"`; `pkg/verify/check.go:13-26`

**Significance:** The gates identified as pure noise don't exist in the verification code - they must have been removed in a prior cleanup.

---

### Finding 2: Gates Only Exist in Knowledge Base Documentation

**Evidence:** Found 9 references total, all in .kb/ directories: 5 in investigation file (2026-02-13-inv-probe-inventory-friction-gates-across.md), 3 in probe file (2026-02-13-friction-gate-inventory-all-subsystems.md), 1 in archived workspace SYNTHESIS.md. Zero references in pkg/ or cmd/ directories.

**Source:** Full grep results show only .kb/investigations/, .kb/models/completion-verification/probes/, and .orch/workspace/archived/ paths

**Significance:** These gates only exist as historical references in documentation, not in executable code. The probe itself notes: "Some gates in events (`agent_running`, `model_connection`, `commit_evidence`) don't appear in current code constants — they may be from a prior codebase version."

---

### Finding 3: No Skip Flags in complete_cmd.go

**Evidence:** Reviewed cmd/orch/complete_cmd.go skip flag definitions (lines 40-53). Found 11 skip flags: test_evidence, visual, git_diff, synthesis, build, constraint, phase_gate, skill_output, decision_patch, phase_complete, handoff_content. No flags for agent_running, model_connection, or commit_evidence.

**Source:** `cmd/orch/complete_cmd.go:40-53`

**Significance:** The CLI already has no bypass flags for these gates, confirming they were completely removed from the verification system.

---

## Synthesis

**Key Insights:**

1. **Gates Already Removed** - The three pure-noise gates were removed from the codebase in a prior cleanup. They exist only as historical references in events logs (analyzed by the probe) and knowledge base documentation.

2. **Probe Analyzed Historical Data** - The 2026-02-13 probe analyzed events.jsonl logs which contain verification events from previous completion attempts, including bypasses/failures of gates that no longer exist in the code.

3. **Documentation Lag** - The completion-verification model hasn't been updated to document that these gates were removed. This creates confusion for future investigations that might search for gate implementations.

**Answer to Investigation Question:**

The three gates (agent_running, model_connection, commit_evidence) are not defined anywhere in the current codebase. They were already removed from pkg/verify/check.go and cmd/orch/complete_cmd.go in a prior cleanup. The only remaining work is to update .kb/models/completion-verification.md to document their removal in the Evolution section, explaining why they were removed (pure noise with ∞:1, 71:1, and 11.8:1 bypass:fail ratios). No code changes are needed.

---

## Structured Uncertainty

**What's tested:**

- ✅ Gates not in pkg/verify/check.go constants (verified: read file, searched all 642 lines)
- ✅ Gates not in cmd/orch/complete_cmd.go skip flags (verified: read flags section, no matches)
- ✅ No Go code references exist (verified: grep -r across entire repo returned zero .go files)

**What's untested:**

- ⚠️ Whether events.jsonl still contains references (probable based on probe analysis, but not verified)
- ⚠️ Whether historical git commits show when gates were removed (could trace removal, but not critical)
- ⚠️ Whether removing model documentation references will break any tooling (unlikely, model is human-readable only)

**What would change this:**

- Finding would be wrong if grep search in pkg/verify/ or cmd/orch/ returned any gate constant definitions
- Finding would be wrong if SkipConfig struct in complete_cmd.go contained fields for these gates
- Finding would be wrong if verification.bypassed events in code referenced these gate names

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Document gate removal in completion-verification model | implementation | Documentation update within existing patterns, no architectural impact or cross-boundary effects |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Document Removal in Model** - Update .kb/models/completion-verification.md Evolution section to document that agent_running, model_connection, and commit_evidence gates were removed due to pure noise patterns identified in probe analysis.

**Why this approach:**
- Prevents future confusion when investigators search for gate implementations
- Preserves historical context about why gates were removed (noise ratios, model compatibility issues)
- Requires only documentation update, zero code risk

**Trade-offs accepted:**
- Not removing historical references from investigation files (preserves probe analysis trail)
- Not attempting to trace exact removal date in git history (not critical for understanding)

**Implementation sequence:**
1. Add Phase 7 to Evolution section documenting gate removal (builds on existing phase pattern)
2. Verify build and tests still pass (confirms no unexpected dependencies)
3. Mark investigation complete

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
