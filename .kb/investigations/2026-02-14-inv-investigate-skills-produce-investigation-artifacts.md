<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Spawn template defaults ALL worker skills to creating investigation files, causing 312:1 investigation-to-synthesis ratio.

**Evidence:** pkg/spawn/context.go:203-226 includes "SET UP investigation file" for all spawns; only 5-7 skills should produce investigations as primary deliverable; workspace analysis shows feature-impl creating investigations for implementation tasks.

**Knowledge:** Investigation artifacts should be reserved for exploratory/understanding work (investigation, architect, research, codebase-audit, reliability-testing skills), not default for all spawns.

**Next:** Remove investigation deliverable from default spawn template; make it skill-specific via skill manifest flag (ProducesInvestigation: bool).

**Authority:** architectural - Crosses spawn architecture, skill system, and verification gates requiring coordination across subsystems.

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

# Investigation: Investigate Skills Produce Investigation Artifacts

**Question:** Which skills produce investigation artifacts, and should they?

**Started:** 2026-02-14
**Updated:** 2026-02-14
**Owner:** orch-go-7lp
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

### Finding 1: Spawn context template defaults ALL spawns to creating investigation files

**Evidence:** The SPAWN_CONTEXT template in pkg/spawn/context.go (lines 203-226) includes "**SET UP investigation file:** Run `kb create investigation {{.InvestigationSlug}}`" as a default deliverable for ALL spawns (except orchestrator/meta-orchestrator types). This is injected regardless of which skill is being spawned.

**Source:** /Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:203-226

**Significance:** This explains the 312:1 ratio of investigations to synthesized work. Every agent spawn (feature-impl, systematic-debugging, etc.) is being told to create an investigation file by default, not just skills that actually need exploratory/understanding artifacts.

---

### Finding 2: Skills with explicit investigation artifact production

**Evidence:** Grepped all SKILL.md files for investigation artifact mentions. Found these skills explicitly document producing investigation artifacts:

**Skills that SHOULD produce investigations (exploratory/understanding work):**
- `investigation` - Purpose: "Answer a question by testing, not by reasoning" - delivers `.kb/investigations/{date}-inv-*.md`
- `architect` - Purpose: "Strategic design skill for deciding what should exist" - delivers investigation in `.kb/investigations/` with `design-` prefix
- `research` - Purpose: "Web-based research producing structured recommendations" - delivers investigation with `research-` prefix
- `codebase-audit` - Purpose: "Systematic codebase audit" - delivers investigation at `.kb/investigations/YYYY-MM-DD-audit-{dimension}.md`
- `reliability-testing` - Purpose: "Hardening system for production" - delivers investigation file

**Skills that have investigation as ONE PHASE (not primary artifact):**
- `feature-impl` - Has investigation phase as OPTIONAL configurablephase (via `--phases` flag)
- `design-session` - Can produce investigation artifact based on clarity level

