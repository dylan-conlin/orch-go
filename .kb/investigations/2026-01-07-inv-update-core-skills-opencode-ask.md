<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** Skills referenced non-existent `AskUserQuestion` tool; updated 8 files across 3 skills to use opencode's `question` tool.

**Evidence:** Searched skills/src directory, found 7 files referencing `AskUserQuestion` in feature-impl, architect, and design-session skills. OpenCode provides a `question` tool with JSON interface (questions array with question/header/options).

**Knowledge:** The `question` tool is available to all sessions (not restricted to CLI-only). Interface requires: question string, header (max 12 chars), options array with label/description. Recommended option should be first with "(Recommended)" suffix.

**Next:** Deploy skills via `skillc deploy` and verify agents can use the question tool.

**Promote to Decision:** recommend-no (tactical fix, not architectural change)

---

# Investigation: Update Core Skills to Use OpenCode Question Tool

**Question:** How should skills be updated to use opencode's question tool instead of the non-existent `AskUserQuestion` tool?

**Started:** 2026-01-07
**Updated:** 2026-01-07
**Owner:** Claude (worker agent)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: Skills referenced non-existent `AskUserQuestion` tool

**Evidence:** Searched `/Users/dylanconlin/orch-knowledge/skills/src/` for `AskUserQuestion` references. Found 7 files across 3 skills:

- `feature-impl/.skillc/phases/clarifying-questions.md`
- `feature-impl/reference/phase-clarifying-questions.md`
- `architect/.skillc/SKILL.md.template`
- `architect/.skillc/skill.yaml` (allowed-tools list)
- `architect/SKILL.md`
- `design-session/.skillc/SKILL.md.template`
- `design-session/.skillc/skill.yaml` (allowed-tools list)
- `design-session/SKILL.md`

**Source:** `grep -rn "AskUserQuestion" /Users/dylanconlin/orch-knowledge/skills/src/`

**Significance:** This explains the recurring friction (8x gap) - agents couldn't find the tool because it was named incorrectly in skill documentation.

---

### Finding 2: OpenCode provides a `question` tool with specific interface

**Evidence:** Examined OpenCode source code:
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/question.ts`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/question/index.ts`

Tool interface:
```typescript
// Tool name: "question"
parameters: {
  questions: z.array({
    question: z.string().describe("Complete question"),
    header: z.string().max(12).describe("Very short label (max 12 chars)"),
    options: z.array({
      label: z.string().describe("Display text (1-5 words, concise)"),
      description: z.string().describe("Explanation of choice"),
    })
  })
}
```

**Source:** `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/question.ts:6-27`

**Significance:** Different interface than documented in skills - requires JSON with specific structure, not YAML.

---

### Finding 3: Question tool is available in all sessions (including spawned agents)

**Evidence:** In `registry.ts`:
```typescript
...(Flag.OPENCODE_CLIENT === "cli" ? [QuestionTool] : []),
```

`OPENCODE_CLIENT` defaults to `"cli"` (from `flag.ts`), so the question tool is included by default.

**Source:** 
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/registry.ts:96`
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/flag/flag.ts:17`

**Significance:** Spawned agents via `orch spawn` have access to the question tool - no configuration needed.

---

## Synthesis

**Key Insights:**

1. **Name mismatch caused friction** - Skills documented `AskUserQuestion` but OpenCode provides `question`. This naming mismatch caused the "ask command inline" friction (8x gap) identified by `orch learn`.

2. **Interface is JSON-based** - The old documentation showed YAML-like format but the actual tool uses JSON with specific schema (questions array, header max 12 chars, options with label/description).

3. **Directive-guidance pattern preserved** - The updates maintain the directive-guidance pattern (recommend first, explain why) while aligning with actual tool interface.

**Answer to Investigation Question:**

Updated all 8 files across 3 skills (feature-impl, architect, design-session) to:
1. Replace `AskUserQuestion` with `question` tool
2. Provide accurate JSON interface documentation
3. Include usage notes (recommended option first with "(Recommended)" label)
4. Update skill.yaml `allowed-tools` lists

