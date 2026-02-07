<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Load-bearing links belong in skill.yaml with a `load_bearing` array - this enables skillc verification without polluting SKILL.md or fragmenting knowledge.

**Evidence:** Analyzed 3 options against requirements. skill.yaml already has structured data (outputs, phases, spawn_requires); adding load_bearing follows the same pattern. skillc already parses skill.yaml and can verify patterns exist in compiled output.

**Knowledge:** The key insight is that load-bearing guidance is a *build-time constraint* (verify during compilation), not *runtime knowledge* (query during work). This matches skill.yaml's role as manifest of build requirements.

**Next:** Create decision record with data model, implement in Feature: orch-go-lv3yx.4

**Promote to Decision:** recommend-yes - This establishes the data model for an entire feature epic

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

# Investigation: Design Data Model Load Bearing

**Question:** What's the data model for linking friction → guidance? Where does canonical data live, how does skillc consume it, how do we query load-bearing patterns?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** orch-go worker agent
**Phase:** Complete
**Next Step:** None - ready for decision record
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: skill.yaml already supports structured metadata for build-time verification

**Evidence:** Current skill.yaml manifest supports:
- `outputs.required[]` - patterns that must exist at completion
- `phases[]` with `exit_criteria[]` - verification conditions for workflow phases
- `spawn_requires` - spawn-time behavior requirements
- `deliverables` - expected deliverables per skill

The Manifest struct in skillc already parses all these as typed fields.

**Source:** `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/manifest.go:73-91` - Shows existing pattern of structured YAML → Go struct → verification

**Significance:** Adding `load_bearing[]` follows an established pattern. No new infrastructure needed - just extend Manifest struct and add verification logic.

---

### Finding 2: kn/kb entries track *what we learned*, not *where it's used*

**Evidence:** kn entries have fields:
- `content` - what was decided/tried/constrained
- `reason` - why
- `type` - decision/constraint/attempt/question
- `source` - where the constraint came from (internal/external)

No field links to where guidance appears in skills. Entries are indexed by topic/tags, not by skill location.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kn/entries.jsonl` - Shows kn entry structure

**Significance:** kb is the wrong place for load-bearing links. It tracks knowledge atoms, not their deployment locations. Adding skill-location links would conflate "what we know" with "where we enforce it."

---

### Finding 3: SKILL.md inline comments would be lost during refactors

**Evidence:** The problem statement is that guidance gets swept away during refactors. If load-bearing metadata is inline in SKILL.md:
1. It's in the file being refactored
2. Agents doing refactors might treat comments as cruft
3. No external verification - the guard is inside the thing being guarded

HTML comments in SKILL.md would be compilation output, not source. But SKILL.md is generated from .skillc/ sources.

**Source:** Epic description orch-go-lv3yx - "When refactoring, the question isn't 'is this verbose?' but 'what breaks if we remove this?'"

**Significance:** The metadata must be *external* to SKILL.md to survive refactors. It must be in the source (.skillc/) not the output.

---

### Finding 4: skillc already has verification patterns (skillc verify)

**Evidence:** `skillc verify` validates:
- Output patterns exist
- D.E.K.N. sections present
- Checklist completion

This is the right model: declare constraints in skill.yaml, verify during build/deploy.

**Source:** `skillc verify --help` - "Validates: outputs.required patterns exist, D.E.K.N. sections, checklist completion"

**Significance:** Load-bearing verification slots naturally into `skillc verify` or as a new `skillc check` warning. The tooling model exists.

---

## Synthesis

**Key Insights:**

1. **Load-bearing is a build-time constraint, not runtime knowledge** - The goal is to prevent erosion during refactors. This is verification at compile/deploy time, not query at work time. skill.yaml is where build-time constraints live (Finding 1, 4).

2. **External guards survive what they guard** - Metadata must live outside SKILL.md to protect SKILL.md content from refactors. skill.yaml is source; SKILL.md is compiled output. Guards in source survive output changes (Finding 3).

3. **kb tracks knowledge atoms, not deployment locations** - Adding skill-location links to kb would conflate two different concerns. kb answers "what do we know?", skill.yaml answers "how do we enforce it?" (Finding 2).

**Answer to Investigation Question:**

**Where does canonical data live?** In skill.yaml as `load_bearing[]` array. Each entry specifies a pattern to verify and its provenance.

**How does skillc consume it?** During `skillc check` or `skillc verify`, search compiled output for each pattern. Warn if any load-bearing pattern is missing. Could be blocking (error) or advisory (warning) based on severity field.

**How do we query load-bearing patterns?** New command `skillc protected` lists all load-bearing patterns across skills. Before refactoring, run this to see what's protected and why.

---

## Structured Uncertainty

**What's tested:**

- ✅ skill.yaml already supports structured arrays (verified: read manifest.go:73-91)
- ✅ skillc verify already validates patterns (verified: ran skillc verify --help)
- ✅ kn entries don't have skill-location fields (verified: read entries.jsonl)

**What's untested:**

- ⚠️ Pattern matching performance with many load-bearing entries (not benchmarked)
- ⚠️ User experience of registering load-bearing links (no prototype yet)
- ⚠️ Whether "severity" field is needed or if all load-bearing should be errors

**What would change this:**

- If kb needed to answer "which skills use this constraint?" → would need bidirectional links
- If load-bearing metadata needs runtime query (not just build-time) → would need different storage
- [Falsifiability criteria]

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Option A: skill.yaml load_bearing array** - Add structured metadata to skill.yaml that skillc verifies during check/deploy.

**Data model:**
```yaml
load_bearing:
  - pattern: "ABSOLUTE DELEGATION RULE"
    provenance: "2025-11 orchestrator doing investigations led to 3-day derailment"
    evidence: ".kb/investigations/2025-11-xx-orchestrator-investigation-derailment.md"
    severity: error  # error = block deploy, warn = advisory
    
  - pattern: "Filter Before Presenting"
    provenance: "2026-01-08 Dylan observed option theater pattern"
    evidence: "orch-go-lv3yx epic description"
    severity: warn
