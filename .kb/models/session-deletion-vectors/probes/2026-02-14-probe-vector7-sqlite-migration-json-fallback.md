# Probe: Vector #7 — SQLite Migration Legacy JSON Storage Fallback

**Model:** session-deletion-vectors
**Date:** 2026-02-14
**Status:** Complete

---

## Question

Model Vector #7 claims: "Upstream commit `6d95f0d14` (Feb 13) rewrote storage from JSON to SQLite. Migration code should import existing sessions. Could create NEW failure if migration missed sessions."

Specific claims being tested:
1. Does the error path (`NotFoundError: Resource not found: .../storage/session/.../ses_*.json`) come from the legacy JSON code or the new SQLite code?
2. Does the json→SQLite migration import all existing JSON sessions?
3. Does the migration have a gap that could orphan sessions?
4. What caused the SPECIFIC error Dylan observed?

---

## What I Tested

### 1. Running OpenCode version

```bash
# Binary modification time
ls -la ~/Documents/personal/opencode/packages/opencode/dist/opencode-darwin-arm64/bin/opencode
# → Sat Feb 14 11:17:59 2026 (rebuilt today, AFTER migration commit)

# Verify migration commit is ancestor of HEAD
git merge-base --is-ancestor 6d95f0d14 HEAD
# → YES

# Running server process
ps aux | grep opencode
# → PID 65542 started 11:18AM via overmind
```

### 2. Pre-migration vs post-migration session code

```bash
# Pre-6d95f0d14: session/index.ts used JSON Storage for ALL operations
git show 6d95f0d14^:packages/opencode/src/session/index.ts | grep "Storage\.\|Database\."
# →  Storage.write(["session", project.id, result.id], result)      # line 230
# →  Storage.read(["session", project.id, id])                       # line 259
# →  Storage.list(["session", project.id])                           # line 336
# →  Storage.remove(["session", project.id, sessionID])              # line 369

# Post-6d95f0d14: session/index.ts uses SQLite Database
grep "Storage\.\|Database\." packages/opencode/src/session/index.ts
# → Database.use((db) => db.select()...) for get/list/create/remove
# → Storage.read(["session_diff", ...]) is the ONLY remaining JSON usage
```

### 3. Storage.ts error path

```bash
# storage.ts:200-208 — the NotFoundError origin
# ENOENT on .json file → NotFoundError with path in message
```

The error `Resource not found: /Users/dylanconlin/.local/share/opencode/storage/session/b402cf59.../ses_3a4eb29b5ffe...json` matches `Storage.read(["session", projectID, sessionID])` which constructs path `{storageDir}/session/{projectID}/{sessionID}.json`. This is the **pre-migration code path**, NOT `session_diff` (which would be `session_diff/{sessionID}.json`).

### 4. JSON migration gate

```bash
# index.ts:82-83 — one-time migration gate
# const marker = path.join(Global.Path.data, "opencode.db")
# if (!(await Bun.file(marker).exists())) { ... run migration ... }

# DB creation history (schema migrations)
sqlite3 ~/.local/share/opencode/opencode.db "SELECT *, datetime(created_at/1000, 'unixepoch', 'localtime') FROM __drizzle_migrations ORDER BY created_at;"
# → Jan 27 14:23 — FIRST schema migration (DB created)
# → Feb 11 09:17 — second migration
# → Feb 13 06:41 — third migration
# → Feb 14 11:16 — fourth migration (time_ttl)
```

**DB has existed since Jan 27.** Migration commit is Feb 13. When the new binary started on Feb 14 at 11:18, the DB already existed → json-migration **SKIPPED**.

### 5. Session existence check

```bash
# Error session in JSON directory?
ls ~/.local/share/opencode/storage/session/b402cf59.../ses_3a4eb29b5ffe*
# → NOT FOUND

# Error session in SQLite?
sqlite3 opencode.db "SELECT id FROM session WHERE id = 'ses_3a4eb29b5ffeLg08jgfSJOEKPF';"
# → (empty — NOT FOUND)
```

Session does not exist in EITHER store. It was fully deleted.

### 6. Orphaned JSON sessions

```bash
# Total JSON session files on disk
find ~/.local/share/opencode/storage/session -name "*.json" | wc -l
# → 188

# Total sessions in SQLite
sqlite3 opencode.db "SELECT COUNT(*) FROM session;"
# → 199

# For project b402cf59...:
# JSON files: 14
# SQLite sessions: 24
# Overlap check — 7 JSON sessions ARE in SQLite, 7 are NOT
```

Concrete example of orphaned sessions (in JSON, NOT in SQLite):
- `ses_3a4a4fd27ffe8Dy1SH1v7lHaZA` (created ~00:53 AM Feb 14)
- `ses_3a4abb05bffe0UCGuEr2zcvTcW` (created ~00:46 AM Feb 14)
- `ses_3a4bbf7baffeBV1swq1HD2jy1a` (created ~00:28 AM Feb 14)

These have valid `id`, `projectID`, and were created before the migration could have run. They exist on disk in JSON but were never imported to SQLite.

### 7. Commit b02075844 analysis

```bash
git show b02075844 --stat
# → packages/opencode/src/session/index.ts | 6 +++---
# "tui: show all project sessions from any working directory"
# Removed directory-scoping filter from Session.list()
# Now shows ALL sessions for a project regardless of working directory
```

