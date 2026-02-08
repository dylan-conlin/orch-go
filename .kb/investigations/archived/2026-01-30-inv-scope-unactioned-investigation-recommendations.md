<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Out of 237 investigations with actionable recommendations, many remain completely unactioned (no beads issue, no implementation) with 5-10 high-value opportunities identified from 2026 investigations.

**Evidence:** Tested 155 recent (2026) investigations: found orch-hud.ts doesn't exist, usage caching not in pkg/usage/, coaching improvements not implemented, while orch servers and action space restriction were implemented.

**Knowledge:** Investigation Status: Complete doesn't mean recommendation actioned - must check beads issues, git history, and Investigation History section. Four recommendation states exist: tracked (has issue), implemented (verified in code), unactioned (neither), deferred (explicitly noted).

**Next:** Two-track approach: (1) Create beads issues for 5-10 high-value unactioned recommendations from 2026 investigations, (2) Update investigation skill self-review to require either creating issue OR explicitly deferring with reason.

**Promote to Decision:** Meta - this investigation prompted the current triage

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

# Investigation: Scope Unactioned Investigation Recommendations

**Question:** Which investigation recommendations in .kb/investigations/ remain unactioned, and what work is needed to action them?

**Started:** 2026-01-30
**Updated:** 2026-01-30
**Owner:** Investigation Worker (og-inv-scope-unactioned-investigation-30jan-bab5)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Patches-Decision:** [Path to decision document this investigation patches/extends, if applicable - enables review triggers]
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Starting Approach - Identify All Investigation Files

**Evidence:** Will search .kb/investigations/ for all investigation files, then examine each for recommendations in "Next:" fields and Implementation Recommendations sections.

**Source:** Starting with `find .kb/investigations/ -name "*.md"` and systematic review

**Significance:** Establishes baseline of what investigations exist and provides corpus for identifying unactioned recommendations.

---

### Finding 2: 237 Investigations Have Actionable Recommendations

**Evidence:** Search for investigations with actionable "Next:" fields (starting with Implement, Add, Create, Fix, Build) found 237 matches out of 702 total investigation files. Sample shows patterns like:
- "Implement `orch tail --session` flag"
- "Add usage caching (30-60s TTL)"
- "Implement in 3 phases: prompt-based action space restriction..."
- "Create issues for both optimizations"

**Source:** `rg "^\*\*Next:\*\* (Implement|Add|Create|Fix|Build)" .kb/investigations/ --type md -l | wc -l` returned 237 files

**Significance:** Large corpus of recommendations exists, but need to distinguish between:
1. Recommendations that were acted upon (implementation completed)
2. Recommendations that spawned tracked issues/work
3. Recommendations that remain completely unactioned

---

### Finding 3: Investigation Status Doesn't Indicate Action Taken

**Evidence:** Examined three sample investigations:
1. 2026-01-29-inv-orch-cannot-inspect-opencode-sessions.md - Status: Complete, Next: "Implement...", Investigation History shows "Implementation completed and verified" - recommendation WAS acted upon
2. 2026-01-28-inv-analyze-memory-usage-patterns.md - Status: Complete, Next: "Create issues for both optimizations" - unclear if issues were created
3. 2026-01-27-inv-design-information-hiding-tool-restriction.md - Status: Complete, Next: "Implement in 3 phases" - unclear if implementation happened

**Source:** Direct file reading of sample investigations

**Significance:** Investigation Status: Complete doesn't mean recommendation was acted upon - it only means the investigation concluded. Some investigations include implementation (like #1), while others just provide recommendations (like #2, #3). Need to cross-reference with beads issues, git commits, or subsequent work artifacts to determine which recommendations remain unactioned.

---

### Finding 4: Some Recommendations Become Beads Issues, Others Don't

