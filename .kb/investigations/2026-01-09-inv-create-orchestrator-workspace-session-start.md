<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Workspace creation for orchestrator sessions was implemented Jan 6 at `~/.orch/session/{date}/` but differs from issue requirements in location, naming, and contents (missing SPAWN_CONTEXT.md and SYNTHESIS.md).

**Evidence:** Code inspection of cmd/orch/session.go:96-181 shows createSessionWorkspace() exists; git history shows commit 205f7ba from Jan 6; current session has workspace at ~/.orch/session/2026-01-09/ with only SESSION_HANDOFF.md.

**Knowledge:** Issue description is partially outdated; multiple conflicting patterns exist (session/ vs workspace/, date-based vs timestamp-based, og-orch-* naming); progressive vs end-population approaches conflict; workspace path not tracked in session.json.

**Next:** Escalate 5 design questions to orchestrator (workspace location, SESSION_HANDOFF.md population timing, SPAWN_CONTEXT.md necessity, SYNTHESIS.md purpose, issue status); cannot proceed with implementation until decisions made.

**Promote to Decision:** recommend-no - These are one-time clarifications, not architectural patterns

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

# Investigation: Create Orchestrator Workspace Session Start

**Question:** What workspace creation functionality already exists for orchestrator sessions, and what's missing vs the issue requirements?

**Started:** 2026-01-09
**Updated:** 2026-01-09
**Owner:** orch-go-xn6ok
**Phase:** Investigating
**Next Step:** Clarify requirements with orchestrator (issue description may be outdated)
**Status:** In Progress

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Workspace creation already exists (added Jan 6)

**Evidence:** 
- `createSessionWorkspace()` function exists in `cmd/orch/session.go:141-181`
- Creates directory at `~/.orch/session/{date}/` (not `.orch/workspace/orch-session-{timestamp}/` as issue requests)
- Generates SESSION_HANDOFF.md using `spawn.GeneratePreFilledSessionHandoff()`
- Workspace path logged to events but not saved to session.json
- Current session has workspace at `~/.orch/session/2026-01-09/` with SESSION_HANDOFF.md

**Source:** 
- Commit 205f7ba (2026-01-06 12:56): "fix: create workspace with SESSION_HANDOFF.md on orch session start"
- `cmd/orch/session.go:96-129` - `runSessionStart()` calls `createSessionWorkspace()`
- `ls ~/.orch/session/2026-01-09/` shows SESSION_HANDOFF.md created at 14:41 today

**Significance:** The issue description states "No workspace directory created" but this is outdated - workspace creation was implemented 3 days before the issue was created. However, the location and contents differ from issue requirements.

---

### Finding 2: Location discrepancy (session/ vs workspace/)

**Evidence:**
- Current: `~/.orch/session/{date}/SESSION_HANDOFF.md`
- Issue requests: `.orch/workspace/orch-session-YYYY-MM-DD-HHMM/`
- Prior decision says "orchestrator workspaces use og-orch-* naming" (.kb/decisions/2026-01-04-orchestrator-session-lifecycle.md)
- Current implementation uses date-based directory (one per day) vs timestamp-based (one per session)

**Source:**
- `cmd/orch/session.go:148-149` - uses `time.Now().Format("2006-01-02")` for directory
- Issue orch-go-xn6ok description requests different path pattern
- SPAWN_CONTEXT.md prior knowledge line 84: "orchestrator workspaces use og-orch-* naming instead of og-work-*"

**Significance:** There's tension between existing implementation (~/.orch/session/{date}/), issue requirements (.orch/workspace/orch-session-{timestamp}/), and prior decision (og-orch-* naming). Need clarification on correct pattern.

---

### Finding 3: Missing SPAWN_CONTEXT.md and SYNTHESIS.md