**Source:** grep -r "investigation" ~/.claude/skills/*/SKILL.md and manual SKILL.md review

**Significance:** Only 5-7 skills legitimately SHOULD produce investigation artifacts as their primary deliverable, yet the spawn template tells ALL agents to create investigations.

---

### Finding 3: feature-impl investigation phase is optional, not default

**Evidence:** feature-impl deliverables section shows: "| investigation phase | Investigation file |" - this is only required when investigation is included in the --phases configuration. The skill is configurable via `--phases "investigation,design,implementation"` etc.

**Source:** ~/.claude/skills/worker/feature-impl/SKILL.md deliverables table

**Significance:** feature-impl shouldn't always produce investigations - only when explicitly configured with investigation phase. Yet spawn template tells it to create investigation file regardless of phase configuration.

---

### Finding 4: Workspace analysis confirms non-investigative skills creating investigations

**Evidence:** Examined recent workspaces:
- `og-feat-implement-kb-reflect-14feb-8a59` → feature-impl skill → created investigation file
- `og-feat-fix-claude-md-14feb-e330` → feature-impl skill (bug fix) → created investigation file  
- Both are implementation tasks, not exploratory/understanding work
- Recent investigation files include many `inv-fix-*` and `inv-implement-*` prefixes indicating bug fixes and feature implementations

**Source:** .orch/workspace/* directories, SPAWN_CONTEXT.md files, .kb/investigations/*.md filenames

**Significance:** Confirms the spawn template default is causing investigation file proliferation. Skills like feature-impl and systematic-debugging are creating investigation files even for straightforward implementation/bug-fix work that doesn't require exploratory artifacts.

---

### Finding 5: Skills that explicitly state investigations are NOT the deliverable

**Evidence:**
- `issue-creation` skill explicitly states: "Unlike investigation skill (produces investigation file), this skill produces a beads issue directly. The investigation happens, but it's internalized - the issue captures the understanding without a separate artifact."
- `systematic-debugging` skill states: "Investigation files are **recommended** for complex bugs but **optional** for simple fixes."
- `kb-reflect` handles synthesis OF investigations but doesn't create new investigation files itself

**Source:** ~/.claude/skills/src/worker/issue-creation/SKILL.md, ~/.claude/skills/worker/systematic-debugging/SKILL.md

**Significance:** Multiple skills explicitly recognize that not all work needs investigation artifacts, yet they're still being told to create them by the spawn template.

---

## Synthesis

**Key Insights:**

1. **Root Cause: Spawn Template Default** - The SPAWN_CONTEXT template (pkg/spawn/context.go:203-226) includes "SET UP investigation file" as a hardcoded deliverable for ALL worker spawns, regardless of skill type. This universal default explains the 312:1 ratio (936 investigations, only 3 synthesized).

2. **Skill Intent vs Template Reality** - Only 5-7 skills should produce investigations as their PRIMARY artifact (investigation, architect, research, codebase-audit, reliability-testing). Skills like feature-impl only need investigations when explicitly configured with investigation phase. Systematic-debugging and issue-creation explicitly state investigations are optional/not-the-deliverable. Yet all receive the same template directive.

3. **Investigation Artifact Pollution** - Recent workspace analysis shows feature-impl spawns for bug fixes (`og-feat-fix-claude-md`) and implementations (`og-feat-implement-kb-reflect`) creating investigation files. The `inv-fix-*` and `inv-implement-*` filename pattern confirms non-exploratory work is creating exploratory artifacts.

**Answer to Investigation Question:**

**Which skills produce investigation artifacts?** Currently, ALL worker skills produce investigations because the spawn template defaults to including investigation file creation as a deliverable (Finding 1).

**Should they?** No. Only these skills should produce investigations as primary deliverables:
- **investigation** - Exploratory understanding work  
- **architect** - Strategic design decisions (produces investigation with `design-` prefix)
- **research** - External research (produces investigation with `research-` prefix)
- **codebase-audit** - Systematic codebase analysis (produces investigation with `audit-` prefix)
- **reliability-testing** - Production hardening work
- **feature-impl** - ONLY when spawned with `--phases investigation,*` (investigation phase explicitly configured)
- **systematic-debugging** - ONLY for complex bugs (optional, not default)

The spawn template should NOT include investigation file creation as a universal default. Instead, it should be skill-specific: only skills whose PURPOSE is exploratory/understanding work should default to creating investigation artifacts.

---

## Structured Uncertainty

**What's tested:**

- ✅ Spawn template includes investigation deliverable (verified: read pkg/spawn/context.go:203-226)
- ✅ Only 5-7 skills explicitly document investigation artifact production (verified: grepped all SKILL.md files)
- ✅ feature-impl spawns are creating investigation files for implementation work (verified: checked workspaces og-feat-implement-kb-reflect and og-feat-fix-claude-md)
- ✅ Investigation file count is 936 total, 3 synthesized, 45 in Feb 2026 (verified: from spawn context evidence)

**What's untested:**

- ⚠️ Whether removing investigation default will break existing agent workflows (assumption: agents will adapt, but not validated)
- ⚠️ Whether skill-specific deliverable injection is feasible with current template architecture (would need code review to confirm)
- ⚠️ Impact on synthesis workflow if investigation count drops significantly (kb reflect may need adjustment)

**What would change this:**

- Finding would be wrong if other skills beyond the 5-7 identified have legitimate exploratory purposes requiring investigation artifacts
- Finding would be wrong if investigation files serve additional purposes beyond exploratory work (e.g., required for completion verification)
- Finding would be wrong if spawn template injection is skill-aware already (code inspection shows it's not)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

Classify each recommendation by authority level to route to the appropriate decision-maker:

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Remove investigation file deliverable from default spawn template; make it skill-specific | architectural | Crosses spawn architecture, skill system, and verification gates - requires coordination across subsystems |

**Authority Levels:**
- **implementation**: Worker decides within scope (reversible, single-scope, clear criteria, no cross-boundary impact)
- **architectural**: Orchestrator decides across boundaries (cross-component, multiple valid approaches, requires synthesis)
- **strategic**: Dylan decides on direction (irreversible, resource commitment, value judgment, premise-level question)

**Classification test:** "Does this decision stay inside my scoped context, or does it reach out?"
- Stays inside → implementation
- Reaches to other components/agents → architectural
- Reaches to values/direction/irreversibility → strategic

### Recommended Approach ⭐

**Skill-Aware Investigation Deliverable Injection** - Move investigation file creation from default spawn template to skill-specific configuration

**Why this approach:**
- Directly addresses root cause (universal default in spawn template)
- Aligns deliverables with skill intent (exploratory skills produce investigations, implementation skills don't)
- Should reduce investigation count from 936 → ~200-300 (assuming 70% are non-exploratory work)
- Preserves investigation artifacts for skills that genuinely need them

**Trade-offs accepted:**
- Requires spawn template refactoring (skill-specific sections)
- May need to update completion verification if it depends on investigation file existence
- Skills that currently rely on investigation file as coordination artifact will need alternative (WORKSPACE.md or beads comments)

**Implementation sequence:**
1. **Add skill metadata flag** - Add `ProducesInvestigation: bool` to skill manifest/config
2. **Conditional template injection** - Modify pkg/spawn/context.go to only inject investigation deliverable when skill.ProducesInvestigation == true
3. **Update skill configs** - Mark investigation, architect, research, codebase-audit, reliability-testing as ProducesInvestigation=true
4. **Update completion verification** - Ensure `orch complete` doesn't require investigation files for non-investigative skills
5. **Validate with test spawn** - Spawn feature-impl without investigation phase, verify no investigation deliverable in template

### Alternative Approaches Considered

**Option B: Keep universal default, add --skip-investigation flag**
- **Pros:** Minimal template changes, backwards compatible
- **Cons:** Requires orchestrators to remember to add flag for every non-investigative spawn (moves burden to spawn caller, doesn't fix root cause)
- **When to use instead:** If skill manifest changes are too complex

**Option C: Remove investigation deliverable entirely, let skills create if needed**
- **Pros:** Simplest template change (just delete the section)
- **Cons:** Skills that SHOULD produce investigations won't have guidance; agents may skip creating needed artifacts
- **When to use instead:** If we trust skills to self-document their deliverables completely

**Option D: Phase-based detection for feature-impl only**
- **Pros:** Targeted fix for the biggest offender (feature-impl without investigation phase)
- **Cons:** Doesn't address systematic-debugging, issue-creation, or other non-investigative skills; partial solution
- **When to use instead:** As a quick interim fix before implementing Option A

**Rationale for recommendation:** Option A (skill-aware injection) directly addresses root cause (universal default), scales to all skills, and preserves investigation guidance where needed. Options B and D are workarounds that shift complexity to spawn callers or address symptoms only.

---

### Implementation Details

**What to implement first:**
- **Quick win:** Add feature-impl phase detection (if --phases doesn't include "investigation", skip investigation deliverable injection)
- **Foundation:** Add skill manifest field `produces_investigation: bool` to skill YAML frontmatter
- **Template refactor:** Modify SPAWN_CONTEXT template to conditionally include investigation section based on skill config

**Things to watch out for:**
- ⚠️ **Completion verification dependency:** Check if `orch complete` requires investigation_path in beads comments for verification gates
- ⚠️ **Coordination artifact gap:** Some agents may use investigation files as workspace coordination - need alternative (recommend WORKSPACE.md or rely on beads comments)
- ⚠️ **Template conditionals complexity:** Go template syntax for nested conditionals can get messy - consider extracting to helper function
- ⚠️ **Skill manifest location:** Skills are in multiple locations (~/.claude/skills/src/worker/, ~/.claude/skills/worker/, orch-knowledge repo) - ensure consistent metadata

**Areas needing further investigation:**
- Does `orch complete` evidence gate specifically check for investigation files, or just beads comment mentions?
- Are there other places besides SPAWN_CONTEXT that assume investigation files exist?
- Should WORKSPACE.md be promoted as the universal coordination artifact instead of investigation files?

**Success criteria:**
- ✅ feature-impl spawn without investigation phase produces NO investigation deliverable in SPAWN_CONTEXT
- ✅ investigation skill spawn still produces investigation deliverable
- ✅ Investigation count growth slows from 45/month to ~10-15/month (70% reduction)
- ✅ `orch complete` still works for both investigation-producing and non-investigation-producing skills

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:203-226` - SPAWN_CONTEXT template with universal investigation deliverable
- `~/.claude/skills/worker/investigation/SKILL.md` - investigation skill definition
- `~/.claude/skills/worker/architect/SKILL.md` - architect skill definition  
- `~/.claude/skills/worker/research/SKILL.md` - research skill definition
- `~/.claude/skills/worker/feature-impl/SKILL.md` - feature-impl skill definition (investigation phase optional)
- `~/.claude/skills/worker/systematic-debugging/SKILL.md` - systematic-debugging skill definition (investigations optional)
- `~/.claude/skills/src/worker/issue-creation/SKILL.md` - issue-creation skill definition (explicitly no investigation file)
- `.orch/workspace/og-feat-implement-kb-reflect-14feb-8a59/SPAWN_CONTEXT.md` - Example feature-impl spawn
- `.orch/workspace/og-feat-fix-claude-md-14feb-e330/SPAWN_CONTEXT.md` - Example bug fix spawn

**Commands Run:**
```bash
# Find all SKILL.md files
find ~/.claude/skills -name "SKILL.md" -type f

# Search for investigation artifact references in skills
grep -r "investigation" ~/.claude/skills/*/SKILL.md | grep -i "deliverable\|output\|required\|artifact"
grep -r ".kb/investigations" ~/.claude/skills/*/SKILL.md

# List recent investigation files
ls -lt .kb/investigations/*.md | head -20

# Check recent workspaces for skills
grep "SKILL GUIDANCE" .orch/workspace/og-feat-*/SPAWN_CONTEXT.md
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-02-14-inv-synthesize-synthesize-investigations-10-synthesis.md` - Synthesis work showing 29 clusters needing consolidation
- **Model:** `.kb/models/kb-reflect-cluster-hygiene.md` - kb reflect behavior and synthesis patterns

---

## Investigation History

**2026-02-14 (start):** Investigation started
- Initial question: Which skills produce investigation artifacts and should they?
- Context: 936 investigations exist, only 3 synthesized (312:1 ratio), 45 created in February alone

**2026-02-14 (finding 1):** Discovered root cause in spawn template
- Found SPAWN_CONTEXT template includes investigation file creation as universal default for ALL worker spawns
- Located in pkg/spawn/context.go:203-226

**2026-02-14 (finding 2-3):** Audited skills for investigation artifact production
- Only 5-7 skills should produce investigations as primary deliverable
- feature-impl and systematic-debugging have investigation creation as optional, not required

**2026-02-14 (finding 4-5):** Validated hypothesis with workspace analysis
- Confirmed feature-impl spawns for bug fixes and implementations creating investigation files
- Multiple skills explicitly state investigations are NOT the deliverable

**2026-02-14 (synthesis):** Investigation complete
- Status: Complete
- Key outcome: Spawn template universal default causes investigation proliferation; recommendation to make investigation deliverable skill-specific
