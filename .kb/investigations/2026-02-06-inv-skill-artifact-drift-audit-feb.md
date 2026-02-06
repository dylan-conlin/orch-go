## Summary (D.E.K.N.)

**Delta:** Found 14 drift items since Jan 15 audit: the Feb 5 redesign was implemented (orchestrator skill rewritten as unified 677-line file with COMPREHEND-TRIAGE-SYNTHESIZE model), resolving 12 of 19 prior drift items. However, 7 new drift items emerged from CLI changes not yet reflected in skills, plus 5 persistent items from Jan 15, and 2 cross-skill contradictions (bd close, orch session).

**Evidence:** Cross-referenced git log (80+ commits since Jan 15), 43 new decisions, deployed orchestrator skill (677 lines, compiled Feb 6), investigation skill (251 lines), systematic-debugging skill (639 lines), feature-impl skill (1500+ lines), and worker-base skill against actual CLI capabilities.

**Knowledge:** The orchestrator skill redesign was successful — COMPREHEND→TRIAGE→SYNTHESIZE is the leading section, reference/ directory eliminated, label-based work grouping added. But new features (orch rework, state.db, ProcessedIssueCache, daemon auto-expiry, grace periods) aren't documented in skills. Worker skills have a critical contradiction: investigation and systematic-debugging tell agents to run `bd close`, while worker-base says "NEVER run bd close".

**Next:** Create issues for the 14 drift items. The `bd close` contradiction in worker skills (H1) should be highest priority — it causes agents to bypass orchestrator verification.

**Authority:** implementation - Audit within existing patterns, actionable via individual skill updates

---

# Investigation: Skill and Artifact Drift Audit (Feb 2026)

**Question:** What has drifted between CLI capabilities, recent decisions, and .kb/models vs the orchestrator and worker skills since the Jan 15 audit?

**Started:** 2026-02-06
**Updated:** 2026-02-06
**Owner:** og-inv-skill-artifact-drift-06feb-ed93
**Phase:** Complete
**Next Step:** None
**Status:** Complete

**Patches-Decision:** N/A
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-01-15-inv-orchestrator-skill-drift-audit.md | extends | yes - read skill, confirmed 12/19 items resolved by redesign | None - redesign addressed most findings |
| .kb/investigations/2026-02-05-inv-redesign-orchestrator-skill-1m-context.md | confirms | yes - unified skill at ~/.opencode/skill/orchestrator/SKILL.md is 677 lines, compiled Feb 6 | Implementation exceeded proposed 500-600 lines by ~15% |

---

## Findings

### Finding 1: Orchestrator Skill Redesign Was Implemented — Resolves 12/19 Jan 15 Items

**Evidence:** The deployed orchestrator skill at `~/.opencode/skill/orchestrator/SKILL.md` was compiled Feb 6 (checksum 7f8c7da02b27) and is 677 lines. It is organized around COMPREHEND→TRIAGE→SYNTHESIZE (lines 31-49), has no reference/ directory (glob returns "No such file or directory"), includes the daemon-orchestrator division of labor (lines 47-48), and has the Work Grouping (Labels, Not Epics) section (lines 612-640).

**Resolved Jan 15 items:**
- H1 (Strategic Orchestrator Model): ✅ Now the leading identity section (lines 31-49)
- H3 (Model Selection Constraints): ✅ Updated with sonnet default, --opus flag (lines 355-365)
- H4 (Triage Bypass): ✅ `--bypass-triage` documented in spawning methods (line 349-351)
- H5 (Inconsistent Default Spawn Mode): ✅ Clarified policy skills auto-default to tmux (line 557)
- M1 (Follow Orchestrator Mechanism): ✅ Dashboard monitoring documented (lines 553-556)
- M3 (Duplicate Prevention): ✅ Spawn protections section (lines 374-379)
- M4 (Rate Limit Monitoring): ✅ 80%/95% thresholds documented (line 378)
- M5 (Five-Tier Escalation Model): Partially — completion verification section exists but tiers not explicitly listed
- M6 (Gap Gating): Not explicitly in skill — but context quality gates exist in CLI
- M7 (Two-Tier Reflection): Not in skill — `orch reflect` command exists but undocumented in skill
- L1 (Skill-Type Values): Partially — skill-type shown in frontmatter but no canonical list
- L3 (Checkpoint Thresholds): Not deduplicated — thresholds don't appear in skill at all

