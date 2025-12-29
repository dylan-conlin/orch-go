<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Cross-project change visibility requires a unified changelog mechanism; existing pattern (`orch complete` CLI detection) can be extended but skill changes need semantic parsing to surface behavioral meaning.

**Evidence:** Found active issue orch-go-aqo8 where agent implemented wrong hook system (Claude Code vs OpenCode) because skill change wasn't visible; `detectNewCLICommands()` pattern exists in main.go:3730; dashboard has established section pattern.

**Knowledge:** Visibility gaps cause silent failures - changes are made but consumers don't know. Solution needs (1) detection, (2) semantic categorization (breaking/behavioral/docs), (3) aggregation across repos.

**Next:** Create Epic with children for `orch changelog` command + dashboard integration.

<!--
Example D.E.K.N.:
Delta: Test-running guidance is missing from spawn prompts and CLAUDE.md.
Evidence: Searched 5 agent sessions - none ran tests; guidance exists in separate docs but isn't loaded.
Knowledge: Agents follow documentation literally; guidance must be in loaded context to be followed.
Next: Add test-running instruction to SPAWN_CONTEXT.md template.

Guidelines:
- Keep each line to ONE sentence
- Delta answers "What did we find?"
- Evidence answers "How do we know?"
- Knowledge answers "What does this mean?"
- Next answers "What should happen now?"
- Enable 30-second understanding for fresh Claude
-->

---

# Investigation: Skill Changelog Cross Project Change

**Question:** How should Dylan get visibility into skill changes, CLI additions, and knowledge updates with semantic context (not just file diffs)?

**Started:** 2025-12-29
**Updated:** 2025-12-29
**Owner:** og-work-skill-changelog-cross-29dec
**Phase:** Complete
**Next Step:** Create Epic with implementation tasks
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** N/A
**Supersedes:** N/A (complements 2025-12-23-inv-audit-recent-skill-changes which was one-time, not ongoing)
**Superseded-By:** N/A

---

## Findings

### Finding 1: Existing Pattern for CLI Command Detection

**Evidence:** `detectNewCLICommands()` function in main.go:3730-3775 already detects newly added CLI command files by:
- Parsing git diff for "A" (added) status files in last 5 commits
- Checking if files are in `cmd/orch/*.go`
- Verifying they contain cobra.Command definitions
- Alerting at `orch complete` time about documentation needs

**Source:** `/Users/dylanconlin/Documents/personal/orch-go/cmd/orch/main.go:3730-3775`

**Significance:** Proves the detection + alert pattern works. Can be extended to skill changes. The key insight is coupling detection to workflow (completion time) rather than requiring manual checks.

---

### Finding 2: Skill Changes Currently Invisible Until Failure

**Evidence:** Active issue `orch-go-aqo8` documents a case where an agent implemented Claude Code hooks (wrong platform) because:
- The orchestrator skill referenced Claude Code hook patterns
- No visibility mechanism surfaced that a skill change was needed
- The error was only discovered when implementation failed

kn constraint exists: "Orch uses OpenCode, not Claude Code. Hook system is different"

**Source:** `bd show orch-go-aqo8`, kn entries from spawn context

**Significance:** Demonstrates the visibility gap has real cost - wasted agent work, confusion, need for remediation. The change visibility need is validated by concrete failure.

---

### Finding 3: Multiple Aggregation Sources Already Exist

**Evidence:** 
- `kb chronicle "topic"` aggregates: investigations, decisions, kn entries, git commits, beads issues
- `skillc deploy` generates checksums and tracks source paths (visible in SKILL.md headers)
- Dashboard has established patterns for aggregation sections (PendingReviewsSection, ReadyQueueSection, NeedsAttention)

**Source:** 
- `kb chronicle --help` output
- `~/.claude/skills/meta/orchestrator/SKILL.md` skillc headers
- `web/src/routes/+page.svelte` component imports

**Significance:** The aggregation capability exists in pieces. Need to unify into a "changes" view that pulls from: git (skill repos), skillc (deploy metadata), kb/kn (knowledge changes).

