<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Session resume --check fails due to migration gap between old non-window-scoped handoffs (.orch/session/latest) and new window-scoped discovery logic (.orch/session/{window-name}/latest) - code changed but data wasn't migrated.

**Evidence:** Filesystem shows both structures coexisting - old non-window-scoped files at .orch/session/2026-01-13-1000/ with .orch/session/latest symlink, new window-scoped at .orch/session/pw/2026-01-13-1305/, but discovery code only checks window-scoped paths causing exit 1 despite handoff existing.

**Knowledge:** Window-scoping was added in commit 3385796c to prevent concurrent orchestrators from clobbering each other, but no migration logic or fallback to old structure was implemented, creating discovery failure for all pre-window-scoping handoffs.

**Next:** Implement backward-compatible discovery that checks window-scoped path first, falls back to non-window-scoped path if not found, with optional migration command to move old handoffs to window-scoped structure.

**Promote to Decision:** recommend-yes - This establishes a pattern for schema migrations in orch-go: always provide backward-compatible discovery + optional migration tooling, never break existing data.

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

# Investigation: Session Resume Discovery Failure Pattern

**Question:** Why does `orch session resume --check` return exit 1 despite SESSION_HANDOFF.md existing, and why does this pattern recur across multiple sessions?

**Started:** 2026-01-13
**Updated:** 2026-01-13
**Owner:** og-arch-analyze-session-resume agent (orch-go-6nbug)
**Phase:** Complete
**Next Step:** None
**Status:** Complete

<!-- Lineage (fill only when applicable) -->
**Extracted-From:** [Project/path of original artifact, if this was extracted from another project]
**Supersedes:** [Path to artifact this replaces, if applicable]
**Superseded-By:** [Path to artifact that replaced this, if applicable]

---

## Findings

### Finding 1: Two filesystem structures coexist - old non-window-scoped and new window-scoped

**Evidence:**
- Old structure exists: `.orch/session/latest` → `2026-01-13-1000/SESSION_HANDOFF.md`
- New structure exists: `.orch/session/pw/latest` → `pw/2026-01-13-1305/SESSION_HANDOFF.md`
- Running `orch session resume --check` returns exit 1 (not found)
- Current window name: `📐 og-arch-analyze-session-resume-13jan-46d6 [orch-go-6nbug]`
- Discovery code sanitizes this to: `og-arch-analyze-session-resume-13jan-46d6-orch-go-6nbug`
- Discovery looks for: `.orch/session/og-arch-analyze-session-resume-13jan-46d6-orch-go-6nbug/latest/SESSION_HANDOFF.md`
- This path doesn't exist, but `.orch/session/latest/SESSION_HANDOFF.md` DOES exist

**Source:**
- `cmd/orch/session.go:614-672` - discoverSessionHandoff() function
- `ls -la .orch/session/` - Shows both 2026-01-13-1000/ (old) and pw/ (new) directories
- `readlink .orch/session/latest` → `2026-01-13-1000` (old structure)
- Git commit 3385796c - "feat: implement tmux window-scoped session handoffs"

**Significance:** The code expects window-scoped paths but filesystem contains mix of old (non-scoped) and new (scoped) structures. Discovery fails on old handoffs despite them being valid and recent.

---

### Finding 2: No migration logic or fallback was implemented when window-scoping was added

**Evidence:**
- Commit 3385796c added window-scoping to discovery logic (lines 634, 691, 755 in cmd/orch/session.go)
- Investigation `.kb/investigations/2026-01-13-inv-implement-tmux-window-scoped-session.md` shows window-scoping was intentional feature
- No code exists to migrate old handoffs to new structure
- No fallback logic to check non-window-scoped path if window-scoped path not found
- Discovery function returns error immediately if window-scoped path doesn't exist (line 671)

**Source:**
- `cmd/orch/session.go:614-672` - discoverSessionHandoff() only checks window-scoped paths
- Git log shows no migration-related commits after 3385796c
- `.kb/investigations/2026-01-13-inv-implement-tmux-window-scoped-session.md` - Documents feature but not migration