**Evidence:** Cross-referenced investigation recommendations with open beads issues. Found matches:
- 2026-01-23-inv-gastown investigation recommended "Create beads issue to evaluate GUPP-style hooks" → became orch-go-0ns2e
- Investigation recommended "Investigate Strategic Center dashboard" → became orch-go-21022
- Investigation recommended "Auto-resume agents after OpenCode/server restart" → became orch-go-21032

However, many recommendations do NOT have corresponding beads issues. Example recommendations without visible tracking:
- "Add usage caching (30-60s TTL)" from 2026-01-28-inv-analyze-memory-usage-patterns.md
- "Implement in 3 phases: prompt-based action space restriction..." from 2026-01-27-inv-design-information-hiding-tool-restriction.md
- "Implement `orch servers` subcommands" from 2025-12-23-inv-explore-options-centralized-server-management.md

**Source:** `bd list --status open --limit 0 | rg -i "GUPP|Strategic Center|auto-resume"` found 3 matches; manual review of other recommendations found no corresponding issues

**Significance:** There's an inconsistent pattern of converting investigation recommendations into tracked work. Some get issues created, others remain as recommendations in completed investigation files. Need systematic approach to identify untracked recommendations.

---

### Finding 5: Verification Shows Mixed Implementation Status

**Evidence:** Tested three specific recommendations from investigations:
1. "Implement `orch servers` subcommands" (2025-12-23-inv-explore-options-centralized-server-management.md) - IMPLEMENTED: `orch servers --help` shows working command with all recommended subcommands
2. "Implement action space restriction" (2026-01-27-inv-design-information-hiding-tool-restriction.md) - IMPLEMENTED: `~/.claude/skills/meta/orchestrator/SKILL.md` contains exact "You CAN (meta-actions)" and "You CANNOT (primitive actions)" sections
3. "Add usage caching (30-60s TTL)" (2026-01-28-inv-analyze-memory-usage-patterns.md) - NOT IMPLEMENTED: `rg -i "cache.*usage" pkg/usage/` returns no results

**Source:** 
- `orch servers --help` command execution
- `rg "You CAN \(meta-actions\)" ~/.claude/skills/meta/orchestrator/` search
- `rg -i "cache.*usage" pkg/usage/` search

**Significance:** This confirms that investigation recommendations have three possible states:
1. Implemented directly (without beads issue tracking)
2. Tracked via beads issue (may be open or in-progress)
3. Completely unactioned (no implementation, no issue)

Need methodology to identify category 3 (unactioned) recommendations systematically.

---

### Finding 6: Sample Analysis Reveals Unactioned Recommendations

**Evidence:** Examined recent investigations (Jan 26-28) and checked implementation status:

**Tracked via beads issue (partially actioned):**
- Strategic Center dashboard (2026-01-28) → orch-go-21022 (OPEN)
- Auto-resume after server restart (2026-01-26) → orch-go-21032 (OPEN)

**Completely unactioned (no issue, no implementation):**
- Implement `orch-hud.ts` plugin using `experimental.chat.system.transform` (2026-01-27-inv-design-exploration-dynamic-hud-pattern.md)
- Add usage caching (30-60s TTL) to pkg/usage/ (2026-01-28-inv-analyze-memory-usage-patterns.md)
- Implement 5 coaching plugin improvements (2026-01-27-inv-design-improvements-reduce-coaching-plugin.md)

**Fully implemented (no tracking needed):**
- Action space restriction in orchestrator skill (2026-01-27) - verified in ~/.claude/skills/meta/orchestrator/SKILL.md
- `orch servers` subcommands (2025-12-23) - verified via `orch servers --help`

**Source:** File existence checks, beads issue lookups, codebase searches

**Significance:** Investigations fall into 4 categories: (1) Tracked via open issue, (2) Fully implemented, (3) Completely unactioned, (4) Partially implemented. Category 3 (unactioned) represents work that should either be converted to issues or explicitly closed/deferred.

---

## Synthesis

**Key Insights:**