```

**Why this approach:**
- Follows established skill.yaml pattern (outputs, phases, deliverables)
- External to SKILL.md - survives refactors
- skillc already parses and verifies - minimal new infrastructure
- Provenance field captures the friction story

**Trade-offs accepted:**
- Requires manual registration (vs auto-detection from kn entries)
- Patterns are strings, not semantic understanding
- No runtime query (kb context won't find these) - accepted because build-time verification is the goal

**Implementation sequence:**
1. Add LoadBearingEntry struct to manifest.go (foundational data model)
2. Add verification logic to checker.go (enforce during skillc check)
3. Add `skillc protected` command (query what's protected before refactoring)
4. Migrate existing hard-won patterns in orchestrator skill (orch-go-lv3yx.7)

### Alternative Approaches Considered

**Option B: kb friction command**
- **Pros:** Centralizes knowledge in kb, queryable via kb context
- **Cons:** kb tracks knowledge atoms, not enforcement locations. Adding skill-location links conflates two concerns. Also, kb is per-project but skills are cross-project.
- **When to use instead:** If we need runtime query of "which skills protect this insight" across projects

**Option C: Inline HTML comments in SKILL.md**
- **Pros:** Self-documenting, pattern and provenance in same place
- **Cons:** SKILL.md is compiled output, not source. Comments in output can be swept away during refactors - exactly what we're trying to prevent. Also harder to aggregate across skills.
- **When to use instead:** Never - this is the wrong model

**Rationale for recommendation:** skill.yaml is the manifest for build-time constraints. Load-bearing guidance is a build-time constraint. Therefore load-bearing metadata belongs in skill.yaml.

---

### Implementation Details

**What to implement first:**
- LoadBearingEntry struct in skillc/pkg/compiler/manifest.go (data model)
- YAML parsing for load_bearing array
- Basic existence check during skillc check

**Things to watch out for:**
- ⚠️ Pattern matching: Should be case-insensitive substring match, not exact
- ⚠️ Compiled output variations: SKILL.md may have different whitespace/formatting
- ⚠️ Cross-project skills: orchestrator skill is in orch-knowledge but used everywhere

**Areas needing further investigation:**
- Should severity default to 'error' or 'warn'? (lean error - if you register it, you care)
- Should patterns support regex or just substring? (lean substring - simpler)
- Should `skillc protected` aggregate across all skills or take a path? (lean aggregate)

**Success criteria:**
- ✅ Can add load_bearing entry to orchestrator skill.yaml
- ✅ `skillc check` warns/errors if pattern missing from compiled SKILL.md
- ✅ Refactoring agent that removes protected pattern gets blocked by check
- ✅ `skillc protected` shows all protected patterns before refactoring

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/skillc/pkg/compiler/manifest.go` - Current Manifest struct and parsing
- `/Users/dylanconlin/orch-knowledge/skills/src/worker/feature-impl/.skillc/skill.yaml` - Example skill.yaml with structured data
- `/Users/dylanconlin/orch-knowledge/skills/src/meta/orchestrator/.skillc/skill.yaml` - Orchestrator skill manifest
- `/Users/dylanconlin/Documents/personal/orch-go/.kn/entries.jsonl` - kn entry structure

**Commands Run:**
```bash
# Check kb quick capabilities
kb quick --help

# Check kn entry format
head -20 /Users/dylanconlin/Documents/personal/orch-go/.kn/entries.jsonl

# Check skillc capabilities
skillc --help

# Check kb context for friction-related knowledge
kb context "friction"
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Epic:** orch-go-lv3yx - Parent epic describing the problem
- **Decision:** .kb/decisions/2025-12-21-skillc-architecture-and-principles.md - skillc design principles
- **Issue:** orch-go-lv3yx.4 - Feature implementation using this data model

---

## Investigation History

**2026-01-08 08:15:** Investigation started
- Initial question: What's the data model for linking friction → guidance?
- Context: Epic orch-go-lv3yx created to protect load-bearing guidance from refactor erosion

**2026-01-08 08:20:** Examined existing skill.yaml patterns
- Found: outputs, phases, deliverables already use structured arrays with verification

**2026-01-08 08:25:** Analyzed kb/kn for friction storage
- Found: kn tracks knowledge atoms, not deployment locations - wrong abstraction layer

**2026-01-08 08:30:** Evaluated inline SKILL.md option
- Found: SKILL.md is output, not source - guards in output can't survive output changes

**2026-01-08 08:45:** Investigation completed
- Status: Complete
- Key outcome: load_bearing array in skill.yaml, verified by skillc check
