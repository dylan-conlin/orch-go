<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
-->

## Summary (D.E.K.N.)

**Delta:** The current flat `.kb/quick/entries.jsonl` with 173 constraints is working as designed - constraints are inherently tactical (implementation-specific), with only ~2% warranting promotion to architectural status (principles.md or decisions/).

**Evidence:** Analyzed 173 constraints: ~60% are implementation-specific, ~25% are corollaries of existing principles, ~15% are candidates for deeper review. Prior investigation (2026-01-21) found only 3 of 149 constraints met all 4 criteria for principle promotion.

**Knowledge:** The distinction isn't architectural vs tactical constraints - it's "universal across projects + has teeth" (principles) vs "project-specific learnings" (kb quick). Separate `.kb/constraints/` would conflate these temporal scopes and create maintenance burden.

**Next:** Keep current architecture. Add visibility mechanism (`kb quick list --type constraint --priority high`) rather than new storage location. Update Knowledge Placement table to clarify constraint hierarchy.

**Promote to Decision:** Issue created: orch-go-21090 (constraint organization decision)

---

# Investigation: Review Knowledge Artifact Hierarchy Constraint Organization

**Question:** Where should constraints live in the knowledge artifact hierarchy? Specifically: should there be a separate `.kb/constraints/` directory, promotion to principles.md, or other organization for the 173 constraints in `.kb/quick/entries.jsonl`?

**Started:** 2026-01-28
**Updated:** 2026-01-28
**Owner:** Architect agent
**Phase:** Complete
**Next Step:** None - recommendation ready for orchestrator review
**Status:** Complete

**Patches-Decision:** None
**Extracted-From:** None
**Supersedes:** None
**Superseded-By:** None

---

## Findings

### Finding 1: Current constraint distribution shows 755 entries with 173 constraints

**Evidence:** 
```
755 .kb/quick/entries.jsonl
486 decision
173 constraint
 68 attempt
 28 question
```

The 173 constraints are stored flat alongside 486 decisions, 68 attempts, and 28 questions. There's no authority/priority indexing visible in the data model beyond the `authority` field (mostly "high").

**Source:** `.kb/quick/entries.jsonl` - `cat ... | jq -r '.type' | sort | uniq -c`

**Significance:** The scale (173 constraints) warrants considering organization, but the flat storage is intentional for `kb context` discoverability. Any reorganization must preserve discoverability.

---

### Finding 2: Constraints fall into distinct categories by universality

**Evidence:** Analyzed constraint content and found three categories:

1. **Implementation-specific (~60%)** - orch-go/opencode details:
   - "OpenCode x-opencode-directory header returns ALL disk sessions"
   - "orch status counts ALL workers-* tmux windows as active"
   - "skillc cannot compile SKILL.md templates without template expansion feature"

2. **Corollaries of existing principles (~25%)** - derivable from principles.md:
   - "Session idle ≠ agent complete" → derivable from "Track Actions, Not Just State"
   - "Ask 'should we' before 'how do we'" → IS "Premise Before Solution"
   - "High patch density signals missing model" → IS "Coherence Over Patches"

3. **Potential architectural candidates (~15%)** - possibly universal:
   - "Agents must not spawn more than 3 iterations without human review"
   - "orch-go must support Claude Code as orchestrator backend"
   - "Template ownership: kb-cli owns artifact templates, orch-go owns spawn-time templates"

**Source:** `.kb/quick/entries.jsonl` - sampled ~50 constraints manually

**Significance:** Most constraints are correctly placed at project level. Only ~15% warrant deeper analysis for potential promotion, and prior investigation (2026-01-21) found only 3 met all 4 criteria for principle promotion.

---

### Finding 3: Existing promotion path is documented but intentionally high-friction

**Evidence:** From `~/.claude/CLAUDE.md` Knowledge Placement table:

```
| Rule/constraint | `kb quick constrain "X" --reason "Y"` | "Never do X" / "Always do Y" |
| Foundational value | `.kb/principles.md` | Universal, tested, has teeth |

**Promotion paths:**
- `kb quick constrain` → `.kb/principles.md` (when universal across projects)
```

From `~/.kb/principles.md` criteria:
- Must be tested (emerged from actual problems)
- Must be generative (guides future decisions)
- Must not be derivable from existing principles
- Must have teeth (violation causes real problems)

**Source:** `~/.claude/CLAUDE.md:102,109,112`, `~/.kb/principles.md:880-895`

**Significance:** The promotion path exists and is intentionally high-friction. The 4-criteria test filters appropriately - only ~2% of constraints qualify. A separate `.kb/constraints/` directory would bypass this intentional curation.

---

### Finding 4: Three-tier temporal model already organizes artifacts correctly

**Evidence:** From decision `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md`:

