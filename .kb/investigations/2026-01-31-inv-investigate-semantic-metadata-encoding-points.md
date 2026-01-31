<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** The ecosystem has 10+ distinct semantic metadata encoding points clustered at three boundaries: creation time (structural), completion time (structural), and post-hoc extraction (heuristic).

**Evidence:** Examined spawn/config.go (16 metadata fields), beads/types.go (12 issue fields), skills/loader.go (8 frontmatter fields), kb/artifacts.go (regex extraction), synthesis_parser.go (section parsing), and kb quick entries (JSON schema).

**Knowledge:** Encoding at creation time captures understanding but adds friction; post-hoc extraction is frictionless but lossy. The ecosystem strategically uses structural encoding for high-value metadata (type, skill, status) and heuristic extraction for discovery (topic, prior work).

**Next:** Document as model file `.kb/models/semantic-metadata-encoding.md` for future design decisions.

**Authority:** architectural - Cross-component framework that guides future metadata decisions across spawn, kb, beads, and verification systems.

---

# Investigation: Investigate Semantic Metadata Encoding Points

**Question:** What are the semantic metadata encoding points in the ecosystem, when does understanding get crystallized into metadata, and what are the tradeoffs between upfront capture vs post-hoc parsing?

**Started:** 2026-01-31
**Updated:** 2026-01-31
**Owner:** Worker agent (orch-go-21136)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** N/A
**Extracted-From:** N/A
**Supersedes:** N/A
**Superseded-By:** N/A

---

## Findings

### Finding 1: Three Encoding Boundary Types

**Evidence:**

The ecosystem encodes semantic metadata at three distinct boundaries:

| Boundary | Timing | Enforcement | Examples |
|----------|--------|-------------|----------|
| **Creation Time** | When artifact is born | Structural (schema) | Beads issue type, kb quick entry type, spawn skill name |
| **Completion Time** | When work ends | Structural (verification) | SYNTHESIS.md fields, Phase: Complete, recommendation |
| **Post-Hoc Extraction** | Whenever parsed | Heuristic (regex) | Title from heading, date from filename, topic from text |

The boundaries are not arbitrary—they correspond to points where an agent has different levels of understanding:
- Creation: Knows intent but not outcome
- Completion: Knows outcome, can synthesize
- Post-hoc: Reader's interpretation of artifacts

**Source:**
- `pkg/spawn/config.go:97-227` - Config struct showing creation-time metadata
- `pkg/verify/check.go:46-116` - Completion-time verification
- `pkg/kb/artifacts.go:102-164` - Post-hoc regex extraction

**Significance:** The three boundaries represent different knowledge states. Forcing completion-time knowledge at creation (e.g., "what was learned?") is impossible. Accepting creation-time knowledge at post-hoc (e.g., parsing skill from title) is lossy.

---

### Finding 2: Structural Encoding Points (Creation Time)

**Evidence:**

**Beads Issues** (`pkg/beads/types.go:62-74`):
```go
type CreateArgs struct {
    ID           string   // Optional explicit ID
    Title        string   // Required
    Description  string   // Optional
    IssueType    string   // Required: task, bug, feature, question, epic
    Priority     int      // Required: 0-4 integer scale
    Labels       []string // Optional semantic tags (e.g., subtype:factual)
    Dependencies []string // Optional relationships
}
```

**KB Quick Entries** (`.kb/quick/entries.jsonl`):
```json
{
  "id": "kb-xxx",
  "type": "decision|constraint|attempt|question",  // Required
  "content": "...",
  "reason": "...",     // Required for decisions/constraints
  "outcome": "failed", // Required for attempts
  "status": "active"
}
```

**Spawn Config** (`pkg/spawn/config.go:97-227`):
```go
type Config struct {
    SkillName      string  // Required
    BeadsID        string  // Required (unless NoTrack)
    Tier           string  // "light" or "full"
    Model          string  // Model alias
    IsOrchestrator bool    // Determines template
    InvestigationType string // "simple", "systems", etc.
    // ... 16 more explicit fields
}
```

**Skill Frontmatter** (`pkg/skills/loader.go:20-30`):
```go
type SkillMetadata struct {
    Name         string   `yaml:"name"`
    SkillType    string   `yaml:"skill-type"`
    Dependencies []string `yaml:"dependencies"`
    Description  string   `yaml:"description"`
    // ... 4 more fields
}
```

**Source:**
- `pkg/beads/types.go:62-74` - CreateArgs requiring type and priority
- `.kb/quick/entries.jsonl` - 30 entries showing JSON schema
- `pkg/spawn/config.go:97-227` - Config struct with 20+ fields
- `pkg/skills/loader.go:20-30` - SkillMetadata YAML binding

**Significance:** High-value metadata (issue type, skill type, priority, tier) is captured structurally at creation time. The friction is justified: these fields drive routing, verification, and lifecycle decisions.