**Significance:** This is a schema migration without data migration - a classic breaking change. Every pre-existing handoff became undiscoverable the moment window-scoping shipped. This violates the principle that code changes should not silently break existing data.

---

### Finding 3: Pattern recurs because symptom (exit 1) doesn't reveal root cause (migration gap)

**Evidence:**
- Task description states "3 prior handoff investigations"
- Each investigation likely debugged "why isn't handoff found?" without identifying migration gap
- Exit 1 from --check provides no diagnostic information about which path was checked
- User sees "handoff exists but --check returns 1" which suggests discovery bug, not migration gap
- No warning or migration prompt when old structure detected

**Source:**
- Grep results show 28 files mentioning "session resume"
- Multiple investigations in .kb/investigations/ directory with session-resume in name
- Error message at line 671: `"no session handoff found for window %q"` doesn't mention fallback or migration
- Task description: "3 prior handoff investigations suggest systemic issue not isolated bug"

**Significance:** Poor observability amplifies the migration gap. If the error message revealed "checked window-scoped path, old non-scoped path also exists but wasn't checked", the migration gap would be obvious. Instead, users repeatedly investigate "discovery failure" without seeing the structural cause.

---

## Synthesis

**Key Insights:**

1. **Schema migrations require both code AND data migration** - Window-scoping changed the expected filesystem structure but provided no path for existing data to adapt. This is a common anti-pattern: "evolve the schema, strand the data."

2. **Observability determines whether problems are one-time or recurring** - The discovery failure provides no diagnostic context (which paths were checked, whether old structure exists), so each occurrence appears as an isolated bug rather than revealing the systemic migration gap. Better error messages would have made this obvious on first occurrence.

3. **Backward compatibility is cheap insurance against stranded data** - A fallback check for old structure (5-10 lines of code) would have provided seamless compatibility while users migrated at their own pace. Instead, all old handoffs became immediately undiscoverable.

**Answer to Investigation Question:**

`orch session resume --check` returns exit 1 despite handoffs existing because:

1. Window-scoping was added to discovery logic (commit 3385796c) to prevent concurrent sessions from clobbering each other's context
2. Old handoffs remain in non-window-scoped structure (`.orch/session/latest`) but discovery only checks window-scoped paths (`.orch/session/{window-name}/latest`)
3. No migration logic or fallback was implemented when the schema changed

This pattern recurs across 3+ sessions because the symptom (exit 1) doesn't reveal the root cause (migration gap). Each investigation debugs "why isn't handoff found?" without seeing the structural incompatibility between code expectations and filesystem reality.

---

## Structured Uncertainty

**What's tested:**

- ✅ Old handoff exists at `.orch/session/latest` (verified: `readlink .orch/session/latest` → `2026-01-13-1000`)
- ✅ New window-scoped structure exists (verified: `ls .orch/session/pw/latest`)
- ✅ Discovery checks window-scoped path only (verified: read code at cmd/orch/session.go:614-672)
- ✅ `orch session resume --check` returns exit 1 (verified: ran command, got exit code 1)
- ✅ Window name sanitization produces specific format (verified: investigation 2026-01-13-inv-implement-tmux-window-scoped-session.md)

**What's untested:**

- ⚠️ Fallback to old structure would fix discovery (logical inference, not tested with actual implementation)
- ⚠️ Better error messages would have prevented recurring investigations (assumption about human behavior)
- ⚠️ Migration command would be used if provided (assumes users prefer migration over manual fixing)

**What would change this:**

- Finding would be wrong if discovery code already has fallback logic that I missed during code review
- Finding would be wrong if old handoffs were intentionally deprecated (not migrated) as documented somewhere
- Migration gap diagnosis would be wrong if window-scoping predates all existing handoffs (timestamp check would show this)

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommended Approach ⭐

**Backward-compatible discovery with migration prompt** - Discovery checks window-scoped path first, falls back to non-window-scoped with warning that suggests explicit migration command.

