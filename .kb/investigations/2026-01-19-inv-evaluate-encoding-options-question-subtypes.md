<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Labels (e.g., `subtype:factual`) are the recommended encoding for question subtypes - zero schema changes, already supported, matches existing patterns.

**Evidence:** Beads schema has `Labels []string` field; `bd ready --label subtype:factual` works today; daemon already filters by labels via `issue.HasLabel()`.

**Knowledge:** Question subtype encoding enables daemon to auto-spawn investigations for factual questions while deferring judgment/framing to orchestrator/Dylan.

**Next:** Adopt label convention `subtype:{factual|judgment|framing}` for questions; extend daemon to optionally spawn factual questions.

**Promote to Decision:** recommend-yes - Establishes encoding convention for the decidability graph model.

---

# Investigation: Evaluate Encoding Options Question Subtypes

**Question:** How should question subtypes (factual/judgment/framing) be encoded in beads to enable authority-aware daemon behavior?

**Started:** 2026-01-19
**Updated:** 2026-01-19
**Owner:** Architect agent (orch-go-7hd6h)
**Phase:** Complete
**Next Step:** None - recommendation delivered
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Beads schema already supports labels with no dedicated subtype field

**Evidence:** The `Issue` struct in `pkg/beads/types.go:141-154` shows:
```go
type Issue struct {
    ID           string          `json:"id"`
    Title        string          `json:"title"`
    ...
    Labels       []string        `json:"labels,omitempty"`
    ...
}
```

No `Subtype` or `QuestionType` field exists. Labels are freeform strings.

**Source:** `pkg/beads/types.go:141-154`

**Significance:** Adding a dedicated field would require upstream beads schema changes. Labels are already available and used extensively (e.g., `triage:ready`).

---

### Finding 2: Daemon already filters by labels

**Evidence:** The daemon in `pkg/daemon/daemon.go:331-343` checks for required labels:
```go
if d.Config.Label != "" && !issue.HasLabel(d.Config.Label) {
    // Check if this issue is a child of a triage:ready epic
    if _, isEpicChild := epicChildIDs[issue.ID]; !isEpicChild {
        if d.Config.Verbose {
            fmt.Printf("  DEBUG: Skipping %s (missing label %s, has %v)\n", issue.ID, d.Config.Label, issue.Labels)
        }
        continue
    }
}
```

Default config uses `Label: "triage:ready"` (line 93).

**Source:** `pkg/daemon/daemon.go:93`, `pkg/daemon/daemon.go:331-343`

**Significance:** The daemon's label filtering infrastructure is already in place. Adding support for question subtype labels would follow the same pattern.

---

### Finding 3: BD CLI supports label filtering for ready issues

**Evidence:** `bd ready --help` shows:
```
-l, --label strings       Filter by labels (AND: must have ALL)
    --label-any strings   Filter by labels (OR: must have AT LEAST ONE)
```

Tested: `bd ready --type question --label subtype:factual` would work today (no questions have this label yet).

**Source:** `bd ready --help`, `pkg/beads/types.go:120-121` (ReadyArgs.Labels)

**Significance:** The filtering mechanics exist. Daemon can query factual questions specifically via `bd ready --type question --label subtype:factual`.

---

### Finding 4: Questions are excluded from default bd ready

