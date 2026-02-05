## Summary (D.E.K.N.)

**Delta:** The Feb 4 commit (c6f579a1) removing `no-db: true` was a misdiagnosis - the "JSONL persistence failure" was actually a SQLite foreign key violation, and the daemon never respected no-db mode anyway.

**Evidence:** daemon-start-state.json shows error "failed to import JSONL: foreign key violation in imported data: table=dirty_issues". The daemon code (`daemon.go`) directly calls `sqlite.New()` without checking no-db config. JSONL-only is now the CLI default, making the config setting redundant.

**Knowledge:** (1) The beads daemon ALWAYS uses SQLite regardless of no-db config. (2) CLI commands respect no-db but daemon path doesn't. (3) Daemon restart backoff (from Jan 22 decision) IS implemented and working - this is protecting against WAL corruption. (4) The foreign key violation in dirty_issues table is unresolved.

**Next:** Fix the foreign key violation: run `bd doctor --fix` to re-import JSONL, or identify and remove the orphaned dirty_issues entry causing the FK constraint failure. The no-db config removal was harmless but misleading.

**Authority:** implementation - Data repair task within existing beads tools, no architectural changes needed.

---

# Investigation: Investigate Bd Persistence Bug Jan

**Question:** What was the actual JSONL failure on Feb 4, was removing no-db: true the right fix, and is SQLite now safe from WAL corruption?

**Started:** 2026-02-04
**Updated:** 2026-02-04
**Owner:** og-inv-investigate-bd-persistence-04feb-6b86
**Phase:** Complete
**Next Step:** None (recommend fix for foreign key violation)
**Status:** Complete

**Patches-Decision:** .kb/decisions/2026-01-22-beads-daemon-rapid-restart-prevention.md
**Extracted-From:** N/A

## Prior Work

| Investigation | Relationship | Verified | Conflicts |
|--------------|--------------|----------|-----------|
| .kb/investigations/2026-01-21-inv-investigate-beads-sqlite-database-corruption.md | extends | yes | None - findings confirmed |
| .kb/models/beads-database-corruption.md | confirms | yes | Model accurately describes WAL corruption mechanism |
| .kb/decisions/2026-01-22-beads-daemon-rapid-restart-prevention.md | confirms | yes | Backoff was implemented in beads daemon_start_state.go |

**Relationship types:** extends, confirms, contradicts, deepens
**Verified:** Did you check claims against primary sources?
**Conflicts:** What contradictions did you find?

---

## Findings

### Finding 1: The "JSONL persistence failure" was actually a SQLite foreign key violation

**Evidence:**
```json
// .beads/daemon-start-state.json
{
  "last_attempt": "2026-02-04T15:52:29.838355-08:00",
  "attempt_count": 1,
  "backoff_until": "2026-02-04T15:52:59.838355-08:00",
  "last_error": "cannot open database: failed to hydrate from multi-repo: failed to hydrate primary repo .: failed to import JSONL: foreign key violation in imported data: table=dirty_issues rowid=1 parent=issues"
}
```

**Source:** `.beads/daemon-start-state.json`, commit c6f579a1 message "fix(beads): remove no-db config causing JSONL persistence failures"

**Significance:** The commit message claimed "JSONL persistence failures" but the actual error was a SQLite hydration failure. The JSONL file itself was fine - the problem was importing it into SQLite. This was a misdiagnosis.

---

### Finding 2: The beads daemon ALWAYS uses SQLite, regardless of no-db config

**Evidence:**
```go
// ~/Documents/personal/beads/cmd/bd/daemon.go
store, err := sqlite.New(ctx, daemonDBPath)
```

No references to `no-db`, `NoDb`, or `no_db` in any daemon*.go files. The daemon code directly instantiates SQLite storage without checking config.

**Source:** `grep -rn "no-db\|NoDb\|no_db" ~/Documents/personal/beads/cmd/bd/daemon*.go` - returns empty

