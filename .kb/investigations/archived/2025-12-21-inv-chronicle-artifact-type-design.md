## Summary (D.E.K.N.)

**Delta:** Chronicle should be a VIEW over existing artifacts, not a new artifact type. The structure already exists - git history + kn entries + investigations + decisions provide temporal data; what's missing is a query command (`kb chronicle "topic"`) that synthesizes them.

**Evidence:** Analyzed existing "chronicle-like" artifact (`2025-12-21-synthesis-registry-evolution-and-orch-identity.md`) - it was manually authored by orchestrator synthesizing git history, kn entries, and investigations. All source data exists; only the view is missing.

**Knowledge:** Chronicles capture decision evolution narratives that span multiple artifacts. The structure is narrative (not timeline or graph) because the value is in the "why" arc, not just "what happened when". Creator should be orchestrator (synthesis is its job), triggered by pattern detection or user query.

**Next:** Create `kb chronicle "topic"` command that queries across git + kn + investigations + decisions to surface evolution narrative. Close issue.

**Confidence:** High (85%) - clear evidence from existing practice, but untested implementation

---

# Investigation: Chronicle Artifact Type Design

**Question:** Should Chronicle be a new artifact type or a view over existing artifacts? What structure makes sense (timeline, narrative, graph)? Who creates it (orchestrator synthesis, automated from git history)? How does it capture decision evolution narratives?

**Started:** 2025-12-21
**Updated:** 2025-12-21
**Owner:** og-inv-chronicle-artifact-type-21dec
**Phase:** Complete
**Next Step:** None - recommendation ready
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: A Chronicle Already Exists - It Was Authored Manually

**Evidence:** The file `2025-12-21-synthesis-registry-evolution-and-orch-identity.md` is a de facto chronicle:

- Title: "Synthesis: Registry Evolution and Orch Identity"
- Purpose: "Weave together the full narrative of orch's evolution"
- Structure: Chronological narrative tracing Nov 29 → Dec 21
- Content: Links git history, architectural decisions, kn constraints, investigations
- Format: Narrative with tables for timelines

It was created manually by an orchestrator agent synthesizing across sources. The sources it drew from:
- Git history ("793 commits", "Dec 18 decision")
- Architectural decisions ("Dec 1 five concerns")
- Investigations ("Dec 6 investigation proposed phased migration")
- Observed patterns ("four-layer reconciliation complexity")

**Source:** `.kb/investigations/2025-12-21-synthesis-registry-evolution-and-orch-identity.md` (185 lines)

**Significance:** The chronicle pattern exists and works. What's missing is tooling to assist creation - the orchestrator had to manually trace git history, find kn entries, locate investigations.

---

### Finding 2: All Source Data for Chronicles Already Exists

**Evidence:** Tested query capabilities across existing artifact types:

```
Artifact counts:
- Investigations: 172
- kn entries: 30+ (with timestamps, types, reasons)
- Decisions: 2
- SYNTHESIS.md files: 68
- Git commits with context: 793+ in 22 days
```

The temporal data IS captured:
- **Git history:** Commit messages include decisions, investigations, features
- **kn entries:** Have `created_at` timestamps and explicit reasoning
- **Investigations:** Have `Started/Updated` dates
- **Decisions:** Have `Date` and `Context` sections

What's NOT captured: Cross-artifact evolution narrative. Each artifact is point-in-time; no artifact type naturally captures "how did X evolve?"

**Source:** 
- `.kn/entries.jsonl` - 30 entries with timestamps
- `.kb/investigations/*.md` - 172 files with dates
- `git log --format="%ad: %s"` - chronological commit history

**Significance:** Chronicle is not about capturing NEW data - it's about surfacing EXISTING data in a new way. This suggests VIEW, not new artifact type.

---

### Finding 3: Structure Should Be Narrative, Not Timeline or Graph

**Evidence:** Compared the manually-created registry chronicle structure:

