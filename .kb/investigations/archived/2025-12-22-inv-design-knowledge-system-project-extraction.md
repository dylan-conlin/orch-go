<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Knowledge system needs inline lineage metadata over centralized registry for project extraction scenarios.

**Evidence:** Examined kb-cli codebase (projects.go, migrate.go, publish.go); analyzed 5 extraction scenarios; Session Amnesia principle requires self-describing artifacts.

**Knowledge:** Cross-project knowledge migration is rare but high-impact; inline metadata preserves local-first principle while enabling lineage; centralized registries create fragile dependencies.

**Next:** Implement `kb extract` command with inline lineage headers; add supersedes/extracted-from metadata fields to artifact templates.

**Confidence:** High (85%) - Design validated against principles; implementation not tested.

---

# Investigation: Knowledge System Support for Project Extraction and Refactoring

**Question:** How should the knowledge system handle project extraction/refactoring scenarios? When a component is extracted to a new repo (like skillc from orch-cli), how do decisions/investigations migrate, track lineage, update cross-references, and handle supersedes relationships?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** Agent og-arch-design-knowledge-system-22dec
**Phase:** Complete
**Next Step:** None (design complete)
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Current kb infrastructure handles vertical migration but not horizontal

**Evidence:** Examined kb-cli codebase:
- `kb migrate` (migrate.go:39-145): Moves artifacts from `.orch/` to `.kb/` within the same project (vertical: legacy→modern structure)
- `kb publish` (publish.go:66-159): Copies artifacts from local `.kb/` to global `~/.kb/` (vertical: project→global promotion)
- `kb projects` (projects.go:23-258): Registry of projects with paths, but no relationships between projects

Missing capabilities:
- No artifact migration between projects (horizontal)
- No lineage tracking between projects
- No cross-reference updating when artifacts move
- No supersedes relationship tracking