**Evidence:** From decidability graph model and testing:
- Questions have `type=question`
- `bd ready` excludes questions by default (they're not Work nodes)
- `bd ready --type question` shows questions separately
- Daemon's `IsSpawnableType()` rejects `question` type (only allows bug/feature/task/investigation)
  - **UPDATE 2026-01-27:** Resolved - `question` type now maps to `investigation` skill in daemon spawn logic

**Source:** `.kb/models/decidability-graph.md:246-260`, `pkg/daemon/issue_type.go`

**Significance:** Questions are already treated differently from Work. Subtype encoding adds another dimension - which questions the daemon CAN resolve vs which require escalation.

---

## Synthesis

**Key Insights:**

1. **Labels are the path of least resistance** - Zero schema changes, follows existing patterns (`triage:ready`), daemon infrastructure already supports it.

2. **Subtypes enable authority-aware behavior** - Factual questions can be auto-spawned as investigations. Judgment/framing questions surface to orchestrator/Dylan.

3. **Convention over enforcement** - Labels are freeform, so validation is by convention not schema. This matches the flexibility needed since questions can evolve (factual → framing).

**Answer to Investigation Question:**

Use labels with convention `subtype:{factual|judgment|framing}`:
- `subtype:factual` - "How does X work?" - Daemon can spawn investigation
- `subtype:judgment` - "Should we use X or Y?" - Orchestrator synthesizes
- `subtype:framing` - "Is X even the right question?" - Dylan reframes

This works today with no code changes. Future daemon enhancement can add `--label subtype:factual` filtering to auto-process factual questions.

---

## Structured Uncertainty

**What's tested:**

- ✅ Labels field exists in beads Issue struct (verified: read pkg/beads/types.go:148)
- ✅ Daemon filters by labels via HasLabel() (verified: read pkg/daemon/daemon.go:331)
- ✅ BD CLI supports label filtering (verified: bd ready --help shows --label flag)
- ✅ Question type exists in beads (verified: bd list --type question returns results)

**What's untested:**

- ⚠️ Actual daemon behavior with `subtype:factual` questions (no questions have this label yet)
- ⚠️ Performance impact of additional label filtering (likely negligible)
- ⚠️ Whether questions actually evolve subtypes in practice (hypothesis based on model)

**What would change this:**

- If beads adds a dedicated `QuestionSubtype` field in a future version, Option 2 becomes viable
- If label proliferation causes confusion, a stricter encoding might be needed
- If subtype detection can be reliably automated, Option 3 (inference) becomes attractive

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Labels with Convention** - Use `subtype:{factual|judgment|framing}` labels on question beads.

**Why this approach:**
- Zero schema changes - works today
- Follows existing pattern (`triage:ready` label)
- Labels can change as question evolves (factual → framing)
- Daemon already has label filtering infrastructure

**Trade-offs accepted:**
- No schema validation (freeform labels)
- Could have multiple subtypes (but convention: single subtype)
- Requires manual labeling at question creation

**Implementation sequence:**
1. **Convention adoption** - Document the convention in CLAUDE.md and decidability-graph model
2. **Question creation guidance** - Add `--labels subtype:factual` to bd create examples
3. **Daemon extension (optional)** - Add flag to spawn factual questions as investigations

### Alternative Approaches Considered

**Option B: Dedicated field in beads schema**
- **Pros:** Type-safe enum, single value enforcement, clear semantics
- **Cons:** Requires upstream beads changes, higher implementation cost, couples beads to orch-go concepts
- **When to use instead:** If label proliferation causes confusion, or if strict validation is required

**Option C: Inference from question content/context**
- **Pros:** Automatic, no manual labeling
- **Cons:** Unreliable classification, requires AI/heuristics, questions can change subtype during resolution
- **When to use instead:** When large volume of questions makes manual labeling impractical

**Rationale for recommendation:** Labels match the existing beads pattern, require no changes, and provide the flexibility needed for question subtypes that may evolve during resolution.

---

### Implementation Details

**What to implement first:**
- Document the convention: `subtype:{factual|judgment|framing}`
- Add examples to question creation workflow
- Update decidability graph model to reference this encoding

**Things to watch out for:**
- ⚠️ Subtype may not be known at question creation (may need to add later)
- ⚠️ Multiple labels possible - convention should be single subtype
- ⚠️ `answered` status still doesn't unblock dependencies (only `closed` does) - this is a separate constraint

**Areas needing further investigation:**
- Whether daemon should auto-spawn investigations for factual questions
- How dashboard should display questions by subtype
- Integration with `bd ready --type question --label subtype:factual`

**Success criteria:**
- ✅ Questions can be created with subtype labels
- ✅ Dashboard can show questions grouped by subtype
- ✅ Daemon can optionally filter/spawn factual questions

---

## References

**Files Examined:**
- `pkg/beads/types.go` - Issue struct, Labels field
- `pkg/daemon/daemon.go` - Label filtering in NextIssueExcluding
- `pkg/daemon/issue_adapter.go` - ListReadyIssues implementation
- `cmd/orch/serve_beads.go` - Questions API endpoint
- `.kb/models/decidability-graph.md` - Question subtypes definition

**Commands Run:**
```bash
# Check bd create options
bd create --help

# Check bd ready filtering
bd ready --help

# List open questions
bd list --type question --status open

# Show current question structure
bd show orch-go-7hd6h
```

**External Documentation:**
- N/A - All relevant documentation is internal

**Related Artifacts:**
- **Model:** `.kb/models/decidability-graph.md` - Defines question subtypes and authority levels
- **Decision:** `.kb/decisions/2026-01-18-questions-as-first-class-entities.md` - Question bead type
- **Constraint:** `kb-fe6173` - 'answered' status doesn't unblock dependencies

---

## Investigation History

**2026-01-19 22:41:** Investigation started
- Initial question: How should question subtypes be encoded?
- Context: Spawned from decidability graph model to determine encoding approach

**2026-01-19 22:55:** Found beads schema supports labels, no dedicated field
- Daemon already filters by labels
- BD CLI supports label filtering

**2026-01-19 23:00:** Investigation completed
- Status: Complete
- Key outcome: Recommend labels with convention `subtype:{factual|judgment|framing}`
