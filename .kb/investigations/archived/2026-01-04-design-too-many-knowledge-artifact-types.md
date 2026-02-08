## Summary (D.E.K.N.)

**Delta:** The taxonomy has 4 distinct artifact types by purpose (Investigation, Decision, Guide, Quick), but conflates two separate dimensions: "questions vs answers" and "point-in-time vs evolved."

**Evidence:** Analyzed 9 templates, 460+ investigations, 7 guides. KNOWLEDGE.md template is 327 lines but has 0 usage. RESEARCH.md duplicates INVESTIGATION.md. Guides (agent-lifecycle.md, daemon.md) are synthesized from 20+ investigations each.

**Knowledge:** The taxonomy gap is not "too many types" but "missing the synthesis output type" - when `kb reflect` detects 10+ investigations on a topic, the output should be a Guide (evolved, authoritative), not another Investigation (point-in-time, exploratory).

**Next:** Deprecate KNOWLEDGE.md and RESEARCH.md templates. Clarify that Guides are the synthesis output type. Update `kb reflect` to propose Guide creation.

---

# Investigation: Too Many Knowledge Artifact Types

**Question:** Do we have artifact type sprawl? What is the relationship between investigation, research, knowledge, and guide? What should `kb reflect` synthesis produce?

**Started:** 2026-01-04
**Updated:** 2026-01-04
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Current Taxonomy Has 9 Templates, But Only 4 Serve Distinct Purposes

**Evidence:** Templates in `~/.kb/templates/`:

| Template | Lines | Purpose | Usage |
|----------|-------|---------|-------|
| INVESTIGATION.md | 221 | Exploratory question-answering | 460+ files in orch-go |
| DECISION.md | 74 | Record architectural choice | 8 files in orch-go, 6 in global |
| RESEARCH.md | 94 | External source evaluation | 0 files found |
| KNOWLEDGE.md | 327 | Pattern/principle documentation | 0 files found |
| POST_MORTEM.md | 222 | Failure analysis | 1 file in orch-go |
| SYNTHESIS.md (global) | 21 | DEPRECATED marker only | N/A |
| SPAWN_PROMPT.md | ??? | Part of spawn system | N/A |
| WORKSPACE.md | ??? | Part of workspace system | N/A |

Additionally, `orch-go/.orch/templates/SYNTHESIS.md` (153 lines) is the active session synthesis template.

And `orch-go/.kb/guides/` contains 7 files (average 200 lines each) with NO template - manually created.

**Source:** `ls ~/.kb/templates/`, `find .kb -name "*.md" | wc -l`, file inspection

**Significance:** RESEARCH.md and KNOWLEDGE.md are unused. The distinction they attempt (external vs internal, pattern vs finding) isn't actionable. Meanwhile, Guides have emerged organically without a template.

---

### Finding 2: The Investigation/Research/Knowledge Distinction Is Conflating Two Dimensions

**Evidence:** Comparing template purposes:

- **INVESTIGATION.md**: "How does X work?" (internal, exploratory, point-in-time)
- **RESEARCH.md**: "Which option is best?" (external, comparative, point-in-time)
- **KNOWLEDGE.md**: "Here's a reusable pattern" (synthesized, authoritative, evolved)

The confusion comes from conflating:
1. **Source dimension**: Internal (codebase) vs External (docs, web)
2. **Lifecycle dimension**: Point-in-time (investigation) vs Evolved (pattern)

RESEARCH.md is just an INVESTIGATION.md with external sources. The template difference is artificial - agents use INVESTIGATION.md with web sources already.

KNOWLEDGE.md is trying to be what Guides already are.

**Source:** Template comparison, `~/.kb/principles.md` (principle discovery mechanism lines 212-219)

**Significance:** The source dimension (internal/external) doesn't need separate templates. The lifecycle dimension (point-in-time/evolved) does - but that's already Investigation vs Guide.

---

### Finding 3: Guides Are The Missing "Synthesis Output" Type

**Evidence:** The 7 guides in orch-go:

| Guide | Lines | Source | Purpose |
|-------|-------|--------|---------|
| agent-lifecycle.md | 138 | 20+ investigations | Single authoritative reference |
| completion-gates.md | 430 | 10+ investigations | Single authoritative reference |
| daemon.md | 208 | 15+ investigations | Single authoritative reference |
| skill-system.md | 211 | Many investigations | Single authoritative reference |
| spawn.md | ~200 | Many investigations | Single authoritative reference |
| beads-integration.md | ~200 | Many investigations | Single authoritative reference |
| status-dashboard.md | ~200 | Many investigations | Single authoritative reference |