**Source:** 
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/migrate.go:39-145`
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/publish.go:66-159`
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/projects.go:23-258`

**Significance:** The current system assumes knowledge stays with its originating project. Project extraction breaks this assumption.

---

### Finding 2: Five concrete extraction scenarios exist

**Evidence:** Analyzed the task description and real-world examples:

| Scenario | Example | What happens to knowledge |
|----------|---------|---------------------------|
| **Component extraction** | skillc extracted from orch-cli | Skill-related decisions should migrate to skillc |
| **Lineage tracking** | skillc's origin is orch-cli | Need to record "this project was extracted from X" |
| **Cross-reference updates** | Decision cites "see orch-cli investigation" | Links become broken or misleading |
| **Supersedes relationship** | skillc decision replaces orch-cli decision | Old decision should note "superseded by skillc" |
| **Deprecated project** | orch-cli Python → orch-go | What happens to 200+ investigations in orch-cli? |

**Source:** 
- Task prompt scenarios (1-5)
- orch-knowledge/.kb/ (137 investigations, 85 decisions)
- orch-cli/.kb/ (212 investigations)
- skillc/.kb/ (recent extraction)

**Significance:** These are distinct scenarios requiring different handling. A unified approach must address all five.

---

### Finding 3: Session Amnesia principle dictates inline metadata over centralized tracking

**Evidence:** From `~/.kb/principles.md`:

> "Every pattern in this system compensates for Claude having no memory between sessions."
> 
> "State must externalize to files (workspaces, artifacts, decisions)"
> 
> Self-Describing Artifacts: "If an agent encounters this file with no memory of how it was created, can they modify it correctly?"

Applied to extraction scenarios:
- **Centralized registry**: Agent in skillc must query global registry to understand lineage → requires knowing to check, finding registry, understanding schema
- **Inline metadata**: Agent reads skillc decision, sees `extracted-from: orch-cli` header → immediate context
- **Session amnesia test**: Which helps a fresh Claude understand where this knowledge came from?

**Source:** `/Users/dylanconlin/.kb/principles.md:14-68`

**Significance:** Inline metadata aligns with foundational principles. Centralized tracking violates Session Amnesia and Self-Describing Artifacts.

---

### Finding 4: Git provides proven patterns for lineage in distributed systems

**Evidence:** Git's approach to repository history during splits/merges:
- `git filter-branch` / `git subtree split`: Extracts subdirectory with history
- **Lineage preserved in commits**: Each commit carries its parent references
- **No central registry**: History travels with the repository
- **Cross-references**: Commits reference other commits by SHA (stable identifier)

Applied to knowledge system:
- Artifacts could carry lineage in headers (like commit messages)
- No need for central registry if metadata is inline
- Cross-references need stable identifiers (path + project name)

**Source:** Git documentation, observed `git subtree` behavior

**Significance:** Git successfully handles distributed lineage without central coordination. Knowledge system can follow this pattern.

---

### Finding 5: Existing artifact templates have unused metadata fields

**Evidence:** Investigation template has metadata block:
```markdown
**Started:** [YYYY-MM-DD]
**Updated:** [YYYY-MM-DD]
**Owner:** [Owner name or team]
**Phase:** [Investigating/Synthesizing/Complete]
```

Decision template has:
```markdown
---
date: "YYYY-MM-DD"
status: "Accepted"
---
```

Neither has:
- `extracted-from:` (origin project)
- `supersedes:` (what this replaces)
- `superseded-by:` (what replaced this)
- `original-path:` (path before migration)

**Source:** 
- `/Users/dylanconlin/.kb/templates/investigation.md`
- orch-knowledge/.kb/decisions/*.md (sampled)

**Significance:** Adding lineage fields to existing templates is low-friction. No schema breaking changes needed.

---

## Synthesis

**Key Insights:**

1. **Inline metadata over centralized tracking** - Session Amnesia principle demands that artifacts be self-describing. A fresh agent reading a skillc decision should see its lineage without querying external registries. This aligns with local-first and git-inspired design.

2. **Extraction is rare but high-impact** - Project extraction happens infrequently (maybe 2-3 times/year), but when it happens, knowledge migration is critical. The solution should be low-maintenance when not in use, high-value when needed.

3. **Five scenarios reduce to two operations** - All five scenarios can be addressed by two primitives:
   - `kb extract`: Copy artifacts between projects with lineage headers
   - `kb supersede`: Mark artifacts as superseded with forward reference

4. **Cross-references don't need automated updating** - When references break, agents discover them during use and can fix manually. Automated link rewriting is complex and error-prone. Better to have clear lineage that helps agents understand the relationship.

**Answer to Investigation Question:**

**How should knowledge migrate when extracting components?**

Use `kb extract <artifact> --to <project>` to copy artifacts with inline lineage:
```markdown
<!-- Extracted from: orch-cli/.kb/decisions/2025-11-15-skill-template-system.md -->
<!-- Extraction date: 2025-12-22 -->
<!-- Original created: 2025-11-15 -->
```

**How to track lineage?**

Add `lineage:` block to project's `.kb/manifest.yaml`:
```yaml
lineage:
  extracted-from: orch-cli
  extraction-date: 2025-12-22
  reason: "Skill compilation is distinct concern from orchestration"
```

Plus inline headers in migrated artifacts.

**How to handle cross-reference updates?**

Don't auto-update. Add forward reference in source:
```markdown
<!-- NOTE: Related decisions moved to skillc - see skillc/.kb/decisions/ -->
```

Agents discovering broken references can trace via lineage headers.

**How to mark supersedes relationships?**

Two-way linking:
```markdown
# In old decision (orch-cli)
<!-- SUPERSEDED BY: skillc/.kb/decisions/2025-12-22-skillc-artifact-scope.md -->

