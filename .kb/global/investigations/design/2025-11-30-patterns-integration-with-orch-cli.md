# Investigation: Patterns Integration with orch-cli

**TLDR:** Question: How should docs/patterns/ (7 extracted patterns) integrate with orch-cli? Answer: Keep patterns documentation-only with optional `orch patterns` lookup command. High confidence (85%) - patterns are designed for cold-start human/agent reference, not runtime injection.

---

## Design Question

How should the 7 extracted patterns in docs/patterns/ relate to the orch-cli tooling and orchestrator skill?

Options considered:
1. Reference patterns in orchestrator skill
2. Add `orch patterns` CLI command
3. Inject patterns into spawn prompts
4. Keep patterns documentation-only

---

## Problem Framing

### Success Criteria

- Patterns remain accessible for cold-start resumption (core purpose)
- No redundant duplication between patterns and skill
- Minimal maintenance burden (cross-repo sync already complex)
- Clear separation of concerns (guidance vs. enforcement)

### Constraints

- **Spawn context is already large** - Orchestrator skill is ~777 lines, embedding additional patterns would bloat context
- **Cross-repo sync exists** - orch-cli already syncs with orch-knowledge for skill updates
- **Patterns serve different audiences** - Some for orchestrator, some for workers, some for both
- **Cold-start design** - Patterns are written to be read by fresh Claude instances, not injected programmatically

### Scope

**IN:** How patterns relate to CLI commands, orchestrator skill, spawn workflow
**OUT:** Changes to pattern content itself, new patterns to add

---

## Exploration

### Approach A: Reference Patterns in Orchestrator Skill

**Mechanism:** Add pointer comments/links to pattern files within orchestrator skill sections

**Example:**
```markdown
## Agent Salvage vs Fresh

**See full pattern:** `docs/patterns/agent-salvage-vs-fresh.md`

[Brief 2-3 sentence summary of when to apply]
```

**Pros:**
- Orchestrator knows patterns exist
- Minimal duplication (summary + link)
- Patterns can evolve independently

**Cons:**
- Orchestrators can't actually read linked files during session (file reads cost context)
- Creates maintenance burden: skill must stay in sync with pattern summaries
- Partial information worse than full pattern or no reference
- Orchestrator skill already has inline versions of key guidance (would become redundant)

**Complexity:** Low to implement, Medium to maintain

---

### Approach B: Add `orch patterns` CLI Command

**Mechanism:** CLI command to list, view, or search patterns

**Example usage:**
```bash
# List all patterns
orch patterns list

# View specific pattern
orch patterns show agent-salvage-vs-fresh

# Search patterns by keyword
orch patterns search "validation"
```

**Pros:**
- Human-accessible via CLI (Dylan can look up patterns)
- Agents can be told "run `orch patterns show X` if you need guidance"
- No context bloat (patterns fetched on demand)
- Patterns remain authoritative source (not duplicated)

**Cons:**
- Adds CLI complexity (another command to maintain)
- Agents don't naturally think to run commands for guidance
- Patterns in docs/ already accessible via normal file reads
- Limited value add over `cat docs/patterns/X.md`

**Complexity:** Medium to implement, Low to maintain

---

### Approach C: Inject Patterns into Spawn Prompts

**Mechanism:** Embed relevant pattern content into SPAWN_CONTEXT.md based on skill type

**Example:** When spawning `feature-impl` with `--validation multi-phase`, inject `multi-phase-feature-validation.md` content

**Pros:**
- Workers get guidance contextually when they need it
- No additional file reads needed
- Pattern applied automatically for relevant tasks

**Cons:**
- Significant context bloat (patterns are 200-450 lines each)
- Spawn prompts already large (~600 lines with skill guidance)
- Difficult to determine which patterns apply to which spawns
- Duplicates content already in skill templates (skills already have relevant guidance)
- Creates tension with "skill guidance is the authority" principle

**Complexity:** High to implement (pattern-to-skill mapping), High to maintain (two sources of truth)

---

### Approach D: Keep Patterns Documentation-Only

**Mechanism:** Patterns remain in docs/patterns/ as reference documentation, not integrated with CLI or skills

**How it works:**
- Patterns are for human reference (Dylan reading docs)
- Patterns are for explicit agent guidance ("read docs/patterns/X.md")
- Orchestrator skill contains the operational guidance (no pattern references needed)
- Pattern concepts are already embedded in skill workflows

**Pros:**
- Simplest approach (no changes needed)
- Clear separation: patterns = reusable knowledge, skills = operational guidance
- No maintenance burden beyond patterns themselves
- Patterns can be read when needed, not forced into context

