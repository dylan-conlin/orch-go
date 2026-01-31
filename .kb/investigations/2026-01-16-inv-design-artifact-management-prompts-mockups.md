<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Design mockups need paired .prompt.md files and manifest.json tracking for reproduceability, versioning, and approval workflow.

**Evidence:** ui-design-session stores mockups in prompts/ and mockups/ separately (no pairing); screenshot storage decision establishes manifest pattern; spawn context specifies exact schema with approval metadata.

**Knowledge:** Prompts are first-class artifacts (not just documentation) - reproduceability requires explicit pairing; manifest enables verification and workflow tracking; versioning convention (-v1, -v2) shows iteration not sequence.

**Next:** Update ui-design-session skill guidance with artifact pairing convention, manifest.json schema, prompt capture workflow, and quality checklist verification.

**Promote to Decision:** Actioned - patterns in artifact organization docs

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

# Investigation: Design Artifact Management Prompts Mockups

**Question:** How should ui-design-session store and track mockups with their prompts, including versioning and approval workflow?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** Feature Implementation Agent
**Phase:** Complete
**Next Step:** None - ready for implementation
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: ui-design-session Has Artifact Conventions But No Prompt Tracking

**Evidence:** The skill defines file structure with `mockups/`, `handoff/`, and `prompts/` directories (lines 400-416), but prompts are stored separately from the mockups they generate. No pairing mechanism exists to link `dashboard-prompt.md` to `01-dashboard-desktop.png`.

**Source:** `~/.claude/skills/worker/ui-design-session/SKILL.md:396-441`

**Significance:** Without explicit pairing, it's unclear which prompt generated which mockup, especially across iterations. This makes it hard to reproduce or iterate on specific versions.

---

### Finding 2: Screenshot Storage Decision Establishes Workspace Pattern

