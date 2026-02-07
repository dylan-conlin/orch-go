<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Session handoffs are created in two locations with duplicate work - session start creates comprehensive template in global ~/.orch (never used), session end creates placeholder in project-specific location (used for resume but empty).

**Evidence:** Read session.go:86-902 and orchestrator_context.go:358-549; analyzed git commits 3b8d7392 (added interactive prompts) and 62690d59 (reverted due to stdin blocking); verified PreFilledSessionHandoffTemplate exists but goes to wrong location; confirmed discoverSessionHandoff only looks in project-specific path.

**Knowledge:** This is a routing problem, not a content problem - the comprehensive PreFilledSessionHandoffTemplate already exists, it's just going to the wrong place; stdin blocking is a hard constraint (interactive prompts already tried and failed); progressive documentation requires template available during session (not just at end).

**Next:** Implement Active Directory Pattern - modify session start to create {project}/.orch/session/{window}/active/SESSION_HANDOFF.md with PreFilledSessionHandoffTemplate, modify session end to archive active/ to timestamped directory, update session resume to check active/ as fallback, remove unused global ~/.orch workspace creation.

**Promote to Decision:** Superseded - session handoff machinery removed (2026-01-19-remove-session-handoff-machinery.md)

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

# Investigation: Design Session Handoff Architecture

**Question:** How should session handoffs be created to serve both orchestrator agents (no stdin) AND produce useful resume content?

**Started:** 2026-01-13
**Updated:** 2026-01-13
**Owner:** architect agent (orch-go-98u5q)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Dual-location architecture creates unused workspace + empty handoffs

**Evidence:**
- `orch session start` creates `~/.orch/session/{date}/SESSION_HANDOFF.md` using `GeneratePreFilledSessionHandoff()` (cmd/orch/session.go:106-145)
- `orch session end` creates `{project}/.orch/session/{window}/{timestamp}/SESSION_HANDOFF.md` using placeholders (cmd/orch/session.go:514-517)
- Session resume (`orch session resume`) discovers handoffs via `discoverSessionHandoff()` which looks for `{project}/.orch/session/{window}/latest/` (cmd/orch/session.go:675-769)
- The session start workspace at ~/.orch is NEVER used for resume (wrong location)
- The session end workspace has the RIGHT location but EMPTY/placeholder content

**Source:**
- cmd/orch/session.go:106-145 (createSessionWorkspace)
- cmd/orch/session.go:514-517 (createSessionHandoffDirectory call)
- cmd/orch/session.go:562-588 (promptSessionReflection - returns placeholders)
- cmd/orch/session.go:771-902 (createSessionHandoffDirectory implementation)
- pkg/spawn/orchestrator_context.go:531-549 (GeneratePreFilledSessionHandoff)

**Significance:**
Session start creates work that's never used (global ~/.orch location). Session end creates handoff in correct location but with useless placeholder content. This is duplicate work producing different results.

---

### Finding 2: Interactive prompts were tried and reverted due to stdin blocking

**Evidence:**
- Commit 3b8d7392 added interactive reflection prompts to gather handoff content at session end
- Commit 62690d59 reverted this because "Orchestrators running in Claude Code terminals are the primary users, not humans at CLI. Interactive prompts block orchestrators who can't provide stdin."
- The revert message explicitly states: "Both attempted to optimize for wrong use case"

**Source:**
- `git show 3b8d7392 --stat` - added SessionReflection struct, promptSessionReflection() with stdin gathering
- `git show 62690d59 --stat` - reverted to placeholders without stdin
- cmd/orch/session.go:562-588 - current promptSessionReflection returns minimal placeholders

**Significance:**
The problem of "how to populate handoffs without stdin" was already encountered and the attempted solution (interactive prompts) failed. The constraint is that orchestrator agents cannot provide interactive input.

---

### Finding 3: The comprehensive template exists but goes to the wrong location