**Evidence:**
- Current workspace only creates SESSION_HANDOFF.md
- Issue requests "Initialize with SPAWN_CONTEXT.md equivalent (session goal, focus, starting state)"
- Issue requests "Create empty SYNTHESIS.md for session artifacts"
- `ls ~/.orch/session/2026-01-09/` shows only SESSION_HANDOFF.md, no SPAWN_CONTEXT.md or SYNTHESIS.md

**Source:**
- `cmd/orch/session.go:169-178` - only writes SESSION_HANDOFF.md
- Issue orch-go-xn6ok requirements 2 & 3

**Significance:** While workspace creation exists, it's incomplete vs issue requirements. SPAWN_CONTEXT.md would provide session context, SYNTHESIS.md would accumulate artifacts during session.

---

### Finding 4: SESSION_HANDOFF.md created at start, not updated at end

**Evidence:**
- SESSION_HANDOFF.md is pre-filled and created during `orch session start`
- Issue requests "On `orch session end`, populate SESSION_HANDOFF.md in workspace"
- Current `runSessionEnd()` in cmd/orch/session.go:459-524 does not update SESSION_HANDOFF.md
- Template comment says "Fill this file AS YOU WORK, not at the end"

**Source:**
- `cmd/orch/session.go:169-178` - writes SESSION_HANDOFF.md at session start
- `cmd/orch/session.go:459-524` - `runSessionEnd()` does not interact with workspace
- `.orch/templates/SESSION_HANDOFF.md:12-39` - progressive documentation guidance

**Significance:** The template expects progressive filling during work, not population at end. Issue requirement #4 may conflict with current progressive documentation approach.

---

### Finding 5: Workspace path not tracked in session.json

**Evidence:**
- session.json structure shows: goal, started_at, spawns[] - no workspace_path field
- `createSessionWorkspace()` logs workspace_path to events but doesn't update session.json
- pkg/session package would need to add workspace_path field to Session struct

**Source:**
- `~/.orch/session.json` - current structure
- `cmd/orch/session.go:110-112` - adds workspace_path to event but not session
- Need to check `pkg/session/session.go` for Session struct

**Significance:** Without tracking workspace path in session.json, `orch session status` cannot display workspace location, and `orch session end` cannot easily find/update the workspace.

---

## Synthesis

**Key Insights:**

1. **Issue description is partially outdated** - Workspace creation was implemented Jan 6 (3 days before issue created), but at different location (~/.orch/session/ vs .orch/workspace/) and with different contents (only SESSION_HANDOFF.md, no SPAWN_CONTEXT.md or SYNTHESIS.md).

2. **Multiple conflicting patterns exist** - Current implementation (~/.orch/session/{date}/), issue requirements (.orch/workspace/orch-session-{timestamp}/), and prior decision (og-orch-* naming) all suggest different workspace locations and naming conventions.

3. **Progressive vs end-population tension** - Template expects progressive filling ("AS YOU WORK"), but issue requirement #4 requests population "on orch session end". These approaches are fundamentally different - need to clarify intent.

4. **Workspace not fully integrated** - Path logged to events but not saved to session.json, making it invisible to `orch session status` and inaccessible to `orch session end`.

**Answer to Investigation Question:**

Workspace creation for orchestrator sessions was partially implemented on Jan 6, 2026 (commit 205f7ba). Current implementation creates `~/.orch/session/{date}/SESSION_HANDOFF.md` on session start, but differs from issue requirements in several ways:

**What exists:**
- ✅ Workspace directory creation
- ✅ SESSION_HANDOFF.md with pre-filled metadata
- ✅ Uses same template as spawned orchestrators

**What's missing vs issue:**
- ❌ Location: ~/. orch/session/{date}/ not .orch/workspace/orch-session-{timestamp}/
- ❌ No SPAWN_CONTEXT.md initialization
- ❌ No SYNTHESIS.md creation
- ❌ Workspace path not tracked in session.json
- ❌ SESSION_HANDOFF.md not updated on session end