**Evidence:** The 2026-01-07 screenshot storage investigation established `.orch/workspace/{agent}/screenshots/` as the canonical location with a manifest file for tracking metadata. The manifest includes `captured_at`, `context`, `tool`, and `url` fields.

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2026-01-07-design-screenshot-artifact-storage-decision.md:220-283`

**Significance:** This pattern can be extended for design mockups. The manifest approach enables verification and metadata tracking, which aligns with the spawn context requirements for approval workflow.

---

### Finding 3: No Versioning Convention for Design Iterations

**Evidence:** Current naming convention suggests sequential numbering (`01-dashboard-desktop.png`, `02-dashboard-mobile.png`) but no explicit versioning for iterations of the same view (e.g., `dashboard-v1.png`, `dashboard-v2.png` after feedback).

**Source:** `~/.claude/skills/worker/ui-design-session/SKILL.md:418-424`

**Significance:** Design is inherently iterative. Without versioning, it's unclear which mockup is the "current" version and which are superseded iterations. The spawn context explicitly mentions `ready-queue-v1.png` and `ready-queue-v2.png` as the expected pattern.

---

### Finding 4: No Approval Workflow Metadata

**Evidence:** The skill mentions "orchestrator approval" and "final approval received" checkpoints (lines 256, 522) but provides no mechanism to record approval status, who approved, or when. The quality checklist has manual checkboxes but no persistent tracking.

**Source:** `~/.claude/skills/worker/ui-design-session/SKILL.md:256-279, 500-534`

**Significance:** Approval is critical for design handoff. Without structured metadata, approval state is implicit (must review beads comments) rather than explicit (queryable from manifest).

---

### Finding 5: Spawn Context Specifies Manifest Schema

**Evidence:** The spawn context provides a manifest format example with `artifacts` array containing `filename`, `type`, `prompt_file`, `approved`, and `approved_at` fields.

**Source:** SPAWN_CONTEXT.md:10-23

**Significance:** This is the target schema to implement. It directly addresses the gaps found in Findings 3 and 4 by providing versioning via filename and approval tracking via metadata fields.

---

## Synthesis

**Key Insights:**

1. **Prompts Are First-Class Artifacts, Not Afterthoughts** - The spawn context explicitly pairs every mockup with its prompt file (`.png` + `.prompt.md`). This isn't just documentation—it's reproduceability. If an orchestrator wants iteration 2 tweaked, the agent can load `ready-queue-v2.prompt.md` and adjust rather than guessing what prompt generated it.

2. **Manifest Enables Verification and Workflow** - Following the screenshot storage pattern, a manifest.json provides: (a) structured metadata for tooling (orch complete can verify files exist), (b) approval workflow tracking (which mockups are approved vs drafts), and (c) chronological history (when each artifact was created/approved).

3. **Versioning Is Semantic, Not Sequential** - Current convention (`01-`, `02-`) implies ordering but not iteration. The spawn context uses `-v1`, `-v2` suffix to show "this is the second iteration of ready-queue". This distinction matters: `dashboard-desktop-v1.png` and `dashboard-mobile-v1.png` are different views at the same version, not chronological steps.

**Answer to Investigation Question:**

ui-design-session should store artifacts in `{workspace}/screenshots/` with the following structure:

```
.orch/workspace/{agent}/screenshots/
├── manifest.json                     # Artifact metadata and approval tracking
├── ready-queue-v1.png                # Initial mockup
├── ready-queue-v1.prompt.md          # Prompt that generated v1
├── ready-queue-v2.png                # Iteration after feedback
├── ready-queue-v2.prompt.md          # Updated prompt for v2
└── dashboard-mobile-v1.png           # Different view, same version
└── dashboard-mobile-v1.prompt.md
```

The manifest.json format:
```json
{
  "artifacts": [
    {
      "filename": "ready-queue-v2.png",
      "type": "design_mockup",
      "prompt_file": "ready-queue-v2.prompt.md",
      "created_at": "2026-01-16T10:30:00Z",
      "approved": true,
      "approved_at": "2026-01-16T11:45:00Z",
      "approved_by": "orchestrator",
      "supersedes": "ready-queue-v1.png",
      "context": "Dashboard ready queue - revised after feedback to reduce density"
    }
  ]
}
```

This integrates with existing screenshot storage decision (same directory location, manifest pattern) while adding design-specific metadata (prompt files, approval workflow, versioning).

---

## Structured Uncertainty

**What's tested:**

- ✅ ui-design-session has artifact conventions but no prompt pairing (verified: read SKILL.md:396-441)
- ✅ Screenshot storage decision establishes workspace/manifest pattern (verified: read 2026-01-07 investigation)
- ✅ Spawn context specifies manifest schema with approval fields (verified: read SPAWN_CONTEXT.md:10-23)
- ✅ Current skill uses sequential numbering not versioning (verified: SKILL.md:418-424)

**What's untested:**

- ⚠️ Agent compliance with paired artifact creation (not validated - requires spawn testing)
- ⚠️ Manifest.json workflow fits design iteration cadence (not user-tested - might be too heavyweight)
- ⚠️ `-v1`, `-v2` suffix convention is clear to agents (not validated with actual agent sessions)
- ⚠️ Orchestrator approval workflow integrates with beads comments (not tested end-to-end)

**What would change this:**

- Finding would be wrong if agents consistently forget to update manifest (suggests auto-generation needed)
- Design would fail if `-v1` suffix is confused with view variant (would need different convention)
- Pattern would fail if approval metadata duplicates beads comments with no tooling benefit (suggests simpler approach)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Paired Artifacts with Manifest Tracking** - Store mockups and prompts as `.png` + `.prompt.md` pairs in `{workspace}/screenshots/` with a `manifest.json` tracking metadata, approval workflow, and versioning relationships.

**Why this approach:**
- **Reproduceability:** Agent can load exact prompt that generated any mockup version for iteration
- **Verification:** manifest.json enables `orch complete` to verify artifacts exist (aligns with screenshot storage decision)
- **Approval Workflow:** Structured tracking of which mockups are approved vs drafts
- **Version Clarity:** `-v1`, `-v2` suffix shows iteration relationship, `supersedes` field chains versions
- **Integration:** Extends existing screenshot storage pattern (same directory, same manifest approach)

**Trade-offs accepted:**
- **Manual Manifest Updates:** Agent must update manifest.json when creating artifacts (not auto-generated yet)
- **No Diff Tooling:** Manifest tracks versions but doesn't show visual diffs (acceptable - out of scope)
- **Workspace-Scoped:** Artifacts scoped to single workspace, not shared across agents (acceptable - matches screenshot storage decision)

**Implementation sequence:**
1. **Update ui-design-session skill guidance** - Document artifact pairing convention and manifest schema
2. **Add prompt capture instructions** - When generating mockups, save prompt to `.prompt.md` immediately
3. **Provide manifest.json template** - Include in skill guidance with required/optional fields
4. **Update quality checklist** - Add verification that manifest.json exists and references all artifacts

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
1. **Skill guidance update** - Add "Artifact Management" section to ui-design-session SKILL.md.template with pairing convention and manifest schema
2. **Manifest.json schema specification** - Document required fields (`filename`, `type`, `prompt_file`, `created_at`) and optional fields (`approved`, `approved_at`, `approved_by`, `supersedes`, `context`)
3. **Prompt capture workflow** - Update "Phase 2: Mockup Generation" to save prompts alongside mockups
4. **Quality checklist** - Add manifest verification to completion criteria

**Manifest.json Schema Specification:**

```json
{
  "$schema": "artifact-manifest-v1",
  "artifacts": [
    {
      "filename": "string (required) - mockup filename relative to screenshots/",
      "type": "string (required) - 'design_mockup'",
      "prompt_file": "string (required) - .prompt.md filename relative to screenshots/",
      "created_at": "string (required) - ISO 8601 timestamp",
      "approved": "boolean (optional) - true if orchestrator approved",
      "approved_at": "string (optional) - ISO 8601 timestamp of approval",
      "approved_by": "string (optional) - who approved (typically 'orchestrator')",
      "supersedes": "string (optional) - filename of previous version this replaces",
      "context": "string (optional) - brief description of what this mockup shows"
    }
  ]
}
```

**Prompt File Format (.prompt.md):**

```markdown
# Prompt: [Mockup Name]