# In new decision (skillc)  
supersedes: orch-cli/.kb/decisions/2025-11-10-skill-build-approach.md
```

**How to handle deprecated projects?**

Add `.kb/DEPRECATED.md` marker file:
```markdown
# This project's knowledge base is deprecated

**Reason:** Replaced by orch-go
**Successor project:** orch-go
**Migration date:** 2025-12-22
**Migration scope:** Active development moved; historical artifacts remain for reference

Artifacts here are historical. For current decisions, see orch-go/.kb/
```

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Design aligns with established principles (Session Amnesia, Self-Describing Artifacts, Local-First). Pattern mirrors git's proven approach. However, no implementation exists to validate ergonomics.

**What's certain:**

- ✅ Inline metadata > centralized registry (principles-based)
- ✅ Five scenarios identified cover the problem space
- ✅ Git-inspired lineage pattern is proven at scale
- ✅ Template changes are backwards-compatible

**What's uncertain:**

- ⚠️ Ergonomics of `kb extract` command (needs implementation)
- ⚠️ How often cross-references actually break in practice
- ⚠️ Whether manifest.yaml lineage is sufficient vs artifact-level lineage

**What would increase confidence to Very High (95%+):**

- Implementing `kb extract` and testing on real extraction (skillc retrospective)
- Running extraction workflow on orch-cli → orch-go migration
- Agent testing: Can fresh Claude follow lineage to find relevant knowledge?

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Inline Lineage Metadata** - Add lineage tracking via inline artifact headers and project manifest, with extraction and supersede commands.

**Why this approach:**
- Aligns with Session Amnesia principle (self-describing artifacts)
- Follows Local-First design (no external registry dependencies)
- Git-inspired pattern proven at massive scale
- Low-friction when not extracting, high-value when needed

**Trade-offs accepted:**
- No automated cross-reference updating (manual discovery acceptable given rarity)
- Duplicate metadata (artifact header + manifest) for redundancy
- Requires discipline to mark supersedes relationships

**Implementation sequence:**
1. **Add lineage fields to templates** - `extracted-from:`, `supersedes:`, `superseded-by:` in YAML frontmatter
2. **Create `.kb/manifest.yaml`** - Project-level metadata including lineage
3. **Implement `kb extract <artifact> --to <project>`** - Copy with lineage headers
4. **Implement `kb supersede <old> --by <new>`** - Mark replacement relationships
5. **Add DEPRECATED.md pattern** - Project-level deprecation marker

### Alternative Approaches Considered

**Option B: Centralized Cross-Reference Registry**
- **Pros:** Single source of truth for all relationships; enables automated link rewriting
- **Cons:** Violates Session Amnesia (requires registry lookup); single point of failure; sync complexity; fragile to registry corruption
- **When to use instead:** If cross-project queries are frequent (not our case)

**Option C: Git Submodule for Shared Knowledge**
- **Pros:** Git handles lineage natively; shared knowledge stays in sync
- **Cons:** Couples projects at git level; complex merge conflicts; shared knowledge is actually rare
- **When to use instead:** If multiple projects genuinely share evolving knowledge (not extraction scenario)

**Option D: Do Nothing (Manual Management)**
- **Pros:** Zero implementation cost; works today
- **Cons:** Knowledge gets orphaned; lineage lost; fresh agents confused
- **When to use instead:** If extractions are truly rare (but orch-cli→orch-go migration proves otherwise)

**Rationale for recommendation:** Inline metadata is the only approach that satisfies Session Amnesia while handling all five scenarios. Centralized registry adds fragility without proportional benefit. Git submodules solve a different problem (shared evolving knowledge vs extracted historical knowledge).

---

### Implementation Details

**What to implement first:**
- Template changes (low-effort, immediate value)
- `kb extract` command (enables first migration)
- DEPRECATED.md pattern (needed for orch-cli)

**Things to watch out for:**
- ⚠️ Don't over-engineer automated cross-reference updates - manual discovery is acceptable
- ⚠️ Extraction should COPY, not MOVE - original stays for historical reference
- ⚠️ Supersedes is two-way - update both old and new artifacts

**Areas needing further investigation:**
- Should `kb context` search across lineage relationships?
- Should `kb reflect` surface broken cross-references?
- How to handle circular supersedes (unlikely but possible)?

**Success criteria:**
- ✅ `kb extract` copies artifact with lineage headers
- ✅ Fresh agent in skillc can trace decision origin to orch-cli
- ✅ orch-cli can be marked deprecated with clear successor
- ✅ `kb context "skill template"` finds both old (orch-cli) and new (skillc) decisions

---

### File Targets

**Templates to modify:**
- `~/.kb/templates/investigation.md` - Add lineage fields to frontmatter
- `~/.kb/templates/decision.md` - Add supersedes/superseded-by fields

**New files to create:**
- `.kb/manifest.yaml` schema definition
- `.kb/DEPRECATED.md` template

**Commands to implement (in kb-cli):**
- `kb extract <artifact> --to <project>` - Extract with lineage
- `kb supersede <old> --by <new>` - Mark replacement
- `kb deprecated` - Mark project as deprecated

---

### Acceptance Criteria

- [ ] Investigation template includes `extracted-from:` field (optional)
- [ ] Decision template includes `supersedes:` and `superseded-by:` fields (optional)
- [ ] `kb extract` command copies artifact and adds lineage header
- [ ] `kb supersede` command updates both old and new artifacts
- [ ] DEPRECATED.md pattern documented and templateized
- [ ] `.kb/manifest.yaml` can record project-level lineage
- [ ] `kb context` can follow supersedes relationships

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/projects.go` - Project registry implementation
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/migrate.go` - .orch→.kb migration
- `/Users/dylanconlin/Documents/personal/kb-cli/cmd/kb/publish.go` - Local→global publishing
- `/Users/dylanconlin/.kb/principles.md` - Foundational principles (Session Amnesia, Self-Describing)
- `/Users/dylanconlin/.kb/projects.json` - Current project registry (17 projects)
- `/Users/dylanconlin/Documents/personal/orch-go/.kb/investigations/2025-12-22-inv-investigate-skillc-vs-orch-knowledge.md` - Related investigation on skillc vs orch-knowledge

**Commands Run:**
```bash
# Check kb CLI capabilities
kb --help
kb projects --help
kb migrate --help
kb publish --help

