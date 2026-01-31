<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Domain models should be auto-included in SPAWN_CONTEXT.md via keyword matching, with Summary + Constraints + Why This Fails sections (not full content).

**Evidence:** Token limit constraint (150k max), principles (Surfacing Over Browsing, Progressive Disclosure), existing KBContext mechanism already surfaces constraints/decisions.

**Knowledge:** Models are designed to answer "What enables/constrains X?" - exactly what agents need. But current kb context only lists model paths, doesn't include content. Including key sections gives agents the queryable understanding models were designed to provide.

**Next:** Implement in pkg/spawn/context.go: add model discovery, extract key sections, include in new "## DOMAIN MODELS" section.

**Promote to Decision:** Actioned - patterns in spawn guide (model inclusion)

---

# Investigation: Spawn Context Model Inclusion

**Question:** How should domain models be auto-included in SPAWN_CONTEXT.md so agents work FROM models, not beside them?

**Started:** 2026-01-14
**Updated:** 2026-01-14
**Owner:** og-arch-design-spawn-context-14jan-80af
**Phase:** Complete
**Next Step:** Implementation by feature-impl agent
**Status:** Complete

---

## Findings

### Finding 1: KB Context Already Surfaces Models (But Only as Paths)

**Evidence:** Running `kb context "spawn"` returns:
```
## MODELS (synthesized understanding)
- Spawn Architecture
  Path: /Users/dylanconlin/Documents/personal/orch-go/.kb/models/spawn-architecture.md
```

But agents must then navigate to read the model content. This violates "Surfacing Over Browsing" principle.

**Source:**
- `kb context "spawn context" --limit 5` output
- `~/.kb/principles.md` lines 115-127 (Surfacing Over Browsing)

**Significance:** Current mechanism surfaces *where* models are, not *what* they contain. Agents get a pointer, not the understanding.

---

### Finding 2: KBContext Is Already a Template Variable

**Evidence:** In `pkg/spawn/context.go`:
- Line 404: `KBContext string` field in contextData struct
- Line 450: `KBContext: cfg.KBContext` populates template
- Lines 51-53: Template conditional `{{if .KBContext}}{{.KBContext}}{{end}}`

The infrastructure exists - KBContext is already populated and rendered.

**Source:** `pkg/spawn/context.go:404`, `pkg/spawn/context.go:450-451`, `pkg/spawn/context.go:51-53`

**Significance:** We can enhance the existing KBContext content OR add a separate field for models. Both are architecturally viable.

---

### Finding 3: Model Template Has Natural Section Boundaries

**Evidence:** From `.kb/models/TEMPLATE.md` and existing models like `spawn-architecture.md`:
- Summary (30 seconds) - Quick orientation
- Core Mechanism - How it works
- Why This Fails - Failure modes
- Constraints - What enables/constrains (key for agents!)
- Evolution - Historical changes

The "Constraints" section is specifically designed for "What enables/constrains X?" queries - exactly what agents need.

**Source:** `.kb/models/TEMPLATE.md`, `.kb/models/spawn-architecture.md:184-218`

**Significance:** We can selectively include the most agent-relevant sections without full model content.

---

### Finding 4: Token Limits Constrain Full Model Inclusion

**Evidence:** From spawn-architecture model:
- Token estimation: 4 chars/token
- Warning at 100k tokens, hard error at 150k
- Skill content can be 10-40k tokens
- KB context can be 30-50k tokens
- Full models are 200-300 lines each

Including full content of multiple models risks hitting token limits.

**Source:** `.kb/models/spawn-architecture.md:159-166`, `pkg/spawn/context.go` token estimation logic

**Significance:** Must be selective about what model content to include. Progressive disclosure pattern applies.

---

## Fork Navigation

### Fork 1: How to Determine Relevant Models?

**Options:**
- A: **Keyword matching on task description** - Same approach as current kb context
- B: **Skill-to-model mapping** - Explicit registry of which skills need which models
- C: **Domain inference** - Infer domain from task + issue type

**Substrate says:**
- Principle (Surfacing Over Browsing): "Commands answer 'what's relevant now?'"
- Existing behavior: kb context uses keyword matching
- Constraint (Token limits): Can't include all models, must be selective

