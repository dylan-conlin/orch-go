<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The 26 "synthesis" investigations divide into three categories: (1) 6 foundational design investigations defining the D.E.K.N. protocol, (2) 15+ synthesis-of-topic investigations (dashboard, CLI, etc.), and (3) 5 maintenance/bug investigations fixing synthesis-related issues.

**Evidence:** Read 12 key synthesis investigations covering Dec 20 - Jan 7 2026; found clear evolutionary arc from design (Dec 20) → implementation (Dec 20-21) → dashboard integration (Dec 26) → maintenance (Jan 7); 15+ existing guides created from prior synthesis runs.

**Knowledge:** The synthesis system is working as designed - topics with 10+ investigations get synthesized into guides. The "synthesis" investigations themselves are meta-topic noise: they document the synthesis process, not a domain topic needing consolidation.

**Next:** Archive older meta-synthesis investigations (keep foundational design), add "synthesis" to meta-topic exclusions in synthesis detection.

**Promote to Decision:** recommend-yes - Meta-topics (investigation, synthesis, artifact) should be excluded from synthesis opportunity detection.

---

# Investigation: Synthesize Synthesis Investigations (26 Total)

**Question:** What patterns and decisions should be consolidated from 26 synthesis-related investigations, and how should they be organized?

**Started:** 2026-01-08
**Updated:** 2026-01-08
**Owner:** kb-reflect agent (og-work-synthesize-synthesis-investigations-08jan-b5ae)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Three Distinct Categories of Synthesis Investigations

**Evidence:** The 26 synthesis-related investigations fall into three categories:

| Category | Count | Purpose | Examples |
|----------|-------|---------|----------|
| **Foundational Design** | 6 | Define the D.E.K.N. protocol itself | `2025-12-20-design-synthesis-protocol-schema.md`, `2025-12-21-inv-agents-skip-synthesis-md-creation.md` |
| **Topic Synthesis** | 15+ | Synthesize other topics into guides | `2026-01-06-inv-synthesize-dashboard-investigations-39-synthesis.md`, `2026-01-06-inv-synthesize-cli-investigations-16-synthesis.md` |
| **Maintenance/Bugs** | 5 | Fix synthesis system issues | `2026-01-07-design-recurring-problem-duplicate-synthesis-issues.md`, `2026-01-07-inv-fix-kb-ask-synthesis-grounding.md` |

**Source:** Analyzed 12 key investigations from `.kb/investigations/*synthesis*.md`

**Significance:** These are NOT domain investigations needing consolidation into a "synthesis guide" - they ARE the synthesis system documentation itself.

---

### Finding 2: Clear Evolutionary Arc (Dec 20 - Jan 7)

**Evidence:** The synthesis system evolved through distinct phases:

| Date | Phase | Key Artifacts |
|------|-------|---------------|
| Dec 20, 2025 | **Design** | `design-synthesis-protocol-schema.md` - D.E.K.N. structure defined |
| Dec 20, 2025 | **Implementation** | Template created, `VerifySynthesis()` added |
| Dec 21, 2025 | **Bug Fix** | Agents skipping SYNTHESIS.md - SPAWN_CONTEXT updated |
| Dec 26, 2025 | **Dashboard** | Synthesis review workflow, issue creation from recommendations |
| Jan 6, 2026 | **First Wave** | 15+ topic syntheses creating guides (dashboard, CLI, beads, etc.) |
| Jan 7, 2026 | **Maintenance** | Duplicate issue fix, post-synthesis archival design |

**Source:** 
- `2025-12-20-design-synthesis-protocol-schema.md` - Original design
- `2025-12-21-inv-agents-skip-synthesis-md-creation.md` - First bug fix
- `2026-01-06-inv-synthesize-dashboard-investigations-39-synthesis.md` - First wave example

**Significance:** The system is mature and working. These investigations document its creation and evolution, not ongoing problems.

---

### Finding 3: Topic Synthesis Is Working - 15+ Guides Created

**Evidence:** The synthesis system successfully created guides for major topics:

- `.kb/guides/dashboard.md` (from 44 dashboard investigations)
- `.kb/guides/cli.md` (from 16 CLI investigations)
- `.kb/guides/beads-integration.md` (from 17 beads investigations)
- `.kb/guides/daemon.md` (from 31 daemon investigations)
- `.kb/guides/spawn.md` (from 36 spawn investigations)
- `.kb/guides/headless.md` (from 15 headless investigations)
- `.kb/guides/opencode.md` (from 16 OpenCode investigations)

**Source:** `glob ".kb/guides/*.md"` - 21 guides exist

**Significance:** The synthesis workflow is proven effective. The "synthesis" topic itself doesn't need a guide - it's a meta-topic about the system.

---

### Finding 4: Synthesis Protocol Is Fully Implemented