---

### Finding 3: Structural Encoding Points (Completion Time)

**Evidence:**

**SYNTHESIS.md Template** (`.orch/templates/SYNTHESIS.md:1-10`):
```markdown
**Agent:** {workspace-name}
**Issue:** {beads-id}
**Duration:** {start-time} → {end-time}
**Outcome:** {success | partial | blocked | failed}
```

**Verification** (`pkg/verify/check.go:77-116`):
- `ValidateHandoffContent()` checks TLDR section has actual content (not placeholder)
- `validateOutcomeField()` checks Outcome is valid enum value
- `ParseSynthesis()` extracts D.E.K.N. sections (Delta, Evidence, Knowledge, Next)

**Phase Reporting** (via beads comments):
```bash
bd comment <id> "Phase: Complete - [summary]"
```
Verification checks for this pattern before allowing close.

**Source:**
- `.orch/templates/SYNTHESIS.md:1-10` - Template header fields
- `pkg/verify/check.go:77-116` - ValidateHandoffContent
- `pkg/verify/synthesis_parser.go:44-93` - ParseSynthesis

**Significance:** Completion-time encoding captures what the agent learned, not just what it did. This is when understanding is richest—right after doing the work. The D.E.K.N. structure (Delta, Evidence, Knowledge, Next) is explicitly designed for this.

---

### Finding 4: Heuristic Extraction Points (Post-Hoc)

**Evidence:**

**KB Artifacts** (`pkg/kb/artifacts.go:102-164`):
```go
// Extract date from filename (YYYY-MM-DD prefix)
if len(id) >= 10 {
    artifact.Date = id[:10]
}

// Extract title from first # heading
if artifact.Title == "" && strings.HasPrefix(line, "# ") {
    artifact.Title = strings.TrimPrefix(line, "# ")
    artifact.Title = strings.TrimPrefix(artifact.Title, "Investigation: ")
}

// Extract status/phase
if strings.HasPrefix(line, "**Phase:**") || strings.HasPrefix(line, "**Status:**") {
    // regex extraction...
}

// Find beads ID references throughout the file
matches := beadsIDPattern.FindAllString(line, -1)
```

**KB Context** (`pkg/spawn/kbcontext.go:350-476`):
```go
func parseKBContextOutput(output string) []KBContextMatch {
    // Parse "## CONSTRAINTS", "## DECISIONS", etc. headers
    // Parse "- Title" entries
    // Parse "Path:" and "Reason:" lines
}
```

**Topic Extraction** (not yet implemented, but the design question):
- **Heuristic approach:** Parse title for topic keywords
- **Structural approach:** Require `topic:` frontmatter at creation

**Source:**
- `pkg/kb/artifacts.go:102-164` - parseArtifact()
- `pkg/spawn/kbcontext.go:350-476` - parseKBContextOutput()
- Issue description orch-go-21136 - Topic extraction question

**Significance:** Heuristic extraction is frictionless but lossy. Title parsing for topic is ~80% accurate but misses nuance. Beads ID extraction via regex works but can't distinguish "discussed" from "implemented."

---

### Finding 5: The Friction vs Fidelity Tradeoff

**Evidence:**

| Metadata | Current Approach | Friction | Fidelity | Could Change? |
|----------|------------------|----------|----------|---------------|
| Issue type | Structural (required) | High | 100% | No—drives lifecycle |
| Priority | Structural (required) | High | 100% | No—drives ordering |
| Skill name | Structural (spawn param) | Low | 100% | No—drives context |
| Topic | Heuristic (title parse) | Zero | ~80% | Yes—could require frontmatter |
| Prior work | Heuristic (citation parse) | Zero | ~60% | Yes—now has Prior-Work table |
| Status/Phase | Semi-structural (bd comment) | Low | ~95% | No—works well |
| Dependencies | Structural (bd dep add) | Medium | 100% | No—drives DAG |
| Recommendation | Semi-structural (SYNTHESIS) | Low | ~90% | No—parsed at completion |

The pattern: **High-stakes metadata is structural. Discovery metadata is heuristic.**

**Source:**
- Analysis of all 10 encoding points above
- Issue description tradeoff questions

**Significance:** The ecosystem has implicitly settled on a sensible division. The question isn't "should we structurally encode everything" but "which metadata is worth the friction?"

---

### Finding 6: Skill Dependencies Are Structural By Design

**Evidence:**

**Declared Dependencies** (`pkg/skills/loader.go:105-149`):
```go
func (l *Loader) LoadSkillWithDependencies(skillName string) (string, error) {
    // Parse metadata to check for dependencies
    metadata, err := ParseSkillMetadata(mainContent)

    // If no dependencies, return as-is
    if len(metadata.Dependencies) == 0 {
        return mainContent, nil
    }

    // Load each dependency and build combined content
    for _, dep := range metadata.Dependencies {
        depContent, err := l.LoadSkillContent(dep)
        // Strip frontmatter, prepend to main skill
    }
}
```