Each guide header says: "Single authoritative reference for X. Read this before debugging X issues."

Each guide was created AFTER multiple investigations accumulated on a topic. Example from agent-lifecycle.md:
> "Created after spending 1 hour debugging a problem that was already documented in kn. Synthesized from 20+ investigations about sessions/completion/lifecycle."

**Source:** `.kb/guides/*.md` inspection

**Significance:** Guides ARE the synthesis output. When `kb reflect` detects 10+ investigations on a topic, the recommended action is "Create a Guide." The feedback loop isn't broken - we just haven't documented this.

---

### Finding 4: Quick Entries Serve A Different Purpose (Operational Memory)

**Evidence:** From `.kb/quick/entries.jsonl`:
- Type: constraint, decision, question, attempt
- Format: JSON lines, 1-2 sentences each
- Lifecycle: Quick capture during work, queryable via `kb context`

This is NOT a knowledge artifact type competing with Investigation/Decision/Guide. It's operational memory - quick decisions made during implementation that might later graduate to proper artifacts.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/quick/entries.jsonl`

**Significance:** Quick entries are in a different category entirely. They're not confused with investigations - they serve as the input funnel, not the output.

---

### Finding 5: The Prior Decision Identified This But Didn't Fully Solve It

**Evidence:** From `2025-12-21-minimal-artifact-taxonomy.md`:
> "Adopt a minimal artifact set of 5 essential + 3 supplementary types"

The 5 essential:
- SPAWN_CONTEXT.md (ephemeral)
- SYNTHESIS.md (ephemeral)
- Investigation (persistent)
- Decision (persistent)
- Beads Comments (operational)

The 3 supplementary:
- SESSION_HANDOFF.md
- FAILURE_REPORT.md
- kn entries

**Missing from this list:** Guides.

The decision focused on session lifecycle artifacts (spawn → work → complete) but didn't address knowledge lifecycle artifacts (investigate → synthesize → guide).