**Timeline format (rejected):**
```
Nov 29: Registry created
Dec 1: Five concerns architecture
Dec 6: Registry removal investigation
Dec 8: Registry stripped to minimal
Dec 18: Go + OpenCode decision
Dec 21: Registry drift causing pain
```

**Graph format (rejected):**
```
Registry → Five Concerns → Beads Integration → OpenCode API → Registry Removal
```

**Narrative format (what was actually written):**
```
## The Core Identity (Nov 29, 2025)
From day one, orch was conceived as "kubectl for AI agents"...

## Why the Registry Exists
The registry was necessary because: [reasoning]...

## The Pivot: Go + OpenCode (Dec 18)
This eliminated the need for: [impact]...
```

The narrative format provides:
1. **Why** things happened, not just what
2. Causal connections between events
3. Insight synthesis (the "ah-ha" that emerges)
4. Decision guidance (what to do next)

**Source:** Structural analysis of `2025-12-21-synthesis-registry-evolution-and-orch-identity.md`

**Significance:** Pure timelines show chronology but miss causation. Graphs show connections but miss narrative. Chronicles need narrative structure because the VALUE is in understanding the decision evolution arc.

---

### Finding 4: Orchestrator Is The Right Creator

**Evidence:** From the orchestrator skill, synthesis is explicitly an orchestrator responsibility:

```
**Orchestrators do THREE things:**
1. **DELEGATE** - Create issues, spawn agents
2. **TRIAGE** - Review issues, adjust scope
3. **SYNTHESIZE** - Combine results from completed agents
```

And:
```
**Orchestrator Core Responsibilities (Never Delegate):**
- **Cross-agent synthesis** - Combining findings from multiple agent executions
- **Knowledge integration** - Promoting findings to decisions, linking artifacts
```

The registry chronicle was created by orchestrator synthesis, not by a spawned agent or automation.

