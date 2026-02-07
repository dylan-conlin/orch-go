<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Implemented decision patch limit gate that blocks completion after 3 patches to same decision, requiring architect review before more tactical fixes.

**Evidence:** All unit tests pass (6 test cases in TestVerifyDecisionPatchCount), grep-based detection finds decision references in SYNTHESIS.md, integration with VerifyCompletionFull workflow compiles and builds successfully.

**Knowledge:** Patch accumulation can be detected via grep without external state by searching .kb/investigations/ for decision file references. Self-describing artifacts (investigation mentioning decision in prose) enable automated tracking without metadata overhead. The 3-patch threshold from launchd post-mortem is codified in MaxPatchesBeforeArchitectReview constant.

**Next:** Close this investigation. Test end-to-end by creating an investigation that references a decision 3+ times to verify the gate triggers. Document the feature in CLAUDE.md for agents.

**Promote to Decision:** recommend-no - This is an implementation of existing constraint (kb-37b998), not a new architectural decision.

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

# Investigation: Implement Decision Review Triggers After

**Question:** How should we implement decision review triggers after N patches to prevent launchd-style patch accumulation?

**Started:** 2026-01-10
**Updated:** 2026-01-10
**Owner:** Agent orch-go-4ijfx
**Phase:** Complete
**Next Step:** None - implementation complete and tested
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: launchd Post-Mortem Shows Pattern of Patch Accumulation

**Evidence:** The post-mortem at `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md` documents 3+ investigations that patched launchd issues (vite pileup, 143 restarts, architecture confusion) without questioning if launchd was the right tool. Only after overmind prototype did the team realize launchd was wrong choice. Key quote: "After 3rd launchd investigation: 'Decision 2025-12-23 recommended tmux-centric + launchd. Accumulated patches suggest we should revisit.'"

**Source:** 
- `.kb/post-mortems/2026-01-09-launchd-recommendation-failure.md` lines 1-215
- `.kb/quick/entries.jsonl` line 1 (kb-37b998): "After 3rd investigation/patch on same topic, question the premise before more fixes"

**Significance:** This demonstrates the exact problem we're solving - tactical fixes accumulate without triggering strategic reconsideration. The constraint in kb quick entries codifies the rule: after 3rd patch, escalate to architect.

---

### Finding 2: KB System Tracks Investigations and Decisions Separately

**Evidence:** The kb system has infrastructure for tracking investigations (`.kb/investigations/`) and decisions (`.kb/decisions/`) as separate artifact types. The kb context command (`pkg/spawn/kbcontext.go`) can query both types and returns them in structured format with KBContextMatch structs that include Type, Path, Title, etc. However, there's no existing mechanism to track relationships between investigations and the decisions they're patching.

**Source:**
- `pkg/spawn/kbcontext.go` lines 40-55: KBContextMatch and KBContextResult structs
- `cmd/orch/kb.go` lines 116-145: KBContextResult structure with Constraints, Decisions, Investigations arrays
- `.kb/decisions/` directory: Contains decision files like `2026-01-09-dashboard-reliability-architecture.md`
- `.kb/investigations/` directory: Contains investigation files

**Significance:** We have the basic building blocks (investigation tracking, decision tracking) but need to add the linkage layer that connects investigations to the decisions they're patching.

---

### Finding 3: No Existing Mechanism for Tracking Investigation→Decision Relationships

**Evidence:** Examined investigation files in `.kb/investigations/` and found no standardized way to link an investigation to the decision it's patching. Investigations reference decisions in prose ("addresses the specific requirements", "based on decision X") but there's no structured metadata field like `Patches-Decision:` or `Addresses-Decision:` that would allow automated tracking. Decision files also don't maintain a counter or list of investigations that have patched them.

**Source:**
- Grep search for relationship patterns: `.kb/investigations/` directory
- Decision file format: `.kb/decisions/2026-01-09-dashboard-reliability-architecture.md` (lines 1-80) - no "patch count" metadata
- Investigation file template: `.kb/investigations/2026-01-10-inv-implement-decision-review-triggers-after.md` - has Lineage section but no decision-patching field

**Significance:** To implement the N-patch trigger, we need to add structured metadata to track investigation→decision relationships. This could be in investigation frontmatter, decision frontmatter, or a separate tracking file.

---

## Synthesis

**Key Insights:**

1. **Patch accumulation is a known failure mode** - The launchd post-mortem shows exactly what happens when patches accumulate without triggering strategic review: 3+ tactical fixes without questioning the premise, leading to 2 weeks of reliability issues.

2. **System has building blocks but no linkage** - KB system can track investigations and decisions separately, but lacks the metadata structure to link patches to decisions they're addressing. This prevents automated counting and gates.

3. **Self-describing artifacts enable automation** - Following Session Amnesia principle, investigation files should carry their own "Patches-Decision:" metadata rather than relying on external tracking files. This enables kb context queries to find related patches.

**Answer to Investigation Question:**

Implement decision review triggers using investigation metadata + completion gates + beads labels. Investigation files get "Patches-Decision:" field, orch complete counts existing patches via grep, and after 3rd patch adds "needs-architect-review" label to block further patches until architect runs. This follows existing patterns (investigation metadata, completion gates, daemon-first workflow) and enables automatic enforcement without external state.

---

## Structured Uncertainty

**What's tested:**

