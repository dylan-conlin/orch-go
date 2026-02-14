<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Successfully added optional metadata?: Record<string, string> field to OpenCode Session.Info schema and wired it through POST /session and PATCH /session/:id endpoints, then updated orch-go to pass beads_id, workspace_path, tier, and spawn_mode during session creation.

**Evidence:** OpenCode builds without TypeScript errors (npx tsc --noEmit), orch-go builds without errors (go build ./cmd/orch && go build ./pkg/opencode), metadata field added to Session.Info Zod schema, SessionTable SQL schema, fromRow/toRow mappings, Session.create.schema, and PATCH endpoint validator.

**Knowledge:** OpenCode uses Drizzle ORM with JSON columns for complex types like metadata, session creation/update follows a consistent pattern with dedicated setter functions (setTitle, setArchived, setMetadata), and orch-go passes metadata during CreateSession calls in headless and inline spawn modes.

**Next:** Commit changes to both repos, then optionally test by creating a session and verifying metadata is persisted and retrievable.

**Authority:** implementation - Changes follow established patterns within both codebases, no architectural decisions required.

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

# Investigation: Add Session Metadata Api Opencode

**Question:** How do I add optional metadata field to OpenCode Session.Info schema and wire it through the API endpoints?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** Agent orch-go-ac2
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

### Finding 1: Session.Info Zod schema in session/index.ts

**Evidence:** Session.Info schema defined at lines 114-155 in `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/index.ts`. Currently has fields: id, slug, projectID, directory, parentID, summary, share, title, version, time, permission, revert. No metadata field exists.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/index.ts:114-155`

**Significance:** This is where I need to add the metadata?: Record<string, string> field to the Zod schema.

---

### Finding 2: SQL table schema in session.sql.ts

**Evidence:** SessionTable defined in `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/session.sql.ts` using Drizzle ORM. Has columns matching Session.Info fields but stored in snake_case (project_id, parent_id, etc.). No metadata column exists.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/session.sql.ts:11-35`

**Significance:** Need to add metadata column to SessionTable using text({ mode: "json" }) similar to how permission is stored.

---

### Finding 3: fromRow and toRow transformation functions

**Evidence:** `fromRow` (line 48-79) converts SQL row to Session.Info. `toRow` (line 81-102) converts Session.Info to SQL row. Both functions manually map each field between snake_case SQL and camelCase TypeScript.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/session/index.ts:48-102`

**Significance:** Need to add metadata mapping in both functions to persist and retrieve the field from database.

---

### Finding 4: POST /session endpoint uses Session.create.schema

**Evidence:** POST /session at line 185-209 in session route validates with `Session.create.schema.optional()`. Session.create schema (line 192-199) allows parentID, title, and permission. No metadata field.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/server/routes/session.ts:185-209` and `index.ts:192-199`

**Significance:** Need to add metadata to Session.create.schema to accept it during session creation.

---

### Finding 5: PATCH /session/:id has custom validation schema

**Evidence:** PATCH endpoint (lines 240-289) uses inline validation with only title and time.archived fields. Does not use a shared schema.

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/server/routes/session.ts:264-273`

**Significance:** Need to add metadata field to the PATCH validator and handle it in the update logic.

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
