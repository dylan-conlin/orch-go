## Summary (D.E.K.N.)

**Delta:** Supersession chain and current state reference now exist for model stack decisions.

**Evidence:** Jan 9 decision marked superseded, current-model-stack.md created, GLM investigation corrected (4 files modified).

**Knowledge:** Workers cite outdated decisions because no "current state" reference exists and old decisions aren't marked superseded. The fix is structural: create authoritative reference + mark historical decisions.

**Next:** Close - knowledge hygiene complete.

**Promote to Decision:** recommend-no (tactical fix, establishes pattern but not architectural)

---

# Investigation: Fix Model Stack Knowledge Hygiene

**Question:** How do we prevent workers from citing outdated model stack decisions?

**Started:** 2026-01-22
**Updated:** 2026-01-22
**Owner:** Worker agent (spawned by orchestrator)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Jan 9 Decision Not Marked as Superseded

**Evidence:** The decision `2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md` had `Status: Accepted` despite being superseded by Jan 18 decision.

**Source:** `.kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md:4`

**Significance:** Workers searching for model stack decisions find Jan 9 first (alphabetically or by search) and treat it as current policy because nothing indicates it was superseded.

---

### Finding 2: No Authoritative "Current State" Reference Existed

**Evidence:** The existing model `.kb/models/model-access-spawn-paths.md` is comprehensive but architectural, not a quick "what's current now" reference. No document clearly stated: "This is the current model stack, cite this document."

**Source:** Grep for "current.*model.*stack" returned no quick-reference results.

**Significance:** Workers must trace through decision history to determine current policy. This is error-prone and led to the GLM investigation citing the wrong decision.

---

### Finding 3: GLM Investigation Cited Superseded Decision

**Evidence:** Lines 172, 186, and 327 of `2026-01-22-inv-research-glm-ai-orchestration-context.md` referenced Jan 9 decision as "current model strategy" when Jan 18 decision superseded it.

**Source:** `.kb/investigations/2026-01-22-inv-research-glm-ai-orchestration-context.md:172`

**Significance:** This is the symptom that triggered this hygiene task. The root cause was structural (no supersession markers, no current state reference).

---

### Finding 4: CLAUDE.md Already Reflects Current State

**Evidence:** Line 313 of CLAUDE.md states "Model default: Opus (Max subscription), not Gemini (pay-per-token)" which aligns with Jan 18 decision.

**Source:** `CLAUDE.md:313`

**Significance:** The project documentation is correct; the knowledge base was out of sync.

---

## Synthesis

**Key Insights:**

1. **Supersession must be explicit** - Just creating a new decision doesn't communicate that old decisions are outdated. Workers search and find the old one first.

2. **"Current state" references are authoritative** - A document that explicitly says "this is current, cite this" prevents workers from tracing decision chains.

3. **Pattern for future decisions** - When creating new decisions that supersede old ones: (a) mark old as superseded, (b) update current state reference, (c) update any artifacts that cite the old decision.

**Answer to Investigation Question:**

To prevent workers from citing outdated decisions: (1) mark old decisions as superseded with `Status: Superseded` and `Superseded-By:` field, (2) create authoritative "current state" reference documents that workers should cite instead of historical decisions, (3) update any existing artifacts that cite superseded decisions.

---

## Structured Uncertainty

**What's tested:**

- ✅ Jan 9 decision marked superseded (verified: file edited with note at top)
- ✅ Current model stack reference created (verified: `.kb/models/current-model-stack.md` exists)
- ✅ GLM investigation corrected (verified: 4 edits to update references)
- ✅ CLAUDE.md already correct (verified: line 313 matches Jan 18 decision)

**What's untested:**

- ⚠️ Whether workers will actually cite current-model-stack.md (not validated yet)
- ⚠️ Whether kb search will return current-model-stack.md for "model stack" queries

**What would change this:**

- If kb search doesn't surface current-model-stack.md, may need to add keywords
- If workers continue citing old decisions, may need to add more prominent supersession notices

---

## Implementation Recommendations

### Recommended Approach ⭐

**Supersession pattern for all future decisions** - When creating a new decision that supersedes an old one, always: mark old as superseded, create/update current state reference, fix artifacts citing old decision.

**Why this approach:**
- Prevents knowledge rot structurally
- Creates single source of truth for workers to cite
- Makes decision evolution visible

**Implementation sequence:**
1. Mark old decision superseded (Status + Superseded-By fields)
2. Add note at top of old decision explaining it's outdated
3. Create/update current state reference
4. Fix any artifacts that cite the old decision

---

## References

**Files Modified:**
- `.kb/decisions/2026-01-09-abandon-claude-max-oauth-use-gemini-primary.md` - Marked superseded
- `.kb/models/current-model-stack.md` - Created (new)
- `.kb/investigations/2026-01-22-inv-research-glm-ai-orchestration-context.md` - Fixed references

**Files Examined:**
- `.kb/decisions/2026-01-18-max-subscription-primary-spawn-path.md` - Current policy
- `.kb/models/model-access-spawn-paths.md` - Existing architectural model
- `CLAUDE.md` - Verified already correct

---

## Investigation History

**2026-01-22:** Investigation started
- Initial question: How to prevent workers from citing outdated model stack decisions
- Context: GLM investigation cited Jan 9 decision when Jan 18 superseded it

**2026-01-22:** Investigation completed
- Status: Complete
- Key outcome: Created supersession pattern - mark old decisions, create current state reference, fix citing artifacts