# Check project registry
cat ~/.kb/projects.json

# Count artifacts in relevant projects
ls -la ~/orch-knowledge/.kb/decisions/ | wc -l  # 85 decisions
ls -la ~/orch-knowledge/.kb/investigations/ | wc -l  # 137 investigations
ls -la ~/Documents/personal/orch-cli/.kb/investigations/ | wc -l  # 212 investigations
```

**Related Artifacts:**
- **Investigation:** `orch-go/.kb/investigations/2025-12-22-inv-investigate-skillc-vs-orch-knowledge.md` - Clarified skillc vs orch-cli relationship
- **Principles:** `~/.kb/principles.md` - Session Amnesia, Self-Describing Artifacts

---

## Investigation History

**2025-12-22 ~08:30:** Investigation started
- Initial question: How to handle knowledge migration during project extraction?
- Context: Spawned as architect to design knowledge system support for extraction scenarios

**2025-12-22 ~08:45:** Found existing kb infrastructure
- Discovered kb migrate, publish, projects commands
- Identified gap: no horizontal (cross-project) migration

**2025-12-22 ~09:00:** Analyzed five extraction scenarios
- Component extraction, lineage tracking, cross-references, supersedes, deprecated projects
- Recognized these are distinct but related problems

**2025-12-22 ~09:15:** Applied Session Amnesia principle
- Concluded inline metadata > centralized registry
- Git's distributed lineage model as pattern

**2025-12-22 ~09:30:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: Inline lineage metadata with extract/supersede commands recommended