**Significance:** The `no-db: true` config only affects CLI commands using direct mode. When orch-go uses the daemon via RPC, it always gets SQLite. This is why removing `no-db: true` had no practical effect - the daemon path was always using SQLite anyway.

---

### Finding 3: JSONL-only is now the CLI default, making no-db config redundant

**Evidence:**
```go
// ~/Documents/personal/beads/cmd/bd/main.go:202-206
// JSONL-only is the default mode (noDb = true)
// --db flag opts into SQLite (sets noDb = false)
// --no-db flag is kept for backwards compatibility (confirms default)
// Config can have sqlite: true to opt into SQLite permanently
noDb = true // Default to JSONL-only mode
```

**Source:** `~/Documents/personal/beads/cmd/bd/main.go:202-216`

**Significance:** The Jan 26 addition of `no-db: true` was redundant because JSONL-only mode became the default. The Feb 4 removal was equally inconsequential for CLI commands. SQLite mode now requires explicit `sqlite: true` in config.

---

### Finding 4: Daemon restart backoff WAS implemented and IS working

**Evidence:**
```go
// ~/Documents/personal/beads/cmd/bd/daemon_start_state.go
minRestartInterval = 30 * time.Second
daemonStartBackoffSchedule = []time.Duration{
    30 * time.Second, 1 * time.Minute, 2 * time.Minute,
    5 * time.Minute, 10 * time.Minute, 30 * time.Minute,
}
```

The daemon-start-state.json shows `attempt_count: 1` and `backoff_until` set 30 seconds after the failure - exactly as designed.

**Source:** `~/Documents/personal/beads/cmd/bd/daemon_start_state.go`, `.beads/daemon-start-state.json`

**Significance:** The Jan 22 decision (.kb/decisions/2026-01-22-beads-daemon-rapid-restart-prevention.md) was implemented. The backoff mechanism is preventing rapid restart loops that caused WAL corruption in Jan 21-22. SQLite is "safe" from the corruption pattern, but the underlying data integrity issue remains.

---

### Finding 5: There's an unresolved foreign key violation in the data

**Evidence:**
```bash
$ bd doctor 2>&1 | grep "DB-JSONL Sync"
⚠  DB-JSONL Sync Count mismatch: database has 2743 issues, JSONL has 2744
```

The foreign key violation error mentions `table=dirty_issues rowid=1 parent=issues` - suggesting a dirty_issues entry references a non-existent issue.

**Source:** `bd doctor` output, daemon-start-state.json error message

**Significance:** The data has an integrity problem that needs to be resolved. The `dirty_issues` table has a foreign key to `issues`, and there's an orphaned entry. This is the actual bug that needs fixing.

---

## Synthesis

**Key Insights:**

1. **The commit message was misleading** - The Feb 4 commit claimed "JSONL persistence failures" but investigation found it was a SQLite hydration failure. The JSONL file was intact; the problem was importing it into SQLite.

2. **no-db config has limited scope** - The setting only affects CLI direct mode, not the daemon path used by orch-go. This created a false sense of security - agents thought they were using JSONL-only mode but daemon calls still touched SQLite.

3. **The backoff mechanism is working as designed** - The Jan 22 decision was implemented correctly. Daemon restarts are now rate-limited (30s minimum, exponential backoff to 30m). This protects against WAL corruption from rapid restart loops.

**Answer to Investigation Question:**

**Q1: What was the actual JSONL failure?**
It wasn't a JSONL failure - it was a SQLite foreign key violation when the daemon tried to hydrate from JSONL. The error message "failed to import JSONL: foreign key violation in imported data: table=dirty_issues rowid=1 parent=issues" shows the JSONL was readable, but contained a dirty_issues entry referencing a non-existent issue.

**Q2: Was removing no-db the right fix?**
No. The removal had no practical effect because:
- The daemon never respected no-db config (Finding 2)
- JSONL-only mode is now the default for CLI commands anyway (Finding 3)
The actual fix needed is to resolve the foreign key violation in the data.