**Why this approach:**
- **Zero disruption**: Existing handoffs discoverable immediately without user action (addresses Finding 1)
- **Pressure over compensation**: Warning creates pressure to migrate rather than silently working around the gap (addresses Finding 3)
- **User control**: Migration happens when user chooses via explicit command, not automatic file moves (respects user agency)
- **Migration observability**: Warning provides diagnostic information missing from original error (addresses Finding 3)

**Trade-offs accepted:**
- Old structure persists until user runs migration command (acceptable because fallback prevents breakage)
- Users might ignore migration warning indefinitely (acceptable because fallback keeps working)
- Requires implementing migration command in addition to fallback (small implementation cost for clear user intent)

**Implementation sequence:**
1. **Add fallback logic to discoverSessionHandoff()** (5-10 lines):
   - After window-scoped check fails, check `.orch/session/latest` path
   - If found, log warning with migration suggestion
   - Return old path for now

2. **Implement `orch session migrate` command**:
   - Discovers old handoffs in current project
   - Moves to window-scoped structure preserving timestamps
   - Updates latest symlink
   - Reports what was migrated

3. **Update error messages for observability**:
   - When neither path exists: "No handoff found (checked: {window-scoped}, {non-scoped})"
   - When fallback used: "⚠️ Using old handoff structure. Run 'orch session migrate' to update."
   - After migration: "✅ Migrated N handoff(s) to window-scoped structure"

### Alternative Approaches Considered

**Option B: Automatic migration on discovery**
- **Pros:** Self-healing, no user action required, old structure cleaned up immediately
- **Cons:** Surprising file moves without user knowledge, dangerous if multiple processes access handoffs, violates principle of explicit over implicit
- **When to use instead:** If handoffs were guaranteed single-process access and migration was idempotent

**Option C: Fallback only (no migration command)**
- **Pros:** Simplest implementation (5 lines), zero user action required, works forever
- **Cons:** Old structure persists indefinitely, no pressure to migrate, technical debt accumulates (Finding 3 - "pressure over compensation")
- **When to use instead:** If window-scoping was a failed experiment and we plan to revert it

**Option D: Hard break with migration tool only (no fallback)**
- **Pros:** Forces migration, clean cutover, no lingering old structure
- **Cons:** Breaks all existing workflows until user runs migration, violates Coherence Over Patches principle by forcing disruption
- **When to use instead:** If we're doing a major version bump and can tolerate breaking changes

**Rationale for recommendation:**

Option A (backward-compatible discovery with migration prompt) balances zero-disruption (Finding 1) with pressure-over-compensation (Finding 3). The fallback prevents immediate breakage while the migration command provides clear intent and observability. This follows the pattern: "make it work immediately, create pressure to do it right, provide tool to do it right."

Options B and D prioritize cleanup over compatibility (unacceptable given Finding 1). Option C compensates without creating pressure (violates principle from Finding 3).

---

### Implementation Details

**What to implement first:**
1. **Fallback discovery logic** (highest priority, unblocks all users immediately):
   - Modify `discoverSessionHandoff()` in cmd/orch/session.go
   - After line 669 (window-scoped check fails), add check for `.orch/session/latest`
   - If found, emit warning to stderr: `"⚠️ Using legacy session handoff. Run 'orch session migrate' to update to window-scoped structure."`
   - Return the old path

2. **Enhanced error message** (quick win, prevents future recurring investigations):
   - Update line 671 error message to show both paths checked
   - Format: `"No session handoff found. Checked:\n  - Window-scoped: {path1}\n  - Legacy: {path2}"`

3. **Migration command** (enables cleanup):
   - Add `orch session migrate` subcommand
   - Discovers all `.orch/session/{timestamp}/` directories (non-window-scoped)
   - Prompts: "Migrate N handoff(s) to window-scoped structure for window '{name}'?"
   - Creates `.orch/session/{window-name}/` directory
   - Moves timestamp directories preserving structure
   - Updates latest symlink
   - Removes old non-scoped latest symlink