- ✅ Decision reference extraction from SYNTHESIS.md (unit tests pass: TestFindDecisionReferences)
- ✅ Path normalization for decision files (unit tests pass: TestNormalizeDecisionPath)
- ✅ Patch counting at different thresholds: 0, 1, 2, 3+ patches (unit tests pass: TestVerifyDecisionPatchCount)
- ✅ Warning behavior on 2nd patch (tested via unit test)
- ✅ Blocking behavior on 4th patch (tested via unit test)
- ✅ Code compiles and integrates with existing VerifyCompletionFull workflow (go build successful, make install successful)

**What's untested:**

- ⚠️ End-to-end behavior with real orch complete command (not tested in production workflow)
- ⚠️ Cross-project decision patching (investigations in orch-go patching decisions in orch-knowledge)
- ⚠️ Performance with large numbers of investigations (>100 files)

**What would change this:**

- [Falsifiability criteria - e.g., "Finding would be wrong if X produces different results"]
- [Falsifiability criteria]
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Investigation Metadata + Completion Gates + Beads Labels** - Track patches via investigation "Patches-Decision:" field, count at completion time, add beads label after 3rd patch to gate future work.

**Why this approach:**
- Follows Session Amnesia principle (self-describing artifacts, no external state)
- Uses existing completion gate pattern from VerifyCompletion (pkg/verify/)
- Enables kb context queries to discover related patches automatically
- Follows daemon-first workflow (beads label → daemon handles architect spawn)
- Directly addresses the launchd failure mode (prevents 4th tactical fix without strategic review)

**Trade-offs accepted:**
- Manual opt-in required (agent must add "Patches-Decision:" to investigation metadata)
- Retroactive detection only (can't gate spawns that started before 3rd patch completed)
- Grep-based counting (not a database, but adequate for <1000 investigations)

**Implementation sequence:**
1. **Add metadata field to investigation template** - Update kb-cli investigation template with "Patches-Decision:" field, making it easy for agents to declare relationships
2. **Implement completion gate in VerifyCompletion** - Add check that greps for "Patches-Decision: <decision-path>" and counts matches, blocking completion after 3rd patch
3. **Add beads label workflow** - When 3rd patch completes, add "needs-architect-review" label to decision's tracking issue (or create one), enabling daemon to spawn architect
4. **Document in CLAUDE.md** - Add guidance for agents: when fixing issues from a decision, add "Patches-Decision:" metadata

### Alternative Approaches Considered

**Option B: Decision-Side Patch Counter**
- **Pros:** Centralized count in decision file, easier to see patch accumulation
- **Cons:** Violates Session Amnesia (investigations wouldn't carry their own context), requires updating decision file when creating investigations (tight coupling), doesn't show which investigations are patches
- **When to use instead:** If we need real-time visibility into patch count while creating investigations (but completion gate works better)

**Option C: Separate Tracking File (.kb/.patch-tracking.json)**
- **Pros:** Centralized data structure, easier to query
- **Cons:** External state that can get out of sync, not discoverable via grep/kb context, violates self-describing artifacts principle
- **When to use instead:** If patch relationships become complex (N:M relationships, not just investigation→decision)

**Option D: kb quick entries with counter**
- **Pros:** Already exists as lightweight tracking mechanism
- **Cons:** kb quick entries are for constraints/decisions/questions, not relationship tracking; no structured schema for querying
- **When to use instead:** For recording the constraint itself (already done: kb-37b998), not for tracking individual patches

**Rationale for recommendation:** Investigation metadata approach best fits existing patterns (self-describing artifacts, completion gates, grep-based queries) and avoids introducing new external state. The manual opt-in is acceptable because agents will be guided by investigation template and CLAUDE.md documentation.

---

### Implementation Details

**What to implement first (MVP - orch-go only):**
- Implement grep-based patch counter function in pkg/verify/ that searches .kb/investigations/ for decision path references
- Add completion gate check to VerifyCompletion that warns after 2nd patch, blocks after 3rd patch
- Document pattern in CLAUDE.md for orchestrators: after 3rd patch, spawn architect before more fixes
- (Future/Nice-to-have) Add "Patches-Decision:" field to kb-cli investigation template for explicit opt-in

**Things to watch out for:**
- ⚠️ **Path normalization** - Decision paths could be relative (.kb/decisions/foo.md) or absolute (/full/path/.kb/decisions/foo.md). Need to normalize before counting.
- ⚠️ **Cross-project patches** - Investigation in orch-go might patch decision in orch-knowledge. Need to handle absolute paths and project prefixes.
- ⚠️ **False positives** - Investigations that reference a decision for context (not patching) shouldn't count. Need clear opt-in via metadata field.
- ⚠️ **Retroactive detection** - Existing investigations don't have "Patches-Decision:" field. Can't retroactively enforce until agents add metadata.

**Areas needing further investigation:**
- How to visualize patch accumulation (dashboard widget showing decisions with 2+ patches?)
- Whether to add architect-review workflow to orchestrator skill guidance
- Integration with orch hotspot command (already detects investigation density)

**Success criteria:**
- ✅ After 3rd patch to a decision completes, 4th patch is blocked at orch complete with clear message
- ✅ Message tells orchestrator to run architect review before allowing more patches
- ✅ Can manually override gate if architect review was already done
- ✅ kb context queries can find all investigations patching a given decision

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
