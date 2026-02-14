# Session Synthesis

**Agent:** og-inv-probe-vector-sqlite-14feb-63fc
**Issue:** orch-go-q9c
**Duration:** ~45 min
**Outcome:** success

---

## TLDR

Probed Vector #7 (SQLite migration JSON fallback) of the session-deletion-vectors model. The specific error Dylan observed was from the **pre-migration binary** still running JSON storage code, NOT from the migration itself. However, the json→SQLite migration has a real gap: its one-time gate (checks if opencode.db exists) means it **never ran** because the DB existed since Jan 27. Result: **188 JSON sessions permanently orphaned on disk**, invisible to current SQLite-based code.

---

## Delta (What Changed)

### Files Created
- `.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector7-sqlite-migration-json-fallback.md` - Probe artifact with full evidence

### Files Modified
- None (research-only probe)

### Commits
- (pending)

---

## Evidence (What Was Observed)

- **Pre-6d95f0d14 code** (`opencode/src/session/index.ts`): ALL session operations used `Storage.read/write(["session", projectID, sessionID])` → JSON files
- **Post-6d95f0d14 code**: Session operations use `Database.use()` → SQLite. Only `session_diff` still uses JSON Storage
- **Error path match**: `storage/session/{projectID}/{sessionID}.json` matches ONLY the pre-migration `Storage.read` pattern, NOT the current `session_diff` pattern
- **Binary rebuilt Feb 14 at 11:17 AM**, server restarted 11:18 AM — old binary was running during the error
- **DB created Jan 27** (earliest `__drizzle_migrations` entry), migration commit is Feb 13 → migration gate always finds DB existing → **json-migration never runs**
- **188 JSON session files on disk**, 199 sessions in SQLite — partial overlap, 7 orphaned JSON sessions confirmed for project b402cf59...
- **Error session (`ses_3a4eb29b5ffe...`)** doesn't exist in JSON or SQLite — deleted by another vector
- **Commit b02075844** only removed directory filtering from `Session.list()` in SQLite code — irrelevant to the error

### Verification Contract

| Check | Command | Expected |
|-------|---------|----------|
| Probe file exists | `cat .kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector7-sqlite-migration-json-fallback.md` | Contains all 4 required sections |
| Model Impact has verdict | `grep "Extends" .kb/models/.../probes/...` | One checked box |
| No code changes | `git diff HEAD --stat` | Only new files in .kb/ and .orch/ |

---

## Knowledge (What Was Learned)

### New Artifacts
- `.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector7-sqlite-migration-json-fallback.md` - Probe with verdict: EXTENDS

### Constraints Discovered
- **JSON migration one-time gate**: `json-migration.ts` only runs if `opencode.db` doesn't exist. DB was created Jan 27 by schema migrations, so the JSON→SQLite data migration was never triggered. 188 sessions permanently orphaned.
- **Two NotFoundError classes coexist**: `storage.ts` and `db.ts` both export `NotFoundError` — they're from different namespaces and produce different error message formats (filesystem path vs session ID)
- **storage.ts is still active code**: Used for `session_diff` data. Not dead code — don't remove it.

### Externalized via `kb`
- `kb quick constrain "JSON→SQLite migration has one-time gate..." --reason "..."` → kb-76c738

---

## Next (What Should Happen)

**Recommendation:** close

### If Close
- [x] All deliverables complete (probe artifact written with all 4 sections)
- [x] Probe has Status: Complete
- [x] Model impact documented with recommended updates
- [x] Ready for `orch complete orch-go-q9c`

### Recommended Model Updates (for orchestrator)

1. Update Vector #7 status from "NEEDS PROBE" to "CONFIRMED — migration gap orphans 188 JSON sessions; does NOT cause runtime errors in current binary"
2. Add Invariant 6: "JSON→SQLite migration is one-time; pre-migration sessions in JSON are invisible to current code"
3. Add to Evolution: "Feb 14 probe confirmed the JSON-path error was from the pre-migration binary, not the migration gap itself"

CROSS_REPO_ISSUE:
  repo: ~/Documents/personal/opencode
  title: "JSON→SQLite migration never ran — 188 sessions orphaned"
  type: bug
  priority: 2
  description: "The json-migration in index.ts has a one-time gate (checks if opencode.db exists). Since the DB was created by earlier schema migrations on Jan 27, the JSON→SQLite data migration was never triggered. 188 JSON session files at ~/.local/share/opencode/storage/session/ are permanently invisible to the current SQLite-based code. Fix: either add a CLI command to trigger migration manually, change the gate to use a separate marker file, or accept the loss and clean up JSON files."

---

## Unexplored Questions

- **Why do some JSON sessions exist in SQLite too?** 7 of 14 JSON sessions for project b402cf59... are also in SQLite. This could mean: (a) the code briefly used SQLite before reverting to JSON at some point, or (b) a migration did run at some earlier point when the DB was recreated. The exact history is unclear from commit archaeology alone.
- **Are the 188 orphaned JSON sessions recoverable?** Deleting and recreating opencode.db would force the migration to run, but would lose any SQLite-only sessions (24 for this project). A merge strategy would be ideal.

*(No need to explore further — the probe answers the model's question.)*

---

## Session Metadata

**Skill:** investigation (probe mode)
**Model:** opus 4.6
**Workspace:** `.orch/workspace/og-inv-probe-vector-sqlite-14feb-63fc/`
**Probe:** `.kb/models/session-deletion-vectors/probes/2026-02-14-probe-vector7-sqlite-migration-json-fallback.md`
**Beads:** `bd show orch-go-q9c`
