<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Tool failures are logged but not connected to skill guidance; we can build a Skill-Execution Bridge to map failures to documentation gaps.

**Evidence:** Found action logging (pkg/action), skill loading (pkg/skills), OpenCode tool events (pkg/opencode), but no code linking them. Skills have parseable markdown structure.

**Knowledge:** Bridge requires 4 components: (1) Skill Parser to extract structure, (2) Context Annotator to enrich action logs, (3) Failure Analyzer to lookup guidance, (4) Friction Reporter to output findings.

**Next:** Implement incrementally starting with Skill Parser, validate with real sessions before building full analyzer.

**Promote to Decision:** recommend-yes - This establishes architectural approach for skill quality feedback loop.

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

# Investigation: Design Skill Execution Bridge Friction

**Question:** How can we cross-reference tool failures with SKILL.md guidance to identify when incorrect documentation is causing friction?

**Started:** 2026-01-16
**Updated:** 2026-01-16
**Owner:** og-feat-design-skill-execution-16jan-654b
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Tool outcome logging exists but is separate from skill context

**Evidence:** 
- `pkg/action/action.go` implements ActionEvent logging with outcomes: success, empty, error, fallback
- ActionEvent captures: Tool (e.g., "Read", "Bash"), Target (e.g., file path), Outcome, ErrorMessage
- Events logged to `~/.orch/action-log.jsonl` with SessionID and Workspace fields
- Pattern detection identifies repeated failures (PatternThreshold = 3 occurrences)

**Source:** 
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/action/action.go:1-680`
- Comment at line 7: "Tool failures are ephemeral and untracked, making behavioral pattern detection impossible"

**Significance:** 
Tool failures ARE being tracked, but the system doesn't know which skill guidance section was relevant to the failed tool usage. This is the missing bridge.

---

### Finding 2: Skills are embedded as full SKILL.md content in SPAWN_CONTEXT

**Evidence:**
- SPAWN_CONTEXT template includes `{{.SkillContent}}` section (line 238-249)
- Skills loaded via `pkg/skills/loader.go` from `~/.claude/skills/`
- LoadSkillWithDependencies() loads skill + dependencies as concatenated markdown
- Full skill content (often 1000+ lines) embedded in spawn context

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go:238-249`
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/skills/loader.go:105-149`

**Significance:**
The skill guidance is available in structured markdown format with sections/phases. We could parse this structure to map tool failures to relevant guidance sections.

---

### Finding 3: OpenCode provides tool invocation data via MessagePart events

**Evidence:**
- OpenCode returns Message objects with Parts array
- MessagePart has Type field including "tool-invocation" and "tool"
- MessagePart includes Text field with tool details
- OpenCode client extracts activity from tool invocations for status display

**Source:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/types.go:118-125`
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/client.go` (tool extraction logic)

**Significance:**
We have access to tool invocations from OpenCode events, but we're not currently capturing tool results/failures from these events. The action logging happens separately.

---

### Finding 4: No current mechanism maps tool failures to skill sections

**Evidence:**
- Action logging captures tool + target + outcome
- Skill content is full markdown with headers, sections, phases
- No code currently parses skill structure or maps tools to sections
- FindPatterns() groups repeated failures but doesn't reference skill guidance

**Source:**
- Searched codebase for "skill.*section", "guidance.*map", "tool.*skill" - no results
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/action/action.go:420-502` (FindPatterns implementation)

**Significance:**
This is the core gap. When a tool fails repeatedly, we know WHAT failed (tool + target) and HOW MANY times, but we don't know WHICH section of the skill guidance was supposed to help prevent that failure.

---

## Synthesis

**Key Insights:**

1. **The pieces exist but aren't connected** - We have action logging (tool failures), skill content (guidance), and OpenCode events (tool invocations), but no mapping between them. The bridge is the missing piece.

2. **Skills have parseable structure** - SKILL.md files use markdown with clear hierarchy (phases, sections, subsections). We can parse this structure to identify which sections relate to which tools and phases.

3. **Phase context is available** - Agents report phase via `bd comment` (e.g., "Phase: Investigation", "Phase: Implementation"). We can use this to narrow down which skill sections are relevant when a tool fails.

4. **Current pattern detection is skill-blind** - FindPatterns() identifies repeated failures but suggests generic "kb tried" commands. It doesn't reference the skill guidance that should have prevented the failure.

**Answer to Investigation Question:**

To cross-reference tool failures with SKILL.md guidance, we need a **Skill-Execution Bridge** with four components:

1. **Skill Parser** - Parse SKILL.md structure to extract phases, tool mentions, and guidance sections
2. **Context Annotator** - Enrich action logs with phase/skill context at spawn time
3. **Failure Analyzer** - When patterns detected, lookup relevant skill sections based on phase + tool
4. **Friction Reporter** - Generate reports showing: tool failed X times → skill section Y said do Z → evidence of gap