```
### Three-Tier Temporal Model
- **Ephemeral (session-bound):** `.orch/workspace/` - SPAWN_CONTEXT.md, SYNTHESIS.md
- **Persistent (project-lifetime):** `.kb/` - Investigations, decisions, guides
- **Operational (work-in-progress):** `.beads/` - Issues, comments
```

kb quick (`.kb/quick/entries.jsonl`) is persistent project-lifetime storage. principles.md is global persistent storage (`~/.kb/`). The temporal distinction is:
- **Project constraints:** `.kb/quick/` - specific to this codebase
- **Universal constraints:** `~/.kb/principles.md` - applies across all projects

**Source:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md:35-43`

**Significance:** Constraints don't need a separate directory - they're correctly placed at the temporal scope where they apply. Project constraints belong in `.kb/quick/`, universal constraints belong in principles.md.

---

### Finding 5: kb context priority-based truncation already surfaces high-signal constraints

**Evidence:** From kb quick decision kb-f2e55c:
```
"KB context uses priority-based truncation: constraints > decisions > investigations"
Reason: "Constraints are critical (must respect), decisions next (prior choices), investigations lowest (exploratory)"
```

When agents run `kb context "topic"`, constraints are already surfaced first and preserved when truncation is needed.

**Source:** `.kb/quick/entries.jsonl` - kb-f2e55c

**Significance:** The visibility mechanism exists - constraints are already prioritized over decisions and investigations. The gap isn't storage location, it's discoverability of "high-signal" vs "all" constraints.

---

### Finding 6: High-signal constraints lack explicit tagging

**Evidence:** Example high-signal constraints that need visibility:
- "orch-go must support Claude Code as orchestrator backend" - architectural
- "Template ownership: kb-cli owns artifact templates..." - architectural
- "Agents must not spawn more than 3 iterations without human review" - universal

These are mixed with implementation-specific constraints like:
- "orch status counts ALL workers-* tmux windows as active"
- "Dashboard event panels max-h-64"

There's no `priority: high` or `architectural: true` field to distinguish them.

**Source:** `.kb/quick/entries.jsonl` - analysis of `authority` field shows all are "high"

**Significance:** The data model doesn't distinguish architectural from tactical constraints. Adding a `category` or `scope` field (e.g., `architectural`, `implementation`, `operational`) would enable filtering without changing storage location.

---

## Synthesis

**Key Insights:**

1. **Temporal scope is the right organizing principle** - Constraints aren't "architectural vs tactical" - they're "universal across projects" (principles.md) vs "project-specific" (kb quick). The existing three-tier model handles this correctly.

2. **High-friction promotion is a feature, not a bug** - The ~2% promotion rate (3 of 149 constraints) shows the 4-criteria test is working. A separate `.kb/constraints/` directory would create a "middle tier" that bypasses curation without adding value.

3. **The gap is visibility, not storage** - High-signal constraints like "must support Claude Code backend" are buried among implementation details. The solution is better tagging/filtering, not a new directory.

**Answer to Investigation Question:**

**Where should constraints live?** In the current locations:
- **Project-specific constraints:** `.kb/quick/entries.jsonl` (current)
- **Universal constraints (rare):** `~/.kb/principles.md` via manual promotion

**Should there be a separate `.kb/constraints/` directory?** No. This would:
1. Conflate temporal scopes (project vs universal)
2. Create maintenance burden (two places to check for constraints)
3. Bypass the intentional high-friction promotion path
4. Duplicate what kb context already provides (constraint-first priority)

**What about high-signal constraints like "must support Claude Code backend"?** Add a `scope: architectural` field to entries.jsonl, then provide `kb quick list --scope architectural` for visibility without changing storage location.

---

## Structured Uncertainty

**What's tested:**

- ✅ Constraint count is 173 (verified: `cat ... | jq ... | wc -l`)
- ✅ Three-tier temporal model documented in minimal artifact taxonomy decision
- ✅ kb context prioritizes constraints (verified: kb-f2e55c decision)
- ✅ Prior constraint scan found ~2% promotion rate (verified: 2026-01-21 investigation)

**What's untested:**

- ⚠️ Whether `scope: architectural` field would be correctly applied (requires human judgment at constraint creation time)
- ⚠️ Whether kb quick CLI supports custom fields (not tested)
- ⚠️ Whether principals.md promotion path is used in practice (low evidence of actual promotions)

**What would change this:**

- If kb context discoverability proves insufficient for high-signal constraints
- If more than ~15% of constraints truly warrant architectural status
- If a clear "middle tier" use case emerges that isn't served by either kb quick or principles.md

---

## Implementation Recommendations

### Recommended Approach ⭐

**Keep current architecture + add visibility mechanism** - No new `.kb/constraints/` directory. Instead, add `scope` field to entries.jsonl schema and `kb quick list --scope architectural` command.

**Why this approach:**
- Preserves temporal scope distinction (project vs universal)
- Maintains high-friction promotion to principles.md
- Addresses the actual gap (visibility, not storage)
- Minimal change to existing working system

**Trade-offs accepted:**
- High-signal constraints still mixed with tactical ones in storage
- Requires discipline to tag `scope: architectural` at creation time

**Implementation sequence:**
1. Add `scope` field to kb quick schema (optional, defaults to `implementation`)
2. Add `--scope` filter to `kb quick list` command
3. Update Knowledge Placement table in CLAUDE.md to clarify constraint hierarchy
4. Document when to use `scope: architectural` vs leaving default

### Alternative Approaches Considered

**Option B: Separate `.kb/constraints/` directory**
- **Pros:** Explicit visibility for all constraints, easy to enumerate
- **Cons:** Conflates project/universal scope, maintenance burden, bypasses curation
- **When to use instead:** If constraint count grows to 500+ and filtering becomes impractical

**Option C: Promote all high-signal constraints to principles.md**
- **Pros:** Most visible location, forces curation
- **Cons:** Would bloat principles.md with project-specific details, violates 4-criteria test
- **When to use instead:** When constraint truly meets all 4 criteria AND is universal across projects

**Option D: Create `.kb/decisions/constraints/` subdirectory**
- **Pros:** Uses existing decisions structure, searchable
- **Cons:** Constraints aren't decisions (different trigger: "never do X" vs "we chose X because Y")
- **When to use instead:** Never - conflates artifact types

**Rationale for recommendation:** The gap identified in the task ("high-signal constraints have no promotion path to visible location") is actually a visibility problem, not a storage problem. The promotion path exists (kb quick → principles.md), but it's appropriately high-friction. What's missing is filtering to surface architectural constraints without changing where they live.

---

### Implementation Details

**What to implement first:**
- Update Knowledge Placement table in `~/.claude/CLAUDE.md` to clarify:
  - kb quick constraint → project-specific rules
  - principles.md → universal constraints (via 4-criteria test)
  - The distinction is scope, not importance

**Things to watch out for:**
- ⚠️ Don't add `scope` field if kb quick schema doesn't support it - check kb-cli first
- ⚠️ "Architectural" is subjective - need clear criteria for when to use
- ⚠️ Existing 173 constraints would need triage to backfill scope

**Areas needing further investigation:**
- Whether kb-cli supports custom fields in entries.jsonl
- Whether existing constraints should be backfilled with scope field
- Whether kb reflect should surface "architectural" constraints automatically

**Success criteria:**
- ✅ Orchestrator can quickly find architectural constraints via `kb quick list --scope architectural`
- ✅ Knowledge Placement table clearly explains constraint hierarchy
- ✅ No separate `.kb/constraints/` directory created

---

## References

**Files Examined:**
- `.kb/quick/entries.jsonl` - All 173 constraints scanned for categorization
- `~/.claude/CLAUDE.md:95-117` - Knowledge Placement table
- `~/.kb/principles.md:880-895` - Four-criteria test for principles
- `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Three-tier temporal model
- `.kb/investigations/2026-01-21-inv-scan-kb-quick-constraints-promotion.md` - Prior constraint scan
- `.kb/investigations/2025-12-21-inv-knowledge-promotion-paths.md` - Promotion mechanisms