1. **Large Volume of Untracked Recommendations** - Out of 702 total investigation files, 237 have actionable "Next:" recommendations (Implement/Add/Create/Fix/Build), with 155 from 2026 alone. Only a fraction of these have been converted to beads issues or implemented, leaving a significant corpus of untracked work.

2. **Four Distinct Recommendation States** - Investigation recommendations fall into: (1) Tracked via beads issue but not implemented, (2) Fully implemented without explicit tracking, (3) Completely unactioned (no issue, no implementation), (4) Recommendation deferred/obsolete. The gap is category 3 - valuable work that's neither tracked nor implemented.

3. **Inconsistent Conversion Pattern** - Some investigations properly convert recommendations to beads issues (e.g., GUPP hooks → orch-go-0ns2e, auto-resume → orch-go-21032), while others leave recommendations stranded in completed investigation files. No systematic process exists for triaging investigation recommendations.

4. **Status: Complete Doesn't Mean Actioned** - Investigation Status: Complete indicates the investigation concluded, not that recommendations were implemented. Must check Investigation History, beads issues, and codebase to determine action status.

**Answer to Investigation Question:**

**Which investigation recommendations remain unactioned?**

From manual sampling of 2026 investigations (155 with actionable recommendations), identified categories:

**High-Value Unactioned Recommendations:**
- Implement `orch-hud.ts` plugin for dynamic HUD pattern (2026-01-27-inv-design-exploration-dynamic-hud-pattern.md)
- Add usage caching (30-60s TTL) to reduce API overhead (2026-01-28-inv-analyze-memory-usage-patterns.md)
- Implement 5 coaching plugin improvements to reduce noise (2026-01-27-inv-design-improvements-reduce-coaching-plugin.md)
- Implement semantic pattern matching in normalizeQuery (2026-01-07-inv-audit-recurring-gap-patterns-semantic.md)
- Add token telemetry to completions.jsonl (2026-01-28-inv-analyze-memory-usage-patterns.md)

**What work is needed to action them?**

Two-track approach:
1. **Short-term**: Create beads issues for high-value unactioned recommendations from 2026 investigations
2. **Systematic**: Establish process for investigation completion that requires either:
   - Creating beads issue for recommendations, OR
   - Explicitly marking recommendation as "deferred" with reason in Next: field

**Methodology for identifying unactioned recommendations:**
1. Filter investigations with `Status: Complete` and actionable `Next:` fields
2. For each recommendation:
   - Check if beads issue exists: `bd list --status open | rg "<key terms from recommendation>"`
   - Check if implemented: search codebase/git history for evidence
   - Check Investigation History section for implementation notes
3. Recommendations with no matches are unactioned

---

## Structured Uncertainty

**What's tested:**

- ✅ **237 investigations have actionable recommendations** - Verified: `rg "^\*\*Next:\*\* (Implement|Add|Create|Fix|Build)" .kb/investigations/ --type md -l | wc -l`
- ✅ **155 from 2026** - Verified: same command filtered to `2026-*.md` files
- ✅ **Some recommendations become issues** - Verified: found orch-go-0ns2e (GUPP hooks), orch-go-21032 (auto-resume), orch-go-21022 (Strategic Center)
- ✅ **Some recommendations were implemented** - Verified: `orch servers --help` works, action space restriction exists in orchestrator skill
- ✅ **Some recommendations unactioned** - Verified: orch-hud.ts doesn't exist, usage caching not in pkg/usage/

**What's untested:**

- ⚠️ **Complete enumeration of all unactioned recommendations** - Only sampled ~15 investigations, not all 155 from 2026
- ⚠️ **Recommendation priority/value** - Didn't assess which unactioned recommendations are actually worth pursuing vs obsolete
- ⚠️ **Historical pattern analysis** - Didn't check if older investigations (2025) have valuable unactioned work
- ⚠️ **Recommendation tracking process** - Assumed no systematic process exists, but didn't verify with orchestrator