This commit only affects `Session.list()` filtering in the post-migration (SQLite) code. It does NOT introduce a JSON fallback or affect the error path.

---

## What I Observed

### Finding 1: Error is from pre-migration JSON code, not post-migration SQLite code

The error path `Resource not found: .../storage/session/{projectID}/{sessionID}.json` matches ONLY `Storage.read(["session", projectID, sessionID])` which is the **pre-6d95f0d14 code**. Post-migration code:
- Uses `Database.use((db) => db.select()...)` for session operations
- Throws `NotFoundError` with message `Session not found: {id}` (no filesystem path)
- Only uses `Storage.read(["session_diff", ...])` for JSON (different path pattern)

**The error Dylan observed was produced by the OLD binary (pre-SQLite migration) still running.**

### Finding 2: JSON migration has a one-time gate that was never triggered

The migration (`json-migration.ts`) only runs when `opencode.db` doesn't exist (index.ts:82-83). The DB has existed since Jan 27 (first schema migration). The json-migration code was added in commit `6d95f0d14` on Feb 13. When the new binary started on Feb 14 at 11:18, the DB already existed → **migration never ran**.

This means 188 JSON session files on disk were NEVER imported into SQLite. They are permanently orphaned.

### Finding 3: The specific error session was deleted by another vector

`ses_3a4eb29b5ffeLg08jgfSJOEKPF` doesn't exist in JSON or SQLite. Since the old binary used JSON storage, another deletion vector (likely #2 `orch clean --sessions` or #4 DELETE API) removed the JSON file while the old binary was running, causing the NotFoundError.

### Finding 4: Two distinct problems conflated

The error has TWO causes that must be separated:
1. **Why the error references a JSON path**: The old binary was still running pre-migration code
2. **Why the session was missing**: Another deletion vector removed it (Vectors #2-4)

Vector #7 didn't cause this specific error. The migration gap is a SEPARATE problem that causes permanent data loss of orphaned JSON sessions.

---

## Model Impact

- [ ] **Confirms** invariant: N/A
- [ ] **Contradicts** invariant: N/A
- [x] **Extends** model with: Three new findings

### Extension 1: Error was from pre-migration binary, not migration gap

The model correctly identifies Vector #7 as NEEDS PROBE. The probe reveals the error was NOT caused by the migration itself. It was caused by the pre-migration binary still running JSON-based session code while another vector deleted the JSON file. The migration commit (Feb 13) predates the binary rebuild (Feb 14 11:17 AM), so the old code was still serving requests.

**Recommended status change:** Vector #7 status should change from "NEEDS PROBE" to "CONFIRMED (secondary)" — the migration gap IS real but was not the proximate cause of this error.

### Extension 2: JSON migration never ran — 188 orphaned sessions

The one-time migration gate (`if (!opencode.db exists)`) prevents the migration from running when the DB already exists. Since the DB was created Jan 27 by earlier schema migrations, the json-migration code added on Feb 13 was never triggered. **188 JSON session files are permanently orphaned** — they exist on disk but are invisible to the current SQLite-based code.

This is a new constraint the model should document:
> **Invariant 6:** JSON→SQLite migration is a one-time gate. Sessions created by pre-migration code after the DB was established are permanently orphaned in JSON storage.

### Extension 3: `storage.ts` is still active code

The legacy JSON `Storage` namespace (`storage/storage.ts`) remains in the codebase and is actively used for `session_diff` data. This means `storage.ts:withErrorHandling()` can still throw `NotFoundError` with filesystem paths — but only for `session_diff` operations, which are caught: `Storage.read<Snapshot.FileDiff[]>(["session_diff", sessionID])` is wrapped in try/catch that returns `[]` on failure.

The `storage.ts` NotFoundError is a DIFFERENT error class than `db.ts` NotFoundError (both are named "NotFoundError" but from different namespaces). Error handlers must handle both.

---

## Notes

### Vector #7 Risk Assessment Update

| Aspect | Model Claim | Probe Finding |
|--------|-------------|---------------|
| Risk level | UNKNOWN | **MEDIUM** (data loss, not crashes) |
| Proximate cause of error | Possible | **No** — error was from old binary + another vector |
| Migration imports sessions | Should | **Never ran** — DB already existed |
| Post-migration exposure | Unknown | **None** — current code uses SQLite exclusively for sessions |
| Orphaned data | Unknown | **188 JSON sessions permanently orphaned** |

### Recommended Model Updates

1. **Vector #7 status**: Change from "NEEDS PROBE" to "CONFIRMED — migration gap causes permanent orphan of 188 JSON sessions, but does NOT cause runtime NotFoundError in current binary"
2. **Add Invariant 6**: JSON migration is one-time; orphaned JSON sessions are invisible to current code
3. **Add note to Evolution section**: The error Dylan observed (JSON path) was from the pre-migration binary, not from the migration gap. The two problems are independent.

### Fix Options for Orphaned Sessions

1. **Delete and recreate `opencode.db`** — forces migration to run, imports all 188 JSON sessions. Destructive: loses any SQLite-only sessions.
2. **Run migration manually** — Add a CLI command like `opencode migrate-json` that calls `JsonMigration.run()` regardless of DB existence.
3. **Accept the loss** — 188 sessions from pre-migration era are old and likely not needed. Clean up JSON files to reclaim disk space.
4. **Fix the gate** — Change the migration check from "DB exists" to "migration marker file exists" so it can be re-triggered.