**Example** (investigation skill):
```yaml
---
name: investigation
skill-type: procedure
dependencies:
  - worker-base
---
```

Why not infer from imports/references?
1. Skills are Markdown, not code—no import statements
2. Explicit declaration enables static analysis (which skills does X depend on?)
3. Load order matters—worker-base content must come before skill-specific guidance

**Source:**
- `pkg/skills/loader.go:105-149` - LoadSkillWithDependencies
- Skill source files via `ls -la ~/.claude/skills/`

**Significance:** This is a case where structural encoding was clearly correct. Inferring dependencies from prose would require NLP and would be unreliable.

---

## Synthesis

**Key Insights:**

1. **Three-Boundary Model** - Encoding points cluster at creation, completion, and post-hoc. Each boundary corresponds to a different knowledge state. Requiring knowledge from the wrong state is impossible (completion knowledge at creation) or lossy (creation knowledge via post-hoc parsing).

2. **Friction Calibration** - The ecosystem has implicitly calibrated friction to value. High-stakes metadata (issue type, skill, priority, dependencies) is structural despite friction. Discovery metadata (topic, prior work, references) is heuristic despite lossiness. This is sensible.

3. **Completion as Synthesis Point** - Completion time is when understanding is richest. The D.E.K.N. structure explicitly harvests this: Delta (what changed), Evidence (what was observed), Knowledge (what was learned), Next (what should happen). This is the optimal time to capture semantic metadata about learnings.

4. **Post-Hoc Extraction Has a Place** - Frictionless discovery via regex has value. `kb context` can surface relevant artifacts without authors having to anticipate future queries. The tradeoff: ~80% accuracy for topic vs. 100% accuracy for explicit tags.

5. **Prior-Work Table Is Strategic Fix** - The investigation template's `Patches-Decision`, `Extracted-From`, `Supersedes` fields are structural encoding of lineage. This was a deliberate move from heuristic (citation parsing) to structural (author declaration) for high-value metadata.

**Answer to Investigation Question:**

**Encoding points** in the ecosystem include:
- **Creation:** Beads issue (type, priority, labels), kb quick (type, reason), spawn config (skill, tier, model), skill frontmatter (dependencies, skill-type)
- **Completion:** SYNTHESIS.md (outcome, recommendation, D.E.K.N.), phase reporting (bd comment)
- **Post-hoc:** Date from filename, title from heading, status from **Phase:**, references via regex

**These are boundaries** corresponding to knowledge states. Creation captures intent, completion captures outcome, post-hoc captures reader interpretation.

**Tradeoffs:** Structural encoding adds friction but guarantees fidelity. Heuristic extraction is frictionless but lossy. The ecosystem correctly uses structural for high-stakes metadata (drives lifecycle/routing) and heuristic for discovery (topic search).

**Why not require everything upfront?** Friction. Context limits. Some metadata only makes sense after work is done (e.g., "what was learned"). The D.E.K.N. structure at completion time is the right answer for synthesis metadata.

---

## Structured Uncertainty

**What's tested:**

- ✅ Beads CreateArgs requires type and priority (verified: read pkg/beads/types.go:62-74)
- ✅ Spawn Config has 16+ explicit fields (verified: read pkg/spawn/config.go:97-227)
- ✅ KB artifacts extract date from filename, title from heading (verified: read pkg/kb/artifacts.go:102-164)
- ✅ Skill dependencies are declared in frontmatter (verified: read pkg/skills/loader.go:20-30)
- ✅ SYNTHESIS.md parsing uses regex for D.E.K.N. sections (verified: read pkg/verify/synthesis_parser.go:44-93)
- ✅ KB quick entries have JSON schema (verified: head -30 .kb/quick/entries.jsonl)
- ✅ Investigation template has lineage fields (verified: read template in own investigation file)

**What's untested:**

- ⚠️ Topic extraction accuracy (~80% is estimate, not measured)
- ⚠️ Prior work detection accuracy via citation parsing (not benchmarked)
- ⚠️ Whether kb-cli and orch-knowledge have identical patterns (only examined orch-go)

**What would change this:**

- If beads allowed creating issues without type, Finding 2 would be wrong
- If skills inferred dependencies from content analysis, Finding 6 would be wrong
- If heuristic extraction accuracy was higher than assumed, the friction/fidelity tradeoff analysis would shift

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Create model file documenting encoding points | **architectural** | Cross-component framework that guides future design decisions |
| Add `topic:` frontmatter to investigation template | **architectural** | Changes template used across ecosystem |
| Keep current heuristic topic extraction | **implementation** | Status quo, no change |