**Source:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md`

**Significance:** The prior work is correct but incomplete. The knowledge lifecycle needs the same clarity as the session lifecycle.

---

## Synthesis

**Key Insights:**

1. **Two artifacts are unused and should be deprecated** - RESEARCH.md and KNOWLEDGE.md have zero usage. Their intended purposes are already served by Investigation (with external sources) and Guide (evolved patterns).

2. **Guides are the synthesis output type** - When investigations cluster on a topic, the resolution is a Guide. This is already happening organically. `kb reflect` should propose Guide creation, not more investigations.

3. **The taxonomy has 4 distinct purposes** - Investigation (exploratory question), Decision (architectural choice), Guide (evolved authoritative reference), Quick (operational memory). Everything else is either ephemeral (SYNTHESIS.md, SPAWN_CONTEXT.md) or deprecated.

4. **The "knowledge promotion path" is clear** - Investigation → (if accepted recommendation) → Decision. Multiple Investigations on same topic → Guide. Quick entry → (if architecturally significant) → Decision. Quick constraint → (if universal) → principles.md.

**Answer to Investigation Question:**

**Q: What is the difference between investigation, research, and knowledge?**

A: Research and Knowledge are unused template names that should be deprecated. Investigation is the only exploratory artifact type. For external source research, use Investigation with web sources.

**Q: When kb reflect detects 10+ investigations on a topic, what should it produce?**

A: A Guide. Guides are synthesized from investigation clusters into single authoritative references. The `kb reflect` synthesis issue should recommend: "Create guide: {topic}" not "Create synthesis investigation."

**Q: Do we have artifact type sprawl? Should some be consolidated?**

A: Yes, but consolidation is simple: deprecate RESEARCH.md and KNOWLEDGE.md. The remaining types serve distinct purposes.

**Q: What's the lifecycle: investigation → ??? → guide?**

A: Investigation (exploratory) → [if recommendation accepted] → Decision (choice record). Multiple Investigations (cluster) → Guide (evolved reference). These are parallel paths, not a single sequence.

---

## Structured Uncertainty

**What's tested:**

- ✅ RESEARCH.md has 0 files using it (verified: `find .kb -name "*research*.md" -exec head -1 {} \;`)
- ✅ KNOWLEDGE.md has 0 files using it (verified: `find .kb -name "*knowledge*.md"`)
- ✅ Guides exist and serve synthesis purpose (verified: read 7 guide files)
- ✅ Quick entries are JSON lines format (verified: `head .kb/quick/entries.jsonl`)

**What's untested:**

- ⚠️ Whether deprecating RESEARCH.md breaks any agent expectations (agents might be trained on it)
- ⚠️ Whether `kb reflect` can actually create guides (need to check kb-cli implementation)
- ⚠️ Whether all projects follow this pattern (only checked orch-go and global)

**What would change this:**

- Finding that RESEARCH.md is used in other projects would suggest keeping it
- Finding that Guide creation is automated elsewhere would change the `kb reflect` recommendation
- Finding that agents actively choose RESEARCH.md over INVESTIGATION.md would suggest the distinction matters

---

## Implementation Recommendations

**Purpose:** Clarify the knowledge artifact taxonomy and update `kb reflect` to produce the right output type.

### Recommended Approach ⭐

**Deprecate unused templates, document Guide as synthesis output** - Clean up RESEARCH.md and KNOWLEDGE.md templates. Update `kb reflect` to propose Guide creation when investigation clusters are detected.

**Why this approach:**
- Zero-usage templates are noise (Finding 1)
- Guides are already the synthesis output, just undocumented (Finding 3)
- Aligns with Evolve by Distinction principle - the distinction is Investigation (point-in-time) vs Guide (evolved), not Investigation vs Research vs Knowledge

**Trade-offs accepted:**
- Agents trained on RESEARCH.md may be confused (but they don't use it anyway)
- Guide template needs to be created (small effort)

**Implementation sequence:**
1. Add deprecation notice to RESEARCH.md and KNOWLEDGE.md (like SYNTHESIS.md has)
2. Create GUIDE.md template in `~/.kb/templates/`
3. Update `kb reflect` to output "Create guide: {topic}" for investigation clusters
4. Update global CLAUDE.md knowledge placement table to show Guide

### Alternative Approaches Considered

**Option B: Keep all templates, better document when to use each**
- **Pros:** No breaking changes
- **Cons:** Unused templates remain confusion; no one reads documentation to choose templates
- **When to use instead:** If we discover RESEARCH.md or KNOWLEDGE.md are actually used elsewhere

**Option C: Collapse everything into a single "KB Article" type**
- **Pros:** Maximum simplicity
- **Cons:** Loses meaningful distinctions (Investigation IS different from Decision in purpose and lifecycle)
- **When to use instead:** If agent confusion about types becomes severe

**Rationale for recommendation:** The unused templates should be removed (simplicity). Guides should be formalized (they already exist and work). The core distinction (exploratory vs authoritative) is meaningful and should be preserved.

---

### Implementation Details

**What to implement first:**
- GUIDE.md template creation (enables the rest)
- Deprecation notices in unused templates (quick, reversible)

**Things to watch out for:**
- ⚠️ Skills that reference RESEARCH.md in their guidance (search skills for "research" references)
- ⚠️ `kb create` command options (does it support creating guides?)
- ⚠️ `kb reflect` current behavior (need to understand what it currently outputs)

**Areas needing further investigation:**
- How does kb-cli's `kb reflect` command actually work?
- Are there other projects using RESEARCH.md or KNOWLEDGE.md?

**Success criteria:**
- ✅ RESEARCH.md and KNOWLEDGE.md have deprecation notices
- ✅ GUIDE.md template exists and is documented
- ✅ `kb reflect` proposes guide creation for investigation clusters
- ✅ Global CLAUDE.md knowledge placement table includes Guide

---

## References

**Files Examined:**
- `~/.kb/templates/*.md` - All templates
- `.kb/guides/*.md` - All guides in orch-go
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Prior decision on taxonomy
- `.kb/investigations/2025-12-21-inv-knowledge-promotion-paths.md` - Prior investigation
- `~/.kb/principles.md` - Principle discovery mechanism
- `.kb/quick/entries.jsonl` - Quick entries format

**Commands Run:**
```bash
# Count investigations
find .kb -name "*.md" | wc -l

# Check for research/knowledge files
find .kb -name "*research*.md"
find .kb -name "*knowledge*.md"

# Line counts
wc -l ~/.kb/templates/*.md
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Prior taxonomy work
- **Investigation:** `.kb/investigations/2025-12-21-inv-knowledge-promotion-paths.md` - Promotion paths

---

## Investigation History

**2026-01-04 12:20:** Investigation started
- Initial question: Too many knowledge artifact types with unclear relationships
- Context: kb reflect feedback loop broken because synthesis output type unclear

**2026-01-04 12:45:** Key finding
- RESEARCH.md and KNOWLEDGE.md have zero usage
- Guides are the missing synthesis output type

**2026-01-04 13:00:** Investigation completed
- Status: Complete
- Key outcome: Taxonomy has 4 types (Investigation, Decision, Guide, Quick). Deprecate RESEARCH.md and KNOWLEDGE.md. `kb reflect` synthesis → Guide.