**Things to watch out for:**
- ⚠️ **Multiple windows**: User might have handoffs for different windows - migration should preserve all, not just current window
- ⚠️ **Concurrent access**: Don't migrate while `orch session end` is running (race condition)
- ⚠️ **Symlink resolution**: Relative vs absolute paths when creating new symlinks (use relative like existing code)
- ⚠️ **Window name sanitization**: Use same sanitization logic as GetCurrentWindowName() to avoid mismatch

**Areas needing further investigation:**
- Should migration be per-window or all-windows-at-once?
- Should old non-window-scoped handoffs auto-migrate to "default" window?
- Should `orch session start` auto-migrate if old structure detected?

**Success criteria:**
- ✅ `orch session resume --check` returns exit 0 for both old and new structure handoffs
- ✅ Warning message appears when old structure used (observability)
- ✅ `orch session migrate` successfully moves old handoffs to window-scoped structure
- ✅ After migration, `orch session resume` works from window-scoped path
- ✅ Error message shows both paths when neither exists (prevents future recurring investigations)

---

## References

**Files Examined:**
- `cmd/orch/session.go:614-672` - discoverSessionHandoff() function showing window-scoped discovery logic
- `cmd/orch/session.go:674-769` - createSessionHandoffDirectory() function showing window-scoped creation logic
- `.kb/investigations/2026-01-13-inv-implement-tmux-window-scoped-session.md` - Investigation documenting window-scoping feature
- `.kb/guides/session-resume-protocol.md` - Guide documenting expected behavior (shows window-scoped structure)
- `.orch/session/` - Directory showing coexistence of old and new structures

**Commands Run:**
```bash
# Check filesystem structure
ls -la .orch/session/
# Output: Shows both 2026-01-13-1000/ (old) and pw/ (new) directories

# Check latest symlink target
readlink .orch/session/latest
# Output: 2026-01-13-1000 (points to old structure)

# Test session resume check
orch session resume --check; echo "Exit code: $?"
# Output: Exit code: 1 (not found despite handoff existing)

# Check current window name
tmux display-message -p '#W'
# Output: 📐 og-arch-analyze-session-resume-13jan-46d6 [orch-go-6nbug]

# Check git commits
git log --oneline --all --grep="window" --since="2026-01-10"
# Output: Shows commit 3385796c - "feat: implement tmux window-scoped session handoffs"
```

**External Documentation:**
- N/A (internal system investigation)

**Related Artifacts:**
- **Investigation:** `.kb/investigations/2026-01-13-inv-implement-tmux-window-scoped-session.md` - Documents window-scoping feature implementation
- **Investigation:** `.kb/investigations/2026-01-13-inv-implement-session-resume-protocol-orch.md` - Documents original session resume protocol
- **Guide:** `.kb/guides/session-resume-protocol.md` - Authoritative reference for session resume system
- **Principle:** Pressure Over Compensation (from ~/.kb/principles.md) - Informs recommendation for warning rather than silent fallback

---

## Investigation History

**2026-01-13 13:20:** Investigation started
- Initial question: Why does `orch session resume --check` return exit 1 despite SESSION_HANDOFF.md existing?
- Context: Task indicated 3 prior handoff investigations suggesting systemic issue not isolated bug

**2026-01-13 13:25:** Discovered filesystem structure mismatch
- Found both old (`.orch/session/latest`) and new (`.orch/session/pw/latest`) structures coexisting
- Discovery code only checks window-scoped paths, causing exit 1 on old handoffs

**2026-01-13 13:30:** Identified root cause as migration gap
- Window-scoping added in commit 3385796c without data migration or fallback logic
- This is classic schema migration without data migration pattern

**2026-01-13 13:40:** Synthesized findings and created recommendations
- Recommended backward-compatible discovery with migration prompt
- Explored 4 alternatives (automatic migration, fallback only, hard break, recommended hybrid)
- Cited Pressure Over Compensation principle to support recommendation

**2026-01-13 13:45:** Investigation completed
- Status: Complete
- Key outcome: Migration gap identified, backward-compatible solution designed with explicit migration tooling
