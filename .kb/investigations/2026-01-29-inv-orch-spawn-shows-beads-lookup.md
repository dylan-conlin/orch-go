<!--
D.E.K.N. Summary - 30-second handoff for fresh Claude
Fill this at the END of your investigation, before marking Complete.
-->

## Summary (D.E.K.N.)

**Delta:** `bd show` CLI returns exit code 0 with empty stdout (not JSON parse error) when issue doesn't exist, causing noisy warnings for cross-project lookups.

**Evidence:** Tested `bd show specs-platform-28 --json` from orch-go directory - exit code 0, empty stdout, error goes to stderr.

**Knowledge:** When looking up cross-project issues (e.g., `specs-platform-36` from `orch-go`), the issue doesn't exist locally. This is expected behavior, not an error that needs logging.

**Next:** Fix implemented - added `ErrIssueNotFound` error type and suppress warnings for expected "not found" cases.

**Promote to Decision:** recommend-no (bug fix, not architectural)

---

# Investigation: Orch Spawn Shows Beads Lookup Warnings

**Question:** Why does `orch spawn` show "beads lookup failed" warnings for cross-project issues?

**Started:** 2026-01-29
**Updated:** 2026-01-29
**Owner:** spawned worker agent
**Phase:** Complete
**Next Step:** None
**Status:** Complete

---

## Findings

### Finding 1: `bd show` returns exit code 0 with empty output for non-existent issues

**Evidence:** 
```bash
$ cd /Users/dylanconlin/Documents/personal/orch-go
$ bd show specs-platform-28 --json 2>/dev/null
# (empty output)
$ echo $?
0
```

The error message "no issue found" goes to stderr, but the exit code is 0.

**Source:** Manual testing in the orch-go directory

**Significance:** The Go code was trying to parse empty output as JSON, causing "unexpected end of JSON input" errors. This is a bd CLI bug (should return non-zero exit code for not found) but we need to handle it gracefully.

---

### Finding 2: Cross-project sessions are expected and common

**Evidence:** OpenCode sessions list shows agents from multiple projects:
- `og-arch-orch-spawn-shows-29jan-f11a [orch-go-21012]`
- `sp-feat-high-replace-deprecated-29jan-6fb6 [specs-platform-36]`
- `sp-feat-critical-cookie-security-29jan-f8b1 [specs-platform-32]`

**Source:** `curl -s http://127.0.0.1:4096/session | jq -r '.[].title'`

**Significance:** When running `orch spawn` from `orch-go`, it checks ALL sessions for concurrency limiting, including `specs-platform-*` sessions whose beads IDs don't exist in the `orch-go` beads database. These lookups will always fail, making the warnings noise rather than actionable errors.

---

### Finding 3: Project resolution works correctly but lookup still fails

**Evidence:** 
- `extractProjectFromBeadsID("specs-platform-36")` correctly returns `"specs-platform"`
- `kb projects list` includes `specs-platform` with correct path
- However, `GetClosedIssuesBatch` was called without `projectDirs`, so it used current directory for lookup

**Source:** `pkg/daemon/active_count.go:151` calls `daemon.GetClosedIssuesBatch(beadsIDs)` without project dirs

**Significance:** The `spawn_concurrency.go` path doesn't use the session directory for cross-project resolution, so it falls back to kb projects lookup which might not always work.

---

## Synthesis

**Key Insights:**

1. **Empty output is not a parse error** - When `bd show` doesn't find an issue, it returns empty stdout with exit code 0. This should be treated as "not found", not "parse error".

2. **"Issue not found" is expected for cross-project lookups** - When running orch commands from one project while agents from other projects are active, their beads IDs won't be found locally. This is normal, not an error.

3. **Warnings should be for unexpected errors only** - Logging warnings for expected "not found" cases creates noise that obscures real problems.

**Answer to Investigation Question:**

The warnings appear because:
1. OpenCode sessions persist from multiple projects
2. `orch spawn` checks ALL recent sessions for concurrency limiting
3. When checking `specs-platform-*` IDs from `orch-go` directory, bd returns empty output (exit code 0)
4. The code was logging all lookup failures as warnings, not distinguishing "not found" from actual errors

---

## Structured Uncertainty

**What's tested:**

