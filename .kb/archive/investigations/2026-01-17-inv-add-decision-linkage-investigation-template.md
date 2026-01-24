<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Investigation template now includes Patches-Decision field in Lineage metadata section to enable traceable chains from decisions to investigations that patch/extend them.

**Evidence:** Updated kb-cli/cmd/kb/create.go:60 and ~/.kb/templates/INVESTIGATION.md:49 with new field, rebuilt kb binary, verified field appears in newly generated investigations via test creation and grep verification.

**Knowledge:** kb-cli template system has dual-location architecture (hardcoded + override), requiring updates in both places; loadTemplate function loads from ~/.kb/templates/ first, falling back to hardcoded constant only if file doesn't exist.

**Next:** Commit changes to both repositories (kb-cli for source, orch-go for investigation), create SYNTHESIS.md, mark Phase: Complete.

**Promote to Decision:** [recommend-yes | recommend-no | unclear] - Orchestrator/human decides; worker flags

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

# Investigation: Add Decision Linkage Investigation Template

**Question:** How should decision linkage be added to investigation template to enable traceable chains from decisions to patches?

**Started:** 2026-01-17
**Updated:** 2026-01-17
**Owner:** Agent og-feat-add-decision-linkage-17jan-4adf
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Investigation template is hardcoded in kb-cli

**Evidence:** The investigationTemplate constant is defined starting at line 15 in /Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go. The template is a Go string constant that gets applied via variable substitution.

**Source:** /Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go:15-235

**Significance:** To add decision linkage, I need to modify the hardcoded template string in this file and rebuild the kb binary.

---

### Finding 2: Template has existing Lineage section for metadata

**Evidence:** Template already has a Lineage section at lines 59-62 with fields for Extracted-From, Supersedes, and Superseded-By. This section is marked with "fill only when applicable" comment.

**Source:** /Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go:59-62

**Significance:** The decision linkage field fits naturally in the metadata section near Lineage, making it visible at the top of investigations for easy review trigger detection.

---

### Finding 3: Template has Related Artifacts section

**Evidence:** Template has a "Related Artifacts" section at lines 216-220 that references decisions, investigations, and workspaces. This section appears at the end of the template.

**Source:** /Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go:216-220

**Significance:** Decision linkage in metadata (top) enables programmatic detection for review triggers, while Related Artifacts (bottom) provides contextual references. Both serve different purposes.

---

### Finding 4: Template override mechanism exists

**Evidence:** The loadTemplate function in create.go loads templates from ~/.kb/templates/ first, falling back to hardcoded templates only if file doesn't exist. The ~/.kb/templates/INVESTIGATION.md file was overriding the hardcoded template.

**Source:** /Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go:560-574, ~/.kb/templates/INVESTIGATION.md

**Significance:** Changes must be made to BOTH the hardcoded template (for other users/systems) AND the override file (for current system). This two-location requirement is a maintenance consideration.

---

## Synthesis

**Key Insights:**

1. **Template modification requires dual-location updates** - The kb-cli system has both hardcoded templates in Go source and override templates in ~/.kb/templates/. Both must be updated for consistent behavior across rebuilds and runtime.

2. **Metadata placement enables programmatic detection** - Placing Patches-Decision in the Lineage metadata section (near top) makes it machine-readable for review triggers, distinct from the Related Artifacts section which serves documentation purposes.

3. **Template override mechanism provides flexibility** - The loadTemplate fallback pattern allows users to customize templates without modifying source code, but requires awareness for maintenance tasks.

**Answer to Investigation Question:**

Decision linkage should be added as a **Patches-Decision** field in the Lineage metadata section of the investigation template. Implementation requires updating both the hardcoded template in kb-cli/cmd/kb/create.go (line 60) and the override template at ~/.kb/templates/INVESTIGATION.md (line 49) to ensure consistent behavior. The field format is: `**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]`

---

## Structured Uncertainty

**What's tested:**

