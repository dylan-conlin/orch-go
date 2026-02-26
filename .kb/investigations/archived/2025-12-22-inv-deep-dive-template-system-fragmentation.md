## Summary (D.E.K.N.)

**Delta:** Found 6 distinct template systems with significant divergence - same templates exist in multiple places with different content (e.g., ~/.kb/templates/INVESTIGATION.md has D.E.K.N. summary, orch-knowledge version doesn't).

**Evidence:** Diff between ~/.kb/templates/INVESTIGATION.md and orch-knowledge/skills/src/worker/investigation/templates/investigation.md shows 110+ lines of divergence. Similar divergence exists for SYNTHESIS.md (~100 line diff).

**Knowledge:** Template systems evolved independently without synchronization. No single source of truth exists for artifact templates. skillc handles CLAUDE.md compilation but NOT artifact templates.

**Next:** Create beads issue to consolidate artifact templates. Recommend kb-cli as template owner for investigation/decision/guide templates, orch-go for spawn-time templates (SYNTHESIS, FAILURE_REPORT, SPAWN_CONTEXT).

**Confidence:** High (85%) - examined actual code in all 6 systems, verified with diffs.

---

# Investigation: Template System Fragmentation Deep Dive

**Question:** What template systems exist in the orch ecosystem, what are their relationships, and should skillc own all template compilation?

**Started:** 2025-12-22
**Updated:** 2025-12-22
**Owner:** investigation agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete
**Confidence:** High (85%)

---

## Findings

### Finding 1: Six Distinct Template Systems Identified

**Evidence:** Found 6 separate template sources with different purposes:

| # | System | Location | Purpose | Outputs |
|---|--------|----------|---------|---------|
| 1 | orch-go spawn | `pkg/spawn/context.go` | SPAWN_CONTEXT.md generation | Embedded Go templates for SPAWN_CONTEXT, SYNTHESIS, FAILURE_REPORT |
| 2 | kb-cli hardcoded | `kb-cli/cmd/kb/create.go` | `kb create` fallback templates | Embedded const strings for INVESTIGATION, DECISION, GUIDE |
| 3 | ~/.kb/templates/ | User home dir | `kb create` override templates | INVESTIGATION.md, DECISION.md, GUIDE.md, SPAWN_PROMPT.md, etc. |
| 4 | orch-go .orch/templates/ | Per-project | Project-level templates | SYNTHESIS.md, FAILURE_REPORT.md, SESSION_HANDOFF.md |
| 5 | orch-knowledge skills/src/ | Skills source repo | Skill-embedded templates | investigation.md template in skills/src/worker/investigation/templates/ |
| 6 | skillc compiler | `skillc/pkg/compiler/` | CLAUDE.md compilation | Compiles .skillc/ sources into CLAUDE.md files |

**Source:** 
- `kb-cli/cmd/kb/create.go:14-111` (hardcoded templates)
- `orch-go/pkg/spawn/context.go:14-611` (embedded templates)
- `skillc/pkg/compiler/compiler.go:1-240` (compiler logic)
- `~/.kb/templates/` directory listing
- `orch-knowledge/skills/src/worker/investigation/templates/`

**Significance:** Template fragmentation is worse than expected - 6 distinct systems with overlapping concerns. The same artifact type (investigation) has templates in 3 different places.

---

### Finding 2: Significant Content Divergence Between Same-Named Templates

**Evidence:** Diff analysis reveals:

1. **INVESTIGATION.md divergence (110+ lines difference):**
   - `~/.kb/templates/INVESTIGATION.md`: 234 lines, includes D.E.K.N. summary, Implementation Recommendations section, structured confidence assessment
   - `orch-knowledge/.../investigation.md`: 124 lines, simpler structure, no D.E.K.N., references deprecated meta-orchestration paths

2. **SYNTHESIS.md divergence (100+ lines difference):**
   - `~/.orch/templates/SYNTHESIS.md`: 20 lines, **DEPRECATED** with warning message
   - `orch-go/.orch/templates/SYNTHESIS.md`: 122 lines, full template with D.E.K.N.-inspired sections

**Source:**
```bash
diff -u ~/.kb/templates/INVESTIGATION.md orch-knowledge/.../investigation.md
diff -u ~/.orch/templates/SYNTHESIS.md orch-go/.orch/templates/SYNTHESIS.md
```

**Significance:** Templates have evolved independently in different repos. The "source of truth" isn't clear - different systems have different versions of the same template.

---

### Finding 3: Template Loading Hierarchy Creates Confusion

**Evidence:** kb-cli's `loadTemplate()` function (create.go:199-212) implements fallback:
1. Try `~/.kb/templates/{name}.md`
2. Fall back to hardcoded const

orch-go's `EnsureSynthesisTemplate()` (context.go:237-260):
1. Check if `.orch/templates/SYNTHESIS.md` exists
2. If not, create from embedded `DefaultSynthesisTemplate`

orch-cli's `load_spawn_prompt_template()` (spawn_prompt.py:692-760):
1. Try `~/.orch/templates/SPAWN_PROMPT.md`
2. Try local `templates-src/SPAWN_PROMPT.md`
3. Fall back to hardcoded string

**Source:**
- `kb-cli/cmd/kb/create.go:199-212`
- `orch-go/pkg/spawn/context.go:237-260`
- `orch-cli/src/orch/spawn_prompt.py:692-760`

**Significance:** Each system has its own fallback chain. No coordination between them. User overrides in `~/.kb/templates/` are separate from `~/.orch/templates/` even though both serve similar purposes.

---

### Finding 4: skillc Only Handles CLAUDE.md Compilation

**Evidence:** `skillc/pkg/compiler/compiler.go` shows:
- Reads `.skillc/` directory
- Loads `skill.yaml` manifests
- Concatenates source `.md` files
- Outputs CLAUDE.md with header

skillc does NOT:
- Manage artifact templates (investigation, decision, synthesis)
- Handle user-level template overrides
- Compile templates for kb/orch usage

**Source:** `skillc/pkg/compiler/compiler.go:21-114`

**Significance:** skillc's scope is narrow by design - CLAUDE.md assembly from skill sources. It's NOT a general-purpose template compiler and shouldn't become one.

---

### Finding 5: orch-cli Has Legacy Python Template System

**Evidence:** `orch-cli/src/orch/spawn_prompt.py` contains:
- `load_spawn_prompt_template()` function
- `build_spawn_prompt()` 1400+ line function building prompts
- Fallback template strings
- Template filtering for feature-impl phases

This is separate from orch-go's template system in `pkg/spawn/context.go`, which appears to be a reimplementation/port.

**Source:** `orch-cli/src/orch/spawn_prompt.py`

**Significance:** orch-go is a Go rewrite. Template generation exists in both Python and Go codebases. This is expected during migration but represents temporary duplication.

---

## Synthesis

**Key Insights:**

1. **Organic Growth, Not Design** - Template systems grew independently as each tool needed templates. No central coordination led to 6 separate systems.

2. **Two Distinct Categories** - Templates fall into two buckets:
   - **Artifact templates** (investigation, decision, guide) - used by `kb create`
   - **Spawn-time templates** (SYNTHESIS, SPAWN_CONTEXT, FAILURE_REPORT) - used during agent spawn

3. **skillc Is Wrong Tool for This** - skillc's purpose is CLAUDE.md compilation from skill sources. Expanding it to artifact templates would conflate two different concerns.

**Answer to Investigation Question:**

Six template systems exist:
1. **orch-go spawn embedded** - SPAWN_CONTEXT, SYNTHESIS, FAILURE_REPORT
2. **kb-cli hardcoded** - Investigation, decision, guide fallbacks
3. **~/.kb/templates/** - User-level kb overrides
4. **Per-project .orch/templates/** - Project-level orch templates
5. **orch-knowledge skill templates** - Skill-embedded artifact templates
6. **skillc compiler** - CLAUDE.md assembly only

**Relationships:** Mostly independent. kb-cli uses ~/.kb/templates/ for overrides. orch-go copies embedded templates to .orch/templates/ on first spawn. orch-knowledge skill templates are loaded directly into skills, not via kb/orch systems.

**Should skillc own template compilation?** No. skillc's purpose (CLAUDE.md from skill sources) is distinct from artifact template management. Conflating them would create a muddled abstraction.

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**

Examined actual source code in all 6 systems. Ran diffs to verify divergence. Traced template loading paths through code.

**What's certain:**

- ✅ 6 distinct template systems exist
- ✅ INVESTIGATION.md and SYNTHESIS.md have significant divergence between locations
- ✅ skillc only handles CLAUDE.md compilation
- ✅ Each system has independent fallback chains

**What's uncertain:**

- ⚠️ Whether Dylan intended ~/.kb/templates/ as the canonical source (it appears to be most evolved)
- ⚠️ Whether orch-knowledge skill templates should be retired in favor of kb templates
- ⚠️ Exact consolidation path without breaking existing workflows

**What would increase confidence to 95%:**

- Interview Dylan about intended design
- Test full spawn → completion cycle with each template source
- Verify which templates are actually used at runtime vs. which are stale

---

## Implementation Recommendations

**Purpose:** Consolidate to clear ownership while minimizing disruption.

### Recommended Approach: Domain-Based Ownership

**Two owners, clear boundaries:**

1. **kb-cli owns artifact templates** (investigation, decision, guide)
   - Source: `~/.kb/templates/` (user-level, shared across projects)
   - kb-cli hardcodes become true fallbacks
   - Retire orch-knowledge/skills/src/.../templates/ (duplicate)

2. **orch-go owns spawn-time templates** (SYNTHESIS, SPAWN_CONTEXT, FAILURE_REPORT)
   - Source: Embedded in pkg/spawn/context.go
   - Copied to .orch/templates/ per-project on first spawn
   - Per-project customization still works

**Why this approach:**
- Matches current usage patterns (kb create uses kb templates, orch spawn uses spawn templates)
- Clear ownership prevents drift
- No changes to skillc needed

**Trade-offs accepted:**
- Two template owners instead of one (but domains are distinct)
- ~/.kb/templates/ and .orch/templates/ remain separate (but serve different purposes)

**Implementation sequence:**
1. Document ownership model in a decision record
2. Sync ~/.kb/templates/INVESTIGATION.md as source of truth → update kb-cli hardcoded fallback
3. Retire orch-knowledge/skills/src/worker/investigation/templates/ (stale)
4. Verify orch-go embedded templates match .orch/templates/ in this project

### Alternative Approaches Considered

**Option B: skillc owns all templates**
- **Pros:** Single source of truth, unified build
- **Cons:** Conflates skill compilation with artifact templates; skillc not designed for this
- **When to use instead:** If templates needed preprocessing/composition that skillc provides

**Option C: Create new template-cli tool**
- **Pros:** Clean separation, purpose-built
- **Cons:** Yet another tool; overhead not justified for simple templates
- **When to use instead:** If template needs become complex (inheritance, composition)

**Rationale for recommendation:** Domain-based ownership aligns with existing usage patterns and requires minimal code changes.

---

## Self-Review

- [x] Real test performed (ran diffs between template files)
- [x] Conclusion from evidence (not speculation)
- [x] Question answered
- [x] File complete

**Self-Review Status:** PASSED

---

## Discovered Work

- **bd create "Sync ~/.kb/templates/INVESTIGATION.md to kb-cli hardcoded fallback"** - kb-cli's const investigationTemplate is 25 lines, ~/.kb/templates/ version is 234 lines. Should sync.
- **bd create "Retire orch-knowledge investigation template"** - skills/src/worker/investigation/templates/investigation.md is stale (124 lines, no D.E.K.N.).
- **bd create "Verify orch-go embedded SYNTHESIS matches .orch/templates/"** - May have diverged.

---

## References

**Files Examined:**
- `kb-cli/cmd/kb/create.go` - kb create command with embedded templates
- `orch-go/pkg/spawn/context.go` - Spawn context generation with embedded templates
- `skillc/pkg/compiler/compiler.go` - CLAUDE.md compiler
- `~/.kb/templates/` - User-level kb templates
- `orch-knowledge/skills/src/worker/investigation/templates/` - Skill-embedded templates
- `orch-cli/src/orch/spawn_prompt.py` - Python template generation

**Commands Run:**
```bash
# Compare investigation templates
diff -u ~/.kb/templates/INVESTIGATION.md orch-knowledge/skills/src/worker/investigation/templates/investigation.md

# Compare synthesis templates
diff -u ~/.orch/templates/SYNTHESIS.md orch-go/.orch/templates/SYNTHESIS.md

# List all template directories
ls -la ~/.kb/templates/
ls -la ~/.orch/templates/
ls -la orch-go/.orch/templates/
```

---

## Investigation History

**2025-12-22 07:00:** Investigation started
- Initial question: What template systems exist and should skillc own them?
- Context: Spawned from orchestrator investigating template fragmentation

**2025-12-22 07:30:** Found 6 template systems
- More fragmentation than expected
- Significant divergence between same-named templates

**2025-12-22 08:00:** Investigation completed
- Final confidence: High (85%)
- Status: Complete
- Key outcome: 6 systems, domain-based ownership recommended, skillc should NOT own artifact templates