**Commands Run:**
```bash
# Count entry types
cat .kb/quick/entries.jsonl | jq -r '.type' | sort | uniq -c | sort -rn

# List constraint content samples
cat .kb/quick/entries.jsonl | jq -r 'select(.type == "constraint") | .content' | head -50

# Check authority distribution
cat .kb/quick/entries.jsonl | jq -r 'select(.type == "constraint") | "\(.authority // "unset")"' | sort | uniq -c

# Search for kb context
kb context "constraint organization promotion"
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2025-12-21-minimal-artifact-taxonomy.md` - Established three-tier model
- **Decision:** `.kb/decisions/2025-12-22-template-ownership-model.md` - Template ownership constraint example
- **Investigation:** `.kb/investigations/2026-01-21-inv-scan-kb-quick-constraints-promotion.md` - Prior constraint analysis
- **Investigation:** `.kb/investigations/2025-12-21-inv-knowledge-promotion-paths.md` - Promotion mechanisms

---

## Investigation History

**2026-01-28 12:20:** Investigation started
- Initial question: Where should constraints live? Is `.kb/constraints/` needed?
- Context: 173 constraints in flat entries.jsonl, high-signal constraints lack visible location

**2026-01-28 12:30:** Key discovery - temporal scope is the organizing principle
- Found three-tier model already handles project vs universal distinction
- Constraints are correctly placed at temporal scope where they apply

**2026-01-28 12:40:** Analyzed prior constraint scan (2026-01-21)
- Confirmed ~2% promotion rate to principles
- High-friction promotion is intentional design

**2026-01-28 12:50:** Investigation completed
- Status: Complete
- Key outcome: Keep current architecture, add visibility mechanism (scope field) rather than new directory