**Ambiguity requiring clarification:**
- Where should orchestrator workspaces live? (session/ vs workspace/, date-based vs timestamp-based)
- Should SESSION_HANDOFF.md be progressively filled (current template approach) or populated at end (issue requirement)?
- Does "og-orch-*" naming from prior decision apply to interactive sessions or only spawned orchestrators?

---

## Structured Uncertainty

**What's tested:**

- ✅ Workspace creation exists (verified: read cmd/orch/session.go:96-181, checked git history)
- ✅ Current workspace location (verified: ls ~/.orch/session/2026-01-09/, shows SESSION_HANDOFF.md)
- ✅ Missing files (verified: only SESSION_HANDOFF.md present, no SPAWN_CONTEXT.md or SYNTHESIS.md)
- ✅ Workspace path not in session.json (verified: cat ~/.orch/session.json shows no workspace_path field)

**What's untested:**

- ⚠️ Whether issue author was aware of Jan 6 implementation (assumption: may not have been)
- ⚠️ Whether location discrepancy is intentional or oversight (need to ask)
- ⚠️ Whether SPAWN_CONTEXT.md is truly needed for interactive sessions (not tested if orchestrators use it)
- ⚠️ Whether progressive vs end-population is design choice or miscommunication

**What would change this:**

- Finding would be wrong if there's a newer commit after Jan 6 that moved workspace to .orch/workspace/
- Assumptions about issue author would change if they created orch-go-38zik (the closed issue from Jan 6)
- Progressive vs end-population tension would resolve if template comments are wrong/outdated

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

**⚠️ ESCALATION REQUIRED:** Multiple design decisions need orchestrator input before implementation can proceed.

### Questions for Orchestrator

1. **Workspace location:** Should orchestrator workspaces live in:
   - a) `.orch/workspace/orch-session-{timestamp}/` (as issue requests)
   - b) `~/.orch/session/{date}/` (current implementation)
   - c) `.orch/workspace/og-orch-{timestamp}/` (aligns with prior decision on naming)