**Evidence:**
- `PreFilledSessionHandoffTemplate` at pkg/spawn/orchestrator_context.go:360-522 contains a FULL structured template with:
  - Progressive documentation instructions
  - TLDR, Spawns, Evidence, Knowledge, Friction, Focus Progress, Next, Unexplored Questions sections
  - Detailed guidance on when to fill each section
- This template is used by `GeneratePreFilledSessionHandoff()` which is called by session start (cmd/orch/session.go:178)
- Session start creates workspace at `~/.orch/session/{date}/` - GLOBAL location not project-specific
- Session resume looks for `{project}/.orch/session/{window}/latest/` - PROJECT-SPECIFIC location
- The two never connect

**Source:**
- pkg/spawn/orchestrator_context.go:358-522 (PreFilledSessionHandoffTemplate definition)
- pkg/spawn/orchestrator_context.go:531-549 (GeneratePreFilledSessionHandoff function)
- cmd/orch/session.go:150-190 (createSessionWorkspace - uses GeneratePreFilledSessionHandoff)
- cmd/orch/session.go:675-769 (discoverSessionHandoff - looks in project/.orch/session/{window}/)

**Significance:**
The solution already exists (comprehensive template), but it's going to the wrong place. Session start creates global workspace that's never resumed. Session end creates project-specific workspace with placeholders.

---

## Synthesis

**Key Insights:**

1. **The architecture has the pieces but they're misconnected** - The comprehensive PreFilledSessionHandoffTemplate exists and is used at session start, but goes to a global ~/.orch location that's never consulted for resume. Session end creates handoffs in the correct project-specific location but fills them with placeholders. This is a routing problem, not a content problem.

2. **Stdin blocking is a hard constraint** - Interactive prompts cannot work because orchestrator agents run in terminals without stdin access. Any solution must work non-interactively. The revert of commit 3b8d7392 proves this was already tried and failed.

3. **Session start workspace serves no purpose** - The global ~/.orch/session/{date}/ workspace created at session start is never used for anything. It doesn't participate in resume (which looks in project/.orch/session/{window}/), and its handoff is never read. This is wasted work.

**Answer to Investigation Question:**

The solution is to **use the PreFilledSessionHandoffTemplate at BOTH session start AND session end, routing to the project-specific location** instead of creating two different handoffs in two locations. Session start should create `{project}/.orch/session/{window}/active/SESSION_HANDOFF.md` with the full template (for progressive documentation during session). Session end should move/copy this to `{project}/.orch/session/{window}/{timestamp}/` and update the `latest` symlink. This eliminates the duplicate work, puts content in the right location, and works without stdin.

---

## Structured Uncertainty

**What's tested:**

- ✅ Session start creates ~/.orch/session/{date}/ workspace (verified: read cmd/orch/session.go:150-190)
- ✅ Session end creates {project}/.orch/session/{window}/{timestamp}/ workspace (verified: read cmd/orch/session.go:771-902)
- ✅ Resume discovers handoffs at {project}/.orch/session/{window}/latest/ (verified: read cmd/orch/session.go:675-769)
- ✅ PreFilledSessionHandoffTemplate has comprehensive structure (verified: read pkg/spawn/orchestrator_context.go:358-522)
- ✅ Interactive prompts were reverted due to stdin blocking (verified: git show 62690d59)

**What's untested:**

- ⚠️ Progressive documentation actually happens during sessions (assumption - not validated with real orchestrator sessions)
- ⚠️ The "active" directory name won't conflict with timestamp directories (naming convention assumption)
- ⚠️ Orchestrators will fill the template sections during work (behavioral assumption)

**What would change this:**

- Finding would be wrong if session start workspace IS used somewhere and I missed it
- Finding would be wrong if there's a valid reason for the dual-location architecture I haven't discovered
- Recommendation would be wrong if orchestrators need DIFFERENT handoff structure than spawned orchestrators

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Active Directory Pattern** - Session start creates `{project}/.orch/session/{window}/active/SESSION_HANDOFF.md` with PreFilledSessionHandoffTemplate; session end archives active/ to timestamped directory and updates latest symlink.