This enables identifying when documentation is incomplete, wrong, or not being followed.

---

## Structured Uncertainty

**What's tested:**

- ✅ Action logging captures tool failures (verified: read pkg/action/action.go code, confirmed ActionEvent structure)
- ✅ Skills are embedded in SPAWN_CONTEXT as markdown (verified: read pkg/spawn/context.go template)
- ✅ OpenCode provides tool-invocation events (verified: read pkg/opencode/types.go MessagePart structure)
- ✅ Skills have parseable structure (verified: read ~/.claude/skills/worker/feature-impl/SKILL.md)

**What's untested:**

- ⚠️ **Heuristics will accurately classify friction type** (need real session testing)
- ⚠️ **Phase detection from bd comments is reliable** (agents may not follow convention)
- ⚠️ **Skill authors will find reports actionable** (need user feedback)
- ⚠️ **Parser handles all skill structure variations** (only examined feature-impl)
- ⚠️ **Tool name normalization covers all cases** (may miss variations)

**What would change this:**

- If skill structure varies significantly across skills, parser may need skill-specific logic
- If phase detection is unreliable, may need alternative (workspace metadata, opencode events)
- If false positive rate >20%, heuristics need refinement
- If reports aren't actionable, format/content needs redesign

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Incremental Skill-Execution Bridge** - Build four components that work independently and compose together, starting with minimal friction reporting.

**Why this approach:**
- Provides immediate value at each step (not all-or-nothing)
- Validates design assumptions before full implementation
- Allows testing with real agent sessions to refine heuristics
- Builds on existing infrastructure (action logging, skill loading, events)

**Trade-offs accepted:**
- First iteration will have manual review step (not automated)
- Initial skill parsing will be simple (markdown headers only, not full semantic analysis)
- Phase detection relies on bd comment convention (not foolproof)

**Implementation sequence:**
1. **Skill Parser (foundation)** - Extract structure from SKILL.md files to enable other components
2. **Context Annotator (enrichment)** - Add skill/phase metadata to action logs at spawn time
3. **Failure Analyzer (intelligence)** - Lookup relevant guidance when patterns detected
4. **Friction Reporter (output)** - Generate human-readable reports for skill authors

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

**Component 1: Skill Parser (`pkg/skills/parser.go`)**

Create parser to extract skill structure:

```go
type SkillSection struct {
    Level      int                    // Heading level (1-6)
    Title      string                 // Section title
    Content    string                 // Full section content
    Tools      []string               // Mentioned tools (Read, Bash, etc.)
    Phase      string                 // Phase name if in phase section
    Children   []SkillSection         // Nested sections
    LineStart  int                    // Line number where section starts
    LineEnd    int                    // Line number where section ends
}

type SkillStructure struct {
    Metadata    *SkillMetadata        // Existing frontmatter
    Sections    []SkillSection        // Top-level sections
    PhaseMap    map[string][]SkillSection // phase name -> relevant sections
}

func ParseSkillStructure(content string) (*SkillStructure, error)
func (s *SkillStructure) FindSectionsForTool(tool, phase string) []SkillSection
```