**Why automated generation would fail:**
- Narrative requires judgment (what's important?)
- Causation requires understanding (why did X lead to Y?)
- Insight synthesis requires reasoning (what does this mean?)

LLMs can assist but shouldn't fully automate - curation matters.

**Source:** 
- `~/.claude/skills/policy/orchestrator/SKILL.md` lines 63-74
- Observation that existing chronicle was orchestrator-created

**Significance:** Chronicle creation is orchestrator work - part of synthesis responsibility. Automation can provide source data, but narrative requires orchestrator judgment.

---

### Finding 5: Triggers For Chronicle Creation

**Evidence:** The registry chronicle was triggered by a question:

> "What should orch be?"

This led to backward tracing:
1. Why is registry causing problems? (drift)
2. Why does drift happen? (cache vs source of truth)
3. Why was registry created? (historical constraints)
4. What changed? (OpenCode API)
5. What now? (remove registry)

**Natural triggers for chronicles:**
- "How did X become Y?" (evolution question)
- Pattern recognition: "Multiple investigations about X without synthesis"
- Decision preparation: "Need to understand context for decision on X"
- Onboarding: "How does the system handle X?"

**Source:** SESSION_HANDOFF.md mentions "the question progression" as the trigger

**Significance:** Chronicles are demand-driven (answer a question) or pattern-triggered (system notices gap). Not routinely created.

---

### Finding 6: Chronicle-Like Queries Are Already Possible

**Evidence:** Tested queries that could power chronicle generation:

```bash
# Timeline for a topic
git log --format="%ad: %s" --date=short -- .kb/decisions/*.md

# kn entries about a topic
cat .kn/entries.jsonl | jq -r 'select(.content | test("registry"))...'

# Investigations about a topic
rg -l "registry" .kb/investigations/
```

These return raw data. What's missing is a unified query that:
1. Searches across all sources (git, kn, kb, beads)
2. Sorts by date
3. Groups by topic
4. Presents as synthesis-ready input

**Source:** Test commands run during investigation

**Significance:** A `kb chronicle "registry"` command could gather source material. Orchestrator then synthesizes into narrative.

---

## Synthesis

**Key Insights:**

1. **Chronicle = View, not artifact type** - The data already exists across git, kn, investigations, decisions. Chronicle is a way of presenting that data, not new data to capture. This matches the minimal artifact taxonomy principle (5 essential + 3 supplementary - don't add more).

2. **Narrative structure is essential** - Timelines show events; graphs show connections; narratives show causation and meaning. The value of a chronicle is understanding "why did X evolve to Y?" which requires narrative.

3. **Orchestrator creates, tooling assists** - Automated chronicle generation would lose the judgment and insight that makes chronicles valuable. The right pattern: tooling gathers sources, orchestrator synthesizes narrative.

4. **Demand or pattern triggered** - Chronicles answer evolution questions. Triggers: explicit question ("how did X evolve?"), pattern detection ("5 investigations about X, no synthesis"), or decision preparation.

**Answer to Investigation Question:**

**Is Chronicle a new artifact type or a view over existing artifacts?**
VIEW over existing. The minimal artifact taxonomy (5 essential + 3 supplementary) is sufficient. Adding a new type violates the minimalism principle and creates overhead without new information.

**What's the structure?**
NARRATIVE. Not timeline (misses causation), not graph (misses meaning). Narrative captures the "why" arc that makes decision evolution understandable.

**Who creates it?**
ORCHESTRATOR synthesis, assisted by tooling. Automated generation would lose curation judgment. Pattern: `kb chronicle "topic"` gathers sources → orchestrator writes narrative.

**How does it capture decision evolution?**
By tracing across time: constraints discovered → decisions made → patterns emerged → principles crystallized. The registry chronicle example traces: initial constraints → registry creation → architecture conflict → OpenCode changes equation → registry becomes vestigial.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Strong evidence from existing practice (registry chronicle exists and works), clear alignment with minimal taxonomy decision, and testable recommendation (can implement `kb chronicle`). Limited by lack of testing the proposed tooling.

**What's certain:**

- ✅ Chronicle pattern already works (registry evolution file proves it)
- ✅ Source data exists (git + kn + kb all have temporal data)
- ✅ Narrative structure is preferred (observed in manual example)
- ✅ Orchestrator is the right creator (matches synthesis responsibility)

**What's uncertain:**

- ⚠️ How expensive is `kb chronicle` query across large artifact sets?
- ⚠️ What's the threshold for "pattern triggers chronicle creation"?
- ⚠️ Should chronicles be saved or ephemeral?

**What would increase confidence to Very High (95%+):**

- Implement `kb chronicle "topic"` and test on 3+ topics
- Test orchestrator workflow: query → synthesize → save or discard
- Validate that view-based approach scales to 500+ artifacts

---

## Implementation Recommendations

### Recommended Approach ⭐

**Chronicle as View via `kb chronicle` command** - Create a query command that gathers temporal data about a topic, presents it to orchestrator for narrative synthesis.

**Why this approach:**
- Aligns with minimal taxonomy (no new artifact type)
- Leverages existing data (git, kn, kb, beads)
- Preserves orchestrator judgment (human-in-loop synthesis)
- Fits existing patterns (`kb context` already queries, this adds temporal dimension)

**Trade-offs accepted:**
- Not fully automated (requires orchestrator synthesis)
- Chronicles are ephemeral unless explicitly saved to `.kb/investigations/`
- Depends on quality of existing artifacts (garbage in, garbage out)

**Implementation sequence:**
1. **Add `kb chronicle "topic"` command** - Query git, kn, kb, beads for topic matches with dates
2. **Output sorted timeline** - Present as structured input for orchestrator
3. **Orchestrator synthesizes** - Manual narrative creation from sources
4. **Optional save** - If valuable, save to `.kb/investigations/YYYY-MM-DD-synthesis-{topic}.md`

### Alternative Approaches Considered

**Option B: Chronicle as new artifact type**
- **Pros:** Explicit type, clear schema, first-class citizen
- **Cons:** Violates minimal taxonomy; overhead for infrequent use
- **When to use instead:** If chronicles become common (>10% of artifacts)

**Option C: Automated chronicle generation**
- **Pros:** No orchestrator effort, could be triggered by patterns
- **Cons:** Loses judgment; narratives require insight; would produce low-quality output
- **When to use instead:** For simple timelines (not narratives)

**Option D: Graph-based structure**
- **Pros:** Shows connections visually, can compute metrics
- **Cons:** Graphs don't convey causation or meaning
- **When to use instead:** For citation analysis (separate from chronicle)

**Rationale for recommendation:** Option A matches observed practice (registry chronicle was created this way), requires no new artifact types, and preserves the value-add of orchestrator synthesis while automating the tedious gathering step.

---

## Test Performed

**Test:** Analyzed existing chronicle-like artifact (`2025-12-21-synthesis-registry-evolution-and-orch-identity.md`) to understand how it was created, what sources it drew from, and what structure emerged. Also tested queries across git, kn, and kb to verify source data availability.

**Result:** 
- Chronicle was manually created by orchestrator
- All source data (git, kn, investigations) is queryable with timestamps
- Narrative structure emerged naturally (not timeline or graph)
- Orchestrator synthesis added value (causation, insight, recommendations)

---

## Self-Review

- [x] Real test performed (analyzed existing chronicle, tested queries)
- [x] Conclusion from evidence (based on observed practice)
- [x] Question answered (view vs type, structure, creator, mechanism)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled (summary section complete)

**Self-Review Status:** PASSED

---

## Leave it Better

```bash
kn decide "Chronicle should be a view over existing artifacts, not new artifact type" --reason "Minimal taxonomy principle; source data already exists in git/kn/kb; value is in narrative synthesis not data capture"
```

---

## References

**Files Examined:**
- `.kb/investigations/2025-12-21-synthesis-registry-evolution-and-orch-identity.md` - Existing chronicle example
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Minimal artifact set decision
- `.kb/investigations/2025-12-21-design-deep-pattern-analysis-orchestration-artifacts.md` - Artifact architecture
- `~/.claude/skills/policy/orchestrator/SKILL.md` - Orchestrator synthesis responsibility
- `~/.kb/principles.md` - Foundational principles including session amnesia
- `.kn/entries.jsonl` - kn entries with temporal data
- `.orch/SESSION_HANDOFF.md` - Session handoff showing chronicle trigger

**Commands Run:**
```bash
# Count artifacts
ls -1 .kb/investigations/*.md | wc -l  # 172 investigations
ls -1 .orch/workspace/*/SYNTHESIS.md | wc -l  # 68 synthesis files

# Query git history for decisions
git log --format="%ad: %s" --date=short -- .kb/decisions/*.md

# Query kn entries timeline
cat .kn/entries.jsonl | jq -r '.created_at[:10] + ": " + .type + " - " + .content[:80]'

# Search for topic across artifacts
rg -l "registry" .kb/
cat .kn/entries.jsonl | jq -r 'select(.content | test("registry"))'
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Establishes 5+3 artifact types
- **Investigation:** `.kb/investigations/2025-12-21-design-deep-pattern-analysis-orchestration-artifacts.md` - Artifact architecture
- **Epic:** orch-go-ws4z - System Self-Reflection (parent of this task)

---

## Investigation History

**2025-12-21 16:25:** Investigation started
- Initial question: Should chronicle be new artifact type or view over existing?
- Context: Part of orch-go-ws4z epic on System Self-Reflection

**2025-12-21 16:35:** Found existing chronicle example
- Discovered `2025-12-21-synthesis-registry-evolution-and-orch-identity.md`
- Analyzed structure: narrative, not timeline or graph
- Confirmed: manually created by orchestrator

**2025-12-21 16:45:** Tested query capabilities
- Verified git, kn, kb all have temporal data
- Tested cross-source queries
- Confirmed: data exists, view is missing

**2025-12-21 16:55:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Chronicle = view via `kb chronicle` command, orchestrator synthesizes narrative
