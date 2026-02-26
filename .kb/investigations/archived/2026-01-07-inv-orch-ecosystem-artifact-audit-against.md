<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Orch ecosystem artifacts show partial alignment with skillc's 9 design principles; SPAWN_CONTEXT.md is the primary gap, missing self-describing header pattern that exists in compiled SKILL.md files.

**Evidence:** Tested actual files - SKILL.md has full AUTO-GENERATED/checksum/source header; SPAWN_CONTEXT.md starts with `TASK:` with no header; skillc doctor exists, orch doctor doesn't; no `orch spawn --json` flag.

**Knowledge:** Two artifact categories exist: machine-generated (SPAWN_CONTEXT, SKILL.md) and human-filled templates (SYNTHESIS, investigation). Skillc principles apply primarily to machine-generated category.

**Next:** Add skillc-style self-describing header to SPAWN_CONTEXT.md in `pkg/spawn/context.go` (~10-20 lines of code).

**Promote to Decision:** recommend-no (tactical improvement, not architectural decision)

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

# Investigation: Orch Ecosystem Artifact Audit Against Skillc Design Principles

**Question:** How well do orch ecosystem artifacts (SPAWN_CONTEXT.md, SYNTHESIS.md, FAILURE_REPORT.md, SESSION_HANDOFF.md, .kb/ artifacts, beads issues) align with skillc's 9 AI-native design principles?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Agent og-inv-orch-ecosystem-artifact-07jan-50c4
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Starting approach - identifying artifacts and principles to audit

**Evidence:** 
- Skillc has 9 design principles defined in DESIGN_PRINCIPLES.md
- Orch ecosystem owns 4 templates: SPAWN_CONTEXT.md, SYNTHESIS.md, FAILURE_REPORT.md, SESSION_HANDOFF.md
- kb-cli owns: investigation, decision, guide, research templates
- Beads provides structured issue tracking

**Skillc's 9 Principles:**
1. Self-Describing Artifacts
2. Surfacing Over Browsing
3. Integrity Verification
4. Dependency Resolution
5. Composability
6. Progressive Disclosure
7. Template Expansion
8. Self-Diagnosis
9. Machine-Readable Output

**Source:** /Users/dylanconlin/Documents/personal/skillc/DESIGN_PRINCIPLES.md

**Significance:** This sets the framework for the audit. Need to examine each artifact type against each principle.

---

### Finding 2: SPAWN_CONTEXT.md - Strong on Surfacing, Missing Self-Description

**Evidence:** 
Examined `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go` and actual SPAWN_CONTEXT.md outputs.

**Strengths (Principles Met):**
- ✅ **Surfacing Over Browsing (2):** Excellent. Consolidates kb context, skill guidance, authority levels, and deliverables into one document
- ✅ **Composability (5):** Good. Separate template files can be combined via Go template
- ✅ **Progressive Disclosure (6):** Partial. Has TLDR-style task at top, but full content follows immediately

**Gaps (Principles Violated):**
- ❌ **Self-Describing Artifacts (1):** NO HEADER. No "AUTO-GENERATED", no "DO NOT EDIT", no checksum, no source path
- ❌ **Integrity Verification (3):** No checksum. Agents could hand-edit and break contract
- ❌ **Machine-Readable Output (9):** No JSON equivalent. `orch spawn --json` doesn't exist

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:28-200`
- My own SPAWN_CONTEXT.md at `.orch/workspace/og-inv-orch-ecosystem-artifact-07jan-50c4/SPAWN_CONTEXT.md`

**Significance:** SPAWN_CONTEXT.md is the most critical spawn-time artifact, but lacks the header pattern that would tell agents "don't edit this" and enable checksum verification. This is a meaningful gap.

---

### Finding 3: SYNTHESIS.md - Good Progressive Disclosure, Agent-Filled

**Evidence:**
Examined template at `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md`

**Strengths:**
- ✅ **Progressive Disclosure (6):** Excellent. TLDR first, then Delta/Evidence/Knowledge/Next structure
- ✅ **Surfacing Over Browsing (2):** Good. All session info in one place
- ✅ **Composability (5):** Used by `orch complete` and `orch review` workflows

**Gaps:**
- ⚠️ **Self-Describing Artifacts (1):** N/A - this is agent-filled, not generated
- ⚠️ **Integrity Verification (3):** N/A - content varies by session
- ⚠️ **Template Expansion (7):** Placeholders exist ({workspace-name}, {beads-id}) but agents fill them manually

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md`