**Why this approach:**
- Reuses the comprehensive PreFilledSessionHandoffTemplate that already exists (Finding 3)
- Puts handoff in project-specific location where resume discovers it (Finding 1)
- Enables progressive documentation (orchestrators can fill sections during session)
- Works without stdin - template pre-created, orchestrators fill at their discretion (Finding 2)
- Eliminates duplicate work - single handoff file, two lifecycle events (start creates, end archives)

**Trade-offs accepted:**
- Orchestrators must remember to fill SESSION_HANDOFF.md during work (not automatic)
- Active directory pattern adds slight complexity to session discovery logic
- If session end never runs (crash/manual kill), active/ directory orphaned

**Implementation sequence:**
1. **Modify session start** (cmd/orch/session.go:runSessionStart) - Instead of createSessionWorkspace (global ~/.orch), create `{project}/.orch/session/{window}/active/` and write PreFilledSessionHandoffTemplate there
2. **Modify session end** (cmd/orch/session.go:runSessionEnd) - Instead of promptSessionReflection + createSessionHandoffDirectory, move/copy active/ to timestamped directory, update latest symlink, clean up active/
3. **Update session resume** (cmd/orch/session.go:discoverSessionHandoff) - Add fallback: if latest/ doesn't exist but active/ does, use active/ (enables mid-session resume)
4. **Remove createSessionWorkspace** - Delete global ~/.orch session workspace creation (lines 150-190), no longer needed

### Alternative Approaches Considered

**Option B: Auto-populate from session state at session end**
- **Pros:**
  - No reliance on orchestrators filling template during session
  - Guaranteed content at session end (derived from session.json state)
  - No orphaned active/ directories
- **Cons:**
  - Requires complex logic to derive handoff content from session state
  - Loses progressive documentation benefit (only creates at end)
  - Auto-generated content may be less useful than human-curated
  - Still requires writing new template population logic (Finding 2 showed placeholders are insufficient)
- **When to use instead:** If orchestrators never fill handoffs during sessions (behavioral evidence needed)

**Option C: Remove session start workspace entirely**
- **Pros:**
  - Simplest implementation (minimal changes)
  - Eliminates unused global ~/.orch workspace (Finding 1)
  - Only creates handoff at session end (single responsibility)