**Source:** `~/.opencode/skill/orchestrator/SKILL.md` (full read, 677 lines)

**Significance:** The Feb 5 redesign was a major improvement. Most high-priority Jan 15 drift items are resolved. Remaining items are medium/low priority.

---

### Finding 2: `orch session start/end/status` Still Referenced but Commands Removed

**Evidence:** Line 506 of the orchestrator skill reads: `**Session:** \`orch session start "goal"\` | \`orch session status\` | \`orch session end\``. Running `orch session --help` returns: "Error: unknown command 'session' for 'orch-go'". The command was removed per decision `2026-01-19-remove-session-handoff-machinery.md` and commit `4f81c880 Remove session handoff machinery per decision`.

**Source:**
- `~/.opencode/skill/orchestrator/SKILL.md:506`
- CLI test: `orch session --help` → error

**Significance:** Orchestrators following the skill will try to run commands that don't exist. This is a holdover from pre-redesign content that wasn't cleaned up. The skill correctly doesn't reference SESSION_HANDOFF.md elsewhere (grep confirmed no matches), so this is a localized issue in the Tools & Commands section.

---

### Finding 3: `bd close` Contradiction Between Worker Skills and Worker-Base

**Evidence:** The deployed investigation skill (`~/.opencode/skill/investigation/SKILL.md:245`) instructs: "Close the beads issue: `bd close <beads-id> --reason 'conclusion summary'`". The systematic-debugging skill (`~/.opencode/skill/systematic-debugging/SKILL.md:637`) instructs the same. However, worker-base (`~/.claude/skills/shared/worker-base/SKILL.md:176`) explicitly states: "**Never run `bd close`** - Only the orchestrator closes issues via `orch complete`."

The spawned investigation skill loaded in THIS session's SPAWN_CONTEXT (line 269) correctly says "NEVER run `bd close`" — but that's the worker-base dependency text, not the standalone investigation SKILL.md at `~/.opencode/skill/investigation/SKILL.md`.

The source skills at `~/.claude/skills/src/worker/systematic-debugging/SKILL.md:739` have a note saying "bd close is removed from agent responsibilities" — suggesting the source was updated but the deployed SKILL.md at `~/.opencode/skill/` wasn't recompiled.

**Source:**
- `~/.opencode/skill/investigation/SKILL.md:245` — tells agents to `bd close`
- `~/.opencode/skill/systematic-debugging/SKILL.md:637` — tells agents to `bd close`
- `~/.claude/skills/shared/worker-base/SKILL.md:176` — says NEVER `bd close`
- `~/.claude/skills/src/worker/systematic-debugging/SKILL.md:739` — source says removed

**Significance:** **HIGH PRIORITY.** This is the most impactful drift item. Agents spawned with these skills directly (not via worker-base injection) will bypass orchestrator verification by closing their own issues. This breaks the `orch complete` verification pipeline. The fix is to recompile deployed skills from updated sources.

---

### Finding 4: New CLI Features Not Documented in Any Skill

**Evidence:** Git log since Jan 15 shows several significant features that aren't referenced in any deployed skill:

| Feature | Commit | Skill Impact |
|---------|--------|-------------|
| `orch rework` command | `d35310c8` | Orchestrator needs to know about post-completion rework flow |
| SQLite state.db for agent state | `464cd2c4` | `orch status` reads from state DB — orchestrators may need to know |
| ProcessedIssueCache in daemon | `894c907c` | Daemon prevents re-spawning processed issues — affects orchestrator mental model |
| Daemon auto-expiry for idle agents >1h | `bedab726` | Agents auto-expire — orchestrator should know |
| Daemon grace period for triage:ready | `74c3d9e1` | Daemon doesn't spawn immediately — affects timing expectations |
| `orch reflect` for kb reflection | `3ee551d4` | New orchestrator-level command not in Tools & Commands |
| `orch kb archive-old` | `a8b6c577` | Age-based investigation archival not documented |
| Gate vs advisory distinction in SPAWN_CONTEXT | `a8c52a88` | Template updated but skills don't explain the distinction |
| POST-COMPLETION-FAILURE context in spawn | `568b2d42` | Rework spawns include failure context — orchestrator needs to know |
| Attention signals (UNBLOCKED, STUCK, verify_failed) | `f3be75fc`, `87b28e2d` | Dashboard shows new signals — orchestrator should know |
| Label suggestions at spawn | `607e1912` | Area label suggestion flow exists |