- ✅ Template modification works in both locations (verified: created test investigations, confirmed Patches-Decision field appears)
- ✅ kb binary rebuild includes hardcoded template changes (verified: strings command shows new field in binary)
- ✅ Template override mechanism loads from ~/.kb/templates/ first (verified: changes to hardcoded template didn't appear until override file was updated)

**What's untested:**

- ⚠️ Review trigger automation will correctly parse Patches-Decision field (implementation not built yet)
- ⚠️ Decision documents will have corresponding mechanism to track patches (cross-reference system not implemented)
- ⚠️ Template change compatibility with existing investigations (no migration path defined)

**What would change this:**

- Finding would be wrong if kb binary loads from different template location than ~/.kb/templates/ and hardcoded constant
- Finding would be wrong if template override precedence order differs from observed behavior
- Finding would be wrong if rebuild process doesn't update binary with source changes

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Metadata field in Lineage section** - Add Patches-Decision field to investigation template's Lineage metadata section for machine-readable decision linkage.

**Why this approach:**
- Metadata placement enables programmatic detection for review triggers (top of file = easy parsing)
- Lineage section already exists for artifact relationships (consistent pattern)
- Field format matches existing template conventions (bold label + description)

**Trade-offs accepted:**
- Requires dual-location updates (hardcoded + override templates) for maintenance
- Manual field population by investigation authors (not auto-detected)

**Implementation sequence:**
1. Update hardcoded template in kb-cli/cmd/kb/create.go - establishes source of truth
2. Update override template at ~/.kb/templates/INVESTIGATION.md - makes it work on current system
3. Rebuild kb binary - applies hardcoded template changes
4. Test generation - verify field appears in new investigations

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
- Hardcoded template modification (source of truth for new kb installations)
- Override template modification (makes it work on current system)
- Binary rebuild (applies source changes)

**Things to watch out for:**
- ⚠️ Template override mechanism can silently override hardcoded changes (discovered during testing)
- ⚠️ String escaping in Go template constant (backticks require concatenation)
- ⚠️ Consistent formatting with existing Lineage fields (spacing, brackets, descriptions)

**Areas needing further investigation:**
- How review triggers will detect and parse Patches-Decision field
- Whether decision documents need corresponding "Patched-By" field for bidirectional linking
- Migration strategy for existing investigations that should have decision linkage

**Success criteria:**
- ✅ New investigations generated with `kb create investigation` include Patches-Decision field
- ✅ Field appears in Lineage section with proper formatting
- ✅ Both template locations updated (verified via git diff and file read)

---

## References

**Files Examined:**
- /Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/create.go - Investigation template definition and loadTemplate mechanism
- ~/.kb/templates/INVESTIGATION.md - Template override file that takes precedence over hardcoded template

**Commands Run:**
```bash
# Build kb binary
cd /Users/dylanconlin/Documents/personal/kb-cli && make build

# Test template generation
kb create investigation test-decision-linkage-validation

# Verify field in binary
strings /Users/dylanconlin/Documents/personal/kb-cli/build/kb | grep "Patches-Decision"

# Check template override
grep -B 2 -A 4 "Patches-Decision" ~/.kb/templates/INVESTIGATION.md
```

**External Documentation:**
- None

**Related Artifacts:**
- **Workspace:** /Users/dylanconlin/Documents/personal/orch-go/.orch/workspace/og-feat-add-decision-linkage-17jan-4adf/ - Spawn workspace for this task

---

## Investigation History

**[2026-01-17 14:40]:** Investigation started
- Initial question: How should decision linkage be added to investigation template?
- Context: Spawned from beads issue orch-go-aj93a to enable traceable chains from decisions to patches

**[2026-01-17 14:42]:** Template override mechanism discovered
- Found that ~/.kb/templates/INVESTIGATION.md overrides hardcoded template
- Initial confusion: binary had changes but generated files didn't

**[2026-01-17 14:45]:** Implementation completed
- Updated both template locations (hardcoded + override)
- Tested and verified field appears in generated investigations

**[2026-01-17 14:46]:** Investigation completed
- Status: Complete
- Key outcome: Patches-Decision field successfully added to investigation template in both required locations