**RECOMMENDATION:** Option A (keyword matching) because:
1. Consistent with existing kb context mechanism
2. No new configuration/registry to maintain
3. Works dynamically based on task content
4. Already proven to work for constraints/decisions

**Trade-off accepted:** May miss models where terminology differs (e.g., task says "agent lifecycle" but model is "opencode-session-lifecycle"). Acceptable because agents can still navigate to models if needed.

---

### Fork 2: What to Include from Each Model?

**Options:**
- A: **Full model content** - Complete context
- B: **Summary only** - Minimal footprint
- C: **Summary + Constraints + Why This Fails** - Key queryable sections

**Substrate says:**
- Principle (Progressive Disclosure): "TLDR first. Key sections next."
- Principle (Understanding Through Engagement): "Models create surface area for questions by making implicit constraints explicit."
- Constraint (Token limits): 150k limit, competing with skill content

**RECOMMENDATION:** Option C (Summary + Constraints + Why This Fails) because:
1. Summary provides orientation (what is this about?)
2. Constraints provides "What enables/constrains?" answers - the core value of models
3. Why This Fails prevents agents from hitting known pitfalls
4. Core Mechanism and Evolution are reference material, not immediately actionable
5. Estimated ~50-80 lines per model vs 200-300 for full

**Trade-off accepted:** Agents don't get full mechanism understanding. But they can read full model if needed - we're optimizing for the 80% case.

---

### Fork 3: Where in SPAWN_CONTEXT.md?

**Options:**
- A: **Merge into existing KBContext section** - Single knowledge section
- B: **Separate DOMAIN MODELS section** - Distinct from constraints/decisions
- C: **Before the task** - Set frame before presenting work

**Substrate says:**
- Principle (Progressive Disclosure): Summary → Key Findings → Details
- Current structure: KBContext appears after TASK (lines 51-53 in template)
- Model purpose: "Descriptive (how system IS)" vs guides "Prescriptive (how to DO)"

**RECOMMENDATION:** Option B (separate DOMAIN MODELS section) because:
1. Models are distinct artifact type - deserves distinct section
2. Clear separation from constraints/decisions (different type of knowledge)
3. Agent can quickly identify "do I have model context for this domain?"
4. Placed after KBContext but before DELIVERABLES - knowledge before action

**Trade-off accepted:** Slightly longer SPAWN_CONTEXT.md structure. Acceptable because clarity > brevity.

---

### Fork 4: What if No Models Found?

**Options:**
- A: **Silent (no section)** - Avoid noise
- B: **Note "No relevant models found"** - Explicit absence
- C: **Suggest model creation** - Could be noisy

**Substrate says:**
- Principle (Surfacing Over Browsing): Surface relevant state, don't require navigation
- Design pattern: Existing kb context sections are omitted if empty
- Constraint (Keep spawn context clean): Don't add noise

**RECOMMENDATION:** Option A (silent) because:
1. No models = nothing to surface - consistent with Surfacing principle
2. Adding explicit absence creates noise without value
3. Agent doesn't need to know "we looked and found nothing"
4. Matches existing pattern where KBContext is omitted if empty

**Trade-off accepted:** Agent won't know if models were searched for. Acceptable because knowledge of absence doesn't change behavior.

---

## Synthesis

**Key Insights:**

1. **Models were designed for this use case** - The "Constraints" section specifically answers "What enables/constrains?" which is exactly what agents need when working in a domain.

2. **Progressive disclosure is essential** - Full models are too large (200-300 lines), but Summary + Constraints + Why This Fails (~50-80 lines) provides the queryable understanding.

3. **Infrastructure already exists** - KBContext field in spawn config, template rendering, keyword-based discovery via kb context. This is enhancement, not new architecture.

**Answer to Investigation Question:**

Domain models should be auto-included in SPAWN_CONTEXT.md via:
1. **Discovery:** Keyword matching on task description (same as kb context)
2. **Content:** Summary + Constraints + Why This Fails sections (progressive disclosure)
3. **Placement:** New "## DOMAIN MODELS" section after KBContext, before DELIVERABLES
4. **Empty handling:** Silent omission if no models match

This follows "Surfacing Over Browsing" by bringing model understanding to the agent, respects token limits via selective sections, and leverages models' designed purpose of making constraints explicit.

---

## Structured Uncertainty

**What's tested:**

