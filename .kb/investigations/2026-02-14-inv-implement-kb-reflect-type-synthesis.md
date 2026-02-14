<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** kb reflect --type synthesis does not check for model existence, only guides/decisions, and scans archived/synthesized directories creating false positives.

**Evidence:** buildSynthesizedTopicsSet (reflect.go:503-547) omits models directory; filepath.Walk (561-569) has no path filtering; confirmed by reading source code.

**Knowledge:** Model promotion requires: 1) checking .kb/models/ directory, 2) excluding archived/synthesized/ from scans, 3) suggesting "create model" when no model exists.

**Next:** Implement model detection in buildSynthesizedTopicsSet, add path filtering to exclude archived/synthesized, modify suggestion logic.

**Authority:** implementation - Extending existing pattern with same logic, fixing known bug, within feature scope

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

# Investigation: Implement Kb Reflect Type Synthesis

**Question:** How should kb reflect --type synthesis detect investigation clusters that should become models vs guides?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** feature-impl agent
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

### Finding 1: Current synthesis detection only checks guides and decisions, not models

**Evidence:** The `buildSynthesizedTopicsSet` function (reflect.go:503-547) only scans .kb/guides/ and .kb/decisions/ directories. It does not check .kb/models/ directory.

**Source:** /Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go:503-547

**Significance:** This means kb reflect --type synthesis will flag topic clusters even if a model already exists for that domain, creating false positives. The task requires detecting when NO model exists to suggest model promotion.

---

### Finding 2: Investigation scanning does NOT exclude archived/ or synthesized/ directories

**Evidence:** The filepath.Walk at line 561 in findSynthesisCandidates walks the entire investigations directory without filtering out archived/ or synthesized/ subdirectories. The kb reflect Cluster Hygiene model (spawn context line 82-93) documents this as Failure Mode 5.

**Source:** /Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go:561-569, spawn context prior knowledge

**Significance:** This causes false positives where already-consolidated investigations in synthesized/ directories are re-flagged for synthesis. This needs to be fixed as part of the implementation.

---

### Finding 3: SynthesisCandidate structure already supports flexible suggestions

**Evidence:** The SynthesisCandidate struct (line 26-32) has a `Suggestion` field that currently suggests "kb create guide". This can be modified to suggest "kb create model" when appropriate.

**Source:** /Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/reflect.go:26-32, line 611

**Significance:** The existing structure supports the change - we just need to modify the suggestion logic to detect when a model should be created instead of a guide.

---

### Finding 4: Model naming follows kebab-case pattern

**Evidence:** Examined .kb/models/ directory - filenames use kebab-case: agent-lifecycle-state-model.md, beads-integration-architecture.md, spawn-architecture.md, etc.

**Source:** ls /Users/dylanconlin/Documents/personal/orch-go/.kb/models/

**Significance:** To check if a model exists for a topic, we need to convert the normalized topic to kebab-case and check for {topic}.md in the models directory. Some models have subdirectories for probes (e.g., beads-integration-architecture/).

---

## Synthesis

**Key Insights:**

1. **Model detection is missing from synthesis** - buildSynthesizedTopicsSet only checks guides and decisions (Finding 1). This means clusters get flagged even when a model exists, missing the opportunity to suggest model-specific actions like probe creation.

2. **Directory exclusion prevents false positives** - The archived/ and synthesized/ directories should be excluded from scanning (Finding 2). This is a known issue (Failure Mode 5 in kb reflect Cluster Hygiene model) that needs fixing.

3. **Suggestion field enables model promotion** - The existing SynthesisCandidate.Suggestion field (Finding 3) can be modified to recommend "kb create model {topic}" instead of "kb create guide {topic}" when no model exists.

4. **Model existence check requires kebab-case lookup** - Models use kebab-case filenames (Finding 4), so checking requires converting normalized topics (e.g., "daemon" → "daemon.md" or "daemon-autonomous-operation.md").

**Answer to Investigation Question:**