---

### Finding 4: Semantic Categorization Needed Beyond File Diffs

**Evidence:** Prior investigation `2025-12-27-inv-skill-change-taxonomy` established 6 categories of skill changes:
- Documentation-only (~30%)
- Single-skill behavioral (~25%)
- Single-skill structural (~15%)
- Cross-skill refactor (~15%)
- Infrastructure coupling (~10%)
- New skill creation (~5%)

And two axes: blast radius (local/cross-skill/infrastructure) × change type (documentation/behavioral/structural)

**Source:** `.kb/investigations/2025-12-27-inv-skill-change-taxonomy.md`

**Significance:** Raw git diffs don't convey meaning. "File X changed" is less useful than "Breaking behavioral change to feature-impl skill affects all spawns". Need semantic parsing layer.

---

## Synthesis

**Key Insights:**

1. **Detection + Workflow Coupling Works** - The CLI command detection pattern (Finding 1) succeeds because it surfaces information at the right time (completion). Extend this pattern to skill changes rather than requiring manual checks.

2. **Visibility Prevents Wasted Work** - The Claude Code/OpenCode confusion (Finding 2) cost agent time and required remediation. Proactive visibility of what changed prevents these failures. The ROI is clear.

3. **Semantic Layer Required** - File diffs (Finding 3) need the taxonomy layer (Finding 4) to be actionable. A changelog that says "BREAKING: investigation skill now requires workspace" is 10x more useful than "investigation.md modified".

**Answer to Investigation Question:**

Dylan should get change visibility through a new `orch changelog` command that:

1. **Aggregates changes** from skill repos (orch-knowledge/skills/), CLI (orch-go/cmd/), and kb (any .kb/)
2. **Parses semantic meaning** using the taxonomy from Finding 4 (documentation/behavioral/structural × blast radius)
3. **Surfaces at workflow points** like `orch complete` or via dashboard section
4. **Supports cross-project queries** using the ecosystem repo list already in spawn context

This builds on proven patterns (CLI detection) rather than inventing new mechanisms.

---

## Structured Uncertainty

**What's tested:**

- ✅ CLI detection pattern exists and works (verified: read main.go:3730-3775, pattern deployed)
- ✅ Skill taxonomy is documented (verified: read 2025-12-27-inv-skill-change-taxonomy.md)
- ✅ Dashboard section pattern established (verified: read +page.svelte component structure)

**What's untested:**

- ⚠️ Semantic parsing from git diffs to taxonomy categories (requires implementation)
- ⚠️ Performance of cross-repo aggregation (may need caching for large history)
- ⚠️ User workflow fit - will Dylan actually check a changelog, or need push notifications?

**What would change this:**

- Finding would be wrong if semantic parsing proves too unreliable (false positives/negatives)
- Finding would be wrong if cross-repo aggregation is too slow for interactive use
- Finding would be wrong if the visibility gap is already solved by kb chronicle (test: does kb chronicle surface skill changes?)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**`orch changelog` CLI Command** - New command that aggregates and semantically categorizes changes across skill repos, CLI, and kb.

**Why this approach:**
- Builds on proven detection pattern (Finding 1 - CLI command detection)
- CLI-first allows testing before dashboard integration
- Cross-project by nature (uses ecosystem repo list from spawn context)
- Directly addresses semantic meaning need (Finding 4)

**Trade-offs accepted:**
- Pull model (user runs command) vs push (auto-notification) - start with pull, add push later
- Initial implementation may have imperfect semantic parsing - iterate based on usage

**Implementation sequence:**
1. **Core command** - `orch changelog [--days 7]` aggregates git commits from skill/kb repos
2. **Semantic parser** - Classify commits using taxonomy (documentation/behavioral/structural)
3. **Dashboard integration** - Add `/api/changelog` endpoint and "Recent Changes" section
4. **Workflow hooks** - Surface in `orch complete` like CLI command detection

### Alternative Approaches Considered