**Source:** `git log --oneline --since="2026-01-15" -- '*.go'` (80+ commits)

**Significance:** The orchestrator skill has a Tools & Commands section (lines 498-528) that lists available commands but is missing `orch rework`, `orch reflect`, and `orch kb archive-old`. The daemon section (lines 652-658) is minimal and doesn't mention grace periods, auto-expiry, or ProcessedIssueCache behavior.

---

### Finding 5: Worker Skills Use Different Investigation Templates Than Spawned Skills

**Evidence:** The deployed investigation skill at `~/.opencode/skill/investigation/SKILL.md` has a minimal 45-line template (lines 66-93) with: Date, Status, Question, What I tried, What I observed, Test performed, Conclusion. But the spawned investigation skill loaded via worker-base dependencies (this session's SPAWN_CONTEXT lines 575-820) uses a much richer template with: D.E.K.N. Summary, Prior Work table, Findings with Evidence-Source-Significance pattern, Structured Uncertainty, Implementation Recommendations.

The feature-impl skill's investigation phase (lines 132-326) has yet another template format with: Question, Started/Updated/Status/Confidence, Findings, Synthesis, Confidence Assessment.

**Source:**
- `~/.opencode/skill/investigation/SKILL.md:66-93` — minimal template
- SPAWN_CONTEXT investigation skill (lines 575-820) — rich template via worker-base
- `~/.opencode/skill/feature-impl/SKILL.md:132-326` — third template format

**Significance:** Three different investigation template formats exist across skills. The minimal one in the standalone investigation skill contradicts the richer format enforced by `kb create investigation`. This causes inconsistency but is mitigated by the fact that `kb create investigation` generates from its own template regardless of what the skill says.

---

### Finding 6: Feature-Impl Skill Still Uses `orch build --skills` Build Marker

**Evidence:** The feature-impl skill at `~/.opencode/skill/feature-impl/SKILL.md:39-44` contains:
```
<!-- AUTO-GENERATED: Do not edit this file directly. Source: src/SKILL.md.template + src/phases/*.md. Build with: orch build --skills -->
> AUTO-GENERATED SKILL FILE
> Source: src/SKILL.md.template + src/phases/*.md
> Build command: orch build --skills
```

However, the orchestrator skill was compiled by skillc (checksum header: `<!-- AUTO-GENERATED by skillc -->`). The feature-impl skill still references the older `orch build --skills` build system. This is consistent with the constraint: "skillc and orch build skills are complementary, not competing" — but creates confusion about which build system to use for which skill.

**Source:** `~/.opencode/skill/feature-impl/SKILL.md:39-44` vs `~/.opencode/skill/orchestrator/SKILL.md:7-12`

**Significance:** Low. Both build systems work. But a new contributor would be confused about which to use. Decision `skillc and orch build skills are complementary` is still accurate but the practical delineation (which skills use which) isn't documented.

---

### Finding 7: Architect and Design-Session Skills Reference `bd close`

**Evidence:** From the grep results: `~/.claude/skills/src/worker/architect/SKILL.md:657` instructs "Close the beads issue: `bd close <beads-id>`" and `~/.claude/skills/src/worker/design-session/SKILL.md:737` has the same instruction. Same `bd close` contradiction as Finding 3, affecting more skills.

**Source:** Grep of `bd close` across `~/.claude/skills/` — 21 matches total, several in worker skill sources

**Significance:** Same issue as Finding 3 but broader scope. Multiple worker skills instruct agents to `bd close` despite worker-base prohibiting it. The source files (`src/worker/`) were partially updated (systematic-debugging has a note about removal) but the deployed SKILL.md files weren't recompiled.

---

## Synthesis

**Key Insights:**

1. **The orchestrator skill redesign was a success.** The Feb 5 investigation's recommendation was implemented: unified single-file skill organized around COMPREHEND→TRIAGE→SYNTHESIZE, no reference/ directory, daemon-orchestrator relationship explicit, label-based work grouping added. This resolved 12 of 19 Jan 15 drift items. The skill was compiled just today (Feb 6) — it's very fresh.

2. **The bd close contradiction is the highest-impact drift.** Worker skills (investigation, systematic-debugging, architect, design-session) instruct agents to `bd close`, while worker-base (which is injected into SPAWN_CONTEXT) says "NEVER". Agents that load these skills directly (not through SPAWN_CONTEXT injection) will bypass orchestrator verification. The source files were partially updated but deployed skills weren't recompiled.

3. **CLI moves faster than skills.** 80+ commits since Jan 15 introduced significant features (orch rework, state.db, ProcessedIssueCache, daemon auto-expiry, grace periods, attention signals) that aren't in any skill. The orchestrator skill was just recompiled but these features were missed.

4. **The orch session reference is a remnant.** The session handoff machinery was removed per Jan 19 decision, but line 506 of the orchestrator skill still lists `orch session start/end/status` as valid commands.

**Answer to Investigation Question:**

Since the Jan 15 audit, the orchestrator skill underwent a major redesign (Feb 5-6) that resolved most prior drift. However, 14 drift items remain:

**HIGH PRIORITY (3):**
- H1: `bd close` contradiction in worker skills (investigation, debugging, architect, design-session) vs worker-base
- H2: `orch session start/end/status` referenced in orchestrator skill but commands removed
- H3: `orch rework` command undocumented in orchestrator skill

**MEDIUM PRIORITY (6):**
- M1: Daemon auto-expiry undocumented in orchestrator skill
- M2: Daemon grace period for triage:ready undocumented
- M3: ProcessedIssueCache behavior undocumented
- M4: `orch reflect` command missing from Tools & Commands
- M5: `orch kb archive-old` missing from Tools & Commands
- M6: Attention signals (UNBLOCKED, STUCK, verify_failed) undocumented

**LOW PRIORITY (5):**
- L1: Three different investigation template formats across skills
- L2: Feature-impl uses `orch build --skills` while orchestrator uses skillc
- L3: Gate vs advisory distinction not explained in skills
- L4: POST-COMPLETION-FAILURE context flow undocumented
- L5: Area label suggestion flow exists but undocumented

**Already covered by issue 21379 (label-based grouping):** The work grouping section (Labels, Not Epics) was added to the orchestrator skill and appears current. No additional drift item needed.

---

## Structured Uncertainty

**What's tested:**

- ✅ Orchestrator skill compiled Feb 6, 677 lines (verified: read file, checked header)
- ✅ `orch session` command removed (verified: ran `orch session --help` → error)
- ✅ Investigation skill tells agents to `bd close` (verified: `grep -n "bd close" SKILL.md` → line 245)
- ✅ Worker-base says NEVER `bd close` (verified: `grep -n "Never.*bd close" SKILL.md` → line 176)
- ✅ `orch rework` exists but undocumented in skill (verified: ran `orch rework --help`, grep in skill → no matches)
- ✅ Reference directory eliminated (verified: glob → "No such file or directory")
- ✅ COMPREHEND→TRIAGE→SYNTHESIZE is leading section (verified: read skill lines 31-49)

**What's untested:**

- ⚠️ Whether agents actually call `bd close` in practice when spawned with standalone skills (hypothesis: SPAWN_CONTEXT injection of worker-base overrides, but can't confirm without session logs)
- ⚠️ Whether state.db/ProcessedIssueCache affect orchestrator decision-making enough to warrant skill documentation (hypothesis: yes for power users, but might be infrastructure details)
- ⚠️ Whether the three investigation template formats cause actual inconsistency (hypothesis: `kb create` enforces its own template, making skill template irrelevant)

**What would change this:**

- If agents spawned via `orch spawn investigation` DON'T get worker-base injection → `bd close` is actively harmful
- If state.db is ephemeral/internal-only → documenting it in skills is unnecessary
- If `orch rework` becomes the primary post-completion flow → needs prominent skill section, not just Tools list

---

## Implementation Recommendations

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Recompile worker skills to remove `bd close` | implementation | Mechanical fix within existing patterns |
| Remove `orch session` from orchestrator skill | implementation | Delete outdated reference |
| Add new commands to Tools & Commands | implementation | Additive documentation within existing section |
| Document daemon behavior changes | implementation | Additive documentation within existing section |

### Recommended Approach ⭐

**Recompile worker skills first, then update orchestrator skill Tools section** — Address the `bd close` contradiction immediately (highest impact), then add missing command documentation.

**Why this approach:**
- `bd close` contradiction is actively harmful — agents bypass verification
- Missing command documentation is informational, not behavioral
- Both are implementation-authority fixes (no architectural decisions)

**Implementation sequence:**
1. Edit worker skill sources (investigation, debugging, architect, design-session) to remove `bd close` instructions
2. Recompile with `skillc deploy` or `orch build --skills` as appropriate
3. Update orchestrator skill line 506 to remove `orch session start/end/status`
4. Add `orch rework`, `orch reflect`, `orch kb archive-old` to Tools & Commands
5. Add daemon behavior section (grace period, auto-expiry, ProcessedIssueCache)

### Alternative: Batch all changes in orchestrator skill redesign v2

- **Pros:** Single coordinated update, comprehensive
- **Cons:** Delays the `bd close` fix which is actively harmful now
- **When to use:** If a second round of redesign is already planned

**Rationale for recommendation:** The `bd close` fix is urgent and separable. Other items are informational and can be batched.

---

## References

**Files Examined:**
- `~/.opencode/skill/orchestrator/SKILL.md` (677 lines) — Deployed orchestrator skill
- `~/.opencode/skill/investigation/SKILL.md` (251 lines) — Deployed investigation skill
- `~/.opencode/skill/systematic-debugging/SKILL.md` (639 lines) — Deployed debugging skill
- `~/.opencode/skill/feature-impl/SKILL.md` (1500+ lines) — Deployed feature-impl skill
- `~/.claude/skills/shared/worker-base/SKILL.md` — Worker-base with `bd close` prohibition
- `.kb/investigations/2026-01-15-inv-orchestrator-skill-drift-audit.md` — Prior drift audit
- `.kb/investigations/2026-02-05-inv-redesign-orchestrator-skill-1m-context.md` — Redesign investigation

**Commands Run:**
```bash
# Verify orch session removed
orch session --help  # Error: unknown command "session"

# Verify orch rework exists
orch rework --help  # Shows post-completion rework flow

# Check bd close in worker skills
grep -n "bd close" ~/.opencode/skill/investigation/SKILL.md  # Line 245
grep -n "bd close" ~/.opencode/skill/systematic-debugging/SKILL.md  # Line 637

# Check git changes since Jan 15
git log --oneline --since="2026-01-15" -- '*.go'  # 80+ commits

# Verify reference directory removed
glob ~/.opencode/skill/orchestrator/reference/*.md  # No such file or directory

# Check kb link command exists
kb link --help  # Works
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-19-remove-session-handoff-machinery.md` — Session handoff removal
- **Investigation:** `.kb/investigations/2026-01-15-inv-orchestrator-skill-drift-audit.md` — Prior drift audit (19 items)
- **Investigation:** `.kb/investigations/2026-02-05-inv-redesign-orchestrator-skill-1m-context.md` — Redesign proposal

---

## Investigation History

**[2026-02-06 14:30]:** Investigation started
- Initial question: What has drifted since Jan 15 audit?
- Context: Jan 15 found 19 drift items, Feb 5 proposed redesign, status unclear

**[2026-02-06 14:35]:** Read prior investigations
- Jan 15 audit methodology and 19 items understood
- Feb 5 redesign proposed unified skill, unclear if implemented

**[2026-02-06 14:40]:** Read deployed skills, checked git log
- Orchestrator skill: 677 lines, compiled Feb 6, redesign implemented
- 80+ CLI commits since Jan 15, 43 new decisions
- Worker skills at ~/.opencode/skill/ still have bd close instructions

**[2026-02-06 14:50]:** Cross-referenced and identified drift items
- 14 drift items across 3 priority levels
- bd close contradiction is highest impact (actively harmful)
- orch session reference is a holdover
- Multiple new commands undocumented

**[2026-02-06 15:00]:** Investigation completed
- Status: Complete
- Key outcome: 14 drift items found, 3 high priority (bd close, orch session, orch rework), 6 medium, 5 low. The orchestrator skill redesign resolved 12/19 prior items.