kb reflect --type synthesis should be enhanced to:
1. Extend buildSynthesizedTopicsSet to also check .kb/models/ directory
2. Exclude archived/ and synthesized/ subdirectories from investigation scanning (fixes Failure Mode 5)
3. Modify suggestion logic: when 3+ investigations cluster on a topic with no model, suggest "kb create model {topic}" instead of "kb create guide {topic}"
4. This aligns with the Feb 8 model-centric probes decision: model exists → probes, no model → investigations, 3+ investigations → create model

---

## Structured Uncertainty

**What's tested:**

- ✅ Verified buildSynthesizedTopicsSet does not check models (read source code at line 503-547)
- ✅ Verified filepath.Walk does not exclude archived/ or synthesized/ (read source code at line 561-569)
- ✅ Verified SynthesisCandidate.Suggestion field exists and is flexible (read struct definition line 26-32)
- ✅ Verified model naming pattern is kebab-case (ls .kb/models/)

**What's untested:**

- ⚠️ Exact algorithm for matching normalized topics to model filenames (hypothesis: direct match or substring match needed)
- ⚠️ Whether to suggest model OR guide, or both in some cases (hypothesis: model only when investigation cluster suggests mechanism understanding)

**What would change this:**

- Finding would be wrong if buildSynthesizedTopicsSet already checks models (would need different approach)
- Finding would be wrong if there's already exclusion logic for archived/synthesized (grep would find it)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Extend buildSynthesizedTopicsSet to check models directory | implementation | Within scope - enhancing existing function with same pattern |
| Exclude archived/synthesized from investigation scan | implementation | Within scope - fixing known bug (Failure Mode 5) |
| Modify suggestion logic based on model existence | implementation | Within scope - conditional suggestion based on artifact existence |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Extend synthesis detection with model awareness** - Modify findSynthesisCandidates to check for model existence and suggest model creation when appropriate, while fixing the archived/synthesized scan issue.

**Why this approach:**
- Reuses existing SynthesisCandidate structure (Finding 3) - minimal new code
- Fixes Failure Mode 5 (Finding 2) as side effect - two bugs, one PR
- Aligns with Feb 8 model-centric probes decision (spawn context) - correct artifact routing

**Trade-offs accepted:**
- Simple substring matching for model detection may have false positives (e.g., "spawn" matches "spawn-architecture.md" and "tmux-spawn-guide.md")
- Not distinguishing between model vs guide heuristically - just checking existence

**Implementation sequence:**
1. Add archived/synthesized exclusion to filepath.Walk - prevents false positives before adding model logic
2. Create buildModelTopicsSet function similar to buildSynthesizedTopicsSet - scans .kb/models/ directory
3. Modify findSynthesisCandidates to check model existence and adjust suggestion - "create model" vs "create guide"

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
- Add path filtering to exclude archived/ and synthesized/ subdirectories in filepath.Walk
- Create buildModelTopicsSet function by copying buildSynthesizedTopicsSet pattern
- Modify findSynthesisCandidates to call buildModelTopicsSet and adjust suggestion string

**Things to watch out for:**
- ⚠️ Model filenames may not exactly match normalized topics (e.g., "daemon" vs "daemon-autonomous-operation.md") - may need substring matching
- ⚠️ Some models have subdirectories (probes/) - need to check both .md files and directories
- ⚠️ Ensure backward compatibility - existing synthesis suggestions should still work when model exists

**Areas needing further investigation:**
- Whether to suggest model AND guide, or just model (task description implies just model)
- Heuristics for when to suggest model vs guide (deferred - just use existence check for now)
- Should existing tests be updated or new tests added?

**Success criteria:**
- ✅ kb reflect --type synthesis shows "kb create model {topic}" suggestion for clusters without models
- ✅ kb reflect --type synthesis does NOT flag clusters in archived/ or synthesized/ directories
- ✅ Existing synthesis behavior preserved for topics with guides/decisions

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