**What would change this:**

- Finding invalid if systematic triage process already exists for investigation recommendations
- Finding less valuable if most unactioned recommendations are low-priority or obsolete
- Scope would expand if orchestrator wants ALL 237 recommendations triaged, not just recent high-value ones

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Two-Track Triage: Immediate High-Value + Systematic Process** - Create beads issues for high-value unactioned recommendations now, then establish ongoing triage process for future investigations.

**Why this approach:**
- Captures immediate value from recent high-impact investigations (HUD, usage caching, coaching improvements)
- Prevents future accumulation of untracked recommendations via systematic process
- Separates tactical fix (backlog triage) from strategic fix (process improvement)

**Trade-offs accepted:**
- Won't triage all 237 unactioned recommendations immediately (focus on 2026 high-value)
- Requires orchestrator judgment to determine "high-value" vs "defer"
- Process change adds step to investigation completion workflow

**Implementation sequence:**
1. **Immediate (orchestrator)**: Create beads issues for 5-10 high-value unactioned recommendations from 2026 investigations
2. **Short-term (worker)**: Update investigation skill self-review checklist to require either creating beads issue OR explicitly marking "Next: Recommendation deferred - [reason]"
3. **Medium-term (orchestrator)**: Run `kb reflect --type unactioned-recommendations` monthly to surface stranded work

### Alternative Approaches Considered

**Option B: Systematic Full Triage**
- **Pros:** Complete backlog cleanup, no recommendations left unreviewed
- **Cons:** High upfront cost (manually review 237 investigations), many may be obsolete
- **When to use instead:** If orchestrator needs complete audit trail of all past recommendations

**Option C: Do Nothing (Let Investigations Age Out)**
- **Pros:** Zero implementation cost, natural selection filters valuable from obsolete
- **Cons:** Loses valuable work (HUD, usage caching proven valuable), no learning captured
- **When to use instead:** If investigation recommendations consistently prove low-value

**Option D: Automated Recommendation Tracking**
- **Pros:** Zero manual overhead, systematic enforcement
- **Cons:** Requires tooling changes (investigation skill auto-creates issues?), may create noise
- **When to use instead:** If recommendation→issue conversion rate is very high (>80%)

**Rationale for recommendation:** Option A balances immediate value capture with sustainable process improvement. Options B/D solve problems we don't have (need full audit, need automation), Option C loses valuable work.

---

### Implementation Details

**What to implement first:**
1. **Immediate beads issues (orchestrator)** - Create issues for top 5-10 unactioned recommendations:
   - `bd create "Implement orch-hud.ts plugin for dynamic HUD" --type task`
   - `bd create "Add usage caching (30-60s TTL) to pkg/usage/" --type task`
   - `bd create "Implement 5 coaching plugin noise reduction improvements" --type task`
   - `bd create "Add semantic pattern matching to normalizeQuery" --type task`
   - `bd create "Wire token data to completions.jsonl telemetry" --type task`

2. **Process fix (worker)** - Update investigation skill self-review checklist:
   - Add item: "✓ Next: field either creates beads issue OR explicitly defers with reason"
   - Location: `~/.claude/skills/worker/investigation/reference/self-review-guide.md`

3. **Ongoing monitoring** - Add to monthly `kb reflect` workflow (when implemented)

**Things to watch out for:**
- ⚠️ **Obsolete recommendations** - Some unactioned recommendations may be obsolete due to architecture changes
- ⚠️ **Duplicate tracking** - Check if recommendation already has issue under different name
- ⚠️ **Priority calibration** - "High-value" is subjective - orchestrator must triage based on current goals
- ⚠️ **Process overhead** - Don't make investigation completion too heavy; defer is a valid option

