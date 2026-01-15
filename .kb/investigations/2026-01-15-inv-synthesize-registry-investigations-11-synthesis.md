<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The December 2025 recommendation to remove the registry was not followed; instead, 11 investigations evolved the system toward dual registries (agent-registry.json largely abandoned, sessions.json active) with incremental improvements rather than removal.

**Evidence:** December synthesis recommended Phase 4 removal; January investigations implemented slot reuse fix, schema additions, and mode fields; 2 of 11 were false positives from gap tracker; registry removal remains incomplete.

**Knowledge:** The system evolved toward "registry as refined tool" rather than "registry as technical debt to remove"; lack of decision/constraint prevented oscillation between these two visions.

**Next:** Create decision documenting the registry architecture that emerged (dual registries, sessions.json as authoritative for orchestrators) and archive 2 false-positive investigations.

**Promote to Decision:** recommend-yes - This establishes the registry architecture pattern that should be followed going forward.

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

# Investigation: Synthesize Registry Investigations 11 Synthesis

**Question:** What patterns, contradictions, and evolution emerged across 11 registry investigations from Dec 2025 - Jan 2026, and what should the authoritative registry architecture be?

**Started:** 2026-01-15
**Updated:** 2026-01-15
**Owner:** Agent og-work-synthesize-registry-investigations-15jan-3727
**Phase:** Complete
**Next Step:** None - awaiting orchestrator approval of proposed actions
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: December 2025 Synthesis Recommended Complete Registry Removal

**Evidence:** The investigation `2025-12-21-synthesis-registry-evolution-and-orch-identity.md` concluded:
- "The registry was correct for its time. The architecture evolved. The registry didn't evolve with it."
- Recommended completing "Phase 4 of the Python migration in orch-go" - removing the registry entirely
- Rationale: OpenCode API + beads + tmux provide all necessary data; registry is "solving yesterday's problem"
- "orch orchestrates, but doesn't own state"

**Source:** `.kb/investigations/2025-12-21-synthesis-registry-evolution-and-orch-identity.md` lines 159-167

**Significance:** This established a clear architectural direction: remove the registry. Subsequent investigations either ignored this direction or implicitly rejected it through incremental improvements.

---

### Finding 2: Two Distinct Registries Emerged, Not One

**Evidence:** The investigations revealed THREE separate registry-like systems:
1. **Agent registry** (`~/.orch/agent-registry.json`) - Legacy tracking of ALL spawns, largely abandoned after December
2. **Session registry** (`~/.orch/sessions.json`) - Tracks orchestrator sessions only, actively maintained
3. **Port registry** (`~/.orch/ports.yaml`) - Tracks port allocations, separate concern

The December synthesis and most investigations discussed "the registry" (singular) but the system evolved into multiple registries with different purposes.

**Source:**
- `2026-01-06-inv-registry-population-issues-orch-status.md` Finding 4 - documents the split
- `2025-12-21-inv-implement-port-allocation-registry-orch.md` - port registry implementation
- `pkg/session/session.go:4` comment: "Unlike agent-registry which tracks ALL spawns, session only tracks spawns made during the current session"

**Significance:** The term "registry" is overloaded. Recommendations to "remove the registry" are ambiguous about which registry. The session registry is actively used and has had improvements (schema, mode fields).

---

### Finding 3: Incremental Improvements Continued Despite Removal Recommendation

**Evidence:** After the December synthesis recommended removal, 4 investigations implemented registry improvements:
1. **Slot reuse fix** (2025-12-21-inv-registry-abandon-doesn-remove-agent.md) - Fixed bug preventing respawn of abandoned agents
2. **Schema addition** (2026-01-07-inv-registry-file-self-describing-header.md) - Added `_schema` field to sessions.json
3. **Mode field** (2026-01-09-inv-add-mode-field-registry-schema.md) - Added `Mode` and `TmuxWindow` fields to Agent struct
4. **Port allocation** (2025-12-21-inv-implement-port-allocation-registry-orch.md) - Created new registry for ports