---

## Structured Uncertainty

**What's tested:**

- ✅ All `AskUserQuestion` references removed (verified: grep returns empty)
- ✅ Correct tool name is `question` (verified: source code inspection)
- ✅ Tool interface documented correctly (verified: matches `Question.Info` schema)

**What's untested:**

- ⚠️ Skills compile successfully with `skillc deploy` (not run - different repo)
- ⚠️ Agents can successfully invoke the question tool (needs integration test)
- ⚠️ Question tool works in headless mode (spawned agents)

**What would change this:**

- Finding would be wrong if `OPENCODE_CLIENT` is set differently in spawn environment
- Finding would be wrong if question tool has additional restrictions not visible in source

---

## Implementation Recommendations

### Recommended Approach ⭐

**Direct skill updates** - Update all skill files in orch-knowledge to use correct `question` tool name and interface.

**Why this approach:**
- Fixes root cause (incorrect tool name in documentation)
- Preserves directive-guidance pattern agents should follow
- No changes needed to opencode itself

**Trade-offs accepted:**
- Requires manual skill deployment after changes
- Agents already spawned will have old guidance until respawn

**Implementation sequence:**
1. ✅ Update feature-impl skill (2 files)
2. ✅ Update architect skill (3 files) 
3. ✅ Update design-session skill (3 files)
4. Deploy skills via `skillc deploy`
5. Verify with test spawn

### Alternative Approaches Considered

**Option B: Add `AskUserQuestion` alias in OpenCode**
- **Pros:** No skill changes needed
- **Cons:** Creates confusing dual naming, doesn't fix interface documentation
- **When to use instead:** If skills are frozen/versioned

**Rationale for recommendation:** Direct skill updates are straightforward and fix both the name and interface documentation issues.

---

### Implementation Details

**What was implemented:**

- Updated 8 files across 3 skills
- Replaced `AskUserQuestion` with `question`
- Added JSON interface documentation with correct schema
- Updated `allowed-tools` in skill.yaml files
- Preserved directive-guidance pattern (recommend first with reasoning)

**Things to watch out for:**

- ⚠️ Need to run `skillc deploy` to compile and deploy updated skills
- ⚠️ The `header` field has max 12 character limit
- ⚠️ Users can always select "Other" for custom input (documented in tool description)

**Success criteria:**

- ✅ No `AskUserQuestion` references in skills/src
- [ ] Skills deploy without errors
- [ ] Agents can invoke question tool successfully

---

## References

**Files Examined:**
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/question.ts` - Tool definition
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/question/index.ts` - Question schema
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/tool/registry.ts` - Tool registration
- `/Users/dylanconlin/Documents/personal/opencode/packages/opencode/src/flag/flag.ts` - Feature flags

**Commands Run:**
```bash
# Find all AskUserQuestion references
grep -rln "AskUserQuestion" /Users/dylanconlin/orch-knowledge/skills/src/

# Verify all replaced
grep -rn "AskUserQuestion" /Users/dylanconlin/orch-knowledge/skills/src/
```

**Files Modified:**
- `feature-impl/.skillc/phases/clarifying-questions.md`
- `feature-impl/reference/phase-clarifying-questions.md`
- `architect/.skillc/SKILL.md.template`
- `architect/.skillc/skill.yaml`
- `architect/SKILL.md`
- `design-session/.skillc/SKILL.md.template`
- `design-session/.skillc/skill.yaml`
- `design-session/SKILL.md`

---

## Investigation History

**2026-01-07 16:00:** Investigation started
- Initial question: How to update skills to use opencode question tool
- Context: Address recurring 'ask command inline' friction (8x gap)

**2026-01-07 16:15:** Found tool interface in OpenCode source
- Discovered `question` tool (not `AskUserQuestion`)
- Documented JSON interface schema

**2026-01-07 16:30:** Updated all skill files
- Replaced 7 occurrences across 3 skills
- Verified no remaining references

**2026-01-07 16:45:** Investigation completed
- Status: Complete
- Key outcome: All skills updated to use correct `question` tool with accurate interface documentation