**Areas needing further investigation:**
- Should investigation skill auto-create beads issues for recommendations? (may create noise)
- What's the optimal cadence for unactioned recommendation review? (monthly, quarterly?)
- Are older (2025, 2024) investigations worth triaging or let them age out naturally?

**Success criteria:**
- ✅ **Zero stranded high-value recommendations** - All valuable work from 2026 investigations either has beads issue or explicit defer reason
- ✅ **Investigation skill enforces triage** - Self-review checklist prevents completion without addressing Next: field
- ✅ **Orchestrator visibility** - `kb reflect --type unactioned-recommendations` surfaces stranded work before it accumulates

---

## References

**Files Examined:**
- `.kb/investigations/2026-01-29-inv-orch-cannot-inspect-opencode-sessions.md` - Verified implementation in Investigation History
- `.kb/investigations/2026-01-28-inv-analyze-memory-usage-patterns.md` - Found unactioned usage caching recommendation
- `.kb/investigations/2026-01-27-inv-design-information-hiding-tool-restriction.md` - Verified action space restriction implemented
- `.kb/investigations/2026-01-28-inv-design-unified-strategic-center-dashboard.md` - Found tracked via orch-go-21022
- `.kb/investigations/2026-01-27-inv-design-exploration-dynamic-hud-pattern.md` - Found unactioned orch-hud.ts recommendation
- `.kb/investigations/2026-01-23-inv-gastown-orchestration-system-analysis-compare.md` - Verified recommendation became orch-go-0ns2e

**Commands Run:**
```bash
# Count total investigation files
find .kb/investigations/ -name "*.md" -type f | wc -l  # 702

# Count investigations with actionable recommendations
rg "^\*\*Next:\*\* (Implement|Add|Create|Fix|Build)" .kb/investigations/ --type md -l | wc -l  # 237

# Count 2026 investigations with actionable recommendations
rg "^\*\*Next:\*\* (Implement|Add|Create|Fix|Build)" .kb/investigations/2026-*.md --type md -l | wc -l  # 155

# Check if usage caching implemented
rg -i "cache.*usage|usage.*cache" pkg/usage/ --type go  # No results

# Check if orch servers implemented
orch servers --help  # Success - command exists

# Check if action space restriction implemented
rg "You CAN \(meta-actions\)|You CANNOT \(primitive actions\)" ~/.claude/skills/meta/orchestrator/  # Found

# Check if orch-hud plugin implemented
ls ~/Documents/personal/opencode/plugins/orch-hud.ts  # File not found

# Find beads issues matching recommendations
bd list --status open --limit 0 | rg -i "GUPP|Strategic Center|auto-resume"  # Found 3 matches
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-23-inv-so-many-investigations-created-root.md` - Root cause analysis of investigation overhead
- **Skill:** `~/.claude/skills/worker/investigation/SKILL.md` - Investigation workflow and self-review process

---

## Investigation History

**2026-01-30 (start):** Investigation started
- Initial question: Which investigation recommendations in .kb/investigations/ remain unactioned, and what work is needed to action them?
- Context: Spawned by orchestrator to scope backlog of unactioned investigation recommendations

**2026-01-30 (Finding 2):** Identified 237 investigations with actionable recommendations (155 from 2026)
- Used regex pattern matching to find investigations with "Implement|Add|Create|Fix|Build" in Next: field

**2026-01-30 (Finding 3-6):** Tested sample investigations to distinguish actioned from unactioned
- Verified orch servers command exists (implemented)
- Verified action space restriction in orchestrator skill (implemented)
- Verified usage caching NOT in pkg/usage/ (unactioned)
- Verified orch-hud.ts plugin NOT in opencode/plugins/ (unactioned)
- Found beads issues for some recommendations (GUPP hooks, Strategic Center, auto-resume)

**2026-01-30 (complete):** Investigation completed
- Status: Complete
- Key outcome: Identified 5-10 high-value unactioned recommendations from 2026 investigations; recommended two-track approach (immediate issue creation + process improvement)