**Q3: Is SQLite now safe from WAL corruption?**
Yes, for the rapid-restart pattern. The daemon restart backoff (Finding 4) prevents the 57+ restart cycles that caused Jan 21-22 corruption. However, the data still has an integrity issue that needs to be fixed.

---

## Structured Uncertainty

**What's tested:**

- ✅ Daemon always uses SQLite (verified: grepped daemon*.go for no-db references - none found)
- ✅ JSONL-only is CLI default (verified: read main.go:202-206 confirming `noDb = true`)
- ✅ Daemon backoff implemented (verified: read daemon_start_state.go, confirmed 30s minRestartInterval)
- ✅ bd create persists to JSONL (verified: ran `bd create` and `grep` on issues.jsonl)
- ✅ Foreign key error exists (verified: read daemon-start-state.json error message)

**What's untested:**

- ⚠️ `bd doctor --fix` will resolve the foreign key violation (not run - would modify production data)
- ⚠️ The specific dirty_issues entry causing FK error can be identified (didn't query SQLite directly)
- ⚠️ Daemon would work correctly if data was fixed (can't test without fixing the data)

**What would change this:**

- Finding 1 would be wrong if there was a separate JSONL write failure not captured in daemon-start-state.json
- Finding 2 would be wrong if there's a hidden no-db check in daemon code I didn't find
- Finding 4 conclusion "SQLite is safe" would be wrong if there's another corruption vector besides rapid restarts

---

## Implementation Recommendations

**Purpose:** Bridge from investigation findings to actionable implementation using directive guidance pattern (strong recommendations + visible reasoning).

### Recommendation Authority

| Recommendation | Authority | Rationale |
|----------------|-----------|-----------|
| Fix foreign key violation via bd doctor --fix | implementation | Data repair using existing beads tools, reversible, no architecture change |
| Consider making daemon respect no-db config | architectural | Would require beads codebase changes, affects multiple components |

### Recommended Approach ⭐

**Fix the data integrity issue** - Run `bd doctor --fix` or manually identify and remove the orphaned dirty_issues entry.

**Why this approach:**
- The daemon backoff mechanism IS working - the protection against WAL corruption is in place (Finding 4)
- The immediate problem is data integrity, not architecture (Finding 5)
- The no-db config removal was harmless - fixing the data is the actual fix needed

**Trade-offs accepted:**
- Not fixing the daemon's no-db handling - acceptable because CLI mode is now JSONL-only by default
- Relying on backoff instead of JSONL-only daemon - acceptable because backoff proven effective

**Implementation sequence:**
1. Run `bd doctor --fix` to attempt automatic repair
2. If that fails, identify the orphaned dirty_issues entry: `sqlite3 .beads/beads.db "SELECT * FROM dirty_issues WHERE issue_id NOT IN (SELECT id FROM issues)"`
3. Delete the orphaned entry and verify daemon starts cleanly

### Alternative Approaches Considered

**Option B: Make daemon respect no-db config**
- **Pros:** Would allow true JSONL-only mode system-wide
- **Cons:** Requires beads codebase changes, daemon architecture assumes SQLite
- **When to use instead:** If foreign key violations recur frequently

**Option C: Delete and rebuild SQLite database**
- **Pros:** Clean slate, definitely fixes FK violations
- **Cons:** Loses database state (dirty_issues, indexes), more disruptive
- **When to use instead:** If bd doctor --fix fails and manual repair is complex

**Rationale for recommendation:** Option A addresses the immediate issue with minimal disruption. The daemon backoff is already protecting against WAL corruption, so the urgency of the original Jan 26 decision has been addressed.

---

### Implementation Details

**What to implement first:**
- Run `bd doctor --fix` as immediate action
- Clear daemon-start-state.json after successful repair: `rm .beads/daemon-start-state.json`

**Things to watch out for:**
- ⚠️ The dirty_issues table tracks "issues modified since last JSONL sync" - deleting entries may cause sync drift
- ⚠️ If orphan came from cross-repo contamination (the original Jan 26 issue), ensure repos are properly isolated
- ⚠️ After repair, verify daemon starts cleanly before committing

**Areas needing further investigation:**
- Why did the orphaned dirty_issues entry get created? (Possible cross-repo contamination as noted in Jan 26 decision)
- Should beads daemon support JSONL-only mode? (Architectural question for beads maintainer)

**Success criteria:**
- ✅ `bd doctor` shows no warnings for DB-JSONL sync
- ✅ Daemon starts without foreign key errors
- ✅ daemon-start-state.json is removed (clean state)

---

## References

**Files Examined:**
- `.beads/config.yaml` - Current beads configuration (no-db not present)
- `.beads/daemon-start-state.json` - Shows foreign key error from Feb 4 15:52
- `~/Documents/personal/beads/cmd/bd/main.go:200-300` - CLI flag handling, shows noDb=true default
- `~/Documents/personal/beads/cmd/bd/daemon.go` - Daemon SQLite usage, no no-db check
- `~/Documents/personal/beads/cmd/bd/nodb.go` - JSONL-only mode implementation
- `~/Documents/personal/beads/cmd/bd/daemon_start_state.go` - Restart backoff implementation
- `.kb/decisions/2026-01-22-beads-daemon-rapid-restart-prevention.md` - Original backoff decision
- `.kb/models/beads-database-corruption.md` - WAL corruption model

**Commands Run:**
```bash
# Check commit that removed no-db
git show c6f579a1 --full-diff

# Check when no-db was added
git log --all --oneline -p -- .beads/config.yaml

# Verify daemon backoff implementation
grep -r "backoff\|minRestartInterval" ~/Documents/personal/beads/cmd/bd/daemon*.go

# Check daemon no-db handling
grep -rn "no-db\|NoDb" ~/Documents/personal/beads/cmd/bd/daemon*.go

# Test bd persistence
bd create "test-persistence-02" --type task && grep "test-persistence-02" .beads/issues.jsonl

# Check BD-JSONL sync status
bd doctor 2>&1 | grep -A2 "DB-JSONL Sync"
```

**Related Artifacts:**
- **Decision:** `.kb/decisions/2026-01-22-beads-daemon-rapid-restart-prevention.md` - The backoff implementation decision
- **Model:** `.kb/models/beads-database-corruption.md` - WAL corruption mechanism
- **Investigation:** `.kb/investigations/2026-01-21-inv-investigate-beads-sqlite-database-corruption.md` - Original corruption analysis
- **kb quick entry:** `kb-cc4851` - Jan 26 decision to use JSONL-only mode

---

## Investigation History

**2026-02-04 17:44:** Investigation started
- Initial question: What was the actual JSONL failure, was removing no-db correct, is SQLite now safe?
- Context: Commit c6f579a1 removed no-db: true claiming "JSONL persistence failures" but no investigation documented the failure

**2026-02-04 17:50:** Found daemon-start-state.json with foreign key error
- Discovered the "JSONL persistence failure" was actually a SQLite hydration error

**2026-02-04 17:55:** Verified daemon doesn't respect no-db config
- Grepped daemon*.go files, found zero references to no-db handling
- Confirmed daemon always uses SQLite via direct sqlite.New() call

**2026-02-04 18:00:** Verified backoff implementation is working
- Read daemon_start_state.go, confirmed 30s minimum interval and exponential backoff
- Confirmed Jan 22 decision was implemented

**2026-02-04 18:10:** Investigation completed
- Status: Complete
- Key outcome: Feb 4 commit was misdiagnosed fix; actual issue is foreign key violation in dirty_issues table that should be fixed with `bd doctor --fix`