**Cons:**
- Patterns might not be discovered (agents don't know to look for them)
- No programmatic access (can't list/search via CLI)
- Dylan must remember patterns exist and their locations

**Complexity:** None to implement, Low to maintain

---

## Synthesis

### Recommendation: Approach D (Documentation-Only) + Optional B (CLI Lookup)

**Primary recommendation:** Keep patterns documentation-only

**Why:**

1. **Patterns serve cold-start resumption, not runtime injection**
   - The patterns are designed for a fresh Claude to read when starting work
   - They're not meant to be injected mid-conversation
   - Spawned agents already get skill guidance which contains the operational equivalent

2. **Skills already contain pattern concepts**
   - Orchestrator skill has "salvage vs fresh" decision tree inline
   - feature-impl skill has validation checkpoint guidance
   - architect skill has directive guidance embedded
   - Adding pattern references would create redundancy, not add value

3. **Avoid two sources of truth**
   - If patterns are referenced in skills, both must stay in sync
   - Current approach: patterns capture reusable knowledge, skills apply it to specific contexts
   - This separation is healthy

4. **Context budget is precious**
   - Spawn context is already ~600+ lines
   - Adding patterns (200-450 lines each) would bloat significantly
   - Skills are the right granularity for spawned agents

**Optional enhancement:** Add `orch patterns list` command

- Low-cost addition (~50 lines of code)
- Helps Dylan discover available patterns
- Agents can be explicitly told to use it when debugging or investigating
- Doesn't affect normal spawn workflow

**Trade-offs accepted:**
- Patterns might be under-discovered (acceptable - they're reference material)
- No automatic pattern application (acceptable - skills handle this)

**When this would change:**
- If agents frequently ask "what patterns should I follow?" → consider pattern injection
- If pattern/skill drift becomes a problem → consider automated sync checking
- If Dylan requests it → implement `orch patterns` command

---

## Recommendations

⭐ **RECOMMENDED:** Documentation-only (Approach D)

- **Why:** Patterns and skills serve different purposes; skills already contain operational guidance derived from patterns; injection would bloat context without adding value
- **Trade-off:** Patterns may be under-discovered, but they're reference material for edge cases, not primary guidance
- **Expected outcome:** Clean separation between reusable knowledge (patterns) and operational guidance (skills); no additional maintenance burden

**Alternative: Add `orch patterns list` command (Approach B - partial)**
- **Pros:** Makes patterns discoverable via CLI; low implementation cost
- **Cons:** Limited value-add over direct file reading; adds another command
- **When to choose:** If Dylan wants CLI-based pattern discovery, implement as separate enhancement (not prerequisite for patterns integration decision)

**Not recommended: Pattern injection (Approach C)**
- Would bloat spawn context significantly
- Creates two sources of truth (patterns vs skills)
- Hard to determine pattern-to-spawn mapping
- Skills already provide contextual guidance

**Not recommended: Skill references (Approach A)**
- Partial information (summaries) worse than full patterns or no reference
- Creates maintenance burden (summaries must sync with patterns)
- Orchestrator skill already has inline guidance

---

## Confidence Assessment

**Current Confidence:** High (85%)

**Why this level?**
Thorough analysis of patterns content, skill structure, and spawn workflow. Clear understanding of design principles (cold-start, separation of concerns). Main uncertainty is whether pattern under-discovery becomes a problem in practice.

**What's certain:**
- ✅ Patterns are designed for cold-start reading, not injection
- ✅ Skills already contain operational versions of pattern concepts
- ✅ Spawn context is already large (~600 lines)
- ✅ Pattern injection would bloat context significantly (200-450 lines each)

**What's uncertain:**
- ⚠️ Whether patterns get discovered and used in practice (need to observe)
- ⚠️ Whether skill/pattern drift becomes a problem over time
- ⚠️ Whether Dylan would find `orch patterns` command valuable

**What would increase confidence to 95%+:**
- Observe pattern usage over 5+ sessions
- Confirm skills adequately cover pattern concepts (spot-check)
- Get Dylan's feedback on discoverability

---

## Amnesia-Resilience

**For future Claude instances:**

Patterns in docs/patterns/ are **reference documentation**, not runtime configuration.

**Checklist:**
- [ ] Don't look for patterns in spawn context (they're not there by design)
- [ ] Read patterns directly from docs/patterns/ when investigating a topic
- [ ] Look to skills for operational guidance during spawn work
- [ ] Patterns are for understanding "why", skills are for understanding "how"

**Red flag test:** Are you trying to inject pattern content into spawn prompts?

If yes → Stop. Skills already contain operational guidance. Patterns are reference material, not injection targets.

**Success indicators:**
- ✅ Patterns remain stable (not modified for injection)
- ✅ Skills continue to evolve with operational guidance
- ✅ No pattern/skill duplication or sync issues

---

## Feature List Review

**Location:** No features.json found in orch-cli project.

This investigation doesn't require feature list changes. The recommendation is to NOT add features (documentation-only approach).

If `orch patterns` command is later desired, it would be a separate enhancement tracked in beads or a project board.

---

## Related Artifacts

- **Patterns location:** `/Users/dylanconlin/orch-cli/docs/patterns/`
- **Orchestrator skill:** `/Users/dylanconlin/.claude/skills/orchestrator/SKILL.md`
- **Existing patterns.py:** `/Users/dylanconlin/orch-cli/src/orch/patterns.py` (CDD violation checker - different purpose)

---

**Created:** 2025-11-30
**Workspace:** `.orch/workspace/analyze-relationship-between-30nov/`