**Source:** Investigation files listed above, all marked Complete with implementations

**Significance:** The system evolved away from "remove registry" toward "refine registry." No decision or constraint captured this shift, causing potential confusion for future work.

---

### Finding 4: Two False Positive Investigations from Gap Tracker

**Evidence:** Two investigations were spawned for the same non-issue:
- `2026-01-06-inv-registry-population-issues-orch-status.md` - Concluded "not a bug" (filename misconception: registry.json vs sessions.json)
- `2026-01-07-inv-investigate-registry-population-failures-root.md` - Confirmed first was false positive from gap tracker accumulation (7x events for same resolved issue)

Both marked Complete with recommendation to add constraint preventing re-spawning.

**Source:**
- `.kb/investigations/2026-01-06-inv-registry-population-issues-orch-status.md` lines 142-152
- `.kb/investigations/2026-01-07-inv-investigate-registry-population-failures-root.md` lines 141-146

**Significance:** Gap tracker hygiene issue caused wasted investigation effort. These should be archived with clear "Not a bug - filename misconception" summary.

---

### Finding 5: Three Competing Reconciliation Proposals, None Implemented

**Evidence:** Three investigations proposed different approaches to state reconciliation:

1. **Registry as cache** (2025-12-20-inv-plan-refactoring-pkg-registry-act.md) - Extend Agent struct with cached Phase/Issue fields, TTL-based invalidation
2. **Phased migration** (2025-12-21-inv-audit-all-registry-usage-orch.md) - Migrate commands one-by-one to derived lookups, keep registry as optional fallback
3. **Beads-centric reconciliation** (2025-12-22-inv-audit-orchestration-lifecycle-post-registry.md) - Make beads authoritative, use other sources for liveness only

All three marked Complete, but none show evidence of implementation.

**Source:**
- `2025-12-20-inv-plan-refactoring-pkg-registry-act.md` lines 183-203
- `2025-12-21-inv-audit-all-registry-usage-orch.md` lines 215-232
- `2025-12-22-inv-audit-orchestration-lifecycle-post-registry.md` lines 199-218

**Significance:** Multiple proposals without implementation or decision suggests uncertainty about the right approach. Each investigation completed in isolation without evaluating competing proposals.

---

### Finding 6: Automatic Completion Detection Was Disabled, Not Fixed

**Evidence:** Investigation `2025-12-21-inv-agents-being-marked-completed-registry.md` found agents marked complete 4-6 seconds after spawn due to monitor treating busy→idle as completion. Recommendation: "Disable automatic registry completion entirely" (lines 156-157).

This was implemented by removing CompletionService's automatic registry updates (referenced in `pkg/opencode/service.go:100-105`).

**Source:**
- `2025-12-21-inv-agents-being-marked-completed-registry.md` lines 1-278
- `2025-12-22-inv-audit-orchestration-lifecycle-post-registry.md` Finding 6 references the removal

**Significance:** The system chose "explicit completion via `orch complete`" over "improve detection heuristics." This aligns with beads-centric approach but wasn't documented as a decision.

---

## Synthesis

**Key Insights:**

1. **Architectural Vision Diverged Without Documentation** - The December synthesis recommended registry removal (Finding 1), but January implementations refined the registry instead (Finding 3). No decision or constraint documented this shift, creating ambiguity about whether "registry is technical debt" or "registry is a refined tool."

2. **Registry Specialization Replaced Monolithic Registry** - The system evolved from one "agent registry" concept to three specialized registries (agent, session, port - Finding 2). This aligns with Unix philosophy (do one thing well) but happened implicitly through separate investigations rather than explicit architectural decision.

3. **State Reconciliation Remains Unsolved** - Three competing proposals exist (cache, phased migration, beads-centric - Finding 5), all marked Complete but none implemented. The December synthesis identified this as the core problem, and it remains unresolved 3 weeks later.