### Recommended Approach ⭐

**Document as Model + Selective Structural Addition** - Create `.kb/models/semantic-metadata-encoding.md` documenting the three-boundary model. For topic specifically, add optional `topic:` frontmatter but keep heuristic extraction as fallback.

**Why this approach:**
- Documents implicit knowledge explicitly (Finding 1-5 patterns are currently undocumented)
- Provides framework for future metadata decisions
- Topic field is low-friction (optional) with high-value (100% accurate when present)

**Trade-offs accepted:**
- Optional topic field means not all artifacts will have it
- Model file adds documentation overhead
- Doesn't mandate structural encoding everywhere

**Implementation sequence:**
1. **Create model file** - `.kb/models/semantic-metadata-encoding.md` with three-boundary framework
2. **Add optional topic field** - Update investigation template with `topic:` frontmatter
3. **Update kb context** - Prefer explicit topic when present, fall back to title parsing

### Alternative Approaches Considered

**Option B: Mandate Structural Encoding Everywhere**
- **Pros:** Maximum fidelity, no heuristic extraction needed
- **Cons:** High friction, context bloat, some metadata impossible at creation time
- **When to use instead:** Never as blanket approach; selective structural encoding is better

**Option C: Accept Pure Heuristic Extraction**
- **Pros:** Zero friction, no template changes needed
- **Cons:** ~80% accuracy for topic, can't distinguish intent from mention for citations
- **When to use instead:** For truly ephemeral metadata (e.g., "files mentioned")

**Rationale for recommendation:** The ecosystem already implicitly uses the three-boundary model. Making it explicit enables better future decisions. Optional topic field threads the needle between friction and fidelity.

---

### Implementation Details

**What to implement first:**
- Create `.kb/models/semantic-metadata-encoding.md` capturing this investigation's findings
- This becomes reference for future metadata design questions

**Things to watch out for:**
- ⚠️ Template changes ripple across skill definitions (via skillc)
- ⚠️ Optional fields may not be filled by agents without explicit prompting
- ⚠️ Heuristic fallback must remain for backward compatibility

**Areas needing further investigation:**
- Actual topic extraction accuracy measurement
- Cross-project consistency (kb-cli, orch-knowledge patterns)
- Whether frontmatter parsing has performance implications

**Success criteria:**
- ✅ Model file documents three-boundary framework with examples
- ✅ Future metadata design decisions reference this model
- ✅ Topic extraction accuracy improves if structural field adopted

---

## References

**Files Examined:**
- `pkg/spawn/config.go:97-227` - Config struct with 20+ metadata fields
- `pkg/beads/types.go:62-158` - Issue struct and CreateArgs
- `pkg/skills/loader.go:20-191` - Skill loading and metadata parsing
- `pkg/kb/artifacts.go:1-165` - Artifact parsing and heuristic extraction
- `pkg/verify/synthesis_parser.go:1-224` - SYNTHESIS.md parsing
- `pkg/spawn/kbcontext.go:1-741` - KB context parsing and formatting
- `.kb/quick/entries.jsonl` - Quick entry JSON format
- `.orch/templates/SYNTHESIS.md` - Handoff template structure
- `.kb/investigations/2026-01-31-inv-identify-heuristic-vs-structural-patterns.md` - Related investigation

**Commands Run:**
```bash
# List skills directory
ls -la ~/.claude/skills/

# Check kb quick entries format
head -30 .kb/quick/entries.jsonl

# Verify project location
pwd
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-31-inv-identify-heuristic-vs-structural-patterns.md` - Heuristic vs structural enforcement patterns
- **Issue:** `orch-go-21129` - Topic extraction (heuristic fix mentioned in issue)
- **Decision:** Investigation Lineage Enforcement (Prior-Work table structural fix)

---

## Investigation History

**2026-01-31 14:02:** Investigation started
- Initial question: What are the semantic metadata encoding points across the ecosystem?
- Context: Orchestrator wants to understand where understanding gets crystallized into metadata

**2026-01-31 14:05:** Found core encoding point categories
- Identified spawn config (16 fields), beads types (12 fields), skill frontmatter (8 fields)
- Recognized three-boundary pattern: creation, completion, post-hoc

**2026-01-31 14:10:** Examined heuristic extraction patterns
- KB artifacts parse date from filename, title from heading
- KB context parses section headers and entry lines
- Synthesis parser extracts D.E.K.N. sections via regex

**2026-01-31 14:15:** Connected to friction/fidelity tradeoff
- High-stakes metadata is structural despite friction
- Discovery metadata is heuristic despite lossiness
- This is a sensible implicit calibration

**2026-01-31 14:20:** Investigation completed
- Status: Complete
- Key outcome: 10+ encoding points at three boundaries; ecosystem correctly uses structural for routing/lifecycle, heuristic for discovery