- ✅ KB context returns model paths for keyword queries (verified: `kb context "spawn"` output)
- ✅ Token limits are enforced (verified: spawn-architecture model documents 100k warning, 150k error)
- ✅ Model template has consistent section structure (verified: read TEMPLATE.md and spawn-architecture.md)

**What's untested:**

- ⚠️ Token impact of including model sections (not measured actual token counts)
- ⚠️ Keyword matching quality for model discovery (may miss semantic matches)
- ⚠️ Agent actually uses model content vs ignores it (need post-implementation validation)

**What would change this:**

- If token counts show Summary+Constraints+WhyThisFails exceeds 100 lines regularly → reduce to Summary+Constraints only
- If keyword matching misses important models frequently → consider skill-to-model registry
- If agents don't use included model content → question whether inclusion adds value

---

## Implementation Recommendations

### Recommended Approach ⭐

**Keyword-based model discovery with selective section extraction** - Extend existing kb context mechanism to include model sections in SPAWN_CONTEXT.md.

**Why this approach:**
- Builds on proven kb context infrastructure
- No new configuration to maintain (dynamic discovery)
- Respects token limits via progressive disclosure
- Directly addresses the goal: agents work FROM models

**Trade-offs accepted:**
- Keyword matching may miss semantic relationships
- Not all model content included (but enough for common cases)

**Implementation sequence:**
1. Add model discovery function to `pkg/spawn/` - Extract model file paths matching task keywords
2. Add section extraction function - Parse Summary, Constraints, Why This Fails from markdown
3. Extend contextData struct - Add `DomainModels string` field
4. Update template - Add `## DOMAIN MODELS` conditional section
5. Wire up in GenerateContext - Call discovery and extraction, populate field

### Alternative Approaches Considered

**Option B: Skill-to-model registry**
- **Pros:** Precise mapping, no keyword matching errors
- **Cons:** Manual maintenance, doesn't adapt to new models, configuration burden
- **When to use instead:** If keyword matching proves unreliable after implementation

**Option C: Include full model content**
- **Pros:** Complete context available
- **Cons:** Token limit risk, information overload, violates progressive disclosure
- **When to use instead:** If agents consistently need full mechanism understanding

---

### Implementation Details

**What to implement first:**
1. Model discovery function (can reuse kb context logic or shell out to `kb context --format json`)
2. Section extraction (markdown parsing for specific headers)
3. Template integration (low risk, straightforward)

**Things to watch out for:**
- ⚠️ Markdown parsing edge cases (nested headers, code blocks containing `##`)
- ⚠️ Token counting may be needed to warn if model content is large
- ⚠️ Multiple models matching could exceed reasonable size - consider limit (2-3 max)

**Areas needing further investigation:**
- How does `kb context` determine relevance scores? Could filter to top matches.
- Should model section extraction use regex or proper markdown parser?
- Should we cache parsed model sections for performance?

**Success criteria:**
- ✅ Spawned agents receive relevant model context without navigation
- ✅ Token usage stays within limits (no new 150k errors)
- ✅ Agents reference model constraints in their work (verify in completed SYNTHESIS.md files)

---

## References

**Files Examined:**
- `pkg/spawn/context.go` - Spawn context generation template and logic
- `pkg/spawn/config.go` - SpawnConfig struct definition
- `.kb/models/spawn-architecture.md` - Example model to understand structure
- `.kb/models/TEMPLATE.md` - Model template structure
- `~/.kb/principles.md` - Surfacing Over Browsing, Progressive Disclosure, Session Amnesia

**Commands Run:**
```bash
# Check kb context output
kb context "spawn context" --limit 5

# Check kb context help for understanding mechanism
kb context --help
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-12-models-as-understanding-artifacts.md` - Why models exist
- **Model:** `.kb/models/spawn-architecture.md` - Current spawn understanding

---

## Investigation History

**2026-01-14 16:30:** Investigation started
- Initial question: How should domain models be auto-included in SPAWN_CONTEXT.md?
- Context: Models exist but agents don't work FROM them - they work beside them

**2026-01-14 16:45:** Substrate consultation complete
- Identified 4 decision forks
- Consulted principles: Surfacing Over Browsing, Progressive Disclosure, Understanding Through Engagement
- Identified constraint: Token limits

**2026-01-14 17:00:** Fork navigation complete
- Recommendations made for all 4 forks
- Implementation sequence identified