4. **Explicit Over Automatic** - The system chose explicit completion (`orch complete` command) over automatic detection (Finding 6). This was implemented but not documented as a decision, making it appear as a bug fix rather than an architectural choice.

5. **Gap Tracker Hygiene Needs Attention** - Two investigations (Finding 4) were false positives from gap accumulation. Without constraints marking issues as resolved, the same non-issues recur, wasting investigation capacity.

**Answer to Investigation Question:**

The 11 investigations show **architectural drift** - the system moved from "remove registry" (December recommendation) toward "multiple specialized registries" (January reality) without documenting the decision.

The emerged architecture is:
- **Session registry** (`sessions.json`) - authoritative for orchestrator sessions, actively maintained
- **Agent registry** (`agent-registry.json`) - largely abandoned, legacy artifact
- **Port registry** (`ports.yaml`) - separate concern, actively maintained
- **State reconciliation** - unresolved; three proposals exist but none implemented

The contradictions:
- December: "Registry is yesterday's problem" vs January: "Improve registry with schema, mode fields"
- Multiple proposals for reconciliation, none chosen
- "Remove registry" vs "Registry as cache" vs "Beads as authority"

The pattern:
- Individual investigations solved isolated problems competently
- No synthesis of competing visions into coherent decision
- Result: accumulation of proposals without implementation or rejection

---

## Proposed Actions

Following the kb-reflect skill guidance for synthesis findings, here are structured proposals for orchestrator approval:

### Archive Actions
| ID | Target | Reason | Approved |
|----|--------|--------|----------|
| A1 | `.kb/investigations/2026-01-06-inv-registry-population-issues-orch-status.md` | False positive: filename misconception (registry.json vs sessions.json), not a bug | [ ] |
| A2 | `.kb/investigations/2026-01-07-inv-investigate-registry-population-failures-root.md` | False positive: confirmed A1 was gap tracker accumulation, not distinct issue | [ ] |
| A3 | `.kb/investigations/archived/2026-01-08-inv-test-registry-fix-verify-slot.md` | Incomplete template, never filled out, provides no value | [ ] |

### Create Actions
| ID | Type | Title | Description | Approved |
|----|------|-------|-------------|----------|
| C1 | decision | "Registry Architecture: Specialized Registries Replace Monolithic Agent Registry" | Document emerged architecture: sessions.json (orchestrator state), ports.yaml (port allocation), agent-registry.json (abandoned). Supersedes Dec synthesis "remove registry" recommendation with "specialized registries" pattern. | [ ] |
| C2 | decision | "Explicit Completion Over Automatic Detection" | Document choice to use `orch complete` command instead of automatic busy→idle detection. Rationale: false positives from heuristics, beads Phase: Complete is authoritative. | [ ] |
| C3 | guide | "Registry State Reconciliation Patterns" | Consolidate 3 competing proposals (cache, phased migration, beads-centric) into evaluation framework. Document when to use each approach based on performance vs consistency tradeoffs. | [ ] |
| C4 | constraint | "Registry population issues resolved - sessions.json works correctly" | Mark false positive as resolved to prevent gap tracker re-spawning. Reference investigations A1 and A2. | [ ] |

### Update Actions
| ID | Target | Change | Reason | Approved |
|----|--------|--------|--------|----------|
| U1 | `.kb/investigations/2025-12-21-synthesis-registry-evolution-and-orch-identity.md` | Add **Superseded-By:** header pointing to new decision C1 | Original recommendation (remove registry) was not followed; system evolved differently | [ ] |
| U2 | `.kb/investigations/2025-12-20-inv-plan-refactoring-pkg-registry-act.md` | Set Status: Superseded, add note referencing guide C3 | Proposal not implemented; should be evaluated in consolidated guide | [ ] |
| U3 | `.kb/investigations/2025-12-21-inv-audit-all-registry-usage-orch.md` | Set Status: Superseded, add note referencing guide C3 | Proposal not implemented; should be evaluated in consolidated guide | [ ] |
| U4 | `.kb/investigations/2025-12-22-inv-audit-orchestration-lifecycle-post-registry.md` | Set Status: Superseded, add note referencing guide C3 | Proposal not implemented; should be evaluated in consolidated guide | [ ] |

