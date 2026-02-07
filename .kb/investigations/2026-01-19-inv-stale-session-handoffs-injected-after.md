<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** Cross-window scan in `discoverSessionHandoff()` returns archived handoffs even after explicit `orch session end`, because it doesn't distinguish "ended explicitly" from "crashed mid-session".

**Evidence:** Traced flow: `orch session end` archives handoff to `{window}/latest/`, but `scanAllWindowsForMostRecent()` still finds and returns it. Session store shows `null` after end, but this isn't checked by resume logic.

**Knowledge:** The system can't distinguish two scenarios: (A) user explicitly ended session (wants fresh start), (B) user closed Claude without ending (wants resume). Both leave archived handoffs that the cross-window scan finds.

**Next:** Implement fix in `discoverSessionHandoff()` - only return archived handoffs if there's an active/ directory in some window, indicating a session is in progress.

**Promote to Decision:** Superseded - session handoff machinery removed (2026-01-19-remove-session-handoff-machinery.md)

---

# Investigation: Stale Session Handoffs After orch session end

**Question:** Why do stale session handoffs continue to be injected after `orch session end` completes?

**Started:** 2026-01-19
**Updated:** 2026-01-19
**Owner:** Agent (og-arch-stale-session-handoffs-19jan-fd5d)
**Phase:** Complete
**Next Step:** Implement recommended fix
**Status:** Complete

---

## Findings

### Finding 1: Session end archives handoff but leaves `latest` symlink pointing to it

**Evidence:** After `orch session end`:
- `archiveActiveSessionHandoff()` renames `active/` to `{timestamp}/` (line 594)
- Updates `latest` symlink to point to the new timestamped directory (lines 598-606)
- Session store (`~/.orch/session.json`) is set to `null` (line 273-280)

```bash
# After session end:
$ ls -la .orch/session/orch-go-4/
latest -> 2026-01-19-1529    # Points to archived session
2026-01-19-1529/             # Archived handoff here
# NO active/ directory       # Correctly removed
```

**Source:** `cmd/orch/session.go:576-613` (archiveActiveSessionHandoff)

**Significance:** The `latest` symlink is designed to persist for manual review (`orch session resume`), but this creates a problem for auto-injection.

---

### Finding 2: Cross-window scan finds archived handoffs regardless of session state

**Evidence:** In `discoverSessionHandoff()`:
1. Priority 1: Check `{currentWindow}/active/` - if none, continues
2. Priority 2: `scanAllWindowsForMostRecent()` scans ALL windows' `latest` symlinks
3. Returns the most recent archived handoff across all windows

The cross-window scan (lines 1232-1319) compares timestamps and returns the most recent handoff. It does NOT check:
- Whether the session store is active
- Whether any `active/` directory exists
- Whether the session was explicitly ended

**Source:** `cmd/orch/session.go:1331-1424` (discoverSessionHandoff), lines 1232-1319 (scanAllWindowsForMostRecent)

**Significance:** This is the root cause. The scan was added for the use case "user wants latest context regardless of window name" but doesn't account for "user explicitly ended session".

---

### Finding 3: Plugin calls resume unconditionally on session.created

**Evidence:** The `session-resume.js` plugin:
```javascript
// On session.created event:
const checkResult = await execAsync('orch session resume --check', { cwd: sessionDirectory });
if (checkResult.exitCode === undefined || checkResult.exitCode === 0) {
  const result = await execAsync('orch session resume --for-injection', { cwd: sessionDirectory });
  // Injects handoff...
}
```

The plugin trusts `--check` to determine if injection should happen. But `--check` returns 0 whenever ANY handoff exists (active or archived).

**Source:** `~/.config/opencode/plugin/session-resume.js:94-127`

**Significance:** The plugin has no way to distinguish "active session that should be resumed" from "ended session that should be fresh start".

---

### Finding 4: The system cannot distinguish explicit end from crash

**Evidence:** Both scenarios result in the same state:

| Scenario | active/ | latest symlink | session.json |
|----------|---------|----------------|--------------|
| Explicit `orch session end` | gone | points to archived | `null` |
| User closed Claude mid-session | exists | points to previous | active |

Actually, there IS a difference: with explicit end, the `active/` is archived. With crash, `active/` still exists.

**Source:** Code analysis of `runSessionEnd()` vs implicit close behavior

**Significance:** This difference can be leveraged for the fix: only return archived handoffs if there's an `active/` directory somewhere (indicating crash recovery scenario).

---

## Synthesis

**Key Insights:**

1. **active/ as session-in-progress signal** - The presence of an `active/` directory indicates a session is in progress. No `active/` means either: (a) fresh start, or (b) session was explicitly ended. In both cases, auto-injection shouldn't happen.

2. **Cross-window scan over-reaches** - The scan was designed for "find latest context across windows" but should only apply when there's evidence of a session needing recovery (i.e., an `active/` directory exists somewhere).

3. **Two use cases conflated** - Auto-injection (via plugin) should only resume active sessions. Manual `orch session resume` can still show archived handoffs for review.

**Answer to Investigation Question:**