- ✅ `bd show` returns empty stdout + exit code 0 for non-existent issues (verified: ran `bd show specs-platform-28 --json` from orch-go)
- ✅ `FallbackShowWithDir` now returns `ErrIssueNotFound` for empty output (verified: manual test script)
- ✅ `FallbackShowWithDir` finds issues when called with correct project directory (verified: tested with specs-platform path)
- ✅ `errors.Is(err, ErrIssueNotFound)` works correctly (verified: unit test)

**What's untested:**

- ⚠️ RPC path (Client.Show) for ErrIssueNotFound - not tested due to daemon availability
- ⚠️ Actual suppression of warnings during `orch spawn` - would need end-to-end test

**What would change this:**

- If bd CLI is fixed to return non-zero exit code for not found, our empty-output handling becomes redundant (but harmless)
- If OpenCode session API starts including project directory reliably, we could use projectDirs parameter

---

## Implementation Recommendations

### Recommended Approach ⭐

**Add ErrIssueNotFound and suppress warnings for expected failures**

**Why this approach:**
- Distinguishes expected "not found" from unexpected errors
- Reduces noise in logs without losing visibility into real problems
- Gracefully handles bd CLI quirk (exit code 0 with empty output)

**Trade-offs accepted:**
- "Issue not found" is silently treated as "closed" - could miss actual issues if bd has bugs
- Acceptable because: conservative behavior (treats unknown as closed) prevents capacity leaks

**Implementation sequence:**
1. Add `ErrIssueNotFound` error type to `pkg/beads/client.go`
2. Update `FallbackShow`, `FallbackShowWithDir`, and `Client.Show` to return `ErrIssueNotFound`
3. Update `getClosedIssuesForProject` to only log warnings for non-ErrIssueNotFound errors

### Alternative Approaches Considered

**Option B: Always pass projectDirs to GetClosedIssuesBatch**
- **Pros:** Would look up issues in correct project directories
- **Cons:** spawn_concurrency.go doesn't have session.Directory available; would require significant refactor
- **When to use instead:** If we need accurate cross-project status checking

**Option C: Skip cross-project beads lookups entirely**
- **Pros:** No lookups for issues we know won't be found
- **Cons:** Would miss issues that actually exist in other projects
- **When to use instead:** If performance becomes an issue

**Rationale for recommendation:** Option A fixes the symptom (noisy logs) without requiring architectural changes to the spawn concurrency code.

---

## References

**Files Examined:**
- `pkg/beads/client.go` - FallbackShow, FallbackShowWithDir, Client.Show methods
- `pkg/daemon/active_count.go` - getClosedIssuesForProject, GetClosedIssuesBatch
- `cmd/orch/spawn_concurrency.go` - checkConcurrencyLimit function

**Commands Run:**
```bash
# Check bd show behavior for non-existent issue
cd /Users/dylanconlin/Documents/personal/orch-go
bd show specs-platform-28 --json 2>/dev/null; echo "exit: $?"

# Check bd show for existing issue in correct project
cd /Users/dylanconlin/Documents/work/SendCutSend/scs-special-projects/specs-platform
bd show specs-platform-28 --json

# List OpenCode sessions
curl -s http://127.0.0.1:4096/session | jq -r '.[].title'
```

**Related Artifacts:**
- **Issue:** `orch-go-21012` - Bug report for this issue
- **Constraint:** "Untracked spawns generate placeholder beads IDs that fail bd comment commands" - Related known behavior

---

## Investigation History

**2026-01-29 19:35:** Investigation started
- Initial question: Why does orch spawn show 'beads lookup failed' warnings?
- Context: Warnings appearing for specs-platform-28, specs-platform-29, etc. on every spawn

**2026-01-29 19:37:** Root cause identified
- bd show returns exit code 0 with empty output for non-existent issues
- This is a bd CLI bug, but we need to handle it gracefully

**2026-01-29 19:40:** Fix implemented
- Added ErrIssueNotFound error type
- Updated FallbackShow, FallbackShowWithDir, Client.Show to return appropriate error
- Updated getClosedIssuesForProject to suppress warnings for ErrIssueNotFound

**2026-01-29 19:45:** Fix verified
- Manual test confirmed ErrIssueNotFound is returned for cross-project lookups
- Tests pass for beads and daemon packages