2. **SESSION_HANDOFF.md population:** Should it be:
   - a) Pre-created at start with progressive filling (current approach + template guidance)
   - b) Only populated at end when `orch session end` runs (issue requirement #4)
   
3. **SPAWN_CONTEXT.md necessity:** Is this truly needed for interactive sessions? Spawned orchestrators get ORCHESTRATOR_CONTEXT.md, but interactive sessions have direct Claude Code access. What would go in SPAWN_CONTEXT.md that isn't in SESSION_HANDOFF.md?

4. **SYNTHESIS.md purpose:** How should this differ from SESSION_HANDOFF.md? Both seem to serve similar "accumulate session context" purpose.

5. **Issue status:** Was the issue author aware of the Jan 6 implementation? Should issue be updated to reflect current state?

### Recommended Approach ⭐ (pending clarification)

**Enhance existing workspace creation with missing artifacts** - Build on the Jan 6 implementation rather than replace it.

**Why this approach:**
- Avoids throwing away working code (workspace creation exists and works)
- Jan 6 implementation already handles SESSION_HANDOFF.md generation correctly
- Can incrementally add SPAWN_CONTEXT.md and SYNTHESIS.md
- Maintains compatibility with existing sessions

**Trade-offs accepted:**
- Defers location decision until orchestrator clarifies requirement
- May need to migrate existing workspaces if location changes
- Assumes progressive filling is correct approach (aligns with template)

**Implementation sequence (after decisions):**
1. Add workspace_path field to session.json (enables tracking)
2. Create SPAWN_CONTEXT.md template and generation function
3. Create empty SYNTHESIS.md on session start
4. Update `orch session status` to display workspace location
5. Update `orch session end` to validate/reminder about SESSION_HANDOFF.md completion

### Alternative Approaches Considered

**Option B: Replace existing implementation with issue requirements exactly**
- **Pros:** Matches issue description literally
- **Cons:** Throws away working code; issue description may be outdated; location conflicts with prior decisions
- **When to use instead:** If orchestrator confirms issue is correct and current implementation is wrong

**Option C: Keep current implementation as-is, close issue as duplicate**
- **Pros:** No code changes needed; workspace creation already works
- **Cons:** Missing SPAWN_CONTEXT.md and SYNTHESIS.md; workspace not tracked in session.json; doesn't fully solve original problem
- **When to use instead:** If orchestrator determines SPAWN_CONTEXT.md/SYNTHESIS.md aren't actually needed

**Rationale for recommendation:** Option A (enhance existing) balances pragmatism (don't throw away working code) with completeness (add missing pieces). However, cannot proceed without orchestrator clarifications on the 5 questions above.

---

### Implementation Details

**What to implement first:**
- [Highest priority change based on findings]
- [Quick wins or foundational work]
- [Dependencies that need to be addressed early]

**Things to watch out for:**
- ⚠️ [Edge cases or gotchas discovered during investigation]
- ⚠️ [Areas of uncertainty that need validation during implementation]
- ⚠️ [Performance, security, or compatibility concerns to address]

**Areas needing further investigation:**
- [Questions that arose but weren't in scope]
- [Uncertainty areas that might affect implementation]
- [Optional deep-dives that could improve the solution]

**Success criteria:**
- ✅ [How to know the implementation solved the investigated problem]
- ✅ [What to test or validate]
- ✅ [Metrics or observability to add]

---

## References

**Files Examined:**
- `cmd/orch/session.go:96-181` - runSessionStart() and createSessionWorkspace() implementation
- `pkg/spawn/orchestrator_context.go:531-565` - GeneratePreFilledSessionHandoff() function
- `~/.orch/session.json` - Current session structure (no workspace_path field)
- `~/.orch/session/2026-01-09/SESSION_HANDOFF.md` - Current session's workspace
- `.orch/templates/SESSION_HANDOFF.md` - Template with progressive documentation guidance

**Commands Run:**
```bash
# Check recent commits
git log --oneline --all | grep -i "workspace\|session" | head -15

# Find when createSessionWorkspace was added
git log --all -1 --format="%H %ai %s" --grep="createSessionWorkspace"
# Output: 205f7ba 2026-01-06 12:56 fix: create workspace with SESSION_HANDOFF.md on orch session start

# Check current session workspace
ls -la ~/.orch/session/2026-01-09/
# Output: SESSION_HANDOFF.md created at 14:41

# Verify workspace creation code exists
grep -n "createSessionWorkspace" cmd/orch/session.go
```

**External Documentation:**
- None applicable

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-04-orchestrator-session-lifecycle.md` - Establishes orchestrator completion hierarchy
- **Investigation:** `.kb/investigations/2026-01-06-inv-interactive-orchestrator-sessions-don-create.md` - Prior investigation that led to Jan 6 fix
- **Issue:** `orch-go-38zik` (closed) - Original issue that was fixed Jan 6
- **Issue:** `orch-go-xn6ok` (current) - May be duplicate or requesting enhancements to Jan 6 fix

---

## Investigation History

**2026-01-09 14:43:** Investigation started
- Initial question: How to implement orchestrator workspace creation on session start per issue orch-go-xn6ok
- Context: Issue requests workspace creation, but unclear if this overlaps with Jan 6 implementation

**2026-01-09 14:50:** Discovered Jan 6 implementation
- Found createSessionWorkspace() already exists in cmd/orch/session.go
- Realized issue description may be outdated (created 3 days after implementation)
- Identified gaps: missing SPAWN_CONTEXT.md, SYNTHESIS.md, workspace tracking

**2026-01-09 15:00:** Investigation paused for escalation
- Status: Paused - needs orchestrator clarification
- Key outcome: Workspace creation exists but differs from issue requirements; need design decisions before proceeding