**Option B: Dashboard-only**
- **Pros:** Visual, always-visible
- **Cons:** Can't be called by agents; harder to test; doesn't work for CLI-only workflows
- **When to use instead:** After Option A ships, as the display layer

**Option C: skillc deploy hooks**
- **Pros:** Captures at source (when skill is deployed)
- **Cons:** Only covers skills; requires skillc modification; doesn't aggregate across repos
- **When to use instead:** As enhancement to Option A for real-time skill change tracking

**Option D: Extend kb chronicle**
- **Pros:** Already does temporal aggregation
- **Cons:** Topic-based query, not "what changed recently"; doesn't surface by default
- **When to use instead:** For deep dives into specific topics, not overview visibility

**Rationale for recommendation:** Option A (CLI command) provides the foundation that other options build on. It's testable, scriptable, and follows existing patterns. Dashboard and hooks are enhancements, not replacements.

---

### Implementation Details

**What to implement first:**
- Core `orch changelog` command with git log parsing from ecosystem repos
- Category assignment based on file paths (skills/ → skill, .kb/ → kb, cmd/ → cli)
- Basic semantic parsing from commit messages (feat: → behavioral, docs: → documentation)

**Things to watch out for:**
- ⚠️ Cross-repo git access - need to handle repos that aren't cloned
- ⚠️ Conventional commit format not always followed - need fallback parsing
- ⚠️ Performance for large history - consider --since flag as default (7 days)

**Areas needing further investigation:**
- How to detect breaking changes beyond "BREAKING:" prefix
- Whether to track deployed skills vs source skills (skillc checksum mismatch)
- Integration with daemon - should changelog entries trigger notifications?

**Success criteria:**
- ✅ `orch changelog` shows skill changes from last 7 days with semantic categories
- ✅ Dashboard "Recent Changes" section displays aggregated feed
- ✅ Claude Code/OpenCode style confusion (Finding 2) would have been surfaced
- ✅ Cross-project visibility works (changes in orch-knowledge visible from orch-go)

---

## References

**Files Examined:**
- `cmd/orch/main.go:3730-3775` - CLI command detection pattern (detectNewCLICommands)
- `web/src/routes/+page.svelte` - Dashboard section patterns
- `~/.claude/skills/meta/orchestrator/SKILL.md` - skillc deployment headers
- `.kb/investigations/2025-12-27-inv-skill-change-taxonomy.md` - Semantic taxonomy for skill changes
- `.kb/investigations/2025-12-23-inv-audit-recent-skill-changes-their.md` - Prior one-time audit

**Commands Run:**
```bash
# Check for existing changelog patterns
kb context "changelog"

# Check skillc capabilities
skillc --help

# Find related issues
bd list | grep -i skill

# Check kb chronicle capabilities
kb chronicle --help
```

**External Documentation:**
- N/A

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2025-12-27-inv-skill-change-taxonomy.md` - Provides semantic categories for classification
- **Investigation:** `.kb/investigations/2025-12-23-inv-audit-recent-skill-changes-their.md` - Prior one-time audit, this makes it ongoing
- **Issue:** `orch-go-aqo8` - Active example of visibility gap (Claude Code vs OpenCode confusion)

---

## Investigation History

**2025-12-29 13:00:** Investigation started
- Initial question: How should Dylan get visibility into skill changes with semantic context?
- Context: Spawned from design-session task; motivated by Claude Code vs OpenCode hooks confusion

**2025-12-29 13:15:** Context gathering complete
- Found existing CLI detection pattern in orch complete
- Found semantic taxonomy in prior skill-change-taxonomy investigation
- Found active issue (orch-go-aqo8) demonstrating the visibility gap

**2025-12-29 13:30:** Design synthesis complete
- Determined output type: Investigation → Epic
- Recommended approach: `orch changelog` CLI command + dashboard integration

**2025-12-29 13:45:** Investigation completed
- Status: Complete
- Key outcome: Recommend `orch changelog` command building on existing CLI detection pattern with semantic categorization from taxonomy investigation