**What to extract:**
- Markdown headers (##, ###, ####)
- Tool mentions (case-insensitive: "Read", "Write", "Bash", "bd comment", "kb create", etc.)
- Phase associations (sections under "### Investigation Phase", "### Implementation Phase", etc.)
- Anti-pattern markers ("⚠️", "RED FLAG", "Critical")

**Component 2: Context Annotator (extend `pkg/action/action.go`)**

Enrich ActionEvent with skill context:

```go
type ActionEvent struct {
    // ... existing fields ...
    Skill       string   `json:"skill,omitempty"`         // Which skill (e.g., "feature-impl")
    Phase       string   `json:"phase,omitempty"`         // Current phase (from bd comment)
    BeadsID     string   `json:"beads_id,omitempty"`      // Issue being worked on
}

// Called at spawn time to set skill context for session
func (l *Logger) SetSessionContext(sessionID, skill, beadsID string)
```

**How to populate:**
- At spawn time: capture skill name from spawn command
- From beads comments: parse "Phase: X" from recent comments
- From workspace metadata: read SPAWN_CONTEXT.md for skill name

**Component 3: Failure Analyzer (extend `pkg/action/action.go`)**

Add method to analyze failures with skill context:

```go
type FailureAnalysis struct {
    Pattern           ActionPattern
    RelevantSections  []SkillSection      // Sections that mentioned this tool
    PhaseGuidance     string              // What the phase said to do
    GuidanceFollowed  bool                // Best guess if guidance was followed
    FrictionType      string              // "missing-guidance" | "unclear-guidance" | "ignored-guidance"
}

func (t *Tracker) AnalyzeFailureWithSkill(pattern ActionPattern, skillPath string) (*FailureAnalysis, error)
```

**Analysis heuristics:**
1. Load skill structure for the pattern's Skill field
2. Find sections matching pattern.Tool + pattern.Phase (if available)
3. Check if sections exist:
   - No sections found → FrictionType: "missing-guidance"
   - Sections found but vague → "unclear-guidance"
   - Sections found with clear steps → "ignored-guidance" (or skill is wrong)

**Component 4: Friction Reporter (new `cmd/orch/friction_cmd.go`)**

CLI command to generate reports:

```bash
orch friction report                    # Show all friction patterns
orch friction report --session <id>     # For specific session
orch friction report --skill feature-impl  # For specific skill
orch friction report --export friction-report.md  # Export to file
```

**Report format:**
```markdown
# Skill Friction Report

## Pattern: Read → error (5 occurrences)
**Session:** og-feat-xyz-123
**Skill:** feature-impl
**Phase:** Investigation

### What Failed
- Tool: Read
- Target: .kb/investigations/2026-01-16-inv-*.md
- Outcome: error (file not found)
- Occurrences: 5 times over 10 minutes

### What Skill Guidance Said
From "Investigation Phase" section (line 120-137):
> Create investigation template BEFORE exploring (not at end)
> 1. Create investigation template BEFORE exploring (not at end)

### Friction Analysis
**Type:** ignored-guidance
**Evidence:** Agent attempted to read investigation file 5 times before creating it. 
Skill explicitly says "BEFORE exploring" but agent explored first.

**Recommendation:** Either:
1. Strengthen guidance with red flag: "⚠️ CRITICAL: Create file BEFORE reading"
2. Add pre-flight check: "Run `ls .kb/investigations/` to verify file exists"
3. Investigation skill issue - agent doesn't follow sequence
```

**What to implement first:**
1. Skill Parser (enables everything else)
2. Simple friction report for one known pattern (validate approach)
3. Context Annotator (once proven valuable)
4. Full Failure Analyzer (after testing with real sessions)

**Things to watch out for:**
- ⚠️ **Phase detection fragility** - Relies on bd comment convention; agents may not always report phase correctly
- ⚠️ **Tool name variations** - "Bash" vs "bash", "Read" vs "read file", need normalization
- ⚠️ **False positives** - Some failures are legitimate exploration (grep returns empty), not friction
- ⚠️ **Skill structure variance** - Not all skills have phase sections; need graceful degradation
- ⚠️ **Privacy** - Action logs may contain file paths, commands with sensitive data; filter before export

**Areas needing further investigation:**
- How to detect if guidance was "followed but insufficient" vs "not followed"
- Should we track *successful* tool usage against guidance (to validate guidance works)?
- Can we use LLM to semantically analyze if tool usage matches guidance intent?
- Integration with `orch complete` verification - should friction be a quality gate?

**Success criteria:**
- ✅ Can parse feature-impl SKILL.md and extract phase sections with tool mentions
- ✅ Can generate friction report showing tool failures mapped to skill guidance
- ✅ Skill authors can identify documentation gaps from real agent sessions
- ✅ `orch friction report` command works and produces actionable output
- ✅ False positive rate < 20% (manual review of first 20 reports)

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/action/action.go` - Action logging implementation, ActionEvent structure, pattern detection
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/spawn/context.go` - SPAWN_CONTEXT template, skill embedding
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/skills/loader.go` - Skill loading, SKILL.md discovery
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/types.go` - OpenCode event types, MessagePart structure
- `/Users/dylanconlin/Documents/personal/orch-go/pkg/events/logger.go` - Event logging, verification tracking
- `~/.claude/skills/worker/feature-impl/SKILL.md` - Example skill structure, phase sections

**Commands Run:**
```bash
# Search for tool failure tracking
rg "tool.*fail|failure|error.*tool" --type go

# Search for skill context usage
rg "skill.*context|SKILL\.md|skill guidance" --type go

# Search for tool invocation handling
rg "tool.*invocation|tool.*result" /Users/dylanconlin/Documents/personal/orch-go/pkg/opencode/ -A 3 -B 3
```

**External Documentation:**
- None referenced

**Related Artifacts:**
- **Investigation:** Referenced in spawn context - orchestrator skill drift, skill investigations synthesis
- **Workspace:** `.orch/workspace/og-feat-design-skill-execution-16jan-654b/`

---

## Investigation History

**[YYYY-MM-DD HH:MM]:** Investigation started
- Initial question: [Original question as posed]
- Context: [Why this investigation was initiated]

**[YYYY-MM-DD HH:MM]:** [Milestone or significant finding]
- [Description of what happened or was discovered]

**[YYYY-MM-DD HH:MM]:** Investigation completed
- Status: [Complete/Paused with reason]
- Key outcome: [One sentence summary of result]