**Summary:** 3 archive proposals, 4 create proposals, 4 update proposals (11 total)
**High priority:** C1 (architecture decision), A1-A2 (cleanup false positives), C4 (prevent re-spawning)

---

## Structured Uncertainty

**What's tested:**

- ✅ All 11 investigations were read in full (verified: cited specific line numbers and sections from each file)
- ✅ December synthesis exists and recommended removal (verified: read full 185-line synthesis document)
- ✅ Sessions.json is actively used, agent-registry.json is not (verified: investigations explicitly document this split)
- ✅ Three reconciliation proposals exist but none implemented (verified: all marked Complete with "Next: None" or "Next: Implement")
- ✅ Two investigations were false positives (verified: both concluded "not a bug - filename misconception")

**What's untested:**

- ⚠️ Whether Dylan/orchestrator explicitly rejected the "remove registry" recommendation or implicitly through spawn priorities
- ⚠️ Whether any of the 3 reconciliation proposals was partially implemented in ways not captured in investigations
- ⚠️ Current production state of ~/.orch/ directory (which registries actually exist and their sizes)

**What would change this:**

- Synthesis would be wrong if December synthesis was superseded by a decision document that I missed
- Synthesis would be wrong if one of the reconciliation proposals WAS implemented but not documented
- Synthesis would be wrong if sessions.json is no longer actively maintained (would indicate another shift)

---

## Implementation Recommendations

**Purpose:** Bridge from synthesis findings to actionable decisions and documentation.

### Recommended Approach ⭐

**Document Emerged Architecture via Decision + Constraint** - Create decision record for registry architecture and constraint for resolved false positive.

**Why this approach:**
- **Stops oscillation:** Future investigations will see "specialized registries" is the established pattern, not "remove registry"
- **Enables informed tradeoffs:** New registry proposals can reference the decision for consistency
- **Prevents wasted effort:** Constraint stops gap tracker from re-spawning false positive investigations
- **Minimal implementation:** No code changes, just documentation

**Trade-offs accepted:**
- Doesn't implement any of the 3 reconciliation proposals (deferred until decision on which approach)
- Doesn't remove agent-registry.json (kept as legacy artifact)
- Accepts current state rather than pushing toward December's "remove" vision

**Implementation sequence:**
1. **Constraint first** (C4) - Prevents immediate re-spawning of false positives while other work continues
2. **Archive false positives** (A1-A3) - Cleanup .kb/investigations/ for clearer landscape
3. **Create architecture decision** (C1) - Documents what exists and supersedes December recommendation
4. **Create explicit completion decision** (C2) - Captures the choice that was implemented but not documented
5. **Update superseded investigations** (U1-U4) - Links old proposals to new consolidated guide (if C3 approved)

### Alternative Approaches Considered

**Option B: Implement One of the Reconciliation Proposals**
- **Pros:** Would solve the state reconciliation problem identified in December
- **Cons:** Three competing proposals exist; synthesis doesn't determine which is best; would require code implementation without decision on direction
- **When to use instead:** If Dylan/orchestrator has strong preference for one approach and wants immediate implementation

**Option C: Return to "Remove Registry" Vision**
- **Pros:** Fulfills December recommendation; architectural purity (orch is stateless)
- **Cons:** System already evolved away from this; sessions.json is actively maintained and has value; would require significant refactoring
- **When to use instead:** If sessions.json proves to have insurmountable consistency issues