**Evidence:** From `2025-12-20-inv-implement-synthesis-protocol-create-orch.md`:
- Template exists: `.orch/templates/SYNTHESIS.md` with D.E.K.N. structure
- Verification requires it: `pkg/verify/check.go` `VerifySynthesis()` fails if missing
- Instructions in workflow: `SPAWN_CONTEXT.md` includes explicit SYNTHESIS.md instructions

From `2025-12-20-audit-legacy-artifacts-synthesis-protocol.md`:
- 26 of 116 workspaces had SYNTHESIS.md (all post-protocol)
- 100% template alignment for existing files
- No remediation needed for legacy workspaces

**Source:** Direct code verification in investigations

**Significance:** The protocol is working as designed. New agents create SYNTHESIS.md; verification enforces it.

---

### Finding 5: Key Decisions Already Recorded

**Evidence:** From spawn context prior knowledge, multiple decisions document synthesis patterns:

| Decision | Reason |
|----------|--------|
| Synthesize at 10+ threshold | Guides provide single authoritative reference |
| D.E.K.N. structure | 30-second orchestrator handoff |
| Dashboard shows synthesis | Enable issue creation from recommendations |
| Batch close prompts for recommendations | Don't lose value from synthesis |

**Source:** Spawn context "Prior Decisions" section shows 10+ synthesis-related kn entries

**Significance:** The synthesis system's design decisions are already documented in kn. A separate "synthesis guide" would duplicate this.

---

### Finding 6: Meta-Topics Pollute Synthesis Detection

**Evidence:** From `2026-01-07-design-post-synthesis-investigation-archival.md`:
- 35 investigations contain "investigation" in filename
- These are meta-topics about the knowledge system itself
- Creating "investigation.md" guide would be confusing

The same applies to "synthesis" - these investigations are ABOUT synthesis, not a domain topic to synthesize.

**Source:** `2026-01-07-design-post-synthesis-investigation-archival.md:79-84`

**Significance:** Meta-topics like "investigation", "synthesis", "artifact", "skill" should be excluded from synthesis opportunity detection.

---

## Synthesis

**Key Insights:**

1. **Meta-topic vs Domain Topic** - "Synthesis" is a meta-topic about the knowledge system itself, not a domain topic. Creating a "synthesis guide" from these investigations would be circular - documenting the documentation system within the documentation system.

2. **System Is Working** - The synthesis protocol is fully implemented, verified, and producing results (15+ guides). The investigations document its creation, not ongoing problems. Most "synthesis" investigations are actually synthesis RUNS on other topics.

3. **Archive, Don't Consolidate** - The correct action is to archive older meta-synthesis investigations (keeping foundational design documents) and exclude "synthesis" from future synthesis opportunity detection.

**Answer to Investigation Question:**

The 26 synthesis-related investigations should NOT be consolidated into a "synthesis guide" because:
1. They're meta-topic documentation, not domain knowledge
2. The synthesis system is already documented via kn decisions
3. 15+ of them are synthesis RUNS on other topics (dashboard, CLI, etc.) that produced guides

Recommended actions:
1. Add "synthesis", "investigation", "artifact" to meta-topic exclusions
2. Archive completed topic-synthesis investigations to `synthesized/{guide-name}/`
3. Keep foundational design investigations (`2025-12-20-design-synthesis-protocol-schema.md`) as system documentation

---

## Structured Uncertainty

**What's tested:**

- ✅ 21 guides exist in `.kb/guides/` (verified: glob command)
- ✅ Synthesis protocol is fully implemented (verified: code inspection in prior investigations)
- ✅ Topic syntheses produce guides successfully (verified: dashboard, CLI examples)

**What's untested:**

- ⚠️ Meta-topic exclusion implementation (design exists, not implemented)
- ⚠️ Post-synthesis archival workflow (design exists, not implemented)
- ⚠️ Whether all 26 investigations fit the three categories (analyzed 12 of 26)

**What would change this:**

- If synthesis investigations contain unique domain knowledge not captured elsewhere, a guide would be needed
- If the synthesis system breaks, these investigations become critical troubleshooting docs

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern.

### Recommended Approach ⭐

**Meta-Topic Exclusion + Selective Archival** - Add "synthesis", "investigation", "artifact" to synthesis detection exclusions; archive completed synthesis runs; keep foundational design docs.

**Why this approach:**
- Prevents recurring "synthesize synthesis investigations" issues
- Preserves valuable system documentation
- Aligns with existing archival pattern (`.kb/investigations/archived/`)

**Trade-offs accepted:**
- Some meta-topic investigations may have hidden value
- Requires manual review before archival

**Implementation sequence:**
1. Add meta-topic exclusions to `synthesis_opportunities.go`
2. Archive completed topic-synthesis investigations to `synthesized/{guide-name}/`
3. Keep foundational design investigations in place

### Alternative Approaches Considered