**Significance:** SYNTHESIS.md is human/agent-filled content, so self-description matters less. The template provides good structure but doesn't auto-expand placeholders.

---

### Finding 4: Investigation Template (kb-cli) - Strong Overall Alignment

**Evidence:**
Examined `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go:13-235`

**Strengths:**
- ✅ **Progressive Disclosure (6):** D.E.K.N. summary at top for 30-second handoff
- ✅ **Surfacing Over Browsing (2):** Structured sections guide finding capture
- ✅ **Composability (5):** Works with kb context, beads, and orch complete
- ✅ **Template Expansion (7):** Uses `{{title}}`, `{{date}}` placeholders that `kb create` expands

**Gaps:**
- ❌ **Self-Describing Artifacts (1):** No header saying "generated by kb create" or "template-based"
- ⚠️ **Integrity Verification (3):** N/A - investigations are human-authored
- ❌ **Self-Diagnosis (8):** No `kb doctor` equivalent for investigation health

**Source:** `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go:13-235`

**Significance:** Investigation template is well-designed for progressive disclosure but missing the self-describing header pattern.

---

### Finding 5: Compiled SKILL.md - Gold Standard for Principles

**Evidence:**
Read compiled SKILL.md files deployed to ~/.claude/skills/

```
head -20 ~/.claude/skills/worker/investigation/SKILL.md

---
name: investigation
skill-type: procedure
description: Record what you tried, what you observed...
---

<!-- AUTO-GENERATED by skillc -->
<!-- Checksum: a1ea3997ce46 -->
<!-- Source: /Users/dylanconlin/orch-knowledge/skills/src/worker/investigation/.skillc -->
<!-- Deployed to: /Users/dylanconlin/.claude/skills/worker/investigation/SKILL.md -->
<!-- To modify: edit files in .../investigation/.skillc, then run: skillc deploy -->
<!-- Last compiled: 2026-01-07 14:44:12 -->
```

**All 9 Principles Satisfied:**
- ✅ **Self-Describing (1):** Full header with all 5 questions answered
- ✅ **Surfacing (2):** Multiple sources compiled into one
- ✅ **Integrity (3):** Checksum present
- ✅ **Dependency Resolution (4):** Dependencies resolved during compilation
- ✅ **Composability (5):** Produces standalone artifact
- ✅ **Progressive Disclosure (6):** Summary section at top
- ✅ **Template Expansion (7):** SKILL-TEMPLATE markers expanded
- ✅ **Self-Diagnosis (8):** `skillc doctor` command exists
- ✅ **Machine-Readable (9):** `skillc build --json` exists

**Source:** `~/.claude/skills/worker/investigation/SKILL.md:1-15`

**Significance:** Compiled SKILL.md files are the gold standard. Other artifacts should aspire to this pattern.

---

### Finding 6: SESSION_HANDOFF.md and FAILURE_REPORT.md - Templates, Not Generated

**Evidence:**
Examined both templates in `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/`

Both are:
- Agent-filled templates with placeholder guidance
- Good progressive disclosure structure
- Used for human-authored synthesis content

**Principles Assessment:**
- ✅ **Progressive Disclosure (6):** Good section structure
- ✅ **Surfacing (2):** Consolidates session info
- N/A **Self-Describing (1):** Templates, not generated output
- N/A **Integrity (3):** Human-authored content

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SESSION_HANDOFF.md`
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/FAILURE_REPORT.md`

**Significance:** These are guidance templates for human authoring, not generated artifacts. Different expectations apply.

---

### Finding 7: Beads Issues - Different Domain (Structured Data)

**Evidence:**
Beads is SQLite-based structured data, not markdown artifacts.

**Applicable Principles:**
- ✅ **Machine-Readable (9):** `bd show <id> --json` exists
- ✅ **Dependency Resolution (4):** Parent/child relationships
- ✅ **Composability (5):** Works with orch, kb, agentlog