**Generated:** 2026-01-16T10:30:00Z
**Model:** Gemini 2.5 Flash (Nano Banana)
**Version:** v1

## Design Brief Reference

- **Feature:** Dashboard Ready Queue
- **View:** Desktop (1440px)
- **Design Direction:** Precision & Density
- **Color Foundation:** Warm neutral (#FAFAFA) with blue accent (#3B82F6)

## Prompt

[Full Nano Banana prompt in markdown format - copy exactly what was sent to the model]

## Iteration Notes

[If v2+: What changed from previous version and why]
```

**File Naming Conventions:**

- **Mockups:** `{feature-name}-{view}-v{version}.png`
  - Examples: `ready-queue-desktop-v1.png`, `ready-queue-mobile-v2.png`
- **Prompts:** Same name as mockup but `.prompt.md` extension
  - Examples: `ready-queue-desktop-v1.prompt.md`
- **Versions:** Increment when iterating on same view (v1 → v2 → v3)
- **Views:** Describe viewport or detail level (desktop, mobile, detail, overview)

**Workflow Integration:**

1. **Before generating mockup:**
   - Create prompt file: `ready-queue-desktop-v1.prompt.md`
   - Document design brief context in prompt file
   - Write Nano Banana prompt in markdown

2. **After generating mockup:**
   - Save mockup: `ready-queue-desktop-v1.png` in `{workspace}/screenshots/`
   - Update manifest.json with artifact entry
   - Commit both files together: `git add screenshots/ && git commit -m "design: add ready-queue desktop mockup v1"`

3. **After orchestrator approval:**
   - Update manifest.json: set `approved: true`, `approved_at`, `approved_by`
   - Commit approval update: `git commit -m "design: approve ready-queue desktop mockup"`

4. **When iterating:**
   - Create new version files: `ready-queue-desktop-v2.png` and `.prompt.md`
   - In manifest, add `supersedes: "ready-queue-desktop-v1.png"` to v2 entry
   - Document iteration notes in prompt file

**Things to watch out for:**
- ⚠️ **Manual Manifest Sync:** Agents might forget to update manifest.json after creating artifacts. Add explicit instruction in skill to update manifest immediately after saving mockup.
- ⚠️ **Timestamp Consistency:** Use ISO 8601 format consistently. Provide example: `2026-01-16T10:30:00Z`
- ⚠️ **Approval Authority:** Only orchestrator should set `approved: true`. Worker agents propose but don't self-approve.
- ⚠️ **Supersedes Chain:** If v3 supersedes v2 which supersedes v1, manifest should only record direct supersession (v3 supersedes v2), not transitive (v3 doesn't list v1).

**Areas needing further investigation:**
- **Auto-manifest generation:** Could `orch complete` auto-generate manifest from directory scan? (Out of scope - requires tooling changes)
- **Visual diff tooling:** Should manifest link to diff images showing v1→v2 changes? (Out of scope - no tooling exists yet)
- **Cross-workspace references:** If design-session creates mockups, can feature-impl reference them? (Out of scope - workspace artifacts are ephemeral)

**Success criteria:**
- ✅ ui-design-session skill documents artifact pairing and manifest schema
- ✅ Agents create `.prompt.md` files paired with every mockup
- ✅ manifest.json exists and references all artifacts in screenshots/ directory
- ✅ Versioning convention (`-v1`, `-v2`) consistently applied
- ✅ Approval workflow metadata captured in manifest (approved, approved_at, approved_by)
- ✅ Quality checklist includes manifest verification

---

## References

**Files Examined:**
- `~/.claude/skills/worker/ui-design-session/SKILL.md` - Current artifact conventions and workflow
- `~/orch-knowledge/skills/src/worker/ui-design-session/.skillc/` - Skill source for updates
- `.kb/investigations/2026-01-07-design-screenshot-artifact-storage-decision.md` - Screenshot storage pattern
- `.kb/investigations/2026-01-09-inv-create-ui-design-session-skill.md` - Skill creation context
- `SPAWN_CONTEXT.md` - Task requirements and manifest schema specification

**Commands Run:**
```bash
# Find ui-design-session skill files
find ~/.claude/skills -name "*ui-design*" -o -name "*design-session*"

# Check skill source structure
ls -la ~/orch-knowledge/skills/src/worker/ui-design-session/.skillc/

# Verify screenshot storage decision location
find .kb -name "*screenshot*storage*.md"
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-07-design-screenshot-artifact-storage-decision.md` - Establishes workspace screenshots/ directory and manifest pattern
- **Investigation:** `.kb/investigations/2026-01-09-inv-create-ui-design-session-skill.md` - Documents skill structure and dependencies
- **Skill:** `~/.claude/skills/worker/ui-design-session/SKILL.md` - Target for implementation

---

## Investigation History

**2026-01-16 (session start):** Investigation started
- Initial question: How should ui-design-session store and track mockups with their prompts, including versioning and approval workflow?
- Context: Spawned from orch-go-gy1o4.3.3 to design artifact management for ui-design-session skill

**2026-01-16 (findings phase):** Examined existing artifacts
- Found ui-design-session has artifact conventions but no prompt pairing
- Found screenshot storage decision establishes workspace/manifest pattern
- Found spawn context specifies exact manifest schema with approval fields
- Identified versioning gap (sequential numbering vs iteration versioning)

**2026-01-16 (synthesis phase):** Designed artifact management system
- Defined paired artifact pattern (`.png` + `.prompt.md`)
- Specified manifest.json schema with approval workflow fields
- Designed file naming convention with version suffix (`-v1`, `-v2`)
- Documented workflow integration for prompt capture and approval tracking

**2026-01-16 (completion):** Investigation completed
- Status: Complete
- Key outcome: Full artifact management specification ready for ui-design-session skill update