- **Cons:**
  - No progressive documentation (can't fill during session)
  - Still leaves the problem: what content to put in session end handoff? (Finding 2)
  - Doesn't solve the core issue (empty handoffs), just reduces duplication
- **When to use instead:** If progressive documentation proves to have no value

**Option D: Status quo with better placeholders**
- **Pros:**
  - Minimal code changes
  - Preserves existing dual-location architecture
- **Cons:**
  - Doesn't address root cause (template in wrong location, Finding 1)
  - Still creates unused global workspace
  - "Better placeholders" still face stdin blocking constraint (Finding 2)
- **When to use instead:** If the dual-location architecture serves a purpose I haven't discovered

**Rationale for recommendation:**

Option A (Active Directory Pattern) is the only approach that:
1. Reuses existing comprehensive template (Finding 3)
2. Puts content in discoverable location (Finding 1)
3. Works without stdin (Finding 2)
4. Enables progressive documentation (orchestrators can fill as they work)
5. Eliminates duplicate workspace creation

Options B and C defer the content problem. Option D doesn't solve it at all. Option A solves the routing problem AND the content problem by putting the right template in the right place.

---

### Implementation Details

**What to implement first:**
1. **Session start modification** - Create project-specific active/ directory instead of global ~/.orch workspace (highest impact, unblocks progressive docs)
2. **Session end archival** - Move active/ to timestamped directory, update latest symlink (completes the lifecycle)
3. **Session resume fallback** - Check active/ if latest/ doesn't exist (enables mid-session resume)
4. **Cleanup** - Remove createSessionWorkspace function and ~/.orch session creation (eliminates confusion)

**Things to watch out for:**
- ⚠️ **Window name resolution** - Both session start and end need to call `tmux.GetCurrentWindowName()` to ensure they use the same window scope
- ⚠️ **Concurrent session handling** - What if session start is called twice without session end? Should overwrite active/ or create active-2/?
- ⚠️ **Project directory discovery** - Session start runs from arbitrary cwd, needs to find project root (walk up to .git or accept current dir)
- ⚠️ **File permissions** - Ensure active/ directory is writable by orchestrator agents (may run as different user contexts)
- ⚠️ **Symlink handling** - latest symlink should be relative path (not absolute) to avoid issues with different mount points

**Areas needing further investigation:**
- Do orchestrators actually fill handoffs during sessions? (behavioral validation needed - may require instrumentation)
- Should active/ be tracked in .gitignore? (transient state, probably yes)
- What happens if orchestrator crashes mid-session? (active/ orphaned, needs cleanup mechanism)
- Should orch status show if SESSION_HANDOFF.md exists in active/? (visibility into progressive docs)

**Success criteria:**
- ✅ Session start creates `{project}/.orch/session/{window}/active/SESSION_HANDOFF.md` with full PreFilledSessionHandoffTemplate
- ✅ Session end moves active/ to `{project}/.orch/session/{window}/{timestamp}/` and updates latest symlink
- ✅ orch session resume discovers handoffs from both latest/ and active/ (fallback)
- ✅ No more ~/.orch/session/{date}/ directories created
- ✅ Handoffs have comprehensive content (not placeholders)

---

## References

**Files Examined:**
- cmd/orch/session.go:86-145 - runSessionStart and createSessionWorkspace (session start creates global workspace)
- cmd/orch/session.go:150-190 - createSessionWorkspace implementation (unused global ~/.orch location)
- cmd/orch/session.go:478-560 - runSessionEnd (creates project-specific handoff with placeholders)
- cmd/orch/session.go:562-588 - promptSessionReflection (returns minimal placeholders, no stdin)
- cmd/orch/session.go:675-769 - discoverSessionHandoff (looks for project/.orch/session/{window}/latest)
- cmd/orch/session.go:771-902 - createSessionHandoffDirectory (creates project-specific handoff)
- pkg/spawn/orchestrator_context.go:358-522 - PreFilledSessionHandoffTemplate (comprehensive template)
- pkg/spawn/orchestrator_context.go:531-549 - GeneratePreFilledSessionHandoff (generates from template)

**Commands Run:**
```bash
# View interactive prompts commit
git show 3b8d7392 --stat

# View revert commit
git show 62690d59 --stat

# Find session handoff references
grep -r "session.*handoff\|SESSION_HANDOFF" --include="*.go"
```

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-13-inv-session-end-creates-empty-handoff.md` - Investigation into why session end creates empty handoffs (referenced in commit 3b8d7392)

---

## Investigation History

**[2026-01-13 16:00]:** Investigation started
- Initial question: How should session handoffs be created to serve both orchestrator agents (no stdin) AND produce useful resume content?
- Context: Spawned as architect agent (orch-go-98u5q) to diagnose symptom: session start creates working handoff at ~/.orch/session/{date}/ (never used), session end creates archived handoff at {project}/.orch/session/{window}/{timestamp}/ (empty placeholders)

**[2026-01-13 16:15]:** Identified dual-location architecture
- Session start creates global ~/.orch workspace with comprehensive template
- Session end creates project-specific workspace with placeholders
- The two never connect - routing problem, not content problem

**[2026-01-13 16:30]:** Analyzed interactive prompts revert
- Commit 3b8d7392 added stdin prompts, commit 62690d59 reverted
- Stdin blocking is hard constraint for orchestrator agents
- Interactive approach already tried and failed

**[2026-01-13 16:45]:** Synthesis and recommendation
- Status: Recommendations complete
- Key outcome: Recommend Active Directory Pattern - use PreFilledSessionHandoffTemplate at session start in project-specific location, archive to timestamped directory at session end