Stale handoffs are injected after `orch session end` because `discoverSessionHandoff()` unconditionally scans all windows for archived handoffs via `scanAllWindowsForMostRecent()`. This scan doesn't check if any session is actually in progress (has `active/` directory). When user explicitly ends a session with `orch session end`, the `active/` directory is archived but the `latest` symlink still points to the archived handoff, which the scan finds and returns.

---

## Structured Uncertainty

**What's tested:**

- ✅ `orch session end` correctly archives `active/` to timestamped directory (verified: `ls -la .orch/session/orch-go-4/`)
- ✅ `latest` symlink updated to point to archived session (verified: `latest -> 2026-01-19-1529`)
- ✅ Session store cleared after end (verified: `session.json` shows `null`)
- ✅ Plugin calls `orch session resume --check` which succeeds when archived handoff exists (code review)

**What's untested:**

- ⚠️ The proposed fix doesn't break crash recovery scenario (needs implementation + test)
- ⚠️ Multi-project behavior (session store is global, projects have separate `.orch/session/`)

**What would change this:**

- Finding would be wrong if `scanAllWindowsForMostRecent()` already checks for active sessions (it doesn't)
- Finding would be incomplete if there's another code path injecting handoffs (there isn't)

---

## Implementation Recommendations

**Purpose:** Fix the stale handoff injection by modifying discovery logic.

### Recommended Approach ⭐

**Require active/ for archived handoff return** - Modify `discoverSessionHandoff()` to only return archived handoffs if there's an `active/` directory in at least one window.

**Why this approach:**
- Minimal change (single function modification)
- Leverages existing signal (`active/` directory presence)
- Preserves manual `orch session resume` for review
- Handles crash recovery correctly (active/ exists after crash)

**Trade-offs accepted:**
- Users who want to resume after explicit end must run `orch session start` first
- This is actually correct behavior: explicit end = explicit restart needed

**Implementation sequence:**
1. Add helper function `hasActiveSessionAnywhere(sessionBaseDir)` that checks for `active/` in any window
2. In `discoverSessionHandoff()`, after Priority 1 (current window active/) fails:
   - Call `hasActiveSessionAnywhere()` to check if ANY window has active/
   - If yes, proceed to cross-window scan (crash recovery scenario)
   - If no, skip to legacy fallback or return "no handoff" (explicit end scenario)
3. Add test cases for both scenarios

### Alternative Approaches Considered

**Option B: Add staleness marker file on session end**
- **Pros:** Explicit signal that session was ended
- **Cons:** More state to track, marker must be cleaned up on `orch session start`
- **When to use instead:** If `active/` logic proves insufficient

**Option C: Check session store before injection**
- **Pros:** Session store already tracks active state
- **Cons:** Session store is global (not per-project), plugin would need to parse JSON
- **When to use instead:** If multi-project use case requires global awareness

**Option D: Add `--require-active` flag to resume command**
- **Pros:** Plugin can opt-in to strict behavior
- **Cons:** Requires changes to both Go code and JS plugin
- **When to use instead:** If we need backward compatibility with existing behavior

**Rationale for recommendation:** Option A (active/ check) is simplest and leverages existing directory structure as the signal. It correctly handles both scenarios without adding new state.

---

### Implementation Details

**What to implement first:**
1. `hasActiveSessionAnywhere()` helper in `cmd/orch/session.go`
2. Gate the cross-window scan on this check
3. Test: explicit end → new session → no injection
4. Test: mid-session crash → new session → injection works

**Things to watch out for:**
- ⚠️ Multi-window scenarios: user has `active/` in window A, starts new session in window B - should NOT inject A's context
- ⚠️ Race condition: user ends session just as new session starts

**Areas needing further investigation:**
- Should `orch session resume` (manual command) also respect this gate? Current recommendation: NO - manual command can always show archived handoffs for review

**Success criteria:**
- ✅ After `orch session end`, new session starts fresh (no handoff injected)
- ✅ After implicit close (crash), new session resumes correctly
- ✅ `orch session resume` (manual) still shows archived handoffs

---

## References

**Files Examined:**
- `cmd/orch/session.go` - Session start/end logic, discoverSessionHandoff, scanAllWindowsForMostRecent
- `~/.config/opencode/plugin/session-resume.js` - Plugin injection logic
- `.kb/guides/session-resume-protocol.md` - Design documentation
- `pkg/session/session.go` - Session store implementation

**Commands Run:**
```bash
# Check session directory structure
ls -la .orch/session/
ls -la .orch/session/orch-go-4/

# Check session store state
cat ~/.orch/session.json

# Get current tmux window
tmux display-message -p '#W'
```

**Related Artifacts:**
- **Guide:** `.kb/guides/session-resume-protocol.md` - Session resume design
- **Investigation:** `.kb/investigations/2026-01-11-design-session-resume-protocol.md` - Original design

---

## Investigation History

**2026-01-19 15:30:** Investigation started
- Initial question: Why do stale handoffs inject after explicit session end?
- Context: Bug report from user observing old handoff injected after orch session end

**2026-01-19 15:45:** Root cause identified
- Found that cross-window scan returns archived handoffs without checking for active sessions
- active/ directory presence is the key signal

**2026-01-19 16:00:** Investigation completed
- Status: Complete
- Key outcome: Fix recommended - gate cross-window scan on active/ directory presence