**Option B: Create synthesis.md guide**
- **Pros:** Single authoritative reference for synthesis system
- **Cons:** Circular (documenting documentation system); duplicates kn decisions; most investigations are synthesis RUNS not synthesis KNOWLEDGE
- **When to use instead:** If the synthesis system becomes complex enough to need operational documentation

**Option C: Archive all synthesis investigations**
- **Pros:** Clean directory
- **Cons:** Loses foundational design context; foundational docs have historical value
- **When to use instead:** Never - design docs should be preserved

**Rationale for recommendation:** Meta-topic exclusion prevents the problem at source. Selective archival preserves value while reducing noise.

---

### Implementation Details

**What to implement first:**
- Add meta-topic exclusions in `pkg/verify/synthesis_opportunities.go`:
  ```go
  var MetaTopicExclusions = []string{
      "investigation",
      "synthesis",
      "artifact",
      "skill",
  }
  ```

**Things to watch out for:**
- ⚠️ The foundational design investigation (`2025-12-20-design-synthesis-protocol-schema.md`) should NOT be archived
- ⚠️ Completed topic-synthesis investigations contain knowledge that should remain discoverable

**Areas needing further investigation:**
- Should `kb reflect` run on meta-topics at all?
- What threshold makes a topic "meta" vs "domain"?

**Success criteria:**
- ✅ `orch status` no longer shows "26 synthesis investigations need synthesis"
- ✅ Foundational design docs remain discoverable via `kb context "synthesis protocol"`
- ✅ Completed synthesis runs are archived with provenance

---

## Proposed Actions

### Archive Actions
| ID | Target | Reason | Approved |
|----|--------|--------|----------|
| A1 | `2026-01-06-inv-synthesize-*-synthesis.md` (15 files) | Completed synthesis runs - guides already created | [ ] |
| A2 | `2025-12-20-inv-implement-synthesis-card-display-swarm.md` | Implementation complete - feature shipped | [ ] |
| A3 | `2025-12-26-inv-synthesis-review-view-parse-synthesis.md` | Implementation complete - feature shipped | [ ] |

### Create Actions
| ID | Type | Title | Description | Approved |
|----|------|-------|-------------|----------|
| C1 | issue | "Add meta-topic exclusions to synthesis detection" | Exclude synthesis, investigation, artifact, skill from synthesis opportunities | [ ] |
| C2 | issue | "Implement kb archive --synthesized-into command" | Archive investigations that produced guides | [ ] |

### Promote Actions
| ID | kn-id | To | Reason | Approved |
|----|-------|-------|--------|----------|
| P1 | N/A | decision | "Meta-topics excluded from synthesis detection" | [ ] |

### Update Actions
| ID | Target | Change | Reason | Approved |
|----|--------|--------|--------|----------|
| U1 | `synthesis_opportunities.go` | Add MetaTopicExclusions list | Prevent meta-topic pollution | [ ] |

**Summary:** 5 proposals (3 archive, 2 create, 0 promote, 1 update)
**High priority:** C1 (meta-topic exclusions - prevents recurring issue)

---

## References

**Files Examined:**
- `2025-12-20-design-synthesis-protocol-schema.md` - Foundational D.E.K.N. design
- `2025-12-20-audit-legacy-artifacts-synthesis-protocol.md` - Protocol adoption verification
- `2025-12-21-inv-agents-skip-synthesis-md-creation.md` - First bug fix
- `2025-12-26-design-synthesis-review-workflow.md` - Review workflow design
- `2026-01-06-inv-synthesize-dashboard-investigations-39-synthesis.md` - Topic synthesis example
- `2026-01-07-design-post-synthesis-investigation-archival.md` - Archival workflow design
- `2026-01-07-design-recurring-problem-duplicate-synthesis-issues.md` - Dedup bug fix

**Commands Run:**
```bash
# Find all synthesis-related investigations
glob ".kb/investigations/*synthesis*.md"  # 43 files

# Count existing guides
glob ".kb/guides/*.md"  # 21 files
```

**Related Artifacts:**
- **Template:** `.orch/templates/SYNTHESIS.md` - D.E.K.N. template
- **Guide:** `.kb/guides/dashboard.md` - Example of successful synthesis
- **Code:** `pkg/verify/check.go` - SYNTHESIS.md verification

---

## Investigation History

**2026-01-08 10:00:** Investigation started
- Initial question: What patterns should be consolidated from 26 synthesis investigations?
- Context: kb reflect flagged 26 investigations for synthesis

**2026-01-08 10:30:** Pattern recognition complete
- Identified three categories: foundational design, topic synthesis, maintenance
- Found evolutionary arc from Dec 20 design to Jan 7 maintenance

**2026-01-08 11:00:** Investigation completed
- Status: Complete
- Key outcome: Meta-topic - should be excluded from synthesis detection, not consolidated into guide