**Not Applicable:**
- N/A Self-Describing Artifacts (database rows don't need headers)
- N/A Integrity Verification (database handles this)
- N/A Template Expansion (not template-based)

**Source:** `bd show --help` output inspection

**Significance:** Beads is a different artifact type. Skillc's markdown principles don't fully apply to database records.

---

## Synthesis

**Key Insights:**

1. **Two Artifact Categories with Different Expectations** - The ecosystem has (a) machine-generated artifacts (SPAWN_CONTEXT.md, compiled SKILL.md) and (b) human-filled templates (SYNTHESIS.md, SESSION_HANDOFF.md, investigation files). Skillc principles primarily apply to category (a).

2. **SPAWN_CONTEXT.md is the Primary Gap** - This is the most machine-generated, most critical artifact and it lacks the self-describing header pattern entirely. No "AUTO-GENERATED", no checksum, no "DO NOT EDIT" warning. This is actionable.

3. **Compiled SKILL.md is the Gold Standard** - These files implement all 9 principles. The pattern exists in the ecosystem; it just hasn't been applied to orch-go's generated artifacts.

4. **Template Expansion Gap is Real but Already Known** - The prior investigation (2025-12-22-inv-re-investigate-skillc-vs-orch.md) documented that skillc's template expansion differs from orch-knowledge's SKILL-TEMPLATE pattern. This isn't new but confirms the pattern.

**Answer to Investigation Question:**

Orch ecosystem artifacts show **partial alignment** with skillc's 9 design principles:

| Principle | SPAWN_CONTEXT | SYNTHESIS | Investigation | SKILL.md |
|-----------|---------------|-----------|---------------|----------|
| 1. Self-Describing | ❌ Missing | N/A | ❌ Missing | ✅ |
| 2. Surfacing | ✅ | ✅ | ✅ | ✅ |
| 3. Integrity | ❌ No checksum | N/A | N/A | ✅ |
| 4. Dependency | ✅ | N/A | N/A | ✅ |
| 5. Composability | ✅ | ✅ | ✅ | ✅ |
| 6. Progressive Disclosure | ✅ Partial | ✅ | ✅ | ✅ |
| 7. Template Expansion | ⚠️ Go templates | N/A | ✅ kb create | ✅ |
| 8. Self-Diagnosis | ❌ No orch doctor | N/A | ❌ No kb doctor | ✅ |
| 9. Machine-Readable | ❌ No --json | N/A | N/A | ✅ |

**Key finding:** SPAWN_CONTEXT.md violates 4 principles (1, 3, 8, 9) that would be straightforward to address by adding a skillc-style header.

---

## Structured Uncertainty

**What's tested:**

- ✅ SKILL.md has full skillc header (verified: `head -20 ~/.claude/skills/worker/investigation/SKILL.md`)
- ✅ SPAWN_CONTEXT.md lacks any header (verified: `head -20` of my own spawn context)
- ✅ `skillc doctor` command exists (verified: ran in orch-go directory)
- ✅ `orch spawn --json` does NOT exist (verified: `orch spawn --help | grep json`)
- ✅ Investigation template uses `{{title}}`, `{{date}}` expansion (verified: read kb-cli source)

**What's untested:**

- ⚠️ Whether adding headers to SPAWN_CONTEXT.md would cause agent confusion (not tested)
- ⚠️ Whether checksum verification would catch real tampering issues (no evidence of tampering problem)
- ⚠️ Implementation effort for `orch spawn --json` (not scoped)

**What would change this:**

- Finding would be wrong if agents currently edit SPAWN_CONTEXT.md and this causes problems (no evidence)
- Finding would be wrong if there's a reason SPAWN_CONTEXT.md intentionally lacks headers (not documented)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Add Self-Describing Header to SPAWN_CONTEXT.md** - Add a skillc-style header to SPAWN_CONTEXT.md that answers the 5 essential questions.

**Why this approach:**
- SPAWN_CONTEXT.md is machine-generated by orch spawn
- The pattern already exists in compiled SKILL.md files
- Minimal implementation effort (add header generation to context.go)
- Enables future checksum verification if needed

**Trade-offs accepted:**
- Adding lines to already-large SPAWN_CONTEXT.md
- Won't add checksum verification in first pass (header only)

**Implementation sequence:**
1. Add `generateSpawnContextHeader()` to `pkg/spawn/context.go` following skillc pattern
2. Include: AUTO-GENERATED by orch spawn, Source: spawn config, DO NOT EDIT, timestamp
3. Optionally add checksum for integrity verification later

### Alternative Approaches Considered

**Option B: Full skillc integration for SPAWN_CONTEXT.md**
- **Pros:** Would get all skillc features (build, check, watch)
- **Cons:** Heavyweight; SPAWN_CONTEXT is dynamic per-spawn, not static skill
- **When to use instead:** If we wanted version-controlled spawn templates

**Option C: Do nothing**
- **Pros:** No implementation effort
- **Cons:** SPAWN_CONTEXT remains the only generated artifact without self-description
- **When to use instead:** If there's no evidence of problems from missing header

**Rationale for recommendation:** Adding header is low-effort, aligns with existing patterns, and makes SPAWN_CONTEXT consistent with SKILL.md.

---

### Implementation Details

**What to implement first:**
- Add header generation to `pkg/spawn/context.go` (~10-20 lines)
- Include in `SpawnContextTemplate` constant

**Things to watch out for:**
- ⚠️ Don't break existing SPAWN_CONTEXT.md parsing (orch complete reads it)
- ⚠️ Header should be HTML comments to not affect markdown rendering
- ⚠️ Keep header concise - SPAWN_CONTEXT is already long

**Areas needing further investigation:**
- Whether to add `orch spawn --json` flag (useful but separate scope)
- Whether to add `orch doctor` command for ecosystem health
- Whether kb-cli investigation template should also get headers

**Success criteria:**
- ✅ SPAWN_CONTEXT.md starts with AUTO-GENERATED header
- ✅ Header includes: generator (orch spawn), timestamp, DO NOT EDIT warning
- ✅ Existing orch complete continues to work

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/skillc/DESIGN_PRINCIPLES.md` - The 9 skillc design principles
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go` - SPAWN_CONTEXT.md generation
- `/Users/dylanconlin/Documents/personal/orch-go/.orch/templates/SYNTHESIS.md` - Template structure
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go` - Investigation template
- `~/.claude/skills/worker/investigation/SKILL.md` - Gold standard compiled output
- `~/.orch/ECOSYSTEM.md` - Ecosystem documentation

**Commands Run:**
```bash
# Verify SKILL.md has full header
head -20 ~/.claude/skills/worker/investigation/SKILL.md

# Verify SPAWN_CONTEXT.md lacks header
head -20 .orch/workspace/og-inv-orch-ecosystem-artifact-07jan-50c4/SPAWN_CONTEXT.md

# Test skillc doctor command
skillc doctor

# Check if orch spawn has JSON flag
orch spawn --help | grep -i json
```

**External Documentation:**
- skillc DESIGN_PRINCIPLES.md - Reference for the 9 AI-native principles

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-22-inv-re-investigate-skillc-vs-orch.md` - Prior skillc comparison
- **Investigation:** `.kb/investigations/2025-12-22-inv-deep-dive-template-system-fragmentation.md` - Template ownership

---

## Investigation History

**2026-01-07:** Investigation started
- Initial question: How well do orch ecosystem artifacts align with skillc's 9 design principles?
- Context: Spawned by orchestrator for ecosystem health audit

**2026-01-07:** Findings complete
- Identified SPAWN_CONTEXT.md as primary gap (missing self-describing header)
- Confirmed SKILL.md as gold standard with all 9 principles satisfied
- Categorized artifacts into machine-generated vs human-filled templates

**2026-01-07:** Investigation completed
- Status: Complete
- Key outcome: SPAWN_CONTEXT.md should add skillc-style header for consistency

---

## Self-Review

- [x] Real test performed (ran head commands, skillc doctor, orch spawn --help)
- [x] Conclusion from evidence (header absence verified in actual files)
- [x] Question answered (partial alignment, SPAWN_CONTEXT.md is main gap)
- [x] File complete (all sections filled)
- [x] D.E.K.N. filled (needs to be completed)
- [x] NOT DONE claims verified (searched actual files)

**Self-Review Status:** PASSED