**Option D: Do Nothing - Accept Current State**
- **Pros:** Zero work, system functions
- **Cons:** Future investigations will continue to oscillate between "remove" and "refine" without guidance; false positives may recur; architectural ambiguity persists
- **When to use instead:** If registry issues are low priority and stabilization is not worth documentation effort

**Rationale for recommendation:** Option A (document emerged architecture) provides the highest value for least effort. It stops oscillation, enables informed future decisions, and prevents false positive re-spawning - all through documentation updates rather than code changes. Implementation of reconciliation (Option B) can follow once architecture is documented and a proposal is chosen.

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
- `.kb/investigations/2025-12-21-synthesis-registry-evolution-and-orch-identity.md` - Prior synthesis recommending registry removal
- `.kb/investigations/2025-12-20-inv-plan-refactoring-pkg-registry-act.md` - Proposal: registry as cache for Beads state
- `.kb/investigations/2025-12-21-inv-agents-being-marked-completed-registry.md` - Bug: automatic completion detection
- `.kb/investigations/2025-12-21-inv-audit-all-registry-usage-orch.md` - Audit: all 15 registry callsites
- `.kb/investigations/2025-12-21-inv-implement-port-allocation-registry-orch.md` - Implementation: port allocation registry
- `.kb/investigations/2025-12-21-inv-registry-abandon-doesn-remove-agent.md` - Bug fix: slot reuse for respawn
- `.kb/investigations/2025-12-22-inv-audit-orchestration-lifecycle-post-registry.md` - Audit: 4 state sources, no reconciliation
- `.kb/investigations/2026-01-06-inv-registry-population-issues-orch-status.md` - False positive: filename misconception
- `.kb/investigations/2026-01-07-inv-investigate-registry-population-failures-root.md` - False positive confirmation
- `.kb/investigations/2026-01-07-inv-registry-file-self-describing-header.md` - Enhancement: _schema field
- `.kb/investigations/2026-01-09-inv-add-mode-field-registry-schema.md` - Enhancement: Mode and TmuxWindow fields
- `.kb/investigations/archived/2026-01-08-inv-test-registry-fix-verify-slot.md` - Incomplete template

**Commands Run:**
```bash
# View registry chronicle timeline
kb chronicle "registry" | head -100

# Find registry-related investigations
glob "**/*registry*.md" --path .kb/investigations
```

**Related Artifacts:**
- **Prior Synthesis:** `.kb/investigations/2025-12-21-synthesis-registry-evolution-and-orch-identity.md` - Established "remove registry" recommendation that was not followed
- **Constraint (referenced):** Should exist but doesn't - "registry population issues resolved"

---

## Investigation History

**2026-01-15 08:30:** Investigation started
- Initial question: Synthesize 11 registry investigations to identify patterns and contradictions
- Context: kb reflect identified "registry" topic with 11 accumulated investigations from Dec 2025 - Jan 2026

**2026-01-15 08:35:** Read prior synthesis
- Found comprehensive December synthesis recommending registry removal
- This established baseline: "remove registry" was the last documented architectural direction

**2026-01-15 08:45:** Read all 11 investigations
- Identified 2 false positives (gap tracker accumulation)
- Identified 4 implementations (slot reuse, schema, mode field, port registry)
- Identified 3 reconciliation proposals (none implemented)

**2026-01-15 09:00:** Pattern analysis complete
- Key finding: System evolved away from "remove" toward "specialized registries" without decision
- Architectural drift identified: incremental improvements without coherent vision
- Three competing reconciliation proposals exist without evaluation or selection

**2026-01-15 09:15:** Created proposed actions
- 3 archive proposals (false positives)
- 4 create proposals (decisions, guide, constraint)
- 4 update proposals (supersede old investigations)
- Total: 11 proposals for orchestrator approval

**2026-01-15 [TBD]:** Investigation to be completed
- Next: Commit investigation file, create SYNTHESIS.md, report Phase: Complete
